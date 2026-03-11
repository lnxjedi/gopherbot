package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
)

const openAIChatCompletionsURL = "https://api.openai.com/v1/chat/completions"

var defaultConfig = []byte(`
---
AllowDirect: true
AllChannels: true
CatchAll: true
CatchAllModes:
- alias
AllowedHiddenCommands:
- catchall
Config:
  HeardNotice: "_(looking for the closest command...)_"
  MaxVisibleMatches: 4
  MaxBrowseableMatches: 6
  OpenAIModel: "gpt-5.2-chat-latest"
  Temperature: 1
  MaxCompletionTokens: 240
  SystemPrompt: |
    You are a command-recovery assistant for Gopherbot.
    Help the user recover from an unmatched alias command by suggesting the most likely next step.
    Prefer concrete, deterministic guidance over speculation.
    Never invent commands or channels that are not present in the provided metadata.
    If the user is probably in the wrong channel, say so plainly and point them to the likely channel or DM context.
    Respond using BasicMarkdown v1-compatible output only.
`)

type fallbackConfig struct {
	HeardNotice          string  `json:"HeardNotice"`
	MaxVisibleMatches    int     `json:"MaxVisibleMatches"`
	MaxBrowseableMatches int     `json:"MaxBrowseableMatches"`
	OpenAIModel          string  `json:"OpenAIModel"`
	Temperature          float64 `json:"Temperature"`
	MaxCompletionTokens  int     `json:"MaxCompletionTokens"`
	SystemPrompt         string  `json:"SystemPrompt"`
}

type fallbackAdviceResponse struct {
	Context            fallbackAdviceContext `json:"context"`
	Advice             string                `json:"advice"`
	WrongChannelHint   string                `json:"wrong_channel_hint"`
	DeterministicReply string                `json:"deterministic_reply"`
	Here               []fallbackAdviceEntry `json:"here"`
	Elsewhere          []fallbackAdviceEntry `json:"elsewhere"`
}

type fallbackAdviceContext struct {
	BotName         string `json:"bot_name"`
	BotAlias        string `json:"bot_alias"`
	User            string `json:"user"`
	Channel         string `json:"channel"`
	CommandMode     string `json:"command_mode"`
	Direct          bool   `json:"direct"`
	Threaded        bool   `json:"threaded"`
	Protocol        string `json:"protocol"`
	RawQuery        string `json:"raw_query"`
	NormalizedQuery string `json:"normalized_query"`
}

type fallbackAdviceEntry struct {
	PluginName  string   `json:"plugin"`
	Command     string   `json:"command"`
	Usage       string   `json:"usage"`
	Summary     string   `json:"summary"`
	Keywords    []string `json:"keywords"`
	VisibleHere bool     `json:"visible_here"`
	Channels    []string `json:"channels"`
	AllowDirect bool     `json:"allow_direct"`
	DirectOnly  bool     `json:"direct_only"`
	Score       int      `json:"score"`
}

type compactEntry struct {
	PluginName  string   `json:"plugin"`
	Command     string   `json:"command"`
	Usage       string   `json:"usage,omitempty"`
	Summary     string   `json:"summary,omitempty"`
	Channels    []string `json:"channels,omitempty"`
	DirectOnly  bool     `json:"direct_only,omitempty"`
	AllowDirect bool     `json:"allow_direct,omitempty"`
	VisibleHere bool     `json:"visible_here,omitempty"`
}

type compactPayload struct {
	Advice           string         `json:"advice,omitempty"`
	Deterministic    string         `json:"deterministic,omitempty"`
	Attempted        string         `json:"attempted"`
	Normalized       string         `json:"normalized"`
	Channel          string         `json:"channel,omitempty"`
	Direct           bool           `json:"direct"`
	WrongChannelHint string         `json:"wrong_channel_hint,omitempty"`
	Here             []compactEntry `json:"here,omitempty"`
	Elsewhere        []compactEntry `json:"elsewhere,omitempty"`
}

const (
	adviceWrongChannel = "wrong_channel"
	adviceCloseHere    = "close_match_here"
)

func Configure() *[]byte {
	return &defaultConfig
}

func PluginHandler(r robot.Robot, command string, args ...string) robot.TaskRetVal {
	switch command {
	case "init":
		return robot.Normal
	case "catchall":
		return handleCatchAll(r, args...)
	default:
		return robot.Normal
	}
}

