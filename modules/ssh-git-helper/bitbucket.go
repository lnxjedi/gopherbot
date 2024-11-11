package sshhostkeys

import (
	"fmt"
	"io"
	"net/http"
)

// getBitbucketHostKeys fetches Bitbucket's SSH host keys and parses them
func getBitbucketHostKeys() (string, error) {
	resp, err := http.Get("https://bitbucket.org/site/ssh")
	if err != nil {
		return "", fmt.Errorf("failed to fetch Bitbucket SSH keys: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected HTTP status: %s", resp.Status)
	}

	// Read the response body directly since it's in plain text format
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read Bitbucket SSH keys: %w", err)
	}

	return string(body), nil
}
