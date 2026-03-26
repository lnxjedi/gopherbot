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

func unsupportedHiddenCommandMessage(protocol string) string {
	protocol = strings.TrimSpace(normalizeProtocolName(protocol))
	if protocol == "" {
		return "This command isn't supported here because hidden commands are unavailable for this connector. Check with the robot administrator."
	}
	return fmt.Sprintf("This command isn't supported with %s because hidden commands are unavailable for this connector. Check with the robot administrator.", protocol)
}

func defaultHiddenCommandHint(botName string) string {
	botName = strings.TrimSpace(botName)
	if botName == "" {
		return "Hidden commands must be addressed to the robot."
	}
	return fmt.Sprintf("Hidden commands must be addressed to %s.", botName)
}

// Check whether a given command is allowed to run as a hidden command. Connectors set HiddenMessage
// to true if the command isn't visible in the team chat. For security/visibility, commands need to be
// explicitly allowed to run "hidden". This occurs, for instance, with a slack slash command.
func (r Robot) checkHiddenCommands(w *worker, t interface{}, command string) (retval robot.TaskRetVal) {
	if !w.Incoming.HiddenMessage {
		return robot.Success
	}
	protocol := protocolFromIncoming(r.Incoming, r.Protocol)
	if !hiddenCommandsSupportedForProtocol(protocol) {
		r.Reply(unsupportedHiddenCommandMessage(protocol))
		return robot.Fail
	}
	// Hidden commands from connectors should still be explicitly addressed to the
	// robot unless the connector marks them as BotMessage (e.g. slash commands
	// already routed to this robot by the platform).
	if !hiddenMessageAddressedToRobot(w.Incoming.BotMessage, w.cmdMode) {
		hint := strings.TrimSpace(r.expandHelpPlaceholders(hiddenCommandHintForProtocol(protocol)))
		if hint == "" {
			hint = defaultHiddenCommandHint(r.GetBotAttribute("name").String())
		}
		r.Reply(hint)
		return robot.Fail
	}
	_, plugin, _ := getTask(t)
	if plugin == nil {
		return robot.Success
	}
	if len(plugin.AllowedHiddenCommands) == 0 {
		return robot.Fail
	}
	for _, i := range plugin.AllowedHiddenCommands {
		if command == i {
			w.Log(robot.Audit, "Hidden command '%s' from plugin '%s' issued by user '%s' in channel '%s'", command, plugin.name, r.User, r.Channel)
			return robot.Success
		}
	}
	return robot.Fail
}
