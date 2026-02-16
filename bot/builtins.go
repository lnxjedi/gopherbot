package bot

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
	"golang.org/x/sys/unix"
	"gopkg.in/yaml.v3"
)

// Cut off for listing channels after help text
const tooManyChannels = 4

func init() {
	robot.RegisterPlugin("builtin-fallback", robot.PluginHandler{Handler: fallback})
	robot.RegisterPlugin("builtin-dmadmin", robot.PluginHandler{Handler: dmadmin})
	robot.RegisterPlugin("builtin-help", robot.PluginHandler{Handler: help})
	robot.RegisterPlugin("builtin-admin", robot.PluginHandler{Handler: admin})
	robot.RegisterPlugin("builtin-logging", robot.PluginHandler{Handler: logging})
}

func defaultHelp() []string {
	return []string{
		"(alias) help <keyword> - get help for the provided <keyword>",
		"(alias) commands - browse command groups available in this channel",
		"(alias) help-all - help for all commands available in this channel, including global commands",
	}
}

/* builtin plugins, like help */

func fallback(m robot.Robot, command string, args ...string) (retval robot.TaskRetVal) {
	r := m.(Robot)
	if command == "init" {
		return // ignore init
	}
	botAlias := r.GetBotAttribute("alias").String()
	if command == "catchall" {
		channelName := r.GetMessage().Channel
		term := ""
		if len(args) > 0 {
			term = strings.TrimSpace(args[0])
		}
		term = normalizeFallbackTerm(term, botAlias, r.GetBotAttribute("name").String())

		entries := r.collectHelpCommandMetadata(true)
		matches := rankHelpMatches(entries, term)
		if len(matches) > 0 {
			limit := 4
			display := matches
			if len(display) > limit {
				display = display[:limit]
			}
			lines := make([]string, 0, len(display)*3+4)
			if len(channelName) > 0 {
				lines = append(lines, fmt.Sprintf("No command matched in channel '%s'.", channelName))
			} else {
				lines = append(lines, "Command not found.")
			}
			if len(term) > 0 {
				lines = append(lines, fmt.Sprintf("Closest matches for \"%s\":", term))
			} else {
				lines = append(lines, "Closest command matches:")
			}
			for i, match := range display {
				entry := match.Entry
				lines = append(lines, fmt.Sprintf("%d) [%s] %s", i+1, entry.PluginName, entry.Command))
				if len(entry.Usage) > 0 {
					lines = append(lines, "   Usage: "+r.formatHelpLine(entry.Usage))
				}
				if len(entry.Summary) > 0 {
					lines = append(lines, "   Summary: "+entry.Summary)
				}
			}
			top := display[0].Entry
			lines = append(lines, "Try: "+r.formatHelpLine("(alias) help "+top.PluginName))
			if len(channelName) > 0 {
				r.SayThread(strings.Join(lines, "\n"))
			} else {
				r.Say(strings.Join(lines, "\n"))
			}
			return
		}
		if len(channelName) > 0 {
			r.SayThread("No command matched in channel '%s'; try '%shelp'", channelName, botAlias)
		} else {
			r.Say("Command not found; try your command in a channel, or use '%shelp'", botAlias)
		}
	}
	return
}

var botRegex = regexp.MustCompile(`^([^(]*)\(bot\)(,?) *`)
var aliasRegex = regexp.MustCompile(`^\(alias\) *`)
var helpTokenRegex = regexp.MustCompile(`[A-Za-z0-9_-]+`)

func (r Robot) formatHelpLine(input string) (ret string) {
	w := getLockedWorker(r.tid)
	w.Unlock()
	botName := r.cfg.botinfo.UserName
	botAlias := string(r.cfg.alias)
	if len(botName) == 0 && len(botAlias) == 0 {
		ret = input
	} else {
		if botRegex.MatchString(input) {
			if len(botName) > 0 {
				ret = botRegex.ReplaceAllString(input, "${1}"+botName+"${2} ")
				w.Log(robot.Debug, "Sending '%s' to FormatHelp", ret)
			} else {
				ret = botRegex.ReplaceAllString(input, botAlias)
			}
		} else if aliasRegex.MatchString(input) {
			if len(botAlias) > 0 {
				ret = aliasRegex.ReplaceAllString(input, botAlias)
			} else {
				ret = aliasRegex.ReplaceAllString(input, botName+", ")
			}
		}
	}
	conn := getConnectorForProtocol(protocolFromIncoming(r.Incoming, r.Protocol))
	if conn == nil {
		return ret
	}
	return conn.FormatHelp(ret)
}

