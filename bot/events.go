package bot

/* events.go - definitions for message disposition events used by the
integration testing framework. These mostly relate to security, and relate
what happened to a particular message. Note that these are only used for the
'test' and 'terminal' connectors, primarily for development and testing.
*/

//go:generate stringer -type=Event

// Generate String method with: go generate ./bot/

// Event is a type for message disposition events
type Event int

const (
	IgnoredUser Event = iota
	BotDirectMessage
	AdminCheckPassed
	AdminCheckFailed
	MultipleMatchesNoAction
	AuthNoRunMisconfigured
	AuthNoRunPlugNotAvailable
	AuthRanSuccess
	AuthRanFail
	AuthRanMechanismFailed
	AuthRanFailNormal
	AuthRanFailOther
	AuthNoRunNotFound
	ElevNoRunMisconfigured
	ElevNoRunNotAvailable
	ElevRanSuccess
	ElevRanFail
	ElevRanMechanismFailed
	ElevRanFailNormal
	ElevRanFailOther
	ElevNoRunNotFound
	CommandPluginRan
	AmbientPluginRan
	CatchAllsRan
	GoPluginRan
	ScriptPluginBadPath
	ScriptPluginBadInterpreter
	ScriptTaskRan
	ScriptPluginStderrOutput
	ScriptPluginErrExit
)
