package newrobotflow

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lnxjedi/gopherbot/robot"
)

func TestStartPluginConfigRequiresAdminPrivateContext(t *testing.T) {
	config := string(StartPluginConfig)
	for _, required := range []string{
		"RequireAdmin: true",
		"RequireAllCommandsPrivate: true",
	} {
		if !strings.Contains(config, required) {
			t.Fatalf("StartPluginConfig missing %q:\n%s", required, config)
		}
	}
}

func TestHandleStartCommandRequiresDirectMessage(t *testing.T) {
	tempDir := t.TempDir()
	previousDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() failed: %v", err)
	}
	defer func() {
		if chdirErr := os.Chdir(previousDir); chdirErr != nil {
			t.Fatalf("restoring cwd failed: %v", chdirErr)
		}
	}()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Chdir(%q) failed: %v", tempDir, err)
	}

	r := &onboardingTestRobot{
		message: &robot.Message{Incoming: &robot.ConnectorMessage{
			Protocol:      "ssh",
			UserName:      "alice",
			ValidatedUser: true,
			HiddenMessage: true,
		}},
	}
	HandleStartCommand(r, CommandStart)

	if len(r.replies) != 1 || !strings.Contains(r.replies[0], "`|c`") {
		t.Fatalf("direct-message guidance = %#v, want one reply containing `|c`", r.replies)
	}
	if _, err := os.Stat(StateFileName); !os.IsNotExist(err) {
		t.Fatalf("non-DM invocation created onboarding state: %v", err)
	}
}

func TestValidateNewSetupScaffoldPath(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(t *testing.T, root, customPath string)
		unsafe bool
	}{
		{name: "absent"},
		{
			name: "empty directory",
			setup: func(t *testing.T, _, customPath string) {
				t.Helper()
				if err := os.Mkdir(customPath, 0755); err != nil {
					t.Fatalf("Mkdir(custom) failed: %v", err)
				}
			},
		},
		{
			name: "ordinary file",
			setup: func(t *testing.T, _, customPath string) {
				t.Helper()
				if err := os.Mkdir(customPath, 0755); err != nil {
					t.Fatalf("Mkdir(custom) failed: %v", err)
				}
				if err := os.WriteFile(filepath.Join(customPath, "keep.txt"), []byte("keep me"), 0600); err != nil {
					t.Fatalf("WriteFile(custom/keep.txt) failed: %v", err)
				}
			},
			unsafe: true,
		},
		{
			name: "hidden git directory",
			setup: func(t *testing.T, _, customPath string) {
				t.Helper()
				if err := os.MkdirAll(filepath.Join(customPath, ".git"), 0755); err != nil {
					t.Fatalf("MkdirAll(custom/.git) failed: %v", err)
				}
			},
			unsafe: true,
		},
		{
			name: "symlink entry",
			setup: func(t *testing.T, root, customPath string) {
				t.Helper()
				if err := os.Mkdir(customPath, 0755); err != nil {
					t.Fatalf("Mkdir(custom) failed: %v", err)
				}
				if err := os.Symlink(filepath.Join(root, "target"), filepath.Join(customPath, "linked")); err != nil {
					t.Fatalf("Symlink(custom/linked) failed: %v", err)
				}
			},
			unsafe: true,
		},
		{
			name: "custom path is symlink",
			setup: func(t *testing.T, root, customPath string) {
				t.Helper()
				target := filepath.Join(root, "target")
				if err := os.Mkdir(target, 0755); err != nil {
					t.Fatalf("Mkdir(target) failed: %v", err)
				}
				if err := os.Symlink(target, customPath); err != nil {
					t.Fatalf("Symlink(custom) failed: %v", err)
				}
			},
			unsafe: true,
		},
		{
			name: "custom path is file",
			setup: func(t *testing.T, _, customPath string) {
				t.Helper()
				if err := os.WriteFile(customPath, []byte("keep me"), 0600); err != nil {
					t.Fatalf("WriteFile(custom) failed: %v", err)
				}
			},
			unsafe: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			customPath := filepath.Join(root, defaultScaffoldPath)
			if tt.setup != nil {
				tt.setup(t, root, customPath)
			}
			err := validateNewSetupScaffoldPath(customPath)
			if tt.unsafe && !errors.Is(err, errUnsafeScaffoldPath) {
				t.Fatalf("validateNewSetupScaffoldPath() error = %v, want errUnsafeScaffoldPath", err)
			}
			if !tt.unsafe && err != nil {
				t.Fatalf("validateNewSetupScaffoldPath() error = %v, want nil", err)
			}
		})
	}
}

