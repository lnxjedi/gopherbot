package bot

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

const (
	pipelineChildExecCommand    = "pipeline-child-exec"
	pipelineChildRPCCommand     = "pipeline-child-rpc"
	pipelineChildExecRequestEnv = "GOPHER_PIPE_CHILD_EXEC_REQUEST"
)

type pipelineChildExecRequest struct {
	TaskPath string   `json:"task_path"`
	Dir      string   `json:"dir"`
	Args     []string `json:"args"`
	Env      []string `json:"env"`
	EID      string   `json:"eid"`
	NullConn bool     `json:"null_conn"`
}

func encodePipelineChildExecRequest(req pipelineChildExecRequest) (string, error) {
	raw, err := json.Marshal(req)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(raw), nil
}

func decodePipelineChildExecRequest(encoded string) (pipelineChildExecRequest, error) {
	var req pipelineChildExecRequest
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return req, err
	}
	if err := json.Unmarshal(raw, &req); err != nil {
		return req, err
	}
	return req, nil
}

func validatePipelineChildExecRequest(req pipelineChildExecRequest) error {
	if strings.TrimSpace(req.TaskPath) == "" {
		return fmt.Errorf("task_path is required")
	}
	if strings.TrimSpace(req.Dir) == "" {
		return fmt.Errorf("dir is required")
	}
	if len(req.Env) == 0 {
		return fmt.Errorf("env is required")
	}
	if !req.NullConn && strings.TrimSpace(req.EID) == "" {
		return fmt.Errorf("eid is required when null_conn is false")
	}
	return nil
}

func newPipelineChildExecCommand(req pipelineChildExecRequest) (*exec.Cmd, error) {
	if err := validatePipelineChildExecRequest(req); err != nil {
		return nil, err
	}
	reqEncoded, err := encodePipelineChildExecRequest(req)
	if err != nil {
		return nil, err
	}
	childEnv := append(os.Environ(), pipelineChildExecRequestEnv+"="+reqEncoded)
	cmd := exec.Command(execPath(), pipelineChildExecCommand)
	cmd.Env = childEnv
	return cmd, nil
}

func runPipelineChildExec() int {
	encoded, ok := os.LookupEnv(pipelineChildExecRequestEnv)
	if !ok || strings.TrimSpace(encoded) == "" {
		fmt.Fprintf(os.Stderr, "Missing %s\n", pipelineChildExecRequestEnv)
		return 2
	}
	req, err := decodePipelineChildExecRequest(encoded)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Decoding %s: %v\n", pipelineChildExecRequestEnv, err)
		return 2
	}
	if err := validatePipelineChildExecRequest(req); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid %s: %v\n", pipelineChildExecRequestEnv, err)
		return 2
	}

	cmd := exec.Command(req.TaskPath, req.Args...)
	cmd.Dir = req.Dir
	cmd.Env = req.Env

	if req.NullConn {
		cmd.Stdin = os.Stdin
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Creating stdout pipe: %v\n", err)
		return 1
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Creating stderr pipe: %v\n", err)
		return 1
	}

	var stdinPipe io.WriteCloser
	if !req.NullConn {
		stdinPipe, err = cmd.StdinPipe()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Creating stdin pipe: %v\n", err)
			return 1
		}
	}

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Starting command: %v\n", err)
		return 1
	}

	if !req.NullConn {
		go func() {
			defer stdinPipe.Close()
			io.WriteString(stdinPipe, req.EID+"\n")
		}()
	}

	closed := make(chan struct{}, 2)
	go func() {
		io.Copy(os.Stdout, stdout)
		closed <- struct{}{}
	}()
	go func() {
		io.Copy(os.Stderr, stderr)
		closed <- struct{}{}
	}()
	<-closed
	<-closed

	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				return status.ExitStatus()
			}
		}
		fmt.Fprintf(os.Stderr, "Waiting for command: %v\n", err)
		return 1
	}
	return 0
}
