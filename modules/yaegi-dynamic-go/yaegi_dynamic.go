package yaegidynamicgo

import (
	"fmt"
	"io"
	"log"
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

func init() {
	initOnce.Do(func() {
		currentDir, err := os.Getwd()
		if err != nil {
			initErr = fmt.Errorf("failed to get current directory: %w", err)
			return
		}
		ex, err := os.Executable()
		if err != nil {
			initErr = fmt.Errorf("failed to get executable path: %w", err)
			return
		}
		installPath, err := filepath.Abs(filepath.Dir(ex))
		if err != nil {
			initErr = fmt.Errorf("failed to get install path: %w", err)
			return
		}

		goPath = filepath.Join(currentDir, ".gopath")
		if _, err := os.Stat(goPath); err == nil {
			err = os.RemoveAll(goPath)
			if err != nil {
				initErr = fmt.Errorf("failed to remove existing .gopath: %w", err)
				return
			}
		}

		robotSrcPath := filepath.Join(goPath, "src", "github.com", "lnxjedi", "gopherbot", "robot")
		err = os.MkdirAll(robotSrcPath, 0755)
		if err != nil {
			initErr = fmt.Errorf("failed to create robot source directory: %w", err)
			return
		}

		robotInstallPath := filepath.Join(installPath, "robot")
		err = copyDir(robotInstallPath, robotSrcPath)
		if err != nil {
			initErr = fmt.Errorf("failed to copy robot package: %w", err)
			return
		}

		os.Setenv("GOPATH", goPath)
		log.Printf("Yaegi GOPATH set to: %s", goPath)
	})
}

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

func initializeInterpreter() (*interp.Interpreter, error) {
	if initErr != nil {
		return nil, initErr
	}
	i := interp.New(interp.Options{
		GoPath: goPath,
	})
	if err := i.Use(stdlib.Symbols); err != nil {
		return nil, fmt.Errorf("failed to load standard library: %w", err)
	}
	if err := i.Use(robot.Symbols); err != nil {
		return nil, fmt.Errorf("failed to load robot symbols: %w", err)
	}
	return i, nil
}

func GetPluginHandler(path string) (func(r robot.Robot, command string, args ...string) robot.TaskRetVal, error) {
	i, err := initializeInterpreter()
	if err != nil {
		return nil, err
	}
	_, err = i.CompilePath(path)
	if err != nil {
		return nil, fmt.Errorf("failed to compile plugin: %w", err)
	}
	v, err := i.Eval("PluginHandler")
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve PluginHandler: %w", err)
	}
	handler, ok := v.Interface().(func(robot.Robot, string, ...string) robot.TaskRetVal)
	if !ok {
		return nil, fmt.Errorf("PluginHandler has incorrect signature: got %T", v.Interface())
	}
	return handler, nil
}

func GetJobHandler(path string) (func(r robot.Robot, args ...string) robot.TaskRetVal, error) {
	i, err := initializeInterpreter()
	if err != nil {
		return nil, err
	}
	_, err = i.CompilePath(path)
	if err != nil {
		return nil, fmt.Errorf("failed to compile job: %w", err)
	}
	v, err := i.Eval("JobHandler")
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve JobHandler: %w", err)
	}
	handler, ok := v.Interface().(func(robot.Robot, ...string) robot.TaskRetVal)
	if !ok {
		return nil, fmt.Errorf("JobHandler has incorrect signature: got %T", v.Interface())
	}
	return handler, nil
}

func GetTaskHandler(path string) (func(r robot.Robot, args ...string) robot.TaskRetVal, error) {
	i, err := initializeInterpreter()
	if err != nil {
		return nil, err
	}
	_, err = i.CompilePath(path)
	if err != nil {
		return nil, fmt.Errorf("failed to compile task: %w", err)
	}
	v, err := i.Eval("TaskHandler")
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve TaskHandler: %w", err)
	}
	handler, ok := v.Interface().(func(robot.Robot, ...string) robot.TaskRetVal)
	if !ok {
		return nil, fmt.Errorf("TaskHandler has incorrect signature: got %T", v.Interface())
	}
	return handler, nil
}
