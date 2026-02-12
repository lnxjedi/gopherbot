package bot

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"

	"github.com/lnxjedi/gopherbot/robot"
)

type managedConnector struct {
	protocol  string
	connector robot.Connector
	stop      chan struct{}
	done      chan struct{}
	running   bool
	stopping  bool
	lastError string
}

type connectorStatus struct {
	protocol string
	role     string
	state    string
	err      string
}

var runtimeConnectors = struct {
	sync.RWMutex
	primary          string
	runtimes         map[string]*managedConnector
	desiredSecondary map[string]bool
	userMaps         map[string]map[string]string
	fallbackUserMap  map[string]string
}{
	runtimes:         map[string]*managedConnector{},
	desiredSecondary: map[string]bool{},
	userMaps:         map[string]map[string]string{},
	fallbackUserMap:  map[string]string{},
}

type runtimeConnectorRouter struct{}

func normalizeProtocolName(protocol string) string {
	return strings.ToLower(strings.TrimSpace(protocol))
}

func protocolNameFromEnum(protocol robot.Protocol) string {
	switch protocol {
	case robot.Slack:
		return "slack"
	case robot.Rocket:
		return "rocket"
	case robot.Terminal:
		return "terminal"
	case robot.Test:
		return "test"
	case robot.Null:
		return "nullconn"
	case robot.SSH:
		return "ssh"
	default:
		return "test"
	}
}

func protocolFromIncoming(in *robot.ConnectorMessage, fallback robot.Protocol) string {
	if in != nil {
		if p := normalizeProtocolName(in.Protocol); p != "" {
			return p
		}
	}
	return protocolNameFromEnum(fallback)
}

func getRuntimePrimaryProtocol() (string, bool) {
	runtimeConnectors.RLock()
	defer runtimeConnectors.RUnlock()
	if runtimeConnectors.primary == "" {
		return "", false
	}
	return runtimeConnectors.primary, true
}

func getRuntimeConnector(protocol string) (robot.Connector, bool) {
	p := normalizeProtocolName(protocol)
	runtimeConnectors.RLock()
	defer runtimeConnectors.RUnlock()
	mc, ok := runtimeConnectors.runtimes[p]
	if !ok || mc == nil || mc.connector == nil {
		return nil, false
	}
	return mc.connector, true
}

func getPrimaryConnector() robot.Connector {
	runtimeConnectors.RLock()
	defer runtimeConnectors.RUnlock()
	mc, ok := runtimeConnectors.runtimes[runtimeConnectors.primary]
	if !ok || mc == nil {
		return nil
	}
	return mc.connector
}

func getConnectorForProtocol(protocol string) robot.Connector {
	p := normalizeProtocolName(protocol)
	if c, ok := getRuntimeConnector(p); ok {
		return c
	}
	return getPrimaryConnector()
}

func protocolForMessage(msgObject *robot.ConnectorMessage) string {
	if msgObject != nil {
		if p := normalizeProtocolName(msgObject.Protocol); p != "" {
			return p
		}
	}
	if primary, ok := getRuntimePrimaryProtocol(); ok {
		return primary
	}
	return ""
}

func configuredProtocols() (string, []string) {
	currentCfg.RLock()
	defer currentCfg.RUnlock()
	primary := normalizeProtocolName(currentCfg.protocol)
	secondary := normalizeSecondaryProtocols(primary, currentCfg.secondaryProtocols)
	for i := range secondary {
		secondary[i] = normalizeProtocolName(secondary[i])
	}
	return primary, secondary
}

func userMapForProtocolLocked(protocol string) map[string]string {
	if m, ok := runtimeConnectors.userMaps[protocol]; ok && len(m) > 0 {
		c := make(map[string]string, len(m))
		for k, v := range m {
			c[k] = v
		}
		return c
	}
	if len(runtimeConnectors.fallbackUserMap) == 0 {
		return nil
	}
	c := make(map[string]string, len(runtimeConnectors.fallbackUserMap))
	for k, v := range runtimeConnectors.fallbackUserMap {
		c[k] = v
	}
	return c
}

func setConnectorUserMaps(perProtocol map[string]map[string]string, fallback map[string]string) {
	runtimeConnectors.Lock()
	runtimeConnectors.userMaps = make(map[string]map[string]string, len(perProtocol))
	for protocol, users := range perProtocol {
		p := normalizeProtocolName(protocol)
		if p == "" {
			continue
		}
		um := make(map[string]string, len(users))
		for u, id := range users {
			um[u] = id
		}
		runtimeConnectors.userMaps[p] = um
	}
	runtimeConnectors.fallbackUserMap = make(map[string]string, len(fallback))
	for u, id := range fallback {
		runtimeConnectors.fallbackUserMap[u] = id
	}
	runtimes := make([]*managedConnector, 0, len(runtimeConnectors.runtimes))
	for _, mc := range runtimeConnectors.runtimes {
		if mc != nil && mc.connector != nil {
			runtimes = append(runtimes, mc)
		}
	}
	runtimeConnectors.Unlock()

	for _, mc := range runtimes {
		if um := userMapForProtocol(mc.protocol); len(um) > 0 {
			mc.connector.SetUserMap(um)
		}
	}
}

