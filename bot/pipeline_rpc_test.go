package bot

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestNewPipelineChildRPCCommand(t *testing.T) {
	cmd := newPipelineChildRPCCommand()
	if len(cmd.Args) < 2 || cmd.Args[1] != pipelineChildRPCCommand {
		t.Fatalf("child rpc command args = %#v, want second arg %q", cmd.Args, pipelineChildRPCCommand)
	}
}

func TestRunPipelineChildRPCWithIOHandshakeAndShutdown(t *testing.T) {
	input := strings.Join([]string{
		`{"version":1,"id":"hello-1","type":"hello"}`,
		`{"version":1,"id":"req-1","type":"request","method":"shutdown"}`,
	}, "\n") + "\n"
	in := bytes.NewBufferString(input)
	var out bytes.Buffer

	code := runPipelineChildRPCWithIO(in, &out)
	if code != 0 {
		t.Fatalf("runPipelineChildRPCWithIO() code = %d, want 0", code)
	}

	dec := json.NewDecoder(&out)
	var msg1 pipelineRPCMessage
	if err := dec.Decode(&msg1); err != nil {
		t.Fatalf("decode msg1: %v", err)
	}
	if msg1.Type != "hello_ack" || msg1.ID != "hello-1" || msg1.Version != pipelineRPCProtocolVersion {
		t.Fatalf("msg1 = %#v, want hello_ack for hello-1", msg1)
	}
	var msg2 pipelineRPCMessage
	if err := dec.Decode(&msg2); err != nil {
		t.Fatalf("decode msg2: %v", err)
	}
	if msg2.Type != "response" || msg2.ID != "req-1" || msg2.Version != pipelineRPCProtocolVersion {
		t.Fatalf("msg2 = %#v, want shutdown response for req-1", msg2)
	}
	if len(msg2.Result) == 0 {
		t.Fatalf("msg2 result missing: %#v", msg2)
	}
}

func TestRunPipelineChildRPCWithIORequiresHello(t *testing.T) {
	in := bytes.NewBufferString(`{"version":1,"id":"bad-1","type":"request","method":"shutdown"}` + "\n")
	var out bytes.Buffer

	code := runPipelineChildRPCWithIO(in, &out)
	if code != 2 {
		t.Fatalf("runPipelineChildRPCWithIO() code = %d, want 2", code)
	}

	var msg pipelineRPCMessage
	if err := json.NewDecoder(&out).Decode(&msg); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if msg.Type != "error" || msg.Error == nil || msg.Error.Code != "protocol_error" {
		t.Fatalf("msg = %#v, want protocol_error response", msg)
	}
}
