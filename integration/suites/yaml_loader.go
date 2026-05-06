package suites

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
	"github.com/lnxjedi/gopherbot/v2/bot"
	"gopkg.in/yaml.v3"
)

//go:embed data/*.yaml
var yamlSuiteFS embed.FS

type yamlSuite struct {
	Name         string                      `yaml:"name"`
	ConfigDir    string                      `yaml:"config_dir"`
	LogName      string                      `yaml:"log_name"`
	FullGate     string                      `yaml:"full_gate"`
	Metadata     Metadata                    `yaml:"metadata"`
	BeforeStart  string                      `yaml:"before_start"`
	Capabilities map[string]yamlCapabilities `yaml:"capabilities"`
	Cases        []yamlCase                  `yaml:"cases"`
	Flow         []yamlFlowStep              `yaml:"flow"`
}

type yamlCapabilities struct {
	HiddenCommands bool `yaml:"hidden_commands"`
}

type yamlCase struct {
	Name        string                `yaml:"name"`
	Input       yamlMessage           `yaml:"input"`
	Replies     []yamlExpectedMessage `yaml:"replies"`
	Events      []string              `yaml:"events"`
	RepliesOnly bool                  `yaml:"replies_only"`
	Pause       string                `yaml:"pause"`
}

type yamlMessage struct {
	MessageID string `yaml:"message_id"`
	User      string `yaml:"user"`
	Channel   string `yaml:"channel"`
	Text      string `yaml:"text"`
	Threaded  bool   `yaml:"threaded"`
	Hidden    bool   `yaml:"hidden"`
}

type yamlExpectedMessage struct {
	InReplyTo   string `yaml:"in_reply_to"`
	User        string `yaml:"user"`
	Channel     string `yaml:"channel"`
	TextPattern string `yaml:"text_pattern"`
	Threaded    bool   `yaml:"threaded"`
}

type yamlFlowStep struct {
	Name         string       `yaml:"name"`
	Case         *yamlCase    `yaml:"case"`
	WaitForIdle  bool         `yaml:"wait_for_idle"`
	DrainEvents  bool         `yaml:"drain_events"`
	Sleep        string       `yaml:"sleep"`
	Send         *yamlMessage `yaml:"send"`
	Receive      *yamlReceive `yaml:"receive"`
	ExpectEvents []string     `yaml:"expect_events"`
}

type yamlReceive struct {
	User               string                  `yaml:"user"`
	Channel            string                  `yaml:"channel"`
	TextPattern        string                  `yaml:"text_pattern"`
	Threaded           bool                    `yaml:"threaded"`
	SaveTextAs         string                  `yaml:"save_text_as"`
	ExactText          string                  `yaml:"exact_text"`
	Contains           []string                `yaml:"contains"`
	ContainsUpper      []string                `yaml:"contains_upper"`
	ContainsNone       []string                `yaml:"contains_none"`
	ContainsNoneUpper  []string                `yaml:"contains_none_upper"`
	CaptureRegex       map[string]string       `yaml:"capture_regex"`
	CapturePipelineWID *yamlPipelineWIDCapture `yaml:"capture_pipeline_wid"`
}

type yamlPipelineWIDCapture struct {
	Name     string `yaml:"name"`
	Pipeline string `yaml:"pipeline"`
	Command  string `yaml:"command"`
}

func init() {
	if err := loadEmbeddedYAMLSuites(); err != nil {
		panic(err)
	}
}

func loadEmbeddedYAMLSuites() error {
	entries, err := fs.ReadDir(yamlSuiteFS, "data")
	if err != nil {
		return fmt.Errorf("read YAML integration suites: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		path := "data/" + entry.Name()
		data, err := yamlSuiteFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("%s: read: %w", path, err)
		}
		var ys yamlSuite
		if err := yaml.Unmarshal(data, &ys); err != nil {
			return fmt.Errorf("%s: decode: %w", path, err)
		}
		suite, err := ys.toSuite()
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		Register(suite)
	}
	return nil
}

func (ys yamlSuite) toSuite() (Suite, error) {
	cases, err := yamlCasesToCases(ys.Cases)
	if err != nil {
		return Suite{}, err
	}
	capabilities := make(map[string]robot.ConnectorCapabilities, len(ys.Capabilities))
	for name, caps := range ys.Capabilities {
		capabilities[name] = robot.ConnectorCapabilities{HiddenCommands: caps.HiddenCommands}
	}
	suite := Suite{
		Name:         ys.Name,
		ConfigDir:    ys.ConfigDir,
		LogName:      ys.LogName,
		FullGate:     ys.FullGate,
		Metadata:     normalizeMetadata(ys.Metadata),
		Capabilities: capabilities,
		Cases:        cases,
	}
	if len(capabilities) == 0 {
		suite.Capabilities = nil
	}
	switch ys.BeforeStart {
	case "":
	case "test_http_server":
		suite.BeforeStart = withTestHTTPServer
	default:
		return Suite{}, fmt.Errorf("unknown before_start hook %q", ys.BeforeStart)
	}
	if len(ys.Flow) > 0 {
		flowSteps := append([]yamlFlowStep(nil), ys.Flow...)
		suite.Flow = func(ctx context.Context, d Driver) []Failure {
			return runYAMLFlow(ctx, d, ys.Name, flowSteps)
		}
	}
	return suite, nil
}

