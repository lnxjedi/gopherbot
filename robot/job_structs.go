package robot

// Parameter items are provided to jobs and plugins as environment variables
type Parameter struct {
	Name  string `yaml:"Name"`  // Name of the parameter
	Value string `yaml:"Value"` // Value of the parameter
}
