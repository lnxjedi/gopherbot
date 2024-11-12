package hostkeys

/*
Package hostkeys provides the Gopherbot 'ssh-git-helper' task for managing SSH known_hosts files within pipelines.
This task is available only in privileged pipelines and allows for secure SSH host verification handling through known_hosts management.

### Usage

The `ssh-git-helper` task supports the following sub-commands:

- **addhostkeys**: Adds provided host keys to a known_hosts file.
  - **Arguments**:
    - `hostKeys`: Host keys provided as a single encoded string.
  - **Example**:
    ```yaml
    AddTask("ssh-git-helper", "addhostkeys", "<encoded-host-keys>")
    ```

- **loadhostkeys**: Loads host keys for known providers based on the repository URL.
  - **Arguments**:
    - `repoURL`: The repository URL to parse and load host keys for.
  - **Example**:
    ```yaml
    AddTask("ssh-git-helper", "loadhostkeys", "git@github.com:user/repo.git")
    ```

- **scan**: Scans a host to retrieve its host key and adds it to a known_hosts file.
  - **Arguments**:
    - `host`: The hostname to scan.
  - **Example**:
    ```yaml
    AddTask("ssh-git-helper", "scan", "example.com")
    ```

- **publishenv**: Publishes environment variables needed for SSH and Git commands to recognize the created known_hosts file.
  - **Example**:
    ```yaml
    AddTask("ssh-git-helper", "publishenv")
    ```
  - **Environment Variables Set**:
    - `GIT_SSH_COMMAND`: Configures Git to use SSH with the options defined by `SSH_OPTIONS`.
    - `SSH_OPTIONS`: Contains SSH options, including the path to the custom known_hosts file, for direct SSH commands in the pipeline.

- **delete**: Deletes the known_hosts file associated with the handle.
  - **Example**:
    ```yaml
    AddTask("ssh-git-helper", "delete")
    ```

### Task Return Values

- **Normal** (0): Indicates successful execution within a pipeline context.
- **Fail**: Used for any errors that prevent successful execution.

### Final Task

The `addhostkeys`, `loadhostkeys`, and `scan` sub-commands automatically add a `FinalTask("ssh-git-helper", "delete")` to ensure the known_hosts file is cleaned up after the pipeline completes.

### Prerequisites

- The task is only available in privileged pipelines.
- Ensure the `GOPHER_HOSTKEYS` parameter is securely set when using `addhostkeys`.
*/

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lnxjedi/gopherbot/robot"
	sshgithelper "github.com/lnxjedi/gopherbot/v2/modules/ssh-git-helper"
)

func init() {
	robot.RegisterTask("ssh-git-helper", true, robot.TaskHandler{
		Handler: sshHostKeysTask,
	})
}

func sshHostKeysTask(r robot.Robot, args ...string) (retval robot.TaskRetVal) {
	if len(args) < 1 {
		r.Log(robot.Error, "no sub-command provided to ssh-git-helper task")
		return robot.Fail
	}

	subCommand := args[0]

	switch subCommand {
	case "addhostkeys", "loadhostkeys", "scan":
		// Handle host keys management sub-commands
		if len(args) < 2 {
			r.Log(robot.Error, "missing required argument for "+subCommand+" command")
			return robot.Fail
		}
		return handleHostKeysSubCommand(r, subCommand, args[1])

	case "publishenv":
		// Publish environment variables for SSH and Git
		return publishEnvVars(r)

	case "delete":
		// Handle delete known_hosts file
		return handleDelete(r)

	default:
		r.Log(robot.Error, "unknown sub-command for ssh-git-helper task: "+subCommand)
		return robot.Fail
	}
}

