package bot

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"syscall"
)

// Windows argument parsing is all over the map; try to fix it here
// Currently powershell only
func fixInterpreterArgs(interpreter string, args []string) []string {
	ire := regexp.MustCompile(`.*[\/\\!](.*)`)
	var i string
	imatch := ire.FindStringSubmatch(interpreter)
	if len(imatch) == 0 {
		i = interpreter
	} else {
		i = imatch[1]
	}
	switch i {
	case "powershell", "powershell.exe":
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
		_, err := os.Stat(robot.localPath + "/" + plugin.pluginPath)
		if err != nil {
			_, err := os.Stat(robot.installPath + "/" + plugin.pluginPath)
			if err != nil {
				err = fmt.Errorf("Couldn't locate external plugin %s: %v", plugin.name, err)
				return nil, err
			}
			fullPath = robot.installPath + "/" + plugin.pluginPath
			Log(Debug, "Using stock external plugin:", fullPath)
		} else {
			fullPath = robot.localPath + "/" + plugin.pluginPath
			Log(Debug, "Using local external plugin:", fullPath)
		}
	}
	var cfg []byte
	var err error
	if runtime.GOOS == "windows" {
		var interpreter string
		interpreter, err = getInterpreter(fullPath)
		if err != nil {
			err = fmt.Errorf("looking up interpreter for %s: %s", fullPath, err)
			return nil, err
		}
		args := fixInterpreterArgs(interpreter, []string{fullPath, "configure"})
		Log(Debug, fmt.Sprintf("Calling \"%s\" with args: %q", interpreter, args))
		cfg, err = exec.Command(interpreter, args...).Output()
	} else {
		Log(Debug, fmt.Sprintf("Calling \"%s\" with arg: configure", fullPath))
		cfg, err = exec.Command(fullPath, "configure").Output()
	}
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
func callPlugin(bot *Robot, plugin *Plugin, interactive bool, command string, args ...string) (retval PlugRetVal) {
	pluginsRunning.Lock()
	pluginsRunning.count++
	pluginsRunning.Unlock()
	defer func() {
		pluginsRunning.Lock()
		pluginsRunning.count--
		if pluginsRunning.count >= 0 {
			pluginsRunning.Done()
		}
		pluginsRunning.Unlock()
	}()
	var errString string
	defer func() {
		if interactive && errString != "" {
			bot.Reply(errString)
		}
	}()
	if !(plugin.name == "builtInadmin" && command == "abort") {
		defer checkPanic(bot, fmt.Sprintf("Plugin: %s, command: %s, arguments: %v", plugin.name, command, args))
	}
	Log(Debug, fmt.Sprintf("Dispatching command \"%s\" to plugin \"%s\" with arguments \"%#v\"", command, plugin.name, args))
	bot.pluginID = plugin.pluginID
	switch plugin.pluginType {
	case plugBuiltin, plugGo:
		return pluginHandlers[plugin.name].Handler(bot, command, args...)
	case plugExternal:
		var fullPath string // full path to the executable
		if len(plugin.pluginPath) == 0 {
			Log(Error, "pluginPath empty for external plugin:", plugin.name)
			errString = "There was a problem calling the external plugin"
			return MechanismFail
		}
		if byte(plugin.pluginPath[0]) == byte("/"[0]) {
			fullPath = plugin.pluginPath
		} else {
			_, err := os.Stat(robot.localPath + "/" + plugin.pluginPath)
			if err != nil {
				_, err := os.Stat(robot.installPath + "/" + plugin.pluginPath)
				if err != nil {
					Log(Error, fmt.Errorf("Couldn't locate external plugin %s: %v", plugin.name, err))
					errString = "There was a problem calling the external plugin"
					return MechanismFail
				}
				fullPath = robot.installPath + "/" + plugin.pluginPath
				Log(Debug, "Using stock external plugin:", fullPath)
			} else {
				fullPath = robot.localPath + "/" + plugin.pluginPath
				Log(Debug, "Using local external plugin:", fullPath)
			}
		}
		interpreter, err := getInterpreter(fullPath)
		if err != nil {
			err = fmt.Errorf("looking up interpreter for %s: %s", fullPath, err)
			Log(Error, fmt.Sprintf("Unable to call external plugin %s, no interpreter found: %s", fullPath, err))
			errString = "There was a problem calling the external plugin"
			return MechanismFail
		}
		externalArgs := make([]string, 0, 5+len(args))
		// on Windows, we exec the interpreter with the script as first arg
		if runtime.GOOS == "windows" {
			externalArgs = append(externalArgs, fullPath)
		}
		externalArgs = append(externalArgs, command)
		externalArgs = append(externalArgs, args...)
		externalArgs = fixInterpreterArgs(interpreter, externalArgs)
		Log(Debug, fmt.Sprintf("Calling \"%s\" with interpreter \"%s\" and args: %q", fullPath, interpreter, externalArgs))
		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.Command(interpreter, externalArgs...)
		} else {
			cmd = exec.Command(fullPath, externalArgs...)
		}
		cmd.Env = append(os.Environ(), []string{
			fmt.Sprintf("GOPHER_CHANNEL=%s", bot.Channel),
			fmt.Sprintf("GOPHER_USER=%s", bot.User),
			fmt.Sprintf("GOPHER_PLUGIN_ID=%s", plugin.pluginID),
		}...)
		// close stdout on the external plugin...
		cmd.Stdout = nil
		// but hold on to stderr in case we need to log an error
		stderr, err := cmd.StderrPipe()
		if err != nil {
			Log(Error, fmt.Errorf("Creating stderr pipe for external command \"%s\": %v", fullPath, err))
			errString = fmt.Sprintf("There were errors calling external plugin \"%s\", you might want to ask an administrator to check the logs", plugin.name)
			return MechanismFail
		}
		if err = cmd.Start(); err != nil {
			Log(Error, fmt.Errorf("Starting command \"%s\": %v", fullPath, err))
			errString = fmt.Sprintf("There were errors calling external plugin \"%s\", you might want to ask an administrator to check the logs", plugin.name)
			return MechanismFail
		}
		var stdErrBytes []byte
		if stdErrBytes, err = ioutil.ReadAll(stderr); err != nil {
			Log(Error, fmt.Errorf("Reading from stderr for external command \"%s\": %v", fullPath, err))
			errString = fmt.Sprintf("There were errors calling external plugin \"%s\", you might want to ask an administrator to check the logs", plugin.name)
			return MechanismFail
		}
		stdErrString := string(stdErrBytes)
		if len(stdErrString) > 0 {
			Log(Warn, fmt.Errorf("Output from stderr of external command \"%s\": %s", fullPath, stdErrString))
			errString = fmt.Sprintf("There was error output while calling external plugin \"%s\", you might want to ask an administrator to check the logs", plugin.name)
		}
		if err = cmd.Wait(); err != nil {
			Log(Error, fmt.Errorf("Waiting on external command \"%s\": %v", fullPath, err))
			errString = fmt.Sprintf("There were errors calling external plugin \"%s\", you might want to ask an administrator to check the logs", plugin.name)
			if exitstatus, ok := err.(*exec.ExitError); ok {
				if status, ok := exitstatus.Sys().(syscall.WaitStatus); ok {
					retval = PlugRetVal(status.ExitStatus())
				}
			}
		}
	}
	return retval
}
