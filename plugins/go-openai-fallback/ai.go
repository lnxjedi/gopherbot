package main

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
)

const (
	shortTermMemoryPrefix      = "openai-fallback-conversation"
	shortTermMemoryDebugPrefix = "openai-fallback-debug"
	conversationDatumPrefix    = "openaifallback:conversation:v2"
	conversationIndexDatumKey  = "openaifallback:conversation:index:v1"
	conversationIndexVersion   = 1
	defaultProfile             = "default"
	maxPendingMessages         = 24
	maxProcessedMessages       = 48
	maxStoredExchanges         = 48
	defaultMaxRecentExchanges  = 12
	defaultSummaryBudgetTokens = 768
	openAIChatCompletionsURL   = "https://api.openai.com/v1/chat/completions"
	defaultChunkSoftLimit      = 420
	defaultChunkHardLimit      = 620
	delayedWaitNoticeDelay     = 900 * time.Millisecond
	streamProgressNoticeDelay  = 1300 * time.Millisecond
)

const (
	defaultStandardSystemPrompt = `You are an AI assistant participating in a multi-user chat conversation.
Messages are provided with speaker prefixes like "username says: ...".
Use these prefixes to identify who is speaking.
When replying to a specific person, address them as "@username" naturally.
If users are mainly talking to each other and no bot response is needed, keep your response minimal and non-intrusive, or reply with "(no response)".
Do not echo speaker prefixes unless it helps clarity.`
	defaultCustomSystemPrompt = "You are helpful, concise, and collaborative."
	// See OpenAI reasoning best practices: "Formatting re-enabled" should be the first
	// line when markdown formatting is desired from reasoning-capable models.
	basicMarkdownSystemPrefix = "Formatting re-enabled\n" +
		"Respond using BasicMarkdown v1-compatible output only.\n" +
		"Allowed constructs: paragraphs, bold (**), italic (*), inline code (`), fenced code blocks (```), " +
		"block quotes (>), unordered lists (-), links [label](url), @username mentions, and :emoji: shortcodes.\n" +
		"Avoid headings (#), ordered lists (1.), tables, HTML tags, and platform-specific markdown."
)

var defaultConfig = []byte(`
---
AllowDirect: true
AllChannels: true
CatchAll: true
Commands:
- Command: "debug"
  Regex: '(?i:d(ebug[ -]ai)?)'
  Keywords: [ "ai", "debug" ]
  Usage: "(bot), debug-ai"
  Summary: "enable debug output during AI interactions"
- Command: "compact-model"
  Regex: '(?i:(?:compact|summarize)[ -]ai[ -](?:model|api))'
  Keywords: [ "ai", "compact", "model" ]
  Usage: "(bot), compact-ai-model"
  Summary: "admin-only: force deterministic + model-assisted compaction for this AI context"
- Command: "compact"
  Regex: '(?i:(?:compact|summarize)[ -]ai)'
  Keywords: [ "ai", "compact" ]
  Usage: "(bot), compact-ai"
  Summary: "admin-only: force deterministic compaction for this AI context"
- Command: "close"
  Regex: '(?i:(?:dismiss|banish|close|stop|deactivate|disengage|dispel|reset)[ -]ai)'
  Keywords: [ "ai", "stop" ]
  Usage: "(bot), stop-ai"
  Summary: "stop an AI conversation"
- Command: "status"
  Regex: '^\?$'
  Keywords: [ "ai", "status" ]
  Usage: "(bot), ?"
  Summary: "show AI conversation status in thread"
- Command: "status"
  Regex: '(?i:ai[ -]status)'
  Keywords: [ "ai", "status" ]
  Usage: "(bot), ai-status"
  Summary: "show AI conversation status in thread"
AdminCommands:
- compact
- compact-model
Config:
  WaitMessages:
  - "hold on a moment while I think this through"
  - "working on that now"
  - "thinking through the details"
  - "just a moment while I work on a response"
  DrawMessages:
  - "working on an image"
  - "drawing now"
  - "rendering your request"
  CompactionTriggerTokens: 6144
  MaxRecentExchanges: 12
  SummaryBudgetTokens: 768
  EnableModelCompaction: false
  Profiles:
    "default":
      "params":
        "model": "gpt-5.2-chat-latest"
        "temperature": 0.7
      "SystemPrompt":
        "Standard": |
          You are an AI assistant participating in a multi-user chat conversation.
          Messages are provided with speaker prefixes like "username says: ...".
          Use these prefixes to identify who is speaking.
          When replying to a specific person, address them as "@username" naturally.
          If users are mainly talking to each other and no bot response is needed, keep your response minimal and non-intrusive, or reply with "(no response)".
          Do not echo speaker prefixes unless it helps clarity.
        "Custom": |
          You are helpful, concise, and collaborative.
      "max_context": 7168
`)

type systemPromptConfig struct {
	Standard string `json:"Standard"`
	Custom   string `json:"Custom"`
}

type aiProfile struct {
	Params       map[string]interface{} `json:"params"`
	SystemPrompt systemPromptConfig     `json:"SystemPrompt"`
	MaxContext   int                    `json:"max_context"`
}

type aiConfig struct {
	WaitMessages            []string             `json:"WaitMessages"`
	DrawMessages            []string             `json:"DrawMessages"`
	Profiles                map[string]aiProfile `json:"Profiles"`
	CompactionTriggerTokens int                  `json:"CompactionTriggerTokens"`
	MaxRecentExchanges      int                  `json:"MaxRecentExchanges"`
	SummaryBudgetTokens     int                  `json:"SummaryBudgetTokens"`
	EnableModelCompaction   bool                 `json:"EnableModelCompaction"`
}

type conversationExchange struct {
	Human string `json:"human"`
	AI    string `json:"ai"`
}

type streamUIHints struct {
	WaitNotice      string
	WaitNoticeReply bool
	QueuedNotice    string
}

type streamProgressState struct {
	startedAt      time.Time
	lastOutputAt   time.Time
	firstOutputSet bool
	waitShown      bool
}

type pendingMessage struct {
	MessageID string `json:"message_id"`
	User      string `json:"user"`
	Text      string `json:"text"`
	At        string `json:"at"`
}

type conversationState struct {
	Profile    string                 `json:"profile"`
	Tokens     int                    `json:"tokens"`
	Owner      string                 `json:"owner"`
	Summary    string                 `json:"summary,omitempty"`
	Exchanges  []conversationExchange `json:"exchanges"`
	Pending    []pendingMessage       `json:"pending"`
	Processed  []string               `json:"processed"`
	InProgress bool                   `json:"in_progress"`
	UpdatedAt  string                 `json:"updated_at"`
}

type conversationContext struct {
	Direct          bool
	Threaded        bool
	User            string
	Channel         string
	ThreadID        string
	MessageID       string
	Prompt          string
	ConversationID  string
	ConversationKey string
	LegacyMemoryKey string
	DebugKey        string
	ExclusiveTag    string
}

