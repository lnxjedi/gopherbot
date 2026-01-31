package bot

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/lnxjedi/gopherbot/robot"
	"golang.org/x/sys/unix"
)

var (
	// Set for CLI commands
	cliOp   = false
	fileLog = false

	// CLI flags
	logFile         string
	overrideIDEMode bool
	plainlog        bool
	helpRequested   bool
	sshPortOverride int

	hostName  string
	ideMode   bool
	startMode string

	deployEnvironment string
)

const defaultLogFile = "robot.log"

func init() {
	hostName = os.Getenv("HOSTNAME")

	// loggers of last resort, initialize early and update in start.go
	botStdErrLogger = log.New(os.Stderr, "", log.LstdFlags)
	botStdOutLogger = log.New(os.Stdout, "", log.LstdFlags)
}

// Start gets the robot going
// SEE ALSO: start_t.go for "make test"
func Start(v VersionInfo) {
	botVersion = v

	var err error
	// Installpath is where the default config and stock external
	// plugins are.
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	// See initBot for homePath
	installPath, err = filepath.Abs(filepath.Dir(ex))
	if err != nil {
		panic(err)
	}

	var ok bool
	if deployEnvironment, ok = lookupEnv("GOPHER_ENVIRONMENT"); !ok {
		deployEnvironment = "production"
	}

	var overrideDevEnv string
	// Save args in case we need to spawn child
	args := os.Args[1:]
	// Process command-line flags
	lusage := "path to robot's log file (or 'stdout' or 'stderr')"
	flag.StringVar(&logFile, "log", "", lusage)
	flag.StringVar(&logFile, "l", "", "")
	envusage := "alternate dev environment override (default GOPHER_ENVIRONMENT / production)"
	flag.StringVar(&overrideDevEnv, "devenv", "", envusage)
	flag.StringVar(&overrideDevEnv, "d", "", "")
	ovusage := "Override GOPHER_IDE mode"
	flag.BoolVar(&overrideIDEMode, "override", false, ovusage)
	flag.BoolVar(&overrideIDEMode, "o", false, "")
	plusage := "omit timestamps from the log"
	flag.BoolVar(&plainlog, "plainlog", false, plusage)
	flag.BoolVar(&plainlog, "p", false, "")
	husage := "help for gopherbot"
	flag.BoolVar(&helpRequested, "help", false, husage)
	flag.BoolVar(&helpRequested, "h", false, "")
	spusage := "override SSH listen port for the local connector"
	flag.IntVar(&sshPortOverride, "ssh-port", 0, spusage)
	// TODO: Gopherbot CLI commands suck. Make them suck less.
	flag.Parse()

	if len(overrideDevEnv) > 0 {
		deployEnvironment = overrideDevEnv
	}
	if sshPortOverride > 0 {
		os.Setenv("GOPHER_SSH_PORT", fmt.Sprintf("%d", sshPortOverride))
	}

	/*
		To prevent inadvertently bootstrapping a production
		robot in a random directory, we force it to run in $HOME
		when in IDE mode. This behavior can only be overridden by
		unsetting GOPHER_IDE in the terminal before invoking gopherbot.
		(`unset GOPHER_IDE; gopherbot`).

		IDE mode also ensures that the protocol is "ssh" and the
		brain is "mem" (in-memory); these can be overridden with the "-o"
		CLI flag.
	*/
	_, ideMode = lookupEnv("GOPHER_IDE")
	startMode = detectStartupMode()
	if ideMode && (startMode != "test-dev") {
		homeDir := os.Getenv("HOME")
		os.Chdir(homeDir)
	} else if startMode == "test-dev" {
		cwd, _ := os.Getwd()
		configPath = cwd
	}

	logFlags := log.LstdFlags
	if plainlog {
		logFlags = 0
	}
	botStdErrLogger = log.New(os.Stderr, "", logFlags)
	botStdOutLogger = log.New(os.Stdout, "", logFlags)
	// Container support
	pid := os.Getpid()
	if pid == 1 {
		Log(robot.Info, "PID == 1, spawning child")
		bin, _ := os.Executable()
		// Used by autosetup
		os.Setenv("GOPHER_CONTAINER", "iscontainer")
		env := os.Environ()
		cmd := exec.Command(bin, args...)
		cmd.Env = env
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		raiseThreadPrivExternal("exec child process for container")
		err := cmd.Start()
		if err != nil {
			log.Fatal(err)
		}
		go initSigHandle(cmd.Process)
		if pid == 1 {
			go func() {
				var ws unix.WaitStatus
				// Reap children FOREVER...
				for {
					pid, err := unix.Wait4(-1, &ws, 0, nil)
					if err == nil {
						Log(robot.Debug, "Reaped child pid: %d, status: %+v", pid, ws)
					}
				}
			}()
		}
		cmd.Wait()
		Log(robot.Info, "Quitting on child exit")
		return
	}

	usage := `Usage: gopherbot [options] [command [command options] [command args]]
  "command" can be one of:
	decrypt - decrypt a string or file
	encrypt - encrypt a string or file
	gentotp - generate a user TOTP string
	delete - delete a memory
	dump (installed|configured) [path/to/file.yaml] -
	  read and dump a raw config file, for yaml troubleshooting
	fetch - fetch the contents of a memory
	init (protocol) - create a new robot in currect directory
	list - list robot memories
	run - run the robot (default)
	store - store a memory
	validate [path/to/robot_repo] - syntax check a robot's repository
	version - display the gopherbot version
  <command> -h for help on a given command

  Common options:`

	if helpRequested {
		fmt.Println(usage)
		flag.PrintDefaults()
		os.Exit(0)
	}

	var envFile string
	var fixed = []string{}
	// NOTE: the subdirectories in test/ all use private/environment
	for _, ef := range []string{"private/environment", ".env"} {
		if es, err := os.Stat(ef); err == nil {
			em := es.Mode()
			if (uint32(em) & 0077) != 0 {
				mask := os.FileMode(0700)
				want := em & mask
				if err := os.Chmod(ef, want); err != nil {
					log.Fatalf("Invalid file mode '%o' on environment file '%s', can't fix: %v", em, ef, err)
				}
				fixed = append(fixed, ef)
			}
			envFile = ef
		}
	}
	penvErr := godotenv.Overload(envFile)

	var logger *log.Logger
	var logOut *os.File

	cliOp = len(flag.Args()) > 0 && flag.Arg(0) != "run"
	var cliCommand string

	// Get CLI command and set up pre-initBot logging
	if cliOp {
		cliCommand = flag.Arg(0)
		logOut, err = os.OpenFile(defaultLogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalf("Error creating log file: (%T %v)\n", err, err)
		}
	} else {
		logOut = os.Stdout
	}
	logger = log.New(logOut, "", logFlags)
	botLogger.logger = logger
	if elle, ok := lookupEnv("GOPHER_LOGLEVEL"); ok {
		ele := logStrToLevel(elle)
		setLogLevel(ele)
	}
	var shortDesc string
	switch startMode {
	case "setup":
		shortDesc = "processes answerfile.txt/ANS* env vars "
	case "demo":
		shortDesc = "no configuration or env vars, demo robot"
	case "test-dev":
		shortDesc = "found existing configuration outside of 'custom/'"
	case "bootstrap":
		shortDesc = "env vars set, need to clone config"
		if _, ok := lookupEnv("GOPHER_DEPLOY_KEY"); !ok {
			Log(robot.Fatal, "unable to start in bootstrap mode with no GOPHER_DEPLOY_KEY in the environment (or .env)")
		}
	case "cli":
		shortDesc = fmt.Sprintf("running CLI command '%s'", cliCommand)
	case "ide":
		shortDesc = "local dev environment overriding protocol/brain"
	case "ide-override":
		shortDesc = "local dev environment with configured protocol/brain"
	case "production":
		shortDesc = "fully configured robot"
	default:
		shortDesc = "unknown"
	}
	Log(robot.Info, "******* GOPHERBOT STARTING UP -> mode '%s' (%s) with config dir: %s, and install dir: %s\n", startMode, shortDesc, configPath, installPath)
	checkprivsep()
	if penvErr != nil {
		Log(robot.Info, "No private environment loaded from '.env': %v\n", penvErr)
	} else {
		Log(robot.Info, "Loaded initial private environment from '%s'\n", envFile)
	}
	if len(fixed) > 0 {
		Log(robot.Warn, "Notice! Fixed invalid file modes for environment file(s): %s", strings.Join(fixed, ", "))
	}

	// Process CLI commands that don't need/want full initBot + brain
	switch cliCommand {
	case "dump", "validate":
		processCLI(usage)
		os.Exit(0)
	}

	// Create the 'bot and load configuration, supplying configpath and installpath.
	// When loading configuration, gopherbot first loads default configuration
	// from internal config, then loads from configpath/conf/..., which
	// overrides defaults.
	initBot()

	// Remove all GOPHER_ variables from the environment; nearly everywhere else,
	// os.Getenv|Setenv|LookupEnv are replaced with calls to getEnv, setEnv and lookupEnv -
	// see env_vars.go.
	scrubEnvironment()

	// Set up Logging
	var logDest string
	if cliOp {
		logDest = "robot.log"
	}
	// Override from CLI --log / -l
	if len(logFile) > 0 {
		logDest = logFile
	}
	if len(logDest) == 0 {
		logDest = currentCfg.logDest
	}
	// last ditch fallback - log to stdout
	if len(logDest) == 0 {
		logDest = "stdout"
	}
	if logDest == "stderr" {
		logOut = os.Stderr
	} else if logDest == "stdout" {
		logOut = os.Stdout
	} else {
		lf, err := os.OpenFile(logDest, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalf("Error creating log file: (%T %v)\n", err, err)
		}
		fileLog = true
		logFileName = logFile
		logOut = lf
	}

	logger = log.New(logOut, "", logFlags)
	botLogger.logger = logger

	setLogLevel(currentCfg.logLevel)

	if cliOp {
		go runBrain()
		processCLI(usage)
		brainQuit()
		os.Exit(0)
	}
	if currentCfg.protocol == "terminal" {
		localTerm = true
	}
	if currentCfg.protocol == "nullconn" {
		nullConn = true
	}
	initializeConnector, ok := connectors[currentCfg.protocol]
	if !ok {
		logger.Fatalf("No connector registered with name: %s", currentCfg.protocol)
	}

	// handler{} is just a placeholder struct for implementing the Handler interface
	conn := initializeConnector(handle, logger)

	// NOTE: we use setConnector instead of passing the connector to run()
	// because of the way Windows services were run. Maybe remove eventually?
	setConnector(conn)

	// Start the robot loops
	run()
	// ... and wait for the robot to stop
	restart := <-done
	Log(robot.Info, "robot quit/exit/restart")
	time.Sleep(time.Second)
	if restart {
		raiseThreadPrivExternal("restart is set, re-exec'ing")
		// Make sure all the GOPHER_* env vars are present for the
		// new process.
		restoreGopherEnvironment()
		bin, _ := os.Executable()
		env := os.Environ()
		defer func() {
			err := unix.Exec(bin, os.Args, env)
			if err != nil {
				fmt.Printf("Error re-exec'ing: %v", err)
			}
		}()
	}
}
