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

//go:generate stringer -type=Protocol constants.go

// Generate String method with: go generate ./bot/