type conversationIndexEntry struct {
	Key       string `json:"key"`
	UpdatedAt string `json:"updated_at"`
}

type conversationIndex struct {
	Version       int                               `json:"version"`
	Conversations map[string]conversationIndexEntry `json:"conversations"`
}

func Configure() *[]byte {
	return &defaultConfig
}

func PluginHandler(r robot.Robot, command string, args ...string) robot.TaskRetVal {
	switch command {
	case "init":
		return robot.Normal
	case "debug":
		return handleDebug(r)
	case "compact":
		return handleManualCompaction(r, false)
	case "compact-model":
		return handleManualCompaction(r, true)
	case "close":
		return handleClose(r)
	case "status":
		return handleStatus(r)
	case "image":
		return handleImage(r, args...)
	case "ambient", "catchall", "subscribed":
		return handleConversationEntry(r, command, args...)
	default:
		return robot.Normal
	}
}

func handleConversationEntry(r robot.Robot, command string, args ...string) robot.TaskRetVal {
	cmdMode := strings.TrimSpace(r.GetParameter("GOPHER_CMDMODE"))
	ctx := makeConversationContext(r, args...)
	cfg := loadConfig(r)
	direct := ctx.Direct
	botAlias := strings.TrimSpace(r.GetBotAttribute("alias").Attribute)
	channel := ctx.Channel

	// Preserve existing behavior where alias catchall routes to fallback/help style responses.
	if command == "catchall" && cmdMode == "alias" {
		if direct {
			r.Say("Command not found; try your command in a channel, or use '%shelp'", botAlias)
		} else {
			r.SayThread("No command matched in channel '%s'; try '%shelp'", channel, botAlias)
		}
		return robot.Normal
	}

	state, _ := loadConversationState(r, ctx)
	ensureConversationDefaults(&state, ctx)
	if wasProcessed(state.Processed, ctx.MessageID) {
		return robot.Normal
	}

	if !r.Exclusive(ctx.ExclusiveTag, true) {
		state.Pending = queuePendingMessage(state.Pending, pendingMessage{
			MessageID: ctx.MessageID,
			User:      ctx.User,
			Text:      ctx.Prompt,
			At:        nowString(),
		})
		state.UpdatedAt = nowString()
		saveConversationState(r, ctx, state)
		r.ReplyThread("(I hear you and queued this while I finish the current reply)")
		return robot.Normal
	}

	state.Pending = removePendingMessage(state.Pending, ctx.MessageID)
	state.Processed = appendProcessed(state.Processed, ctx.MessageID)
	state = maybeCompactConversationDeterministic(state, cfg)
	state.InProgress = true
	state.UpdatedAt = nowString()
	saveConversationState(r, ctx, state)

	tbot := r
	if !ctx.Direct {
		tbot = r.Threaded()
	}
	if !ctx.Direct && ctx.ThreadID != "" && len(state.Exchanges) == 0 {
		r.Subscribe()
	}

	uiHints := makeStreamUIHints(r, state, cfg)

	reply := ""
	if strings.TrimSpace(ctx.Prompt) != "" {
		var err error
		reply, err = queryOpenAI(tbot.MessageFormat(robot.BasicMarkdown), r, ctx, state, cfg, uiHints)
		if err != nil {
			tbot.Say("Sorry, there was an error contacting the AI: %s", err)
			state.InProgress = false
			state.UpdatedAt = nowString()
			saveConversationState(r, ctx, state)
			return robot.Normal
		}
		state.Exchanges = append(state.Exchanges, conversationExchange{
			Human: fmt.Sprintf("%s says: %s", ctx.User, ctx.Prompt),
			AI:    reply,
		})
		if len(state.Exchanges) > maxStoredExchanges {
			state.Exchanges = state.Exchanges[len(state.Exchanges)-maxStoredExchanges:]
		}
		state.Tokens = estimateConversationTokens(state.Exchanges) + estimateTokens(state.Summary)
		if len(state.Pending) > 0 {
			state.Pending = nil
		}
		state = maybeCompactConversationWithModel(r, state, cfg)
	}
	state.InProgress = false
	state.UpdatedAt = nowString()
	saveConversationState(r, ctx, state)
	return robot.Normal
}

func handleStatus(r robot.Robot) robot.TaskRetVal {
	ctx := makeConversationContext(r)
	if !ctx.Direct && !ctx.Threaded {
		r.Reply("I can hear you.")
		return robot.Normal
	}

	state, ok := loadConversationState(r, ctx)
	if !ok || (len(state.Exchanges) == 0 && strings.TrimSpace(state.Summary) == "") {
		r.Reply("I hear you, but I have no memory of a conversation in this context.")
		return robot.Normal
	}
	tokens := state.Tokens
	if tokens <= 0 {
		tokens = estimateConversationTokens(state.Exchanges) + estimateTokens(state.Summary)
	}
	summaryPart := ""
	if strings.TrimSpace(state.Summary) != "" {
		summaryPart = ", compact summary present"
	}
	if state.InProgress {
		r.Reply("I hear you and remember an AI conversation in progress (%d exchanges, ~%d tokens, %d queued%s).", len(state.Exchanges), tokens, len(state.Pending), summaryPart)
		return robot.Normal
	}
	r.Reply("I hear you and remember an AI conversation (%d exchanges, ~%d tokens, %d queued%s).", len(state.Exchanges), tokens, len(state.Pending), summaryPart)
	return robot.Normal
}

func handleClose(r robot.Robot) robot.TaskRetVal {
	ctx := makeConversationContext(r)
	state, ok := loadConversationState(r, ctx)
	if ok && (len(state.Exchanges) > 0 || strings.TrimSpace(state.Summary) != "") {
		deleteConversationState(r, ctx)
		if !ctx.Direct {
			r.Unsubscribe()
		}
		if ctx.Direct {
			r.Say("Ok, I'll forget this conversation.")
		} else {
			r.Say("Ok, I'll forget this conversation and unsubscribe this thread.")
		}
		return robot.Normal
	}
	if ctx.Direct || ctx.Threaded {
		r.Say("I have no memory of a conversation in progress.")
		return robot.Normal
	}
	r.Say("That command doesn't apply in this context.")
	return robot.Normal
}

func handleDebug(r robot.Robot) robot.TaskRetVal {
	ctx := makeConversationContext(r)
	if !ctx.Direct && !ctx.Threaded {
		r.SayThread("You can only initialize debugging in a conversation thread.")
		return robot.Normal
	}
	r.Remember(ctx.DebugKey, "true", true)
	r.SayThread("(ok, debugging output is enabled for this conversation)")
	return robot.Normal
}

