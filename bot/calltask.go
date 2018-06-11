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
		for i := range args {
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

func getTaskPath(task *botTask) (string, error) {
	if len(task.scriptPath) == 0 {
		err := fmt.Errorf("Path empty for external task: %s", task.name)
		Log(Error, err)
		return "", err
	}
	var fullPath string
	robot.RLock()
	configPath := robot.configPath
	installPath := robot.installPath
	robot.RUnlock()
	if byte(task.scriptPath[0]) == byte("/"[0]) {
		fullPath = task.scriptPath
		_, err := os.Stat(fullPath)
		if err == nil {
			Log(Debug, "Using fully specified path to plugin:", fullPath)
			return fullPath, nil
		}
		err = fmt.Errorf("Invalid path for external plugin: %s (%v)", fullPath, err)
		Log(Error, err)
		return "", err
	}
	if len(configPath) > 0 {
		_, err := os.Stat(configPath + "/" + task.scriptPath)
		if err == nil {
			fullPath = configPath + "/" + task.scriptPath
			Log(Debug, "Using external plugin from configPath:", fullPath)
			return fullPath, nil
		}
	}
	_, err := os.Stat(installPath + "/" + task.scriptPath)
	if err == nil {
		fullPath = installPath + "/" + task.scriptPath
		Log(Debug, "Using stock external plugin:", fullPath)
		return fullPath, nil
	}
	err = fmt.Errorf("Couldn't locate external plugin %s: %v", task.name, err)
	Log(Error, err)
	return "", err
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

func getExtDefCfg(task *botTask) (*[]byte, error) {
	var fullPath string
	var err error
	if fullPath, err = getTaskPath(task); err != nil {
		return nil, err
	}
	var cfg []byte
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

// callTask does the real work of running a job or plugin with a command and arguments.
func (bot *Robot) callTask(t interface{}, background bool, interactive bool, command string, args ...string) (retval TaskRetVal) {
	task, plugin, _ := getTask(t)
	isPlugin := plugin != nil
	// This should only happen in the rare case that a configured authorizer or elevator is disabled
	if task.Disabled {
		msg := fmt.Sprintf("callTask failed on disabled task %s; reason: %s", task.name, task.reason)
		bot.Log(Error, msg)
		bot.debug(bot.callerID, msg, false)
		return ConfigurationError
	}
	if background {
		robot.Add(1)
		robot.Lock()
		robot.pluginsRunning++
		robot.Unlock()
		defer func() {
			robot.Lock()
			robot.pluginsRunning--
			// TODO: this check shouldn't be necessary; remove and test
			if robot.pluginsRunning >= 0 {
				robot.Done()
			}
			robot.Unlock()
		}()
	}
	var errString string
	defer func() {
		if interactive && errString != "" {
			bot.Reply(errString)
		}
	}()
	if !(task.name == "builtInadmin" && command == "abort") {
		defer checkPanic(bot, fmt.Sprintf("Plugin: %s, command: %s, arguments: %v", task.name, command, args))
	}
	Log(Debug, fmt.Sprintf("Dispatching command \"%s\" to plugin \"%s\" with arguments \"%#v\"", command, task.name, args))
	bot.callerID = task.taskID
	if isPlugin && plugin.pluginType == plugGo {
		if command != "init" {
			emit(GoPluginRan)
		}
		Log(Debug, fmt.Sprintf("Call go plugin: \"%s\" with args: %q", task.name, args))
		return pluginHandlers[task.name].Handler(bot, command, args...)
	}
	var fullPath string // full path to the executable
	var err error
	fullPath, err = getTaskPath(task)
	if err != nil {
		emit(ScriptPluginBadPath)
		return MechanismFail
	}
	interpreter, err := getInterpreter(fullPath)
	if err != nil {
		err = fmt.Errorf("looking up interpreter for %s: %s", fullPath, err)
		Log(Error, fmt.Sprintf("Unable to call external plugin %s, no interpreter found: %s", fullPath, err))
		errString = "There was a problem calling an external plugin"
		emit(ScriptPluginBadInterpreter)
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
		fmt.Sprintf("GOPHER_CALLER_ID=%s", task.taskID),
		fmt.Sprintf("GOPHER_PROTOCOL=%s", bot.Protocol),
	}...)
	// close stdout on the external plugin...
	cmd.Stdout = nil
	// but hold on to stderr in case we need to log an error
	stderr, err := cmd.StderrPipe()
	if err != nil {
		Log(Error, fmt.Errorf("Creating stderr pipe for external command \"%s\": %v", fullPath, err))
		errString = fmt.Sprintf("There were errors calling external plugin \"%s\", you might want to ask an administrator to check the logs", task.name)
		return MechanismFail
	}
	if err = cmd.Start(); err != nil {
		Log(Error, fmt.Errorf("Starting command \"%s\": %v", fullPath, err))
		errString = fmt.Sprintf("There were errors calling external plugin \"%s\", you might want to ask an administrator to check the logs", task.name)
		return MechanismFail
	}
	if command != "init" {
		emit(ScriptPluginRan)
	}
	var stdErrBytes []byte
	if stdErrBytes, err = ioutil.ReadAll(stderr); err != nil {
		Log(Error, fmt.Errorf("Reading from stderr for external command \"%s\": %v", fullPath, err))
		errString = fmt.Sprintf("There were errors calling external plugin \"%s\", you might want to ask an administrator to check the logs", task.name)
		return MechanismFail
	}
	stdErrString := string(stdErrBytes)
	if len(stdErrString) > 0 {
		Log(Warn, fmt.Errorf("Output from stderr of external command \"%s\": %s", fullPath, stdErrString))
		errString = fmt.Sprintf("There was error output while calling external plugin \"%s\", you might want to ask an administrator to check the logs", task.name)
		emit(ScriptPluginStderrOutput)
	}
	if err = cmd.Wait(); err != nil {
		retval = Fail
		success := false
		if exitstatus, ok := err.(*exec.ExitError); ok {
			if status, ok := exitstatus.Sys().(syscall.WaitStatus); ok {
				retval = TaskRetVal(status.ExitStatus())
				if retval == Success {
					success = true
				}
			}
		}
		if !success {
			Log(Error, fmt.Errorf("Waiting on external command \"%s\": %v", fullPath, err))
			errString = fmt.Sprintf("There were errors calling external plugin \"%s\", you might want to ask an administrator to check the logs", task.name)
			emit(ScriptPluginErrExit)
		}
	}
	return retval
}
