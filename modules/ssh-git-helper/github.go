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

// getBogusGitHubHostKeys returns a known_hosts string with invalid SSH host keys for github.com
func getBogusGitHubHostKeys() (string, error) {
	var knownHostsEntries strings.Builder
	// Adding a bogus RSA key (valid Base64, but not a real key)
	knownHostsEntries.WriteString("github.com ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBFy+SstHHltLtIhGOxOvPQK9db1wAH9HA6Ebg+drGlXDqk4IZ9uv16CBliBqRh3pLa9T7GQvONCAkQaWKs44cCo=\n")
	// Adding a bogus ED25519 key (valid Base64, but not a real key)
	knownHostsEntries.WriteString("github.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIDD6BhVyBEZpINid9F5s3w+MsKvlqEi4NNIczWWpOITl\n")
	return knownHostsEntries.String(), nil
}
