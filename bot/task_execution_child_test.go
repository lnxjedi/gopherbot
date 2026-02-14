package bot

import "testing"

func TestPipelineChildExecRequestRoundTrip(t *testing.T) {
	original := pipelineChildExecRequest{
		TaskPath: "/tmp/task.sh",
		Dir:      "/tmp",
		Args:     []string{"run", "x"},
		Env:      []string{"A=B", "C=D"},
		EID:      "abc123",
		NullConn: false,
	}
	encoded, err := encodePipelineChildExecRequest(original)
	if err != nil {
		t.Fatalf("encodePipelineChildExecRequest() error = %v", err)
	}
	decoded, err := decodePipelineChildExecRequest(encoded)
	if err != nil {
		t.Fatalf("decodePipelineChildExecRequest() error = %v", err)
	}
	if decoded.TaskPath != original.TaskPath || decoded.Dir != original.Dir || decoded.EID != original.EID || decoded.NullConn != original.NullConn {
		t.Fatalf("decoded request mismatch: got %#v want %#v", decoded, original)
	}
	if len(decoded.Args) != len(original.Args) || len(decoded.Env) != len(original.Env) {
		t.Fatalf("decoded slices mismatch: got args=%#v env=%#v want args=%#v env=%#v", decoded.Args, decoded.Env, original.Args, original.Env)
	}
}

func TestValidatePipelineChildExecRequest(t *testing.T) {
	valid := pipelineChildExecRequest{
		TaskPath: "/tmp/task.sh",
		Dir:      "/tmp",
		Args:     []string{"x"},
		Env:      []string{"A=B"},
		EID:      "eid",
	}
	if err := validatePipelineChildExecRequest(valid); err != nil {
		t.Fatalf("validatePipelineChildExecRequest(valid) error = %v", err)
	}

	cases := []struct {
		name string
		req  pipelineChildExecRequest
	}{
		{name: "missing task path", req: pipelineChildExecRequest{Dir: "/tmp", Env: []string{"A=B"}, EID: "x"}},
		{name: "missing dir", req: pipelineChildExecRequest{TaskPath: "/tmp/x", Env: []string{"A=B"}, EID: "x"}},
		{name: "missing env", req: pipelineChildExecRequest{TaskPath: "/tmp/x", Dir: "/tmp", EID: "x"}},
		{name: "missing eid when not nullconn", req: pipelineChildExecRequest{TaskPath: "/tmp/x", Dir: "/tmp", Env: []string{"A=B"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := validatePipelineChildExecRequest(tc.req); err == nil {
				t.Fatal("validatePipelineChildExecRequest() error = nil, want error")
			}
		})
	}

	nullConnAllowed := pipelineChildExecRequest{
		TaskPath: "/tmp/task.sh",
		Dir:      "/tmp",
		Env:      []string{"A=B"},
		NullConn: true,
	}
	if err := validatePipelineChildExecRequest(nullConnAllowed); err != nil {
		t.Fatalf("validatePipelineChildExecRequest(nullConnAllowed) error = %v", err)
	}
}
