// +build !modular test

package main

import (

	// *** Included Authorizer plugins
	_ "github.com/lnxjedi/gopherbot/goplugins/groups"

	// *** Included Go plugins, of varying quality
	_ "github.com/lnxjedi/gopherbot/goplugins/help"
	_ "github.com/lnxjedi/gopherbot/goplugins/ping"
)
