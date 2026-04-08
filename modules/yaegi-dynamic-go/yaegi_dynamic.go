package yaegidynamicgo

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/lnxjedi/gopherbot/robot"
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

var (
	goPath   string
	initOnce sync.Once
	initErr  error
)

const (
	sharedGoPathDirName  = ".yaegi-gopath"
	robotImportPath      = "github.com/lnxjedi/gopherbot/robot"
	installLibImportRoot = "gopherbot.internal/lib"
	configLibImportRoot  = "robot.internal/lib"
)

func sharedGoPath(homePath string) string {
	return filepath.Join(homePath, sharedGoPathDirName)
}

func ensureManagedPathRemoved(dst string) error {
	info, err := os.Lstat(dst)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to inspect managed path '%s': %w", dst, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || info.IsDir() {
		if err := os.RemoveAll(dst); err != nil {
			return fmt.Errorf("failed to remove managed path '%s': %w", dst, err)
		}
		return nil
	}
	if err := os.Remove(dst); err != nil {
		return fmt.Errorf("failed to remove managed file '%s': %w", dst, err)
	}
	return nil
}

func ensureSymlinkIfDirExists(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		if os.IsNotExist(err) {
			if err := ensureManagedPathRemoved(dst); err != nil {
				return err
			}
			return nil
		}
		return fmt.Errorf("failed to stat %s: %w", src, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", src)
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("failed to create GOPATH parent for %s: %w", dst, err)
	}

	if existing, err := os.Lstat(dst); err == nil {
		if existing.Mode()&os.ModeSymlink != 0 {
			linkTarget, err := os.Readlink(dst)
			if err != nil {
				return fmt.Errorf("failed to read symlink %s: %w", dst, err)
			}
			if linkTarget == src {
				return nil
			}
		}
		if err := ensureManagedPathRemoved(dst); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to inspect %s: %w", dst, err)
	}

	if err := os.Symlink(src, dst); err != nil {
		if errors.Is(err, os.ErrExist) {
			if existing, statErr := os.Lstat(dst); statErr == nil && existing.Mode()&os.ModeSymlink != 0 {
				linkTarget, readErr := os.Readlink(dst)
				if readErr == nil && linkTarget == src {
					return nil
				}
			}
		}
		return fmt.Errorf("failed to create symlink %s -> %s: %w", dst, src, err)
	}
	return nil
}

func ensureGoPath(homePath, robotSrcDir, installLibDir, configLibDir string) (string, error) {
	if homePath == "" {
		return "", fmt.Errorf("empty GOPHER_HOME for Yaegi GOPATH")
	}
	root := sharedGoPath(homePath)

	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		return "", fmt.Errorf("failed to create shared Yaegi GOPATH root: %w", err)
	}

	robotDst := filepath.Join(root, "src", filepath.FromSlash(robotImportPath))
	if err := ensureSymlinkIfDirExists(robotSrcDir, robotDst); err != nil {
		return "", fmt.Errorf("failed to stage robot package: %w", err)
	}

	installLibDst := filepath.Join(root, "src", filepath.FromSlash(installLibImportRoot))
	if err := ensureSymlinkIfDirExists(installLibDir, installLibDst); err != nil {
		return "", fmt.Errorf("failed to stage install lib packages: %w", err)
	}

	configLibDst := filepath.Join(root, "src", filepath.FromSlash(configLibImportRoot))
	if err := ensureSymlinkIfDirExists(configLibDir, configLibDst); err != nil {
		return "", fmt.Errorf("failed to stage config lib packages: %w", err)
	}

	return root, nil
}

func initializeInterpreter(privileged bool, env []string) (*interp.Interpreter, error) {
	if initErr != nil {
		return nil, initErr
	}
	i := interp.New(interp.Options{
		GoPath:       goPath,
		Unrestricted: privileged,
		Env:          env,
	})
	if err := i.Use(stdlib.Symbols); err != nil {
		return nil, fmt.Errorf("failed to load standard library: %w", err)
	}
	if err := i.Use(Symbols); err != nil {
		return nil, fmt.Errorf("failed to load robot symbols: %w", err)
	}
	return i, nil
}

func GetPluginConfig(path, name string, privileged bool) (cfg *[]byte, err error) {
	var nullcfg []byte

	defer func() {
		if r := recover(); r != nil {
			cfg = &nullcfg
			err = fmt.Errorf("recovered from panic in GetPluginConfig for plugin '%s': %v", name, r)
		}
	}()

	i, err := initializeInterpreter(privileged, []string{})
	if err != nil {
		return &nullcfg, err
	}
	program, err := i.CompilePath(path)
	if err != nil {
		return &nullcfg, fmt.Errorf("failed to compile plugin: %w", err)
	}
	_, err = i.Execute(program)
	if err != nil {
		return &nullcfg, fmt.Errorf("failed to execute compiled code: %w", err)
	}
	v, err := i.Eval("Configure")
	if err != nil {
		return &nullcfg, fmt.Errorf("failed to retrieve func Configure: %w", err)
	}
	cfgFunc, ok := v.Interface().(func() *[]byte)
	if !ok {
		return &nullcfg, fmt.Errorf("func Configure has incorrect signature: got %T", v.Interface())
	}

	cfg = cfgFunc()

	return cfg, nil
}

