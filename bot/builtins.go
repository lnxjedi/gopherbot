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
	"github.com/lnxjedi/gopherbot/robot/util"
	"golang.org/x/sys/unix"
	"gopkg.in/yaml.v3"
)

// Cut off for listing channels after help text
const tooManyChannels = 4

func init() {
	robot.RegisterPlugin("builtin-fallback", robot.PluginHandler{Handler: fallback})
	robot.RegisterPlugin("builtin-help", robot.PluginHandler{Handler: help})
	robot.RegisterPlugin("builtin-admin", robot.PluginHandler{Handler: admin})
	robot.RegisterPlugin("builtin-logging", robot.PluginHandler{Handler: logging})
}

func defaultHelp() []string {
	return []string{
		"(alias) help <keyword> - get help for the provided <keyword>",
		"(alias) help <keyword> brief - compact help for a likely command",
		"(alias) commands - browse plugins and command groups available in this channel",
		"(alias) help-all - help for all commands available in this channel, including global commands",
	}
}

/* builtin plugins, like help */

func fallback(m robot.Robot, command string, args ...string) (retval robot.TaskRetVal) {
	r := m.(Robot)
	if command == "init" {
		return // ignore init
	}
	if command == "catchall" {
		term := ""
		if len(args) > 0 {
			term = strings.TrimSpace(args[0])
		}
		reply := strings.TrimSpace(r.collectFallbackAdvice(term).DeterministicReply)
		if reply == "" {
			reply = "I couldn't match that command."
		}
		if msg := r.GetMessage(); msg != nil && len(strings.TrimSpace(msg.Channel)) == 0 {
			r.MessageFormat(robot.BasicMarkdown).Say(reply)
		} else {
			r.MessageFormat(robot.BasicMarkdown).SayThread(reply)
		}
	}
	return
}

var botRegex = regexp.MustCompile(`^([^(]*)\(bot\)(,?) *`)
var aliasRegex = regexp.MustCompile(`^\(alias\) *`)
var helpTokenRegex = regexp.MustCompile(`[A-Za-z0-9_-]+`)
var helpAddressPrefixRegex = regexp.MustCompile(`^\s*/?\((?:alias|bot)\)(?:[,:])?\s*`)

func aliasString(alias rune) string {
	if alias == 0 {
		return ""
	}
	return string(alias)
}

func (r Robot) expandHelpPlaceholders(input string) (ret string) {
	w := getLockedWorker(r.tid)
	w.Unlock()
	ret = input
	botName := r.cfg.botinfo.UserName
	botAlias := aliasString(r.cfg.alias)
	if len(botName) == 0 && len(botAlias) == 0 {
		ret = input
	} else {
		if botRegex.MatchString(input) {
			if len(botName) > 0 {
				ret = botRegex.ReplaceAllString(input, "${1}"+botName+"${2} ")
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
	return ret
}

func formatBasicMarkdownInlineCode(input string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return ""
	}
	return "`" + strings.ReplaceAll(trimmed, "`", "\\`") + "`"
}

func (r Robot) formatInlineLiteral(input string) string {
	return formatBasicMarkdownInlineCode(r.expandHelpPlaceholders(input))
}

func stripHelpAddressPrefix(input string) string {
	trimmed := strings.TrimSpace(input)
	if len(trimmed) == 0 {
		return trimmed
	}
	stripped := strings.TrimSpace(helpAddressPrefixRegex.ReplaceAllString(trimmed, ""))
	if len(stripped) == 0 {
		return trimmed
	}
	return stripped
}

func helpSurfaceCommandText(input, alias, botName string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return ""
	}
	trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "/"))
	trimmed = stripHelpAddressPrefix(trimmed)
	if trimmed == "" {
		return ""
	}
	if alias != "" && strings.HasPrefix(trimmed, alias) {
		rest := strings.TrimSpace(trimmed[len(alias):])
		if rest != "" {
			return rest
		}
	}
	botCandidates := []string{}
	botName = strings.TrimSpace(botName)
	if botName != "" {
		botCandidates = append(botCandidates, botName)
		fields := strings.Fields(botName)
		if len(fields) > 0 && !strings.EqualFold(fields[0], botName) {
			botCandidates = append(botCandidates, fields[0])
		}
	}
	for _, candidate := range botCandidates {
		lowerTrimmed := strings.ToLower(trimmed)
		lowerCandidate := strings.ToLower(candidate)
		if !strings.HasPrefix(lowerTrimmed, lowerCandidate) {
			continue
		}
		if len(trimmed) == len(candidate) {
			return ""
		}
		next := trimmed[len(candidate)]
		if next != ' ' && next != ',' && next != ':' {
			continue
		}
		rest := strings.TrimSpace(trimmed[len(candidate):])
		rest = strings.TrimLeft(rest, ",: ")
		if rest != "" {
			return rest
		}
	}
	return trimmed
}

func hiddenSlashBotCommand(botName, command string) string {
	name := strings.TrimSpace(strings.TrimPrefix(botName, "/"))
	if name == "" {
		return ""
	}
	fields := strings.Fields(name)
	if len(fields) > 0 {
		name = fields[0]
	}
	command = strings.TrimSpace(command)
	name = strings.ToLower(name)
	if command == "" {
		return "/" + name
	}
	return "/" + name + " " + command
}

