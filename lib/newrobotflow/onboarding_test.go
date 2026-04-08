package newrobotflow

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lnxjedi/gopherbot/robot"
)

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
	for _, unwanted := range []string{
		"GOPHER_CUSTOM_REPOSITORY=",
		"GOPHER_DEPLOY_KEY=",
		"GOPHER_BOTNAME=",
		"GOPHER_ENVIRONMENT=development",
	} {
		if strings.Contains(got, unwanted) {
			t.Fatalf(".env still contains %q: %q", unwanted, got)
		}
	}
	if !strings.Contains(got, "KEEP_ME=yes") {
		t.Fatalf(".env dropped unrelated content: %q", got)
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

func TestFindSessionForJoinMatchesCanonicalUser(t *testing.T) {
	state := setupStateFile{
		Sessions: map[string]setupSession{
			"alice": {
				Status:        statusCompleted,
				Stage:         stageRepoReady,
				CanonicalUser: "samantha",
			},
		},
	}
	key, session, found := findSessionForJoin(state, "samantha")
	if !found {
		t.Fatal("findSessionForJoin() did not find canonical user match")
	}
	if key != "alice" {
		t.Fatalf("findSessionForJoin() key = %q, want alice", key)
	}
	if session.CanonicalUser != "samantha" {
		t.Fatalf("findSessionForJoin() canonical user = %q", session.CanonicalUser)
	}
}

type onboardingTestRobot struct {
	parameters map[string]string
	message    *robot.Message
}

func (r *onboardingTestRobot) CheckAdmin() bool                      { return false }
func (r *onboardingTestRobot) Subscribe() bool                       { return false }
func (r *onboardingTestRobot) Unsubscribe() bool                     { return false }
func (r *onboardingTestRobot) Elevate(bool) bool                     { return false }
func (r *onboardingTestRobot) GetBotAttribute(string) *robot.AttrRet { return &robot.AttrRet{} }
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
func (r *onboardingTestRobot) SendChannelMessage(string, string, ...interface{}) robot.RetVal {
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
func (r *onboardingTestRobot) SendUserMessage(string, string, ...interface{}) robot.RetVal {
	return robot.Ok
}
func (r *onboardingTestRobot) Reply(string, ...interface{}) robot.RetVal       { return robot.Ok }
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
func (r *onboardingTestRobot) PromptUserForReply(string, string, string, ...interface{}) (string, robot.RetVal) {
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
func (r *onboardingTestRobot) DeleteDatum(string) robot.RetVal             { return robot.Ok }
func (r *onboardingTestRobot) Remember(string, string, bool)               {}
func (r *onboardingTestRobot) RememberThread(string, string, bool)         {}
func (r *onboardingTestRobot) RememberContext(string, string)              {}
func (r *onboardingTestRobot) RememberContextThread(string, string)        {}
func (r *onboardingTestRobot) Recall(string, bool) string                  { return "" }
func (r *onboardingTestRobot) DeleteMemory(string, bool)                   {}
func (r *onboardingTestRobot) SpawnJob(string, ...string) robot.RetVal     { return robot.Ok }
func (r *onboardingTestRobot) AddTask(string, ...string) robot.RetVal      { return robot.Ok }
func (r *onboardingTestRobot) FinalTask(string, ...string) robot.RetVal    { return robot.Ok }
func (r *onboardingTestRobot) FailTask(string, ...string) robot.RetVal     { return robot.Ok }
func (r *onboardingTestRobot) AddJob(string, ...string) robot.RetVal       { return robot.Ok }
func (r *onboardingTestRobot) AddCommand(string, string) robot.RetVal      { return robot.Ok }
func (r *onboardingTestRobot) FinalCommand(string, string) robot.RetVal    { return robot.Ok }
func (r *onboardingTestRobot) FailCommand(string, string) robot.RetVal     { return robot.Ok }
func (r *onboardingTestRobot) EncryptSecret(string) (string, robot.RetVal) { return "", robot.Failed }
func (r *onboardingTestRobot) RaisePriv(string)                            {}
func (r *onboardingTestRobot) SetParameter(string, string) bool            { return true }
func (r *onboardingTestRobot) SetWorkingDirectory(string) bool             { return true }

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
