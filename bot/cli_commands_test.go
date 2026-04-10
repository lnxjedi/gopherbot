package bot

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe(): %v", err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = old
	}()

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("Close(stdout writer): %v", err)
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("Copy(stdout): %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("Close(stdout reader): %v", err)
	}
	return buf.String()
}

func TestPrintCLIUsageIncludesHelpDiscovery(t *testing.T) {
	output := captureStdout(t, func() {
		printCLIUsage()
	})
	for _, needle := range []string{
		"Usage: gopherbot [options] [command [command options] [command args]]",
		"help [command]",
		"gopherbot help <command>",
		"gopherbot <command> -h",
	} {
		if !strings.Contains(output, needle) {
			t.Fatalf("printCLIUsage() missing %q in output:\n%s", needle, output)
		}
	}
}

func TestProcessCLIHelpEncryptShowsCommandDetails(t *testing.T) {
	output := captureStdout(t, func() {
		code := processCLI("help", []string{"encrypt"})
		if code != 0 {
			t.Fatalf("processCLI(help encrypt) = %d, want 0", code)
		}
	})
	for _, needle := range []string{
		"Usage: gopherbot encrypt [options] <string>",
		"-f, -file <path|->",
		"-b, -binary",
	} {
		if !strings.Contains(output, needle) {
			t.Fatalf("processCLI(help encrypt) missing %q in output:\n%s", needle, output)
		}
	}
}

func TestProcessCLIEncryptHelpFlagShowsHelp(t *testing.T) {
	output := captureStdout(t, func() {
		code := processCLI("encrypt", []string{"-h"})
		if code != 0 {
			t.Fatalf("processCLI(encrypt -h) = %d, want 0", code)
		}
	})
	if !strings.Contains(output, "Usage: gopherbot encrypt [options] <string>") {
		t.Fatalf("encrypt -h output missing usage:\n%s", output)
	}
}

func TestShouldShowCLICommandHelpForValidateFlag(t *testing.T) {
	if !shouldShowCLICommandHelp("validate", []string{"-h"}) {
		t.Fatal("expected validate -h to be recognized as help")
	}
	if shouldShowCLICommandHelp("validate", []string{"/tmp/robot"}) {
		t.Fatal("did not expect validate path arg to be recognized as help")
	}
}

func TestProcessCLIUnknownCommandShowsError(t *testing.T) {
	output := captureStdout(t, func() {
		code := processCLI("wat", nil)
		if code != 2 {
			t.Fatalf("processCLI(unknown) = %d, want 2", code)
		}
	})
	if !strings.Contains(output, `Error: unknown command "wat"`) {
		t.Fatalf("unknown command output missing error:\n%s", output)
	}
}
