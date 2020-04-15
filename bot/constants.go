package bot

const maxIndex = 2 ^ 16

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
	unset pipelineType = iota
	plugCommand
	plugMessage
	catchAll
	jobTrigger
	spawnedTask
	scheduled
	jobCommand // i.e. run job xx
)

//go:generate stringer -type=pipeAddFlavor constants.go
//go:generate stringer -type=pipeAddType constants.go
//go:generate stringer -type=taskType constants.go
//go:generate stringer -type=pipelineType constants.go

// Generate String methods with: go generate ./bot/
