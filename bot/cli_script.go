package bot

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
	"gopkg.in/yaml.v3"
)

const cliScriptDefaultFixturePath = "conf/default-fixture.yaml"

type cliScriptOptions struct {
	inlineSource  string
	fixturePath   string
	newFixture    string
	force         bool
	kind          string
	language      string
	noInteractive bool
	workDir       string
	jsonOutput    bool
}

type cliScriptInvocation struct {
	ScriptPath string
	Language   string
	Kind       string
	TaskName   string
	WorkDir    string
	Command    string
	Args       []string
	Inline     bool
}

type cliScriptFixture struct {
	TaskName   string                                `json:"task_name,omitempty"`
	Kind       string                                `json:"kind,omitempty"`
	Language   string                                `json:"language,omitempty"`
	Command    string                                `json:"command,omitempty"`
	Args       []string                              `json:"args,omitempty"`
	WorkDir    string                                `json:"workdir,omitempty"`
	Message    cliScriptFixtureMessage               `json:"message,omitempty"`
	Bot        cliScriptFixtureBot                   `json:"bot,omitempty"`
	Admin      bool                                  `json:"admin,omitempty"`
	Elevated   bool                                  `json:"elevated,omitempty"`
	Parameters map[string]string                     `json:"parameters,omitempty"`
	Config     json.RawMessage                       `json:"config,omitempty"`
	Memory     cliScriptFixtureMemory                `json:"memory,omitempty"`
	Prompts    cliScriptFixturePrompts               `json:"prompts,omitempty"`
	Users      map[string]map[string]string          `json:"users,omitempty"`
	Identities map[string]map[string]json.RawMessage `json:"identities,omitempty"`
}

type cliScriptFixtureMessage struct {
	User            string `json:"user,omitempty"`
	ProtocolUser    string `json:"protocol_user,omitempty"`
	Channel         string `json:"channel,omitempty"`
	ProtocolChannel string `json:"protocol_channel,omitempty"`
	ThreadID        string `json:"thread_id,omitempty"`
	MessageID       string `json:"message_id,omitempty"`
	Protocol        string `json:"protocol,omitempty"`
	Text            string `json:"text,omitempty"`
	Direct          bool   `json:"direct,omitempty"`
	Threaded        bool   `json:"threaded,omitempty"`
	Hidden          bool   `json:"hidden,omitempty"`
}

type cliScriptFixtureBot struct {
	Name     string `json:"name,omitempty"`
	Alias    string `json:"alias,omitempty"`
	FullName string `json:"full_name,omitempty"`
	Contact  string `json:"contact,omitempty"`
	Email    string `json:"email,omitempty"`
	ID       string `json:"id,omitempty"`
}

type cliScriptFixtureMemory struct {
	LongTerm  map[string]json.RawMessage `json:"long_term,omitempty"`
	ShortTerm map[string]string          `json:"short_term,omitempty"`
}

type cliScriptFixturePrompts struct {
	Replies []string `json:"replies,omitempty"`
}

type cliScriptRunReport struct {
	Status     string           `json:"status"`
	Path       string           `json:"path"`
	Language   string           `json:"language"`
	Kind       string           `json:"kind"`
	TaskName   string           `json:"task_name"`
	Command    string           `json:"command,omitempty"`
	Args       []string         `json:"args,omitempty"`
	RetVal     string           `json:"ret_val"`
	RetValCode int              `json:"ret_val_code"`
	Error      string           `json:"error,omitempty"`
	Events     []cliScriptEvent `json:"events,omitempty"`
}

type cliScriptEvent struct {
	Type    string   `json:"type"`
	Method  string   `json:"method,omitempty"`
	Target  string   `json:"target,omitempty"`
	Message string   `json:"message,omitempty"`
	Prompt  string   `json:"prompt,omitempty"`
	Reply   string   `json:"reply,omitempty"`
	Level   string   `json:"level,omitempty"`
	Name    string   `json:"name,omitempty"`
	Args    []string `json:"args,omitempty"`
	RetVal  string   `json:"ret_val,omitempty"`
}

type cliLocalRobotShared struct {
	apiMu         sync.Mutex
	mu            sync.Mutex
	events        []cliScriptEvent
	fixture       cliScriptFixture
	interactive   bool
	jsonOutput    bool
	input         *bufio.Reader
	output        io.Writer
	errOutput     io.Writer
	promptReplies []string
	promptIndex   int
	environment   map[string]string
	parameters    map[string]string
	longTerm      map[string]json.RawMessage
	shortTerm     map[string]string
	rng           *rand.Rand
	workDir       string
}

type cliLocalRobot struct {
	shared  *cliLocalRobotShared
	message robot.Message
}

func processCLIScriptCommand(args []string, opts cliScriptOptions) int {
	if opts.newFixture != "" {
		if len(args) > 0 {
			fmt.Println("Error: script -new-fixture does not take script arguments")
			fmt.Println()
			printCLICommandHelp("script")
			return 2
		}
		if err := copyCLIScriptDefaultFixture(opts.newFixture, opts.force); err != nil {
			fmt.Printf("Error: %v\n", err)
			return 1
		}
		fmt.Printf("Created fixture %s\n", opts.newFixture)
		return 0
	}

	inv, fixture, cleanup, err := prepareCLIScriptInvocation(args, opts)
	if cleanup != nil {
		defer cleanup()
	}
	if err != nil {
		fmt.Println("Error:", err)
		fmt.Println()
		printCLICommandHelp("script")
		return 2
	}

	localRobot := newCLILocalRobot(fixture, inv, !opts.noInteractive, opts.jsonOutput, os.Stdin, os.Stdout, os.Stderr)
	ret, runErr := runCLIScript(inv, localRobot)

	report := cliScriptRunReport{
		Status:     "ok",
		Path:       inv.ScriptPath,
		Language:   inv.Language,
		Kind:       inv.Kind,
		TaskName:   inv.TaskName,
		Command:    inv.Command,
		Args:       append([]string(nil), inv.Args...),
		RetVal:     ret.String(),
		RetValCode: int(ret),
		Events:     localRobot.events(),
	}
	exitCode := 0
	if runErr != nil {
		report.Status = "error"
		report.Error = runErr.Error()
		exitCode = 1
	} else if ret != robot.Normal && ret != robot.Success {
		report.Status = "error"
		report.Error = fmt.Sprintf("script returned %s", ret)
		exitCode = 1
	}

	if opts.jsonOutput {
		if err := printCLIScriptRunReport(report); err != nil {
			fmt.Printf("Error: writing script report: %v\n", err)
			return 1
		}
	} else if report.Error != "" {
		fmt.Fprintf(os.Stderr, "Error: %s\n", report.Error)
	}
	return exitCode
}

