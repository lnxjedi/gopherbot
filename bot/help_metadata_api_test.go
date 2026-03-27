package bot

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/lnxjedi/gopherbot/robot"
)

type hiddenHelpTestConnector struct{}

func (c *hiddenHelpTestConnector) GetProtocolUserAttribute(string, string) (string, robot.RetVal) {
	return "", robot.AttributeNotFound
}

func (c *hiddenHelpTestConnector) MessageHeard(string, string) {}

func (c *hiddenHelpTestConnector) FormatHelp(input string) string { return input }

func (c *hiddenHelpTestConnector) DefaultHelp() []string { return nil }

func (c *hiddenHelpTestConnector) JoinChannel(string) robot.RetVal { return robot.Ok }

func (c *hiddenHelpTestConnector) SendProtocolChannelThreadMessage(string, string, string, robot.MessageFormat, *robot.ConnectorMessage) robot.RetVal {
	return robot.Ok
}

func (c *hiddenHelpTestConnector) SendProtocolUserChannelThreadMessage(string, string, string, string, robot.MessageFormat, *robot.ConnectorMessage) robot.RetVal {
	return robot.Ok
}

func (c *hiddenHelpTestConnector) SendProtocolUserMessage(string, string, robot.MessageFormat, *robot.ConnectorMessage) robot.RetVal {
	return robot.Ok
}

func (c *hiddenHelpTestConnector) Run(<-chan struct{}) {}

func (c *hiddenHelpTestConnector) FormatHiddenCommand(input string) string {
	return hiddenSlashBotCommand("Clu", input)
}

func TestGetHelpMetadataFiltersAndMarksVisibility(t *testing.T) {
	tasks := &taskList{
		t: []interface{}{
			&Task{name: "namespace"},
			&Plugin{
				Task: &Task{
					name:        "lists",
					Channels:    []string{"general"},
					AllowDirect: true,
				},
				Commands: []InputMatcher{{
					Command:  "add",
					Usage:    "(alias) add <item> to list",
					Summary:  "Adds an item to a list.",
					Keywords: []string{"list", "add"},
				}},
			},
			&Plugin{
				Task: &Task{
					name:     "theia-plugin",
					Channels: []string{"devops"},
				},
				Commands: []InputMatcher{{
					Command:  "start-ide",
					Usage:    "(alias) start-ide",
					Summary:  "Starts a remote IDE session.",
					Keywords: []string{"launch", "server", "ide"},
				}},
			},
			&Plugin{
				Task: &Task{
					name:         "admin-tool",
					Channels:     []string{"general"},
					RequireAdmin: true,
				},
				Commands: []InputMatcher{{
					Command:  "reboot-server",
					Usage:    "(alias) reboot-server",
					Summary:  "Reboots the server.",
					Keywords: []string{"reboot", "server"},
				}},
			},
		},
		nameMap: map[string]int{
			"lists":        1,
			"theia-plugin": 2,
			"admin-tool":   3,
		},
		nameSpaces:    map[string]ParameterSet{},
		parameterSets: map[string]ParameterSet{},
	}

	w := &worker{
		User:       "alice",
		Channel:    "general",
		Protocol:   robot.Test,
		Incoming:   &robot.ConnectorMessage{},
		cfg:        &configuration{alias: '!', botinfo: UserInfo{UserName: "Clu"}, adminUsers: []string{"parsley"}},
		tasks:      tasks,
		listedUser: true,
		pipeContext: &pipeContext{
			parameters:  map[string]string{"GOPHER_CMDMODE": "alias"},
			environment: map[string]string{},
		},
	}
	r := w.makeRobot()
	w.registerWorker(r.tid)
	defer deregisterWorker(r.tid)

	var payload helpMetadataResponse
	if err := json.Unmarshal([]byte(r.GetHelpMetadata("!launch-server")), &payload); err != nil {
		t.Fatalf("GetHelpMetadata unmarshal: %v", err)
	}

	if payload.Context.NormalizedQuery != "launch-server" {
		t.Fatalf("normalized query = %q, want %q", payload.Context.NormalizedQuery, "launch-server")
	}
	if len(payload.VisibleHere) != 1 || payload.VisibleHere[0].PluginName != "lists" {
		t.Fatalf("visible_here = %+v, want only lists/add", payload.VisibleHere)
	}

	foundBrowseableElsewhere := false
	for _, entry := range payload.Browseable {
		if entry.PluginName == "admin-tool" {
			t.Fatalf("admin-only command should not be browseable for non-admin user: %+v", entry)
		}
		if entry.PluginName == "theia-plugin" {
			foundBrowseableElsewhere = true
			if entry.VisibleHere {
				t.Fatalf("theia-plugin should not be marked visible here: %+v", entry)
			}
			if len(entry.Channels) != 1 || entry.Channels[0] != "devops" {
				t.Fatalf("theia-plugin channels = %+v, want [devops]", entry.Channels)
			}
		}
	}
	if !foundBrowseableElsewhere {
		t.Fatal("expected theia-plugin to be browseable outside the current channel")
	}
}

