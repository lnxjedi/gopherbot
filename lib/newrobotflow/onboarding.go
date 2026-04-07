package newrobotflow

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
)

const (
	StateFileName     = ".setup-state"
	stateFileVersion  = 4
	StateExclusiveTag = "new-robot-state"

	CommandStart  = "new-robot"
	CommandCancel = "new-robot-cancel"

	statusActive    = "active"
	statusCompleted = "completed"

	stageShell              = "wizard-shell" // slice-1 compatibility
	stageAwaitingEncryption = "awaiting-encryption-key"
	stageAwaitingBotName    = "awaiting-bot-name"
	stageAwaitingBotAlias   = "awaiting-bot-alias"
	stageAwaitingJobChan    = "awaiting-job-channel"
	stageAwaitingRobotEmail = "awaiting-robot-email"
	stageAwaitingAdminEmail = "awaiting-admin-email"
	stageAwaitingUsername   = "awaiting-username"
	stageAwaitingConfirm    = "awaiting-confirmation" // backward compatibility
	stageAwaitingSSHKey     = "awaiting-ssh-key"
	stageScaffolded         = "scaffolded"
	stageAwaitingRepoURL    = "awaiting-repository-url"
	stageAwaitingGitPush    = "awaiting-user-git-push"
	stageRepoReady          = "repository-ready"

	defaultScaffoldPath     = "custom"
	defaultEnvironment      = "development"
	defaultCustomRepository = "local"
	defaultBotAlias         = ";"

	paramOnboardingUser     = "GOPHER_ONBOARDING_USER"
	paramOnboardingChannel  = "GOPHER_ONBOARDING_CHANNEL"
	paramOnboardingProtocol = "GOPHER_ONBOARDING_PROTOCOL"

	onboardingJobBeginMarker = "# BEGIN NEW-ROBOT ONBOARDING JOB"
	onboardingJobEndMarker   = "# END NEW-ROBOT ONBOARDING JOB"
)

var (
	usernameRe  = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,31}$`)
	botNameRe   = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,31}$`)
	channelRe   = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)
	sshPubKeyRe = regexp.MustCompile(`^ssh-(?:ed25519|rsa|ecdsa|dss)\s+[A-Za-z0-9+/=]+(?:\s+[-._@A-Za-z0-9]+)?$`)
	envKeyRe    = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

	errScaffoldExists = errors.New("scaffold already exists")
)

var StartPluginConfig = []byte(`
Commands:
- Command: "new-robot"
  Regex: '(?i:new(?:-|[[:space:]]+)robot)$'
  Keywords: [ "new", "robot", "setup", "onboarding" ]
  Usage: "new robot"
  Summary: "Starts guided onboarding for creating a new robot repository and config."
  Examples:
  - "(alias) new robot"
  - "(bot) new robot"
- Command: "new-robot-cancel"
  Regex: '(?i:(?:cancel|abort|stop)[[:space:]]+new(?:-|[[:space:]]+)robot|new(?:-|[[:space:]]+)robot[[:space:]]+(?:cancel|abort|stop))$'
  Keywords: [ "new", "robot", "cancel", "onboarding" ]
  Usage: "new robot cancel"
  Summary: "Cancels and removes your onboarding session state."
  Examples:
  - "(alias) new robot cancel"
  - "(bot) stop new robot"
`)

type setupStateFile struct {
	Version  int                     `json:"version"`
	Sessions map[string]setupSession `json:"sessions"`
}

type setupSession struct {
	Status             string `json:"status"`
	Stage              string `json:"stage"`
	StartedAtUTC       string `json:"startedAtUtc"`
	UpdatedAtUTC       string `json:"updatedAtUtc"`
	CompletedAtUTC     string `json:"completedAtUtc,omitempty"`
	StartedBy          string `json:"startedBy"`
	LastCommand        string `json:"lastCommand"`
	LastChannel        string `json:"lastChannel"`
	LastProtocol       string `json:"lastProtocol"`
	BotName            string `json:"botName,omitempty"`
	BotAlias           string `json:"botAlias,omitempty"`
	JobChannel         string `json:"jobChannel,omitempty"`
	RobotEmail         string `json:"robotEmail,omitempty"`
	AdminEmail         string `json:"adminEmail,omitempty"`
	CanonicalUser      string `json:"canonicalUser,omitempty"`
	SSHPublicKey       string `json:"sshPublicKey,omitempty"`
	SSHPublicKeySource string `json:"sshPublicKeySource,omitempty"`
	RepositoryURL      string `json:"repositoryUrl,omitempty"`
	DeployPublicKey    string `json:"deployPublicKey,omitempty"`
}

type conversation struct {
	r       robot.Robot
	user    string
	channel string
	target  bool
}

func contextualConversation(r robot.Robot) *conversation {
	return &conversation{r: r}
}

func targetedConversation(r robot.Robot, user, channel string) *conversation {
	return &conversation{
		r:       r,
		user:    canonicalUserKey(user),
		channel: canonicalChannelName(channel),
		target:  true,
	}
}

func (c *conversation) Say(msg string, v ...interface{}) {
	if c.target {
		c.r.MessageFormat(robot.BasicMarkdown).SendUserChannelMessage(c.user, c.channel, msg, v...)
		return
	}
	c.r.MessageFormat(robot.BasicMarkdown).Say(msg, v...)
}

func (c *conversation) Reply(msg string, v ...interface{}) {
	if c.target {
		c.r.MessageFormat(robot.BasicMarkdown).SendUserChannelMessage(c.user, c.channel, msg, v...)
		return
	}
	c.r.MessageFormat(robot.BasicMarkdown).Reply(msg, v...)
}

func (c *conversation) FixedSay(msg string, v ...interface{}) {
	if c.target {
		c.r.Fixed().SendUserChannelMessage(c.user, c.channel, msg, v...)
		return
	}
	c.r.Fixed().Say(msg, v...)
}

func (c *conversation) Prompt(regexID, prompt string, v ...interface{}) (string, robot.RetVal) {
	if c.target {
		return c.r.PromptUserChannelForReply(regexID, c.user, c.channel, prompt, v...)
	}
	return c.r.PromptForReply(regexID, prompt, v...)
}

func (c *conversation) Pause(seconds float64) {
	c.r.Pause(seconds)
}

func HasAnySetupState() bool {
	body, err := os.ReadFile(StateFileName)
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(body)) != ""
}

