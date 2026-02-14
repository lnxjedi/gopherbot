package bot

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/lnxjedi/gopherbot/robot"
	yaegi "github.com/lnxjedi/gopherbot/v2/modules/yaegi-dynamic-go"
)

type pipelineRPCGoRunRequest struct {
	TaskPath   string   `json:"task_path"`
	TaskName   string   `json:"task_name"`
	Env        []string `json:"env"`
	Privileged bool     `json:"privileged"`
	Args       []string `json:"args"`
}

type pipelineRPCGoRunResponse struct {
	RetVal int    `json:"ret_val"`
	Error  string `json:"error,omitempty"`
}

type pipelineRPCGoGetConfigRequest struct {
	TaskPath string `json:"task_path"`
	TaskName string `json:"task_name"`
}

type pipelineRPCGoGetConfigResponse struct {
	Config string `json:"config,omitempty"`
	Error  string `json:"error,omitempty"`
}

func runGoPluginViaRPC(taskPath, taskName string, env []string, privileged bool, r robot.Robot, args []string) (robot.TaskRetVal, error) {
	params := pipelineRPCGoRunRequest{
		TaskPath:   taskPath,
		TaskName:   taskName,
		Env:        env,
		Privileged: privileged,
		Args:       args,
	}
	return runGoViaRPCMethod("go_plugin_run", params, r)
}

func runGoJobViaRPC(taskPath, taskName string, env []string, privileged bool, r robot.Robot, args []string) (robot.TaskRetVal, error) {
	params := pipelineRPCGoRunRequest{
		TaskPath:   taskPath,
		TaskName:   taskName,
		Env:        env,
		Privileged: privileged,
		Args:       args,
	}
	return runGoViaRPCMethod("go_job_run", params, r)
}

func runGoTaskViaRPC(taskPath, taskName string, env []string, privileged bool, r robot.Robot, args []string) (robot.TaskRetVal, error) {
	params := pipelineRPCGoRunRequest{
		TaskPath:   taskPath,
		TaskName:   taskName,
		Env:        env,
		Privileged: privileged,
		Args:       args,
	}
	return runGoViaRPCMethod("go_task_run", params, r)
}