type helpCommandMetadata struct {
	PluginName string
	Command    string
	Usage      string
	Summary    string
	Examples   []string
	Keywords   []string
	Helptext   []string
	Scope      string
}

type rankedHelpMatch struct {
	Entry helpCommandMetadata
	Score int
}

func appendUniqueStrings(dst []string, src ...string) []string {
	seen := make(map[string]struct{}, len(dst)+len(src))
	for _, line := range dst {
		seen[line] = struct{}{}
	}
	for _, line := range src {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) == 0 {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		dst = append(dst, trimmed)
		seen[trimmed] = struct{}{}
	}
	return dst
}

func firstHelpLineAsUsage(lines []string) string {
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) == 0 {
			continue
		}
		parts := strings.SplitN(trimmed, " - ", 2)
		return strings.TrimSpace(parts[0])
	}
	return ""
}

func firstHelpLineSummary(lines []string) string {
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) == 0 {
			continue
		}
		parts := strings.SplitN(trimmed, " - ", 2)
		if len(parts) == 2 {
			return strings.TrimSpace(parts[1])
		}
	}
	return ""
}

func normalizeHelpPhrase(input string) string {
	normalized := strings.ToLower(strings.TrimSpace(input))
	normalized = strings.ReplaceAll(normalized, "-", " ")
	normalized = strings.Join(strings.Fields(normalized), " ")
	return normalized
}

func singularizeHelpToken(token string) string {
	if strings.HasSuffix(token, "ies") && len(token) > 3 {
		return strings.TrimSuffix(token, "ies") + "y"
	}
	if strings.HasSuffix(token, "es") && len(token) > 3 {
		return strings.TrimSuffix(token, "es")
	}
	if strings.HasSuffix(token, "s") && len(token) > 2 {
		return strings.TrimSuffix(token, "s")
	}
	return token
}

func helpTokenEquivalent(a, b string) bool {
	if a == b {
		return true
	}
	return singularizeHelpToken(a) == singularizeHelpToken(b)
}

func normalizeFallbackTerm(input, alias, botName string) string {
	term := strings.TrimSpace(input)
	if len(term) == 0 {
		return term
	}
	if len(alias) > 0 && strings.HasPrefix(term, alias) {
		term = strings.TrimSpace(strings.TrimPrefix(term, alias))
	}
	if len(botName) > 0 {
		lower := strings.ToLower(term)
		name := strings.ToLower(strings.TrimSpace(botName))
		trimNamePrefix := func(sep string) {
			prefix := name + sep
			if strings.HasPrefix(lower, prefix) {
				term = strings.TrimSpace(term[len(prefix):])
				lower = strings.ToLower(term)
			}
		}
		trimNamePrefix(",")
		trimNamePrefix(":")
		trimNamePrefix(" ")
		if strings.HasPrefix(lower, "@"+name+" ") {
			term = strings.TrimSpace(term[len(name)+2:])
		}
	}
	return strings.TrimSpace(term)
}

func helpScopeText(task *Task) string {
	if task.DirectOnly {
		return "direct message only"
	}
	if len(task.Channels) > 0 {
		if len(task.Channels) > tooManyChannels {
			return "channels: (many)"
		}
		return "channels: " + strings.Join(task.Channels, ", ")
	}
	if task.AllChannels {
		if task.AllowDirect {
			return "all channels + direct messages"
		}
		return "all channels"
	}
	if task.AllowDirect {
		return "direct messages"
	}
	return "channel scoped"
}

func (r Robot) collectHelpCommandMetadata(includeGlobal bool) []helpCommandMetadata {
	w := getLockedWorker(r.tid)
	w.Unlock()

	byCommand := make(map[string]*helpCommandMetadata)
	for _, t := range r.tasks.t[1:] {
		task, plugin, _ := getTask(t)
		if task == nil || plugin == nil || task.Disabled {
			continue
		}
		available, specific := w.pluginAvailable(task, true, true)
		if !available {
			continue
		}
		if !includeGlobal && !specific {
			continue
		}
		for _, matcher := range plugin.Commands {
			command := strings.TrimSpace(strings.ToLower(matcher.Command))
			if len(command) == 0 {
				continue
			}
			key := task.name + "|" + command
			entry, ok := byCommand[key]
			if !ok {
				entry = &helpCommandMetadata{
					PluginName: task.name,
					Command:    command,
					Scope:      helpScopeText(task),
				}
				byCommand[key] = entry
			}
			if len(entry.Usage) == 0 && len(strings.TrimSpace(matcher.Usage)) > 0 {
				entry.Usage = strings.TrimSpace(matcher.Usage)
			}
			if len(entry.Summary) == 0 && len(strings.TrimSpace(matcher.Summary)) > 0 {
				entry.Summary = strings.TrimSpace(matcher.Summary)
			}
			entry.Examples = appendUniqueStrings(entry.Examples, matcher.Examples...)
			entry.Keywords = appendUniqueStrings(entry.Keywords, matcher.Keywords...)
			entry.Helptext = appendUniqueStrings(entry.Helptext, matcher.Helptext...)
		}
	}

	results := make([]helpCommandMetadata, 0, len(byCommand))
	for _, entry := range byCommand {
		if len(entry.Usage) == 0 {
			entry.Usage = firstHelpLineAsUsage(entry.Helptext)
		}
		if len(entry.Usage) == 0 {
			entry.Usage = "(alias) " + entry.Command
		}
		if len(entry.Summary) == 0 {
			entry.Summary = firstHelpLineSummary(entry.Helptext)
		}
		results = append(results, *entry)
	}
	return results
}

