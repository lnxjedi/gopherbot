package bot

import (
	"reflect"
	"testing"
)

func TestResolvePrimaryProtocol(t *testing.T) {
	tests := []struct {
		name         string
		primary      string
		legacy       string
		wantProtocol string
		wantConflict bool
	}{
		{
			name:         "primary wins with no legacy",
			primary:      "ssh",
			legacy:       "",
			wantProtocol: "ssh",
			wantConflict: false,
		},
		{
			name:         "legacy used when primary missing",
			primary:      "",
			legacy:       "slack",
			wantProtocol: "slack",
			wantConflict: false,
		},
		{
			name:         "equal values no conflict",
			primary:      "ssh",
			legacy:       "ssh",
			wantProtocol: "ssh",
			wantConflict: false,
		},
		{
			name:         "case-only difference no conflict",
			primary:      "SSH",
			legacy:       "ssh",
			wantProtocol: "SSH",
			wantConflict: false,
		},
		{
			name:         "different values conflict",
			primary:      "ssh",
			legacy:       "slack",
			wantProtocol: "ssh",
			wantConflict: true,
		},
		{
			name:         "both empty",
			primary:      "",
			legacy:       "",
			wantProtocol: "",
			wantConflict: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotProtocol, gotConflict := resolvePrimaryProtocol(tc.primary, tc.legacy)
			if gotProtocol != tc.wantProtocol {
				t.Fatalf("resolvePrimaryProtocol() protocol = %q, want %q", gotProtocol, tc.wantProtocol)
			}
			if gotConflict != tc.wantConflict {
				t.Fatalf("resolvePrimaryProtocol() conflict = %t, want %t", gotConflict, tc.wantConflict)
			}
		})
	}
}

func TestNormalizeSecondaryProtocols(t *testing.T) {
	primary := "ssh"
	in := []string{
		"slack",
		"ssh",
		"  test  ",
		"SLACK",
		"",
		"   ",
		"rocket",
		"test",
	}

	got := normalizeSecondaryProtocols(primary, in)
	want := []string{"slack", "test", "rocket"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeSecondaryProtocols() = %#v, want %#v", got, want)
	}
}

func TestIsValidRosterUserName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "lowercase", input: "david", want: true},
		{name: "digits allowed", input: "david2", want: true},
		{name: "uppercase rejected", input: "David", want: false},
		{name: "empty rejected", input: "", want: false},
		{name: "spaces rejected", input: "   ", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isValidRosterUserName(tc.input); got != tc.want {
				t.Fatalf("isValidRosterUserName(%q) = %t, want %t", tc.input, got, tc.want)
			}
		})
	}
}
