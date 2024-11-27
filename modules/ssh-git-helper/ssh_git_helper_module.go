// Package sshhostkeys provides functionality for managing SSH known_hosts files
// within Gopherbot, allowing for secure SSH host verification without relying
// on external files or global known_hosts.

package sshhostkeys

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

const (
	knownHostsDirName = ".ssh-host-keys" // Directory for known_hosts files
)

var knownHostsDirPath string

// HostKeysManager manages known_hosts files and their lifecycle.
type HostKeysManager struct {
	hosts map[string]*HostKeysInstance
	mu    sync.Mutex
}

// HostKeysInstance represents an individual known_hosts file.
type HostKeysInstance struct {
	handle         string
	knownHostsPath string
}

// Global host keys manager instance.
var manager = &HostKeysManager{
	hosts: make(map[string]*HostKeysInstance),
}

// Initialize sets up the known_hosts directory and ensures the known_hosts file exists.
// It logs informational messages using the provided handler and returns any encountered errors.
func Initialize(r robot.Handler) (err error) {
	defer func() {
		if err == nil {
			r.Log(robot.Info, "ssh_git_helper knownHostsDirPath set to: %s", knownHostsDirPath)
		}
	}()

	// Attempt to get the current working directory
	currentDir, err := os.Getwd()
	if err == nil {
		knownHostsDirPath = filepath.Join(currentDir, knownHostsDirName)
		err = os.MkdirAll(knownHostsDirPath, 0700)
		if err == nil {
			// Successfully created the directory in the current working directory
			if err := createKnownHostsFile(currentDir); err != nil {
				return fmt.Errorf("failed to create known_hosts file in %s: %w", currentDir, err)
			}
			return nil
		}
	}

	// If creating in the current directory failed, attempt to use the user's home directory
	usr, err := user.Current()
	if err != nil {
		return fmt.Errorf("failed to determine current user: %w", err)
	}

	knownHostsDirPath = filepath.Join(usr.HomeDir, knownHostsDirName)
	err = os.MkdirAll(knownHostsDirPath, 0700)
	if err != nil {
		return fmt.Errorf("failed to create %s directory in both current and home directories: %w", knownHostsDirName, err)
	}

	// Successfully created the directory in the user's home directory
	if err := createKnownHostsFile(usr.HomeDir); err != nil {
		return fmt.Errorf("failed to create known_hosts file in %s: %w", usr.HomeDir, err)
	}

	return nil
}

// createKnownHostsFile ensures the known_hosts file exists and has the correct permissions.
// It returns an error if any operation fails.
func createKnownHostsFile(basePath string) error {
	sshDir := filepath.Join(basePath, ".ssh")
	err := os.MkdirAll(sshDir, 0700)
	if err != nil {
		return fmt.Errorf("failed to create %s directory: %w", sshDir, err)
	}

	hostsFile := filepath.Join(sshDir, "known_hosts")

	// Check if the known_hosts file already exists
	info, err := os.Stat(hostsFile)
	if err == nil {
		// File exists, ensure it has the correct permissions
		if info.Mode().Perm() != 0600 {
			err = os.Chmod(hostsFile, 0600)
			if err != nil {
				return fmt.Errorf("failed to set permissions on %s: %w", hostsFile, err)
			}
		}
		return nil
	}

	if !os.IsNotExist(err) {
		// An error other than "not exists" occurred
		return fmt.Errorf("failed to stat %s: %w", hostsFile, err)
	}

	// Create the known_hosts file since it does not exist
	file, err := os.OpenFile(hostsFile, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", hostsFile, err)
	}
	defer file.Close()

	return nil
}

// generateHandle generates a unique handle for each host keys instance.
func generateHandle() string {
	return fmt.Sprintf("hostkeys-%d", time.Now().UnixNano())
}

// AddHostKeys adds provided host keys to a new known_hosts file and returns the path and handle.
func AddHostKeys(hostKeys string) (handle string, err error) {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	handle = generateHandle()
	knownHostsPath := filepath.Join(knownHostsDirPath, handle)

	// Write the host keys to the known_hosts file
	err = os.WriteFile(knownHostsPath, []byte(hostKeys), 0600)
	if err != nil {
		return "", fmt.Errorf("failed to write known_hosts file: %w", err)
	}

	instance := &HostKeysInstance{
		handle:         handle,
		knownHostsPath: knownHostsPath,
	}

	manager.hosts[handle] = instance

	return handle, nil
}

