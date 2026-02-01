package ssh

import (
	"bufio"
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/lnxjedi/gopherbot/robot"
	botwrap "github.com/lnxjedi/gopherbot/v2/bot"
)

type filterMode int

const (
	filterAll filterMode = iota
	filterChannel
	filterThread
)

const (
	defaultListenHost = "localhost"
	defaultListenPort = 4221
	defaultReplaySize = 42
	defaultMaxMsg     = 16384
	defaultChannel    = "general"
	maxBufferBytes    = 4096
)

const (
	pasteStart = "\x1b[200~"
	pasteEnd   = "\x1b[201~"
	pasteOn    = "\x1b[?2004h"
	pasteOff   = "\x1b[?2004l"
)

type userKeyInfo struct {
	userName string
	userID   string
}

type bufferMsg struct {
	timestamp time.Time
	userName  string
	isBot     bool
	channel   string
	threadID  string
	threaded  bool
	text      string
}

type sshConfig struct {
	ListenHost       string
	ListenPort       int
	HostKey          string
	ReplayBufferSize int
	MaxMsgBytes      int
	DefaultChannel   string
	BotName          string
	Channels         []string
	WrapWidth        int
	UserHistoryLines int
}

type sshConnector struct {
	cfg     sshConfig
	handler robot.Handler
	logger  *log.Logger

	botName string
	botID   string

	mu        sync.RWMutex
	clients   map[*sshClient]struct{}
	userKeys  map[string]userKeyInfo
	threads   map[string]int
	buffer    []bufferMsg
	bufIndex  int
	bufFilled bool
	histories map[string]*userHistory
}

type userHistory struct {
	lines []string
}

type sshClient struct {
	userName string
	userID   string

	channel        string
	threadID       string
	typingInThread bool
	filter         filterMode
	lastThread     map[string]string
	echo           bool
	inputBuf       bytes.Buffer
	bufMu          sync.Mutex
	width          int
	history        *userHistory
	histPos        int
	histSaved      string
	cursor         int

	ch     ssh.Channel
	conn   *ssh.ServerConn
	writer io.Writer
	wmu    sync.Mutex
}

// Initialize sets up the SSH connector and returns a connector object.
func Initialize(handler robot.Handler, l *log.Logger) robot.Connector {
	var cfg sshConfig
	if err := handler.GetProtocolConfig(&cfg); err != nil {
		handler.Log(robot.Fatal, "Unable to retrieve protocol configuration: %v", err)
	}
	if portEnv := os.Getenv("GOPHER_SSH_PORT"); portEnv != "" {
		if p, err := strconv.Atoi(portEnv); err == nil && p > 0 {
			cfg.ListenPort = p
		}
	}
	if cfg.ListenHost == "" {
		cfg.ListenHost = defaultListenHost
	}
	if cfg.ListenPort == 0 {
		cfg.ListenPort = defaultListenPort
	}
	if cfg.ReplayBufferSize == 0 {
		cfg.ReplayBufferSize = defaultReplaySize
	}
	if cfg.MaxMsgBytes == 0 {
		cfg.MaxMsgBytes = defaultMaxMsg
	}
	if cfg.DefaultChannel == "" {
		cfg.DefaultChannel = defaultChannel
	}
	if cfg.BotName == "" {
		cfg.BotName = "gopherbot"
	}
	if cfg.UserHistoryLines == 0 {
		cfg.UserHistoryLines = 14
	}

	sc := &sshConnector{
		cfg:       cfg,
		handler:   handler,
		logger:    l,
		botName:   cfg.BotName,
		clients:   make(map[*sshClient]struct{}),
		userKeys:  make(map[string]userKeyInfo),
		threads:   make(map[string]int),
		buffer:    make([]bufferMsg, cfg.ReplayBufferSize),
		histories: make(map[string]*userHistory),
	}

	return robot.Connector(sc)
}

func (sc *sshConnector) Run(stop <-chan struct{}) {
	signer, pubLine := sc.loadHostKey()
	sc.botID = pubLine
	sc.handler.SetBotID(pubLine)

	serverConfig := &ssh.ServerConfig{
		PublicKeyCallback: sc.authorizeKey,
	}
	serverConfig.AddHostKey(signer)

	listeners, listenHost := sc.listenAll()
	if len(listeners) == 0 {
		sc.handler.Log(robot.Fatal, "SSH connector failed to bind")
		return
	}
	sc.handler.Log(robot.Info, "SSH connector listening on %s:%d", listenHost, sc.cfg.ListenPort)
	defer func() {
		for _, ln := range listeners {
			_ = ln.Close()
		}
	}()

	sc.writeConnectFile(listenHost, sc.cfg.ListenPort, pubLine)

	stopOnce := sync.Once{}
	stopFn := func() {
		stopOnce.Do(func() {
			for _, ln := range listeners {
				_ = ln.Close()
			}
			sc.closeAllClients()
		})
	}

	go func() {
		<-stop
		sc.handler.Log(robot.Info, "Received stop in SSH connector")
		stopFn()
	}()

	var wg sync.WaitGroup
	for _, ln := range listeners {
		ln := ln
		wg.Add(1)
		go func() {
			defer wg.Done()
			sc.acceptLoop(ln, serverConfig, stop)
		}()
	}
	wg.Wait()
}

