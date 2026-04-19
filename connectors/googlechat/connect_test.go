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
	logs []string
}

func (h *logOnlyHandler) IncomingMessage(*robot.ConnectorMessage)  {}
func (h *logOnlyHandler) GetProtocolConfig(interface{}) error      { return nil }
func (h *logOnlyHandler) GetBrainConfig(interface{}) error         { return nil }
func (h *logOnlyHandler) GetEventStrings() *[]string               { return nil }
func (h *logOnlyHandler) GetHistoryConfig(interface{}) error       { return nil }
func (h *logOnlyHandler) GetBotInfo() robot.BotInfo                { return robot.BotInfo{} }
func (h *logOnlyHandler) SetBotID(string)                          {}
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
