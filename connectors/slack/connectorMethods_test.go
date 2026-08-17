package slack

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
	"github.com/slack-go/slack"
)

func startTestSlackSender(t *testing.T, post func(context.Context, string, ...slack.MsgOption) (string, string, error)) (*slackConnector, func()) {
	t.Helper()
	stop := make(chan struct{})
	done := make(chan struct{})
	queue := make(chan *sendRequest, sendQueueSize)
	s := &slackConnector{
		Handler:         &testHandler{},
		maxMessageSplit: 1,
		postMessage:     post,
		retrySleep: func(context.Context, time.Duration) error {
			return nil
		},
		running:   true,
		sendQueue: queue,
		sendStop:  stop,
	}
	go s.startSendLoop(queue, stop, done)
	var once sync.Once
	cleanup := func() {
		once.Do(func() {
			s.Lock()
			s.running = false
			close(stop)
			s.Unlock()
			<-done
		})
	}
	t.Cleanup(cleanup)
	return s, cleanup
}

func sendTestSlackChannelMessage(s *slackConnector, text string) robot.RetVal {
	return s.SendProtocolChannelThreadMessage("<C123>", "", text, robot.Variable, &robot.ConnectorMessage{})
}

func TestDefaultHelpUsesEngineDefault(t *testing.T) {
	s := &slackConnector{}
	lines := s.DefaultHelp()
	if len(lines) != 0 {
		t.Fatalf("DefaultHelp() = %#v, want nil/empty to defer to engine defaults", lines)
	}
}

func TestFormatHiddenCommand(t *testing.T) {
	s := &slackConnector{slashCommand: "clu"}
	if got := s.FormatHiddenCommand("help ping"); got != "/clu help ping" {
		t.Fatalf("FormatHiddenCommand() = %q", got)
	}
}

func TestSlackSendWaitsForWebAPIAcceptance(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	s, _ := startTestSlackSender(t, func(context.Context, string, ...slack.MsgOption) (string, string, error) {
		close(entered)
		<-release
		return "C123", "123.456", nil
	})

	result := make(chan robot.RetVal, 1)
	go func() {
		result <- sendTestSlackChannelMessage(s, "hello")
	}()
	<-entered
	select {
	case ret := <-result:
		t.Fatalf("send returned %v before Slack accepted the message", ret)
	default:
	}
	close(release)
	select {
	case ret := <-result:
		if ret != robot.Ok {
			t.Fatalf("send returned %v, want %v", ret, robot.Ok)
		}
	case <-time.After(time.Second):
		t.Fatal("send did not return after Slack accepted the message")
	}
}

func TestSlackSendReturnsPermanentFailureWithoutRetry(t *testing.T) {
	var attempts atomic.Int32
	s, _ := startTestSlackSender(t, func(context.Context, string, ...slack.MsgOption) (string, string, error) {
		attempts.Add(1)
		return "", "", slack.SlackErrorResponse{Err: "channel_not_found"}
	})

	if ret := sendTestSlackChannelMessage(s, "hello"); ret != robot.FailedMessageSend {
		t.Fatalf("send returned %v, want %v", ret, robot.FailedMessageSend)
	}
	if got := attempts.Load(); got != 1 {
		t.Fatalf("attempts = %d, want 1", got)
	}
}

func TestSlackSendRetriesTransientFailureWithCorrectBackoff(t *testing.T) {
	var attempts atomic.Int32
	s, _ := startTestSlackSender(t, func(context.Context, string, ...slack.MsgOption) (string, string, error) {
		if attempts.Add(1) == 1 {
			return "", "", slack.StatusCodeError{Code: 503, Status: "service unavailable"}
		}
		return "C123", "123.456", nil
	})
	var delays []time.Duration
	s.retrySleep = func(_ context.Context, delay time.Duration) error {
		delays = append(delays, delay)
		return nil
	}

	if ret := sendTestSlackChannelMessage(s, "hello"); ret != robot.Ok {
		t.Fatalf("send returned %v, want %v", ret, robot.Ok)
	}
	if got := attempts.Load(); got != 2 {
		t.Fatalf("attempts = %d, want 2", got)
	}
	if len(delays) != 1 || delays[0] != time.Second {
		t.Fatalf("retry delays = %v, want [1s]", delays)
	}
}

