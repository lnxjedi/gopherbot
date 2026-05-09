//go:build test

package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/pprof"
	"sort"
	"strings"
	"time"

	"github.com/lnxjedi/gopherbot/v2/bot"
	testc "github.com/lnxjedi/gopherbot/v2/connectors/test"
	"github.com/lnxjedi/gopherbot/v2/integration/suites"

	_ "github.com/lnxjedi/gopherbot/v2/gojobs/go-bootstrap"
	_ "github.com/lnxjedi/gopherbot/v2/goplugins/groups"
	_ "github.com/lnxjedi/gopherbot/v2/goplugins/help"
	_ "github.com/lnxjedi/gopherbot/v2/goplugins/ping"
	_ "github.com/lnxjedi/gopherbot/v2/history/file"
)

var Version = "(no version set)"
var Commit = "(not set)"

type suiteResult struct {
	Suite      string           `json:"suite"`
	Status     string           `json:"status"`
	StartedAt  string           `json:"started_at"`
	FinishedAt string           `json:"finished_at"`
	OutputDir  string           `json:"output_dir"`
	RobotDir   string           `json:"robot_dir"`
	RobotLog   string           `json:"robot_log"`
	Transcript string           `json:"transcript"`
	Goroutines string           `json:"goroutines"`
	ResultPath string           `json:"result_path"`
	Failures   []suites.Failure `json:"failures"`
}

type runOutcome struct {
	result suiteResult
	err    error
}

func main() {
	if len(os.Args) == 1 {
		printIntegrationUsage(os.Stdout)
		return
	}
	switch os.Args[1] {
	case "-h", "--help", "help":
		printIntegrationUsage(os.Stdout)
		return
	case "list-suites":
		os.Exit(listSuitesCommand(os.Args[2:]))
	case "run-suite":
		os.Exit(runSuiteCommand(os.Args[2:]))
	case "run":
		runRobot(os.Args[2:])
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown gopherbot-integration command %q\n\n", os.Args[1])
		printIntegrationUsage(os.Stderr)
		os.Exit(2)
	}
}

func runRobot(args []string) {
	bot.ProcessRegistrations()
	os.Args = append([]string{os.Args[0]}, append(args, "run")...)
	bot.Start(versionInfo())
}

func versionInfo() bot.VersionInfo {
	return bot.VersionInfo{
		Version: Version,
		Commit:  Commit,
	}
}

func isHelpArg(arg string) bool {
	return arg == "-h" || arg == "--help" || arg == "help"
}

func printIntegrationUsage(out io.Writer) {
	fmt.Fprintf(out, `gopherbot-integration runs process-backed Gopherbot integration suites.

Usage:
  gopherbot-integration <command> [arguments]

Commands:
  list-suites              List registered suite names and config directories.
  run-suite [flags] SELECT Run one or more suites by exact name, glob, comma list, or all.
  run [gopherbot flags]    Start a real robot using this integration binary.
  help                     Show this help.

Examples:
  gopherbot-integration list-suites
  gopherbot-integration run-suite TestBotName
  gopherbot-integration run-suite 'TestShFull*'
  gopherbot-integration run-suite TestLuaFullEncryptSecret,TestShFullEncryptSecret
  gopherbot-integration run -log robot.log

The suite runner uses the real engine and scripted test connector. The explicit
"run" command is available for fidelity/debugging, but no-argument invocation is
reserved for this integration CLI help.
`)
}

func printListSuitesUsage(out io.Writer) {
	fmt.Fprintf(out, `Usage:
  gopherbot-integration list-suites [flags]

Lists registered integration suite names and their config directories.

Flags:
  -json                 Print suite metadata as JSON.
  -subsystem NAME       Filter by subsystem label. Comma-separated values allowed.
  -tag NAME             Filter by tag label. Comma-separated values allowed.
  -runtime NAME         Filter by runtime label. Comma-separated values allowed.
  -tier NAME            Filter by tier label.
`)
}

