package main

import (
	"bytes"
	crand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
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
	minMCPSSHPort      = 4222
	maxMCPSSHPort      = 4229
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

type mcpToolError struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

func (e *mcpToolError) Error() string {
	return e.Message
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
	s := &mcpServer{
		rootDir: rootDir,
	}
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
			Name:        "list_robots",
			Description: "List robot state files in target directories and report running/stale status.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"roots": map[string]interface{}{
						"type":        "array",
						"description": "Optional directories to search recursively. Defaults to MCP working directory.",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
					"max_depth": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum recursive depth under each root (default 8).",
					},
					"include_not_running": map[string]interface{}{
						"type":        "boolean",
						"description": "Include robots with stale/dead pid state (default true).",
					},
				},
			},
		},
		{
			Name:        "cleanup_stale_state",
			Description: "Remove stale robot state files where pid is no longer running.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"roots": map[string]interface{}{
						"type":        "array",
						"description": "Optional directories to search recursively. Defaults to MCP working directory.",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
					"max_depth": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum recursive depth under each root (default 8).",
					},
					"dry_run": map[string]interface{}{
						"type":        "boolean",
						"description": "When true, report stale files without removing them.",
					},
				},
			},
		},
		{
			Name:        "wait_robot_ready",
			Description: "Wait until a robot has a live process and reachable AI-dev listener.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"robot_dir": map[string]interface{}{
						"type":        "string",
						"description": "Directory where the robot is running.",
					},
					"timeout_ms": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum wait time in milliseconds (default 15000).",
					},
					"poll_ms": map[string]interface{}{
						"type":        "integer",
						"description": "Polling interval in milliseconds (default 200).",
					},
				},
				"required": []string{"robot_dir"},
			},
		},
		{
			Name:        "restart_robot",
			Description: "Restart a robot via stop + start, preserving prior binary/token/extra args when omitted.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"robot_dir": map[string]interface{}{
						"type":        "string",
						"description": "Directory where the robot is running.",
					},
					"auth_token": map[string]interface{}{
						"type":        "string",
						"description": "Optional auth token override. Defaults to previous token when available.",
					},
					"gopherbot_binary": map[string]interface{}{
						"type":        "string",
						"description": "Optional binary override. Defaults to previous binary when available.",
					},
					"extra_args": map[string]interface{}{
						"type":        "array",
						"description": "Optional extra args inserted before 'run'. Defaults to previous extras when available.",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
					"env": map[string]interface{}{
						"type":        "object",
						"description": "Optional environment variables to set for the new process.",
						"additionalProperties": map[string]interface{}{
							"type": "string",
						},
					},
					"wait_ready": map[string]interface{}{
						"type":        "boolean",
						"description": "Wait for readiness after start (default true).",
					},
					"timeout_ms": map[string]interface{}{
						"type":        "integer",
						"description": "Readiness timeout when wait_ready is true (default 15000).",
					},
					"poll_ms": map[string]interface{}{
						"type":        "integer",
						"description": "Readiness poll interval when wait_ready is true (default 200).",
					},
				},
				"required": []string{"robot_dir"},
			},
		},
		{
			Name:        "tail_robot_log",
			Description: "Read the last N lines from a robot log file.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"robot_dir": map[string]interface{}{
						"type":        "string",
						"description": "Directory where the robot is running.",
					},
					"lines": map[string]interface{}{
						"type":        "integer",
						"description": "Number of lines to return from the end of the log (default 120).",
					},
					"max_bytes": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum bytes to read from the end of the log file (default 262144).",
					},
				},
				"required": []string{"robot_dir"},
			},
		},
		{
			Name:        "read_robot_log",
			Description: "Read a byte range from a robot log file for paged log retrieval.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"robot_dir": map[string]interface{}{
						"type":        "string",
						"description": "Directory where the robot is running.",
					},
					"offset": map[string]interface{}{
						"type":        "integer",
						"description": "Starting byte offset into the log file (default 0).",
					},
					"max_bytes": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum bytes to read from offset (default 65536).",
					},
				},
				"required": []string{"robot_dir"},
			},
		},
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
					"ssh_port_min": map[string]interface{}{
						"type":        "integer",
						"description": "Minimum auto-assigned SSH port when GOPHER_SSH_PORT is unset (default 4222).",
					},
					"ssh_port_max": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum auto-assigned SSH port when GOPHER_SSH_PORT is unset (default 4229).",
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
		{
			Name:        "send_as_robot",
			Description: "Send a message as the robot on a target protocol/channel/thread (optionally direct to a user).",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"robot_dir": map[string]interface{}{
						"type":        "string",
						"description": "Directory where the robot is running.",
					},
					"text": map[string]interface{}{
						"type":        "string",
						"description": "Message text to send as the robot.",
					},
					"protocol": map[string]interface{}{
						"type":        "string",
						"description": "Protocol name, defaults to active primary protocol.",
					},
					"channel": map[string]interface{}{
						"type":        "string",
						"description": "Channel to post in (required unless direct=true).",
					},
					"thread_id": map[string]interface{}{
						"type":        "string",
						"description": "Optional thread target.",
					},
					"user": map[string]interface{}{
						"type":        "string",
						"description": "Optional user target for directed/DM sends.",
					},
					"direct": map[string]interface{}{
						"type":        "boolean",
						"description": "When true, send as direct message to user.",
					},
				},
				"required": []string{"robot_dir", "text"},
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
	case "list_robots":
		result, err = s.toolListRobots(args)
	case "cleanup_stale_state":
		result, err = s.toolCleanupStaleState(args)
	case "wait_robot_ready":
		result, err = s.toolWaitRobotReady(args)
	case "restart_robot":
		result, err = s.toolRestartRobot(args)
	case "tail_robot_log":
		result, err = s.toolTailRobotLog(args)
	case "read_robot_log":
		result, err = s.toolReadRobotLog(args)
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
	case "send_as_robot":
		result, err = s.toolSendAsRobot(args)
	default:
		return toolErrorResult(newToolError("UNKNOWN_TOOL", fmt.Sprintf("unknown tool: %s", params.Name), map[string]interface{}{
			"tool_name": params.Name,
		}))
	}
	if err != nil {
		return toolErrorResult(err)
	}
	return toolSuccessResult(result)
}

