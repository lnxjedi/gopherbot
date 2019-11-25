// +build !modular test

package main

// blank imports used in static builds
import (
	// Included connectors
	_ "github.com/lnxjedi/gopherbot/connectors/rocket"
	_ "github.com/lnxjedi/gopherbot/connectors/slack"
	// NOTE: if you build with '-tags test', the terminal connector will also
	// show emitted events.
	_ "github.com/lnxjedi/gopherbot/connectors/terminal"

	// A brain using AWS DynamoDB
	_ "github.com/lnxjedi/gopherbot/brains/dynamodb"

	// A joke telling plugin
	_ "github.com/lnxjedi/gopherbot/goplugins/knock"
)