func prepareCLIScriptInvocation(args []string, opts cliScriptOptions) (cliScriptInvocation, cliScriptFixture, func(), error) {
	fixture, err := loadCLIScriptFixture(opts.fixturePath)
	if err != nil {
		return cliScriptInvocation{}, fixture, nil, err
	}
	inv, err := parseCLIScriptInvocationArgs(args, opts)
	if err != nil {
		return cliScriptInvocation{}, fixture, nil, err
	}

	cleanup := func() {}
	if opts.inlineSource != "" {
		if strings.TrimSpace(opts.language) == "" && strings.TrimSpace(fixture.Language) == "" {
			return inv, fixture, cleanup, fmt.Errorf("-c requires -language lua|js|gsh|go or fixture.language")
		}
		path, remove, err := writeInlineCLIScript(opts.inlineSource, firstNonBlank(opts.language, fixture.Language))
		if err != nil {
			return inv, fixture, cleanup, err
		}
		inv.ScriptPath = path
		inv.Inline = true
		cleanup = remove
	}

	if inv.ScriptPath == "" {
		return inv, fixture, cleanup, fmt.Errorf("script requires a script path or -c <source>")
	}
	if _, err := os.Stat(inv.ScriptPath); err != nil {
		return inv, fixture, cleanup, fmt.Errorf("script path %q is not readable: %w", inv.ScriptPath, err)
	}

	languageOverride := firstNonBlank(opts.language, fixture.Language)
	language, err := normalizeCLIScriptLanguage(inv.ScriptPath, languageOverride)
	if err != nil {
		return inv, fixture, cleanup, err
	}
	inv.Language = language
	inv.Kind, err = normalizeCLIScriptKind(firstNonBlank(opts.kind, fixture.Kind, "plugin"))
	if err != nil {
		return inv, fixture, cleanup, err
	}
	if inv.TaskName == "" {
		inv.TaskName = firstNonBlank(fixture.TaskName, strings.TrimSuffix(filepath.Base(inv.ScriptPath), filepath.Ext(inv.ScriptPath)))
	}
	inv.WorkDir = firstNonBlank(opts.workDir, fixture.WorkDir)
	if inv.WorkDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return inv, fixture, cleanup, fmt.Errorf("getting current working directory: %w", err)
		}
		inv.WorkDir = cwd
	}
	inv.Command, inv.Args = mergeCLIScriptArgs(inv.Kind, inv.Args, fixture)
	if inv.Kind == "plugin" && inv.Command == "" {
		return inv, fixture, cleanup, fmt.Errorf("plugin scripts require a command argument, for example: gopherbot script %s -- _init", inv.ScriptPath)
	}
	if err := ensureCLIScriptRuntimePaths(); err != nil {
		return inv, fixture, cleanup, err
	}
	return inv, fixture, cleanup, nil
}

func parseCLIScriptInvocationArgs(args []string, opts cliScriptOptions) (cliScriptInvocation, error) {
	var inv cliScriptInvocation
	before, after, hasBoundary := splitCLIScriptBoundary(args)
	if opts.inlineSource != "" {
		if hasBoundary {
			if len(before) > 0 {
				return inv, fmt.Errorf("-c does not accept positional arguments before --")
			}
			inv.Args = after
		} else {
			inv.Args = before
		}
		return inv, nil
	}
	if len(before) == 0 {
		return inv, fmt.Errorf("missing script path")
	}
	if len(before) > 1 && hasBoundary {
		return inv, fmt.Errorf("expected one script path before --, got %d arguments", len(before))
	}
	inv.ScriptPath = before[0]
	if hasBoundary {
		inv.Args = after
		return inv, nil
	}
	inv.Args = append([]string(nil), before[1:]...)
	return inv, nil
}

func splitCLIScriptBoundary(args []string) (before, after []string, ok bool) {
	for i, arg := range args {
		if arg == "--" {
			return append([]string(nil), args[:i]...), append([]string(nil), args[i+1:]...), true
		}
	}
	return append([]string(nil), args...), nil, false
}

func mergeCLIScriptArgs(kind string, rawArgs []string, fixture cliScriptFixture) (string, []string) {
	if kind != "plugin" {
		if len(rawArgs) > 0 {
			return "", append([]string(nil), rawArgs...)
		}
		return "", append([]string(nil), fixture.Args...)
	}
	if len(rawArgs) > 0 {
		return rawArgs[0], append([]string(nil), rawArgs[1:]...)
	}
	return fixture.Command, append([]string(nil), fixture.Args...)
}

func normalizeCLIScriptKind(kind string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "", "plugin":
		return "plugin", nil
	case "job":
		return "job", nil
	case "task":
		return "task", nil
	default:
		return "", fmt.Errorf("unsupported script kind %q; use plugin, job, or task", kind)
	}
}