func userMapForProtocol(protocol string) map[string]string {
	p := normalizeProtocolName(protocol)
	runtimeConnectors.RLock()
	defer runtimeConnectors.RUnlock()
	return userMapForProtocolLocked(p)
}

func initializeConnectorRuntime(logger *log.Logger) error {
	primary, secondaries := configuredProtocols()
	if primary == "" {
		return fmt.Errorf("primary protocol not configured")
	}

	runtimeConnectors.Lock()
	runtimeConnectors.primary = primary
	runtimeConnectors.runtimes = map[string]*managedConnector{}
	runtimeConnectors.desiredSecondary = map[string]bool{}
	for _, protocol := range secondaries {
		if protocol == "" || protocol == primary {
			continue
		}
		runtimeConnectors.desiredSecondary[protocol] = true
	}
	runtimeConnectors.Unlock()

	if err := ensureConnectorInitialized(primary, true, logger); err != nil {
		return err
	}
	for _, protocol := range secondaries {
		if protocol == "" || protocol == primary {
			continue
		}
		if err := ensureConnectorInitialized(protocol, false, logger); err != nil {
			Log(robot.Error, "Secondary protocol '%s' initialization failed: %v", protocol, err)
		}
	}
	setConnector(&runtimeConnectorRouter{})
	return nil
}

func ensureConnectorInitialized(protocol string, allowBotIdentity bool, logger *log.Logger) error {
	p := normalizeProtocolName(protocol)
	if p == "" {
		return fmt.Errorf("invalid empty protocol name")
	}

	runtimeConnectors.Lock()
	mc, ok := runtimeConnectors.runtimes[p]
	if !ok || mc == nil {
		mc = &managedConnector{protocol: p}
		runtimeConnectors.runtimes[p] = mc
	}
	if mc.connector != nil {
		runtimeConnectors.Unlock()
		return nil
	}
	runtimeConnectors.Unlock()

	initializeConnector, ok := connectors[p]
	if !ok {
		return fmt.Errorf("no connector registered with name '%s'", p)
	}
	conn := initializeConnector(connectorHandler{
		handler:          handle,
		protocol:         p,
		allowBotIdentity: allowBotIdentity,
	}, logger)
	if conn == nil {
		return fmt.Errorf("connector '%s' returned nil from initializer", p)
	}

	if um := userMapForProtocol(p); len(um) > 0 {
		conn.SetUserMap(um)
	}

	runtimeConnectors.Lock()
	mc = runtimeConnectors.runtimes[p]
	if mc == nil {
		mc = &managedConnector{protocol: p}
		runtimeConnectors.runtimes[p] = mc
	}
	mc.connector = conn
	mc.lastError = ""
	runtimeConnectors.Unlock()
	return nil
}

func startConnectorRuntime(protocol string, required bool) error {
	p := normalizeProtocolName(protocol)
	if p == "" {
		return fmt.Errorf("invalid empty protocol name")
	}
	if err := ensureConnectorInitialized(p, p == runtimeConnectorsPrimary(), botLogger.logger); err != nil {
		if required {
			return err
		}
		Log(robot.Error, "Connector '%s' failed to initialize: %v", p, err)
		runtimeConnectors.Lock()
		if mc, ok := runtimeConnectors.runtimes[p]; ok && mc != nil {
			mc.lastError = err.Error()
		}
		runtimeConnectors.Unlock()
		return err
	}

	runtimeConnectors.Lock()
	mc := runtimeConnectors.runtimes[p]
	if mc == nil || mc.connector == nil {
		runtimeConnectors.Unlock()
		err := fmt.Errorf("connector '%s' is unavailable", p)
		if !required {
			Log(robot.Error, err.Error())
		}
		return err
	}
	if mc.running {
		runtimeConnectors.Unlock()
		return nil
	}
	stop := make(chan struct{})
	done := make(chan struct{})
	mc.stop = stop
	mc.done = done
	mc.running = true
	mc.stopping = false
	mc.lastError = ""
	conn := mc.connector
	runtimeConnectors.Unlock()

	go func(protocol string, connector robot.Connector, stop <-chan struct{}, done chan struct{}) {
		raiseThreadPriv("connector loop (" + protocol + ")")
		connector.Run(stop)

		var shouldLogError bool
		runtimeConnectors.Lock()
		if mc, ok := runtimeConnectors.runtimes[protocol]; ok && mc != nil {
			shouldLogError = !mc.stopping
			mc.running = false
			mc.stopping = false
			if shouldLogError && !state.shuttingDown {
				mc.lastError = "connector exited"
			}
		}
		runtimeConnectors.Unlock()
		close(done)
		if shouldLogError && !state.shuttingDown {
			Log(robot.Error, "Connector '%s' exited unexpectedly", protocol)
		} else {
			Log(robot.Info, "Connector '%s' stopped", protocol)
		}
	}(p, conn, stop, done)
	return nil
}

