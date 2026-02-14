package bot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
	luamod "github.com/lnxjedi/gopherbot/v2/modules/lua"
	"golang.org/x/sys/unix"
)

type pipelineRPCRobotOptions struct {
	Direct   bool `json:"direct,omitempty"`
	Threaded bool `json:"threaded,omitempty"`
	Format   *int `json:"format,omitempty"`
}

type pipelineRPCRobotCallRequest struct {
	Method  string                  `json:"method"`
	Options pipelineRPCRobotOptions `json:"options,omitempty"`
	Args    []interface{}           `json:"args,omitempty"`
}

type pipelineRPCLuaRunRequest struct {
	ExecPath string            `json:"exec_path"`
	TaskPath string            `json:"task_path"`
	TaskName string            `json:"task_name"`
	PkgPath  []string          `json:"pkg_path"`
	Bot      map[string]string `json:"bot"`
	Args     []string          `json:"args"`
}

type pipelineRPCLuaRunResponse struct {
	RetVal int    `json:"ret_val"`
	Error  string `json:"error,omitempty"`
}

type pipelineRPCLuaGetConfigRequest struct {
	ExecPath string            `json:"exec_path"`
	TaskPath string            `json:"task_path"`
	TaskName string            `json:"task_name"`
	PkgPath  []string          `json:"pkg_path"`
	Bot      map[string]string `json:"bot"`
}

type pipelineRPCLuaGetConfigResponse struct {
	Config string `json:"config,omitempty"`
	Error  string `json:"error,omitempty"`
}

func runLuaExtensionViaRPC(taskPath, taskName string, pkgPath []string, bot map[string]string, w *worker, r robot.Robot, args []string) (robot.TaskRetVal, error) {
	params := pipelineRPCLuaRunRequest{
		ExecPath: execPath(),
		TaskPath: taskPath,
		TaskName: taskName,
		PkgPath:  pkgPath,
		Bot:      bot,
		Args:     args,
	}
	resRaw, err := runPipelineRPCRequest("lua_run", params, w, r)
	if err != nil {
		return robot.MechanismFail, err
	}
	var res pipelineRPCLuaRunResponse
	if err := json.Unmarshal(resRaw, &res); err != nil {
		return robot.MechanismFail, fmt.Errorf("decoding lua_run response: %v", err)
	}
	if res.Error != "" {
		ret := robot.TaskRetVal(res.RetVal)
		if ret == robot.Normal {
			ret = robot.MechanismFail
		}
		return ret, errors.New(res.Error)
	}
	return robot.TaskRetVal(res.RetVal), nil
}

func runLuaGetConfigViaRPC(taskPath, taskName string, pkgPath []string, bot map[string]string) (*[]byte, error) {
	params := pipelineRPCLuaGetConfigRequest{
		ExecPath: execPath(),
		TaskPath: taskPath,
		TaskName: taskName,
		PkgPath:  pkgPath,
		Bot:      bot,
	}
	resRaw, err := runPipelineRPCRequest("lua_get_config", params, nil, nil)
	if err != nil {
		return nil, err
	}
	var res pipelineRPCLuaGetConfigResponse
	if err := json.Unmarshal(resRaw, &res); err != nil {
		return nil, fmt.Errorf("decoding lua_get_config response: %v", err)
	}
	if res.Error != "" {
		return nil, errors.New(res.Error)
	}
	cfg := []byte(res.Config)
	return &cfg, nil
}

const (
	pipelineRPCHelloTimeout     = 5 * time.Second
	pipelineRPCShutdownTimeout  = 3 * time.Second
	pipelineRPCGetConfigTimeout = 20 * time.Second
	pipelineRPCRunTimeout       = 24 * time.Hour
	pipelineRPCChildWaitTimeout = 3 * time.Second
)

type pipelineRPCError struct {
	Code   string
	Method string
	Detail string
	Cause  error
}

func (e *pipelineRPCError) Error() string {
	label := "pipeline rpc"
	if e.Method != "" {
		label += " (" + e.Method + ")"
	}
	if e.Code != "" {
		label += " [" + e.Code + "]"
	}
	if e.Detail == "" && e.Cause != nil {
		return label + ": " + e.Cause.Error()
	}
	if e.Cause != nil {
		return label + ": " + e.Detail + ": " + e.Cause.Error()
	}
	if e.Detail != "" {
		return label + ": " + e.Detail
	}
	return label
}

func (e *pipelineRPCError) Unwrap() error {
	return e.Cause
}

func newPipelineRPCError(code, method, detail string, cause error) error {
	return &pipelineRPCError{
		Code:   code,
		Method: method,
		Detail: detail,
		Cause:  cause,
	}
}

func rpcMethodTimeout(method string) time.Duration {
	if strings.HasSuffix(method, "_run") {
		return pipelineRPCRunTimeout
	}
	return pipelineRPCGetConfigTimeout
}

func readPipelineRPCMessageWithContext(dec *json.Decoder, ctx context.Context, method string) (pipelineRPCMessage, error) {
	type msgResult struct {
		msg pipelineRPCMessage
		err error
	}
	ch := make(chan msgResult, 1)
	go func() {
		msg, err := readPipelineRPCMessage(dec)
		ch <- msgResult{msg: msg, err: err}
	}()
	select {
	case <-ctx.Done():
		switch ctx.Err() {
		case context.Canceled:
			return pipelineRPCMessage{}, newPipelineRPCError("canceled", method, "rpc request canceled", ctx.Err())
		case context.DeadlineExceeded:
			return pipelineRPCMessage{}, newPipelineRPCError("timeout", method, "rpc response timed out", ctx.Err())
		default:
			return pipelineRPCMessage{}, newPipelineRPCError("context_error", method, "rpc context error", ctx.Err())
		}
	case res := <-ch:
		if res.err != nil {
			if errors.Is(res.err, io.EOF) {
				return pipelineRPCMessage{}, newPipelineRPCError("child_exit", method, "rpc child closed stream", res.err)
			}
			return pipelineRPCMessage{}, newPipelineRPCError("io_error", method, "reading rpc message", res.err)
		}
		return res.msg, nil
	}
}

func terminatePipelineRPCChild(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	pid := cmd.Process.Pid
	if pid <= 0 {
		return
	}
	if err := unix.Kill(-pid, unix.SIGKILL); err != nil && !errors.Is(err, unix.ESRCH) {
		_ = cmd.Process.Kill()
	}
}