func HandleStartCommand(r robot.Robot, command string) {
	if !r.Exclusive(StateExclusiveTag, false) {
		r.Reply("Another onboarding command is already updating setup state. Please try again in a few seconds.")
		return
	}

	m := r.GetMessage()
	userName, channelName, protocol := onboardingContext(r, m)
	userKey := canonicalUserKey(userName)
	if userKey == "" {
		r.Reply("I couldn't determine your username for onboarding state.")
		return
	}

	state, err := loadState()
	if err != nil {
		r.Log(robot.Error, "Loading %s: %v", StateFileName, err)
		r.Reply("I couldn't read onboarding state from %s", StateFileName)
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	session, exists := state.Sessions[userKey]

	switch command {
	case CommandCancel:
		if !exists {
			r.Reply("You don't have an onboarding session to cancel.")
			return
		}
		delete(state.Sessions, userKey)
		if err := saveState(state); err != nil {
			r.Log(robot.Error, "Saving %s: %v", StateFileName, err)
			r.Reply("I couldn't clear onboarding state in %s", StateFileName)
			return
		}
		if session.Status == statusCompleted {
			r.Reply("Cleared completed onboarding state from %s.", StateFileName)
		} else {
			r.Reply("Canceled your onboarding session and removed it from %s.", StateFileName)
		}
		return
	}

	if !exists {
		session = setupSession{
			Status:       statusActive,
			Stage:        stageAwaitingEncryption,
			StartedAtUTC: now,
			StartedBy:    userKey,
		}
	} else if session.Status == statusCompleted && session.Stage == stageRepoReady {
		r.Reply("Repository handoff is already complete for %s.", session.CanonicalUser)
		if session.RepositoryURL != "" {
			r.Say("Configured GOPHER_CUSTOM_REPOSITORY: %s", session.RepositoryURL)
		}
		return
	} else if session.Status == statusActive && session.Stage != "" && session.Stage != stageAwaitingEncryption && session.Stage != stageShell {
		r.Reply("Setup is already in progress in `%s`.", StateFileName)
		r.Say("Reconnect as @%s and the setup resume job will pick up automatically after restart.", session.StartedBy)
		return
	}

	session.LastCommand = command
	session.LastChannel = channelName
	session.LastProtocol = protocol
	session.UpdatedAtUTC = now
	if session.Stage == "" || session.Stage == stageShell {
		session.Stage = stageAwaitingEncryption
	}

	state.Sessions[userKey] = session
	if err := saveState(state); err != nil {
		r.Log(robot.Error, "Saving %s: %v", StateFileName, err)
		r.Reply("I couldn't update onboarding state in %s", StateFileName)
		return
	}

	session = state.Sessions[userKey]
	continueWizard(contextualConversation(r), &state, userKey, &session)
}

func HandleResumeJoin(r robot.Robot, user, channel, protocol string) {
	user = canonicalUserKey(user)
	channel = canonicalChannelName(channel)
	protocol = strings.ToLower(strings.TrimSpace(protocol))
	if user == "" || channel == "" {
		return
	}
	if !r.Exclusive(StateExclusiveTag, false) {
		r.Log(robot.Warn, "Skipping setup resume for '%s': state is busy", user)
		return
	}
	state, err := loadState()
	if err != nil {
		r.Log(robot.Error, "Loading %s: %v", StateFileName, err)
		return
	}
	sessionKey, session, found := findSessionForJoin(state, user)
	if !found {
		return
	}
	conv := targetedConversation(r, user, channel)
	if strings.ToLower(strings.TrimSpace(session.Status)) == statusCompleted && strings.ToLower(strings.TrimSpace(session.Stage)) == stageRepoReady {
		sendFinalBootstrapInstructions(conv, session)
		if err := ClearSession(user); err != nil {
			r.Log(robot.Error, "Clearing completed onboarding state for '%s': %v", user, err)
		}
		return
	}
	if session.Stage == "" || session.Stage == stageAwaitingEncryption || session.Stage == stageShell {
		return
	}
	conv.Pause(0.5)
	conv.Say("@%s Welcome back - I found onboarding progress in `%s`.", user, StateFileName)
	session.LastChannel = channel
	session.LastProtocol = protocol
	session.UpdatedAtUTC = time.Now().UTC().Format(time.RFC3339)
	state.Sessions[sessionKey] = session
	if err := saveState(state); err != nil {
		r.Log(robot.Error, "Saving %s: %v", StateFileName, err)
		return
	}
	continueWizard(conv, &state, sessionKey, &session)
}

func findSessionForJoin(state setupStateFile, user string) (string, setupSession, bool) {
	if state.Sessions == nil {
		return "", setupSession{}, false
	}
	if session, ok := state.Sessions[user]; ok {
		return user, session, true
	}
	for key, candidate := range state.Sessions {
		if canonicalUserKey(candidate.CanonicalUser) == user {
			return key, candidate, true
		}
	}
	return "", setupSession{}, false
}

func continueWizard(conv *conversation, state *setupStateFile, userKey string, session *setupSession) {
	sessionKey := userKey
	nowUTC := func() string {
		return time.Now().UTC().Format(time.RFC3339)
	}
	persist := func(saveErrorMsg string) bool {
		state.Sessions[sessionKey] = *session
		if err := saveState(*state); err != nil {
			conv.r.Log(robot.Error, "Saving %s: %v", StateFileName, err)
			conv.Reply(saveErrorMsg, StateFileName)
			return false
		}
		return true
	}

	defaultUser := preferredOnboardingUser(conv.r, session.StartedBy, conv.r.GetMessage())
	if session.Stage == stageAwaitingConfirm {
		// Compatibility for older session state values.
		session.Stage = stageAwaitingSSHKey
	}
	defaultJobChannel := preferredJobChannel(session)

	if session.Stage == stageScaffolded {
		session.Stage = stageAwaitingRepoURL
	}

	if session.Stage == stageAwaitingEncryption {
		encryptionKey, ok := promptEncryptionKey(conv)
		if !ok {
			session.Stage = stageAwaitingEncryption
			session.UpdatedAtUTC = nowUTC()
			persist("I couldn't save onboarding progress to %s")
			return
		}
		if err := writeInitialEnv(encryptionKey); err != nil {
			conv.r.Log(robot.Error, "Writing initial onboarding .env: %v", err)
			conv.Reply("I couldn't write your encryption key to .env: %v", err)
			return
		}
		if err := clearOnboardingScaffoldState(); err != nil {
			conv.r.Log(robot.Error, "Clearing onboarding scaffold state: %v", err)
			conv.Reply("I couldn't prepare the directory for restart: %v", err)
			return
		}
		session.Status = statusActive
		session.Stage = stageAwaitingBotName
		session.UpdatedAtUTC = nowUTC()
		if !persist("I couldn't save onboarding progress to %s") {
			return
		}
		conv.Reply("Done. Your encryption key is now in the environment, usually through `.env`, and that's the only setup state I've written so far.")
		conv.Say("Keep that value safe and never commit `.env` to git.")
		conv.Say("I'm restarting now. Reconnect as @%s and the setup resume job will pick up automatically right where we left off.", session.StartedBy)
		conv.Pause(0.5)
		conv.r.AddTask("restart-robot")
		return
	}

	if session.BotName == "" || session.Stage == stageAwaitingBotName {
		name, ok := promptBotName(conv)
		if !ok {
			session.Stage = stageAwaitingBotName
			session.UpdatedAtUTC = nowUTC()
			persist("I couldn't save onboarding progress to %s")
			return
		}
		session.BotName = name
		session.Stage = stageAwaitingBotAlias
		session.UpdatedAtUTC = nowUTC()
		if !persist("I couldn't save onboarding progress to %s") {
			return
		}
		defaultJobChannel = preferredJobChannel(session)
	}

	if session.BotAlias == "" || session.Stage == stageAwaitingBotAlias {
		alias, ok := promptBotAlias(conv)
		if !ok {
			session.Stage = stageAwaitingBotAlias
			session.UpdatedAtUTC = nowUTC()
			persist("I couldn't save onboarding progress to %s")
			return
		}
		session.BotAlias = alias
		session.Stage = stageAwaitingJobChan
		session.UpdatedAtUTC = nowUTC()
		if !persist("I couldn't save onboarding progress to %s") {
			return
		}
	}

	if session.JobChannel == "" || session.Stage == stageAwaitingJobChan {
		channel, ok := promptJobChannel(conv, defaultJobChannel)
		if !ok {
			session.Stage = stageAwaitingJobChan
			session.UpdatedAtUTC = nowUTC()
			persist("I couldn't save onboarding progress to %s")
			return
		}
		session.JobChannel = channel
		session.Stage = stageAwaitingRobotEmail
		session.UpdatedAtUTC = nowUTC()
		if !persist("I couldn't save onboarding progress to %s") {
			return
		}
	}

	if session.RobotEmail == "" || session.Stage == stageAwaitingRobotEmail {
		email, ok := promptRobotEmail(conv)
		if !ok {
			session.Stage = stageAwaitingRobotEmail
			session.UpdatedAtUTC = nowUTC()
			persist("I couldn't save onboarding progress to %s")
			return
		}
		session.RobotEmail = email
		session.Stage = stageAwaitingAdminEmail
		session.UpdatedAtUTC = nowUTC()
		if !persist("I couldn't save onboarding progress to %s") {
			return
		}
	}

	if session.AdminEmail == "" || session.Stage == stageAwaitingAdminEmail {
		email, ok := promptAdminEmail(conv)
		if !ok {
			session.Stage = stageAwaitingAdminEmail
			session.UpdatedAtUTC = nowUTC()
			persist("I couldn't save onboarding progress to %s")
			return
		}
		session.AdminEmail = email
		session.Stage = stageAwaitingUsername
		session.UpdatedAtUTC = nowUTC()
		if !persist("I couldn't save onboarding progress to %s") {
			return
		}
	}

	if session.CanonicalUser == "" || session.Stage == stageAwaitingUsername {
		user, ok := promptCanonicalUser(conv, defaultUser)
		if !ok {
			session.Stage = stageAwaitingUsername
			session.UpdatedAtUTC = nowUTC()
			persist("I couldn't save onboarding progress to %s")
			return
		}
		session.CanonicalUser = user
		session.Stage = stageAwaitingSSHKey
		session.UpdatedAtUTC = nowUTC()
		if !persist("I couldn't save onboarding progress to %s") {
			return
		}
	}

	if session.SSHPublicKey == "" || session.Stage == stageAwaitingSSHKey {
		key, source, ok := resolveSSHPublicKey(conv)
		if !ok {
			session.Stage = stageAwaitingSSHKey
			session.UpdatedAtUTC = nowUTC()
			persist("I couldn't save onboarding progress to %s")
			return
		}
		session.SSHPublicKey = key
		session.SSHPublicKeySource = source
		session.Stage = stageAwaitingSSHKey
		session.UpdatedAtUTC = nowUTC()
		if !persist("I couldn't save onboarding progress to %s") {
			return
		}
	}

	if session.Stage == stageAwaitingSSHKey {
		if err := applyScaffold(conv.r, *session); err != nil {
			if errors.Is(err, errScaffoldExists) {
				conv.Reply("Scaffold already exists under '%s'; continuing with repository handoff.", defaultScaffoldPath)
			} else {
				conv.r.Log(robot.Error, "Applying scaffold for user '%s': %v", session.CanonicalUser, err)
				conv.Reply("I couldn't apply scaffold changes: %v", err)
				conv.Say("Your session is preserved. Fix the issue and reconnect with @%s so setup can continue automatically.", session.StartedBy)
				return
			}
		} else {
			conv.Reply("Scaffold created under '%s' and local identity configured for '%s'.", defaultScaffoldPath, session.CanonicalUser)
			conv.Say("Saved SSH server public key to '%s/robot-ssh.pub'.", defaultScaffoldPath)
			conv.Pause(0.5)
			conv.Say("The last setup step is git. We'll make sure this robot can bootstrap itself from a repository in a brand-new directory.")
		}
		session.Status = statusActive
		session.Stage = stageAwaitingRepoURL
		session.CompletedAtUTC = ""
		session.UpdatedAtUTC = nowUTC()
		if !persist("I couldn't save onboarding progress to %s") {
			return
		}
	}

	if session.Stage == stageScaffolded || session.Stage == stageAwaitingRepoURL || session.RepositoryURL == "" {
		repoURL, ok := promptRepositoryURL(conv, session.RepositoryURL)
		if !ok {
			session.Stage = stageAwaitingRepoURL
			session.UpdatedAtUTC = nowUTC()
			persist("I couldn't save onboarding progress to %s")
			return
		}
		session.RepositoryURL = repoURL
		session.Stage = stageAwaitingRepoURL
		session.UpdatedAtUTC = nowUTC()
		if !persist("I couldn't save onboarding progress to %s") {
			return
		}
	}

	if session.Stage == stageAwaitingRepoURL {
		deployPubKey, err := applyRepositoryHandoff(*session)
		if err != nil {
			conv.r.Log(robot.Error, "Applying repository handoff for user '%s': %v", session.CanonicalUser, err)
			conv.Reply("I couldn't finish repository handoff: %v", err)
			conv.Say("Your session is preserved. Reconnect with @%s after fixing the issue and setup will continue automatically.", session.StartedBy)
			return
		}
		session.DeployPublicKey = deployPubKey
		session.Stage = stageAwaitingGitPush
		session.UpdatedAtUTC = nowUTC()
		if !persist("I couldn't save onboarding progress to %s") {
			return
		}
		sendRepositoryInstructions(conv, *session)
	}

	if session.Stage == stageAwaitingGitPush {
		done, ok := promptGitPushComplete(conv)
		if !ok {
			session.Stage = stageAwaitingGitPush
			session.UpdatedAtUTC = nowUTC()
			persist("I couldn't save onboarding progress to %s")
			return
		}
		if !done {
			sendRepositoryInstructions(conv, *session)
			session.Stage = stageAwaitingGitPush
			session.UpdatedAtUTC = nowUTC()
			if !persist("I couldn't save onboarding progress to %s") {
				return
			}
			done, ok = promptGitPushComplete(conv)
			if !ok {
				session.Stage = stageAwaitingGitPush
				session.UpdatedAtUTC = nowUTC()
				persist("I couldn't save onboarding progress to %s")
				return
			}
			if !done {
				session.Stage = stageAwaitingGitPush
				session.UpdatedAtUTC = nowUTC()
				persist("I couldn't save onboarding progress to %s")
				return
			}
		}
	}

	session.Status = statusCompleted
	session.Stage = stageRepoReady
	session.CompletedAtUTC = nowUTC()
	session.UpdatedAtUTC = session.CompletedAtUTC
	if !persist("Repository handoff succeeded but I couldn't persist final state in %s") {
		return
	}
	conv.Reply("Beautiful. That gives me everything I need.")
	conv.Say("I'm doing the final restart now so the robot comes back with its real configuration, its real name, and its bootstrap settings already in place.")
	conv.Pause(0.5)
	conv.r.AddTask("restart-robot")
}

func promptEncryptionKey(conv *conversation) (string, bool) {
	conv.Say("Let's build your robot together. First we'll create the one secret every robot needs: `GOPHER_ENCRYPTION_KEY`.")
	conv.Say("This key protects secrets stored by the robot. It lives in the environment, usually a `.env` file, stays outside git, and is the one value you'll carry with you when you deploy the robot somewhere else.")
	for i := 0; i < 3; i++ {
		rep, ret := conv.Prompt("SimpleString", "Would you like me to generate a fresh key for you, or would you rather paste one you already have? (generate / supply)")
		switch ret {
		case robot.Interrupted:
			conv.Reply("Setup paused. Run 'new robot' when you're ready to continue.")
			return "", false
		case robot.TimeoutExpired:
			conv.Reply("Timed out waiting for the encryption-key choice. Run 'new robot' when you're ready.")
			return "", false
		case robot.Ok:
			choice := strings.ToLower(strings.TrimSpace(rep))
			switch choice {
			case "generate", "g":
				conv.Reply("Perfect. I'll generate one now and write it into the environment via `.env`.")
				key, err := randomAlphaNum(32)
				if err != nil {
					conv.r.Log(robot.Error, "Generating encryption key: %v", err)
					conv.Reply("I couldn't generate a fresh encryption key.")
					return "", false
				}
				return key, true
			case "supply", "supplied", "paste":
				key, ok := promptSuppliedEncryptionKey(conv)
				if !ok {
					return "", false
				}
				conv.Reply("Thanks - that looks valid, so I'll write it into the environment via `.env`.")
				return key, true
			default:
				conv.Reply("Please reply `generate` or `supply`.")
			}
		default:
			conv.Reply("I couldn't read your encryption-key choice (%s).", ret)
		}
	}
	conv.Reply("Too many invalid encryption-key attempts. Run 'new robot' to try again.")
	return "", false
}

func promptSuppliedEncryptionKey(conv *conversation) (string, bool) {
	for i := 0; i < 3; i++ {
		rep, ret := conv.Prompt("SimpleString", "Please paste the encryption key you'd like to use.")
		switch ret {
		case robot.Interrupted:
			conv.Reply("Setup paused. Run 'new robot' when you're ready to continue.")
			return "", false
		case robot.TimeoutExpired:
			conv.Reply("Timed out waiting for the encryption key. Run 'new robot' when you're ready.")
			return "", false
		case robot.Ok:
			key := strings.TrimSpace(rep)
			if validEncryptionKey(key) {
				return key, true
			}
			conv.Reply("That key doesn't look valid. Please provide a single 32-character key with no spaces.")
		default:
			conv.Reply("I couldn't read your encryption key (%s).", ret)
		}
	}
	conv.Reply("Too many invalid encryption keys. Run 'new robot' to try again.")
	return "", false
}

func validEncryptionKey(key string) bool {
	if strings.TrimSpace(key) != key {
		return false
	}
	return len(key) == 32 && !strings.ContainsAny(key, " \t\r\n")
}

func promptBotName(conv *conversation) (string, bool) {
	conv.Say("The robot's name is the given name your robot will recognize.")
	conv.Say("For maximum compatibility and portability across chat platforms, the robot will also look for messages addressed to it, for example 'floyd, ping'.")
	for i := 0; i < 3; i++ {
		rep, ret := conv.Prompt("botname", "What name would you like to give your robot?")
		switch ret {
		case robot.Interrupted:
			conv.Reply("Setup paused. Run 'new robot' when you're ready to continue.")
			return "", false
		case robot.TimeoutExpired:
			conv.Reply("Timed out waiting for robot name. Run 'new robot' when you're ready.")
			return "", false
		case robot.Ok:
			// use provided response
		default:
			conv.Reply("I couldn't read your robot name response (%s).", ret)
			continue
		}
		name := canonicalBotName(rep)
		if botNameRe.MatchString(name) {
			return name, true
		}
		conv.Reply("'%s' isn't valid. Use lowercase letters, digits, '_' or '-', starting with a letter.", strings.TrimSpace(rep))
	}
	conv.Reply("Too many invalid robot name attempts. Run 'new robot' to try again.")
	return "", false
}

func promptBotAlias(conv *conversation) (string, bool) {
	conv.Say("Your robot alias is a one-character shorthand name for concise commands, for example ';ping'.")
	conv.Say("Choose one character from ! ; - %% ~ * + ^ $ ? [ ] { } or \\.")
	for i := 0; i < 3; i++ {
		rep, ret := conv.Prompt("botalias", "What one-character alias should your robot use?")
		switch ret {
		case robot.Interrupted:
			conv.Reply("Setup paused. Run 'new robot' when you're ready to continue.")
			return "", false
		case robot.TimeoutExpired:
			conv.Reply("Timed out waiting for robot alias. Run 'new robot' when you're ready.")
			return "", false
		case robot.Ok:
			// use provided response
		default:
			conv.Reply("I couldn't read your alias response (%s).", ret)
			continue
		}
		alias := canonicalBotAlias(rep)
		if validBotAlias(alias) {
			return alias, true
		}
		conv.Reply("'%s' isn't a supported alias. Choose one character from ! ; - %% ~ * + ^ $ ? [ ] { } or \\\\.", strings.TrimSpace(rep))
	}
	conv.Reply("Too many invalid alias attempts. Run 'new robot' to try again.")
	return "", false
}

func promptJobChannel(conv *conversation, fallback string) (string, bool) {
	fallback = canonicalChannelName(fallback)
	if !channelRe.MatchString(fallback) {
		fallback = "general"
	}
	conv.Say("Your robot may run scheduled jobs periodically, for example to rotate logs or perform maintenance.")
	conv.Say("Any output from these jobs goes to a default job channel. A common convention is '<robotname>-jobs'.")
	for i := 0; i < 3; i++ {
		rep, ret := conv.Prompt("jobchannel",
			"What channel should receive scheduled job status messages? Suggested '%s'; reply '=' to use suggested.",
			fallback)
		switch ret {
		case robot.Interrupted:
			conv.Reply("Setup paused. Run 'new robot' when you're ready to continue.")
			return "", false
		case robot.TimeoutExpired:
			conv.Reply("Timed out waiting for job channel. Run 'new robot' when you're ready.")
			return "", false
		case robot.UseDefaultValue:
			rep = fallback
		case robot.Ok:
			// use provided response
		default:
			conv.Reply("I couldn't read your job channel response (%s).", ret)
			continue
		}
		channel := canonicalChannelName(rep)
		if channelRe.MatchString(channel) {
			return channel, true
		}
		conv.Reply("'%s' isn't valid. Use letters/digits with optional '-' or '_', and no spaces.", strings.TrimSpace(rep))
	}
	conv.Reply("Too many invalid job channel attempts. Run 'new robot' to try again.")
	return "", false
}

func promptRobotEmail(conv *conversation) (string, bool) {
	for i := 0; i < 3; i++ {
		rep, ret := conv.Prompt("Email", "What email address should the robot use for its own identity? If you don't have a dedicated one yet, your own address is fine for now.")
		switch ret {
		case robot.Interrupted:
			conv.Reply("Setup paused. Run 'new robot' when you're ready to continue.")
			return "", false
		case robot.TimeoutExpired:
			conv.Reply("Timed out waiting for the robot email. Run 'new robot' when you're ready.")
			return "", false
		case robot.Ok:
			email := strings.TrimSpace(rep)
			if email != "" {
				return email, true
			}
			conv.Reply("Please provide a valid email address.")
		default:
			conv.Reply("I couldn't read your robot email response (%s).", ret)
		}
	}
	conv.Reply("Too many invalid robot email attempts. Run 'new robot' to try again.")
	return "", false
}

func promptAdminEmail(conv *conversation) (string, bool) {
	for i := 0; i < 3; i++ {
		rep, ret := conv.Prompt("Email", "And what email address should the robot advertise for its administrator? This is what people may see in help or info output.")
		switch ret {
		case robot.Interrupted:
			conv.Reply("Setup paused. Run 'new robot' when you're ready to continue.")
			return "", false
		case robot.TimeoutExpired:
			conv.Reply("Timed out waiting for the administrator email. Run 'new robot' when you're ready.")
			return "", false
		case robot.Ok:
			email := strings.TrimSpace(rep)
			if email != "" {
				return email, true
			}
			conv.Reply("Please provide a valid email address.")
		default:
			conv.Reply("I couldn't read your administrator email response (%s).", ret)
		}
	}
	conv.Reply("Too many invalid administrator email attempts. Run 'new robot' to try again.")
	return "", false
}

func promptCanonicalUser(conv *conversation, fallback string) (string, bool) {
	if fallback == "" {
		fallback = "alice"
	}
	for i := 0; i < 3; i++ {
		rep, ret := conv.Prompt("username",
			"What username do you want to use with your robot for local ssh login? (bot-ssh <username>) For team-chat robots, use your team-chat username. Default '%s'; reply '=' to use default.",
			fallback)
		switch ret {
		case robot.Interrupted:
			conv.Reply("Setup paused. Run 'new robot' when you're ready to continue.")
			return "", false
		case robot.TimeoutExpired:
			conv.Reply("Timed out waiting for username. Run 'new robot' when you're ready.")
			return "", false
		case robot.UseDefaultValue:
			rep = fallback
		case robot.Ok:
			// use provided response
		default:
			conv.Reply("I couldn't read your username response (%s).", ret)
			continue
		}
		candidate := canonicalUserKey(rep)
		if usernameRe.MatchString(candidate) {
			return candidate, true
		}
		conv.Reply("'%s' isn't valid. Use lowercase letters, digits, '_' or '-', starting with a letter.", strings.TrimSpace(rep))
	}
	conv.Reply("Too many invalid username attempts. Run 'new robot' to try again.")
	return "", false
}

func resolveSSHPublicKey(conv *conversation) (string, string, bool) {
	if key, source, ok := detectLocalSSHPublicKey(); ok {
		rep, ret := conv.Prompt("YesNo", "Detected local SSH public key: %s, use that one? (y|n)", source)
		switch ret {
		case robot.Interrupted:
			conv.Reply("Setup paused. Run 'new robot' when you're ready to continue.")
			return "", "", false
		case robot.TimeoutExpired:
			conv.Reply("Timed out waiting for SSH key confirmation. Run 'new robot' when you're ready.")
			return "", "", false
		case robot.Ok:
			v := strings.ToLower(strings.TrimSpace(rep))
			if v == "y" || v == "yes" {
				return key, source, true
			}
			if v != "n" && v != "no" {
				conv.Reply("Please answer y or n.")
				return "", "", false
			}
		default:
			conv.Reply("I couldn't read your SSH key confirmation (%s).", ret)
			return "", "", false
		}
	}

	for i := 0; i < 3; i++ {
		rep, ret := conv.Prompt("sshpubkey", "Paste the SSH public key line to use for local login (e.g. 'ssh-ed25519 AAAA...').")
		switch ret {
		case robot.Interrupted:
			conv.Reply("Setup paused. Run 'new robot' when you're ready to continue.")
			return "", "", false
		case robot.TimeoutExpired:
			conv.Reply("Timed out waiting for SSH key. Run 'new robot' when you're ready.")
			return "", "", false
		case robot.Ok:
			key := normalizeSSHPublicKey(rep)
			if !sshPubKeyRe.MatchString(key) {
				conv.Reply("That doesn't look like a valid SSH public key line.")
				continue
			}
			return key, "prompt", true
		default:
			conv.Reply("I couldn't read your SSH key response (%s).", ret)
		}
	}
	conv.Reply("Too many invalid SSH key attempts. Run 'new robot' to try again.")
	return "", "", false
}

func promptRepositoryURL(conv *conversation, current string) (string, bool) {
	defaultRepo := strings.TrimSpace(current)
	if defaultRepo == "" || defaultRepo == defaultCustomRepository {
		defaultRepo = ""
	}
	conv.Say("The standard workflow is to store robot configuration and scripts in a single git repository for bootstrap and deployment.")
	conv.Say("This value should be a clone URL using SSH credentials. An empty repository is recommended.")
	prompt := "Let's get this robot ready for the first deployment - what's the repository clone URL? (e.g. 'git@github.com:owner/repo.git')"
	if defaultRepo != "" {
		prompt = fmt.Sprintf("%s Reply '=' to keep '%s'.", prompt, defaultRepo)
	}
	for i := 0; i < 3; i++ {
		rep, ret := conv.Prompt("repourl", prompt)
		switch ret {
		case robot.Interrupted:
			conv.Reply("Repository handoff paused. Reconnect after the restart and setup will continue automatically.")
			return "", false
		case robot.TimeoutExpired:
			conv.Reply("Timed out waiting for repository URL. Reconnect and setup will continue automatically.")
			return "", false
		case robot.UseDefaultValue:
			if defaultRepo == "" {
				conv.Reply("No default repository is available yet.")
				continue
			}
			return defaultRepo, true
		case robot.Ok:
			repo := strings.TrimSpace(rep)
			if validRepositoryURL(repo) {
				return repo, true
			}
			conv.Reply("That doesn't look like a supported clone URL.")
		default:
			conv.Reply("I couldn't read your repository response (%s).", ret)
		}
	}
	conv.Reply("Too many invalid repository attempts. Reconnect and setup will continue automatically.")
	return "", false
}

func promptGitPushComplete(conv *conversation) (done bool, ok bool) {
	for i := 0; i < 3; i++ {
		rep, ret := conv.Prompt("SimpleString", "Reply `done` once the push succeeds. If you'd like me to repeat the instructions, just say `repeat`.")
		switch ret {
		case robot.Interrupted:
			conv.Reply("Setup paused. Reconnect and setup will continue automatically.")
			return false, false
		case robot.TimeoutExpired:
			conv.Reply("Timed out waiting for git-push confirmation. Reconnect and setup will continue automatically.")
			return false, false
		case robot.Ok:
			switch strings.ToLower(strings.TrimSpace(rep)) {
			case "done":
				return true, true
			case "repeat":
				return false, true
			default:
				conv.Reply("Please reply `done` or `repeat`.")
			}
		default:
			conv.Reply("I couldn't read your git-push confirmation (%s).", ret)
		}
	}
	conv.Reply("Too many invalid git-push replies. Reconnect and setup will continue automatically.")
	return false, false
}

func sendRepositoryInstructions(conv *conversation, session setupSession) {
	conv.Reply("Repository handoff is ready. Updated `.env` with `GOPHER_CUSTOM_REPOSITORY` and `GOPHER_DEPLOY_KEY`.")
	conv.Say("Add this read-only deploy key to your repository:")
	conv.FixedSay("%s", strings.TrimSpace(session.DeployPublicKey))
	conv.Say("Then, from the '%s' directory, run:", defaultScaffoldPath)
	conv.FixedSay("git init\ngit add .\ngit branch -m main\ngit commit -m \"Initial robot scaffold\"\ngit remote add origin %s\ngit push -u origin main", session.RepositoryURL)
}

func sendFinalBootstrapInstructions(conv *conversation, session setupSession) {
	conv.Pause(0.5)
	conv.Say("Welcome back, @%s. I'm now running with your full robot configuration.", conv.user)
	conv.Pause(0.5)
	conv.Say("Your working directory is ready to keep using as the source repo checkout. Now let's verify the bootstrap path the way a fresh deployment would.")
	conv.Pause(0.5)
	conv.Say("Create a brand-new empty directory somewhere else, copy only `.env` into it, and start `gopherbot` there.")
	conv.Pause(0.5)
	if strings.TrimSpace(session.RepositoryURL) != "" {
		conv.Say("On first start, Gopherbot should read `.env`, clone `%s`, build out `custom/`, and restart itself into the same fully configured robot.", session.RepositoryURL)
	} else {
		conv.Say("On first start, Gopherbot should read `.env`, clone your configured repository, build out `custom/`, and restart itself into the same fully configured robot.")
	}
	conv.Pause(0.5)
	conv.Say("If that works, you're done - you now have a robot that can be deployed by carrying only `.env` into an empty directory, or by starting the gopherbot engine in a new empty directory with the required environment variables already set.")
}

func validRepositoryURL(repo string) bool {
	repo = strings.TrimSpace(repo)
	if repo == "" || repo == defaultCustomRepository {
		return false
	}
	if strings.HasPrefix(repo, "git@") {
		return strings.Contains(repo, ":")
	}
	if strings.HasPrefix(repo, "ssh://") || strings.HasPrefix(repo, "https://") || strings.HasPrefix(repo, "http://") {
		return true
	}
	if strings.HasPrefix(repo, "/") || strings.HasPrefix(repo, "./") || strings.HasPrefix(repo, "../") {
		return true
	}
	return false
}

func applyRepositoryHandoff(s setupSession) (deployPubKey string, err error) {
	repo := strings.TrimSpace(s.RepositoryURL)
	if !validRepositoryURL(repo) {
		return "", fmt.Errorf("invalid repository URL '%s'", repo)
	}
	deployPrivatePEM, deployPub, err := generateDeployKeyPair(robotMetaFromSession(s).botName)
	if err != nil {
		return "", fmt.Errorf("generating deploy keypair: %w", err)
	}
	deployEncoded := encodeDeployKeyForEnv(deployPrivatePEM)
	if deployEncoded == "" {
		return "", fmt.Errorf("generated deploy key is empty")
	}
	deployPubKey = strings.TrimSpace(deployPub)
	if deployPubKey == "" {
		return "", fmt.Errorf("generated deploy public key is empty")
	}
	deployPublicPath := filepath.Join(defaultScaffoldPath, "ssh", "deploy_key.pub")
	if err := writePublicKey(deployPublicPath, deployPubKey); err != nil {
		return "", err
	}

	if err := writeOrUpdateEnvRepository(repo, deployEncoded); err != nil {
		return "", err
	}
	return deployPubKey, nil
}

func encodeDeployKeyForEnv(privateKey string) string {
	k := strings.TrimSpace(privateKey)
	if k == "" {
		return ""
	}
	k = strings.ReplaceAll(k, "\r\n", "\n")
	k = strings.ReplaceAll(k, " ", "_")
	k = strings.ReplaceAll(k, "\n", ":")
	return k
}

func applyScaffold(r robot.Robot, s setupSession) error {
	robotConf := filepath.Join(defaultScaffoldPath, "conf", "robot.yaml")
	if _, err := os.Stat(robotConf); err == nil {
		return errScaffoldExists
	}

	installDir := strings.TrimSpace(os.Getenv("GOPHER_INSTALLDIR"))
	if installDir == "" {
		if exePath, exErr := os.Executable(); exErr == nil {
			installDir = filepath.Dir(exePath)
		}
	}
	if installDir == "" {
		return fmt.Errorf("GOPHER_INSTALLDIR is not set and executable path could not be determined")
	}

	skelRoot := filepath.Join(installDir, "robot.skel")
	if info, err := os.Stat(skelRoot); err != nil {
		return fmt.Errorf("robot skeleton not found at %s: %w", skelRoot, err)
	} else if !info.IsDir() {
		return fmt.Errorf("robot skeleton path is not a directory: %s", skelRoot)
	}

	if err := copyTreeNoOverwrite(skelRoot, defaultScaffoldPath); err != nil {
		return fmt.Errorf("copying robot skeleton: %w", err)
	}

	meta := robotMetaFromSession(s)
	hostPrivatePEM, hostPubKey, err := generateDeployKeyPair(meta.botName)
	if err != nil {
		return fmt.Errorf("generating SSH host keypair: %w", err)
	}
	hostKeyTemplateLiteral, err := jsonString(hostPrivatePEM)
	if err != nil {
		return fmt.Errorf("encoding SSH host private key for template: %w", err)
	}
	hostKeyEncrypted, ret := r.EncryptSecret(hostKeyTemplateLiteral)
	if ret != robot.Ok {
		return fmt.Errorf("encrypting SSH host private key via EncryptSecret returned %s", ret)
	}

	replace := map[string]string{
		"<botname>":             meta.botName,
		"<botemail>":            meta.botEmail,
		"<botfullname>":         meta.botFullName,
		"<botalias>":            meta.botAlias,
		"<sshhostkeyencrypted>": hostKeyEncrypted,
	}

	for _, rel := range []string{
		"conf/robot.yaml",
		"conf/protocols/ssh.yaml",
		"conf/protocols/terminal.yaml",
		"conf/protocols/slack.yaml",
		"git/config",
	} {
		fp := filepath.Join(defaultScaffoldPath, rel)
		if err := replaceTokensInFile(fp, replace); err != nil {
			return err
		}
	}
	if err := enableOnboardingHooks(filepath.Join(defaultScaffoldPath, "conf", "robot.yaml")); err != nil {
		return err
	}
	if err := writePublicKey(filepath.Join(defaultScaffoldPath, "robot-ssh.pub"), hostPubKey); err != nil {
		return fmt.Errorf("writing robot ssh public key: %w", err)
	}

	if err := appendIdentityConfig(
		filepath.Join(defaultScaffoldPath, "conf", "robot.yaml"),
		filepath.Join(defaultScaffoldPath, "conf", "protocols", "ssh.yaml"),
		meta,
		s.CanonicalUser,
		s.SSHPublicKey,
	); err != nil {
		return err
	}

	return nil
}

type generatedMeta struct {
	botName         string
	botEmail        string
	botFullName     string
	botAlias        string
	jobChannel      string
	userEmail       string
	userDisplayName string
	userFirstName   string
}

func robotMetaFromSession(s setupSession) generatedMeta {
	clean := canonicalUserKey(s.CanonicalUser)
	if clean == "" {
		clean = "alice"
	}
	botName := canonicalBotName(s.BotName)
	if !botNameRe.MatchString(botName) {
		botName = "floyd"
	}
	botAlias := canonicalBotAlias(s.BotAlias)
	botLabel := strings.Title(strings.ReplaceAll(botName, "_", " "))
	botLabel = strings.Title(strings.ReplaceAll(botLabel, "-", " "))
	botParts := strings.Fields(botLabel)
	botShort := "Robot"
	if len(botParts) > 0 {
		botShort = botParts[0]
	}
	userLabel := strings.Title(strings.ReplaceAll(clean, "_", " "))
	userLabel = strings.Title(strings.ReplaceAll(userLabel, "-", " "))
	userParts := strings.Fields(userLabel)
	userShort := "User"
	if len(userParts) > 0 {
		userShort = userParts[0]
	}
	botEmail := strings.TrimSpace(s.RobotEmail)
	if botEmail == "" {
		botEmail = fmt.Sprintf("%s@example.com", botName)
	}
	adminEmail := strings.TrimSpace(s.AdminEmail)
	if adminEmail == "" {
		adminEmail = fmt.Sprintf("%s@example.com", clean)
	}
	return generatedMeta{
		botName:         botName,
		botEmail:        botEmail,
		botFullName:     fmt.Sprintf("%s Gopherbot", botShort),
		botAlias:        botAlias,
		jobChannel:      s.JobChannel,
		userEmail:       adminEmail,
		userDisplayName: fmt.Sprintf("%s User", userShort),
		userFirstName:   userShort,
	}
}

func appendIdentityConfig(robotConfig, sshConfig string, meta generatedMeta, user, sshKey string) error {
	escapedUser := yamlDoubleQuoteEscape(user)
	escapedMail := yamlDoubleQuoteEscape(meta.userEmail)
	escapedFull := yamlDoubleQuoteEscape(meta.userDisplayName)
	escapedFirst := yamlDoubleQuoteEscape(meta.userFirstName)
	jobChannel := canonicalChannelName(meta.jobChannel)
	if !channelRe.MatchString(jobChannel) {
		jobChannel = "general"
	}
	escapedJobChannel := yamlDoubleQuoteEscape(jobChannel)
	channelList := yamlQuotedList(uniqueChannels("general", "random", jobChannel))

	if err := ensureSSHProtocolChannels(sshConfig, []string{"general", "random", jobChannel}); err != nil {
		return fmt.Errorf("updating %s protocol channels: %w", sshConfig, err)
	}
	if err := ensureSSHProtocolUserKey(sshConfig, user, sshKey); err != nil {
		return fmt.Errorf("updating %s protocol users: %w", sshConfig, err)
	}

	robotBlock := fmt.Sprintf(`
# Added by new-robot onboarding
AdminContact: "%s"
AdminUsers: [ "%s" ]
DefaultChannels: [ %s ]
DefaultJobChannel: %s
UserRoster:
- UserName: "%s"
  Email: "%s"
  FullName: "%s"
  FirstName: "%s"
  LastName: "User"
`, yamlDoubleQuoteEscape(meta.userEmail), escapedUser, channelList, escapedJobChannel, escapedUser, escapedMail, escapedFull, escapedFirst)
	if err := appendFile(robotConfig, robotBlock); err != nil {
		return fmt.Errorf("updating %s: %w", robotConfig, err)
	}

	return nil
}

func writeInitialEnv(encryptionKey string) error {
	path := ".env"
	original, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("reading .env: %w", err)
	}

	required := map[string]string{
		"GOPHER_ENCRYPTION_KEY": encryptionKey,
	}

	lines := []string{}
	if len(original) > 0 {
		lines = strings.Split(strings.ReplaceAll(string(original), "\r\n", "\n"), "\n")
	}
	lines = stripSetupPlaceholderLines(lines)

	seen := map[string]bool{}
	for i, line := range lines {
		key, _, ok := parseEnvLine(line)
		if !ok {
			if strings.HasPrefix(strings.TrimSpace(line), "GOPHER_CUSTOM_BRANCH=") {
				lines[i] = ""
			}
			continue
		}
		if val, shouldSet := required[key]; shouldSet {
			lines[i] = fmt.Sprintf("%s=%s", key, val)
			seen[key] = true
			continue
		}
		if key == "GOPHER_CUSTOM_REPOSITORY" || key == "GOPHER_DEPLOY_KEY" || key == "GOPHER_CUSTOM_BRANCH" || key == "GOPHER_PROTOCOL" || key == "GOPHER_BRAIN" || key == "GOPHER_DEFAULT_PROTOCOL" || key == "GOPHER_BOTNAME" {
			lines[i] = ""
			continue
		}
		// Explicitly setting development is redundant because startup defaults
		// to development when GOPHER_ENVIRONMENT is not set.
		if key == "GOPHER_ENVIRONMENT" && strings.EqualFold(strings.TrimSpace(strings.Trim(valueForEnvLine(line), `"'`)), defaultEnvironment) {
			lines[i] = ""
		}
	}

	for key, val := range required {
		if seen[key] {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s=%s", key, val))
	}
	lines = ensureEnvironmentGuidanceComment(lines)
	lines = compactLines(lines)
	content := strings.TrimRight(strings.Join(lines, "\n"), "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		return fmt.Errorf("writing .env: %w", err)
	}
	return nil
}

func clearOnboardingScaffoldState() error {
	if err := os.RemoveAll(defaultScaffoldPath); err != nil {
		return fmt.Errorf("removing %s: %w", defaultScaffoldPath, err)
	}
	return nil
}

func writeOrUpdateEnvRepository(repositoryURL, deployKey string) error {
	path := ".env"
	original, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("reading .env: %w", err)
	}

	required := map[string]string{
		"GOPHER_CUSTOM_REPOSITORY": strings.TrimSpace(repositoryURL),
		"GOPHER_DEPLOY_KEY":        strings.TrimSpace(deployKey),
	}
	if required["GOPHER_CUSTOM_REPOSITORY"] == "" {
		return fmt.Errorf("repository URL is empty")
	}
	if required["GOPHER_DEPLOY_KEY"] == "" {
		return fmt.Errorf("deploy key is empty")
	}

	lines := []string{}
	if len(original) > 0 {
		lines = strings.Split(strings.ReplaceAll(string(original), "\r\n", "\n"), "\n")
	}
	lines = stripSetupPlaceholderLines(lines)

	newLines := make([]string, 0, len(lines)+len(required))
	seen := map[string]bool{}
	for _, line := range lines {
		key, _, ok := parseEnvLine(line)
		if !ok {
			trim := strings.TrimSpace(line)
			if strings.HasPrefix(trim, "GOPHER_CUSTOM_BRANCH=") {
				continue
			}
			newLines = append(newLines, line)
			continue
		}
		if val, shouldSet := required[key]; shouldSet {
			newLines = append(newLines, fmt.Sprintf("%s=%s", key, val))
			seen[key] = true
			continue
		}
		if key == "GOPHER_CUSTOM_BRANCH" || key == "GOPHER_BOTNAME" {
			continue
		}
		newLines = append(newLines, line)
	}
	for key, val := range required {
		if seen[key] {
			continue
		}
		newLines = append(newLines, fmt.Sprintf("%s=%s", key, val))
	}
	newLines = compactLines(newLines)
	content := strings.TrimRight(strings.Join(newLines, "\n"), "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		return fmt.Errorf("writing .env: %w", err)
	}
	return nil
}

