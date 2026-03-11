package main

import (
	"testing"

	"github.com/lnxjedi/gopherbot/robot"
)

func TestShouldUseDeterministicAdviceWrongChannel(t *testing.T) {
	advice := fallbackAdviceResponse{
		Advice: adviceWrongChannel,
		Elsewhere: []fallbackAdviceEntry{
			{PluginName: "theia-plugin", Command: "start-ide", Channels: []string{"devops"}, Score: 95},
		},
	}
	if !shouldUseDeterministicAdvice(advice) {
		t.Fatal("expected wrong-channel advice to use deterministic fast path")
	}
}

func TestShouldUseDeterministicAdviceStrongCloseMatch(t *testing.T) {
	advice := fallbackAdviceResponse{
		Advice: adviceCloseHere,
		Here: []fallbackAdviceEntry{
			{PluginName: "lists", Command: "add", Score: 96},
		},
	}
	if !shouldUseDeterministicAdvice(advice) {
		t.Fatal("expected very strong close match to use deterministic fast path")
	}
}

func TestShouldUseDeterministicAdviceAmbiguousCloseMatch(t *testing.T) {
	advice := fallbackAdviceResponse{
		Advice: adviceCloseHere,
		Here: []fallbackAdviceEntry{
			{PluginName: "lists", Command: "add", Score: 84},
			{PluginName: "lists", Command: "delete", Score: 78},
		},
	}
	if shouldUseDeterministicAdvice(advice) {
		t.Fatal("did not expect ambiguous close match to skip AI")
	}
}

func TestDeterministicNoMatchIncludesChannel(t *testing.T) {
	got := deterministicNoMatch("launch-server", "!", "general")
	if !contains(got, "#general") {
		t.Fatalf("response %q missing channel context", got)
	}
}

func TestAIRecoveryEnabledRequiresModelAndKey(t *testing.T) {
	cfg := fallbackConfig{OpenAIModel: "gpt-5.2-chat-latest"}
	r := &fakeConfigRobot{params: map[string]string{"OPENAI_KEY": "secret"}}
	if !aiRecoveryEnabled(r, cfg) {
		t.Fatal("expected AI recovery to be enabled with both model and key")
	}

	cfg.OpenAIModel = ""
	if aiRecoveryEnabled(r, cfg) {
		t.Fatal("did not expect AI recovery with blank model")
	}

	cfg.OpenAIModel = "gpt-5.2-chat-latest"
	r.params["OPENAI_KEY"] = ""
	if aiRecoveryEnabled(r, cfg) {
		t.Fatal("did not expect AI recovery with blank key")
	}
}

func TestSuggestedHelpTermsPrefersCommandAndKeywords(t *testing.T) {
	advice := fallbackAdviceResponse{
		Advice: adviceWrongChannel,
		Elsewhere: []fallbackAdviceEntry{
			{
				PluginName: "knock",
				Command:    "knock",
				Keywords:   []string{"joke", "knock"},
			},
		},
	}
	got := suggestedHelpTerms(advice)
	if len(got) != 1 {
		t.Fatalf("suggestedHelpTerms() = %#v, want only the top command term for wrong-channel advice", got)
	}
	if got[0] != "knock" {
		t.Fatalf("suggestedHelpTerms() first term = %q, want %q", got[0], "knock")
	}
}

func contains(haystack, needle string) bool {
	return len(needle) == 0 || (len(haystack) >= len(needle) && stringIndex(haystack, needle) >= 0)
}

func stringIndex(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}

type fakeConfigRobot struct {
	robot.Robot
	params map[string]string
}

func (r *fakeConfigRobot) GetParameter(name string) string {
	return r.params[name]
}
