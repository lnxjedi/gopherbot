// +build !modular test

package main

// blank imports used in static builds
import (
	// Included connectors
	_ "github.com/lnxjedi/gopherbot/connectors/rocket"
	_ "github.com/lnxjedi/gopherbot/connectors/slack"

	// A brain using AWS DynamoDB
	_ "github.com/lnxjedi/gopherbot/brains/dynamodb"

	// A joke telling plugin
	_ "github.com/lnxjedi/gopherbot/goplugins/knock"

	// Memes! (courtesy of imgflip.com)
	_ "github.com/lnxjedi/gopherbot/goplugins/meme"

	// *** Included Elevator plugins
	_ "github.com/lnxjedi/gopherbot/goplugins/duo"
	_ "github.com/lnxjedi/gopherbot/goplugins/totp"
)