func (sc *sshConnector) acceptLoop(ln net.Listener, cfg *ssh.ServerConfig, stop <-chan struct{}) {
	for {
		nc, err := ln.Accept()
		if err != nil {
			select {
			case <-stop:
				return
			default:
			}
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				continue
			}
			return
		}
		go sc.handleConn(nc, cfg)
	}
}

func (sc *sshConnector) handleConn(nc net.Conn, cfg *ssh.ServerConfig) {
	sshConn, chans, reqs, err := ssh.NewServerConn(nc, cfg)
	if err != nil {
		return
	}
	go ssh.DiscardRequests(reqs)

	userName := sshConn.Permissions.Extensions["user"]
	userID := sshConn.Permissions.Extensions["userid"]
	if userName == "" || userID == "" {
		_ = sshConn.Close()
		return
	}

	client := &sshClient{
		userName:   userName,
		userID:     userID,
		channel:    sc.cfg.DefaultChannel,
		filter:     filterChannel,
		lastThread: make(map[string]string),
		histPos:    -1,
		conn:       sshConn,
		writer:     nc,
	}
	client.history = sc.getHistory(client.userID)

	sc.addClient(client)
	defer sc.removeClient(client)

	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			_ = newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}
		ch, requests, err := newChannel.Accept()
		if err != nil {
			continue
		}
		client.ch = ch
		client.writer = ch
		go sc.handleSession(client, ch, requests)
	}
}

func (sc *sshConnector) handleSession(client *sshClient, ch ssh.Channel, requests <-chan *ssh.Request) {
	defer func() {
		client.writeString(pasteOff)
		_ = ch.Close()
	}()

	ptyCh := make(chan struct{}, 1)
	shellReady := make(chan struct{})

	go func() {
		for req := range requests {
			switch req.Type {
			case "pty-req":
				client.echo = true
				if w := parsePtyWidth(req.Payload); w > 0 {
					if w > 0 {
						client.setWidth(w)
					}
				}
				select {
				case ptyCh <- struct{}{}:
				default:
				}
				req.Reply(true, nil)
			case "window-change":
				if w := parseWindowWidth(req.Payload); w > 0 {
					if w > 0 {
						client.setWidth(w)
					}
				}
				req.Reply(true, nil)
			case "shell":
				req.Reply(true, nil)
				close(shellReady)
			default:
				req.Reply(false, nil)
			}
		}
	}()

	<-shellReady
	select {
	case <-ptyCh:
		client.writeString(pasteOn)
	default:
	}

	client.writeString("Select filter: (A)ll, (C)hannel, (T)hread [C]: ")
	inputCh := make(chan string)
	go sc.readInput(client, inputCh)

	input, ok := <-inputCh
	if !ok {
		return
	}
	client.filter = parseFilter(input)
	client.writeString("\n")

	sc.replayBuffer(client)
	client.writeString(sshConnectorHelpLine)
	client.writePrompt()

	for line := range inputCh {
		if len(line) == 0 {
			client.writeString(sshConnectorHelpLine)
			client.writePrompt()
			continue
		}
		if strings.HasPrefix(line, "|") {
			sc.handleCommand(client, strings.TrimSpace(line))
			client.writePrompt()
			continue
		}
		sc.handleUserInput(client, line)
		client.writePrompt()
	}
}