func parseEnvLine(line string) (key, value string, ok bool) {
	trim := strings.TrimSpace(line)
	if trim == "" || strings.HasPrefix(trim, "#") {
		return "", "", false
	}
	i := strings.Index(trim, "=")
	if i <= 0 {
		return "", "", false
	}
	k := strings.TrimSpace(trim[:i])
	if !envKeyRe.MatchString(k) {
		return "", "", false
	}
	return k, strings.TrimSpace(trim[i+1:]), true
}

func valueForEnvLine(line string) string {
	_, value, ok := parseEnvLine(line)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

func ensureEnvironmentGuidanceComment(lines []string) []string {
	hasComment := false
	for _, line := range lines {
		if strings.Contains(line, "GOPHER_ENVIRONMENT=production") {
			hasComment = true
			break
		}
	}
	if hasComment {
		return lines
	}
	comment := []string{
		"# For a robot running in production, set:",
		"# GOPHER_ENVIRONMENT=production",
		"# (default is development)",
	}
	if len(lines) == 0 {
		return comment
	}
	out := make([]string, 0, len(lines)+len(comment)+1)
	out = append(out, lines...)
	if strings.TrimSpace(out[len(out)-1]) != "" {
		out = append(out, "")
	}
	out = append(out, comment...)
	return out
}

func stripSetupPlaceholderLines(lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		switch trim {
		case "# Optional for later remote bootstrap",
			"# GOPHER_DEPLOY_KEY=<set this in slice 3>",
			"# GOPHER_CUSTOM_BRANCH=.",
			"# GOPHER_ENVIRONMENT=development":
			continue
		default:
			out = append(out, line)
		}
	}
	return out
}

