// +build windows

package bot

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

// Windows argument parsing is all over the map; try to fix it here
// Currently powershell only
func fixInterpreterArgs(interpreter string, args []string) []string {
	if strings.HasSuffix(interpreter, "powershell.exe") {
		for i, _ := range args {
			args[i] = strings.Replace(args[i], " ", "` ", -1)
			args[i] = strings.Replace(args[i], ",", "`,", -1)
			args[i] = strings.Replace(args[i], ";", "`;", -1)
			if args[i] == "" {
				args[i] = "\"\""
			}
		}
	}
	return args
}

// emulate Unix script convention by calling external scripts with
// an interpreter.
func getInterpreter(scriptPath string) (string, error) {
	script, err := os.Open(scriptPath)
	if err != nil {
		err = fmt.Errorf("opening file: %s", err)
		Log(Error, fmt.Sprintf("Problem getting interpreter for %s: %s", scriptPath, err))
		return "", err
	}
	r := bufio.NewReader(script)
	iline, err := r.ReadString('\n')
	if err != nil {
		err = fmt.Errorf("reading first line: %s", err)
		Log(Error, fmt.Sprintf("Problem getting interpreter for %s: %s", scriptPath, err))
		return "", err
	}
	if !strings.HasPrefix(iline, "#!") {
		err := fmt.Errorf("Problem getting interpreter for %s; first line doesn't start with \"#!\"", scriptPath)
		Log(Error, err)
		return "", err
	}
	iline = strings.TrimRight(iline, "\n\r")
	interpreter := strings.TrimPrefix(iline, "#!")
	Log(Debug, fmt.Sprintf("Detected interpreter for %s: %s", scriptPath, interpreter))
	return interpreter, nil
}

func getExtDefCfg(plugin *Plugin) (*[]byte, error) {
	var fullPath string
	if byte(plugin.pluginPath[0]) == byte("/"[0]) {
		fullPath = plugin.pluginPath
	} else {
		_, err := os.Stat(b.localPath + "/" + plugin.pluginPath)
		if err != nil {
			_, err := os.Stat(b.installPath + "/" + plugin.pluginPath)
			if err != nil {
				err = fmt.Errorf("Couldn't locate external plugin %s: %v", plugin.name, err)
				return nil, err
			}
			fullPath = b.installPath + "/" + plugin.pluginPath
			Log(Debug, "Using stock external plugin:", fullPath)
		} else {
			fullPath = b.localPath + "/" + plugin.pluginPath
			Log(Debug, "Using local external plugin:", fullPath)
		}
	}
	interpreter, err := getInterpreter(fullPath)
	if err != nil {
		err = fmt.Errorf("looking up interpreter for %s: %s", fullPath, err)
		return nil, err
	}
	externalArgs := []string{fullPath, "", "", "", "configure"}
	externalArgs = fixInterpreterArgs(interpreter, externalArgs)
	Log(Debug, fmt.Sprintf("Calling \"%s\" with interpreter \"%s\" and args: %q", fullPath, interpreter, externalArgs))
	cfg, err := exec.Command(interpreter, externalArgs...).Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			err = fmt.Errorf("Problem retrieving default configuration for external plugin \"%s\", skipping: \"%v\", output: %s", fullPath, err, exitErr.Stderr)
		} else {
			err = fmt.Errorf("Problem retrieving default configuration for external plugin \"%s\", skipping: \"%v\"", fullPath, err)
		}
		return nil, err
	}
	return &cfg, nil
}