func handleHostKeysSubCommand(r robot.Robot, subCommand string, arg string) (retval robot.TaskRetVal) {

	var (
		handle string
		err    error
	)

	switch subCommand {
	case "addhostkeys":
		hostKeysEncoded := arg
		hostKeys := decodeHostKeys(hostKeysEncoded)
		handle, err = sshgithelper.AddHostKeys(hostKeys)
	case "loadhostkeys":
		repoURL := arg
		handle, err = sshgithelper.LoadHostKeys(repoURL)
		if err != nil {
			insecure_ok := r.GetParameter("GOPHER_INSECURE_SSH") == "true"
			if !insecure_ok {
				r.Log(robot.Error, "unable to detect and load ssh hostkeys, and GOPHER_INSECURE_SSH not 'true', giving up")
				return robot.Fail
			}
			host, err := sshgithelper.ParseHostFromRepoURL(repoURL)
			if err != nil {
				r.Log(robot.Error, "unable to parse the host name from repo URL, giving up: "+err.Error())
			}
			handle, err = sshgithelper.ScanHost(host)
			if err != nil {
				r.Log(robot.Error, "failed to scan hostkeys for host "+host)
			}
		}
	case "scan":
		host := arg
		handle, err = sshgithelper.ScanHost(host)
	}

	if err != nil {
		r.Log(robot.Error, "failed to process "+subCommand+": "+err.Error())
		return robot.Fail
	}

	r.SetParameter("_HOSTKEYS_HANDLE", handle)
	r.Log(robot.Info, "SSH known_hosts file created successfully with handle "+handle)
	r.FinalTask("ssh-git-helper", "delete")
	return robot.Normal
}

func publishEnvVars(r robot.Robot) robot.TaskRetVal {
	knownHostsHandle := r.GetParameter("_HOSTKEYS_HANDLE")
	knownHostsPath, err := sshgithelper.GetKnownHostsPath(knownHostsHandle)
	if err != nil {
		r.Log(robot.Error, "failed to obtain known_hosts path: "+err.Error())
		return robot.Fail
	}

	sshOptions, err := buildSSHOptions(r, knownHostsPath)
	if err != nil {
		r.Log(robot.Error, "failed to build SSH options: "+err.Error())
		return robot.Fail
	}

	r.SetParameter("SSH_OPTIONS", sshOptions)
	r.SetParameter("GIT_SSH_COMMAND", "ssh "+sshOptions)
	r.Log(robot.Info, "Environment variables published successfully for SSH and Git")
	return robot.Normal
}

func handleDelete(r robot.Robot) robot.TaskRetVal {
	handle := r.GetParameter("_HOSTKEYS_HANDLE")
	if handle == "" {
		r.Log(robot.Error, "no handle found in _HOSTKEYS_HANDLE parameter for deleting the known_hosts file")
		return robot.Fail
	}

	err := sshgithelper.Delete(handle)
	if err != nil {
		r.Log(robot.Error, "failed to delete known_hosts file: "+err.Error())
		return robot.Fail
	}

	r.Log(robot.Info, "SSH known_hosts file deleted successfully with handle "+handle)
	return robot.Normal
}

// Helper function to decode host keys from encoded string
func decodeHostKeys(encoded string) string {
	decoded := strings.ReplaceAll(encoded, "_", " ")
	decoded = strings.ReplaceAll(decoded, ":", "\n")
	return decoded
}

// Helper function to build SSH_OPTIONS
func buildSSHOptions(r robot.Robot, knownHostsPath string) (string, error) {
	sshOptions := "-o PasswordAuthentication=no"

	configDir := r.GetParameter("GOPHER_CONFIGDIR")
	sshConfigPath := filepath.Join(configDir, "ssh", "config")
	if _, err := os.Stat(sshConfigPath); err == nil {
		err = os.Chmod(sshConfigPath, 0600)
		if err != nil {
			return "", fmt.Errorf("failed to set permissions on ssh config: %w", err)
		}
		sshOptions += fmt.Sprintf(" -F %s", sshConfigPath)
	}

	sshOptions += fmt.Sprintf(" -o UserKnownHostsFile=%s", knownHostsPath)
	return sshOptions, nil
}
