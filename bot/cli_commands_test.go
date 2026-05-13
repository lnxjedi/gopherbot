package bot

import (
	"bytes"
	"encoding/base64"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
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

func TestProcessCLIHelpUUIDShowsCommandDetails(t *testing.T) {
	output := captureStdout(t, func() {
		code := processCLI("help", []string{"uuid"})
		if code != 0 {
			t.Fatalf("processCLI(help uuid) = %d, want 0", code)
		}
	})
	for _, needle := range []string{
		"Usage: gopherbot uuid",
		"Generates a random UUID",
		"encrypted value suitable for custom/conf/variables/<environment>.yaml",
	} {
		if !strings.Contains(output, needle) {
			t.Fatalf("processCLI(help uuid) missing %q in output:\n%s", needle, output)
		}
	}
}

func TestCLIUUIDRunsBeforeFullInit(t *testing.T) {
	if !cliCommandRunsBeforeInit("uuid") {
		t.Fatal("uuid command should run before full robot initialization")
	}
}

func TestUserCLICommandsRunBeforeFullInit(t *testing.T) {
	for _, command := range []string{
		"delete",
		"decrypt",
		"dump",
		"encrypt",
		"fetch",
		"genkey",
		"gentotp",
		"help",
		"init",
		"list",
		"store",
		"uuid",
		"validate",
		"version",
	} {
		if !cliCommandRunsBeforeInit(command) {
			t.Fatalf("%s command should run before full robot initialization", command)
		}
	}
}

func TestGenerateEncryptedUUID(t *testing.T) {
	oldKey := cryptKey.key
	oldInitialized := cryptKey.initialized
	testKey := []byte("0123456789abcdef0123456789abcdef")
	cryptKey.Lock()
	cryptKey.key = testKey
	cryptKey.initialized = true
	cryptKey.Unlock()
	t.Cleanup(func() {
		cryptKey.Lock()
		cryptKey.key = oldKey
		cryptKey.initialized = oldInitialized
		cryptKey.Unlock()
	})

	plain, encrypted, err := generateEncryptedUUID()
	if err != nil {
		t.Fatalf("generateEncryptedUUID() error = %v", err)
	}
	if _, err := uuid.Parse(plain); err != nil {
		t.Fatalf("generated plaintext is not a UUID: %v", err)
	}
	ct, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		t.Fatalf("encrypted UUID is not base64: %v", err)
	}
	decrypted, err := decrypt(ct, testKey)
	if err != nil {
		t.Fatalf("decrypt(encrypted UUID) error = %v", err)
	}
	if string(decrypted) != plain {
		t.Fatalf("decrypted UUID = %q, want %q", decrypted, plain)
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
