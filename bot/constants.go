package bot

// 65536
const maxIndex = 1 << 16

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
	initJob    // scheduled job schedule: @init
	jobCommand // i.e. run job xx
)

//go:generate stringer -type=pipeAddFlavor constants.go
//go:generate stringer -type=pipeAddType constants.go
//go:generate stringer -type=taskType constants.go
//go:generate stringer -type=pipelineType constants.go

// Generate String methods with: go generate ./bot/
