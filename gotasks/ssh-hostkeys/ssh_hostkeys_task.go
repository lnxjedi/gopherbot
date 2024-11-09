package hostkeys

/*
Package hostkeys provides the Gopherbot 'ssh-hostkeys' task for managing SSH known_hosts files within pipelines.
This task is available only in privileged pipelines and allows for secure SSH host verification handling through known_hosts management.

### Usage

The `ssh-hostkeys` task supports the following sub-commands:

- **addhostkeys**: Adds provided host keys to a known_hosts file and sets SSH_OPTIONS for the pipeline.
  - **Arguments**:
    - `hostKeys`: Host keys provided as a single encoded string.
  - **Example**:
    ```yaml
    AddTask("ssh-hostkeys", "addhostkeys", "<encoded-host-keys>")
    ```

- **loadhostkeys**: Loads host keys for known providers based on the repository URL.
  - **Arguments**:
    - `repoURL`: The repository URL to parse and load host keys for.
  - **Example**:
    ```yaml
    AddTask("ssh-hostkeys", "loadhostkeys", "git@github.com:user/repo.git")
    ```

- **scan**: Scans a host to retrieve its host key and sets SSH_OPTIONS for the pipeline.
  - **Arguments**:
    - `host`: The hostname to scan.
  - **Example**:
    ```yaml
    AddTask("ssh-hostkeys", "scan", "example.com")
    ```

- **delete**: Deletes the known_hosts file associated with the handle.
  - **Example**:
    ```yaml
    AddTask("ssh-hostkeys", "delete")
    ```

### Task Return Values

- **Normal** (0): Indicates successful execution within a pipeline context.
- **Fail**: Used for any errors that prevent successful execution.

### Final Task

The `addhostkeys`, `loadhostkeys`, and `scan` sub-commands automatically add a `FinalTask("ssh-hostkeys", "delete")` to ensure the known_hosts file is cleaned up after the pipeline completes.

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
	sshhostkeys "github.com/lnxjedi/gopherbot/v2/modules/ssh-hostkeys"
)

func init() {
	robot.RegisterTask("ssh-hostkeys", true, robot.TaskHandler{
		Handler: sshHostKeysTask,
	})
}

func sshHostKeysTask(r robot.Robot, args ...string) (retval robot.TaskRetVal) {
	if len(args) < 1 {
		r.Log(robot.Error, "no sub-command provided to ssh-hostkeys task")
		return robot.Fail
	}

	subCommand := args[0]

	switch subCommand {
	case "addhostkeys":
		if len(args) < 2 {
			r.Log(robot.Error, "missing host keys for ssh-hostkeys addhostkeys command")
			return robot.Fail
		}

		hostKeysEncoded := args[1]

		// Decode hostKeysEncoded (spaces replaced by underscores, colons replaced by newlines)
		hostKeys := decodeHostKeys(hostKeysEncoded)

		knownHostsPath, handle, err := sshhostkeys.AddHostKeys(hostKeys)
		if err != nil {
			r.Log(robot.Error, "failed to add host keys: "+err.Error())
			return robot.Fail
		}

		// Set SSH_OPTIONS and GIT_SSH_COMMAND
		sshOptions, err := buildSSHOptions(r, knownHostsPath)
		if err != nil {
			r.Log(robot.Error, "failed to build SSH options: "+err.Error())
			return robot.Fail
		}

		r.SetParameter("SSH_OPTIONS", sshOptions)
		r.SetParameter("GIT_SSH_COMMAND", "ssh "+sshOptions)
		r.SetParameter("HOSTKEYS_HANDLE", handle)
		r.Log(robot.Info, "SSH known_hosts file created successfully with handle "+handle)
		r.FinalTask("ssh-hostkeys", "delete")
		return robot.Normal

	case "loadhostkeys":
		if len(args) < 2 {
			r.Log(robot.Error, "missing repository URL for ssh-hostkeys loadhostkeys command")
			return robot.Fail
		}

		repoURL := args[1]

		insecureCloneStr := r.GetParameter("GOPHER_INSECURE_CLONE")
		insecureClone := insecureCloneStr == "true"

		knownHostsPath, handle, err := sshhostkeys.LoadHostKeys(repoURL)
		if err != nil {
			r.Log(robot.Warn, "failed to load host keys automatically: "+err.Error())
			if !insecureClone {
				r.Log(robot.Error, "host not recognized and GOPHER_INSECURE_CLONE is not set to 'true'; cannot proceed")
				return robot.Fail
			}
			// Fallback to scanning the host
			host, parseErr := sshhostkeys.ParseHostFromRepoURL(repoURL)
			if parseErr != nil {
				r.Log(robot.Error, "failed to parse repository URL: "+parseErr.Error())
				return robot.Fail
			}
			r.Log(robot.Warn, "GOPHER_INSECURE_CLONE='true' - proceeding to scan host key for "+host+". This may expose the connection to security risks.")
			knownHostsPath, handle, err = sshhostkeys.ScanHost(host)
			if err != nil {
				r.Log(robot.Error, "failed to scan host key: "+err.Error())
				return robot.Fail
			}
		}

		// Set SSH_OPTIONS and GIT_SSH_COMMAND
		sshOptions, err := buildSSHOptions(r, knownHostsPath)
		if err != nil {
			r.Log(robot.Error, "failed to build SSH options: "+err.Error())
			return robot.Fail
		}

		r.SetParameter("SSH_OPTIONS", sshOptions)
		r.SetParameter("GIT_SSH_COMMAND", "ssh "+sshOptions)
		r.SetParameter("HOSTKEYS_HANDLE", handle)
		r.Log(robot.Info, "SSH known_hosts file created successfully with handle "+handle)
		r.FinalTask("ssh-hostkeys", "delete")
		return robot.Normal

	case "scan":
		if len(args) < 2 {
			r.Log(robot.Error, "missing host for ssh-hostkeys scan command")
			return robot.Fail
		}

		host := args[1]

		knownHostsPath, handle, err := sshhostkeys.ScanHost(host)
		if err != nil {
			r.Log(robot.Error, "failed to scan host key: "+err.Error())
			return robot.Fail
		}

		// Set SSH_OPTIONS and GIT_SSH_COMMAND
		sshOptions, err := buildSSHOptions(r, knownHostsPath)
		if err != nil {
			r.Log(robot.Error, "failed to build SSH options: "+err.Error())
			return robot.Fail
		}

		r.SetParameter("SSH_OPTIONS", sshOptions)
		r.SetParameter("GIT_SSH_COMMAND", "ssh "+sshOptions)
		r.SetParameter("HOSTKEYS_HANDLE", handle)
		r.Log(robot.Info, "SSH known_hosts file created successfully with handle "+handle)
		r.FinalTask("ssh-hostkeys", "delete")
		return robot.Normal

	case "delete":
		// Retrieve the hostkeys handle from the parameter
		handle := r.GetParameter("HOSTKEYS_HANDLE")
		if handle == "" {
			r.Log(robot.Error, "no handle found in HOSTKEYS_HANDLE parameter for deleting the known_hosts file")
			return robot.Fail
		}

		err := sshhostkeys.Delete(handle)
		if err != nil {
			r.Log(robot.Error, "failed to delete known_hosts file: "+err.Error())
			return robot.Fail
		}

		r.Log(robot.Info, "SSH known_hosts file deleted successfully with handle "+handle)
		return robot.Normal

	default:
		r.Log(robot.Error, "unknown sub-command for ssh-hostkeys task: "+subCommand)
		return robot.Fail
	}
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

	// Check for $GOPHER_CONFIGDIR/ssh/config
	configDir := r.GetParameter("GOPHER_CONFIGDIR")
	sshConfigPath := filepath.Join(configDir, "ssh", "config")
	if _, err := os.Stat(sshConfigPath); err == nil {
		// File exists
		err = os.Chmod(sshConfigPath, 0600)
		if err != nil {
			return "", fmt.Errorf("failed to set permissions on ssh config: %w", err)
		}
		sshOptions += fmt.Sprintf(" -F %s", sshConfigPath)
	}

	sshOptions += fmt.Sprintf(" -o UserKnownHostsFile=%s", knownHostsPath)

	return sshOptions, nil
}
