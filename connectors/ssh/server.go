package ssh

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/chzyer/readline"
	"golang.org/x/crypto/ssh"

	"github.com/lnxjedi/gopherbot/robot"
)

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
		_, _ = ch.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{Status: 0}))
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
	origFilter := client.rl.Config.FuncFilterInputRune
	client.rl.Config.FuncFilterInputRune = func(r rune) (rune, bool) {
		switch r {
		case readline.CharEnter, readline.CharCtrlJ, readline.CharInterrupt:
			return r, true
		}
		switch r {
		case 'a', 'A', 'c', 'C', 't', 'T':
			client.rl.Operation.SetBuffer(string(r))
			return readline.CharEnter, true
		default:
			return r, false
		}
	}
	input, err := client.rl.Readline()
	client.rl.Config.FuncFilterInputRune = origFilter
	if err != nil {
		if !errors.Is(err, readline.ErrInterrupt) {
			return
		}
		input = ""
	}
	client.filter = parseFilter(input)
	// Readline emits a LF without CR here; move to column 0 on the next line.
	client.writeOutput("\r")

	replayed := sc.replayBuffer(client)
	if replayed == 0 {
		client.writeLineKind("info", "(INFO: no recent messages matched the selected filter)")
	}
	client.writeLineKind("system", sshConnectorHelpLine)
	client.setPrompt()

	inputCh := make(chan inputEvent, 8)
	go sc.readInput(client, inputCh)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for evt := range inputCh {
			line := evt.line
			if len(line) == 0 {
				if evt.multiline {
					continue
				}
				client.writeLineKind("system", sshConnectorHelpLine)
				continue
			}
			if !evt.multiline && strings.HasPrefix(line, "|") {
				sc.handleCommand(client, strings.TrimSpace(line))
				continue
			}
			sc.handleUserInput(client, line)
		}
	}()

	<-done
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