func TestSlackSendReturnsFailureAfterRetryExhaustion(t *testing.T) {
	var attempts atomic.Int32
	s, _ := startTestSlackSender(t, func(context.Context, string, ...slack.MsgOption) (string, string, error) {
		attempts.Add(1)
		return "", "", slack.StatusCodeError{Code: 503, Status: "service unavailable"}
	})
	var delays []time.Duration
	s.retrySleep = func(_ context.Context, delay time.Duration) error {
		delays = append(delays, delay)
		return nil
	}

	if ret := sendTestSlackChannelMessage(s, "hello"); ret != robot.FailedMessageSend {
		t.Fatalf("send returned %v, want %v", ret, robot.FailedMessageSend)
	}
	if got := attempts.Load(); got != slackSendAttempts {
		t.Fatalf("attempts = %d, want %d", got, slackSendAttempts)
	}
	wantDelays := []time.Duration{time.Second, 2 * time.Second}
	if len(delays) != len(wantDelays) || delays[0] != wantDelays[0] || delays[1] != wantDelays[1] {
		t.Fatalf("retry delays = %v, want %v", delays, wantDelays)
	}
}

func TestSlackSendFailsWholeRequestAfterPartialChunkDelivery(t *testing.T) {
	var attempts atomic.Int32
	s, _ := startTestSlackSender(t, func(context.Context, string, ...slack.MsgOption) (string, string, error) {
		if attempts.Add(1) == 2 {
			return "", "", slack.SlackErrorResponse{Err: "invalid_blocks"}
		}
		return "C123", "123.456", nil
	})
	msgs := []slackOutgoingPayload{
		{text: "first"},
		{text: "second"},
		{text: "third"},
	}

	ret := s.sendMessages(msgs, "", "C123", "", robot.Variable, &robot.ConnectorMessage{})
	if ret != robot.FailedMessageSend {
		t.Fatalf("send returned %v, want %v", ret, robot.FailedMessageSend)
	}
	if got := attempts.Load(); got != 2 {
		t.Fatalf("attempts = %d, want 2 so no chunks follow the failure", got)
	}
}

func TestSlackSendFailsWhenQueueIsFull(t *testing.T) {
	stop := make(chan struct{})
	queue := make(chan *sendRequest, 1)
	queue <- &sendRequest{}
	s := &slackConnector{
		Handler:         &testHandler{},
		maxMessageSplit: 1,
		running:         true,
		sendQueue:       queue,
		sendStop:        stop,
	}
	defer close(stop)

	if ret := sendTestSlackChannelMessage(s, "hello"); ret != robot.FailedMessageSend {
		t.Fatalf("send returned %v, want %v", ret, robot.FailedMessageSend)
	}
}

func TestSlackSendFailsPromptlyWhenConnectorStops(t *testing.T) {
	entered := make(chan struct{})
	s, stopSender := startTestSlackSender(t, func(ctx context.Context, _ string, _ ...slack.MsgOption) (string, string, error) {
		close(entered)
		<-ctx.Done()
		return "", "", ctx.Err()
	})

	result := make(chan robot.RetVal, 1)
	go func() {
		result <- sendTestSlackChannelMessage(s, "hello")
	}()
	<-entered
	stopSender()
	select {
	case ret := <-result:
		if ret != robot.FailedMessageSend {
			t.Fatalf("send returned %v, want %v", ret, robot.FailedMessageSend)
		}
	case <-time.After(time.Second):
		t.Fatal("send remained blocked after connector stopped")
	}
}
