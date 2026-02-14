package ssh

import (
	"io"
	"sync"
	"testing"
	"time"

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

func TestInjectMessageReturnsCursorAndThreadMetadata(t *testing.T) {
	h := &testHandler{}
	sc := &sshConnector{
		handler:      h,
		cfg:          sshConfig{DefaultChannel: "general", MaxMsgBytes: defaultMaxMsg},
		botName:      "floyd",
		botNameLower: "floyd",
		botID:        "botid",
		clients:      make(map[*sshClient]struct{}),
		userNames:    map[string]userKeyInfo{"alice": {userName: "alice", userID: "aliceid"}},
		userIDs:      map[string]userKeyInfo{"aliceid": {userName: "alice", userID: "aliceid"}},
		threads:      make(map[string]int),
		buffer:       make([]bufferMsg, 8),
		waiters:      make(map[chan struct{}]struct{}),
	}

	res, err := sc.InjectMessage(robot.InjectMessageRequest{
		AsUser: "alice",
		Text:   "status",
	})
	if err != nil {
		t.Fatalf("InjectMessage returned error: %v", err)
	}
	if len(h.msgs) != 1 {
		t.Fatalf("expected 1 incoming message, got %d", len(h.msgs))
	}
	if res.Cursor == 0 {
		t.Fatalf("expected non-zero cursor")
	}
	if res.ThreadID == "" {
		t.Fatalf("expected thread ID in response")
	}
	if res.MessageID == "" {
		t.Fatalf("expected message ID in response")
	}
}

func TestGetMessagesHiddenVisibilityScopedToViewer(t *testing.T) {
	h := &testHandler{}
	sc := &sshConnector{
		handler:      h,
		cfg:          sshConfig{DefaultChannel: "general"},
		botName:      "floyd",
		botNameLower: "floyd",
		botID:        "botid",
		clients:      make(map[*sshClient]struct{}),
		userNames: map[string]userKeyInfo{
			"alice": {userName: "alice", userID: "aliceid"},
			"bob":   {userName: "bob", userID: "bobid"},
		},
		userIDs: map[string]userKeyInfo{
			"aliceid": {userName: "alice", userID: "aliceid"},
			"bobid":   {userName: "bob", userID: "bobid"},
		},
		threads: make(map[string]int),
		buffer:  make([]bufferMsg, 8),
		waiters: make(map[chan struct{}]struct{}),
	}

	evt := bufferMsg{
		timestamp: time.Now(),
		userName:  sc.botName,
		userID:    sc.botID,
		isBot:     true,
		channel:   "general",
		threadID:  "0001",
		threaded:  true,
		text:      "private reply",
	}
	sc.broadcast(evt, &robot.ConnectorMessage{HiddenMessage: true, UserID: "aliceid"})

	aliceBatch, err := sc.GetMessages(robot.MessageQuery{Viewer: "alice", All: true})
	if err != nil {
		t.Fatalf("alice GetMessages error: %v", err)
	}
	if len(aliceBatch.Messages) != 1 {
		t.Fatalf("expected alice to receive 1 hidden message, got %d", len(aliceBatch.Messages))
	}
	if !aliceBatch.Messages[0].Hidden {
		t.Fatalf("expected hidden message flag for alice")
	}

	bobBatch, err := sc.GetMessages(robot.MessageQuery{Viewer: "bob", All: true})
	if err != nil {
		t.Fatalf("bob GetMessages error: %v", err)
	}
	if len(bobBatch.Messages) != 0 {
		t.Fatalf("expected bob to receive 0 hidden messages, got %d", len(bobBatch.Messages))
	}
}

func TestGetMessagesWaitsForNewMessage(t *testing.T) {
	sc := &sshConnector{
		cfg: sshConfig{DefaultChannel: "general"},
		userNames: map[string]userKeyInfo{
			"alice": {userName: "alice", userID: "aliceid"},
		},
		userIDs: map[string]userKeyInfo{
			"aliceid": {userName: "alice", userID: "aliceid"},
		},
		buffer:  make([]bufferMsg, 8),
		waiters: make(map[chan struct{}]struct{}),
	}

	after := sc.latestCursor()
	done := make(chan robot.MessageBatch, 1)
	go func() {
		batch, _ := sc.GetMessages(robot.MessageQuery{
			Viewer:      "alice",
			AfterCursor: after,
			TimeoutMS:   200,
		})
		done <- batch
	}()

	time.Sleep(30 * time.Millisecond)
	sc.appendBuffer(bufferMsg{
		timestamp: time.Now(),
		userName:  "carol",
		userID:    "carolid",
		channel:   "general",
		text:      "hello",
	})

	select {
	case batch := <-done:
		if batch.TimedOut {
			t.Fatalf("expected message arrival, got timed out response")
		}
		if len(batch.Messages) == 0 {
			t.Fatalf("expected at least one message in batch")
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for GetMessages result")
	}
}
