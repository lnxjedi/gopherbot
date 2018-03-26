package bot

/*
start_t.go - non-interactive StartTest() function for automated "black box"
testing.
*/

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"
)

// Start a robot for testing, and return the exit / robot stopped channel
func StartTest(cfgdir, logfile string, t *testing.T) (<-chan struct{}, Connector) {
	wd, _ := os.Getwd()
	installdir := filepath.Dir(wd)
	localdir := filepath.Join(installdir, cfgdir)
	os.Setenv("GOPHER_INSTALLDIR", installdir)
	os.Setenv("GOPHER_CONFIGDIR", localdir)
	t.Logf("Initializing test bot with installdir: \"%s\" and localdir: \"%s\"", installdir, localdir)

	var botLogger *log.Logger
	if len(logfile) == 0 {
		botLogger = log.New(ioutil.Discard, "", 0)
	} else {
		lf, err := os.Create("/tmp/bot.log")
		if err != nil {
			log.Fatalf("Error creating log file: (%T %v)", err, err)
		}
		botLogger = log.New(lf, "", log.LstdFlags)
	}

	initBot(localdir, installdir, botLogger)

	initializeConnector, ok := connectors[robot.protocol]
	if !ok {
		botLogger.Fatalf("No connector registered with name: %s", robot.protocol)
	}

	// handler{} is just a placeholder struct for implementing the Handler interface
	h := handler{}
	conn := initializeConnector(h, botLogger)

	// NOTE: we use setConnector instead of passing the connector to run()
	// because of the way Windows services run. See 'start_win.go'.
	setConnector(conn)

	stopped := run()
	return stopped, conn
}
