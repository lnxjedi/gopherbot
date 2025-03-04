package main

import (
	"os"
	"path/filepath"

	"github.com/lnxjedi/gopherbot/robot"
)

/*
switchbranches.go - This job is dynamically loaded, compiled and run by Yaegi (https://github.com/traefik/yaegi).

Switch the robot to a different branch, for quick test/backout development.
*/

func JobHandler(r robot.Robot, args ...string) robot.TaskRetVal {
	if len(args) != 1 {
		r.Log(robot.Error, "wrong number of arguments, expected 1, got %d", len(args))
		return robot.Fail
	}
	repoDir := r.GetParameter("GOPHER_CONFIGDIR")
	branch := args[0]

	confDir := filepath.Join(repoDir, "conf")
	_, err := os.Stat(confDir)
	if err != nil {
		r.Log(robot.Error, "go-switchbranch error locating current config: "+err.Error())
		return robot.Fail
	}

	// We need the URL for loadhostkeys
	cloneURL := r.GetParameter("GOPHER_CUSTOM_REPOSITORY")
	if cloneURL == "" {
		r.Log(robot.Warn, "GOPHER_CUSTOM_REPOSITORY not set")
	}

	// Ensure deploy key exists for SSH access
	deployKey := r.GetParameter("GOPHER_DEPLOY_KEY")
	if deployKey == "" {
		r.Log(robot.Error, "No GOPHER_DEPLOY_KEY provided for SSH access")
		return robot.Fail
	}

	if !r.Exclusive("configrepo", true) {
		r.Log(robot.Error, "go-switchbranch couldn't obtain exclusive access to 'configrepo', queueing ")
		return robot.Normal
	}

	// Start SSH agent using GOPHER_DEPLOY_KEY
	r.AddTask("ssh-agent", "deploy")

	// Host key verification handling
	hostKeys := r.GetParameter("GOPHER_HOST_KEYS")
	if hostKeys != "" {
		r.AddTask("ssh-git-helper", "addhostkeys", hostKeys)
	} else {
		// This could fail if the repository domain isn't supported,
		// and GOPHER_INSECURE_SSH isn't set "true".
		r.AddTask("ssh-git-helper", "loadhostkeys", cloneURL)
	}
	r.AddTask("git-command", "switch", branch, repoDir)
	r.AddCommand("builtin-admin", "reload")

	return robot.Normal
}