func stopConnectorRuntime(protocol string) error {
	p := normalizeProtocolName(protocol)
	if p == "" {
		return fmt.Errorf("invalid empty protocol name")
	}
	runtimeConnectors.Lock()
	mc, ok := runtimeConnectors.runtimes[p]
	if !ok || mc == nil || !mc.running {
		runtimeConnectors.Unlock()
		return nil
	}
	mc.stopping = true
	stop := mc.stop
	done := mc.done
	runtimeConnectors.Unlock()

	close(stop)
	<-done
	return nil
}

func startConnectorRuntimes() error {
	primary, secondaries := configuredProtocols()
	if primary == "" {
		return fmt.Errorf("primary protocol not configured")
	}
	if err := startConnectorRuntime(primary, true); err != nil {
		return err
	}
	for _, protocol := range secondaries {
		if protocol == "" || protocol == primary {
			continue
		}
		_ = startConnectorRuntime(protocol, false)
	}
	return nil
}

func shutdownConnectorRuntimes() {
	runtimeConnectors.RLock()
	protocols := make([]string, 0, len(runtimeConnectors.runtimes))
	for protocol, mc := range runtimeConnectors.runtimes {
		if mc != nil && mc.running {
			protocols = append(protocols, protocol)
		}
	}
	runtimeConnectors.RUnlock()
	sort.Strings(protocols)
	for _, protocol := range protocols {
		if err := stopConnectorRuntime(protocol); err != nil {
			Log(robot.Error, "Stopping connector '%s': %v", protocol, err)
		}
	}
}

func reconcileSecondaryConnectorRuntimes(secondaries []string) {
	primary, ok := getRuntimePrimaryProtocol()
	if !ok || primary == "" {
		return
	}
	desired := make(map[string]bool)
	for _, protocol := range secondaries {
		p := normalizeProtocolName(protocol)
		if p == "" || p == primary {
			continue
		}
		desired[p] = true
	}

	runtimeConnectors.RLock()
	current := make([]string, 0, len(runtimeConnectors.desiredSecondary))
	for protocol := range runtimeConnectors.desiredSecondary {
		current = append(current, protocol)
	}
	runtimeConnectors.RUnlock()

	for _, protocol := range current {
		if !desired[protocol] {
			_ = stopConnectorRuntime(protocol)
			runtimeConnectors.Lock()
			delete(runtimeConnectors.runtimes, protocol)
			runtimeConnectors.Unlock()
		}
	}

	runtimeConnectors.Lock()
	runtimeConnectors.desiredSecondary = desired
	runtimeConnectors.Unlock()

	for protocol := range desired {
		_ = startConnectorRuntime(protocol, false)
	}
}

func runtimeConnectorsPrimary() string {
	runtimeConnectors.RLock()
	defer runtimeConnectors.RUnlock()
	return runtimeConnectors.primary
}

func startSecondaryConnectorRuntime(protocol string) error {
	p := normalizeProtocolName(protocol)
	if p == "" {
		return fmt.Errorf("protocol name is required")
	}
	primary := runtimeConnectorsPrimary()
	if p == primary {
		return fmt.Errorf("protocol '%s' is the primary protocol and is always managed by startup/shutdown", p)
	}
	runtimeConnectors.RLock()
	_, desired := runtimeConnectors.desiredSecondary[p]
	runtimeConnectors.RUnlock()
	if !desired {
		return fmt.Errorf("protocol '%s' is not configured in SecondaryProtocols", p)
	}
	return startConnectorRuntime(p, false)
}

func stopSecondaryConnectorRuntime(protocol string) error {
	p := normalizeProtocolName(protocol)
	if p == "" {
		return fmt.Errorf("protocol name is required")
	}
	primary := runtimeConnectorsPrimary()
	if p == primary {
		return fmt.Errorf("cannot stop the primary protocol '%s' while running", p)
	}
	runtimeConnectors.RLock()
	_, desired := runtimeConnectors.desiredSecondary[p]
	runtimeConnectors.RUnlock()
	if !desired {
		return fmt.Errorf("protocol '%s' is not configured in SecondaryProtocols", p)
	}
	return stopConnectorRuntime(p)
}

