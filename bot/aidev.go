package bot

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
)

var aidevToken string

func setAIDevToken(token string) {
	aidevToken = strings.TrimSpace(token)
}

func isAIDevMode() bool {
	return len(aidevToken) > 0
}

func getAIDevToken() string {
	return aidevToken
}

func writeAIPortFile(listenAddr string) error {
	if !isAIDevMode() {
		return nil
	}
	_, port, err := net.SplitHostPort(listenAddr)
	if err != nil {
		return fmt.Errorf("extracting port from listener address '%s': %w", listenAddr, err)
	}
	path := filepath.Join(homePath, ".aiport")
	raiseThreadPriv("writing .aiport for aidev mode")
	if err := os.WriteFile(path, []byte(port+"\n"), 0644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}