func TestGetHelpMetadataIncludesHiddenExamplesForSupportedConnector(t *testing.T) {
	runtimeConnectors.Lock()
	originalPrimary := runtimeConnectors.primary
	originalDefault := runtimeConnectors.defaultProtocol
	originalRuntimes := runtimeConnectors.runtimes
	runtimeConnectors.primary = "testhidden"
	runtimeConnectors.defaultProtocol = "testhidden"
	runtimeConnectors.runtimes = map[string]*managedConnector{
		"testhidden": {protocol: "testhidden", connector: &hiddenHelpTestConnector{}, caps: robot.ConnectorCapabilities{HiddenCommands: true}},
	}
	runtimeConnectors.Unlock()
	defer func() {
		runtimeConnectors.Lock()
		runtimeConnectors.primary = originalPrimary
		runtimeConnectors.defaultProtocol = originalDefault
		runtimeConnectors.runtimes = originalRuntimes
		runtimeConnectors.Unlock()
	}()

	tasks := &taskList{
		t: []interface{}{
			&Task{name: "namespace"},
			&Plugin{
				Task:                  &Task{name: "builtin-help", Channels: []string{"general"}},
				AllowedHiddenCommands: []string{"help"},
				Commands: []InputMatcher{{
					Command:  "help",
					Usage:    "(alias) help <keyword>",
					Summary:  "find help for commands matching <keyword>",
					Examples: []string{"(alias) help ping"},
				}},
			},
		},
		nameMap:       map[string]int{"builtin-help": 1},
		nameSpaces:    map[string]ParameterSet{},
		parameterSets: map[string]ParameterSet{},
	}
	w := &worker{
		User:       "alice",
		Channel:    "general",
		Incoming:   &robot.ConnectorMessage{Protocol: "testhidden"},
		cfg:        &configuration{alias: '!', botinfo: UserInfo{UserName: "Clu"}},
		tasks:      tasks,
		listedUser: true,
		pipeContext: &pipeContext{
			parameters:  map[string]string{"GOPHER_CMDMODE": "alias"},
			environment: map[string]string{},
		},
	}
	r := w.makeRobot()
	w.registerWorker(r.tid)
	defer deregisterWorker(r.tid)

	payload := r.collectHelpMetadata("help")
	if !payload.Context.HiddenCommandsSupported {
		t.Fatalf("expected hidden command support in metadata context")
	}
	if got := payload.Context.HiddenCommandHint; got != "Use `/clu <command>` to address a hidden command." {
		t.Fatalf("unexpected hidden command hint in metadata context: %q", got)
	}
	if len(payload.VisibleHere) != 1 {
		t.Fatalf("visible_here = %+v, want one help entry", payload.VisibleHere)
	}
	entry := payload.VisibleHere[0]
	if !entry.HiddenSupported {
		t.Fatalf("expected hidden support on help entry: %+v", entry)
	}
	if len(entry.HiddenExamples) != 1 || entry.HiddenExamples[0] != "/clu help ping" {
		t.Fatalf("hidden examples = %+v, want [/clu help ping]", entry.HiddenExamples)
	}
}