func (sc *sshConnector) handleCommand(client *sshClient, input string) {
	if len(input) < 2 {
		return
	}
	switch input[1] {
	case 'C', 'c':
		arg := strings.TrimSpace(input[2:])
		if arg == "?" {
			client.writeString("Available channels:\n")
			for _, ch := range sc.cfg.Channels {
				client.writeString(fmt.Sprintf("'%s'; type: '|c%s'\n", ch, ch))
			}
			return
		}
		if arg == "" {
			client.writeString("Invalid 0-length channel\n")
			return
		}
		if !sc.isValidChannel(arg) {
			client.writeString("Invalid channel\n")
			return
		}
		client.channel = arg
		client.typingInThread = false
		client.writeString(fmt.Sprintf("Changed current channel to: %s\n", arg))
	case 'T', 't':
		arg := strings.TrimSpace(input[2:])
		if arg == "?" {
			client.writeString("Use '|t' to toggle typing in a thread, '|t<string>' to set the current thread ID, or '|j' to join the last thread seen\n")
			return
		}
		if arg == "" {
			client.typingInThread = !client.typingInThread
			if client.typingInThread {
				client.threadID = sc.nextThreadID(client.channel)
			}
		} else {
			client.typingInThread = true
			client.threadID = arg
		}
		if client.typingInThread {
			client.writeString(fmt.Sprintf("(now typing in thread: %s)\n", client.threadID))
		} else {
			client.writeString("(typing in channel now)\n")
		}
	case 'J', 'j':
		last, ok := client.lastThread[client.channel]
		if !ok || last == "" {
			client.writeString("(sorry, I don't see a thread to join)\n")
			return
		}
		client.typingInThread = true
		client.threadID = last
		client.writeString(fmt.Sprintf("(now typing in thread: %s)\n", client.threadID))
	case 'F', 'f':
		arg := strings.TrimSpace(input[2:])
		if arg == "" || arg == "?" {
			client.writeString("Available filters:\n")
			client.writeString("'All' - type: '|fA'\n")
			client.writeString("'Channel' - type: '|fC'\n")
			client.writeString("'Thread' - type: '|fT'\n")
			return
		}
		client.filter = parseFilter(arg)
		client.writeString(fmt.Sprintf("Filter set to: %s\n", filterLabel(client.filter)))
	default:
		client.writeString("Invalid SSH connector command\n")
	}
}

func (sc *sshConnector) handleUserInput(client *sshClient, line string) {
	now := time.Now()
	client.writeString(fmt.Sprintf("(%s)\n", now.Format("15:04:05")))

	if len(line) > sc.cfg.MaxMsgBytes {
		client.writeString("(ERROR: message too long; > 16k - dropped)\n")
		return
	}

	if len(line) > maxBufferBytes {
		client.writeString("(WARNING: message truncated to 4k in buffer)\n")
	}

	client.recordHistory(line, sc.cfg.UserHistoryLines)

	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "/") {
		if strings.HasPrefix(trimmed, "/ ") || trimmed == "/" {
			client.writeString("(INFO: '/' note to self message not sent to other users)\n")
			return
		}
		if sc.isBotHiddenMessage(trimmed) {
			payload := strings.TrimSpace(trimmed[1+len(sc.botName):])
			sc.sendIncoming(client, payload, true)
			return
		}
		payload := strings.TrimSpace(trimmed[1:])
		if payload != "" {
			sc.sendIncoming(client, payload, true)
		}
		return
	}

	sc.sendIncoming(client, line, false)
	sc.broadcastUserMessage(client, line, now)
}

func (sc *sshConnector) sendIncoming(client *sshClient, line string, hidden bool) {
	threaded := client.typingInThread
	threadID := ""
	messageID := ""
	if threaded {
		messageID = sc.nextThreadID(client.channel)
		threadID = client.threadID
	} else {
		threadID = sc.nextThreadID(client.channel)
		messageID = threadID
	}

	msg := &robot.ConnectorMessage{
		Protocol:        "ssh",
		UserName:        client.userName,
		UserID:          client.userID,
		ChannelName:     client.channel,
		ChannelID:       "#" + client.channel,
		MessageID:       messageID,
		ThreadedMessage: threaded,
		ThreadID:        threadID,
		MessageText:     line,
		HiddenMessage:   hidden,
		DirectMessage:   false,
	}
	sc.handler.IncomingMessage(msg)
}

func (sc *sshConnector) MessageHeard(u, c string) {}

func (sc *sshConnector) SetUserMap(m map[string]string) {
	keys := make(map[string]userKeyInfo)
	for name, id := range m {
		norm := normalizeKeyLine(id)
		if norm == "" {
			continue
		}
		keys[norm] = userKeyInfo{userName: name, userID: id}
	}
	sc.mu.Lock()
	sc.userKeys = keys
	sc.mu.Unlock()
}

func (sc *sshConnector) GetProtocolUserAttribute(u, attr string) (value string, ret robot.RetVal) {
	return "", robot.AttributeNotFound
}

func (sc *sshConnector) FormatHelp(input string) string {
	arr := strings.SplitN(input, " - ", 2)
	if len(arr) != 2 {
		return "*" + input + "*"
	}
	return "*" + arr[0] + "* - " + arr[1]
}

func (sc *sshConnector) DefaultHelp() []string {
	return []string{}
}

func (sc *sshConnector) JoinChannel(c string) (ret robot.RetVal) {
	return robot.Ok
}

func (sc *sshConnector) SendProtocolChannelThreadMessage(ch, thr, msg string, f robot.MessageFormat, msgObject *robot.ConnectorMessage) (ret robot.RetVal) {
	ch = sc.normalizeChannel(ch)
	threaded := len(thr) > 0
	evt := bufferMsg{
		timestamp: time.Now(),
		userName:  sc.botName,
		isBot:     true,
		channel:   ch,
		threadID:  thr,
		threaded:  threaded,
		text:      msg,
	}
	sc.broadcast(evt, msgObject)
	return robot.Ok
}