func (s *mcpServer) toolListRobots(args map[string]interface{}) (map[string]interface{}, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	roots, err := optionalRootsArg(s.rootDir, args, "roots")
	if err != nil {
		return nil, err
	}
	maxDepth, err := optionalIntArg(args, "max_depth", 8)
	if err != nil {
		return nil, err
	}
	if maxDepth < 0 {
		return nil, newToolError("INVALID_ARGUMENT", "argument max_depth must be >= 0", map[string]interface{}{"argument": "max_depth"})
	}
	includeNotRunning, err := optionalBoolArg(args, "include_not_running", true)
	if err != nil {
		return nil, err
	}

	stateFiles, warnings, err := discoverStateFiles(roots, maxDepth)
	if err != nil {
		return nil, err
	}
	robots := make([]map[string]interface{}, 0, len(stateFiles))
	for _, statePath := range stateFiles {
		robotDir := filepath.Dir(statePath)
		state, exists, readErr := readStateFile(statePath)
		if readErr != nil {
			warnings = append(warnings, map[string]interface{}{
				"state_file": statePath,
				"error":      readErr.Error(),
			})
			continue
		}
		if !exists {
			continue
		}
		running := isProcessRunning(state.PID)
		if !includeNotRunning && !running {
			continue
		}

		aiportPath := filepath.Join(robotDir, ".aiport")
		aiport := ""
		if data, err := os.ReadFile(aiportPath); err == nil {
			aiport = strings.TrimSpace(string(data))
		}
		robots = append(robots, map[string]interface{}{
			"robot_dir":        robotDir,
			"running":          running,
			"stale":            !running,
			"pid":              state.PID,
			"gopherbot_binary": state.GopherbotBin,
			"log_path":         state.LogPath,
			"state_file":       statePath,
			"started_at":       state.StartedAt,
			"command_args":     state.CommandArgs,
			"aiport_file":      aiportPath,
			"aiport":           aiport,
		})
	}

	sort.Slice(robots, func(i, j int) bool {
		left, _ := robots[i]["robot_dir"].(string)
		right, _ := robots[j]["robot_dir"].(string)
		return left < right
	})

	return map[string]interface{}{
		"roots":    roots,
		"count":    len(robots),
		"robots":   robots,
		"warnings": warnings,
	}, nil
}

