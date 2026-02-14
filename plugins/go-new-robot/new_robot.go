package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
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

	statusActive    = "active"
	statusCompleted = "completed"

	stageShell              = "wizard-shell" // slice-1 compatibility
	stageAwaitingUsername   = "awaiting-username"
	stageAwaitingConfirm    = "awaiting-confirmation"
	stageScaffolded         = "scaffolded"
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
CommandMatchers:
- Command: "new-robot"
  Regex: '(?i:new(?:-|[[:space:]]+)robot)$'
- Command: "new-robot-resume"
  Regex: '(?i:(?:resume|continue)[[:space:]]+new(?:-|[[:space:]]+)robot|new(?:-|[[:space:]]+)robot[[:space:]]+(?:resume|continue))$'
- Command: "new-robot-cancel"
  Regex: '(?i:(?:cancel|abort|stop)[[:space:]]+new(?:-|[[:space:]]+)robot|new(?:-|[[:space:]]+)robot[[:space:]]+(?:cancel|abort|stop))$'
ReplyMatchers:
- Label: username
  Regex: '(?i:[a-z][a-z0-9_-]{0,31})'
- Label: sshpubkey
  Regex: '(?i:ssh-(?:ed25519|rsa|ecdsa|dss)[[:space:]]+[A-Za-z0-9+/=]+(?:[[:space:]]+[-._@A-Za-z0-9]+)?)'
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
}

func Configure() *[]byte {
	return &defaultConfig
}

