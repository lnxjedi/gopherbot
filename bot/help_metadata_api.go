package bot

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/lnxjedi/gopherbot/robot"
)

type helpMetadataContext struct {
	BotName                  string `json:"bot_name,omitempty"`
	BotAlias                 string `json:"bot_alias,omitempty"`
	User                     string `json:"user,omitempty"`
	Channel                  string `json:"channel,omitempty"`
	CommandMode              string `json:"command_mode,omitempty"`
	Direct                   bool   `json:"direct"`
	Threaded                 bool   `json:"threaded"`
	Protocol                 string `json:"protocol,omitempty"`
	PrivateCommandsSupported bool   `json:"private_commands_supported,omitempty"`
	PrivateCommandHint       string `json:"private_command_hint,omitempty"`
	RawQuery                 string `json:"raw_query,omitempty"`
	NormalizedQuery          string `json:"normalized_query,omitempty"`
}

type helpMetadataEntry struct {
	PluginName       string   `json:"plugin"`
	Command          string   `json:"command"`
	SimpleMatcher    string   `json:"-"`
	Usage            string   `json:"usage,omitempty"`
	Summary          string   `json:"summary,omitempty"`
	Examples         []string `json:"examples,omitempty"`
	PrivateExamples  []string `json:"private_examples,omitempty"`
	Keywords         []string `json:"keywords,omitempty"`
	Scope            string   `json:"scope,omitempty"`
	PrivateOK        bool     `json:"private_ok,omitempty"`
	PrivateRequired  bool     `json:"private_required,omitempty"`
	PrivateSupported bool     `json:"private_supported,omitempty"`
	PrivateHint      string   `json:"private_hint,omitempty"`
	VisibleHere      bool     `json:"visible_here"`
	Channels         []string `json:"channels,omitempty"`
	AllChannels      bool     `json:"all_channels,omitempty"`
	PluginSummary    string   `json:"plugin_summary,omitempty"`
}

type helpMetadataMatch struct {
	PluginName  string `json:"plugin"`
	Command     string `json:"command"`
	Score       int    `json:"score"`
	VisibleHere bool   `json:"visible_here"`
}

type helpMetadataResponse struct {
	Context          helpMetadataContext `json:"context"`
	VisibleHere      []helpMetadataEntry `json:"visible_here"`
	Browseable       []helpMetadataEntry `json:"browseable"`
	RankedHere       []helpMetadataMatch `json:"ranked_here,omitempty"`
	RankedBrowseable []helpMetadataMatch `json:"ranked_browseable,omitempty"`
}

type fallbackAdviceEntry struct {
	PluginName    string   `json:"plugin"`
	Command       string   `json:"command"`
	SimpleMatcher string   `json:"-"`
	Usage         string   `json:"usage,omitempty"`
	Summary       string   `json:"summary,omitempty"`
	Examples      []string `json:"examples,omitempty"`
	Keywords      []string `json:"keywords,omitempty"`
	Channels      []string `json:"channels,omitempty"`
	VisibleHere   bool     `json:"visible_here,omitempty"`
	Score         int      `json:"score,omitempty"`
	PluginSummary string   `json:"plugin_summary,omitempty"`
}

type fallbackAdviceResponse struct {
	Context            helpMetadataContext   `json:"context"`
	Advice             string                `json:"advice"`
	WrongChannelHint   string                `json:"wrong_channel_hint,omitempty"`
	DeterministicReply string                `json:"deterministic_reply,omitempty"`
	Here               []fallbackAdviceEntry `json:"here,omitempty"`
	Elsewhere          []fallbackAdviceEntry `json:"elsewhere,omitempty"`
}

const (
	fallbackAdviceNoMatch      = "no_match"
	fallbackAdviceCloseHere    = "close_match_here"
	fallbackAdviceElsewhere    = "close_match_elsewhere"
	fallbackAdviceWrongChannel = "wrong_channel"
)

const (
	fallbackAdviceMaxHere      = 4
	fallbackAdviceMaxElsewhere = 4
	fallbackAdviceStrongScore  = 80
	fallbackAdviceLikelyScore  = 72
	fallbackAdviceMinScore     = 60
	fallbackAdviceScoreMargin  = 8
)

