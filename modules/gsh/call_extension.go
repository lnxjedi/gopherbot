package gsh

import (
	"bufio"
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/lnxjedi/gopherbot/robot"
	luamod "github.com/lnxjedi/gopherbot/v2/modules/lua"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

//go:embed assets/gopherbot_v1.gsh
var assets embed.FS

type BotAPI = luamod.BotAPI

type shellContext struct {
	taskName string
	taskPath string
	logger   robot.Logger
	bot      BotAPI
	envList  []string
	envMap   map[string]string
}

func CallExtension(taskPath, taskName string, env []string, logger robot.Logger, bot BotAPI, args []string) (robot.TaskRetVal, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	ret, err := runScript(taskPath, taskName, env, logger, bot, args, &stdout, &stderr)
	logBufferedOutput(logger, &stdout, &stderr)
	return ret, err
}

func GetPluginConfig(taskPath, taskName string, env []string, logger robot.Logger) (*[]byte, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	ret, err := runScript(taskPath, taskName, env, logger, nil, []string{"configure"}, &stdout, &stderr)
	if err != nil {
		return nil, err
	}
	if ret != robot.Normal {
		return nil, fmt.Errorf("gsh configure for '%s' returned %s", taskName, ret)
	}
	cfg := stdout.Bytes()
	if len(bytes.TrimSpace(cfg)) == 0 {
		empty := []byte{}
		return &empty, nil
	}
	out := append([]byte(nil), cfg...)
	return &out, nil
}

func runScript(taskPath, taskName string, env []string, logger robot.Logger, bot BotAPI, args []string, stdout, stderr io.Writer) (robot.TaskRetVal, error) {
	shim, err := assets.ReadFile("assets/gopherbot_v1.gsh")
	if err != nil {
		return robot.MechanismFail, fmt.Errorf("loading built-in gsh shim: %w", err)
	}
	script, err := os.ReadFile(taskPath)
	if err != nil {
		return robot.MechanismFail, fmt.Errorf("reading gsh script '%s': %w", taskName, err)
	}
	source := string(shim) + "\n\n" + string(script)
	file, err := syntax.NewParser().Parse(strings.NewReader(source), taskPath)
	if err != nil {
		return robot.MechanismFail, fmt.Errorf("parsing gsh script '%s': %w", taskName, err)
	}

	ctx := &shellContext{
		taskName: taskName,
		taskPath: taskPath,
		logger:   logger,
		bot:      bot,
		envList:  append([]string(nil), env...),
		envMap:   envListToMap(env),
	}

	runner, err := interp.New(
		interp.Dir(filepath.Dir(taskPath)),
		interp.Env(expand.ListEnviron(env...)),
		interp.StdIO(nil, stdout, stderr),
		interp.Params(args...),
		interp.OpenHandler(ctx.openHandler),
		interp.ExecHandlers(ctx.execHandler),
	)
	if err != nil {
		return robot.MechanismFail, fmt.Errorf("initializing gsh runner for '%s': %w", taskName, err)
	}

	if err := runner.Run(context.Background(), file); err != nil {
		var status interp.ExitStatus
		if errors.As(err, &status) {
			return robot.TaskRetVal(status), nil
		}
		return robot.MechanismFail, fmt.Errorf("running gsh script '%s': %w", taskName, err)
	}
	return robot.Normal, nil
}

func logBufferedOutput(logger robot.Logger, stdout, stderr *bytes.Buffer) {
	if logger == nil {
		return
	}
	logBuffer := func(level robot.LogLevel, prefix string, buf *bytes.Buffer) {
		scanner := bufio.NewScanner(bytes.NewReader(buf.Bytes()))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			logger.Log(level, "%s%s", prefix, line)
		}
	}
	logBuffer(robot.Debug, "gsh stdout: ", stdout)
	logBuffer(robot.Warn, "gsh stderr: ", stderr)
}

func envListToMap(env []string) map[string]string {
	m := make(map[string]string, len(env))
	for _, item := range env {
		key, value, ok := strings.Cut(item, "=")
		if !ok || key == "" {
			continue
		}
		m[key] = value
	}
	return m
}

func (c *shellContext) lookupEnv(key string) (string, bool) {
	value, ok := c.envMap[key]
	return value, ok
}

func (c *shellContext) openHandler(ctx context.Context, path string, flag int, perm os.FileMode) (io.ReadWriteCloser, error) {
	if compat, ok := c.compatLibrary(path); ok {
		return newMemoryReadWriteCloser(compat), nil
	}
	return interp.DefaultOpenHandler()(ctx, path, flag, perm)
}

func (c *shellContext) compatLibrary(path string) (string, bool) {
	clean := filepath.Clean(path)
	base := filepath.Base(clean)
	switch base {
	case "gopherbot_v1.sh", "gopherbot_v1.gsh":
		data, err := assets.ReadFile("assets/gopherbot_v1.gsh")
		if err != nil {
			return "", false
		}
		return string(data), true
	}
	if installDir, ok := c.lookupEnv("GOPHER_INSTALLDIR"); ok {
		if clean == filepath.Join(installDir, "lib", "gopherbot_v1.sh") || clean == filepath.Join(installDir, "lib", "gopherbot_v1.gsh") {
			data, err := assets.ReadFile("assets/gopherbot_v1.gsh")
			if err != nil {
				return "", false
			}
			return string(data), true
		}
	}
	return "", false
}

type memoryReadWriteCloser struct {
	*bytes.Reader
}

func newMemoryReadWriteCloser(s string) io.ReadWriteCloser {
	return &memoryReadWriteCloser{Reader: bytes.NewReader([]byte(s))}
}

func (m *memoryReadWriteCloser) Write([]byte) (int, error) {
	return 0, io.EOF
}

func (m *memoryReadWriteCloser) Close() error {
	return nil
}