func runPipelineRPCRequest(method string, params interface{}, w *worker, r robot.Robot) (json.RawMessage, error) {
	cmd := newPipelineChildRPCCommand()
	cmd.SysProcAttr = &unix.SysProcAttr{Setpgid: true}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, newPipelineRPCError("io_error", method, "creating rpc stdin pipe", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, newPipelineRPCError("io_error", method, "creating rpc stdout pipe", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, newPipelineRPCError("io_error", method, "creating rpc stderr pipe", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, newPipelineRPCError("child_start", method, "starting rpc child", err)
	}
	defer stdin.Close()

	reqCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if w != nil {
		w.Lock()
		w.osCmd = cmd
		w.rpcCancel = cancel
		w.Unlock()
		defer func() {
			w.Lock()
			if w.osCmd == cmd {
				w.osCmd = nil
			}
			if w.rpcCancel != nil {
				w.rpcCancel = nil
			}
			w.Unlock()
		}()
	}

	var stderrBuf bytes.Buffer
	stderrDone := make(chan struct{}, 1)
	go func() {
		_, _ = io.Copy(&stderrBuf, stderr)
		stderrDone <- struct{}{}
	}()
	waitCh := make(chan error, 1)
	go func() {
		waitCh <- cmd.Wait()
	}()

	enc := json.NewEncoder(stdin)
	dec := json.NewDecoder(stdout)

	if err := enc.Encode(pipelineRPCMessage{Version: pipelineRPCProtocolVersion, ID: "hello", Type: "hello"}); err != nil {
		terminatePipelineRPCChild(cmd)
		<-waitCh
		<-stderrDone
		return nil, newPipelineRPCError("protocol_error", method, "sending rpc hello", err)
	}
	helloCtx, helloCancel := context.WithTimeout(reqCtx, pipelineRPCHelloTimeout)
	_, helloErr := waitPipelineRPCResponse(dec, enc, "hello", r, helloCtx, method)
	helloCancel()
	if helloErr != nil {
		terminatePipelineRPCChild(cmd)
		<-waitCh
		<-stderrDone
		stderrOut := strings.TrimSpace(stderrBuf.String())
		if stderrOut != "" {
			return nil, newPipelineRPCError("protocol_error", method, "rpc hello failed (child stderr: "+stderrOut+")", helloErr)
		}
		return nil, newPipelineRPCError("protocol_error", method, "rpc hello failed", helloErr)
	}

	paramsRaw, err := json.Marshal(params)
	if err != nil {
		terminatePipelineRPCChild(cmd)
		<-waitCh
		<-stderrDone
		return nil, newPipelineRPCError("encoding_error", method, "encoding rpc request params", err)
	}
	requestID := "req-1"
	if err := enc.Encode(pipelineRPCMessage{Version: pipelineRPCProtocolVersion, ID: requestID, Type: "request", Method: method, Params: paramsRaw}); err != nil {
		terminatePipelineRPCChild(cmd)
		<-waitCh
		<-stderrDone
		return nil, newPipelineRPCError("protocol_error", method, "sending rpc request", err)
	}

	reqTimeout := rpcMethodTimeout(method)
	reqWaitCtx := reqCtx
	var reqCancel context.CancelFunc
	if reqTimeout > 0 {
		reqWaitCtx, reqCancel = context.WithTimeout(reqCtx, reqTimeout)
	} else {
		reqWaitCtx, reqCancel = context.WithCancel(reqCtx)
	}
	result, reqErr := waitPipelineRPCResponse(dec, enc, requestID, r, reqWaitCtx, method)
	reqCancel()

	shutdownErr := error(nil)
	if reqErr == nil {
		if err := enc.Encode(pipelineRPCMessage{Version: pipelineRPCProtocolVersion, ID: "shutdown", Type: "request", Method: "shutdown"}); err != nil {
			shutdownErr = newPipelineRPCError("protocol_error", method, "sending rpc shutdown", err)
		} else {
			shutdownCtx, shutdownCancel := context.WithTimeout(reqCtx, pipelineRPCShutdownTimeout)
			_, shutdownErr = waitPipelineRPCResponse(dec, enc, "shutdown", r, shutdownCtx, method)
			shutdownCancel()
			if shutdownErr != nil {
				shutdownErr = newPipelineRPCError("protocol_error", method, "waiting for rpc shutdown", shutdownErr)
			}
		}
	}
	if reqErr != nil || shutdownErr != nil {
		terminatePipelineRPCChild(cmd)
	}
	var waitErr error
	select {
	case waitErr = <-waitCh:
	case <-time.After(pipelineRPCChildWaitTimeout):
		terminatePipelineRPCChild(cmd)
		select {
		case waitErr = <-waitCh:
		case <-time.After(pipelineRPCChildWaitTimeout):
			waitErr = context.DeadlineExceeded
		}
	}
	<-stderrDone
	stderrOut := strings.TrimSpace(stderrBuf.String())

	if reqErr != nil {
		if stderrOut == "" {
			return nil, reqErr
		}
		return nil, newPipelineRPCError("request_failed", method, "rpc request failed (child stderr: "+stderrOut+")", reqErr)
	}
	if shutdownErr != nil {
		if stderrOut == "" {
			return nil, shutdownErr
		}
		return nil, newPipelineRPCError("shutdown_failed", method, "rpc shutdown failed (child stderr: "+stderrOut+")", shutdownErr)
	}
	if waitErr != nil {
		if errors.Is(waitErr, context.DeadlineExceeded) {
			return nil, newPipelineRPCError("child_timeout", method, "rpc child did not exit in time", waitErr)
		}
		if stderrOut != "" {
			return nil, newPipelineRPCError("child_exit", method, "rpc child exit (stderr: "+stderrOut+")", waitErr)
		}
		return nil, newPipelineRPCError("child_exit", method, "rpc child exit", waitErr)
	}
	return result, nil
}

func waitPipelineRPCResponse(dec *json.Decoder, enc *json.Encoder, targetID string, r robot.Robot, ctx context.Context, method string) (json.RawMessage, error) {
	for {
		msg, err := readPipelineRPCMessageWithContext(dec, ctx, method)
		if err != nil {
			return nil, err
		}
		switch msg.Type {
		case "response", "hello_ack":
			if msg.ID == targetID {
				return msg.Result, nil
			}
		case "error":
			if msg.ID == targetID {
				if msg.Error == nil {
					return nil, newPipelineRPCError("protocol_error", method, "rpc error with empty payload", nil)
				}
				return nil, newPipelineRPCError(msg.Error.Code, method, msg.Error.Message, nil)
			}
		case "request":
			if msg.Method != "robot_call" {
				_ = writePipelineRPCError(enc, msg.ID, "method_not_found", fmt.Sprintf("unsupported method '%s'", msg.Method))
				continue
			}
			if r == nil {
				_ = writePipelineRPCError(enc, msg.ID, "invalid_state", "robot is not available for this rpc request")
				continue
			}
			res, handleErr := handlePipelineRPCRobotCall(msg.Params, r)
			if handleErr != nil {
				_ = writePipelineRPCError(enc, msg.ID, "robot_call_failed", handleErr.Error())
				continue
			}
			if err := writePipelineRPCResponse(enc, msg.ID, res); err != nil {
				return nil, newPipelineRPCError("protocol_error", method, "writing rpc response", err)
			}
		default:
			// Ignore unexpected message types.
		}
	}
}

func writePipelineRPCResponse(enc *json.Encoder, id string, result interface{}) error {
	resultRaw, err := json.Marshal(result)
	if err != nil {
		return err
	}
	return enc.Encode(pipelineRPCMessage{
		Version: pipelineRPCProtocolVersion,
		ID:      id,
		Type:    "response",
		Result:  resultRaw,
	})
}

func handlePipelineRPCLuaRun(dec *json.Decoder, enc *json.Encoder, msg pipelineRPCMessage) error {
	var req pipelineRPCLuaRunRequest
	if err := json.Unmarshal(msg.Params, &req); err != nil {
		return writePipelineRPCError(enc, msg.ID, "invalid_params", fmt.Sprintf("invalid lua_run params: %v", err))
	}
	client := newPipelineRPCLuaRobotClient(dec, enc, req.Bot)
	ret, err := luamod.CallExtension(req.ExecPath, req.TaskPath, req.TaskName, req.PkgPath, client, req.Bot, client, req.Args)
	res := pipelineRPCLuaRunResponse{RetVal: int(ret)}
	if err != nil {
		res.Error = err.Error()
	}
	return writePipelineRPCResponse(enc, msg.ID, res)
}

func handlePipelineRPCLuaGetConfig(enc *json.Encoder, msg pipelineRPCMessage) error {
	var req pipelineRPCLuaGetConfigRequest
	if err := json.Unmarshal(msg.Params, &req); err != nil {
		return writePipelineRPCError(enc, msg.ID, "invalid_params", fmt.Sprintf("invalid lua_get_config params: %v", err))
	}
	cfg, err := luamod.GetPluginConfig(req.ExecPath, req.TaskPath, req.TaskName, req.Bot, req.PkgPath)
	res := pipelineRPCLuaGetConfigResponse{}
	if err != nil {
		res.Error = err.Error()
	} else if cfg != nil {
		res.Config = string(*cfg)
	}
	return writePipelineRPCResponse(enc, msg.ID, res)
}

func handlePipelineRPCRobotCall(paramsRaw json.RawMessage, base robot.Robot) (map[string]interface{}, error) {
	var req pipelineRPCRobotCallRequest
	if err := json.Unmarshal(paramsRaw, &req); err != nil {
		return nil, err
	}
	if base == nil {
		return nil, fmt.Errorf("nil robot")
	}
	r := applyPipelineRPCRobotOptions(base, req.Options)
	args := req.Args

	switch req.Method {
	case "CheckAdmin":
		return map[string]interface{}{"bool": r.CheckAdmin()}, nil
	case "Elevate":
		immediate, err := pipelineRPCArgBool(args, 0)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"bool": r.Elevate(immediate)}, nil
	case "GetBotAttribute":
		a, err := pipelineRPCArgString(args, 0)
		if err != nil {
			return nil, err
		}
		attr := r.GetBotAttribute(a)
		if attr == nil {
			return map[string]interface{}{"attribute": "", "ret_val": int(robot.AttributeNotFound)}, nil
		}
		return map[string]interface{}{"attribute": attr.Attribute, "ret_val": int(attr.RetVal)}, nil
	case "GetUserAttribute":
		u, err := pipelineRPCArgString(args, 0)
		if err != nil {
			return nil, err
		}
		a, err := pipelineRPCArgString(args, 1)
		if err != nil {
			return nil, err
		}
		attr := r.GetUserAttribute(u, a)
		if attr == nil {
			return map[string]interface{}{"attribute": "", "ret_val": int(robot.AttributeNotFound)}, nil
		}
		return map[string]interface{}{"attribute": attr.Attribute, "ret_val": int(attr.RetVal)}, nil
	case "GetSenderAttribute":
		a, err := pipelineRPCArgString(args, 0)
		if err != nil {
			return nil, err
		}
		attr := r.GetSenderAttribute(a)
		if attr == nil {
			return map[string]interface{}{"attribute": "", "ret_val": int(robot.AttributeNotFound)}, nil
		}
		return map[string]interface{}{"attribute": attr.Attribute, "ret_val": int(attr.RetVal)}, nil
	case "GetTaskConfig":
		mapCfg := make(map[string]interface{})
		ret := r.GetTaskConfig(&mapCfg)
		if ret == robot.Ok {
			return map[string]interface{}{"ret_val": int(ret), "config": mapCfg}, nil
		}
		if ret == robot.ConfigUnmarshalError {
			var sliceCfg []interface{}
			ret = r.GetTaskConfig(&sliceCfg)
			if ret == robot.Ok {
				return map[string]interface{}{"ret_val": int(ret), "config": sliceCfg}, nil
			}
		}
		return map[string]interface{}{"ret_val": int(ret), "config": nil}, nil
	case "GetMessage":
		msg := r.GetMessage()
		if msg == nil {
			return map[string]interface{}{"message": nil}, nil
		}
		return map[string]interface{}{
			"message": map[string]interface{}{
				"user":             msg.User,
				"protocol_user":    msg.ProtocolUser,
				"channel":          msg.Channel,
				"protocol_channel": msg.ProtocolChannel,
				"protocol":         int(msg.Protocol),
				"format":           int(msg.Format),
			},
		}, nil
	case "GetParameter":
		name, err := pipelineRPCArgString(args, 0)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"string": r.GetParameter(name)}, nil
	case "Email":
		subject, err := pipelineRPCArgString(args, 0)
		if err != nil {
			return nil, err
		}
		body, err := pipelineRPCArgString(args, 1)
		if err != nil {
			return nil, err
		}
		html := false
		if len(args) > 2 {
			html, err = pipelineRPCArgBool(args, 2)
			if err != nil {
				return nil, err
			}
		}
		buf := bytes.NewBufferString(body)
		return map[string]interface{}{"ret_val": int(r.Email(subject, buf, html))}, nil
	case "EmailUser":
		user, err := pipelineRPCArgString(args, 0)
		if err != nil {
			return nil, err
		}
		subject, err := pipelineRPCArgString(args, 1)
		if err != nil {
			return nil, err
		}
		body, err := pipelineRPCArgString(args, 2)
		if err != nil {
			return nil, err
		}
		html := false
		if len(args) > 3 {
			html, err = pipelineRPCArgBool(args, 3)
			if err != nil {
				return nil, err
			}
		}
		buf := bytes.NewBufferString(body)
		return map[string]interface{}{"ret_val": int(r.EmailUser(user, subject, buf, html))}, nil
	case "EmailAddress":
		address, err := pipelineRPCArgString(args, 0)
		if err != nil {
			return nil, err
		}
		subject, err := pipelineRPCArgString(args, 1)
		if err != nil {
			return nil, err
		}
		body, err := pipelineRPCArgString(args, 2)
		if err != nil {
			return nil, err
		}
		html := false
		if len(args) > 3 {
			html, err = pipelineRPCArgBool(args, 3)
			if err != nil {
				return nil, err
			}
		}
		buf := bytes.NewBufferString(body)
		return map[string]interface{}{"ret_val": int(r.EmailAddress(address, subject, buf, html))}, nil
	case "Exclusive":
		tag, err := pipelineRPCArgString(args, 0)
		if err != nil {
			return nil, err
		}
		queueTask, err := pipelineRPCArgBool(args, 1)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"bool": r.Exclusive(tag, queueTask)}, nil
	case "Log":
		level, err := pipelineRPCArgInt(args, 0)
		if err != nil {
			return nil, err
		}
		msg, err := pipelineRPCArgString(args, 1)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"bool": r.Log(robot.LogLevel(level), msg)}, nil
	case "SendChannelMessage":
		ch, err := pipelineRPCArgString(args, 0)
		if err != nil {
			return nil, err
		}
		msg, err := pipelineRPCArgString(args, 1)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"ret_val": int(r.SendChannelMessage(ch, msg))}, nil
	case "SendChannelThreadMessage":
		ch, err := pipelineRPCArgString(args, 0)
		if err != nil {
			return nil, err
		}
		thr, err := pipelineRPCArgString(args, 1)
		if err != nil {
			return nil, err
		}
		msg, err := pipelineRPCArgString(args, 2)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"ret_val": int(r.SendChannelThreadMessage(ch, thr, msg))}, nil
	case "SendUserChannelMessage":
		u, err := pipelineRPCArgString(args, 0)
		if err != nil {
			return nil, err
		}
		ch, err := pipelineRPCArgString(args, 1)
		if err != nil {
			return nil, err
		}
		msg, err := pipelineRPCArgString(args, 2)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"ret_val": int(r.SendUserChannelMessage(u, ch, msg))}, nil
	case "SendProtocolUserChannelMessage":
		protocol, err := pipelineRPCArgString(args, 0)
		if err != nil {
			return nil, err
		}
		u, err := pipelineRPCArgString(args, 1)
		if err != nil {
			return nil, err
		}
		ch, err := pipelineRPCArgString(args, 2)
		if err != nil {
			return nil, err
		}
		msg, err := pipelineRPCArgString(args, 3)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"ret_val": int(r.SendProtocolUserChannelMessage(protocol, u, ch, msg))}, nil
	case "SendUserChannelThreadMessage":
		u, err := pipelineRPCArgString(args, 0)
		if err != nil {
			return nil, err
		}
		ch, err := pipelineRPCArgString(args, 1)
		if err != nil {
			return nil, err
		}
		thr, err := pipelineRPCArgString(args, 2)
		if err != nil {
			return nil, err
		}
		msg, err := pipelineRPCArgString(args, 3)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"ret_val": int(r.SendUserChannelThreadMessage(u, ch, thr, msg))}, nil
	case "SendUserMessage":
		u, err := pipelineRPCArgString(args, 0)
		if err != nil {
			return nil, err
		}
		msg, err := pipelineRPCArgString(args, 1)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"ret_val": int(r.SendUserMessage(u, msg))}, nil
	case "Reply":
		msg, err := pipelineRPCArgString(args, 0)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"ret_val": int(r.Reply(msg))}, nil
	case "ReplyThread":
		msg, err := pipelineRPCArgString(args, 0)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"ret_val": int(r.ReplyThread(msg))}, nil
	case "Say":
		msg, err := pipelineRPCArgString(args, 0)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"ret_val": int(r.Say(msg))}, nil
	case "SayThread":
		msg, err := pipelineRPCArgString(args, 0)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"ret_val": int(r.SayThread(msg))}, nil
	case "RandomInt":
		n, err := pipelineRPCArgInt(args, 0)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"int": r.RandomInt(n)}, nil
	case "RandomString":
		s, err := pipelineRPCArgStringSlice(args, 0)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"string": r.RandomString(s)}, nil
	case "Pause":
		s, err := pipelineRPCArgFloat(args, 0)
		if err != nil {
			return nil, err
		}
		r.Pause(s)
		return map[string]interface{}{"ok": true}, nil
	case "PromptForReply":
		regexID, err := pipelineRPCArgString(args, 0)
		if err != nil {
			return nil, err
		}
		prompt, err := pipelineRPCArgString(args, 1)
		if err != nil {
			return nil, err
		}
		reply, ret := r.PromptForReply(regexID, prompt)
		return map[string]interface{}{"reply": reply, "ret_val": int(ret)}, nil
	case "PromptThreadForReply":
		regexID, err := pipelineRPCArgString(args, 0)
		if err != nil {
			return nil, err
		}
		prompt, err := pipelineRPCArgString(args, 1)
		if err != nil {
			return nil, err
		}
		reply, ret := r.PromptThreadForReply(regexID, prompt)
		return map[string]interface{}{"reply": reply, "ret_val": int(ret)}, nil
	case "PromptUserForReply":
		regexID, err := pipelineRPCArgString(args, 0)
		if err != nil {
			return nil, err
		}
		user, err := pipelineRPCArgString(args, 1)
		if err != nil {
			return nil, err
		}
		prompt, err := pipelineRPCArgString(args, 2)
		if err != nil {
			return nil, err
		}
		reply, ret := r.PromptUserForReply(regexID, user, prompt)
		return map[string]interface{}{"reply": reply, "ret_val": int(ret)}, nil
	case "PromptUserChannelForReply":
		regexID, err := pipelineRPCArgString(args, 0)
		if err != nil {
			return nil, err
		}
		user, err := pipelineRPCArgString(args, 1)
		if err != nil {
			return nil, err
		}
		channel, err := pipelineRPCArgString(args, 2)
		if err != nil {
			return nil, err
		}
		prompt, err := pipelineRPCArgString(args, 3)
		if err != nil {
			return nil, err
		}
		reply, ret := r.PromptUserChannelForReply(regexID, user, channel, prompt)
		return map[string]interface{}{"reply": reply, "ret_val": int(ret)}, nil
	case "PromptUserChannelThreadForReply":
		regexID, err := pipelineRPCArgString(args, 0)
		if err != nil {
			return nil, err
		}
		user, err := pipelineRPCArgString(args, 1)
		if err != nil {
			return nil, err
		}
		channel, err := pipelineRPCArgString(args, 2)
		if err != nil {
			return nil, err
		}
		thread, err := pipelineRPCArgString(args, 3)
		if err != nil {
			return nil, err
		}
		prompt, err := pipelineRPCArgString(args, 4)
		if err != nil {
			return nil, err
		}
		reply, ret := r.PromptUserChannelThreadForReply(regexID, user, channel, thread, prompt)
		return map[string]interface{}{"reply": reply, "ret_val": int(ret)}, nil
	case "CheckoutDatum":
		key, err := pipelineRPCArgString(args, 0)
		if err != nil {
			return nil, err
		}
		rw, err := pipelineRPCArgBool(args, 1)
		if err != nil {
			return nil, err
		}
		var datum interface{}
		lockToken, exists, ret := r.CheckoutDatum(key, &datum, rw)
		return map[string]interface{}{"ret_val": int(ret), "exists": exists, "lock_token": lockToken, "datum": datum}, nil
	case "CheckinDatum":
		key, err := pipelineRPCArgString(args, 0)
		if err != nil {
			return nil, err
		}
		lockToken, err := pipelineRPCArgString(args, 1)
		if err != nil {
			return nil, err
		}
		r.CheckinDatum(key, lockToken)
		return map[string]interface{}{"ok": true}, nil
	case "UpdateDatum":
		key, err := pipelineRPCArgString(args, 0)
		if err != nil {
			return nil, err
		}
		lockToken, err := pipelineRPCArgString(args, 1)
		if err != nil {
			return nil, err
		}
		datum, err := pipelineRPCArgAny(args, 2)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"ret_val": int(r.UpdateDatum(key, lockToken, datum))}, nil
	case "Remember":
		key, err := pipelineRPCArgString(args, 0)
		if err != nil {
			return nil, err
		}
		value, err := pipelineRPCArgString(args, 1)
		if err != nil {
			return nil, err
		}
		shared, err := pipelineRPCArgBool(args, 2)
		if err != nil {
			return nil, err
		}
		r.Remember(key, value, shared)
		return map[string]interface{}{"ok": true}, nil
	case "RememberThread":
		key, err := pipelineRPCArgString(args, 0)
		if err != nil {
			return nil, err
		}
		value, err := pipelineRPCArgString(args, 1)
		if err != nil {
			return nil, err
		}
		shared, err := pipelineRPCArgBool(args, 2)
		if err != nil {
			return nil, err
		}
		r.RememberThread(key, value, shared)
		return map[string]interface{}{"ok": true}, nil
	case "RememberContext":
		context, err := pipelineRPCArgString(args, 0)
		if err != nil {
			return nil, err
		}
		value, err := pipelineRPCArgString(args, 1)
		if err != nil {
			return nil, err
		}
		r.RememberContext(context, value)
		return map[string]interface{}{"ok": true}, nil
	case "RememberContextThread":
		context, err := pipelineRPCArgString(args, 0)
		if err != nil {
			return nil, err
		}
		value, err := pipelineRPCArgString(args, 1)
		if err != nil {
			return nil, err
		}
		r.RememberContextThread(context, value)
		return map[string]interface{}{"ok": true}, nil
	case "Recall":
		key, err := pipelineRPCArgString(args, 0)
		if err != nil {
			return nil, err
		}
		shared, err := pipelineRPCArgBool(args, 1)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"string": r.Recall(key, shared)}, nil
	case "SpawnJob":
		name, err := pipelineRPCArgString(args, 0)
		if err != nil {
			return nil, err
		}
		extras, err := pipelineRPCArgTailStrings(args, 1)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"ret_val": int(r.SpawnJob(name, extras...))}, nil
	case "AddTask":
		name, err := pipelineRPCArgString(args, 0)
		if err != nil {
			return nil, err
		}
		extras, err := pipelineRPCArgTailStrings(args, 1)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"ret_val": int(r.AddTask(name, extras...))}, nil
	case "FinalTask":
		name, err := pipelineRPCArgString(args, 0)
		if err != nil {
			return nil, err
		}
		extras, err := pipelineRPCArgTailStrings(args, 1)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"ret_val": int(r.FinalTask(name, extras...))}, nil
	case "FailTask":
		name, err := pipelineRPCArgString(args, 0)
		if err != nil {
			return nil, err
		}
		extras, err := pipelineRPCArgTailStrings(args, 1)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"ret_val": int(r.FailTask(name, extras...))}, nil
	case "AddJob":
		name, err := pipelineRPCArgString(args, 0)
		if err != nil {
			return nil, err
		}
		extras, err := pipelineRPCArgTailStrings(args, 1)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"ret_val": int(r.AddJob(name, extras...))}, nil
	case "AddCommand":
		plugin, err := pipelineRPCArgString(args, 0)
		if err != nil {
			return nil, err
		}
		command, err := pipelineRPCArgString(args, 1)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"ret_val": int(r.AddCommand(plugin, command))}, nil
	case "FinalCommand":
		plugin, err := pipelineRPCArgString(args, 0)
		if err != nil {
			return nil, err
		}
		command, err := pipelineRPCArgString(args, 1)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"ret_val": int(r.FinalCommand(plugin, command))}, nil
	case "FailCommand":
		plugin, err := pipelineRPCArgString(args, 0)
		if err != nil {
			return nil, err
		}
		command, err := pipelineRPCArgString(args, 1)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"ret_val": int(r.FailCommand(plugin, command))}, nil
	case "SetParameter":
		name, err := pipelineRPCArgString(args, 0)
		if err != nil {
			return nil, err
		}
		value, err := pipelineRPCArgString(args, 1)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"bool": r.SetParameter(name, value)}, nil
	case "Subscribe":
		subscriber, ok := base.(interface{ Subscribe() bool })
		if !ok {
			return nil, fmt.Errorf("Subscribe unsupported by current robot implementation")
		}
		return map[string]interface{}{"bool": subscriber.Subscribe()}, nil
	case "Unsubscribe":
		subscriber, ok := base.(interface{ Unsubscribe() bool })
		if !ok {
			return nil, fmt.Errorf("Unsubscribe unsupported by current robot implementation")
		}
		return map[string]interface{}{"bool": subscriber.Unsubscribe()}, nil
	case "RaisePriv":
		reason, err := pipelineRPCArgString(args, 0)
		if err != nil {
			return nil, err
		}
		r.RaisePriv(reason)
		return map[string]interface{}{"ok": true}, nil
	case "SetWorkingDirectory":
		path, err := pipelineRPCArgString(args, 0)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"bool": r.SetWorkingDirectory(path)}, nil
	default:
		return nil, fmt.Errorf("unsupported robot call '%s'", req.Method)
	}
}

