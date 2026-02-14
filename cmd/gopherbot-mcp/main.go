package main

import (
	"bytes"
	crand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	mcpProtocolVersion = "2024-11-05"
	mcpServerName      = "gopherbot-mcp"
	mcpServerVersion   = "0.1.0"
	stateFileName      = ".gopherbot-mcp-state.json"
)

type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type mcpTool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
}

type toolsCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

type processState struct {
	PID          int      `json:"pid"`
	RobotDir     string   `json:"robot_dir"`
	GopherbotBin string   `json:"gopherbot_binary"`
	AuthToken    string   `json:"auth_token"`
	LogPath      string   `json:"log_path"`
	StartedAt    string   `json:"started_at"`
	CommandArgs  []string `json:"command_args"`
}

type mcpServer struct {
	rootDir string
	mu      sync.Mutex
}

func main() {
	rootDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to get cwd: %v\n", err)
		os.Exit(1)
	}
	s := &mcpServer{rootDir: rootDir}
	if err := s.serve(); err != nil {
		fmt.Fprintf(os.Stderr, "mcp server error: %v\n", err)
		os.Exit(1)
	}
}

func (s *mcpServer) serve() error {
	decoder := json.NewDecoder(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)

	for {
		var req jsonRPCRequest
		if err := decoder.Decode(&req); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		if req.JSONRPC == "" {
			req.JSONRPC = "2.0"
		}
		if !hasRequestID(req.ID) {
			s.handleNotification(req)
			continue
		}
		resp := s.handleRequest(req)
		if err := encoder.Encode(resp); err != nil {
			return err
		}
	}
}

func (s *mcpServer) handleNotification(req jsonRPCRequest) {
	// Currently no-op; clients may send notifications/initialized.
	_ = req
}

func (s *mcpServer) handleRequest(req jsonRPCRequest) jsonRPCResponse {
	switch req.Method {
	case "initialize":
		return jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"protocolVersion": mcpProtocolVersion,
				"capabilities": map[string]interface{}{
					"tools": map[string]interface{}{
						"listChanged": false,
					},
				},
				"serverInfo": map[string]interface{}{
					"name":    mcpServerName,
					"version": mcpServerVersion,
				},
			},
		}
	case "ping":
		return jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  map[string]interface{}{},
		}
	case "tools/list":
		return jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"tools": s.tools(),
			},
		}
	case "tools/call":
		var params toolsCallParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return jsonRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &jsonRPCError{
					Code:    -32602,
					Message: fmt.Sprintf("invalid tools/call params: %v", err),
				},
			}
		}
		result := s.callTool(params)
		return jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  result,
		}
	default:
		return jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &jsonRPCError{
				Code:    -32601,
				Message: fmt.Sprintf("method not found: %s", req.Method),
			},
		}
	}
}