func printRunSuiteUsage(out io.Writer) {
	fmt.Fprintf(out, `Usage:
  gopherbot-integration run-suite [flags] <suite-name|glob|all> [suite-name|glob...]

Selectors:
  TestBotName                         Exact suite name.
  'TestShFull*'                       Shell-style glob matched against suite names.
  TestLuaFull*,TestShFullEncrypt*     Comma-separated selector list.
all                                 All registered suites.
  subsystem:pipeline                All suites tagged with subsystem "pipeline".
  tag:hidden-commands               All suites with tag "hidden-commands".
  runtime:lua                       All suites with runtime "lua".
  tier:smoke                        All suites in tier "smoke".

Flags:
  -output-root DIR   Directory for integration artifacts (default integration/runs).
  -run-id ID         Shared run identifier for grouped suite artifacts.
  -timeout DURATION  Per-suite timeout (default 2m).
  -case-timeout DURATION
                      Per-test timeout before dumping goroutines and exiting hard (default 14s).
  -live              Print scripted interaction while running (default true).
`)
}

func listSuitesCommand(args []string) int {
	fs := flag.NewFlagSet("list-suites", flag.ContinueOnError)
	fs.Usage = func() { printListSuitesUsage(fs.Output()) }
	jsonOut := fs.Bool("json", false, "print suite metadata as JSON")
	subsystemFilter := fs.String("subsystem", "", "filter by subsystem label")
	tagFilter := fs.String("tag", "", "filter by tag label")
	runtimeFilter := fs.String("runtime", "", "filter by runtime label")
	tierFilter := fs.String("tier", "", "filter by tier label")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() > 0 {
		printListSuitesUsage(os.Stderr)
		return 2
	}
	selected, err := filterSuites(suites.List(), metadataFilters{
		Subsystems: splitSelectorValues(*subsystemFilter),
		Tags:       splitSelectorValues(*tagFilter),
		Runtimes:   splitSelectorValues(*runtimeFilter),
		Tier:       normalizeSelectorValue(*tierFilter),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	if *jsonOut {
		if err := json.NewEncoder(os.Stdout).Encode(suiteInfos(selected)); err != nil {
			fmt.Fprintf(os.Stderr, "encoding suites: %v\n", err)
			return 1
		}
		return 0
	}
	for _, suite := range selected {
		fmt.Printf("%s\t%s\n", suite.Name, suite.ConfigDir)
	}
	return 0
}

func runSuiteCommand(args []string) int {
	fs := flag.NewFlagSet("run-suite", flag.ContinueOnError)
	fs.Usage = func() { printRunSuiteUsage(fs.Output()) }
	outputRoot := fs.String("output-root", filepath.Join("integration", "runs"), "directory for integration run artifacts")
	live := fs.Bool("live", true, "print live scripted interaction")
	timeout := fs.Duration("timeout", 2*time.Minute, "per-suite timeout")
	caseTimeout := fs.Duration("case-timeout", suites.DefaultCaseTimeout, "per-test timeout")
	runID := fs.String("run-id", "", "shared run identifier for grouped suite artifacts")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() < 1 {
		printRunSuiteUsage(os.Stderr)
		return 2
	}

	root, err := findRepoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "finding repository root: %v\n", err)
		return 1
	}
	absOutputRoot := *outputRoot
	if !filepath.IsAbs(absOutputRoot) {
		absOutputRoot = filepath.Join(root, absOutputRoot)
	}

	selected, err := resolveSuiteSelectors(fs.Args())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if len(selected) > 1 {
		return runSuites(absOutputRoot, selected, *live, *timeout, *caseTimeout, *runID)
	}
	suite := selected[0]
	outcome := runOneSuite(root, absOutputRoot, suite, *live, *timeout, *caseTimeout, *runID)
	if outcome.err != nil {
		fmt.Fprintf(os.Stderr, "suite %s failed to run: %v\n", suite.Name, outcome.err)
		return 1
	}
	printSuiteSummary(outcome.result)
	if len(outcome.result.Failures) > 0 {
		return 1
	}
	return 0
}

