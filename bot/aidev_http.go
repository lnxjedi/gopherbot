package bot

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/lnxjedi/gopherbot/robot"
)

type aidevSendMessageRequest struct {
	Protocol string `json:"protocol"`
	AsUser   string `json:"as_user"`
	Text     string `json:"text"`
	Channel  string `json:"channel"`
	ThreadID string `json:"thread_id"`
	Hidden   bool   `json:"hidden"`
	Direct   bool   `json:"direct"`
}

type aidevGetMessagesRequest struct {
	Protocol    string `json:"protocol"`
	Viewer      string `json:"viewer"`
	AfterCursor uint64 `json:"after_cursor"`
	All         bool   `json:"all"`
	Limit       int    `json:"limit"`
	TimeoutMS   int    `json:"timeout_ms"`
}

func serveAIDevSendMessage(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		rw.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if err := authorizeAIDevRequest(req); err != nil {
		writeAIDevError(rw, http.StatusUnauthorized, err)
		return
	}
	var in aidevSendMessageRequest
	if err := decodeAIDevJSON(req.Body, &in); err != nil {
		writeAIDevError(rw, http.StatusBadRequest, err)
		return
	}
	protocol, err := resolveAIDevProtocol(in.Protocol)
	if err != nil {
		writeAIDevError(rw, http.StatusBadRequest, err)
		return
	}
	api, err := aidevConnectorAPI(protocol)
	if err != nil {
		writeAIDevError(rw, http.StatusBadRequest, err)
		return
	}
	injector, ok := api.(robot.Injector)
	if !ok {
		writeAIDevError(rw, http.StatusBadRequest, fmt.Errorf("protocol '%s' does not support message injection", protocol))
		return
	}
	res, err := injector.InjectMessage(robot.InjectMessageRequest{
		AsUser:  in.AsUser,
		Text:    in.Text,
		Channel: in.Channel,
		Thread:  in.ThreadID,
		Hidden:  in.Hidden,
		Direct:  in.Direct,
	})
	if err != nil {
		writeAIDevError(rw, http.StatusBadRequest, err)
		return
	}
	if res.Protocol == "" {
		res.Protocol = protocol
	}
	writeAIDevJSON(rw, res)
}

func serveAIDevGetMessages(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		rw.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if err := authorizeAIDevRequest(req); err != nil {
		writeAIDevError(rw, http.StatusUnauthorized, err)
		return
	}
	var in aidevGetMessagesRequest
	if err := decodeAIDevJSON(req.Body, &in); err != nil {
		writeAIDevError(rw, http.StatusBadRequest, err)
		return
	}
	protocol, err := resolveAIDevProtocol(in.Protocol)
	if err != nil {
		writeAIDevError(rw, http.StatusBadRequest, err)
		return
	}
	api, err := aidevConnectorAPI(protocol)
	if err != nil {
		writeAIDevError(rw, http.StatusBadRequest, err)
		return
	}
	source, ok := api.(robot.MessageSource)
	if !ok {
		writeAIDevError(rw, http.StatusBadRequest, fmt.Errorf("protocol '%s' does not support message retrieval", protocol))
		return
	}
	timeoutMS := in.TimeoutMS
	if timeoutMS == 0 {
		timeoutMS = 1400
	}
	res, err := source.GetMessages(robot.MessageQuery{
		Viewer:      in.Viewer,
		AfterCursor: in.AfterCursor,
		Limit:       in.Limit,
		TimeoutMS:   timeoutMS,
		All:         in.All,
	})
	if err != nil {
		writeAIDevError(rw, http.StatusBadRequest, err)
		return
	}
	if res.Protocol == "" {
		res.Protocol = protocol
	}
	writeAIDevJSON(rw, res)
}

func decodeAIDevJSON(body io.ReadCloser, v interface{}) error {
	defer body.Close()
	dec := json.NewDecoder(io.LimitReader(body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("invalid request JSON: %w", err)
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return errors.New("request body must contain exactly one JSON object")
	}
	return nil
}

func resolveAIDevProtocol(protocol string) (string, error) {
	p := normalizeProtocolName(protocol)
	if p != "" {
		return p, nil
	}
	if primary, ok := getRuntimePrimaryProtocol(); ok {
		return primary, nil
	}
	return "", errors.New("no active protocol available")
}

func aidevConnectorAPI(protocol string) (interface{}, error) {
	conn, ok := getRuntimeConnector(protocol)
	if !ok {
		return nil, fmt.Errorf("protocol '%s' is not active", protocol)
	}
	if provider, ok := conn.(robot.ConnectorAPIProvider); ok {
		return provider.ConnectorAPI(), nil
	}
	return conn, nil
}

func authorizeAIDevRequest(req *http.Request) error {
	if !isAIDevMode() {
		return errors.New("aidev mode is not enabled")
	}
	expected := getAIDevToken()
	if expected == "" {
		return errors.New("aidev token is not configured")
	}
	auth := strings.TrimSpace(req.Header.Get("Authorization"))
	if len(auth) < 8 || !strings.EqualFold(auth[:7], "Bearer ") {
		return errors.New("missing bearer authorization token")
	}
	provided := strings.TrimSpace(auth[7:])
	if provided == "" {
		return errors.New("empty bearer authorization token")
	}
	if subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) != 1 {
		return errors.New("invalid aidev token")
	}
	return nil
}

func writeAIDevJSON(rw http.ResponseWriter, payload interface{}) {
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(rw).Encode(payload)
}

func writeAIDevError(rw http.ResponseWriter, code int, err error) {
	writeAIDevJSONWithCode(rw, code, map[string]string{"error": err.Error()})
}

func writeAIDevJSONWithCode(rw http.ResponseWriter, code int, payload interface{}) {
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(code)
	_ = json.NewEncoder(rw).Encode(payload)
}
