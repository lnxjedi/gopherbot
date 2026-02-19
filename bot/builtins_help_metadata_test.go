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