func TestHandleStartCommandRefusesAndPreservesExistingCustomData(t *testing.T) {
	tempDir := t.TempDir()
	previousDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() failed: %v", err)
	}
	defer func() {
		if chdirErr := os.Chdir(previousDir); chdirErr != nil {
			t.Fatalf("restoring cwd failed: %v", chdirErr)
		}
	}()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Chdir(%q) failed: %v", tempDir, err)
	}
	if err := os.Mkdir(defaultScaffoldPath, 0755); err != nil {
		t.Fatalf("Mkdir(custom) failed: %v", err)
	}
	const original = "do not remove\n"
	existingPath := filepath.Join(defaultScaffoldPath, "existing.txt")
	if err := os.WriteFile(existingPath, []byte(original), 0600); err != nil {
		t.Fatalf("WriteFile(%q) failed: %v", existingPath, err)
	}

	r := directOnboardingTestRobot()
	HandleStartCommand(r, CommandStart)

	if len(r.replies) != 1 || !strings.Contains(r.replies[0], "absent or completely empty") {
		t.Fatalf("preflight reply = %#v, want one actionable refusal", r.replies)
	}
	if _, err := os.Stat(StateFileName); !os.IsNotExist(err) {
		t.Fatalf("unsafe preflight created onboarding state: %v", err)
	}
	if _, err := os.Stat(".env"); !os.IsNotExist(err) {
		t.Fatalf("unsafe preflight created .env: %v", err)
	}
	body, err := os.ReadFile(existingPath)
	if err != nil {
		t.Fatalf("existing custom data was removed: %v", err)
	}
	if string(body) != original {
		t.Fatalf("existing custom data changed to %q, want %q", body, original)
	}
}

func TestHandleStartCommandRechecksCustomBeforeWritingEnv(t *testing.T) {
	tempDir := t.TempDir()
	previousDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() failed: %v", err)
	}
	defer func() {
		if chdirErr := os.Chdir(previousDir); chdirErr != nil {
			t.Fatalf("restoring cwd failed: %v", chdirErr)
		}
	}()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Chdir(%q) failed: %v", tempDir, err)
	}

	base := directOnboardingTestRobot()
	existingPath := filepath.Join(defaultScaffoldPath, "appeared-during-prompt.txt")
	r := &onboardingPromptTestRobot{
		onboardingTestRobot: base,
		onPrompt: func() {
			if err := os.Mkdir(defaultScaffoldPath, 0755); err != nil {
				t.Fatalf("Mkdir(custom) failed: %v", err)
			}
			if err := os.WriteFile(existingPath, []byte("keep me\n"), 0600); err != nil {
				t.Fatalf("WriteFile(%q) failed: %v", existingPath, err)
			}
		},
	}
	HandleStartCommand(r, CommandStart)

	if _, err := os.Stat(".env"); !os.IsNotExist(err) {
		t.Fatalf("second preflight allowed .env write: %v", err)
	}
	if body, err := os.ReadFile(existingPath); err != nil || string(body) != "keep me\n" {
		t.Fatalf("data created during prompt was changed: body=%q err=%v", body, err)
	}
	if len(base.replies) == 0 || !strings.Contains(base.replies[len(base.replies)-1], "absent or completely empty") {
		t.Fatalf("second preflight replies = %#v, want actionable refusal", base.replies)
	}
}

func directOnboardingTestRobot() *onboardingTestRobot {
	return &onboardingTestRobot{
		message: &robot.Message{
			User:    "alice",
			Channel: "alice",
			Incoming: &robot.ConnectorMessage{
				Protocol:      "ssh",
				UserName:      "alice",
				ValidatedUser: true,
				DirectMessage: true,
			},
		},
	}
}

func TestWriteInitialEnvOnlyKeepsEncryptionKeyBootstrapState(t *testing.T) {
	tempDir := t.TempDir()
	prevDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() failed: %v", err)
	}
	defer func() {
		if chdirErr := os.Chdir(prevDir); chdirErr != nil {
			t.Fatalf("restoring cwd failed: %v", chdirErr)
		}
	}()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Chdir(%q) failed: %v", tempDir, err)
	}

	original := strings.Join([]string{
		"KEEP_ME=yes",
		"GOPHER_CUSTOM_REPOSITORY=git@github.com:example/robot.git",
		"GOPHER_DEPLOY_KEY=oldkey",
		"GOPHER_BOTNAME=clu",
		"GOPHER_ENVIRONMENT=development",
		"",
	}, "\n")
	if err := os.WriteFile(".env", []byte(original), 0600); err != nil {
		t.Fatalf("WriteFile(.env) failed: %v", err)
	}

	const key = "12345678901234567890123456789012"
	if err := writeInitialEnv(key); err != nil {
		t.Fatalf("writeInitialEnv() failed: %v", err)
	}

	body, err := os.ReadFile(".env")
	if err != nil {
		t.Fatalf("ReadFile(.env) failed: %v", err)
	}
	got := string(body)
	if !strings.Contains(got, "GOPHER_ENCRYPTION_KEY="+key) {
		t.Fatalf(".env missing encryption key: %q", got)
	}
	if !strings.Contains(got, "GOPHER_ENVIRONMENT=development") {
		t.Fatalf(".env missing explicit development environment: %q", got)
	}
	for _, unwanted := range []string{
		"GOPHER_CUSTOM_REPOSITORY=",
		"GOPHER_DEPLOY_KEY=",
		"GOPHER_BOTNAME=",
	} {
		if strings.Contains(got, unwanted) {
			t.Fatalf(".env still contains %q: %q", unwanted, got)
		}
	}
	if !strings.Contains(got, "KEEP_ME=yes") {
		t.Fatalf(".env dropped unrelated content: %q", got)
	}
	if !strings.Contains(got, "# Remove GOPHER_ENVIRONMENT=development when deploying to production.") {
		t.Fatalf(".env missing deployment guidance comment: %q", got)
	}
	if !strings.Contains(got, "# GOPHER_ENVIRONMENT=production") {
		t.Fatalf(".env missing environment guidance comment: %q", got)
	}
}

