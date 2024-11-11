package sshhostkeys

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// getGitHubHostKeys fetches GitHub's SSH host keys from the API
func getGitHubHostKeys() (string, error) {
	resp, err := http.Get("https://api.github.com/meta")
	if err != nil {
		return "", fmt.Errorf("failed to fetch GitHub meta: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected HTTP status: %s", resp.Status)
	}

	var data struct {
		SSHKeys []string `json:"ssh_keys"`
	}

	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&data)
	if err != nil {
		return "", fmt.Errorf("failed to decode GitHub meta JSON: %w", err)
	}

	// Construct the known_hosts entries for github.com
	var knownHostsEntries strings.Builder
	for _, key := range data.SSHKeys {
		knownHostsEntries.WriteString(fmt.Sprintf("github.com %s\n", key))
	}

	return knownHostsEntries.String(), nil
}
