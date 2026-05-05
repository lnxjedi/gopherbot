package bot

import (
	"strings"
	"testing"

	"github.com/lnxjedi/gopherbot/robot"
)

type formatCaptureConnector struct {
	lastFormat  robot.MessageFormat
	lastMessage string
	lastUserID  string
	lastUser    string
	lastChannel string
}

func (c *formatCaptureConnector) GetProtocolUserAttribute(string, string) (string, robot.RetVal) {
	return "", robot.AttributeNotFound
}

func (c *formatCaptureConnector) MessageHeard(string, string) {}

func (c *formatCaptureConnector) DefaultHelp() []string { return nil }

func (c *formatCaptureConnector) JoinChannel(string) robot.RetVal { return robot.Ok }

func (c *formatCaptureConnector) SendProtocolChannelThreadMessage(_, _, msg string, f robot.MessageFormat, _ *robot.ConnectorMessage) robot.RetVal {
	c.lastFormat = f
	c.lastMessage = msg
	return robot.Ok
}

func (c *formatCaptureConnector) SendProtocolUserChannelThreadMessage(uid, username, channel, _, msg string, f robot.MessageFormat, _ *robot.ConnectorMessage) robot.RetVal {
	c.lastFormat = f
	c.lastUserID = uid
	c.lastUser = username
	c.lastChannel = channel
	c.lastMessage = msg
	return robot.Ok
}

func (c *formatCaptureConnector) SendProtocolUserMessage(_, msg string, f robot.MessageFormat, _ *robot.ConnectorMessage) robot.RetVal {
	c.lastFormat = f
	c.lastMessage = msg
	return robot.Ok
}

func (c *formatCaptureConnector) Reload() error { return nil }

func (c *formatCaptureConnector) Run(<-chan struct{}) {}

func makeFormatTestRobot(t *testing.T) Robot {
	t.Helper()
	w := &worker{
		User:       "alice",
		Channel:    "general",
		Protocol:   robot.Test,
		Incoming:   &robot.ConnectorMessage{Protocol: "test", ChannelName: "general"},
		cfg:        &configuration{alias: '!', botinfo: UserInfo{UserName: "Clu"}, defaultMessageFormat: robot.Raw},
		tasks:      &taskList{t: []interface{}{&Task{name: "namespace"}}, nameMap: map[string]int{}, nameSpaces: map[string]ParameterSet{}, parameterSets: map[string]ParameterSet{}},
		listedUser: true,
		pipeContext: &pipeContext{
			parameters:  map[string]string{"GOPHER_CMDMODE": "alias"},
			environment: map[string]string{},
		},
	}
	r := w.makeRobot()
	w.registerWorker(r.tid)
	t.Cleanup(func() {
		deregisterWorker(r.tid)
	})
	return r
}

func TestHelpTokenEquivalentSingularPlural(t *testing.T) {
	if !helpTokenEquivalent("siding", "sidings") {
		t.Fatalf("helpTokenEquivalent() expected siding and sidings to match")
	}
	if !helpTokenEquivalent("list", "lists") {
		t.Fatalf("helpTokenEquivalent() expected list and lists to match")
	}
	if helpTokenEquivalent("robot", "channel") {
		t.Fatalf("helpTokenEquivalent() expected robot and channel to differ")
	}
}

func TestScoreHelpCommandMatch(t *testing.T) {
	entry := helpCommandMetadata{
		PluginName: "lists",
		Command:    "add",
		Usage:      "(alias) add <item> to the <type> list",
		Summary:    "Adds an item to a named list.",
		Keywords:   []string{"list", "lists", "add"},
	}

	scoreList := scoreHelpCommandMatch(entry, "list")
	scoreAdd := scoreHelpCommandMatch(entry, "add")
	scoreNone := scoreHelpCommandMatch(entry, "sidings")
	if scoreList <= 0 {
		t.Fatalf("scoreHelpCommandMatch() expected positive score for list, got %d", scoreList)
	}
	if scoreAdd <= scoreList {
		t.Fatalf("scoreHelpCommandMatch() expected exact command score (%d) > keyword score (%d)", scoreAdd, scoreList)
	}
	if scoreNone != 0 {
		t.Fatalf("scoreHelpCommandMatch() expected 0 for non-match, got %d", scoreNone)
	}
}