func (sc *sshConnector) SendProtocolUserChannelThreadMessage(uid, uname, ch, thr, msg string, f robot.MessageFormat, msgObject *robot.ConnectorMessage) (ret robot.RetVal) {
	ch = sc.normalizeChannel(ch)
	formatted := "@" + uname + " " + msg
	threaded := len(thr) > 0
	evt := bufferMsg{
		timestamp: time.Now(),
		userName:  sc.botName,
		isBot:     true,
		channel:   ch,
		threadID:  thr,
		threaded:  threaded,
		text:      formatted,
	}
	sc.broadcast(evt, msgObject)
	return robot.Ok
}

func (sc *sshConnector) SendProtocolUserMessage(u string, msg string, f robot.MessageFormat, msgObject *robot.ConnectorMessage) (ret robot.RetVal) {
	u = sc.normalizeUser(u)
	clients := sc.clientsForUser(u)
	if len(clients) == 0 {
		return robot.UserNotFound
	}

	evt := bufferMsg{
		timestamp: time.Now(),
		userName:  sc.botName,
		isBot:     true,
		channel:   "(direct)",
		threadID:  "",
		threaded:  false,
		text:      msg,
	}
	for _, client := range clients {
		client.writeLine(sc.formatMessage(evt, false))
	}
	return robot.Ok
}

func (sc *sshConnector) broadcastUserMessage(client *sshClient, line string, ts time.Time) {
	threaded := client.typingInThread
	threadID := client.threadID
	evt := bufferMsg{
		timestamp: ts,
		userName:  client.userName,
		isBot:     false,
		channel:   client.channel,
		threadID:  threadID,
		threaded:  threaded,
		text:      line,
	}
	sc.broadcast(evt, &robot.ConnectorMessage{UserID: client.userID, HiddenMessage: false})
}

func (sc *sshConnector) broadcast(evt bufferMsg, msgObject *robot.ConnectorMessage) {
	private := msgObject != nil && msgObject.HiddenMessage
	var hiddenUser string
	if msgObject != nil && msgObject.HiddenMessage {
		hiddenUser = msgObject.UserID
	}

	if !private {
		sc.appendBuffer(evt)
	}

	sc.mu.RLock()
	clients := make([]*sshClient, 0, len(sc.clients))
	for client := range sc.clients {
		clients = append(clients, client)
	}
	sc.mu.RUnlock()

	for _, client := range clients {
		if private {
			if client.userID != hiddenUser {
				continue
			}
			client.writeAsyncMessage(sc.formatMessage(evt, true))
			continue
		}
		if client.userName == evt.userName {
			continue
		}
		if !client.matchFilter(evt) {
			continue
		}
		client.writeAsyncMessage(sc.formatMessage(evt, false))
		if evt.threaded {
			client.lastThread[evt.channel] = evt.threadID
		}
	}
}

func (sc *sshConnector) formatMessage(evt bufferMsg, private bool) string {
	stamp := evt.timestamp.Format("15:04:05")
	prefix := "@" + evt.userName
	if evt.isBot {
		prefix = "=@" + evt.userName
	}
	thread := ""
	if evt.threaded && evt.threadID != "" {
		thread = fmt.Sprintf("(%s)", evt.threadID)
	}
	channel := "#" + evt.channel
	if evt.channel == "" {
		channel = "#(direct)"
	}
	line := fmt.Sprintf("(%s)%s/%s%s: %s", stamp, prefix, channel, thread, evt.text)
	if private {
		line = fmt.Sprintf("(private/%s)%s/%s%s: %s", stamp, prefix, channel, thread, evt.text)
	}
	return sc.wrap(line)
}

func (sc *sshConnector) appendBuffer(evt bufferMsg) {
	msg := evt.text
	if len(msg) > maxBufferBytes {
		msg = msg[:maxBufferBytes]
	}
	evt.text = msg

	sc.mu.Lock()
	sc.buffer[sc.bufIndex] = evt
	sc.bufIndex = (sc.bufIndex + 1) % len(sc.buffer)
	if sc.bufIndex == 0 {
		sc.bufFilled = true
	}
	sc.mu.Unlock()
}

func (sc *sshConnector) replayBuffer(client *sshClient) {
	msgs := sc.snapshotBuffer()
	for _, evt := range msgs {
		if !client.matchFilter(evt) {
			continue
		}
		client.writeLine(sc.formatMessage(evt, false))
		if evt.threaded {
			client.lastThread[evt.channel] = evt.threadID
		}
	}
}

func (sc *sshConnector) snapshotBuffer() []bufferMsg {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	if len(sc.buffer) == 0 {
		return nil
	}
	var out []bufferMsg
	if sc.bufFilled {
		out = append(out, sc.buffer[sc.bufIndex:]...)
	}
	out = append(out, sc.buffer[:sc.bufIndex]...)
	return out
}