func scoreHelpCommandMatch(entry helpCommandMetadata, term string) int {
	termPhrase := normalizeHelpPhrase(term)
	if len(termPhrase) == 0 {
		return 0
	}

	commandPhrase := normalizeHelpPhrase(entry.Command)
	pluginPhrase := normalizeHelpPhrase(entry.PluginName)
	score := 0
	if helpTokenEquivalent(termPhrase, commandPhrase) {
		score = 100
	}
	if helpTokenEquivalent(termPhrase, pluginPhrase) && score < 95 {
		score = 95
	}

	for _, keyword := range entry.Keywords {
		keyPhrase := normalizeHelpPhrase(keyword)
		if len(keyPhrase) == 0 {
			continue
		}
		if helpTokenEquivalent(termPhrase, keyPhrase) && score < 92 {
			score = 92
		} else if strings.Contains(keyPhrase, termPhrase) && score < 72 {
			score = 72
		}
	}

	if strings.Contains(commandPhrase, termPhrase) && score < 84 {
		score = 84
	}
	if strings.Contains(pluginPhrase, termPhrase) && score < 80 {
		score = 80
	}

	usage := normalizeHelpPhrase(entry.Usage)
	summary := normalizeHelpPhrase(entry.Summary)
	helpLines := normalizeHelpPhrase(strings.Join(entry.Helptext, " "))
	if strings.Contains(usage, termPhrase) && score < 65 {
		score = 65
	}
	if strings.Contains(summary, termPhrase) && score < 62 {
		score = 62
	}
	if strings.Contains(helpLines, termPhrase) && score < 58 {
		score = 58
	}

	termTokens := helpTokenRegex.FindAllString(termPhrase, -1)
	if len(termTokens) == 0 {
		return score
	}
	haystack := strings.Join([]string{
		entry.PluginName,
		entry.Command,
		strings.Join(entry.Keywords, " "),
		entry.Usage,
		entry.Summary,
		strings.Join(entry.Helptext, " "),
	}, " ")
	hayTokens := helpTokenRegex.FindAllString(normalizeHelpPhrase(haystack), -1)
	if len(hayTokens) == 0 {
		return score
	}
	tokenSet := make(map[string]struct{}, len(hayTokens))
	for _, token := range hayTokens {
		tokenSet[token] = struct{}{}
		tokenSet[singularizeHelpToken(token)] = struct{}{}
	}
	hits := 0
	for _, token := range termTokens {
		if _, ok := tokenSet[token]; ok {
			hits++
			continue
		}
		if _, ok := tokenSet[singularizeHelpToken(token)]; ok {
			hits++
		}
	}
	if hits > 0 {
		candidate := 40 + hits
		if candidate > score {
			score = candidate
		}
	}
	return score
}

func rankHelpMatches(entries []helpCommandMetadata, term string) []rankedHelpMatch {
	matches := make([]rankedHelpMatch, 0, len(entries))
	for _, entry := range entries {
		score := scoreHelpCommandMatch(entry, term)
		if score > 0 {
			matches = append(matches, rankedHelpMatch{Entry: entry, Score: score})
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Score == matches[j].Score {
			if matches[i].Entry.PluginName == matches[j].Entry.PluginName {
				return matches[i].Entry.Command < matches[j].Entry.Command
			}
			return matches[i].Entry.PluginName < matches[j].Entry.PluginName
		}
		return matches[i].Score > matches[j].Score
	})
	return matches
}

