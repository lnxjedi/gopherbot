package bot

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
)

const (
	codexSessionQueueSize      = 32
	codexDefaultModel          = "gpt-5.1-codex"
	codexDefaultApprovalPolicy = "never"
	codexStartTimeout          = 20 * time.Second
	codexTurnTimeout           = 10 * time.Minute
)

type codexSessionKey struct {
	Protocol string
	Channel  string
	ThreadID string
}

type codexSessionInput struct {
	User string
	Text string
}

type codexSession struct {
	key            codexSessionKey
	owner          string
	workspaceDir   string
	workspaceLabel string
	codexHome      string
	model          string
	approvalPolicy string
	networkAccess  bool

	threadID string

	cmd       *exec.Cmd
	rpc       *codexRPCClient
	inputCh   chan codexSessionInput
	stopCh    chan struct{}
	doneCh    chan struct{}
	stopOnce  sync.Once
	startedAt time.Time
}

type codexSessionRegistry struct {
	sync.Mutex
	sessions map[codexSessionKey]*codexSession
	starting map[codexSessionKey]struct{}
}

var codexSessions = codexSessionRegistry{
	sessions: map[codexSessionKey]*codexSession{},
	starting: map[codexSessionKey]struct{}{},
}

type codexRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type codexRPCEnvelope struct {
	JSONRPC string           `json:"jsonrpc,omitempty"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Method  string           `json:"method,omitempty"`
	Params  json.RawMessage  `json:"params,omitempty"`
	Result  json.RawMessage  `json:"result,omitempty"`
	Error   *codexRPCError   `json:"error,omitempty"`
}

type codexRPCEvent struct {
	Method string
	Params json.RawMessage
	ID     *json.RawMessage
}

type codexRPCResult struct {
	Result json.RawMessage
	Err    error
}

type codexRPCClient struct {
	enc *json.Encoder
	dec *json.Decoder
	in  io.Closer

	writeMu sync.Mutex
	pending map[int64]chan codexRPCResult
	mu      sync.Mutex
	nextID  int64

	events chan codexRPCEvent
	closed chan struct{}
	once   sync.Once
}

func codexStartSessionCommand(r Robot, args []string) {
	key, ok := codexSessionKeyFromRobot(r)
	if !ok {
		r.Say("start-codex must be run in a channel/thread context")
		return
	}
	codexSessions.Lock()
	if _, exists := codexSessions.sessions[key]; exists {
		codexSessions.Unlock()
		r.Say("A Codex session is already active in this thread")
		return
	}
	if _, starting := codexSessions.starting[key]; starting {
		codexSessions.Unlock()
		r.Say("A Codex session is already starting in this thread")
		return
	}
	codexSessions.starting[key] = struct{}{}
	codexSessions.Unlock()
	defer func() {
		codexSessions.Lock()
		delete(codexSessions.starting, key)
		codexSessions.Unlock()
	}()

	user := codexCanonicalUser(r.User)
	if user == "" {
		r.Say("Unable to determine your user identity for Codex session startup")
		return
	}
	_, exists, authRecord, ret := codexGetAuthRecord(r, user)
	if ret != robot.Ok {
		r.Say("Unable to read your Codex credentials (%s)", ret)
		return
	}
	if !exists || strings.TrimSpace(authRecord.AuthJSON) == "" {
		r.Say("No linked Codex credentials found; run '/%s link-codex' first", r.GetBotAttribute("name"))
		return
	}

	workspaceArg := "."
	if len(args) > 0 {
		workspaceArg = strings.TrimSpace(args[0])
	}
	workspaceDir, workspaceLabel, err := codexResolveWorkspaceDir(workspaceArg)
	if err != nil {
		r.Say("Invalid start-codex directory '%s': %v", workspaceArg, err)
		return
	}
	sessionHome, err := codexCreateSessionHome(key, user)
	if err != nil {
		r.Say("Unable to prepare Codex session storage: %v", err)
		return
	}
	if err := codexWriteConfigFile(sessionHome, codexAuthStoreFileSetting); err != nil {
		codexRemovePath(sessionHome)
		r.Say("Unable to initialize Codex config: %v", err)
		return
	}
	if err := codexWriteAuthJSON(sessionHome, authRecord.AuthJSON); err != nil {
		codexRemovePath(sessionHome)
		r.Say("Unable to initialize Codex auth: %v", err)
		return
	}

	session := &codexSession{
		key:            key,
		owner:          user,
		workspaceDir:   workspaceDir,
		workspaceLabel: workspaceLabel,
		codexHome:      sessionHome,
		model:          codexModelFromEnv(),
		approvalPolicy: codexApprovalPolicyFromEnv(),
		networkAccess:  codexNetworkAccessFromEnv(),
		inputCh:        make(chan codexSessionInput, codexSessionQueueSize),
		stopCh:         make(chan struct{}),
		doneCh:         make(chan struct{}),
		startedAt:      time.Now().UTC(),
	}
	if err := session.start(); err != nil {
		codexRemovePath(sessionHome)
		r.Say("Unable to start Codex app-server session: %v", err)
		return
	}

	codexSessions.Lock()
	codexSessions.sessions[key] = session
	codexSessions.Unlock()

	if !r.Subscribe() {
		session.stop("subscription failed")
		<-session.doneCh
		r.Say("I couldn't subscribe to this thread for Codex session routing")
		return
	}

	r.SayThread("Codex session started for '%s' in `%s`. Continue in this thread. Type `end-session` to stop.", user, workspaceLabel)
}

func codexEndSessionCommand(r Robot) {
	key, ok := codexSessionKeyFromRobot(r)
	if !ok {
		r.Say("No thread context available to end a Codex session")
		return
	}
	session, exists := codexLookupSession(key)
	if !exists {
		r.Say("No active Codex session found in this thread")
		return
	}
	if !codexUserMayControlSession(r, session) {
		r.Say("Only '%s' or a bot admin can control this Codex session", session.owner)
		return
	}
	session.stop("end-session command")
	<-session.doneCh
	_ = r.Unsubscribe()
	r.Say("Codex session ended")
}

func codexHandleSubscribedMessage(r Robot, args []string) {
	key, ok := codexSessionKeyFromRobot(r)
	if !ok {
		return
	}
	session, exists := codexLookupSession(key)
	if !exists {
		return
	}
	if !codexUserMayControlSession(r, session) {
		return
	}
	if len(args) == 0 {
		return
	}
	text := strings.TrimSpace(args[0])
	if text == "" {
		return
	}
	switch strings.ToLower(text) {
	case "end-session", ";end-session", "codex end-session", "codex end":
		session.stop("thread end-session command")
		<-session.doneCh
		_ = r.Unsubscribe()
		r.Say("Codex session ended")
		return
	}
	select {
	case session.inputCh <- codexSessionInput{User: codexCanonicalUser(r.User), Text: text}:
	default:
		r.Reply("Codex session queue is full; please wait for current responses")
	}
}

func codexStatusWithSession(r Robot) {
	user := codexCanonicalUser(r.User)
	if user == "" {
		r.Say("Unable to determine your user identity")
		return
	}
	_, exists, record, ret := codexGetAuthRecord(r, user)
	if ret != robot.Ok {
		r.Say("Unable to retrieve Codex auth status (%s)", ret)
		return
	}
	authStatus := "not linked"
	if exists && strings.TrimSpace(record.AuthJSON) != "" {
		authStatus = "linked"
	}
	key, ok := codexSessionKeyFromRobot(r)
	if !ok {
		r.Say("Codex auth for '%s': %s", user, authStatus)
		return
	}
	session, active := codexLookupSession(key)
	if !active {
		r.Say("Codex auth for '%s': %s; session in this thread: inactive", user, authStatus)
		return
	}
	r.Say("Codex auth for '%s': %s; session in this thread: active (owner=%s, dir=%s)", user, authStatus, session.owner, session.workspaceLabel)
}

func codexLookupSession(key codexSessionKey) (*codexSession, bool) {
	codexSessions.Lock()
	defer codexSessions.Unlock()
	session, ok := codexSessions.sessions[key]
	return session, ok
}

func codexSessionKeyFromRobot(r Robot) (codexSessionKey, bool) {
	protocol := normalizeProtocolName(protocolFromIncoming(r.Incoming, r.Protocol))
	channel := strings.TrimSpace(r.Channel)
	threadID := ""
	if r.Incoming != nil {
		threadID = strings.TrimSpace(r.Incoming.ThreadID)
	}
	if protocol == "" || channel == "" || threadID == "" {
		return codexSessionKey{}, false
	}
	return codexSessionKey{
		Protocol: protocol,
		Channel:  channel,
		ThreadID: threadID,
	}, true
}

func codexUserMayControlSession(r Robot, session *codexSession) bool {
	user := codexCanonicalUser(r.User)
	if user == session.owner {
		return true
	}
	return r.CheckAdmin()
}

func codexModelFromEnv() string {
	model := strings.TrimSpace(os.Getenv("GOPHER_CODEX_MODEL"))
	if model == "" {
		return codexDefaultModel
	}
	return model
}

func codexApprovalPolicyFromEnv() string {
	policy := strings.TrimSpace(os.Getenv("GOPHER_CODEX_APPROVAL_POLICY"))
	if policy == "" {
		return codexDefaultApprovalPolicy
	}
	return policy
}

func codexNetworkAccessFromEnv() bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv("GOPHER_CODEX_NETWORK")))
	switch value {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func codexCreateSessionHome(key codexSessionKey, user string) (string, error) {
	root := filepath.Join(homePath, codexDefaultLinkStateDir, "sessions")
	parts := []string{
		codexSanitizePathPart(key.Protocol),
		codexSanitizePathPart(key.Channel),
		codexSanitizePathPart(key.ThreadID),
		codexSanitizePathPart(user),
		strconv.FormatInt(time.Now().UnixNano(), 10),
	}
	dir := filepath.Join(append([]string{root}, parts...)...)
	if err := codexPrivilegedFS("creating codex session directory", func() error {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return err
		}
		return os.Chmod(dir, 0700)
	}); err != nil {
		return "", err
	}
	return dir, nil
}

func codexWriteAuthJSON(codexHome, authJSON string) error {
	path := filepath.Join(codexHome, codexAuthFileName)
	return codexPrivilegedFS("writing codex auth json", func() error {
		return os.WriteFile(path, []byte(strings.TrimSpace(authJSON)+"\n"), 0600)
	})
}

func codexResolveWorkspaceDir(input string) (string, string, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		trimmed = "."
	}
	if filepath.IsAbs(trimmed) {
		return "", "", errors.New("path must be relative to robot startup directory")
	}
	cleanRel := filepath.Clean(trimmed)
	target := filepath.Clean(filepath.Join(homePath, cleanRel))
	baseEval := filepath.Clean(homePath)
	targetEval := target
	if err := codexPrivilegedFS("resolving codex workspace path", func() error {
		baseResolved, err := filepath.EvalSymlinks(baseEval)
		if err == nil {
			baseEval = filepath.Clean(baseResolved)
		}
		resolvedTarget, err := filepath.EvalSymlinks(target)
		if err == nil {
			targetEval = filepath.Clean(resolvedTarget)
		} else {
			targetEval = filepath.Clean(target)
		}
		info, err := os.Stat(targetEval)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return fmt.Errorf("'%s' is not a directory", input)
		}
		return nil
	}); err != nil {
		return "", "", err
	}
	if !codexPathWithin(baseEval, targetEval) {
		return "", "", errors.New("directory escapes robot startup root")
	}
	return targetEval, cleanRel, nil
}

func codexPathWithin(base, target string) bool {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".."
}

func (s *codexSession) start() error {
	cmd := exec.Command(codexBinaryPath(), "app-server", "--listen", "stdio://")
	cmd.Env = codexCommandEnv(s.codexHome)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("creating app-server stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("creating app-server stderr pipe: %w", err)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("creating app-server stdin pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting codex app-server: %w", err)
	}
	s.cmd = cmd
	s.rpc = newCodexRPCClient(stdin, stdout)

	go s.consumeStderr(stderr)
	if err := s.initializeRPCSession(); err != nil {
		s.stop("initialization failed")
		<-s.doneCh
		return err
	}
	go s.run()
	return nil
}

func (s *codexSession) initializeRPCSession() error {
	ctx, cancel := context.WithTimeout(context.Background(), codexStartTimeout)
	defer cancel()
	_, err := s.rpc.Call(ctx, "initialize", map[string]interface{}{
		"clientInfo": map[string]interface{}{
			"name":    "gopherbot-codex",
			"title":   "Gopherbot Codex Session",
			"version": botVersion.Version,
		},
		"capabilities": map[string]interface{}{
			"agentCommands": map[string]interface{}{},
		},
	})
	if err != nil {
		return fmt.Errorf("initialize failed: %w", err)
	}
	if err := s.rpc.Notify("initialized", map[string]interface{}{}); err != nil {
		return fmt.Errorf("initialized notify failed: %w", err)
	}

	accountRaw, err := s.rpc.Call(ctx, "account/read", map[string]interface{}{"refreshToken": false})
	if err == nil {
		needsAuth := codexJSONPathBool(accountRaw, "requiresOpenaiAuth")
		if needsAuth && codexJSONPath(accountRaw, "account") == nil {
			return errors.New("codex account is not authenticated")
		}
	}

	threadStartRaw, err := s.rpc.Call(ctx, "thread/start", map[string]interface{}{
		"cwd":            s.workspaceDir,
		"model":          s.model,
		"approvalPolicy": s.approvalPolicy,
		"sandboxPolicy": map[string]interface{}{
			"networkAccess": s.networkAccess,
			"writableRoots": []string{s.workspaceDir},
		},
	})
	if err != nil {
		return fmt.Errorf("thread/start failed: %w", err)
	}

	threadID := codexJSONPathString(threadStartRaw, "thread", "id")
	if threadID == "" {
		threadID = codexJSONPathString(threadStartRaw, "id")
	}
	if threadID == "" {
		return errors.New("thread/start did not return a thread id")
	}
	s.threadID = threadID
	return nil
}

func (s *codexSession) run() {
	defer close(s.doneCh)
	defer s.cleanup()
	for {
		select {
		case <-s.stopCh:
			return
		case input := <-s.inputCh:
			if strings.TrimSpace(input.Text) == "" {
				continue
			}
			if err := s.runTurn(input); err != nil {
				codexSendThreadMessage(s.key, fmt.Sprintf("Codex turn failed: %v", err))
			}
		}
	}
}

func (s *codexSession) runTurn(input codexSessionInput) error {
	ctx, cancel := context.WithTimeout(context.Background(), codexTurnTimeout)
	defer cancel()

	turnRaw, err := s.rpc.Call(ctx, "turn/start", map[string]interface{}{
		"threadId":       s.threadID,
		"input":          input.Text,
		"cwd":            s.workspaceDir,
		"model":          s.model,
		"approvalPolicy": s.approvalPolicy,
		"sandboxPolicy": map[string]interface{}{
			"networkAccess": s.networkAccess,
			"writableRoots": []string{s.workspaceDir},
		},
	})
	if err != nil {
		return err
	}
	turnID := codexJSONPathString(turnRaw, "turn", "id")
	if turnID == "" {
		turnID = codexJSONPathString(turnRaw, "id")
	}
	response, err := s.waitForTurnCompletion(ctx, turnID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(response) != "" {
		codexSendThreadMessage(s.key, response)
	}
	return nil
}

func (s *codexSession) waitForTurnCompletion(ctx context.Context, turnID string) (string, error) {
	var deltaBuilder strings.Builder
	var finalText string
	for {
		select {
		case <-ctx.Done():
			return strings.TrimSpace(finalTextOrDelta(finalText, deltaBuilder.String())), ctx.Err()
		case <-s.stopCh:
			return "", errors.New("session stopped")
		case ev, ok := <-s.rpc.events:
			if !ok {
				return strings.TrimSpace(finalTextOrDelta(finalText, deltaBuilder.String())), errors.New("codex rpc stream closed")
			}
			if ev.ID != nil && codexIsApprovalRequest(ev.Method) {
				_ = s.rpc.Reply(ev.ID, map[string]interface{}{"decision": "decline"})
				continue
			}
			switch ev.Method {
			case "item/agentMessage/delta":
				if delta := codexExtractEventText(ev.Params); delta != "" {
					deltaBuilder.WriteString(delta)
				}
			case "item/completed":
				if text := codexExtractCompletedAgentText(ev.Params); text != "" {
					finalText = text
				}
			case "turn/completed":
				completedTurnID := codexJSONPathString(ev.Params, "turn", "id")
				if completedTurnID == "" {
					completedTurnID = codexJSONPathString(ev.Params, "turnId")
				}
				if turnID == "" || completedTurnID == "" || completedTurnID == turnID {
					return strings.TrimSpace(finalTextOrDelta(finalText, deltaBuilder.String())), nil
				}
			case "error":
				errText := codexExtractEventText(ev.Params)
				if errText == "" {
					errText = "unknown codex error"
				}
				return strings.TrimSpace(finalTextOrDelta(finalText, deltaBuilder.String())), errors.New(errText)
			}
		}
	}
}

func finalTextOrDelta(finalText, delta string) string {
	if strings.TrimSpace(finalText) != "" {
		return finalText
	}
	return delta
}

func (s *codexSession) consumeStderr(reader io.Reader) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 1024), 1024*1024)
	for scanner.Scan() {
		line := codexNormalizeOutputLine(scanner.Text())
		if line == "" {
			continue
		}
		Log(robot.Debug, "codex app-server stderr (%s/%s/%s): %s", s.key.Protocol, s.key.Channel, s.key.ThreadID, line)
	}
}

func (s *codexSession) stop(reason string) {
	s.stopOnce.Do(func() {
		Log(robot.Info, "Stopping codex session %s/%s/%s (%s)", s.key.Protocol, s.key.Channel, s.key.ThreadID, reason)
		close(s.stopCh)
	})
}

func (s *codexSession) cleanup() {
	if s.rpc != nil {
		s.rpc.Close()
	}
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
		_ = s.cmd.Wait()
	}
	codexRemovePath(s.codexHome)
	codexSessions.Lock()
	delete(codexSessions.sessions, s.key)
	codexSessions.Unlock()
}

func codexStopAllSessions() {
	codexSessions.Lock()
	sessions := make([]*codexSession, 0, len(codexSessions.sessions))
	for _, session := range codexSessions.sessions {
		sessions = append(sessions, session)
	}
	codexSessions.Unlock()
	for _, session := range sessions {
		session.stop("robot shutdown")
	}
	for _, session := range sessions {
		<-session.doneCh
	}
}

func codexSendThreadMessage(key codexSessionKey, msg string) {
	normalized := codexNormalizeForBasicMarkdown(msg)
	trimmed := strings.TrimSpace(normalized)
	if trimmed == "" {
		return
	}
	conn := getConnectorForProtocol(key.Protocol)
	if conn == nil {
		Log(robot.Error, "No active connector for codex session protocol '%s'", key.Protocol)
		return
	}
	msgObject := &robot.ConnectorMessage{Protocol: key.Protocol}
	ret := conn.SendProtocolChannelThreadMessage(key.Channel, key.ThreadID, trimmed, robot.BasicMarkdown, msgObject)
	if ret != robot.Ok {
		Log(robot.Error, "Sending codex session message failed (%s)", ret)
	}
}

func newCodexRPCClient(in io.WriteCloser, out io.Reader) *codexRPCClient {
	client := &codexRPCClient{
		enc:     json.NewEncoder(in),
		dec:     json.NewDecoder(out),
		in:      in,
		pending: map[int64]chan codexRPCResult{},
		events:  make(chan codexRPCEvent, 512),
		closed:  make(chan struct{}),
	}
	go client.readLoop()
	return client
}

func (c *codexRPCClient) Close() {
	c.once.Do(func() {
		close(c.closed)
		if c.in != nil {
			_ = c.in.Close()
		}
		c.failPending(errors.New("rpc client closed"))
	})
}

func (c *codexRPCClient) readLoop() {
	defer close(c.events)
	for {
		var env codexRPCEnvelope
		if err := c.dec.Decode(&env); err != nil {
			c.failPending(fmt.Errorf("rpc decode failed: %w", err))
			return
		}
		if env.Method != "" {
			event := codexRPCEvent{
				Method: env.Method,
				Params: env.Params,
				ID:     env.ID,
			}
			select {
			case c.events <- event:
			case <-c.closed:
				return
			}
			continue
		}
		if env.ID == nil {
			continue
		}
		id, ok := codexParseJSONID(env.ID)
		if !ok {
			continue
		}
		c.mu.Lock()
		ch, exists := c.pending[id]
		if exists {
			delete(c.pending, id)
		}
		c.mu.Unlock()
		if !exists {
			continue
		}
		if env.Error != nil {
			ch <- codexRPCResult{Err: fmt.Errorf("rpc error %d: %s", env.Error.Code, env.Error.Message)}
		} else {
			ch <- codexRPCResult{Result: env.Result}
		}
		close(ch)
	}
}

func (c *codexRPCClient) failPending(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for id, ch := range c.pending {
		delete(c.pending, id)
		ch <- codexRPCResult{Err: err}
		close(ch)
	}
}

func (c *codexRPCClient) Call(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	id := atomic.AddInt64(&c.nextID, 1)
	env := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
		"params":  params,
	}
	replyCh := make(chan codexRPCResult, 1)
	c.mu.Lock()
	c.pending[id] = replyCh
	c.mu.Unlock()

	c.writeMu.Lock()
	err := c.enc.Encode(env)
	c.writeMu.Unlock()
	if err != nil {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		close(replyCh)
		return nil, err
	}

	select {
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, ctx.Err()
	case result, ok := <-replyCh:
		if !ok {
			return nil, errors.New("rpc response channel closed")
		}
		if result.Err != nil {
			return nil, result.Err
		}
		return result.Result, nil
	}
}

func (c *codexRPCClient) Notify(method string, params interface{}) error {
	env := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.enc.Encode(env)
}

func (c *codexRPCClient) Reply(id *json.RawMessage, result interface{}) error {
	if id == nil {
		return errors.New("missing request id")
	}
	raw := json.RawMessage(*id)
	env := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      raw,
		"result":  result,
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.enc.Encode(env)
}

func codexParseJSONID(raw *json.RawMessage) (int64, bool) {
	if raw == nil {
		return 0, false
	}
	var intID int64
	if err := json.Unmarshal(*raw, &intID); err == nil {
		return intID, true
	}
	var floatID float64
	if err := json.Unmarshal(*raw, &floatID); err == nil {
		return int64(floatID), true
	}
	var strID string
	if err := json.Unmarshal(*raw, &strID); err == nil {
		parsed, err := strconv.ParseInt(strID, 10, 64)
		if err == nil {
			return parsed, true
		}
	}
	return 0, false
}

func codexIsApprovalRequest(method string) bool {
	switch method {
	case "item/commandExecution/requestApproval", "item/applyPatchApproval/requestApproval":
		return true
	default:
		return false
	}
}

func codexExtractEventText(raw json.RawMessage) string {
	text := codexJSONPathText(raw, "delta", "text")
	if text != "" {
		return codexNormalizeForBasicMarkdown(text)
	}
	text = codexJSONPathText(raw, "delta")
	if text != "" {
		return codexNormalizeForBasicMarkdown(text)
	}
	text = codexJSONPathText(raw, "text")
	if text != "" {
		return codexNormalizeForBasicMarkdown(text)
	}
	text = codexJSONPathText(raw, "message")
	if text != "" {
		return codexNormalizeForBasicMarkdown(text)
	}
	text = codexExtractTextFromContent(codexJSONPath(raw, "content"))
	if text != "" {
		return codexNormalizeForBasicMarkdown(text)
	}
	return codexNormalizeForBasicMarkdown(codexJSONPathText(raw, "content", "text"))
}

func codexExtractCompletedAgentText(raw json.RawMessage) string {
	itemType := strings.ToLower(codexJSONPathString(raw, "item", "type"))
	if itemType != "" && itemType != "agentmessage" && itemType != "agent_message" {
		return ""
	}
	if text := codexExtractTextFromItemOutput(codexJSONPath(raw, "item", "output")); text != "" {
		return codexNormalizeForBasicMarkdown(text)
	}
	if text := codexExtractTextFromContent(codexJSONPath(raw, "item", "content")); text != "" {
		return codexNormalizeForBasicMarkdown(text)
	}
	if text := codexJSONPathText(raw, "item", "text"); text != "" {
		return codexNormalizeForBasicMarkdown(text)
	}
	if text := codexJSONPathText(raw, "item", "content", "text"); text != "" {
		return codexNormalizeForBasicMarkdown(text)
	}
	if text := codexJSONPathText(raw, "item", "message"); text != "" {
		return codexNormalizeForBasicMarkdown(text)
	}
	return ""
}

func codexJSONPath(raw json.RawMessage, path ...string) interface{} {
	if len(raw) == 0 {
		return nil
	}
	var payload interface{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil
	}
	current := payload
	for _, part := range path {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil
		}
		next, exists := m[part]
		if !exists {
			return nil
		}
		current = next
	}
	return current
}

func codexJSONPathString(raw json.RawMessage, path ...string) string {
	value := codexJSONPath(raw, path...)
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case json.Number:
		return v.String()
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	default:
		return ""
	}
}

func codexJSONPathText(raw json.RawMessage, path ...string) string {
	value := codexJSONPath(raw, path...)
	if value == nil {
		return ""
	}
	return codexValueToString(value)
}

func codexJSONPathBool(raw json.RawMessage, path ...string) bool {
	value := codexJSONPath(raw, path...)
	if value == nil {
		return false
	}
	switch v := value.(type) {
	case bool:
		return v
	case string:
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "1", "true", "yes", "on":
			return true
		}
	}
	return false
}

func codexValueToString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case json.Number:
		return v.String()
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	default:
		return ""
	}
}

func codexExtractTextFromItemOutput(value interface{}) string {
	items, ok := value.([]interface{})
	if !ok {
		return ""
	}
	var out strings.Builder
	for _, item := range items {
		part, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		partType := strings.ToLower(strings.TrimSpace(codexValueToString(part["type"])))
		if partType != "" && partType != "output_text" && partType != "outputtext" && partType != "text" {
			continue
		}
		text := codexValueToString(part["text"])
		if text == "" {
			text = codexExtractTextFromContent(part["content"])
		}
		if text != "" {
			out.WriteString(text)
		}
	}
	return out.String()
}

func codexExtractTextFromContent(value interface{}) string {
	switch typed := value.(type) {
	case map[string]interface{}:
		if text := codexValueToString(typed["text"]); text != "" {
			return text
		}
	case []interface{}:
		var out strings.Builder
		for _, item := range typed {
			part, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			partType := strings.ToLower(strings.TrimSpace(codexValueToString(part["type"])))
			if partType != "" && partType != "text" && partType != "output_text" && partType != "outputtext" {
				continue
			}
			if text := codexValueToString(part["text"]); text != "" {
				out.WriteString(text)
			}
		}
		return out.String()
	}
	return ""
}

func codexNormalizeForBasicMarkdown(text string) string {
	if text == "" {
		return ""
	}
	normalized := strings.ReplaceAll(text, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	normalized = codexStripANSISequences(normalized)
	return codexFilterControlRunes(normalized)
}

func codexStripANSISequences(text string) string {
	if text == "" || !strings.ContainsRune(text, '\x1b') {
		return text
	}
	var out strings.Builder
	out.Grow(len(text))
	for i := 0; i < len(text); {
		if text[i] != 0x1b {
			out.WriteByte(text[i])
			i++
			continue
		}
		if i+1 >= len(text) {
			i++
			continue
		}
		switch text[i+1] {
		case '[':
			i += 2
			for i < len(text) {
				if text[i] >= 0x40 && text[i] <= 0x7e {
					i++
					break
				}
				i++
			}
		case ']':
			i += 2
			for i < len(text) {
				if text[i] == 0x07 {
					i++
					break
				}
				if text[i] == 0x1b && i+1 < len(text) && text[i+1] == '\\' {
					i += 2
					break
				}
				i++
			}
		default:
			i += 2
		}
	}
	return out.String()
}

func codexFilterControlRunes(text string) string {
	var out strings.Builder
	out.Grow(len(text))
	for _, r := range text {
		if r == '\n' || r == '\t' {
			out.WriteRune(r)
			continue
		}
		if r < 0x20 || r == 0x7f {
			continue
		}
		out.WriteRune(r)
	}
	return out.String()
}
