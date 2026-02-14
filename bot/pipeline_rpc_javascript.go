package bot

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/lnxjedi/gopherbot/robot"
	jsmod "github.com/lnxjedi/gopherbot/v2/modules/javascript"
)

type pipelineRPCJSRunRequest struct {
	ExecPath     string            `json:"exec_path"`
	TaskPath     string            `json:"task_path"`
	TaskName     string            `json:"task_name"`
	RequirePaths []string          `json:"require_paths"`
	Bot          map[string]string `json:"bot"`
	Args         []string          `json:"args"`
}

type pipelineRPCJSRunResponse struct {
	RetVal int    `json:"ret_val"`
	Error  string `json:"error,omitempty"`
}

type pipelineRPCJSGetConfigRequest struct {
	ExecPath     string            `json:"exec_path"`
	TaskPath     string            `json:"task_path"`
	TaskName     string            `json:"task_name"`
	RequirePaths []string          `json:"require_paths"`
	Bot          map[string]string `json:"bot"`
}

type pipelineRPCJSGetConfigResponse struct {
	Config string `json:"config,omitempty"`
	Error  string `json:"error,omitempty"`
}

func runJSExtensionViaRPC(taskPath, taskName string, requirePaths []string, bot map[string]string, w *worker, r robot.Robot, args []string) (robot.TaskRetVal, error) {
	params := pipelineRPCJSRunRequest{
		ExecPath:     execPath(),
		TaskPath:     taskPath,
		TaskName:     taskName,
		RequirePaths: requirePaths,
		Bot:          bot,
		Args:         args,
	}
	resRaw, err := runPipelineRPCRequest("js_run", params, w, r)
	if err != nil {
		return robot.MechanismFail, err
	}
	var res pipelineRPCJSRunResponse
	if err := json.Unmarshal(resRaw, &res); err != nil {
		return robot.MechanismFail, fmt.Errorf("decoding js_run response: %v", err)
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

func runJSGetConfigViaRPC(taskPath, taskName string, requirePaths []string, bot map[string]string) (*[]byte, error) {
	params := pipelineRPCJSGetConfigRequest{
		ExecPath:     execPath(),
		TaskPath:     taskPath,
		TaskName:     taskName,
		RequirePaths: requirePaths,
		Bot:          bot,
	}
	resRaw, err := runPipelineRPCRequest("js_get_config", params, nil, nil)
	if err != nil {
		return nil, err
	}
	var res pipelineRPCJSGetConfigResponse
	if err := json.Unmarshal(resRaw, &res); err != nil {
		return nil, fmt.Errorf("decoding js_get_config response: %v", err)
	}
	if res.Error != "" {
		return nil, errors.New(res.Error)
	}
	cfg := []byte(res.Config)
	return &cfg, nil
}

func handlePipelineRPCJSRun(dec *json.Decoder, enc *json.Encoder, msg pipelineRPCMessage) error {
	var req pipelineRPCJSRunRequest
	if err := json.Unmarshal(msg.Params, &req); err != nil {
		return writePipelineRPCError(enc, msg.ID, "invalid_params", fmt.Sprintf("invalid js_run params: %v", err))
	}
	client := newPipelineRPCJSRobotClient(dec, enc, req.Bot)
	ret, err := jsmod.CallExtension(req.ExecPath, req.TaskPath, req.TaskName, req.RequirePaths, client, req.Bot, client, req.Args)
	res := pipelineRPCJSRunResponse{RetVal: int(ret)}
	if err != nil {
		res.Error = err.Error()
	}
	return writePipelineRPCResponse(enc, msg.ID, res)
}

func handlePipelineRPCJSGetConfig(enc *json.Encoder, msg pipelineRPCMessage) error {
	var req pipelineRPCJSGetConfigRequest
	if err := json.Unmarshal(msg.Params, &req); err != nil {
		return writePipelineRPCError(enc, msg.ID, "invalid_params", fmt.Sprintf("invalid js_get_config params: %v", err))
	}
	cfg, err := jsmod.GetPluginConfig(req.ExecPath, req.TaskPath, req.TaskName, req.Bot, req.RequirePaths)
	res := pipelineRPCJSGetConfigResponse{}
	if err != nil {
		res.Error = err.Error()
	} else if cfg != nil {
		res.Config = string(*cfg)
	}
	return writePipelineRPCResponse(enc, msg.ID, res)
}

type pipelineRPCJSRobotClient struct {
	*pipelineRPCLuaRobotClient
}

func newPipelineRPCJSRobotClient(dec *json.Decoder, enc *json.Encoder, bot map[string]string) *pipelineRPCJSRobotClient {
	return &pipelineRPCJSRobotClient{pipelineRPCLuaRobotClient: newPipelineRPCLuaRobotClient(dec, enc, bot)}
}

func (c *pipelineRPCJSRobotClient) cloneJS() *pipelineRPCJSRobotClient {
	return &pipelineRPCJSRobotClient{pipelineRPCLuaRobotClient: c.pipelineRPCLuaRobotClient.clone()}
}

func (c *pipelineRPCJSRobotClient) Fixed() jsmod.BotAPI {
	clone := c.cloneJS()
	f := int(robot.Fixed)
	clone.opts.Format = &f
	return clone
}

func (c *pipelineRPCJSRobotClient) MessageFormat(f robot.MessageFormat) jsmod.BotAPI {
	clone := c.cloneJS()
	fi := int(f)
	clone.opts.Format = &fi
	return clone
}

func (c *pipelineRPCJSRobotClient) Direct() jsmod.BotAPI {
	clone := c.cloneJS()
	clone.opts.Direct = true
	return clone
}

func (c *pipelineRPCJSRobotClient) Threaded() jsmod.BotAPI {
	clone := c.cloneJS()
	clone.opts.Threaded = true
	return clone
}

var _ jsmod.BotAPI = (*pipelineRPCJSRobotClient)(nil)
var _ robot.Logger = (*pipelineRPCJSRobotClient)(nil)
