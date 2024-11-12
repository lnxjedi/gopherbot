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
		homeDir := r.GetParameter("GOPHER_HOME")
		// If the provided directory isn't absolute, assume relative to GOPHER_HOME.
		absDirectory := directory
		if !filepath.IsAbs(directory) {
			absDirectory = filepath.Join(homeDir, directory)
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

	case "pull":
		if len(args) < 2 {
			r.Log(robot.Error, "not enough arguments for pull command; usage: pull <directory>")
			return robot.Fail
		}
		directory := args[1]

		// Resolve absolute directory path
		homeDir := r.GetParameter("GOPHER_HOME")
		// If the provided directory isn't absolute, assume relative to GOPHER_HOME.
		absDirectory := directory
		if !filepath.IsAbs(directory) {
			absDirectory = filepath.Join(homeDir, directory)
		}

		pullOpts := gitcommand.PullOptions{
			Directory: absDirectory,
			Auth:      authMethod,
		}

		if err := gitcommand.Pull(pullOpts); err != nil {
			r.Log(robot.Error, "git pull failed: "+err.Error())
			return robot.Fail
		}

		r.Log(robot.Info, "git pull successful in directory "+absDirectory)

		// Attempt to get the current branch name
		branchName, err := gitcommand.GetCurrentBranch(absDirectory)
		if err != nil {
			r.Log(robot.Warn, "unable to determine current branch name: "+err.Error())
		} else {
			r.SetParameter("GOPHER_CUSTOM_BRANCH", branchName)
			r.Log(robot.Info, "current branch detected: "+branchName)
		}

		return robot.Normal

	default:
		r.Log(robot.Error, "unknown sub-command for git-command task: "+subCommand)
		return robot.Fail
	}
}