func TestEnsureSSHProtocolUserKeyReplacesTemplateBlock(t *testing.T) {
	tempDir := t.TempDir()
	sshConfigPath := filepath.Join(tempDir, "ssh.yaml")
	original := strings.Join([]string{
		"ProtocolConfig:",
		"  ListenHost: localhost",
		"  DefaultChannel: general",
		"  UserKeys: []",
		"  # - UserName: johndoe",
		"  #   PublicKeys:",
		"  #   - \"ssh-ed25519 AAAA...firstkey\"",
		"",
	}, "\n")
	if err := os.WriteFile(sshConfigPath, []byte(original), 0600); err != nil {
		t.Fatalf("WriteFile(%q) failed: %v", sshConfigPath, err)
	}

	key := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITestKeyExample samantha@example"
	if err := ensureSSHProtocolUserKey(sshConfigPath, "samantha", key); err != nil {
		t.Fatalf("ensureSSHProtocolUserKey() failed: %v", err)
	}

	body, err := os.ReadFile(sshConfigPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) failed: %v", sshConfigPath, err)
	}
	got := string(body)
	if strings.Contains(got, "UserKeys: []") {
		t.Fatalf("ssh config still contains placeholder UserKeys block: %q", got)
	}
	expected := strings.Join([]string{
		"  UserKeys:",
		"  - UserName: \"samantha\"",
		"    PublicKeys:",
		"    - \"" + key + "\"",
	}, "\n")
	if !strings.Contains(got, expected) {
		t.Fatalf("ssh config missing rewritten UserKeys block:\nwant substring:\n%s\n\ngot:\n%s", expected, got)
	}
	if !strings.Contains(got, "# - UserName: johndoe") {
		t.Fatalf("ssh config lost surrounding comments: %q", got)
	}
}

func TestStateMatchesOwnerAndConfiguredUser(t *testing.T) {
	state := setupStateFile{Owner: "alice", ConfiguredUser: "samantha"}
	for _, user := range []string{"alice", "Alice", "samantha", "Samantha"} {
		if !stateMatchesUser(state, user) {
			t.Errorf("stateMatchesUser(%q) = false, want true", user)
		}
	}
	if stateMatchesUser(state, "bob") {
		t.Error("stateMatchesUser(\"bob\") = true, want false")
	}
}

func TestSetupStateRoundTripsOnlyDurableCheckpointData(t *testing.T) {
	tests := []setupStateFile{
		{Checkpoint: checkpointEncryptionRestart, Owner: "alice"},
		{Checkpoint: checkpointRepositoryHandoff, Owner: "alice", ConfiguredUser: "samantha"},
		{Checkpoint: checkpointFinalRestart, Owner: "alice", ConfiguredUser: "samantha"},
	}
	for _, want := range tests {
		t.Run(want.Checkpoint, func(t *testing.T) {
			useOnboardingTestDir(t)
			if err := saveState(want); err != nil {
				t.Fatalf("saveState() failed: %v", err)
			}
			got, err := loadState()
			if err != nil {
				t.Fatalf("loadState() failed: %v", err)
			}
			want.Version = stateFileVersion
			if got != want {
				t.Fatalf("loadState() = %+v, want %+v", got, want)
			}
			body, err := os.ReadFile(StateFileName)
			if err != nil {
				t.Fatalf("ReadFile(%s) failed: %v", StateFileName, err)
			}
			for _, obsolete := range []string{"sessions", "stage", "botName", "botAlias", "repositoryUrl", "sshPublicKey"} {
				if strings.Contains(string(body), obsolete) {
					t.Errorf("durable state contains obsolete questionnaire field %q: %s", obsolete, body)
				}
			}
		})
	}
}