type helpCommandMetadata struct {
	PluginName       string
	Command          string
	SimpleMatcher    string
	Usage            string
	Summary          string
	Examples         []string
	PrivateExamples  []string
	Keywords         []string
	Scope            string
	Channels         []string
	AllChannels      bool
	PrivateOK        bool
	PrivateRequired  bool
	PrivateSupported bool
	PrivateHint      string
	PluginSummary    string
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

func normalizeHelpPhrase(input string) string {
	normalized := strings.ToLower(strings.TrimSpace(input))
	normalized = strings.ReplaceAll(normalized, "-", " ")
	normalized = strings.Join(strings.Fields(normalized), " ")
	return normalized
}

func collectHelpSearchTokens(entry helpCommandMetadata) []string {
	haystack := strings.Join([]string{
		entry.PluginName,
		entry.Command,
		strings.Join(entry.Keywords, " "),
		entry.Usage,
		entry.Summary,
		strings.Join(entry.Examples, " "),
	}, " ")
	parsed := helpTokenRegex.FindAllString(normalizeHelpPhrase(haystack), -1)
	unique := make([]string, 0, len(parsed))
	seen := make(map[string]struct{}, len(parsed))
	for _, token := range parsed {
		if len(token) == 0 {
			continue
		}
		if _, ok := seen[token]; ok {
			continue
		}
		seen[token] = struct{}{}
		unique = append(unique, token)
	}
	return unique
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

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
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
	if len(task.Channels) > 0 {
		if len(task.Channels) > tooManyChannels {
			return "channels: (many)"
		}
		return "channels: " + strings.Join(task.Channels, ", ")
	}
	if task.AllChannels {
		return "all channels"
	}
	return "channel scoped"
}

func commandAllowsPrivate(plugin *Plugin, command string) bool {
	if plugin == nil {
		return false
	}
	if commandRequiresPrivate(plugin, command) {
		return true
	}
	command = strings.TrimSpace(strings.ToLower(command))
	for _, allowed := range plugin.AllowedPrivateCommands {
		key := strings.TrimSpace(strings.ToLower(allowed))
		if key == "*" || key == command {
			return true
		}
	}
	return false
}

func commandRequiresPrivate(plugin *Plugin, command string) bool {
	if plugin == nil {
		return false
	}
	if plugin.RequireAllCommandsPrivate {
		return true
	}
	command = strings.TrimSpace(strings.ToLower(command))
	for _, required := range plugin.RequiredPrivateCommands {
		if strings.TrimSpace(strings.ToLower(required)) == command {
			return true
		}
	}
	return false
}

func helpPluginSummary(task *Task, summaries ...string) string {
	if task != nil {
		if summary := strings.TrimSpace(task.Description); summary != "" {
			return summary
		}
	}
	for _, summary := range summaries {
		if trimmed := strings.TrimSpace(summary); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func (r Robot) collectHelpCommandMetadata(includeGlobal bool) []helpCommandMetadata {
	w := getLockedWorker(r.tid)
	w.Unlock()

	type authorizerGroupLookup struct {
		groups map[string]struct{}
		known  bool
	}
	groupCache := make(map[string]authorizerGroupLookup)
	protocol := protocolFromIncoming(r.Incoming, r.Protocol)
	hiddenSupported := hiddenCommandsSupportedForProtocol(protocol)
	hiddenHint := hiddenCommandHintForProtocol(protocol)

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
			if commandRequiresAuthorization(plugin, command) && strings.TrimSpace(task.AuthRequire) != "" {
				authorizer := effectiveAuthorizerName(task, w.cfg.defaultAuthorizer)
				cached, ok := groupCache[authorizer]
				if !ok {
					groups, known := r.getAuthorizerUserGroups(w, authorizer, w.User)
					cached = authorizerGroupLookup{groups: groups, known: known}
					groupCache[authorizer] = cached
				}
				if cached.known && !userHasRequiredGroup(cached.groups, task.AuthRequire) {
					continue
				}
			}
			key := task.name + "|" + command
			entry, ok := byCommand[key]
			if !ok {
				entry = &helpCommandMetadata{
					PluginName:    task.name,
					Command:       command,
					Scope:         helpScopeText(task),
					Channels:      append([]string(nil), task.Channels...),
					AllChannels:   task.AllChannels,
					PluginSummary: helpPluginSummary(task),
				}
				byCommand[key] = entry
			}
			if len(entry.Usage) == 0 && len(strings.TrimSpace(matcher.Usage)) > 0 {
				entry.Usage = strings.TrimSpace(matcher.Usage)
			}
			if len(entry.SimpleMatcher) == 0 && len(strings.TrimSpace(matcher.SimpleMatcher)) > 0 {
				entry.SimpleMatcher = strings.TrimSpace(matcher.SimpleMatcher)
			}
			if len(entry.Summary) == 0 && len(strings.TrimSpace(matcher.Summary)) > 0 {
				entry.Summary = strings.TrimSpace(matcher.Summary)
			}
			if entry.PluginSummary == "" {
				entry.PluginSummary = helpPluginSummary(task, matcher.Summary)
			}
			if commandAllowsPrivate(plugin, command) {
				entry.PrivateOK = true
				entry.PrivateRequired = commandRequiresPrivate(plugin, command)
				if hiddenSupported {
					entry.PrivateSupported = true
				}
				if entry.PrivateHint == "" && strings.TrimSpace(hiddenHint) != "" {
					entry.PrivateHint = strings.TrimSpace(hiddenHint)
				}
			}
			entry.Examples = appendUniqueStrings(entry.Examples, matcher.Examples...)
			if entry.PrivateOK && hiddenSupported {
				for _, example := range matcher.Examples {
					commandText := helpSurfaceCommandText(example, aliasString(w.cfg.alias), w.cfg.botinfo.UserName)
					hidden := strings.TrimSpace(formatHiddenCommand(protocol, commandText))
					if hidden == "" {
						continue
					}
					entry.PrivateExamples = appendUniqueStrings(entry.PrivateExamples, hidden)
				}
			}
			entry.Keywords = appendUniqueStrings(entry.Keywords, matcher.Keywords...)
		}
	}

	results := make([]helpCommandMetadata, 0, len(byCommand))
	for _, entry := range byCommand {
		if len(entry.Usage) == 0 {
			entry.Usage = "(alias) " + entry.Command
		}
		results = append(results, *entry)
	}
	return results
}

type parsedHelpQuery struct {
	Term       string
	Brief      bool
	PluginName string
	Command    string
	HasPath    bool
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

	commandTokens := helpTokenRegex.FindAllString(commandPhrase, -1)
	pluginTokens := helpTokenRegex.FindAllString(pluginPhrase, -1)

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

	searchTokens := collectHelpSearchTokens(entry)
	for _, token := range searchTokens {
		if helpTokenEquivalent(termPhrase, token) && score < 70 {
			score = 70
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
	examples := normalizeHelpPhrase(strings.Join(entry.Examples, " "))
	if strings.Contains(usage, termPhrase) && score < 65 {
		score = 65
	}
	if strings.Contains(summary, termPhrase) && score < 62 {
		score = 62
	}
	if strings.Contains(examples, termPhrase) && score < 74 {
		score = 74
	}

	for _, token := range commandTokens {
		if fuzzyHelpTokenMatch(termPhrase, token, 1) && score < 88 {
			score = 88
		}
	}
	for _, token := range pluginTokens {
		if fuzzyHelpTokenMatch(termPhrase, token, 1) && score < 84 {
			score = 84
		}
	}
	for _, keyword := range entry.Keywords {
		keyPhrase := normalizeHelpPhrase(keyword)
		if fuzzyHelpTokenMatch(termPhrase, keyPhrase, 1) && score < 82 {
			score = 82
		}
	}
	for _, token := range searchTokens {
		if fuzzyHelpTokenMatch(termPhrase, token, 1) && score < 78 {
			score = 78
		}
	}

	termTokens := helpTokenRegex.FindAllString(termPhrase, -1)
	if len(termTokens) == 0 {
		return score
	}
	if len(searchTokens) == 0 {
		return score
	}
	tokenSet := make(map[string]struct{}, len(searchTokens)*2)
	for _, token := range searchTokens {
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
	meaningfulTokens := fallbackMeaningfulTokens(termPhrase)
	if len(meaningfulTokens) > 0 {
		meaningfulHits := 0
		firstHit := false
		commandTokens := helpTokenRegex.FindAllString(commandPhrase, -1)
		commandFirstHit := false
		if len(commandTokens) > 0 {
			commandFirstHit = helpTokenEquivalent(meaningfulTokens[0], commandTokens[0]) || strings.HasPrefix(commandTokens[0], meaningfulTokens[0])
		}
		for _, token := range meaningfulTokens {
			if _, ok := tokenSet[token]; ok {
				meaningfulHits++
				if token == meaningfulTokens[0] {
					firstHit = true
				}
				continue
			}
			if _, ok := tokenSet[singularizeHelpToken(token)]; ok {
				meaningfulHits++
				if token == meaningfulTokens[0] {
					firstHit = true
				}
			}
		}
		if meaningfulHits > 0 && (firstHit || commandFirstHit || meaningfulHits == len(meaningfulTokens)) {
			candidate := 60 + meaningfulHits*4
			if firstHit {
				candidate += 8
			}
			if commandFirstHit {
				candidate += 8
			}
			if meaningfulHits == len(meaningfulTokens) {
				candidate += 4
			}
			if candidate > 89 {
				candidate = 89
			}
			if candidate > score {
				score = candidate
			}
		}
	}
	return score
}

func fuzzyHelpTokenMatch(term, candidate string, maxDistance int) bool {
	term = strings.TrimSpace(strings.ToLower(term))
	candidate = strings.TrimSpace(strings.ToLower(candidate))
	if term == "" || candidate == "" || helpTokenEquivalent(term, candidate) {
		return false
	}
	if strings.Contains(term, " ") || strings.Contains(candidate, " ") {
		return false
	}
	if len(term) < 4 || len(candidate) < 4 {
		return false
	}
	if absInt(len(term)-len(candidate)) > maxDistance {
		return false
	}
	_, ok := boundedLevenshtein(term, candidate, maxDistance)
	return ok
}

func boundedLevenshtein(left, right string, maxDistance int) (int, bool) {
	if left == right {
		return 0, true
	}
	if maxDistance < 0 {
		return 0, false
	}
	if absInt(len(left)-len(right)) > maxDistance {
		return 0, false
	}
	if len(left) == 0 {
		if len(right) <= maxDistance {
			return len(right), true
		}
		return 0, false
	}
	if len(right) == 0 {
		if len(left) <= maxDistance {
			return len(left), true
		}
		return 0, false
	}

	previous := make([]int, len(right)+1)
	current := make([]int, len(right)+1)
	for j := range previous {
		previous[j] = j
	}

	for i := 1; i <= len(left); i++ {
		current[0] = i
		rowMin := current[0]
		start := maxInt(1, i-maxDistance)
		end := minInt(len(right), i+maxDistance)
		for j := 1; j < start; j++ {
			current[j] = maxDistance + 1
		}
		for j := start; j <= end; j++ {
			cost := 0
			if left[i-1] != right[j-1] {
				cost = 1
			}
			insertCost := current[j-1] + 1
			deleteCost := previous[j] + 1
			replaceCost := previous[j-1] + cost
			value := minInt(insertCost, minInt(deleteCost, replaceCost))
			current[j] = value
			if value < rowMin {
				rowMin = value
			}
		}
		for j := end + 1; j <= len(right); j++ {
			current[j] = maxDistance + 1
		}
		if rowMin > maxDistance {
			return 0, false
		}
		previous, current = current, previous
	}

	if previous[len(right)] > maxDistance {
		return 0, false
	}
	return previous[len(right)], true
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

type fallbackQueryKind int

const (
	fallbackQueryIdentifier fallbackQueryKind = iota
	fallbackQueryPhrase
	fallbackQueryPath
)

func classifyFallbackQuery(term string) fallbackQueryKind {
	normalized := strings.TrimSpace(normalizeHelpPhrase(term))
	if normalized == "" {
		return fallbackQueryIdentifier
	}
	if strings.Contains(normalized, "/") && !strings.Contains(normalized, " ") {
		return fallbackQueryPath
	}
	if strings.Contains(normalized, " ") {
		return fallbackQueryPhrase
	}
	return fallbackQueryIdentifier
}

func helpCommandLooksPhraseShaped(entry helpCommandMetadata) bool {
	for _, sequence := range fallbackLiteralSequences(entry) {
		if len(sequence) > 1 {
			return true
		}
	}
	return false
}

var helpPlaceholderRegex = regexp.MustCompile(`<[^>]+>`)

func fallbackLiteralTokens(surface string) []string {
	normalized := strings.TrimSpace(surface)
	if normalized == "" {
		return nil
	}
	normalized = helpPlaceholderRegex.ReplaceAllString(normalized, " ")
	normalized = normalizeHelpPhrase(normalized)
	if normalized == "" {
		return nil
	}
	return helpTokenRegex.FindAllString(normalized, -1)
}

func fallbackLiteralSequencesFromSurface(surface string) [][]string {
	trimmed := strings.TrimSpace(stripHelpAddressPrefix(surface))
	if trimmed == "" {
		return nil
	}
	parts := strings.Split(trimmed, "|")
	sequences := make([][]string, 0, len(parts))
	for _, part := range parts {
		tokens := fallbackLiteralTokens(part)
		if len(tokens) == 0 {
			continue
		}
		sequences = append(sequences, tokens)
	}
	return sequences
}

func fallbackLiteralSequences(entry helpCommandMetadata) [][]string {
	sequences := make([][]string, 0, 8)
	appendSequence := func(tokens []string) {
		if len(tokens) == 0 {
			return
		}
		key := strings.Join(tokens, "\x00")
		for _, existing := range sequences {
			if strings.Join(existing, "\x00") == key {
				return
			}
		}
		sequences = append(sequences, append([]string(nil), tokens...))
	}

	if spec := strings.TrimSpace(entry.SimpleMatcher); spec != "" {
		if compiled, err := simpleMatcherLiteralSequences(spec); err == nil {
			for _, seq := range compiled {
				appendSequence(seq)
			}
		}
	}

	for _, surface := range append([]string{entry.Usage}, entry.Examples...) {
		for _, seq := range fallbackLiteralSequencesFromSurface(surface) {
			appendSequence(seq)
		}
	}

	if len(sequences) == 0 {
		appendSequence(fallbackLiteralTokens(entry.Command))
	}
	return sequences
}

func fallbackTokenMatches(query, candidate string) bool {
	if helpTokenEquivalent(query, candidate) {
		return true
	}
	return fuzzyHelpTokenMatch(query, candidate, 1)
}

func fallbackLiteralSequenceScore(queryTokens, sequence []string) int {
	if len(queryTokens) == 0 || len(sequence) == 0 {
		return 0
	}

	qIdx := 0
	hits := 0
	firstMatchIndex := -1
	for _, literal := range sequence {
		matched := false
		for qIdx < len(queryTokens) {
			if fallbackTokenMatches(queryTokens[qIdx], literal) {
				if firstMatchIndex == -1 {
					firstMatchIndex = qIdx
				}
				hits++
				qIdx++
				matched = true
				break
			}
			qIdx++
		}
		if !matched {
			break
		}
	}
	if hits == 0 {
		return 0
	}

	score := 35 + hits*12
	if hits == len(sequence) {
		score += 22
	}
	switch firstMatchIndex {
	case 0:
		score += 18
	case 1:
		score += 8
	}
	if len(queryTokens) == 1 && firstMatchIndex == 0 && hits == 1 {
		score += 18
	}
	if score > 96 {
		score = 96
	}
	return score
}

func fallbackCloseMatchScore(entry helpCommandMetadata, term string) int {
	normalized := normalizeHelpPhrase(term)
	if normalized == "" {
		return 0
	}
	kind := classifyFallbackQuery(normalized)
	queryTokens := helpTokenRegex.FindAllString(normalized, -1)
	if len(queryTokens) == 0 {
		return 0
	}

	bestSequenceScore := 0
	for _, sequence := range fallbackLiteralSequences(entry) {
		score := fallbackLiteralSequenceScore(queryTokens, sequence)
		if score > bestSequenceScore {
			bestSequenceScore = score
		}
	}

	phraseShaped := helpCommandLooksPhraseShaped(entry)
	base := scoreHelpCommandMatch(entry, normalized)

	switch kind {
	case fallbackQueryPath:
		return base
	case fallbackQueryIdentifier:
		if phraseShaped {
			return bestSequenceScore
		}
		return maxInt(bestSequenceScore, base)
	case fallbackQueryPhrase:
		if phraseShaped {
			return bestSequenceScore
		}
		if bestSequenceScore > 0 {
			return maxInt(bestSequenceScore, base)
		}
	}
	return bestSequenceScore
}

func (r Robot) formatInlineHelpCommand(input string) string {
	return r.formatInlineLiteral(input)
}

func (r Robot) formatSuggestedCommand(input string) string {
	alias := r.GetBotAttribute("alias").String()
	botName := r.GetBotAttribute("name").String()
	command := helpSurfaceCommandText(input, alias, botName)
	protocol := protocolFromIncoming(r.Incoming, r.Protocol)
	if command != "" && hiddenCommandsSupportedForProtocol(protocol) {
		if hidden := strings.TrimSpace(formatHiddenCommand(protocol, command)); hidden != "" {
			return hidden
		}
	}
	return strings.TrimSpace(r.expandHelpPlaceholders(input))
}

func (r Robot) formatInlineSuggestedCommand(input string) string {
	return "`" + r.formatSuggestedCommand(input) + "`"
}

func (r Robot) formatSuggestedHelpLine(input string) string {
	if strings.Contains(input, " - ") {
		parts := strings.SplitN(input, " - ", 2)
		return "- " + formatBasicMarkdownInlineCode(r.formatSuggestedCommand(parts[0])) + " - " + parts[1]
	}
	return "- " + formatBasicMarkdownInlineCode(r.formatSuggestedCommand(input))
}

func (r Robot) formatHelpExample(entry helpCommandMetadata, example string) string {
	line := strings.TrimSpace(example)
	if len(line) == 0 {
		return ""
	}
	return r.formatInlineLiteral(line)
}

func (r Robot) pluginHelpExample(entry helpCommandMetadata) string {
	for _, example := range entry.Examples {
		rendered := strings.TrimSpace(r.formatHelpExample(entry, example))
		if rendered != "" {
			return rendered
		}
	}
	if usage := strings.TrimSpace(stripHelpAddressPrefix(entry.Usage)); usage != "" {
		return formatBasicMarkdownInlineCode(usage)
	}
	return ""
}

func (r Robot) renderHelpEntry(entry helpCommandMetadata, includeExamples, includeScope bool, exampleLimit int) string {
	lines := make([]string, 0, 8)
	if len(entry.Usage) > 0 {
		lines = append(lines, "**Usage:** "+formatBasicMarkdownInlineCode(stripHelpAddressPrefix(entry.Usage)))
	}
	if len(entry.Summary) > 0 {
		lines = append(lines, "**Summary:** "+entry.Summary)
	}
	if includeExamples && len(entry.Examples) > 0 {
		examples := entry.Examples
		if exampleLimit > 0 && len(examples) > exampleLimit {
			examples = examples[:exampleLimit]
		}
		rendered := make([]string, 0, len(examples))
		for _, example := range examples {
			line := r.formatHelpExample(entry, example)
			if len(strings.TrimSpace(line)) == 0 {
				continue
			}
			rendered = append(rendered, line)
		}
		if len(rendered) > 0 {
			lines = append(lines, "**Examples:**")
			for _, example := range rendered {
				lines = append(lines, "- "+example)
			}
		}
		hiddenExamples := entry.PrivateExamples
		if exampleLimit > 0 && len(hiddenExamples) > exampleLimit {
			hiddenExamples = hiddenExamples[:exampleLimit]
		}
		renderedHidden := make([]string, 0, len(hiddenExamples))
		for _, example := range hiddenExamples {
			line := r.formatInlineLiteral(example)
			if len(strings.TrimSpace(line)) == 0 {
				continue
			}
			renderedHidden = append(renderedHidden, line)
		}
		if len(renderedHidden) > 0 {
			lines = append(lines, "**Private examples:**")
			for _, example := range renderedHidden {
				lines = append(lines, "- "+example)
			}
		} else if entry.PrivateSupported && strings.TrimSpace(entry.PrivateHint) != "" {
			lines = append(lines, "**Private:** "+r.expandHelpPlaceholders(entry.PrivateHint))
		}
	}
	if includeScope {
		if availability := strings.TrimSpace(formatExactHelpAvailability(entry)); availability != "" {
			lines = append(lines, "**Availability:** "+availability)
		}
	}
	return strings.Join(lines, "\n")
}

func formatExactHelpAvailability(entry helpCommandMetadata) string {
	if entry.PrivateRequired {
		return "private context only"
	}
	if len(entry.Channels) > 0 {
		channels := append([]string(nil), entry.Channels...)
		sort.Strings(channels)
		display := make([]string, 0, len(channels))
		for _, channel := range channels {
			display = append(display, "`#"+channel+"`")
		}
		return strings.Join(display, ", ")
	}
	if entry.AllChannels {
		return "all robot channels"
	}
	return entry.Scope
}

func (r Robot) renderHelpListingEntry(entry helpCommandMetadata, includeExamples, includeScope bool, exampleLimit int) string {
	lines := []string{fmt.Sprintf("**Command:** `%s/%s`", entry.PluginName, entry.Command)}
	if len(entry.Summary) > 0 {
		lines = append(lines, "**Summary:** "+entry.Summary)
	}
	if len(entry.Usage) > 0 {
		lines = append(lines, "**Usage:** "+formatBasicMarkdownInlineCode(stripHelpAddressPrefix(entry.Usage)))
	}
	if includeExamples && len(entry.Examples) > 0 {
		examples := entry.Examples
		if exampleLimit > 0 && len(examples) > exampleLimit {
			examples = examples[:exampleLimit]
		}
		rendered := make([]string, 0, len(examples))
		for _, example := range examples {
			line := r.formatHelpExample(entry, example)
			if len(strings.TrimSpace(line)) == 0 {
				continue
			}
			rendered = append(rendered, line)
		}
		if len(rendered) > 0 {
			lines = append(lines, "**Examples:**")
			for _, example := range rendered {
				lines = append(lines, "- "+example)
			}
		}
		hiddenExamples := entry.PrivateExamples
		if exampleLimit > 0 && len(hiddenExamples) > exampleLimit {
			hiddenExamples = hiddenExamples[:exampleLimit]
		}
		renderedHidden := make([]string, 0, len(hiddenExamples))
		for _, example := range hiddenExamples {
			line := r.formatInlineLiteral(example)
			if len(strings.TrimSpace(line)) == 0 {
				continue
			}
			renderedHidden = append(renderedHidden, line)
		}
		if len(renderedHidden) > 0 {
			lines = append(lines, "**Private examples:**")
			for _, example := range renderedHidden {
				lines = append(lines, "- "+example)
			}
		} else if entry.PrivateSupported && strings.TrimSpace(entry.PrivateHint) != "" {
			lines = append(lines, "**Private:** "+r.expandHelpPlaceholders(entry.PrivateHint))
		}
	}
	if includeScope {
		if availability := strings.TrimSpace(formatExactHelpAvailability(entry)); availability != "" {
			lines = append(lines, "**Availability:** "+availability)
		}
	}
	lines = append(lines, "**Exact help:** "+r.formatInlineSuggestedCommand("(alias) help "+entry.PluginName+"/"+entry.Command))
	return strings.Join(lines, "\n")
}

func parseHelpQuery(args []string) parsedHelpQuery {
	term, brief := parseHelpQueryMode(args)
	parsed := parsedHelpQuery{
		Term:  term,
		Brief: brief,
	}
	if strings.Contains(term, "/") {
		parts := strings.Split(term, "/")
		if len(parts) == 2 {
			plugin := strings.TrimSpace(strings.ToLower(parts[0]))
			command := strings.TrimSpace(strings.ToLower(parts[1]))
			if plugin != "" && command != "" && !strings.Contains(plugin, " ") && !strings.Contains(command, " ") {
				parsed.PluginName = plugin
				parsed.Command = command
				parsed.HasPath = true
			}
		}
	}
	return parsed
}

func parseHelpQueryMode(args []string) (term string, brief bool) {
	parts := make([]string, 0, len(args))
	for _, arg := range args {
		for _, piece := range strings.Fields(arg) {
			trimmed := strings.TrimSpace(piece)
			if trimmed == "" {
				continue
			}
			parts = append(parts, trimmed)
		}
	}
	if len(parts) == 0 {
		return "", false
	}
	if strings.EqualFold(parts[len(parts)-1], "brief") {
		brief = true
		parts = parts[:len(parts)-1]
	}
	return strings.TrimSpace(strings.Join(parts, " ")), brief
}

func groupHelpEntriesByPlugin(entries []helpCommandMetadata) map[string][]helpCommandMetadata {
	byPlugin := make(map[string][]helpCommandMetadata)
	for _, entry := range entries {
		byPlugin[entry.PluginName] = append(byPlugin[entry.PluginName], entry)
	}
	return byPlugin
}

func pluginOverviewSummary(entries []helpCommandMetadata) string {
	for _, entry := range entries {
		if summary := strings.TrimSpace(entry.PluginSummary); summary != "" {
			return summary
		}
	}
	if len(entries) == 1 {
		return strings.TrimSpace(entries[0].Summary)
	}
	for _, entry := range entries {
		if summary := strings.TrimSpace(entry.Summary); summary != "" {
			return summary
		}
	}
	return ""
}

func summarizePluginCommands(entries []helpCommandMetadata, limit int) string {
	if limit <= 0 {
		limit = 3
	}
	commands := make([]string, 0, len(entries))
	seen := map[string]struct{}{}
	for _, entry := range entries {
		if _, ok := seen[entry.Command]; ok {
			continue
		}
		seen[entry.Command] = struct{}{}
		commands = append(commands, entry.Command)
	}
	sort.Strings(commands)
	if len(commands) == 0 {
		return ""
	}
	if len(commands) <= limit {
		return strings.Join(commands, ", ")
	}
	return strings.Join(commands[:limit], ", ") + fmt.Sprintf(", +%d more", len(commands)-limit)
}

func summarizeQualifiedPluginCommands(plugin string, entries []helpCommandMetadata, limit int) string {
	if limit <= 0 {
		limit = 4
	}
	commands := make([]string, 0, len(entries))
	seen := map[string]struct{}{}
	for _, entry := range entries {
		if _, ok := seen[entry.Command]; ok {
			continue
		}
		seen[entry.Command] = struct{}{}
		commands = append(commands, entry.Command)
	}
	sort.Strings(commands)
	if len(commands) == 0 {
		return ""
	}
	if len(commands) < limit {
		limit = len(commands)
	}
	parts := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		command := commands[i]
		if i == 0 {
			parts = append(parts, "`"+plugin+"/"+command+"`")
		} else {
			parts = append(parts, "`/"+command+"`")
		}
	}
	line := strings.Join(parts, ", ")
	if len(commands) > limit {
		line += fmt.Sprintf(" ... (+%d more)", len(commands)-limit)
	}
	return line
}

func findHelpEntryByPath(entries []helpCommandMetadata, plugin, command string) (helpCommandMetadata, bool) {
	for _, entry := range entries {
		if strings.EqualFold(entry.PluginName, plugin) && strings.EqualFold(entry.Command, command) {
			return entry, true
		}
	}
	return helpCommandMetadata{}, false
}

func findHelpEntriesByPlugin(entries []helpCommandMetadata, plugin string) []helpCommandMetadata {
	matches := make([]helpCommandMetadata, 0, 4)
	for _, entry := range entries {
		if strings.EqualFold(entry.PluginName, plugin) {
			matches = append(matches, entry)
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Command < matches[j].Command
	})
	return matches
}

func (r Robot) renderCommandsOverview(entries []helpCommandMetadata) string {
	byPlugin := groupHelpEntriesByPlugin(entries)
	pluginNames := make([]string, 0, len(byPlugin))
	for plugin := range byPlugin {
		pluginNames = append(pluginNames, plugin)
	}
	sort.Strings(pluginNames)

	lines := []string{"**Plugins and command groups available in this channel**"}
	for _, plugin := range pluginNames {
		pluginEntries := byPlugin[plugin]
		sort.Slice(pluginEntries, func(i, j int) bool {
			return pluginEntries[i].Command < pluginEntries[j].Command
		})
		summary := pluginOverviewSummary(pluginEntries)
		if summary == "" {
			summary = "Commands available."
		}
		lines = append(lines, fmt.Sprintf("**%s**", plugin))
		lines = append(lines, summary)
		if preview := summarizeQualifiedPluginCommands(plugin, pluginEntries, 4); preview != "" {
			lines = append(lines, "**Commands:** "+preview)
		}
		lines = append(lines, "**Help:** "+r.formatInlineSuggestedCommand("(alias) help "+plugin)+" or "+r.formatInlineSuggestedCommand("(alias) help "+plugin+"/<command>"))
		lines = append(lines, "")
	}
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	lines = append(lines, "**Exact help:** "+r.formatInlineSuggestedCommand("(alias) help <plugin>/<command>"))
	lines = append(lines, "**Search by keyword:** "+r.formatInlineSuggestedCommand("(alias) help <plugin|command|keyword>"))
	return strings.Join(lines, "\n")
}

func (r Robot) renderPluginHelpOverview(plugin string, entries []helpCommandMetadata) string {
	if len(entries) == 0 {
		return "Sorry, I couldn't find that plugin."
	}
	lines := []string{fmt.Sprintf("**Plugin help:** `%s`", plugin)}
	if summary := pluginOverviewSummary(entries); summary != "" {
		lines = append(lines, summary)
	}
	lines = append(lines, "**Commands:**")
	for _, entry := range entries {
		line := "- `" + entry.PluginName + "/" + entry.Command + "`"
		if summary := strings.TrimSpace(entry.Summary); summary != "" {
			line += " - " + summary
		}
		lines = append(lines, line)
		if example := r.pluginHelpExample(entry); example != "" {
			lines = append(lines, "Example: "+example)
		}
	}
	if len(entries) == 1 {
		lines = append(lines, "**More detail:** "+r.formatInlineSuggestedCommand("(alias) help "+entries[0].PluginName+"/"+entries[0].Command))
	} else {
		lines = append(lines, "**More detail:** "+r.formatInlineSuggestedCommand("(alias) help "+plugin+"/<command>"))
	}
	return strings.Join(lines, "\n")
}

func helpAddressKey(entry helpCommandMetadata) string {
	return entry.PluginName + "/" + entry.Command
}

func renderKeywordPluginSection(r Robot, term string, pluginEntries []helpCommandMetadata, matches []rankedHelpMatch, brief bool) string {
	sections := []string{
		fmt.Sprintf("**Help for keyword:** `%s`", strings.ToLower(term)),
		r.renderPluginHelpOverview(term, pluginEntries),
	}

	pluginCommands := make(map[string]struct{}, len(pluginEntries))
	for _, entry := range pluginEntries {
		pluginCommands[helpAddressKey(entry)] = struct{}{}
	}

	filtered := make([]rankedHelpMatch, 0, len(matches))
	seen := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		key := helpAddressKey(match.Entry)
		if _, skip := pluginCommands[key]; skip {
			continue
		}
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		filtered = append(filtered, match)
	}
	if len(filtered) == 0 {
		return strings.Join(sections, "\n\n")
	}

	limit := 6
	if brief {
		limit = 3
	}
	if len(filtered) > limit {
		filtered = filtered[:limit]
	}
	otherLines := []string{"**Other command matches:**"}
	for _, match := range filtered {
		exampleLimit := 2
		if brief {
			exampleLimit = 1
		}
		otherLines = append(otherLines, r.renderHelpListingEntry(match.Entry, true, true, exampleLimit))
	}
	sections = append(sections, strings.Join(otherLines, "\n\n"))
	return strings.Join(sections, "\n\n")
}

func (r Robot) renderExactHelpEntry(entry helpCommandMetadata, siblingCount int) string {
	bodyLines := []string{r.renderHelpEntry(entry, true, false, 3)}
	if availability := strings.TrimSpace(formatExactHelpAvailability(entry)); availability != "" {
		bodyLines = append(bodyLines, "**Availability:** "+availability)
	}
	lines := []string{
		fmt.Sprintf("**Command help:** `%s/%s`", entry.PluginName, entry.Command),
		strings.Join(bodyLines, "\n"),
	}
	if siblingCount > 1 {
		lines = append(lines, "**More from this plugin:** "+r.formatInlineSuggestedCommand("(alias) help "+entry.PluginName))
	}
	return strings.Join(lines, "\n\n")
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
		ID, ok := getRuntimeBotIDForContext(r.Incoming, r.Protocol)
		if !ok {
			ID = ""
		}
		if len(ID) == 0 {
			ID = "(unknown)"
		}
		var alias string
		if aliasCh == 0 {
			alias = "(not set)"
		} else {
			alias = string(aliasCh)
		}
		channelID, _ := util.ExtractID(r.ProtocolChannel)
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
		gitSnapshot := getRuntimeGitSnapshot()
		if refreshed, err := refreshRuntimeGitStateFromConfig(false); err == nil {
			gitSnapshot = refreshed
			persistRuntimeGitSnapshotToEnv(gitSnapshot)
		}
		msg = append(msg, runtimeGitSummaryLine(gitSnapshot))
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
		query := parseHelpQuery(args)
		term := query.Term
		brief := query.Brief
		hasKeyword := command == "help" && len(term) > 0
		sendOutput := func(message string) {
			if r.Incoming.ThreadedMessage {
				r.MessageFormat(robot.BasicMarkdown).Reply(message)
			} else {
				r.MessageFormat(robot.BasicMarkdown).SayThread(message)
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
			lines = append(lines, "**Quick help**")
			for _, line := range defaultHelpLines {
				lines = append(lines, r.formatSuggestedHelpLine(line))
			}
			lines = append(lines, "")
			lines = append(lines, "**Plugin help:** "+r.formatInlineSuggestedCommand("(alias) help <plugin>"))
			lines = append(lines, "**Exact command help:** "+r.formatInlineSuggestedCommand("(alias) help <plugin>/<command>"))
			lines = append(lines, "**Browse this channel:** "+r.formatInlineSuggestedCommand("(alias) commands"))
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
				helpLines = append(helpLines, r.renderHelpListingEntry(entry, true, true, 0))
			}
			sendOutput("**Commands available in this channel (including global)**\n\n" + strings.Join(helpLines, lineSeparator))
		case "help":
			entries := r.collectHelpCommandMetadata(true)
			if query.HasPath {
				entry, ok := findHelpEntryByPath(entries, query.PluginName, query.Command)
				if !ok {
					sendOutput("Sorry, I couldn't find that command. Try " + r.formatInlineSuggestedCommand("(alias) help "+query.PluginName) + " or " + r.formatInlineSuggestedCommand("(alias) commands") + ".")
					return
				}
				siblings := findHelpEntriesByPlugin(entries, query.PluginName)
				sendOutput(r.renderExactHelpEntry(entry, len(siblings)))
				return
			}
			if pluginEntries := findHelpEntriesByPlugin(entries, term); len(pluginEntries) > 0 {
				matches := rankHelpMatches(entries, term)
				sendOutput(renderKeywordPluginSection(r, term, pluginEntries, matches, brief))
				return
			}
			matches := rankHelpMatches(entries, term)
			if len(matches) == 0 {
				sendOutput("Sorry, I didn't find any commands matching your keyword. Try " + r.formatInlineSuggestedCommand("(alias) commands") + " or " + r.formatInlineSuggestedCommand("(alias) help <plugin>/<command>") + ".")
				return
			}

			limit := 12
			if brief {
				limit = 3
			}
			display := matches
			if len(display) > limit {
				display = display[:limit]
			}
			helpLines := make([]string, 0, len(display))
			for _, match := range display {
				exampleLimit := 2
				if brief {
					exampleLimit = 1
				}
				helpLines = append(helpLines, r.renderHelpListingEntry(match.Entry, true, true, exampleLimit))
			}
			header := fmt.Sprintf("**Command matches for keyword:** `%s`", strings.ToLower(term))
			if brief {
				header = fmt.Sprintf("**Brief help for keyword:** `%s`", strings.ToLower(term))
			}
			if len(matches) > len(display) {
				header = fmt.Sprintf("%s (showing top %d of %d)", header, len(display), len(matches))
			}
			body := header + "\n" + strings.Join(helpLines, lineSeparator)
			if len(matches) > len(display) {
				seeAlso := make([]string, 0, len(matches)-len(display))
				for _, match := range matches[len(display):] {
					seeAlso = append(seeAlso, "`"+match.Entry.PluginName+"/"+match.Entry.Command+"`")
				}
				body += lineSeparator + "**Optionally see also:** " + strings.Join(seeAlso, ", ")
				body += "\n" + "**Specific help:** " + r.formatInlineSuggestedCommand("(alias) help <plugin>/<command>")
			}
			sendOutput(body)
		}
	}
	return
}

func adminDumpRobot(r Robot) {
	confLock.RLock()
	c, _ := yaml.Marshal(config)
	confLock.RUnlock()
	r.Fixed().Say("Here's how I've been configured, irrespective of interactive changes:\n%s", c)
}

func adminDumpPluginDefault(r Robot, args []string) {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		r.Say("Usage: dump plugin default <plugin>")
		return
	}
	plugName := strings.TrimSpace(args[0])
	found := false
	for _, t := range r.tasks.t[1:] {
		task, plugin, _ := getTask(t)
		if plugName == task.name {
			if plugin == nil {
				r.Say("No default configuration available for task type 'job'")
				return
			}
			if plugin.taskType == taskExternal {
				found = true
				if cfg, err := getDefCfg(t); err == nil {
					r.Fixed().Say("Here's the default configuration for \"%s\":\n%s", plugName, *cfg)
				} else {
					r.Say("I had a problem looking that up - somebody should check my logs")
				}
			}
		}
	}
	if !found {
		r.Say("Didn't find a plugin named " + plugName)
	}
}

func adminDumpPlugin(r Robot, args []string) {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		r.Say("Usage: dump plugin <plugin>")
		return
	}
	plugName := strings.TrimSpace(args[0])
	found := false
	for _, t := range r.tasks.t[1:] {
		task, plugin, _ := getTask(t)
		if plugName == task.name {
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
		r.Say("Didn't find a plugin named " + plugName)
	}
}

func adminListPlugins(r Robot, args []string) {
	joiner := ", "
	message := "Here are the plugins I have configured:\n%s"
	wantDisabled := false
	if len(args) > 0 && len(args[0]) > 0 {
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
	} else {
		r.Say("There are no disabled plugins")
	}
}

func handleAdminInspectCommand(r Robot, command string, args []string) bool {
	switch command {
	case "dumprobot":
		adminDumpRobot(r)
	case "dumpplugdefault":
		adminDumpPluginDefault(r, args)
	case "dumpplugin":
		adminDumpPlugin(r, args)
	case "listplugins":
		adminListPlugins(r, args)
	default:
		return false
	}
	return true
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

type psEntry struct {
	id       int
	pid      string
	class    string
	pipeName string
	taskName string
	command  string
	args     string
	started  string
	age      string
	user     string
	source   string
	parent   string
	isJob    bool
}

type psEntries []psEntry

func psVerboseRequested(args []string) bool {
	for _, arg := range args {
		switch strings.ToLower(strings.TrimSpace(arg)) {
		case "-v", "--verbose", "verbose":
			return true
		}
	}
	return false
}

func psAllowedInContext(incoming *robot.ConnectorMessage) bool {
	return privateCommandContext(incoming)
}

func psDisplayValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}
	return value
}

