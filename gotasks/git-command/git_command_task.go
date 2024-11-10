package gitcommand

import (
	"path/filepath"

	"github.com/lnxjedi/gopherbot/robot"
	gitcommand "github.com/lnxjedi/gopherbot/v2/modules/git-command"
	sshagent "github.com/lnxjedi/gopherbot/v2/modules/ssh-agent"
	sshgithelper "github.com/lnxjedi/gopherbot/v2/modules/ssh-git-helper"
)

func init() {
	robot.RegisterTask("git-command", true, robot.TaskHandler{
		Handler: gitCommandTask,
	})
}

func gitCommandTask(r robot.Robot, args ...string) robot.TaskRetVal {
	if len(args) < 1 {
		r.Log(robot.Error, "no sub-command provided to git-command task")
		return robot.Fail
	}

	subCommand := args[0]

	// Ensure the required parameters are set
	sshAgentHandle := r.GetParameter("_SSH_AGENT_HANDLE")
	if sshAgentHandle == "" {
		r.Log(robot.Error, "SSH agent handle not found; ensure ssh-agent task has been run")
		return robot.Fail
	}

	hostKeysHandle := r.GetParameter("_HOSTKEYS_HANDLE")
	if hostKeysHandle == "" {
		r.Log(robot.Error, "Host keys handle not found; ensure ssh-git-helper task has been run")
		return robot.Fail
	}

	// Obtain the SSH agent
	agentClient, err := sshagent.Get(sshAgentHandle)
	if err != nil {
		r.Log(robot.Error, "failed to get SSH agent: "+err.Error())
		return robot.Fail
	}

	// Obtain the host keys
	hostKeysData, err := sshgithelper.GetHostKeys(hostKeysHandle)
	if err != nil {
		r.Log(robot.Error, "failed to get host keys: "+err.Error())
		return robot.Fail
	}

	// Create HostKeyCallback
	hostKeyCallback, err := gitcommand.CreateHostKeyCallback(hostKeysData)
	if err != nil {
		r.Log(robot.Error, "failed to create host key callback: "+err.Error())
		return robot.Fail
	}

	// Create SSH auth method
	authMethod, err := gitcommand.CreateSSHAuthMethod(agentClient, hostKeyCallback)
	if err != nil {
		r.Log(robot.Error, "failed to create SSH auth method: "+err.Error())
		return robot.Fail
	}

	switch subCommand {
	case "clone":
		if len(args) < 4 {
			r.Log(robot.Error, "not enough arguments for clone command; usage: clone <repoURL> <branch> <directory>")
			return robot.Fail
		}
		repoURL := args[1]
		branch := args[2]
		directory := args[3]

		// Handle "." as an empty branch to clone the default branch
		if branch == "." {
			branch = ""
		}

		// Resolve absolute directory path
		configDir := r.GetParameter("GOPHER_CONFIGDIR")
		absDirectory := directory
		if !filepath.IsAbs(directory) {
			absDirectory = filepath.Join(configDir, directory)
		}

		cloneOpts := gitcommand.CloneOptions{
			RepoURL:   repoURL,
			Branch:    branch,
			Directory: absDirectory,
			Auth:      authMethod,
		}

		if err := gitcommand.Clone(cloneOpts); err != nil {
			r.Log(robot.Error, "git clone failed: "+err.Error())
			return robot.Fail
		}

		r.Log(robot.Info, "git clone successful to directory "+absDirectory)
		return robot.Normal

	default:
		r.Log(robot.Error, "unknown sub-command for git-command task: "+subCommand)
		return robot.Fail
	}
}