func (sc *sshConnector) clientsForUser(u string) []*sshClient {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	clients := make([]*sshClient, 0, len(sc.clients))
	for c := range sc.clients {
		if c.userID == u || c.userName == u {
			clients = append(clients, c)
		}
	}
	return clients
}

func (sc *sshConnector) getHistory(userID string) *userHistory {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	hist, ok := sc.histories[userID]
	if ok {
		return hist
	}
	hist = &userHistory{}
	sc.histories[userID] = hist
	return hist
}

func (sc *sshConnector) normalizeChannel(ch string) string {
	if ch == "" {
		return ch
	}
	if id, ok := sc.handler.ExtractID(ch); ok {
		ch = id
	}
	if strings.HasPrefix(ch, "#") {
		return strings.TrimPrefix(ch, "#")
	}
	return ch
}

func (sc *sshConnector) normalizeUser(u string) string {
	if u == "" {
		return u
	}
	if id, ok := sc.handler.ExtractID(u); ok {
		return id
	}
	return u
}

func (sc *sshConnector) addClient(c *sshClient) {
	sc.mu.Lock()
	sc.clients[c] = struct{}{}
	sc.mu.Unlock()
}

func (sc *sshConnector) removeClient(c *sshClient) {
	sc.mu.Lock()
	delete(sc.clients, c)
	sc.mu.Unlock()
}

func (sc *sshConnector) closeAllClients() {
	sc.mu.RLock()
	clients := make([]*sshClient, 0, len(sc.clients))
	for c := range sc.clients {
		clients = append(clients, c)
	}
	sc.mu.RUnlock()
	for _, c := range clients {
		_ = c.conn.Close()
	}
}

func (sc *sshConnector) authorizeKey(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
	normalized := normalizeKeyLine(string(ssh.MarshalAuthorizedKey(key)))
	sc.mu.RLock()
	info, ok := sc.userKeys[normalized]
	sc.mu.RUnlock()
	if !ok {
		return nil, errors.New("unknown public key")
	}
	perms := &ssh.Permissions{Extensions: map[string]string{
		"user":   info.userName,
		"userid": info.userID,
	}}
	return perms, nil
}

func normalizeKeyLine(line string) string {
	line = strings.TrimSpace(line)
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return ""
	}
	return parts[0] + " " + parts[1]
}

func (sc *sshConnector) listenAll() ([]net.Listener, string) {
	addr := sc.cfg.ListenHost
	port := sc.cfg.ListenPort

	if port == 0 {
		port = defaultListenPort
	}

	var listeners []net.Listener
	var listenHost string
	if addr == "all" {
		ln4, err4 := net.Listen("tcp4", fmt.Sprintf("0.0.0.0:%d", port))
		if err4 == nil {
			listeners = append(listeners, ln4)
			listenHost = "0.0.0.0"
		}
		ln6, err6 := net.Listen("tcp6", fmt.Sprintf("[::]:%d", port))
		if err6 == nil {
			listeners = append(listeners, ln6)
			if listenHost == "" {
				listenHost = "[::]"
			}
		}
		if len(listeners) == 0 {
			sc.handler.Log(robot.Fatal, "Unable to bind to 0.0.0.0 or [::]: %v / %v", err4, err6)
		}
		return listeners, listenHost
	}

	if addr == "localhost" {
		addr = "127.0.0.1"
	}
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", addr, port))
	if err != nil {
		sc.handler.Log(robot.Fatal, "Unable to bind SSH connector on %s:%d: %v", addr, port, err)
		return nil, addr
	}
	return []net.Listener{ln}, addr
}

func (sc *sshConnector) writeConnectFile(host string, port int, pubKey string) {
	if host == "" {
		host = "127.0.0.1"
	}
	path := ".ssh-connect"
	content := fmt.Sprintf("BOT_SSH_PORT=%s:%d\nBOT_SERVER_PUBKEY='%s'\n", host, port, pubKey)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		sc.handler.Log(robot.Error, "Writing .ssh-connect: %v", err)
	}
}

func (sc *sshConnector) loadHostKey() (ssh.Signer, string) {
	if sc.cfg.HostKey != "" {
		signer, err := ssh.ParsePrivateKey([]byte(sc.cfg.HostKey))
		if err != nil {
			sc.handler.Log(robot.Fatal, "Parsing configured SSH host key: %v", err)
		}
		pub := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(signer.PublicKey())))
		return signer, pub
	}

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		sc.handler.Log(robot.Fatal, "Generating SSH host key: %v", err)
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		sc.handler.Log(robot.Fatal, "Creating SSH host key signer: %v", err)
	}
	pub := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(signer.PublicKey())))
	sc.handler.Log(robot.Info, "Generated ephemeral SSH host key: %s", pub)
	return signer, pub
}