func loadCLIScriptFixture(path string) (cliScriptFixture, error) {
	fixture := defaultCLIScriptFixture()
	sourcePath := strings.TrimSpace(path)
	if sourcePath == "" {
		sourcePath = installedCLIScriptDefaultFixturePath()
	}
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		if strings.TrimSpace(path) == "" && os.IsNotExist(err) {
			return fixture, nil
		}
		return fixture, fmt.Errorf("reading fixture %q: %w", sourcePath, err)
	}
	if err := unmarshalCLIScriptFixture(data, sourcePath, &fixture); err != nil {
		return fixture, fmt.Errorf("parsing fixture %q: %w", sourcePath, err)
	}
	applyCLIScriptFixtureDefaults(&fixture)
	return fixture, nil
}

func installedCLIScriptDefaultFixturePath() string {
	base := installPath
	if base == "" {
		if cwd, err := os.Getwd(); err == nil {
			base = cwd
		}
	}
	return filepath.Join(base, cliScriptDefaultFixturePath)
}

func unmarshalCLIScriptFixture(data []byte, path string, fixture *cliScriptFixture) error {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		return json.Unmarshal(data, fixture)
	case ".yaml", ".yml":
		return unmarshalCLIScriptYAMLFixture(data, fixture)
	default:
		if err := json.Unmarshal(data, fixture); err == nil {
			return nil
		}
		return unmarshalCLIScriptYAMLFixture(data, fixture)
	}
}

func unmarshalCLIScriptYAMLFixture(data []byte, fixture *cliScriptFixture) error {
	var decoded interface{}
	if err := yaml.Unmarshal(data, &decoded); err != nil {
		return err
	}
	normalized := normalizeCLIScriptYAMLValue(decoded)
	jsonData, err := json.Marshal(normalized)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonData, fixture)
}

func normalizeCLIScriptYAMLValue(value interface{}) interface{} {
	switch typed := value.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(typed))
		for key, item := range typed {
			out[key] = normalizeCLIScriptYAMLValue(item)
		}
		return out
	case map[interface{}]interface{}:
		out := make(map[string]interface{}, len(typed))
		for key, item := range typed {
			out[fmt.Sprint(key)] = normalizeCLIScriptYAMLValue(item)
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(typed))
		for i, item := range typed {
			out[i] = normalizeCLIScriptYAMLValue(item)
		}
		return out
	default:
		return value
	}
}

func copyCLIScriptDefaultFixture(dest string, force bool) error {
	dest = strings.TrimSpace(dest)
	if dest == "" {
		return fmt.Errorf("-new-fixture requires a destination path")
	}
	source := installedCLIScriptDefaultFixturePath()
	data, err := os.ReadFile(source)
	if err != nil {
		return fmt.Errorf("reading installed default fixture %q: %w", source, err)
	}
	if parent := filepath.Dir(dest); parent != "." && parent != "" {
		if err := os.MkdirAll(parent, 0755); err != nil {
			return fmt.Errorf("creating fixture destination directory %q: %w", parent, err)
		}
	}
	flags := os.O_WRONLY | os.O_CREATE
	if force {
		flags |= os.O_TRUNC
	} else {
		flags |= os.O_EXCL
	}
	file, err := os.OpenFile(dest, flags, 0644)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("fixture %q already exists; use -force to overwrite", dest)
		}
		return fmt.Errorf("creating fixture %q: %w", dest, err)
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return fmt.Errorf("writing fixture %q: %w", dest, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("closing fixture %q: %w", dest, err)
	}
	return nil
}

func defaultCLIScriptFixture() cliScriptFixture {
	fixture := cliScriptFixture{}
	applyCLIScriptFixtureDefaults(&fixture)
	return fixture
}

func applyCLIScriptFixtureDefaults(fixture *cliScriptFixture) {
	if fixture.Message.User == "" {
		fixture.Message.User = "alice"
	}
	if fixture.Message.ProtocolUser == "" {
		fixture.Message.ProtocolUser = fixture.Message.User
	}
	if fixture.Message.Channel == "" && !fixture.Message.Direct {
		fixture.Message.Channel = "general"
	}
	if fixture.Message.ProtocolChannel == "" {
		fixture.Message.ProtocolChannel = fixture.Message.Channel
	}
	if fixture.Message.MessageID == "" {
		fixture.Message.MessageID = "local-message"
	}
	if fixture.Message.Protocol == "" {
		fixture.Message.Protocol = "test"
	}
	if fixture.Bot.Name == "" {
		fixture.Bot.Name = "floyd"
	}
	if fixture.Bot.Alias == "" {
		fixture.Bot.Alias = "floyd"
	}
	if fixture.Bot.FullName == "" {
		fixture.Bot.FullName = "Floyd Gopherbot"
	}
	if fixture.Bot.Contact == "" {
		fixture.Bot.Contact = fixture.Bot.Name
	}
	if fixture.Parameters == nil {
		fixture.Parameters = map[string]string{}
	}
	if _, ok := fixture.Parameters["GOPHER_ENVIRONMENT"]; !ok {
		fixture.Parameters["GOPHER_ENVIRONMENT"] = "development"
	}
	if fixture.Memory.LongTerm == nil {
		fixture.Memory.LongTerm = map[string]json.RawMessage{}
	}
	if fixture.Memory.ShortTerm == nil {
		fixture.Memory.ShortTerm = map[string]string{}
	}
	if fixture.Users == nil {
		fixture.Users = map[string]map[string]string{}
	}
	if _, ok := fixture.Users[fixture.Message.User]; !ok {
		fixture.Users[fixture.Message.User] = map[string]string{
			"name":       fixture.Message.User,
			"fullName":   fixture.Message.User,
			"firstName":  fixture.Message.User,
			"internalID": fixture.Message.ProtocolUser,
		}
	}
}

func writeInlineCLIScript(source, language string) (string, func(), error) {
	language, err := normalizeCLIScriptLanguage("inline."+strings.TrimSpace(language), language)
	if err != nil {
		return "", nil, err
	}
	ext := map[string]string{
		"lua": ".lua",
		"js":  ".js",
		"gsh": ".gsh",
		"go":  ".go",
	}[language]
	file, err := os.CreateTemp("", "gopherbot-script-*"+ext)
	if err != nil {
		return "", nil, fmt.Errorf("creating temporary inline script: %w", err)
	}
	path := file.Name()
	if _, err := file.WriteString(source); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return "", nil, fmt.Errorf("writing temporary inline script: %w", err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return "", nil, fmt.Errorf("closing temporary inline script: %w", err)
	}
	return path, func() { _ = os.Remove(path) }, nil
}