func psSourceLabel(ptype pipelineType, hasParent bool) string {
	if hasParent {
		return "spawn"
	}
	switch ptype {
	case scheduled:
		return "sched"
	case initJob:
		return "init"
	case jobTrigger:
		return "trigger"
	case jobCommand:
		return "run"
	case spawnedTask:
		return "spawn"
	default:
		return psDisplayValue(ptype.String())
	}
}

func activePipelineByWID(raw string) (*worker, int, error) {
	widx, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 0)
	if err != nil {
		return nil, 0, err
	}
	activePipelines.Lock()
	worker, ok := activePipelines.i[int(widx)]
	activePipelines.Unlock()
	if !ok {
		return nil, int(widx), fmt.Errorf("not found")
	}
	return worker, int(widx), nil
}

func (p psEntries) Len() int {
	return len(p)
}

func (p psEntries) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p psEntries) Less(i, j int) bool {
	return p[i].id < p[j].id
}

func appendPsSection(lines []string, title string, entries psEntries, verbose bool) []string {
	if len(entries) == 0 {
		return lines
	}
	if len(lines) > 0 {
		lines = append(lines, "")
	}
	lines = append(lines, title)
	if title == "Plugins" {
		if verbose {
			lines = append(lines, "WID    AGE      STARTED         USER           PLUGIN          CMD          TASK            CLASS OSPID FROM   ARGS")
			for _, e := range entries {
				lines = append(lines, fmt.Sprintf("%-6d %-8.8s %-15.15s %-14.14s %-15.15s %-12.12s %-15.15s %-5.5s %-5.5s %-6.6s %s", e.id, e.age, e.started, e.user, e.pipeName, e.command, e.taskName, e.class, e.pid, e.parent, e.args))
			}
		} else {
			lines = append(lines, "WID    AGE      USER           PLUGIN          CMD          TASK            ARGS")
			for _, e := range entries {
				lines = append(lines, fmt.Sprintf("%-6d %-8.8s %-14.14s %-15.15s %-12.12s %-15.15s %s", e.id, e.age, e.user, e.pipeName, e.command, e.taskName, e.args))
			}
		}
		return lines
	}
	if verbose {
		lines = append(lines, "WID    AGE      STARTED         JOB             TASK            SOURCE   FROM   USER           CLASS OSPID ARGS")
		for _, e := range entries {
			lines = append(lines, fmt.Sprintf("%-6d %-8.8s %-15.15s %-15.15s %-15.15s %-8.8s %-6.6s %-14.14s %-5.5s %-5.5s %s", e.id, e.age, e.started, e.pipeName, e.taskName, e.source, e.parent, e.user, e.class, e.pid, e.args))
		}
	} else {
		lines = append(lines, "WID    AGE      JOB             TASK            SOURCE   FROM   USER           ARGS")
		for _, e := range entries {
			lines = append(lines, fmt.Sprintf("%-6d %-8.8s %-15.15s %-15.15s %-8.8s %-6.6s %-14.14s %s", e.id, e.age, e.pipeName, e.taskName, e.source, e.parent, e.user, e.args))
		}
	}
	return lines
}

