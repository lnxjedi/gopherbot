package bot

import "testing"

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
		PluginName: "knock",
		Command:    "knock",
		Usage:      "tell me a knock-knock joke",
		Summary:    "Starts an interactive knock-knock joke.",
		Keywords:   []string{"joke", "knock"},
	}
	if got := scoreHelpCommandMatch(entry, "knok"); got < 80 {
		t.Fatalf("scoreHelpCommandMatch() expected strong typo score for knok, got %d", got)
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

func TestHiddenSlashBotExample(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"(bot) list lists", "/(bot) list lists"},
		{"(bot), list lists", "/(bot) list lists"},
		{"(alias) ping", "(alias) ping"},
		{"ping", "ping"},
	}
	for _, tc := range cases {
		if got := hiddenSlashBotExample(tc.in); got != tc.want {
			t.Fatalf("hiddenSlashBotExample(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
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