func TestCollectFallbackAdviceWrongChannel(t *testing.T) {
	tasks := &taskList{
		t: []interface{}{
			&Task{name: "namespace"},
			&Plugin{
				Task: &Task{
					name:     "theia-plugin",
					Channels: []string{"devops"},
				},
				Commands: []InputMatcher{{
					Command:  "start-ide",
					Usage:    "(alias) start-ide",
					Summary:  "Starts a remote IDE session.",
					Keywords: []string{"launch", "server", "ide"},
				}},
			},
		},
		nameMap: map[string]int{
			"theia-plugin": 1,
		},
		nameSpaces:    map[string]ParameterSet{},
		parameterSets: map[string]ParameterSet{},
	}

	w := &worker{
		User:       "alice",
		Channel:    "general",
		Protocol:   robot.Test,
		Incoming:   &robot.ConnectorMessage{},
		cfg:        &configuration{alias: '!', botinfo: UserInfo{UserName: "Clu"}, adminUsers: []string{"parsley"}},
		tasks:      tasks,
		listedUser: true,
		pipeContext: &pipeContext{
			parameters:  map[string]string{"GOPHER_CMDMODE": "alias"},
			environment: map[string]string{},
		},
	}
	r := w.makeRobot()
	w.registerWorker(r.tid)
	defer deregisterWorker(r.tid)

	advice := r.collectFallbackAdvice("!launch-server")
	if advice.Advice != fallbackAdviceWrongChannel {
		t.Fatalf("advice = %q, want %q", advice.Advice, fallbackAdviceWrongChannel)
	}
	if len(advice.Elsewhere) != 1 || advice.Elsewhere[0].Command != "start-ide" {
		t.Fatalf("elsewhere = %+v, want theia start-ide", advice.Elsewhere)
	}
	if advice.DeterministicReply == "" {
		t.Fatal("expected deterministic reply")
	}
	if !strings.Contains(advice.DeterministicReply, "#devops") {
		t.Fatalf("deterministic reply %q missing devops hint", advice.DeterministicReply)
	}
}

func TestCollectFallbackAdviceSuppressesWeakConversationalMatches(t *testing.T) {
	tasks := &taskList{
		t: []interface{}{
			&Task{name: "namespace"},
			&Plugin{
				Task: &Task{
					name:     "builtin-help",
					Channels: []string{"general"},
				},
				Commands: []InputMatcher{{
					Command:  "info",
					Usage:    "(alias) info",
					Summary:  "Provide useful information for admins, including git branch status.",
					Keywords: []string{"admin", "status", "branch"},
				}},
			},
			&Plugin{
				Task: &Task{
					name:     "ping",
					Channels: []string{"general"},
				},
				Commands: []InputMatcher{{
					Command:  "whoami",
					Usage:    "(alias) whoami",
					Summary:  "Tell the user a little bit about themselves.",
					Keywords: []string{"user", "identity", "profile"},
				}},
			},
		},
		nameMap: map[string]int{
			"builtin-help": 1,
			"ping":         2,
		},
		nameSpaces:    map[string]ParameterSet{},
		parameterSets: map[string]ParameterSet{},
	}

	w := &worker{
		User:       "alice",
		Channel:    "general",
		Protocol:   robot.Test,
		Incoming:   &robot.ConnectorMessage{},
		cfg:        &configuration{alias: '!', botinfo: UserInfo{UserName: "Clu"}, adminUsers: []string{"parsley"}},
		tasks:      tasks,
		listedUser: true,
		pipeContext: &pipeContext{
			parameters:  map[string]string{"GOPHER_CMDMODE": "alias"},
			environment: map[string]string{},
		},
	}
	r := w.makeRobot()
	w.registerWorker(r.tid)
	defer deregisterWorker(r.tid)

	advice := r.collectFallbackAdvice("!tell me a joke")
	if advice.Advice != fallbackAdviceNoMatch {
		t.Fatalf("advice = %q, want %q", advice.Advice, fallbackAdviceNoMatch)
	}
	if len(advice.Here) != 0 {
		t.Fatalf("here = %+v, want no weak conversational matches", advice.Here)
	}
	if !strings.Contains(advice.DeterministicReply, "#general") {
		t.Fatalf("deterministic reply %q missing current channel context", advice.DeterministicReply)
	}
	if !strings.Contains(advice.DeterministicReply, "!commands") {
		t.Fatalf("deterministic reply %q missing generic help fallback", advice.DeterministicReply)
	}
}

