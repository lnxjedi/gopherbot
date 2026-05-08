package bot

import (
	"strings"
	"testing"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
)

type validationCaptureConnector struct {
	messages []validationCapturedMessage
}

type validationCapturedMessage struct {
	user     string
	protocol string
	channel  string
	thread   string
	message  string
}

func (c *validationCaptureConnector) GetProtocolUserAttribute(string, string) (string, robot.RetVal) {
	return "", robot.AttributeNotFound
}

func (c *validationCaptureConnector) MessageHeard(string, string) {}
func (c *validationCaptureConnector) DefaultHelp() []string       { return nil }
func (c *validationCaptureConnector) JoinChannel(string) robot.RetVal {
	return robot.Ok
}
func (c *validationCaptureConnector) SendProtocolChannelThreadMessage(string, string, string, robot.MessageFormat, *robot.ConnectorMessage) robot.RetVal {
	return robot.Ok
}
func (c *validationCaptureConnector) SendProtocolUserChannelThreadMessage(user, _ string, channel, thread, msg string, _ robot.MessageFormat, msgObject *robot.ConnectorMessage) robot.RetVal {
	protocol := ""
	if msgObject != nil {
		protocol = msgObject.Protocol
	}
	c.messages = append(c.messages, validationCapturedMessage{user: user, protocol: protocol, channel: channel, thread: thread, message: msg})
	return robot.Ok
}
func (c *validationCaptureConnector) SendProtocolUserMessage(user, msg string, _ robot.MessageFormat, msgObject *robot.ConnectorMessage) robot.RetVal {
	protocol := ""
	if msgObject != nil {
		protocol = msgObject.Protocol
	}
	c.messages = append(c.messages, validationCapturedMessage{user: user, protocol: protocol, message: msg})
	return robot.Ok
}
func (c *validationCaptureConnector) Reload() error       { return nil }
func (c *validationCaptureConnector) Run(<-chan struct{}) {}

func resetUserValidationRequestsForTesting() {
	userValidationRequests.Lock()
	userValidationRequests.byCode = make(map[string]userValidationRequest)
	userValidationRequests.Unlock()
}

func TestShouldAcceptIncomingUser(t *testing.T) {
	cases := []struct {
		name           string
		listed         bool
		ignoreUnlisted bool
		validated      bool
		want           bool
	}{
		{name: "listed validated strict", listed: true, ignoreUnlisted: true, validated: true, want: true},
		{name: "listed unvalidated strict", listed: true, ignoreUnlisted: true, validated: false, want: false},
		{name: "unlisted validated strict", listed: false, ignoreUnlisted: true, validated: true, want: false},
		{name: "unlisted permissive", listed: false, ignoreUnlisted: false, validated: false, want: true},
		{name: "listed unvalidated permissive still blocked", listed: true, ignoreUnlisted: false, validated: false, want: false},
		{name: "listed validated permissive", listed: true, ignoreUnlisted: false, validated: true, want: true},
	}
	for _, tc := range cases {
		if got := shouldAcceptIncomingUser(tc.listed, tc.ignoreUnlisted, tc.validated); got != tc.want {
			t.Fatalf("%s: shouldAcceptIncomingUser() = %t, want %t", tc.name, got, tc.want)
		}
	}
}