func ensureCLIScriptRuntimePaths() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting current working directory: %w", err)
	}
	if homePath == "" {
		homePath = cwd
	}
	if configFull == "" {
		configFull = filepath.Join(homePath, configPath)
	}
	if installPath == "" {
		installPath = cwd
	}
	return nil
}

func runCLIScript(inv cliScriptInvocation, r *cliLocalRobot) (robot.TaskRetVal, error) {
	env := r.environment()
	botMap := scriptBot(envMapFromList(env))
	switch inv.Language {
	case "lua":
		args := inv.scriptArgs()
		return runLuaExtensionViaRPC(inv.ScriptPath, inv.TaskName, inv.WorkDir, libPaths(), botMap, false, nil, r, args)
	case "js":
		args := inv.scriptArgs()
		return runJSExtensionViaRPC(inv.ScriptPath, inv.TaskName, inv.WorkDir, libPaths(), botMap, false, nil, r, args)
	case "gsh":
		return runGSHExtensionViaRPC(inv.ScriptPath, inv.TaskName, inv.WorkDir, env, false, nil, r, inv.scriptArgs())
	case "go":
		args := inv.scriptArgs()
		switch inv.Kind {
		case "plugin":
			return runGoPluginViaRPC(inv.ScriptPath, inv.TaskName, inv.WorkDir, env, false, false, nil, r, args)
		case "job":
			return runGoJobViaRPC(inv.ScriptPath, inv.TaskName, inv.WorkDir, env, false, false, nil, r, args)
		case "task":
			return runGoTaskViaRPC(inv.ScriptPath, inv.TaskName, inv.WorkDir, env, false, false, nil, r, args)
		}
	}
	return robot.MechanismFail, fmt.Errorf("unsupported script language %q", inv.Language)
}

func (inv cliScriptInvocation) scriptArgs() []string {
	if inv.Kind == "plugin" {
		return append([]string{inv.Command}, inv.Args...)
	}
	return append([]string(nil), inv.Args...)
}

func printCLIScriptRunReport(report cliScriptRunReport) error {
	var out bytes.Buffer
	encoder := json.NewEncoder(&out)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		return err
	}
	_, err := fmt.Print(out.String())
	return err
}

func newCLILocalRobot(fixture cliScriptFixture, inv cliScriptInvocation, interactive, jsonOutput bool, input io.Reader, output, errOutput io.Writer) *cliLocalRobot {
	if input == nil {
		input = os.Stdin
	}
	if output == nil {
		output = io.Discard
	}
	if errOutput == nil {
		errOutput = io.Discard
	}
	msg := fixture.Message
	channel := msg.Channel
	protocolChannel := msg.ProtocolChannel
	if msg.Direct {
		channel = ""
		protocolChannel = ""
	}
	connectorMessage := &robot.ConnectorMessage{
		Protocol:        msg.Protocol,
		UserName:        msg.User,
		UserID:          msg.ProtocolUser,
		ValidatedUser:   true,
		ChannelName:     channel,
		ChannelID:       protocolChannel,
		ThreadID:        msg.ThreadID,
		MessageID:       msg.MessageID,
		ThreadedMessage: msg.Threaded,
		DirectMessage:   msg.Direct,
		HiddenMessage:   msg.Hidden,
		MessageText:     msg.Text,
	}
	shared := &cliLocalRobotShared{
		fixture:       fixture,
		interactive:   interactive,
		jsonOutput:    jsonOutput,
		input:         bufio.NewReader(input),
		output:        output,
		errOutput:     errOutput,
		promptReplies: append([]string(nil), fixture.Prompts.Replies...),
		environment:   cliScriptRuntimeEnvironment(inv, connectorMessage, msg),
		parameters:    copyCLIScriptStringMap(fixture.Parameters),
		longTerm:      copyRawMessageMap(fixture.Memory.LongTerm),
		shortTerm:     copyCLIScriptStringMap(fixture.Memory.ShortTerm),
		rng:           rand.New(rand.NewSource(1)),
		workDir:       inv.WorkDir,
	}
	return &cliLocalRobot{
		shared: shared,
		message: robot.Message{
			User:            msg.User,
			ProtocolUser:    msg.ProtocolUser,
			Channel:         channel,
			ProtocolChannel: protocolChannel,
			Protocol:        getProtocol(msg.Protocol),
			Incoming:        connectorMessage,
			Format:          robot.BasicMarkdown,
		},
	}
}

func (r *cliLocalRobot) beginSerializedExternalAPICall() bool {
	r.shared.apiMu.Lock()
	return true
}

func (r *cliLocalRobot) finishSerializedExternalAPICall() {
	r.shared.apiMu.Unlock()
}

func (r *cliLocalRobot) clone() *cliLocalRobot {
	return &cliLocalRobot{
		shared:  r.shared,
		message: r.message,
	}
}

func (r *cliLocalRobot) CheckAdmin() bool {
	return r.shared.fixture.Admin
}

func (r *cliLocalRobot) Subscribe() bool {
	r.record(cliScriptEvent{Type: "pipeline", Method: "Subscribe", RetVal: robot.Ok.String()})
	return true
}

func (r *cliLocalRobot) Unsubscribe() bool {
	r.record(cliScriptEvent{Type: "pipeline", Method: "Unsubscribe", RetVal: robot.Ok.String()})
	return true
}

func (r *cliLocalRobot) Elevate(bool) bool {
	r.record(cliScriptEvent{Type: "auth", Method: "Elevate"})
	return r.shared.fixture.Elevated
}

