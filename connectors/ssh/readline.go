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
	pasteActive    bool
	continuing     bool
	stampColor     bool

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

func (c *sshClient) writeLineKind(kind, s string) {
	c.writeOutputKind(kind, s, true)
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

type inlineStampPainter struct {
	client *sshClient
}

func (p *inlineStampPainter) Paint(line []rune, _ int) []rune {
	if p == nil || p.client == nil || !p.client.isStampColor() {
		return line
	}
	if len(line) < 10 {
		return line
	}
	hasNL := line[len(line)-1] == '\n'
	base := line
	if hasNL {
		line = line[:len(line)-1]
		if len(line) < 10 {
			return base
		}
	}
	if line[len(line)-1] != ')' {
		return base
	}
	start := len(line) - 10
	if line[start] != '(' || line[start+3] != ':' || line[start+6] != ':' {
		return base
	}
	for _, idx := range []int{start + 1, start + 2, start + 4, start + 5, start + 7, start + 8} {
		if line[idx] < '0' || line[idx] > '9' {
			return base
		}
	}
	stamp := string(line[start:])
	colored := p.client.colorize("timestamp", stamp)
	if colored == stamp {
		return base
	}
	out := make([]rune, 0, start+len([]rune(colored))+1)
	out = append(out, line[:start]...)
	out = append(out, []rune(colored)...)
	if hasNL {
		out = append(out, '\n')
	}
	return out
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
	client.setContinuing(false)
	for {
		line, err := client.rl.Readline()
		client.clearStampColor()
		if err != nil {
			if errors.Is(err, readline.ErrInterrupt) {
				continue
			}
			if errors.Is(err, io.EOF) {
				// Commit the current line so the disconnect message prints on the next line.
				client.writeOutput("\r")
				client.writeLineKind("system", "disconnecting ...")
			}
			return
		}
		if strings.HasSuffix(line, "\\") || client.isPasteActive() {
			line = strings.TrimSuffix(line, "\\")
			buf.WriteString(line)
			buf.WriteString("\n")
			if !continuing {
				client.setPromptText(client.colorize("prompt", "> "))
				continuing = true
				client.setContinuing(true)
			}
			continue
		}
		if continuing {
			buf.WriteString(line)
			// Move to column 0 before emitting the standalone timestamp.
			client.writeOutput("\r")
			client.writeLineKind("timestamp", fmt.Sprintf("(%s)", time.Now().Format("15:04:05")))
			out <- inputEvent{line: buf.String(), multiline: true}
			buf.Reset()
			continuing = false
			client.setContinuing(false)
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
		Stdin:                  ch,
		Stdout:                 ch,
		Stderr:                 ch,
		ForceUseInteractive:    true,
		FuncGetWidth:           client.getWidth,
		FuncSetPasteMode:       client.setPasteActive,
		FuncIsTerminal:         func() bool { return true },
		FuncMakeRaw:            func() error { return nil },
		FuncExitRaw:            func() error { return nil },
		FuncOnWidthChanged:     func(func()) {},
		Painter:                &inlineStampPainter{client: client},
		FuncBeforeSubmit: func(line []rune) ([]rune, int) {
			return client.stampSuffixWrapSmart(line, time.Now())
		},
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

func (c *sshClient) setPasteActive(on bool) {
	c.bufMu.Lock()
	c.pasteActive = on
	c.bufMu.Unlock()
}

func (c *sshClient) isPasteActive() bool {
	c.bufMu.Lock()
	defer c.bufMu.Unlock()
	return c.pasteActive
}

func (c *sshClient) setContinuing(on bool) {
	c.bufMu.Lock()
	c.continuing = on
	c.bufMu.Unlock()
}

func (c *sshClient) isContinuing() bool {
	c.bufMu.Lock()
	defer c.bufMu.Unlock()
	return c.continuing
}

func (c *sshClient) setStampColor(on bool) {
	c.bufMu.Lock()
	c.stampColor = on
	c.bufMu.Unlock()
}

func (c *sshClient) clearStampColor() {
	c.setStampColor(false)
}

func (c *sshClient) isStampColor() bool {
	c.bufMu.Lock()
	defer c.bufMu.Unlock()
	return c.stampColor
}

func (c *sshClient) stampSuffixWrapSmart(line []rune, ts time.Time) ([]rune, int) {
	if len(line) == 0 {
		c.clearStampColor()
		return nil, 0
	}
	if c.isContinuing() || c.isPasteActive() {
		c.clearStampColor()
		return nil, 0
	}
	if line[len(line)-1] == '\\' {
		c.clearStampColor()
		return nil, 0
	}
	stamp := fmt.Sprintf(" (%s)", ts.Format("15:04:05"))
	width := c.getWidth()
	if width > 0 {
		promptWidth := readline.Runes{}.WidthAll([]rune(c.promptString()))
		lineWidth := readline.Runes{}.WidthAll(line)
		col := (promptWidth + lineWidth) % width
		if col == 0 {
			stamp = strings.TrimPrefix(stamp, " ")
		} else {
			stampWidth := len(stamp)
			if col+stampWidth > width {
				padding := width - col
				if padding > 1 {
					stamp = strings.Repeat(" ", padding-1) + stamp
				}
			}
		}
	}
	suffix := []rune(stamp)
	if len(suffix) == 0 {
		c.clearStampColor()
		return nil, 0
	}
	c.setStampColor(true)
	return suffix, len(suffix)
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