func (r Robot) renderHelpEntry(entry helpCommandMetadata, includeExamples, includeScope bool) string {
	lines := []string{fmt.Sprintf("[%s] %s", entry.PluginName, entry.Command)}
	if len(entry.Usage) > 0 {
		lines = append(lines, "Usage: "+r.formatHelpLine(entry.Usage))
	}
	if len(entry.Summary) > 0 {
		lines = append(lines, "Summary: "+entry.Summary)
	}
	if includeExamples && len(entry.Examples) > 0 {
		lines = append(lines, "Example: "+r.formatHelpLine(entry.Examples[0]))
	}
	if includeScope && len(entry.Scope) > 0 {
		lines = append(lines, "Availability: "+entry.Scope)
	}
	return strings.Join(lines, "\n")
}

func (r Robot) renderCommandsOverview(entries []helpCommandMetadata) string {
	byPlugin := make(map[string][]helpCommandMetadata)
	for _, entry := range entries {
		byPlugin[entry.PluginName] = append(byPlugin[entry.PluginName], entry)
	}
	pluginNames := make([]string, 0, len(byPlugin))
	for plugin := range byPlugin {
		pluginNames = append(pluginNames, plugin)
	}
	sort.Strings(pluginNames)

	lines := []string{"Command groups available in this channel:"}
	for _, plugin := range pluginNames {
		pluginEntries := byPlugin[plugin]
		sort.Slice(pluginEntries, func(i, j int) bool {
			return pluginEntries[i].Command < pluginEntries[j].Command
		})
		commands := make([]string, 0, len(pluginEntries))
		seen := map[string]struct{}{}
		firstUsage := ""
		for _, entry := range pluginEntries {
			if _, ok := seen[entry.Command]; ok {
				continue
			}
			seen[entry.Command] = struct{}{}
			commands = append(commands, entry.Command)
			if len(firstUsage) == 0 && len(entry.Usage) > 0 {
				firstUsage = r.formatHelpLine(entry.Usage)
			}
		}
		lines = append(lines, fmt.Sprintf("- %s: %s", plugin, strings.Join(commands, ", ")))
		if len(firstUsage) > 0 {
			lines = append(lines, "  Example: "+firstUsage)
		}
	}
	lines = append(lines, "Try: "+r.formatHelpLine("(alias) help <plugin|command|keyword>"))
	return strings.Join(lines, "\n")
}

