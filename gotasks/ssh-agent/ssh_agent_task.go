package agent

/*
Package agent provides the Gopherbot 'ssh-agent' task for managing SSH agent processes within pipelines.
This task is available only in privileged pipelines and allows for secure SSH authentication
handling through agent management.

### Usage

The `ssh-agent` task supports the following sub-commands:

- **start**: Starts an SSH agent, loads a key from a specified path, and sets the `SSH_AUTH_SOCK` environment variable for the pipeline.
  - **Arguments**:
    - `keyPath`: Path to the private key file.
    - `timeoutMinutes` (optional): Number of minutes before the agent auto-expires. Defaults to 7 if not provided or invalid.
  - **Paramters**:
    - `BOT_SSH_PHRASE`: The passphrase for decrypting the key needs to be set here.
  - **Example**:
    ```yaml
    AddTask("ssh-agent", "start", "/path/to/private_key", "10")
    ```

- **stop**: Stops the SSH agent using the handle stored in the `_SSH_AGENT_HANDLE` parameter. This ensures the agent is cleaned up after use. **NOTE:** This is automatically added as a `FinalTask` by "start" and "deploy", and so isn't normally needed for user add-ons.
  - **Example**:
    ```yaml
    AddTask("ssh-agent", "stop")
    ```

- **deploy**: Starts an SSH agent using a deployment key stored in the `GOPHER_DEPLOY_KEY` parameter, processes the key for loading, and sets the `SSH_AUTH_SOCK` environment variable.
  - **Arguments**:
    - `timeoutMinutes` (optional): Number of minutes before the agent auto-expires. Defaults to 7 if not provided or invalid.
  - **Paramters**:
    - `GOPHER_DEPLOY_KEY`: The unencrypted encoded deployment key obtained during initial configuration.
  - **Example**:
    ```yaml
    AddTask("ssh-agent", "deploy", "15")
    ```

### Task Return Values

- **Normal** (0): Indicates successful execution within a pipeline context.
- **Fail**: Used for any errors that prevent successful execution.
- **Success** (7): Special return value for Authorization plugins, not applicable to regular pipeline tasks.

### Final Task

The `start` and `deploy` sub-commands automatically add a `FinalTask("ssh-agent", "stop")` to ensure the agent is stopped and cleaned up after the pipeline completes.

### Prerequisites

- The task is only available in privileged pipelines.
- Ensure the `BOT_SSH_PHRASE` and `GOPHER_DEPLOY_KEY` parameters are securely set when using `start` and `deploy`.

*/

import (
	"path/filepath"
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
		configDir := r.GetParameter("GOPHER_CONFIGDIR")
		fullKeyPath := filepath.Join(configDir, keyPath)
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
		agentPath, handle, keyID, err := sshagent.New(fullKeyPath, passphrase, timeoutMinutes)
		if err != nil {
			r.Log(robot.Error, "failed to start ssh-agent: "+err.Error())
			return robot.Fail
		}

		// Publish SSH_AUTH_SOCK for the pipeline
		r.SetParameter("SSH_AUTH_SOCK", agentPath)
		r.SetParameter("_SSH_AGENT_HANDLE", handle)
		r.Log(robot.Info, "SSH agent started successfully with key '%s' and handle "+handle, keyID)
		r.FinalTask("ssh-agent", "stop")
		return robot.Normal

	case "stop":
		// Retrieve the agent handle from the parameter
		handle := r.GetParameter("_SSH_AGENT_HANDLE")
		if handle == "" {
			r.Log(robot.Error, "no handle found in _SSH_AGENT_HANDLE parameter for stopping the ssh-agent")
			return robot.Fail
		}

		err := sshagent.Close(handle)
		if err != nil {
			r.Log(robot.Error, "failed to stop ssh-agent: "+err.Error())
			return robot.Fail
		}

		r.Log(robot.Info, "SSH agent stopped successfully with handle "+handle)
		return robot.Normal

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
		agentPath, handle, keyID, err := sshagent.NewWithDeployKey(deployKey, timeoutMinutes)
		if err != nil {
			r.Log(robot.Error, "failed to start ssh-agent with deployment key: "+err.Error())
			return robot.Fail
		}

		// Publish SSH_AUTH_SOCK for the pipeline
		r.SetParameter("SSH_AUTH_SOCK", agentPath)
		r.SetParameter("_SSH_AGENT_HANDLE", handle)
		r.Log(robot.Info, "SSH agent with deployment key started successfully with key '%s' and handle "+handle, keyID)
		r.FinalTask("ssh-agent", "stop")
		return robot.Normal

	default:
		r.Log(robot.Error, "unknown sub-command for ssh-agent task: "+subCommand)
		return robot.Fail
	}
}
