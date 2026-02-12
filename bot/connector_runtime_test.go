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
	mu        sync.Mutex
	runCount  int
	stopCount int
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

func (fc *fakeRuntimeConnector) SendProtocolChannelThreadMessage(string, string, string, robot.MessageFormat, *robot.ConnectorMessage) robot.RetVal {
	return robot.Ok
}

func (fc *fakeRuntimeConnector) SendProtocolUserChannelThreadMessage(string, string, string, string, string, robot.MessageFormat, *robot.ConnectorMessage) robot.RetVal {
	return robot.Ok
}

func (fc *fakeRuntimeConnector) SendProtocolUserMessage(string, string, robot.MessageFormat, *robot.ConnectorMessage) robot.RetVal {
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
	runtimeConnectors.fallbackUserMap = map[string]string{}
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
