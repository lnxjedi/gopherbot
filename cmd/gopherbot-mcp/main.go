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
	ID          string `json:"ID"`
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

type connectInfo struct {
	URL      string `json:"url"`
	Key      string `json:"key"`
	Config   string `json:"config"`
	Protocol string `json:"protocol"`
}

type server struct {
	baseURL  string
	secret   string
	protocol string
	cfg      config
	client   *http.Client
	debug    bool
}

func main() {
	var (
		aidevURL    string
		aidevKey    string
		configPath  string
		protocol    string
		connectFile string
		noDefaults  bool
	)
	flag.StringVar(&aidevURL, "aidev-url", "", "base URL for gopherbot aidev (e.g. http://127.0.0.1:12345)")
	flag.StringVar(&aidevKey, "aidev-key", "", "shared secret for aidev")
	flag.StringVar(&configPath, "config", "", "path to YAML config with usermaps")
	flag.StringVar(&protocol, "protocol", "", "protocol key to select from usermaps (default \"terminal\" or connect file)")
	flag.StringVar(&connectFile, "connect-file", "", "path to .mcp-connect JSON")
	flag.BoolVar(&noDefaults, "no-defaults", false, "disable built-in terminal user defaults")
	flag.Parse()

	if connectFile != "" {
		path := connectFile
		info, err := loadConnectFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "read connect file: %v\n", err)
			os.Exit(2)
		}
		if aidevURL == "" {
			aidevURL = info.URL
		}
		if aidevKey == "" {
			aidevKey = info.Key
		}
		if configPath == "" && info.Config != "" {
			configPath = info.Config
		}
		if protocol == "" && info.Protocol != "" {
			protocol = info.Protocol
		}
	}

	if aidevURL == "" || aidevKey == "" {
		fmt.Fprintln(os.Stderr, "missing --aidev-url or --aidev-key (or --connect-file)")
		os.Exit(2)
	}

	if protocol == "" {
		protocol = "terminal"
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
		client:   &http.Client{Timeout: 0},
		debug:    os.Getenv("GOPHER_AIDEV_MCP_DEBUG") != "",
	}

	go s.sendHelloLoop()
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

