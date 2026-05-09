package suites

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
	"github.com/lnxjedi/gopherbot/v2/bot"
)

const (
	DefaultCaseTimeout = 14 * time.Second

	Alice = "alice"
	Bob   = "bob"
	Carol = "carol"
	David = "david"
	Erin  = "erin"

	AliceID = "u0001"
	BobID   = "u0002"
	CarolID = "u0003"
	DavidID = "u0004"
	ErinID  = "u0005"

	General  = "general"
	Random   = "random"
	BotTest  = "bottest"
	Deadzone = "deadzone"
)

type Message struct {
	MessageID string `json:"message_id"`
	User      string `json:"user"`
	Channel   string `json:"channel"`
	Text      string `json:"text"`
	Threaded  bool   `json:"threaded"`
	Hidden    bool   `json:"hidden"`
}

type ExpectedMessage struct {
	InReplyTo   string `json:"in_reply_to,omitempty"`
	User        string `json:"user"`
	Channel     string `json:"channel"`
	TextPattern string `json:"text_pattern"`
	Threaded    bool   `json:"threaded"`
}

type Case struct {
	Name        string
	Input       Message
	Replies     []ExpectedMessage
	Events      []bot.Event
	RepliesOnly bool
	Pause       time.Duration
}

type Suite struct {
	Name         string
	ConfigDir    string
	LogName      string
	FullGate     string
	Metadata     Metadata
	Capabilities map[string]robot.ConnectorCapabilities
	Cases        []Case
	BeforeStart  func() (func(), error)
	Flow         func(context.Context, Driver) []Failure
}

type Metadata struct {
	Subsystems []string `json:"subsystems,omitempty"`
	Tags       []string `json:"tags,omitempty"`
	Runtimes   []string `json:"runtimes,omitempty"`
	Tier       string   `json:"tier,omitempty"`
}

type Driver interface {
	WaitForIdle(context.Context) error
	DrainEvents(context.Context) ([]bot.Event, error)
	Send(context.Context, Message) error
	Receive(context.Context, ExpectedMessage) (Message, error)
}

type StepLogger interface {
	LogStep(suiteName, caseName, step, format string, args ...interface{})
}

type Failure struct {
	Suite    string `json:"suite"`
	Case     string `json:"case"`
	Step     string `json:"step"`
	Error    string `json:"error"`
	TimedOut bool   `json:"timed_out,omitempty"`
}

type RunOptions struct {
	CaseTimeout time.Duration
}

var registry = make(map[string]Suite)

type runOptionsContextKey struct{}

func Register(s Suite) {
	if strings.TrimSpace(s.Name) == "" {
		panic("integration suite with empty name")
	}
	if strings.TrimSpace(s.ConfigDir) == "" {
		panic(fmt.Sprintf("integration suite %q has empty config dir", s.Name))
	}
	if _, exists := registry[s.Name]; exists {
		panic(fmt.Sprintf("duplicate integration suite %q", s.Name))
	}
	registry[s.Name] = s
}

func Get(name string) (Suite, bool) {
	s, ok := registry[name]
	return s, ok
}

func MustGet(name string) Suite {
	s, ok := Get(name)
	if !ok {
		panic(fmt.Sprintf("unknown integration suite %q", name))
	}
	return s
}

