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

type aidevGetCommandsRequest struct {
	AfterCursor uint64 `json:"after_cursor"`
	All         bool   `json:"all"`
	Limit       int    `json:"limit"`
	TimeoutMS   int    `json:"timeout_ms"`
}

type aidevSendAsRobotRequest struct {
	Protocol string `json:"protocol"`
	Text     string `json:"text"`
	Channel  string `json:"channel"`
	ThreadID string `json:"thread_id"`
	User     string `json:"user"`
	Direct   bool   `json:"direct"`
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

func serveAIDevGetCommands(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		rw.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if err := authorizeAIDevRequest(req); err != nil {
		writeAIDevError(rw, http.StatusUnauthorized, err)
		return
	}
	var in aidevGetCommandsRequest
	if err := decodeAIDevJSON(req.Body, &in); err != nil {
		writeAIDevError(rw, http.StatusBadRequest, err)
		return
	}
	enabled, user, prefix, consume := aidevCommandConduitInfo()
	if !enabled {
		writeAIDevError(rw, http.StatusBadRequest, errors.New("aidev command conduit is not enabled"))
		return
	}
	res := getAIDevCommands(aidevCommandQuery{
		AfterCursor: in.AfterCursor,
		All:         in.All,
		Limit:       in.Limit,
		TimeoutMS:   in.TimeoutMS,
	})
	payload := map[string]interface{}{
		"user":        user,
		"prefix":      prefix,
		"consume":     consume,
		"commands":    res.Commands,
		"next_cursor": res.NextCursor,
		"latest":      res.Latest,
		"timed_out":   res.TimedOut,
		"has_more":    res.HasMore,
	}
	writeAIDevJSON(rw, payload)
}

func serveAIDevSendAsRobot(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		rw.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if err := authorizeAIDevRequest(req); err != nil {
		writeAIDevError(rw, http.StatusUnauthorized, err)
		return
	}
	var in aidevSendAsRobotRequest
	if err := decodeAIDevJSON(req.Body, &in); err != nil {
		writeAIDevError(rw, http.StatusBadRequest, err)
		return
	}
	text := strings.TrimSpace(in.Text)
	if text == "" {
		writeAIDevError(rw, http.StatusBadRequest, errors.New("text is required"))
		return
	}
	protocol, err := resolveAIDevProtocol(in.Protocol)
	if err != nil {
		writeAIDevError(rw, http.StatusBadRequest, err)
		return
	}
	conn := getConnectorForProtocol(protocol)
	if conn == nil {
		writeAIDevError(rw, http.StatusBadRequest, fmt.Errorf("protocol '%s' is not active", protocol))
		return
	}

	channel := strings.TrimSpace(in.Channel)
	threadID := strings.TrimSpace(in.ThreadID)
	user := strings.TrimSpace(in.User)
	resolvedUser := resolveAIDevUserForProtocol(protocol, user)
	msgObject := &robot.ConnectorMessage{Protocol: protocol}

	var ret robot.RetVal
	switch {
	case in.Direct:
		if user == "" {
			writeAIDevError(rw, http.StatusBadRequest, errors.New("user is required when direct is true"))
			return
		}
		ret = conn.SendProtocolUserMessage(resolvedUser, text, robot.Raw, msgObject)
	case user != "":
		if channel == "" {
			writeAIDevError(rw, http.StatusBadRequest, errors.New("channel is required when user is set and direct is false"))
			return
		}
		ret = conn.SendProtocolUserChannelThreadMessage(resolvedUser, user, channel, threadID, text, robot.Raw, msgObject)
	default:
		if channel == "" {
			writeAIDevError(rw, http.StatusBadRequest, errors.New("channel is required for non-direct send_as_robot"))
			return
		}
		ret = conn.SendProtocolChannelThreadMessage(channel, threadID, text, robot.Raw, msgObject)
	}
	if ret != robot.Ok {
		writeAIDevError(rw, http.StatusBadRequest, fmt.Errorf("connector send failed: %s", ret.String()))
		return
	}
	writeAIDevJSON(rw, map[string]interface{}{
		"protocol":  protocol,
		"channel":   channel,
		"thread_id": threadID,
		"user":      user,
		"direct":    in.Direct,
		"text":      text,
	})
}

func resolveAIDevUserForProtocol(protocol, user string) string {
	trimmed := strings.TrimSpace(user)
	if trimmed == "" {
		return trimmed
	}
	if _, ok := handle.ExtractID(trimmed); ok {
		return trimmed
	}
	name := strings.ToLower(trimmed)
	currentUCMaps.Lock()
	maps := currentUCMaps.ucmap
	currentUCMaps.Unlock()
	if maps == nil {
		return trimmed
	}
	if pm, ok := maps.userProto[protocol]; ok {
		if ui, ok := pm[name]; ok && ui != nil && ui.UserID != "" {
			return bracket(ui.UserID)
		}
	}
	return trimmed
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
