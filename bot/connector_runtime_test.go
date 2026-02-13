package bot

import (
	"io"
	"log"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
)

type fakeRuntimeConnector struct {
	mu                 sync.Mutex
	runCount           int
	stopCount          int
	channelCalls       int
	userChannelCalls   int
	userCalls          int
	lastChannel        string
	lastThread         string
	lastMessage        string
	lastUser           string
	lastUserID         string
	lastUserName       string
	lastProtocolOnSend string
}

func (fc *fakeRuntimeConnector) SetUserMap(map[string]string) {}

func (fc *fakeRuntimeConnector) GetProtocolUserAttribute(string, string) (string, robot.RetVal) {
	return "", robot.AttributeNotFound
}

func (fc *fakeRuntimeConnector) MessageHeard(string, string) {}

func (fc *fakeRuntimeConnector) FormatHelp(input string) string {
	return input
}

func (fc *fakeRuntimeConnector) DefaultHelp() []string {
	return nil
}

func (fc *fakeRuntimeConnector) JoinChannel(string) robot.RetVal {
	return robot.Ok
}

func (fc *fakeRuntimeConnector) SendProtocolChannelThreadMessage(ch, thr, msg string, _ robot.MessageFormat, msgObject *robot.ConnectorMessage) robot.RetVal {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	fc.channelCalls++
	fc.lastChannel = ch
	fc.lastThread = thr
	fc.lastMessage = msg
	if msgObject != nil {
		fc.lastProtocolOnSend = msgObject.Protocol
	}
	return robot.Ok
}

func (fc *fakeRuntimeConnector) SendProtocolUserChannelThreadMessage(uid, uname, ch, thr, msg string, _ robot.MessageFormat, msgObject *robot.ConnectorMessage) robot.RetVal {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	fc.userChannelCalls++
	fc.lastUserID = uid
	fc.lastUserName = uname
	fc.lastChannel = ch
	fc.lastThread = thr
	fc.lastMessage = msg
	if msgObject != nil {
		fc.lastProtocolOnSend = msgObject.Protocol
	}
	return robot.Ok
}

func (fc *fakeRuntimeConnector) SendProtocolUserMessage(u, msg string, _ robot.MessageFormat, msgObject *robot.ConnectorMessage) robot.RetVal {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	fc.userCalls++
	fc.lastUser = u
	fc.lastMessage = msg
	if msgObject != nil {
		fc.lastProtocolOnSend = msgObject.Protocol
	}
	return robot.Ok
}

func (fc *fakeRuntimeConnector) Run(stop <-chan struct{}) {
	fc.mu.Lock()
	fc.runCount++
	fc.mu.Unlock()
	<-stop
	fc.mu.Lock()
	fc.stopCount++
	fc.mu.Unlock()
}

func (fc *fakeRuntimeConnector) metrics() (runs, stops int) {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	return fc.runCount, fc.stopCount
}

func (fc *fakeRuntimeConnector) sendMetrics() (channelCalls, userChannelCalls, userCalls int, protocol, channel string) {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	return fc.channelCalls, fc.userChannelCalls, fc.userCalls, fc.lastProtocolOnSend, fc.lastChannel
}

type runtimeHarness struct {
	t                *testing.T
	originalConnects map[string]func(robot.Handler, *log.Logger) robot.Connector
	originalCfg      *configuration
	originalIface    robot.Connector
	originalLogger   *log.Logger
	instances        map[string]*fakeRuntimeConnector
}

func newRuntimeHarness(t *testing.T) *runtimeHarness {
	t.Helper()
	h := &runtimeHarness{
		t:                t,
		originalConnects: make(map[string]func(robot.Handler, *log.Logger) robot.Connector, len(connectors)),
		instances:        make(map[string]*fakeRuntimeConnector),
	}
	for name, initFn := range connectors {
		h.originalConnects[name] = initFn
	}
	currentCfg.RLock()
	h.originalCfg = currentCfg.configuration
	currentCfg.RUnlock()
	h.originalIface = interfaces.Connector
	h.originalLogger = botLogger.logger
	botLogger.logger = log.New(io.Discard, "", 0)
	h.resetRuntimeState()
	t.Cleanup(h.cleanup)
	return h
}

func (h *runtimeHarness) cleanup() {
	shutdownConnectorRuntimes()
	h.resetRuntimeState()
	connectors = h.originalConnects
	currentCfg.Lock()
	currentCfg.configuration = h.originalCfg
	currentCfg.Unlock()
	interfaces.Connector = h.originalIface
	botLogger.logger = h.originalLogger
}

func (h *runtimeHarness) resetRuntimeState() {
	runtimeConnectors.Lock()
	runtimeConnectors.primary = ""
	runtimeConnectors.runtimes = map[string]*managedConnector{}
	runtimeConnectors.desiredSecondary = map[string]bool{}
	runtimeConnectors.userMaps = map[string]map[string]string{}
	runtimeConnectors.Unlock()
}

func (h *runtimeHarness) setConfig(primary string, secondaries ...string) {
	currentCfg.Lock()
	currentCfg.configuration = &configuration{
		protocol:           primary,
		secondaryProtocols: secondaries,
	}
	currentCfg.Unlock()
}

