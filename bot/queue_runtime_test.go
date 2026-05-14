package bot

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseQueueBodyNoArgs(t *testing.T) {
	id, args, err := parseQueueBody([]byte("1104df4c-feeb-43ab-8c85-83663288cea9"))
	if err != nil {
		t.Fatalf("parseQueueBody returned error: %v", err)
	}
	if id != "1104df4c-feeb-43ab-8c85-83663288cea9" {
		t.Fatalf("id = %q", id)
	}
	if len(args) != 0 {
		t.Fatalf("args = %#v, want none", args)
	}
}

func TestParseQueueBodyShellEscapedArgs(t *testing.T) {
	id, args, err := parseQueueBody([]byte("1104df4c-feeb-43ab-8c85-83663288cea9 alpha two\\ words 'three four'"))
	if err != nil {
		t.Fatalf("parseQueueBody returned error: %v", err)
	}
	if id != "1104df4c-feeb-43ab-8c85-83663288cea9" {
		t.Fatalf("id = %q", id)
	}
	want := []string{"alpha", "two words", "three four"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("args = %#v, want %#v", args, want)
	}
}

func TestParseQueueBodyRejectsMalformedBody(t *testing.T) {
	tests := []struct {
		name    string
		body    []byte
		errPart string
	}{
		{name: "too short", body: []byte("short"), errPart: "too short"},
		{name: "tab instead of space", body: []byte("1104df4c-feeb-43ab-8c85-83663288cea9\targ"), errPart: "not followed by a space"},
		{name: "invalid uuid", body: []byte("xxxxxxxx-feeb-43ab-8c85-83663288cea9 arg"), errPart: "invalid queue UUID prefix"},
		{name: "unterminated shell arg", body: []byte("1104df4c-feeb-43ab-8c85-83663288cea9 'unterminated"), errPart: "parsing shell-escaped queue arguments"},
	}
	for _, tc := range tests {
		_, _, err := parseQueueBody(tc.body)
		if err == nil {
			t.Fatalf("%s: parseQueueBody(%q) returned nil error", tc.name, string(tc.body))
		}
		if !strings.Contains(err.Error(), tc.errPart) {
			t.Fatalf("%s: parseQueueBody(%q) error = %q, want substring %q", tc.name, string(tc.body), err, tc.errPart)
		}
	}
}