var fallbackStopWords = map[string]struct{}{
	"a": {}, "an": {}, "and": {}, "are": {}, "be": {}, "do": {}, "for": {}, "get": {}, "help": {},
	"i": {}, "in": {}, "is": {}, "it": {}, "me": {}, "my": {}, "of": {}, "on": {}, "please": {},
	"say": {}, "tell": {}, "the": {}, "to": {}, "what": {}, "whats": {}, "with": {}, "you": {},
	"your": {},
}

type fallbackMatchSignal struct {
	MeaningfulHits  int
	MeaningfulTerms int
}

func (e helpMetadataEntry) toHelpCommandMetadata() helpCommandMetadata {
	return helpCommandMetadata{
		PluginName:       e.PluginName,
		Command:          e.Command,
		SimpleMatcher:    e.SimpleMatcher,
		Usage:            e.Usage,
		Summary:          e.Summary,
		Examples:         append([]string(nil), e.Examples...),
		PrivateExamples:  append([]string(nil), e.PrivateExamples...),
		Keywords:         append([]string(nil), e.Keywords...),
		Scope:            e.Scope,
		Channels:         append([]string(nil), e.Channels...),
		AllChannels:      e.AllChannels,
		PrivateOK:        e.PrivateOK,
		PrivateRequired:  e.PrivateRequired,
		PrivateSupported: e.PrivateSupported,
		PrivateHint:      e.PrivateHint,
		PluginSummary:    e.PluginSummary,
	}
}

func catchAllModeMatches(plugin *Plugin, mode string) bool {
	if plugin == nil {
		return false
	}
	if len(plugin.CatchAllModes) == 0 {
		return true
	}
	mode = strings.TrimSpace(strings.ToLower(mode))
	for _, configured := range plugin.CatchAllModes {
		if strings.TrimSpace(strings.ToLower(configured)) == mode {
			return true
		}
	}
	return false
}

func selectCatchAllTarget(tasks []interface{}, mode string, available func(*Task, *Plugin) (bool, bool)) (specificCatchAll, fallbackCatchAll interface{}, multipleSpecific, multipleFallback bool) {
	for _, t := range tasks {
		task, plugin, _ := getTask(t)
		if plugin == nil || !plugin.CatchAll || !catchAllModeMatches(plugin, mode) {
			continue
		}
		taskAvailable, specific := available(task, plugin)
		if !taskAvailable {
			continue
		}
		if len(plugin.CatchAllModes) > 0 {
			specific = true
		}
		if specific {
			if specificCatchAll == nil {
				specificCatchAll = t
			} else {
				multipleSpecific = true
				return
			}
		} else {
			if fallbackCatchAll == nil {
				fallbackCatchAll = t
			} else {
				multipleFallback = true
			}
		}
	}
	return
}

func (r Robot) GetHelpMetadata(query string) string {
	payload := r.collectHelpMetadata(query)
	blob, err := json.Marshal(payload)
	if err != nil {
		r.Log(robot.Error, "GetHelpMetadata marshal failed: %v", err)
		return `{}`
	}
	return string(blob)
}