func (h *runtimeHarness) registerFake(protocol string) {
	p := normalizeProtocolName(protocol)
	connectors[p] = func(robot.Handler, *log.Logger) robot.Connector {
		fc := &fakeRuntimeConnector{}
		h.instances[p] = fc
		return fc
	}
}

func statusMap() map[string]connectorStatus {
	statuses := listConnectorProtocolStatus()
	out := make(map[string]connectorStatus, len(statuses))
	for _, status := range statuses {
		out[status.protocol] = status
	}
	return out
}

func waitFor(t *testing.T, desc string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for condition: %s", desc)
}

func TestRuntimeLifecycleStartStopRestart(t *testing.T) {
	h := newRuntimeHarness(t)
	h.registerFake("prime")
	h.registerFake("secondary")
	h.setConfig("prime", "secondary")

	if err := initializeConnectorRuntime(log.New(io.Discard, "", 0)); err != nil {
		t.Fatalf("initializeConnectorRuntime() error = %v", err)
	}
	if err := startConnectorRuntimes(); err != nil {
		t.Fatalf("startConnectorRuntimes() error = %v", err)
	}

	waitFor(t, "both protocols running", func() bool {
		sm := statusMap()
		return sm["prime"].state == "running" && sm["secondary"].state == "running"
	})

	if err := stopSecondaryConnectorRuntime("secondary"); err != nil {
		t.Fatalf("stopSecondaryConnectorRuntime() error = %v", err)
	}
	waitFor(t, "secondary stopped", func() bool {
		return statusMap()["secondary"].state == "stopped"
	})

	if err := startSecondaryConnectorRuntime("secondary"); err != nil {
		t.Fatalf("startSecondaryConnectorRuntime() error = %v", err)
	}
	waitFor(t, "secondary restarted", func() bool {
		return statusMap()["secondary"].state == "running"
	})

	shutdownConnectorRuntimes()
	waitFor(t, "all connectors stopped", func() bool {
		sm := statusMap()
		return sm["prime"].state == "stopped" && sm["secondary"].state == "stopped"
	})
}

func TestReconcileSecondaryProtocolsStopsRemovedStartsAdded(t *testing.T) {
	h := newRuntimeHarness(t)
	h.registerFake("prime")
	h.registerFake("seca")
	h.registerFake("secb")
	h.setConfig("prime", "seca")

	if err := initializeConnectorRuntime(log.New(io.Discard, "", 0)); err != nil {
		t.Fatalf("initializeConnectorRuntime() error = %v", err)
	}
	if err := startConnectorRuntimes(); err != nil {
		t.Fatalf("startConnectorRuntimes() error = %v", err)
	}
	waitFor(t, "prime+seca running", func() bool {
		sm := statusMap()
		return sm["prime"].state == "running" && sm["seca"].state == "running"
	})

	reconcileSecondaryConnectorRuntimes([]string{"secb"})

	waitFor(t, "reconciled protocols", func() bool {
		sm := statusMap()
		_, hasOld := sm["seca"]
		return sm["prime"].state == "running" && sm["secb"].state == "running" && !hasOld
	})

	waitFor(t, "old secondary stopped at least once", func() bool {
		_, stops := h.instances["seca"].metrics()
		return stops >= 1
	})
}

func TestStartSecondaryValidation(t *testing.T) {
	h := newRuntimeHarness(t)
	h.registerFake("prime")
	h.registerFake("secondary")
	h.setConfig("prime", "secondary")

	if err := initializeConnectorRuntime(log.New(io.Discard, "", 0)); err != nil {
		t.Fatalf("initializeConnectorRuntime() error = %v", err)
	}

	if err := startSecondaryConnectorRuntime("prime"); err == nil || !strings.Contains(err.Error(), "primary protocol") {
		t.Fatalf("startSecondaryConnectorRuntime(primary) error = %v, want primary-protocol error", err)
	}
	if err := startSecondaryConnectorRuntime("unknown"); err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("startSecondaryConnectorRuntime(unknown) error = %v, want not-configured error", err)
	}
	if err := startSecondaryConnectorRuntime("secondary"); err != nil {
		t.Fatalf("startSecondaryConnectorRuntime(secondary) error = %v", err)
	}
}

func TestProtocolForMessageFallsBackToPrimary(t *testing.T) {
	newRuntimeHarness(t)
	runtimeConnectors.Lock()
	runtimeConnectors.primary = "prime"
	runtimeConnectors.Unlock()

	if got := protocolForMessage(&robot.ConnectorMessage{Protocol: "secondary"}); got != "secondary" {
		t.Fatalf("protocolForMessage(with msg protocol) = %q, want %q", got, "secondary")
	}
	if got := protocolForMessage(nil); got != "prime" {
		t.Fatalf("protocolForMessage(nil) = %q, want %q", got, "prime")
	}
}