func applyPipelineRPCRobotOptions(base robot.Robot, opts pipelineRPCRobotOptions) robot.Robot {
	r := base
	if opts.Direct {
		r = r.Direct()
	}
	if opts.Threaded {
		r = r.Threaded()
	}
	if opts.Format != nil {
		r = r.MessageFormat(robot.MessageFormat(*opts.Format))
	}
	return r
}

type pipelineRPCLuaRobotClient struct {
	dec  *json.Decoder
	enc  *json.Encoder
	seq  int
	bot  map[string]string
	opts pipelineRPCRobotOptions
}

func newPipelineRPCLuaRobotClient(dec *json.Decoder, enc *json.Encoder, bot map[string]string) *pipelineRPCLuaRobotClient {
	cpy := make(map[string]string, len(bot))
	for k, v := range bot {
		cpy[k] = v
	}
	return &pipelineRPCLuaRobotClient{dec: dec, enc: enc, bot: cpy}
}

func (c *pipelineRPCLuaRobotClient) clone() *pipelineRPCLuaRobotClient {
	clone := *c
	clone.bot = make(map[string]string, len(c.bot))
	for k, v := range c.bot {
		clone.bot[k] = v
	}
	return &clone
}

func (c *pipelineRPCLuaRobotClient) call(method string, args ...interface{}) (map[string]interface{}, error) {
	c.seq++
	id := fmt.Sprintf("robot-%d", c.seq)
	params := pipelineRPCRobotCallRequest{Method: method, Options: c.opts, Args: args}
	paramsRaw, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	if err := c.enc.Encode(pipelineRPCMessage{Version: pipelineRPCProtocolVersion, ID: id, Type: "request", Method: "robot_call", Params: paramsRaw}); err != nil {
		return nil, err
	}
	for {
		msg, err := readPipelineRPCMessage(c.dec)
		if err != nil {
			return nil, err
		}
		switch msg.Type {
		case "response":
			if msg.ID != id {
				continue
			}
			if len(msg.Result) == 0 {
				return map[string]interface{}{}, nil
			}
			var res map[string]interface{}
			if err := json.Unmarshal(msg.Result, &res); err != nil {
				return nil, err
			}
			return res, nil
		case "error":
			if msg.ID != id {
				continue
			}
			if msg.Error == nil {
				return nil, fmt.Errorf("rpc robot call failed")
			}
			return nil, fmt.Errorf("%s: %s", msg.Error.Code, msg.Error.Message)
		default:
			// Ignore non-response messages while waiting.
		}
	}
}