func formatReloadOutcome(r Robot, reloadErr error) string {
	op := strings.ToLower(strings.TrimSpace(r.GetParameter("GIT_OPERATION")))
	targetBranch := strings.TrimSpace(r.GetParameter("GIT_TARGET_BRANCH"))
	activeBranch := strings.TrimSpace(r.GetParameter("GOPHER_CUSTOM_BRANCH"))
	if activeBranch != "" {
		targetBranch = activeBranch
	}
	var status string
	switch op {
	case "update":
		if reloadErr == nil {
			status = "Reload completed successfully after git update."
		} else {
			status = "Reload failed after git update; continuing with previous configuration."
		}
	case "switch":
		if targetBranch != "" {
			if reloadErr == nil {
				status = fmt.Sprintf("Reload completed successfully after switching to branch '%s'.", targetBranch)
			} else {
				status = fmt.Sprintf("Reload failed after switching to branch '%s'; continuing with previous configuration.", targetBranch)
			}
		} else if reloadErr == nil {
			status = "Reload completed successfully after git branch switch."
		} else {
			status = "Reload failed after git branch switch; continuing with previous configuration."
		}
	default:
		if reloadErr == nil {
			status = "Configuration reload completed successfully."
		} else {
			status = "Configuration reload failed; continuing with previous configuration."
		}
	}
	if reloadErr == nil {
		return status
	}
	msg := strings.Join(strings.Fields(reloadErr.Error()), " ")
	if msg == "" {
		return status
	}
	if len(msg) > 320 {
		msg = msg[:320] + "..."
	}
	return status + " Error: " + msg
}