func (s *mcpServer) toolCleanupStaleState(args map[string]interface{}) (map[string]interface{}, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	roots, err := optionalRootsArg(s.rootDir, args, "roots")
	if err != nil {
		return nil, err
	}
	maxDepth, err := optionalIntArg(args, "max_depth", 8)
	if err != nil {
		return nil, err
	}
	if maxDepth < 0 {
		return nil, newToolError("INVALID_ARGUMENT", "argument max_depth must be >= 0", map[string]interface{}{"argument": "max_depth"})
	}
	dryRun, err := optionalBoolArg(args, "dry_run", false)
	if err != nil {
		return nil, err
	}

	stateFiles, warnings, err := discoverStateFiles(roots, maxDepth)
	if err != nil {
		return nil, err
	}

	removed := make([]map[string]interface{}, 0)
	skipped := make([]map[string]interface{}, 0)
	for _, statePath := range stateFiles {
		state, exists, readErr := readStateFile(statePath)
		if readErr != nil {
			warnings = append(warnings, map[string]interface{}{
				"state_file": statePath,
				"error":      readErr.Error(),
			})
			continue
		}
		if !exists {
			continue
		}
		if isProcessRunning(state.PID) {
			skipped = append(skipped, map[string]interface{}{
				"state_file": statePath,
				"pid":        state.PID,
				"reason":     "process_running",
			})
			continue
		}
		if dryRun {
			removed = append(removed, map[string]interface{}{
				"state_file": statePath,
				"pid":        state.PID,
				"dry_run":    true,
			})
			continue
		}
		if err := removeStateFile(statePath); err != nil {
			warnings = append(warnings, map[string]interface{}{
				"state_file": statePath,
				"error":      err.Error(),
			})
			continue
		}
		removed = append(removed, map[string]interface{}{
			"state_file": statePath,
			"pid":        state.PID,
			"dry_run":    false,
		})
	}

	return map[string]interface{}{
		"roots":         roots,
		"dry_run":       dryRun,
		"removed_count": len(removed),
		"removed":       removed,
		"skipped_count": len(skipped),
		"skipped":       skipped,
		"warnings":      warnings,
	}, nil
}

func (s *mcpServer) toolWaitRobotReady(args map[string]interface{}) (map[string]interface{}, error) {
	robotDir, err := requiredStringArg(args, "robot_dir")
	if err != nil {
		return nil, err
	}
	timeoutMS, err := optionalIntArg(args, "timeout_ms", 15000)
	if err != nil {
		return nil, err
	}
	pollMS, err := optionalIntArg(args, "poll_ms", 200)
	if err != nil {
		return nil, err
	}
	if timeoutMS < 0 {
		return nil, newToolError("INVALID_ARGUMENT", "argument timeout_ms must be >= 0", map[string]interface{}{"argument": "timeout_ms"})
	}
	if pollMS <= 0 {
		return nil, newToolError("INVALID_ARGUMENT", "argument poll_ms must be > 0", map[string]interface{}{"argument": "poll_ms"})
	}

	start := time.Now()
	deadline := start.Add(time.Duration(timeoutMS) * time.Millisecond)
	for {
		probe, ready, err := s.probeRobotReady(robotDir)
		if err != nil {
			return nil, err
		}
		if ready {
			probe["ready"] = true
			probe["waited_ms"] = time.Since(start).Milliseconds()
			return probe, nil
		}
		if timeoutMS == 0 || time.Now().After(deadline) {
			probe["ready"] = false
			probe["waited_ms"] = time.Since(start).Milliseconds()
			return nil, newToolError("ROBOT_NOT_READY_TIMEOUT", fmt.Sprintf("robot '%s' was not ready within %dms", probe["robot_dir"], timeoutMS), probe)
		}
		time.Sleep(time.Duration(pollMS) * time.Millisecond)
	}
}