func (sc *sshConnector) isValidChannel(ch string) bool {
	if len(sc.cfg.Channels) == 0 {
		return true
	}
	for _, c := range sc.cfg.Channels {
		if ch == c {
			return true
		}
	}
	return false
}

func (sc *sshConnector) nextThreadID(channel string) string {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.threads[channel]++
	return fmt.Sprintf("%04x", sc.threads[channel]%65536)
}

func (sc *sshConnector) isBotHiddenMessage(line string) bool {
	if !strings.HasPrefix(line, "/") {
		return false
	}
	trimmed := strings.TrimPrefix(line, "/")
	if !strings.HasPrefix(trimmed, sc.botName) {
		return false
	}
	if len(trimmed) == len(sc.botName) {
		return false
	}
	if trimmed[len(sc.botName)] != ' ' {
		return false
	}
	remainder := strings.TrimSpace(trimmed[len(sc.botName):])
	return remainder != ""
}

func parseFilter(input string) filterMode {
	input = strings.TrimSpace(input)
	if input == "" {
		return filterChannel
	}
	switch strings.ToUpper(input[:1]) {
	case "A":
		return filterAll
	case "C":
		return filterChannel
	case "T":
		return filterThread
	default:
		return filterChannel
	}
}

func filterLabel(f filterMode) string {
	switch f {
	case filterAll:
		return "All"
	case filterChannel:
		return "Channel"
	case filterThread:
		return "Thread"
	default:
		return "Thread"
	}
}

const sshConnectorHelpLine = "SSH connector: Type '|c?' to list channels, '|t?' for thread help, '|j' to join last thread, '|f?' to list/change filter\n"

func (c *sshClient) writePrompt() {
	thread := ""
	if c.typingInThread && c.threadID != "" {
		thread = fmt.Sprintf("(%s)", c.threadID)
	}
	prompt := fmt.Sprintf("@%s/#%s%s -> ", c.userName, c.channel, thread)
	c.writeString(prompt)
}

func (c *sshClient) writeString(s string) {
	c.wmu.Lock()
	defer c.wmu.Unlock()
	_, _ = c.writer.Write([]byte(normalizeNewlines(s)))
}

func (c *sshClient) writeLine(s string) {
	c.wmu.Lock()
	defer c.wmu.Unlock()
	_, _ = c.writer.Write([]byte(normalizeNewlines(c.wrapLine(s) + "\n")))
}

func (c *sshClient) writeAsyncMessage(msg string) {
	c.wmu.Lock()
	defer c.wmu.Unlock()
	// Clear current input line, print message, then redraw prompt + buffer.
	_, _ = c.writer.Write([]byte("\r\033[2K"))
	_, _ = c.writer.Write([]byte(normalizeNewlines(c.wrapLine(msg) + "\n")))
	c.redrawInputLineLocked()
}

func (c *sshClient) redrawInputLine() {
	c.wmu.Lock()
	defer c.wmu.Unlock()
	c.redrawInputLineLocked()
}

func (c *sshClient) redrawInputLineLocked() {
	_, _ = c.writer.Write([]byte("\r\033[2K"))
	thread := ""
	if c.typingInThread && c.threadID != "" {
		thread = fmt.Sprintf("(%s)", c.threadID)
	}
	prompt := fmt.Sprintf("@%s/#%s%s -> ", c.userName, c.channel, thread)
	_, _ = c.writer.Write([]byte(prompt))
	c.bufMu.Lock()
	buf := append([]byte(nil), c.inputBuf.Bytes()...)
	cursor := c.cursor
	c.bufMu.Unlock()
	if len(buf) > 0 {
		_, _ = c.writer.Write(buf)
	}
	if cursor < 0 {
		cursor = 0
	}
	if cursor > len(buf) {
		cursor = len(buf)
	}
	// Move cursor left if needed.
	if len(buf)-cursor > 0 {
		_, _ = c.writer.Write([]byte(fmt.Sprintf("\033[%dD", len(buf)-cursor)))
	}
}

func (c *sshClient) matchFilter(evt bufferMsg) bool {
	if c.filter == filterAll {
		return true
	}
	if c.filter == filterChannel {
		return evt.channel == c.channel
	}
	if c.filter == filterThread {
		if c.typingInThread {
			return evt.channel == c.channel && evt.threaded && evt.threadID == c.threadID
		}
		return evt.channel == c.channel && !evt.threaded
	}
	return false
}

