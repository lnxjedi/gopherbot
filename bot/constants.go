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

//go:generate stringer -type=Protocol constants.go
//go:generate stringer -type=pipeAddFlavor constants.go
//go:generate stringer -type=pipeAddType constants.go

// Generate String method with: go generate ./bot/
