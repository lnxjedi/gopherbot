package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
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
	stateFileName     = ".setup-state"
	stateFileVersion  = 2
	stateExclusiveTag = "new-robot-state"

	commandStart  = "new-robot"
	commandResume = "new-robot-resume"
	commandCancel = "new-robot-cancel"
	commandRepo   = "new-robot-repo"

	statusActive    = "active"
	statusCompleted = "completed"

	stageShell              = "wizard-shell" // slice-1 compatibility
	stageAwaitingUsername   = "awaiting-username"
	stageAwaitingConfirm    = "awaiting-confirmation" // backward compatibility
	stageAwaitingSSHKey     = "awaiting-ssh-key"
	stageScaffolded         = "scaffolded"
	stageAwaitingRepoURL    = "awaiting-repository-url"
	stageRepoReady          = "repository-ready"
	defaultScaffoldPath     = "custom"
	defaultEnvironment      = "development"
	defaultCustomRepository = "local"
	binaryKeyFileName       = "binary-encrypted-key"

	paramOnboardingUser     = "GOPHER_ONBOARDING_USER"
	paramOnboardingChannel  = "GOPHER_ONBOARDING_CHANNEL"
	paramOnboardingProtocol = "GOPHER_ONBOARDING_PROTOCOL"

	onboardingPluginBeginMarker = "# BEGIN NEW-ROBOT ONBOARDING PLUGIN"
	onboardingPluginEndMarker   = "# END NEW-ROBOT ONBOARDING PLUGIN"
	onboardingJobBeginMarker    = "# BEGIN NEW-ROBOT ONBOARDING JOB"
	onboardingJobEndMarker      = "# END NEW-ROBOT ONBOARDING JOB"
)

var (
	usernameRe  = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,31}$`)
	sshPubKeyRe = regexp.MustCompile(`^ssh-(?:ed25519|rsa|ecdsa|dss)\s+[A-Za-z0-9+/=]+(?:\s+[-._@A-Za-z0-9]+)?$`)
	envKeyRe    = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

	errScaffoldExists = errors.New("scaffold already exists")
)

var defaultConfig = []byte(`
Help:
- Keywords: [ "new", "robot", "setup", "onboarding" ]
  Helptext: [ "(bot), new robot - start guided setup for a new robot" ]
- Keywords: [ "new", "robot", "resume", "onboarding" ]
  Helptext: [ "(bot), new robot resume - resume your onboarding session" ]
- Keywords: [ "new", "robot", "cancel", "onboarding" ]
  Helptext: [ "(bot), new robot cancel - cancel your onboarding session" ]
- Keywords: [ "new", "robot", "repo", "repository", "onboarding" ]
  Helptext: [ "(bot), new robot repo - continue repository handoff and .env bootstrap setup" ]
CommandMatchers:
- Command: "new-robot"
  Regex: '(?i:new(?:-|[[:space:]]+)robot)$'
- Command: "new-robot-resume"
  Regex: '(?i:(?:resume|continue)[[:space:]]+new(?:-|[[:space:]]+)robot|new(?:-|[[:space:]]+)robot[[:space:]]+(?:resume|continue))$'
- Command: "new-robot-cancel"
  Regex: '(?i:(?:cancel|abort|stop)[[:space:]]+new(?:-|[[:space:]]+)robot|new(?:-|[[:space:]]+)robot[[:space:]]+(?:cancel|abort|stop))$'
- Command: "new-robot-repo"
  Regex: '(?i:new(?:-|[[:space:]]+)robot[[:space:]]+(?:repo|repository)|(?:repo|repository)[[:space:]]+new(?:-|[[:space:]]+)robot)$'
ReplyMatchers:
- Label: username
  Regex: '(?i:[a-z][a-z0-9_-]{0,31})'
