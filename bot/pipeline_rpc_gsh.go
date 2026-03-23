package bot

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/lnxjedi/gopherbot/robot"
	gshmod "github.com/lnxjedi/gopherbot/v2/modules/gsh"
)

type pipelineRPCGSHRunRequest struct {
	TaskPath string   `json:"task_path"`
	TaskName string   `json:"task_name"`
	Env      []string `json:"env"`
	Args     []string `json:"args"`
}

type pipelineRPCGSHRunResponse struct {
	RetVal int    `json:"ret_val"`
	Error  string `json:"error,omitempty"`
}

type pipelineRPCGSHGetConfigRequest struct {
	TaskPath string   `json:"task_path"`
	TaskName string   `json:"task_name"`
	Env      []string `json:"env"`
}

type pipelineRPCGSHGetConfigResponse struct {
	Config string `json:"config,omitempty"`
	Error  string `json:"error,omitempty"`
}

func runGSHExtensionViaRPC(taskPath, taskName string, env []string, w *worker, r robot.Robot, args []string) (robot.TaskRetVal, error) {
	params := pipelineRPCGSHRunRequest{
		TaskPath: taskPath,
		TaskName: taskName,
		Env:      env,
		Args:     args,
	}
	resRaw, err := runPipelineRPCRequest("gsh_run", params, w, r)
	if err != nil {
		return robot.MechanismFail, err
	}
	var res pipelineRPCGSHRunResponse
	if err := json.Unmarshal(resRaw, &res); err != nil {
		return robot.MechanismFail, fmt.Errorf("decoding gsh_run response: %v", err)
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

func runGSHGetConfigViaRPC(taskPath, taskName string, env []string) (*[]byte, error) {
	params := pipelineRPCGSHGetConfigRequest{
		TaskPath: taskPath,
		TaskName: taskName,
		Env:      env,
	}
	resRaw, err := runPipelineRPCRequest("gsh_get_config", params, nil, nil)
	if err != nil {
		return nil, err
	}
	var res pipelineRPCGSHGetConfigResponse
	if err := json.Unmarshal(resRaw, &res); err != nil {
		return nil, fmt.Errorf("decoding gsh_get_config response: %v", err)
	}
	if res.Error != "" {
		return nil, errors.New(res.Error)
	}
	cfg := []byte(res.Config)
	return &cfg, nil
}

func handlePipelineRPCGSHRun(dec *json.Decoder, enc *json.Encoder, msg pipelineRPCMessage) error {
	var req pipelineRPCGSHRunRequest
	if err := json.Unmarshal(msg.Params, &req); err != nil {
		return writePipelineRPCError(enc, msg.ID, "invalid_params", fmt.Sprintf("invalid gsh_run params: %v", err))
	}
	client := newPipelineRPCInterpreterRobotClient(dec, enc, map[string]string{})
	ret, err := gshmod.CallExtension(req.TaskPath, req.TaskName, req.Env, client, client, req.Args)
	res := pipelineRPCGSHRunResponse{RetVal: int(ret)}
	if err != nil {
		res.Error = err.Error()
	}
	return writePipelineRPCResponse(enc, msg.ID, res)
}

func handlePipelineRPCGSHGetConfig(enc *json.Encoder, msg pipelineRPCMessage) error {
	var req pipelineRPCGSHGetConfigRequest
	if err := json.Unmarshal(msg.Params, &req); err != nil {
		return writePipelineRPCError(enc, msg.ID, "invalid_params", fmt.Sprintf("invalid gsh_get_config params: %v", err))
	}
	cfg, err := gshmod.GetPluginConfig(req.TaskPath, req.TaskName, req.Env, nil)
	res := pipelineRPCGSHGetConfigResponse{}
	if err != nil {
		res.Error = err.Error()
	} else if cfg != nil {
		res.Config = string(*cfg)
	}
	return writePipelineRPCResponse(enc, msg.ID, res)
}