func yamlCasesToCases(in []yamlCase) ([]Case, error) {
	cases := make([]Case, 0, len(in))
	for _, yc := range in {
		c, err := yamlCaseToCase(yc)
		if err != nil {
			return nil, err
		}
		cases = append(cases, c)
	}
	return cases, nil
}

func yamlCaseToCase(yc yamlCase) (Case, error) {
	events, err := yamlEventsToEvents(yc.Events)
	if err != nil {
		return Case{}, err
	}
	pause, err := parseOptionalDuration(yc.Pause)
	if err != nil {
		return Case{}, err
	}
	replies := make([]ExpectedMessage, 0, len(yc.Replies))
	for _, reply := range yc.Replies {
		replies = append(replies, ExpectedMessage{
			InReplyTo:   reply.InReplyTo,
			User:        reply.User,
			Channel:     reply.Channel,
			TextPattern: reply.TextPattern,
			Threaded:    reply.Threaded,
		})
	}
	return Case{
		Name: yc.Name,
		Input: Message{
			MessageID: yc.Input.MessageID,
			User:      resolveInputUser(yc.Input.User),
			Channel:   yc.Input.Channel,
			Text:      yc.Input.Text,
			Threaded:  yc.Input.Threaded,
			Hidden:    yc.Input.Hidden,
		},
		Replies:     replies,
		Events:      events,
		RepliesOnly: yc.RepliesOnly,
		Pause:       pause,
	}, nil
}

func yamlEventsToEvents(names []string) ([]bot.Event, error) {
	events := make([]bot.Event, 0, len(names))
	for _, name := range names {
		event, ok := eventByName[name]
		if !ok {
			return nil, fmt.Errorf("unknown event %q", name)
		}
		events = append(events, event)
	}
	return events, nil
}

func parseOptionalDuration(raw string) (time.Duration, error) {
	if strings.TrimSpace(raw) == "" {
		return 0, nil
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("duration %q: %w", raw, err)
	}
	return d, nil
}

func normalizeMetadata(metadata Metadata) Metadata {
	metadata.Subsystems = normalizeLabels(metadata.Subsystems)
	metadata.Tags = normalizeLabels(metadata.Tags)
	metadata.Runtimes = normalizeLabels(metadata.Runtimes)
	metadata.Tier = normalizeLabel(metadata.Tier)
	return metadata
}

func normalizeLabels(labels []string) []string {
	if len(labels) == 0 {
		return nil
	}
	out := make([]string, 0, len(labels))
	seen := make(map[string]bool, len(labels))
	for _, label := range labels {
		label = normalizeLabel(label)
		if label == "" || seen[label] {
			continue
		}
		seen[label] = true
		out = append(out, label)
	}
	sort.Strings(out)
	return out
}

func normalizeLabel(label string) string {
	return strings.ToLower(strings.TrimSpace(label))
}