func help(m robot.Robot, command string, args ...string) (retval robot.TaskRetVal) {
	r := m.(Robot)
	if command == "init" {
		return // ignore init
	}
	if command == "info" {
		admins := strings.Join(r.cfg.adminUsers, ", ")
		aliasCh := r.cfg.alias
		name := r.cfg.botinfo.UserName
		if len(name) == 0 {
			name = "(unknown)"
		}
		ID := r.cfg.botinfo.UserID
		if len(ID) == 0 {
			ID = "(unknown)"
		}
		var alias string
		if aliasCh == 0 {
			alias = "(not set)"
		} else {
			alias = string(aliasCh)
		}
		channelID, _ := handle.ExtractID(r.ProtocolChannel)
		msg := make([]string, 0, 7)
		msg = append(msg, "Here's some information about me and my running environment:")
		msg = append(msg, fmt.Sprintf("The hostname for the server I'm running on is: %s", hostName))
		msg = append(msg, fmt.Sprintf("My name is '%s', alias '%s', and my %s internal ID is '%s'", name, alias, r.Protocol, ID))
		msg = append(msg, fmt.Sprintf("This is channel '%s', %s internal ID: %s", r.Channel, r.Protocol, channelID))
		if r.CheckAdmin() {
			msg = append(msg, fmt.Sprintf("The gopherbot install directory is: %s", installPath))
			msg = append(msg, fmt.Sprintf("My home directory ($GOPHER_HOME) is: %s", homePath))
			if custom, ok := lookupEnv("GOPHER_CUSTOM_REPOSITORY"); ok {
				msg = append(msg, fmt.Sprintf("My git repository is: %s", custom))
			}
		}
		msg = append(msg, fmt.Sprintf("My software version is: Gopherbot %s, commit: %s", botVersion.Version, botVersion.Commit))
		msg = append(msg, fmt.Sprintf("The administrators for this robot are: %s", admins))
		adminContact := r.GetBotAttribute("contact")
		if len(adminContact.Attribute) > 0 {
			msg = append(msg, fmt.Sprintf("The administrative contact for this robot is: %s", adminContact))
		}
		r.MessageFormat(robot.Variable).SayThread(strings.Join(msg, "\n"))
	}
	if command == "help" || command == "help-all" || command == "commands" {
		lineSeparator := "\n\n"
		term := ""
		if len(args) > 0 {
			term = strings.TrimSpace(args[0])
		}
		hasKeyword := command == "help" && len(term) > 0
		sendOutput := func(message string) {
			if r.Incoming.ThreadedMessage {
				r.Reply(message)
			} else {
				r.SayThread(message)
			}
		}

		if command == "help" && !hasKeyword {
			conn := getConnectorForProtocol(protocolFromIncoming(r.Incoming, r.Protocol))
			var defaultHelpLines []string
			if conn != nil {
				defaultHelpLines = conn.DefaultHelp()
			}
			if len(defaultHelpLines) == 0 {
				defaultHelpLines = defaultHelp()
			}
			lines := make([]string, 0, len(defaultHelpLines)+2)
			lines = append(lines, "Quick help:")
			for _, line := range defaultHelpLines {
				lines = append(lines, r.formatHelpLine(line))
			}
			lines = append(lines, "Tip: "+r.formatHelpLine("(alias) commands")+" shows command groups in this channel.")
			sendOutput(strings.Join(lines, "\n"))
			return
		}

		switch command {
		case "commands":
			entries := r.collectHelpCommandMetadata(false)
			if len(entries) == 0 {
				sendOutput("Sorry, I couldn't find any commands available in this channel")
				return
			}
			sendOutput(r.renderCommandsOverview(entries))
		case "help-all":
			entries := r.collectHelpCommandMetadata(true)
			if len(entries) == 0 {
				sendOutput("Sorry, I couldn't find any commands available in this channel")
				return
			}
			sort.Slice(entries, func(i, j int) bool {
				if entries[i].PluginName == entries[j].PluginName {
					return entries[i].Command < entries[j].Command
				}
				return entries[i].PluginName < entries[j].PluginName
			})
			helpLines := make([]string, 0, len(entries))
			for _, entry := range entries {
				helpLines = append(helpLines, r.renderHelpEntry(entry, true, true))
			}
			sendOutput("Commands available in this channel (including global):\n" + strings.Join(helpLines, lineSeparator))
		case "help":
			entries := r.collectHelpCommandMetadata(true)
			matches := rankHelpMatches(entries, term)
			if len(matches) == 0 {
				sendOutput("Sorry, I didn't find any commands matching your keyword")
				return
			}

			limit := 12
			display := matches
			if len(display) > limit {
				display = display[:limit]
			}
			helpLines := make([]string, 0, len(display))
			for _, match := range display {
				helpLines = append(helpLines, r.renderHelpEntry(match.Entry, true, true))
			}
			header := fmt.Sprintf("Command matches for keyword: %s", strings.ToLower(term))
			if len(matches) > len(display) {
				header = fmt.Sprintf("%s (showing top %d of %d)", header, len(display), len(matches))
			}
			sendOutput(header + "\n" + strings.Join(helpLines, lineSeparator))
		}
	}
	return
}

func dmadmin(m robot.Robot, command string, args ...string) (retval robot.TaskRetVal) {
	r := m.(Robot)
	if command == "init" {
		return // ignore init
	}
	switch command {
	case "dumprobot":
		if r.Protocol != robot.Terminal && r.Protocol != robot.Test && r.Protocol != robot.SSH {
			r.Say("This command is only valid with the 'terminal' or 'ssh' connector")
			return
		}
		confLock.RLock()
		c, _ := yaml.Marshal(config)
		confLock.RUnlock()
		r.Fixed().Say("Here's how I've been configured, irrespective of interactive changes:\n%s", c)
	case "dumpplugdefault":
		found := false
		for _, t := range r.tasks.t[1:] {
			task, plugin, _ := getTask(t)
			if args[0] == task.name {
				if plugin == nil {
					r.Say("No default configuration available for task type 'job'")
					return
				}
				if plugin.taskType == taskExternal {
					found = true
					if cfg, err := getDefCfg(t); err == nil {
						r.Fixed().Say("Here's the default configuration for \"%s\":\n%s", args[0], *cfg)
					} else {
						r.Say("I had a problem looking that up - somebody should check my logs")
					}
				}
			}
		}
		if !found {
			r.Say("Didn't find a plugin named " + args[0])
		}
	case "dumpplugin":
		if r.Protocol != robot.Terminal && r.Protocol != robot.Test && r.Protocol != robot.SSH {
			r.Say("This command is only valid with the 'terminal' or 'ssh' connector")
			return
		}
		found := false
		for _, t := range r.tasks.t[1:] {
			task, plugin, _ := getTask(t)
			if args[0] == task.name {
				if plugin == nil {
					r.Say("Task '%s' is a job, not a plugin", task.name)
					return
				}
				found = true
				c, _ := yaml.Marshal(plugin)
				r.Fixed().Say("%s", c)
			}
		}
		if !found {
			r.Say("Didn't find a plugin named " + args[0])
		}
	case "listplugins":
		joiner := ", "
		message := "Here are the plugins I have configured:\n%s"
		wantDisabled := false
		if len(args[0]) > 0 {
			wantDisabled = true
			joiner = "\n"
			message = "Here's a list of all disabled plugins:\n%s"
		}
		plist := make([]string, 0, len(r.tasks.t))
		for _, t := range r.tasks.t[1:] {
			task, plugin, _ := getTask(t)
			if plugin == nil {
				continue
			}
			ptext := task.name
			if wantDisabled {
				if task.Disabled {
					ptext += "; reason: " + task.reason
					plist = append(plist, ptext)
				}
			} else {
				if task.Disabled {
					ptext += " (disabled)"
				}
				plist = append(plist, ptext)
			}
		}
		if len(plist) > 0 {
			r.Say(message, strings.Join(plist, joiner))
		} else { // note because of builtin plugins, plist is ALWAYS > 0 if disabled wasn't specified
			r.Say("There are no disabled plugins")
		}
	}
	return
}