func (s *mcpServer) tools() []mcpTool {
	return []mcpTool{
		{
			Name:        "start_robot",
			Description: "Start a gopherbot robot in a target directory with --aidev <auth_token>.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"robot_dir": map[string]interface{}{
						"type":        "string",
						"description": "Directory where the robot should be started.",
					},
					"auth_token": map[string]interface{}{
						"type":        "string",
						"description": "Shared auth token for aidev mode. Generated automatically when omitted.",
					},
					"gopherbot_binary": map[string]interface{}{
						"type":        "string",
						"description": "Path to gopherbot executable. Defaults to <mcp cwd>/gopherbot.",
					},
					"extra_args": map[string]interface{}{
						"type":        "array",
						"description": "Optional additional arguments passed before the 'run' command.",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
					"env": map[string]interface{}{
						"type":        "object",
						"description": "Optional environment variables to set for the gopherbot process.",
						"additionalProperties": map[string]interface{}{
							"type": "string",
						},
					},
				},
				"required": []string{"robot_dir"},
			},
		},
		{
			Name:        "stop_robot",
			Description: "Stop a gopherbot robot started via start_robot for a given directory.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"robot_dir": map[string]interface{}{
						"type":        "string",
						"description": "Directory where the robot is running.",
					},
				},
				"required": []string{"robot_dir"},
			},
		},
		{
			Name:        "robot_status",
			Description: "Report status for a robot directory including PID and .aiport when present.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"robot_dir": map[string]interface{}{
						"type":        "string",
						"description": "Directory where the robot is expected to run.",
					},
				},
				"required": []string{"robot_dir"},
			},
		},
		{
			Name:        "send_message",
			Description: "Inject a message into a running connector workflow (default protocol: ssh).",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"robot_dir": map[string]interface{}{
						"type":        "string",
						"description": "Directory where the robot is running.",
					},
					"as_user": map[string]interface{}{
						"type":        "string",
						"description": "Viewer/roster user to inject as (username or internal ID).",
					},
					"text": map[string]interface{}{
						"type":        "string",
						"description": "Message text to inject.",
					},
					"protocol": map[string]interface{}{
						"type":        "string",
						"description": "Protocol name, defaults to active primary protocol (typically ssh).",
					},
					"channel": map[string]interface{}{
						"type":        "string",
						"description": "Channel target, defaults to connector default channel.",
					},
					"thread_id": map[string]interface{}{
						"type":        "string",
						"description": "Optional thread ID for threaded follow-up.",
					},
					"hidden": map[string]interface{}{
						"type":        "boolean",
						"description": "Inject as a hidden/ephemeral command (viewer-scoped replies).",
					},
				},
				"required": []string{"robot_dir", "as_user", "text"},
			},
		},
		{
			Name:        "get_messages",
			Description: "Retrieve connector-visible messages with cursor-based polling and optional long-poll wait.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"robot_dir": map[string]interface{}{
						"type":        "string",
						"description": "Directory where the robot is running.",
					},
					"viewer": map[string]interface{}{
						"type":        "string",
						"description": "Viewer/roster user whose visibility scope applies.",
					},
					"protocol": map[string]interface{}{
						"type":        "string",
						"description": "Protocol name, defaults to active primary protocol (typically ssh).",
					},
					"all": map[string]interface{}{
						"type":        "boolean",
						"description": "When true, fetch visible buffer snapshot (ignores after_cursor).",
					},
					"after_cursor": map[string]interface{}{
						"type":        "integer",
						"description": "Return messages strictly after this cursor.",
					},
					"timeout_ms": map[string]interface{}{
						"type":        "integer",
						"description": "Long-poll wait timeout in ms (default 1400).",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum messages returned in one call (default connector value).",
					},
				},
				"required": []string{"robot_dir", "viewer"},
			},
		},
	}
}

func (s *mcpServer) callTool(params toolsCallParams) map[string]interface{} {
	args := params.Arguments
	if args == nil {
		args = map[string]interface{}{}
	}

	var (
		result map[string]interface{}
		err    error
	)
	switch params.Name {
	case "start_robot":
		result, err = s.toolStartRobot(args)
	case "stop_robot":
		result, err = s.toolStopRobot(args)
	case "robot_status":
		result, err = s.toolRobotStatus(args)
	case "send_message":
		result, err = s.toolSendMessage(args)
	case "get_messages":
		result, err = s.toolGetMessages(args)
	default:
		return toolErrorResult(fmt.Errorf("unknown tool: %s", params.Name))
	}
	if err != nil {
		return toolErrorResult(err)
	}
	return toolSuccessResult(result)
}