func RunPluginHandler(path, name string, env []string, r robot.Robot, l robot.Logger, privileged bool, command string, args ...string) (ret robot.TaskRetVal, err error) {
	defer func() {
		if p := recover(); p != nil {
			ret = robot.MechanismFail
			err = fmt.Errorf("recovered from panic in RunPluginHandler for plugin '%s': %v", name, p)
			l.Log(robot.Error, err.Error())
		}
	}()

	i, err := initializeInterpreter(privileged, env)
	if err != nil {
		return robot.MechanismFail, err
	}
	program, err := i.CompilePath(path)
	if err != nil {
		return robot.MechanismFail, fmt.Errorf("failed to compile plugin: %w", err)
	}
	_, err = i.Execute(program)
	if err != nil {
		return robot.MechanismFail, fmt.Errorf("failed to execute compiled code: %w", err)
	}
	v, err := i.Eval("PluginHandler")
	if err != nil {
		return robot.MechanismFail, fmt.Errorf("failed to retrieve func PluginHandler: %w", err)
	}
	handler, ok := v.Interface().(func(robot.Robot, string, ...string) robot.TaskRetVal)
	if !ok {
		return robot.MechanismFail, fmt.Errorf("PluginHandler has incorrect signature: got %T", v.Interface())
	}

	l.Log(robot.Debug, "Calling external Go plugin: '%s' with command '%s' and args: %q", name, command, args)
	ret = handler(r, command, args...)

	return
}

func RunJobHandler(path, name string, env []string, r robot.Robot, l robot.Logger, privileged bool, args ...string) (ret robot.TaskRetVal, err error) {
	defer func() {
		if p := recover(); p != nil {
			ret = robot.MechanismFail
			err = fmt.Errorf("recovered from panic in RunJobHandler for job '%s': %v", name, p)
			l.Log(robot.Error, err.Error())
		}
	}()

	i, err := initializeInterpreter(privileged, env)
	if err != nil {
		return robot.MechanismFail, err
	}
	program, err := i.CompilePath(path)
	if err != nil {
		return robot.MechanismFail, fmt.Errorf("failed to compile job: %w", err)
	}
	_, err = i.Execute(program)
	if err != nil {
		return robot.MechanismFail, fmt.Errorf("failed to execute compiled code: %w", err)
	}
	v, err := i.Eval("JobHandler")
	if err != nil {
		return robot.MechanismFail, fmt.Errorf("failed to retrieve func JobHandler: %w", err)
	}
	handler, ok := v.Interface().(func(robot.Robot, ...string) robot.TaskRetVal)
	if !ok {
		return robot.MechanismFail, fmt.Errorf("JobHandler has incorrect signature: got %T", v.Interface())
	}

	l.Log(robot.Debug, "Calling external Go job: '%s' with args: %q", name, args)
	ret = handler(r, args...)

	return
}

func RunTaskHandler(path, name string, env []string, r robot.Robot, l robot.Logger, privileged bool, args ...string) (ret robot.TaskRetVal, err error) {
	defer func() {
		if p := recover(); p != nil {
			ret = robot.MechanismFail
			err = fmt.Errorf("recovered from panic in RunTaskHandler for task '%s': %v", name, p)
			l.Log(robot.Error, err.Error())
		}
	}()

	i, err := initializeInterpreter(privileged, env)
	if err != nil {
		return robot.MechanismFail, err
	}
	program, err := i.CompilePath(path)
	if err != nil {
		return robot.MechanismFail, fmt.Errorf("failed to compile task: %w", err)
	}
	_, err = i.Execute(program)
	if err != nil {
		return robot.MechanismFail, fmt.Errorf("failed to execute compiled code: %w", err)
	}
	v, err := i.Eval("TaskHandler")
	if err != nil {
		return robot.MechanismFail, fmt.Errorf("failed to retrieve TaskHandler: %w", err)
	}
	handler, ok := v.Interface().(func(robot.Robot, ...string) robot.TaskRetVal)
	if !ok {
		return robot.MechanismFail, fmt.Errorf("TaskHandler has incorrect signature: got %T", v.Interface())
	}

	l.Log(robot.Debug, "Calling external Go task: '%s' with args: %q", name, args)
	ret = handler(r, args...)

	return
}
