package bot

/*
start_t.go - non-interactive StartTest() function for automated "black box"
testing.
*/

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/lnxjedi/robot"
)

var testInstallPath string

func init() {
	wd, _ := os.Getwd()
	testInstallPath = filepath.Dir(wd)
}

// StartTest will start a robot for testing, and return the exit / robot stopped channel
func StartTest(v VersionInfo, cfgdir, logfile string, t *testing.T) (chan bool, robot.Connector) {
	botVersion = v
	configpath := filepath.Join(testInstallPath, cfgdir)
	t.Logf("Initializing test bot with installpath: '%s', configpath: '%s' and logfile: %s", testInstallPath, configpath, logfile)

	if botLogger.l == nil {
		var testLogger *log.Logger
		botStdOutLogger = log.New(os.Stdout, "", log.LstdFlags)
		if len(logfile) == 0 {
			testLogger = log.New(ioutil.Discard, "", 0)
		} else {
			lf, err := os.Create(logfile)
			if err != nil {
				log.Fatalf("Error creating log file: (%T %v)", err, err)
			}
			testLogger = log.New(lf, "", log.LstdFlags)
		}
		botLogger.l = testLogger
	} else {
		lf, err := os.Create(logfile)
		if err != nil {
			log.Fatalf("Error creating log file: (%T %v)", err, err)
		}
		botLogger.setOutputFile(lf)
	}

	// install signal handler early for SIGUSR1 troubleshooting startup code.
	sigBreak := make(chan struct{})
	go sigHandle(sigBreak)

	initBot(configpath, testInstallPath)

	initializeConnector, ok := connectors[currentCfg.protocol]
	if !ok {
		Log(robot.Fatal, "No connector registered with name: %s", currentCfg.protocol)
	}
	Log(robot.Info, "Starting new test with cfgdir: %s\n", cfgdir)

	// handler{} is just a placeholder struct for implementing the Handler interface
	conn := initializeConnector(handle, botLogger.l)

	// NOTE: we use setConnector instead of passing the connector to run()
	// because of the way Windows services were run. Maybe remove eventually?
	setConnector(conn)

	run(sigBreak)

	bk := filepath.Join(testInstallPath, cfgdir, "binary-encrypted-key")
	if err := os.Remove(bk); err != nil {
		fmt.Printf("Removing temporary key: %v\n", err)
	}
	return done, conn
}
