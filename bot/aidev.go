package bot

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
	"golang.org/x/sys/unix"
)

type aidevConfig struct {
	enabled bool
	secret  string
}

var aidev = struct {
	cfg aidevConfig
}{
	cfg: aidevConfig{},
}

type pendingInjection struct {
	nonce   string
	user    string
	userID  string
	channel string
	thread  string
	created time.Time
}

var pendingInjections = struct {
	sync.Mutex
	m map[string]pendingInjection
}{
	m: make(map[string]pendingInjection),
}

const (
	aidevNonceLen = 7
	aidevTTL      = 30 * time.Second
)

var aidevPrefixRe = regexp.MustCompile(`^\(#([0-9a-fA-F]{7}) as: ([^)]+)\)\s*`)

type tapEvent struct {
	Direction   string
	Protocol    string
	UserName    string
	UserID      string
	ChannelName string
	ChannelID   string
	ThreadID    string
	MessageID   string
	SelfMessage bool
	BotMessage  bool
	Hidden      bool
	Direct      bool
	Text        string
}

type aidevInjectRequest struct {
	User    string `json:"user"`
	UserID  string `json:"user_id"`
	Channel string `json:"channel"`
	Thread  string `json:"thread"`
	Message string `json:"message"`
	Direct  bool   `json:"direct"`
	Hidden  bool   `json:"hidden"`
}

type aidevControlRequest struct {
	Action string `json:"action"`
}

type tapListener struct {
	ch chan tapEvent
}

var aidevTaps = struct {
	sync.Mutex
	list map[*tapListener]struct{}
}{
	list: make(map[*tapListener]struct{}),
}

var aidevHello = struct {
	sync.Mutex
	ch chan struct{}
}{
	ch: make(chan struct{}, 1),
}

var aidevReady = struct {
	sync.Mutex
	ch chan struct{}
}{
	ch: make(chan struct{}, 1),
}

var aidevStart = struct {
	sync.Mutex
	ch chan struct{}
}{
	ch: make(chan struct{}, 1),
}

var aidevInitialized = struct {
	sync.RWMutex
	ready bool
}{
	ready: false,
}

func aidevEnabled() bool {
	return aidev.cfg.enabled
}

func aidevSecretValue() string {
	return aidev.cfg.secret
}

func newAidevNonce() (string, error) {
	buf := make([]byte, (aidevNonceLen+1)/2)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	nonce := hex.EncodeToString(buf)
	if len(nonce) > aidevNonceLen {
		nonce = nonce[:aidevNonceLen]
	}
	return nonce, nil
}

func newAidevToken() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return ""
	}
	return hex.EncodeToString(buf)
}

func aidevPrefix(nonce, user string) string {
	return fmt.Sprintf("(#%s as: %s) ", nonce, user)
}

func parseAidevPrefix(msg string) (nonce, user, stripped string, ok bool) {
	matches := aidevPrefixRe.FindStringSubmatch(msg)
	if len(matches) != 3 {
		return "", "", msg, false
	}
	nonce = strings.ToLower(matches[1])
	user = matches[2]
	stripped = strings.TrimSpace(msg[len(matches[0]):])
	return nonce, user, stripped, true
}

func unbracketID(v string) string {
	h := handler{}
	if id, ok := h.ExtractID(v); ok {
		return id
	}
	return v
}

func enqueueInjection(p pendingInjection) error {
	if !aidevEnabled() {
		return errors.New("aidev disabled")
	}
	if p.nonce == "" {
		return errors.New("missing nonce")
	}
	p.created = time.Now()
	pendingInjections.Lock()
	pendingInjections.m[p.nonce] = p
	pendingInjections.Unlock()
	return nil
}

func consumeInjection(nonce string) (pendingInjection, bool) {
	pendingInjections.Lock()
	defer pendingInjections.Unlock()
	p, ok := pendingInjections.m[nonce]
	if ok {
		delete(pendingInjections.m, nonce)
	}
	return p, ok
}