func (s *mcpServer) toolStartRobot(args map[string]interface{}) (map[string]interface{}, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	robotDir, err := requiredStringArg(args, "robot_dir")
	if err != nil {
		return nil, err
	}
	robotDir = resolvePath(s.rootDir, robotDir)
	info, err := os.Stat(robotDir)
	if err != nil {
		return nil, fmt.Errorf("checking robot_dir: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("robot_dir is not a directory: %s", robotDir)
	}

	statePath := filepath.Join(robotDir, stateFileName)
	prevState, exists, err := readStateFile(statePath)
	if err != nil {
		return nil, err
	}
	if exists && isProcessRunning(prevState.PID) {
		return nil, fmt.Errorf("robot already running for '%s' with pid %d", robotDir, prevState.PID)
	}

	gopherbotBin, err := optionalStringArg(args, "gopherbot_binary")
	if err != nil {
		return nil, err
	}
	if gopherbotBin == "" {
		gopherbotBin = filepath.Join(s.rootDir, "gopherbot")
	}
	gopherbotBin = resolvePath(s.rootDir, gopherbotBin)
	if err := verifyExecutableFile(gopherbotBin); err != nil {
		return nil, err
	}

	authToken, err := optionalStringArg(args, "auth_token")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(authToken) == "" {
		authToken, err = generateAuthToken()
		if err != nil {
			return nil, err
		}
	}

	extraArgs, err := optionalStringSliceArg(args, "extra_args")
	if err != nil {
		return nil, err
	}
	extraEnv, err := optionalStringMapArg(args, "env")
	if err != nil {
		return nil, err
	}

	logPath := filepath.Join(robotDir, "robot.log")
	logFile, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("opening robot log file '%s': %w", logPath, err)
	}
	defer logFile.Close()

	cmdArgs := []string{"--aidev", authToken}
	cmdArgs = append(cmdArgs, extraArgs...)
	cmdArgs = append(cmdArgs, "run")
	cmd := exec.Command(gopherbotBin, cmdArgs...)
	cmd.Dir = robotDir
	cmd.Env = append(os.Environ(), stringifyEnvMap(extraEnv)...)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting gopherbot: %w", err)
	}
	if waitForExit(cmd.Process.Pid, 600*time.Millisecond) {
		_ = cmd.Wait()
		return nil, fmt.Errorf("gopherbot exited shortly after start; check %s", logPath)
	}
	go func() {
		_ = cmd.Wait()
	}()

	state := processState{
		PID:          cmd.Process.Pid,
		RobotDir:     robotDir,
		GopherbotBin: gopherbotBin,
		AuthToken:    authToken,
		LogPath:      logPath,
		StartedAt:    time.Now().UTC().Format(time.RFC3339),
		CommandArgs:  cmdArgs,
	}
	if err := writeStateFile(statePath, state); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"running":          true,
		"pid":              state.PID,
		"robot_dir":        state.RobotDir,
		"gopherbot_binary": state.GopherbotBin,
		"auth_token":       state.AuthToken,
		"log_path":         state.LogPath,
		"state_file":       statePath,
		"command_args":     state.CommandArgs,
	}, nil
}

func (s *mcpServer) toolStopRobot(args map[string]interface{}) (map[string]interface{}, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	robotDir, err := requiredStringArg(args, "robot_dir")
	if err != nil {
		return nil, err
	}
	robotDir = resolvePath(s.rootDir, robotDir)
	statePath := filepath.Join(robotDir, stateFileName)
	state, exists, err := readStateFile(statePath)
	if err != nil {
		return nil, err
	}
	if !exists {
		return map[string]interface{}{
			"running":    false,
			"robot_dir":  robotDir,
			"state_file": statePath,
			"message":    "no state file found",
		}, nil
	}

	wasRunning := isProcessRunning(state.PID)
	if wasRunning {
		if err := syscall.Kill(state.PID, syscall.SIGTERM); err != nil && !errors.Is(err, syscall.ESRCH) {
			return nil, fmt.Errorf("sending SIGTERM to pid %d: %w", state.PID, err)
		}
	}

	stoppedGracefully := waitForExit(state.PID, 8*time.Second)
	forceKilled := false
	if !stoppedGracefully && isProcessRunning(state.PID) {
		forceKilled = true
		if err := syscall.Kill(state.PID, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
			return nil, fmt.Errorf("sending SIGKILL to pid %d: %w", state.PID, err)
		}
		_ = waitForExit(state.PID, 2*time.Second)
	}

	if err := removeStateFile(statePath); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"running":            false,
		"robot_dir":          robotDir,
		"pid":                state.PID,
		"was_running":        wasRunning,
		"stopped_gracefully": stoppedGracefully,
		"force_killed":       forceKilled,
		"state_file":         statePath,
	}, nil
}

