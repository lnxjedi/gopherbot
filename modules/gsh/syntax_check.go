package gsh

import (
	"fmt"
	"os"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// CheckSyntax parses a gsh script the same way the runtime does: with the
// built-in compatibility shim prepended before the script body.
func CheckSyntax(path string) error {
	shim, err := assets.ReadFile("assets/gopherbot_v1.gsh")
	if err != nil {
		return fmt.Errorf("loading built-in gsh shim: %w", err)
	}
	script, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading gsh script: %w", err)
	}
	source := string(shim) + "\n\n" + string(script)
	_, err = syntax.NewParser().Parse(strings.NewReader(source), path)
	return err
}

// SyntaxLineOffset returns the number of lines prepended before a user script
// when CheckSyntax builds the same combined source used by the runtime.
func SyntaxLineOffset() int {
	shim, err := assets.ReadFile("assets/gopherbot_v1.gsh")
	if err != nil {
		return 0
	}
	return strings.Count(string(shim), "\n") + 2
}