func (c *pipelineRPCLuaRobotClient) CheckAdmin() bool {
	res, err := c.call("CheckAdmin")
	if err != nil {
		return false
	}
	return pipelineRPCMapBool(res, "bool")
}

func (c *pipelineRPCLuaRobotClient) Elevate(immediate bool) bool {
	res, err := c.call("Elevate", immediate)
	if err != nil {
		return false
	}
	return pipelineRPCMapBool(res, "bool")
}

func (c *pipelineRPCLuaRobotClient) GetBotAttribute(a string) *robot.AttrRet {
	res, err := c.call("GetBotAttribute", a)
	if err != nil {
		return &robot.AttrRet{RetVal: robot.AttributeNotFound}
	}
	return &robot.AttrRet{Attribute: pipelineRPCMapString(res, "attribute"), RetVal: robot.RetVal(pipelineRPCMapInt(res, "ret_val"))}
}

func (c *pipelineRPCLuaRobotClient) GetUserAttribute(u, a string) *robot.AttrRet {
	res, err := c.call("GetUserAttribute", u, a)
	if err != nil {
		return &robot.AttrRet{RetVal: robot.AttributeNotFound}
	}
	return &robot.AttrRet{Attribute: pipelineRPCMapString(res, "attribute"), RetVal: robot.RetVal(pipelineRPCMapInt(res, "ret_val"))}
}

