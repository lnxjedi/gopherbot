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
	stageAwaitingConfirm    = "awaiting-confirmation"
	stageScaffolded         = "scaffolded"
	stageAwaitingRepoURL    = "awaiting-repository-url"
	stageRepoReady          = "repository-ready"
	defaultScaffoldPath     = "custom"
	defaultProtocol         = "ssh"
	defaultBrain            = "file"
	defaultCustomRepository = "local"
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
	userName := ""
	channelName := ""
	if m != nil {
		userName = m.User
		channelName = m.Channel
	}
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
			StartedBy:    userName,
		}
	} else if session.Status == statusCompleted && session.Stage == stageRepoReady {
		r.Reply("Repository handoff is already complete for %s.", session.CanonicalUser)
		if session.RepositoryURL != "" {
			r.Say("Configured GOPHER_CUSTOM_REPOSITORY: %s", session.RepositoryURL)
		}
		return
	} else if session.Status == statusCompleted && session.Stage == stageScaffolded {
		// Backward compatibility for Slice 2 sessions: continue into repository handoff.
		session.Status = statusActive
		session.Stage = stageAwaitingRepoURL
		session.CompletedAtUTC = ""
	}

	session.LastCommand = command
	session.LastChannel = channelName
	session.LastProtocol = protocolName(m)
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
	now := time.Now().UTC().Format(time.RFC3339)
	persist := func(saveErrorMsg string) bool {
		state.Sessions[userKey] = *session
		if err := saveState(*state); err != nil {
			r.Log(robot.Error, "Saving %s: %v", stateFileName, err)
			r.Reply(saveErrorMsg, stateFileName)
			return false
		}
		return true
	}

	defaultUser := canonicalUserKey(session.StartedBy)
	if defaultUser == "" {
		if m := r.GetMessage(); m != nil {
			defaultUser = canonicalUserKey(m.User)
		}
	}

	scaffoldReady := session.Stage == stageScaffolded || session.Stage == stageAwaitingRepoURL || session.Stage == stageRepoReady
	if !scaffoldReady && (session.CanonicalUser == "" || session.Stage == stageAwaitingUsername) {
		user, ok := promptCanonicalUser(r, defaultUser)
		if !ok {
			session.Stage = stageAwaitingUsername
			session.UpdatedAtUTC = now
			persist("I couldn't save onboarding progress to %s")
			return
		}
		session.CanonicalUser = user
		session.Stage = stageAwaitingConfirm
		session.UpdatedAtUTC = now
		if !persist("I couldn't save onboarding progress to %s") {
			return
		}
	}

	if !scaffoldReady && session.SSHPublicKey == "" {
		key, source, ok := resolveSSHPublicKey(r)
		if !ok {
			session.Stage = stageAwaitingConfirm
			session.UpdatedAtUTC = now
			persist("I couldn't save onboarding progress to %s")
			return
		}
		session.SSHPublicKey = key
		session.SSHPublicKeySource = source
		session.Stage = stageAwaitingConfirm
		session.UpdatedAtUTC = now
		if !persist("I couldn't save onboarding progress to %s") {
			return
		}
	}

	if !scaffoldReady {
		proceed, ok := promptForConfirmation(r, session)
		if !ok {
			session.Stage = stageAwaitingConfirm
			session.UpdatedAtUTC = now
			persist("I couldn't save onboarding progress to %s")
			return
		}
		if !proceed {
			session.Stage = stageAwaitingConfirm
			session.UpdatedAtUTC = now
			persist("I couldn't save onboarding progress to %s")
			r.Reply("No changes were applied. Use 'new robot resume' when you're ready.")
			return
		}

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
			r.Say("Restart gopherbot, then connect with: bot-ssh -l %s", session.CanonicalUser)
		}
		session.Status = statusActive
		session.Stage = stageScaffolded
		session.CompletedAtUTC = ""
		session.UpdatedAtUTC = now
		if !persist("I couldn't save onboarding progress to %s") {
			return
		}
	}

	if session.Stage == stageScaffolded || session.Stage == stageAwaitingRepoURL || session.RepositoryURL == "" {
		repoURL, ok := promptRepositoryURL(r, session.RepositoryURL)
		if !ok {
			session.Stage = stageAwaitingRepoURL
			session.UpdatedAtUTC = now
			persist("I couldn't save onboarding progress to %s")
			return
		}
		session.RepositoryURL = repoURL
		session.Stage = stageAwaitingRepoURL
		session.UpdatedAtUTC = now
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
	session.CompletedAtUTC = now
	session.UpdatedAtUTC = now
	if !persist("Repository handoff succeeded but I couldn't persist final state in %s") {
		return
	}

	r.Reply("Repository handoff is ready. Updated .env with GOPHER_CUSTOM_REPOSITORY and GOPHER_DEPLOY_KEY.")
	r.Say("Add this read-only deploy key to your repository (%s):", session.RepositoryURL)
	r.Fixed().Say("%s", deployPubKey)
	r.Say("Next: from '%s', run: git init && git add . && git commit -m \"Initial robot config\" && git remote add origin %s && git push -u origin main", defaultScaffoldPath, session.RepositoryURL)
	r.Say("Bootstrap test: stop robot, keep only .env, start gopherbot; bootstrap should clone %s.", session.RepositoryURL)
}