// callPlugin (normally called with go ...) sends a command to a plugin.
func callPlugin(bot *Robot, plugin *Plugin, command string, args ...string) {
	shutdownMutex.Lock()
	plugRunningCounter++
	shutdownMutex.Unlock()
	defer func() {
		shutdownMutex.Lock()
		plugRunningCounter--
		shutdownMutex.Unlock()
		plugRunningWaitGroup.Done()
	}()
	defer checkPanic(bot, fmt.Sprintf("Plugin: %s, command: %s, arguments: %v", plugin.name, command, args))
	Log(Debug, fmt.Sprintf("Dispatching command \"%s\" to plugin \"%s\" with arguments \"%#v\"", command, plugin.name, args))
	bot.pluginID = plugin.pluginID
	switch plugin.pluginType {
	case plugBuiltin, plugGo:
		pluginHandlers[plugin.name].Handler(bot, command, args...)
	case plugExternal:
		var fullPath string // full path to the executable
		if len(plugin.pluginPath) == 0 {
			Log(Error, "pluginPath empty for external plugin:", plugin.name)
		}
		if byte(plugin.pluginPath[0]) == byte("/"[0]) {
			fullPath = plugin.pluginPath
		} else {
			_, err := os.Stat(b.localPath + "/" + plugin.pluginPath)
			if err != nil {
				_, err := os.Stat(b.installPath + "/" + plugin.pluginPath)
				if err != nil {
					Log(Error, fmt.Errorf("Couldn't locate external plugin %s: %v", plugin.name, err))
					return
				}
				fullPath = b.installPath + "/" + plugin.pluginPath
				Log(Debug, "Using stock external plugin:", fullPath)
			} else {
				fullPath = b.localPath + "/" + plugin.pluginPath
				Log(Debug, "Using local external plugin:", fullPath)
			}
		}
		interpreter, err := getInterpreter(fullPath)
		if err != nil {
			err = fmt.Errorf("looking up interpreter for %s: %s", fullPath, err)
			Log(Error, fmt.Sprintf("Unable to call external plugin %s, no interpreter found: %s", fullPath, err))
			return
		}
		externalArgs := make([]string, 0, 5+len(args))
		externalArgs = append(externalArgs, fullPath)
		externalArgs = append(externalArgs, bot.Channel, bot.User, plugin.pluginID, command)
		externalArgs = append(externalArgs, args...)
		externalArgs = fixInterpreterArgs(interpreter, externalArgs)
		Log(Debug, fmt.Sprintf("Calling \"%s\" with interpreter \"%s\" and args: %q", fullPath, interpreter, externalArgs))
		cmd := exec.Command(interpreter, externalArgs...)
		// close stdout on the external plugin...
		cmd.Stdout = nil
		// but hold on to stderr in case we need to log an error
		stderr, err := cmd.StderrPipe()
		if err != nil {
			Log(Error, fmt.Errorf("Creating stderr pipe for external command \"%s\": %v", fullPath, err))
			bot.Reply(fmt.Sprintf("There were errors calling external plugin \"%s\", you might want to ask an administrator to check the logs", plugin.name))
			return
		}
		if err = cmd.Start(); err != nil {
			Log(Error, fmt.Errorf("Starting command \"%s\": %v", fullPath, err))
			bot.Reply(fmt.Sprintf("There were errors calling external plugin \"%s\", you might want to ask an administrator to check the logs", plugin.name))
			return
		}
		defer func() {
			if err = cmd.Wait(); err != nil {
				Log(Error, fmt.Errorf("Waiting on external command \"%s\": %v", fullPath, err))
				bot.Reply(fmt.Sprintf("There were errors calling external plugin \"%s\", you might want to ask an administrator to check the logs", plugin.name))
			}
		}()
		stdErrBytes, err := ioutil.ReadAll(stderr)
		if err != nil {
			Log(Error, fmt.Errorf("Reading from stderr for external command \"%s\": %v", fullPath, err))
			bot.Reply(fmt.Sprintf("There were errors calling external plugin \"%s\", you might want to ask an administrator to check the logs", plugin.name))
			return
		}
		stdErrString := string(stdErrBytes)
		if len(stdErrString) > 0 {
			Log(Warn, fmt.Errorf("Output from stderr of external command \"%s\": %s", fullPath, stdErrString))
			bot.Reply(fmt.Sprintf("There was error output while calling external plugin \"%s\", you might want to ask an administrator to check the logs", plugin.name))
		}
	}
}
