package gitcommand

import (
	"fmt"
	"io"
	"os"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
)

// CloneOptions holds the parameters for cloning a repository.
type CloneOptions struct {
	RepoURL   string
	Branch    string // Empty string means default branch
	Directory string
	Auth      transport.AuthMethod
}

// Clone clones a Git repository based on the provided options.
func Clone(opts CloneOptions) error {
	// Prepare clone options
	cloneOptions := &git.CloneOptions{
		URL:      opts.RepoURL,
		Auth:     opts.Auth,
		Progress: os.Stdout,
	}

	// Set reference name if a branch is specified
	if opts.Branch != "" {
		cloneOptions.ReferenceName = plumbing.NewBranchReferenceName(opts.Branch)
		cloneOptions.SingleBranch = true
	}

	// Ensure the directory is clean or create it
	if err := prepareDirectory(opts.Directory); err != nil {
		return err
	}

	// Clone the repository
	_, err := git.PlainClone(opts.Directory, false, cloneOptions)
	if err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	return nil
}

// prepareDirectory ensures that the target directory is empty or creates it.
func prepareDirectory(dir string) error {
	// Check if the directory exists
	if _, err := os.Stat(dir); err == nil {
		// Directory exists, check if it's empty
		isEmpty, err := isDirEmpty(dir)
		if err != nil {
			return fmt.Errorf("failed to check if directory is empty: %w", err)
		}
		if !isEmpty {
			return fmt.Errorf("directory %s exists and is not empty", dir)
		}
	} else {
		// Directory does not exist, create it
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	return nil
}

// isDirEmpty checks if a directory is empty.
func isDirEmpty(dir string) (bool, error) {
	f, err := os.Open(dir)
	if err != nil {
		return false, fmt.Errorf("failed to open directory %s: %w", dir, err)
	}
	defer f.Close()

	// Read directory entries
	_, err = f.Readdirnames(1)
	if err == nil {
		// Directory is not empty
		return false, nil
	}
	if err == io.EOF {
		// Directory is empty
		return true, nil
	}
	return false, fmt.Errorf("failed to read directory %s: %w", dir, err)
}

// PublicKeysWithHostKeyCallback extends gitssh.PublicKeys to include a HostKeyCallback.
type PublicKeysWithHostKeyCallback struct {
	*gitssh.PublicKeys
	HostKeyCallback ssh.HostKeyCallback
}

// ClientConfig returns an ssh.ClientConfig with the HostKeyCallback set.
func (p *PublicKeysWithHostKeyCallback) ClientConfig() (*ssh.ClientConfig, error) {
	config, err := p.PublicKeys.ClientConfig()
	if err != nil {
		return nil, err
	}
	config.HostKeyCallback = p.HostKeyCallback
	return config, nil
}

// CreateSSHAuthMethod creates an AuthMethod using the provided SSH agent and host key callback.
func CreateSSHAuthMethod(agentClient agent.Agent, hostKeyCallback ssh.HostKeyCallback) (transport.AuthMethod, error) {
	signers, err := agentClient.Signers()
	if err != nil {
		return nil, err
	}

	if len(signers) == 0 {
		return nil, fmt.Errorf("no signers found in SSH agent")
	}

	// Create gitssh.PublicKeys with the first signer
	publicKeys := &gitssh.PublicKeys{
		User:   "git",
		Signer: signers[0],
	}

	// Return the custom AuthMethod with HostKeyCallback
	return &PublicKeysWithHostKeyCallback{
		PublicKeys:      publicKeys,
		HostKeyCallback: hostKeyCallback,
	}, nil
}

// CreateHostKeyCallback creates a HostKeyCallback using the known hosts data.
func CreateHostKeyCallback(knownHostsData string) (ssh.HostKeyCallback, error) {
	// Write known hosts data to a temporary file
	tmpFile, err := os.CreateTemp("", "known_hosts")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary known_hosts file: %w", err)
	}
	defer os.Remove(tmpFile.Name()) // Clean up the file later

	_, err = tmpFile.WriteString(knownHostsData)
	if err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("failed to write known hosts data to temporary file: %w", err)
	}
	tmpFile.Close()

	// Create a host key callback from the temporary known_hosts file
	hostKeyCallback, err := knownhosts.New(tmpFile.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to create host key callback: %w", err)
	}
	return hostKeyCallback, nil
}
