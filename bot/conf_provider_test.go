package bot

import "testing"

func TestProviderConfigDirectoryForKey(t *testing.T) {
	tests := []struct {
		key      string
		wantDir  string
		wantBool bool
	}{
		{key: "BrainConfig", wantDir: "brains", wantBool: true},
		{key: "HistoryConfig", wantDir: "history", wantBool: true},
		{key: "ProtocolConfig", wantDir: "", wantBool: false},
	}

	for _, tc := range tests {
		gotDir, gotBool := providerConfigDirectoryForKey(tc.key)
		if gotDir != tc.wantDir || gotBool != tc.wantBool {
			t.Fatalf("providerConfigDirectoryForKey(%q) = (%q,%t), want (%q,%t)", tc.key, gotDir, gotBool, tc.wantDir, tc.wantBool)
		}
	}
}
