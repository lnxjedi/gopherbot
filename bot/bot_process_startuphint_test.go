package bot

import "testing"

func TestStartupSSHHint(t *testing.T) {
	tests := []struct {
		name     string
		mode     string
		protocol string
		admin    []string
		wantHint string
	}{
		{
			name:     "demo_uses_first_admin_user",
			mode:     "demo",
			protocol: "ssh",
			admin:    []string{"david"},
			wantHint: "Running in DEMO mode; in another terminal window, connect as david with 'bot-ssh -d david'",
		},
		{
			name:     "demo_falls_back_to_alice_when_admin_empty",
			mode:     "demo",
			protocol: "ssh",
			admin:    []string{"   "},
			wantHint: "Running in DEMO mode; in another terminal window, connect as alice with 'bot-ssh -d alice'",
		},
		{
			name:     "test_dev_uses_first_admin_user",
			mode:     "test-dev",
			protocol: "ssh",
			admin:    []string{"alice"},
			wantHint: "Default robot running in test-dev mode with ssh-connector; connect with e.g. 'bot-ssh -d alice'",
		},
		{
			name:     "non_ssh_protocol_has_no_hint",
			mode:     "demo",
			protocol: "terminal",
			admin:    []string{"alice"},
			wantHint: "",
		},
		{
			name:     "non_demo_mode_has_no_hint",
			mode:     "production",
			protocol: "ssh",
			admin:    []string{"alice"},
			wantHint: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := startupSSHHint(tt.mode, tt.protocol, tt.admin)
			if got != tt.wantHint {
				t.Fatalf("startupSSHHint(%q,%q,%v) = %q, want %q", tt.mode, tt.protocol, tt.admin, got, tt.wantHint)
			}
		})
	}
}