func TestSendProtocolUserChannelMessageRouting(t *testing.T) {
	h := newRuntimeHarness(t)
	h.registerFake("prime")
	h.registerFake("secondary")
	h.setConfig("prime", "secondary")

	if err := initializeConnectorRuntime(log.New(io.Discard, "", 0)); err != nil {
		t.Fatalf("initializeConnectorRuntime() error = %v", err)
	}
	if err := startConnectorRuntimes(); err != nil {
		t.Fatalf("startConnectorRuntimes() error = %v", err)
	}
	waitFor(t, "prime+secondary running", func() bool {
		sm := statusMap()
		return sm["prime"].state == "running" && sm["secondary"].state == "running"
	})

	r := Robot{
		Message: &robot.Message{
			Incoming: &robot.ConnectorMessage{Protocol: "prime"},
			Format:   robot.Variable,
		},
		maps: &userChanMaps{
			user: map[string]*DirectoryUser{
				"alice": {UserName: "alice"},
			},
			userProto: map[string]map[string]*UserInfo{
				"secondary": {
					"alice": &UserInfo{UserName: "alice", UserID: "sec-alice"},
				},
			},
			channel: map[string]*ChannelInfo{
				"general": &ChannelInfo{ChannelName: "general", ChannelID: "prime-general"},
			},
			channelProto: map[string]map[string]*ChannelInfo{
				"secondary": {
					"general": &ChannelInfo{ChannelName: "general", ChannelID: "sec-general"},
				},
			},
		},
	}

	if ret := r.SendProtocolUserChannelMessage("secondary", "alice", "general", "hello"); ret != robot.Ok {
		t.Fatalf("SendProtocolUserChannelMessage(user+channel) ret = %v, want %v", ret, robot.Ok)
	}
	if channelCalls, userChannelCalls, userCalls, protocol, channel := h.instances["secondary"].sendMetrics(); channelCalls != 0 || userChannelCalls != 1 || userCalls != 0 || protocol != "secondary" || channel != "<sec-general>" {
		t.Fatalf("secondary send metrics = channel:%d userChannel:%d user:%d protocol:%q channelArg:%q", channelCalls, userChannelCalls, userCalls, protocol, channel)
	}

	if ret := r.SendProtocolUserChannelMessage("secondary", "alice", "", "dm"); ret != robot.Ok {
		t.Fatalf("SendProtocolUserChannelMessage(dm) ret = %v, want %v", ret, robot.Ok)
	}
	if channelCalls, userChannelCalls, userCalls, protocol, _ := h.instances["secondary"].sendMetrics(); channelCalls != 0 || userChannelCalls != 1 || userCalls != 1 || protocol != "secondary" {
		t.Fatalf("secondary send metrics after dm = channel:%d userChannel:%d user:%d protocol:%q", channelCalls, userChannelCalls, userCalls, protocol)
	}

	if ret := r.SendProtocolUserChannelMessage("secondary", "", "general", "chan"); ret != robot.Ok {
		t.Fatalf("SendProtocolUserChannelMessage(channel) ret = %v, want %v", ret, robot.Ok)
	}
	if channelCalls, userChannelCalls, userCalls, protocol, channel := h.instances["secondary"].sendMetrics(); channelCalls != 1 || userChannelCalls != 1 || userCalls != 1 || protocol != "secondary" || channel != "<sec-general>" {
		t.Fatalf("secondary send metrics after channel send = channel:%d userChannel:%d user:%d protocol:%q channelArg:%q", channelCalls, userChannelCalls, userCalls, protocol, channel)
	}

	if ret := r.SendProtocolUserChannelMessage("secondary", "", "", "bad"); ret != robot.MissingArguments {
		t.Fatalf("SendProtocolUserChannelMessage(empty targets) ret = %v, want %v", ret, robot.MissingArguments)
	}
	if ret := r.SendProtocolUserChannelMessage("unknown", "alice", "general", "bad"); ret != robot.Failed {
		t.Fatalf("SendProtocolUserChannelMessage(unknown protocol) ret = %v, want %v", ret, robot.Failed)
	}
}

func TestProtocolChannelLookupPrefersProtocolScopedMaps(t *testing.T) {
	maps := &userChanMaps{
		channel: map[string]*ChannelInfo{
			"general": {ChannelName: "general", ChannelID: "prime-general"},
		},
		channelID: map[string]*ChannelInfo{
			"C-shared": {ChannelName: "prime-general", ChannelID: "C-shared"},
		},
		channelProto: map[string]map[string]*ChannelInfo{
			"secondary": {
				"general": {ChannelName: "general", ChannelID: "sec-general"},
			},
		},
		channelIDProto: map[string]map[string]*ChannelInfo{
			"secondary": {
				"C-shared": {ChannelName: "sec-general", ChannelID: "C-shared"},
			},
		},
	}

	byName, ok := getProtocolChannelByName(maps, "secondary", "general")
	if !ok || byName.ChannelID != "sec-general" {
		t.Fatalf("getProtocolChannelByName() = (%v, %t), want sec-general,true", byName, ok)
	}
	byID, ok := getProtocolChannelByID(maps, "secondary", "C-shared")
	if !ok || byID.ChannelName != "sec-general" {
		t.Fatalf("getProtocolChannelByID() = (%v, %t), want sec-general,true", byID, ok)
	}
}