func compactLines(lines []string) []string {
	out := make([]string, 0, len(lines))
	prevBlank := false
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == "" {
			if prevBlank {
				continue
			}
			prevBlank = true
			out = append(out, "")
			continue
		}
		prevBlank = false
		out = append(out, line)
	}
	for len(out) > 0 && strings.TrimSpace(out[0]) == "" {
		out = out[1:]
	}
	for len(out) > 0 && strings.TrimSpace(out[len(out)-1]) == "" {
		out = out[:len(out)-1]
	}
	return out
}

func replaceTokensInFile(path string, replacements map[string]string) error {
	body, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("reading %s: %w", path, err)
	}
	txt := string(body)
	for token, repl := range replacements {
		txt = strings.ReplaceAll(txt, token, repl)
	}
	if err := os.WriteFile(path, []byte(txt), 0600); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

func enableOnboardingHooks(robotConfigPath string) error {
	body, err := os.ReadFile(robotConfigPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", robotConfigPath, err)
	}
	txt := string(body)
	if strings.Contains(txt, onboardingJobBeginMarker) {
		return nil
	}

	jobBlock := strings.TrimRight(`
  # BEGIN NEW-ROBOT ONBOARDING JOB
  "resume-setup":
    Description: Temporary onboarding resume-on-join job
    Path: jobs/go-resume-setup/resume_setup.go
    Homed: true
  # END NEW-ROBOT ONBOARDING JOB
`, "\n")

	updated, err := insertBlockAfterLine(txt, "ExternalJobs:", jobBlock)
	if err != nil {
		return fmt.Errorf("adding onboarding resume job to %s: %w", robotConfigPath, err)
	}

	if err := os.WriteFile(robotConfigPath, []byte(updated), 0600); err != nil {
		return fmt.Errorf("writing %s: %w", robotConfigPath, err)
	}
	return nil
}

