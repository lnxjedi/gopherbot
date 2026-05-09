package bot

import (
	"fmt"
	"strings"

	"github.com/lnxjedi/gopherbot/robot"
)

func hiddenMessageAddressedToRobot(botMessage bool, cmdMode string) bool {
	if botMessage {
		return true
	}
	return cmdMode == "name"
}

func privateCommandContext(incoming *robot.ConnectorMessage) bool {
	return incoming != nil && (incoming.DirectMessage || incoming.HiddenMessage)
}

func pluginHasPrivatePolicy(plugin *Plugin) bool {
	return plugin != nil && (plugin.RequireAllCommandsPrivate || len(plugin.AllowedPrivateCommands) > 0 || len(plugin.RequiredPrivateCommands) > 0)
}

func unsupportedPrivateCommandMessage(protocol string) string {
	protocol = strings.TrimSpace(normalizeProtocolName(protocol))
	if protocol == "" {
		return "This command isn't supported here because private command transport is unavailable for this connector. Check with the robot administrator."
	}
	return fmt.Sprintf("This command isn't supported with %s because private command transport is unavailable for this connector. Check with the robot administrator.", protocol)
}

func defaultPrivateCommandHint(botName string) string {
	botName = strings.TrimSpace(botName)
	if botName == "" {
		return "Private commands must be addressed to the robot."
	}
	return fmt.Sprintf("Private commands must be addressed to %s.", botName)
}

func requiredPrivateCommandMessage() string {
	return "This command is only available in a private context."
}

func (r Robot) checkRequiredPrivateCommand(w *worker, t interface{}, command string) robot.TaskRetVal {
	if privateCommandContext(w.Incoming) {
		return robot.Success
	}
	_, plugin, _ := getTask(t)
	if !commandRequiresPrivate(plugin, command) {
		return robot.Success
	}
	r.Say(requiredPrivateCommandMessage())
	return robot.Fail
}

// Check whether a given command is allowed to run in a private context. Direct
// messages and transport-private hidden messages both count as private, but
// hidden messages still require connector support and explicit robot addressing.
func (r Robot) checkPrivateCommands(w *worker, t interface{}, command string) (retval robot.TaskRetVal) {
	if !privateCommandContext(w.Incoming) {
		return robot.Success
	}
	if w.Incoming.HiddenMessage {
		protocol := protocolFromIncoming(r.Incoming, r.Protocol)
		if !hiddenCommandsSupportedForProtocol(protocol) {
			r.Reply(unsupportedPrivateCommandMessage(protocol))
			return robot.Fail
		}
		// Hidden/private commands from connectors should still be explicitly addressed to
		// the robot unless the connector marks them as BotMessage (e.g. slash
		// commands already routed to this robot by the platform).
		if !hiddenMessageAddressedToRobot(w.Incoming.BotMessage, w.cmdMode) {
			hint := strings.TrimSpace(r.expandHelpPlaceholders(hiddenCommandHintForProtocol(protocol)))
			if hint == "" {
				hint = defaultPrivateCommandHint(r.GetBotAttribute("name").String())
			}
			r.Reply(hint)
			return robot.Fail
		}
	}
	_, plugin, _ := getTask(t)
	if plugin == nil {
		return robot.Success
	}
	if commandAllowsPrivate(plugin, command) {
		w.Log(robot.Audit, "Private command '%s' from plugin '%s' issued by user '%s' in channel '%s'", command, plugin.name, r.User, r.Channel)
		return robot.Success
	}
	return robot.Fail
}