func pruneExpiredInjections() {
	if !aidevEnabled() {
		return
	}
	cutoff := time.Now().Add(-aidevTTL)
	pendingInjections.Lock()
	for nonce, pending := range pendingInjections.m {
		if pending.created.Before(cutoff) {
			delete(pendingInjections.m, nonce)
		}
	}
	pendingInjections.Unlock()
}

func aidevAuthOK(r *http.Request) bool {
	if !aidevEnabled() {
		return false
	}
	secret := aidevSecretValue()
	if secret == "" {
		return false
	}
	return r.Header.Get("X-AIDEV-KEY") == secret
}

func addTapListener() *tapListener {
	l := &tapListener{ch: make(chan tapEvent, 64)}
	aidevTaps.Lock()
	aidevTaps.list[l] = struct{}{}
	aidevTaps.Unlock()
	return l
}

func removeTapListener(l *tapListener) {
	aidevTaps.Lock()
	delete(aidevTaps.list, l)
	aidevTaps.Unlock()
	close(l.ch)
}

func emitTapEvent(evt tapEvent) {
	if !aidevEnabled() {
		return
	}
	aidevTaps.Lock()
	for l := range aidevTaps.list {
		select {
		case l.ch <- evt:
		default:
		}
	}
	aidevTaps.Unlock()
}

func markAidevHello() {
	aidevHello.Lock()
	select {
	case aidevHello.ch <- struct{}{}:
	default:
	}
	aidevHello.Unlock()
}

func markAidevReady() {
	aidevReady.Lock()
	select {
	case aidevReady.ch <- struct{}{}:
	default:
	}
	aidevReady.Unlock()
}

func markAidevStart() {
	aidevStart.Lock()
	select {
	case aidevStart.ch <- struct{}{}:
	default:
	}
	aidevStart.Unlock()
}

func waitForAidevHello(timeout time.Duration) error {
	select {
	case <-aidevHello.ch:
		return nil
	case <-time.After(timeout):
		return errors.New("aidev hello timeout")
	}
}

func waitForAidevStart(timeout time.Duration) error {
	select {
	case <-aidevStart.ch:
		return nil
	case <-time.After(timeout):
		return errors.New("aidev start timeout")
	}
}

