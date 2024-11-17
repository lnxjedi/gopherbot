package gitcommand

import (
	"fmt"
	"io"
	"os"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
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

// PullOptions holds the parameters for pulling updates in a repository.
type PullOptions struct {
	Directory string
	Auth      transport.AuthMethod
}

// Pull pulls the latest changes in the specified Git repository.
func Pull(opts PullOptions) error {
	// Open the existing repository
	repo, err := git.PlainOpen(opts.Directory)
	if err != nil {
		return fmt.Errorf("failed to open repository at %s: %w", opts.Directory, err)
	}

	// Get the worktree
	w, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Perform the pull operation
	pullOptions := &git.PullOptions{
		Auth:       opts.Auth,
		RemoteName: "origin",
		Progress:   os.Stdout,
	}

	err = w.Pull(pullOptions)
	if err != nil {
		if err == git.NoErrAlreadyUpToDate {
			// No changes to pull; consider this as a successful pull
			return nil
		}
		return fmt.Errorf("failed to pull repository: %w", err)
	}

	return nil
}

// CheckoutOptions holds the parameters for checking out a branch in a repository.
type CheckoutOptions struct {
	Directory string
	Branch    string
	Auth      transport.AuthMethod
}

// Checkout performs a branch checkout in the specified Git repository.
// It fetches the latest changes, switches to the desired branch, sets up tracking, and pulls the latest commits.
func Checkout(opts CheckoutOptions) error {
	// Open the existing repository
	repo, err := git.PlainOpen(opts.Directory)
	if err != nil {
		return fmt.Errorf("failed to open repository at %s: %w", opts.Directory, err)
	}

	// Fetch the latest changes from the remote
	fetchOptions := &git.FetchOptions{
		RemoteName: "origin",
		Auth:       opts.Auth,
		Progress:   os.Stdout,
		Tags:       git.AllTags,
		Force:      true,
		// To avoid errors if the branch is already up to date
		// Also, allow empty fetches
		// No need to specify refspecs to fetch all
	}
	err = repo.Fetch(fetchOptions)
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("failed to fetch updates: %w", err)
	}

	// Get the worktree
	w, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Attempt to checkout the desired branch
	branchRefName := plumbing.NewBranchReferenceName(opts.Branch)
	checkoutOptions := &git.CheckoutOptions{
		Branch: branchRefName,
		Force:  true,
	}

	err = w.Checkout(checkoutOptions)
	if err != nil {
		// If the branch does not exist locally, attempt to create it tracking the remote branch
		if err == plumbing.ErrReferenceNotFound || err.Error() == "reference not found" {
			// Attempt to checkout the remote branch into a new local branch
			checkoutOptions = &git.CheckoutOptions{
				Branch: branchRefName,
				Create: true,
				// Set the starting point to origin/<branch>
				Hash: plumbing.ZeroHash, // Use ZeroHash to indicate starting from HEAD after checkout
			}
			err = w.Checkout(checkoutOptions)
			if err != nil {
				return fmt.Errorf("failed to create and checkout branch %s: %w", opts.Branch, err)
			}

			// Manually set the branch to track origin/<branch>
			cfg, err := repo.Config()
			if err != nil {
				return fmt.Errorf("failed to get repository config: %w", err)
			}

			branchConfig, ok := cfg.Branches[opts.Branch]
			if !ok {
				branchConfig = &config.Branch{
					Name:   opts.Branch,
					Remote: "origin",
					Merge:  plumbing.ReferenceName("refs/heads/" + opts.Branch),
				}
			} else {
				branchConfig.Remote = "origin"
				branchConfig.Merge = plumbing.ReferenceName("refs/heads/" + opts.Branch)
			}

			cfg.Branches[opts.Branch] = branchConfig

			err = repo.Storer.SetConfig(cfg)
			if err != nil {
				return fmt.Errorf("failed to set branch tracking for %s: %w", opts.Branch, err)
			}
		} else {
			return fmt.Errorf("failed to checkout branch %s: %w", opts.Branch, err)
		}
	}

	// After checkout, perform a pull to ensure the branch is up-to-date
	pullOptions := &git.PullOptions{
		Auth:       opts.Auth,
		RemoteName: "origin",
		Progress:   os.Stdout,
	}

	err = w.Pull(pullOptions)
	if err != nil {
		if err == git.NoErrAlreadyUpToDate {
			// No changes to pull; consider this as a successful pull
			return nil
		}
		return fmt.Errorf("failed to pull after checkout: %w", err)
	}

	return nil
}

// GetCurrentBranch returns the name of the current active branch in the repository.
func GetCurrentBranch(directory string) (string, error) {
	repo, err := git.PlainOpen(directory)
	if err != nil {
		return "", fmt.Errorf("failed to open repository at %s: %w", directory, err)
	}

	headRef, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD reference: %w", err)
	}

	if !headRef.Name().IsBranch() {
		return "", fmt.Errorf("HEAD is not pointing to a branch")
	}

	branchName := headRef.Name().Short()
	return branchName, nil
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
	if err == os.ErrNotExist || err == io.EOF {
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