func handleManualCompaction(r robot.Robot, withModel bool) robot.TaskRetVal {
	ctx := makeConversationContext(r)
	if !ctx.Direct && !ctx.Threaded {
		r.Reply("This command only applies in a direct message or conversation thread.")
		return robot.Normal
	}

	state, ok := loadConversationState(r, ctx)
	if !ok || (len(state.Exchanges) == 0 && strings.TrimSpace(state.Summary) == "") {
		r.Reply("I have no AI conversation memory to compact in this context.")
		return robot.Normal
	}
	cfg := loadConfig(r)

	beforeExchanges := len(state.Exchanges)
	beforeSummary := strings.TrimSpace(state.Summary)
	beforeTokens := state.Tokens
	if beforeTokens <= 0 {
		beforeTokens = estimateConversationTokens(state.Exchanges) + estimateTokens(state.Summary)
	}

	compacted, older := forceCompactConversationDeterministic(state, cfg)
	deterministicApplied := len(older) > 0
	modelApplied := false
	var modelErr error

	if withModel && deterministicApplied {
		refined, err := modelAssistSummary(r, compacted.Profile, compacted.Summary, older, cfg)
		if err != nil {
			modelErr = err
			r.Log(robot.Warn, "openai-fallback: manual model compaction failed conversation=%s older=%d: %v", ctx.ConversationID, len(older), err)
		} else if trimmed := strings.TrimSpace(refined); trimmed != "" {
			compacted.Summary = clipText(trimmed, summaryBudgetChars(cfg.SummaryBudgetTokens))
			compacted.Tokens = estimateConversationTokens(compacted.Exchanges) + estimateTokens(compacted.Summary)
			modelApplied = true
		}
	}

	changed := deterministicApplied || (compacted.Summary != beforeSummary)
	if changed {
		compacted.UpdatedAt = nowString()
		saveConversationState(r, ctx, compacted)
	}

	afterTokens := compacted.Tokens
	if afterTokens <= 0 {
		afterTokens = estimateConversationTokens(compacted.Exchanges) + estimateTokens(compacted.Summary)
	}
	r.Log(robot.Info, "openai-fallback: manual compaction mode=%s conversation=%s deterministic=%t model=%t older=%d exchanges=%d->%d tokens=%d->%d",
		manualCompactionMode(withModel), ctx.ConversationID, deterministicApplied, modelApplied, len(older), beforeExchanges, len(compacted.Exchanges), beforeTokens, afterTokens)

	if !deterministicApplied {
		r.Reply("No compaction was needed in this context (%d exchanges currently stored).", len(compacted.Exchanges))
		return robot.Normal
	}

	if withModel && modelErr != nil {
		r.Reply("Model-assisted compaction failed (%s), but deterministic compaction succeeded (%d -> %d exchanges).", modelErr.Error(), beforeExchanges, len(compacted.Exchanges))
		return robot.Normal
	}

	if withModel {
		r.Reply("Model-assisted compaction succeeded (%d -> %d exchanges, summary %s).", beforeExchanges, len(compacted.Exchanges), compactSummaryState(compacted.Summary))
		return robot.Normal
	}

	r.Reply("Deterministic compaction succeeded (%d -> %d exchanges, summary %s).", beforeExchanges, len(compacted.Exchanges), compactSummaryState(compacted.Summary))
	return robot.Normal
}

func manualCompactionMode(withModel bool) string {
	if withModel {
		return "api"
	}
	return "deterministic"
}

func compactSummaryState(summary string) string {
	if strings.TrimSpace(summary) == "" {
		return "absent"
	}
	return "present"
}

func handleImage(r robot.Robot, args ...string) robot.TaskRetVal {
	r.SayThread("Image generation is not wired yet for openai-fallback.")
	return robot.Normal
}

func isDirectMessage(r robot.Robot) bool {
	msg := r.GetMessage()
	if msg == nil {
		return strings.TrimSpace(r.GetParameter("GOPHER_CHANNEL")) == ""
	}
	return strings.TrimSpace(msg.Channel) == ""
}

func randomWaitMessage(r robot.Robot, cfg aiConfig) string {
	if len(cfg.WaitMessages) == 0 {
		return ""
	}
	return r.RandomString(cfg.WaitMessages)
}

func makeStreamUIHints(r robot.Robot, state conversationState, cfg aiConfig) streamUIHints {
	hints := streamUIHints{}
	hold := randomWaitMessage(r, cfg)
	if hold != "" && len(state.Exchanges) == 0 {
		hints.WaitNotice = fmt.Sprintf("( %s )", hold)
		hints.WaitNoticeReply = true
	} else if len(state.Exchanges) > 0 {
		hints.WaitNotice = fmt.Sprintf("(%s)", r.RandomString([]string{"pondering", "working", "thinking", "cogitating", "processing", "analyzing"}))
	} else {
		hints.WaitNotice = "(thinking...)"
		hints.WaitNoticeReply = true
	}
	if len(state.Pending) > 0 {
		hints.QueuedNotice = fmt.Sprintf("(I picked up %d queued messages for context)", len(state.Pending))
	}
	return hints
}

func loadConfig(r robot.Robot) aiConfig {
	cfg := aiConfig{}
	if ret := r.GetTaskConfig(&cfg); ret != robot.Ok {
		return cfg
	}
	return cfg
}

func debugJSON(v interface{}) string {
	buf, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(buf)
}

func makeConversationContext(r robot.Robot, args ...string) conversationContext {
	direct := isDirectMessage(r)
	channel := strings.TrimSpace(r.GetParameter("GOPHER_CHANNEL"))
	threadID := strings.TrimSpace(r.GetParameter("GOPHER_THREAD_ID"))
	threaded := threadID != ""
	messageID := strings.TrimSpace(r.GetParameter("GOPHER_MESSAGE_ID"))
	user := strings.TrimSpace(r.GetParameter("GOPHER_USER"))
	protocol := strings.TrimSpace(r.GetParameter("GOPHER_PROTOCOL"))
	if protocol == "" {
		protocol = "unknown"
	}
	if user == "" {
		user = "unknown"
	}
	protocol = strings.ToLower(protocol)
	user = strings.ToLower(user)
	if messageID != "" {
		messageID = protocol + ":" + messageID
	}
	if channel == "" {
		channel = "default"
	}
	prompt := ""
	if len(args) > 0 {
		prompt = strings.TrimSpace(args[0])
	}
	if threadID == "" {
		threadID = messageID
	}
	if threadID == "" {
		threadID = "root"
	}

	ctx := conversationContext{
		Direct:    direct,
		Threaded:  threaded,
		User:      user,
		Channel:   channel,
		ThreadID:  threadID,
		MessageID: messageID,
		Prompt:    prompt,
	}

	if direct {
		ctx.ConversationID = fmt.Sprintf("dm:%s", user)
		ctx.LegacyMemoryKey = fmt.Sprintf("%s:%s:dm:%s", shortTermMemoryPrefix, protocol, user)
		ctx.DebugKey = fmt.Sprintf("%s:%s:dm:%s", shortTermMemoryDebugPrefix, protocol, user)
		ctx.ExclusiveTag = fmt.Sprintf("%s:dm:%s", shortTermMemoryPrefix, user)
	} else {
		ctx.ConversationID = fmt.Sprintf("thread:%s:%s:%s", protocol, strings.ToLower(channel), threadID)
		ctx.LegacyMemoryKey = fmt.Sprintf("%s:%s:%s:%s", shortTermMemoryPrefix, protocol, strings.ToLower(channel), threadID)
		ctx.DebugKey = fmt.Sprintf("%s:%s:%s:%s", shortTermMemoryDebugPrefix, protocol, strings.ToLower(channel), threadID)
		ctx.ExclusiveTag = fmt.Sprintf("%s:%s:%s:%s", shortTermMemoryPrefix, protocol, strings.ToLower(channel), threadID)
	}
	ctx.ConversationKey = conversationDatumKey(ctx.ConversationID)

	if ctx.ExclusiveTag == "" {
		ctx.ExclusiveTag = shortTermMemoryPrefix + ":fallback"
	}
	return ctx
}

