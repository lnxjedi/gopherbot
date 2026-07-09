package bot

import (
	"testing"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
)

type startupGateCaptureConnector struct {
	user     string
	username string
	channel  string
	thread   string
	msg      string
	format   robot.MessageFormat
	protocol string
	sends    int
}

func (c *startupGateCaptureConnector) GetProtocolUserAttribute(user, attr string) (string, robot.RetVal) {
	return "", robot.AttributeNotFound
}
func (c *startupGateCaptureConnector) MessageHeard(user, channel string)  {}
func (c *startupGateCaptureConnector) DefaultHelp() []string              { return nil }
func (c *startupGateCaptureConnector) JoinChannel(ch string) robot.RetVal { return robot.Ok }
func (c *startupGateCaptureConnector) SendProtocolChannelThreadMessage(channel, thread, msg string, format robot.MessageFormat, msgObject *robot.ConnectorMessage) robot.RetVal {
	c.channel = channel
	c.thread = thread
	c.msg = msg
	c.format = format
	c.sends++
	if msgObject != nil {
		c.protocol = msgObject.Protocol
	}
	return robot.Ok
}
func (c *startupGateCaptureConnector) SendProtocolUserChannelThreadMessage(user, username, channel, thread, msg string, format robot.MessageFormat, msgObject *robot.ConnectorMessage) robot.RetVal {
	c.user = user
	c.username = username
	c.channel = channel
	c.thread = thread
	c.msg = msg
	c.format = format
	c.sends++
	if msgObject != nil {
		c.protocol = msgObject.Protocol
	}
	return robot.Ok
}

func TestReadyMessageSkippedWhenUnset(t *testing.T) {
	conn := &startupGateCaptureConnector{}
	oldConnector := interfaces.Connector
	interfaces.Connector = conn
	defer func() { interfaces.Connector = oldConnector }()

	currentCfg.Lock()
	oldCfg := currentCfg.configuration
	currentCfg.configuration = &configuration{
		defaultJobChannel:    "jobs",
		defaultProtocol:      "test",
		defaultMessageFormat: robot.BasicMarkdown,
	}
	currentCfg.Unlock()
	defer func() {
		currentCfg.Lock()
		currentCfg.configuration = oldCfg
		currentCfg.Unlock()
	}()

	sendReadyMessageIfConfigured()

	if conn.msg != "" {
		t.Fatalf("ready message sent = %q, want none", conn.msg)
	}
}

func TestReadyMessageDefaultsToJobChannel(t *testing.T) {
	conn := &startupGateCaptureConnector{}
	oldConnector := interfaces.Connector
	interfaces.Connector = conn
	defer func() { interfaces.Connector = oldConnector }()

	currentCfg.Lock()
	oldCfg := currentCfg.configuration
	currentCfg.configuration = &configuration{
		defaultJobChannel:    "jobs",
		readyMessage:         "Robot ready",
		defaultProtocol:      "test",
		defaultMessageFormat: robot.BasicMarkdown,
	}
	currentCfg.Unlock()
	defer func() {
		currentCfg.Lock()
		currentCfg.configuration = oldCfg
		currentCfg.Unlock()
	}()

	sendReadyMessageIfConfigured()

	if conn.msg != "Robot ready" {
		t.Fatalf("ready message = %q, want %q", conn.msg, "Robot ready")
	}
	if conn.channel != "jobs" {
		t.Fatalf("ready channel = %q, want jobs", conn.channel)
	}
	if conn.protocol != "test" {
		t.Fatalf("ready protocol = %q, want test", conn.protocol)
	}
	if conn.format != robot.BasicMarkdown {
		t.Fatalf("ready format = %v, want BasicMarkdown", conn.format)
	}
}

func TestReadyMessageUsesConfiguredChannel(t *testing.T) {
	conn := &startupGateCaptureConnector{}
	oldConnector := interfaces.Connector
	interfaces.Connector = conn
	defer func() { interfaces.Connector = oldConnector }()

	currentCfg.Lock()
	oldCfg := currentCfg.configuration
	currentCfg.configuration = &configuration{
		defaultJobChannel:    "jobs",
		readyMessage:         "Robot ready",
		readyChannel:         "ops",
		defaultProtocol:      "test",
		defaultMessageFormat: robot.Raw,
	}
	currentCfg.Unlock()
	defer func() {
		currentCfg.Lock()
		currentCfg.configuration = oldCfg
		currentCfg.Unlock()
	}()

	sendReadyMessageIfConfigured()

	if conn.channel != "ops" {
		t.Fatalf("ready channel = %q, want ops", conn.channel)
	}
	if conn.format != robot.Raw {
		t.Fatalf("ready format = %v, want Raw", conn.format)
	}
}

func (c *startupGateCaptureConnector) SendProtocolUserMessage(user, msg string, format robot.MessageFormat, msgObject *robot.ConnectorMessage) robot.RetVal {
	c.user = user
	c.msg = msg
	c.format = format
	c.sends++
	if msgObject != nil {
		c.protocol = msgObject.Protocol
	}
	return robot.Ok
}
func (c *startupGateCaptureConnector) Reload() error                         { return nil }
func (c *startupGateCaptureConnector) Run(stopchannel <-chan struct{}) error { return nil }

func TestIncomingCommandDuringStartupGateGetsStartingMessage(t *testing.T) {
	conn := &startupGateCaptureConnector{}
	oldConnector := interfaces.Connector
	interfaces.Connector = conn
	defer func() { interfaces.Connector = oldConnector }()

	currentCfg.Lock()
	oldCfg := currentCfg.configuration
	oldTasks := currentCfg.taskList
	oldIgnoreUnlisted := currentCfg.ignoreUnlistedUsers
	currentCfg.configuration = &configuration{}
	currentCfg.taskList = &taskList{}
	currentCfg.ignoreUnlistedUsers = false
	currentCfg.Unlock()
	currentUCMaps.Lock()
	oldMaps := currentUCMaps.ucmap
	currentUCMaps.ucmap = &userChanMaps{
		userIDProto:   map[string]map[string]*UserInfo{},
		directoryUser: map[string]bool{"alice": true},
		user:          map[string]*DirectoryUser{},
	}
	currentUCMaps.Unlock()
	defer func() {
		currentCfg.Lock()
		currentCfg.configuration = oldCfg
		currentCfg.taskList = oldTasks
		currentCfg.ignoreUnlistedUsers = oldIgnoreUnlisted
		currentCfg.Unlock()
		currentUCMaps.Lock()
		currentUCMaps.ucmap = oldMaps
		currentUCMaps.Unlock()
	}()

	resetStartupGate()
	defer releaseStartupGate()
	handle.IncomingMessage(&robot.ConnectorMessage{
		Protocol:      "test",
		UserName:      "alice",
		UserID:        "u1",
		ValidatedUser: true,
		DirectMessage: true,
		MessageText:   "help",
	})

	deadline := time.After(100 * time.Millisecond)
	for conn.msg == "" {
		select {
		case <-deadline:
			t.Fatal("timed out waiting for startup gate response")
		default:
			time.Sleep(time.Millisecond)
		}
	}
	if conn.msg != startupGateMessage {
		t.Fatalf("startup gate message = %q, want %q", conn.msg, startupGateMessage)
	}
	if conn.user != "<u1>" {
		t.Fatalf("startup gate response user = %q, want <u1>", conn.user)
	}
}