func admin(m robot.Robot, command string, args ...string) (retval robot.TaskRetVal) {
	if command == "init" {
		return // ignore init
	}
	r := m.(Robot)
	w := getLockedWorker(r.tid)
	w.Unlock()
	if handleAdminInspectCommand(r, command, args) {
		return
	}
	switch command {
	case "update":
		if ret := r.AddJob("go-update"); ret != robot.Ok {
			r.Say("Unable to start go-update: %s", ret)
			return
		}
		r.Say("Ok, I'll run go-update to pull configuration changes and reload.")
	case "defaultbranch":
		snapshot := getRuntimeGitSnapshot()
		if refreshed, err := refreshRuntimeGitStateFromConfig(false); err == nil {
			snapshot = refreshed
			persistRuntimeGitSnapshotToEnv(snapshot)
		}
		defaultBranch := strings.TrimSpace(snapshot.DefaultBranch)
		if defaultBranch == "" {
			r.Say("I don't currently know the repository default branch from local metadata; run git-info and verify repository status.")
			return
		}
		if ret := r.AddJob("go-switchbranch", defaultBranch); ret != robot.Ok {
			r.Say("Unable to start go-switchbranch for default branch '%s': %s", defaultBranch, ret)
			return
		}
		r.Say("Ok, I'll switch to default branch '%s', pull latest changes, and reload.", defaultBranch)
	case "branch":
		if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
			r.Say("Usage: switch-branch <branch>")
			return
		}
		branch := strings.TrimSpace(args[0])
		if ret := r.AddJob("go-switchbranch", branch); ret != robot.Ok {
			r.Say("Unable to start go-switchbranch for '%s': %s", branch, ret)
			return
		}
		r.Say("Ok, I'll switch to branch '%s', pull latest changes, and reload.", branch)
	case "reload":
		err := loadConfig(false)
		if err != nil {
			r.Reply("Error encountered during reload:")
			r.Fixed().Say("%v", err)
			Log(robot.Error, "Reloading configuration, requested by %s: %v", r.User, err)
			status := formatReloadOutcome(r, err)
			if attempted, ret := notifyPipelineStartContext(r, status); attempted && ret != robot.Ok {
				Log(robot.Warn, "Unable to send reload-failed origin notification for user '%s': %s", r.User, ret)
			}
			return
		}
		r.Reply("Configuration reloaded successfully")
		w.Log(robot.Info, "Configuration successfully reloaded by a request from: %s", r.User)
		status := formatReloadOutcome(r, nil)
		if attempted, ret := notifyPipelineStartContext(r, status); attempted && ret != robot.Ok {
			Log(robot.Warn, "Unable to send reload-success origin notification for user '%s': %s", r.User, ret)
		}
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
	case "gitinfo":
		snapshot := getRuntimeGitSnapshot()
		if refreshed, err := refreshRuntimeGitStateFromConfig(false); err == nil {
			snapshot = refreshed
			persistRuntimeGitSnapshotToEnv(snapshot)
		} else {
			w.Log(robot.Debug, "Unable to refresh runtime git state from info command: %v", err)
		}
		r.Say(strings.Join(runtimeGitDetailLines(snapshot), "\n"))
	case "validateuser":
		if !r.Incoming.ValidatedUser {
			r.Say("This command requires a validated administrator account.")
			return
		}
		if !privateCommandContext(r.Incoming) {
			r.Say("This command is only available in a private context.")
			return
		}
		if len(args) == 0 {
			r.Say("Usage: validate user <username>")
			return
		}
		userName := strings.ToLower(strings.TrimSpace(args[0]))
		if !isValidRosterUserName(userName) {
			r.Say("Usernames must be lower-case roster usernames.")
			return
		}
		if r.maps == nil {
			r.Say("I couldn't access the user directory.")
			return
		}
		if _, ok := r.maps.user[userName]; !ok {
			r.Say("I don't know a roster user named '%s'.", userName)
			return
		}
		code, err := issueUserValidationRequest(userName, r.User, protocolFromIncoming(r.Incoming, r.Protocol), time.Now())
		if err != nil {
			Log(robot.Error, "Issuing user validation request for '%s' by '%s': %v", userName, r.User, err)
			r.Say("I couldn't issue a validation code right now.")
			return
		}
		r.Reply("Validation code for '%s': %s (expires in about 42 seconds); instruct %s to send this exact code to me from the account you want to validate, using a private app message to the robot", userName, code, userName)
	case "encryptsecret":
		if len(args) == 0 || args[0] == "" {
			r.Say("Usage: encrypt-secret <secret>")
			return
		}
		encrypted, err := encryptPlaintextBase64(args[0])
		if err != nil {
			w.Log(robot.Error, "encrypt-secret admin command failed: %v", err)
			r.Say("Error: %v", err)
			return
		}
		r.MessageFormat(robot.Raw).Say("%s", encrypted)
	case "generateuuid":
		plain, encrypted, err := generateEncryptedUUID()
		if err != nil {
			w.Log(robot.Error, "generate-uuid admin command failed: %v", err)
			r.Say("Error: %v", err)
			return
		}
		r.MessageFormat(robot.Raw).Say("UUID: %s\nEncrypted: %s", plain, encrypted)
	case "abort":
		buf := make([]byte, 32768)
		runtime.Stack(buf, true)
		log.Printf("%s", buf)
		time.Sleep(2 * time.Second)
		panic("Abort command issued")
	case "ps":
		if !psAllowedInContext(r.Incoming) {
			r.Say("This command is only available in a private context.")
			return
		}
		verbose := psVerboseRequested(args)
		var pluginEntries psEntries
		var jobEntries psEntries
		activePipelines.Lock()
		for widx, worker := range activePipelines.i {
			worker.Lock()
			pipename := worker.pipeName
			command := worker.plugCommand
			if pipename == "builtin-admin" && command == "ps" {
				worker.Unlock()
				continue
			}
			parent := "-"
			hasParent := worker._parent != nil
			if worker._parent != nil {
				parent = strconv.Itoa(worker._parent.id)
			}
			pid := "-"
			if worker.osCmd != nil {
				pid = strconv.Itoa(worker.osCmd.Process.Pid)
			}
			entry := psEntry{
				id:       widx,
				pid:      pid,
				class:    psDisplayValue(worker.taskClass),
				pipeName: psDisplayValue(pipename),
				taskName: psDisplayValue(worker.taskName),
				command:  psDisplayValue(command),
				args:     strings.Join(worker.taskArgs, " "),
				started:  formatPipelineClock(worker.startedAt, worker.timeZone),
				age:      formatPipelineAge(time.Since(worker.startedAt)),
				user:     psDisplayValue(worker.User),
				source:   psSourceLabel(worker.ptype, hasParent),
				parent:   parent,
				isJob:    worker.jobName != "",
			}
			worker.Unlock()
			if entry.isJob {
				jobEntries = append(jobEntries, entry)
			} else {
				pluginEntries = append(pluginEntries, entry)
			}
		}
		activePipelines.Unlock()
		if len(pluginEntries) == 0 && len(jobEntries) == 0 {
			r.Say("No pipelines running")
			return
		}
		sort.Sort(pluginEntries)
		sort.Sort(jobEntries)
		lines := []string{}
		lines = appendPsSection(lines, "Plugins", pluginEntries, verbose)
		lines = appendPsSection(lines, "Jobs", jobEntries, verbose)
		if !verbose {
			lines = append(lines, "", "(use 'ps -v' for more verbose output)")
		}
		r.Fixed().Say(strings.Join(lines, "\n"))
	case "getpipelinelog":
		if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
			r.Say("Usage: get-pipeline-log <id>")
			return
		}
		worker, widx, err := activePipelineByWID(args[0])
		if err != nil {
			if strings.Contains(err.Error(), "invalid syntax") {
				r.Say("Couldn't convert '%s' to an int", args[0])
				return
			}
			r.Say("Pipeline %d not found", widx)
			return
		}
		snapshot := strings.TrimSpace(worker.liveLogSnapshot())
		if snapshot == "" {
			r.Say("No live log buffered for pipeline %d", widx)
			return
		}
		r.Fixed().Say("Live log for pipeline %d:\n%s", widx, snapshot)
	case "kill":
		if len(args) == 0 {
			r.Say("Usage: kill <id>")
			return
		}
		wid := args[0]
		worker, widx, err := activePipelineByWID(wid)
		if err != nil {
			if strings.Contains(err.Error(), "invalid syntax") {
				r.Say("Couldn't convert '%s' to an int", wid)
				return
			}
			r.Say("Pipeline %d not found", widx)
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
		if len(args) == 0 {
			r.Say("Usage: pause <job>")
			return
		}
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
		if len(args) == 0 {
			r.Say("Usage: resume <job>")
			return
		}
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
