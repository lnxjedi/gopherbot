package ssh

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/chzyer/readline"
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
	ansiReset  = "\x1b[0m"
)

type userKeyInfo struct {
	userName string
	userID   string
}

type userListing struct {
	name  string
	isBot bool
}

type bufferMsg struct {
	timestamp time.Time
	userName  string
	userID    string
	isBot     bool
	channel   string
	threadID  string
	threaded  bool
	text      string
	isDM      bool
	dmPeer    string
	dmPeerID  string
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
	Color            bool
	ColorScheme      map[string]int
}

type sshConnector struct {
	cfg     sshConfig
	handler robot.Handler
	logger  *log.Logger

	botName      string
	botNameLower string
	botID        string

	mu        sync.RWMutex
	clients   map[*sshClient]struct{}
	userKeys  map[string]userKeyInfo
	userNames map[string]userKeyInfo
	userIDs   map[string]userKeyInfo
	threads   map[string]int
	buffer    []bufferMsg
	bufIndex  int
	bufFilled bool
}

type sshClient struct {
	userName string
	userID   string

	channel         string
	threadID        string
	typingInThread  bool
	filter          filterMode
	lastThread      map[string]string
	threadSeen      map[string]map[string]struct{}
	echo            bool
	bufMu           sync.Mutex
	width           int
	color           bool
	colorScheme     map[string]int
	dmPeer          string
	dmPeerID        string
	dmIsBot         bool

	ch     ssh.Channel
	conn   *ssh.ServerConn
	writer io.Writer
	wmu    sync.Mutex
	rl     *readline.Instance
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
	if cfg.ColorScheme == nil {
		cfg.ColorScheme = map[string]int{
			"prompt":    39,
			"timestamp": 244,
			"bot":       81,
			"user":      114,
			"system":    220,
			"info":      45,
			"warning":   214,
			"error":     196,
			"private":   129,
		}
	}

	sc := &sshConnector{
		cfg:          cfg,
		handler:      handler,
		logger:       l,
		botName:      cfg.BotName,
		botNameLower: strings.ToLower(cfg.BotName),
		clients:      make(map[*sshClient]struct{}),
		userKeys:     make(map[string]userKeyInfo),
		userNames:    make(map[string]userKeyInfo),
		userIDs:      make(map[string]userKeyInfo),
		threads:      make(map[string]int),
		buffer:       make([]bufferMsg, cfg.ReplayBufferSize),
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
		threadSeen: make(map[string]map[string]struct{}),
		color:      sc.cfg.Color,
		colorScheme: func() map[string]int {
			if sc.cfg.ColorScheme == nil {
				return nil
			}
			cpy := make(map[string]int, len(sc.cfg.ColorScheme))
			for k, v := range sc.cfg.ColorScheme {
				cpy[strings.ToLower(k)] = v
			}
			return cpy
		}(),
		conn:   sshConn,
		writer: nc,
	}

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

	if err := sc.initReadline(client, ch); err != nil {
		return
	}
	defer func() {
		_ = client.rl.Close()
	}()

	client.setPromptText("Select filter: (A)ll, (C)hannel, (T)hread [C]: ")
	input, err := client.rl.Readline()
	if err != nil {
		if !errors.Is(err, readline.ErrInterrupt) {
			return
		}
		input = ""
	}
	client.filter = parseFilter(input)
	client.clearPromptLine()

	replayed := sc.replayBuffer(client)
	if replayed == 0 {
		client.writeLineKind("info", "(INFO: no recent messages matched the selected filter)")
	}
	client.writeLineKind("system", sshConnectorHelpLine)
	client.setPrompt()

	inputCh := make(chan string)
	go sc.readInput(client, inputCh)

	for line := range inputCh {
		if len(line) == 0 {
			client.writeLineKind("system", sshConnectorHelpLine)
			continue
		}
		if strings.HasPrefix(line, "|") {
			sc.handleCommand(client, strings.TrimSpace(line))
			continue
		}
		sc.handleUserInput(client, line)
	}
}

