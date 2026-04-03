package bot

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
)

const brainLockKey = "bot:instance-lock"

// instanceLockData captures identifying information about the running robot
// instance stored in the brain lock. This mirrors the data surfaced by the
// "info" admin command so that an operator can identify which instance holds
// the lock.
type instanceLockData struct {
	RobotName        string `json:"robot_name"`
	FullName         string `json:"full_name,omitempty"`
	Hostname         string `json:"hostname"`
	PID              int    `json:"pid"`
	StartMode        string `json:"start_mode"`
	InstallPath      string `json:"install_path"`
	HomePath         string `json:"home_path"`
	ConfigPath       string `json:"config_path"`
	CustomRepository string `json:"custom_repository,omitempty"`
	Version          string `json:"version"`
	Commit           string `json:"commit"`
	StartTime        string `json:"start_time"`
}

// acquireBrainLock checks for an existing instance lock in the brain and
// creates one if absent. It is called during initBot() after brain and
// encryption initialization, and only for non-CLI robot startup.
//
// If a lock is already present, the robot logs identifying information about
// the holding instance and calls Log(Fatal,...) to abort startup.
func acquireBrainLock() {
	_, existing, exists, ret := getDatum(brainLockKey, false)
	if ret != robot.Ok {
		Log(robot.Warn, "Unable to check brain instance lock (ret=%s); proceeding without lock", ret)
		return
	}
	if exists {
		var msg string
		if existing != nil {
			var lock instanceLockData
			if err := json.Unmarshal(*existing, &lock); err == nil {
				msg = fmt.Sprintf(
					"Brain instance lock held by another robot instance:\n"+
						"  Robot:   %s (%s)\n"+
						"  Host:    %s  PID: %d\n"+
						"  Mode:    %s  Started: %s\n"+
						"  Version: Gopherbot %s commit %s\n"+
						"  Home:    %s\n"+
						"  Config:  %s\n"+
						"If this lock is stale, remove it with: gopherbot delete %s",
					lock.RobotName, lock.FullName,
					lock.Hostname, lock.PID,
					lock.StartMode, lock.StartTime,
					lock.Version, lock.Commit,
					lock.HomePath,
					lock.ConfigPath,
					brainLockKey,
				)
			}
		}
		if msg == "" {
			msg = fmt.Sprintf(
				"Brain instance lock exists (unreadable data). "+
					"If stale, remove it with: gopherbot delete %s",
				brainLockKey,
			)
		}
		Log(robot.Fatal, "%s", msg)
		return // unreachable after Fatal, but satisfies the compiler
	}

	customRepo, _ := lookupEnv("GOPHER_CUSTOM_REPOSITORY")
	lock := instanceLockData{
		RobotName:        currentCfg.botinfo.UserName,
		FullName:         currentCfg.botinfo.FullName,
		Hostname:         hostName,
		PID:              os.Getpid(),
		StartMode:        startMode,
		InstallPath:      installPath,
		HomePath:         homePath,
		ConfigPath:       configFull,
		CustomRepository: customRepo,
		Version:          botVersion.Version,
		Commit:           botVersion.Commit,
		StartTime:        time.Now().Format(time.RFC3339),
	}

	data, err := json.Marshal(lock)
	if err != nil {
		Log(robot.Warn, "Unable to marshal brain lock data: %v; proceeding without lock", err)
		return
	}
	if ret := storeDatum(brainLockKey, &data); ret != robot.Ok {
		Log(robot.Warn, "Unable to store brain instance lock (ret=%s); proceeding without lock", ret)
	} else {
		Log(robot.Debug, "Brain instance lock acquired (%s)", brainLockKey)
	}
}

// releaseBrainLock deletes the instance lock from the brain. It is called
// during stop() before brainQuit() so that clean shutdowns and restarts do
// not leave a stale lock behind.
func releaseBrainLock() {
	brain := interfaces.brain
	if brain == nil {
		return
	}
	if err := brain.Delete(brainLockKey); err != nil {
		Log(robot.Warn, "Unable to release brain instance lock: %v", err)
	} else {
		Log(robot.Debug, "Brain instance lock released (%s)", brainLockKey)
	}
}