func handleCatchAll(r robot.Robot, args ...string) robot.TaskRetVal {
	raw := ""
	if len(args) > 0 {
		raw = strings.TrimSpace(args[0])
	}
	if raw == "" {
		return robot.Normal
	}

	cfg := loadConfig(r)
	adviceStart := time.Now()
	advice, err := loadFallbackAdvice(r, raw)
	if err != nil {
		replyFallback(r, deterministicNoMatch(raw, "!", ""))
		return robot.Normal
	}
	r.Log(robot.Debug, "ai-fallback-help: fallback advice generated in %s", time.Since(adviceStart))

	deterministic := strings.TrimSpace(advice.DeterministicReply)
	if deterministic == "" {
		deterministic = deterministicNoMatch(raw, advice.Context.BotAlias, advice.Context.Channel)
	}
	if !aiRecoveryEnabled(r, cfg) {
		replyFallback(r, deterministic)
		addDeterministicHelp(r, advice)
		return robot.Normal
	}
	if shouldUseDeterministicAdvice(advice) {
		replyFallback(r, deterministic)
		addDeterministicHelp(r, advice)
		return robot.Normal
	}

	emitHeardNoticeIfNeeded(r, cfg)

	aiStart := time.Now()
	refined, err := askOpenAIForRecovery(r, cfg, advice)
	r.Log(robot.Debug, "ai-fallback-help: OpenAI recovery round-trip took %s", time.Since(aiStart))
	if err != nil || strings.TrimSpace(refined) == "" {
		if err != nil {
			r.Log(robot.Warn, "ai-fallback-help: OpenAI recovery failed: %v", err)
		}
		replyFallback(r, deterministic)
		return robot.Normal
	}
	replyFallback(r, refined)
	return robot.Normal
}

func loadConfig(r robot.Robot) fallbackConfig {
	cfg := fallbackConfig{
		HeardNotice:          "_(looking for the closest command...)_",
		MaxVisibleMatches:    4,
		MaxBrowseableMatches: 6,
		OpenAIModel:          "gpt-5.2-chat-latest",
		Temperature:          1,
		MaxCompletionTokens:  240,
	}
	_ = r.GetTaskConfig(&cfg)
	return cfg
}

func loadFallbackAdvice(r robot.Robot, raw string) (fallbackAdviceResponse, error) {
	var advice fallbackAdviceResponse
	blob := strings.TrimSpace(r.GetFallbackAdvice(raw))
	if blob == "" {
		return advice, fmt.Errorf("empty fallback advice")
	}
	if err := json.Unmarshal([]byte(blob), &advice); err != nil {
		return advice, err
	}
	return advice, nil
}

func aiRecoveryEnabled(r robot.Robot, cfg fallbackConfig) bool {
	if strings.TrimSpace(cfg.OpenAIModel) == "" {
		return false
	}
	return strings.TrimSpace(r.GetParameter("OPENAI_KEY")) != ""
}

func shouldUseDeterministicAdvice(advice fallbackAdviceResponse) bool {
	switch advice.Advice {
	case adviceWrongChannel:
		return true
	case adviceCloseHere:
		if len(advice.Here) <= 1 {
			return true
		}
		return advice.Here[0].Score >= 92
	default:
		return len(advice.Here) == 0
	}
}

func addDeterministicHelp(r robot.Robot, advice fallbackAdviceResponse) {
	for _, term := range suggestedHelpTerms(advice) {
		if ret := r.AddCommand("builtin-help", "help "+term+" brief"); ret != robot.Ok {
			r.Log(robot.Debug, "ai-fallback-help: AddCommand for help %q failed: %s", term, ret)
		}
	}
}

func suggestedHelpTerms(advice fallbackAdviceResponse) []string {
	terms := make([]string, 0, 3)
	seen := make(map[string]struct{}, 8)
	addTerm := func(term string) {
		term = strings.TrimSpace(strings.ToLower(term))
		if term == "" {
			return
		}
		if _, ok := seen[term]; ok {
			return
		}
		seen[term] = struct{}{}
		terms = append(terms, term)
	}
	addEntry := func(entry fallbackAdviceEntry) {
		addTerm(entry.Command)
		for _, keyword := range entry.Keywords {
			addTerm(keyword)
			if len(terms) >= 3 {
				return
			}
		}
	}
	switch advice.Advice {
	case adviceWrongChannel:
		if len(advice.Elsewhere) > 0 {
			addEntry(advice.Elsewhere[0])
		}
		if len(terms) > 1 {
			return terms[:1]
		}
	case adviceCloseHere:
		for _, entry := range advice.Here {
			addEntry(entry)
			if len(terms) >= 3 {
				break
			}
		}
	default:
		for _, entry := range advice.Elsewhere {
			addEntry(entry)
			if len(terms) >= 3 {
				break
			}
		}
	}
	return terms
}

func toCompactEntry(entry fallbackAdviceEntry) compactEntry {
	return compactEntry{
		PluginName:  entry.PluginName,
		Command:     entry.Command,
		Usage:       entry.Usage,
		Summary:     entry.Summary,
		Channels:    append([]string(nil), entry.Channels...),
		DirectOnly:  entry.DirectOnly,
		AllowDirect: entry.AllowDirect,
		VisibleHere: entry.VisibleHere,
	}
}

