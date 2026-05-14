package gcloud

import "testing"

func TestNormalizeSubscriptionID(t *testing.T) {
	tests := map[string]string{
		"job-triggers-pull": "job-triggers-pull",
		"projects/example/subscriptions/job-triggers-pull":                   "job-triggers-pull",
		"//pubsub.googleapis.com/projects/p/subscriptions/job-triggers-pull": "job-triggers-pull",
		"  subscriptions/job-triggers-pull  ":                                "job-triggers-pull",
		"":                                                                   "",
	}
	for in, want := range tests {
		if got := normalizeSubscriptionID(in); got != want {
			t.Fatalf("normalizeSubscriptionID(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalizeMaxBodySize(t *testing.T) {
	tests := []struct {
		name string
		in   int
		want int
	}{
		{name: "positive value", in: 8192, want: 8192},
		{name: "zero defaults", in: 0, want: defaultMaxBodySize},
		{name: "negative defaults", in: -1, want: defaultMaxBodySize},
	}

	for _, tc := range tests {
		got := normalizeMaxBodySize(tc.in)
		if got != tc.want {
			t.Fatalf("%s: max body size = %d, want %d", tc.name, got, tc.want)
		}
	}
}