func runYAMLFlow(ctx context.Context, d Driver, suiteName string, steps []yamlFlowStep) []Failure {
	failures := make([]Failure, 0)
	vars := make(map[string]string)
	for i, step := range steps {
		caseName := step.Name
		if caseName == "" {
			caseName = fmt.Sprintf("step-%03d", i+1)
		}
		if step.Case != nil {
			c, err := yamlCaseToCase(*step.Case)
			if err != nil {
				addYAMLFlowFailure(&failures, suiteName, caseName, "case", "%v", err)
				return failures
			}
			if c.Name == "" {
				c.Name = caseName
			}
			caseFailures := RunCases(ctx, d, suiteName, []Case{c})
			if len(caseFailures) > 0 {
				failures = append(failures, caseFailures...)
				return failures
			}
		}
		if step.WaitForIdle {
			if err := d.WaitForIdle(ctx); err != nil {
				addYAMLFlowFailure(&failures, suiteName, caseName, "wait", "%v", err)
				return failures
			}
		}
		if step.DrainEvents {
			if _, err := d.DrainEvents(ctx); err != nil {
				addYAMLFlowFailure(&failures, suiteName, caseName, "drain", "%v", err)
				return failures
			}
		}
		if step.Sleep != "" {
			pause, err := parseOptionalDuration(step.Sleep)
			if err != nil {
				addYAMLFlowFailure(&failures, suiteName, caseName, "sleep", "%v", err)
				return failures
			}
			timer := time.NewTimer(pause)
			select {
			case <-ctx.Done():
				timer.Stop()
				addYAMLFlowFailure(&failures, suiteName, caseName, "sleep", "%v", ctx.Err())
				return failures
			case <-timer.C:
			}
		}
		if step.Send != nil {
			msg := yamlMessageToMessage(*step.Send)
			msg.Text = expandFlowVars(msg.Text, vars)
			if err := d.Send(ctx, msg); err != nil {
				addYAMLFlowFailure(&failures, suiteName, caseName, "send", "%v", err)
				return failures
			}
		}
		if step.Receive != nil {
			if err := runYAMLReceive(ctx, d, suiteName, caseName, step.Receive, vars, &failures); err != nil {
				return failures
			}
		}
		if len(step.ExpectEvents) > 0 {
			want, err := yamlEventsToEvents(step.ExpectEvents)
			if err != nil {
				addYAMLFlowFailure(&failures, suiteName, caseName, "events", "%v", err)
				return failures
			}
			got, err := d.DrainEvents(ctx)
			if err != nil {
				addYAMLFlowFailure(&failures, suiteName, caseName, "events", "%v", err)
				return failures
			}
			if err := matchEvents(want, got); err != nil {
				addYAMLFlowFailure(&failures, suiteName, caseName, "events", "%v", err)
				return failures
			}
		}
	}
	return failures
}

func runYAMLReceive(ctx context.Context, d Driver, suiteName, caseName string, receive *yamlReceive, vars map[string]string, failures *[]Failure) error {
	want := ExpectedMessage{
		User:        receive.User,
		Channel:     receive.Channel,
		TextPattern: expandFlowVars(receive.TextPattern, vars),
		Threaded:    receive.Threaded,
	}
	got, err := d.Receive(ctx, want)
	if err != nil {
		addYAMLFlowFailure(failures, suiteName, caseName, "reply", "%v", err)
		return err
	}
	if err := matchMessage(want, got); err != nil {
		addYAMLFlowFailure(failures, suiteName, caseName, "reply", "%v", err)
		return err
	}
	if receive.SaveTextAs != "" {
		vars[receive.SaveTextAs] = got.Text
	}
	if receive.ExactText != "" {
		wantText := expandFlowVars(receive.ExactText, vars)
		if got.Text != wantText {
			addYAMLFlowFailure(failures, suiteName, caseName, "reply", "want %q, got %q", wantText, got.Text)
		}
	}
	checkContains(failures, suiteName, caseName, got.Text, receive.Contains, false, false, vars)
	checkContains(failures, suiteName, caseName, got.Text, receive.ContainsUpper, true, false, vars)
	checkContains(failures, suiteName, caseName, got.Text, receive.ContainsNone, false, true, vars)
	checkContains(failures, suiteName, caseName, got.Text, receive.ContainsNoneUpper, true, true, vars)
	for name, pattern := range receive.CaptureRegex {
		re, err := regexp.Compile(expandFlowVars(pattern, vars))
		if err != nil {
			addYAMLFlowFailure(failures, suiteName, caseName, "capture", "regex %q did not compile: %v", pattern, err)
			return err
		}
		matches := re.FindStringSubmatch(got.Text)
		if len(matches) < 2 {
			addYAMLFlowFailure(failures, suiteName, caseName, "capture", "regex %q did not match %q", pattern, got.Text)
			return fmt.Errorf("capture %s did not match", name)
		}
		vars[name] = matches[1]
	}
	if receive.CapturePipelineWID != nil {
		capture := receive.CapturePipelineWID
		wid := findPipelineWID(got.Text, expandFlowVars(capture.Pipeline, vars), expandFlowVars(capture.Command, vars))
		if wid == "" {
			addYAMLFlowFailure(failures, suiteName, caseName, "wid", "unable to find %s wid in output: %q", capture.Pipeline, got.Text)
			return fmt.Errorf("pipeline wid not found")
		}
		name := capture.Name
		if name == "" {
			name = "wid"
		}
		vars[name] = wid
	}
	return nil
}

func yamlMessageToMessage(msg yamlMessage) Message {
	return Message{
		MessageID: msg.MessageID,
		User:      resolveInputUser(msg.User),
		Channel:   msg.Channel,
		Text:      msg.Text,
		Threaded:  msg.Threaded,
		Hidden:    msg.Hidden,
	}
}

func resolveInputUser(user string) string {
	switch user {
	case Alice:
		return AliceID
	case Bob:
		return BobID
	case Carol:
		return CarolID
	case David:
		return DavidID
	case Erin:
		return ErinID
	default:
		return user
	}
}