func runGoViaRPCMethod(method string, params pipelineRPCGoRunRequest, r robot.Robot) (robot.TaskRetVal, error) {
	resRaw, err := runPipelineRPCRequest(method, params, r)
	if err != nil {
		return robot.MechanismFail, err
	}
	var res pipelineRPCGoRunResponse
	if err := json.Unmarshal(resRaw, &res); err != nil {
		return robot.MechanismFail, fmt.Errorf("decoding %s response: %v", method, err)
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

func runGoGetConfigViaRPC(taskPath, taskName string) (*[]byte, error) {
	params := pipelineRPCGoGetConfigRequest{
		TaskPath: taskPath,
		TaskName: taskName,
	}
	resRaw, err := runPipelineRPCRequest("go_get_config", params, nil)
	if err != nil {
		return nil, err
	}
	var res pipelineRPCGoGetConfigResponse
	if err := json.Unmarshal(resRaw, &res); err != nil {
		return nil, fmt.Errorf("decoding go_get_config response: %v", err)
	}
	if res.Error != "" {
		return nil, errors.New(res.Error)
	}
	cfg := []byte(res.Config)
	return &cfg, nil
}

func handlePipelineRPCGoPluginRun(dec *json.Decoder, enc *json.Encoder, msg pipelineRPCMessage) error {
	var req pipelineRPCGoRunRequest
	if err := json.Unmarshal(msg.Params, &req); err != nil {
		return writePipelineRPCError(enc, msg.ID, "invalid_params", fmt.Sprintf("invalid go_plugin_run params: %v", err))
	}
	client := newPipelineRPCGoRobotClient(dec, enc)
	res := pipelineRPCGoRunResponse{}
	if len(req.Args) == 0 {
		res.RetVal = int(robot.MechanismFail)
		res.Error = "go_plugin_run requires command argument"
		return writePipelineRPCResponse(enc, msg.ID, res)
	}
	ret, err := yaegi.RunPluginHandler(req.TaskPath, req.TaskName, req.Env, client, client, req.Privileged, req.Args[0], req.Args[1:]...)
	res.RetVal = int(ret)
	if err != nil {
		res.Error = err.Error()
	}
	return writePipelineRPCResponse(enc, msg.ID, res)
}

func handlePipelineRPCGoJobRun(dec *json.Decoder, enc *json.Encoder, msg pipelineRPCMessage) error {
	var req pipelineRPCGoRunRequest
	if err := json.Unmarshal(msg.Params, &req); err != nil {
		return writePipelineRPCError(enc, msg.ID, "invalid_params", fmt.Sprintf("invalid go_job_run params: %v", err))
	}
	client := newPipelineRPCGoRobotClient(dec, enc)
	ret, err := yaegi.RunJobHandler(req.TaskPath, req.TaskName, req.Env, client, client, req.Privileged, req.Args...)
	res := pipelineRPCGoRunResponse{RetVal: int(ret)}
	if err != nil {
		res.Error = err.Error()
	}
	return writePipelineRPCResponse(enc, msg.ID, res)
}

func handlePipelineRPCGoTaskRun(dec *json.Decoder, enc *json.Encoder, msg pipelineRPCMessage) error {
	var req pipelineRPCGoRunRequest
	if err := json.Unmarshal(msg.Params, &req); err != nil {
		return writePipelineRPCError(enc, msg.ID, "invalid_params", fmt.Sprintf("invalid go_task_run params: %v", err))
	}
	client := newPipelineRPCGoRobotClient(dec, enc)
	ret, err := yaegi.RunTaskHandler(req.TaskPath, req.TaskName, req.Env, client, client, req.Privileged, req.Args...)
	res := pipelineRPCGoRunResponse{RetVal: int(ret)}
	if err != nil {
		res.Error = err.Error()
	}
	return writePipelineRPCResponse(enc, msg.ID, res)
}

func handlePipelineRPCGoGetConfig(enc *json.Encoder, msg pipelineRPCMessage) error {
	var req pipelineRPCGoGetConfigRequest
	if err := json.Unmarshal(msg.Params, &req); err != nil {
		return writePipelineRPCError(enc, msg.ID, "invalid_params", fmt.Sprintf("invalid go_get_config params: %v", err))
	}
	cfg, err := yaegi.GetPluginConfig(req.TaskPath, req.TaskName)
	res := pipelineRPCGoGetConfigResponse{}
	if err != nil {
		res.Error = err.Error()
	} else if cfg != nil {
		res.Config = string(*cfg)
	}
	return writePipelineRPCResponse(enc, msg.ID, res)
}

type pipelineRPCGoRobotClient struct {
	*pipelineRPCLuaRobotClient
}

func newPipelineRPCGoRobotClient(dec *json.Decoder, enc *json.Encoder) *pipelineRPCGoRobotClient {
	return &pipelineRPCGoRobotClient{
		pipelineRPCLuaRobotClient: newPipelineRPCLuaRobotClient(dec, enc, map[string]string{}),
	}
}

func (c *pipelineRPCGoRobotClient) cloneGo() *pipelineRPCGoRobotClient {
	return &pipelineRPCGoRobotClient{
		pipelineRPCLuaRobotClient: c.pipelineRPCLuaRobotClient.clone(),
	}
}

func (c *pipelineRPCGoRobotClient) Fixed() robot.Robot {
	clone := c.cloneGo()
	f := int(robot.Fixed)
	clone.opts.Format = &f
	return clone
}

func (c *pipelineRPCGoRobotClient) MessageFormat(f robot.MessageFormat) robot.Robot {
	clone := c.cloneGo()
	fi := int(f)
	clone.opts.Format = &fi
	return clone
}

func (c *pipelineRPCGoRobotClient) Direct() robot.Robot {
	clone := c.cloneGo()
	clone.opts.Direct = true
	return clone
}

func (c *pipelineRPCGoRobotClient) Threaded() robot.Robot {
	clone := c.cloneGo()
	clone.opts.Threaded = true
	return clone
}

func (c *pipelineRPCGoRobotClient) GetMessage() *robot.Message {
	res, err := c.call("GetMessage")
	if err != nil {
		return nil
	}
	raw, ok := res["message"]
	if !ok || raw == nil {
		return nil
	}
	msgBlob, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var payload struct {
		User            string `json:"user"`
		ProtocolUser    string `json:"protocol_user"`
		Channel         string `json:"channel"`
		ProtocolChannel string `json:"protocol_channel"`
		Protocol        int    `json:"protocol"`
		Format          int    `json:"format"`
	}
	if err := json.Unmarshal(msgBlob, &payload); err != nil {
		return nil
	}
	return &robot.Message{
		User:            payload.User,
		ProtocolUser:    payload.ProtocolUser,
		Channel:         payload.Channel,
		ProtocolChannel: payload.ProtocolChannel,
		Protocol:        robot.Protocol(payload.Protocol),
		Format:          robot.MessageFormat(payload.Format),
	}
}

func (c *pipelineRPCGoRobotClient) Email(subject string, messageBody *bytes.Buffer, html ...bool) (ret robot.RetVal) {
	body := ""
	if messageBody != nil {
		body = messageBody.String()
	}
	htmlFlag := false
	if len(html) > 0 {
		htmlFlag = html[0]
	}
	res, err := c.call("Email", subject, body, htmlFlag)
	if err != nil {
		return robot.Failed
	}
	return robot.RetVal(pipelineRPCMapInt(res, "ret_val"))
}

func (c *pipelineRPCGoRobotClient) EmailUser(user, subject string, messageBody *bytes.Buffer, html ...bool) (ret robot.RetVal) {
	body := ""
	if messageBody != nil {
		body = messageBody.String()
	}
	htmlFlag := false
	if len(html) > 0 {
		htmlFlag = html[0]
	}
	res, err := c.call("EmailUser", user, subject, body, htmlFlag)
	if err != nil {
		return robot.Failed
	}
	return robot.RetVal(pipelineRPCMapInt(res, "ret_val"))
}

func (c *pipelineRPCGoRobotClient) EmailAddress(address, subject string, messageBody *bytes.Buffer, html ...bool) (ret robot.RetVal) {
	body := ""
	if messageBody != nil {
		body = messageBody.String()
	}
	htmlFlag := false
	if len(html) > 0 {
		htmlFlag = html[0]
	}
	res, err := c.call("EmailAddress", address, subject, body, htmlFlag)
	if err != nil {
		return robot.Failed
	}
	return robot.RetVal(pipelineRPCMapInt(res, "ret_val"))
}

func (c *pipelineRPCGoRobotClient) RaisePriv(reason string) {
	_, _ = c.call("RaisePriv", reason)
}

func (c *pipelineRPCGoRobotClient) SetWorkingDirectory(path string) bool {
	res, err := c.call("SetWorkingDirectory", path)
	if err != nil {
		return false
	}
	return pipelineRPCMapBool(res, "bool")
}

var _ robot.Robot = (*pipelineRPCGoRobotClient)(nil)
var _ robot.Logger = (*pipelineRPCGoRobotClient)(nil)