func (sc *sshConnector) handleCommand(client *sshClient, input string) {
	if len(input) < 2 {
		return
	}
	client.writeString("\n")
	switch input[1] {
	case 'C', 'c':
		arg := strings.TrimSpace(input[2:])
		if arg == "?" {
			client.writeLineKind("system", "Available channels:")
			client.writeLineKind("system", "(direct message to bot); type: '|c'")
			client.writeLineKind("system", "(direct message to user); type: '|c @user'")
			for _, ch := range sc.cfg.Channels {
				client.writeLineKind("system", fmt.Sprintf("'%s'; type: '|c%s'", ch, ch))
			}
			return
		}
		if arg == "" {
			sc.setDMTarget(client, sc.botName, sc.botID, true)
			client.writeLineKind("info", "Changed current channel to: direct message")
			return
		}
		if strings.HasPrefix(arg, "@") {
			target := normalizeUserName(strings.TrimPrefix(arg, "@"))
			if target == "" {
				client.writeLineKind("error", "Invalid 0-length user")
				return
			}
			if target == sc.botNameLower {
			sc.setDMTarget(client, sc.botName, sc.botID, true)
			client.writeLineKind("info", "Changed current channel to: direct message")
			return
		}
			info, ok := sc.lookupUser(target)
			if !ok {
				client.writeLineKind("error", "Unknown user")
				return
			}
			sc.setDMTarget(client, info.userName, info.userID, false)
			client.writeLineKind("info", fmt.Sprintf("Changed current channel to: direct message with @%s", info.userName))
			return
		}
		if !sc.isValidChannel(arg) {
			client.writeLineKind("error", "Invalid channel")
			return
		}
		client.channel = arg
		client.typingInThread = false
		client.threadID = ""
		client.dmPeer = ""
		client.dmPeerID = ""
		client.dmIsBot = false
		client.setPrompt()
		client.writeLineKind("info", fmt.Sprintf("Changed current channel to: %s", arg))
	case 'T', 't':
		if client.dmPeer != "" {
			client.writeLineKind("warning", "(threads are not supported in direct messages)")
			return
		}
		arg := strings.TrimSpace(input[2:])
		if arg == "?" {
			client.writeLineKind("system", "Use '|t' to toggle typing in a thread, '|t<string>' to set the current thread ID, or '|j' to join the last thread seen")
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
			client.setPrompt()
			client.writeLineKind("info", fmt.Sprintf("(now typing in thread: %s)", client.threadID))
		} else {
			client.setPrompt()
			client.writeLineKind("info", "(typing in channel now)")
		}
	case 'J', 'j':
		if client.dmPeer != "" {
			client.writeLineKind("warning", "(threads are not supported in direct messages)")
			return
		}
		last, ok := client.lastThread[client.channel]
		if !ok || last == "" {
			client.writeLineKind("warning", "(sorry, I don't see a thread to join)")
			return
		}
		client.typingInThread = true
		client.threadID = last
		client.setPrompt()
		client.writeLineKind("info", fmt.Sprintf("(now typing in thread: %s)", client.threadID))
	case 'F', 'f':
		arg := strings.TrimSpace(input[2:])
		if arg == "" || arg == "?" {
			client.writeLineKind("system", "Available filters:")
			client.writeLineKind("system", "'All' - type: '|fA'")
			client.writeLineKind("system", "'Channel' - type: '|fC'")
			client.writeLineKind("system", "'Thread' - type: '|fT'")
			return
		}
		client.filter = parseFilter(arg)
		client.writeLineKind("info", fmt.Sprintf("Filter set to: %s", filterLabel(client.filter)))
	case 'L', 'l':
		client.writeLineKind("system", "Available users:")
		for _, entry := range sc.listUsers() {
			label := fmt.Sprintf("'%s'", entry.name)
			if entry.isBot {
				label = fmt.Sprintf("'%s' (bot)", entry.name)
			}
			client.writeLineKind("system", label)
		}
	case '?':
		client.writeLineKind("system", "SSH connector help:")
		client.writeLineKind("system", "|c? - list channels and DM shortcuts")
		client.writeLineKind("system", "|c - direct message with bot")
		client.writeLineKind("system", "|c @user - direct message with user")
		client.writeLineKind("system", "/@user <message> - one-shot direct message to user or bot")
		client.writeLineKind("system", "|t? - thread help (threads disabled in DMs)")
		client.writeLineKind("system", "|j - join last thread")
		client.writeLineKind("system", "|f? - list/change filter")
		client.writeLineKind("system", "|l - list users")
	default:
		client.writeLineKind("error", "Invalid SSH connector command")
	}
}