func loadConversationState(r robot.Robot, ctx conversationContext) (conversationState, bool) {
	state := conversationState{}
	_, exists, ret := r.CheckoutDatum(ctx.ConversationKey, &state, false)
	if ret == robot.Ok && exists {
		return state, true
	}

	encoded := strings.TrimSpace(r.Recall(ctx.LegacyMemoryKey, true))
	if encoded == "" {
		return conversationState{}, false
	}
	state, err := decodeConversationState(encoded)
	if err != nil {
		return conversationState{}, false
	}
	saveConversationState(r, ctx, state)
	r.Remember(ctx.LegacyMemoryKey, "", true)
	return state, true
}

func saveConversationState(r robot.Robot, ctx conversationContext, state conversationState) {
	state.UpdatedAt = nowString()
	if !storeConversationStateDatum(r, ctx.ConversationKey, state) {
		return
	}
	upsertConversationIndex(r, ctx.ConversationID, ctx.ConversationKey, state.UpdatedAt)
}

func decodeConversationState(encoded string) (conversationState, error) {
	state := conversationState{}
	// Support raw JSON state.
	if err := json.Unmarshal([]byte(encoded), &state); err == nil {
		return state, nil
	}
	// Support legacy base64(JSON) state.
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return conversationState{}, err
	}
	if err := json.Unmarshal(decoded, &state); err != nil {
		return conversationState{}, err
	}
	return state, nil
}

func storeConversationStateDatum(r robot.Robot, key string, state conversationState) bool {
	existing := conversationState{}
	locktoken, _, ret := r.CheckoutDatum(key, &existing, true)
	if ret != robot.Ok {
		return false
	}
	if ret = r.UpdateDatum(key, locktoken, state); ret != robot.Ok {
		r.CheckinDatum(key, locktoken)
		return false
	}
	return true
}

func ensureConversationIndexDefaults(idx *conversationIndex) {
	if idx.Version == 0 {
		idx.Version = conversationIndexVersion
	}
	if idx.Conversations == nil {
		idx.Conversations = make(map[string]conversationIndexEntry)
	}
}

func upsertConversationIndexEntry(idx *conversationIndex, conversationID, key, updatedAt string) {
	ensureConversationIndexDefaults(idx)
	idx.Conversations[conversationID] = conversationIndexEntry{
		Key:       key,
		UpdatedAt: updatedAt,
	}
}

func deleteConversationIndexEntry(idx *conversationIndex, conversationID string) {
	if idx == nil || idx.Conversations == nil {
		return
	}
	delete(idx.Conversations, conversationID)
}

func upsertConversationIndex(r robot.Robot, conversationID, key, updatedAt string) {
	idx := conversationIndex{}
	locktoken, _, ret := r.CheckoutDatum(conversationIndexDatumKey, &idx, true)
	if ret != robot.Ok {
		return
	}
	upsertConversationIndexEntry(&idx, conversationID, key, updatedAt)
	if ret = r.UpdateDatum(conversationIndexDatumKey, locktoken, idx); ret != robot.Ok {
		r.CheckinDatum(conversationIndexDatumKey, locktoken)
	}
}

func removeConversationIndex(r robot.Robot, conversationID string) {
	idx := conversationIndex{}
	locktoken, exists, ret := r.CheckoutDatum(conversationIndexDatumKey, &idx, true)
	if ret != robot.Ok {
		return
	}
	if !exists {
		r.CheckinDatum(conversationIndexDatumKey, locktoken)
		return
	}
	deleteConversationIndexEntry(&idx, conversationID)
	if ret = r.UpdateDatum(conversationIndexDatumKey, locktoken, idx); ret != robot.Ok {
		r.CheckinDatum(conversationIndexDatumKey, locktoken)
	}
}

func deleteConversationState(r robot.Robot, ctx conversationContext) {
	r.DeleteDatum(ctx.ConversationKey)
	removeConversationIndex(r, ctx.ConversationID)
	if ctx.LegacyMemoryKey != "" {
		r.Remember(ctx.LegacyMemoryKey, "", true)
	}
	r.Remember(ctx.DebugKey, "", true)
}

func conversationDatumKey(conversationID string) string {
	base := strings.ToLower(strings.TrimSpace(conversationID))
	if base == "" {
		base = "unknown"
	}
	return fmt.Sprintf("%s:%s", conversationDatumPrefix, sha1String(base))
}

func ensureConversationDefaults(state *conversationState, ctx conversationContext) {
	if state.Profile == "" {
		state.Profile = defaultProfile
	}
	if state.Owner == "" {
		state.Owner = ctx.User
	}
}

func queuePendingMessage(pending []pendingMessage, msg pendingMessage) []pendingMessage {
	if strings.TrimSpace(msg.Text) == "" {
		return pending
	}
	if msg.MessageID != "" {
		for _, existing := range pending {
			if existing.MessageID == msg.MessageID {
				return pending
			}
		}
	}
	pending = append(pending, msg)
	if len(pending) > maxPendingMessages {
		return pending[len(pending)-maxPendingMessages:]
	}
	return pending
}

func removePendingMessage(pending []pendingMessage, messageID string) []pendingMessage {
	if messageID == "" || len(pending) == 0 {
		return pending
	}
	out := make([]pendingMessage, 0, len(pending))
	for _, msg := range pending {
		if msg.MessageID == messageID {
			continue
		}
		out = append(out, msg)
	}
	return out
}

func appendProcessed(processed []string, messageID string) []string {
	if messageID == "" {
		return processed
	}
	processed = append(processed, messageID)
	if len(processed) > maxProcessedMessages {
		return processed[len(processed)-maxProcessedMessages:]
	}
	return processed
}

func wasProcessed(processed []string, messageID string) bool {
	if messageID == "" {
		return false
	}
	for _, item := range processed {
		if item == messageID {
			return true
		}
	}
	return false
}

func estimateTokens(text string) int {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return 0
	}
	// Rough approximation: ~4 characters per token for English text.
	return (len([]rune(trimmed)) / 4) + 1
}