func resolveSuiteSelectors(selectors []string) ([]suites.Suite, error) {
	expanded := make([]string, 0, len(selectors))
	for _, selector := range selectors {
		for _, part := range strings.Split(selector, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				expanded = append(expanded, part)
			}
		}
	}
	if len(expanded) == 0 {
		return nil, errors.New("no suite selectors provided")
	}
	all := suites.List()
	selected := make([]suites.Suite, 0, len(all))
	seen := make(map[string]bool)
	for _, selector := range expanded {
		matchedAny := false
		kind, value, metadataSelector := parseMetadataSelector(selector)
		if metadataSelector {
			matches, err := selectByMetadata(all, kind, value)
			if err != nil {
				return nil, err
			}
			for _, suite := range matches {
				matchedAny = true
				if seen[suite.Name] {
					continue
				}
				seen[suite.Name] = true
				selected = append(selected, suite)
			}
			if !matchedAny {
				return nil, fmt.Errorf("suite selector %q did not match any registered suites", selector)
			}
			continue
		}
		if strings.EqualFold(selector, "all") {
			selector = "*"
		}
		if suite, ok := suites.Get(selector); ok {
			matchedAny = true
			if !seen[suite.Name] {
				seen[suite.Name] = true
				selected = append(selected, suite)
			}
			continue
		}
		for _, suite := range all {
			matched, err := filepath.Match(selector, suite.Name)
			if err != nil {
				return nil, fmt.Errorf("invalid suite glob %q: %w", selector, err)
			}
			if !matched {
				continue
			}
			matchedAny = true
			if seen[suite.Name] {
				continue
			}
			seen[suite.Name] = true
			selected = append(selected, suite)
		}
		if !matchedAny {
			return nil, fmt.Errorf("suite selector %q did not match any registered suites", selector)
		}
	}
	return selected, nil
}

type metadataFilters struct {
	Subsystems []string
	Tags       []string
	Runtimes   []string
	Tier       string
}

type suiteInfo struct {
	Name      string          `json:"name"`
	ConfigDir string          `json:"config_dir"`
	Metadata  suites.Metadata `json:"metadata,omitempty"`
}

func suiteInfos(selected []suites.Suite) []suiteInfo {
	out := make([]suiteInfo, 0, len(selected))
	for _, suite := range selected {
		out = append(out, suiteInfo{
			Name:      suite.Name,
			ConfigDir: suite.ConfigDir,
			Metadata:  suite.Metadata,
		})
	}
	return out
}

func parseMetadataSelector(selector string) (kind, value string, ok bool) {
	kind, value, found := strings.Cut(selector, ":")
	if !found {
		return "", "", false
	}
	kind = normalizeSelectorValue(kind)
	value = normalizeSelectorValue(value)
	switch kind {
	case "subsystem", "tag", "runtime", "tier":
		return kind, value, value != ""
	default:
		return "", "", false
	}
}

func selectByMetadata(all []suites.Suite, kind, value string) ([]suites.Suite, error) {
	switch kind {
	case "subsystem":
		return filterSuites(all, metadataFilters{Subsystems: []string{value}})
	case "tag":
		return filterSuites(all, metadataFilters{Tags: []string{value}})
	case "runtime":
		return filterSuites(all, metadataFilters{Runtimes: []string{value}})
	case "tier":
		return filterSuites(all, metadataFilters{Tier: value})
	default:
		return nil, fmt.Errorf("unknown metadata selector %q", kind)
	}
}

func filterSuites(all []suites.Suite, filters metadataFilters) ([]suites.Suite, error) {
	selected := make([]suites.Suite, 0, len(all))
	for _, suite := range all {
		if !matchesMetadataFilters(suite.Metadata, filters) {
			continue
		}
		selected = append(selected, suite)
	}
	return selected, nil
}

func matchesMetadataFilters(metadata suites.Metadata, filters metadataFilters) bool {
	if len(filters.Subsystems) > 0 && !containsAny(metadata.Subsystems, filters.Subsystems) {
		return false
	}
	if len(filters.Tags) > 0 && !containsAny(metadata.Tags, filters.Tags) {
		return false
	}
	if len(filters.Runtimes) > 0 && !containsAny(metadata.Runtimes, filters.Runtimes) {
		return false
	}
	if filters.Tier != "" && metadata.Tier != filters.Tier {
		return false
	}
	return true
}