func (sc *sshConnector) handleUserInput(client *sshClient, line string) {
	now := time.Now()
	client.echoInputWithTimestamp(line, now)

	if len(line) > sc.cfg.MaxMsgBytes {
		client.writeLineKind("error", "(ERROR: message too long; > 16k - dropped)")
		return
	}

	if len(line) > maxBufferBytes {
		client.writeLineKind("warning", "(WARNING: message truncated to 4k in buffer)")
	}

	trimmed := strings.TrimSpace(line)
	if client.dmPeer != "" && !client.dmIsBot && strings.HasPrefix(trimmed, "/") {
		client.writeLineKind("warning", "(input dropped: '/' commands are disabled in user-to-user DMs)")
		return
	}
	if strings.HasPrefix(trimmed, "/@") {
		sc.handleDirectAt(client, trimmed, now)
		return
	}
	if strings.HasPrefix(trimmed, "/") {
		if strings.HasPrefix(trimmed, "/ ") || trimmed == "/" {
			client.writeLineKind("info", "(INFO: '/' note to self message not sent to other users)")
			return
		}
		if payload, ok := sc.botHiddenPayload(trimmed); ok {
			sc.sendIncoming(client, payload, true, client.dmPeer != "")
			return
		}
		payload := strings.TrimSpace(trimmed[1:])
		if payload != "" {
			sc.sendIncoming(client, payload, true, client.dmPeer != "")
		}
		return
	}

	if client.dmPeer != "" {
		if client.dmIsBot {
			sc.sendIncoming(client, line, false, true)
			sc.appendDirectBuffer(client, sc.botName, sc.botID, false, now, line)
		} else {
			sc.sendDirectUserMessage(client, line, now)
		}
		return
	}

	sc.sendIncoming(client, line, false, false)
	sc.broadcastUserMessage(client, line, now)
}

func (sc *sshConnector) handleDirectAt(client *sshClient, trimmed string, ts time.Time) {
	rest := strings.TrimSpace(strings.TrimPrefix(trimmed, "/@"))
	if rest == "" {
		client.writeLineKind("error", "Missing user for direct message")
		return
	}
	parts := strings.Fields(rest)
	targetRaw := parts[0]
	message := strings.TrimSpace(strings.TrimPrefix(rest, targetRaw))
	if message == "" {
		client.writeLineKind("error", "Direct message requires text")
		return
	}
	target := normalizeUserName(targetRaw)
	if target == "" {
		client.writeLineKind("error", "Invalid user")
		return
	}
	if target == sc.botNameLower {
		sc.sendIncoming(client, message, client.dmIsBot, true)
		sc.appendDirectBuffer(client, sc.botName, sc.botID, false, ts, message)
		return
	}
	info, ok := sc.lookupUser(target)
	if !ok {
		client.writeLineKind("error", "Unknown user")
		return
	}
	sc.sendDirectUserMessageTo(client, info, message, ts)
}

