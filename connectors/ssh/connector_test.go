package ssh

import (
	"io"
	"sync"
	"testing"

	"github.com/lnxjedi/gopherbot/robot"
)

type testHandler struct {
	mu   sync.Mutex
	msgs []*robot.ConnectorMessage
	logs []string
}

func (t *testHandler) IncomingMessage(msg *robot.ConnectorMessage) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.msgs = append(t.msgs, msg)
}

func (t *testHandler) GetProtocolConfig(_ interface{}) error { return nil }
func (t *testHandler) GetBrainConfig(_ interface{}) error    { return nil }
func (t *testHandler) GetEventStrings() *[]string            { return nil }
func (t *testHandler) GetHistoryConfig(_ interface{}) error  { return nil }
func (t *testHandler) SetBotID(_ string)                     {}
func (t *testHandler) SetTerminalWriter(_ io.Writer)         {}
func (t *testHandler) SetBotMention(_ string)                {}
func (t *testHandler) GetLogLevel() robot.LogLevel           { return robot.Info }
func (t *testHandler) GetInstallPath() string                { return "" }
func (t *testHandler) GetConfigPath() string                 { return "" }
func (t *testHandler) Log(_ robot.LogLevel, m string, _ ...interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.logs = append(t.logs, m)
}
func (t *testHandler) GetDirectory(_ string) error { return nil }
func (t *testHandler) ExtractID(_ string) (string, bool) {
	return "", false
}
func (t *testHandler) RaisePriv(_ string) {}

func TestShouldSendDMRoutesToSenderAndRecipient(t *testing.T) {
	evt := bufferMsg{
		isDM:     true,
		userName: "alice",
		userID:   "aliceid",
		dmPeer:   "bob",
		dmPeerID: "bobid",
	}

	sender := &sshClient{userName: "alice", userID: "aliceid"}
	recipient := &sshClient{userName: "bob", userID: "bobid"}
	other := &sshClient{userName: "carol", userID: "carolid"}

	if send, _ := sender.shouldSend(evt); !send {
		t.Fatalf("expected sender to receive DM")
	}
	if send, _ := recipient.shouldSend(evt); !send {
		t.Fatalf("expected recipient to receive DM")
	}
	if send, _ := other.shouldSend(evt); send {
		t.Fatalf("expected other user to not receive DM")
	}
}

func TestShouldSendDMIgnoresFilterMode(t *testing.T) {
	evt := bufferMsg{
		isDM:     true,
		userName: "alice",
		userID:   "aliceid",
		dmPeer:   "bob",
		dmPeerID: "bobid",
	}

	client := &sshClient{
		userName:       "bob",
		userID:         "bobid",
		filter:         filterThread,
		typingInThread: true,
		threadID:       "0001",
		channel:        "general",
	}

	if send, _ := client.shouldSend(evt); !send {
		t.Fatalf("expected DM to bypass filter mode")
	}
}

func TestDMLabelVariants(t *testing.T) {
	client := &sshClient{userName: "alice", userID: "aliceid"}

	evtOut := bufferMsg{
		isDM:     true,
		userName: "alice",
		userID:   "aliceid",
		dmPeer:   "bob",
	}
	if got := client.dmLabel(evtOut); got != "to:@bob" {
		t.Fatalf("expected outbound label to be to:@bob, got %q", got)
	}

	evtIn := bufferMsg{
		isDM:     true,
		userName: "bob",
		userID:   "bobid",
		dmPeer:   "alice",
	}
	if got := client.dmLabel(evtIn); got != "from:@bob" {
		t.Fatalf("expected inbound label to be from:@bob, got %q", got)
	}

	client.dmPeer = "bob"
	if got := client.dmLabel(evtIn); got != "@bob" {
		t.Fatalf("expected DM channel inbound label to be @bob, got %q", got)
	}
}

func TestHandleUserInputDirectToBot(t *testing.T) {
	h := &testHandler{}
	sc := &sshConnector{
		handler:      h,
		cfg:          sshConfig{MaxMsgBytes: defaultMaxMsg, UserHistoryLines: 5},
		botName:      "floyd",
		botNameLower: "floyd",
		botID:        "botid",
		buffer:       make([]bufferMsg, 4),
		clients:      make(map[*sshClient]struct{}),
		threads:      make(map[string]int),
	}
	client := &sshClient{
		userName: "alice",
		userID:   "aliceid",
		channel:  "general",
		dmPeer:   "floyd",
		dmPeerID: "botid",
		dmIsBot:  true,
		writer:   io.Discard,
	}

	sc.handleUserInput(client, "ping")

	if len(h.msgs) != 1 {
		t.Fatalf("expected 1 incoming message, got %d", len(h.msgs))
	}
	msg := h.msgs[0]
	if !msg.DirectMessage {
		t.Fatalf("expected DirectMessage=true")
	}
	if msg.HiddenMessage {
		t.Fatalf("expected HiddenMessage=false")
	}
	if msg.ChannelName != "" || msg.ChannelID != "" {
		t.Fatalf("expected empty channel for direct message, got %q/%q", msg.ChannelName, msg.ChannelID)
	}
	if msg.MessageText != "ping" {
		t.Fatalf("expected message text 'ping', got %q", msg.MessageText)
	}
	if msg.UserName != "alice" || msg.UserID != "aliceid" {
		t.Fatalf("unexpected user info: %s/%s", msg.UserName, msg.UserID)
	}

	if sc.buffer[0].isDM != true || sc.buffer[0].dmPeer != "floyd" {
		t.Fatalf("expected DM buffer entry for bot")
	}
}