func (r *cliLocalRobot) GetBotAttribute(a string) *robot.AttrRet {
	attrs := map[string]string{
		"name":     r.shared.fixture.Bot.Name,
		"alias":    r.shared.fixture.Bot.Alias,
		"fullName": r.shared.fixture.Bot.FullName,
		"contact":  r.shared.fixture.Bot.Contact,
		"email":    r.shared.fixture.Bot.Email,
		"id":       r.shared.fixture.Bot.ID,
	}
	if value, ok := attrs[a]; ok && value != "" {
		return &robot.AttrRet{Attribute: value, RetVal: robot.Ok}
	}
	return &robot.AttrRet{RetVal: robot.AttributeNotFound}
}

func (r *cliLocalRobot) GetUserAttribute(u, a string) *robot.AttrRet {
	user, ok := r.shared.fixture.Users[u]
	if !ok {
		return &robot.AttrRet{RetVal: robot.UserNotFound}
	}
	if value, ok := user[a]; ok && value != "" {
		return &robot.AttrRet{Attribute: value, RetVal: robot.Ok}
	}
	switch a {
	case "name":
		if value := firstNonBlank(user["name"], u); value != "" {
			return &robot.AttrRet{Attribute: value, RetVal: robot.Ok}
		}
	case "internalID":
		if value := firstNonBlank(user["internalID"], user["id"]); value != "" {
			return &robot.AttrRet{Attribute: value, RetVal: robot.Ok}
		}
	}
	return &robot.AttrRet{RetVal: robot.AttributeNotFound}
}

func (r *cliLocalRobot) GetSenderAttribute(a string) *robot.AttrRet {
	return r.GetUserAttribute(r.message.User, a)
}

func (r *cliLocalRobot) GetTaskConfig(cfgptr interface{}) robot.RetVal {
	if len(bytes.TrimSpace(r.shared.fixture.Config)) == 0 || bytes.Equal(bytes.TrimSpace(r.shared.fixture.Config), []byte("null")) {
		return robot.NoConfigFound
	}
	if cfgptr == nil {
		return robot.InvalidConfigPointer
	}
	if err := json.Unmarshal(r.shared.fixture.Config, cfgptr); err != nil {
		return robot.ConfigUnmarshalError
	}
	return robot.Ok
}

func (r *cliLocalRobot) GetHelpMetadata(query string) string {
	payload := map[string]interface{}{
		"query": query,
		"local": true,
	}
	data, _ := json.Marshal(payload)
	return string(data)
}

func (r *cliLocalRobot) GetMessage() *robot.Message {
	msg := r.message
	if r.message.Incoming != nil {
		in := *r.message.Incoming
		msg.Incoming = &in
	}
	return &msg
}

func (r *cliLocalRobot) GetParameter(name string) string {
	r.shared.mu.Lock()
	defer r.shared.mu.Unlock()
	if value, ok := r.shared.parameters[name]; ok {
		return value
	}
	return r.shared.environment[name]
}

func (r *cliLocalRobot) GetIdentityCredential(provider, user string) (*robot.IdentityCredential, robot.RetVal) {
	if providerMap, ok := r.shared.fixture.Identities[provider]; ok {
		if raw, ok := providerMap[user]; ok {
			var credential robot.IdentityCredential
			if err := json.Unmarshal(raw, &credential); err == nil {
				return &credential, robot.Ok
			}
			return nil, robot.IdentityConfigError
		}
	}
	return nil, robot.IdentityNotLinked
}

func (r *cliLocalRobot) LinkOAuth2Identity(*robot.OAuth2IdentityLinkRequest) robot.RetVal {
	r.record(cliScriptEvent{Type: "identity", Method: "LinkOAuth2Identity", RetVal: robot.Failed.String()})
	return robot.Failed
}

func (r *cliLocalRobot) UnlinkIdentity(provider, user string) robot.RetVal {
	r.record(cliScriptEvent{Type: "identity", Method: "UnlinkIdentity", Name: provider + "/" + user, RetVal: robot.Failed.String()})
	return robot.Failed
}

func (r *cliLocalRobot) Email(subject string, messageBody *bytes.Buffer, html ...bool) robot.RetVal {
	r.record(cliScriptEvent{Type: "email", Method: "Email", Target: "default", Message: subject})
	return robot.Ok
}

func (r *cliLocalRobot) EmailUser(user, subject string, messageBody *bytes.Buffer, html ...bool) robot.RetVal {
	r.record(cliScriptEvent{Type: "email", Method: "EmailUser", Target: "@" + user, Message: subject})
	return robot.Ok
}

func (r *cliLocalRobot) EmailAddress(address, subject string, messageBody *bytes.Buffer, html ...bool) robot.RetVal {
	r.record(cliScriptEvent{Type: "email", Method: "EmailAddress", Target: address, Message: subject})
	return robot.Ok
}

func (r *cliLocalRobot) Exclusive(tag string, queueTask bool) bool {
	r.record(cliScriptEvent{Type: "pipeline", Method: "Exclusive", Name: tag})
	return true
}

func (r *cliLocalRobot) Fixed() robot.Robot {
	clone := r.clone()
	clone.message.Format = robot.Fixed
	return clone
}

func (r *cliLocalRobot) MessageFormat(f robot.MessageFormat) robot.Robot {
	clone := r.clone()
	clone.message.Format = f
	return clone
}

func (r *cliLocalRobot) Direct() robot.Robot {
	clone := r.clone()
	clone.message.Channel = ""
	clone.message.ProtocolChannel = ""
	if clone.message.Incoming != nil {
		in := *clone.message.Incoming
		in.ChannelName = ""
		in.ChannelID = ""
		in.DirectMessage = true
		clone.message.Incoming = &in
	}
	return clone
}

func (r *cliLocalRobot) Threaded() robot.Robot {
	clone := r.clone()
	if clone.message.Incoming != nil {
		in := *clone.message.Incoming
		in.ThreadedMessage = true
		clone.message.Incoming = &in
	}
	return clone
}

