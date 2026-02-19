package bot

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
)

const (
	codexAuthDatumKey         = "bot:codex-auth-users"
	codexDefaultLoginTimeout  = 10 * time.Minute
	codexDefaultLinkStateDir  = ".gopherbot-codex"
	codexConfigFileName       = "config.toml"
	codexAuthFileName         = "auth.json"
	codexAuthStoreFileSetting = "cli_auth_credentials_store = \"file\"\n"
)

var codexPathPartRe = regexp.MustCompile(`[^a-z0-9_.-]+`)

type codexUserAuthRecord struct {
	AuthJSON  string `json:"auth_json"`
	Method    string `json:"method"`
	UpdatedAt string `json:"updated_at"`
}

type codexAuthStore map[string]codexUserAuthRecord

func codex(m robot.Robot, command string, args ...string) (retval robot.TaskRetVal) {
	r := m.(Robot)
	switch command {
	case "link":
		codexLink(r)
	case "unlink":
		codexUnlink(r, "")
	case "admin-unlink":
		target := ""
		if len(args) > 0 {
			target = strings.TrimSpace(args[0])
		}
		codexUnlink(r, target)
	case "status":
		codexStatus(r)
	case "admin-list":
		codexList(r)
	default:
		r.Say("Unsupported codex command '%s'", command)
	}
	return
}