func (c *pipelineRPCLuaRobotClient) GetSenderAttribute(a string) *robot.AttrRet {
	res, err := c.call("GetSenderAttribute", a)
	if err != nil {
		return &robot.AttrRet{RetVal: robot.AttributeNotFound}
	}
	return &robot.AttrRet{Attribute: pipelineRPCMapString(res, "attribute"), RetVal: robot.RetVal(pipelineRPCMapInt(res, "ret_val"))}
}

func (c *pipelineRPCLuaRobotClient) GetTaskConfig(cfgptr interface{}) robot.RetVal {
	res, err := c.call("GetTaskConfig")
	if err != nil {
		return robot.Failed
	}
	ret := robot.RetVal(pipelineRPCMapInt(res, "ret_val"))
	cfg, ok := res["config"]
	if !ok || cfg == nil || cfgptr == nil {
		return ret
	}
	blob, err := json.Marshal(cfg)
	if err != nil {
		return robot.ConfigUnmarshalError
	}
	if err := json.Unmarshal(blob, cfgptr); err != nil {
		return robot.ConfigUnmarshalError
	}
	return ret
}

func (c *pipelineRPCLuaRobotClient) GetParameter(name string) string {
	res, err := c.call("GetParameter", name)
	if err != nil {
		return ""
	}
	return pipelineRPCMapString(res, "string")
}

