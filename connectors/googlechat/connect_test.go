package googlechat

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/chat/apiv1/chatpb"
	"github.com/lnxjedi/gopherbot/robot"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type logOnlyHandler struct {
	logs           []string
	botID          string
	protocolConfig *config
}

func (h *logOnlyHandler) IncomingMessage(*robot.ConnectorMessage) {}
func (h *logOnlyHandler) GetProtocolConfig(v interface{}) error {
	if h.protocolConfig != nil {
		*(v.(*config)) = *h.protocolConfig
	}
	return nil
}
func (h *logOnlyHandler) GetBrainConfig(interface{}) error         { return nil }
func (h *logOnlyHandler) GetEventStrings() *[]string               { return nil }
func (h *logOnlyHandler) GetHistoryConfig(interface{}) error       { return nil }
func (h *logOnlyHandler) GetBotInfo() robot.BotInfo                { return robot.BotInfo{} }
func (h *logOnlyHandler) SetBotID(id string)                       { h.botID = id }
func (h *logOnlyHandler) SetTerminalWriter(io.Writer)              {}
func (h *logOnlyHandler) SetBotMention(string)                     {}
func (h *logOnlyHandler) GetLogLevel() robot.LogLevel              { return robot.Debug }
func (h *logOnlyHandler) GetInstallPath() string                   { return "" }
func (h *logOnlyHandler) GetConfigPath() string                    { return "" }
func (h *logOnlyHandler) ReadEncryptedFile(string) ([]byte, error) { return nil, nil }
func (h *logOnlyHandler) Log(_ robot.LogLevel, m string, v ...interface{}) {
	h.logs = append(h.logs, m)
}
func (h *logOnlyHandler) GetDirectory(string) error { return nil }
func (h *logOnlyHandler) RaisePriv(string)          {}

func TestNormalizeConfiguredUserMap(t *testing.T) {
	h := &logOnlyHandler{}
	got := normalizeConfiguredUserMap(map[string]string{
		"alice": "users/123",
		"bob":   "456",
		"":      "users/789",
	}, h)
	if got["alice"] != "users/123" {
		t.Fatalf("alice = %q", got["alice"])
	}
	if got["bob"] != "users/456" {
		t.Fatalf("bob = %q", got["bob"])
	}
	if _, ok := got[""]; ok {
		t.Fatal("unexpected empty username entry")
	}
}

func TestNormalizeSubscriptionID(t *testing.T) {
	if got := normalizeSubscriptionID("projects/p/subscriptions/gopherbot-chat-sub"); got != "gopherbot-chat-sub" {
		t.Fatalf("normalizeSubscriptionID() = %q", got)
	}
}

func TestResolveUserIDPrefersConfiguredMapForUsername(t *testing.T) {
	connector := &googleChatConnector{
		botUserMap: map[string]string{"parsley": "users/104265192829011490173"},
		usersByID:  make(map[string]chatUserRecord),
		usersByName: map[string]chatUserRecord{
			"parsley": {ResourceName: "users/104265192829011490173", CanonicalName: "parsley"},
		},
	}

	got, ok := connector.resolveUserID("parsley", "parsley")
	if !ok {
		t.Fatal("resolveUserID() = not ok")
	}
	if got != "users/104265192829011490173" {
		t.Fatalf("resolveUserID() = %q", got)
	}
}

func TestResolveUserIDDoesNotInventResourceNameFromUsername(t *testing.T) {
	connector := &googleChatConnector{
		botUserMap:       map[string]string{},
		usersByID:        make(map[string]chatUserRecord),
		usersByName:      make(map[string]chatUserRecord),
		channelsByID:     make(map[string]chatChannelRecord),
		channelIDsByName: make(map[string]string),
	}

	got, ok := connector.resolveUserID("parsley", "parsley")
	if ok {
		t.Fatalf("resolveUserID() unexpectedly succeeded with %q", got)
	}
}