func addYAMLFlowFailure(failures *[]Failure, suiteName, caseName, step string, format string, args ...interface{}) {
	*failures = append(*failures, Failure{
		Suite: suiteName,
		Case:  caseName,
		Step:  step,
		Error: fmt.Sprintf(format, args...),
	})
}

func checkContains(failures *[]Failure, suiteName, caseName, text string, needles []string, upper, forbidden bool, vars map[string]string) {
	if len(needles) == 0 {
		return
	}
	haystack := text
	if upper {
		haystack = strings.ToUpper(haystack)
	}
	for _, needle := range needles {
		needle = expandFlowVars(needle, vars)
		if upper {
			needle = strings.ToUpper(needle)
		}
		found := strings.Contains(haystack, needle)
		if forbidden && found {
			addYAMLFlowFailure(failures, suiteName, caseName, "content", "unexpected %q in %q", needle, text)
		}
		if !forbidden && !found {
			addYAMLFlowFailure(failures, suiteName, caseName, "content", "missing %q in %q", needle, text)
		}
	}
}

func expandFlowVars(value string, vars map[string]string) string {
	for name, val := range vars {
		value = strings.ReplaceAll(value, "${"+name+"}", val)
	}
	return value
}

func findPipelineWID(psOutput, pipelineName, command string) string {
	needlePipeline := strings.ToUpper(pipelineName)
	needleCommand := strings.ToUpper(command)
	for _, line := range strings.Split(psOutput, "\n") {
		upperLine := strings.ToUpper(line)
		if !strings.Contains(upperLine, needlePipeline) || !strings.Contains(upperLine, needleCommand) {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) > 0 {
			return fields[0]
		}
	}
	return ""
}

var eventByName = map[string]bot.Event{
	bot.IgnoredUser.String():                bot.IgnoredUser,
	bot.BotDirectMessage.String():           bot.BotDirectMessage,
	bot.AdminCheckPassed.String():           bot.AdminCheckPassed,
	bot.AdminCheckFailed.String():           bot.AdminCheckFailed,
	bot.MultipleMatchesNoAction.String():    bot.MultipleMatchesNoAction,
	bot.AuthNoRunMisconfigured.String():     bot.AuthNoRunMisconfigured,
	bot.AuthNoRunPlugNotAvailable.String():  bot.AuthNoRunPlugNotAvailable,
	bot.AuthRanSuccess.String():             bot.AuthRanSuccess,
	bot.AuthRanFail.String():                bot.AuthRanFail,
	bot.AuthRanMechanismFailed.String():     bot.AuthRanMechanismFailed,
	bot.AuthRanFailNormal.String():          bot.AuthRanFailNormal,
	bot.AuthRanFailOther.String():           bot.AuthRanFailOther,
	bot.AuthNoRunNotFound.String():          bot.AuthNoRunNotFound,
	bot.ElevNoRunMisconfigured.String():     bot.ElevNoRunMisconfigured,
	bot.ElevNoRunNotAvailable.String():      bot.ElevNoRunNotAvailable,
	bot.ElevRanSuccess.String():             bot.ElevRanSuccess,
	bot.ElevRanFail.String():                bot.ElevRanFail,
	bot.ElevRanMechanismFailed.String():     bot.ElevRanMechanismFailed,
	bot.ElevRanFailNormal.String():          bot.ElevRanFailNormal,
	bot.ElevRanFailOther.String():           bot.ElevRanFailOther,
	bot.ElevNoRunNotFound.String():          bot.ElevNoRunNotFound,
	bot.CommandTaskRan.String():             bot.CommandTaskRan,
	bot.AmbientTaskRan.String():             bot.AmbientTaskRan,
	bot.CatchAllsRan.String():               bot.CatchAllsRan,
	bot.CatchAllTaskRan.String():            bot.CatchAllTaskRan,
	bot.TriggeredTaskRan.String():           bot.TriggeredTaskRan,
	bot.SpawnedTaskRan.String():             bot.SpawnedTaskRan,
	bot.ScheduledTaskRan.String():           bot.ScheduledTaskRan,
	bot.JobTaskRan.String():                 bot.JobTaskRan,
	bot.GoPluginRan.String():                bot.GoPluginRan,
	bot.ExternalTaskBadPath.String():        bot.ExternalTaskBadPath,
	bot.ExternalTaskBadInterpreter.String(): bot.ExternalTaskBadInterpreter,
	bot.ExternalTaskRan.String():            bot.ExternalTaskRan,
	bot.ExternalTaskStderrOutput.String():   bot.ExternalTaskStderrOutput,
	bot.ExternalTaskErrExit.String():        bot.ExternalTaskErrExit,
}
