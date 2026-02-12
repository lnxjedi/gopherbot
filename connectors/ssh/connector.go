package ssh

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/lnxjedi/gopherbot/robot"
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

func (sc *sshConnector) handleCommand(client *sshClient, input string) {
	if len(input) < 2 {
		return
	}
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
	// TODO(v3 multi-protocol): On roster reload, disconnect active sessions whose
	// key/identity is no longer present in the updated map.
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

func (sc *sshConnector) wrap(s string) string {
	return s
}
