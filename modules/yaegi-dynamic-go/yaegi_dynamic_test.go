package yaegidynamicgo

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/lnxjedi/gopherbot/robot"
)

type compiledConversationExchange struct {
	Human string `json:"human"`
	AI    string `json:"ai"`
}

type compiledConversationState struct {
	Profile    string                         `json:"profile"`
	Tokens     int                            `json:"tokens"`
	Owner      string                         `json:"owner"`
	Summary    string                         `json:"summary,omitempty"`
	Exchanges  []compiledConversationExchange `json:"exchanges"`
	Pending    []compiledPendingMessage       `json:"pending"`
	Processed  []string                       `json:"processed"`
	InProgress bool                           `json:"in_progress"`
	UpdatedAt  string                         `json:"updated_at"`
}

type compiledCompactionResult struct {
	State compiledConversationState
	Older []compiledConversationExchange
}

type compiledPendingMessage struct {
	MessageID string `json:"message_id"`
	User      string `json:"user"`
	Text      string `json:"text"`
	At        string `json:"at"`
}

type testLogger struct {
	lines []string
}

func (l *testLogger) Log(level robot.LogLevel, msg string, v ...interface{}) bool {
	l.lines = append(l.lines, msg)
	return true
}

func compiledCompact(state compiledConversationState) (compiledConversationState, []compiledConversationExchange) {
	older := append([]compiledConversationExchange(nil), state.Exchanges[:1]...)
	state.Exchanges = append([]compiledConversationExchange(nil), state.Exchanges[1:]...)
	return state, older
}

func compiledCompactInternal(state compiledConversationState, force bool) (compiledConversationState, []compiledConversationExchange) {
	if !force {
		return state, nil
	}
	return compiledCompact(state)
}

func compiledForceCompact(state compiledConversationState) (compiledConversationState, []compiledConversationExchange) {
	return compiledCompactInternal(state, true)
}

func compiledCompactWrapped(state compiledConversationState) compiledCompactionResult {
	compacted, older := compiledForceCompact(state)
	return compiledCompactionResult{
		State: compacted,
		Older: older,
	}
}

func TestCompiledGoMultiReturnStateAndSliceWorks(t *testing.T) {
	state := compiledConversationState{
		Profile: "default",
		Tokens:  42,
		Owner:   "alice",
		Exchanges: []compiledConversationExchange{
			{Human: "alice says: one", AI: "ai: one"},
			{Human: "alice says: two", AI: "ai: two"},
		},
	}
	compacted, older := compiledForceCompact(state)
	if len(compacted.Exchanges) != 1 || len(older) != 1 {
		t.Fatalf("compiled multi-return result = %+v / %+v, want 1 recent and 1 older", compacted, older)
	}

	wrapped := compiledCompactWrapped(state)
	if len(wrapped.State.Exchanges) != 1 || len(wrapped.Older) != 1 {
		t.Fatalf("compiled wrapped result = %+v, want 1 recent and 1 older", wrapped)
	}
}

func TestRunPluginHandlerYaegiMultiReturnPanics(t *testing.T) {
	pluginPath := writeTempPlugin(t, yaegiPluginWithMultiReturn())
	logger := &testLogger{}

	ret, err := RunPluginHandler(pluginPath, "multi-return-repro", nil, nil, logger, false, "compact")
	if ret != robot.MechanismFail {
		t.Fatalf("RunPluginHandler ret = %v, want %v", ret, robot.MechanismFail)
	}
	if err == nil {
		t.Fatal("expected RunPluginHandler to return an error for yaegi multi-return repro")
	}
	if !strings.Contains(err.Error(), "reflect.Set") {
		t.Fatalf("expected reflect.Set panic in error, got %v", err)
	}
	if !strings.Contains(err.Error(), "not assignable") {
		t.Fatalf("expected assignment detail in error, got %v", err)
	}
}

func TestRunPluginHandlerYaegiWrappedReturnWorks(t *testing.T) {
	pluginPath := writeTempPlugin(t, yaegiPluginWithWrappedReturn())
	logger := &testLogger{}

	ret, err := RunPluginHandler(pluginPath, "wrapped-return-repro", nil, nil, logger, false, "compact")
	if err != nil {
		t.Fatalf("RunPluginHandler wrapped return error = %v", err)
	}
	if ret != robot.Normal {
		t.Fatalf("RunPluginHandler wrapped return ret = %v, want %v", ret, robot.Normal)
	}
}

func TestRunPluginHandlerYaegiRobotHelpMetadataMethodCompiles(t *testing.T) {
	pluginPath := writeTempPlugin(t, yaegiPluginUsingHelpMetadata())
	logger := &testLogger{}

	ret, err := RunPluginHandler(pluginPath, "help-metadata-repro", nil, nil, logger, false, "catchall")
	if err != nil {
		t.Fatalf("RunPluginHandler help metadata error = %v", err)
	}
	if ret != robot.Normal {
		t.Fatalf("RunPluginHandler help metadata ret = %v, want %v", ret, robot.Normal)
	}
}

