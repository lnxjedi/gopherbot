package bot

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/lnxjedi/gopherbot/robot"
)

type helpMetadataContext struct {
	BotName         string `json:"bot_name,omitempty"`
	BotAlias        string `json:"bot_alias,omitempty"`
	User            string `json:"user,omitempty"`
	Channel         string `json:"channel,omitempty"`
	CommandMode     string `json:"command_mode,omitempty"`
	Direct          bool   `json:"direct"`
	Threaded        bool   `json:"threaded"`
	Protocol        string `json:"protocol,omitempty"`
	RawQuery        string `json:"raw_query,omitempty"`
	NormalizedQuery string `json:"normalized_query,omitempty"`
}

type helpMetadataEntry struct {
	PluginName  string   `json:"plugin"`
	Command     string   `json:"command"`
	Usage       string   `json:"usage,omitempty"`
	Summary     string   `json:"summary,omitempty"`
	Examples    []string `json:"examples,omitempty"`
	Keywords    []string `json:"keywords,omitempty"`
	Scope       string   `json:"scope,omitempty"`
	HiddenOK    bool     `json:"hidden_ok,omitempty"`
	VisibleHere bool     `json:"visible_here"`
	Channels    []string `json:"channels,omitempty"`
	AllChannels bool     `json:"all_channels,omitempty"`
	AllowDirect bool     `json:"allow_direct,omitempty"`
	DirectOnly  bool     `json:"direct_only,omitempty"`
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
	PluginName  string   `json:"plugin"`
	Command     string   `json:"command"`
	Usage       string   `json:"usage,omitempty"`
	Summary     string   `json:"summary,omitempty"`
	Keywords    []string `json:"keywords,omitempty"`
	Channels    []string `json:"channels,omitempty"`
	DirectOnly  bool     `json:"direct_only,omitempty"`
	AllowDirect bool     `json:"allow_direct,omitempty"`
	VisibleHere bool     `json:"visible_here,omitempty"`
	Score       int      `json:"score,omitempty"`
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
		PluginName: e.PluginName,
		Command:    e.Command,
		Usage:      e.Usage,
		Summary:    e.Summary,
		Examples:   append([]string(nil), e.Examples...),
		Keywords:   append([]string(nil), e.Keywords...),
		Scope:      e.Scope,
		HiddenOK:   e.HiddenOK,
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

func (r Robot) GetFallbackAdvice(query string) string {
	payload := r.collectFallbackAdvice(query)
	blob, err := json.Marshal(payload)
	if err != nil {
		r.Log(robot.Error, "GetFallbackAdvice marshal failed: %v", err)
		return `{}`
	}
	return string(blob)
}

func (r Robot) collectHelpMetadata(query string) helpMetadataResponse {
	alias := r.GetBotAttribute("alias").String()
	botName := r.GetBotAttribute("name").String()
	normalized := normalizeFallbackTerm(query, alias, botName)

	result := helpMetadataResponse{
		Context: helpMetadataContext{
			BotName:         botName,
			BotAlias:        alias,
			User:            r.User,
			Channel:         r.Channel,
			CommandMode:     strings.TrimSpace(r.GetParameter("GOPHER_CMDMODE")),
			Direct:          len(strings.TrimSpace(r.Channel)) == 0,
			Threaded:        r.Incoming != nil && r.Incoming.ThreadedMessage,
			Protocol:        protocolFromIncoming(r.Incoming, r.Protocol),
			RawQuery:        strings.TrimSpace(query),
			NormalizedQuery: normalized,
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

			key := task.name + "|" + command
			entry, ok := byCommand[key]
			if !ok {
				entry = &helpMetadataEntry{
					PluginName:  task.name,
					Command:     command,
					Scope:       helpScopeText(task),
					VisibleHere: visibleHere,
					Channels:    append([]string(nil), task.Channels...),
					AllChannels: task.AllChannels,
					AllowDirect: task.AllowDirect,
					DirectOnly:  task.DirectOnly,
				}
				byCommand[key] = entry
			}
			if visibleHere {
				entry.VisibleHere = true
			}
			if len(entry.Usage) == 0 && len(strings.TrimSpace(matcher.Usage)) > 0 {
				entry.Usage = strings.TrimSpace(matcher.Usage)
			}
			if len(entry.Summary) == 0 && len(strings.TrimSpace(matcher.Summary)) > 0 {
				entry.Summary = strings.TrimSpace(matcher.Summary)
			}
			if commandAllowsHidden(plugin, command) {
				entry.HiddenOK = true
			}
			entry.Examples = appendUniqueStrings(entry.Examples, matcher.Examples...)
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

	advice.DeterministicReply = buildDeterministicFallbackReply(advice)
	return advice
}

func toFallbackAdviceEntry(entry helpMetadataEntry, score int) fallbackAdviceEntry {
	return fallbackAdviceEntry{
		PluginName:  entry.PluginName,
		Command:     entry.Command,
		Usage:       entry.Usage,
		Summary:     entry.Summary,
		Keywords:    append([]string(nil), entry.Keywords...),
		Channels:    append([]string(nil), entry.Channels...),
		DirectOnly:  entry.DirectOnly,
		AllowDirect: entry.AllowDirect,
		VisibleHere: entry.VisibleHere,
		Score:       score,
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
		PluginName:  elsewhere.PluginName,
		Command:     elsewhere.Command,
		Usage:       elsewhere.Usage,
		Summary:     elsewhere.Summary,
		Keywords:    append([]string(nil), elsewhere.Keywords...),
		Channels:    append([]string(nil), elsewhere.Channels...),
		AllowDirect: elsewhere.AllowDirect,
		DirectOnly:  elsewhere.DirectOnly,
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
			PluginName:  hereTop.PluginName,
			Command:     hereTop.Command,
			Usage:       hereTop.Usage,
			Summary:     hereTop.Summary,
			Keywords:    append([]string(nil), hereTop.Keywords...),
			Channels:    append([]string(nil), hereTop.Channels...),
			AllowDirect: hereTop.AllowDirect,
			DirectOnly:  hereTop.DirectOnly,
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
	if entry.DirectOnly {
		return "This looks more likely to be a direct-message-only command."
	}
	if len(entry.Channels) > 0 {
		channels := append([]string(nil), entry.Channels...)
		sort.Strings(channels)
		return "This looks more likely to belong in " + joinFallbackChannels(channels) + "."
	}
	if entry.AllowDirect && !entry.AllChannels {
		return "This might work better in a direct message."
	}
	return ""
}

func buildDeterministicFallbackReply(advice fallbackAdviceResponse) string {
	attempted := advice.Context.NormalizedQuery
	if attempted == "" {
		attempted = advice.Context.RawQuery
	}
	alias := strings.TrimSpace(advice.Context.BotAlias)
	if alias == "" {
		alias = "!"
	}

	switch advice.Advice {
	case fallbackAdviceWrongChannel:
		if len(advice.Elsewhere) > 0 {
			top := advice.Elsewhere[0]
			line := "I couldn't match `" + attempted + "` here."
			if advice.WrongChannelHint != "" {
				line += " " + advice.WrongChannelHint
			}
			return strings.TrimSpace(line + " " + formatFallbackNextStep(alias, top))
		}
	case fallbackAdviceCloseHere:
		lines := []string{
			"I couldn't match `" + attempted + "`, but these look close:",
		}
		for _, entry := range advice.Here {
			line := "- [" + entry.PluginName + "] `" + entry.Command + "`"
			if strings.TrimSpace(entry.Summary) != "" {
				line += " - " + strings.TrimSpace(entry.Summary)
			}
			lines = append(lines, line)
		}
		if len(advice.Here) > 0 {
			lines = append(lines, "Try `"+alias+"help "+advice.Here[0].Command+"` or `"+alias+"commands`.")
		}
		return strings.Join(lines, "\n")
	}

	if len(advice.Elsewhere) > 0 {
		top := advice.Elsewhere[0]
		contextHint := advice.WrongChannelHint
		if contextHint == "" {
			contextHint = fallbackContextFromAdviceEntry(top)
		}
		line := "I couldn't match `" + attempted + "` here."
		if contextHint != "" {
			line += " " + contextHint
		}
		return strings.TrimSpace(line + " " + formatFallbackNextStep(alias, top))
	}

	if strings.TrimSpace(attempted) == "" {
		if channel := strings.TrimSpace(advice.Context.Channel); channel != "" && !advice.Context.Direct {
			return "I couldn't match that command in #" + channel + ". Try `" + alias + "commands` or `" + alias + "help <keyword>`."
		}
		return "I couldn't match that command. Try `" + alias + "commands` or `" + alias + "help <keyword>`."
	}
	if channel := strings.TrimSpace(advice.Context.Channel); channel != "" && !advice.Context.Direct {
		return "I couldn't match `" + attempted + "` in #" + channel + ". Try `" + alias + "commands` or `" + alias + "help <keyword>`."
	}
	return "I couldn't match `" + attempted + "`. Try `" + alias + "commands` or `" + alias + "help <keyword>`."
}

func formatFallbackNextStep(alias string, entry fallbackAdviceEntry) string {
	if len(entry.Channels) > 0 {
		return "Try `" + alias + "help " + entry.Command + "` in " + joinFallbackChannels(entry.Channels) + "."
	}
	if entry.DirectOnly {
		return "Try `" + alias + "help " + entry.Command + "` in a direct message."
	}
	return "Try `" + alias + "help " + entry.Command + "`."
}

func fallbackContextFromAdviceEntry(entry fallbackAdviceEntry) string {
	if entry.DirectOnly {
		return "This looks more likely to be a direct-message-only command."
	}
	if len(entry.Channels) > 0 {
		return "This looks more likely to belong in " + joinFallbackChannels(entry.Channels) + "."
	}
	if entry.AllowDirect {
		return "This might work better in a direct message."
	}
	return ""
}

func joinFallbackChannels(channels []string) string {
	if len(channels) == 0 {
		return ""
	}
	quoted := make([]string, 0, len(channels))
	for _, ch := range channels {
		quoted = append(quoted, "#"+ch)
	}
	if len(quoted) == 1 {
		return quoted[0]
	}
	return strings.Join(quoted[:len(quoted)-1], ", ") + " or " + quoted[len(quoted)-1]
}