func (s *mcpServer) toolRestartRobot(args map[string]interface{}) (map[string]interface{}, error) {
	robotDir, err := requiredStringArg(args, "robot_dir")
	if err != nil {
		return nil, err
	}
	waitReady, err := optionalBoolArg(args, "wait_ready", true)
	if err != nil {
		return nil, err
	}

	prevState, _, _ := s.readStateForRobotDir(robotDir)
	startArgs := map[string]interface{}{"robot_dir": robotDir}

	gopherbotBin, err := optionalStringArg(args, "gopherbot_binary")
	if err != nil {
		return nil, err
	}
	if gopherbotBin == "" && prevState != nil {
		gopherbotBin = strings.TrimSpace(prevState.GopherbotBin)
	}
	if gopherbotBin != "" {
		startArgs["gopherbot_binary"] = gopherbotBin
	}

	authToken, err := optionalStringArg(args, "auth_token")
	if err != nil {
		return nil, err
	}
	if authToken == "" && prevState != nil {
		authToken = strings.TrimSpace(prevState.AuthToken)
	}
	if authToken != "" {
		startArgs["auth_token"] = authToken
	}

	if rawExtraArgs, ok := args["extra_args"]; ok {
		startArgs["extra_args"] = rawExtraArgs
	} else if prevState != nil {
		prevExtraArgs := extractExtraArgs(prevState.CommandArgs)
		if len(prevExtraArgs) > 0 {
			startArgs["extra_args"] = prevExtraArgs
		}
	}
	if rawEnv, ok := args["env"]; ok {
		startArgs["env"] = rawEnv
	}
	if rawPortMin, ok := args["ssh_port_min"]; ok {
		startArgs["ssh_port_min"] = rawPortMin
	}
	if rawPortMax, ok := args["ssh_port_max"]; ok {
		startArgs["ssh_port_max"] = rawPortMax
	}

	stopResult, err := s.toolStopRobot(map[string]interface{}{"robot_dir": robotDir})
	if err != nil {
		return nil, newToolError("RESTART_STOP_FAILED", err.Error(), map[string]interface{}{"robot_dir": resolvePath(s.rootDir, robotDir)})
	}
	startResult, err := s.toolStartRobot(startArgs)
	if err != nil {
		return nil, newToolError("RESTART_START_FAILED", err.Error(), map[string]interface{}{
			"robot_dir": resolvePath(s.rootDir, robotDir),
			"stop":      stopResult,
		})
	}

	result := map[string]interface{}{
		"robot_dir": resolvePath(s.rootDir, robotDir),
		"stop":      stopResult,
		"start":     startResult,
	}

	if waitReady {
		waitArgs := map[string]interface{}{"robot_dir": robotDir}
		if timeoutMS, ok := args["timeout_ms"]; ok {
			waitArgs["timeout_ms"] = timeoutMS
		}
		if pollMS, ok := args["poll_ms"]; ok {
			waitArgs["poll_ms"] = pollMS
		}
		readyResult, err := s.toolWaitRobotReady(waitArgs)
		if err != nil {
			return nil, newToolError("RESTART_WAIT_FAILED", err.Error(), map[string]interface{}{
				"robot_dir": resolvePath(s.rootDir, robotDir),
				"stop":      stopResult,
				"start":     startResult,
			})
		}
		result["ready"] = readyResult
	}

	return result, nil
}

func (s *mcpServer) toolTailRobotLog(args map[string]interface{}) (map[string]interface{}, error) {
	robotDir, err := requiredStringArg(args, "robot_dir")
	if err != nil {
		return nil, err
	}
	lines, err := optionalIntArg(args, "lines", 120)
	if err != nil {
		return nil, err
	}
	maxBytes, err := optionalIntArg(args, "max_bytes", 262144)
	if err != nil {
		return nil, err
	}
	if lines <= 0 {
		return nil, newToolError("INVALID_ARGUMENT", "argument lines must be > 0", map[string]interface{}{"argument": "lines"})
	}
	if maxBytes <= 0 {
		return nil, newToolError("INVALID_ARGUMENT", "argument max_bytes must be > 0", map[string]interface{}{"argument": "max_bytes"})
	}

	logPath, err := s.resolveRobotLogPath(robotDir)
	if err != nil {
		return nil, err
	}
	chunk, fileSize, readStart, err := readTailBytes(logPath, int64(maxBytes))
	if err != nil {
		return nil, err
	}
	logText, lineCount := tailLinesFromChunk(chunk, lines)
	return map[string]interface{}{
		"robot_dir":              resolvePath(s.rootDir, robotDir),
		"log_path":               logPath,
		"text":                   logText,
		"line_count":             lineCount,
		"requested_lines":        lines,
		"max_bytes":              maxBytes,
		"file_size":              fileSize,
		"read_start_offset":      readStart,
		"truncated_by_max_bytes": readStart > 0,
	}, nil
}