func TestHandleStartCommandRejectsUnsupportedVersionFourState(t *testing.T) {
	useOnboardingTestDir(t)
	const legacy = `{"version":4,"sessions":{"alice":{"stage":"awaiting-bot-name","botName":"partial"}}}`
	if err := os.WriteFile(StateFileName, []byte(legacy), 0600); err != nil {
		t.Fatalf("WriteFile(%s) failed: %v", StateFileName, err)
	}

	r := directOnboardingTestRobot()
	HandleStartCommand(r, CommandStart)

	if len(r.replies) != 1 || !strings.Contains(r.replies[0], "unsupported setup version") {
		t.Fatalf("unsupported-state reply = %#v, want explicit recovery guidance", r.replies)
	}
	body, err := os.ReadFile(StateFileName)
	if err != nil {
		t.Fatalf("unsupported state was removed: %v", err)
	}
	if string(body) != legacy {
		t.Fatalf("unsupported state changed to %q, want original", body)
	}
}

func TestInterruptedQuestionnaireKeepsOnlyEncryptionCheckpoint(t *testing.T) {
	useOnboardingTestDir(t)
	want := setupStateFile{Checkpoint: checkpointEncryptionRestart, Owner: "alice"}
	if err := saveState(want); err != nil {
		t.Fatalf("saveState() failed: %v", err)
	}

	r := directOnboardingTestRobot()
	HandleStartCommand(r, CommandStart)

	got, err := loadState()
	if err != nil {
		t.Fatalf("loadState() failed: %v", err)
	}
	want.Version = stateFileVersion
	if got != want {
		t.Fatalf("state after interrupted questionnaire = %+v, want %+v", got, want)
	}
}

func TestHandleStartCommandRefusesDifferentOwner(t *testing.T) {
	useOnboardingTestDir(t)
	want := setupStateFile{Checkpoint: checkpointEncryptionRestart, Owner: "alice"}
	if err := saveState(want); err != nil {
		t.Fatalf("saveState() failed: %v", err)
	}
	r := directOnboardingTestRobot()
	r.message.User = "bob"
	r.message.Incoming.UserName = "bob"

	HandleStartCommand(r, CommandStart)

	if len(r.replies) != 1 || !strings.Contains(r.replies[0], "owned by another administrator") {
		t.Fatalf("different-owner replies = %#v, want refusal", r.replies)
	}
	got, err := loadState()
	if err != nil {
		t.Fatalf("loadState() failed: %v", err)
	}
	want.Version = stateFileVersion
	if got != want {
		t.Fatalf("different-owner attempt changed state to %+v, want %+v", got, want)
	}
}

func TestClearSessionRemovesStateAndPreservesSetupArtifacts(t *testing.T) {
	useOnboardingTestDir(t)
	state := setupStateFile{
		Checkpoint:     checkpointRepositoryHandoff,
		Owner:          "alice",
		ConfiguredUser: "samantha",
	}
	if err := saveState(state); err != nil {
		t.Fatalf("saveState() failed: %v", err)
	}
	if err := os.WriteFile(".env", []byte("KEEP=yes\n"), 0600); err != nil {
		t.Fatalf("WriteFile(.env) failed: %v", err)
	}
	if err := os.Mkdir(defaultScaffoldPath, 0755); err != nil {
		t.Fatalf("Mkdir(custom) failed: %v", err)
	}
	customFile := filepath.Join(defaultScaffoldPath, "keep.txt")
	if err := os.WriteFile(customFile, []byte("keep me\n"), 0600); err != nil {
		t.Fatalf("WriteFile(%s) failed: %v", customFile, err)
	}

	if err := ClearSession("samantha"); err != nil {
		t.Fatalf("ClearSession() failed: %v", err)
	}
	if _, err := os.Stat(StateFileName); !os.IsNotExist(err) {
		t.Fatalf("ClearSession() left state file: %v", err)
	}
	if body, err := os.ReadFile(".env"); err != nil || string(body) != "KEEP=yes\n" {
		t.Fatalf("ClearSession() changed .env: body=%q err=%v", body, err)
	}
	if body, err := os.ReadFile(customFile); err != nil || string(body) != "keep me\n" {
		t.Fatalf("ClearSession() changed custom data: body=%q err=%v", body, err)
	}
}

