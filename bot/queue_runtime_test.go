package bot

import (
	"reflect"
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
	tests := [][]byte{
		[]byte("short"),
		[]byte("1104df4c-feeb-43ab-8c85-83663288cea9\targ"),
		[]byte("xxxxxxxx-feeb-43ab-8c85-83663288cea9 arg"),
		[]byte("1104df4c-feeb-43ab-8c85-83663288cea9 'unterminated"),
	}
	for _, tc := range tests {
		if _, _, err := parseQueueBody(tc); err == nil {
			t.Fatalf("parseQueueBody(%q) returned nil error", string(tc))
		}
	}
}
