package bot

import (
	"strings"

	"github.com/lnxjedi/gopherbot/robot"
)

func hiddenMessageAddressedToRobot(botMessage bool, cmdMode string) bool {
	if botMessage {
		return true
	}
	return cmdMode == "name"
}

// Check whether a given command is allowed to run as a hidden command. Connectors set HiddenMessage
// to true if the command isn't visible in the team chat. For security/visibility, commands need to be
// explicitly allowed to run "hidden". This occurs, for instance, with a slack slash command.
func (r Robot) checkHiddenCommands(w *worker, t interface{}, command string) (retval robot.TaskRetVal) {
	if !w.Incoming.HiddenMessage {
		return robot.Success
	}
	// Hidden commands from connectors should still be explicitly addressed to the
	// robot unless the connector marks them as BotMessage (e.g. slash commands
	// already routed to this robot by the platform).
	if !hiddenMessageAddressedToRobot(w.Incoming.BotMessage, w.cmdMode) {
		hint := strings.TrimSpace(r.expandHelpPlaceholders(hiddenCommandHintForProtocol(protocolFromIncoming(r.Incoming, r.Protocol))))
		if hint == "" {
			botName := r.GetBotAttribute("name").String()
			if botName == "" {
				r.Reply("Sorry, hidden commands must be addressed to the robot")
			} else {
				r.Reply("Sorry, hidden commands must be addressed to %s", botName)
			}
		} else {
			r.Reply("Sorry, hidden commands must be addressed to the robot")
			r.Reply(hint)
		}
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