var byebye = []string{
	"Sayonara!",
	"Adios",
	"Hasta la vista!",
	"Later gator!",
}

var rightback = []string{
	"Back in a flash!",
	"Be right back!",
	"You won't even have time to miss me...",
}

func logging(m robot.Robot, command string, args ...string) (retval robot.TaskRetVal) {
	r := m.(Robot)
	switch command {
	case "init":
		return
	case "level":
		setLogLevel(logStrToLevel(args[0]))
		r.Say("I've adjusted the log level to %s", args[0])
		Log(robot.Info, "User %s changed logging level to %s", r.User, args[0])
	case "show":
		page := 0
		if len(args) == 1 {
			page, _ = strconv.Atoi(args[0])
		}
		lines, wrap := logPage(page)
		if wrap {
			r.Say("(warning: value too large for pages, wrapped past beginning of log)")
		}
		r.Fixed().Say(strings.Join(lines, ""))
	case "showlevel":
		l := getLogLevel()
		r.Say("My current logging level is: %s", logLevelToStr(l))
	case "setlines":
		l, _ := strconv.Atoi(args[0])
		set := setLogPageLines(l)
		r.Say("Lines per page of log output set to: %d", set)
	}
	return
}

type psList struct {
	pslines []string
	wids    []int
}

func (p *psList) Len() int {
	return len(p.pslines)
}

func (p *psList) Swap(i, j int) {
	p.pslines[i], p.pslines[j] = p.pslines[j], p.pslines[i]
	p.wids[i], p.wids[j] = p.wids[j], p.wids[i]
}

func (p *psList) Less(i, j int) bool {
	return p.wids[i] < p.wids[j]
}