func (s *mcpServer) toolRobotStatus(args map[string]interface{}) (map[string]interface{}, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	robotDir, err := requiredStringArg(args, "robot_dir")
	if err != nil {
		return nil, err
	}
	robotDir = resolvePath(s.rootDir, robotDir)
	statePath := filepath.Join(robotDir, stateFileName)
	state, exists, err := readStateFile(statePath)
	if err != nil {
		return nil, err
	}

	aiportPath := filepath.Join(robotDir, ".aiport")
	aiport := ""
	if data, err := os.ReadFile(aiportPath); err == nil {
		aiport = strings.TrimSpace(string(data))
	}

	if !exists {
		return map[string]interface{}{
			"running":     false,
			"robot_dir":   robotDir,
			"state_file":  statePath,
			"aiport_file": aiportPath,
			"aiport":      aiport,
		}, nil
	}

	return map[string]interface{}{
		"running":          isProcessRunning(state.PID),
		"robot_dir":        robotDir,
		"pid":              state.PID,
		"gopherbot_binary": state.GopherbotBin,
		"auth_token":       state.AuthToken,
		"log_path":         state.LogPath,
		"state_file":       statePath,
		"started_at":       state.StartedAt,
		"command_args":     state.CommandArgs,
		"aiport_file":      aiportPath,
		"aiport":           aiport,
	}, nil
}

func (s *mcpServer) toolSendMessage(args map[string]interface{}) (map[string]interface{}, error) {
	robotDir, err := requiredStringArg(args, "robot_dir")
	if err != nil {
		return nil, err
	}
	asUser, err := requiredStringArg(args, "as_user")
	if err != nil {
		return nil, err
	}
	text, err := requiredStringArg(args, "text")
	if err != nil {
		return nil, err
	}
	protocol, err := optionalStringArg(args, "protocol")
	if err != nil {
		return nil, err
	}
	channel, err := optionalStringArg(args, "channel")
	if err != nil {
		return nil, err
	}
	threadID, err := optionalStringArg(args, "thread_id")
	if err != nil {
		return nil, err
	}
	hidden, err := optionalBoolArg(args, "hidden", false)
	if err != nil {
		return nil, err
	}

	client, err := s.loadAIDevClient(robotDir)
	if err != nil {
		return nil, err
	}
	payload := map[string]interface{}{
		"as_user": asUser,
		"text":    text,
		"hidden":  hidden,
	}
	if protocol != "" {
		payload["protocol"] = protocol
	}
	if channel != "" {
		payload["channel"] = channel
	}
	if threadID != "" {
		payload["thread_id"] = threadID
	}

	res, err := callAIDevEndpoint(client, "/aidev/send_message", payload, 4*time.Second)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (s *mcpServer) toolGetMessages(args map[string]interface{}) (map[string]interface{}, error) {
	robotDir, err := requiredStringArg(args, "robot_dir")
	if err != nil {
		return nil, err
	}
	viewer, err := requiredStringArg(args, "viewer")
	if err != nil {
		return nil, err
	}
	protocol, err := optionalStringArg(args, "protocol")
	if err != nil {
		return nil, err
	}
	all, err := optionalBoolArg(args, "all", false)
	if err != nil {
		return nil, err
	}
	after, err := optionalUint64Arg(args, "after_cursor", 0)
	if err != nil {
		return nil, err
	}
	timeoutMS, err := optionalIntArg(args, "timeout_ms", 1400)
	if err != nil {
		return nil, err
	}
	if timeoutMS < 0 {
		timeoutMS = 0
	}
	limit, err := optionalIntArg(args, "limit", 0)
	if err != nil {
		return nil, err
	}
	if limit < 0 {
		limit = 0
	}

	client, err := s.loadAIDevClient(robotDir)
	if err != nil {
		return nil, err
	}
	payload := map[string]interface{}{
		"viewer":       viewer,
		"all":          all,
		"after_cursor": after,
		"timeout_ms":   timeoutMS,
	}
	if protocol != "" {
		payload["protocol"] = protocol
	}
	if limit > 0 {
		payload["limit"] = limit
	}

	httpTimeout := time.Duration(timeoutMS+1500) * time.Millisecond
	if httpTimeout < 2*time.Second {
		httpTimeout = 2 * time.Second
	}
	res, err := callAIDevEndpoint(client, "/aidev/get_messages", payload, httpTimeout)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func requiredStringArg(args map[string]interface{}, key string) (string, error) {
	val, ok := args[key]
	if !ok {
		return "", fmt.Errorf("missing required argument: %s", key)
	}
	str, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("argument %s must be a string", key)
	}
	str = strings.TrimSpace(str)
	if str == "" {
		return "", fmt.Errorf("argument %s cannot be empty", key)
	}
	return str, nil
}

func optionalStringArg(args map[string]interface{}, key string) (string, error) {
	val, ok := args[key]
	if !ok || val == nil {
		return "", nil
	}
	str, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("argument %s must be a string", key)
	}
	return strings.TrimSpace(str), nil
}

