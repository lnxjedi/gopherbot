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