func TestResumeJoinUsesDirectConversationAndKeepsCheckpointOnInterruption(t *testing.T) {
	useOnboardingTestDir(t)
	want := setupStateFile{Checkpoint: checkpointEncryptionRestart, Owner: "alice"}
	if err := saveState(want); err != nil {
		t.Fatalf("saveState() failed: %v", err)
	}
	r := &onboardingTestRobot{}

	HandleResumeJoin(r, "alice", "general", "ssh")

	if len(r.userMessages) == 0 {
		t.Fatal("resume join sent no direct user message")
	}
	if len(r.channelMessages) != 0 {
		t.Fatalf("resume join sent channel messages: %#v", r.channelMessages)
	}
	if len(r.promptUsers) == 0 || r.promptUsers[0] != "alice" {
		t.Fatalf("resume prompt users = %#v, want alice", r.promptUsers)
	}
	got, err := loadState()
	if err != nil {
		t.Fatalf("loadState() failed: %v", err)
	}
	want.Version = stateFileVersion
	if got != want {
		t.Fatalf("state after interrupted direct resume = %+v, want %+v", got, want)
	}
}

func TestFinalRestartCheckpointWaitsForConfiguredUserAndThenClears(t *testing.T) {
	useOnboardingTestDir(t)
	state := setupStateFile{
		Checkpoint:     checkpointFinalRestart,
		Owner:          "alice",
		ConfiguredUser: "samantha",
	}
	if err := saveState(state); err != nil {
		t.Fatalf("saveState() failed: %v", err)
	}
	if err := os.WriteFile(".env", []byte("GOPHER_CUSTOM_REPOSITORY=git@example.com:robots/example-robot.git\n"), 0600); err != nil {
		t.Fatalf("WriteFile(.env) failed: %v", err)
	}
	r := &onboardingTestRobot{}

	HandleResumeJoin(r, "alice", "general", "ssh")
	if _, err := os.Stat(StateFileName); err != nil {
		t.Fatalf("owner join cleared final checkpoint before configured user joined: %v", err)
	}
	HandleResumeJoin(r, "samantha", "general", "ssh")
	if _, err := os.Stat(StateFileName); !os.IsNotExist(err) {
		t.Fatalf("configured-user join did not clear final checkpoint: %v", err)
	}
	if len(r.channelMessages) != 0 {
		t.Fatalf("final instructions were sent to a channel: %#v", r.channelMessages)
	}
	joined := strings.Join(r.userMessages, "\n")
	if !strings.Contains(joined, "git@example.com:robots/example-robot.git") {
		t.Fatalf("final direct instructions did not recover repository URL from .env: %q", joined)
	}
}

func TestApplyScaffoldConcentratesGeneratedScalarsInVariables(t *testing.T) {
	installDir, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve install directory: %v", err)
	}
	useOnboardingTestDir(t)
	t.Setenv("GOPHER_INSTALLDIR", installDir)

	session := setupSession{
		BotName:       "acme-bot",
		BotAlias:      ";",
		JobChannel:    "acme-bot-jobs",
		RobotEmail:    "robot@example.com",
		AdminEmail:    "admin@example.com",
		CanonicalUser: "alice",
		SSHPublicKey:  "ssh-ed25519 AAAAC3NzaExample alice@example.com",
	}
	r := &onboardingTestRobot{encryptedSecret: "encrypted-host-key"}
	if err := applyScaffold(r, session); err != nil {
		t.Fatalf("applyScaffold() failed: %v", err)
	}

	common := readOnboardingTestFile(t, filepath.Join(defaultScaffoldPath, "conf", "variables", "common.yaml"))
	for _, want := range []string{
		`ROBOT_NAME: "acme-bot"`,
		`ROBOT_EMAIL: "robot@example.com"`,
		`ROBOT_FULL_NAME: "Acme Gopherbot"`,
		`ROBOT_ALIAS: ";"`,
		`DEFAULT_JOB_CHANNEL: "acme-bot-jobs"`,
		`SSH_HOST_KEY: "encrypted-host-key"`,
	} {
		if !strings.Contains(common, want) {
			t.Errorf("generated common variables missing %q:\n%s", want, common)
		}
	}
	if strings.Contains(common, "<bot") || strings.Contains(common, "<jobchannel>") || strings.Contains(common, "<sshhostkeyencrypted>") {
		t.Fatalf("generated common variables retain onboarding placeholders:\n%s", common)
	}

	robotConfig := readOnboardingTestFile(t, filepath.Join(defaultScaffoldPath, "conf", "robot.yaml"))
	if !strings.Contains(robotConfig, `DefaultJobChannel: {{ variable "DEFAULT_JOB_CHANNEL"`) {
		t.Fatalf("generated robot.yaml does not consume DEFAULT_JOB_CHANNEL:\n%s", robotConfig)
	}
	if strings.Contains(robotConfig, "DefaultJobChannel: acme-bot-jobs") {
		t.Fatalf("generated robot.yaml contains copied job-channel scalar:\n%s", robotConfig)
	}

	sshConfig := readOnboardingTestFile(t, filepath.Join(defaultScaffoldPath, "conf", "protocols", "ssh.yaml"))
	for _, want := range []string{"ListenHost: localhost", "ListenPort: 4221", `UserName: "alice"`} {
		if !strings.Contains(sshConfig, want) {
			t.Errorf("generated ssh.yaml missing %q:\n%s", want, sshConfig)
		}
	}
	if strings.Contains(sshConfig, "GOPHER_SSH_") {
		t.Fatalf("generated ssh.yaml retains listener environment lookups:\n%s", sshConfig)
	}

	if _, err := os.Stat(filepath.Join(defaultScaffoldPath, "conf", "protocols", "terminal.yaml")); !os.IsNotExist(err) {
		t.Fatalf("generated scaffold contains deprecated terminal config: %v", err)
	}
	if _, err := os.Stat(filepath.Join(defaultScaffoldPath, "ssh-host-key.pub")); err != nil {
		t.Fatalf("generated scaffold missing SSH host public key: %v", err)
	}
	if _, err := os.Stat(filepath.Join(defaultScaffoldPath, "robot-ssh.pub")); !os.IsNotExist(err) {
		t.Fatalf("generated scaffold contains legacy host public-key filename: %v", err)
	}
}