func (s *mcpServer) toolReadRobotLog(args map[string]interface{}) (map[string]interface{}, error) {
	robotDir, err := requiredStringArg(args, "robot_dir")
	if err != nil {
		return nil, err
	}
	offset, err := optionalIntArg(args, "offset", 0)
	if err != nil {
		return nil, err
	}
	maxBytes, err := optionalIntArg(args, "max_bytes", 65536)
	if err != nil {
		return nil, err
	}
	if offset < 0 {
		return nil, newToolError("INVALID_ARGUMENT", "argument offset must be >= 0", map[string]interface{}{"argument": "offset"})
	}
	if maxBytes <= 0 {
		return nil, newToolError("INVALID_ARGUMENT", "argument max_bytes must be > 0", map[string]interface{}{"argument": "max_bytes"})
	}

	logPath, err := s.resolveRobotLogPath(robotDir)
	if err != nil {
		return nil, err
	}
	text, fileSize, nextOffset, eof, err := readLogRange(logPath, int64(offset), int64(maxBytes))
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"robot_dir":   resolvePath(s.rootDir, robotDir),
		"log_path":    logPath,
		"text":        text,
		"offset":      offset,
		"next_offset": nextOffset,
		"max_bytes":   maxBytes,
		"file_size":   fileSize,
		"eof":         eof,
	}, nil
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
	sshPortMin, err := optionalIntArg(args, "ssh_port_min", minMCPSSHPort)
	if err != nil {
		return nil, err
	}
	sshPortMax, err := optionalIntArg(args, "ssh_port_max", maxMCPSSHPort)
	if err != nil {
		return nil, err
	}
	if sshPortMin <= 0 || sshPortMax <= 0 || sshPortMin > sshPortMax {
		return nil, newToolError("INVALID_ARGUMENT", "invalid ssh_port_min/ssh_port_max range", map[string]interface{}{
			"ssh_port_min": sshPortMin,
			"ssh_port_max": sshPortMax,
		})
	}
	assignedSSHPort := ""
	extraEnv, assignedSSHPort, err = ensureSSHPortEnv(extraEnv, sshPortMin, sshPortMax)
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
		"ssh_port":         assignedSSHPort,
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

