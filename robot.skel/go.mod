module robot.internal

go 1.25.0

require (
	github.com/lnxjedi/gopherbot/robot v0.0.0
	gopherbot.internal/lib v0.0.0
)

// For local development against a nearby Gopherbot checkout or install tree,
// uncomment and adjust these replace directives to point at the engine's
// shared Go surfaces:
//
// replace github.com/lnxjedi/gopherbot/robot => /opt/gopherbot/robot
// replace gopherbot.internal/lib => /opt/gopherbot/lib