func insertBlockAfterLine(text, anchor, block string) (string, error) {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) != anchor {
			continue
		}
		blockLines := strings.Split(strings.TrimRight(block, "\n"), "\n")
		updated := make([]string, 0, len(lines)+len(blockLines))
		updated = append(updated, lines[:i+1]...)
		updated = append(updated, blockLines...)
		updated = append(updated, lines[i+1:]...)
		return strings.Join(updated, "\n"), nil
	}
	return "", fmt.Errorf("anchor not found: %s", anchor)
}

func appendFile(path, block string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(block)
	return err
}

func copyTreeNoOverwrite(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return os.MkdirAll(dst, 0755)
		}
		if rel == "README.md" {
			return nil
		}
		target := filepath.Join(dst, rel)
		info, err := d.Info()
		if err != nil {
			return err
		}
		if d.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		if _, err := os.Stat(target); err == nil {
			return fmt.Errorf("destination file already exists: %s", target)
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return copyFile(path, target, info.Mode().Perm())
	})
}

func copyFile(src, dst string, mode fs.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return nil
}

func generateDeployKeyPair(comment string) (privatePEM string, publicLine string, err error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("generating ed25519 deploy key: %w", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return "", "", fmt.Errorf("encoding deploy private key: %w", err)
	}
	block := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: der,
	}
	privatePEM = string(pem.EncodeToMemory(block))
	publicLine, err = formatSSHEd25519PublicKey(pub, comment)
	if err != nil {
		return "", "", fmt.Errorf("formatting deploy public key: %w", err)
	}
	return privatePEM, publicLine, nil
}