func findAidevMCPBinary() (string, error) {
	candidates := []string{
		filepath.Join(installPath, "gopherbot-mcp"),
		filepath.Join(installPath, "cmd", "gopherbot-mcp", "gopherbot-mcp"),
	}
	for _, candidate := range candidates {
		if st, err := os.Stat(candidate); err == nil && !st.IsDir() {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("gopherbot-mcp binary not found under %s", installPath)
}

func aidevStartCommand(listenAddr string, bin string) string {
	url := "http://" + listenAddr
	cmd := fmt.Sprintf("%s --aidev-url %s --aidev-key %s", bin, url, aidev.cfg.secret)
	if cfg := os.Getenv("GOPHER_AIDEV_MCP_CONFIG"); cfg != "" {
		cmd += " --config " + cfg
	}
	if proto := os.Getenv("GOPHER_AIDEV_MCP_PROTOCOL"); proto != "" {
		cmd += " --protocol " + proto
	}
	return cmd
}

func aidevStreamHandler(w http.ResponseWriter, r *http.Request) {
	if !aidevAuthOK(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	l := addTapListener()
	defer removeTapListener(l)
	for {
		select {
		case <-r.Context().Done():
			return
		case evt, ok := <-l.ch:
			if !ok {
				return
			}
			data, err := json.Marshal(evt)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

func aidevInjectHandler(w http.ResponseWriter, r *http.Request) {
	if !aidevAuthOK(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req aidevInjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.User == "" || req.UserID == "" || req.Message == "" {
		http.Error(w, "missing user, user_id, or message", http.StatusBadRequest)
		return
	}
	botStdErrLogger.Printf("AIDEV inject: user=%s user_id=%s channel=%s direct=%t message=%q", req.User, req.UserID, req.Channel, req.Direct, req.Message)
	if !req.Direct && req.Channel == "" {
		http.Error(w, "missing channel", http.StatusBadRequest)
		return
	}
	aidevInitialized.RLock()
	ready := aidevInitialized.ready
	aidevInitialized.RUnlock()
	if !ready {
		http.Error(w, "robot not initialized", http.StatusConflict)
		return
	}
	if interfaces.Connector == nil {
		http.Error(w, "connector not ready", http.StatusServiceUnavailable)
		return
	}
	nonce, err := newAidevNonce()
	if err != nil {
		http.Error(w, "nonce error", http.StatusInternalServerError)
		return
	}
	p := pendingInjection{
		nonce:   nonce,
		user:    req.User,
		userID:  req.UserID,
		channel: req.Channel,
		thread:  req.Thread,
	}
	if err := enqueueInjection(p); err != nil {
		http.Error(w, "enqueue error", http.StatusInternalServerError)
		return
	}
	msg := aidevPrefix(nonce, req.User) + req.Message
	channel := req.Channel
	thread := req.Thread
	currentCfg.RLock()
	format := currentCfg.defaultMessageFormat
	protocol := currentCfg.protocol
	currentCfg.RUnlock()
	var ret robot.RetVal
	msgObj := &robot.ConnectorMessage{
		Protocol:      protocol,
		UserID:        req.UserID,
		UserName:      req.User,
		ChannelName:   req.Channel,
		ThreadID:      req.Thread,
		HiddenMessage: req.Hidden,
		DirectMessage: req.Direct,
	}
	if req.Direct {
		ret = interfaces.SendProtocolUserMessage(bracket(req.UserID), msg, format, msgObj)
	} else {
		ret = interfaces.SendProtocolChannelThreadMessage(channel, thread, msg, format, msgObj)
	}
	if ret != robot.Ok {
		consumeInjection(nonce)
		http.Error(w, fmt.Sprintf("send error: %d", ret), http.StatusBadRequest)
		return
	}
	if protocol == "terminal" {
		h := handler{}
		h.IncomingMessage(&robot.ConnectorMessage{
			Protocol:      protocol,
			UserName:      "aidev",
			UserID:        "aidev",
			ChannelName:   req.Channel,
			ThreadID:      req.Thread,
			DirectMessage: req.Direct,
			SelfMessage:   true,
			MessageText:   msg,
		})
	}
	w.WriteHeader(http.StatusAccepted)
}

func aidevControlHandler(w http.ResponseWriter, r *http.Request) {
	if !aidevAuthOK(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req aidevControlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	switch req.Action {
	case "hello":
		markAidevHello()
		Log(robot.Info, "AIDEV MCP hello received")
		botStdErrLogger.Printf("AIDEV control: hello")
		w.WriteHeader(http.StatusOK)
	case "ready":
		markAidevReady()
		Log(robot.Info, "AIDEV MCP ready received")
		botStdErrLogger.Printf("AIDEV control: ready")
		w.WriteHeader(http.StatusOK)
	case "start":
		markAidevStart()
		Log(robot.Info, "AIDEV MCP start received")
		botStdErrLogger.Printf("AIDEV control: start")
		w.WriteHeader(http.StatusAccepted)
	case "exit":
		go stop()
		w.WriteHeader(http.StatusAccepted)
	case "force_exit":
		w.WriteHeader(http.StatusAccepted)
		go func() {
			time.Sleep(100 * time.Millisecond)
			if p, err := os.FindProcess(os.Getpid()); err == nil {
				_ = p.Signal(unix.SIGUSR1)
			}
		}()
	case "stack_dump":
		buf := make([]byte, 65536)
		n := runtime.Stack(buf, true)
		log.Printf("%s", buf[:n])
		w.WriteHeader(http.StatusOK)
	default:
		http.Error(w, "unknown action", http.StatusBadRequest)
	}
}