func (c *pipelineRPCLuaRobotClient) Exclusive(tag string, queueTask bool) bool {
	res, err := c.call("Exclusive", tag, queueTask)
	if err != nil {
		return false
	}
	return pipelineRPCMapBool(res, "bool")
}

func (c *pipelineRPCLuaRobotClient) Fixed() luamod.BotAPI {
	clone := c.clone()
	f := int(robot.Fixed)
	clone.opts.Format = &f
	return clone
}

func (c *pipelineRPCLuaRobotClient) MessageFormat(f robot.MessageFormat) luamod.BotAPI {
	clone := c.clone()
	fi := int(f)
	clone.opts.Format = &fi
	return clone
}

func (c *pipelineRPCLuaRobotClient) Direct() luamod.BotAPI {
	clone := c.clone()
	clone.opts.Direct = true
	return clone
}

func (c *pipelineRPCLuaRobotClient) Threaded() luamod.BotAPI {
	clone := c.clone()
	clone.opts.Threaded = true
	return clone
}

func (c *pipelineRPCLuaRobotClient) Log(l robot.LogLevel, m string, v ...interface{}) bool {
	msg := pipelineRPCFormatMessage(m, v...)
	res, err := c.call("Log", int(l), msg)
	if err != nil {
		return false
	}
	return pipelineRPCMapBool(res, "bool")
}

func (c *pipelineRPCLuaRobotClient) SendChannelMessage(ch, msg string, v ...interface{}) robot.RetVal {
	res, err := c.call("SendChannelMessage", ch, pipelineRPCFormatMessage(msg, v...))
	if err != nil {
		return robot.Failed
	}
	return robot.RetVal(pipelineRPCMapInt(res, "ret_val"))
}

func (c *pipelineRPCLuaRobotClient) SendChannelThreadMessage(ch, thr, msg string, v ...interface{}) robot.RetVal {
	res, err := c.call("SendChannelThreadMessage", ch, thr, pipelineRPCFormatMessage(msg, v...))
	if err != nil {
		return robot.Failed
	}
	return robot.RetVal(pipelineRPCMapInt(res, "ret_val"))
}

func (c *pipelineRPCLuaRobotClient) SendUserChannelMessage(u, ch, msg string, v ...interface{}) robot.RetVal {
	res, err := c.call("SendUserChannelMessage", u, ch, pipelineRPCFormatMessage(msg, v...))
	if err != nil {
		return robot.Failed
	}
	return robot.RetVal(pipelineRPCMapInt(res, "ret_val"))
}

func (c *pipelineRPCLuaRobotClient) SendProtocolUserChannelMessage(protocol, u, ch, msg string, v ...interface{}) robot.RetVal {
	res, err := c.call("SendProtocolUserChannelMessage", protocol, u, ch, pipelineRPCFormatMessage(msg, v...))
	if err != nil {
		return robot.Failed
	}
	return robot.RetVal(pipelineRPCMapInt(res, "ret_val"))
}

