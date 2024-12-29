package yaegidynamicgo

import (
	"fmt"
	"io"
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

func copyDir(src string, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat source directory: %w", err)
	}
	if !srcInfo.IsDir() {
		return fmt.Errorf("source is not a directory")
	}
	err = os.MkdirAll(dst, srcInfo.Mode())
	if err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("failed to read source directory: %w", err)
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			err = copyDir(srcPath, dstPath)
			if err != nil {
				return err
			}
		} else {
			err = copyFile(srcPath, dstPath)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file '%s': %w", src, err)
	}
	defer sourceFile.Close()

	srcInfo, err := sourceFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat source file '%s': %w", src, err)
	}

	destFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return fmt.Errorf("failed to create destination file '%s': %w", dst, err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy from '%s' to '%s': %w", src, dst, err)
	}
	return nil
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

func GetPluginConfig(path, name string) (cfg *[]byte, err error) {
	var nullcfg []byte

	defer func() {
		if r := recover(); r != nil {
			cfg = &nullcfg
			err = fmt.Errorf("recovered from panic in GetPluginConfig for plugin '%s': %v", name, r)
		}
	}()

	i, err := initializeInterpreter(false, []string{})
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