func List() []Suite {
	out := make([]Suite, 0, len(registry))
	for _, s := range registry {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

func RunSuite(ctx context.Context, d Driver, s Suite) []Failure {
	return RunSuiteWithOptions(ctx, d, s, RunOptions{})
}

func RunSuiteWithOptions(ctx context.Context, d Driver, s Suite, opts RunOptions) []Failure {
	opts = normalizeRunOptions(opts)
	ctx = context.WithValue(ctx, runOptionsContextKey{}, opts)
	if s.Flow != nil {
		failures := s.Flow(ctx, d)
		for i := range failures {
			if failures[i].Suite == "" {
				failures[i].Suite = s.Name
			}
		}
		return failures
	}
	return RunCases(ctx, d, s.Name, s.Cases)
}

func RunCases(ctx context.Context, d Driver, suiteName string, cases []Case) []Failure {
	opts := runOptionsFromContext(ctx)
	failures := make([]Failure, 0)
	addFailure := func(c Case, step string, timedOut bool, format string, args ...interface{}) {
		failures = append(failures, Failure{
			Suite:    suiteName,
			Case:     c.Name,
			Step:     step,
			Error:    fmt.Sprintf(format, args...),
			TimedOut: timedOut,
		})
	}

	for i, c := range cases {
		if c.Name == "" {
			c.Name = fmt.Sprintf("case-%03d", i+1)
		}
		caseCtx, cancel := context.WithTimeout(ctx, opts.CaseTimeout)
		logStep(d, suiteName, c.Name, "case", "start timeout=%s", opts.CaseTimeout)
		if err := d.WaitForIdle(caseCtx); err != nil {
			timedOut := contextTimedOut(caseCtx, err)
			addFailure(c, "wait-before", timedOut, "%v", err)
			cancel()
			if timedOut {
				return failures
			}
			continue
		}
		if _, err := d.DrainEvents(caseCtx); err != nil {
			timedOut := contextTimedOut(caseCtx, err)
			addFailure(c, "drain-before", timedOut, "%v", err)
			cancel()
			if timedOut {
				return failures
			}
			continue
		}

		input := c.Input
		if input.MessageID == "" {
			input.MessageID = fmt.Sprintf("%s.%03d", suiteName, i+1)
		}
		if strings.HasPrefix(input.Text, "/") && !input.Hidden {
			input.Hidden = true
			input.Text = strings.TrimPrefix(input.Text, "/")
		}
		if err := d.Send(caseCtx, input); err != nil {
			timedOut := contextTimedOut(caseCtx, err)
			addFailure(c, "send", timedOut, "%v", err)
			cancel()
			if timedOut {
				return failures
			}
			continue
		}
		for _, want := range c.Replies {
			if want.InReplyTo == "" {
				want.InReplyTo = input.MessageID
			}
			got, err := d.Receive(caseCtx, want)
			if err != nil {
				timedOut := contextTimedOut(caseCtx, err)
				addFailure(c, "receive", timedOut, "timeout waiting for reply %q: %v", want.TextPattern, err)
				if timedOut {
					cancel()
					return failures
				}
				continue
			}
			if err := matchMessage(want, got); err != nil {
				addFailure(c, "reply", false, "%v", err)
			}
		}
		if !c.RepliesOnly {
			gotEvents, err := d.DrainEvents(caseCtx)
			if err != nil {
				timedOut := contextTimedOut(caseCtx, err)
				addFailure(c, "events", timedOut, "%v", err)
				if timedOut {
					cancel()
					return failures
				}
			} else if err := matchEvents(c.Events, gotEvents); err != nil {
				addFailure(c, "events", false, "%v", err)
			}
		}
		if c.Pause > 0 {
			timer := time.NewTimer(c.Pause)
			select {
			case <-caseCtx.Done():
				timer.Stop()
				timedOut := contextTimedOut(caseCtx, caseCtx.Err())
				addFailure(c, "pause", timedOut, "%v", caseCtx.Err())
				cancel()
				if timedOut {
					return failures
				}
			case <-timer.C:
			}
		}
		if err := d.WaitForIdle(caseCtx); err != nil {
			timedOut := contextTimedOut(caseCtx, err)
			addFailure(c, "wait-after", timedOut, "%v", err)
			cancel()
			if timedOut {
				return failures
			}
			continue
		}
		if _, err := d.DrainEvents(caseCtx); err != nil {
			timedOut := contextTimedOut(caseCtx, err)
			addFailure(c, "drain-after", timedOut, "%v", err)
			cancel()
			if timedOut {
				return failures
			}
		}
		cancel()
		logStep(d, suiteName, c.Name, "case", "finish")
	}
	return failures
}

func normalizeRunOptions(opts RunOptions) RunOptions {
	if opts.CaseTimeout <= 0 {
		opts.CaseTimeout = DefaultCaseTimeout
	}
	return opts
}

func runOptionsFromContext(ctx context.Context) RunOptions {
	opts, _ := ctx.Value(runOptionsContextKey{}).(RunOptions)
	return normalizeRunOptions(opts)
}

func contextTimedOut(ctx context.Context, err error) bool {
	return errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded)
}

func logStep(d Driver, suiteName, caseName, step, format string, args ...interface{}) {
	if logger, ok := d.(StepLogger); ok {
		logger.LogStep(suiteName, caseName, step, format, args...)
	}
}

func matchMessage(want ExpectedMessage, got Message) error {
	re, err := regexp.Compile(want.TextPattern)
	if err != nil {
		return fmt.Errorf("regex %q did not compile: %w", want.TextPattern, err)
	}
	if !re.MatchString(got.Text) {
		return fmt.Errorf("message regex mismatch; want %q, got %q", want.TextPattern, got.Text)
	}
	if got.User != want.User || got.Channel != want.Channel || got.Threaded != want.Threaded {
		return fmt.Errorf("message target mismatch; want u:%s,c:%s,t:%t; got u:%s,c:%s,t:%t",
			want.User, want.Channel, want.Threaded, got.User, got.Channel, got.Threaded)
	}
	return nil
}

func matchEvents(want, got []bot.Event) error {
	if len(got) != len(want) {
		return fmt.Errorf("event count mismatch; want %q, got %q", eventNames(want), eventNames(got))
	}
	for i := range want {
		if got[i] != want[i] {
			return fmt.Errorf("event mismatch; want %q, got %q", eventNames(want), eventNames(got))
		}
	}
	return nil
}

func eventNames(events []bot.Event) string {
	names := make([]string, len(events))
	for i, e := range events {
		names[i] = e.String()
	}
	return strings.Join(names, ", ")
}
