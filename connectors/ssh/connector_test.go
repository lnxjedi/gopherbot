package ssh

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
)

type stubAddr string

func (a stubAddr) Network() string { return "tcp" }
func (a stubAddr) String() string  { return string(a) }

type stubListener struct {
	addr net.Addr
}

func (l *stubListener) Accept() (net.Conn, error) { return nil, io.EOF }
func (l *stubListener) Close() error              { return nil }
func (l *stubListener) Addr() net.Addr            { return l.addr }

func portFromListenAddress(t *testing.T, address string) int {
	t.Helper()
	_, portText, err := net.SplitHostPort(address)
	if err != nil {
		t.Fatalf("SplitHostPort(%q): %v", address, err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		t.Fatalf("Atoi(%q): %v", portText, err)
	}
	return port
}

func withStubSSHNetListen(t *testing.T, fn func(network, address string) (net.Listener, error)) {
	t.Helper()
	original := sshNetListen
	sshNetListen = fn
	t.Cleanup(func() {
		sshNetListen = original
	})
}

type testHandler struct {
	mu             sync.Mutex
	msgs           []*robot.ConnectorMessage
	logs           []string
	protocolConfig *sshConfig
}

func (t *testHandler) IncomingMessage(msg *robot.ConnectorMessage) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.msgs = append(t.msgs, msg)
}

func (t *testHandler) GetProtocolConfig(v interface{}) error {
	if t.protocolConfig != nil {
		*(v.(*sshConfig)) = *t.protocolConfig
	}
	return nil
}
func (t *testHandler) GetBrainConfig(_ interface{}) error   { return nil }
func (t *testHandler) GetEventStrings() *[]string           { return nil }
func (t *testHandler) GetHistoryConfig(_ interface{}) error { return nil }
func (t *testHandler) GetBotInfo() robot.BotInfo {
	return robot.BotInfo{UserName: "floyd", FullName: "Floyd Gopherbot"}
}
func (t *testHandler) SetBotID(_ string)                          {}
func (t *testHandler) SetTerminalWriter(_ io.Writer)              {}
func (t *testHandler) SetBotMention(_ string)                     {}
func (t *testHandler) GetLogLevel() robot.LogLevel                { return robot.Info }
func (t *testHandler) GetInstallPath() string                     { return "" }
func (t *testHandler) GetConfigPath() string                      { return "" }
func (t *testHandler) ReadEncryptedFile(_ string) ([]byte, error) { return nil, nil }
func (t *testHandler) Log(_ robot.LogLevel, m string, args ...interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.logs = append(t.logs, fmt.Sprintf(m, args...))
}
func (t *testHandler) GetDirectory(_ string) error { return nil }
func (t *testHandler) RaisePriv(_ string)          {}

func TestAnnounceJoinBroadcastsToOtherUsersOnlyAndForwardsIncoming(t *testing.T) {
	h := &testHandler{}
	joiningOut := &bytes.Buffer{}
	otherOut := &bytes.Buffer{}
	joining := &sshClient{
		userName: "alice",
		userID:   "aliceid",
		channel:  "general",
		filter:   filterChannel,
		writer:   joiningOut,
	}
	other := &sshClient{
		userName: "bob",
		userID:   "bobid",
		channel:  "general",
		filter:   filterChannel,
		writer:   otherOut,
	}
	sc := &sshConnector{
		handler: h,
		cfg: sshConfig{
			DefaultChannel: "general",
			HearSelf:       true,
		},
		botName: "floyd",
		botID:   "botid",
		buffer:  make([]bufferMsg, 8),
		clients: map[*sshClient]struct{}{
			joining: {},
			other:   {},
		},
		threads: make(map[string]int),
		waiters: make(map[chan struct{}]struct{}),
	}

	sc.announceJoin(joining)

	if got := sc.latestCursor(); got != 1 {
		t.Fatalf("expected cursor=1 after join announcement, got %d", got)
	}
	snap := sc.snapshotBuffer()
	if len(snap) != 1 {
		t.Fatalf("expected one buffered join announcement, got %d", len(snap))
	}
	evt := snap[0]
	if evt.userName != "floyd" || !evt.isBot {
		t.Fatalf("expected bot-authored announcement, got user=%q isBot=%t", evt.userName, evt.isBot)
	}
	if evt.channel != "general" {
		t.Fatalf("expected join announcement in #general, got %q", evt.channel)
	}
	if evt.text != "@alice has joined #general" {
		t.Fatalf("unexpected announcement text: %q", evt.text)
	}

	if joiningOut.Len() != 0 {
		t.Fatalf("joining user should not receive self join announcement, got %q", joiningOut.String())
	}
	if otherOut.Len() == 0 {
		t.Fatalf("other user should receive join announcement")
	}
	if len(h.msgs) != 1 {
		t.Fatalf("expected one forwarded incoming message, got %d", len(h.msgs))
	}
	msg := h.msgs[0]
	if msg.Protocol != "ssh" {
		t.Fatalf("expected protocol ssh, got %q", msg.Protocol)
	}
	if msg.UserName != "floyd" || msg.UserID != "botid" {
		t.Fatalf("expected message from bot identity, got %s/%s", msg.UserName, msg.UserID)
	}
	if msg.ChannelName != "general" || msg.ChannelID != "#general" {
		t.Fatalf("expected channel general/#general, got %q/%q", msg.ChannelName, msg.ChannelID)
	}
	if msg.MessageText != "@alice has joined #general" {
		t.Fatalf("unexpected incoming message text: %q", msg.MessageText)
	}
	if !msg.SelfMessage {
		t.Fatalf("join announcement must be marked SelfMessage when HearSelf is enabled")
	}
	if msg.BotMessage {
		t.Fatalf("join announcement should not force BotMessage command semantics")
	}
}

