package bot

// constants.go - separate file for constants so stringer can run

// Protocol - connector protocols
type Protocol int

const (
	// Slack connector
	Slack Protocol = iota
	// Terminal connector
	Terminal
	// Test connector for automated test suites
	Test
)

type pipeAddFlavor int
type pipeAddType int
type taskType int

const (
	flavorSpawn pipeAddFlavor = iota
	flavorAdd
	flavorFinal
	flavorFail
)

const (
	typeTask pipeAddType = iota
	typePlugin
	typeJob
)

const (
	taskGo taskType = iota
	taskExternal
)

// Indicates what started the pipeline
type pipelineType int

const (
	plugCommand pipelineType = iota
	plugMessage
	catchAll
	jobTrigger
	spawnedTask
	scheduled
	jobCmd  // i.e. run job xx
	pipeAdd // from e.g. AddCommand
)

//go:generate stringer -type=Protocol constants.go
//go:generate stringer -type=pipeAddFlavor constants.go
//go:generate stringer -type=pipeAddType constants.go
//go:generate stringer -type=taskType constants.go
//go:generate stringer -type=pipelineType constants.go

// Generate String methods with: go generate ./bot/
