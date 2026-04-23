//go:build integration
// +build integration

package tbot_test

import (
	"strings"
	"testing"
	"time"

	. "github.com/lnxjedi/gopherbot/v2/bot"
	testc "github.com/lnxjedi/gopherbot/v2/connectors/test"
)

func mustGetBotMessage(t *testing.T, conn *testc.TestConnector) *testc.TestMessage {
	t.Helper()
	got, err := conn.GetBotMessage()
	if err != nil {
		t.Fatalf("timed out waiting for bot message: %v", err)
	}
	return got
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

func TestHiddenPSAndGetPipelineLog(t *testing.T) {
	done, conn := setup("test/membrain", "/tmp/bottest-admin-watchdog.log", t)

	WaitForBackgroundInitsForTesting()
	GetEvents()

	conn.SendBotMessage(&testc.TestMessage{aliceID, general, "bender: admin inspect", false, false})
	time.Sleep(200 * time.Millisecond)
	conn.SendBotMessage(&testc.TestMessage{aliceID, general, "bender: ps", false, true})

	psMsg := mustGetBotMessage(t, conn)
	upperPS := strings.ToUpper(psMsg.Message)
	if !strings.Contains(upperPS, "WID") || !strings.Contains(upperPS, "ADMININSPECT") {
		t.Fatalf("unexpected ps output: %q", psMsg.Message)
	}
	wid := findPipelineWID(psMsg.Message, "admininspect", "inspect")
	if wid == "" {
		t.Fatalf("unable to find admininspect wid in ps output: %q", psMsg.Message)
	}

	conn.SendBotMessage(&testc.TestMessage{aliceID, general, "bender: get pipeline log " + wid, false, true})
	logMsg := mustGetBotMessage(t, conn)
	upperLog := strings.ToUpper(logMsg.Message)
	if !strings.Contains(upperLog, "LIVE LOG FOR PIPELINE "+wid) {
		t.Fatalf("unexpected get-pipeline-log heading: %q", logMsg.Message)
	}
	if !strings.Contains(upperLog, "INSPECT STDOUT READY") {
		t.Fatalf("live log missing stdout excerpt: %q", logMsg.Message)
	}
	if !strings.Contains(upperLog, "INSPECT STDERR READY") {
		t.Fatalf("live log missing stderr excerpt: %q", logMsg.Message)
	}

	doneMsg := mustGetBotMessage(t, conn)
	if !strings.Contains(strings.ToUpper(doneMsg.Message), "INSPECT DONE") {
		t.Fatalf("unexpected inspect completion message: %q", doneMsg.Message)
	}

	teardown(t, done, conn)
}

func TestPipelineTimeoutWarnAndKillAlerts(t *testing.T) {
	done, conn := setup("test/membrain", "/tmp/bottest-admin-timeout.log", t)

	WaitForBackgroundInitsForTesting()
	GetEvents()

	conn.SendBotMessage(&testc.TestMessage{aliceID, general, "bender: admin slow", false, false})

	warnMsg := mustGetBotMessage(t, conn)
	if !strings.Contains(warnMsg.Message, "Pipeline timeout warning") {
		t.Fatalf("unexpected timeout warn message: %q", warnMsg.Message)
	}
	if !strings.Contains(warnMsg.Message, "slow stdout before sleep") {
		t.Fatalf("timeout warn missing stdout excerpt: %q", warnMsg.Message)
	}

	killMsg := mustGetBotMessage(t, conn)
	if !strings.Contains(killMsg.Message, "Pipeline timeout kill threshold reached") {
		t.Fatalf("unexpected timeout kill message: %q", killMsg.Message)
	}
	if !strings.Contains(killMsg.Message, "slow stderr before sleep") {
		t.Fatalf("timeout kill missing stderr excerpt: %q", killMsg.Message)
	}

	teardown(t, done, conn)
}

func TestPipelineFailureAlertIncludesTracebackExcerpt(t *testing.T) {
	done, conn := setup("test/membrain", "/tmp/bottest-admin-failure.log", t)

	WaitForBackgroundInitsForTesting()
	GetEvents()

	conn.SendBotMessage(&testc.TestMessage{aliceID, general, "bender: admin fail", false, false})

	alertMsg := mustGetBotMessage(t, conn)
	if !strings.Contains(alertMsg.Message, "Pipeline failure: exit code") {
		t.Fatalf("unexpected failure alert heading: %q", alertMsg.Message)
	}
	if !strings.Contains(alertMsg.Message, "RuntimeError: boom") {
		t.Fatalf("failure alert missing traceback excerpt: %q", alertMsg.Message)
	}

	replyMsg := mustGetBotMessage(t, conn)
	if !strings.Contains(replyMsg.Message, "Pipeline failed in external task 'admintimeout'") {
		t.Fatalf("unexpected user-facing failure reply: %q", replyMsg.Message)
	}

	teardown(t, done, conn)
}
