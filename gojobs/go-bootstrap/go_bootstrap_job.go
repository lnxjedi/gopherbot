package bootstrap

import (
	"os"
	"path/filepath"

	"github.com/lnxjedi/gopherbot/robot"
)

func init() {
	robot.RegisterJob("go-bootstrap", robot.JobHandler{
		Handler: bootstrapHandler,
	})
}

func bootstrapHandler(r robot.Robot, args ...string) robot.TaskRetVal {
	repoDir := r.GetParameter("GOPHER_CONFIGDIR")

	confDir := filepath.Join(repoDir, "conf")
	info, err := os.Stat(confDir)
	if err == nil && info.IsDir() {
		r.Log(robot.Debug, "go-bootstrap found existing config directory, exiting")
		// Configuration directory exists, no further action needed
		return robot.Normal
	}

	// Proceed with bootstrapping
	cloneURL := r.GetParameter("GOPHER_CUSTOM_REPOSITORY")
	if cloneURL == "" {
		r.Log(robot.Warn, "GOPHER_CUSTOM_REPOSITORY not set, skipping bootstrapping")
		return robot.Normal
	}

	// Ensure deploy key exists for SSH access
	deployKey := r.GetParameter("GOPHER_DEPLOY_KEY")
	if deployKey == "" {
		r.Log(robot.Fatal, "No GOPHER_DEPLOY_KEY provided for SSH access")
		return robot.Fail
	}

	if !r.Exclusive("configrepo", false) {
		// Hard to imagine when this might happen, but we must protect the configrepo
		// from access by parallel goroutines.
		r.Log(robot.Error, "go-bootstrap couldn't obtain exclusive access to 'configrepo'")
		return robot.Fail
	}

	// Begin bootstrapping
	r.Log(robot.Info, "Starting built-in go-bootstrap job for repository: "+cloneURL)

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

	// Remove any temporary binary encryption key created by unconfigured start-up.
	tmpKeyName := "binary-encrypted-key"
	deployEnv := r.GetParameter("GOPHER_ENVIRONMENT")
	if len(deployEnv) > 0 {
		tmpKeyName = tmpKeyName + "." + deployEnv
	}
	tmpKeyPath := filepath.Join(repoDir, tmpKeyName)
	if err := os.Remove(tmpKeyPath); err != nil {
		if !os.IsNotExist(err) {
			r.Log(robot.Fatal, "failed to remove temporary key: "+err.Error())
			return robot.Fail
		}
		r.Log(robot.Warn, "failed to remove temporary key - file not found: %s", tmpKeyPath)
	} else {
		r.Log(robot.Debug, "removed temporary key: "+tmpKeyPath)
	}

	// Clone the repository to the config directory using ssh-agent credentials
	// and known_hosts for server validation.
	cloneBranch := r.GetParameter("GOPHER_CUSTOM_BRANCH")
	r.AddTask("git-command", "clone", cloneURL, cloneBranch, repoDir)

	// Restart robot so it can start configured.
	r.AddTask("restart-robot")

	return robot.Normal
}