func (s *mcpServer) toolSendAsRobot(args map[string]interface{}) (map[string]interface{}, error) {
	robotDir, err := requiredStringArg(args, "robot_dir")
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
	user, err := optionalStringArg(args, "user")
	if err != nil {
		return nil, err
	}
	direct, err := optionalBoolArg(args, "direct", false)
	if err != nil {
		return nil, err
	}

	client, err := s.loadAIDevClient(robotDir)
	if err != nil {
		return nil, err
	}
	payload := map[string]interface{}{
		"text":      text,
		"channel":   channel,
		"thread_id": threadID,
		"user":      user,
		"direct":    direct,
	}
	if protocol != "" {
		payload["protocol"] = protocol
	}
	res, err := callAIDevEndpoint(client, "/aidev/send_as_robot", payload, 4*time.Second)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func requiredStringArg(args map[string]interface{}, key string) (string, error) {
	val, ok := args[key]
	if !ok {
		return "", newToolError("INVALID_ARGUMENT", fmt.Sprintf("missing required argument: %s", key), map[string]interface{}{"argument": key})
	}
	str, ok := val.(string)
	if !ok {
		return "", newToolError("INVALID_ARGUMENT", fmt.Sprintf("argument %s must be a string", key), map[string]interface{}{"argument": key})
	}
	str = strings.TrimSpace(str)
	if str == "" {
		return "", newToolError("INVALID_ARGUMENT", fmt.Sprintf("argument %s cannot be empty", key), map[string]interface{}{"argument": key})
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
		return "", newToolError("INVALID_ARGUMENT", fmt.Sprintf("argument %s must be a string", key), map[string]interface{}{"argument": key})
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
		return nil, newToolError("INVALID_ARGUMENT", fmt.Sprintf("argument %s must be an array of strings", key), map[string]interface{}{"argument": key})
	}
	out := make([]string, 0, len(raw))
	for i, item := range raw {
		str, ok := item.(string)
		if !ok {
			return nil, newToolError("INVALID_ARGUMENT", fmt.Sprintf("argument %s[%d] must be a string", key, i), map[string]interface{}{"argument": key, "index": i})
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
		return nil, newToolError("INVALID_ARGUMENT", fmt.Sprintf("argument %s must be an object with string values", key), map[string]interface{}{"argument": key})
	}
	out := make(map[string]string, len(raw))
	for k, v := range raw {
		vs, ok := v.(string)
		if !ok {
			return nil, newToolError("INVALID_ARGUMENT", fmt.Sprintf("argument %s[%s] must be a string", key, k), map[string]interface{}{"argument": key, "map_key": k})
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
		return false, newToolError("INVALID_ARGUMENT", fmt.Sprintf("argument %s must be a boolean", key), map[string]interface{}{"argument": key})
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
		return 0, newToolError("INVALID_ARGUMENT", fmt.Sprintf("argument %s must be an integer", key), map[string]interface{}{"argument": key})
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
			return 0, newToolError("INVALID_ARGUMENT", fmt.Sprintf("argument %s must be >= 0", key), map[string]interface{}{"argument": key})
		}
		return uint64(v), nil
	case int:
		if v < 0 {
			return 0, newToolError("INVALID_ARGUMENT", fmt.Sprintf("argument %s must be >= 0", key), map[string]interface{}{"argument": key})
		}
		return uint64(v), nil
	case uint64:
		return v, nil
	default:
		return 0, newToolError("INVALID_ARGUMENT", fmt.Sprintf("argument %s must be an integer", key), map[string]interface{}{"argument": key})
	}
}

func optionalRootsArg(base string, args map[string]interface{}, key string) ([]string, error) {
	rootsRaw, err := optionalStringSliceArg(args, key)
	if err != nil {
		return nil, err
	}
	if len(rootsRaw) == 0 {
		return []string{base}, nil
	}

	seen := make(map[string]struct{}, len(rootsRaw))
	roots := make([]string, 0, len(rootsRaw))
	for _, root := range rootsRaw {
		resolved := resolvePath(base, root)
		if _, ok := seen[resolved]; ok {
			continue
		}
		seen[resolved] = struct{}{}
		roots = append(roots, resolved)
	}
	sort.Strings(roots)
	return roots, nil
}

func (s *mcpServer) readStateForRobotDir(robotDir string) (*processState, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	resolvedDir := resolvePath(s.rootDir, robotDir)
	statePath := filepath.Join(resolvedDir, stateFileName)
	state, exists, err := readStateFile(statePath)
	if err != nil {
		return nil, statePath, err
	}
	if !exists {
		return nil, statePath, nil
	}
	return &state, statePath, nil
}

func (s *mcpServer) probeRobotReady(robotDir string) (map[string]interface{}, bool, error) {
	s.mu.Lock()
	resolvedDir := resolvePath(s.rootDir, robotDir)
	statePath := filepath.Join(resolvedDir, stateFileName)
	state, exists, err := readStateFile(statePath)
	s.mu.Unlock()
	if err != nil {
		return nil, false, err
	}

	probe := map[string]interface{}{
		"robot_dir":  resolvedDir,
		"state_file": statePath,
	}
	if !exists {
		probe["reason"] = "no_state_file"
		return probe, false, nil
	}
	probe["pid"] = state.PID
	running := isProcessRunning(state.PID)
	probe["running"] = running
	if !running {
		probe["reason"] = "process_not_running"
		return probe, false, nil
	}

	aiportPath := filepath.Join(resolvedDir, ".aiport")
	probe["aiport_file"] = aiportPath
	data, err := os.ReadFile(aiportPath)
	if err != nil {
		probe["reason"] = "aiport_unavailable"
		probe["error"] = err.Error()
		return probe, false, nil
	}
	aiport := strings.TrimSpace(string(data))
	probe["aiport"] = aiport
	if aiport == "" {
		probe["reason"] = "aiport_empty"
		return probe, false, nil
	}

	if _, err := strconv.Atoi(aiport); err != nil {
		probe["reason"] = "aiport_invalid"
		probe["error"] = err.Error()
		return probe, false, nil
	}
	conn, err := net.DialTimeout("tcp", "127.0.0.1:"+aiport, 300*time.Millisecond)
	if err != nil {
		probe["reason"] = "listener_unreachable"
		probe["error"] = err.Error()
		return probe, false, nil
	}
	_ = conn.Close()
	probe["reason"] = "ready"
	return probe, true, nil
}

func extractExtraArgs(commandArgs []string) []string {
	if len(commandArgs) < 3 {
		return nil
	}
	if commandArgs[0] != "--aidev" {
		return nil
	}
	if commandArgs[len(commandArgs)-1] != "run" {
		return nil
	}
	if len(commandArgs) == 3 {
		return nil
	}
	out := make([]string, 0, len(commandArgs)-3)
	out = append(out, commandArgs[2:len(commandArgs)-1]...)
	return out
}

func ensureSSHPortEnv(env map[string]string, portMin, portMax int) (map[string]string, string, error) {
	if env == nil {
		env = map[string]string{}
	}
	protocol := strings.TrimSpace(strings.ToLower(env["GOPHER_PROTOCOL"]))
	if protocol != "" && protocol != "ssh" {
		return env, "", nil
	}
	if existing := strings.TrimSpace(env["GOPHER_SSH_PORT"]); existing != "" {
		return env, existing, nil
	}
	port, err := findAvailableTCPPort("127.0.0.1", portMin, portMax)
	if err != nil {
		return nil, "", newToolError("NO_AVAILABLE_SSH_PORT", err.Error(), map[string]interface{}{
			"ssh_port_min": portMin,
			"ssh_port_max": portMax,
		})
	}
	env["GOPHER_SSH_PORT"] = strconv.Itoa(port)
	return env, env["GOPHER_SSH_PORT"], nil
}

func findAvailableTCPPort(host string, portMin, portMax int) (int, error) {
	for port := portMin; port <= portMax; port++ {
		addr := net.JoinHostPort(host, strconv.Itoa(port))
		listener, err := net.Listen("tcp", addr)
		if err != nil {
			continue
		}
		_ = listener.Close()
		return port, nil
	}
	return 0, fmt.Errorf("no available TCP ports in range %d-%d", portMin, portMax)
}

func (s *mcpServer) resolveRobotLogPath(robotDir string) (string, error) {
	resolvedDir := resolvePath(s.rootDir, robotDir)
	logPath := filepath.Join(resolvedDir, "robot.log")

	state, _, err := s.readStateForRobotDir(robotDir)
	if err != nil {
		return "", err
	}
	if state != nil && strings.TrimSpace(state.LogPath) != "" {
		logPath = state.LogPath
	}
	if !filepath.IsAbs(logPath) {
		logPath = resolvePath(s.rootDir, logPath)
	}
	info, err := os.Stat(logPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", newToolError("LOG_NOT_FOUND", fmt.Sprintf("log file not found: %s", logPath), map[string]interface{}{
				"robot_dir": resolvedDir,
				"log_path":  logPath,
			})
		}
		return "", fmt.Errorf("checking log file '%s': %w", logPath, err)
	}
	if info.IsDir() {
		return "", newToolError("INVALID_LOG_PATH", fmt.Sprintf("log path is a directory: %s", logPath), map[string]interface{}{
			"robot_dir": resolvedDir,
			"log_path":  logPath,
		})
	}
	return logPath, nil
}