func estimateConversationTokens(exchanges []conversationExchange) int {
	total := 0
	for _, exchange := range exchanges {
		total += estimateTokens(exchange.Human) + estimateTokens(exchange.AI) + 8
	}
	return total
}

func maybeCompactConversationDeterministic(state conversationState, cfg aiConfig) conversationState {
	compacted, _ := compactConversationDeterministic(state, cfg)
	return compacted
}

func forceCompactConversationDeterministic(state conversationState, cfg aiConfig) (conversationState, []conversationExchange) {
	return compactConversationDeterministicInternal(state, cfg, true)
}

func maybeCompactConversationWithModel(r robot.Robot, state conversationState, cfg aiConfig) conversationState {
	compacted, older := compactConversationDeterministic(state, cfg)
	if len(older) == 0 {
		return compacted
	}

	r.Log(robot.Info, "openai-fallback: automatic deterministic compaction applied older=%d exchanges_now=%d", len(older), len(compacted.Exchanges))
	if !cfg.EnableModelCompaction {
		return compacted
	}

	refined, err := modelAssistSummary(r, state.Profile, compacted.Summary, older, cfg)
	if err != nil {
		r.Log(robot.Warn, "openai-fallback: automatic model-assisted compaction failed; keeping deterministic summary: %v", err)
		return compacted
	}
	refined = strings.TrimSpace(refined)
	if refined == "" {
		r.Log(robot.Warn, "openai-fallback: automatic model-assisted compaction returned empty summary; keeping deterministic summary")
		return compacted
	}
	compacted.Summary = clipText(refined, summaryBudgetChars(cfg.SummaryBudgetTokens))
	compacted.Tokens = estimateConversationTokens(compacted.Exchanges) + estimateTokens(compacted.Summary)
	r.Log(robot.Info, "openai-fallback: automatic model-assisted compaction applied exchanges_now=%d", len(compacted.Exchanges))
	return compacted
}

type summaryRefiner func(existing string, older []conversationExchange, cfg aiConfig) (string, error)

func maybeCompactConversationWithRefiner(state conversationState, cfg aiConfig, refiner summaryRefiner) conversationState {
	compacted, older := compactConversationDeterministic(state, cfg)
	if len(older) == 0 || !cfg.EnableModelCompaction || refiner == nil {
		return compacted
	}
	refined, err := refiner(compacted.Summary, older, cfg)
	if err != nil {
		return compacted
	}
	refined = strings.TrimSpace(refined)
	if refined == "" {
		return compacted
	}
	compacted.Summary = clipText(refined, summaryBudgetChars(cfg.SummaryBudgetTokens))
	compacted.Tokens = estimateConversationTokens(compacted.Exchanges) + estimateTokens(compacted.Summary)
	return compacted
}

func compactConversationDeterministic(state conversationState, cfg aiConfig) (conversationState, []conversationExchange) {
	return compactConversationDeterministicInternal(state, cfg, false)
}

func compactConversationDeterministicInternal(state conversationState, cfg aiConfig, force bool) (conversationState, []conversationExchange) {
	maxRecent := cfg.MaxRecentExchanges
	if maxRecent <= 0 {
		maxRecent = defaultMaxRecentExchanges
	}
	if len(state.Exchanges) <= maxRecent {
		if state.Tokens <= 0 {
			state.Tokens = estimateConversationTokens(state.Exchanges) + estimateTokens(state.Summary)
		}
		return state, nil
	}

	trigger := cfg.CompactionTriggerTokens
	if trigger <= 0 {
		trigger = resolveCompactionTriggerTokens(resolveProfile(state.Profile, cfg))
	}
	if trigger <= 0 {
		trigger = 6144
	}

	if state.Tokens <= 0 {
		state.Tokens = estimateConversationTokens(state.Exchanges) + estimateTokens(state.Summary)
	}
	if !force && state.Tokens < trigger {
		return state, nil
	}

	split := len(state.Exchanges) - maxRecent
	older := state.Exchanges[:split]
	recent := state.Exchanges[split:]
	state.Summary = mergeDeterministicSummary(state.Summary, older, cfg.SummaryBudgetTokens)
	state.Exchanges = append([]conversationExchange(nil), recent...)
	state.Tokens = estimateConversationTokens(state.Exchanges) + estimateTokens(state.Summary)
	return state, append([]conversationExchange(nil), older...)
}

func resolveCompactionTriggerTokens(profile aiProfile) int {
	if profile.MaxContext <= 0 {
		return 6144
	}
	trigger := profile.MaxContext - 1024
	if trigger < 1024 {
		return profile.MaxContext
	}
	return trigger
}

func mergeDeterministicSummary(existing string, older []conversationExchange, summaryBudgetTokens int) string {
	if len(older) == 0 {
		return strings.TrimSpace(existing)
	}
	if summaryBudgetTokens <= 0 {
		summaryBudgetTokens = defaultSummaryBudgetTokens
	}
	maxChars := summaryBudgetChars(summaryBudgetTokens)

	lines := make([]string, 0, 8)
	if prior := strings.TrimSpace(existing); prior != "" {
		lines = append(lines, "Previous summary:")
		lines = append(lines, clipText(prior, maxChars/2))
	}
	lines = append(lines, fmt.Sprintf("Compacted %d earlier exchange(s):", len(older)))

	firstIntent := ""
	for _, item := range older {
		if text := strings.TrimSpace(item.Human); text != "" {
			firstIntent = text
			break
		}
	}
	if firstIntent != "" {
		lines = append(lines, "Initial intent: "+clipText(firstIntent, 180))
	}

	tail := older
	if len(tail) > 4 {
		tail = tail[len(tail)-4:]
	}
	for _, item := range tail {
		h := clipText(strings.TrimSpace(item.Human), 140)
		a := clipText(strings.TrimSpace(item.AI), 180)
		switch {
		case h != "" && a != "":
			lines = append(lines, fmt.Sprintf("- %s -> %s", h, a))
		case h != "":
			lines = append(lines, fmt.Sprintf("- %s", h))
		case a != "":
			lines = append(lines, fmt.Sprintf("- AI: %s", a))
		}
	}

	return clipText(strings.Join(lines, "\n"), maxChars)
}

func summaryBudgetChars(summaryBudgetTokens int) int {
	if summaryBudgetTokens <= 0 {
		summaryBudgetTokens = defaultSummaryBudgetTokens
	}
	maxChars := summaryBudgetTokens * 4
	if maxChars < 400 {
		maxChars = 400
	}
	return maxChars
}

func clipText(text string, maxChars int) string {
	text = strings.TrimSpace(text)
	if maxChars <= 0 || len([]rune(text)) <= maxChars {
		return text
	}
	runes := []rune(text)
	if maxChars <= 1 {
		return string(runes[:maxChars])
	}
	if maxChars <= 3 {
		return string(runes[:maxChars])
	}
	return strings.TrimSpace(string(runes[:maxChars-3])) + "..."
}

