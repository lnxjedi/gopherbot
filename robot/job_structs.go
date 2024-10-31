package robot

// Repository represents a buildable git repository, for CI/CD
type Repository struct {
	Type         string // task extending the namespace needs to match for parameters
	CloneURL     string
	Dependencies []string // List of repositories this one depends on; changes to a dependency trigger a build
	// Logs to keep for this repo; pointer allows "undefined" to be detected,
	// in which case the value is inherited from the build type.
	KeepLogs   *int
	Parameters []Parameter // per-repository parameters
}

// Parameter items are provided to jobs and plugins as environment variables
type Parameter struct {
	Name  string `yaml:"Name"`  // Name of the parameter
	Value string `yaml:"Value"` // Value of the parameter
}