func optionalStringSliceArg(args map[string]interface{}, key string) ([]string, error) {
	val, ok := args[key]
	if !ok || val == nil {
		return nil, nil
	}
	raw, ok := val.([]interface{})
	if !ok {
		return nil, fmt.Errorf("argument %s must be an array of strings", key)
	}
	out := make([]string, 0, len(raw))
	for i, item := range raw {
		str, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("argument %s[%d] must be a string", key, i)
		}
		out = append(out, str)
	}
	return out, nil
}

func optionalStringMapArg(args map[string]interface{}, key string) (map[string]string, error) {
	val, ok := args[key]
	if !ok || val == nil {
		return nil, nil
	}
	raw, ok := val.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("argument %s must be an object with string values", key)
	}
	out := make(map[string]string, len(raw))
	for k, v := range raw {
		vs, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("argument %s[%s] must be a string", key, k)
		}
		out[k] = vs
	}
	return out, nil
}

func optionalBoolArg(args map[string]interface{}, key string, def bool) (bool, error) {
	val, ok := args[key]
	if !ok || val == nil {
		return def, nil
	}
	b, ok := val.(bool)
	if !ok {
		return false, fmt.Errorf("argument %s must be a boolean", key)
	}
	return b, nil
}

func optionalIntArg(args map[string]interface{}, key string, def int) (int, error) {
	val, ok := args[key]
	if !ok || val == nil {
		return def, nil
	}
	switch v := val.(type) {
	case float64:
		return int(v), nil
	case int:
		return v, nil
	default:
		return 0, fmt.Errorf("argument %s must be an integer", key)
	}
}

func optionalUint64Arg(args map[string]interface{}, key string, def uint64) (uint64, error) {
	val, ok := args[key]
	if !ok || val == nil {
		return def, nil
	}
	switch v := val.(type) {
	case float64:
		if v < 0 {
			return 0, fmt.Errorf("argument %s must be >= 0", key)
		}
		return uint64(v), nil
	case int:
		if v < 0 {
			return 0, fmt.Errorf("argument %s must be >= 0", key)
		}
		return uint64(v), nil
	case uint64:
		return v, nil
	default:
		return 0, fmt.Errorf("argument %s must be an integer", key)
	}
}

type aidevClientInfo struct {
	state    processState
	robotDir string
	aiport   string
}

func (s *mcpServer) loadAIDevClient(robotDir string) (aidevClientInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	resolvedDir := resolvePath(s.rootDir, robotDir)
	statePath := filepath.Join(resolvedDir, stateFileName)
	state, exists, err := readStateFile(statePath)
	if err != nil {
		return aidevClientInfo{}, err
	}
	if !exists {
		return aidevClientInfo{}, fmt.Errorf("no running robot state found for '%s'", resolvedDir)
	}
	if !isProcessRunning(state.PID) {
		return aidevClientInfo{}, fmt.Errorf("robot pid %d is not running for '%s'", state.PID, resolvedDir)
	}
	aiportPath := filepath.Join(resolvedDir, ".aiport")
	data, err := os.ReadFile(aiportPath)
	if err != nil {
		return aidevClientInfo{}, fmt.Errorf("reading '%s': %w", aiportPath, err)
	}
	aiport := strings.TrimSpace(string(data))
	if aiport == "" {
		return aidevClientInfo{}, fmt.Errorf("'%s' is empty", aiportPath)
	}
	if state.AuthToken == "" {
		return aidevClientInfo{}, fmt.Errorf("state file '%s' does not contain auth_token", statePath)
	}
	return aidevClientInfo{
		state:    state,
		robotDir: resolvedDir,
		aiport:   aiport,
	}, nil
}