func readTailBytes(path string, maxBytes int64) ([]byte, int64, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("opening log file '%s': %w", path, err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, 0, 0, fmt.Errorf("stat log file '%s': %w", path, err)
	}
	fileSize := info.Size()
	if fileSize == 0 {
		return []byte{}, 0, 0, nil
	}
	start := int64(0)
	if fileSize > maxBytes {
		start = fileSize - maxBytes
	}
	if _, err := file.Seek(start, io.SeekStart); err != nil {
		return nil, 0, 0, fmt.Errorf("seek log file '%s': %w", path, err)
	}
	chunk, err := io.ReadAll(file)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("reading log file '%s': %w", path, err)
	}
	return chunk, fileSize, start, nil
}

func tailLinesFromChunk(chunk []byte, lines int) (string, int) {
	if len(chunk) == 0 {
		return "", 0
	}
	raw := string(chunk)
	parts := strings.Split(raw, "\n")
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	if len(parts) == 0 {
		return "", 0
	}
	start := len(parts) - lines
	if start < 0 {
		start = 0
	}
	selected := parts[start:]
	return strings.Join(selected, "\n"), len(selected)
}

func readLogRange(path string, offset, maxBytes int64) (string, int64, int64, bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", 0, 0, false, fmt.Errorf("opening log file '%s': %w", path, err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return "", 0, 0, false, fmt.Errorf("stat log file '%s': %w", path, err)
	}
	fileSize := info.Size()
	if offset > fileSize {
		return "", fileSize, fileSize, true, nil
	}
	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return "", 0, 0, false, fmt.Errorf("seek log file '%s': %w", path, err)
	}
	reader := io.LimitReader(file, maxBytes)
	chunk, err := io.ReadAll(reader)
	if err != nil {
		return "", 0, 0, false, fmt.Errorf("reading log file '%s': %w", path, err)
	}
	nextOffset := offset + int64(len(chunk))
	return string(chunk), fileSize, nextOffset, nextOffset >= fileSize, nil
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

