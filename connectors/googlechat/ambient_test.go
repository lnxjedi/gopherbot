package googlechat

import (
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
	chatapi "google.golang.org/api/chat/v1"
)

type recordingHandler struct {
	logOnlyHandler
	messages []*robot.ConnectorMessage
}

func (h *recordingHandler) IncomingMessage(msg *robot.ConnectorMessage) {
	h.messages = append(h.messages, msg)
}

func TestTargetResourceForSpace(t *testing.T) {
	if got := targetResourceForSpace("spaces/AAAA"); got != "//chat.googleapis.com/spaces/AAAA" {
		t.Fatalf("targetResourceForSpace() = %q", got)
	}
}

func TestNormalizeAmbientMessageMappedUser(t *testing.T) {
	handler := &recordingHandler{}
	connector := &googleChatConnector{
		Handler:          handler,
		chatClient:       nil,
		botUserMap:       map[string]string{"alice": "users/123"},
		configuredUsers:  map[string]string{"users/123": "alice"},
		usersByID:        make(map[string]chatUserRecord),
		usersByName:      make(map[string]chatUserRecord),
		channelsByID:     make(map[string]chatChannelRecord),
		channelIDsByName: make(map[string]string),
		unmappedUsers:    make(map[string]bool),
	}
	msg, ok := connector.normalizeAmbientMessage(&chatapi.Message{
		Name:        "spaces/AAAA/messages/BBBB",
		Text:        "hello world",
		ThreadReply: true,
		Thread:      &chatapi.Thread{Name: "spaces/AAAA/threads/CCCC"},
		Sender:      &chatapi.User{Name: "users/123", DisplayName: "Alice Example"},
		Space:       &chatapi.Space{Name: "spaces/AAAA", DisplayName: "Ops", SpaceType: "SPACE"},
	})
	if !ok {
		t.Fatal("normalizeAmbientMessage() = not ok")
	}
	if msg.UserName != "alice" {
		t.Fatalf("UserName = %q", msg.UserName)
	}
	if msg.ChannelID != "spaces/AAAA" {
		t.Fatalf("ChannelID = %q", msg.ChannelID)
	}
	if msg.BotMessage {
		t.Fatal("plain ambient message should not be treated as addressed to the bot")
	}
	if !msg.ThreadedMessage {
		t.Fatal("expected threaded ambient message")
	}
}

func TestNormalizeAmbientMessageMentionBecomesBotMessage(t *testing.T) {
	mentionText := "@Bishop Gopherbot"
	handler := &recordingHandler{}
	connector := &googleChatConnector{
		Handler:          handler,
		botName:          "bishop",
		usersByID:        make(map[string]chatUserRecord),
		usersByName:      make(map[string]chatUserRecord),
		channelsByID:     make(map[string]chatChannelRecord),
		channelIDsByName: make(map[string]string),
		unmappedUsers:    make(map[string]bool),
	}
	msg, ok := connector.normalizeAmbientMessage(&chatapi.Message{
		Name:         "spaces/AAAA/messages/BBBB",
		Text:         mentionText + " ping",
		ArgumentText: "ping",
		Sender:       &chatapi.User{Name: "users/123", DisplayName: "Alice Example"},
		Space:        &chatapi.Space{Name: "spaces/AAAA", DisplayName: "Ops", SpaceType: "SPACE"},
		Annotations: []*chatapi.Annotation{
			{
				Type:       "USER_MENTION",
				StartIndex: 0,
				Length:     int64(len([]rune(mentionText))),
				UserMention: &chatapi.UserMentionMetadata{
					User: &chatapi.User{Name: "users/app"},
					Type: "MENTION",
				},
			},
		},
	})
	if !ok {
		t.Fatal("normalizeAmbientMessage() = not ok")
	}
	if msg.BotMessage {
		t.Fatal("mentioned ambient message should still flow through normal name matching")
	}
	if msg.MessageText != "@bishop ping" {
		t.Fatalf("MessageText = %q", msg.MessageText)
	}
}