func pendingForContext(pending []pendingMessage, currentMessageID string, processed []string) []pendingMessage {
	if len(pending) == 0 {
		return nil
	}
	out := make([]pendingMessage, 0, len(pending))
	for _, item := range pending {
		if item.MessageID != "" && item.MessageID == currentMessageID {
			continue
		}
		if wasProcessed(processed, item.MessageID) {
			continue
		}
		if strings.TrimSpace(item.Text) == "" {
			continue
		}
		out = append(out, item)
	}
	return out
}

func trimExchangesForContext(system, summary string, exchanges []conversationExchange, pending []pendingMessage, prompt string, maxContext int) []conversationExchange {
	if len(exchanges) == 0 {
		return nil
	}
	if maxContext <= 0 {
		maxContext = 4096
	}
	budget := maxContext - estimateTokens(system) - estimateTokens(summary) - estimateTokens(prompt) - 64
	for _, item := range pending {
		budget -= estimateTokens(fmt.Sprintf("%s says: %s", item.User, item.Text)) + 4
	}
	if budget <= 0 {
		return nil
	}

	kept := make([]conversationExchange, 0, len(exchanges))
	used := 0
	for i := len(exchanges) - 1; i >= 0; i-- {
		cost := estimateTokens(exchanges[i].Human) + estimateTokens(exchanges[i].AI) + 8
		if used+cost > budget {
			break
		}
		kept = append(kept, exchanges[i])
		used += cost
	}
	for i, j := 0, len(kept)-1; i < j; i, j = i+1, j-1 {
		kept[i], kept[j] = kept[j], kept[i]
	}
	return kept
}

func nowString() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func modelAssistSummary(r robot.Robot, profileName, deterministic string, older []conversationExchange, cfg aiConfig) (string, error) {
	token := strings.TrimSpace(r.GetParameter("OPENAI_KEY"))
	if token == "" {
		return "", fmt.Errorf("OPENAI_KEY not set")
	}

	profile := resolveProfile(profileName, cfg)
	model := "gpt-5.2-chat-latest"
	if raw, ok := profile.Params["model"]; ok {
		if asString := strings.TrimSpace(fmt.Sprintf("%v", raw)); asString != "" {
			model = asString
		}
	}

	payload := map[string]interface{}{
		"model":  model,
		"stream": false,
		"messages": []map[string]string{
			{
				"role": "system",
				"content": "You condense older chat history into a concise, factual summary for future context. " +
					"Keep key user goals, decisions, constraints, and unresolved questions. " +
					"Avoid headings and tables. Keep output compact and actionable.",
			},
			{
				"role":    "user",
				"content": modelCompactionSource(deterministic, older, cfg.SummaryBudgetTokens),
			},
		},
	}
	if cfg.SummaryBudgetTokens > 0 {
		payload["max_tokens"] = cfg.SummaryBudgetTokens
	}
	if userID := strings.TrimSpace(r.GetParameter("GOPHER_USER_ID")); userID != "" {
		payload["user"] = sha1String(userID)
	}

	reqBody, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, openAIChatCompletionsURL, bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	if org := strings.TrimSpace(r.GetParameter("OPENAI_ORGANIZATION_ID")); org != "" {
		req.Header.Set("OpenAI-Organization", org)
	}

	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("%s", friendlyOpenAIError(resp.StatusCode, resp.Status, body))
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("no choices returned for model-assisted compaction")
	}
	content := strings.TrimSpace(parsed.Choices[0].Message.Content)
	if content == "" {
		return "", fmt.Errorf("empty model-assisted compaction summary")
	}
	return clipText(normalizeChunkText(content), summaryBudgetChars(cfg.SummaryBudgetTokens)), nil
}

func modelCompactionSource(deterministic string, older []conversationExchange, summaryBudgetTokens int) string {
	maxChars := summaryBudgetChars(summaryBudgetTokens)
	sample := older
	if len(sample) > 12 {
		sample = sample[len(sample)-12:]
	}
	var b strings.Builder
	if prior := strings.TrimSpace(deterministic); prior != "" {
		b.WriteString("Current deterministic summary:\n")
		b.WriteString(clipText(prior, maxChars/2))
		b.WriteString("\n\n")
	}
	b.WriteString(fmt.Sprintf("Older exchanges to compact (%d shown):\n", len(sample)))
	for i, item := range sample {
		h := clipText(strings.TrimSpace(item.Human), 220)
		a := clipText(strings.TrimSpace(item.AI), 220)
		b.WriteString(fmt.Sprintf("%d. Human: %s\n", i+1, h))
		b.WriteString(fmt.Sprintf("   AI: %s\n", a))
	}
	return clipText(strings.TrimSpace(b.String()), maxChars*2)
}

func queryOpenAI(outBot robot.Robot, r robot.Robot, ctx conversationContext, state conversationState, cfg aiConfig, uiHints streamUIHints) (string, error) {
	token := strings.TrimSpace(r.GetParameter("OPENAI_KEY"))
	if token == "" {
		return "", fmt.Errorf("no OPENAI_KEY set")
	}

	profile := resolveProfile(state.Profile, cfg)
	systemPrompt := buildSystemPrompt(profile)
	queued := pendingForContext(state.Pending, ctx.MessageID, state.Processed)
	trimmedExchanges := trimExchangesForContext(systemPrompt, state.Summary, state.Exchanges, queued, ctx.Prompt, profile.MaxContext)
	messages := buildMessages(systemPrompt, state.Summary, trimmedExchanges, queued, ctx)
	payload := map[string]interface{}{
		"messages": messages,
		"stream":   true,
	}
	for k, v := range profile.Params {
		payload[k] = v
	}
	if _, ok := payload["model"]; !ok {
		payload["model"] = "gpt-5.2-chat-latest"
	}
	if userID := strings.TrimSpace(r.GetParameter("GOPHER_USER_ID")); userID != "" {
		payload["user"] = sha1String(userID)
	}
	if strings.TrimSpace(r.Recall(ctx.DebugKey, true)) != "" {
		outBot.Say("AI debug: profile=%s model=%v messages=%d queued=%d", state.Profile, payload["model"], len(messages), len(queued))
	}

	reqBody, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, openAIChatCompletionsURL, bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	if org := strings.TrimSpace(r.GetParameter("OPENAI_ORGANIZATION_ID")); org != "" {
		req.Header.Set("OpenAI-Organization", org)
	}

	client := &http.Client{Timeout: 3 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("%s", friendlyOpenAIError(resp.StatusCode, resp.Status, body))
	}

	reply, err := consumeSSEAndEmit(outBot, resp.Body, uiHints)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(reply) == "" {
		return "", fmt.Errorf("AI returned no textual content")
	}
	return reply, nil
}