func loadConnectFile(path string) (connectInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return connectInfo{}, err
	}
	var info connectInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return connectInfo{}, err
	}
	if info.URL == "" || info.Key == "" {
		return connectInfo{}, fmt.Errorf("missing url or key in %s", path)
	}
	return info, nil
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
						Name:        "send_and_fetch",
						Description: "Send a message and return events until a reply arrives (delete-on-read queue)",
						InputSchema: map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"user":       map[string]interface{}{"type": "string"},
								"channel":    map[string]interface{}{"type": "string"},
								"thread":     map[string]interface{}{"type": "string"},
								"message":    map[string]interface{}{"type": "string"},
								"direct":     map[string]interface{}{"type": "boolean"},
								"timeout_ms": map[string]interface{}{"type": "number"},
							},
							"required": []string{"user", "message"},
						},
					},
					{
						Name:        "fetch_events",
						Description: "Fetch queued aidev events (delete-on-read; optionally wait for at least one event)",
						InputSchema: map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"wait_ms": map[string]interface{}{"type": "number"},
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
	case "send_and_fetch":
		return s.toolSendAndFetch(req.ID, payload.Arguments)
	case "fetch_events":
		return s.toolFetchEvents(req.ID, payload.Arguments)
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

func (s *server) toolFetchEvents(id json.RawMessage, args json.RawMessage) jsonRPCResponse {
	var req struct {
		WaitMS int `json:"wait_ms"`
	}
	_ = json.Unmarshal(args, &req)
	waitDeadline := time.Now().Add(time.Duration(req.WaitMS) * time.Millisecond)
	for {
		path := "/aidev/events"
		var resp struct {
			Events []tapEvent `json:"events"`
			LastID string     `json:"last_id"`
		}
		if err := s.getJSON(path, &resp); err != nil {
			return jsonRPCResponse{JSONRPC: "2.0", ID: id, Error: &jsonRPCError{Code: -32004, Message: err.Error()}}
		}
		if req.WaitMS <= 0 || len(resp.Events) > 0 || time.Now().After(waitDeadline) {
			data, _ := json.Marshal(resp)
			return jsonRPCResponse{JSONRPC: "2.0", ID: id, Result: mcpTextResult(string(data))}
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func (s *server) toolControl(id json.RawMessage, action string) jsonRPCResponse {
	payload := map[string]interface{}{"action": action}
	if err := s.postJSON("/aidev/control", payload); err != nil {
		return jsonRPCResponse{JSONRPC: "2.0", ID: id, Error: &jsonRPCError{Code: -32003, Message: err.Error()}}
	}
	return jsonRPCResponse{JSONRPC: "2.0", ID: id, Result: mcpTextResult("ok")}
}

func (s *server) toolSendAndFetch(id json.RawMessage, args json.RawMessage) jsonRPCResponse {
	var req struct {
		User      string `json:"user"`
		Channel   string `json:"channel"`
		Thread    string `json:"thread"`
		Message   string `json:"message"`
		Direct    bool   `json:"direct"`
		TimeoutMS int    `json:"timeout_ms"`
	}
	if err := json.Unmarshal(args, &req); err != nil {
		return jsonRPCResponse{JSONRPC: "2.0", ID: id, Error: &jsonRPCError{Code: -32602, Message: "invalid args"}}
	}
	if req.TimeoutMS <= 0 {
		req.TimeoutMS = 14000
	}
	userID, err := s.lookupUserID(req.User)
	if err != nil {
		return jsonRPCResponse{JSONRPC: "2.0", ID: id, Error: &jsonRPCError{Code: -32000, Message: err.Error()}}
	}
	if req.Message == "" {
		return jsonRPCResponse{JSONRPC: "2.0", ID: id, Error: &jsonRPCError{Code: -32602, Message: "missing message"}}
	}
	if !req.Direct && req.Channel == "" {
		return jsonRPCResponse{JSONRPC: "2.0", ID: id, Error: &jsonRPCError{Code: -32602, Message: "missing channel"}}
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
	waitDeadline := time.Now().Add(time.Duration(req.TimeoutMS) * time.Millisecond)
	seen := make([]tapEvent, 0, 8)
	for {
		path := "/aidev/events"
		var resp struct {
			Events []tapEvent `json:"events"`
			LastID string     `json:"last_id"`
		}
		if err := s.getJSON(path, &resp); err != nil {
			return jsonRPCResponse{JSONRPC: "2.0", ID: id, Error: &jsonRPCError{Code: -32004, Message: err.Error()}}
		}
		if len(resp.Events) > 0 {
			seen = append(seen, resp.Events...)
		}
		if s.debug && len(resp.Events) > 0 {
			fmt.Fprintf(os.Stderr, "send_and_fetch: %d events received\n", len(resp.Events))
			for i, evt := range resp.Events {
				fmt.Fprintf(os.Stderr, "  %d: id=%s dir=%s user=%s self=%t chan=%s thr=%s text=%q\n", i, evt.ID, evt.Direction, evt.UserName, evt.SelfMessage, evt.ChannelName, evt.ThreadID, evt.Text)
			}
		}
		nonSenderInbound := false
		for _, evt := range seen {
			if evt.Direction != "inbound" {
				continue
			}
			if evt.UserName == "" {
				continue
			}
			if evt.UserName == req.User || evt.UserName == "aidev" {
				continue
			}
			if req.Channel != "" && evt.ChannelName != req.Channel {
				continue
			}
			if req.Thread != "" && evt.ThreadID != req.Thread {
				continue
			}
			nonSenderInbound = true
			break
		}
		if nonSenderInbound || time.Now().After(waitDeadline) {
			if s.debug {
				fmt.Fprintf(os.Stderr, "send_and_fetch: returning (nonSenderInbound=%t timeout=%t)\n", nonSenderInbound, time.Now().After(waitDeadline))
			}
			data, _ := json.Marshal(struct {
				Events []tapEvent `json:"events"`
				LastID string     `json:"last_id"`
			}{
				Events: seen,
				LastID: resp.LastID,
			})
			return jsonRPCResponse{JSONRPC: "2.0", ID: id, Result: mcpTextResult(string(data))}
		}
		time.Sleep(200 * time.Millisecond)
	}
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

func (s *server) getJSON(path string, target interface{}) error {
	req, err := http.NewRequest(http.MethodGet, s.baseURL+path, nil)
	if err != nil {
		return err
	}
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
	return json.NewDecoder(resp.Body).Decode(target)
}

func (s *server) sendHelloLoop() {
	payload := map[string]interface{}{"action": "hello"}
	for {
		if err := s.postJSON("/aidev/control", payload); err == nil {
			_ = s.postJSON("/aidev/control", map[string]interface{}{"action": "ready"})
			return
		}
		time.Sleep(250 * time.Millisecond)
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