func TestConsumeIncomingUserValidationCode(t *testing.T) {
	resetUserValidationRequestsForTesting()
	t.Cleanup(resetUserValidationRequestsForTesting)

	oldConnector := interfaces.Connector
	capture := &validationCaptureConnector{}
	interfaces.Connector = capture
	t.Cleanup(func() {
		interfaces.Connector = oldConnector
	})

	code, err := issueUserValidationRequest("parsley", "alice", "ssh", time.Now())
	if err != nil {
		t.Fatalf("issueUserValidationRequest(): %v", err)
	}

	consumed := consumeIncomingUserValidationCode(&robot.ConnectorMessage{
		Protocol:      "googlechat",
		UserID:        "users/1234567",
		DirectMessage: true,
		MessageText:   code,
	})
	if !consumed {
		t.Fatal("consumeIncomingUserValidationCode() = false, want true")
	}
	if len(capture.messages) != 2 {
		t.Fatalf("captured %d messages, want 2: %#v", len(capture.messages), capture.messages)
	}
	if capture.messages[0].user != "<users/1234567>" {
		t.Fatalf("ack user = %q, want %q", capture.messages[0].user, "<users/1234567>")
	}
	if capture.messages[0].protocol != "googlechat" {
		t.Fatalf("ack protocol = %q, want %q", capture.messages[0].protocol, "googlechat")
	}
	if capture.messages[0].message != "Code accepted" {
		t.Fatalf("ack message = %q", capture.messages[0].message)
	}
	if capture.messages[1].user != "alice" {
		t.Fatalf("notification user = %q, want %q", capture.messages[1].user, "alice")
	}
	if capture.messages[1].protocol != "ssh" {
		t.Fatalf("notification protocol = %q, want %q", capture.messages[1].protocol, "ssh")
	}
	if !strings.Contains(capture.messages[1].message, "googlechat user 'parsley' has internal ID 'users/1234567'") {
		t.Fatalf("notification message = %q", capture.messages[1].message)
	}
}

func TestConsumeIncomingUserValidationCodeAcknowledgesHiddenChannelContext(t *testing.T) {
	resetUserValidationRequestsForTesting()
	t.Cleanup(resetUserValidationRequestsForTesting)

	oldConnector := interfaces.Connector
	capture := &validationCaptureConnector{}
	interfaces.Connector = capture
	t.Cleanup(func() {
		interfaces.Connector = oldConnector
	})

	code, err := issueUserValidationRequest("parsley", "alice", "ssh", time.Now())
	if err != nil {
		t.Fatalf("issueUserValidationRequest(): %v", err)
	}

	consumed := consumeIncomingUserValidationCode(&robot.ConnectorMessage{
		Protocol:        "slack",
		UserID:          "U1234567",
		UserName:        "parsley",
		ChannelID:       "C7654321",
		ThreadID:        "1700000000.000100",
		ThreadedMessage: true,
		HiddenMessage:   true,
		MessageText:     code,
	})
	if !consumed {
		t.Fatal("consumeIncomingUserValidationCode() = false, want true")
	}
	if len(capture.messages) != 2 {
		t.Fatalf("captured %d messages, want 2: %#v", len(capture.messages), capture.messages)
	}
	ack := capture.messages[0]
	if ack.user != "<U1234567>" {
		t.Fatalf("ack user = %q, want %q", ack.user, "<U1234567>")
	}
	if ack.protocol != "slack" {
		t.Fatalf("ack protocol = %q, want %q", ack.protocol, "slack")
	}
	if ack.channel != "<C7654321>" {
		t.Fatalf("ack channel = %q, want %q", ack.channel, "<C7654321>")
	}
	if ack.thread != "1700000000.000100" {
		t.Fatalf("ack thread = %q, want %q", ack.thread, "1700000000.000100")
	}
	if ack.message != "Code accepted" {
		t.Fatalf("ack message = %q", ack.message)
	}
}

func TestConsumeIncomingUserValidationCodeExpires(t *testing.T) {
	resetUserValidationRequestsForTesting()
	t.Cleanup(resetUserValidationRequestsForTesting)

	code, err := issueUserValidationRequest("parsley", "alice", "ssh", time.Now().Add(-time.Minute))
	if err != nil {
		t.Fatalf("issueUserValidationRequest(): %v", err)
	}
	if consumed := consumeIncomingUserValidationCode(&robot.ConnectorMessage{
		Protocol:      "googlechat",
		UserID:        "users/1234567",
		DirectMessage: true,
		MessageText:   code,
	}); consumed {
		t.Fatal("consumeIncomingUserValidationCode() = true for expired code, want false")
	}
}