func callAIDevEndpoint(client aidevClientInfo, endpoint string, payload map[string]interface{}, timeout time.Duration) (map[string]interface{}, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("serializing aidev request payload: %w", err)
	}
	url := fmt.Sprintf("http://127.0.0.1:%s%s", client.aiport, endpoint)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+client.state.AuthToken)

	httpClient := &http.Client{Timeout: timeout}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling aidev endpoint '%s': %w", endpoint, err)
	}
	defer resp.Body.Close()
	respData, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("reading aidev endpoint response '%s': %w", endpoint, err)
	}

	var result map[string]interface{}
	if len(respData) > 0 {
		if err := json.Unmarshal(respData, &result); err != nil {
			return nil, fmt.Errorf("parsing aidev endpoint response '%s': %w", endpoint, err)
		}
	} else {
		result = map[string]interface{}{}
	}

	if resp.StatusCode >= 300 {
		if e, ok := result["error"].(string); ok && e != "" {
			return nil, fmt.Errorf("aidev endpoint error: %s", e)
		}
		return nil, fmt.Errorf("aidev endpoint '%s' returned status %d", endpoint, resp.StatusCode)
	}
	return result, nil
}

func stringifyEnvMap(env map[string]string) []string {
	if len(env) == 0 {
		return nil
	}
	out := make([]string, 0, len(env))
	for key, value := range env {
		out = append(out, key+"="+value)
	}
	return out
}

func resolvePath(base, path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Clean(filepath.Join(base, path))
}

func verifyExecutableFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("checking gopherbot binary '%s': %w", path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("gopherbot binary path is a directory: %s", path)
	}
	if info.Mode().Perm()&0111 == 0 {
		return fmt.Errorf("gopherbot binary is not executable: %s", path)
	}
	return nil
}

func generateAuthToken() (string, error) {
	b := make([]byte, 24)
	if _, err := crand.Read(b); err != nil {
		return "", fmt.Errorf("generating auth token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func readStateFile(path string) (processState, bool, error) {
	var state processState
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return state, false, nil
		}
		return state, false, fmt.Errorf("reading state file '%s': %w", path, err)
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return state, false, fmt.Errorf("parsing state file '%s': %w", path, err)
	}
	return state, true, nil
}

func writeStateFile(path string, state processState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("serializing state: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return fmt.Errorf("writing temp state file '%s': %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("renaming temp state file '%s' -> '%s': %w", tmp, path, err)
	}
	return nil
}

func removeStateFile(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("removing state file '%s': %w", path, err)
	}
	return nil
}

func isProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

func waitForExit(pid int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !isProcessRunning(pid) {
			return true
		}
		time.Sleep(200 * time.Millisecond)
	}
	return !isProcessRunning(pid)
}

func hasRequestID(id json.RawMessage) bool {
	trimmed := bytes.TrimSpace(id)
	return len(trimmed) > 0 && string(trimmed) != "null"
}

func toolSuccessResult(payload interface{}) map[string]interface{} {
	return map[string]interface{}{
		"content": []map[string]string{
			{
				"type": "text",
				"text": formatPayload(payload),
			},
		},
	}
}

func toolErrorResult(err error) map[string]interface{} {
	return map[string]interface{}{
		"isError": true,
		"content": []map[string]string{
			{
				"type": "text",
				"text": err.Error(),
			},
		},
	}
}

func formatPayload(payload interface{}) string {
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", payload)
	}
	return string(data)
}
