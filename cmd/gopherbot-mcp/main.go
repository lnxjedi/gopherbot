package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type config struct {
	UserMaps map[string]map[string]string `yaml:"usermaps"`
}

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

type tapEvent struct {
	Direction   string `json:"Direction"`
	Protocol    string `json:"Protocol"`
	UserName    string `json:"UserName"`
	UserID      string `json:"UserID"`
	ChannelName string `json:"ChannelName"`
	ChannelID   string `json:"ChannelID"`
	ThreadID    string `json:"ThreadID"`
	MessageID   string `json:"MessageID"`
	SelfMessage bool   `json:"SelfMessage"`
	BotMessage  bool   `json:"BotMessage"`
	Hidden      bool   `json:"Hidden"`
	Direct      bool   `json:"Direct"`
	Text        string `json:"Text"`
}

type server struct {
	baseURL  string
	secret   string
	protocol string
	cfg      config
	events   chan tapEvent
	client   *http.Client
}

func main() {
	var (
		aidevURL   string
		aidevKey   string
		configPath string
		protocol   string
		noDefaults bool
	)
	flag.StringVar(&aidevURL, "aidev-url", "", "base URL for gopherbot aidev (e.g. http://127.0.0.1:12345)")
	flag.StringVar(&aidevKey, "aidev-key", "", "shared secret for aidev")
	flag.StringVar(&configPath, "config", "", "path to YAML config with usermaps")
	flag.StringVar(&protocol, "protocol", "terminal", "protocol key to select from usermaps")
	flag.BoolVar(&noDefaults, "no-defaults", false, "disable built-in terminal user defaults")
	flag.Parse()

	if aidevURL == "" || aidevKey == "" {
		fmt.Fprintln(os.Stderr, "missing --aidev-url or --aidev-key")
		os.Exit(2)
	}

	cfg := config{}
	if !noDefaults {
		cfg.UserMaps = defaultUserMaps()
	}
	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "read config: %v\n", err)
			os.Exit(2)
		}
		loaded := config{}
		if err := yaml.Unmarshal(data, &loaded); err != nil {
			fmt.Fprintf(os.Stderr, "parse config: %v\n", err)
			os.Exit(2)
		}
		cfg = mergeUserMaps(cfg, loaded)
	}

	s := &server{
		baseURL:  strings.TrimRight(aidevURL, "/"),
		secret:   aidevKey,
		protocol: protocol,
		cfg:      cfg,
		events:   make(chan tapEvent, 256),
		client:   &http.Client{Timeout: 0},
	}

	go s.consumeSSE()
	s.serveStdio()
}

func defaultUserMaps() map[string]map[string]string {
	return map[string]map[string]string{
		"terminal": {
			"alice": "u0001",
			"bob":   "u0002",
			"carol": "u0003",
			"david": "u0004",
			"erin":  "u0005",
		},
	}
}

func mergeUserMaps(base, override config) config {
	if base.UserMaps == nil && override.UserMaps == nil {
		return config{}
	}
	out := config{UserMaps: map[string]map[string]string{}}
	for proto, users := range base.UserMaps {
		out.UserMaps[proto] = map[string]string{}
		for name, id := range users {
			out.UserMaps[proto][name] = id
		}
	}
	for proto, users := range override.UserMaps {
		if out.UserMaps[proto] == nil {
			out.UserMaps[proto] = map[string]string{}
		}
		for name, id := range users {
			out.UserMaps[proto][name] = id
		}
	}
	return out
}

func (s *server) serveStdio() {
	reader := bufio.NewScanner(os.Stdin)
	writer := bufio.NewWriter(os.Stdout)
	for reader.Scan() {
		line := bytes.TrimSpace(reader.Bytes())
		if len(line) == 0 {
			continue
		}
		var req jsonRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			continue
		}
		if req.JSONRPC == "" {
			req.JSONRPC = "2.0"
		}
		if len(req.ID) == 0 || string(req.ID) == "null" {
			s.handleNotification(req)
			continue
		}
		resp := s.handleRequest(req)
		data, err := json.Marshal(resp)
		if err != nil {
			continue
		}
		writer.Write(data)
		writer.Write([]byte("\n"))
		writer.Flush()
	}
}

func (s *server) handleNotification(req jsonRPCRequest) {
	// Currently no-op; MCP clients may send "notifications/initialized".
}

func (s *server) handleRequest(req jsonRPCRequest) jsonRPCResponse {
	switch req.Method {
	case "initialize":
		return jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"protocolVersion": "2024-11-05",
				"capabilities": map[string]interface{}{
					"tools": map[string]interface{}{
						"listChanged": false,
					},
				},
				"serverInfo": map[string]interface{}{
					"name":    "gopherbot-mcp",
					"version": "0.1.0",
				},
			},
		}
	case "tools/list":
		return jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"tools": []mcpTool{
					{
						Name:        "send_message",
						Description: "Inject a message as a mapped user",
						InputSchema: map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"user":    map[string]interface{}{"type": "string"},
								"channel": map[string]interface{}{"type": "string"},
								"thread":  map[string]interface{}{"type": "string"},
								"message": map[string]interface{}{"type": "string"},
								"direct":  map[string]interface{}{"type": "boolean"},
							},
							"required": []string{"user", "message"},
						},
					},
					{
						Name:        "wait_for_event",
						Description: "Wait for the next aidev tap event",
						InputSchema: map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"timeout_ms": map[string]interface{}{"type": "number"},
								"direction":  map[string]interface{}{"type": "string"},
								"user":       map[string]interface{}{"type": "string"},
								"channel":    map[string]interface{}{"type": "string"},
							},
						},
					},
					{
						Name:        "control_exit",
						Description: "Request a graceful robot exit",
						InputSchema: map[string]interface{}{"type": "object"},
					},
					{
						Name:        "control_force_exit",
						Description: "Force exit with stack dump (SIGUSR1)",
						InputSchema: map[string]interface{}{"type": "object"},
					},
				},
			},
		}
	case "tools/call":
		return s.handleToolCall(req)
	default:
		return jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &jsonRPCError{Code: -32601, Message: "method not found"},
		}
	}
}