func codexLink(r Robot) {
	user := codexCanonicalUser(r.User)
	if user == "" {
		r.Say("Unable to determine your user identity for Codex linking")
		return
	}
	if !r.Incoming.HiddenMessage {
		r.Reply("For privacy, run this as a hidden command: '/%s link-codex'", r.GetBotAttribute("name"))
	}
	_, exists, existing, ret := codexGetAuthRecord(r, user)
	if ret != robot.Ok {
		r.Say("I couldn't access Codex auth storage (%s)", ret)
		return
	}
	if exists && strings.TrimSpace(existing.AuthJSON) != "" {
		r.Say("I already have Codex credentials for your account; use ';unlink-codex' first to relink")
		return
	}

	linkDir, err := codexCreateTempLinkHome(user)
	if err != nil {
		r.Say("Unable to prepare Codex auth workspace: %v", err)
		return
	}
	defer codexRemovePath(linkDir)
	if err := codexWriteConfigFile(linkDir, codexAuthStoreFileSetting); err != nil {
		r.Say("Unable to initialize Codex auth configuration: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), codexDefaultLoginTimeout)
	defer cancel()

	r.Say("Starting Codex device login; follow the prompts below in your browser")
	err = codexRunStreamingCommand(ctx, codexBinaryPath(), []string{"login", "--device-auth"}, codexCommandEnv(linkDir), func(line string) {
		r.Reply("%s", line)
	})
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			r.Say("Timed out waiting for Codex login; please try again")
			return
		}
		r.Say("Codex login failed: %v", err)
		return
	}

	authJSON, err := codexReadAuthJSON(linkDir)
	if err != nil {
		r.Say("Codex login completed, but credentials were not found: %v", err)
		return
	}
	if !json.Valid([]byte(authJSON)) {
		r.Say("Codex login returned invalid auth JSON; not saving credentials")
		return
	}

	record := codexUserAuthRecord{
		AuthJSON:  authJSON,
		Method:    "device-auth",
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	if ret := codexPutAuthRecord(r, user, record); ret != robot.Ok {
		r.Say("Codex login succeeded, but saving credentials failed (%s)", ret)
		return
	}
	r.Say("Successfully linked Codex credentials for user '%s'", user)
}

func codexUnlink(r Robot, target string) {
	user := codexCanonicalUser(r.User)
	if user == "" {
		r.Say("Unable to determine your user identity for Codex unlink")
		return
	}
	if target != "" {
		if !r.CheckAdmin() {
			r.Say("Only bot admins can unlink another user's Codex credentials")
			return
		}
		user = codexCanonicalUser(target)
	}
	if user == "" {
		r.Say("Please provide a valid username to unlink")
		return
	}
	if ret := codexDeleteAuthRecord(r, user); ret != robot.Ok {
		r.Say("Unable to unlink Codex credentials for '%s' (%s)", user, ret)
		return
	}
	r.Say("Unlinked Codex credentials for '%s'", user)
}

func codexStatus(r Robot) {
	user := codexCanonicalUser(r.User)
	if user == "" {
		r.Say("Unable to determine your user identity")
		return
	}
	_, exists, record, ret := codexGetAuthRecord(r, user)
	if ret != robot.Ok {
		r.Say("Unable to retrieve Codex auth status (%s)", ret)
		return
	}
	if !exists || strings.TrimSpace(record.AuthJSON) == "" {
		r.Say("You do not have linked Codex credentials; run '/%s link-codex'", r.GetBotAttribute("name"))
		return
	}
	ts := strings.TrimSpace(record.UpdatedAt)
	if ts == "" {
		r.Say("Codex is linked for '%s'", user)
		return
	}
	r.Say("Codex is linked for '%s' (updated %s)", user, ts)
}

func codexList(r Robot) {
	if !r.CheckAdmin() {
		r.Say("Only bot admins can list linked Codex users")
		return
	}
	users, ret := codexListAuthUsers(r)
	if ret != robot.Ok {
		r.Say("Unable to list linked Codex users (%s)", ret)
		return
	}
	if len(users) == 0 {
		r.Say("No users currently have linked Codex credentials")
		return
	}
	r.Say("Linked Codex users: %s", strings.Join(users, ", "))
}

func codexListAuthUsers(r Robot) ([]string, robot.RetVal) {
	lockToken, store, _, ret := codexCheckoutAuthStore(r, false)
	if ret != robot.Ok {
		return nil, ret
	}
	r.CheckinDatum(codexAuthDatumKey, lockToken)
	users := make([]string, 0, len(store))
	for user, record := range store {
		if strings.TrimSpace(record.AuthJSON) == "" {
			continue
		}
		users = append(users, user)
	}
	sort.Strings(users)
	return users, robot.Ok
}

func codexGetAuthRecord(r Robot, user string) (string, bool, codexUserAuthRecord, robot.RetVal) {
	lockToken, store, _, ret := codexCheckoutAuthStore(r, false)
	if ret != robot.Ok {
		return "", false, codexUserAuthRecord{}, ret
	}
	defer r.CheckinDatum(codexAuthDatumKey, lockToken)
	record, exists := store[user]
	return lockToken, exists, record, robot.Ok
}

func codexPutAuthRecord(r Robot, user string, record codexUserAuthRecord) robot.RetVal {
	lockToken, store, _, ret := codexCheckoutAuthStore(r, true)
	if ret != robot.Ok {
		return ret
	}
	store[user] = record
	ret = r.UpdateDatum(codexAuthDatumKey, lockToken, store)
	if ret != robot.Ok {
		return ret
	}
	return robot.Ok
}

func codexDeleteAuthRecord(r Robot, user string) robot.RetVal {
	lockToken, store, _, ret := codexCheckoutAuthStore(r, true)
	if ret != robot.Ok {
		return ret
	}
	delete(store, user)
	ret = r.UpdateDatum(codexAuthDatumKey, lockToken, store)
	if ret != robot.Ok {
		return ret
	}
	return robot.Ok
}

func codexCheckoutAuthStore(r Robot, rw bool) (string, codexAuthStore, bool, robot.RetVal) {
	var store codexAuthStore
	lockToken, exists, ret := r.CheckoutDatum(codexAuthDatumKey, &store, rw)
	if ret != robot.Ok {
		return "", nil, false, ret
	}
	if store == nil {
		store = codexAuthStore{}
	}
	return lockToken, store, exists, robot.Ok
}

func codexCanonicalUser(user string) string {
	return strings.ToLower(strings.TrimSpace(user))
}

func codexBinaryPath() string {
	if path := strings.TrimSpace(os.Getenv("GOPHER_CODEX_BIN")); path != "" {
		return path
	}
	return "codex"
}

func codexCommandEnv(codexHome string) []string {
	env := os.Environ()
	env = append(env,
		"CODEX_HOME="+codexHome,
		"TERM=dumb",
		"NO_COLOR=1",
	)
	return env
}

func codexCreateTempLinkHome(user string) (string, error) {
	root := filepath.Join(homePath, codexDefaultLinkStateDir, "link")
	userPart := codexSanitizePathPart(user)
	if userPart == "" {
		userPart = "user"
	}
	linkDir := filepath.Join(root, fmt.Sprintf("%s-%d", userPart, time.Now().UnixNano()))
	if err := codexPrivilegedFS("creating codex link state directory", func() error {
		if err := os.MkdirAll(linkDir, 0700); err != nil {
			return err
		}
		return os.Chmod(linkDir, 0700)
	}); err != nil {
		return "", err
	}
	return linkDir, nil
}

func codexWriteConfigFile(codexHome, body string) error {
	path := filepath.Join(codexHome, codexConfigFileName)
	return codexPrivilegedFS("writing codex config", func() error {
		return os.WriteFile(path, []byte(body), 0600)
	})
}

func codexReadAuthJSON(codexHome string) (string, error) {
	path := filepath.Join(codexHome, codexAuthFileName)
	var data []byte
	if err := codexPrivilegedFS("reading codex auth json", func() error {
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		data = b
		return nil
	}); err != nil {
		return "", err
	}
	auth := strings.TrimSpace(string(data))
	if auth == "" {
		return "", fmt.Errorf("%s is empty", path)
	}
	return auth, nil
}

func codexRemovePath(path string) {
	_ = codexPrivilegedFS("removing codex temp path", func() error {
		return os.RemoveAll(path)
	})
}

func codexPrivilegedFS(reason string, fn func() error) error {
	if !privSep {
		return fn()
	}
	raiseThreadPriv(reason)
	defer dropThreadPriv("restore unprivileged codex fs context")
	return fn()
}

func codexRunStreamingCommand(ctx context.Context, binary string, args []string, env []string, emit func(string)) error {
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Env = env

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("creating stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("creating stderr pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting command: %w", err)
	}

	lineCh := make(chan string, 64)
	var wg sync.WaitGroup
	scan := func(reader io.Reader) {
		defer wg.Done()
		scanner := bufio.NewScanner(reader)
		scanner.Buffer(make([]byte, 0, 1024), 1024*1024)
		for scanner.Scan() {
			line := codexNormalizeOutputLine(scanner.Text())
			if line == "" {
				continue
			}
			select {
			case lineCh <- line:
			case <-ctx.Done():
				return
			}
		}
	}

	wg.Add(2)
	go scan(stdoutPipe)
	go scan(stderrPipe)
	go func() {
		wg.Wait()
		close(lineCh)
	}()

	for {
		select {
		case <-ctx.Done():
			_ = cmd.Wait()
			return ctx.Err()
		case line, ok := <-lineCh:
			if !ok {
				if err := cmd.Wait(); err != nil {
					return err
				}
				return nil
			}
			emit(line)
		}
	}
}

func codexNormalizeOutputLine(line string) string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return ""
	}
	return strings.Map(func(r rune) rune {
		if r < 32 && r != '\t' {
			return -1
		}
		return r
	}, trimmed)
}

func codexSanitizePathPart(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = codexPathPartRe.ReplaceAllString(normalized, "_")
	normalized = strings.Trim(normalized, "._-")
	return normalized
}