func TestCollectFallbackAdviceUsesExamplesForWrongChannelHint(t *testing.T) {
	tasks := &taskList{
		t: []interface{}{
			&Task{name: "namespace"},
			&Plugin{
				Task: &Task{
					name:     "knock",
					Channels: []string{"chat"},
				},
				Commands: []InputMatcher{{
					Command:  "knock",
					Usage:    "tell me a knock-knock joke",
					Summary:  "Starts an interactive knock-knock joke.",
					Keywords: []string{"knock", "joke"},
					Examples: []string{"(bot) tell me another joke"},
				}},
			},
		},
		nameMap: map[string]int{
			"knock": 1,
		},
		nameSpaces:    map[string]ParameterSet{},
		parameterSets: map[string]ParameterSet{},
	}

	w := &worker{
		User:       "alice",
		Channel:    "general",
		Protocol:   robot.Test,
		Incoming:   &robot.ConnectorMessage{},
		cfg:        &configuration{alias: '!', botinfo: UserInfo{UserName: "Clu"}, adminUsers: []string{"parsley"}},
		tasks:      tasks,
		listedUser: true,
		pipeContext: &pipeContext{
			parameters:  map[string]string{"GOPHER_CMDMODE": "alias"},
			environment: map[string]string{},
		},
	}
	r := w.makeRobot()
	w.registerWorker(r.tid)
	defer deregisterWorker(r.tid)

	advice := r.collectFallbackAdvice("!tell me a joke")
	if advice.Advice != fallbackAdviceWrongChannel {
		t.Fatalf("advice = %q, want %q", advice.Advice, fallbackAdviceWrongChannel)
	}
	if len(advice.Elsewhere) != 1 || advice.Elsewhere[0].Command != "knock" {
		t.Fatalf("elsewhere = %+v, want knock", advice.Elsewhere)
	}
	if !strings.Contains(advice.DeterministicReply, "#chat") {
		t.Fatalf("deterministic reply %q missing chat hint", advice.DeterministicReply)
	}
	if !strings.Contains(advice.DeterministicReply, "looks close to `knock/knock`") {
		t.Fatalf("deterministic reply %q missing close-match phrasing", advice.DeterministicReply)
	}
}

func TestCollectFallbackAdviceSuggestsHighConfidencePhraseNearMiss(t *testing.T) {
	tasks := &taskList{
		t: []interface{}{
			&Task{name: "namespace"},
			&Plugin{
				Task: &Task{
					name:     "knock",
					Channels: []string{"general"},
				},
				Commands: []InputMatcher{{
					Command:  "knock",
					Usage:    "tell me a knock-knock joke",
					Summary:  "Starts an interactive knock-knock joke.",
					Keywords: []string{"knock", "joke"},
					Examples: []string{"(alias) tell me a knock-knock joke"},
				}},
			},
		},
		nameMap: map[string]int{
			"knock": 1,
		},
		nameSpaces:    map[string]ParameterSet{},
		parameterSets: map[string]ParameterSet{},
	}

	w := &worker{
		User:       "alice",
		Channel:    "general",
		Protocol:   robot.Test,
		Incoming:   &robot.ConnectorMessage{},
		cfg:        &configuration{alias: '!', botinfo: UserInfo{UserName: "Clu"}, adminUsers: []string{"parsley"}},
		tasks:      tasks,
		listedUser: true,
		pipeContext: &pipeContext{
			parameters:  map[string]string{"GOPHER_CMDMODE": "alias"},
			environment: map[string]string{},
		},
	}
	r := w.makeRobot()
	w.registerWorker(r.tid)
	defer deregisterWorker(r.tid)

	advice := r.collectFallbackAdvice("!tell me a jok")
	if advice.Advice != fallbackAdviceCloseHere {
		t.Fatalf("advice = %q, want %q", advice.Advice, fallbackAdviceCloseHere)
	}
	if len(advice.Here) != 1 || advice.Here[0].Command != "knock" {
		t.Fatalf("here = %+v, want knock", advice.Here)
	}
	if !strings.Contains(advice.DeterministicReply, "looks close to `knock/knock`") {
		t.Fatalf("deterministic reply %q missing strong suggestion", advice.DeterministicReply)
	}
	if !strings.Contains(advice.DeterministicReply, "Try: `!tell me a knock-knock joke`") {
		t.Fatalf("deterministic reply %q missing direct retry example", advice.DeterministicReply)
	}
	if !strings.Contains(advice.DeterministicReply, "!help knock/knock") {
		t.Fatalf("deterministic reply %q missing exact help path", advice.DeterministicReply)
	}
}