func promptCanonicalUser(r robot.Robot, fallback string) (string, bool) {
	if fallback == "" {
		fallback = "alice"
	}
	for i := 0; i < 3; i++ {
		rep, ret := r.PromptForReply("username",
			"Choose your canonical username for local ssh login (bot-ssh -l <username>). Default '%s'; reply '=' to use default.", fallback)
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
		r.Say("Detected local SSH public key: %s", source)
		return key, source, true
	}

	r.Say("I couldn't auto-detect a local SSH public key in ~/.ssh; please paste one now.")
	rep, ret := r.PromptForReply("sshpubkey", "Paste your SSH public key line (e.g. 'ssh-ed25519 AAAA...').")
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
			return "", "", false
		}
		return key, "prompt", true
	default:
		r.Reply("I couldn't read your SSH key response (%s).", ret)
		return "", "", false
	}
}

func promptForConfirmation(r robot.Robot, s *setupSession) (bool, bool) {
	rep, ret := r.PromptForReply("YesNo",
		"I will create scaffold files in '%s', update .env, and map '%s' to SSH key %s. Proceed? [yes/no]",
		defaultScaffoldPath,
		s.CanonicalUser,
		shortSSHKeySummary(s.SSHPublicKey),
	)
	switch ret {
	case robot.Interrupted:
		r.Reply("Setup paused before apply. Use 'new robot resume' to continue.")
		return false, false
	case robot.TimeoutExpired:
		r.Reply("Timed out waiting for confirmation. Use 'new robot resume' to continue.")
		return false, false
	case robot.Ok:
		v := strings.ToLower(strings.TrimSpace(rep))
		if v == "y" || v == "yes" {
			return true, true
		}
		if v == "n" || v == "no" {
			return false, true
		}
		r.Reply("Please answer yes or no.")
		return false, false
	default:
		r.Reply("I couldn't read your confirmation response (%s).", ret)
		return false, false
	}
}

