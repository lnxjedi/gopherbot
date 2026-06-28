package bot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

const (
	gopherEnvHandoffFDEnv    = "_GOPHERBOT_ENV_FD"
	maxGopherEnvHandoffBytes = 16 * 1024
)

type gopherEnvHandoff struct {
	Env map[string]string `json:"env"`
}

func initializeGopherEnvironmentHandoff() error {
	if fd, ok, err := takeReadEnvFromFD(); err != nil {
		return err
	} else if ok {
		file := os.NewFile(uintptr(fd), "gopher-env-handoff")
		if file == nil {
			return fmt.Errorf("opening GOPHER environment handoff fd %d", fd)
		}
		defer file.Close()
		if err := readGopherEnvHandoff(file); err != nil {
			return err
		}
		scrubEnvironment()
		return nil
	}
	if privsepInternalCommandActive() {
		return nil
	}
	env := copyStartupEnv()
	if len(env) == 0 {
		if len(directGopherEnvironment()) > 0 {
			scrubEnvironment()
		}
		return nil
	}
	return reexecWithGopherEnvironment(env)
}

func takeReadEnvFromFD() (int, bool, error) {
	raw := strings.TrimSpace(os.Getenv(gopherEnvHandoffFDEnv))
	if raw == "" {
		return -1, false, nil
	}
	_ = os.Unsetenv(gopherEnvHandoffFDEnv)
	fd, err := strconv.Atoi(raw)
	if err != nil || fd < 3 {
		return -1, true, fmt.Errorf("invalid GOPHER environment handoff fd %q", raw)
	}
	return fd, true, nil
}

func directGopherEnvironment() map[string]string {
	env := make(map[string]string)
	for _, envVar := range os.Environ() {
		key, value, ok := strings.Cut(envVar, "=")
		if !ok || !strings.HasPrefix(key, "GOPHER_") {
			continue
		}
		env[key] = value
	}
	return env
}

func environmentWithoutDirectGopher() []string {
	env := make([]string, 0, len(os.Environ()))
	for _, envVar := range os.Environ() {
		key, _, ok := strings.Cut(envVar, "=")
		if !ok || strings.HasPrefix(key, "GOPHER_") || key == gopherEnvHandoffFDEnv {
			continue
		}
		env = append(env, envVar)
	}
	return env
}

func readGopherEnvHandoff(r io.Reader) error {
	var payload gopherEnvHandoff
	limited := io.LimitReader(r, maxGopherEnvHandoffBytes+1)
	raw, err := io.ReadAll(limited)
	if err != nil {
		return fmt.Errorf("reading GOPHER environment handoff: %w", err)
	}
	if len(raw) > maxGopherEnvHandoffBytes {
		return fmt.Errorf("GOPHER environment handoff exceeds %d bytes", maxGopherEnvHandoffBytes)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	if err := dec.Decode(&payload); err != nil {
		return fmt.Errorf("reading GOPHER environment handoff: %w", err)
	}
	for key, value := range payload.Env {
		if !validGopherEnvKey(key) {
			return fmt.Errorf("invalid GOPHER environment handoff key %q", key)
		}
		setGopherEnvValue(key, value)
		setStartupEnvValue(key, value)
	}
	return nil
}

func validGopherEnvKey(key string) bool {
	return strings.HasPrefix(key, "GOPHER_") && key != "GOPHER_" && !strings.Contains(key, "=")
}

func encodeGopherEnvHandoff(env map[string]string) ([]byte, error) {
	for key := range env {
		if !validGopherEnvKey(key) {
			return nil, fmt.Errorf("invalid GOPHER environment key %q", key)
		}
	}
	raw, err := json.Marshal(gopherEnvHandoff{Env: env})
	if err != nil {
		return nil, err
	}
	if len(raw) > maxGopherEnvHandoffBytes {
		return nil, fmt.Errorf("GOPHER environment handoff is too large: %d bytes", len(raw))
	}
	return raw, nil
}

func reexecWithGopherEnvironment(env map[string]string) error {
	raw, err := encodeGopherEnvHandoff(env)
	if err != nil {
		return err
	}
	return execSelfWithGopherEnvPayload(raw, environmentWithoutDirectGopher())
}

func restartWithGopherEnvironment() error {
	raw, err := encodeGopherEnvHandoff(copyStartupEnv())
	if err != nil {
		return err
	}
	return execSelfWithGopherEnvPayload(raw, environmentWithoutDirectGopher())
}

func execSelfWithGopherEnvPayload(raw []byte, env []string) error {
	fds := []int{-1, -1}
	if err := unix.Pipe(fds); err != nil {
		return fmt.Errorf("creating GOPHER environment handoff pipe: %w", err)
	}
	readerFD, writerFD := fds[0], fds[1]
	defer unix.Close(readerFD)
	writerOpen := true
	defer func() {
		if writerOpen {
			_ = unix.Close(writerFD)
		}
	}()

	written := 0
	for written < len(raw) {
		n, err := unix.Write(writerFD, raw[written:])
		if err != nil {
			return fmt.Errorf("writing GOPHER environment handoff: %w", err)
		}
		if n == 0 {
			return fmt.Errorf("writing GOPHER environment handoff: short write")
		}
		written += n
	}
	if err := unix.Close(writerFD); err != nil {
		return fmt.Errorf("closing GOPHER environment handoff writer: %w", err)
	}
	writerOpen = false

	args := append([]string{}, os.Args...)
	execPath, err := os.Executable()
	if err != nil {
		return err
	}
	return unix.Exec(execPath, args, append(env, fmt.Sprintf("%s=%d", gopherEnvHandoffFDEnv, readerFD)))
}

func newSelfCommandWithGopherEnvironment(bin string, args []string) (*exec.Cmd, func(), error) {
	env := copyStartupEnv()
	cmdArgs := append([]string{}, args...)
	cleanup := func() {}
	var extraFiles []*os.File
	if len(env) > 0 {
		raw, err := encodeGopherEnvHandoff(env)
		if err != nil {
			return nil, cleanup, err
		}
		reader, writer, err := os.Pipe()
		if err != nil {
			return nil, cleanup, fmt.Errorf("creating GOPHER environment handoff pipe: %w", err)
		}
		cleanup = func() {
			_ = reader.Close()
		}
		if _, err := writer.Write(raw); err != nil {
			_ = writer.Close()
			cleanup()
			return nil, func() {}, fmt.Errorf("writing GOPHER environment handoff: %w", err)
		}
		if err := writer.Close(); err != nil {
			cleanup()
			return nil, func() {}, fmt.Errorf("closing GOPHER environment handoff writer: %w", err)
		}
		extraFiles = []*os.File{reader}
	}
	cmd := exec.Command(bin, cmdArgs...)
	cmd.Env = environmentWithoutDirectGopher()
	if len(extraFiles) > 0 {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%d", gopherEnvHandoffFDEnv, 3))
	}
	cmd.ExtraFiles = extraFiles
	return cmd, cleanup, nil
}
