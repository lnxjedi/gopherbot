package bot

import (
	"reflect"
	"sync"
	"testing"

	"github.com/lnxjedi/gopherbot/robot"
)

type notifyAdminsCaptureConnector struct {
	*fakeRuntimeConnector
	mu       sync.Mutex
	users    []string
	messages []string
	returns  map[string]robot.RetVal
}

func (c *notifyAdminsCaptureConnector) SendProtocolUserMessage(user, message string, _ robot.MessageFormat, _ *robot.ConnectorMessage) robot.RetVal {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.users = append(c.users, user)
	c.messages = append(c.messages, message)
	if ret, ok := c.returns[user]; ok {
		return ret
	}
	return robot.Ok
}

func TestNotifyAdminsSendsDirectMessageToEveryAdmin(t *testing.T) {
	capture := &notifyAdminsCaptureConnector{fakeRuntimeConnector: &fakeRuntimeConnector{}}
	original := interfaces.Connector
	interfaces.Connector = capture
	t.Cleanup(func() { interfaces.Connector = original })

	r := Robot{
		Message:     &robot.Message{Incoming: &robot.ConnectorMessage{Protocol: "test"}},
		pipeContext: &pipeContext{},
		cfg:         &configuration{adminUsers: []string{"alice", "bob"}},
	}
	if ret := notifyAdmins(r, "Printer", "is on fire"); ret != robot.Normal {
		t.Fatalf("notifyAdmins() = %s, want Normal", ret)
	}

	if want := []string{"alice", "bob"}; !reflect.DeepEqual(capture.users, want) {
		t.Fatalf("notifyAdmins() users = %#v, want %#v", capture.users, want)
	}
	if want := []string{"Printer is on fire", "Printer is on fire"}; !reflect.DeepEqual(capture.messages, want) {
		t.Fatalf("notifyAdmins() messages = %#v, want %#v", capture.messages, want)
	}
}

func TestNotifyAdminsAttemptsEveryAdminAndFailsOnDeliveryError(t *testing.T) {
	capture := &notifyAdminsCaptureConnector{
		fakeRuntimeConnector: &fakeRuntimeConnector{},
		returns:              map[string]robot.RetVal{"alice": robot.FailedMessageSend},
	}
	original := interfaces.Connector
	interfaces.Connector = capture
	t.Cleanup(func() { interfaces.Connector = original })

	r := Robot{
		Message:     &robot.Message{Incoming: &robot.ConnectorMessage{Protocol: "test"}},
		pipeContext: &pipeContext{},
		cfg:         &configuration{adminUsers: []string{"alice", "bob"}},
	}
	if ret := notifyAdmins(r, "Printer is on fire"); ret != robot.Fail {
		t.Fatalf("notifyAdmins() = %s, want Fail", ret)
	}
	if want := []string{"alice", "bob"}; !reflect.DeepEqual(capture.users, want) {
		t.Fatalf("notifyAdmins() users = %#v, want %#v", capture.users, want)
	}
}

func TestNotifyAdminsRejectsEmptyMessage(t *testing.T) {
	r := Robot{Message: &robot.Message{}, pipeContext: &pipeContext{}, cfg: &configuration{adminUsers: []string{"alice"}}}
	if ret := notifyAdmins(r, "  "); ret != robot.Fail {
		t.Fatalf("notifyAdmins() = %s, want Fail", ret)
	}
}

func TestNotifyAdminsFailsWithoutConfiguredAdmins(t *testing.T) {
	r := Robot{Message: &robot.Message{}, pipeContext: &pipeContext{}, cfg: &configuration{}}
	if ret := notifyAdmins(r, "Printer is on fire"); ret != robot.Fail {
		t.Fatalf("notifyAdmins() = %s, want Fail", ret)
	}
}