func (r Robot) collectHelpMetadata(query string) helpMetadataResponse {
	alias := r.GetBotAttribute("alias").String()
	botName := r.GetBotAttribute("name").String()
	normalized := normalizeFallbackTerm(query, alias, botName)

	protocol := protocolFromIncoming(r.Incoming, r.Protocol)
	result := helpMetadataResponse{
		Context: helpMetadataContext{
			BotName:                  botName,
			BotAlias:                 alias,
			User:                     r.User,
			Channel:                  r.Channel,
			CommandMode:              strings.TrimSpace(r.GetParameter("GOPHER_CMDMODE")),
			Direct:                   len(strings.TrimSpace(r.Channel)) == 0,
			Threaded:                 r.Incoming != nil && r.Incoming.ThreadedMessage,
			Protocol:                 protocol,
			PrivateCommandsSupported: hiddenCommandsSupportedForProtocol(protocol),
			PrivateCommandHint:       hiddenCommandHintForProtocol(protocol),
			RawQuery:                 strings.TrimSpace(query),
			NormalizedQuery:          normalized,
		},
	}

	w := getLockedWorker(r.tid)
	w.Unlock()

	type authorizerGroupLookup struct {
		groups map[string]struct{}
		known  bool
	}
	groupCache := make(map[string]authorizerGroupLookup)
	byCommand := make(map[string]*helpMetadataEntry)

	for _, t := range r.tasks.t[1:] {
		task, plugin, _ := getTask(t)
		if task == nil || plugin == nil || task.Disabled {
			continue
		}
		visibleHere, _ := w.pluginAvailable(task, false, false)
		browseable, _ := w.pluginAvailable(task, true, true)
		if !browseable {
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
			commandVisibleHere := visibleHere
			if privateCommandContext(w.Incoming) {
				commandVisibleHere = commandVisibleHere && commandAllowsPrivate(plugin, command)
			} else if commandRequiresPrivate(plugin, command) {
				commandVisibleHere = false
			}

			key := task.name + "|" + command
			entry, ok := byCommand[key]
			if !ok {
				entry = &helpMetadataEntry{
					PluginName:    task.name,
					Command:       command,
					Scope:         helpScopeText(task),
					VisibleHere:   commandVisibleHere,
					Channels:      append([]string(nil), task.Channels...),
					AllChannels:   task.AllChannels,
					PluginSummary: helpPluginSummary(task),
				}
				byCommand[key] = entry
			}
			if commandVisibleHere {
				entry.VisibleHere = true
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
				if result.Context.PrivateCommandsSupported {
					entry.PrivateSupported = true
				}
				if entry.PrivateHint == "" && strings.TrimSpace(result.Context.PrivateCommandHint) != "" {
					entry.PrivateHint = strings.TrimSpace(result.Context.PrivateCommandHint)
				}
			}
			entry.Examples = appendUniqueStrings(entry.Examples, matcher.Examples...)
			if entry.PrivateOK && result.Context.PrivateCommandsSupported {
				for _, example := range matcher.Examples {
					commandText := helpSurfaceCommandText(example, alias, botName)
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

	result.Browseable = make([]helpMetadataEntry, 0, len(byCommand))
	for _, entry := range byCommand {
		if len(entry.Usage) == 0 {
			entry.Usage = "(alias) " + entry.Command
		}
		result.Browseable = append(result.Browseable, *entry)
		if entry.VisibleHere {
			result.VisibleHere = append(result.VisibleHere, *entry)
		}
	}

	sort.Slice(result.Browseable, func(i, j int) bool {
		if result.Browseable[i].PluginName == result.Browseable[j].PluginName {
			return result.Browseable[i].Command < result.Browseable[j].Command
		}
		return result.Browseable[i].PluginName < result.Browseable[j].PluginName
	})
	sort.Slice(result.VisibleHere, func(i, j int) bool {
		if result.VisibleHere[i].PluginName == result.VisibleHere[j].PluginName {
			return result.VisibleHere[i].Command < result.VisibleHere[j].Command
		}
		return result.VisibleHere[i].PluginName < result.VisibleHere[j].PluginName
	})

	if normalized == "" {
		return result
	}

	allEntries := make([]helpCommandMetadata, 0, len(result.Browseable))
	for _, entry := range result.Browseable {
		allEntries = append(allEntries, entry.toHelpCommandMetadata())
	}
	for _, match := range rankHelpMatches(allEntries, normalized) {
		result.RankedBrowseable = append(result.RankedBrowseable, helpMetadataMatch{
			PluginName:  match.Entry.PluginName,
			Command:     match.Entry.Command,
			Score:       match.Score,
			VisibleHere: byCommand[match.Entry.PluginName+"|"+match.Entry.Command].VisibleHere,
		})
	}

	visibleEntries := make([]helpCommandMetadata, 0, len(result.VisibleHere))
	for _, entry := range result.VisibleHere {
		visibleEntries = append(visibleEntries, entry.toHelpCommandMetadata())
	}
	for _, match := range rankHelpMatches(visibleEntries, normalized) {
		result.RankedHere = append(result.RankedHere, helpMetadataMatch{
			PluginName:  match.Entry.PluginName,
			Command:     match.Entry.Command,
			Score:       match.Score,
			VisibleHere: true,
		})
	}

	return result
}

func (r Robot) collectFallbackAdvice(query string) fallbackAdviceResponse {
	meta := r.collectHelpMetadata(query)
	advice := fallbackAdviceResponse{
		Context: meta.Context,
		Advice:  fallbackAdviceNoMatch,
	}

	entryMap := make(map[string]helpMetadataEntry, len(meta.Browseable))
	for _, entry := range meta.Browseable {
		entryMap[entry.PluginName+"|"+entry.Command] = entry
	}

	for _, match := range meta.RankedHere {
		entry, ok := entryMap[match.PluginName+"|"+match.Command]
		if !ok {
			continue
		}
		if !isRelevantFallbackMatch(entry, meta.Context.NormalizedQuery, match.Score) {
			continue
		}
		advice.Here = append(advice.Here, toFallbackAdviceEntry(entry, match.Score))
		if len(advice.Here) >= fallbackAdviceMaxHere {
			break
		}
	}

	for _, match := range meta.RankedBrowseable {
		entry, ok := entryMap[match.PluginName+"|"+match.Command]
		if !ok || entry.VisibleHere {
			continue
		}
		if !isRelevantFallbackMatch(entry, meta.Context.NormalizedQuery, match.Score) {
			continue
		}
		advice.Elsewhere = append(advice.Elsewhere, toFallbackAdviceEntry(entry, match.Score))
		if len(advice.Elsewhere) >= fallbackAdviceMaxElsewhere {
			break
		}
	}

	if len(advice.Elsewhere) > 0 && shouldPreferWrongChannel(advice.Here, advice.Elsewhere[0], meta.Context.NormalizedQuery) {
		advice.Advice = fallbackAdviceWrongChannel
		advice.WrongChannelHint = fallbackContextFromAdviceEntry(advice.Elsewhere[0])
	}

	if advice.Advice == fallbackAdviceNoMatch {
		switch {
		case len(advice.Here) > 0:
			advice.Advice = fallbackAdviceCloseHere
		case len(advice.Elsewhere) > 0:
			advice.Advice = fallbackAdviceElsewhere
		}
	}

	advice.DeterministicReply = r.buildDeterministicFallbackReply(advice)
	return advice
}

func toFallbackAdviceEntry(entry helpMetadataEntry, score int) fallbackAdviceEntry {
	return fallbackAdviceEntry{
		PluginName:    entry.PluginName,
		Command:       entry.Command,
		SimpleMatcher: entry.SimpleMatcher,
		Usage:         entry.Usage,
		Summary:       entry.Summary,
		Examples:      append([]string(nil), entry.Examples...),
		Keywords:      append([]string(nil), entry.Keywords...),
		Channels:      append([]string(nil), entry.Channels...),
		VisibleHere:   entry.VisibleHere,
		Score:         score,
		PluginSummary: entry.PluginSummary,
	}
}

func fallbackMeaningfulTokens(input string) []string {
	rawTokens := helpTokenRegex.FindAllString(normalizeHelpPhrase(input), -1)
	if len(rawTokens) == 0 {
		return nil
	}
	out := make([]string, 0, len(rawTokens))
	seen := make(map[string]struct{}, len(rawTokens))
	for _, token := range rawTokens {
		if len(token) < 3 {
			continue
		}
		if _, stop := fallbackStopWords[token]; stop {
			continue
		}
		if _, ok := seen[token]; ok {
			continue
		}
		seen[token] = struct{}{}
		out = append(out, token)
	}
	return out
}

func fallbackMatchSignals(entry helpMetadataEntry, normalized string) fallbackMatchSignal {
	termTokens := fallbackMeaningfulTokens(normalized)
	if len(termTokens) == 0 {
		return fallbackMatchSignal{}
	}
	searchTokens := collectHelpSearchTokens(entry.toHelpCommandMetadata())
	if len(searchTokens) == 0 {
		return fallbackMatchSignal{MeaningfulTerms: len(termTokens)}
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
	return fallbackMatchSignal{
		MeaningfulHits:  hits,
		MeaningfulTerms: len(termTokens),
	}
}

func isRelevantFallbackMatch(entry helpMetadataEntry, normalized string, score int) bool {
	if fallbackCloseMatchScore(entry.toHelpCommandMetadata(), normalized) >= fallbackAdviceLikelyScore {
		return true
	}
	if score >= fallbackAdviceMinScore {
		return true
	}
	signals := fallbackMatchSignals(entry, normalized)
	if signals.MeaningfulTerms == 0 {
		return false
	}
	if signals.MeaningfulTerms == 1 {
		return signals.MeaningfulHits == 1 && score >= fallbackAdviceMinScore-fallbackAdviceScoreMargin
	}
	return signals.MeaningfulHits == signals.MeaningfulTerms && signals.MeaningfulHits >= 2
}

func shouldPreferWrongChannel(here []fallbackAdviceEntry, elsewhere fallbackAdviceEntry, normalized string) bool {
	elseSignals := fallbackMatchSignals(helpMetadataEntry{
		PluginName: elsewhere.PluginName,
		Command:    elsewhere.Command,
		Usage:      elsewhere.Usage,
		Summary:    elsewhere.Summary,
		Keywords:   append([]string(nil), elsewhere.Keywords...),
		Channels:   append([]string(nil), elsewhere.Channels...),
	}, normalized)
	var hereTop *fallbackAdviceEntry
	if len(here) > 0 {
		hereTop = &here[0]
	}
	if elseSignals.MeaningfulTerms >= 2 && elseSignals.MeaningfulHits == elseSignals.MeaningfulTerms {
		if hereTop == nil {
			return true
		}
		hereSignals := fallbackMatchSignals(helpMetadataEntry{
			PluginName: hereTop.PluginName,
			Command:    hereTop.Command,
			Usage:      hereTop.Usage,
			Summary:    hereTop.Summary,
			Keywords:   append([]string(nil), hereTop.Keywords...),
			Channels:   append([]string(nil), hereTop.Channels...),
		}, normalized)
		if hereSignals.MeaningfulHits < elseSignals.MeaningfulHits {
			return true
		}
	}
	if hereTop == nil && elsewhere.Score >= fallbackAdviceLikelyScore && elseSignals.MeaningfulHits > 0 {
		return true
	}
	if hereTop == nil && elseSignals.MeaningfulTerms > 0 && elseSignals.MeaningfulHits == elseSignals.MeaningfulTerms && elsewhere.Score >= fallbackAdviceMinScore {
		return true
	}
	if elsewhere.Score >= fallbackAdviceStrongScore {
		return hereTop == nil || elsewhere.Score >= hereTop.Score+fallbackAdviceScoreMargin
	}
	return false
}

func describeFallbackContext(entry helpMetadataEntry) string {
	if len(entry.Channels) > 0 {
		channels := append([]string(nil), entry.Channels...)
		sort.Strings(channels)
		return "This looks more likely to belong in " + joinFallbackChannels(channels) + "."
	}
	return ""
}

func (r Robot) buildDeterministicFallbackReply(advice fallbackAdviceResponse) string {
	attempted := advice.Context.NormalizedQuery
	if attempted == "" {
		attempted = advice.Context.RawQuery
	}
	alias := strings.TrimSpace(advice.Context.BotAlias)
	botName := strings.TrimSpace(advice.Context.BotName)
	commandPrefix := func() string {
		if advice.Context.CommandMode == "name" && botName != "" {
			return botName + ", "
		}
		if alias != "" {
			return alias
		}
		if botName != "" {
			return botName + ", "
		}
		return ""
	}
	displayPrefix := commandPrefix()
	formatHelpCommand := func(command string) string {
		if advice.Context.PrivateCommandsSupported {
			if hidden := strings.TrimSpace(formatHiddenCommand(advice.Context.Protocol, command)); hidden != "" {
				return "`" + hidden + "`"
			}
		}
		raw := "(alias) " + strings.TrimSpace(command)
		return "`" + strings.TrimSpace(r.expandHelpPlaceholders(raw)) + "`"
	}
	formatDisplayedAttempt := func() string {
		display := strings.TrimSpace(attempted)
		if display == "" {
			return "that command"
		}
		if displayPrefix != "" && !strings.HasPrefix(display, displayPrefix) {
			display = displayPrefix + display
		}
		return "`" + display + "`"
	}
	formatChannel := func() string {
		channel := strings.TrimSpace(advice.Context.Channel)
		if channel == "" || advice.Context.Direct {
			return ""
		}
		return "`#" + channel + "`"
	}
	currentLocationPhrase := func() string {
		if channel := formatChannel(); channel != "" {
			return " in channel " + channel
		}
		return ""
	}
	generalGuidance := "Try " + formatHelpCommand("commands") + " or " + formatHelpCommand("help <keyword>") + "."
	formatHelpPath := func(entry fallbackAdviceEntry) string {
		return "`" + entry.PluginName + "/" + entry.Command + "`"
	}
	formatExactHelp := func(entry fallbackAdviceEntry) string {
		return formatHelpCommand("help " + entry.PluginName + "/" + entry.Command)
	}
	formatExample := func(entry fallbackAdviceEntry) string {
		if example := r.formatFallbackExampleForMode(entry, advice.Context.CommandMode, botName); example != "" {
			return "`" + example + "`"
		}
		return ""
	}
	familyLead := func() string {
		tokens := helpTokenRegex.FindAllString(normalizeHelpPhrase(attempted), -1)
		if len(tokens) == 0 {
			return ""
		}
		return tokens[0]
	}
	renderFamilyOptions := func(entries []fallbackAdviceEntry) string {
		lines := []string{"You may be looking for:"}
		for _, entry := range entries {
			line := "- " + formatHelpPath(entry)
			if example := formatExample(entry); example != "" {
				line += " - " + example
			} else if summary := strings.TrimSpace(entry.Summary); summary != "" {
				line += " - " + summary
			}
			lines = append(lines, line)
		}
		return strings.Join(lines, "\n")
	}
	renderStrongSuggestion := func(entry fallbackAdviceEntry, includeAvailability bool) string {
		lines := []string{
			formatDisplayedAttempt() + " looks close to " + formatHelpPath(entry) + ".",
		}
		if example := formatExample(entry); example != "" {
			lines = append(lines, "Try: "+example)
		}
		if includeAvailability {
			if len(entry.Channels) > 0 {
				lines = append(lines, "That command isn't available in channel "+formatChannel()+"; try "+joinFallbackChannels(entry.Channels)+".")
			}
		}
		lines = append(lines, "More help: "+formatExactHelp(entry))
		return strings.Join(lines, "\n")
	}

	switch advice.Advice {
	case fallbackAdviceWrongChannel:
		if len(advice.Elsewhere) > 0 {
			top := advice.Elsewhere[0]
			if _, ok := fallbackConfidentSuggestion([]fallbackAdviceEntry{top}, advice.Context.NormalizedQuery); ok {
				return renderStrongSuggestion(top, true)
			}
			lines := []string{
				"I couldn't match " + formatDisplayedAttempt() + currentLocationPhrase() + ".",
			}
			if advice.WrongChannelHint != "" {
				lines = append(lines, advice.WrongChannelHint)
			}
			lines = append(lines, "More help: "+formatExactHelp(top))
			return strings.Join(lines, "\n")
		}
	case fallbackAdviceCloseHere:
		if len(advice.Here) == 1 {
			top := advice.Here[0]
			if _, ok := fallbackConfidentSuggestion(advice.Here, advice.Context.NormalizedQuery); ok {
				return renderStrongSuggestion(top, false)
			}
			lines := []string{
				"I couldn't match " + formatDisplayedAttempt() + currentLocationPhrase() + ".",
				"More help: " + formatExactHelp(top),
				generalGuidance,
			}
			return strings.Join(lines, "\n")
		}
		if top, ok := fallbackConfidentSuggestion(advice.Here, advice.Context.NormalizedQuery); ok {
			return renderStrongSuggestion(top, false)
		}
		if lead := familyLead(); lead != "" {
			lines := []string{
				"`" + lead + "` matches several commands.",
				renderFamilyOptions(advice.Here),
				"More help: " + formatHelpCommand("help "+lead),
				"Exact help: " + formatHelpCommand("help <plugin>/<command>"),
			}
			return strings.Join(lines, "\n")
		}
		lines := []string{
			"I couldn't match " + formatDisplayedAttempt() + currentLocationPhrase() + ".",
			generalGuidance,
		}
		return strings.Join(lines, "\n")
	}

	if len(advice.Elsewhere) > 0 {
		top := advice.Elsewhere[0]
		if _, ok := fallbackConfidentSuggestion([]fallbackAdviceEntry{top}, advice.Context.NormalizedQuery); ok {
			return renderStrongSuggestion(top, true)
		}
		lines := []string{
			"I couldn't match " + formatDisplayedAttempt() + currentLocationPhrase() + ".",
			"More help: " + formatExactHelp(top),
			generalGuidance,
		}
		return strings.Join(lines, "\n")
	}

	if strings.TrimSpace(attempted) == "" {
		if channel := strings.TrimSpace(advice.Context.Channel); channel != "" && !advice.Context.Direct {
			return "I couldn't match that command in channel `#" + channel + "`.\n" + generalGuidance
		}
		return "I couldn't match that command.\n" + generalGuidance
	}
	if channel := strings.TrimSpace(advice.Context.Channel); channel != "" && !advice.Context.Direct {
		return "I couldn't match " + formatDisplayedAttempt() + " in channel `#" + channel + "`.\n" + generalGuidance
	}
	return "I couldn't match " + formatDisplayedAttempt() + ".\n" + generalGuidance
}

func fallbackConfidentSuggestion(entries []fallbackAdviceEntry, query string) (fallbackAdviceEntry, bool) {
	if len(entries) == 0 {
		return fallbackAdviceEntry{}, false
	}
	top := entries[0]
	topClose := fallbackCloseMatchScore(top.toHelpCommandMetadata(), query)
	if topClose < fallbackAdviceLikelyScore {
		return fallbackAdviceEntry{}, false
	}
	topSignals := fallbackMatchSignals(helpMetadataEntry{
		PluginName:    top.PluginName,
		Command:       top.Command,
		SimpleMatcher: top.SimpleMatcher,
		Usage:         top.Usage,
		Summary:       top.Summary,
		Examples:      append([]string(nil), top.Examples...),
		Keywords:      append([]string(nil), top.Keywords...),
	}, query)
	if topSignals.MeaningfulTerms > 1 && topSignals.MeaningfulHits < 2 && topSignals.MeaningfulHits < topSignals.MeaningfulTerms {
		return fallbackAdviceEntry{}, false
	}
	if len(entries) == 1 {
		return top, true
	}
	nextClose := fallbackCloseMatchScore(entries[1].toHelpCommandMetadata(), query)
	if topClose >= fallbackAdviceStrongScore && topClose >= nextClose+fallbackAdviceScoreMargin {
		return top, true
	}
	return fallbackAdviceEntry{}, false
}

func (r Robot) formatFallbackNextStep(alias string, entry fallbackAdviceEntry) string {
	helpPath := alias + "help " + entry.PluginName + "/" + entry.Command
	if example := r.formatFallbackExample(entry); example != "" {
		if len(entry.Channels) > 0 {
			return "Try `" + helpPath + "` in " + joinFallbackChannels(entry.Channels) + ", or run `" + example + "` there."
		}
		return "Try `" + helpPath + "` or run `" + example + "`."
	}
	if len(entry.Channels) > 0 {
		return "Try `" + helpPath + "` in " + joinFallbackChannels(entry.Channels) + "."
	}
	return "Try `" + helpPath + "`."
}

func (r Robot) formatFallbackExample(entry fallbackAdviceEntry) string {
	if len(entry.Examples) == 0 {
		return ""
	}
	return strings.TrimSpace(r.expandHelpPlaceholders(entry.Examples[0]))
}

func (r Robot) formatFallbackExampleForMode(entry fallbackAdviceEntry, mode, botName string) string {
	if len(entry.Examples) == 0 {
		return ""
	}
	example := entry.Examples[0]
	if strings.TrimSpace(mode) == "name" && strings.TrimSpace(botName) != "" {
		example = strings.ReplaceAll(example, "(alias)", botName+",")
	}
	return strings.TrimSpace(r.expandHelpPlaceholders(example))
}

func (e fallbackAdviceEntry) toHelpCommandMetadata() helpCommandMetadata {
	return helpCommandMetadata{
		PluginName:    e.PluginName,
		Command:       e.Command,
		SimpleMatcher: e.SimpleMatcher,
		Usage:         e.Usage,
		Summary:       e.Summary,
		Examples:      append([]string(nil), e.Examples...),
		Keywords:      append([]string(nil), e.Keywords...),
		PluginSummary: e.PluginSummary,
	}
}

func fallbackContextFromAdviceEntry(entry fallbackAdviceEntry) string {
	if len(entry.Channels) > 0 {
		return "This looks more likely to belong in " + joinFallbackChannels(entry.Channels) + "."
	}
	return ""
}

func joinFallbackChannels(channels []string) string {
	if len(channels) == 0 {
		return ""
	}
	sort.Strings(channels)
	quoted := make([]string, 0, len(channels))
	for _, ch := range channels {
		quoted = append(quoted, "#"+ch)
	}
	if len(quoted) > 3 {
		return strings.Join(quoted[:3], ", ") + ", or other configured channels"
	}
	if len(quoted) == 1 {
		return quoted[0]
	}
	return strings.Join(quoted[:len(quoted)-1], ", ") + " or " + quoted[len(quoted)-1]
}