func TestReloadAtomicallySwapsConfiguredUserMap(t *testing.T) {
	h := &logOnlyHandler{
		protocolConfig: &config{UserMap: map[string]string{
			"bob":   "users/222",
			"david": "333",
		}},
	}
	connector := &googleChatConnector{
		Handler: h,
		botUserMap: map[string]string{
			"alice": "users/111",
		},
		configuredUsers: map[string]string{
			"users/111": "alice",
		},
		usersByID: map[string]chatUserRecord{
			"users/111": {ResourceName: "users/111", CanonicalName: "alice"},
			"users/222": {ResourceName: "users/222", DisplayName: "Bob Example"},
		},
		usersByName: map[string]chatUserRecord{
			"alice": {ResourceName: "users/111", CanonicalName: "alice"},
		},
	}

	if err := connector.Reload(); err != nil {
		t.Fatalf("Reload() error = %v", err)
	}

	if _, ok := connector.botUserMap["alice"]; ok {
		t.Fatalf("expected removed configured user alice to be absent from botUserMap")
	}
	if connector.botUserMap["bob"] != "users/222" || connector.botUserMap["david"] != "users/333" {
		t.Fatalf("unexpected botUserMap after reload: %#v", connector.botUserMap)
	}
	if _, ok := connector.configuredUsers["users/111"]; ok {
		t.Fatalf("expected old configured resource to be removed")
	}
	if connector.configuredUsers["users/222"] != "bob" || connector.configuredUsers["users/333"] != "david" {
		t.Fatalf("unexpected configuredUsers after reload: %#v", connector.configuredUsers)
	}
	if _, ok := connector.usersByName["alice"]; ok {
		t.Fatalf("expected stale canonical user index to be removed")
	}
	if record := connector.usersByID["users/222"]; record.CanonicalName != "bob" {
		t.Fatalf("expected cached user record to receive new canonical name, got %+v", record)
	}
}

func TestNormalizeIncomingMentionRewritesToBotNameWithoutBotMessage(t *testing.T) {
	mentionText := "@Bishop Gopherbot"
	connector := &googleChatConnector{
		Handler:          &logOnlyHandler{},
		botName:          "bishop",
		usersByID:        make(map[string]chatUserRecord),
		usersByName:      make(map[string]chatUserRecord),
		channelsByID:     make(map[string]chatChannelRecord),
		channelIDsByName: make(map[string]string),
	}
	msg, ok := connector.normalizeIncomingMessage(&chatEvent{
		Type: "MESSAGE",
		User: &chatEventUser{Name: "users/123", DisplayName: "Alice Example"},
		Space: &chatEventSpace{
			Name:        "spaces/AAAA",
			DisplayName: "Ops",
			SpaceType:   "SPACE",
		},
		Message: &chatEventMessage{
			Name:         "spaces/AAAA/messages/BBBB",
			Text:         mentionText + " ping",
			ArgumentText: "ping",
			Annotations: []*chatEventAnnotation{
				{
					Type:       "USER_MENTION",
					StartIndex: 0,
					Length:     len([]rune(mentionText)),
					UserMention: &chatEventUserMentionMeta{
						Type: "MENTION",
						User: &chatEventUser{Name: "users/app"},
					},
				},
			},
		},
	})
	if !ok {
		t.Fatal("normalizeIncomingMessage() = not ok")
	}
	if msg.BotMessage {
		t.Fatal("mention should not force explicit bot-message handling")
	}
	if msg.MessageText != "@bishop ping" {
		t.Fatalf("MessageText = %q", msg.MessageText)
	}
}

func TestNormalizeIncomingMentionRewritesToBotNameWithConfiguredSelfID(t *testing.T) {
	mentionText := "@Bishop Gopherbot"
	connector := &googleChatConnector{
		Handler:          &logOnlyHandler{},
		botName:          "bishop",
		selfID:           "users/999",
		usersByID:        make(map[string]chatUserRecord),
		usersByName:      make(map[string]chatUserRecord),
		channelsByID:     make(map[string]chatChannelRecord),
		channelIDsByName: make(map[string]string),
	}
	msg, ok := connector.normalizeIncomingMessage(&chatEvent{
		Type: "MESSAGE",
		User: &chatEventUser{Name: "users/123", DisplayName: "Alice Example"},
		Space: &chatEventSpace{
			Name:        "spaces/AAAA",
			DisplayName: "Ops",
			SpaceType:   "SPACE",
		},
		Message: &chatEventMessage{
			Name: "spaces/AAAA/messages/BBBB",
			Text: mentionText + " ping",
			Annotations: []*chatEventAnnotation{
				{
					Type:       "USER_MENTION",
					StartIndex: 0,
					Length:     len([]rune(mentionText)),
					UserMention: &chatEventUserMentionMeta{
						Type: "MENTION",
						User: &chatEventUser{Name: "users/999", Type: "BOT"},
					},
				},
			},
		},
	})
	if !ok {
		t.Fatal("normalizeIncomingMessage() = not ok")
	}
	if msg.MessageText != "@bishop ping" {
		t.Fatalf("MessageText = %q", msg.MessageText)
	}
}