func readOnboardingTestFile(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(body)
}

func useOnboardingTestDir(t *testing.T) string {
	t.Helper()
	tempDir := t.TempDir()
	previousDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() failed: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Chdir(%q) failed: %v", tempDir, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(previousDir); err != nil {
			t.Errorf("restoring cwd failed: %v", err)
		}
	})
	return tempDir
}

type onboardingTestRobot struct {
	parameters      map[string]string
	message         *robot.Message
	botAttrs        map[string]string
	encryptedSecret string
	replies         []string
	userMessages    []string
	channelMessages []string
	promptUsers     []string
}

type onboardingPromptTestRobot struct {
	*onboardingTestRobot
	onPrompt func()
}

func (r *onboardingPromptTestRobot) PromptForReply(string, string, ...interface{}) (string, robot.RetVal) {
	if r.onPrompt != nil {
		r.onPrompt()
		r.onPrompt = nil
	}
	return "generate", robot.Ok
}

func (r *onboardingTestRobot) CheckAdmin() bool  { return false }
func (r *onboardingTestRobot) Subscribe() bool   { return false }
func (r *onboardingTestRobot) Unsubscribe() bool { return false }
func (r *onboardingTestRobot) Elevate(bool) bool { return false }
func (r *onboardingTestRobot) GetBotAttribute(name string) *robot.AttrRet {
	if r.botAttrs == nil {
		return &robot.AttrRet{}
	}
	return &robot.AttrRet{Attribute: r.botAttrs[name], RetVal: robot.Ok}
}
func (r *onboardingTestRobot) GetUserAttribute(string, string) *robot.AttrRet {
	return &robot.AttrRet{}
}
func (r *onboardingTestRobot) GetSenderAttribute(string) *robot.AttrRet { return &robot.AttrRet{} }
func (r *onboardingTestRobot) GetTaskConfig(interface{}) robot.RetVal   { return robot.Ok }
func (r *onboardingTestRobot) GetHelpMetadata(string) string            { return "" }
func (r *onboardingTestRobot) GetMessage() *robot.Message               { return r.message }
func (r *onboardingTestRobot) GetParameter(name string) string {
	if r.parameters == nil {
		return ""
	}
	return r.parameters[name]
}
func (r *onboardingTestRobot) GetIdentityCredential(string, string) (*robot.IdentityCredential, robot.RetVal) {
	return nil, robot.IdentityNotLinked
}
func (r *onboardingTestRobot) LinkOAuth2Identity(*robot.OAuth2IdentityLinkRequest) robot.RetVal {
	return robot.Failed
}
func (r *onboardingTestRobot) UnlinkIdentity(string, string) robot.RetVal        { return robot.Failed }
func (r *onboardingTestRobot) Email(string, *bytes.Buffer, ...bool) robot.RetVal { return robot.Failed }
func (r *onboardingTestRobot) EmailUser(string, string, *bytes.Buffer, ...bool) robot.RetVal {
	return robot.Failed
}
func (r *onboardingTestRobot) EmailAddress(string, string, *bytes.Buffer, ...bool) robot.RetVal {
	return robot.Failed
}
func (r *onboardingTestRobot) Exclusive(string, bool) bool                     { return true }
func (r *onboardingTestRobot) Fixed() robot.Robot                              { return r }
func (r *onboardingTestRobot) MessageFormat(robot.MessageFormat) robot.Robot   { return r }
func (r *onboardingTestRobot) Direct() robot.Robot                             { return r }
func (r *onboardingTestRobot) Threaded() robot.Robot                           { return r }
func (r *onboardingTestRobot) Log(robot.LogLevel, string, ...interface{}) bool { return true }
func (r *onboardingTestRobot) SendChannelMessage(_ string, message string, args ...interface{}) robot.RetVal {
	r.channelMessages = append(r.channelMessages, fmt.Sprintf(message, args...))
	return robot.Ok
}
func (r *onboardingTestRobot) SendChannelThreadMessage(string, string, string, ...interface{}) robot.RetVal {
	return robot.Ok
}
func (r *onboardingTestRobot) SendUserChannelMessage(string, string, string, ...interface{}) robot.RetVal {
	return robot.Ok
}
func (r *onboardingTestRobot) SendProtocolUserChannelMessage(string, string, string, string, ...interface{}) robot.RetVal {
	return robot.Ok
}
func (r *onboardingTestRobot) SendUserChannelThreadMessage(string, string, string, string, ...interface{}) robot.RetVal {
	return robot.Ok
}
func (r *onboardingTestRobot) SendUserMessage(_ string, message string, args ...interface{}) robot.RetVal {
	r.userMessages = append(r.userMessages, fmt.Sprintf(message, args...))
	return robot.Ok
}
func (r *onboardingTestRobot) Reply(message string, args ...interface{}) robot.RetVal {
	r.replies = append(r.replies, fmt.Sprintf(message, args...))
	return robot.Ok
}
func (r *onboardingTestRobot) ReplyThread(string, ...interface{}) robot.RetVal { return robot.Ok }
func (r *onboardingTestRobot) Say(string, ...interface{}) robot.RetVal         { return robot.Ok }
func (r *onboardingTestRobot) SayThread(string, ...interface{}) robot.RetVal   { return robot.Ok }
func (r *onboardingTestRobot) RandomInt(int) int                               { return 0 }
func (r *onboardingTestRobot) RandomString([]string) string                    { return "" }
func (r *onboardingTestRobot) Pause(float64)                                   {}
func (r *onboardingTestRobot) PromptForReply(string, string, ...interface{}) (string, robot.RetVal) {
	return "", robot.Failed
}
func (r *onboardingTestRobot) PromptThreadForReply(string, string, ...interface{}) (string, robot.RetVal) {
	return "", robot.Failed
}
func (r *onboardingTestRobot) PromptUserForReply(_ string, user string, _ string, _ ...interface{}) (string, robot.RetVal) {
	r.promptUsers = append(r.promptUsers, user)
	return "", robot.Failed
}
func (r *onboardingTestRobot) PromptUserChannelForReply(string, string, string, string, ...interface{}) (string, robot.RetVal) {
	return "", robot.Failed
}
func (r *onboardingTestRobot) PromptUserChannelThreadForReply(string, string, string, string, string, ...interface{}) (string, robot.RetVal) {
	return "", robot.Failed
}
func (r *onboardingTestRobot) CheckoutDatum(string, interface{}, bool) (string, bool, robot.RetVal) {
	return "", false, robot.DatumNotFound
}
func (r *onboardingTestRobot) CheckinDatum(string, string) {}
func (r *onboardingTestRobot) UpdateDatum(string, string, interface{}) robot.RetVal {
	return robot.Failed
}
func (r *onboardingTestRobot) DeleteDatum(string) robot.RetVal          { return robot.Ok }
func (r *onboardingTestRobot) Remember(string, string, bool)            {}
func (r *onboardingTestRobot) RememberThread(string, string, bool)      {}
func (r *onboardingTestRobot) RememberContext(string, string)           {}
func (r *onboardingTestRobot) RememberContextThread(string, string)     {}
func (r *onboardingTestRobot) Recall(string, bool) string               { return "" }
func (r *onboardingTestRobot) DeleteMemory(string, bool)                {}
func (r *onboardingTestRobot) SpawnJob(string, ...string) robot.RetVal  { return robot.Ok }
func (r *onboardingTestRobot) AddTask(string, ...string) robot.RetVal   { return robot.Ok }
func (r *onboardingTestRobot) FinalTask(string, ...string) robot.RetVal { return robot.Ok }
func (r *onboardingTestRobot) FailTask(string, ...string) robot.RetVal  { return robot.Ok }
func (r *onboardingTestRobot) AddJob(string, ...string) robot.RetVal    { return robot.Ok }
func (r *onboardingTestRobot) AddCommand(string, string) robot.RetVal   { return robot.Ok }
func (r *onboardingTestRobot) FinalCommand(string, string) robot.RetVal { return robot.Ok }
func (r *onboardingTestRobot) FailCommand(string, string) robot.RetVal  { return robot.Ok }
func (r *onboardingTestRobot) EncryptSecret(string) (string, robot.RetVal) {
	if r.encryptedSecret != "" {
		return r.encryptedSecret, robot.Ok
	}
	return "", robot.Failed
}
func (r *onboardingTestRobot) SetParameter(string, string) bool { return true }
func (r *onboardingTestRobot) SetWorkingDirectory(string) bool  { return true }

