package robot

// Repository represents a buildable git repository, for CI/CD
type Repository struct {
	Type         string // task extending the namespace needs to match for parameters
	CloneURL     string
	Dependencies []string    // List of repositories this one depends on; changes to a dependency trigger a build
	KeepHistory  int         // How many job logs to keep for this repo
	Parameters   []Parameter // per-repository parameters
}

// Parameter items are provided to jobs and plugins as environment variables
type Parameter struct {
	Name, Value string
}