func TestScoreHelpCommandMatchWithoutKeywords(t *testing.T) {
	entry := helpCommandMetadata{
		PluginName: "links",
		Command:    "find",
		Usage:      "(alias) find <keyword/phrase>",
		Summary:    "Finds saved links whose keys contain the given phrase.",
	}
	if got := scoreHelpCommandMatch(entry, "phrase"); got <= 0 {
		t.Fatalf("scoreHelpCommandMatch() expected positive score without explicit keywords, got %d", got)
	}
}

func TestScoreHelpCommandMatchKeywordOutranksSummaryToken(t *testing.T) {
	entry := helpCommandMetadata{
		PluginName: "lists",
		Command:    "show",
		Usage:      "(alias) show the <type> list",
		Summary:    "Displays all items in a list.",
		Keywords:   []string{"view"},
	}
	scoreKeyword := scoreHelpCommandMatch(entry, "view")
	scoreSummaryToken := scoreHelpCommandMatch(entry, "displays")
	if scoreKeyword <= scoreSummaryToken {
		t.Fatalf("scoreHelpCommandMatch() expected explicit keyword score (%d) > summary token score (%d)", scoreKeyword, scoreSummaryToken)
	}
}

func TestScoreHelpCommandMatchRecoversSingleTypo(t *testing.T) {
	entry := helpCommandMetadata{
		PluginName:    "knock",
		Command:       "knock",
		SimpleMatcher: "tell me a [:another] [:knock-knock] joke",
		Usage:         "tell me a knock-knock joke",
		Summary:       "Starts an interactive knock-knock joke.",
		Keywords:      []string{"joke", "knock"},
	}
	if got := scoreHelpCommandMatch(entry, "knok"); got < 80 {
		t.Fatalf("scoreHelpCommandMatch() expected strong typo score for knok, got %d", got)
	}
}

func TestFallbackCloseMatchScorePrefersRealPhraseSurface(t *testing.T) {
	entry := helpCommandMetadata{
		PluginName:    "knock",
		Command:       "knock",
		SimpleMatcher: "tell me a [:another] [:knock-knock] joke",
		Usage:         "tell me a knock-knock joke",
		Summary:       "Starts an interactive knock-knock joke.",
		Examples:      []string{"(alias) tell me another joke"},
		Keywords:      []string{"joke", "knock"},
	}
	if got := fallbackCloseMatchScore(entry, "tell me a jok"); got < 80 {
		t.Fatalf("fallbackCloseMatchScore() expected strong phrase score for tell me a jok, got %d", got)
	}
	if got := fallbackCloseMatchScore(entry, "knok"); got != 0 {
		t.Fatalf("fallbackCloseMatchScore() expected bare identifier miss for phrase command, got %d", got)
	}
}

func TestNormalizeFallbackTerm(t *testing.T) {
	got := normalizeFallbackTerm(";create a new grocery list", ";", "bender")
	if got != "create a new grocery list" {
		t.Fatalf("normalizeFallbackTerm() alias got %q", got)
	}
	got = normalizeFallbackTerm("bender, create a new grocery list", ";", "bender")
	if got != "create a new grocery list" {
		t.Fatalf("normalizeFallbackTerm() name got %q", got)
	}
}