func writePublicKey(path, pubKey string) error {
	pubKey = strings.TrimSpace(pubKey)
	if pubKey == "" {
		return fmt.Errorf("public key is empty")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("creating deploy key directory for %s: %w", path, err)
	}
	if err := os.WriteFile(path, []byte(pubKey+"\n"), 0644); err != nil {
		return fmt.Errorf("writing public key %s: %w", path, err)
	}
	return nil
}

func jsonString(v string) (string, error) {
	b, err := json.Marshal(strings.TrimSpace(v))
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func formatSSHEd25519PublicKey(pub ed25519.PublicKey, comment string) (string, error) {
	if len(pub) != ed25519.PublicKeySize {
		return "", fmt.Errorf("ed25519 public key is invalid length %d", len(pub))
	}
	buf := bytes.NewBuffer(nil)
	writeSSHString(buf, []byte("ssh-ed25519"))
	writeSSHString(buf, []byte(pub))

	key := "ssh-ed25519 " + base64.StdEncoding.EncodeToString(buf.Bytes())
	if trimmed := strings.TrimSpace(comment); trimmed != "" {
		key += " " + trimmed
	}
	return key, nil
}

func writeSSHString(buf *bytes.Buffer, b []byte) {
	_ = binary.Write(buf, binary.BigEndian, uint32(len(b)))
	buf.Write(b)
}

func detectLocalSSHPublicKey() (key, source string, ok bool) {
	home := os.Getenv("HOME")
	if home == "" {
		home = os.Getenv("USERPROFILE")
	}
	if home == "" {
		return "", "", false
	}
	candidates := []string{
		filepath.Join(home, ".ssh", "id_ed25519.pub"),
		filepath.Join(home, ".ssh", "id_ecdsa.pub"),
		filepath.Join(home, ".ssh", "id_rsa.pub"),
		filepath.Join(home, ".ssh", "id_dsa.pub"),
	}
	for _, fp := range candidates {
		body, err := os.ReadFile(fp)
		if err != nil {
			continue
		}
		lines := strings.Split(strings.ReplaceAll(string(body), "\r\n", "\n"), "\n")
		for _, line := range lines {
			line = normalizeSSHPublicKey(line)
			if sshPubKeyRe.MatchString(line) {
				return line, fp, true
			}
		}
	}
	return "", "", false
}

func normalizeSSHPublicKey(key string) string {
	parts := strings.Fields(strings.TrimSpace(key))
	if len(parts) == 0 {
		return ""
	}
	if len(parts) == 1 {
		return parts[0]
	}
	if len(parts) == 2 {
		return parts[0] + " " + parts[1]
	}
	return parts[0] + " " + parts[1] + " " + strings.Join(parts[2:], " ")
}

func shortSSHKeySummary(key string) string {
	parts := strings.Fields(key)
	if len(parts) < 2 {
		return "(unknown key)"
	}
	blob := parts[1]
	if len(blob) > 16 {
		blob = blob[:16] + "..."
	}
	return parts[0] + " " + blob
}

func randomAlphaNum(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("invalid length %d", length)
	}
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	buf := make([]byte, length)
	idx := make([]byte, length)
	if _, err := rand.Read(idx); err != nil {
		return "", err
	}
	for i := 0; i < length; i++ {
		buf[i] = chars[int(idx[i])%len(chars)]
	}
	return string(buf), nil
}