func (c *pipelineRPCLuaRobotClient) SendUserChannelThreadMessage(u, ch, thr, msg string, v ...interface{}) robot.RetVal {
	res, err := c.call("SendUserChannelThreadMessage", u, ch, thr, pipelineRPCFormatMessage(msg, v...))
	if err != nil {
		return robot.Failed
	}
	return robot.RetVal(pipelineRPCMapInt(res, "ret_val"))
}

func (c *pipelineRPCLuaRobotClient) SendUserMessage(u, msg string, v ...interface{}) robot.RetVal {
	res, err := c.call("SendUserMessage", u, pipelineRPCFormatMessage(msg, v...))
	if err != nil {
		return robot.Failed
	}
	return robot.RetVal(pipelineRPCMapInt(res, "ret_val"))
}

func (c *pipelineRPCLuaRobotClient) Reply(msg string, v ...interface{}) robot.RetVal {
	res, err := c.call("Reply", pipelineRPCFormatMessage(msg, v...))
	if err != nil {
		return robot.Failed
	}
	return robot.RetVal(pipelineRPCMapInt(res, "ret_val"))
}

func (c *pipelineRPCLuaRobotClient) ReplyThread(msg string, v ...interface{}) robot.RetVal {
	res, err := c.call("ReplyThread", pipelineRPCFormatMessage(msg, v...))
	if err != nil {
		return robot.Failed
	}
	return robot.RetVal(pipelineRPCMapInt(res, "ret_val"))
}

func (c *pipelineRPCLuaRobotClient) Say(msg string, v ...interface{}) robot.RetVal {
	res, err := c.call("Say", pipelineRPCFormatMessage(msg, v...))
	if err != nil {
		return robot.Failed
	}
	return robot.RetVal(pipelineRPCMapInt(res, "ret_val"))
}

func (c *pipelineRPCLuaRobotClient) SayThread(msg string, v ...interface{}) robot.RetVal {
	res, err := c.call("SayThread", pipelineRPCFormatMessage(msg, v...))
	if err != nil {
		return robot.Failed
	}
	return robot.RetVal(pipelineRPCMapInt(res, "ret_val"))
}

func (c *pipelineRPCLuaRobotClient) RandomInt(n int) int {
	res, err := c.call("RandomInt", n)
	if err != nil {
		return 0
	}
	return pipelineRPCMapInt(res, "int")
}

func (c *pipelineRPCLuaRobotClient) RandomString(s []string) string {
	res, err := c.call("RandomString", s)
	if err != nil {
		return ""
	}
	return pipelineRPCMapString(res, "string")
}

func (c *pipelineRPCLuaRobotClient) Pause(s float64) {
	_, _ = c.call("Pause", s)
}

func (c *pipelineRPCLuaRobotClient) PromptForReply(regexID string, prompt string, v ...interface{}) (string, robot.RetVal) {
	res, err := c.call("PromptForReply", regexID, pipelineRPCFormatMessage(prompt, v...))
	if err != nil {
		return "", robot.Failed
	}
	return pipelineRPCMapString(res, "reply"), robot.RetVal(pipelineRPCMapInt(res, "ret_val"))
}

func (c *pipelineRPCLuaRobotClient) PromptThreadForReply(regexID string, prompt string, v ...interface{}) (string, robot.RetVal) {
	res, err := c.call("PromptThreadForReply", regexID, pipelineRPCFormatMessage(prompt, v...))
	if err != nil {
		return "", robot.Failed
	}
	return pipelineRPCMapString(res, "reply"), robot.RetVal(pipelineRPCMapInt(res, "ret_val"))
}

func (c *pipelineRPCLuaRobotClient) PromptUserForReply(regexID string, user string, prompt string, v ...interface{}) (string, robot.RetVal) {
	res, err := c.call("PromptUserForReply", regexID, user, pipelineRPCFormatMessage(prompt, v...))
	if err != nil {
		return "", robot.Failed
	}
	return pipelineRPCMapString(res, "reply"), robot.RetVal(pipelineRPCMapInt(res, "ret_val"))
}

func (c *pipelineRPCLuaRobotClient) PromptUserChannelForReply(regexID string, user, channel string, prompt string, v ...interface{}) (string, robot.RetVal) {
	res, err := c.call("PromptUserChannelForReply", regexID, user, channel, pipelineRPCFormatMessage(prompt, v...))
	if err != nil {
		return "", robot.Failed
	}
	return pipelineRPCMapString(res, "reply"), robot.RetVal(pipelineRPCMapInt(res, "ret_val"))
}

func (c *pipelineRPCLuaRobotClient) PromptUserChannelThreadForReply(regexID string, user, channel, thread string, prompt string, v ...interface{}) (string, robot.RetVal) {
	res, err := c.call("PromptUserChannelThreadForReply", regexID, user, channel, thread, pipelineRPCFormatMessage(prompt, v...))
	if err != nil {
		return "", robot.Failed
	}
	return pipelineRPCMapString(res, "reply"), robot.RetVal(pipelineRPCMapInt(res, "ret_val"))
}

func (c *pipelineRPCLuaRobotClient) CheckoutDatum(key string, datum interface{}, rw bool) (locktoken string, exists bool, ret robot.RetVal) {
	res, err := c.call("CheckoutDatum", key, rw)
	if err != nil {
		return "", false, robot.Failed
	}
	ret = robot.RetVal(pipelineRPCMapInt(res, "ret_val"))
	exists = pipelineRPCMapBool(res, "exists")
	locktoken = pipelineRPCMapString(res, "lock_token")
	if datum != nil {
		if v, ok := res["datum"]; ok && v != nil {
			blob, merr := json.Marshal(v)
			if merr == nil {
				_ = json.Unmarshal(blob, datum)
			}
		}
	}
	return locktoken, exists, ret
}

func (c *pipelineRPCLuaRobotClient) CheckinDatum(key, locktoken string) {
	_, _ = c.call("CheckinDatum", key, locktoken)
}

func (c *pipelineRPCLuaRobotClient) UpdateDatum(key, locktoken string, datum interface{}) (ret robot.RetVal) {
	res, err := c.call("UpdateDatum", key, locktoken, datum)
	if err != nil {
		return robot.Failed
	}
	return robot.RetVal(pipelineRPCMapInt(res, "ret_val"))
}

func (c *pipelineRPCLuaRobotClient) Remember(key, value string, shared bool) {
	_, _ = c.call("Remember", key, value, shared)
}

func (c *pipelineRPCLuaRobotClient) RememberThread(key, value string, shared bool) {
	_, _ = c.call("RememberThread", key, value, shared)
}

func (c *pipelineRPCLuaRobotClient) RememberContext(context, value string) {
	_, _ = c.call("RememberContext", context, value)
}

func (c *pipelineRPCLuaRobotClient) RememberContextThread(context, value string) {
	_, _ = c.call("RememberContextThread", context, value)
}

func (c *pipelineRPCLuaRobotClient) Recall(key string, shared bool) string {
	res, err := c.call("Recall", key, shared)
	if err != nil {
		return ""
	}
	return pipelineRPCMapString(res, "string")
}

