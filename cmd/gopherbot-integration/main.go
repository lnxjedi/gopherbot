//go:build test

package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
	ResultPath string           `json:"result_path"`
	Failures   []suites.Failure `json:"failures"`
}

type runOutcome struct {
	result suiteResult
	err    error
}

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "list-suites":
			os.Exit(listSuites())
		case "run-suite":
			os.Exit(runSuiteCommand(os.Args[2:]))
		}
	}

	bot.ProcessRegistrations()
	bot.Start(versionInfo())
}

func versionInfo() bot.VersionInfo {
	return bot.VersionInfo{
		Version: Version,
		Commit:  Commit,
	}
}

func listSuites() int {
	for _, suite := range suites.List() {
		fmt.Printf("%s\t%s\n", suite.Name, suite.ConfigDir)
	}
	return 0
}

func runSuiteCommand(args []string) int {
	fs := flag.NewFlagSet("run-suite", flag.ContinueOnError)
	outputRoot := fs.String("output-root", filepath.Join("integration", "runs"), "directory for integration run artifacts")
	live := fs.Bool("live", true, "print live scripted interaction")
	timeout := fs.Duration("timeout", 2*time.Minute, "per-suite timeout")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: gopherbot-integration run-suite [flags] <suite-name|all>")
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

	selector := fs.Arg(0)
	if strings.EqualFold(selector, "all") {
		return runAllSuites(absOutputRoot, *live, *timeout)
	}

	suite, ok := suites.Get(selector)
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown suite %q\n", selector)
		return 1
	}
	outcome := runOneSuite(root, absOutputRoot, suite, *live, *timeout)
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

func runAllSuites(outputRoot string, live bool, timeout time.Duration) int {
	code := 0
	for _, suite := range suites.List() {
		cmdArgs := []string{
			"run-suite",
			"-output-root", outputRoot,
			"-timeout", timeout.String(),
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
	return code
}

func runOneSuite(root, outputRoot string, suite suites.Suite, live bool, timeout time.Duration) runOutcome {
	started := time.Now().UTC()
	runID := started.Format("20060102T150405Z")
	outputDir := filepath.Join(outputRoot, runID, safeName(suite.Name))
	robotDir := filepath.Join(outputDir, "robot")
	robotLog := filepath.Join(outputDir, "robot.log")
	resultPath := filepath.Join(outputDir, "result.json")

	result := suiteResult{
		Suite:     suite.Name,
		Status:    "failed",
		StartedAt: started.Format(time.RFC3339),
		OutputDir: outputDir,
		RobotDir:  robotDir,
		RobotLog:  robotLog,
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
	go func() {
		resultCh <- runSuiteAgainstConnector(ctx, suite, outputDir, robotDir, robotLog, resultPath, live, liveOut)
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

func runSuiteAgainstConnector(ctx context.Context, suite suites.Suite, outputDir, robotDir, robotLog, resultPath string, live bool, liveOut *os.File) runOutcome {
	conn, err := testc.WaitForConnector(ctx)
	result := suiteResult{
		Suite:      suite.Name,
		Status:     "failed",
		StartedAt:  time.Now().UTC().Format(time.RFC3339),
		OutputDir:  outputDir,
		RobotDir:   robotDir,
		RobotLog:   robotLog,
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

	driver := &scriptedConnectorDriver{
		conn:    conn,
		live:    live,
		liveOut: liveOut,
	}
	failures := suites.RunSuite(ctx, driver, suite)
	result.Failures = failures
	if len(failures) == 0 {
		result.Status = "passed"
	}

	_ = driver.Send(context.Background(), suites.Message{User: suites.AliceID, Text: "quit"})
	result.FinishedAt = time.Now().UTC().Format(time.RFC3339)
	if err := writeResult(resultPath, result); err != nil {
		return runOutcome{result: result, err: err}
	}
	return runOutcome{result: result}
}

type scriptedConnectorDriver struct {
	conn    *testc.TestConnector
	live    bool
	liveOut *os.File
}

func (d *scriptedConnectorDriver) WaitForIdle(ctx context.Context) error {
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
	if d.live {
		target := msg.Channel
		if target == "" {
			target = "(dm)"
		}
		prefix := ""
		if msg.Hidden {
			prefix = "/"
		}
		fmt.Fprintf(d.output(), "-> %s/%s: %s%s\n", msg.User, target, prefix, msg.Text)
	}
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
		if d.live {
			target := msg.Channel
			if target == "" {
				target = "(dm)"
			}
			fmt.Fprintf(d.output(), "<- %s/%s: %s\n", msg.User, target, msg.Text)
		}
		return msg, nil
	}
}

func (d *scriptedConnectorDriver) output() *os.File {
	if d.liveOut != nil {
		return d.liveOut
	}
	return os.Stdout
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
