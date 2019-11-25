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

	"github.com/lnxjedi/gopherbot/robot"
)

var testInstallPath string

func init() {
	wd, _ := os.Getwd()
	testInstallPath = filepath.Dir(wd)
}

// StartTest will start a robot for testing, and return the exit / robot stopped channel
func StartTest(v VersionInfo, cfgdir, logfile string, t *testing.T) (<-chan bool, robot.Connector) {
	botVersion = v
	configpath := filepath.Join(testInstallPath, cfgdir)
	t.Logf("Initializing test bot with installpath: \"%s\" and configpath: \"%s\"", testInstallPath, configpath)

	var botLogger *log.Logger
	if len(logfile) == 0 {
		botLogger = log.New(ioutil.Discard, "", 0)
	} else {
		lf, err := os.Create(logfile)
		if err != nil {
			log.Fatalf("Error creating log file: (%T %v)", err, err)
		}
		botLogger = log.New(lf, "", log.LstdFlags)
	}

	initBot(configpath, testInstallPath, botLogger)

	initializeConnector, ok := connectors[botCfg.protocol]
	if !ok {
		botLogger.Fatalf("No connector registered with name: %s", botCfg.protocol)
	}

	// handler{} is just a placeholder struct for implementing the Handler interface
	conn := initializeConnector(handle, botLogger)

	// NOTE: we use setConnector instead of passing the connector to run()
	// because of the way Windows services were run. Maybe remove eventually?
	setConnector(conn)

	stopped := run()
	return stopped, conn
}
