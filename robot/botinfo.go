package robot

// BotInfo is the connector-visible view of the robot's configured identity.
// It intentionally excludes engine-only runtime metadata.
type BotInfo struct {
	UserName  string
	Email     string
	FullName  string
	FirstName string
	LastName  string
	Alias     string
}