func TestCollectFallbackAdviceSuppressesBareIdentifierGuessForPhraseCommand(t *testing.T) {
	tasks := &taskList{
		t: []interface{}{
			&Task{name: "namespace"},
			&Plugin{
				Task: &Task{
					name:     "knock",
					Channels: []string{"general"},
				},
				Commands: []InputMatcher{{
					Command:       "knock",
					SimpleMatcher: "tell me a [another] [knock-knock] joke",
					Usage:         "tell me a knock-knock joke",
					Summary:       "Starts an interactive knock-knock joke.",
					Keywords:      []string{"knock", "joke"},
					Examples:      []string{"(alias) tell me a knock-knock joke"},
				}},
			},
		},
		nameMap: map[string]int{
			"knock": 1,
		},
		nameSpaces:    map[string]ParameterSet{},
		parameterSets: map[string]ParameterSet{},
	}

	w := &worker{
		User:       "alice",
		Channel:    "general",
		Protocol:   robot.Test,
		Incoming:   &robot.ConnectorMessage{},
		cfg:        &configuration{alias: '!', botinfo: UserInfo{UserName: "Clu"}, adminUsers: []string{"parsley"}},
		tasks:      tasks,
		listedUser: true,
		pipeContext: &pipeContext{
			parameters:  map[string]string{"GOPHER_CMDMODE": "alias"},
			environment: map[string]string{},
		},
	}
	r := w.makeRobot()
	w.registerWorker(r.tid)
	defer deregisterWorker(r.tid)

	advice := r.collectFallbackAdvice("!knok")
	if advice.Advice != fallbackAdviceCloseHere {
		t.Fatalf("advice = %q, want %q", advice.Advice, fallbackAdviceCloseHere)
	}
	if strings.Contains(advice.DeterministicReply, "looks close to `knock/knock`") {
		t.Fatalf("deterministic reply %q should not over-suggest a phrase command from bare identifier typo", advice.DeterministicReply)
	}
	if !strings.Contains(advice.DeterministicReply, "!help knock/knock") {
		t.Fatalf("deterministic reply %q missing exact-help recovery", advice.DeterministicReply)
	}
	if !strings.Contains(advice.DeterministicReply, "!commands") {
		t.Fatalf("deterministic reply %q missing generic guidance", advice.DeterministicReply)
	}
}

func TestCatchAllModeMatches(t *testing.T) {
	plugin := &Plugin{CatchAll: true, CatchAllModes: []string{"alias"}}
	if !catchAllModeMatches(plugin, "alias") {
		t.Fatal("expected alias catchall mode to match alias input")
	}
	if catchAllModeMatches(plugin, "name") {
		t.Fatal("did not expect alias catchall mode to match name input")
	}
	if !catchAllModeMatches(&Plugin{CatchAll: true}, "direct") {
		t.Fatal("expected generic catchall to match all command modes")
	}
}

func TestSelectCatchAllTargetFiltersByCommandMode(t *testing.T) {
	aliasPlugin := &Plugin{
		Task:          &Task{name: "alias-fallback"},
		CatchAll:      true,
		CatchAllModes: []string{"alias"},
	}
	namePlugin := &Plugin{
		Task:          &Task{name: "openai-fallback"},
		CatchAll:      true,
		CatchAllModes: []string{"name", "direct"},
	}

	specific, fallback, multipleSpecific, multipleFallback := selectCatchAllTarget([]interface{}{aliasPlugin, namePlugin}, "alias", func(task *Task, plugin *Plugin) (bool, bool) {
		return true, false
	})
	if multipleSpecific || multipleFallback {
		t.Fatalf("unexpected multiple matches: specific=%t fallback=%t", multipleSpecific, multipleFallback)
	}
	if specific != aliasPlugin {
		t.Fatalf("specific catchall = %#v, want alias plugin", specific)
	}
	if fallback != nil {
		t.Fatalf("fallback catchall = %#v, want nil", fallback)
	}
}