func TestHandleUserInputDirectUserBypassEngine(t *testing.T) {
	h := &testHandler{}
	sc := &sshConnector{
		handler: h,
		cfg:     sshConfig{MaxMsgBytes: defaultMaxMsg, UserHistoryLines: 5},
		buffer:  make([]bufferMsg, 4),
		clients: make(map[*sshClient]struct{}),
	}
	client := &sshClient{
		userName: "alice",
		userID:   "aliceid",
		channel:  "general",
		dmPeer:   "bob",
		dmPeerID: "bobid",
		dmIsBot:  false,
		writer:   io.Discard,
	}

	sc.handleUserInput(client, "hello")

	if len(h.msgs) != 0 {
		t.Fatalf("expected no incoming message for user-user DM, got %d", len(h.msgs))
	}
	if sc.buffer[0].isDM != true || sc.buffer[0].dmPeer != "bob" {
		t.Fatalf("expected DM buffer entry for user-user DM")
	}
}

func TestHandleUserInputHiddenInUserDMIsDropped(t *testing.T) {
	h := &testHandler{}
	sc := &sshConnector{
		handler: h,
		cfg:     sshConfig{MaxMsgBytes: defaultMaxMsg, UserHistoryLines: 5},
		buffer:  make([]bufferMsg, 2),
		clients: make(map[*sshClient]struct{}),
	}
	client := &sshClient{
		userName: "alice",
		userID:   "aliceid",
		channel:  "general",
		dmPeer:   "bob",
		dmPeerID: "bobid",
		dmIsBot:  false,
		writer:   io.Discard,
	}

	sc.handleUserInput(client, "/ping")

	if len(h.msgs) != 0 {
		t.Fatalf("expected hidden command in user DM to be dropped")
	}
	if sc.bufIndex != 0 {
		t.Fatalf("expected no buffer entries for dropped DM command")
	}
}

func TestHandleUserInputHiddenToBotDirect(t *testing.T) {
	h := &testHandler{}
	sc := &sshConnector{
		handler:      h,
		cfg:          sshConfig{MaxMsgBytes: defaultMaxMsg, UserHistoryLines: 5},
		botName:      "floyd",
		botNameLower: "floyd",
		botID:        "botid",
		buffer:       make([]bufferMsg, 2),
		clients:      make(map[*sshClient]struct{}),
		threads:      make(map[string]int),
	}
	client := &sshClient{
		userName: "alice",
		userID:   "aliceid",
		channel:  "general",
		dmPeer:   "floyd",
		dmPeerID: "botid",
		dmIsBot:  true,
		writer:   io.Discard,
	}

	sc.handleUserInput(client, "/floyd ping")

	if len(h.msgs) != 1 {
		t.Fatalf("expected 1 incoming message, got %d", len(h.msgs))
	}
	msg := h.msgs[0]
	if !msg.DirectMessage {
		t.Fatalf("expected DirectMessage=true")
	}
	if !msg.HiddenMessage {
		t.Fatalf("expected HiddenMessage=true")
	}
	if msg.MessageText != "floyd ping" {
		t.Fatalf("expected hidden payload 'floyd ping', got %q", msg.MessageText)
	}
}

func TestHandleUserInputDirectAtUserBypassEngine(t *testing.T) {
	h := &testHandler{}
	sc := &sshConnector{
		handler:   h,
		cfg:       sshConfig{MaxMsgBytes: defaultMaxMsg, UserHistoryLines: 5},
		buffer:    make([]bufferMsg, 4),
		clients:   make(map[*sshClient]struct{}),
		userNames: map[string]userKeyInfo{"bob": {userName: "bob", userID: "bobid"}},
	}
	client := &sshClient{
		userName: "alice",
		userID:   "aliceid",
		channel:  "general",
		writer:   io.Discard,
	}

	sc.handleUserInput(client, "/@bob hi")

	if len(h.msgs) != 0 {
		t.Fatalf("expected no incoming message for one-shot user DM, got %d", len(h.msgs))
	}
	if sc.buffer[0].isDM != true || sc.buffer[0].dmPeer != "bob" {
		t.Fatalf("expected DM buffer entry for one-shot user DM")
	}
}

func TestSetUserMapRejectsUppercase(t *testing.T) {
	h := &testHandler{}
	sc := &sshConnector{handler: h}
	sc.SetUserMap(map[string]string{
		"Alice": "ssh-ed25519 AAAA",
		"bob":   "ssh-ed25519 BBBB",
	})

	if _, ok := sc.userNames["Alice"]; ok {
		t.Fatalf("expected uppercase username to be rejected")
	}
	if _, ok := sc.userNames["bob"]; !ok {
		t.Fatalf("expected lowercase username to be accepted")
	}
}