func (sc *sshConnector) readInput(client *sshClient, out chan<- string) {
	defer close(out)
	reader := bufio.NewReader(client.ch)
	var pending bytes.Buffer
	inPaste := false
	lastWasCR := false

	appendBuf := func(b byte) {
		client.bufMu.Lock()
		defer client.bufMu.Unlock()
		data := client.inputBuf.Bytes()
		if client.cursor < 0 {
			client.cursor = 0
		}
		if client.cursor > len(data) {
			client.cursor = len(data)
		}
		tmp := make([]byte, 0, len(data)+1)
		tmp = append(tmp, data[:client.cursor]...)
		tmp = append(tmp, b)
		tmp = append(tmp, data[client.cursor:]...)
		client.inputBuf.Reset()
		_, _ = client.inputBuf.Write(tmp)
		client.cursor++
	}
	backspaceBuf := func() {
		client.bufMu.Lock()
		defer client.bufMu.Unlock()
		if client.inputBuf.Len() == 0 || client.cursor == 0 {
			return
		}
		data := client.inputBuf.Bytes()
		if client.cursor > len(data) {
			client.cursor = len(data)
		}
		tmp := make([]byte, 0, len(data)-1)
		tmp = append(tmp, data[:client.cursor-1]...)
		tmp = append(tmp, data[client.cursor:]...)
		client.inputBuf.Reset()
		_, _ = client.inputBuf.Write(tmp)
		client.cursor--
	}
	clearBuf := func() {
		client.bufMu.Lock()
		client.inputBuf.Reset()
		client.histPos = -1
		client.histSaved = ""
		client.cursor = 0
		client.bufMu.Unlock()
	}
	deleteAtCursor := func() {
		client.bufMu.Lock()
		defer client.bufMu.Unlock()
		data := client.inputBuf.Bytes()
		if client.cursor < 0 {
			client.cursor = 0
		}
		if client.cursor >= len(data) {
			return
		}
		tmp := make([]byte, 0, len(data)-1)
		tmp = append(tmp, data[:client.cursor]...)
		tmp = append(tmp, data[client.cursor+1:]...)
		client.inputBuf.Reset()
		_, _ = client.inputBuf.Write(tmp)
	}
	moveLeft := func() {
		client.bufMu.Lock()
		if client.cursor > 0 {
			client.cursor--
		}
		client.bufMu.Unlock()
	}
	moveRight := func() {
		client.bufMu.Lock()
		if client.cursor < client.inputBuf.Len() {
			client.cursor++
		}
		client.bufMu.Unlock()
	}
	moveHome := func() {
		client.bufMu.Lock()
		client.cursor = 0
		client.bufMu.Unlock()
	}
	moveEnd := func() {
		client.bufMu.Lock()
		client.cursor = client.inputBuf.Len()
		client.bufMu.Unlock()
	}

	for {
		b, err := reader.ReadByte()
		if err != nil {
			return
		}

		if pending.Len() > 0 && pending.Bytes()[0] == 0x1b {
			pending.WriteByte(b)
			if strings.HasPrefix(pasteStart, pending.String()) {
				if pending.Len() == len(pasteStart) {
					inPaste = true
					pending.Reset()
				}
				continue
			}
			if pending.Len() == 2 && pending.Bytes()[1] == '[' {
				continue
			}
			if pending.Len() == 3 && pending.Bytes()[1] == '[' {
				switch pending.Bytes()[2] {
				case 'A':
					client.historyUp()
					pending.Reset()
					continue
				case 'B':
					client.historyDown()
					pending.Reset()
					continue
				case 'C':
					moveRight()
					client.redrawInputLine()
					pending.Reset()
					continue
				case 'D':
					moveLeft()
					client.redrawInputLine()
					pending.Reset()
					continue
				case 'H':
					moveHome()
					client.redrawInputLine()
					pending.Reset()
					continue
				case 'F':
					moveEnd()
					client.redrawInputLine()
					pending.Reset()
					continue
				}
			}
			if pending.Len() == 4 && pending.Bytes()[1] == '[' && pending.Bytes()[3] == '~' {
				switch pending.Bytes()[2] {
				case '1':
					moveHome()
					client.redrawInputLine()
					pending.Reset()
					continue
				case '4':
					moveEnd()
					client.redrawInputLine()
					pending.Reset()
					continue
				case '3':
					deleteAtCursor()
					client.redrawInputLine()
					pending.Reset()
					continue
				}
			}
			if pending.Len() == 3 && pending.Bytes()[1] == 'O' {
				switch pending.Bytes()[2] {
				case 'H':
					moveHome()
					client.redrawInputLine()
					pending.Reset()
					continue
				case 'F':
					moveEnd()
					client.redrawInputLine()
					pending.Reset()
					continue
				}
			}
			// Unhandled escape sequence; flush bytes into input buffer.
			for pending.Len() > 0 {
				peek := pending.Bytes()[0]
				pending.ReadByte()
				appendBuf(peek)
				client.redrawInputLine()
			}
			continue
		}

		if inPaste {
			pending.WriteByte(b)
			if strings.HasPrefix(pasteEnd, pending.String()) {
				if pending.Len() == len(pasteEnd) {
					inPaste = false
					pending.Reset()
					client.bufMu.Lock()
					out <- client.inputBuf.String()
					client.bufMu.Unlock()
					clearBuf()
					client.redrawInputLine()
				}
				continue
			}
			appendBuf(pending.Bytes()[0])
			pending.Reset()
			client.redrawInputLine()
			continue
		}

		if b == 0x04 {
			client.bufMu.Lock()
			empty := client.inputBuf.Len() == 0
			client.bufMu.Unlock()
			if empty {
				return
			}
			continue
		}
		if b == 0x01 { // Ctrl-A
			moveHome()
			client.redrawInputLine()
			continue
		}
		if b == 0x05 { // Ctrl-E
			moveEnd()
			client.redrawInputLine()
			continue
		}
		if b == '\r' || b == '\n' {
			if b == '\n' && lastWasCR {
				lastWasCR = false
				continue
			}
			lastWasCR = b == '\r'
			client.bufMu.Lock()
			out <- client.inputBuf.String()
			client.bufMu.Unlock()
			clearBuf()
			continue
		}
		lastWasCR = false

		if b == 0x1b {
			pending.WriteByte(b)
			continue
		}

		pending.WriteByte(b)
		if pending.Bytes()[0] == 0x7f || pending.Bytes()[0] == '\b' {
			backspaceBuf()
			client.redrawInputLine()
			pending.Reset()
			continue
		}
		if strings.HasPrefix(pasteStart, pending.String()) {
			if pending.Len() == len(pasteStart) {
				inPaste = true
				pending.Reset()
			}
			continue
		}
		appendBuf(pending.Bytes()[0])
		client.redrawInputLine()
		pending.Reset()
	}
}

