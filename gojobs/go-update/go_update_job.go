package update

import (
	"os"
	"path/filepath"

	"github.com/lnxjedi/gopherbot/robot"
)

func init() {
	robot.RegisterJob("go-update", robot.JobHandler{
		Handler: updateHandler,
	})
	robot.RegisterJob("updatecfg", robot.JobHandler{
		Handler: compatHandler,
	})
}

func updateHandler(r robot.Robot, args ...string) robot.TaskRetVal {
	repoDir := r.GetParameter("GOPHER_CONFIGDIR")

	confDir := filepath.Join(repoDir, "conf")
	_, err := os.Stat(confDir)
	if err != nil {
		r.Log(robot.Error, "go-update error locating current config: "+err.Error())
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
		r.Log(robot.Error, "go-update couldn't obtain exclusive access to 'configrepo', queueing ")
		return robot.Normal
	}

	// Begin updateing
	r.Log(robot.Info, "Starting update process for robot configuration repository (git pull)")

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

	r.AddTask("git-command", "pull", repoDir)
	r.AddTask("update-report")
	r.FailTask("fail-report")
	r.AddCommand("builtin-admin", "reload")
	return robot.Normal
}

func compatHandler(r robot.Robot, args ...string) robot.TaskRetVal {
	r.Log(robot.Warn, "Deprecated updatecfg job ran, adding job 'go-update'")
	r.AddJob("go-update")
	return robot.Normal
}
