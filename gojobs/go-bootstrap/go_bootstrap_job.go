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
	if repoDir == "" {
		r.Log(robot.Error, "GOPHER_CONFIGDIR is not set")
		return robot.Fail
	}

	confDir := filepath.Join(repoDir, "conf")
	info, err := os.Stat(confDir)
	if err == nil && info.IsDir() {
		// If "conf" exists and is a directory, check for ".restore" to trigger restore job
		restoreFile := filepath.Join(repoDir, ".restore")
		if _, err := os.Stat(restoreFile); err == nil {
			r.AddJob("restore")
			return robot.Normal
		}
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

	// Begin bootstrapping
	r.Log(robot.Info, "Starting bootstrap process for repository: "+cloneURL)
	r.SetParameter("BOOTSTRAP", "true")
	r.SetParameter("GOPHER_DEPLOY_KEY", deployKey)

	// Start SSH agent using deploy key
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

	// Create the .restore file to indicate restore is needed
	restoreFile := filepath.Join(repoDir, ".restore")
	if err := os.WriteFile(restoreFile, []byte{}, 0644); err != nil {
		r.Log(robot.Error, "failed to create .restore file: "+err.Error())
		return robot.Fail
	}

	// Clone the repository to the config directory
	cloneBranch := r.GetParameter("GOPHER_CUSTOM_BRANCH")
	r.AddTask("git-command", "clone", cloneURL, cloneBranch, repoDir)

	// Clean up temporary deployment key if not in production
	deployEnv := r.GetParameter("GOPHER_ENVIRONMENT")
	if deployEnv != "production" {
		tmpKeyPath := filepath.Join(repoDir, "binary-encrypted-key."+deployEnv)
		if err := os.Remove(tmpKeyPath); err != nil && !os.IsNotExist(err) {
			r.Log(robot.Error, "failed to remove temporary key: "+err.Error())
			return robot.Fail
		}
	}

	// Restart robot to apply changes
	r.AddTask("restart-robot")

	return robot.Normal
}
