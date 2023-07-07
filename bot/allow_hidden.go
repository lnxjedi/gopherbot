package bot

import "github.com/lnxjedi/gopherbot/robot"

// Check whether a given command is allowed to run as a hidden command. Connectors set HiddenMessage
// to true if the command isn't visible in the team chat. For security/visibility, commands need to be
// explicitly allowed to run "hidden". This occurs, for instance, with a slack slash command.
func (r Robot) checkHiddenCommands(w *worker, t interface{}, command string) (retval robot.TaskRetVal) {
	if !w.Incoming.HiddenMessage {
		return robot.Success
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
			return robot.Success
		}
	}
	return robot.Fail
}
