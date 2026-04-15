package bot

import (
	"strings"
	"testing"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
)

type validationCaptureConnector struct {
	lastUser     string
	lastProtocol string
	lastMessage  string
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
func (c *validationCaptureConnector) SendProtocolUserChannelThreadMessage(string, string, string, string, string, robot.MessageFormat, *robot.ConnectorMessage) robot.RetVal {
	return robot.Ok
}
func (c *validationCaptureConnector) SendProtocolUserMessage(user, msg string, _ robot.MessageFormat, msgObject *robot.ConnectorMessage) robot.RetVal {
	c.lastUser = user
	c.lastMessage = msg
	if msgObject != nil {
		c.lastProtocol = msgObject.Protocol
	}
	return robot.Ok
}
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
	if capture.lastUser != "alice" {
		t.Fatalf("notification user = %q, want %q", capture.lastUser, "alice")
	}
	if capture.lastProtocol != "ssh" {
		t.Fatalf("notification protocol = %q, want %q", capture.lastProtocol, "ssh")
	}
	if !strings.Contains(capture.lastMessage, "googlechat user 'parsley' has internal ID 'users/1234567'") {
		t.Fatalf("notification message = %q", capture.lastMessage)
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