func TestNormalizeAmbientMessageMidSentenceMentionIsRewritten(t *testing.T) {
	mentionText := "@Bishop Gopherbot"
	prefix := "Did you see what "
	handler := &recordingHandler{}
	connector := &googleChatConnector{
		Handler:          handler,
		botName:          "bishop",
		usersByID:        make(map[string]chatUserRecord),
		usersByName:      make(map[string]chatUserRecord),
		channelsByID:     make(map[string]chatChannelRecord),
		channelIDsByName: make(map[string]string),
		unmappedUsers:    make(map[string]bool),
	}
	msg, ok := connector.normalizeAmbientMessage(&chatapi.Message{
		Name:   "spaces/AAAA/messages/BBBB",
		Text:   prefix + mentionText + " did?",
		Sender: &chatapi.User{Name: "users/123", DisplayName: "Alice Example"},
		Space:  &chatapi.Space{Name: "spaces/AAAA", DisplayName: "Ops", SpaceType: "SPACE"},
		Annotations: []*chatapi.Annotation{
			{
				Type:       "USER_MENTION",
				StartIndex: int64(len([]rune(prefix))),
				Length:     int64(len([]rune(mentionText))),
				UserMention: &chatapi.UserMentionMetadata{
					User: &chatapi.User{Name: "users/app"},
					Type: "MENTION",
				},
			},
		},
	})
	if !ok {
		t.Fatal("normalizeAmbientMessage() = not ok")
	}
	if msg.BotMessage {
		t.Fatal("mid-sentence bot mention should not become an explicit bot message")
	}
	if msg.MessageText != "Did you see what @bishop did?" {
		t.Fatalf("MessageText = %q", msg.MessageText)
	}
}

func TestNormalizeAmbientMessageFallsBackToCachedChannelDisplayName(t *testing.T) {
	handler := &recordingHandler{}
	connector := &googleChatConnector{
		Handler:          handler,
		botName:          "bishop",
		usersByID:        make(map[string]chatUserRecord),
		usersByName:      make(map[string]chatUserRecord),
		channelsByID:     map[string]chatChannelRecord{"spaces/AAAA": {ResourceName: "spaces/AAAA", DisplayName: "random"}},
		channelIDsByName: map[string]string{"random": "spaces/AAAA"},
		unmappedUsers:    make(map[string]bool),
	}

	msg, ok := connector.normalizeAmbientMessage(&chatapi.Message{
		Name:   "spaces/AAAA/messages/BBBB",
		Text:   "Bishop, tell me a joke",
		Sender: &chatapi.User{Name: "users/123", DisplayName: "Alice Example"},
		Space:  &chatapi.Space{Name: "spaces/AAAA", SpaceType: "SPACE"},
	})
	if !ok {
		t.Fatal("normalizeAmbientMessage() = not ok")
	}
	if msg.ChannelName != "random" {
		t.Fatalf("ChannelName = %q", msg.ChannelName)
	}
}

func TestHandleWorkspaceMessageCreated(t *testing.T) {
	handler := &recordingHandler{}
	connector := &googleChatConnector{
		Handler:          handler,
		ambientMessages:  true,
		usersByID:        make(map[string]chatUserRecord),
		usersByName:      make(map[string]chatUserRecord),
		channelsByID:     make(map[string]chatChannelRecord),
		channelIDsByName: make(map[string]string),
		unmappedUsers:    make(map[string]bool),
		recentMessages:   make(map[string]time.Time),
	}

	if err := connector.handleWorkspaceMessageCreated(mustJSON(t, workspaceMessageCreatedEventData{
		Message: &chatapi.Message{
			Name:   "spaces/AAAA/messages/BBBB",
			Text:   "ambient hello",
			Sender: &chatapi.User{Name: "users/123", DisplayName: "Alice Example"},
			Space:  &chatapi.Space{Name: "spaces/AAAA", DisplayName: "Ops", SpaceType: "SPACE"},
		},
	})); err != nil {
		t.Fatalf("handleWorkspaceMessageCreated() error = %v", err)
	}
	if len(handler.messages) != 1 {
		t.Fatalf("incoming messages = %d", len(handler.messages))
	}
	if handler.messages[0].MessageText != "ambient hello" {
		t.Fatalf("MessageText = %q", handler.messages[0].MessageText)
	}
}

func mustJSON(t *testing.T, v interface{}) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return data
}

func (h *recordingHandler) GetProtocolConfig(interface{}) error      { return nil }
func (h *recordingHandler) GetBrainConfig(interface{}) error         { return nil }
func (h *recordingHandler) GetEventStrings() *[]string               { return nil }
func (h *recordingHandler) GetHistoryConfig(interface{}) error       { return nil }
func (h *recordingHandler) GetBotInfo() robot.BotInfo                { return robot.BotInfo{} }
func (h *recordingHandler) SetBotID(string)                          {}
func (h *recordingHandler) SetTerminalWriter(io.Writer)              {}
func (h *recordingHandler) SetBotMention(string)                     {}
func (h *recordingHandler) GetLogLevel() robot.LogLevel              { return robot.Debug }
func (h *recordingHandler) GetInstallPath() string                   { return "" }
func (h *recordingHandler) GetConfigPath() string                    { return "" }
func (h *recordingHandler) ReadEncryptedFile(string) ([]byte, error) { return nil, nil }
func (h *recordingHandler) GetDirectory(string) error                { return nil }
func (h *recordingHandler) RaisePriv(string)                         {}