func normalizeNewlines(s string) string {
	return strings.ReplaceAll(s, "\n", "\r\n")
}

func parsePtyWidth(payload []byte) int {
	if len(payload) < 4 {
		return 0
	}
	termLen := int(binary.BigEndian.Uint32(payload[0:4]))
	if termLen < 0 || len(payload) < 4+termLen+4 {
		return 0
	}
	offset := 4 + termLen
	if len(payload) < offset+4 {
		return 0
	}
	return int(binary.BigEndian.Uint32(payload[offset : offset+4]))
}

func parseWindowWidth(payload []byte) int {
	if len(payload) < 4 {
		return 0
	}
	return int(binary.BigEndian.Uint32(payload[0:4]))
}

func (c *sshClient) setWidth(w int) {
	if w <= 0 {
		return
	}
	if w < 20 {
		return
	}
	c.bufMu.Lock()
	c.width = w
	c.bufMu.Unlock()
}

func (c *sshClient) wrapLine(s string) string {
	c.bufMu.Lock()
	width := c.width
	c.bufMu.Unlock()
	if width <= 0 {
		return s
	}
	return botwrap.Wrap(s, width)
}

func (sc *sshConnector) wrap(s string) string {
	return s
}

func (c *sshClient) recordHistory(line string, limit int) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return
	}
	if c.history == nil {
		return
	}
	c.bufMu.Lock()
	c.history.lines = append(c.history.lines, trimmed)
	if limit > 0 && len(c.history.lines) > limit {
		c.history.lines = c.history.lines[len(c.history.lines)-limit:]
	}
	c.histPos = -1
	c.histSaved = ""
	c.bufMu.Unlock()
}

func (c *sshClient) historyUp() {
	if c.history == nil || len(c.history.lines) == 0 {
		return
	}
	c.bufMu.Lock()
	if c.histPos == -1 {
		c.histSaved = c.inputBuf.String()
		c.histPos = len(c.history.lines) - 1
	} else if c.histPos > 0 {
		c.histPos--
	}
	entry := c.history.lines[c.histPos]
	c.inputBuf.Reset()
	_, _ = c.inputBuf.WriteString(entry)
	c.cursor = c.inputBuf.Len()
	c.bufMu.Unlock()
	c.redrawInputLine()
}

func (c *sshClient) historyDown() {
	if c.history == nil || len(c.history.lines) == 0 {
		return
	}
	c.bufMu.Lock()
	if c.histPos == -1 {
		c.bufMu.Unlock()
		return
	}
	if c.histPos < len(c.history.lines)-1 {
		c.histPos++
		entry := c.history.lines[c.histPos]
		c.inputBuf.Reset()
		_, _ = c.inputBuf.WriteString(entry)
		c.cursor = c.inputBuf.Len()
		c.bufMu.Unlock()
		c.redrawInputLine()
		return
	}
	c.histPos = -1
	entry := c.histSaved
	c.inputBuf.Reset()
	_, _ = c.inputBuf.WriteString(entry)
	c.cursor = c.inputBuf.Len()
	c.bufMu.Unlock()
	c.redrawInputLine()
}
