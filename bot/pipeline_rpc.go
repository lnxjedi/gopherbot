package bot

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
)

const pipelineRPCProtocolVersion = 1

type pipelineRPCMessage struct {
	Version int             `json:"version"`
	ID      string          `json:"id,omitempty"`
	Type    string          `json:"type"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *pipelineRPCErr `json:"error,omitempty"`
}

type pipelineRPCErr struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func newPipelineChildRPCCommand() *exec.Cmd {
	cmd := exec.Command(execPath(), pipelineChildRPCCommand)
	cmd.Env = sanitizedChildEnvironment()
	return cmd
}

func runPipelineChildRPC() int {
	return runPipelineChildRPCWithIO(os.Stdin, os.Stdout)
}

func runPipelineChildRPCWithIO(r io.Reader, w io.Writer) int {
	dec := json.NewDecoder(r)
	enc := json.NewEncoder(w)

	msg, err := readPipelineRPCMessage(dec)
	if err != nil {
		return 2
	}
	if msg.Type != "hello" || msg.Version != pipelineRPCProtocolVersion {
		_ = writePipelineRPCError(enc, msg.ID, "protocol_error", fmt.Sprintf("expected hello v%d", pipelineRPCProtocolVersion))
		return 2
	}
	if err := enc.Encode(pipelineRPCMessage{
		Version: pipelineRPCProtocolVersion,
		ID:      msg.ID,
		Type:    "hello_ack",
	}); err != nil {
		return 2
	}

	for {
		msg, err := readPipelineRPCMessage(dec)
		if err == io.EOF {
			return 0
		}
		if err != nil {
			return 2
		}
		if msg.Version != pipelineRPCProtocolVersion {
			_ = writePipelineRPCError(enc, msg.ID, "protocol_error", fmt.Sprintf("unsupported version %d", msg.Version))
			continue
		}
		if msg.Type != "request" {
			_ = writePipelineRPCError(enc, msg.ID, "protocol_error", "expected request message")
			continue
		}
		switch msg.Method {
		case "shutdown":
			result, _ := json.Marshal(map[string]bool{"ok": true})
			_ = enc.Encode(pipelineRPCMessage{
				Version: pipelineRPCProtocolVersion,
				ID:      msg.ID,
				Type:    "response",
				Result:  result,
			})
			return 0
		case "lua_run":
			if err := handlePipelineRPCLuaRun(dec, enc, msg); err != nil {
				return 2
			}
		case "lua_get_config":
			if err := handlePipelineRPCLuaGetConfig(enc, msg); err != nil {
				return 2
			}
		case "js_run":
			if err := handlePipelineRPCJSRun(dec, enc, msg); err != nil {
				return 2
			}
		case "js_get_config":
			if err := handlePipelineRPCJSGetConfig(enc, msg); err != nil {
				return 2
			}
		case "go_plugin_run":
			if err := handlePipelineRPCGoPluginRun(dec, enc, msg); err != nil {
				return 2
			}
		case "go_job_run":
			if err := handlePipelineRPCGoJobRun(dec, enc, msg); err != nil {
				return 2
			}
		case "go_task_run":
			if err := handlePipelineRPCGoTaskRun(dec, enc, msg); err != nil {
				return 2
			}
		case "go_get_config":
			if err := handlePipelineRPCGoGetConfig(enc, msg); err != nil {
				return 2
			}
		default:
			_ = writePipelineRPCError(enc, msg.ID, "method_not_found", fmt.Sprintf("unsupported method '%s'", msg.Method))
		}
	}
}

func readPipelineRPCMessage(dec *json.Decoder) (pipelineRPCMessage, error) {
	var msg pipelineRPCMessage
	if err := dec.Decode(&msg); err != nil {
		return msg, err
	}
	return msg, nil
}

func writePipelineRPCError(enc *json.Encoder, id, code, message string) error {
	return enc.Encode(pipelineRPCMessage{
		Version: pipelineRPCProtocolVersion,
		ID:      id,
		Type:    "error",
		Error: &pipelineRPCErr{
			Code:    code,
			Message: message,
		},
	})
}
