module github.com/chzyer/readline

go 1.25.0

replace github.com/lnxjedi/gopherbot/robot => ../../robot

require (
	github.com/lnxjedi/gopherbot/robot v0.0.0
	github.com/chzyer/test v1.0.0
	golang.org/x/sys v0.43.0
)

require github.com/chzyer/logex v1.2.1 // indirect