func TestPreferredOnboardingUserPrefersUSER(t *testing.T) {
	t.Setenv("USER", "shelluser")

	r := &onboardingTestRobot{
		parameters: map[string]string{
			"GOPHER_USER":       "pipelineuser",
			paramOnboardingUser: "setupuser",
		},
		message: &robot.Message{User: "messageuser"},
	}

	got := preferredOnboardingUser(r, "startedby", r.message)
	if got != "shelluser" {
		t.Fatalf("preferredOnboardingUser() = %q, want shelluser", got)
	}
}

func TestValidEncryptionKeyAllowsPunctuation(t *testing.T) {
	const key = "Forky-EAT:Lunker@SmashedBUMBLETS"
	if len(key) != 32 {
		t.Fatalf("test key length = %d, want 32", len(key))
	}
	if !validEncryptionKey(key) {
		t.Fatalf("validEncryptionKey(%q) = false, want true", key)
	}
}

func TestValidEncryptionKeyAllowsLongerValues(t *testing.T) {
	const key = "Forky-EAT:Lunker@SmashedBUMBLETcrumplet"
	if len(key) <= 32 {
		t.Fatalf("test key length = %d, want > 32", len(key))
	}
	if !validEncryptionKey(key) {
		t.Fatalf("validEncryptionKey(%q) = false, want true", key)
	}
}