func (sc *sshConnector) sendIncoming(client *sshClient, line string, hidden bool, direct bool) {
	threaded := client.typingInThread && !direct
	threadID := ""
	messageID := ""
	channelName := client.channel
	channelID := "#" + client.channel
	if direct {
		channelName = ""
		channelID = ""
	}
	if threaded {
		messageID = sc.nextThreadID(channelName)
		threadID = client.threadID
	} else {
		threadID = sc.nextThreadID(channelName)
		messageID = threadID
	}

	msg := &robot.ConnectorMessage{
		Protocol:        "ssh",
		UserName:        client.userName,
		UserID:          client.userID,
		ChannelName:     channelName,
		ChannelID:       channelID,
		MessageID:       messageID,
		ThreadedMessage: threaded,
		ThreadID:        threadID,
		MessageText:     line,
		HiddenMessage:   hidden,
		DirectMessage:   direct,
	}
	sc.handler.IncomingMessage(msg)
}

func (sc *sshConnector) MessageHeard(u, c string) {}

func (sc *sshConnector) SetUserMap(m map[string]string) {
	keys := make(map[string]userKeyInfo)
	names := make(map[string]userKeyInfo)
	ids := make(map[string]userKeyInfo)
	for name, id := range m {
		if strings.ToLower(name) != name {
			sc.handler.Log(robot.Error, "SSH connector: rejecting username with uppercase letters: %q", name)
			continue
		}
		norm := normalizeKeyLine(id)
		if norm == "" {
			continue
		}
		if _, exists := names[name]; exists {
			sc.handler.Log(robot.Error, "SSH connector: duplicate username in roster: %q", name)
			continue
		}
		info := userKeyInfo{userName: name, userID: id}
		keys[norm] = info
		names[name] = info
		ids[id] = info
	}
	sc.mu.Lock()
	sc.userKeys = keys
	sc.userNames = names
	sc.userIDs = ids
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
		userID:    sc.botID,
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
		userID:    sc.botID,
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
	info, ok := sc.resolveUser(sc.normalizeUser(u))
	if !ok {
		return robot.UserNotFound
	}
	clients := sc.clientsForUser(info.userID)
	if len(clients) == 0 {
		return robot.UserNotFound
	}

	evt := sc.directEvent(sc.botName, sc.botID, true, info.userName, info.userID, msg, time.Now())
	sc.appendBuffer(evt)
	for _, client := range clients {
		client.writeMessageAsync(evt, false, false)
	}
	return robot.Ok
}

func (sc *sshConnector) broadcastUserMessage(client *sshClient, line string, ts time.Time) {
	threaded := client.typingInThread
	threadID := client.threadID
	evt := bufferMsg{
		timestamp: ts,
		userName:  client.userName,
		userID:    client.userID,
		isBot:     false,
		channel:   client.channel,
		threadID:  threadID,
		threaded:  threaded,
		text:      line,
	}
	sc.broadcast(evt, &robot.ConnectorMessage{UserID: client.userID, HiddenMessage: false})
}

func (sc *sshConnector) directEvent(senderName, senderID string, senderIsBot bool, peerName, peerID, text string, ts time.Time) bufferMsg {
	return bufferMsg{
		timestamp: ts,
		userName:  senderName,
		userID:    senderID,
		isBot:     senderIsBot,
		text:      text,
		isDM:      true,
		dmPeer:    peerName,
		dmPeerID:  peerID,
	}
}

func (sc *sshConnector) appendDirectBuffer(sender *sshClient, peerName, peerID string, senderIsBot bool, ts time.Time, text string) {
	evt := sc.directEvent(sender.userName, sender.userID, senderIsBot, peerName, peerID, text, ts)
	sc.appendBuffer(evt)
}

func (sc *sshConnector) sendDirectUserMessage(sender *sshClient, text string, ts time.Time) {
	if sender.dmPeer == "" || sender.dmPeerID == "" {
		sender.writeLineKind("error", "No direct message target")
		return
	}
	peer := userKeyInfo{userName: sender.dmPeer, userID: sender.dmPeerID}
	sc.sendDirectUserMessageTo(sender, peer, text, ts)
}

