// Package sshagent provides a goroutine-based SSH agent for Gopherbot, allowing both
// internal Go code and external bash plugins to use SSH authentication via unique agents.
package sshagent

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

const (
	socketDirName = ".ssh-agent-sockets" // Constant for socket directory name
)

// Package-level variable for the absolute path of the socket directory
var socketDirPath string

// AgentManager holds agents and manages lifecycle functions.
type AgentManager struct {
	agents map[string]*AgentInstance
	mu     sync.Mutex
}

// AgentInstance represents an individual SSH agent with a unique socket.
type AgentInstance struct {
	keyring  agent.Agent
	socket   string
	handle   string
	stopChan chan struct{}
}

// Global agent manager instance.
var manager = &AgentManager{
	agents: make(map[string]*AgentInstance),
}

func init() {
	// Try creating the socket directory in the current working directory
	currentDir, err := os.Getwd()
	if err == nil {
		socketDirPath = filepath.Join(currentDir, socketDirName)
		err = os.MkdirAll(socketDirPath, 0700)
		if err == nil {
			return // Successfully created in the current directory
		}
	}

	// If the current working directory fails, try the user's home directory
	usr, userErr := user.Current()
	if userErr != nil {
		log.Fatalf("Failed to determine current user: %v", userErr)
	}

	socketDirPath = filepath.Join(usr.HomeDir, socketDirName)
	err = os.MkdirAll(socketDirPath, 0700)
	if err != nil {
		log.Fatalf("Failed to create %s directory in both current and home directories: %v", socketDirName, err)
	}
}

// New starts a new SSH agent with a specified key file and timeout.
func New(keypath, passphrase string, timeoutMinutes int) (agentPath, handle string, err error) {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	// Create a unique handle and absolute socket path
	handle = generateHandle()
	socketPath := filepath.Join(socketDirPath, handle)

	// Initialize agent keyring
	keyring := agent.NewKeyring()
	instance := &AgentInstance{
		keyring:  keyring,
		socket:   socketPath,
		handle:   handle,
		stopChan: make(chan struct{}),
	}

	// Load the SSH key into the agent
	err = loadKey(keyring, keypath, passphrase)
	if err != nil {
		return "", "", fmt.Errorf("failed to load key: %w", err)
	}

	// Start agent serving goroutine
	go instance.serve(timeoutMinutes)
	manager.agents[handle] = instance

	return socketPath, handle, nil
}

// NewWithDeployKey initializes an SSH agent with a deployment key string and timeout.
func NewWithDeployKey(deployKey string, timeoutMinutes int) (agentPath, handle string, err error) {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	// Replace "_" with " " and ":" with "\n" to reconstruct the deploy key
	deployKey = strings.ReplaceAll(deployKey, "_", " ")
	deployKey = strings.ReplaceAll(deployKey, ":", "\n")

	// Parse the deploy key to create a private key object
	privateKey, err := ssh.ParseRawPrivateKey([]byte(deployKey))
	if err != nil {
		return "", "", fmt.Errorf("failed to parse deployment key: %w", err)
	}

	// Create a unique handle and absolute socket path
	handle = generateHandle()
	socketPath := filepath.Join(socketDirPath, handle)

	// Create agent instance and load the key
	keyring := agent.NewKeyring()
	err = keyring.Add(agent.AddedKey{PrivateKey: privateKey})
	if err != nil {
		return "", "", fmt.Errorf("failed to add deployment key to agent: %w", err)
	}

	// Start agent serving goroutine
	instance := &AgentInstance{
		keyring:  keyring,
		socket:   socketPath,
		handle:   handle,
		stopChan: make(chan struct{}),
	}
	go instance.serve(timeoutMinutes)
	manager.agents[handle] = instance

	return socketPath, handle, nil
}

// Get retrieves an agent instance for internal Go library usage.
func Get(handle string) (agent.Agent, error) {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	instance, exists := manager.agents[handle]
	if !exists {
		return nil, errors.New("agent handle not found")
	}
	return instance.keyring, nil
}

// GetKeyID retrieves the key ID (e.g., fingerprint or type and comment) from the agent for the given handle.
// It returns an error if there is more than one key in the agent.
func GetKeyID(handle string) (string, error) {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	instance, exists := manager.agents[handle]
	if !exists {
		return "", errors.New("agent handle not found")
	}

	keys, err := instance.keyring.List()
	if err != nil {
		return "", fmt.Errorf("failed to list keys: %w", err)
	}

	if len(keys) == 0 {
		return "", errors.New("no keys found in the agent")
	}

	if len(keys) > 1 {
		return "", errors.New("multiple keys found in the agent; only one expected")
	}

	key := keys[0]

	// Parse the public key
	pubKey, err := ssh.ParsePublicKey(key.Blob)
	if err != nil {
		return "", fmt.Errorf("failed to parse public key: %w", err)
	}

	// Compute the fingerprint
	fingerprint := ssh.FingerprintSHA256(pubKey)

	// Get the key type
	keyType := pubKey.Type()

	// Get the key length
	var keyLen int

	// Check if the public key implements CryptoPublicKey interface
	if cryptoPubKey, ok := pubKey.(ssh.CryptoPublicKey); ok {
		switch pk := cryptoPubKey.CryptoPublicKey().(type) {
		case *rsa.PublicKey:
			keyLen = pk.N.BitLen()
		case *ecdsa.PublicKey:
			keyLen = pk.Params().BitSize
		case ed25519.PublicKey:
			keyLen = 256
		default:
			return "", fmt.Errorf("unsupported key type %T", pk)
		}
	} else {
		return "", fmt.Errorf("public key does not implement CryptoPublicKey interface")
	}

	// Format the output similar to ssh-add -l
	return fmt.Sprintf("%d %s (%s)", keyLen, fingerprint, keyType), nil
}

// Close stops the SSH agent for a given handle and removes its socket.
func Close(handle string) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	instance, exists := manager.agents[handle]
	if !exists {
		return nil // no error if the agent is already stopped or nonexistent
	}
	close(instance.stopChan)
	delete(manager.agents, handle)
	return nil
}

// serve handles the SSH agent socket and automatically stops after timeout.
func (a *AgentInstance) serve(timeoutMinutes int) {
	// Create the Unix socket
	socketListener, err := net.Listen("unix", a.socket)
	if err != nil {
		fmt.Printf("Error creating socket for handle %s: %v\n", a.handle, err)
		return
	}
	defer socketListener.Close()
	defer os.Remove(a.socket) // cleanup socket on exit

	// Set up a timeout to close the agent if not manually stopped
	timeout := time.After(time.Duration(timeoutMinutes) * time.Minute)
	for {
		select {
		case <-a.stopChan:
			return // manual stop
		case <-timeout:
			return // auto timeout
		default:
			conn, err := socketListener.Accept()
			if err != nil {
				continue
			}
			go agent.ServeAgent(a.keyring, conn)
		}
	}
}

// loadKey adds an SSH key to the agent keyring.
func loadKey(keyring agent.Agent, keypath, passphrase string) error {
	keyBytes, err := os.ReadFile(keypath)
	if err != nil {
		return fmt.Errorf("error reading key file: %w", err)
	}

	privateKey, err := ssh.ParseRawPrivateKeyWithPassphrase(keyBytes, []byte(passphrase))
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}
	return keyring.Add(agent.AddedKey{PrivateKey: privateKey})
}

// generateHandle generates a unique handle for each agent instance.
func generateHandle() string {
	return fmt.Sprintf("agent-%d", time.Now().UnixNano())
}
