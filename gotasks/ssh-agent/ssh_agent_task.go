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
			r.Log(robot.Error, "empty BOT_SSH_PHRASE while initializing the ssh-agent task")
			return robot.Fail
		}

		// Start the SSH agent with a key path
		agentPath, handle, err := sshagent.New(keyPath, passphrase, timeoutMinutes)
		if err != nil {
			r.Log(robot.Error, "failed to start ssh-agent: "+err.Error())
			return robot.Fail
		}

		// Publish SSH_AUTH_SOCK for the pipeline
		r.SetParameter("SSH_AUTH_SOCK", agentPath)
		r.SetParameter("SSH_AGENT_HANDLE", handle)
		r.Log(robot.Info, "SSH agent started successfully with handle "+handle)
		r.FinalTask("ssh-agent", "stop")
		return

	case "stop":
		// Retrieve the agent handle from the parameter
		handle := r.GetParameter("SSH_AGENT_HANDLE")
		if handle == "" {
			r.Log(robot.Error, "no handle found in SSH_AGENT_HANDLE parameter for stopping the ssh-agent")
			return robot.Fail
		}

		err := sshagent.Close(handle)
		if err != nil {
			r.Log(robot.Error, "failed to stop ssh-agent: "+err.Error())
			return robot.Fail
		}

		r.Log(robot.Info, "SSH agent stopped successfully with handle "+handle)
		return

	case "deploy":
		// Retrieve deployment key
		deployKey := r.GetParameter("GOPHER_DEPLOY_KEY")
		if deployKey == "" {
			r.Log(robot.Error, "empty GOPHER_DEPLOY_KEY while initializing ssh-agent deploy task")
			return robot.Fail
		}

		timeoutMinutes := 7 // Default timeout
		if len(args) > 1 {
			parsedTimeout, err := strconv.Atoi(args[1])
			if err == nil {
				timeoutMinutes = parsedTimeout
			} else {
				r.Log(robot.Warn, "invalid timeout provided, defaulting to 7 minutes")
			}
		}

		// Start the SSH agent with the deployment key
		agentPath, handle, err := sshagent.NewWithDeployKey(deployKey, timeoutMinutes)
		if err != nil {
			r.Log(robot.Error, "failed to start ssh-agent with deployment key: "+err.Error())
			return robot.Fail
		}

		// Publish SSH_AUTH_SOCK for the pipeline
		r.SetParameter("SSH_AUTH_SOCK", agentPath)
		r.SetParameter("SSH_AGENT_HANDLE", handle)
		r.Log(robot.Info, "SSH agent with deployment key started successfully with handle "+handle)
		r.FinalTask("ssh-agent", "stop")
		return

	default:
		r.Log(robot.Error, "unknown sub-command for ssh-agent task: "+subCommand)
		return robot.Fail
	}
}