func admin(m robot.Robot, command string, args ...string) (retval robot.TaskRetVal) {
	if command == "init" {
		return // ignore init
	}
	r := m.(Robot)
	w := getLockedWorker(r.tid)
	w.Unlock()
	switch command {
	case "reload":
		err := loadConfig(false)
		if err != nil {
			r.Reply("Error encountered during reload:")
			r.Fixed().Say("%v", err)
			Log(robot.Error, "Reloading configuration, requested by %s: %v", r.User, err)
			return
		}
		r.Reply("Configuration reloaded successfully")
		w.Log(robot.Info, "Configuration successfully reloaded by a request from: %s", r.User)
	case "protocollist":
		sourceProtocol := protocolFromIncoming(r.Incoming, r.Protocol)
		primaryProtocol, _ := getRuntimePrimaryProtocol()
		if !isPrimaryProtocolSource(sourceProtocol) {
			r.Say("This command is only available from the primary protocol '%s'", primaryProtocol)
			return
		}
		statuses := listConnectorProtocolStatus()
		if len(statuses) == 0 {
			r.Say("No protocol runtimes are configured")
			return
		}
		lines := make([]string, 0, len(statuses)+1)
		lines = append(lines, "Protocol runtime status:")
		for _, status := range statuses {
			line := fmt.Sprintf("%s (%s): %s", status.protocol, status.role, status.state)
			if status.err != "" {
				line += " (" + status.err + ")"
			}
			lines = append(lines, line)
		}
		r.Say(strings.Join(lines, "\n"))
	case "protocolstart":
		sourceProtocol := protocolFromIncoming(r.Incoming, r.Protocol)
		primaryProtocol, _ := getRuntimePrimaryProtocol()
		if !isPrimaryProtocolSource(sourceProtocol) {
			r.Say("This command is only available from the primary protocol '%s'", primaryProtocol)
			return
		}
		if len(args) == 0 || len(strings.TrimSpace(args[0])) == 0 {
			r.Say("Usage: protocol-start <protocol>")
			return
		}
		protocol := normalizeProtocolName(args[0])
		if err := startSecondaryConnectorRuntime(protocol); err != nil {
			r.Say("Unable to start protocol '%s': %v", protocol, err)
			return
		}
		r.Say("Started protocol '%s'", protocol)
	case "protocolstop":
		sourceProtocol := protocolFromIncoming(r.Incoming, r.Protocol)
		primaryProtocol, _ := getRuntimePrimaryProtocol()
		if !isPrimaryProtocolSource(sourceProtocol) {
			r.Say("This command is only available from the primary protocol '%s'", primaryProtocol)
			return
		}
		if len(args) == 0 || len(strings.TrimSpace(args[0])) == 0 {
			r.Say("Usage: protocol-stop <protocol>")
			return
		}
		protocol := normalizeProtocolName(args[0])
		if err := stopSecondaryConnectorRuntime(protocol); err != nil {
			r.Say("Unable to stop protocol '%s': %v", protocol, err)
			return
		}
		r.Say("Stopped protocol '%s'", protocol)
	case "protocolrestart":
		sourceProtocol := protocolFromIncoming(r.Incoming, r.Protocol)
		primaryProtocol, _ := getRuntimePrimaryProtocol()
		if !isPrimaryProtocolSource(sourceProtocol) {
			r.Say("This command is only available from the primary protocol '%s'", primaryProtocol)
			return
		}
		if len(args) == 0 || len(strings.TrimSpace(args[0])) == 0 {
			r.Say("Usage: protocol-restart <protocol>")
			return
		}
		protocol := normalizeProtocolName(args[0])
		if err := restartSecondaryConnectorRuntime(protocol); err != nil {
			r.Say("Unable to restart protocol '%s': %v", protocol, err)
			return
		}
		r.Say("Restarted protocol '%s'", protocol)
	case "abort":
		buf := make([]byte, 32768)
		runtime.Stack(buf, true)
		log.Printf("%s", buf)
		time.Sleep(2 * time.Second)
		panic("Abort command issued")
	case "ps":
		// wid pwid pid Go|Ext plugin|task|job
		psl := &psList{
			pslines: []string{
				"WID    PWID  PID   G/E TYPE   PIPENAME         TASK             PLUG-COMMAND ARGS",
			},
			wids: []int{-1},
		}
		activePipelines.Lock()
		if len(activePipelines.i) == 1 {
			activePipelines.Unlock()
			r.Say("No pipelines running")
			return
		}
		for widx, worker := range activePipelines.i {
			pipename := worker.pipeName
			worker.Lock()
			wid := strconv.Itoa(widx)
			pwid := ""
			if worker._parent != nil {
				pwid = strconv.Itoa(worker._parent.id)
			}
			pid := ""
			if worker.osCmd != nil {
				pid = strconv.Itoa(worker.osCmd.Process.Pid)
				wid = wid + "*"
			}
			class := worker.taskClass
			ttype := worker.taskType
			tname := worker.taskName
			command := worker.plugCommand
			args := strings.Join(worker.taskArgs, " ")
			worker.Unlock()
			if pipename == "builtin-admin" && command == "ps" {
				continue
			}
			psline := fmt.Sprintf("%6.6s %5.5s %5.5s %-3.3s %-6.6s %-16.16s %-16.16s %-12.12s %s", wid, pwid, pid, class, ttype, pipename, tname, command, args)
			psl.pslines = append(psl.pslines, psline)
			psl.wids = append(psl.wids, widx)
		}
		activePipelines.Unlock()
		sort.Sort(psl)
		r.Fixed().Say(strings.Join(psl.pslines, "\n"))
	case "kill":
		if len(args) == 0 {
			r.Say("Usage: kill <wid>")
			return
		}
		wid := args[0]
		widx, err := strconv.ParseInt(wid, 10, 0)
		if err != nil {
			r.Say("Couldn't convert '%s' to an int", wid)
			return
		}
		activePipelines.Lock()
		worker, ok := activePipelines.i[int(widx)]
		activePipelines.Unlock()
		if !ok {
			r.Say("Pipeline %s not found", wid)
			return
		}
		var pid int
		var activeTaskTID int
		var rpcCancel context.CancelFunc
		worker.Lock()
		if worker.osCmd != nil {
			pid = worker.osCmd.Process.Pid
		}
		activeTaskTID = worker.activeTaskTID
		rpcCancel = worker.rpcCancel
		worker.Unlock()
		if rpcCancel != nil {
			rpcCancel()
		}
		_ = interruptReplyWaitersForTask(activeTaskTID)
		if pid == 0 {
			r.Say("No active process found for pipeline")
			return
		}
		raiseThreadPriv(fmt.Sprintf("killing process %d", pid))
		if err := unix.Kill(-pid, unix.SIGKILL); err != nil {
			r.Say("Unable to kill pid %d: %v", pid, err)
			return
		}
		r.Say("Killed pid %d", pid)
	case "pause":
		name := args[0]
		notfound := "I don't have a job configured with that name"
		t := r.tasks.getTaskByName(name)
		if t == nil {
			r.Say(notfound)
			return
		}
		_, _, job := getTask(t)
		if job == nil {
			r.Say(notfound)
			return
		}
		pausedJobs.Lock()
		defer pausedJobs.Unlock()
		_, ok := pausedJobs.jobs[name]
		if ok {
			r.Say("That job has already been paused")
			return
		}
		m := r.GetMessage()
		pausedJobs.jobs[name] = m.User
		r.Say("Ok, I'll stop running '%s' as a scheduled task", name)
		return
	case "resume":
		name := args[0]
		t := r.tasks.getTaskByName(name)
		_, _, job := getTask(t)
		if job == nil {
			r.Say("I don't have a job configured with that name")
		}
		pausedJobs.Lock()
		defer pausedJobs.Unlock()
		_, ok := pausedJobs.jobs[name]
		if !ok {
			r.Say("That job isn't paused")
			return
		}
		delete(pausedJobs.jobs, name)
		r.Say("Ok, I'll resume running '%s' as a scheduled task", name)
		return
	case "pauselist":
		pausedJobs.Lock()
		defer pausedJobs.Unlock()
		if len(pausedJobs.jobs) == 0 {
			r.Say("There are no paused jobs")
			return
		}
		jl := make([]string, 0, len(pausedJobs.jobs))
		for job := range pausedJobs.jobs {
			jl = append(jl, job)
		}
		sort.Strings(jl)
		r.Say("These jobs are paused: %s", strings.Join(jl, ", "))
	case "chanlog":
		lchan := r.Channel
		if len(args) > 0 && len(args[0]) > 0 {
			lchan = args[0]
		}
		if len(lchan) == 0 {
			lchan = "dm"
		}
		fname := lchan + "-channel.log"
		cfile, err := os.Create(fname)
		if err != nil {
			r.Say("Sorry, there was a problem creating the log file")
			Log(robot.Error, "Creating '%s': %v", fname, err)
			return
		}
		clog := log.New(cfile, "", log.LstdFlags)
		chanLoggers.Lock()
		chanLoggers.channels[lchan] = clog
		chanLoggers.Unlock()
		r.Say("Ok, I'll start logging all messages in channel '%s' to '%s'", lchan, fname)
	case "stopchanlog":
		chanLoggers.Lock()
		chanLoggers.channels = make(map[string]*log.Logger)
		chanLoggers.Unlock()
		r.Say("Ok, I've stopped all channel logs")
	case "quit", "restart":
		state.Lock()
		if state.shuttingDown {
			state.Unlock()
			Log(robot.Warn, "Received administrator `quit` while shutdown in progress")
			return
		}
		state.shuttingDown = true
		restart := command == "restart"
		if restart {
			state.restart = true
		}
		proto := r.cfg.protocol
		// NOTE: THIS plugin is definitely running, but will end soon!
		if state.pipelinesRunning > 1 {
			runningCount := state.pipelinesRunning - 1
			state.Unlock()
			if proto != "test" {
				r.Say("There are still %d pipelines running; I'll %s when they all complete, or you can issue an \"abort\" command", runningCount, command)
			}
		} else {
			state.Unlock()
			if proto != "test" {
				if restart {
					r.Reply(r.RandomString(rightback))
				} else {
					r.Reply(r.RandomString(byebye))
				}
				// How long does it _actually_ take for the message to go out?
				time.Sleep(time.Second)
			}
		}
		Log(robot.Info, "Exiting on administrator 'quit|restart' command")
		go stop()
	}
	return
}