func (s *server) handleToolCall(req jsonRPCRequest) jsonRPCResponse {
	var payload struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &payload); err != nil {
		return jsonRPCResponse{JSONRPC: "2.0", ID: req.ID, Error: &jsonRPCError{Code: -32602, Message: "invalid params"}}
	}
	switch payload.Name {
	case "send_message":
		return s.toolSendMessage(req.ID, payload.Arguments)
	case "wait_for_event":
		return s.toolWaitForEvent(req.ID, payload.Arguments)
	case "control_exit":
		return s.toolControl(req.ID, "exit")
	case "control_force_exit":
		return s.toolControl(req.ID, "force_exit")
	default:
		return jsonRPCResponse{JSONRPC: "2.0", ID: req.ID, Error: &jsonRPCError{Code: -32601, Message: "tool not found"}}
	}
}

func (s *server) toolSendMessage(id json.RawMessage, args json.RawMessage) jsonRPCResponse {
	var req struct {
		User    string `json:"user"`
		Channel string `json:"channel"`
		Thread  string `json:"thread"`
		Message string `json:"message"`
		Direct  bool   `json:"direct"`
	}
	if err := json.Unmarshal(args, &req); err != nil {
		return jsonRPCResponse{JSONRPC: "2.0", ID: id, Error: &jsonRPCError{Code: -32602, Message: "invalid args"}}
	}
	userID, err := s.lookupUserID(req.User)
	if err != nil {
		return jsonRPCResponse{JSONRPC: "2.0", ID: id, Error: &jsonRPCError{Code: -32000, Message: err.Error()}}
	}
	payload := map[string]interface{}{
		"user":    req.User,
		"user_id": userID,
		"channel": req.Channel,
		"thread":  req.Thread,
		"message": req.Message,
		"direct":  req.Direct,
	}
	if err := s.postJSON("/aidev/inject", payload); err != nil {
		return jsonRPCResponse{JSONRPC: "2.0", ID: id, Error: &jsonRPCError{Code: -32001, Message: err.Error()}}
	}
	return jsonRPCResponse{JSONRPC: "2.0", ID: id, Result: mcpTextResult("sent")}
}

func (s *server) toolWaitForEvent(id json.RawMessage, args json.RawMessage) jsonRPCResponse {
	var req struct {
		TimeoutMS int    `json:"timeout_ms"`
		Direction string `json:"direction"`
		User      string `json:"user"`
		Channel   string `json:"channel"`
	}
	_ = json.Unmarshal(args, &req)
	if req.TimeoutMS <= 0 {
		req.TimeoutMS = 30000
	}
	timeout := time.NewTimer(time.Duration(req.TimeoutMS) * time.Millisecond)
	defer timeout.Stop()
	for {
		select {
		case evt := <-s.events:
			if req.Direction != "" && evt.Direction != req.Direction {
				continue
			}
			if req.User != "" && evt.UserName != req.User {
				continue
			}
			if req.Channel != "" && evt.ChannelName != req.Channel {
				continue
			}
			data, _ := json.Marshal(evt)
			return jsonRPCResponse{JSONRPC: "2.0", ID: id, Result: mcpTextResult(string(data))}
		case <-timeout.C:
			return jsonRPCResponse{JSONRPC: "2.0", ID: id, Error: &jsonRPCError{Code: -32002, Message: "timeout"}}
		}
	}
}

func (s *server) toolControl(id json.RawMessage, action string) jsonRPCResponse {
	payload := map[string]interface{}{"action": action}
	if err := s.postJSON("/aidev/control", payload); err != nil {
		return jsonRPCResponse{JSONRPC: "2.0", ID: id, Error: &jsonRPCError{Code: -32003, Message: err.Error()}}
	}
	return jsonRPCResponse{JSONRPC: "2.0", ID: id, Result: mcpTextResult("ok")}
}

func (s *server) lookupUserID(user string) (string, error) {
	if user == "" {
		return "", errors.New("missing user")
	}
	if s.cfg.UserMaps == nil {
		return "", fmt.Errorf("no usermaps configured for protocol %q", s.protocol)
	}
	m, ok := s.cfg.UserMaps[s.protocol]
	if !ok {
		return "", fmt.Errorf("no usermaps configured for protocol %q", s.protocol)
	}
	id, ok := m[user]
	if !ok {
		return "", fmt.Errorf("no mapping for user %q in protocol %q", user, s.protocol)
	}
	return id, nil
}

func (s *server) postJSON(path string, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, s.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-AIDEV-KEY", s.secret)
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("http %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return nil
}

func (s *server) consumeSSE() {
	req, err := http.NewRequest(http.MethodGet, s.baseURL+"/aidev/stream", nil)
	if err != nil {
		return
	}
	req.Header.Set("X-AIDEV-KEY", s.secret)
	resp, err := s.client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	reader := bufio.NewReader(resp.Body)
	var buf strings.Builder
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			if buf.Len() > 0 {
				var evt tapEvent
				if err := json.Unmarshal([]byte(buf.String()), &evt); err == nil {
					s.events <- evt
				}
				buf.Reset()
			}
			continue
		}
		if strings.HasPrefix(line, "data:") {
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			buf.WriteString(data)
		}
	}
}

func mcpTextResult(text string) map[string]interface{} {
	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": text,
			},
		},
		"isError": false,
	}
}