func TestNormalizeIncomingSlashCommandRemainsExplicit(t *testing.T) {
	connector := &googleChatConnector{
		Handler:          &logOnlyHandler{},
		botName:          "bishop",
		usersByID:        make(map[string]chatUserRecord),
		usersByName:      make(map[string]chatUserRecord),
		channelsByID:     make(map[string]chatChannelRecord),
		channelIDsByName: make(map[string]string),
	}
	msg, ok := connector.normalizeIncomingMessage(&chatEvent{
		Type: "MESSAGE",
		User: &chatEventUser{Name: "users/123", DisplayName: "Alice Example"},
		Message: &chatEventMessage{
			Name:         "spaces/AAAA/messages/BBBB",
			Text:         "/bishop ping",
			ArgumentText: "ping",
			SlashCommand: &chatEventSlashCommand{CommandId: 7},
		},
	})
	if !ok {
		t.Fatal("normalizeIncomingMessage() = not ok")
	}
	if !msg.BotMessage {
		t.Fatal("slash command should remain explicit")
	}
	if !msg.HiddenMessage {
		t.Fatal("slash command should remain hidden")
	}
	if msg.MessageText != "ping" {
		t.Fatalf("MessageText = %q", msg.MessageText)
	}
}

func TestNormalizeIncomingMessageTreatsConfiguredSelfIDAsSelf(t *testing.T) {
	connector := &googleChatConnector{
		Handler:          &logOnlyHandler{},
		botName:          "bishop",
		selfID:           "users/999",
		usersByID:        make(map[string]chatUserRecord),
		usersByName:      make(map[string]chatUserRecord),
		channelsByID:     make(map[string]chatChannelRecord),
		channelIDsByName: make(map[string]string),
	}
	msg, ok := connector.normalizeIncomingMessage(&chatEvent{
		Type: "MESSAGE",
		User: &chatEventUser{Name: "users/999", Type: "BOT"},
		Message: &chatEventMessage{
			Name: "spaces/AAAA/messages/BBBB",
			Text: "hello from self",
		},
	})
	if !ok {
		t.Fatal("normalizeIncomingMessage() = not ok")
	}
	if !msg.SelfMessage {
		t.Fatal("expected self message")
	}
}