func discoverStateFiles(roots []string, maxDepth int) ([]string, []map[string]interface{}, error) {
	files := make([]string, 0)
	warnings := make([]map[string]interface{}, 0)
	for _, root := range roots {
		rootInfo, err := os.Stat(root)
		if err != nil {
			warnings = append(warnings, map[string]interface{}{
				"root":  root,
				"error": fmt.Sprintf("checking root: %v", err),
			})
			continue
		}
		if !rootInfo.IsDir() {
			warnings = append(warnings, map[string]interface{}{
				"root":  root,
				"error": "not a directory",
			})
			continue
		}
		walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				warnings = append(warnings, map[string]interface{}{
					"root":  root,
					"path":  path,
					"error": walkErr.Error(),
				})
				return nil
			}
			if d.IsDir() && relativeDepth(root, path) > maxDepth {
				return filepath.SkipDir
			}
			if !d.IsDir() && d.Name() == stateFileName {
				files = append(files, path)
			}
			return nil
		})
		if walkErr != nil {
			return nil, nil, fmt.Errorf("walking root '%s': %w", root, walkErr)
		}
	}
	sort.Strings(files)
	return files, warnings, nil
}

func relativeDepth(root, path string) int {
	if root == path {
		return 0
	}
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." {
		return 0
	}
	return strings.Count(rel, string(os.PathSeparator)) + 1
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
	toolErr := extractToolError(err)
	return map[string]interface{}{
		"isError": true,
		"error": map[string]interface{}{
			"code":    toolErr.Code,
			"message": toolErr.Message,
			"details": toolErr.Details,
		},
		"content": []map[string]string{
			{
				"type": "text",
				"text": toolErr.Message,
			},
		},
	}
}

func newToolError(code, message string, details map[string]interface{}) *mcpToolError {
	return &mcpToolError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

func extractToolError(err error) *mcpToolError {
	if err == nil {
		return newToolError("INTERNAL_ERROR", "unknown error", nil)
	}
	var toolErr *mcpToolError
	if errors.As(err, &toolErr) && toolErr != nil {
		return toolErr
	}
	return newToolError("INTERNAL_ERROR", err.Error(), nil)
}

func formatPayload(payload interface{}) string {
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", payload)
	}
	return string(data)
}
