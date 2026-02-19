package bot

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
	gitcommand "github.com/lnxjedi/gopherbot/v2/modules/git-command"
)

type runtimeGitSnapshot struct {
	RepoDir       string
	CurrentBranch string
	DefaultBranch string
	StartupBranch string
	LastError     string
	LastSync      time.Time
}

var runtimeGit = struct {
	sync.RWMutex
	repoDir       string
	currentBranch string
	defaultBranch string
	startupBranch string
	lastError     string
	lastSync      time.Time
}{}

func snapshotRuntimeGitLocked() runtimeGitSnapshot {
	return runtimeGitSnapshot{
		RepoDir:       runtimeGit.repoDir,
		CurrentBranch: runtimeGit.currentBranch,
		DefaultBranch: runtimeGit.defaultBranch,
		StartupBranch: runtimeGit.startupBranch,
		LastError:     runtimeGit.lastError,
		LastSync:      runtimeGit.lastSync,
	}
}

func getRuntimeGitSnapshot() runtimeGitSnapshot {
	runtimeGit.RLock()
	s := snapshotRuntimeGitLocked()
	runtimeGit.RUnlock()
	return s
}

func normalizeGitRepoDir(repoDir string) (string, error) {
	repoDir = strings.TrimSpace(repoDir)
	if repoDir == "" {
		return "", fmt.Errorf("empty repository path")
	}
	if filepath.IsAbs(repoDir) {
		return filepath.Clean(repoDir), nil
	}
	if homePath != "" {
		return filepath.Clean(filepath.Join(homePath, repoDir)), nil
	}
	absDir, err := filepath.Abs(repoDir)
	if err != nil {
		return "", err
	}
	return absDir, nil
}

func syncRuntimeGitState(repoDir string, captureStartup bool) (runtimeGitSnapshot, error) {
	normalizedRepoDir, err := normalizeGitRepoDir(repoDir)
	if err != nil {
		return getRuntimeGitSnapshot(), err
	}

	currentBranch, branchErr := gitcommand.GetCurrentBranch(normalizedRepoDir)
	currentBranch = strings.TrimSpace(currentBranch)
	if branchErr == nil && currentBranch == "" {
		branchErr = fmt.Errorf("branch lookup returned empty branch name")
	}
	defaultBranch, defaultErr := gitcommand.GetLocalDefaultBranch(normalizedRepoDir)
	defaultBranch = strings.TrimSpace(defaultBranch)
	if defaultErr != nil {
		defaultBranch = ""
	}

	runtimeGit.Lock()
	runtimeGit.repoDir = normalizedRepoDir
	runtimeGit.lastSync = time.Now()
	if branchErr != nil {
		runtimeGit.currentBranch = ""
		runtimeGit.lastError = branchErr.Error()
		s := snapshotRuntimeGitLocked()
		runtimeGit.Unlock()
		return s, branchErr
	}

	runtimeGit.currentBranch = currentBranch
	runtimeGit.lastError = ""
	if captureStartup || runtimeGit.startupBranch == "" {
		runtimeGit.startupBranch = currentBranch
	}
	// Default branch is detected from local git metadata (refs/remotes/origin/HEAD).
	// If local metadata is missing, this remains unknown.
	runtimeGit.defaultBranch = defaultBranch
	s := snapshotRuntimeGitLocked()
	runtimeGit.Unlock()
	return s, nil
}

func persistRuntimeGitSnapshotToEnv(snapshot runtimeGitSnapshot) {
	_ = setEnv("GOPHER_CUSTOM_BRANCH", snapshot.CurrentBranch)
	_ = setEnv("GOPHER_CUSTOM_DEFAULT_BRANCH", snapshot.DefaultBranch)
	_ = setEnv("GOPHER_CUSTOM_STARTUP_BRANCH", snapshot.StartupBranch)
}

func refreshRuntimeGitStateFromConfig(captureStartup bool) (runtimeGitSnapshot, error) {
	return syncRuntimeGitState(configFull, captureStartup)
}

func initializeRuntimeGitState() {
	snapshot, err := refreshRuntimeGitStateFromConfig(true)
	if err != nil {
		Log(robot.Debug, "Skipping startup git-state capture for config dir '%s': %v", configFull, err)
		return
	}
	persistRuntimeGitSnapshotToEnv(snapshot)
	if snapshot.CurrentBranch == "" {
		return
	}
	if snapshot.DefaultBranch != "" && snapshot.CurrentBranch == snapshot.DefaultBranch {
		Log(robot.Info, "Startup git branch detected: %s (default branch)", snapshot.CurrentBranch)
		return
	}
	if snapshot.DefaultBranch != "" {
		Log(robot.Info, "Startup git branch detected: %s (default branch: %s)", snapshot.CurrentBranch, snapshot.DefaultBranch)
		return
	}
	Log(robot.Info, "Startup git branch detected: %s", snapshot.CurrentBranch)
}

func runtimeGitSummaryLine(snapshot runtimeGitSnapshot) string {
	if snapshot.CurrentBranch == "" {
		if snapshot.LastError != "" {
			return "Git branch status is unavailable (run git-info for details)"
		}
		return "Git branch status is unavailable"
	}
	if snapshot.DefaultBranch == "" {
		return fmt.Sprintf("Git branch status: %s (default branch unknown)", snapshot.CurrentBranch)
	}
	if snapshot.CurrentBranch == snapshot.DefaultBranch {
		return fmt.Sprintf("Git branch status: %s (default branch)", snapshot.CurrentBranch)
	}
	return fmt.Sprintf("Git branch status: %s (non-default; default branch: %s)", snapshot.CurrentBranch, snapshot.DefaultBranch)
}

func runtimeGitDetailLines(snapshot runtimeGitSnapshot) []string {
	lines := []string{"Git repository status:"}
	if snapshot.RepoDir != "" {
		lines = append(lines, fmt.Sprintf("Repository path: %s", snapshot.RepoDir))
	}
	if snapshot.CurrentBranch == "" {
		if snapshot.LastError != "" {
			lines = append(lines, fmt.Sprintf("Current branch: (unavailable: %s)", snapshot.LastError))
		} else {
			lines = append(lines, "Current branch: (unavailable)")
		}
		return lines
	}
	if snapshot.DefaultBranch == "" {
		lines = append(lines, fmt.Sprintf("Current branch: %s (default branch unknown)", snapshot.CurrentBranch))
	} else if snapshot.CurrentBranch == snapshot.DefaultBranch {
		lines = append(lines, fmt.Sprintf("Current branch: %s (default branch)", snapshot.CurrentBranch))
	} else {
		lines = append(lines, fmt.Sprintf("Current branch: %s (non-default)", snapshot.CurrentBranch))
		lines = append(lines, fmt.Sprintf("Default branch: %s", snapshot.DefaultBranch))
	}
	if snapshot.StartupBranch != "" {
		lines = append(lines, fmt.Sprintf("Startup branch: %s", snapshot.StartupBranch))
	}
	if !snapshot.LastSync.IsZero() {
		lines = append(lines, fmt.Sprintf("Last sync: %s", snapshot.LastSync.Format(time.RFC3339)))
	}
	return lines
}
