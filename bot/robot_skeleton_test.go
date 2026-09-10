package bot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRobotSkeletonKeepsGeneratedValuesInVariables(t *testing.T) {
	root := filepath.Join("..", "robot.skel", "conf")

	common := readSkeletonTestFile(t, filepath.Join(root, "variables", "common.yaml"))
	for _, want := range []string{
		`ROBOT_NAME: "<botname>"`,
		`ROBOT_EMAIL: "<botemail>"`,
		`ROBOT_FULL_NAME: "<botfullname>"`,
		`ROBOT_ALIAS: "<botalias>"`,
		`DEFAULT_JOB_CHANNEL: "<jobchannel>"`,
		`SSH_HOST_KEY: "<sshhostkeyencrypted>"`,
	} {
		if !strings.Contains(common, want) {
			t.Errorf("common variables missing %q", want)
		}
	}

	robotConfig := readSkeletonTestFile(t, filepath.Join(root, "robot.yaml"))
	for _, want := range []string{
		`variable "ROBOT_NAME"`,
		`variable "ROBOT_EMAIL"`,
		`variable "ROBOT_FULL_NAME"`,
		`variable "ROBOT_ALIAS"`,
		`variable "DEFAULT_JOB_CHANNEL"`,
	} {
		if !strings.Contains(robotConfig, want) {
			t.Errorf("robot.yaml missing %q", want)
		}
	}
	for _, unwanted := range []string{
		`<botname>`, `<botemail>`, `<botfullname>`, `<botalias>`,
		`default "production"`, "DefaultMessageFormat:", "  Job:",
	} {
		if strings.Contains(robotConfig, unwanted) {
			t.Errorf("robot.yaml contains unwanted %q", unwanted)
		}
	}

	sshConfig := readSkeletonTestFile(t, filepath.Join(root, "protocols", "ssh.yaml"))
	for _, want := range []string{"ListenHost: localhost", "ListenPort: 4221"} {
		if !strings.Contains(sshConfig, want) {
			t.Errorf("ssh.yaml missing %q", want)
		}
	}
	if strings.Contains(sshConfig, "GOPHER_SSH_") {
		t.Fatal("ssh.yaml still uses GOPHER_SSH_* environment configuration")
	}

	if _, err := os.Stat(filepath.Join(root, "protocols", "terminal.yaml")); !os.IsNotExist(err) {
		t.Fatalf("deprecated terminal scaffold file still exists: %v", err)
	}

	for _, environment := range []string{"development", "production"} {
		body := readSkeletonTestFile(t, filepath.Join(root, "variables", environment+".yaml"))
		for _, want := range []string{"override matching entries from common.yaml", "Secrets: {}", "Variables: {}"} {
			if !strings.Contains(body, want) {
				t.Errorf("%s variables file missing %q", environment, want)
			}
		}
	}
}

func readSkeletonTestFile(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(body)
}