- Label: sshpubkey
  Regex: '(?i:ssh-(?:ed25519|rsa|ecdsa|dss)[[:space:]]+[A-Za-z0-9+/=]+(?:[[:space:]]+[-._@A-Za-z0-9]+)?)'
- Label: repourl
  Regex: '[^[:space:]]+'
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
	CanonicalUser      string `json:"canonicalUser,omitempty"`
	SSHPublicKey       string `json:"sshPublicKey,omitempty"`
	SSHPublicKeySource string `json:"sshPublicKeySource,omitempty"`
	RepositoryURL      string `json:"repositoryUrl,omitempty"`
}

func Configure() *[]byte {
	return &defaultConfig
}

func PluginHandler(r robot.Robot, command string, args ...string) (retval robot.TaskRetVal) {
	switch command {
	case "init":
		return
	case commandStart, commandResume, commandCancel, commandRepo:
		handleStateCommand(r, command)
	}
	return
}

func handleStateCommand(r robot.Robot, command string) {
	if !r.Exclusive(stateExclusiveTag, false) {
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
		r.Log(robot.Error, "Loading %s: %v", stateFileName, err)
		r.Reply("I couldn't read onboarding state from %s", stateFileName)
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	session, exists := state.Sessions[userKey]

	switch command {
	case commandCancel:
		if !exists {
			r.Reply("You don't have an onboarding session to cancel.")
			return
		}
		delete(state.Sessions, userKey)
		if err := saveState(state); err != nil {
			r.Log(robot.Error, "Saving %s: %v", stateFileName, err)
			r.Reply("I couldn't clear onboarding state in %s", stateFileName)
			return
		}
		if session.Status == statusCompleted {
			r.Reply("Cleared completed onboarding state from %s.", stateFileName)
		} else {
			r.Reply("Canceled your onboarding session and removed it from %s.", stateFileName)
		}
		return
	}

	if !exists {
		if command == commandRepo {
			r.Reply("No onboarding session found. Start with 'new robot' first.")
			return
		}
		session = setupSession{
			Status:       statusActive,
			Stage:        stageAwaitingUsername,
			StartedAtUTC: now,
			StartedBy:    userKey,
		}
	} else if session.Status == statusCompleted && session.Stage == stageRepoReady {
		r.Reply("Repository handoff is already complete for %s.", session.CanonicalUser)
		if session.RepositoryURL != "" {
			r.Say("Configured GOPHER_CUSTOM_REPOSITORY: %s", session.RepositoryURL)
		}
		return
	} else if session.Status == statusCompleted && session.Stage == stageScaffolded {
		// Backward compatibility for earlier onboarding sessions.
		session.Status = statusActive
		session.Stage = stageAwaitingRepoURL
		session.CompletedAtUTC = ""
	}

	session.LastCommand = command
	session.LastChannel = channelName
	session.LastProtocol = protocol
	session.UpdatedAtUTC = now
	if session.Stage == "" || session.Stage == stageShell {
		session.Stage = stageAwaitingUsername
	}

	state.Sessions[userKey] = session
	if err := saveState(state); err != nil {
		r.Log(robot.Error, "Saving %s: %v", stateFileName, err)
		r.Reply("I couldn't update onboarding state in %s", stateFileName)
		return
	}

	session = state.Sessions[userKey]
	continueWizard(r, &state, userKey, &session)
}

func continueWizard(r robot.Robot, state *setupStateFile, userKey string, session *setupSession) {
	sessionKey := userKey
	nowUTC := func() string {
		return time.Now().UTC().Format(time.RFC3339)
	}
	persist := func(saveErrorMsg string) bool {
		state.Sessions[sessionKey] = *session
		if err := saveState(*state); err != nil {
			r.Log(robot.Error, "Saving %s: %v", stateFileName, err)
			r.Reply(saveErrorMsg, stateFileName)
			return false
		}
		return true
	}

	defaultUser := preferredOnboardingUser(session.StartedBy, r.GetMessage())
	if session.Stage == stageAwaitingConfirm {
		// Compatibility for older session state values.
		session.Stage = stageAwaitingSSHKey
	}

	scaffoldReady := session.Stage == stageScaffolded || session.Stage == stageAwaitingRepoURL || session.Stage == stageRepoReady
	if !scaffoldReady && (session.CanonicalUser == "" || session.Stage == stageAwaitingUsername) {
		user, ok := promptCanonicalUser(r, defaultUser)
		if !ok {
			session.Stage = stageAwaitingUsername
			session.UpdatedAtUTC = nowUTC()
			persist("I couldn't save onboarding progress to %s")
			return
		}
		session.CanonicalUser = user
		if session.CanonicalUser != "" && session.CanonicalUser != sessionKey {
			delete(state.Sessions, sessionKey)
			sessionKey = session.CanonicalUser
		}
		session.Stage = stageAwaitingSSHKey
		session.UpdatedAtUTC = nowUTC()
		if !persist("I couldn't save onboarding progress to %s") {
			return
		}
	}

	if !scaffoldReady && (session.SSHPublicKey == "" || session.Stage == stageAwaitingSSHKey) {
		key, source, ok := resolveSSHPublicKey(r)
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

	if !scaffoldReady {
		if err := applyScaffold(*session); err != nil {
			if errors.Is(err, errScaffoldExists) {
				r.Reply("Scaffold already exists under '%s'; continuing with repository handoff.", defaultScaffoldPath)
			} else {
				r.Log(robot.Error, "Applying scaffold for user '%s': %v", session.CanonicalUser, err)
				r.Reply("I couldn't apply scaffold changes: %v", err)
				r.Say("Your session is preserved. Fix the issue and run 'new robot resume'.")
				return
			}
		} else {
			r.Reply("Scaffold created under '%s' and local identity configured for '%s'.", defaultScaffoldPath, session.CanonicalUser)
			r.Say("Saved SSH server public key to '%s/robot-ssh.pub'.", defaultScaffoldPath)
		}
		session.Status = statusActive
		session.Stage = stageAwaitingRepoURL
		session.CompletedAtUTC = ""
		session.UpdatedAtUTC = nowUTC()
		if !persist("I couldn't save onboarding progress to %s") {
			return
		}
		r.Pause(0.5)
		r.Say("Ok, I'll restart the robot, then you can reconnect as yourself with 'bot-ssh -l %s'.", session.CanonicalUser)
		r.Pause(0.5)
		r.AddTask("restart-robot")
		return
	}

	if session.Stage == stageScaffolded {
		session.Stage = stageAwaitingRepoURL
	}
	promptUser := session.CanonicalUser
	if promptUser == "" {
		promptUser = sessionKey
	}
	promptChannel := strings.TrimSpace(session.LastChannel)
	if promptChannel == "" {
		promptChannel = "general"
	}
	if session.Stage == stageScaffolded || session.Stage == stageAwaitingRepoURL || session.RepositoryURL == "" {
		repoURL, ok := promptRepositoryURL(r, session.RepositoryURL, promptUser, promptChannel)
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

	deployPubKey, err := applyRepositoryHandoff(*session)
	if err != nil {
		r.Log(robot.Error, "Applying repository handoff for user '%s': %v", session.CanonicalUser, err)
		r.Reply("I couldn't finish repository handoff: %v", err)
		r.Say("Your session is preserved. Run 'new robot repo' after fixing the issue.")
		return
	}

	session.Status = statusCompleted
	session.Stage = stageRepoReady
	session.CompletedAtUTC = nowUTC()
	session.UpdatedAtUTC = session.CompletedAtUTC
	if !persist("Repository handoff succeeded but I couldn't persist final state in %s") {
		return
	}

	r.Reply("Repository handoff is ready. Updated .env with GOPHER_CUSTOM_REPOSITORY and GOPHER_DEPLOY_KEY.")
	r.Say("Add this read-only deploy key to your repository (%s):", session.RepositoryURL)
	r.Fixed().Say("%s", deployPubKey)
	r.Say("From '%s', run:", defaultScaffoldPath)
	r.Fixed().Say("git init\ngit add .\ngit branch -m main\ngit commit -m \"New robot!\"\ngit remote add origin %s\ngit push -u origin main", session.RepositoryURL)
	r.Say("Bootstrap test: stop robot, keep only .env, start gopherbot; bootstrap should clone %s.", session.RepositoryURL)
}

func promptCanonicalUser(r robot.Robot, fallback string) (string, bool) {
	if fallback == "" {
		fallback = "alice"
	}
	for i := 0; i < 3; i++ {
		rep, ret := r.PromptForReply("username",
			"What username do you want to use with your robot for local ssh login? (bot-ssh -l <username>) For team-chat robots, use your team-chat username. Default '%s'; reply '=' to use default.",
			fallback)
		switch ret {
		case robot.Interrupted:
			r.Reply("Setup paused. Use 'new robot resume' to continue.")
			return "", false
		case robot.TimeoutExpired:
			r.Reply("Timed out waiting for username. Use 'new robot resume' when ready.")
			return "", false
		case robot.UseDefaultValue:
			rep = fallback
		case robot.Ok:
			// use provided response
		default:
			r.Reply("I couldn't read your username response (%s).", ret)
			continue
		}
		candidate := canonicalUserKey(rep)
		if usernameRe.MatchString(candidate) {
			return candidate, true
		}
		r.Reply("'%s' isn't valid. Use lowercase letters, digits, '_' or '-', starting with a letter.", strings.TrimSpace(rep))
	}
	r.Reply("Too many invalid username attempts. Use 'new robot resume' to try again.")
	return "", false
}

func resolveSSHPublicKey(r robot.Robot) (string, string, bool) {
	if key, source, ok := detectLocalSSHPublicKey(); ok {
		rep, ret := r.PromptForReply("YesNo", "Detected local SSH public key: %s, use that one? (y|n)", source)
		switch ret {
		case robot.Interrupted:
			r.Reply("Setup paused. Use 'new robot resume' to continue.")
			return "", "", false
		case robot.TimeoutExpired:
			r.Reply("Timed out waiting for SSH key confirmation. Use 'new robot resume' when ready.")
			return "", "", false
		case robot.Ok:
			v := strings.ToLower(strings.TrimSpace(rep))
			if v == "y" || v == "yes" {
				return key, source, true
			}
			if v != "n" && v != "no" {
				r.Reply("Please answer y or n.")
				return "", "", false
			}
		default:
			r.Reply("I couldn't read your SSH key confirmation (%s).", ret)
			return "", "", false
		}
	}

	for i := 0; i < 3; i++ {
		rep, ret := r.PromptForReply("sshpubkey", "Paste the SSH public key line to use for local login (e.g. 'ssh-ed25519 AAAA...').")
		switch ret {
		case robot.Interrupted:
			r.Reply("Setup paused. Use 'new robot resume' to continue.")
			return "", "", false
		case robot.TimeoutExpired:
			r.Reply("Timed out waiting for SSH key. Use 'new robot resume' when ready.")
			return "", "", false
		case robot.Ok:
			key := normalizeSSHPublicKey(rep)
			if !sshPubKeyRe.MatchString(key) {
				r.Reply("That doesn't look like a valid SSH public key line.")
				continue
			}
			return key, "prompt", true
		default:
			r.Reply("I couldn't read your SSH key response (%s).", ret)
		}
	}
	r.Reply("Too many invalid SSH key attempts. Use 'new robot resume' to try again.")
	return "", "", false
}

func promptRepositoryURL(r robot.Robot, current, targetUser, targetChannel string) (string, bool) {
	defaultRepo := strings.TrimSpace(current)
	if defaultRepo == "" || defaultRepo == defaultCustomRepository {
		defaultRepo = ""
	}
	prompt := "Let's get this robot ready for the first deployment - what's the repository clone URL? (e.g. 'git@github.com:owner/repo.git')"
	if defaultRepo != "" {
		prompt = fmt.Sprintf("%s Reply '=' to keep '%s'.", prompt, defaultRepo)
	}
	for i := 0; i < 3; i++ {
		rep, ret := promptForTarget(r, "repourl", targetUser, targetChannel, prompt)
		switch ret {
		case robot.Interrupted:
			r.Reply("Repository handoff paused. Use 'new robot repo' to continue.")
			return "", false
		case robot.TimeoutExpired:
			r.Reply("Timed out waiting for repository URL. Use 'new robot repo' when ready.")
			return "", false
		case robot.UseDefaultValue:
			if defaultRepo == "" {
				r.Reply("No default repository is available yet.")
				continue
			}
			return defaultRepo, true
		case robot.Ok:
			repo := strings.TrimSpace(rep)
			if validRepositoryURL(repo) {
				return repo, true
			}
			r.Reply("That doesn't look like a supported clone URL.")
		default:
			r.Reply("I couldn't read your repository response (%s).", ret)
		}
	}
	r.Reply("Too many invalid repository attempts. Use 'new robot repo' to try again.")
	return "", false
}

func promptForTarget(r robot.Robot, regexID, targetUser, targetChannel, prompt string, v ...interface{}) (string, robot.RetVal) {
	targetUser = canonicalUserKey(targetUser)
	targetChannel = strings.TrimSpace(targetChannel)
	if targetUser != "" && targetChannel != "" {
		return r.PromptUserChannelForReply(regexID, targetUser, targetChannel, prompt, v...)
	}
	if targetUser != "" {
		return r.PromptUserForReply(regexID, targetUser, prompt, v...)
	}
	return r.PromptForReply(regexID, prompt, v...)
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
	deployPrivatePEM, deployPub, err := generateDeployKeyPair(robotMetaFromUser(s.CanonicalUser).botName)
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
	if err := disableOnboardingHooks(filepath.Join(defaultScaffoldPath, "conf", "robot.yaml")); err != nil {
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

func applyScaffold(s setupSession) error {
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

	encryptionKey, err := randomAlphaNum(32)
	if err != nil {
		return fmt.Errorf("generating encryption key: %w", err)
	}
	binaryKey, err := randomBytes(32)
	if err != nil {
		return fmt.Errorf("generating binary encryption key: %w", err)
	}
	sshPhrase, err := randomBase64String(24)
	if err != nil {
		return fmt.Errorf("generating ssh phrase: %w", err)
	}
	sshEncrypted, err := encryptSecretForConfig(binaryKey, sshPhrase)
	if err != nil {
		return fmt.Errorf("encrypting ssh phrase: %w", err)
	}
	meta := robotMetaFromUser(s.CanonicalUser)
	hostPrivatePEM, hostPubKey, err := generateDeployKeyPair(meta.botName)
	if err != nil {
		return fmt.Errorf("generating SSH host keypair: %w", err)
	}
	hostKeyTemplateLiteral, err := jsonString(hostPrivatePEM)
	if err != nil {
		return fmt.Errorf("encoding SSH host private key for template: %w", err)
	}
	hostKeyEncrypted, err := encryptSecretForConfig(binaryKey, hostKeyTemplateLiteral)
	if err != nil {
		return fmt.Errorf("encrypting SSH host private key: %w", err)
	}

	replace := map[string]string{
		"<botname>":             meta.botName,
		"<botemail>":            meta.botEmail,
		"<botfullname>":         meta.botFullName,
		"<botalias>":            meta.botAlias,
		"<sshencrypted>":        sshEncrypted,
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
	if err := writeOnboardingJobConfig(filepath.Join(defaultScaffoldPath, "conf", "jobs", "welcome-join.yaml"), meta.botName); err != nil {
		return err
	}

	if err := generateSSHKeyMaterial(filepath.Join(defaultScaffoldPath, "ssh"), sshPhrase, meta.botName); err != nil {
		return err
	}
	if err := writePublicKey(filepath.Join(defaultScaffoldPath, "robot-ssh.pub"), hostPubKey); err != nil {
		return fmt.Errorf("writing robot ssh public key: %w", err)
	}

	if err := appendIdentityConfig(
		filepath.Join(defaultScaffoldPath, "conf", "robot.yaml"),
		filepath.Join(defaultScaffoldPath, "conf", "protocols", "ssh.yaml"),
		s.CanonicalUser,
		s.SSHPublicKey,
		meta.userEmail,
		meta.userDisplayName,
		meta.userFirstName,
	); err != nil {
		return err
	}

	if err := writeOrUpdateEnv(encryptionKey); err != nil {
		return err
	}
	if err := writeBinaryEncryptedKeyFile(filepath.Join(defaultScaffoldPath, binaryKeyFileName), []byte(encryptionKey), binaryKey); err != nil {
		return err
	}

	return nil
}

type generatedMeta struct {
	botName         string
	botEmail        string
	botFullName     string
	botAlias        string
	userEmail       string
	userDisplayName string
	userFirstName   string
}

func robotMetaFromUser(user string) generatedMeta {
	clean := canonicalUserKey(user)
	if clean == "" {
		clean = "alice"
	}
	botName := detectBotName(clean)
	if len(botName) > 24 {
		botName = botName[:24]
	}
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
	return generatedMeta{
		botName:         botName,
		botEmail:        fmt.Sprintf("%s@example.com", botName),
		botFullName:     fmt.Sprintf("%s Gopherbot", botShort),
		botAlias:        ";",
		userEmail:       fmt.Sprintf("%s@example.com", clean),
		userDisplayName: fmt.Sprintf("%s User", userShort),
		userFirstName:   userShort,
	}
}

func detectBotName(fallbackUser string) string {
	if cwd, err := os.Getwd(); err == nil {
		base := strings.ToLower(strings.TrimSpace(filepath.Base(cwd)))
		if base != "" && base != "." && base != "custom" && usernameRe.MatchString(base) {
			return base
		}
	}
	return fallbackUser + "bot"
}

func appendIdentityConfig(robotConfig, sshConfig, user, sshKey, email, fullName, firstName string) error {
	escapedKey := yamlDoubleQuoteEscape(sshKey)
	escapedUser := yamlDoubleQuoteEscape(user)
	escapedMail := yamlDoubleQuoteEscape(email)
	escapedFull := yamlDoubleQuoteEscape(fullName)
	escapedFirst := yamlDoubleQuoteEscape(firstName)

	sshBlock := fmt.Sprintf(`
# Added by new-robot onboarding
UserMap:
  %s: "%s"
`, escapedUser, escapedKey)
	if err := appendFile(sshConfig, sshBlock); err != nil {
		return fmt.Errorf("updating %s: %w", sshConfig, err)
	}

	robotBlock := fmt.Sprintf(`
# Added by new-robot onboarding
AdminUsers: [ "%s" ]
DefaultChannels: [ "general", "random" ]
DefaultJobChannel: general
UserRoster:
- UserName: "%s"
  Email: "%s"
  FullName: "%s"
  FirstName: "%s"
  LastName: "User"
`, escapedUser, escapedUser, escapedMail, escapedFull, escapedFirst)
	if err := appendFile(robotConfig, robotBlock); err != nil {
		return fmt.Errorf("updating %s: %w", robotConfig, err)
	}

	return nil
}

func writeOrUpdateEnv(encryptionKey string) error {
	path := ".env"
	original, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("reading .env: %w", err)
	}

	required := map[string]string{
		"GOPHER_ENCRYPTION_KEY":    encryptionKey,
		"GOPHER_CUSTOM_REPOSITORY": defaultCustomRepository,
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
		if key == "GOPHER_CUSTOM_BRANCH" || key == "GOPHER_PROTOCOL" || key == "GOPHER_BRAIN" || key == "GOPHER_DEFAULT_PROTOCOL" {
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
		if key == "GOPHER_CUSTOM_BRANCH" {
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

func writeOnboardingJobConfig(path, botName string) error {
	botName = canonicalUserKey(botName)
	if botName == "" {
		botName = "floyd"
	}
	content := fmt.Sprintf(`---
Quiet: true
KeepLogs: 2
Triggers:
- User: %s
  Channel: general
  Regex: '(?i:^@([a-z][a-z0-9_-]{0,31}) has joined #([a-z0-9_-]+)$)'
`, botName)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating onboarding jobs directory for %s: %w", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		return fmt.Errorf("writing onboarding job config %s: %w", path, err)
	}
	return nil
}

func enableOnboardingHooks(robotConfigPath string) error {
	body, err := os.ReadFile(robotConfigPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", robotConfigPath, err)
	}
	txt := string(body)
	if strings.Contains(txt, onboardingPluginBeginMarker) && strings.Contains(txt, onboardingJobBeginMarker) {
		return nil
	}

	pluginBlock := strings.TrimRight(`
  # BEGIN NEW-ROBOT ONBOARDING PLUGIN
  "welcome":
    Description: Temporary onboarding welcome plugin
    Privileged: true
    Path: plugins/welcome.lua
  "new-robot":
    Description: Temporary onboarding state machine plugin
    Privileged: true
    Homed: true
    Path: plugins/go-new-robot/new_robot.go
  # END NEW-ROBOT ONBOARDING PLUGIN
`, "\n")
	jobBlock := strings.TrimRight(`
  # BEGIN NEW-ROBOT ONBOARDING JOB
  "welcome-join":
    Description: Temporary onboarding welcome-on-join job
    Path: jobs/go-welcome-join/welcome_join.go
    Homed: true
  # END NEW-ROBOT ONBOARDING JOB
`, "\n")

	updated, err := insertBlockAfterLine(txt, "ExternalPlugins:", pluginBlock)
	if err != nil {
		return fmt.Errorf("adding onboarding welcome plugin to %s: %w", robotConfigPath, err)
	}
	updated, err = insertBlockAfterLine(updated, "ExternalJobs:", jobBlock)
	if err != nil {
		return fmt.Errorf("adding onboarding welcome job to %s: %w", robotConfigPath, err)
	}

	if err := os.WriteFile(robotConfigPath, []byte(updated), 0600); err != nil {
		return fmt.Errorf("writing %s: %w", robotConfigPath, err)
	}
	return nil
}

func disableOnboardingHooks(robotConfigPath string) error {
	body, err := os.ReadFile(robotConfigPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("reading %s: %w", robotConfigPath, err)
	}
	txt := string(body)
	txt = commentMarkerBlock(txt, onboardingPluginBeginMarker, onboardingPluginEndMarker)
	txt = commentMarkerBlock(txt, onboardingJobBeginMarker, onboardingJobEndMarker)
	if err := os.WriteFile(robotConfigPath, []byte(txt), 0600); err != nil {
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

func commentMarkerBlock(text, beginMarker, endMarker string) string {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	inBlock := false
	for i, line := range lines {
		if strings.Contains(line, beginMarker) {
			inBlock = true
		}
		if inBlock {
			trim := strings.TrimSpace(line)
			if trim != "" && !strings.HasPrefix(trim, "#") {
				lines[i] = "# " + line
			}
		}
		if strings.Contains(line, endMarker) {
			inBlock = false
		}
	}
	return strings.Join(lines, "\n")
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

func encryptSecretForConfig(binaryKey []byte, plaintext string) (string, error) {
	switch len(binaryKey) {
	case 16, 24, 32:
	default:
		return "", fmt.Errorf("invalid encryption key length %d", len(binaryKey))
	}
	ct, err := encryptBytes(binaryKey, []byte(plaintext))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ct), nil
}

func encryptBytes(key []byte, plaintext []byte) ([]byte, error) {
	switch len(key) {
	case 16, 24, 32:
	default:
		return nil, fmt.Errorf("invalid encryption key length %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func writeBinaryEncryptedKeyFile(path string, envKey []byte, binaryKey []byte) error {
	if len(envKey) < 32 {
		return fmt.Errorf("invalid environment key length %d", len(envKey))
	}
	if len(binaryKey) != 32 {
		return fmt.Errorf("invalid binary key length %d", len(binaryKey))
	}
	ct, err := encryptBytes(envKey[:32], binaryKey)
	if err != nil {
		return fmt.Errorf("encrypting binary key: %w", err)
	}
	b64 := base64.StdEncoding.EncodeToString(ct)
	if err := os.WriteFile(path, []byte(b64), 0600); err != nil {
		return fmt.Errorf("writing binary key file %s: %w", path, err)
	}
	return nil
}

func generateSSHKeyMaterial(sshDir, passphrase, comment string) error {
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return fmt.Errorf("creating ssh dir %s: %w", sshDir, err)
	}
	robotKey := filepath.Join(sshDir, "robot_key")
	if err := writeEncryptedEd25519KeyPair(robotKey, passphrase, comment); err != nil {
		return err
	}
	return nil
}

func writeEncryptedEd25519KeyPair(path, passphrase, comment string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("ssh key already exists: %s", path)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("generating ed25519 key for %s: %w", path, err)
	}
	if strings.TrimSpace(passphrase) == "" {
		return fmt.Errorf("empty passphrase for %s", path)
	}
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return fmt.Errorf("encoding private key for %s: %w", path, err)
	}
	block, err := x509.EncryptPEMBlock(rand.Reader, "PRIVATE KEY", der, []byte(passphrase), x509.PEMCipherAES256)
	if err != nil {
		return fmt.Errorf("encrypting private key for %s: %w", path, err)
	}
	if err := os.WriteFile(path, pem.EncodeToMemory(block), 0600); err != nil {
		return fmt.Errorf("writing private key %s: %w", path, err)
	}

	pubLine, err := formatSSHEd25519PublicKey(pub, comment)
	if err != nil {
		return fmt.Errorf("formatting public key for %s: %w", path, err)
	}
	if err := os.WriteFile(path+".pub", []byte(pubLine+"\n"), 0644); err != nil {
		return fmt.Errorf("writing public key %s.pub: %w", path, err)
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

func randomBase64String(rawBytes int) (string, error) {
	if rawBytes <= 0 {
		return "", fmt.Errorf("invalid random byte length %d", rawBytes)
	}
	b := make([]byte, rawBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawStdEncoding.EncodeToString(b), nil
}

func randomBytes(length int) ([]byte, error) {
	if length <= 0 {
		return nil, fmt.Errorf("invalid length %d", length)
	}
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	return b, nil
}

func yamlDoubleQuoteEscape(v string) string {
	v = strings.ReplaceAll(v, `\\`, `\\\\`)
	v = strings.ReplaceAll(v, `"`, `\\"`)
	return v
}

func canonicalUserKey(user string) string {
	return strings.ToLower(strings.TrimSpace(user))
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

func preferredOnboardingUser(startedBy string, m *robot.Message) string {
	candidates := []string{
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

	body, err := os.ReadFile(stateFileName)
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

	tmp := stateFileName + ".tmp"
	if err := os.WriteFile(tmp, body, 0600); err != nil {
		return fmt.Errorf("writing temp file: %w", err)
	}
	if err := os.Rename(tmp, stateFileName); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("renaming temp file: %w", err)
	}
	return nil
}
