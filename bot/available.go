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
	PrivateOnly       bool
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
	case h.PrivateOnly:
		return command + " not available in " + location + ", try a private context"
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
		if commandAllowsPrivate(plugin, command) {
			return commandLocationHint{}, false
		}
		if len(task.Channels) > 0 {
			return commandLocationHint{
				PluginName: task.name,
				Command:    command,
				Channels:   append([]string(nil), task.Channels...),
			}, true
		}
		if task.AllChannels {
			return commandLocationHint{
				PluginName:        task.name,
				Command:           command,
				AnyRegularChannel: true,
			}, true
		}
		return commandLocationHint{}, false
	}

	if commandRequiresPrivate(plugin, command) {
		return commandLocationHint{
			PluginName:  task.name,
			Command:     command,
			PrivateOnly: true,
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

func channelInList(channel string, channels []string) bool {
	for _, allowed := range channels {
		if allowed == channel {
			return true
		}
	}
	return false
}

func (w *worker) privateChannelRestricted(task *Task, plugin *Plugin) bool {
	return task != nil && plugin != nil && plugin.RestrictPrivateChannels && len(task.Channels) > 0
}

func (w *worker) privateContextSatisfiesChannels(task *Task, plugin *Plugin) bool {
	if !w.privateChannelRestricted(task, plugin) {
		return true
	}
	if w.Incoming == nil || w.Incoming.DirectMessage {
		return false
	}
	return channelInList(w.Channel, task.Channels)
}

func (w *worker) pluginAvailableForPrivateCommandMatch(task *Task, plugin *Plugin, verboseOnly bool) bool {
	if plugin == nil || task == nil || !privateCommandContext(w.Incoming) {
		return false
	}
	if verboseOnly {
		return false
	}
	if !pluginHasPrivatePolicy(plugin) {
		return false
	}
	if !w.userCanAccessTask(task) {
		return false
	}
	return true
}

// pluginAvailable checks the user and channel against the task's
// configuration to determine if the task should be available. Used by
// both handleMessage and the help builtin. verboseOnly is set when availability
// is being checked for ambient messages or auth/elevation plugins, to indicate
// debugging verboseness. The `specific` bool is set whenever a plugin lists the
// channel explicitly; this is used by the help plugin to differentiate
// "<robot>, help" from "<robot> help-all".
func (w *worker) pluginAvailable(task *Task, helpSystem, verboseOnly bool) (available, specific bool) {
	if task.Disabled {
		return false, false
	}
	if !w.userCanAccessTask(task) {
		return false, false
	}
	if w.Incoming.DirectMessage {
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