func writeTempPlugin(t *testing.T, src string) string {
	t.Helper()
	ensureYaegiInitialized(t)
	pluginPath := filepath.Join(t.TempDir(), "plugin.go")
	if err := os.WriteFile(pluginPath, []byte(src), 0o600); err != nil {
		t.Fatalf("write temp plugin: %v", err)
	}
	return pluginPath
}

func ensureYaegiInitialized(t *testing.T) {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
	tmpGoPath := filepath.Join(t.TempDir(), "gopath")
	robotSrcPath := filepath.Join(tmpGoPath, "src", "github.com", "lnxjedi", "gopherbot", "robot")
	if err := os.MkdirAll(filepath.Dir(robotSrcPath), 0o755); err != nil {
		t.Fatalf("mkdir gopath parents: %v", err)
	}
	if err := copyDir(filepath.Join(repoRoot, "robot"), robotSrcPath); err != nil {
		t.Fatalf("copy robot package into temp gopath: %v", err)
	}
	goPath = tmpGoPath
	initErr = nil
}

func yaegiPluginWithMultiReturn() string {
	return strings.Join([]string{
		"package main",
		"",
		"import \"github.com/lnxjedi/gopherbot/robot\"",
		"",
		"const defaultProfile = \"default\"",
		"const defaultMaxRecentExchanges = 12",
		"",
		"type aiProfile struct {",
		"    MaxContext int `json:\"MaxContext\"`",
		"}",
		"",
		"type aiConfig struct {",
		"    Profiles map[string]aiProfile `json:\"Profiles\"`",
		"    MaxRecentExchanges int `json:\"MaxRecentExchanges\"`",
		"    CompactionTriggerTokens int `json:\"CompactionTriggerTokens\"`",
		"    SummaryBudgetTokens int `json:\"SummaryBudgetTokens\"`",
		"}",
		"",
		"type conversationExchange struct {",
		"    Human string `json:\"human\"`",
		"    AI string `json:\"ai\"`",
		"}",
		"",
		"type conversationState struct {",
		"    Profile string `json:\"profile\"`",
		"    Tokens int `json:\"tokens\"`",
		"    Owner string `json:\"owner\"`",
		"    Summary string `json:\"summary,omitempty\"`",
		"    Exchanges []conversationExchange `json:\"exchanges\"`",
		"    Pending []pendingMessage `json:\"pending\"`",
		"    Processed []string `json:\"processed\"`",
		"    InProgress bool `json:\"in_progress\"`",
		"    UpdatedAt string `json:\"updated_at\"`",
		"}",
		"",
		"type pendingMessage struct {",
		"    MessageID string `json:\"message_id\"`",
		"    User string `json:\"user\"`",
		"    Text string `json:\"text\"`",
		"    At string `json:\"at\"`",
		"}",
		"",
		"func estimateTokens(s string) int {",
		"    return len(s)",
		"}",
		"",
		"func estimateConversationTokens(exchanges []conversationExchange) int {",
		"    total := 0",
		"    for _, ex := range exchanges {",
		"        total += len(ex.Human) + len(ex.AI)",
		"    }",
		"    return total",
		"}",
		"",
		"func resolveProfile(name string, cfg aiConfig) aiProfile {",
		"    if cfg.Profiles != nil {",
		"        if profile, ok := cfg.Profiles[name]; ok {",
		"            return profile",
		"        }",
		"        if profile, ok := cfg.Profiles[defaultProfile]; ok {",
		"            return profile",
		"        }",
		"    }",
		"    return aiProfile{}",
		"}",
		"",
		"func resolveCompactionTriggerTokens(profile aiProfile) int {",
		"    if profile.MaxContext <= 0 {",
		"        return 6144",
		"    }",
		"    trigger := profile.MaxContext - 1024",
		"    if trigger < 1024 {",
		"        return profile.MaxContext",
		"    }",
		"    return trigger",
		"}",
		"",
		"func mergeDeterministicSummary(existing string, older []conversationExchange, summaryBudgetTokens int) string {",
		"    if len(older) == 0 {",
		"        return existing",
		"    }",
		"    return existing + \" summary\"",
		"}",
		"",
		"func compactConversationDeterministicInternal(state conversationState, cfg aiConfig, force bool) (conversationState, []conversationExchange) {",
		"    maxRecent := cfg.MaxRecentExchanges",
		"    if maxRecent <= 0 {",
		"        maxRecent = defaultMaxRecentExchanges",
		"    }",
		"    keepRecent := maxRecent",
		"    if keepRecent > len(state.Exchanges) {",
		"        keepRecent = len(state.Exchanges)",
		"    }",
		"    if force && len(state.Exchanges) > 1 && keepRecent == len(state.Exchanges) {",
		"        keepRecent = len(state.Exchanges) - 1",
		"    }",
		"    if len(state.Exchanges) <= keepRecent {",
		"        if state.Tokens <= 0 {",
		"            state.Tokens = estimateConversationTokens(state.Exchanges) + estimateTokens(state.Summary)",
		"        }",
		"        return state, nil",
		"    }",
		"    trigger := cfg.CompactionTriggerTokens",
		"    if trigger <= 0 {",
		"        trigger = resolveCompactionTriggerTokens(resolveProfile(state.Profile, cfg))",
		"    }",
		"    if trigger <= 0 {",
		"        trigger = 6144",
		"    }",
		"    if state.Tokens <= 0 {",
		"        state.Tokens = estimateConversationTokens(state.Exchanges) + estimateTokens(state.Summary)",
		"    }",
		"    if !force && state.Tokens < trigger {",
		"        return state, nil",
		"    }",
		"    split := len(state.Exchanges) - keepRecent",
		"    older := state.Exchanges[:split]",
		"    recent := state.Exchanges[split:]",
		"    state.Summary = mergeDeterministicSummary(state.Summary, older, cfg.SummaryBudgetTokens)",
		"    state.Exchanges = append([]conversationExchange(nil), recent...)",
		"    state.Tokens = estimateConversationTokens(state.Exchanges) + estimateTokens(state.Summary)",
		"    return state, append([]conversationExchange(nil), older...)",
		"}",
		"",
		"func forceCompactConversationDeterministic(state conversationState, cfg aiConfig) (conversationState, []conversationExchange) {",
		"    return compactConversationDeterministicInternal(state, cfg, true)",
		"}",
		"",
		"func PluginHandler(r robot.Robot, command string, args ...string) robot.TaskRetVal {",
		"    cfg := aiConfig{MaxRecentExchanges: 12, SummaryBudgetTokens: 200}",
		"    state := conversationState{",
		"        Profile: \"default\",",
		"        Tokens: 42,",
		"        Owner: \"alice\",",
		"        Summary: \"older\",",
		"        Exchanges: []conversationExchange{",
		"            {Human: \"alice says: one\", AI: \"ai: one\"},",
		"            {Human: \"alice says: two\", AI: \"ai: two\"},",
		"            {Human: \"alice says: three\", AI: \"ai: three\"},",
		"        },",
		"    }",
		"    compacted, older := forceCompactConversationDeterministic(state, cfg)",
		"    if len(compacted.Exchanges) != 2 || len(older) != 1 {",
		"        return robot.Fail",
		"    }",
		"    return robot.Normal",
		"}",
	}, "\n")
}

