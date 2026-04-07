package bot

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

func TestNewPipelineChildRPCCommand(t *testing.T) {
	oldInstallPath := installPath
	oldConfigFull := configFull
	installPath = "/tmp/test-install"
	configFull = "/tmp/test-config"
	t.Cleanup(func() {
		installPath = oldInstallPath
		configFull = oldConfigFull
	})

	cmd := newPipelineChildRPCCommand()
	if len(cmd.Args) < 2 || cmd.Args[1] != pipelineChildRPCCommand {
		t.Fatalf("child rpc command args = %#v, want second arg %q", cmd.Args, pipelineChildRPCCommand)
	}
	if !envContains(cmd.Env, "GOPHER_INSTALLDIR="+installPath) {
		t.Fatalf("child rpc env missing GOPHER_INSTALLDIR=%s: %#v", installPath, cmd.Env)
	}
	if !envContains(cmd.Env, "GOPHER_CONFIGDIR="+configFull) {
		t.Fatalf("child rpc env missing GOPHER_CONFIGDIR=%s: %#v", configFull, cmd.Env)
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

func TestEnsurePipelineRPCGoInitializedUsesChildEnv(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed")
	}
	repoRoot := filepath.Clean(filepath.Dir(filepath.Dir(thisFile)))
	configDir := t.TempDir()

	oldInstallPath := installPath
	oldConfigFull := configFull
	oldInitOnce := pipelineRPCGoInitOnce
	oldInitErr := pipelineRPCGoInitErr
	t.Setenv("GOPHER_INSTALLDIR", repoRoot)
	t.Setenv("GOPHER_CONFIGDIR", configDir)
	installPath = ""
	configFull = ""
	pipelineRPCGoInitOnce = sync.Once{}
	pipelineRPCGoInitErr = nil
	t.Cleanup(func() {
		installPath = oldInstallPath
		configFull = oldConfigFull
		pipelineRPCGoInitOnce = oldInitOnce
		pipelineRPCGoInitErr = oldInitErr
	})

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	workDir := t.TempDir()
	if err := os.Chdir(workDir); err != nil {
		t.Fatalf("chdir temp workdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})

	if err := ensurePipelineRPCGoInitialized(); err != nil {
		t.Fatalf("ensurePipelineRPCGoInitialized() error = %v", err)
	}
	if installPath != repoRoot {
		t.Fatalf("installPath = %q, want %q", installPath, repoRoot)
	}
	if configFull != configDir {
		t.Fatalf("configFull = %q, want %q", configFull, configDir)
	}
	entries, err := os.ReadDir(workDir)
	if err != nil {
		t.Fatalf("readdir(%s): %v", workDir, err)
	}
	found := ""
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), ".gopath-") {
			found = filepath.Join(workDir, entry.Name())
			break
		}
	}
	if found == "" {
		t.Fatalf("expected staged .gopath-* directory in %s", workDir)
	}
	if _, err := os.Stat(found); err != nil {
		t.Fatalf("expected staged goPath %q: %v", found, err)
	}
}

func envContains(env []string, want string) bool {
	for _, candidate := range env {
		if candidate == want {
			return true
		}
	}
	return false
}