// LoadHostKeys loads host keys for known providers based on the repository URL.
// Currently supports GitHub and Bitbucket.
func LoadHostKeys(repoURL string) (handle string, err error) {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	// Parse the repo URL to determine the host
	host, err := ParseHostFromRepoURL(repoURL)
	if err != nil {
		return "", err
	}

	// Fetch the host keys based on the provider
	var hostKeys string
	switch host {
	case "github.com":
		hostKeys, err = getGitHubHostKeys()
		// hostKeys, err = getBogusGitHubHostKeys()
	case "bitbucket.org":
		hostKeys, err = getBitbucketHostKeys()
	default:
		return "", fmt.Errorf("host keys for %s not supported", host)
	}

	if err != nil {
		return "", fmt.Errorf("failed to load host keys for %s: %w", host, err)
	}

	handle = generateHandle()
	knownHostsPath := filepath.Join(knownHostsDirPath, handle)

	// Write the host keys to the known_hosts file
	err = os.WriteFile(knownHostsPath, []byte(hostKeys), 0600)
	if err != nil {
		return "", fmt.Errorf("failed to write known_hosts file: %w", err)
	}

	instance := &HostKeysInstance{
		handle:         handle,
		knownHostsPath: knownHostsPath,
	}

	manager.hosts[handle] = instance

	return handle, nil
}

// ScanHost scans the host to retrieve its host key and writes it to a known_hosts file.
func ScanHost(host string) (handle string, err error) {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	handle = generateHandle()
	knownHostsPath := filepath.Join(knownHostsDirPath, handle)

	// Use net.Dial to connect to the host's SSH port and retrieve the host key
	addr := net.JoinHostPort(host, "22")
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return "", fmt.Errorf("failed to connect to %s: %w", addr, err)
	}
	defer conn.Close()

	// Prepare a variable to store the host key
	var hostKey ssh.PublicKey

	// Perform SSH handshake to get the host key
	config := &ssh.ClientConfig{
		User: "none",
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			hostKey = key
			return nil
		},
		Timeout: 10 * time.Second,
	}
	sshConn, _, _, err := ssh.NewClientConn(conn, host, config)
	if err != nil {
		return "", fmt.Errorf("failed to perform SSH handshake with %s: %w", host, err)
	}
	defer sshConn.Close()

	// Build known_hosts entry
	hostKeyEntry := knownhosts.Line([]string{host}, hostKey)

	// Write the known_hosts line to the file
	err = os.WriteFile(knownHostsPath, []byte(hostKeyEntry+"\n"), 0600)
	if err != nil {
		return "", fmt.Errorf("failed to write known_hosts file: %w", err)
	}

	instance := &HostKeysInstance{
		handle:         handle,
		knownHostsPath: knownHostsPath,
	}

	manager.hosts[handle] = instance

	return handle, nil
}

// Delete removes the known_hosts file associated with the handle.
func Delete(handle string) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	instance, exists := manager.hosts[handle]
	if !exists {
		return nil // No error if the handle doesn't exist
	}

	err := os.Remove(instance.knownHostsPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete known_hosts file: %w", err)
	}

	delete(manager.hosts, handle)

	return nil
}

// GetHostKeysPath returns the host keys associated with the handle.
func GetHostKeysPath(handle string) (string, error) {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	instance, exists := manager.hosts[handle]
	if !exists {
		return "", errors.New("host keys handle not found")
	}
	return instance.knownHostsPath, nil
}

// GetKnownHostsPath returns the path to the known hosts file
// associated with the handle.
func GetKnownHostsPath(handle string) (string, error) {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	instance, exists := manager.hosts[handle]
	if !exists {
		return "", errors.New("host keys handle not found")
	}
	return instance.knownHostsPath, nil
}

// ParseHostFromRepoURL parses the host from the repository URL and is exported for use by other modules.
func ParseHostFromRepoURL(repoURL string) (string, error) {
	// Handle different URL formats
	// e.g., git@github.com:user/repo.git
	//       ssh://git@github.com/user/repo.git
	//       https://github.com/user/repo.git

	if strings.HasPrefix(repoURL, "git@") {
		// Format: git@host:user/repo.git
		parts := strings.SplitN(repoURL, "@", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid repository URL format")
		}
		hostPart := parts[1]
		hostParts := strings.SplitN(hostPart, ":", 2)
		if len(hostParts) < 1 {
			return "", fmt.Errorf("invalid repository URL format")
		}
		host := hostParts[0]
		return host, nil
	} else if strings.HasPrefix(repoURL, "ssh://") || strings.HasPrefix(repoURL, "https://") {
		// Format: protocol://host/...
		parts := strings.SplitN(repoURL, "://", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid repository URL format")
		}
		rest := parts[1]
		// Extract host
		slashIndex := strings.Index(rest, "/")
		if slashIndex == -1 {
			return "", fmt.Errorf("invalid repository URL format")
		}
		host := rest[:slashIndex]
		return host, nil
	} else {
		return "", fmt.Errorf("unsupported repository URL format")
	}
}