func promptRepositoryURL(r robot.Robot, current string) (string, bool) {
	defaultRepo := strings.TrimSpace(current)
	if defaultRepo == "" || defaultRepo == defaultCustomRepository {
		defaultRepo = ""
	}
	prompt := "Paste your repository clone URL (e.g. 'git@github.com:owner/repo.git')."
	if defaultRepo != "" {
		prompt = fmt.Sprintf("%s Reply '=' to keep '%s'.", prompt, defaultRepo)
	}
	for i := 0; i < 3; i++ {
		rep, ret := r.PromptForReply("repourl", prompt)
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
	deployPrivatePEM, deployPub, err := generateDeployKeyPair(robotMetaFromUser(s.CanonicalUser).botEmail)
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
	if err := writeDeployPublicKey(deployPublicPath, deployPubKey); err != nil {
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
	sshPhrase, err := randomBase64String(24)
	if err != nil {
		return fmt.Errorf("generating ssh phrase: %w", err)
	}
	sshEncrypted, err := encryptSecretForConfig(encryptionKey, sshPhrase)
	if err != nil {
		return fmt.Errorf("encrypting ssh phrase: %w", err)
	}

	meta := robotMetaFromUser(s.CanonicalUser)
	replace := map[string]string{
		"<botname>":      meta.botName,
		"<botemail>":     meta.botEmail,
		"<botfullname>":  meta.botFullName,
		"<botalias>":     meta.botAlias,
		"<sshencrypted>": sshEncrypted,
	}

	for _, rel := range []string{
		"conf/robot.yaml",
		"conf/ssh.yaml",
		"conf/terminal.yaml",
		"conf/slack.yaml",
		"git/config",
	} {
		fp := filepath.Join(defaultScaffoldPath, rel)
		if err := replaceTokensInFile(fp, replace); err != nil {
			return err
		}
	}
	if err := rewriteManageKeyReference(filepath.Join(defaultScaffoldPath, "conf", "robot.yaml")); err != nil {
		return err
	}

	if err := generateSSHKeyMaterial(filepath.Join(defaultScaffoldPath, "ssh"), sshPhrase, meta.botEmail); err != nil {
		return err
	}

	if err := appendIdentityConfig(
		filepath.Join(defaultScaffoldPath, "conf", "robot.yaml"),
		filepath.Join(defaultScaffoldPath, "conf", "ssh.yaml"),
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
# Added by new-robot onboarding (Slice 2)
UserMap:
  %s: "%s"
`, escapedUser, escapedKey)
	if err := appendFile(sshConfig, sshBlock); err != nil {
		return fmt.Errorf("updating %s: %w", sshConfig, err)
	}

	robotBlock := fmt.Sprintf(`
# Added by new-robot onboarding (Slice 2)
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
		"GOPHER_PROTOCOL":          defaultProtocol,
		"GOPHER_BRAIN":             defaultBrain,
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
		if key == "GOPHER_CUSTOM_BRANCH" {
			lines[i] = ""
		}
	}

	for key, val := range required {
		if seen[key] {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s=%s", key, val))
	}
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

func stripSetupPlaceholderLines(lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		switch trim {
		case "# Optional for later remote bootstrap",
			"# GOPHER_DEPLOY_KEY=<set this in slice 3>",
			"# GOPHER_CUSTOM_BRANCH=.":
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

func rewriteManageKeyReference(robotConfigPath string) error {
	body, err := os.ReadFile(robotConfigPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", robotConfigPath, err)
	}
	updated := strings.ReplaceAll(string(body), `Value: "manage_key"`, `Value: "robot_key"`)
	if err := os.WriteFile(robotConfigPath, []byte(updated), 0600); err != nil {
		return fmt.Errorf("writing %s: %w", robotConfigPath, err)
	}
	return nil
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

func encryptSecretForConfig(envKey, plaintext string) (string, error) {
	key := []byte(strings.TrimSpace(envKey))
	switch len(key) {
	case 16, 24, 32:
	default:
		return "", fmt.Errorf("invalid encryption key length %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ct := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ct), nil
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

func writeDeployPublicKey(path, deployPubKey string) error {
	if strings.TrimSpace(deployPubKey) == "" {
		return fmt.Errorf("deploy public key is empty")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("creating deploy key directory for %s: %w", path, err)
	}
	if err := os.WriteFile(path, []byte(strings.TrimSpace(deployPubKey)+"\n"), 0644); err != nil {
		return fmt.Errorf("writing deploy public key %s: %w", path, err)
	}
	return nil
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

func yamlDoubleQuoteEscape(v string) string {
	v = strings.ReplaceAll(v, `\\`, `\\\\`)
	v = strings.ReplaceAll(v, `"`, `\\"`)
	return v
}

func canonicalUserKey(user string) string {
	return strings.ToLower(strings.TrimSpace(user))
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
	p := strings.ToLower(strings.TrimSpace(m.Protocol.String()))
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