func TestInvalidEncryptionKeyMessageExplainsRequirements(t *testing.T) {
	msg := invalidEncryptionKeyMessage("short")
	for _, want := range []string{
		"GOPHER_ENCRYPTION_KEY",
		"at least 32 characters",
		"cannot contain spaces, tabs, or line breaks",
		"Letters, digits, and punctuation are all fine",
		"only the first 32 bytes are used today",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("invalidEncryptionKeyMessage() missing %q in %q", want, msg)
		}
	}
	if !strings.Contains(msg, "I received 5 characters.") {
		t.Fatalf("invalidEncryptionKeyMessage() missing length detail in %q", msg)
	}
}

func TestEnableOnboardingHooksAddsExternalJobsSectionWhenMissing(t *testing.T) {
	tempDir := t.TempDir()
	robotConfigPath := filepath.Join(tempDir, "robot.yaml")
	original := strings.Join([]string{
		"IgnoreUnlistedUsers: true",
		"ScheduledJobs:",
		"- Name: pause-notifies",
		"  Schedule: \"0 0 8 * * *\"",
		"",
	}, "\n")
	if err := os.WriteFile(robotConfigPath, []byte(original), 0600); err != nil {
		t.Fatalf("WriteFile(%q) failed: %v", robotConfigPath, err)
	}

	if err := enableOnboardingHooks(robotConfigPath); err != nil {
		t.Fatalf("enableOnboardingHooks() failed: %v", err)
	}

	body, err := os.ReadFile(robotConfigPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) failed: %v", robotConfigPath, err)
	}
	got := string(body)
	if !strings.Contains(got, "ExternalJobs:\n  # BEGIN NEW-ROBOT ONBOARDING JOB") {
		t.Fatalf("robot config missing inserted ExternalJobs section:\n%s", got)
	}
	if !strings.Contains(got, "\"resume-setup\":") {
		t.Fatalf("robot config missing resume-setup entry:\n%s", got)
	}
	if strings.Index(got, "ExternalJobs:") > strings.Index(got, "ScheduledJobs:") {
		t.Fatalf("ExternalJobs section should appear before ScheduledJobs:\n%s", got)
	}
}

func TestPreferredBotNameHandlesNilSession(t *testing.T) {
	r := &onboardingTestRobot{
		parameters: map[string]string{
			"GOPHER_BOTNAME": "parsley",
		},
	}
	defer func() {
		if p := recover(); p != nil {
			t.Fatalf("preferredBotName() panicked with nil session: %v", p)
		}
	}()
	if got := preferredBotName(r, nil); got != "parsley" {
		t.Fatalf("preferredBotName() = %q, want parsley", got)
	}
}

func TestPreferredBotAliasHandlesNilSession(t *testing.T) {
	r := &onboardingTestRobot{
		parameters: map[string]string{
			"GOPHER_ALIAS": "%",
		},
	}
	defer func() {
		if p := recover(); p != nil {
			t.Fatalf("preferredBotAlias() panicked with nil session: %v", p)
		}
	}()
	if got := preferredBotAlias(r, nil); got != "%" {
		t.Fatalf("preferredBotAlias() = %q, want %%", got)
	}
}
