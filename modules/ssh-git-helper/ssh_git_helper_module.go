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

func init() {
	// Try creating the known_hosts directory in the current working directory
	currentDir, err := os.Getwd()
	if err == nil {
		knownHostsDirPath = filepath.Join(currentDir, knownHostsDirName)
		err = os.MkdirAll(knownHostsDirPath, 0700)
		if err == nil {
			return // Successfully created in the current directory
		}
	}

	// If the current working directory fails, try the user's home directory
	usr, userErr := user.Current()
	if userErr != nil {
		fmt.Printf("Failed to determine current user: %v\n", userErr)
		os.Exit(1)
	}

	knownHostsDirPath = filepath.Join(usr.HomeDir, knownHostsDirName)
	err = os.MkdirAll(knownHostsDirPath, 0700)
	if err != nil {
		fmt.Printf("Failed to create %s directory in both current and home directories: %v\n", knownHostsDirName, err)
		os.Exit(1)
	}
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

// GetHostKeys returns the host keys associated with the handle.
func GetHostKeys(handle string) (string, error) {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	instance, exists := manager.hosts[handle]
	if !exists {
		return "", errors.New("host keys handle not found")
	}

	hostKeysBytes, err := os.ReadFile(instance.knownHostsPath)
	if err != nil {
		return "", fmt.Errorf("failed to read known_hosts file: %w", err)
	}

	return string(hostKeysBytes), nil
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