func containsAny(values, wants []string) bool {
	for _, want := range wants {
		for _, value := range values {
			if value == want {
				return true
			}
		}
	}
	return false
}

func splitSelectorValues(raw string) []string {
	values := make([]string, 0)
	for _, part := range strings.Split(raw, ",") {
		part = normalizeSelectorValue(part)
		if part != "" {
			values = append(values, part)
		}
	}
	sort.Strings(values)
	return values
}

func normalizeSelectorValue(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func runSuites(outputRoot string, selected []suites.Suite, live bool, timeout, caseTimeout time.Duration, runID string) int {
	if strings.TrimSpace(runID) == "" {
		runID = time.Now().UTC().Format("20060102T150405Z")
	}
	runRoot := filepath.Join(outputRoot, runID)
	code := 0
	for _, suite := range selected {
		cmdArgs := []string{
			"run-suite",
			"-output-root", outputRoot,
			"-run-id", runID,
			"-timeout", timeout.String(),
			"-case-timeout", caseTimeout.String(),
		}
		if !live {
			cmdArgs = append(cmdArgs, "-live=false")
		}
		cmdArgs = append(cmdArgs, suite.Name)
		cmd := exec.Command(os.Args[0], cmdArgs...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			code = 1
		}
	}
	fmt.Printf("Results recorded in: %s\n", runRoot)
	return code
}

func runOneSuite(root, outputRoot string, suite suites.Suite, live bool, timeout, caseTimeout time.Duration, runID string) runOutcome {
	started := time.Now().UTC()
	if strings.TrimSpace(runID) == "" {
		runID = started.Format("20060102T150405Z")
	}
	if caseTimeout <= 0 {
		caseTimeout = suites.DefaultCaseTimeout
	}
	outputDir := filepath.Join(outputRoot, runID, safeName(suite.Name))
	robotDir := filepath.Join(outputDir, "robot")
	robotLog := filepath.Join(outputDir, "robot.log")
	transcriptPath := filepath.Join(outputDir, "transcript.txt")
	goroutineDumpPath := filepath.Join(outputDir, "goroutines.txt")
	resultPath := filepath.Join(outputDir, "result.json")

	result := suiteResult{
		Suite:      suite.Name,
		Status:     "failed",
		StartedAt:  started.Format(time.RFC3339),
		OutputDir:  outputDir,
		RobotDir:   robotDir,
		RobotLog:   robotLog,
		Transcript: transcriptPath,
		Goroutines: goroutineDumpPath,
		ResultPath: resultPath,
	}
	if err := prepareRobotDir(root, suite, robotDir); err != nil {
		result.FinishedAt = time.Now().UTC().Format(time.RFC3339)
		result.ResultPath = resultPath
		return runOutcome{result: result, err: err}
	}
	restoreCapabilities, err := bot.ApplyConnectorCapabilitiesForTesting(suite.Capabilities)
	if err != nil {
		result.FinishedAt = time.Now().UTC().Format(time.RFC3339)
		result.ResultPath = resultPath
		return runOutcome{result: result, err: err}
	}
	defer restoreCapabilities()
	if suite.BeforeStart != nil {
		cleanup, err := suite.BeforeStart()
		if cleanup != nil {
			defer cleanup()
		}
		if err != nil {
			result.FinishedAt = time.Now().UTC().Format(time.RFC3339)
			result.ResultPath = resultPath
			return runOutcome{result: result, err: err}
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	testc.ResetCurrentConnector()

	liveOut := os.Stdout
	resultCh := make(chan runOutcome, 1)
	robotExited := make(chan struct{})
	shutdownStarted := make(chan struct{})
	defer close(robotExited)
	go watchSuiteTimeouts(result, resultPath, goroutineDumpPath, timeout, caseTimeout, robotExited, shutdownStarted)
	go func() {
		resultCh <- runSuiteAgainstConnector(ctx, suite, outputDir, robotDir, robotLog, transcriptPath, goroutineDumpPath, resultPath, live, liveOut, caseTimeout, shutdownStarted)
	}()

	if err := os.Chdir(robotDir); err != nil {
		return runOutcome{result: result, err: err}
	}
	setIntegrationEnv()
	os.Args = []string{os.Args[0], "--log", robotLog, "run"}

	logOut, err := os.OpenFile(robotLog, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return runOutcome{result: result, err: err}
	}
	oldStdout := os.Stdout
	os.Stdout = logOut

	bot.ProcessRegistrations()
	bot.Start(versionInfo())
	os.Stdout = oldStdout
	_ = logOut.Close()

	select {
	case outcome := <-resultCh:
		return outcome
	case <-time.After(2 * time.Second):
		result.FinishedAt = time.Now().UTC().Format(time.RFC3339)
		result.ResultPath = resultPath
		return runOutcome{result: result, err: errors.New("suite runner did not report a result after robot exit")}
	}
}

func runSuiteAgainstConnector(ctx context.Context, suite suites.Suite, outputDir, robotDir, robotLog, transcriptPath, goroutineDumpPath, resultPath string, live bool, liveOut *os.File, caseTimeout time.Duration, shutdownStarted chan<- struct{}) runOutcome {
	conn, err := testc.WaitForConnector(ctx)
	result := suiteResult{
		Suite:      suite.Name,
		Status:     "failed",
		StartedAt:  time.Now().UTC().Format(time.RFC3339),
		OutputDir:  outputDir,
		RobotDir:   robotDir,
		RobotLog:   robotLog,
		Transcript: transcriptPath,
		Goroutines: goroutineDumpPath,
		ResultPath: resultPath,
	}
	if err != nil {
		result.FinishedAt = time.Now().UTC().Format(time.RFC3339)
		_ = writeResult(resultPath, result)
		return runOutcome{result: result, err: err}
	}
	if err := bot.WaitForRobotInitialized(ctx); err != nil {
		result.FinishedAt = time.Now().UTC().Format(time.RFC3339)
		_ = writeResult(resultPath, result)
		return runOutcome{result: result, err: err}
	}
	bot.WaitForBackgroundInits()
	_ = conn.DrainBotMessages()
	_ = bot.GetEvents()

	transcript, err := os.OpenFile(transcriptPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		result.FinishedAt = time.Now().UTC().Format(time.RFC3339)
		_ = writeResult(resultPath, result)
		return runOutcome{result: result, err: err}
	}
	defer transcript.Close()

	driver := &scriptedConnectorDriver{
		conn:       conn,
		live:       live,
		liveOut:    liveOut,
		transcript: transcript,
	}
	failures := suites.RunSuiteWithOptions(ctx, driver, suite, suites.RunOptions{CaseTimeout: caseTimeout})
	result.Failures = failures
	if len(failures) == 0 {
		result.Status = "passed"
	}
	if hasTimedOutFailure(failures) {
		result.FinishedAt = time.Now().UTC().Format(time.RFC3339)
		_ = writeResult(resultPath, result)
		driver.writeTranscriptLine("!! suite timed out; dumping goroutines to %s and exiting hard\n", goroutineDumpPath)
		hardExitWithGoroutineDump(result, resultPath, goroutineDumpPath, "test case timed out")
	}

	close(shutdownStarted)
	_ = driver.Send(context.Background(), suites.Message{User: suites.AliceID, Channel: suites.General, Text: "bender quit"})
	result.FinishedAt = time.Now().UTC().Format(time.RFC3339)
	if err := writeResult(resultPath, result); err != nil {
		return runOutcome{result: result, err: err}
	}
	return runOutcome{result: result}
}

func watchSuiteTimeouts(result suiteResult, resultPath, goroutineDumpPath string, suiteTimeout, shutdownTimeout time.Duration, robotExited <-chan struct{}, shutdownStarted <-chan struct{}) {
	if suiteTimeout <= 0 {
		suiteTimeout = 2 * time.Minute
	}
	if shutdownTimeout <= 0 {
		shutdownTimeout = suites.DefaultCaseTimeout
	}
	suiteTimer := time.NewTimer(suiteTimeout)
	defer suiteTimer.Stop()
	var shutdownTimer <-chan time.Time
	for {
		select {
		case <-robotExited:
			return
		case <-shutdownStarted:
			shutdownStarted = nil
			timer := time.NewTimer(shutdownTimeout)
			defer timer.Stop()
			shutdownTimer = timer.C
		case <-suiteTimer.C:
			result.Failures = append(result.Failures, suites.Failure{
				Suite:    result.Suite,
				Case:     result.Suite,
				Step:     "suite-timeout",
				Error:    fmt.Sprintf("suite exceeded timeout %s", suiteTimeout),
				TimedOut: true,
			})
			hardExitWithGoroutineDump(result, resultPath, goroutineDumpPath, "suite timeout")
		case <-shutdownTimer:
			result.Failures = append(result.Failures, suites.Failure{
				Suite:    result.Suite,
				Case:     result.Suite,
				Step:     "shutdown",
				Error:    fmt.Sprintf("robot did not exit within %s after bender quit", shutdownTimeout),
				TimedOut: true,
			})
			hardExitWithGoroutineDump(result, resultPath, goroutineDumpPath, "shutdown timeout after bender quit")
		}
	}
}

func hasTimedOutFailure(failures []suites.Failure) bool {
	for _, failure := range failures {
		if failure.TimedOut {
			return true
		}
	}
	return false
}

func hardExitWithGoroutineDump(result suiteResult, resultPath, goroutineDumpPath, reason string) {
	result.Status = "failed"
	result.FinishedAt = time.Now().UTC().Format(time.RFC3339)
	result.Goroutines = goroutineDumpPath
	if len(result.Failures) == 0 {
		result.Failures = append(result.Failures, suites.Failure{
			Suite:    result.Suite,
			Case:     result.Suite,
			Step:     "timeout",
			Error:    reason,
			TimedOut: true,
		})
	}
	_ = os.MkdirAll(filepath.Dir(goroutineDumpPath), 0755)
	if f, err := os.OpenFile(goroutineDumpPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644); err == nil {
		_, _ = fmt.Fprintf(f, "gopherbot-integration hard exit: %s\nsuite: %s\ntime: %s\n\n", reason, result.Suite, result.FinishedAt)
		if profile := pprof.Lookup("goroutine"); profile != nil {
			_ = profile.WriteTo(f, 2)
		}
		_ = f.Close()
	}
	_ = writeResult(resultPath, result)
	fmt.Fprintf(os.Stderr, "gopherbot-integration hard exit: %s; goroutines: %s\n", reason, goroutineDumpPath)
	os.Exit(1)
}

type scriptedConnectorDriver struct {
	conn       *testc.TestConnector
	live       bool
	liveOut    *os.File
	transcript *os.File
}

func (d *scriptedConnectorDriver) WaitForIdle(ctx context.Context) error {
	d.writeTranscriptLine(".. wait for background init idle\n")
	done := make(chan struct{})
	go func() {
		bot.WaitForBackgroundInits()
		close(done)
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}

func (d *scriptedConnectorDriver) DrainEvents(ctx context.Context) ([]bot.Event, error) {
	d.writeTranscriptLine(".. drain events\n")
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	ev := bot.GetEvents()
	return append([]bot.Event(nil), (*ev)...), nil
}

func (d *scriptedConnectorDriver) Send(ctx context.Context, msg suites.Message) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	target := msg.Channel
	if target == "" {
		target = "(dm)"
	}
	prefix := ""
	if msg.Hidden {
		prefix = "/"
	}
	d.writeTranscriptLine("-> %s/%s: %s%s\n", msg.User, target, prefix, msg.Text)
	d.conn.SendBotMessage(&testc.TestMessage{
		User:     msg.User,
		Channel:  msg.Channel,
		Message:  msg.Text,
		Threaded: msg.Threaded,
		Hidden:   msg.Hidden,
	})
	return nil
}

func (d *scriptedConnectorDriver) Receive(ctx context.Context, want suites.ExpectedMessage) (suites.Message, error) {
	d.writeTranscriptLine(".. expect reply /%s/\n", want.TextPattern)
	type receiveResult struct {
		msg *testc.TestMessage
		err error
	}
	rc := make(chan receiveResult, 1)
	go func() {
		msg, err := d.conn.GetBotMessage()
		rc <- receiveResult{msg: msg, err: err}
	}()
	select {
	case <-ctx.Done():
		return suites.Message{}, ctx.Err()
	case res := <-rc:
		if res.err != nil {
			return suites.Message{}, res.err
		}
		if res.msg == nil {
			return suites.Message{}, errors.New("nil bot message")
		}
		msg := suites.Message{
			User:     res.msg.User,
			Channel:  res.msg.Channel,
			Text:     res.msg.Message,
			Threaded: res.msg.Threaded,
			Hidden:   res.msg.Hidden,
		}
		target := msg.Channel
		if target == "" {
			target = "(dm)"
		}
		d.writeTranscriptLine("<- %s/%s: %s\n", msg.User, target, msg.Text)
		return msg, nil
	}
}

func (d *scriptedConnectorDriver) LogStep(suiteName, caseName, step, format string, args ...interface{}) {
	d.writeTranscriptLine(".. %s/%s %s: %s\n", suiteName, caseName, step, fmt.Sprintf(format, args...))
}

func (d *scriptedConnectorDriver) writeTranscriptLine(format string, args ...interface{}) {
	line := fmt.Sprintf(format, args...)
	if d.transcript != nil {
		_, _ = d.transcript.WriteString(line)
	}
	if d.live {
		out := d.liveOut
		if out == nil {
			out = os.Stdout
		}
		_, _ = out.WriteString(line)
	}
}

func prepareRobotDir(root string, suite suites.Suite, robotDir string) error {
	if err := os.MkdirAll(robotDir, 0755); err != nil {
		return err
	}
	suiteDir := filepath.Join(root, suite.ConfigDir)
	for _, name := range []string{"conf", "plugins", "brain", "tasks", "lib"} {
		src := filepath.Join(suiteDir, name)
		if _, err := os.Stat(src); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return err
		}
		dst := filepath.Join(robotDir, name)
		if err := os.Symlink(src, dst); err != nil {
			return fmt.Errorf("symlinking %s -> %s: %w", dst, src, err)
		}
	}
	if err := os.MkdirAll(filepath.Join(robotDir, "workspace"), 0755); err != nil {
		return err
	}
	return nil
}

func setIntegrationEnv() {
	_ = os.Setenv("GOPHER_PROTOCOL", "test")
	_ = os.Setenv("GOPHER_ENCRYPTION_KEY", "gopherbot-integration-tests-brain-key")
	_ = os.Setenv("GOPHER_LOGLEVEL", "warn")
	_ = os.Unsetenv("GOPHER_CUSTOM_REPOSITORY")
	_ = os.Unsetenv("GOPHER_DEPLOY_KEY")
}

func findRepoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	candidates := []string{wd}
	if ex, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Dir(ex))
	}
	for _, start := range candidates {
		dir, err := filepath.Abs(start)
		if err != nil {
			continue
		}
		for {
			if fileExists(filepath.Join(dir, "go.mod")) && fileExists(filepath.Join(dir, "conf", "robot.yaml")) {
				return dir, nil
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	return "", errors.New("could not locate repository root containing go.mod and conf/robot.yaml")
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func safeName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "suite"
	}
	replacer := strings.NewReplacer("/", "_", "\\", "_", " ", "_", "\t", "_", ":", "_")
	return replacer.Replace(name)
}

func writeResult(path string, result suiteResult) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0644)
}

func printSuiteSummary(result suiteResult) {
	if len(result.Failures) == 0 {
		fmt.Printf("PASS %s\n", result.Suite)
	} else {
		fmt.Printf("FAIL %s (%d failure(s))\n", result.Suite, len(result.Failures))
		for _, failure := range result.Failures {
			fmt.Printf("  %s/%s: %s\n", failure.Case, failure.Step, failure.Error)
		}
	}
	fmt.Printf("Artifacts written to: %s\n", result.OutputDir)
}