func (r *cliLocalRobot) Log(l robot.LogLevel, m string, v ...interface{}) bool {
	msg := formatCLILocalMessage(m, v...)
	event := cliScriptEvent{Type: "log", Method: "Log", Level: l.String(), Message: msg}
	r.record(event)
	if !r.shared.jsonOutput && l >= robot.Audit {
		fmt.Fprintf(r.shared.errOutput, "log[%s]: %s\n", strings.ToLower(l.String()), msg)
	}
	return true
}

func (r *cliLocalRobot) SendChannelMessage(ch, msg string, v ...interface{}) robot.RetVal {
	return r.emitMessage("SendChannelMessage", "", ch, "", formatCLILocalMessage(msg, v...), false)
}

func (r *cliLocalRobot) SendChannelThreadMessage(ch, thr, msg string, v ...interface{}) robot.RetVal {
	return r.emitMessage("SendChannelThreadMessage", "", ch, thr, formatCLILocalMessage(msg, v...), false)
}

func (r *cliLocalRobot) SendUserChannelMessage(u, ch, msg string, v ...interface{}) robot.RetVal {
	return r.emitMessage("SendUserChannelMessage", u, ch, "", formatCLILocalMessage(msg, v...), false)
}

func (r *cliLocalRobot) SendProtocolUserChannelMessage(protocol, u, ch, msg string, v ...interface{}) robot.RetVal {
	return r.emitMessage("SendProtocolUserChannelMessage", u, ch, "", formatCLILocalMessage(msg, v...), false)
}

func (r *cliLocalRobot) SendUserChannelThreadMessage(u, ch, thr, msg string, v ...interface{}) robot.RetVal {
	return r.emitMessage("SendUserChannelThreadMessage", u, ch, thr, formatCLILocalMessage(msg, v...), false)
}

func (r *cliLocalRobot) SendUserMessage(u, msg string, v ...interface{}) robot.RetVal {
	return r.emitMessage("SendUserMessage", u, "", "", formatCLILocalMessage(msg, v...), false)
}

func (r *cliLocalRobot) Reply(msg string, v ...interface{}) robot.RetVal {
	return r.emitMessage("Reply", r.message.User, r.message.Channel, "", formatCLILocalMessage(msg, v...), true)
}

func (r *cliLocalRobot) ReplyThread(msg string, v ...interface{}) robot.RetVal {
	thread := r.threadID()
	return r.emitMessage("ReplyThread", r.message.User, r.message.Channel, thread, formatCLILocalMessage(msg, v...), true)
}

func (r *cliLocalRobot) Say(msg string, v ...interface{}) robot.RetVal {
	return r.emitMessage("Say", "", r.message.Channel, "", formatCLILocalMessage(msg, v...), false)
}

func (r *cliLocalRobot) SayThread(msg string, v ...interface{}) robot.RetVal {
	return r.emitMessage("SayThread", "", r.message.Channel, r.threadID(), formatCLILocalMessage(msg, v...), false)
}

func (r *cliLocalRobot) RandomInt(n int) int {
	if n <= 0 {
		return 0
	}
	r.shared.mu.Lock()
	defer r.shared.mu.Unlock()
	return r.shared.rng.Intn(n)
}

func (r *cliLocalRobot) RandomString(s []string) string {
	if len(s) == 0 {
		return ""
	}
	return s[r.RandomInt(len(s))]
}

func (r *cliLocalRobot) Pause(s float64) {
	if s <= 0 {
		return
	}
	time.Sleep(time.Duration(s * float64(time.Second)))
}

func (r *cliLocalRobot) PromptForReply(regexID string, prompt string, v ...interface{}) (string, robot.RetVal) {
	return r.prompt("PromptForReply", regexID, r.message.User, r.message.Channel, "", formatCLILocalMessage(prompt, v...))
}

func (r *cliLocalRobot) PromptThreadForReply(regexID string, prompt string, v ...interface{}) (string, robot.RetVal) {
	return r.prompt("PromptThreadForReply", regexID, r.message.User, r.message.Channel, r.threadID(), formatCLILocalMessage(prompt, v...))
}

func (r *cliLocalRobot) PromptUserForReply(regexID string, user string, prompt string, v ...interface{}) (string, robot.RetVal) {
	return r.prompt("PromptUserForReply", regexID, user, "", "", formatCLILocalMessage(prompt, v...))
}

func (r *cliLocalRobot) PromptUserChannelForReply(regexID string, user, channel string, prompt string, v ...interface{}) (string, robot.RetVal) {
	return r.prompt("PromptUserChannelForReply", regexID, user, channel, "", formatCLILocalMessage(prompt, v...))
}

func (r *cliLocalRobot) PromptUserChannelThreadForReply(regexID string, user, channel, thread string, prompt string, v ...interface{}) (string, robot.RetVal) {
	return r.prompt("PromptUserChannelThreadForReply", regexID, user, channel, thread, formatCLILocalMessage(prompt, v...))
}

func (r *cliLocalRobot) EncryptSecret(plaintext string) (string, robot.RetVal) {
	r.record(cliScriptEvent{Type: "secret", Method: "EncryptSecret", RetVal: robot.Failed.String()})
	return "", robot.Failed
}

func (r *cliLocalRobot) CheckoutDatum(key string, datum interface{}, rw bool) (string, bool, robot.RetVal) {
	r.shared.mu.Lock()
	defer r.shared.mu.Unlock()
	raw, exists := r.shared.longTerm[key]
	if exists && datum != nil {
		if err := json.Unmarshal(raw, datum); err != nil {
			return "", exists, robot.DataFormatError
		}
	}
	lockToken := ""
	if rw {
		lockToken = "local-lock"
	}
	return lockToken, exists, robot.Ok
}

func (r *cliLocalRobot) CheckinDatum(key, locktoken string) {
	r.record(cliScriptEvent{Type: "memory", Method: "CheckinDatum", Name: key})
}

