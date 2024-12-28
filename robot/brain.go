package robot

// Logger is used in various modules for logging errors
type Logger interface {
	Log(l LogLevel, m string, v ...interface{}) bool
}

// SimpleBrain is the simple interface for a configured brain, where the robot
// handles all locking issues.
type SimpleBrain interface {
	// Store stores a blob of data with a string key, returns error
	// if there's a problem storing the datum.
	Store(key string, blob *[]byte) error
	// Retrieve returns a blob of data (probably JSON) given a string key,
	// and exists=true if the data blob was found, or error if the brain
	// malfunctions.
	Retrieve(key string) (blob *[]byte, exists bool, err error)
	// List returns a list of all memories - Gopherbot isn't a database,
	// so it _should_ be pretty short.
	List() (keys []string, err error)
	// Delete deletes a memory
	Delete(key string) error
}