func yamlDoubleQuoteEscape(v string) string {
	v = strings.ReplaceAll(v, `\\`, `\\\\`)
	v = strings.ReplaceAll(v, `"`, `\\"`)
	return v
}

func canonicalUserKey(user string) string {
	return strings.ToLower(strings.TrimSpace(user))
}

func canonicalBotName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func validBotAlias(alias string) bool {
	if len(alias) != 1 {
		return false
	}
	return strings.ContainsAny(alias, `!;-%~*+^$?[]{}\`)
}

func canonicalBotAlias(alias string) string {
	alias = strings.TrimSpace(alias)
	if validBotAlias(alias) {
		return alias
	}
	return defaultBotAlias
}

func canonicalChannelName(channel string) string {
	channel = strings.TrimSpace(channel)
	channel = strings.TrimPrefix(channel, "#")
	return strings.ToLower(channel)
}

func preferredJobChannel(session *setupSession) string {
	if ch := canonicalChannelName(session.JobChannel); channelRe.MatchString(ch) {
		return ch
	}
	botName := canonicalBotName(session.BotName)
	if !botNameRe.MatchString(botName) {
		botName = "floyd"
	}
	suggested := botName + "-jobs"
	if channelRe.MatchString(suggested) {
		return suggested
	}
	return "general"
}

func uniqueChannels(values ...string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]bool)
	for _, raw := range values {
		ch := canonicalChannelName(raw)
		if !channelRe.MatchString(ch) || seen[ch] {
			continue
		}
		seen[ch] = true
		out = append(out, ch)
	}
	return out
}

func yamlQuotedList(values []string) string {
	if len(values) == 0 {
		return `"general", "random"`
	}
	quoted := make([]string, 0, len(values))
	for _, v := range values {
		quoted = append(quoted, fmt.Sprintf(`"%s"`, yamlDoubleQuoteEscape(v)))
	}
	return strings.Join(quoted, ", ")
}

func ensureSSHProtocolChannels(sshConfigPath string, channels []string) error {
	body, err := os.ReadFile(sshConfigPath)
	if err != nil {
		return err
	}
	lines := strings.Split(strings.ReplaceAll(string(body), "\r\n", "\n"), "\n")
	channelLine := fmt.Sprintf("  Channels: [ %s ]", yamlQuotedList(uniqueChannels(channels...)))

	foundProtocol := false
	inProtocol := false
	inserted := false
	out := make([]string, 0, len(lines)+1)
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == "ProtocolConfig:" {
			foundProtocol = true
			inProtocol = true
			out = append(out, line)
			continue
		}
		if inProtocol {
			if trim != "" && !strings.HasPrefix(line, "  ") {
				if !inserted {
					out = append(out, channelLine)
					inserted = true
				}
				inProtocol = false
			} else if strings.HasPrefix(trim, "Channels:") {
				continue
			}
		}
		out = append(out, line)
		if inProtocol && strings.HasPrefix(trim, "DefaultChannel:") {
			out = append(out, channelLine)
			inserted = true
		}
	}
	if inProtocol && !inserted {
		out = append(out, channelLine)
		inserted = true
	}
	if !foundProtocol {
		return fmt.Errorf("ProtocolConfig block not found")
	}
	if !inserted {
		return fmt.Errorf("could not set ProtocolConfig Channels")
	}
	content := strings.TrimRight(strings.Join(out, "\n"), "\n") + "\n"
	return os.WriteFile(sshConfigPath, []byte(content), 0600)
}

func ensureSSHProtocolUserKey(sshConfigPath, user, sshKey string) error {
	user = canonicalUserKey(user)
	sshKey = strings.TrimSpace(sshKey)
	if !usernameRe.MatchString(user) {
		return fmt.Errorf("invalid ssh username %q", user)
	}
	if !sshPubKeyRe.MatchString(sshKey) {
		return fmt.Errorf("invalid ssh public key")
	}
	body, err := os.ReadFile(sshConfigPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", sshConfigPath, err)
	}
	lines := strings.Split(strings.ReplaceAll(string(body), "\r\n", "\n"), "\n")
	userBlock := []string{
		"  UserKeys:",
		fmt.Sprintf("  - UserName: %q", user),
		"    PublicKeys:",
		fmt.Sprintf("    - %q", sshKey),
	}

	inProtocol := false
	replaced := false
	out := make([]string, 0, len(lines)+len(userBlock))
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trim := strings.TrimSpace(line)
		if trim == "ProtocolConfig:" {
			inProtocol = true
			out = append(out, line)
			continue
		}
		if inProtocol && trim != "" && !strings.HasPrefix(line, "  ") {
			if !replaced {
				out = append(out, userBlock...)
				replaced = true
			}
			inProtocol = false
		}
		if inProtocol && strings.HasPrefix(trim, "UserKeys:") {
			out = append(out, userBlock...)
			replaced = true
			for i+1 < len(lines) {
				next := lines[i+1]
				nextTrim := strings.TrimSpace(next)
				if nextTrim == "" {
					i++
					continue
				}
				if strings.HasPrefix(next, "  - ") || strings.HasPrefix(next, "    ") {
					i++
					continue
				}
				break
			}
			continue
		}
		out = append(out, line)
	}
	if inProtocol && !replaced {
		out = append(out, userBlock...)
		replaced = true
	}
	if !replaced {
		return fmt.Errorf("ProtocolConfig block not found")
	}
	content := strings.TrimRight(strings.Join(out, "\n"), "\n") + "\n"
	return os.WriteFile(sshConfigPath, []byte(content), 0600)
}

func preferredBotName(r robot.Robot, session *setupSession) string {
	candidates := []string{
		session.BotName,
		r.GetParameter("GOPHER_BOTNAME"),
		r.GetBotAttribute("name").String(),
		"floyd",
	}
	for _, raw := range candidates {
		name := canonicalBotName(raw)
		if botNameRe.MatchString(name) {
			return name
		}
	}
	return "floyd"
}

func preferredBotAlias(r robot.Robot, session *setupSession) string {
	candidates := []string{
		session.BotAlias,
		r.GetParameter("GOPHER_ALIAS"),
		r.GetBotAttribute("alias").String(),
		defaultBotAlias,
	}
	for _, raw := range candidates {
		alias := strings.TrimSpace(raw)
		if validBotAlias(alias) {
			return alias
		}
	}
	return defaultBotAlias
}

func onboardingContext(r robot.Robot, m *robot.Message) (userName, channelName, protocol string) {
	userName = canonicalUserKey(r.GetParameter(paramOnboardingUser))
	channelName = strings.TrimSpace(r.GetParameter(paramOnboardingChannel))
	protocol = strings.ToLower(strings.TrimSpace(r.GetParameter(paramOnboardingProtocol)))
	if m != nil {
		if userName == "" {
			userName = canonicalUserKey(m.User)
		}
		if channelName == "" {
			channelName = strings.TrimSpace(m.Channel)
		}
		if protocol == "" {
			protocol = protocolName(m)
		}
	}
	if protocol == "" {
		protocol = "unknown"
	}
	return userName, channelName, protocol
}

func preferredOnboardingUser(r robot.Robot, startedBy string, m *robot.Message) string {
	candidates := []string{
		r.GetParameter("GOPHER_USER"),
		r.GetParameter(paramOnboardingUser),
		os.Getenv("USER"),
		os.Getenv("LOGNAME"),
		startedBy,
	}
	if m != nil {
		candidates = append(candidates, m.User)
	}
	for _, raw := range candidates {
		user := canonicalUserKey(raw)
		if usernameRe.MatchString(user) {
			return user
		}
	}
	for _, raw := range candidates {
		user := canonicalUserKey(raw)
		if user != "" {
			return user
		}
	}
	return ""
}

func protocolName(m *robot.Message) string {
	if m != nil && m.Incoming != nil {
		p := strings.ToLower(strings.TrimSpace(m.Incoming.Protocol))
		if p != "" {
			return p
		}
	}
	if m == nil {
		return "unknown"
	}
	switch m.Protocol {
	case robot.Slack:
		return "slack"
	case robot.Rocket:
		return "rocket"
	case robot.Terminal:
		return "terminal"
	case robot.Test:
		return "test"
	case robot.Null:
		return "nullconn"
	}
	// Legacy plugin import path github.com/lnxjedi/gopherbot/robot may not
	// define robot.SSH, but the enum value still arrives as protocol(5).
	if int(m.Protocol) == 5 {
		return "ssh"
	}
	p := strings.ToLower(strings.TrimSpace(m.Protocol.String()))
	if p == "protocol(5)" {
		return "ssh"
	}
	if p == "" {
		return "unknown"
	}
	return p
}

func loadState() (setupStateFile, error) {
	state := setupStateFile{
		Version:  stateFileVersion,
		Sessions: make(map[string]setupSession),
	}

	body, err := os.ReadFile(StateFileName)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return state, nil
		}
		return state, err
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		return state, nil
	}
	if err := json.Unmarshal(body, &state); err != nil {
		return state, fmt.Errorf("parsing JSON: %w", err)
	}
	if state.Version == 0 {
		state.Version = stateFileVersion
	}
	if state.Sessions == nil {
		state.Sessions = make(map[string]setupSession)
	}
	return state, nil
}

func saveState(state setupStateFile) error {
	state.Version = stateFileVersion
	body, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling JSON: %w", err)
	}

	tmp := StateFileName + ".tmp"
	if err := os.WriteFile(tmp, body, 0600); err != nil {
		return fmt.Errorf("writing temp file: %w", err)
	}
	if err := os.Rename(tmp, StateFileName); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("renaming temp file: %w", err)
	}
	return nil
}

func ClearSession(user string) error {
	state, err := loadState()
	if err != nil {
		return err
	}
	if state.Sessions == nil {
		return nil
	}
	if _, ok := state.Sessions[user]; ok {
		delete(state.Sessions, user)
	} else {
		for key, session := range state.Sessions {
			if canonicalUserKey(session.CanonicalUser) == user {
				delete(state.Sessions, key)
				break
			}
		}
	}
	return saveState(state)
}