func resolveProfile(profileName string, cfg aiConfig) aiProfile {
	if cfg.Profiles == nil {
		return aiProfile{
			Params: map[string]interface{}{
				"model":       "gpt-5.2-chat-latest",
				"temperature": 0.7,
			},
			SystemPrompt: systemPromptConfig{
				Standard: defaultStandardSystemPrompt,
				Custom:   defaultCustomSystemPrompt,
			},
		}
	}
	if profileName != "" {
		if profile, ok := cfg.Profiles[profileName]; ok {
			return profile
		}
	}
	if profile, ok := cfg.Profiles[defaultProfile]; ok {
		return profile
	}
	for _, profile := range cfg.Profiles {
		return profile
	}
	return aiProfile{
		Params: map[string]interface{}{
			"model":       "gpt-5.2-chat-latest",
			"temperature": 0.7,
		},
		SystemPrompt: systemPromptConfig{
			Standard: defaultStandardSystemPrompt,
			Custom:   defaultCustomSystemPrompt,
		},
	}
}

func buildSystemPrompt(profile aiProfile) string {
	standard := strings.TrimSpace(profile.SystemPrompt.Standard)
	custom := strings.TrimSpace(profile.SystemPrompt.Custom)
	if standard == "" {
		standard = defaultStandardSystemPrompt
	}
	if custom == "" {
		custom = defaultCustomSystemPrompt
	}
	return strings.TrimSpace(basicMarkdownSystemPrefix + "\n\n" + standard + "\n\n" + custom)
}

func buildMessages(system, summary string, exchanges []conversationExchange, pending []pendingMessage, ctx conversationContext) []map[string]string {
	if strings.TrimSpace(system) == "" {
		system = strings.TrimSpace(defaultStandardSystemPrompt + "\n\n" + defaultCustomSystemPrompt)
	}
	messages := []map[string]string{
		{
			"role":    "system",
			"content": system,
		},
	}
	if summary = strings.TrimSpace(summary); summary != "" {
		messages = append(messages, map[string]string{
			"role":    "system",
			"content": "Conversation summary (older context):\n" + summary,
		})
	}
	for _, exchange := range exchanges {
		if strings.TrimSpace(exchange.Human) != "" {
			messages = append(messages, map[string]string{
				"role":    "user",
				"content": exchange.Human,
			})
		}
		if strings.TrimSpace(exchange.AI) != "" {
			messages = append(messages, map[string]string{
				"role":    "assistant",
				"content": exchange.AI,
			})
		}
	}
	for _, item := range pending {
		content := strings.TrimSpace(item.Text)
		if content == "" {
			continue
		}
		author := strings.TrimSpace(item.User)
		if author == "" {
			author = "unknown user"
		}
		messages = append(messages, map[string]string{
			"role":    "user",
			"content": fmt.Sprintf("%s says (queued): %s", author, content),
		})
	}
	if strings.TrimSpace(ctx.Prompt) != "" {
		messages = append(messages, map[string]string{
			"role":    "user",
			"content": fmt.Sprintf("%s says: %s", ctx.User, ctx.Prompt),
		})
	}
	return messages
}

func consumeSSEAndEmit(outBot robot.Robot, body io.Reader, uiHints streamUIHints) (string, error) {
	reader := bufio.NewReader(body)
	var pending strings.Builder
	var full strings.Builder
	progress := &streamProgressState{startedAt: time.Now()}

	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			break
		}
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "data:") {
			payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if payload == "[DONE]" {
				break
			}
			chunk, payloadError := extractDeltaContent(payload)
			if payloadError != "" {
				return strings.TrimSpace(full.String()), fmt.Errorf("%s", payloadError)
			}
			if chunk != "" {
				full.WriteString(chunk)
				pending.WriteString(chunk)
				emitAvailableChunks(outBot, &pending, progress, uiHints)
			}
		}
		if err == io.EOF {
			break
		}
	}

	rest := strings.TrimSpace(normalizeChunkText(pending.String()))
	if rest != "" {
		emitStreamChunk(outBot, progress, uiHints, rest)
	}
	return strings.TrimSpace(normalizeChunkText(full.String())), nil
}

func extractDeltaContent(payload string) (string, string) {
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(payload), &parsed); err != nil {
		return "", ""
	}
	if rawErr, ok := parsed["error"].(map[string]interface{}); ok {
		message, _ := rawErr["message"].(string)
		if message == "" {
			code, _ := rawErr["code"].(string)
			message = code
		}
		if message == "" {
			message = "OpenAI returned an unknown stream error"
		}
		return "", message
	}
	rawChoices, ok := parsed["choices"].([]interface{})
	if !ok || len(rawChoices) == 0 {
		return "", ""
	}
	choice, ok := rawChoices[0].(map[string]interface{})
	if !ok {
		return "", ""
	}
	rawDelta, ok := choice["delta"].(map[string]interface{})
	if !ok {
		return "", ""
	}
	content, _ := rawDelta["content"].(string)
	return content, ""
}

func emitAvailableChunks(outBot robot.Robot, pending *strings.Builder, progress *streamProgressState, uiHints streamUIHints) {
	text := pending.String()
	for {
		cut := chunkBoundary(text)
		if cut < 0 {
			break
		}
		if hasUnbalancedFences(text[:cut]) {
			break
		}
		chunk := strings.TrimSpace(normalizeChunkText(text[:cut]))
		if chunk != "" {
			emitStreamChunk(outBot, progress, uiHints, chunk)
		}
		text = text[cut:]
	}
	pending.Reset()
	pending.WriteString(text)
}

func emitStreamChunk(outBot robot.Robot, progress *streamProgressState, uiHints streamUIHints, chunk string) {
	chunk = strings.TrimSpace(chunk)
	if chunk == "" {
		return
	}

	now := time.Now()
	if !progress.firstOutputSet {
		if !progress.waitShown && now.Sub(progress.startedAt) >= delayedWaitNoticeDelay {
			if uiHints.WaitNotice != "" {
				if uiHints.WaitNoticeReply {
					outBot.Reply(uiHints.WaitNotice)
				} else {
					outBot.Say(uiHints.WaitNotice)
				}
			}
			if uiHints.QueuedNotice != "" {
				outBot.Say(uiHints.QueuedNotice)
			}
			progress.waitShown = true
		}
	} else if now.Sub(progress.lastOutputAt) >= streamProgressNoticeDelay {
		outBot.Say("(...)")
	}

	outBot.Say(chunk)
	progress.lastOutputAt = now
	progress.firstOutputSet = true
}

func chunkBoundary(text string) int {
	if text == "" {
		return -1
	}
	if idx := strings.Index(text, "\n\n"); idx >= 0 {
		return idx + 2
	}
	if len(text) < defaultChunkSoftLimit {
		return -1
	}
	cutRegion := text
	if len(cutRegion) > defaultChunkHardLimit {
		cutRegion = cutRegion[:defaultChunkHardLimit]
	}
	for i := len(cutRegion) - 1; i >= 0; i-- {
		switch cutRegion[i] {
		case '\n':
			if i >= defaultChunkSoftLimit/2 {
				return i + 1
			}
		case '.', '!', '?':
			if i+1 < len(cutRegion) && cutRegion[i+1] == ' ' {
				if i >= defaultChunkSoftLimit/2 {
					return i + 1
				}
			}
		}
	}
	for i := len(cutRegion) - 1; i >= defaultChunkSoftLimit/2; i-- {
		if cutRegion[i] == ' ' {
			return i + 1
		}
	}
	return -1
}

