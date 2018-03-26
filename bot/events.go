package bot

/* events.go - definitions for message disposition events used by the
integration testing framework.
*/

// Event is a type for message disposition events
type Event int

const (
	IgnoredUser Event = iota
	PluginRan
	CatchAllsRan
)
