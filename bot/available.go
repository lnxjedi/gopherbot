package bot

import (
	"path/filepath"
	"strings"
)

type commandLocationHint struct {
	PluginName        string
	Command           string
	Channels          []string
	AnyRegularChannel bool
	DirectMessageOnly bool
}

func appendUniquePreserveOrder(dst []string, values ...string) []string {
	seen := make(map[string]struct{}, len(dst)+len(values))
	for _, value := range dst {
		seen[value] = struct{}{}
	}
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		dst = append(dst, value)
		seen[value] = struct{}{}
	}
	return dst
}

func formatChannelForDisplay(channel string) string {
	channel = strings.TrimSpace(channel)
	if channel == "" {
		return "this channel"
	}
	if strings.HasPrefix(channel, "#") || strings.HasPrefix(channel, "<") {
		return channel
	}
	return "#" + channel
}

func (h commandLocationHint) format(currentChannel string, directMessage bool) string {
	location := "this location"
	if directMessage {
		location = "direct messages"
	} else if currentChannel != "" {
		location = formatChannelForDisplay(currentChannel)
	}

	command := h.PluginName + "/" + h.Command
	switch {
	case h.DirectMessageOnly:
		return command + " not available in " + location + ", try direct message"
	case h.AnyRegularChannel:
		return command + " not available in " + location + ", try it in any regular channel"
	case len(h.Channels) == 1:
		return command + " not available in " + location + ", try " + formatChannelForDisplay(h.Channels[0])
	case len(h.Channels) > 1:
		formatted := make([]string, 0, len(h.Channels))
		for _, channel := range h.Channels {
			formatted = append(formatted, formatChannelForDisplay(channel))
		}
		return command + " not available in " + location + ", try one of: " + strings.Join(formatted, ", ")
	default:
		return ""
	}
}

func (w *worker) isAdminUser() bool {
	if w.automaticTask {
		return true
	}
	for _, adminUser := range w.cfg.adminUsers {
		if w.User == adminUser {
			return true
		}
	}
	return false
}

func (w *worker) userMatchesTask(task *Task) bool {
	if task == nil {
		return false
	}
	if len(task.Users) == 0 {
		return true
	}
	for _, allowedUser := range task.Users {
		match, err := filepath.Match(allowedUser, w.User)
		if match && err == nil {
			return true
		}
	}
	return false
}

func (w *worker) userCanAccessTask(task *Task) bool {
	if task == nil || task.Disabled {
		return false
	}
	if task.RequireAdmin && !w.isAdminUser() {
		return false
	}
	return w.userMatchesTask(task)
}

func commandRequiresAdmin(plugin *Plugin, command string) bool {
	if plugin == nil {
		return false
	}
	for _, adminCommand := range plugin.AdminCommands {
		if command == adminCommand {
			return true
		}
	}
	return false
}

func (w *worker) getAuthorizerUserGroupsWithTemporaryContext(authorizer, user string) (groups map[string]struct{}, known bool) {
	if strings.TrimSpace(authorizer) == "" || strings.TrimSpace(user) == "" {
		return nil, false
	}
	if w.tasks == nil {
		return nil, false
	}
	original := w.pipeContext
	if original == nil {
		w.pipeContext = &pipeContext{
			environment: make(map[string]string),
			parameters:  make(map[string]string),
		}
		defer func() {
			w.pipeContext = original
		}()
	}
	r := w.makeRobot()
	return r.getAuthorizerUserGroups(w, authorizer, user)
}

func (w *worker) commandVisibleToUser(task *Task, plugin *Plugin, command string) bool {
	if !w.userCanAccessTask(task) {
		return false
	}
	if commandRequiresAdmin(plugin, command) && !w.isAdminUser() {
		return false
	}
	if !commandRequiresAuthorization(plugin, command) {
		return true
	}
	if strings.TrimSpace(task.AuthRequire) == "" {
		return false
	}
	authorizer := effectiveAuthorizerName(task, w.cfg.defaultAuthorizer)
	if authorizer == "" {
		return false
	}
	groups, known := w.getAuthorizerUserGroupsWithTemporaryContext(authorizer, w.User)
	if !known {
		return false
	}
	return userHasRequiredGroup(groups, task.AuthRequire)
}

func (w *worker) commandLocationHint(task *Task, plugin *Plugin, command string) (commandLocationHint, bool) {
	if task == nil || plugin == nil || !w.commandVisibleToUser(task, plugin, command) {
		return commandLocationHint{}, false
	}

	if w.Incoming.DirectMessage {
		switch {
		case task.AllowDirect || task.DirectOnly:
			return commandLocationHint{}, false
		case len(task.Channels) > 0:
			return commandLocationHint{
				PluginName: task.name,
				Command:    command,
				Channels:   append([]string(nil), task.Channels...),
			}, true
		case task.AllChannels:
			return commandLocationHint{
				PluginName:        task.name,
				Command:           command,
				AnyRegularChannel: true,
			}, true
		default:
			return commandLocationHint{}, false
		}
	}

	if task.DirectOnly {
		return commandLocationHint{
			PluginName:        task.name,
			Command:           command,
			DirectMessageOnly: true,
		}, true
	}
	if len(task.Channels) > 0 {
		for _, channel := range task.Channels {
			if channel == w.Channel {
				return commandLocationHint{}, false
			}
		}
		return commandLocationHint{
			PluginName: task.name,
			Command:    command,
			Channels:   append([]string(nil), task.Channels...),
		}, true
	}
	return commandLocationHint{}, false
}

// pluginAvailable checks the user and channel against the task's
// configuration to determine if the task should be available. Used by
// both handleMessage and the help builtin. verboseOnly is set when availability
// is being checked for ambient messages or auth/elevation plugins, to indicate
// debugging verboseness. The `specific` bool is set whenever a plugin lists the
// channel explicitly, or for direct messages when DirectOnly is true; this is
// used by the help plugin to differentiate "<robot>, help" from "<robot> help-all".
func (w *worker) pluginAvailable(task *Task, helpSystem, verboseOnly bool) (available, specific bool) {
	if task.Disabled {
		return false, false
	}
	if !w.Incoming.DirectMessage && task.DirectOnly && !helpSystem {
		return false, false
	}
	if w.Incoming.DirectMessage && !task.AllowDirect && !helpSystem {
		return false, false
	}
	if !w.userCanAccessTask(task) {
		return false, false
	}
	if w.Incoming.DirectMessage && (task.AllowDirect || task.DirectOnly) {
		if task.DirectOnly {
			return true, true
		}
		return true, helpSystem
	}
	if len(task.Channels) > 0 {
		for _, pchannel := range task.Channels {
			if pchannel == w.Channel {
				return true, true
			}
		}
	} else {
		if task.AllChannels {
			return true, helpSystem
		}
	}
	if helpSystem {
		return true, true
	}
	return false, false
}