func (r *cliLocalRobot) UpdateDatum(key, locktoken string, datum interface{}) robot.RetVal {
	raw, err := json.Marshal(datum)
	if err != nil {
		return robot.DataFormatError
	}
	r.shared.mu.Lock()
	r.shared.longTerm[key] = raw
	r.shared.mu.Unlock()
	r.record(cliScriptEvent{Type: "memory", Method: "UpdateDatum", Name: key, RetVal: robot.Ok.String()})
	return robot.Ok
}

func (r *cliLocalRobot) DeleteDatum(key string) robot.RetVal {
	r.shared.mu.Lock()
	delete(r.shared.longTerm, key)
	r.shared.mu.Unlock()
	r.record(cliScriptEvent{Type: "memory", Method: "DeleteDatum", Name: key, RetVal: robot.Ok.String()})
	return robot.Ok
}

func (r *cliLocalRobot) Remember(key, value string, shared bool) {
	r.remember(key, value, shared, false)
}

func (r *cliLocalRobot) RememberThread(key, value string, shared bool) {
	r.remember(key, value, shared, true)
}

func (r *cliLocalRobot) RememberContext(context, value string) {
	r.Remember("context:"+context, value, false)
}

func (r *cliLocalRobot) RememberContextThread(context, value string) {
	r.RememberThread("context:"+context, value, false)
}

func (r *cliLocalRobot) Recall(key string, shared bool) string {
	r.shared.mu.Lock()
	defer r.shared.mu.Unlock()
	return r.shared.shortTerm[r.shortTermKey(key, shared, false)]
}

func (r *cliLocalRobot) DeleteMemory(key string, shared bool) {
	r.shared.mu.Lock()
	delete(r.shared.shortTerm, r.shortTermKey(key, shared, false))
	r.shared.mu.Unlock()
	r.record(cliScriptEvent{Type: "memory", Method: "DeleteMemory", Name: key})
}

func (r *cliLocalRobot) SpawnJob(name string, args ...string) robot.RetVal {
	return r.pipelineEvent("SpawnJob", name, args...)
}

func (r *cliLocalRobot) AddTask(name string, args ...string) robot.RetVal {
	return r.pipelineEvent("AddTask", name, args...)
}

func (r *cliLocalRobot) FinalTask(name string, args ...string) robot.RetVal {
	return r.pipelineEvent("FinalTask", name, args...)
}

func (r *cliLocalRobot) FailTask(name string, args ...string) robot.RetVal {
	return r.pipelineEvent("FailTask", name, args...)
}

func (r *cliLocalRobot) AddJob(name string, args ...string) robot.RetVal {
	return r.pipelineEvent("AddJob", name, args...)
}

func (r *cliLocalRobot) AddCommand(plugin, command string) robot.RetVal {
	return r.pipelineEvent("AddCommand", plugin, command)
}

func (r *cliLocalRobot) FinalCommand(plugin, command string) robot.RetVal {
	return r.pipelineEvent("FinalCommand", plugin, command)
}

func (r *cliLocalRobot) FailCommand(plugin, command string) robot.RetVal {
	return r.pipelineEvent("FailCommand", plugin, command)
}

func (r *cliLocalRobot) SetParameter(name, value string) bool {
	r.shared.mu.Lock()
	r.shared.parameters[name] = value
	r.shared.mu.Unlock()
	r.record(cliScriptEvent{Type: "parameter", Method: "SetParameter", Name: name, Message: value})
	return true
}

func (r *cliLocalRobot) SetWorkingDirectory(path string) bool {
	if path == "" {
		return false
	}
	next := path
	if !filepath.IsAbs(next) {
		next = filepath.Join(r.shared.workDir, next)
	}
	info, err := os.Stat(next)
	if err != nil || !info.IsDir() {
		return false
	}
	r.shared.mu.Lock()
	r.shared.workDir = filepath.Clean(next)
	r.shared.mu.Unlock()
	r.record(cliScriptEvent{Type: "workdir", Method: "SetWorkingDirectory", Name: next})
	return true
}

func (r *cliLocalRobot) emitMessage(method, user, channel, thread, message string, mention bool) robot.RetVal {
	target := r.formatTarget(user, channel, thread, mention)
	event := cliScriptEvent{
		Type:    "message",
		Method:  method,
		Target:  target,
		Message: message,
		RetVal:  robot.Ok.String(),
	}
	r.record(event)
	if !r.shared.jsonOutput {
		fmt.Fprintf(r.shared.output, "%s: %s\n", target, message)
	}
	return robot.Ok
}

func (r *cliLocalRobot) prompt(method, regexID, user, channel, thread, prompt string) (string, robot.RetVal) {
	target := r.formatTarget(user, channel, thread, true)
	if !r.shared.jsonOutput {
		fmt.Fprintf(r.shared.output, "%s %s: ", target, strings.TrimSuffix(prompt, ": "))
	}
	reply, fromFixture, ok := r.nextPromptReply()
	if !ok {
		r.record(cliScriptEvent{Type: "prompt", Method: method, Target: target, Prompt: prompt, RetVal: robot.TimeoutExpired.String()})
		return "", robot.TimeoutExpired
	}
	if !r.shared.jsonOutput && fromFixture {
		fmt.Fprintln(r.shared.output, reply)
	}
	ret := robot.Ok
	switch reply {
	case "-":
		ret = robot.Interrupted
	case "=":
		ret = robot.UseDefaultValue
	}
	r.record(cliScriptEvent{Type: "prompt", Method: method, Target: target, Prompt: prompt, Reply: reply, Name: regexID, RetVal: ret.String()})
	return reply, ret
}

