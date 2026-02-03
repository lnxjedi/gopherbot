package ssh

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/chzyer/readline"
	"golang.org/x/crypto/ssh"

	botwrap "github.com/lnxjedi/gopherbot/v2/bot"
)

const (
	pasteStart = "\x1b[200~"
	pasteEnd   = "\x1b[201~"
	pasteOn    = "\x1b[?2004h"
	pasteOff   = "\x1b[?2004l"
	ansiReset  = "\x1b[0m"
)

type sshClient struct {
	userName string
	userID   string

	channel        string
	threadID       string
	typingInThread bool
	filter         filterMode
	lastThread     map[string]string
	threadSeen     map[string]map[string]struct{}
	echo           bool
	bufMu          sync.Mutex
	width          int
	color          bool
	colorScheme    map[string]int
	dmPeer         string
	dmPeerID       string
	dmIsBot        bool

	ch     ssh.Channel
	conn   *ssh.ServerConn
	writer io.Writer
	wmu    sync.Mutex
	rl     *readline.Instance
}

type inputEvent struct {
	line      string
	multiline bool
}

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

func (c *sshClient) echoInputWithTimestamp(line string, ts time.Time) {
	if c.rl == nil {
		return
	}
	stampRaw := fmt.Sprintf(" (%s)", ts.Format("15:04:05"))
	stampRawNoLead := strings.TrimPrefix(stampRaw, " ")
	stamp := c.colorize("timestamp", stampRaw)
	padding := ""
	width := c.getWidth()
	if width > 0 {
		promptWidth := readline.Runes{}.WidthAll([]rune(c.promptString()))
		lineWidth := readline.Runes{}.WidthAll([]rune(line))
		stampWidth := readline.Runes{}.WidthAll([]rune(stampRawNoLead))
		col := (promptWidth + lineWidth) % width
		if col > 0 && col+stampWidth > width {
			padding = strings.Repeat(" ", width-col)
			stamp = c.colorize("timestamp", stampRawNoLead)
		}
	}
	out := c.promptColored() + line + padding + stamp + "\n"

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

func (sc *sshConnector) readInput(client *sshClient, out chan<- inputEvent) {
	defer close(out)
	var buf strings.Builder
	continuing := false
	var lineBuf []rune
	origUnique := client.rl.Config.UniqueEditLine
	origFilter := client.rl.Config.FuncFilterInputRune
	origListener := client.rl.Config.Listener
	client.rl.Config.FuncFilterInputRune = func(r rune) (rune, bool) {
		if origFilter != nil {
			var ok bool
			r, ok = origFilter(r)
			if !ok {
				return r, false
			}
		}
		switch r {
		case readline.CharEnter, readline.CharCtrlJ:
			if continuing || (len(lineBuf) > 0 && lineBuf[len(lineBuf)-1] == '\\') {
				client.rl.Config.UniqueEditLine = false
			} else {
				client.rl.Config.UniqueEditLine = true
			}
			return r, true
		default:
			return r, true
		}
	}
	client.rl.Config.SetListener(func(line []rune, _ int, _ rune) (newLine []rune, newPos int, ok bool) {
		lineBuf = append(lineBuf[:0], line...)
		return nil, 0, false
	})
	defer func() {
		client.rl.Config.UniqueEditLine = origUnique
		client.rl.Config.FuncFilterInputRune = origFilter
		client.rl.Config.Listener = origListener
	}()
	for {
		line, err := client.rl.Readline()
		if err != nil {
			if errors.Is(err, readline.ErrInterrupt) {
				continue
			}
			if errors.Is(err, io.EOF) {
				client.writeLineKind("system", "disconnecting ...")
			}
			return
		}
		if strings.HasSuffix(line, "\\") {
			line = strings.TrimSuffix(line, "\\")
			buf.WriteString(line)
			buf.WriteString("\n")
			if !continuing {
				client.setPromptText("")
				continuing = true
			}
			continue
		}
		if continuing {
			buf.WriteString(line)
			out <- inputEvent{line: buf.String(), multiline: true}
			buf.Reset()
			continuing = false
			client.setPrompt()
			continue
		}
		out <- inputEvent{line: line}
	}
}

func normalizeNewlines(s string) string {
	return strings.ReplaceAll(s, "\n", "\r\n")
}

func (sc *sshConnector) initReadline(client *sshClient, ch ssh.Channel) error {
	historyLimit := sc.cfg.UserHistoryLines
	cfg := &readline.Config{
		Prompt:                 "",
		HistoryLimit:           historyLimit,
		DisableAutoSaveHistory: false,
		HistorySearchFold:      true,
		UniqueEditLine:         true,
		Stdin:                  ch,
		Stdout:                 ch,
		Stderr:                 ch,
		ForceUseInteractive:    true,
		FuncGetWidth:           client.getWidth,
		FuncIsTerminal:         func() bool { return true },
		FuncMakeRaw:            func() error { return nil },
		FuncExitRaw:            func() error { return nil },
		FuncOnWidthChanged:     func(func()) {},
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