func (c *pipelineRPCLuaRobotClient) SpawnJob(name string, args ...string) robot.RetVal {
	payload := []interface{}{name}
	for _, s := range args {
		payload = append(payload, s)
	}
	res, err := c.call("SpawnJob", payload...)
	if err != nil {
		return robot.Failed
	}
	return robot.RetVal(pipelineRPCMapInt(res, "ret_val"))
}

func (c *pipelineRPCLuaRobotClient) AddTask(name string, args ...string) robot.RetVal {
	payload := []interface{}{name}
	for _, s := range args {
		payload = append(payload, s)
	}
	res, err := c.call("AddTask", payload...)
	if err != nil {
		return robot.Failed
	}
	return robot.RetVal(pipelineRPCMapInt(res, "ret_val"))
}

func (c *pipelineRPCLuaRobotClient) FinalTask(name string, args ...string) robot.RetVal {
	payload := []interface{}{name}
	for _, s := range args {
		payload = append(payload, s)
	}
	res, err := c.call("FinalTask", payload...)
	if err != nil {
		return robot.Failed
	}
	return robot.RetVal(pipelineRPCMapInt(res, "ret_val"))
}

func (c *pipelineRPCLuaRobotClient) FailTask(name string, args ...string) robot.RetVal {
	payload := []interface{}{name}
	for _, s := range args {
		payload = append(payload, s)
	}
	res, err := c.call("FailTask", payload...)
	if err != nil {
		return robot.Failed
	}
	return robot.RetVal(pipelineRPCMapInt(res, "ret_val"))
}

func (c *pipelineRPCLuaRobotClient) AddJob(name string, args ...string) robot.RetVal {
	payload := []interface{}{name}
	for _, s := range args {
		payload = append(payload, s)
	}
	res, err := c.call("AddJob", payload...)
	if err != nil {
		return robot.Failed
	}
	return robot.RetVal(pipelineRPCMapInt(res, "ret_val"))
}

func (c *pipelineRPCLuaRobotClient) AddCommand(plugin, command string) robot.RetVal {
	res, err := c.call("AddCommand", plugin, command)
	if err != nil {
		return robot.Failed
	}
	return robot.RetVal(pipelineRPCMapInt(res, "ret_val"))
}

func (c *pipelineRPCLuaRobotClient) FinalCommand(plugin, command string) robot.RetVal {
	res, err := c.call("FinalCommand", plugin, command)
	if err != nil {
		return robot.Failed
	}
	return robot.RetVal(pipelineRPCMapInt(res, "ret_val"))
}

func (c *pipelineRPCLuaRobotClient) FailCommand(plugin, command string) robot.RetVal {
	res, err := c.call("FailCommand", plugin, command)
	if err != nil {
		return robot.Failed
	}
	return robot.RetVal(pipelineRPCMapInt(res, "ret_val"))
}

func (c *pipelineRPCLuaRobotClient) SetParameter(name, value string) bool {
	res, err := c.call("SetParameter", name, value)
	if err != nil {
		return false
	}
	return pipelineRPCMapBool(res, "bool")
}

func (c *pipelineRPCLuaRobotClient) Subscribe() bool {
	res, err := c.call("Subscribe")
	if err != nil {
		return false
	}
	return pipelineRPCMapBool(res, "bool")
}

func (c *pipelineRPCLuaRobotClient) Unsubscribe() bool {
	res, err := c.call("Unsubscribe")
	if err != nil {
		return false
	}
	return pipelineRPCMapBool(res, "bool")
}

func pipelineRPCArgAny(args []interface{}, idx int) (interface{}, error) {
	if idx < 0 || idx >= len(args) {
		return nil, fmt.Errorf("missing argument at index %d", idx)
	}
	return args[idx], nil
}

func pipelineRPCArgString(args []interface{}, idx int) (string, error) {
	v, err := pipelineRPCArgAny(args, idx)
	if err != nil {
		return "", err
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("argument %d is not a string", idx)
	}
	return s, nil
}

func pipelineRPCArgBool(args []interface{}, idx int) (bool, error) {
	v, err := pipelineRPCArgAny(args, idx)
	if err != nil {
		return false, err
	}
	b, ok := v.(bool)
	if !ok {
		return false, fmt.Errorf("argument %d is not a bool", idx)
	}
	return b, nil
}

func pipelineRPCArgInt(args []interface{}, idx int) (int, error) {
	v, err := pipelineRPCArgAny(args, idx)
	if err != nil {
		return 0, err
	}
	switch n := v.(type) {
	case float64:
		return int(n), nil
	case int:
		return n, nil
	case int64:
		return int(n), nil
	case string:
		i, parseErr := strconv.Atoi(n)
		if parseErr != nil {
			return 0, fmt.Errorf("argument %d is not an int", idx)
		}
		return i, nil
	default:
		return 0, fmt.Errorf("argument %d is not an int", idx)
	}
}

func pipelineRPCArgFloat(args []interface{}, idx int) (float64, error) {
	v, err := pipelineRPCArgAny(args, idx)
	if err != nil {
		return 0, err
	}
	switch n := v.(type) {
	case float64:
		return n, nil
	case int:
		return float64(n), nil
	case int64:
		return float64(n), nil
	default:
		return 0, fmt.Errorf("argument %d is not a float", idx)
	}
}

func pipelineRPCArgStringSlice(args []interface{}, idx int) ([]string, error) {
	v, err := pipelineRPCArgAny(args, idx)
	if err != nil {
		return nil, err
	}
	rawSlice, ok := v.([]interface{})
	if !ok {
		return nil, fmt.Errorf("argument %d is not a string slice", idx)
	}
	out := make([]string, 0, len(rawSlice))
	for _, e := range rawSlice {
		s, ok := e.(string)
		if !ok {
			return nil, fmt.Errorf("argument %d has non-string element", idx)
		}
		out = append(out, s)
	}
	return out, nil
}

func pipelineRPCArgTailStrings(args []interface{}, start int) ([]string, error) {
	if start >= len(args) {
		return []string{}, nil
	}
	out := make([]string, 0, len(args)-start)
	for i := start; i < len(args); i++ {
		s, err := pipelineRPCArgString(args, i)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

func pipelineRPCMapInt(m map[string]interface{}, key string) int {
	v, ok := m[key]
	if !ok || v == nil {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	case string:
		i, _ := strconv.Atoi(n)
		return i
	default:
		return 0
	}
}

func pipelineRPCMapString(m map[string]interface{}, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

func pipelineRPCMapBool(m map[string]interface{}, key string) bool {
	v, ok := m[key]
	if !ok || v == nil {
		return false
	}
	b, _ := v.(bool)
	return b
}

func pipelineRPCFormatMessage(msg string, v ...interface{}) string {
	if len(v) == 0 {
		return msg
	}
	return fmt.Sprintf(msg, v...)
}

var _ luamod.BotAPI = (*pipelineRPCLuaRobotClient)(nil)
var _ robot.Logger = (*pipelineRPCLuaRobotClient)(nil)