func deterministicNoMatch(attempted, alias, channel string) string {
	if strings.TrimSpace(alias) == "" {
		alias = "!"
	}
	if strings.TrimSpace(attempted) == "" {
		return fmt.Sprintf("I couldn't match that command. Try `%scommands` or `%shelp <keyword>`.", alias, alias)
	}
	if strings.TrimSpace(channel) != "" {
		return fmt.Sprintf("I couldn't match `%s` in #%s. Try `%scommands` or `%shelp <keyword>`.", attempted, channel, alias, alias)
	}
	return fmt.Sprintf("I couldn't match `%s`. Try `%scommands` or `%shelp <keyword>`.", attempted, alias, alias)
}

func emitHeardNoticeIfNeeded(r robot.Robot, cfg fallbackConfig) {
	heard := strings.TrimSpace(cfg.HeardNotice)
	if heard == "" {
		return
	}
	out := r.MessageFormat(robot.BasicMarkdown)
	if msg := r.GetMessage(); msg != nil && strings.TrimSpace(msg.Channel) == "" {
		out.Say(heard)
		return
	}
	out.ReplyThread(heard)
}

func askOpenAIForRecovery(r robot.Robot, cfg fallbackConfig, advice fallbackAdviceResponse) (string, error) {
	here := advice.Here
	if cfg.MaxVisibleMatches > 0 && len(here) > cfg.MaxVisibleMatches {
		here = here[:cfg.MaxVisibleMatches]
	}
	elsewhere := advice.Elsewhere
	if cfg.MaxBrowseableMatches > 0 && len(elsewhere) > cfg.MaxBrowseableMatches {
		elsewhere = elsewhere[:cfg.MaxBrowseableMatches]
	}
	compactHere := make([]compactEntry, 0, len(here))
	for _, entry := range here {
		compactHere = append(compactHere, toCompactEntry(entry))
	}
	compactElsewhere := make([]compactEntry, 0, len(elsewhere))
	for _, entry := range elsewhere {
		compactElsewhere = append(compactElsewhere, toCompactEntry(entry))
	}

	payload := compactPayload{
		Advice:           advice.Advice,
		Deterministic:    advice.DeterministicReply,
		Attempted:        advice.Context.RawQuery,
		Normalized:       advice.Context.NormalizedQuery,
		Channel:          advice.Context.Channel,
		Direct:           advice.Context.Direct,
		WrongChannelHint: advice.WrongChannelHint,
		Here:             compactHere,
		Elsewhere:        compactElsewhere,
	}
	userPrompt, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	requestPayload := map[string]interface{}{
		"model":                 strings.TrimSpace(cfg.OpenAIModel),
		"temperature":           cfg.Temperature,
		"max_completion_tokens": cfg.MaxCompletionTokens,
		"stream":                false,
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": strings.TrimSpace(cfg.SystemPrompt),
			},
			{
				"role":    "user",
				"content": string(userPrompt),
			},
		},
	}
	if requestPayload["model"] == "" {
		requestPayload["model"] = "gpt-5.2-chat-latest"
	}
	if userID := strings.TrimSpace(r.GetParameter("GOPHER_USER_ID")); userID != "" {
		requestPayload["user"] = sha1String(userID)
	}

	reqBody, err := json.Marshal(requestPayload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, openAIChatCompletionsURL, bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(r.GetParameter("OPENAI_KEY")))
	req.Header.Set("Content-Type", "application/json")
	if org := strings.TrimSpace(r.GetParameter("OPENAI_ORGANIZATION_ID")); org != "" {
		req.Header.Set("OpenAI-Organization", org)
	}

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("OpenAI request failed: %s", extractOpenAIError(body))
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
		return "", fmt.Errorf("no choices returned")
	}
	return strings.TrimSpace(parsed.Choices[0].Message.Content), nil
}

func extractOpenAIError(body []byte) string {
	var parsed struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &parsed); err == nil && strings.TrimSpace(parsed.Error.Message) != "" {
		return strings.TrimSpace(parsed.Error.Message)
	}
	return strings.TrimSpace(string(body))
}

func replyFallback(r robot.Robot, msg string) {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return
	}
	out := r.MessageFormat(robot.BasicMarkdown)
	if msgCtx := r.GetMessage(); msgCtx != nil && strings.TrimSpace(msgCtx.Channel) == "" {
		out.Say(msg)
		return
	}
	out.ReplyThread(msg)
}

func sha1String(value string) string {
	sum := sha1.Sum([]byte(value))
	return fmt.Sprintf("%x", sum)
}