func yaegiPluginWithWrappedReturn() string {
	return strings.Join([]string{
		"package main",
		"",
		"import \"github.com/lnxjedi/gopherbot/robot\"",
		"",
		"const defaultProfile = \"default\"",
		"const defaultMaxRecentExchanges = 12",
		"",
		"type aiProfile struct {",
		"    MaxContext int `json:\"MaxContext\"`",
		"}",
		"",
		"type aiConfig struct {",
		"    Profiles map[string]aiProfile `json:\"Profiles\"`",
		"    MaxRecentExchanges int `json:\"MaxRecentExchanges\"`",
		"    CompactionTriggerTokens int `json:\"CompactionTriggerTokens\"`",
		"    SummaryBudgetTokens int `json:\"SummaryBudgetTokens\"`",
		"}",
		"",
		"type conversationExchange struct {",
		"    Human string `json:\"human\"`",
		"    AI string `json:\"ai\"`",
		"}",
		"",
		"type conversationState struct {",
		"    Profile string `json:\"profile\"`",
		"    Tokens int `json:\"tokens\"`",
		"    Owner string `json:\"owner\"`",
		"    Summary string `json:\"summary,omitempty\"`",
		"    Exchanges []conversationExchange `json:\"exchanges\"`",
		"    Pending []pendingMessage `json:\"pending\"`",
		"    Processed []string `json:\"processed\"`",
		"    InProgress bool `json:\"in_progress\"`",
		"    UpdatedAt string `json:\"updated_at\"`",
		"}",
		"",
		"type pendingMessage struct {",
		"    MessageID string `json:\"message_id\"`",
		"    User string `json:\"user\"`",
		"    Text string `json:\"text\"`",
		"    At string `json:\"at\"`",
		"}",
		"",
		"type compactionResult struct {",
		"    State conversationState",
		"    Older []conversationExchange",
		"}",
		"",
		"func estimateTokens(s string) int {",
		"    return len(s)",
		"}",
		"",
		"func estimateConversationTokens(exchanges []conversationExchange) int {",
		"    total := 0",
		"    for _, ex := range exchanges {",
		"        total += len(ex.Human) + len(ex.AI)",
		"    }",
		"    return total",
		"}",
		"",
		"func resolveProfile(name string, cfg aiConfig) aiProfile {",
		"    if cfg.Profiles != nil {",
		"        if profile, ok := cfg.Profiles[name]; ok {",
		"            return profile",
		"        }",
		"        if profile, ok := cfg.Profiles[defaultProfile]; ok {",
		"            return profile",
		"        }",
		"    }",
		"    return aiProfile{}",
		"}",
		"",
		"func resolveCompactionTriggerTokens(profile aiProfile) int {",
		"    if profile.MaxContext <= 0 {",
		"        return 6144",
		"    }",
		"    trigger := profile.MaxContext - 1024",
		"    if trigger < 1024 {",
		"        return profile.MaxContext",
		"    }",
		"    return trigger",
		"}",
		"",
		"func mergeDeterministicSummary(existing string, older []conversationExchange, summaryBudgetTokens int) string {",
		"    if len(older) == 0 {",
		"        return existing",
		"    }",
		"    return existing + \" summary\"",
		"}",
		"",
		"func compactConversationDeterministicInternal(state conversationState, cfg aiConfig, force bool) compactionResult {",
		"    maxRecent := cfg.MaxRecentExchanges",
		"    if maxRecent <= 0 {",
		"        maxRecent = defaultMaxRecentExchanges",
		"    }",
		"    keepRecent := maxRecent",
		"    if keepRecent > len(state.Exchanges) {",
		"        keepRecent = len(state.Exchanges)",
		"    }",
		"    if force && len(state.Exchanges) > 1 && keepRecent == len(state.Exchanges) {",
		"        keepRecent = len(state.Exchanges) - 1",
		"    }",
		"    if len(state.Exchanges) <= keepRecent {",
		"        if state.Tokens <= 0 {",
		"            state.Tokens = estimateConversationTokens(state.Exchanges) + estimateTokens(state.Summary)",
		"        }",
		"        return compactionResult{State: state}",
		"    }",
		"    trigger := cfg.CompactionTriggerTokens",
		"    if trigger <= 0 {",
		"        trigger = resolveCompactionTriggerTokens(resolveProfile(state.Profile, cfg))",
		"    }",
		"    if trigger <= 0 {",
		"        trigger = 6144",
		"    }",
		"    if state.Tokens <= 0 {",
		"        state.Tokens = estimateConversationTokens(state.Exchanges) + estimateTokens(state.Summary)",
		"    }",
		"    if !force && state.Tokens < trigger {",
		"        return compactionResult{State: state}",
		"    }",
		"    split := len(state.Exchanges) - keepRecent",
		"    older := state.Exchanges[:split]",
		"    recent := state.Exchanges[split:]",
		"    state.Summary = mergeDeterministicSummary(state.Summary, older, cfg.SummaryBudgetTokens)",
		"    state.Exchanges = append([]conversationExchange(nil), recent...)",
		"    state.Tokens = estimateConversationTokens(state.Exchanges) + estimateTokens(state.Summary)",
		"    return compactionResult{State: state, Older: append([]conversationExchange(nil), older...)}",
		"}",
		"",
		"func forceCompactConversationDeterministic(state conversationState, cfg aiConfig) compactionResult {",
		"    return compactConversationDeterministicInternal(state, cfg, true)",
		"}",
		"",
		"func PluginHandler(r robot.Robot, command string, args ...string) robot.TaskRetVal {",
		"    cfg := aiConfig{MaxRecentExchanges: 12, SummaryBudgetTokens: 200}",
		"    state := conversationState{",
		"        Profile: \"default\",",
		"        Tokens: 42,",
		"        Owner: \"alice\",",
		"        Summary: \"older\",",
		"        Exchanges: []conversationExchange{",
		"            {Human: \"alice says: one\", AI: \"ai: one\"},",
		"            {Human: \"alice says: two\", AI: \"ai: two\"},",
		"            {Human: \"alice says: three\", AI: \"ai: three\"},",
		"        },",
		"    }",
		"    result := forceCompactConversationDeterministic(state, cfg)",
		"    if len(result.State.Exchanges) != 2 || len(result.Older) != 1 {",
		"        return robot.Fail",
		"    }",
		"    return robot.Normal",
		"}",
	}, "\n")
}

func yaegiPluginUsingHelpMetadata() string {
	return strings.Join([]string{
		"package main",
		"",
		"import \"github.com/lnxjedi/gopherbot/robot\"",
		"",
		"func PluginHandler(r robot.Robot, command string, args ...string) robot.TaskRetVal {",
		"    _ = func(rb robot.Robot) string {",
		"        return rb.GetHelpMetadata(\"launch-server\")",
		"    }",
		"    return robot.Normal",
		"}",
	}, "\n")
}
