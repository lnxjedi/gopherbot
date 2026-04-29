package suites

import (
	"context"
	"strings"
	"time"

	"github.com/lnxjedi/gopherbot/v2/bot"
)

func init() {
	Register(Suite{
		Name:      "TestHiddenAdminInspectCommands",
		ConfigDir: "test/membrain",
		LogName:   "bottest-admin-hidden.log",
		Flow:      hiddenAdminInspectFlow,
	})
	Register(Suite{
		Name:      "TestHiddenPSAndGetPipelineLog",
		ConfigDir: "test/membrain",
		LogName:   "bottest-admin-watchdog.log",
		Flow:      hiddenPSAndGetPipelineLogFlow,
	})
	Register(Suite{
		Name:      "TestPipelineTimeoutWarnAndKillAlerts",
		ConfigDir: "test/membrain",
		LogName:   "bottest-admin-timeout.log",
		Flow:      pipelineTimeoutWarnAndKillAlertsFlow,
	})
	Register(Suite{
		Name:      "TestPipelineFailureAlertIncludesTracebackExcerpt",
		ConfigDir: "test/membrain",
		LogName:   "bottest-admin-failure.log",
		Flow:      pipelineFailureAlertIncludesTracebackExcerptFlow,
	})
}

func hiddenAdminInspectFlow(ctx context.Context, d Driver) []Failure {
	const suiteName = "TestHiddenAdminInspectCommands"
	failures := RunCases(ctx, d, suiteName, legacyCases([]testItem{
		{aliceID, null, "dump robot", false, []TestMessage{{alice, null, "This command is only available as a hidden command.", false}}, []bot.Event{bot.BotDirectMessage, bot.AdminCheckPassed, bot.CommandTaskRan, bot.GoPluginRan}, 0},
		{aliceID, general, "/bender: dump robot", false, []TestMessage{{null, general, "HERE'S HOW I'VE BEEN CONFIGURED.*", false}}, []bot.Event{bot.AdminCheckPassed, bot.CommandTaskRan, bot.GoPluginRan}, 0},
		{aliceID, general, "/bender: dump plugin echo", false, []TestMessage{{null, general, "ALLCHANNELS.*", false}}, []bot.Event{bot.AdminCheckPassed, bot.CommandTaskRan, bot.GoPluginRan}, 0},
		{aliceID, general, "/bender: dump plugin default echo", false, []TestMessage{{null, general, "HERE'S.*", false}}, []bot.Event{bot.AdminCheckPassed, bot.CommandTaskRan, bot.GoPluginRan}, 0},
		{aliceID, general, "/bender: dump plugin junk", false, []TestMessage{{null, general, "Didn't find .* junk", false}}, []bot.Event{bot.AdminCheckPassed, bot.CommandTaskRan, bot.GoPluginRan}, 0},
	}))
	if len(failures) > 0 {
		return failures
	}
	got, err := sendAndReceive(ctx, d, Message{User: aliceID, Channel: general, Text: "bender: list plugins", Hidden: true}, ExpectedMessage{Channel: general, TextPattern: `(?s:.*)`})
	if err != nil {
		addFlowFailure(&failures, suiteName, "list-plugins", "reply", "%v", err)
		return failures
	}
	if err := containsAll(got.Text, "builtin-admin"); err != nil {
		addFlowFailure(&failures, suiteName, "list-plugins", "content", "%v", err)
	}
	if err := containsNone(got.Text, "builtin-dmadmin"); err != nil {
		addFlowFailure(&failures, suiteName, "list-plugins", "content", "%v", err)
	}
	if err := checkEvents(ctx, d, []bot.Event{bot.AdminCheckPassed, bot.CommandTaskRan, bot.GoPluginRan}); err != nil {
		addFlowFailure(&failures, suiteName, "list-plugins", "events", "%v", err)
	}
	return failures
}