func PluginHandler(r robot.Robot, command string, args ...string) (retval robot.TaskRetVal) {
	switch command {
	case "init":
		return
	case commandStart, commandResume, commandCancel:
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
		session = setupSession{
			Status:       statusActive,
			Stage:        stageAwaitingUsername,
			StartedAtUTC: now,
			StartedBy:    userName,
		}
	} else if session.Status == statusCompleted && session.Stage == stageScaffolded {
		r.Reply("Onboarding scaffold is already complete for %s.", session.CanonicalUser)
		r.Say("Restart gopherbot, then connect with: bot-ssh -l %s", session.CanonicalUser)
		return
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
	defaultUser := canonicalUserKey(session.StartedBy)
	if defaultUser == "" {
		if m := r.GetMessage(); m != nil {
			defaultUser = canonicalUserKey(m.User)
		}
	}

	if session.CanonicalUser == "" || session.Stage == stageAwaitingUsername {
		user, ok := promptCanonicalUser(r, defaultUser)
		if !ok {
			session.Stage = stageAwaitingUsername
			session.UpdatedAtUTC = now
			state.Sessions[userKey] = *session
			_ = saveState(*state)
			return
		}
		session.CanonicalUser = user
		session.Stage = stageAwaitingConfirm
		session.UpdatedAtUTC = now
		state.Sessions[userKey] = *session
		if err := saveState(*state); err != nil {
			r.Log(robot.Error, "Saving %s: %v", stateFileName, err)
			r.Reply("I couldn't save onboarding progress to %s", stateFileName)
			return
		}
	}

	if session.SSHPublicKey == "" {
		key, source, ok := resolveSSHPublicKey(r)
		if !ok {
			session.Stage = stageAwaitingConfirm
			session.UpdatedAtUTC = now
			state.Sessions[userKey] = *session
			_ = saveState(*state)
			return
		}
		session.SSHPublicKey = key
		session.SSHPublicKeySource = source
		session.Stage = stageAwaitingConfirm
		session.UpdatedAtUTC = now
		state.Sessions[userKey] = *session
		if err := saveState(*state); err != nil {
			r.Log(robot.Error, "Saving %s: %v", stateFileName, err)
			r.Reply("I couldn't save onboarding progress to %s", stateFileName)
			return
		}
	}

	proceed, ok := promptForConfirmation(r, session)
	if !ok {
		session.Stage = stageAwaitingConfirm
		session.UpdatedAtUTC = now
		state.Sessions[userKey] = *session
		_ = saveState(*state)
		return
	}
	if !proceed {
		session.Stage = stageAwaitingConfirm
		session.UpdatedAtUTC = now
		state.Sessions[userKey] = *session
		_ = saveState(*state)
		r.Reply("No changes were applied. Use 'new robot resume' when you're ready.")
		return
	}

	if err := applyScaffold(*session); err != nil {
		if errors.Is(err, errScaffoldExists) {
			session.Status = statusCompleted
			session.Stage = stageScaffolded
			session.CompletedAtUTC = now
			session.UpdatedAtUTC = now
			state.Sessions[userKey] = *session
			_ = saveState(*state)
			r.Reply("Scaffold already exists under '%s'; marking onboarding as complete.", defaultScaffoldPath)
			r.Say("Restart gopherbot, then connect with: bot-ssh -l %s", session.CanonicalUser)
			return
		}
		r.Log(robot.Error, "Applying scaffold for user '%s': %v", session.CanonicalUser, err)
		r.Reply("I couldn't apply scaffold changes: %v", err)
		r.Say("Your session is preserved. Fix the issue and run 'new robot resume'.")
		return
	}

	session.Status = statusCompleted
	session.Stage = stageScaffolded
	session.CompletedAtUTC = now
	session.UpdatedAtUTC = now
	state.Sessions[userKey] = *session
	if err := saveState(*state); err != nil {
		r.Log(robot.Error, "Saving %s: %v", stateFileName, err)
		r.Reply("Scaffold succeeded but I couldn't persist final session state in %s", stateFileName)
	}

	r.Reply("Scaffold created under '%s' and local identity configured for '%s'.", defaultScaffoldPath, session.CanonicalUser)
	r.Say("Restart gopherbot, then connect with: bot-ssh -l %s", session.CanonicalUser)
	r.Say("Slice 3 will guide repository handoff and final GOPHER_CUSTOM_REPOSITORY settings.")
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

func applyScaffold(s setupSession) error {
	robotConf := filepath.Join(defaultScaffoldPath, "conf", "robot.yaml")
	if _, err := os.Stat(robotConf); err == nil {
		return errScaffoldExists
	}

	installDir := strings.TrimSpace(os.Getenv("GOPHER_INSTALLDIR"))
	if installDir == "" {
		return fmt.Errorf("GOPHER_INSTALLDIR is not set")
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
	sshEncrypted, err := encryptWithCLI(installDir, encryptionKey, sshPhrase)
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
	botName := clean + "bot"
	if len(botName) > 24 {
		botName = botName[:24]
	}
	first := strings.Title(strings.ReplaceAll(clean, "_", " "))
	first = strings.Title(strings.ReplaceAll(first, "-", " "))
	firstParts := strings.Fields(first)
	firstName := "User"
	if len(firstParts) > 0 {
		firstName = firstParts[0]
	}
	return generatedMeta{
		botName:         botName,
		botEmail:        fmt.Sprintf("%s@example.com", botName),
		botFullName:     fmt.Sprintf("%s Gopherbot", firstName),
		botAlias:        ";",
		userEmail:       fmt.Sprintf("%s@example.com", clean),
		userDisplayName: fmt.Sprintf("%s User", firstName),
		userFirstName:   firstName,
	}
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

	seen := map[string]bool{}
	for i, line := range lines {
		key, _, ok := parseEnvLine(line)
		if !ok {
			continue
		}
		if val, shouldSet := required[key]; shouldSet {
			lines[i] = fmt.Sprintf("%s=%s", key, val)
			seen[key] = true
		}
	}

	if len(lines) == 0 {
		lines = []string{}
	}
	for key, val := range required {
		if seen[key] {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s=%s", key, val))
	}

	if !containsPrefixLine(lines, "# Optional for later remote bootstrap") {
		lines = append(lines,
			"",
			"# Optional for later remote bootstrap",
			"# GOPHER_DEPLOY_KEY=<set this in slice 3>",
			"# GOPHER_CUSTOM_BRANCH=.",
		)
	}

	content := strings.TrimRight(strings.Join(lines, "\n"), "\n") + "\n"
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

func containsPrefixLine(lines []string, prefix string) bool {
	for _, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), prefix) {
			return true
		}
	}
	return false
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

func encryptWithCLI(installDir, envKey, plaintext string) (string, error) {
	bin := filepath.Join(installDir, "gopherbot")
	if _, err := os.Stat(bin); err != nil {
		bin = "gopherbot"
	}
	cmd := exec.Command(bin, "encrypt", plaintext)
	cmd.Env = append(os.Environ(), "GOPHER_ENCRYPTION_KEY="+envKey)
	output, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("encrypt command failed: %s", strings.TrimSpace(string(ee.Stderr)))
		}
		return "", err
	}
	enc := strings.TrimSpace(string(output))
	if enc == "" {
		return "", fmt.Errorf("encrypt command returned empty output")
	}
	return enc, nil
}

func generateSSHKeyMaterial(sshDir, passphrase, comment string) error {
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		return fmt.Errorf("ssh-keygen not found in PATH")
	}
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return fmt.Errorf("creating ssh dir %s: %w", sshDir, err)
	}
	robotKey := filepath.Join(sshDir, "robot_key")
	manageKey := filepath.Join(sshDir, "manage_key")
	deployKey := filepath.Join(sshDir, "deploy_key")

	if err := runSSHKeygen(robotKey, passphrase, comment); err != nil {
		return err
	}
	if err := runSSHKeygen(manageKey, passphrase, comment); err != nil {
		return err
	}
	if err := runSSHKeygen(deployKey, "", comment); err != nil {
		return err
	}
	return nil
}

func runSSHKeygen(path, passphrase, comment string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("ssh key already exists: %s", path)
	}
	cmd := exec.Command("ssh-keygen", "-q", "-N", passphrase, "-C", comment, "-t", "ed25519", "-f", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ssh-keygen failed for %s: %v (%s)", path, err, strings.TrimSpace(string(out)))
	}
	return nil
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
