// +build darwin dragonfly freebsd linux netbsd openbsd

package bot

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
)

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
	// cmd := exec.Command(fullPath, channel, user, matcher.Command, matches[0][1:]...)
	cfg, err := exec.Command(fullPath, "configure").Output()
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
		if plugRunningCounter >= 0 {
			plugRunningWaitGroup.Done()
		}
		shutdownMutex.Unlock()
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
		externalArgs := make([]string, 0, 4+len(args))
		externalArgs = append(externalArgs, command)
		externalArgs = append(externalArgs, args...)
		Log(Debug, fmt.Sprintf("Calling \"%s\" with args: %q", fullPath, externalArgs))
		// cmd := exec.Command(fullPath, channel, user, matcher.Command, matches[0][1:]...)
		cmd := exec.Command(fullPath, externalArgs...)
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