func TestHandleEventRobotValidationLearnsSelfID(t *testing.T) {
	handler := &logOnlyHandler{}
	connector := &googleChatConnector{
		Handler:          handler,
		botName:          "bishop",
		retrySleep:       func(time.Duration) {},
		usersByID:        make(map[string]chatUserRecord),
		usersByName:      make(map[string]chatUserRecord),
		channelsByID:     make(map[string]chatChannelRecord),
		channelIDsByName: make(map[string]string),
		recentMessages:   make(map[string]time.Time),
	}

	code, resultCh, err := connector.IssueRobotValidation()
	if err != nil {
		t.Fatalf("IssueRobotValidation() error = %v", err)
	}
	if resultCh == nil {
		t.Fatal("IssueRobotValidation() returned nil result channel")
	}

	err = connector.handleEvent(&chatEvent{
		Type: "MESSAGE",
		User: &chatEventUser{Name: "users/123", DisplayName: "Alice Example", Type: "HUMAN"},
		Space: &chatEventSpace{
			Name:      "spaces/AAA",
			SpaceType: "SPACE",
		},
		Message: &chatEventMessage{
			Name: "spaces/AAA/messages/BBBB",
			Text: "@Bishop " + code,
			Annotations: []*chatEventAnnotation{
				{
					Type:       "USER_MENTION",
					StartIndex: 0,
					Length:     len([]rune("@Bishop")),
					UserMention: &chatEventUserMentionMeta{
						Type: "MENTION",
						User: &chatEventUser{Name: "users/999", Type: "BOT"},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("handleEvent() error = %v", err)
	}
	if got := connector.CurrentSelfID(); got != "users/999" {
		t.Fatalf("CurrentSelfID() = %q", got)
	}
	if handler.botID != "users/999" {
		t.Fatalf("SetBotID() = %q", handler.botID)
	}
	select {
	case result := <-resultCh:
		if result.BotID != "users/999" {
			t.Fatalf("result.BotID = %q", result.BotID)
		}
		if result.AckSpace != "spaces/AAA" {
			t.Fatalf("result.AckSpace = %q", result.AckSpace)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for validation result")
	}
}

func TestNormalizeIncomingMessageFallsBackToCachedChannelDisplayName(t *testing.T) {
	connector := &googleChatConnector{
		Handler:          &logOnlyHandler{},
		botName:          "bishop",
		usersByID:        make(map[string]chatUserRecord),
		usersByName:      make(map[string]chatUserRecord),
		channelsByID:     map[string]chatChannelRecord{"spaces/AAAA": {ResourceName: "spaces/AAAA", DisplayName: "random"}},
		channelIDsByName: map[string]string{"random": "spaces/AAAA"},
	}

	msg, ok := connector.normalizeIncomingMessage(&chatEvent{
		Type: "MESSAGE",
		User: &chatEventUser{Name: "users/123", DisplayName: "Alice Example"},
		Space: &chatEventSpace{
			Name:      "spaces/AAAA",
			SpaceType: "SPACE",
		},
		Message: &chatEventMessage{
			Name: "spaces/AAAA/messages/BBBB",
			Text: "Bishop, tell me a joke",
		},
	})
	if !ok {
		t.Fatal("normalizeIncomingMessage() = not ok")
	}
	if msg.ChannelName != "random" {
		t.Fatalf("ChannelName = %q", msg.ChannelName)
	}
}

func TestChatEventUnmarshalSlashCommandStringID(t *testing.T) {
	var event chatEvent
	payload := []byte(`{
		"type": "MESSAGE",
		"user": {"name": "users/123"},
		"message": {
			"name": "spaces/AAAA/messages/BBBB",
			"text": "/bishop ping",
			"argumentText": "ping",
			"slashCommand": {"commandId": "1"}
		}
	}`)
	if err := json.Unmarshal(payload, &event); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if event.Message == nil || event.Message.SlashCommand == nil {
		t.Fatal("slash command missing after unmarshal")
	}
	if int64(event.Message.SlashCommand.CommandId) != 1 {
		t.Fatalf("CommandId = %d", event.Message.SlashCommand.CommandId)
	}
}

func TestSendMessageRetriesWithStableRequestID(t *testing.T) {
	connector := &googleChatConnector{
		Handler:    &logOnlyHandler{},
		retrySleep: func(time.Duration) {},
	}

	var gotRequestIDs []string
	attempts := 0
	connector.createMessage = func(_ context.Context, req *chatpb.CreateMessageRequest) (*chatpb.Message, error) {
		attempts++
		gotRequestIDs = append(gotRequestIDs, req.GetRequestId())
		if attempts == 1 {
			return nil, context.DeadlineExceeded
		}
		return &chatpb.Message{Name: "spaces/AAA/messages/BBB"}, nil
	}

	ret := connector.sendMessage("spaces/AAA", "", "", "hello", robot.Variable, nil)
	if ret != robot.Ok {
		t.Fatalf("sendMessage() ret = %v", ret)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
	if len(gotRequestIDs) != 2 {
		t.Fatalf("request ID count = %d, want 2", len(gotRequestIDs))
	}
	if gotRequestIDs[0] == "" {
		t.Fatal("request ID was empty")
	}
	if gotRequestIDs[0] != gotRequestIDs[1] {
		t.Fatalf("request IDs differ: %q vs %q", gotRequestIDs[0], gotRequestIDs[1])
	}
}

func TestSendMessageDoesNotRetryPermanentError(t *testing.T) {
	connector := &googleChatConnector{
		Handler:    &logOnlyHandler{},
		retrySleep: func(time.Duration) {},
	}

	attempts := 0
	connector.createMessage = func(_ context.Context, req *chatpb.CreateMessageRequest) (*chatpb.Message, error) {
		attempts++
		if req.GetRequestId() == "" {
			t.Fatal("request ID was empty")
		}
		return nil, status.Error(codes.InvalidArgument, "bad request")
	}

	ret := connector.sendMessage("spaces/AAA", "", "", "hello", robot.Variable, nil)
	if ret != robot.FailedMessageSend {
		t.Fatalf("sendMessage() ret = %v, want %v", ret, robot.FailedMessageSend)
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1", attempts)
	}
}

func TestSendProtocolUserMessageRetriesDirectMessageLookup(t *testing.T) {
	connector := &googleChatConnector{
		Handler:    &logOnlyHandler{},
		retrySleep: func(time.Duration) {},
		usersByID:  make(map[string]chatUserRecord),
		usersByName: map[string]chatUserRecord{
			"alice": {ResourceName: "users/123", CanonicalName: "alice"},
		},
	}

	lookupAttempts := 0
	connector.findDirectMessage = func(_ context.Context, req *chatpb.FindDirectMessageRequest) (*chatpb.Space, error) {
		lookupAttempts++
		if req.GetName() != "users/123" {
			t.Fatalf("FindDirectMessage name = %q", req.GetName())
		}
		if lookupAttempts == 1 {
			return nil, status.Error(codes.Unavailable, "try again")
		}
		return &chatpb.Space{Name: "spaces/DM123"}, nil
	}

	sendAttempts := 0
	connector.createMessage = func(_ context.Context, req *chatpb.CreateMessageRequest) (*chatpb.Message, error) {
		sendAttempts++
		if req.GetParent() != "spaces/DM123" {
			t.Fatalf("CreateMessage parent = %q", req.GetParent())
		}
		return &chatpb.Message{Name: "spaces/DM123/messages/BBB"}, nil
	}

	ret := connector.SendProtocolUserMessage("alice", "hello", robot.Variable, nil)
	if ret != robot.Ok {
		t.Fatalf("SendProtocolUserMessage() ret = %v", ret)
	}
	if lookupAttempts != 2 {
		t.Fatalf("lookup attempts = %d, want 2", lookupAttempts)
	}
	if sendAttempts != 1 {
		t.Fatalf("send attempts = %d, want 1", sendAttempts)
	}
}

func TestSummarizePubSubAttributesSortsKeys(t *testing.T) {
	summary := summarizePubSubAttributes(map[string]string{
		"zeta":  "last",
		"alpha": "first",
	})
	if summary != `alpha="first", zeta="last"` {
		t.Fatalf("summary = %q", summary)
	}
}

func TestSummarizeChatEventIncludesInteractionFlags(t *testing.T) {
	summary := summarizeChatEvent(&chatEvent{
		Type: "MESSAGE",
		User: &chatEventUser{Name: "users/123"},
		Space: &chatEventSpace{
			Name:      "spaces/AAAA",
			SpaceType: "DIRECT_MESSAGE",
		},
		Message: &chatEventMessage{
			Name:         "spaces/AAAA/messages/BBBB",
			Text:         "/bishop ping",
			ArgumentText: "ping",
			SlashCommand: &chatEventSlashCommand{CommandId: 7},
		},
	})
	for _, fragment := range []string{
		`type="MESSAGE"`,
		`message="spaces/AAAA/messages/BBBB"`,
		`user="users/123"`,
		`space="spaces/AAAA"`,
		`direct=true`,
		`slash=true`,
		`appCommand=false`,
		`text="/bishop ping"`,
		`argument="ping"`,
	} {
		if !strings.Contains(summary, fragment) {
			t.Fatalf("summary %q missing %q", summary, fragment)
		}
	}
}