func TestRankHelpMatches(t *testing.T) {
	entries := []helpCommandMetadata{
		{
			PluginName: "links",
			Command:    "list",
			Usage:      "(alias) list links",
			Summary:    "Lists saved links.",
			Keywords:   []string{"link", "links", "list"},
		},
		{
			PluginName: "lists",
			Command:    "add",
			Usage:      "(alias) add <item> to the <type> list",
			Summary:    "Adds an item to a named list.",
			Keywords:   []string{"list", "lists", "add"},
		},
	}
	matches := rankHelpMatches(entries, "create a new grocery list")
	if len(matches) == 0 {
		t.Fatalf("rankHelpMatches() expected at least one match")
	}
	if matches[0].Entry.PluginName != "lists" {
		t.Fatalf("rankHelpMatches() top plugin = %q, want %q", matches[0].Entry.PluginName, "lists")
	}
}

func TestRankHelpMatchesRegressionTable(t *testing.T) {
	entries := []helpCommandMetadata{
		{
			PluginName: "links",
			Command:    "find",
			Usage:      "(alias) find <keyword/phrase>",
			Summary:    "Finds saved links whose keys contain the given phrase.",
			Keywords:   []string{"link", "links", "find"},
		},
		{
			PluginName: "lists",
			Command:    "add",
			Usage:      "(alias) add <item> to the <type> list",
			Summary:    "Adds an item to a named list.",
			Keywords:   []string{"list", "lists", "add"},
		},
		{
			PluginName: "lists",
			Command:    "show",
			Usage:      "(alias) show the <type> list",
			Summary:    "Displays all items in a list.",
			Keywords:   []string{"view"},
		},
	}

	cases := []struct {
		query      string
		wantPlugin string
		wantCmd    string
	}{
		{"create a new grocery list", "lists", "add"},
		{"find links", "links", "find"},
		{"view", "lists", "show"},
		{"phrase", "links", "find"},
	}

	for _, tc := range cases {
		matches := rankHelpMatches(entries, tc.query)
		if len(matches) == 0 {
			t.Fatalf("rankHelpMatches(%q) returned no matches", tc.query)
		}
		got := matches[0].Entry
		if got.PluginName != tc.wantPlugin || got.Command != tc.wantCmd {
			t.Fatalf("rankHelpMatches(%q) top = [%s] %s, want [%s] %s", tc.query, got.PluginName, got.Command, tc.wantPlugin, tc.wantCmd)
		}
	}
}

func TestRenderPluginHelpOverviewIncludesExamplesAndMoreDetail(t *testing.T) {
	r := makeFormatTestRobot(t)
	got := r.renderPluginHelpOverview("knock", []helpCommandMetadata{
		{
			PluginName: "knock",
			Command:    "knock",
			Summary:    "Starts an interactive knock-knock joke.",
			Examples:   []string{"(alias) tell me a knock-knock joke"},
		},
		{
			PluginName: "knock",
			Command:    "another",
			Summary:    "Starts another joke.",
			Usage:      "(alias) tell me another joke",
		},
	})
	if !strings.Contains(got, "**Plugin help:** `knock`") {
		t.Fatalf("renderPluginHelpOverview() missing plugin header: %q", got)
	}
	if !strings.Contains(got, "Example: `!tell me a knock-knock joke`") {
		t.Fatalf("renderPluginHelpOverview() missing explicit example: %q", got)
	}
	if !strings.Contains(got, "Example: `tell me another joke`") {
		t.Fatalf("renderPluginHelpOverview() missing usage fallback example: %q", got)
	}
	if !strings.Contains(got, "**More detail:** `!help knock/<command>`") {
		t.Fatalf("renderPluginHelpOverview() missing more detail hint: %q", got)
	}
}

func TestRankHelpMatchesPrefersFirstMeaningfulTokenStem(t *testing.T) {
	entries := []helpCommandMetadata{
		{
			PluginName: "sidetrack",
			Command:    "sidetrack-story",
			Usage:      "(alias) sidetrack-story <name>",
			Summary:    "Recall a named sidetrack story.",
			Keywords:   []string{"sidetrack", "story"},
			Examples:   []string{"(alias) sidetrack story foo"},
		},
		{
			PluginName: "stories",
			Command:    "story-info",
			Usage:      "(alias) story-info <name>",
			Summary:    "Show metadata about a story.",
			Keywords:   []string{"story", "info"},
			Examples:   []string{"(alias) story info foo"},
		},
	}

	matches := rankHelpMatches(entries, "sidetrack story foo")
	if len(matches) == 0 {
		t.Fatal("rankHelpMatches() returned no matches")
	}
	if got := matches[0].Entry; got.PluginName != "sidetrack" || got.Command != "sidetrack-story" {
		t.Fatalf("top match = [%s] %s, want [sidetrack] sidetrack-story", got.PluginName, got.Command)
	}
}