func restartSecondaryConnectorRuntime(protocol string) error {
	p := normalizeProtocolName(protocol)
	if p == "" {
		return fmt.Errorf("protocol name is required")
	}
	if err := stopSecondaryConnectorRuntime(p); err != nil {
		return err
	}
	return startSecondaryConnectorRuntime(p)
}

func listConnectorProtocolStatus() []connectorStatus {
	runtimeConnectors.RLock()
	defer runtimeConnectors.RUnlock()

	protocols := make(map[string]bool)
	if runtimeConnectors.primary != "" {
		protocols[runtimeConnectors.primary] = true
	}
	for protocol := range runtimeConnectors.desiredSecondary {
		protocols[protocol] = true
	}
	keys := make([]string, 0, len(protocols))
	for protocol := range protocols {
		keys = append(keys, protocol)
	}
	sort.Strings(keys)

	out := make([]connectorStatus, 0, len(keys))
	for _, protocol := range keys {
		role := "secondary"
		if protocol == runtimeConnectors.primary {
			role = "primary"
		}
		status := connectorStatus{
			protocol: protocol,
			role:     role,
			state:    "stopped",
		}
		if mc, ok := runtimeConnectors.runtimes[protocol]; ok && mc != nil {
			if mc.running {
				status.state = "running"
			} else if mc.lastError != "" {
				status.state = "failed"
				status.err = mc.lastError
			}
		} else {
			status.state = "failed"
			status.err = "not initialized"
		}
		out = append(out, status)
	}
	return out
}

func isPrimaryProtocolSource(protocol string) bool {
	p := normalizeProtocolName(protocol)
	primary, ok := getRuntimePrimaryProtocol()
	return ok && p == primary
}

func (rc *runtimeConnectorRouter) SetUserMap(m map[string]string) {
	setConnectorUserMaps(nil, m)
}

func (rc *runtimeConnectorRouter) GetProtocolUserAttribute(user, attr string) (string, robot.RetVal) {
	conn := getPrimaryConnector()
	if conn == nil {
		return "", robot.Failed
	}
	return conn.GetProtocolUserAttribute(user, attr)
}

func (rc *runtimeConnectorRouter) MessageHeard(user, channel string) {
	conn := getPrimaryConnector()
	if conn != nil {
		conn.MessageHeard(user, channel)
	}
}

func (rc *runtimeConnectorRouter) FormatHelp(line string) string {
	conn := getPrimaryConnector()
	if conn == nil {
		return line
	}
	return conn.FormatHelp(line)
}

func (rc *runtimeConnectorRouter) DefaultHelp() []string {
	conn := getPrimaryConnector()
	if conn == nil {
		return nil
	}
	return conn.DefaultHelp()
}

func (rc *runtimeConnectorRouter) JoinChannel(channel string) robot.RetVal {
	runtimeConnectors.RLock()
	runtimes := make([]*managedConnector, 0, len(runtimeConnectors.runtimes))
	for _, mc := range runtimeConnectors.runtimes {
		if mc != nil && mc.connector != nil {
			runtimes = append(runtimes, mc)
		}
	}
	runtimeConnectors.RUnlock()

	ret := robot.Ok
	for _, mc := range runtimes {
		if r := mc.connector.JoinChannel(channel); r != robot.Ok {
			ret = r
		}
	}
	return ret
}

func (rc *runtimeConnectorRouter) SendProtocolChannelThreadMessage(channelname, threadid, msg string, format robot.MessageFormat, msgObject *robot.ConnectorMessage) robot.RetVal {
	conn := getConnectorForProtocol(protocolForMessage(msgObject))
	if conn == nil {
		return robot.Failed
	}
	return conn.SendProtocolChannelThreadMessage(channelname, threadid, msg, format, msgObject)
}

func (rc *runtimeConnectorRouter) SendProtocolUserChannelThreadMessage(userid, username, channelname, threadid, msg string, format robot.MessageFormat, msgObject *robot.ConnectorMessage) robot.RetVal {
	conn := getConnectorForProtocol(protocolForMessage(msgObject))
	if conn == nil {
		return robot.Failed
	}
	return conn.SendProtocolUserChannelThreadMessage(userid, username, channelname, threadid, msg, format, msgObject)
}

func (rc *runtimeConnectorRouter) SendProtocolUserMessage(user, msg string, format robot.MessageFormat, msgObject *robot.ConnectorMessage) robot.RetVal {
	conn := getConnectorForProtocol(protocolForMessage(msgObject))
	if conn == nil {
		return robot.Failed
	}
	return conn.SendProtocolUserMessage(user, msg, format, msgObject)
}

func (rc *runtimeConnectorRouter) Run(stopchannel <-chan struct{}) {
	<-stopchannel
}
