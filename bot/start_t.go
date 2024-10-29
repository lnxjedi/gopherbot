package bot

/*
start_t.go - non-interactive StartTest() function for automated "black box"
testing.
*/

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/lnxjedi/gopherbot/robot"
)

var testInstallPath string
var registrationsProcessed bool

func init() {
	wd, _ := os.Getwd()
	testInstallPath = filepath.Dir(wd)
}

// StartTest will start a robot for testing, and return the exit / robot stopped channel
func StartTest(v VersionInfo, cfgdir, logfile string, t *testing.T) (chan bool, robot.Connector) {
	botVersion = v

	// Collect all the Go Plugins, Jobs and Tasks
	// registered by various init() functions, but
	// only once during tests.
	if !registrationsProcessed {
		registrationsProcessed = true
		ProcessRegistrations()
	}

	configpath := filepath.Join(testInstallPath, cfgdir)
	t.Logf("Initializing test bot with installpath: '%s', configpath: '%s' and logfile: %s", testInstallPath, configpath, logfile)

	if botLogger.logger == nil {
		var testLogger *log.Logger
		botStdOutLogger = log.New(os.Stdout, "", log.LstdFlags)
		if len(logfile) == 0 {
			testLogger = log.New(io.Discard, "", 0)
		} else {
			lf, err := os.Create(logfile)
			if err != nil {
				log.Fatalf("Error creating log file: (%T %v)", err, err)
			}
			testLogger = log.New(lf, "", log.LstdFlags)
		}
		botLogger.logger = testLogger
	} else {
		lf, err := os.Create(logfile)
		if err != nil {
			log.Fatalf("Error creating log file: (%T %v)", err, err)
		}
		botLogger.setOutputFile(lf)
	}
	initBot(configpath, testInstallPath)

	initializeConnector, ok := connectors[currentCfg.protocol]
	if !ok {
		Log(robot.Fatal, "No connector registered with name: %s", currentCfg.protocol)
	}
	Log(robot.Info, "Starting new test with cfgdir: %s\n", cfgdir)

	// handler{} is just a placeholder struct for implementing the Handler interface
	conn := initializeConnector(handle, botLogger.logger)

	// NOTE: we use setConnector instead of passing the connector to run()
	// because of the way Windows services were run. Maybe remove eventually?
	setConnector(conn)

	run()

	bk := filepath.Join(testInstallPath, cfgdir, "binary-encrypted-key")
	if err := os.Remove(bk); err != nil {
		fmt.Printf("Removing temporary key: %v\n", err)
	}
	return done, conn
}
