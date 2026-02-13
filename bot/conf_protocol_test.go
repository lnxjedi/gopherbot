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
		"terminal",
		" TERMINAL ",
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

func TestSecondaryIncludesPrimary(t *testing.T) {
	tests := []struct {
		name      string
		primary   string
		secondary []string
		want      bool
	}{
		{
			name:      "direct match",
			primary:   "ssh",
			secondary: []string{"slack", "ssh"},
			want:      true,
		},
		{
			name:      "case-insensitive with spaces",
			primary:   "ssh",
			secondary: []string{" SLACK ", "  SSh "},
			want:      true,
		},
		{
			name:      "no match",
			primary:   "ssh",
			secondary: []string{"slack", "terminal"},
			want:      false,
		},
		{
			name:      "empty primary",
			primary:   "",
			secondary: []string{"ssh"},
			want:      false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := secondaryIncludesPrimary(tc.primary, tc.secondary)
			if got != tc.want {
				t.Fatalf("secondaryIncludesPrimary(%q, %#v) = %t, want %t", tc.primary, tc.secondary, got, tc.want)
			}
		})
	}
}

func TestSecondaryIncludesProtocol(t *testing.T) {
	tests := []struct {
		name      string
		protocol  string
		secondary []string
		want      bool
	}{
		{
			name:      "direct match",
			protocol:  "terminal",
			secondary: []string{"terminal", "ssh"},
			want:      true,
		},
		{
			name:      "case-insensitive match",
			protocol:  "terminal",
			secondary: []string{"TeRmiNal"},
			want:      true,
		},
		{
			name:      "no match",
			protocol:  "terminal",
			secondary: []string{"ssh", "slack"},
			want:      false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := secondaryIncludesProtocol(tc.protocol, tc.secondary)
			if got != tc.want {
				t.Fatalf("secondaryIncludesProtocol(%q, %#v) = %t, want %t", tc.protocol, tc.secondary, got, tc.want)
			}
		})
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

func TestRoleLabel(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "", want: "Protocol"},
		{in: "primary", want: "Primary"},
		{in: "secondary", want: "Secondary"},
	}
	for _, tc := range tests {
		if got := roleLabel(tc.in); got != tc.want {
			t.Fatalf("roleLabel(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestMergeUserMapsWithOverride(t *testing.T) {
	base := map[string]string{
		"alice": "U001",
		"bob":   "U002",
	}
	override := map[string]string{
		"bob":   "U099",
		"carol": "U003",
	}

	got := mergeUserMapsWithOverride("slack", base, override, "test")

	want := map[string]string{
		"alice": "U001",
		"bob":   "U099",
		"carol": "U003",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mergeUserMapsWithOverride() = %#v, want %#v", got, want)
	}
	if base["bob"] != "U002" {
		t.Fatalf("mergeUserMapsWithOverride() mutated base map, base[bob]=%q", base["bob"])
	}
}
