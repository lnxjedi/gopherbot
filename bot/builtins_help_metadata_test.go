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

func TestFirstHelpLineUsageAndSummary(t *testing.T) {
	lines := []string{"(alias) add <item> to the <type> list - add something to a list"}
	if got := firstHelpLineAsUsage(lines); got != "(alias) add <item> to the <type> list" {
		t.Fatalf("firstHelpLineAsUsage() = %q", got)
	}
	if got := firstHelpLineSummary(lines); got != "add something to a list" {
		t.Fatalf("firstHelpLineSummary() = %q", got)
	}
}

func TestScoreHelpCommandMatch(t *testing.T) {
	entry := helpCommandMetadata{
		PluginName: "lists",
		Command:    "add",
		Usage:      "(alias) add <item> to the <type> list",
		Summary:    "Adds an item to a named list.",
		Keywords:   []string{"list", "lists", "add"},
		Helptext:   []string{"(alias) add <item> to the <type> list - add something to a list"},
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