func (r *cliLocalRobot) nextPromptReply() (string, bool, bool) {
	r.shared.mu.Lock()
	if r.shared.promptIndex < len(r.shared.promptReplies) {
		reply := r.shared.promptReplies[r.shared.promptIndex]
		r.shared.promptIndex++
		r.shared.mu.Unlock()
		return reply, true, true
	}
	interactive := r.shared.interactive
	r.shared.mu.Unlock()
	if !interactive {
		return "", false, false
	}
	line, err := r.shared.input.ReadString('\n')
	if err != nil && len(line) == 0 {
		return "", false, false
	}
	return strings.TrimRight(line, "\r\n"), false, true
}

func (r *cliLocalRobot) remember(key, value string, shared, threaded bool) {
	r.shared.mu.Lock()
	r.shared.shortTerm[r.shortTermKey(key, shared, threaded)] = value
	r.shared.mu.Unlock()
	r.record(cliScriptEvent{Type: "memory", Method: "Remember", Name: key, Message: value})
}

func (r *cliLocalRobot) pipelineEvent(method, name string, args ...string) robot.RetVal {
	r.record(cliScriptEvent{Type: "pipeline", Method: method, Name: name, Args: append([]string(nil), args...), RetVal: robot.Ok.String()})
	return robot.Ok
}

func (r *cliLocalRobot) record(event cliScriptEvent) {
	r.shared.mu.Lock()
	defer r.shared.mu.Unlock()
	r.shared.events = append(r.shared.events, event)
}

func (r *cliLocalRobot) events() []cliScriptEvent {
	r.shared.mu.Lock()
	defer r.shared.mu.Unlock()
	return append([]cliScriptEvent(nil), r.shared.events...)
}

func (r *cliLocalRobot) formatTarget(user, channel, thread string, mention bool) string {
	target := ""
	if channel != "" {
		target = "#" + strings.TrimPrefix(channel, "#")
	}
	if user != "" {
		userText := "@" + strings.TrimPrefix(user, "@")
		if target == "" {
			target = userText
		} else if mention {
			target += " " + userText
		}
	}
	if target == "" {
		target = "@" + strings.TrimPrefix(r.message.User, "@")
	}
	if thread != "" {
		target += "[thread:" + thread + "]"
	}
	return target
}

func (r *cliLocalRobot) threadID() string {
	if r.message.Incoming != nil && r.message.Incoming.ThreadID != "" {
		return r.message.Incoming.ThreadID
	}
	return "local-thread"
}

func (r *cliLocalRobot) shortTermKey(key string, shared, threaded bool) string {
	scope := "local"
	if shared {
		scope = "shared"
	} else if r.message.User != "" {
		scope = r.message.User
	}
	channel := r.message.Channel
	if threaded {
		channel += "/" + r.threadID()
	}
	return scope + ":" + channel + ":" + key
}

func (r *cliLocalRobot) environment() []string {
	r.shared.mu.Lock()
	env := copyCLIScriptStringMap(r.shared.environment)
	for key, value := range r.shared.parameters {
		env[key] = value
	}
	r.shared.mu.Unlock()
	out := make([]string, 0, len(env))
	for key, value := range env {
		out = append(out, key+"="+value)
	}
	return out
}

func cliScriptRuntimeEnvironment(inv cliScriptInvocation, incoming *robot.ConnectorMessage, msg cliScriptFixtureMessage) map[string]string {
	channel := msg.Channel
	protocolChannel := msg.ProtocolChannel
	if msg.Direct {
		channel = ""
		protocolChannel = ""
	}
	env := map[string]string{
		"GOPHER_HOME":          homePath,
		"GOPHER_CONFIGDIR":     configFull,
		"GOPHER_CHANNEL":       channel,
		"GOPHER_CHANNEL_ID":    protocolChannel,
		"GOPHER_MESSAGE_ID":    firstNonBlank(msg.MessageID, "local-message"),
		"GOPHER_THREAD_ID":     msg.ThreadID,
		"GOPHER_CMDMODE":       "local",
		"GOPHER_USER":          msg.User,
		"GOPHER_USER_ID":       msg.ProtocolUser,
		"GOPHER_PROTOCOL":      msg.Protocol,
		"GOPHER_TASK_NAME":     inv.TaskName,
		"GOPHER_PIPELINE_TYPE": inv.Kind,
		"GOPHER_CALLER_ID":     "stdin",
		"GOPHER_INSTALLDIR":    installPath,
		"GOPHER_ENVIRONMENT":   getEnv("GOPHER_ENVIRONMENT"),
		"GOPHER_BRAIN":         "local-fixture",
		"RUBYLIB":              fmt.Sprintf("%s/lib:%s/lib", installPath, configFull),
		"JULIA_LOAD_PATH":      fmt.Sprintf("%s/lib:%s/lib:", installPath, configFull),
		"GEM_HOME":             filepath.Join(homePath, ".bot-gems"),
		"PYTHONPATH":           fmt.Sprintf("%s/lib:%s/lib", installPath, configFull),
		"PYTHONUSERBASE":       filepath.Join(homePath, ".bot-python"),
	}
	if incoming != nil {
		if incoming.ThreadedMessage {
			env["GOPHER_THREADED_MESSAGE"] = "true"
		}
		if incoming.HiddenMessage {
			env["GOPHER_HIDDEN_COMMAND"] = "true"
		}
		if incoming.DirectMessage {
			env["GOPHER_PRIVATE_COMMAND"] = "true"
		}
	}
	return env
}

func envMapFromList(env []string) map[string]string {
	m := make(map[string]string, len(env))
	for _, item := range env {
		key, value, ok := strings.Cut(item, "=")
		if ok && key != "" {
			m[key] = value
		}
	}
	return m
}

func formatCLILocalMessage(msg string, v ...interface{}) string {
	if len(v) == 0 {
		return msg
	}
	return fmt.Sprintf(msg, v...)
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func copyCLIScriptStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func copyRawMessageMap(in map[string]json.RawMessage) map[string]json.RawMessage {
	out := make(map[string]json.RawMessage, len(in))
	for key, value := range in {
		out[key] = append(json.RawMessage(nil), value...)
	}
	return out
}