func TestStripHelpAddressPrefix(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"(alias) ping", "ping"},
		{"(bot), whoami", "whoami"},
		{"/(bot) help ping", "help ping"},
		{"help-all", "help-all"},
	}
	for _, tc := range cases {
		if got := stripHelpAddressPrefix(tc.in); got != tc.want {
			t.Fatalf("stripHelpAddressPrefix(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestHiddenSlashBotCommand(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"list lists", "/clu list lists"},
		{"ping", "/clu ping"},
	}
	for _, tc := range cases {
		if got := hiddenSlashBotCommand("Clu", tc.in); got != tc.want {
			t.Fatalf("hiddenSlashBotCommand(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestFormatExactHelpAvailability(t *testing.T) {
	t.Run("explicit channels stay explicit", func(t *testing.T) {
		entry := helpCommandMetadata{
			Channels: []string{"clu-jobs", "general", "chat", "random", "botdev"},
		}
		want := "`#botdev`, `#chat`, `#clu-jobs`, `#general`, `#random`"
		if got := formatExactHelpAvailability(entry); got != want {
			t.Fatalf("formatExactHelpAvailability(channels) = %q, want %q", got, want)
		}
	})

	t.Run("all channels uses robot wording", func(t *testing.T) {
		entry := helpCommandMetadata{AllChannels: true}
		if got := formatExactHelpAvailability(entry); got != "all robot channels" {
			t.Fatalf("formatExactHelpAvailability(all channels) = %q", got)
		}
	})
}

func TestFormatSuggestedCommandPrefersHiddenHelp(t *testing.T) {
	runtimeConnectors.Lock()
	origPrimary := runtimeConnectors.primary
	origDefault := runtimeConnectors.defaultProtocol
	origRuntimes := runtimeConnectors.runtimes
	runtimeConnectors.primary = "testhidden"
	runtimeConnectors.defaultProtocol = "testhidden"
	runtimeConnectors.runtimes = map[string]*managedConnector{
		"testhidden": {
			connector: &hiddenHelpTestConnector{},
			caps:      robot.ConnectorCapabilities{HiddenCommands: true},
		},
	}
	runtimeConnectors.Unlock()
	t.Cleanup(func() {
		runtimeConnectors.Lock()
		runtimeConnectors.primary = origPrimary
		runtimeConnectors.defaultProtocol = origDefault
		runtimeConnectors.runtimes = origRuntimes
		runtimeConnectors.Unlock()
	})

	w := &worker{
		Protocol: robot.Protocol(0),
		Incoming: &robot.ConnectorMessage{Protocol: "testhidden"},
		cfg:      &configuration{alias: '!', botinfo: UserInfo{UserName: "Clu"}},
		pipeContext: &pipeContext{
			parameters:  map[string]string{},
			environment: map[string]string{},
		},
	}
	r := w.makeRobot()
	w.registerWorker(r.tid)
	defer deregisterWorker(r.tid)

	if got := r.formatSuggestedCommand("(alias) help knock/knock"); got != "/clu help knock/knock" {
		t.Fatalf("formatSuggestedCommand() = %q", got)
	}
}

func TestSummarizeQualifiedPluginCommands(t *testing.T) {
	entries := []helpCommandMetadata{
		{PluginName: "builtin-admin", Command: "branch"},
		{PluginName: "builtin-admin", Command: "abort"},
		{PluginName: "builtin-admin", Command: "chanlog"},
		{PluginName: "builtin-admin", Command: "defaultbranch"},
		{PluginName: "builtin-admin", Command: "quit"},
	}
	want := "`builtin-admin/abort`, `/branch`, `/chanlog`, `/defaultbranch` ... (+1 more)"
	if got := summarizeQualifiedPluginCommands("builtin-admin", entries, 4); got != want {
		t.Fatalf("summarizeQualifiedPluginCommands() = %q, want %q", got, want)
	}
}

func TestHelpSurfaceCommandText(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"(alias) ping", "ping"},
		{"!ping", "ping"},
		{"Clu, ping", "ping"},
		{"Clu ping", "ping"},
		{"/clu help ping", "help ping"},
		{"help ping", "help ping"},
	}
	for _, tc := range cases {
		if got := helpSurfaceCommandText(tc.in, "!", "Clu"); got != tc.want {
			t.Fatalf("helpSurfaceCommandText(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestRenderHelpEntryIncludesHiddenExamples(t *testing.T) {
	w := &worker{
		cfg: &configuration{alias: '!', botinfo: UserInfo{UserName: "Clu"}},
		pipeContext: &pipeContext{
			parameters:  map[string]string{},
			environment: map[string]string{},
		},
	}
	r := w.makeRobot()
	w.registerWorker(r.tid)
	defer deregisterWorker(r.tid)
	entry := helpCommandMetadata{
		PluginName:     "builtin-help",
		Command:        "help",
		Examples:       []string{"(alias) help ping"},
		HiddenExamples: []string{"/clu help ping"},
	}
	rendered := r.renderHelpEntry(entry, true, false, 0)
	if !containsLine(rendered, "**Examples:**") {
		t.Fatalf("rendered help missing Examples section: %q", rendered)
	}
	if !containsLine(rendered, "**Hidden examples:**") {
		t.Fatalf("rendered help missing Hidden examples section: %q", rendered)
	}
}

func containsLine(rendered, target string) bool {
	for _, line := range strings.Split(rendered, "\n") {
		if line == target {
			return true
		}
	}
	return false
}

func TestParseHelpQueryMode(t *testing.T) {
	term, brief := parseHelpQueryMode([]string{"knock", "brief"})
	if term != "knock" || !brief {
		t.Fatalf("parseHelpQueryMode(knock brief) = (%q, %t), want (%q, %t)", term, brief, "knock", true)
	}

	term, brief = parseHelpQueryMode([]string{"knock brief"})
	if term != "knock" || !brief {
		t.Fatalf("parseHelpQueryMode(single arg knock brief) = (%q, %t), want (%q, %t)", term, brief, "knock", true)
	}

	term, brief = parseHelpQueryMode([]string{"sidetrack", "story", "foo"})
	if term != "sidetrack story foo" || brief {
		t.Fatalf("parseHelpQueryMode(sidetrack story foo) = (%q, %t), want (%q, %t)", term, brief, "sidetrack story foo", false)
	}
}

func TestParseHelpQuery(t *testing.T) {
	parsed := parseHelpQuery([]string{"knock/knock"})
	if !parsed.HasPath || parsed.PluginName != "knock" || parsed.Command != "knock" || parsed.Brief {
		t.Fatalf("parseHelpQuery(knock/knock) = %+v, want exact plugin/command path", parsed)
	}

	parsed = parseHelpQuery([]string{"knock/knock", "brief"})
	if !parsed.HasPath || parsed.PluginName != "knock" || parsed.Command != "knock" || !parsed.Brief {
		t.Fatalf("parseHelpQuery(knock/knock brief) = %+v, want exact path with brief mode", parsed)
	}

	parsed = parseHelpQuery([]string{"sidetrack", "story", "foo"})
	if parsed.HasPath || parsed.Term != "sidetrack story foo" {
		t.Fatalf("parseHelpQuery(sidetrack story foo) = %+v, want non-path term", parsed)
	}
}

func TestCommandAllowsHidden(t *testing.T) {
	plugin := &Plugin{AllowedHiddenCommands: []string{"help", "*"}}
	if !commandAllowsHidden(plugin, "help") {
		t.Fatalf("expected explicit hidden command match")
	}
	if !commandAllowsHidden(plugin, "whoami") {
		t.Fatalf("expected wildcard hidden command match")
	}
	if commandAllowsHidden(&Plugin{}, "help") {
		t.Fatalf("expected false for plugin with no hidden commands")
	}
}

func TestHelpUsesBasicMarkdownOutputFormat(t *testing.T) {
	originalConnector := interfaces.Connector
	fake := &formatCaptureConnector{}
	interfaces.Connector = fake
	defer func() {
		interfaces.Connector = originalConnector
	}()

	r := makeFormatTestRobot(t)
	help(r, "help")
	if fake.lastFormat != robot.BasicMarkdown {
		t.Fatalf("help() sent format %v, want %v", fake.lastFormat, robot.BasicMarkdown)
	}
	if !strings.Contains(fake.lastMessage, "**Quick help**") {
		t.Fatalf("help() message missing quick help header: %q", fake.lastMessage)
	}
}

func TestHelpEscapesMarkdownSensitiveAliasByUsingInlineCode(t *testing.T) {
	originalConnector := interfaces.Connector
	fake := &formatCaptureConnector{}
	interfaces.Connector = fake
	defer func() {
		interfaces.Connector = originalConnector
	}()

	r := makeFormatTestRobot(t)
	w := getLockedWorker(r.tid)
	w.cfg.alias = '*'
	w.Unlock()

	help(r, "help")
	if !strings.Contains(fake.lastMessage, "`*help <keyword>`") {
		t.Fatalf("help() message missing literal '*' alias command: %q", fake.lastMessage)
	}
	if strings.Contains(fake.lastMessage, "*help <keyword>*") {
		t.Fatalf("help() message still formats alias command as emphasis: %q", fake.lastMessage)
	}
}

func TestFallbackUsesBasicMarkdownOutputFormat(t *testing.T) {
	originalConnector := interfaces.Connector
	fake := &formatCaptureConnector{}
	interfaces.Connector = fake
	defer func() {
		interfaces.Connector = originalConnector
	}()

	r := makeFormatTestRobot(t)
	fallback(r, "catchall", "tell me a jok")
	if fake.lastFormat != robot.BasicMarkdown {
		t.Fatalf("fallback() sent format %v, want %v", fake.lastFormat, robot.BasicMarkdown)
	}
	if !strings.Contains(fake.lastMessage, "I couldn't match") {
		t.Fatalf("fallback() message missing mismatch text: %q", fake.lastMessage)
	}
}

func TestReplyPreservesProtocolUserAndCanonicalUsername(t *testing.T) {
	originalConnector := interfaces.Connector
	fake := &formatCaptureConnector{}
	interfaces.Connector = fake
	defer func() {
		interfaces.Connector = originalConnector
	}()

	r := makeFormatTestRobot(t)
	r.ProtocolUser = "<U12345>"
	r.ProtocolChannel = "<C2468>"
	r.User = "alice"
	r.Channel = "general"

	if ret := r.Reply("hello"); ret != robot.Ok {
		t.Fatalf("Reply() ret = %v, want %v", ret, robot.Ok)
	}
	if fake.lastUserID != "<U12345>" {
		t.Fatalf("Reply() user id = %q, want %q", fake.lastUserID, "<U12345>")
	}
	if fake.lastUser != "alice" {
		t.Fatalf("Reply() username = %q, want %q", fake.lastUser, "alice")
	}
	if fake.lastChannel != "<C2468>" {
		t.Fatalf("Reply() channel = %q, want %q", fake.lastChannel, "<C2468>")
	}
	if fake.lastMessage != "hello" {
		t.Fatalf("Reply() message = %q, want %q", fake.lastMessage, "hello")
	}
}