func hiddenPSAndGetPipelineLogFlow(ctx context.Context, d Driver) []Failure {
	const suiteName = "TestHiddenPSAndGetPipelineLog"
	failures := make([]Failure, 0)
	if err := d.WaitForIdle(ctx); err != nil {
		addFlowFailure(&failures, suiteName, "setup", "wait", "%v", err)
		return failures
	}
	_, _ = d.DrainEvents(ctx)
	if err := d.Send(ctx, Message{User: aliceID, Channel: general, Text: "bender: admin inspect"}); err != nil {
		addFlowFailure(&failures, suiteName, "admin-inspect", "send", "%v", err)
		return failures
	}
	time.Sleep(200 * time.Millisecond)
	if err := d.Send(ctx, Message{User: aliceID, Channel: general, Text: "bender: ps", Hidden: true}); err != nil {
		addFlowFailure(&failures, suiteName, "ps", "send", "%v", err)
		return failures
	}
	psMsg, err := receiveAndMatch(ctx, d, ExpectedMessage{Channel: general, TextPattern: `(?s:.*)`})
	if err != nil {
		addFlowFailure(&failures, suiteName, "ps", "reply", "%v", err)
		return failures
	}
	upperPS := strings.ToUpper(psMsg.Text)
	if err := containsAll(upperPS, "PLUGINS", "ID", "ADMININSPECT"); err != nil {
		addFlowFailure(&failures, suiteName, "ps", "content", "%v", err)
		return failures
	}
	wid := findPipelineWID(psMsg.Text, "admininspect", "inspect")
	if wid == "" {
		addFlowFailure(&failures, suiteName, "ps", "wid", "unable to find admininspect wid in ps output: %q", psMsg.Text)
		return failures
	}
	if err := d.Send(ctx, Message{User: aliceID, Channel: general, Text: "bender: get pipeline log " + wid, Hidden: true}); err != nil {
		addFlowFailure(&failures, suiteName, "get-pipeline-log", "send", "%v", err)
		return failures
	}
	logMsg, err := receiveAndMatch(ctx, d, ExpectedMessage{Channel: general, TextPattern: `(?s:.*)`})
	if err != nil {
		addFlowFailure(&failures, suiteName, "get-pipeline-log", "reply", "%v", err)
		return failures
	}
	upperLog := strings.ToUpper(logMsg.Text)
	if err := containsAll(upperLog, "LIVE LOG FOR PIPELINE "+wid, "INSPECT STDOUT READY", "INSPECT STDERR READY"); err != nil {
		addFlowFailure(&failures, suiteName, "get-pipeline-log", "content", "%v", err)
	}
	doneMsg, err := receiveAndMatch(ctx, d, ExpectedMessage{Channel: general, TextPattern: `(?s:.*INSPECT DONE.*)`})
	if err != nil {
		addFlowFailure(&failures, suiteName, "admin-inspect", "done", "%v", err)
		return failures
	}
	if !strings.Contains(strings.ToUpper(doneMsg.Text), "INSPECT DONE") {
		addFlowFailure(&failures, suiteName, "admin-inspect", "done", "unexpected inspect completion message: %q", doneMsg.Text)
	}
	return failures
}

func pipelineTimeoutWarnAndKillAlertsFlow(ctx context.Context, d Driver) []Failure {
	const suiteName = "TestPipelineTimeoutWarnAndKillAlerts"
	failures := make([]Failure, 0)
	if err := d.WaitForIdle(ctx); err != nil {
		addFlowFailure(&failures, suiteName, "setup", "wait", "%v", err)
		return failures
	}
	_, _ = d.DrainEvents(ctx)
	if err := d.Send(ctx, Message{User: aliceID, Channel: general, Text: "bender: admin slow"}); err != nil {
		addFlowFailure(&failures, suiteName, "admin-slow", "send", "%v", err)
		return failures
	}
	warnMsg, err := receiveAndMatch(ctx, d, ExpectedMessage{Channel: general, TextPattern: `(?s:Pipeline timeout warning.*slow stdout before sleep.*)`})
	if err != nil {
		addFlowFailure(&failures, suiteName, "warning", "reply", "%v", err)
		return failures
	}
	if err := containsAll(warnMsg.Text, "Pipeline timeout warning", "slow stdout before sleep"); err != nil {
		addFlowFailure(&failures, suiteName, "warning", "content", "%v", err)
	}
	killMsg, err := receiveAndMatch(ctx, d, ExpectedMessage{Channel: general, TextPattern: `(?s:Pipeline timeout kill threshold reached.*slow stderr before sleep.*)`})
	if err != nil {
		addFlowFailure(&failures, suiteName, "kill", "reply", "%v", err)
		return failures
	}
	if err := containsAll(killMsg.Text, "Pipeline timeout kill threshold reached", "slow stderr before sleep"); err != nil {
		addFlowFailure(&failures, suiteName, "kill", "content", "%v", err)
	}
	return failures
}

func pipelineFailureAlertIncludesTracebackExcerptFlow(ctx context.Context, d Driver) []Failure {
	const suiteName = "TestPipelineFailureAlertIncludesTracebackExcerpt"
	failures := make([]Failure, 0)
	if err := d.WaitForIdle(ctx); err != nil {
		addFlowFailure(&failures, suiteName, "setup", "wait", "%v", err)
		return failures
	}
	_, _ = d.DrainEvents(ctx)
	if err := d.Send(ctx, Message{User: aliceID, Channel: general, Text: "bender: admin fail"}); err != nil {
		addFlowFailure(&failures, suiteName, "admin-fail", "send", "%v", err)
		return failures
	}
	alertMsg, err := receiveAndMatch(ctx, d, ExpectedMessage{Channel: general, TextPattern: `(?s:Pipeline failure: exit code.*RuntimeError: boom.*)`})
	if err != nil {
		addFlowFailure(&failures, suiteName, "alert", "reply", "%v", err)
		return failures
	}
	if err := containsAll(alertMsg.Text, "Pipeline failure: exit code", "RuntimeError: boom"); err != nil {
		addFlowFailure(&failures, suiteName, "alert", "content", "%v", err)
	}
	_, err = receiveAndMatch(ctx, d, ExpectedMessage{Channel: general, TextPattern: `(?s:Pipeline failed in external task 'admintimeout'.*)`})
	if err != nil {
		addFlowFailure(&failures, suiteName, "user-reply", "reply", "%v", err)
	}
	return failures
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