func normalizeChunkText(text string) string {
	if text == "" {
		return text
	}
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	text = basicMarkdownBreakTagRE.ReplaceAllString(text, "\n")
	if strings.EqualFold(strings.TrimSpace(text), "```") {
		return ""
	}
	if strings.TrimSpace(text) == "" {
		return ""
	}
	return normalizeAIResponseToBasicMarkdown(text)
}

func hasUnbalancedFences(text string) bool {
	if text == "" {
		return false
	}
	return strings.Count(text, "```")%2 != 0
}

var (
	basicMarkdownHeadingRE      = regexp.MustCompile(`^\s{0,3}#{1,6}\s+(.+?)\s*$`)
	basicMarkdownOrderedListRE  = regexp.MustCompile(`^\s*\d+[.)]\s+(.+)$`)
	basicMarkdownBulletListRE   = regexp.MustCompile(`^\s*[\*\+]\s+(.+)$`)
	basicMarkdownTaskListRE     = regexp.MustCompile(`^\s*[-*+]\s+\[(?: |x|X)\]\s+(.+)$`)
	basicMarkdownLabeledLinkRE  = regexp.MustCompile(`<((?:https?|mailto):[^>|]+)\|([^>]+)>`)
	basicMarkdownBareLinkRE     = regexp.MustCompile(`<((?:https?|mailto):[^>]+)>`)
	basicMarkdownStrongHTMLRE   = regexp.MustCompile(`(?i)<(?:strong|b)>(.*?)</(?:strong|b)>`)
	basicMarkdownEmphasisHTMLRE = regexp.MustCompile(`(?i)<(?:em|i)>(.*?)</(?:em|i)>`)
	basicMarkdownCodeHTMLRE     = regexp.MustCompile(`(?i)<code>(.*?)</code>`)
	basicMarkdownTagRE          = regexp.MustCompile(`(?i)</?[a-z][^>]*>`)
	basicMarkdownBreakTagRE     = regexp.MustCompile(`(?i)<br\s*/?>`)
	basicMarkdownBoldUndersRE   = regexp.MustCompile(`__([^_\n]+?)__`)
	basicMarkdownItalicUndersRE = regexp.MustCompile(`(^|[\s(\[{])_([^_\n]+?)_($|[\s)\]},.!?:;])`)
	basicMarkdownStrikeRE       = regexp.MustCompile(`~~([^~\n]+?)~~`)
)

func normalizeAIResponseToBasicMarkdown(text string) string {
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))
	inFence := false

	for _, line := range lines {
		line = strings.TrimRight(line, " \t")
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			// Preserve optional language hint; BasicMarkdown v1 allows it.
			out = append(out, trimmed)
			inFence = !inFence
			continue
		}
		if inFence {
			out = append(out, line)
			continue
		}
		out = append(out, normalizeBasicMarkdownLine(line))
	}
	return strings.Join(out, "\n")
}

func normalizeBasicMarkdownLine(line string) string {
	if line == "" {
		return line
	}
	if m := basicMarkdownHeadingRE.FindStringSubmatch(line); len(m) == 2 {
		line = "**" + strings.TrimSpace(m[1]) + "**"
	}

	trimmedLeft := strings.TrimLeft(line, " \t")
	switch {
	case strings.HasPrefix(trimmedLeft, "\u2022 "):
		line = "- " + strings.TrimSpace(strings.TrimPrefix(trimmedLeft, "\u2022 "))
	default:
		if m := basicMarkdownTaskListRE.FindStringSubmatch(line); len(m) == 2 {
			line = "- " + strings.TrimSpace(m[1])
		} else if m := basicMarkdownOrderedListRE.FindStringSubmatch(line); len(m) == 2 {
			line = "- " + strings.TrimSpace(m[1])
		} else if m := basicMarkdownBulletListRE.FindStringSubmatch(line); len(m) == 2 {
			line = "- " + strings.TrimSpace(m[1])
		}
	}

	line = normalizeBasicMarkdownInline(line)
	return line
}

func normalizeBasicMarkdownInline(text string) string {
	if text == "" {
		return text
	}
	text = basicMarkdownLabeledLinkRE.ReplaceAllString(text, "[$2]($1)")
	text = basicMarkdownBareLinkRE.ReplaceAllString(text, "$1")
	text = basicMarkdownStrongHTMLRE.ReplaceAllString(text, "**$1**")
	text = basicMarkdownEmphasisHTMLRE.ReplaceAllString(text, "*$1*")
	text = basicMarkdownCodeHTMLRE.ReplaceAllString(text, "`$1`")
	text = basicMarkdownBoldUndersRE.ReplaceAllString(text, "**$1**")
	for {
		next := basicMarkdownItalicUndersRE.ReplaceAllString(text, "$1*$2*$3")
		if next == text {
			break
		}
		text = next
	}
	text = basicMarkdownStrikeRE.ReplaceAllString(text, "$1")
	text = basicMarkdownTagRE.ReplaceAllString(text, "")
	return text
}

func friendlyOpenAIError(statusCode int, status string, body []byte) string {
	type apiErrorEnvelope struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code"`
		} `json:"error"`
	}
	parsed := apiErrorEnvelope{}
	message := strings.TrimSpace(string(body))
	if err := json.Unmarshal(body, &parsed); err == nil {
		if strings.TrimSpace(parsed.Error.Message) != "" {
			message = strings.TrimSpace(parsed.Error.Message)
		}
		code := strings.ToLower(strings.TrimSpace(parsed.Error.Code))
		typ := strings.ToLower(strings.TrimSpace(parsed.Error.Type))
		if code == "insufficient_quota" || typ == "insufficient_quota" {
			return "OpenAI quota is exhausted for the configured token. Please update billing or rotate OPENAI_KEY."
		}
		if code == "invalid_api_key" {
			return "OpenAI rejected the configured API key. Please verify OPENAI_KEY."
		}
	}
	if statusCode == http.StatusUnauthorized {
		return "OpenAI authentication failed (401). Please verify OPENAI_KEY and OPENAI_ORGANIZATION_ID."
	}
	if statusCode == http.StatusTooManyRequests {
		return "OpenAI rate-limited this request (429). Please retry shortly."
	}
	if statusCode >= 500 {
		return fmt.Sprintf("OpenAI is currently unavailable (%s). Please retry shortly.", status)
	}
	if message == "" {
		return fmt.Sprintf("OpenAI request failed: %s", status)
	}
	return fmt.Sprintf("OpenAI request failed: %s", message)
}

func sha1String(s string) string {
	sum := sha1.Sum([]byte(s))
	return fmt.Sprintf("%x", sum)
}
