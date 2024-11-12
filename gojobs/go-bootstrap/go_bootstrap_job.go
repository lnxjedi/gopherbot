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
	// Check if the configuration directory has been set up
	repoDir := r.GetParameter("GOPHER_CONFIGDIR")

	confDir := filepath.Join(repoDir, "conf")
	info, err := os.Stat(confDir)
	if err == nil && info.IsDir() {
		// If "conf" exists and is a directory, check for ".restore" to trigger restore job
		restoreFile := filepath.Join(repoDir, ".restore")
		if _, err := os.Stat(restoreFile); err == nil {
			r.AddJob("restore")
			return robot.Normal
		}
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
		r.Log(robot.Error, "No GOPHER_DEPLOY_KEY provided for SSH access")
		return robot.Fail
	}

	if !r.Exclusive("configrepo", false) {
		// Hard to imagine when this might happen, but we must protect the configrepo
		// from access by parallel goroutines.
		r.Log(robot.Error, "go-bootstrap couldn't obtain exclusive access to 'configrepo'")
		return robot.Fail
	}

	// Begin bootstrapping
	r.Log(robot.Info, "Starting bootstrap process for repository: "+cloneURL)

	// Start SSH agent using GOPHER_DEPLOY_KEY
	r.AddTask("ssh-agent", "deploy")

	// Host key verification handling
	hostKeys := r.GetParameter("GOPHER_HOST_KEYS")
	if hostKeys != "" {
		r.AddTask("ssh-git-helper", "addhostkeys", hostKeys)
	} else {
		// This could fail if the repository domain isn't supported,
		// and GOPHER_INSECURE_CLONE isn't set "true".
		r.AddTask("ssh-git-helper", "loadhostkeys", cloneURL)
	}

	// Create the .restore file in the current working directory to indicate restore
	// of file-based memories is needed.
	if err := os.WriteFile(".restore", []byte{}, 0644); err != nil {
		r.Log(robot.Error, "failed to create .restore file: "+err.Error())
		return robot.Fail
	}

	// Remove any temporary binary encryption key created by unconfigured start-up.
	tmpKeyName := "binary-encrypted-key"
	deployEnv := r.GetParameter("GOPHER_ENVIRONMENT")
	if deployEnv != "production" {
		tmpKeyName = tmpKeyName + "." + deployEnv
	}
	tmpKeyPath := filepath.Join(repoDir, tmpKeyName)
	if err := os.Remove(tmpKeyPath); err != nil && !os.IsNotExist(err) {
		r.Log(robot.Error, "failed to remove temporary key: "+err.Error())
		return robot.Fail
	}
	r.Log(robot.Debug, "removed temporary key: "+tmpKeyPath)

	// Clone the repository to the config directory using ssh-agent credentials
	// and known_hosts for server validation.
	cloneBranch := r.GetParameter("GOPHER_CUSTOM_BRANCH")
	r.AddTask("git-command", "clone", cloneURL, cloneBranch, repoDir)

	// Restart robot so it can start configured.
	r.AddTask("restart-robot")

	return robot.Normal
}
