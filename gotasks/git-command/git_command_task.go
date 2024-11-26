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
	hostKeysPath, err := sshgithelper.GetHostKeysPath(hostKeysHandle)
	if err != nil {
		r.Log(robot.Error, "failed to get host keys: "+err.Error())
		return robot.Fail
	}

	// Create HostKeyCallback
	hostKeyCallback, err := gitcommand.CreateHostKeyCallback(hostKeysPath)
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

		if err := gitcommand.Clone(r, cloneOpts); err != nil {
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

		if err := gitcommand.Pull(r, pullOpts); err != nil {
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

	case "switch":
		if len(args) < 3 {
			r.Log(robot.Error, "not enough arguments for switch command; usage: switch <branch> <directory>")
			return robot.Fail
		}
		branch := args[1]
		directory := args[2]

		// Resolve absolute directory path
		homeDir := r.GetParameter("GOPHER_HOME")
		// If the provided directory isn't absolute, assume relative to GOPHER_HOME.
		absDirectory := directory
		if !filepath.IsAbs(directory) {
			absDirectory = filepath.Join(homeDir, directory)
		}

		checkoutOpts := gitcommand.SwitchBranchOptions{
			Directory: absDirectory,
			Branch:    branch,
			Auth:      authMethod,
		}

		if err := gitcommand.SwitchBranch(r, checkoutOpts); err != nil {
			r.Log(robot.Error, "switch branch failed: "+err.Error())
			return robot.Fail
		}

		r.Log(robot.Info, "git switch to branch "+branch+" successful in directory "+absDirectory)
		return robot.Normal

	case "push":
		if len(args) < 4 {
			r.Log(robot.Error, "not enough arguments for push command; usage: push <branch-if-no-upstream> <commit_msg> <directory>")
			return robot.Fail
		}
		branchIfNoUpstream := args[1]
		commitMsg := args[2]
		directory := args[3]

		// Resolve absolute directory path
		homeDir := r.GetParameter("GOPHER_HOME")
		// If the provided directory isn't absolute, assume relative to GOPHER_HOME.
		absDirectory := directory
		if !filepath.IsAbs(directory) {
			absDirectory = filepath.Join(homeDir, directory)
		}

		pushOpts := gitcommand.PushOptions{
			Directory:          absDirectory,
			BranchIfNoUpstream: branchIfNoUpstream,
			CommitMsg:          commitMsg,
			Auth:               authMethod,
		}

		err := gitcommand.Push(pushOpts)
		if err != nil {
			if err.Error() == "no changes to commit" {
				// Non-fatal warning
				r.Log(robot.Warn, "git push skipped: no changes to commit")
				return robot.Normal
			}
			r.Log(robot.Error, "git push failed: "+err.Error())
			return robot.Fail
		}

		r.Log(robot.Info, "git push successful in directory "+absDirectory)
		return robot.Normal

	default:
		r.Log(robot.Error, "unknown sub-command for git-command task: "+subCommand)
		return robot.Fail
	}
}
