package main

import (
	"reflect"
	"testing"
	"time"
)

func TestBuildSearchQueries(t *testing.T) {
	t.Parallel()

	got := buildSearchQueries(defaultPhrase, []string{"opsbot", "gopherbot", "opsbot"})
	want := []string{
		"\"No command matched in channel\" has:thread from:opsbot",
		"\"No command matched in channel\" from:opsbot",
		"\"No command matched in channel\" has:thread from:gopherbot",
		"\"No command matched in channel\" from:gopherbot",
		"\"No command matched in channel\" has:thread",
		"\"No command matched in channel\"",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildSearchQueries() mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

func TestExpandPath(t *testing.T) {
	t.Parallel()

	got, err := expandPath("~/tmp/slack-error-commands.out")
	if err != nil {
		t.Fatalf("expandPath() unexpected error: %v", err)
	}
	if got == "~/tmp/slack-error-commands.out" {
		t.Fatalf("expandPath() did not expand home dir: %q", got)
	}
}

func TestParseSlackTimestamp(t *testing.T) {
	t.Parallel()

	got := parseSlackTimestamp("1700000000.123456")
	want := time.Unix(1700000000, 123456000).UTC()
	if !got.Equal(want) {
		t.Fatalf("parseSlackTimestamp() = %s, want %s", got.Format(time.RFC3339Nano), want.Format(time.RFC3339Nano))
	}
}

func TestThreadTimestampFromPermalink(t *testing.T) {
	t.Parallel()

	got := threadTimestampFromPermalink("https://example.slack.com/archives/C7VN49LH3/p1756229326022309?thread_ts=1756229324.706879&cid=C7VN49LH3")
	want := "1756229324.706879"
	if got != want {
		t.Fatalf("threadTimestampFromPermalink() = %q, want %q", got, want)
	}
}

func TestThreadRootTimestampFallsBackToReplyTS(t *testing.T) {
	t.Parallel()

	entry := reportEntry{
		ReplyTS:   "1756229326.022309",
		Permalink: "https://example.slack.com/archives/C7VN49LH3/p1756229326022309",
	}
	if got := threadRootTimestamp(entry); got != entry.ReplyTS {
		t.Fatalf("threadRootTimestamp() = %q, want fallback %q", got, entry.ReplyTS)
	}
}