func (sc *sshConnector) sendDirectUserMessageTo(sender *sshClient, peer userKeyInfo, text string, ts time.Time) {
	evt := sc.directEvent(sender.userName, sender.userID, false, peer.userName, peer.userID, text, ts)
	sc.appendBuffer(evt)
	clients := sc.clientsForUser(peer.userID)
	for _, client := range clients {
		if client.userID == sender.userID {
			continue
		}
		client.writeMessageAsync(evt, false, false)
	}
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
			client.writeMessageAsync(evt, true, false)
			continue
		}
		if client.userName == evt.userName {
			continue
		}
		send, announceThread := client.shouldSend(evt)
		if !send {
			continue
		}
		client.writeMessageAsync(evt, false, announceThread)
		if evt.threaded {
			client.lastThread[evt.channel] = evt.threadID
		}
	}
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

func (sc *sshConnector) replayBuffer(client *sshClient) int {
	msgs := sc.snapshotBuffer()
	count := 0
	for _, evt := range msgs {
		send, announceThread := client.shouldSend(evt)
		if !send {
			continue
		}
		client.writeMessage(evt, false, announceThread)
		count++
		if evt.threaded {
			client.lastThread[evt.channel] = evt.threadID
		}
	}
	return count
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

func normalizeUserName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.TrimPrefix(name, "@")
	return strings.ToLower(name)
}

func (sc *sshConnector) lookupUser(name string) (userKeyInfo, bool) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	info, ok := sc.userNames[name]
	return info, ok
}

func (sc *sshConnector) lookupUserByID(id string) (userKeyInfo, bool) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	info, ok := sc.userIDs[id]
	return info, ok
}

func (sc *sshConnector) resolveUser(u string) (userKeyInfo, bool) {
	if u == "" {
		return userKeyInfo{}, false
	}
	if info, ok := sc.lookupUserByID(u); ok {
		return info, true
	}
	name := normalizeUserName(u)
	if name == "" {
		return userKeyInfo{}, false
	}
	return sc.lookupUser(name)
}

func (sc *sshConnector) listUsers() []userListing {
	sc.mu.RLock()
	names := make([]userListing, 0, len(sc.userNames)+1)
	for name := range sc.userNames {
		names = append(names, userListing{name: name})
	}
	sc.mu.RUnlock()
	if sc.botName != "" {
		found := false
		for _, entry := range names {
			if entry.name == sc.botName {
				found = true
				break
			}
		}
		if !found {
			names = append(names, userListing{name: sc.botName, isBot: true})
		}
	}
	sort.Slice(names, func(i, j int) bool {
		return names[i].name < names[j].name
	})
	return names
}