func TestAnnounceJoinDoesNotForwardIncomingWhenHearSelfDisabled(t *testing.T) {
	h := &testHandler{}
	sc := &sshConnector{
		handler: h,
		cfg: sshConfig{
			DefaultChannel: "general",
			HearSelf:       false,
		},
		botName: "floyd",
		botID:   "botid",
		buffer:  make([]bufferMsg, 8),
		clients: map[*sshClient]struct{}{},
		threads: make(map[string]int),
		waiters: make(map[chan struct{}]struct{}),
	}
	client := &sshClient{
		userName: "alice",
		userID:   "aliceid",
		channel:  "general",
	}

	sc.announceJoin(client)

	if len(h.msgs) != 0 {
		t.Fatalf("expected no incoming self-forward when HearSelf is disabled, got %d", len(h.msgs))
	}
}

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

func TestFormatHiddenCommand(t *testing.T) {
	sc := &sshConnector{botName: "Floyd"}
	if got := sc.FormatHiddenCommand("help ping"); got != "/floyd help ping" {
		t.Fatalf("FormatHiddenCommand() = %q", got)
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

func TestConfigureUsersRejectsUppercase(t *testing.T) {
	h := &testHandler{}
	sc := &sshConnector{handler: h}
	sc.configureUsers([]userKeysEntry{
		{
			UserName:   "Alice",
			PublicKeys: []string{"ssh-ed25519 AAAA"},
		},
		{
			UserName:   "bob",
			PublicKeys: []string{"ssh-ed25519 BBBB"},
		},
	})

	if _, ok := sc.userNames["Alice"]; ok {
		t.Fatalf("expected uppercase username to be rejected")
	}
	if _, ok := sc.userNames["bob"]; !ok {
		t.Fatalf("expected lowercase username to be accepted")
	}
}

func TestConfigureUsersSupportsMultipleKeysPerUser(t *testing.T) {
	h := &testHandler{}
	sc := &sshConnector{handler: h}
	sc.configureUsers([]userKeysEntry{
		{
			UserName: "parsley",
			PublicKeys: []string{
				"ssh-ed25519 AAAA1111",
				"ssh-ed25519 BBBB2222",
			},
		},
	})

	if len(sc.userKeys) != 2 {
		t.Fatalf("expected 2 configured public keys, got %d", len(sc.userKeys))
	}
	if info, ok := sc.userNames["parsley"]; !ok || info.userName != "parsley" {
		t.Fatalf("expected parsley in userNames map, got %+v (ok=%t)", info, ok)
	}
	if _, ok := sc.userIDs["ssh-ed25519 AAAA1111"]; !ok {
		t.Fatalf("expected first key in userIDs map")
	}
	if _, ok := sc.userIDs["ssh-ed25519 BBBB2222"]; !ok {
		t.Fatalf("expected second key in userIDs map")
	}
}

func TestReloadAtomicallySwapsConfiguredUserKeys(t *testing.T) {
	h := &testHandler{
		protocolConfig: &sshConfig{UserKeys: []userKeysEntry{
			{
				UserName:   "bob",
				PublicKeys: []string{"ssh-ed25519 BBBB2222"},
			},
		}},
	}
	sc := &sshConnector{handler: h}
	sc.configureUsers([]userKeysEntry{
		{
			UserName:   "alice",
			PublicKeys: []string{"ssh-ed25519 AAAA1111"},
		},
	})

	if err := sc.Reload(); err != nil {
		t.Fatalf("Reload() error = %v", err)
	}

	sc.mu.RLock()
	defer sc.mu.RUnlock()
	if _, ok := sc.userNames["alice"]; ok {
		t.Fatalf("expected old user alice to be removed")
	}
	if info, ok := sc.userNames["bob"]; !ok || info.userID != "ssh-ed25519 BBBB2222" {
		t.Fatalf("expected bob to be configured after reload, got %+v (ok=%t)", info, ok)
	}
	if _, ok := sc.userKeys["ssh-ed25519 AAAA1111"]; ok {
		t.Fatalf("expected old key to be removed")
	}
	if info, ok := sc.userKeys["ssh-ed25519 BBBB2222"]; !ok || info.userName != "bob" {
		t.Fatalf("expected new key to be configured after reload, got %+v (ok=%t)", info, ok)
	}
	if len(sc.cfg.UserKeys) != 1 || sc.cfg.UserKeys[0].UserName != "bob" {
		t.Fatalf("expected cfg.UserKeys to be replaced, got %#v", sc.cfg.UserKeys)
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

func TestListenAllSkipsInUsePorts(t *testing.T) {
	basePort := 41000
	withStubSSHNetListen(t, func(network, address string) (net.Listener, error) {
		port := portFromListenAddress(t, address)
		if port >= basePort && port < basePort+3 {
			return nil, syscall.EADDRINUSE
		}
		if network != "tcp" {
			t.Fatalf("unexpected network %q", network)
		}
		return &stubListener{addr: stubAddr(address)}, nil
	})
	h := &testHandler{}
	sc := &sshConnector{
		handler: h,
		cfg: sshConfig{
			ListenHost: "127.0.0.1",
			ListenPort: basePort,
		},
	}

	listeners, host, port := sc.listenAll()
	if len(listeners) != 1 {
		t.Fatalf("expected 1 listener, got %d", len(listeners))
	}
	defer closeListeners(listeners)

	if host != "127.0.0.1" {
		t.Fatalf("expected listen host 127.0.0.1, got %q", host)
	}
	if want := basePort + 3; port != want {
		t.Fatalf("expected selected port %d, got %d", want, port)
	}
	if !containsLog(h.logs, "skipped 3 in-use ports") {
		t.Fatalf("expected skip log, got %v", h.logs)
	}
}

func TestListenAllFailsAfterSevenIncrements(t *testing.T) {
	basePort := 42000
	withStubSSHNetListen(t, func(_ string, address string) (net.Listener, error) {
		port := portFromListenAddress(t, address)
		if port >= basePort && port <= basePort+maxListenPortSkips {
			return nil, syscall.EADDRINUSE
		}
		return &stubListener{addr: stubAddr(address)}, nil
	})
	h := &testHandler{}
	sc := &sshConnector{
		handler: h,
		cfg: sshConfig{
			ListenHost: "127.0.0.1",
			ListenPort: basePort,
		},
	}

	listeners, host, port := sc.listenAll()
	closeListeners(listeners)
	if len(listeners) != 0 {
		t.Fatalf("expected no listeners, got %d", len(listeners))
	}
	if host != "" || port != 0 {
		t.Fatalf("expected empty bind result on failure, got host=%q port=%d", host, port)
	}
	if !containsLog(h.logs, fmt.Sprintf("ports %d-%d", basePort, basePort+maxListenPortSkips)) {
		t.Fatalf("expected exhausted-range log, got %v", h.logs)
	}
}

func TestWriteConnectFileUsesActualPort(t *testing.T) {
	h := &testHandler{}
	sc := &sshConnector{handler: h}
	dir := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir temp dir: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldwd); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	}()

	sc.writeConnectFile("127.0.0.1", 4224, "ssh-ed25519 AAAATEST")

	data, err := os.ReadFile(filepath.Join(dir, ".ssh-connect"))
	if err != nil {
		t.Fatalf("ReadFile(.ssh-connect): %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "BOT_SSH_PORT=127.0.0.1:4224") {
		t.Fatalf("expected actual port in metadata file, got %q", content)
	}
}

func containsLog(logs []string, want string) bool {
	for _, log := range logs {
		if strings.Contains(log, want) {
			return true
		}
	}
	return false
}
