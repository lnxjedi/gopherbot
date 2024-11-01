package agent

import (
	"strconv"

	"github.com/lnxjedi/gopherbot/robot"
	sshagent "github.com/lnxjedi/gopherbot/v2/modules/ssh-agent"
)

func init() {
	robot.RegisterTask("ssh-agent", true, robot.TaskHandler{
		Handler: sshAgentTask,
	})
}

func sshAgentTask(r robot.Robot, args ...string) (retval robot.TaskRetVal) {
	if len(args) < 1 {
		r.Log(robot.Error, "no sub-command provided to ssh-agent task")
		return robot.Fail
	}

	subCommand := args[0]

	switch subCommand {
	case "start":
		if len(args) < 2 {
			r.Log(robot.Error, "missing key path for ssh-agent start command")
			return robot.Fail
		}

		keyPath := args[1]
		timeoutMinutes := 7 // Default timeout

		if len(args) > 2 {
			parsedTimeout, err := strconv.Atoi(args[2])
			if err == nil {
				timeoutMinutes = parsedTimeout
			} else {
				r.Log(robot.Warn, "invalid timeout provided, defaulting to 7 minutes")
			}
		}

		// Retrieve passphrase from parameter
		passphrase := r.GetParameter("BOT_SSH_PHRASE")
		if passphrase == "" {
			r.Log(robot.Warn, "empty BOT_SSH_PHRASE while initializing the ssh-agent task")
		}

		// Start the SSH agent
		agentPath, handle, err := sshagent.New(keyPath, passphrase, timeoutMinutes)
		if err != nil {
			r.Log(robot.Error, "failed to start ssh-agent: "+err.Error())
			return robot.Fail
		}

		// Publish SSH_AUTH_SOCK for the pipeline
		r.SetParameter("SSH_AUTH_SOCK", agentPath)
		r.SetParameter("SSH_AGENT_HANDLE", handle)
		r.Log(robot.Info, "SSH agent started successfully with handle "+handle)
		return robot.Success

	case "stop":
		if len(args) < 2 {
			r.Log(robot.Error, "missing handle for ssh-agent stop command")
			return robot.Fail
		}

		handle := args[1]
		err := sshagent.Close(handle)
		if err != nil {
			r.Log(robot.Error, "failed to stop ssh-agent: "+err.Error())
			return robot.Fail
		}

		r.Log(robot.Info, "SSH agent stopped successfully with handle "+handle)
		return robot.Success

	default:
		r.Log(robot.Error, "unknown sub-command for ssh-agent task: "+subCommand)
		return robot.Fail
	}
}