func (sc *sshConnector) setDMTarget(client *sshClient, peerName, peerID string, isBot bool) {
	client.dmPeer = peerName
	client.dmPeerID = peerID
	client.dmIsBot = isBot
	client.channel = peerName
	client.typingInThread = false
	client.threadID = ""
	client.setPrompt()
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

func (sc *sshConnector) botHiddenPayload(line string) (string, bool) {
	if !strings.HasPrefix(line, "/") {
		return "", false
	}
	trimmed := strings.TrimPrefix(line, "/")
	if trimmed == "" {
		return "", false
	}
	lower := strings.ToLower(trimmed)
	if !strings.HasPrefix(lower, sc.botNameLower) {
		return "", false
	}
	if len(trimmed) == len(sc.botNameLower) {
		return "", false
	}
	if trimmed[len(sc.botNameLower)] != ' ' {
		return "", false
	}
	remainder := strings.TrimSpace(trimmed[len(sc.botNameLower):])
	if remainder == "" {
		return "", false
	}
	payload := strings.TrimSpace(sc.botName + " " + remainder)
	return payload, true
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

const sshConnectorHelpLine = "SSH connector: Type '|?' for help, '|c?' to list channels, '|l' to list users, '|t?' for thread help, '|j' to join last thread, '|f?' to list/change filter"

func (c *sshClient) promptTarget() string {
	if c.dmPeer != "" {
		return fmt.Sprintf("dm:@%s", c.dmPeer)
	}
	return fmt.Sprintf("#%s", c.channel)
}

func (c *sshClient) writePrompt() {
	c.refreshPrompt()
}

func (c *sshClient) refreshPrompt() {
	if c.rl == nil {
		return
	}
	c.rl.Refresh()
}

func (c *sshClient) writeString(s string) {
	c.writeStringKind("", s)
}

func (c *sshClient) writeStringKind(kind, s string) {
	c.writeOutputKind(kind, s, false)
}

func (c *sshClient) writeLine(s string) {
	c.writeLineKind("", s)
}

func (c *sshClient) writeLineKind(kind, s string) {
	c.writeOutputKind(kind, s, true)
}

func (c *sshClient) writeAsyncMessage(msg string) {
	c.writeAsyncMessageKind("", msg)
}

func (c *sshClient) writeAsyncMessageKind(kind, msg string) {
	c.writeOutputKind(kind, msg, true)
}

func (c *sshClient) writeMessage(evt bufferMsg, private, announceThread bool) {
	msg := c.formatMessage(evt, private, announceThread)
	c.writeOutput(msg + "\n")
}

func (c *sshClient) writeMessageAsync(evt bufferMsg, private, announceThread bool) {
	msg := c.formatMessage(evt, private, announceThread)
	c.writeOutput(msg + "\n")
}

func (c *sshClient) writeOutputKind(kind, s string, addNewline bool) {
	c.wmu.Lock()
	defer c.wmu.Unlock()
	out := s
	if addNewline {
		out = c.wrapLine(out)
	}
	out = c.colorizeLines(kind, out)
	if addNewline {
		out += "\n"
	}
	c.writeOutputLocked(out)
}

func (c *sshClient) writeOutput(s string) {
	c.wmu.Lock()
	defer c.wmu.Unlock()
	c.writeOutputLocked(s)
}

func (c *sshClient) writeOutputLocked(s string) {
	writer := c.writer
	if c.rl != nil {
		writer = c.rl
	}
	_, _ = writer.Write([]byte(normalizeNewlines(s)))
}

func (c *sshClient) promptString() string {
	thread := ""
	if c.dmPeer == "" && c.typingInThread && c.threadID != "" {
		thread = fmt.Sprintf("(%s)", c.threadID)
	}
	return fmt.Sprintf("@%s/%s%s -> ", c.userName, c.promptTarget(), thread)
}

func (c *sshClient) promptColored() string {
	return c.colorize("prompt", c.promptString())
}

func (c *sshClient) setPrompt() {
	c.setPromptText(c.promptColored())
}

func (c *sshClient) setPromptText(prompt string) {
	if c.rl == nil {
		return
	}
	c.rl.SetPrompt(prompt)
	c.rl.Refresh()
}

func (c *sshClient) clearPromptLine() {
	if c.rl == nil {
		return
	}
	c.rl.SetPrompt("")
	c.rl.Clean()
}

func (c *sshClient) echoInputWithTimestamp(line string, ts time.Time) {
	if c.rl == nil {
		return
	}
	stamp := c.colorize("timestamp", fmt.Sprintf(" (%s)", ts.Format("15:04:05")))
	out := c.promptColored() + line + stamp + "\n"

	c.wmu.Lock()
	defer c.wmu.Unlock()
	c.rl.Clean()
	_, _ = c.rl.Write([]byte(normalizeNewlines(out)))
}

func (c *sshClient) formatMessage(evt bufferMsg, private, announceThread bool) string {
	stamp := evt.timestamp.Format("15:04:05")
	prefix := "@" + evt.userName
	if evt.isBot {
		prefix = "=@" + evt.userName
	}
	thread := ""
	if !evt.isDM && evt.threaded && evt.threadID != "" {
		if announceThread {
			thread = fmt.Sprintf("(+%s)", evt.threadID)
		} else {
			thread = fmt.Sprintf("(%s)", evt.threadID)
		}
	}
	channel := ""
	if evt.isDM {
		label := c.dmLabel(evt)
		channel = label
		header := fmt.Sprintf("(%s)%s:", stamp, label)
		if private {
			header = fmt.Sprintf("(private/%s)%s:", stamp, label)
		}
		if !c.color {
			return c.formatMessageBodyWithHeader(header, "", evt)
		}
		headerColored := c.colorizeDMHeader(stamp, label, private)
		return c.formatMessageBodyWithHeader(header, headerColored, evt)
	}
	channel = "#" + evt.channel
	if evt.channel == "" {
		channel = "#(direct)"
	}

	header := fmt.Sprintf("(%s)%s/%s%s:", stamp, prefix, channel, thread)
	if private {
		header = fmt.Sprintf("(private/%s)%s/%s%s:", stamp, prefix, channel, thread)
	}
	headerColored := ""
	if c.color {
		headerColored = c.colorizeHeader(stamp, prefix, channel, thread, evt.isBot, private)
	}
	return c.formatMessageBodyWithHeader(header, headerColored, evt)
}

func (c *sshClient) colorize(kind, s string) string {
	if !c.color || s == "" {
		return s
	}
	if c.colorScheme == nil {
		return s
	}
	code, ok := c.colorScheme[strings.ToLower(kind)]
	if !ok {
		return s
	}
	return fmt.Sprintf("\x1b[38;5;%dm%s%s", code, s, ansiReset)
}

func (c *sshClient) colorizeHeader(stamp, prefix, channel, thread string, isBot, private bool) string {
	var b strings.Builder
	b.WriteString("(")
	if private {
		b.WriteString(c.colorize("private", "private"))
		b.WriteString("/")
	}
	b.WriteString(c.colorize("timestamp", stamp))
	b.WriteString(")")

	userKind := "user"
	if isBot {
		userKind = "bot"
	}
	b.WriteString(c.colorize(userKind, prefix))
	b.WriteString("/")
	b.WriteString(c.colorize("prompt", channel))
	if thread != "" {
		b.WriteString(c.colorize("prompt", thread))
	}
	b.WriteString(":")
	return b.String()
}

func (c *sshClient) colorizeDMHeader(stamp, channel string, private bool) string {
	var b strings.Builder
	b.WriteString("(")
	if private {
		b.WriteString(c.colorize("private", "private"))
		b.WriteString("/")
	}
	b.WriteString(c.colorize("timestamp", stamp))
	b.WriteString(")")
	b.WriteString(c.colorize("prompt", channel))
	b.WriteString(":")
	return b.String()
}

func (c *sshClient) dmLabel(evt bufferMsg) string {
	isSelf := false
	if evt.userID != "" && c.userID != "" {
		isSelf = evt.userID == c.userID
	} else {
		isSelf = evt.userName == c.userName
	}
	if isSelf {
		peer := evt.dmPeer
		if peer == "" {
			peer = "(direct)"
		}
		return "to:@" + peer
	}
	sender := evt.userName
	if sender == "" {
		sender = "(direct)"
	}
	if c.dmPeer != "" && normalizeUserName(c.dmPeer) == normalizeUserName(sender) {
		return "@" + sender
	}
	return "from:@" + sender
}

func (c *sshClient) formatMessageBodyWithHeader(header, headerColored string, evt bufferMsg) string {
	body := " " + evt.text
	wrapped := c.wrapLine(header + body)
	if !c.color || headerColored == "" {
		return wrapped
	}

	bodyKind := "user"
	if evt.isBot {
		bodyKind = "bot"
	}

	lines := strings.Split(wrapped, "\n")
	if len(lines) == 0 {
		return wrapped
	}
	if strings.HasPrefix(lines[0], header) {
		remainder := lines[0][len(header):]
		lines[0] = headerColored + c.colorize(bodyKind, remainder)
		for i := 1; i < len(lines); i++ {
			lines[i] = c.colorize(bodyKind, lines[i])
		}
		return strings.Join(lines, "\n")
	}
	for i := 0; i < len(lines); i++ {
		lines[i] = c.colorize(bodyKind, lines[i])
	}
	return strings.Join(lines, "\n")
}

func (c *sshClient) colorizeLines(kind, s string) string {
	if !c.color || s == "" {
		return s
	}
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if line == "" {
			continue
		}
		lines[i] = c.colorize(kind, line)
	}
	return strings.Join(lines, "\n")
}

func (c *sshClient) shouldSend(evt bufferMsg) (bool, bool) {
	if evt.isDM {
		if evt.userID != "" && c.userID != "" {
			return evt.userID == c.userID || evt.dmPeerID == c.userID, false
		}
		return evt.userName == c.userName || evt.dmPeer == c.userName, false
	}
	switch c.filter {
	case filterAll:
		return true, false
	case filterChannel:
		return evt.channel == c.channel, false
	case filterThread:
		if c.typingInThread {
			return evt.channel == c.channel && evt.threaded && evt.threadID == c.threadID, false
		}
		if evt.channel != c.channel {
			return false, false
		}
		if !evt.threaded {
			return true, false
		}
		if c.seenThread(evt.channel, evt.threadID) {
			return false, false
		}
		c.markThreadSeen(evt.channel, evt.threadID)
		return true, true
	default:
		return false, false
	}
}

func (c *sshClient) seenThread(channel, threadID string) bool {
	if channel == "" || threadID == "" {
		return false
	}
	c.bufMu.Lock()
	defer c.bufMu.Unlock()
	seen, ok := c.threadSeen[channel]
	if !ok {
		return false
	}
	_, ok = seen[threadID]
	return ok
}

func (c *sshClient) markThreadSeen(channel, threadID string) {
	if channel == "" || threadID == "" {
		return
	}
	c.bufMu.Lock()
	defer c.bufMu.Unlock()
	seen, ok := c.threadSeen[channel]
	if !ok {
		seen = make(map[string]struct{})
		c.threadSeen[channel] = seen
	}
	seen[threadID] = struct{}{}
}

func (sc *sshConnector) readInput(client *sshClient, out chan<- string) {
	defer close(out)
	for {
		line, err := client.rl.Readline()
		if err != nil {
			if errors.Is(err, readline.ErrInterrupt) {
				continue
			}
			return
		}
		out <- line
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

func (sc *sshConnector) initReadline(client *sshClient, ch ssh.Channel) error {
	historyLimit := sc.cfg.UserHistoryLines
	cfg := &readline.Config{
		Prompt:                "",
		HistoryLimit:          historyLimit,
		DisableAutoSaveHistory: false,
		HistorySearchFold:     true,
		UniqueEditLine:        true,
		Stdin:                 ch,
		Stdout:                ch,
		Stderr:                ch,
		ForceUseInteractive:   true,
		FuncGetWidth:          client.getWidth,
		FuncIsTerminal:        func() bool { return true },
		FuncMakeRaw:           func() error { return nil },
		FuncExitRaw:           func() error { return nil },
		FuncOnWidthChanged:    func(func()) {},
	}
	rl, err := readline.NewEx(cfg)
	if err != nil {
		return err
	}
	client.rl = rl
	return nil
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
	c.refreshPrompt()
}

func (c *sshClient) getWidth() int {
	c.bufMu.Lock()
	defer c.bufMu.Unlock()
	if c.width <= 0 {
		return 80
	}
	return c.width
}

func (c *sshClient) wrapLine(s string) string {
	c.bufMu.Lock()
	width := c.width
	c.bufMu.Unlock()
	if width <= 0 {
		return s
	}
	wrapper := botwrap.NewWrapper()
	wrapper.StripTrailingNewline = true
	return wrapper.Wrap(s, width)
}

func (sc *sshConnector) wrap(s string) string {
	return s
}
