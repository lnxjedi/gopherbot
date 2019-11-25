package main

import (
	"fmt"
	"os"
	"syscall"

	"github.com/lnxjedi/gopherbot/bot"

	// NOTE: If compiling gopherbot yourself, you can comment out or remove
	// most of the import lines below to shrink the binary or remove unwanted
	// or unneeded funcationality. You'll need at least one connector for your
	// bot to be useful, however.

	// *** Compiled-in brain implementations
	_ "github.com/lnxjedi/gopherbot/brains/file"

	// *** Compiled-in history implementations
	_ "github.com/lnxjedi/gopherbot/history/file"

	// Many included plugins already have 'Disabled: true', but you can also
	// disable by adding that line to conf/plugins/<plugname>.yaml

	// *** Included Elevator plugins

	_ "github.com/lnxjedi/gopherbot/goplugins/duo"
	_ "github.com/lnxjedi/gopherbot/goplugins/totp"

	// *** Included Authorizer plugins

	_ "github.com/lnxjedi/gopherbot/goplugins/groups"

	// *** Included Go plugins, of varying quality

	_ "github.com/lnxjedi/gopherbot/goplugins/help"
	_ "github.com/lnxjedi/gopherbot/goplugins/links"
	_ "github.com/lnxjedi/gopherbot/goplugins/lists"
	_ "github.com/lnxjedi/gopherbot/goplugins/meme"
	_ "github.com/lnxjedi/gopherbot/goplugins/ping"

	/* Enable profiling. You can shrink the binary by removing this, but if the
	   robot ever stops responding for any reason, it's handy for getting a
	   dump of all goroutines. Example usage:

	   $ go tool pprof http://localhost:8888/debug/pprof/goroutine
	   ...
	   Entering interactive mode (type "help" for commands, "o" for options)
	   (pprof) list lnxjedi
	   Total: 11
	   ROUTINE ======================== github.com/lnxjedi/gopherbot/bot...
	*/
	_ "net/http/pprof"
)

// Version of gopherbot
var Version = "v2.0.0-snapshot"

// Commit supplied during linking
var Commit = "(not set)"

func main() {
	versionInfo := bot.VersionInfo{
		Version: Version,
		Commit:  Commit,
	}
	restart := bot.Start(versionInfo)
	if restart {
		bot.DropThreadPriv("restarting")
		bin, _ := os.Executable()
		defer func() {
			err := syscall.Exec(bin, os.Args, os.Environ())
			if err != nil {
				fmt.Printf("Error re-exec'ing: %v", err)
			}
		}()
	}
}
