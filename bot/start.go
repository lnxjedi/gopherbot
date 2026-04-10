package bot

import (
	"flag"
	"fmt"
	"io"
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
	plainlog        bool
	helpRequested   bool
	sshPortOverride int
	aidevFlagToken  string

	hostName  string
	startMode string
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

	// Save args in case we need to spawn child
	args := os.Args[1:]
	// Process command-line flags
	lusage := "path to robot's log file (or 'stdout' or 'stderr')"
	plusage := "omit timestamps from the log"
	husage := "help for gopherbot"
	rootFlags := flag.NewFlagSet("gopherbot", flag.ContinueOnError)
	rootFlags.SetOutput(io.Discard)
	rootFlags.StringVar(&logFile, "log", "", lusage)
	rootFlags.StringVar(&logFile, "l", "", "")
	rootFlags.BoolVar(&plainlog, "plainlog", false, plusage)
	rootFlags.BoolVar(&plainlog, "p", false, "")
	rootFlags.BoolVar(&helpRequested, "help", false, husage)
	rootFlags.BoolVar(&helpRequested, "h", false, "")
	spusage := "override SSH listen port for the local connector"
	rootFlags.IntVar(&sshPortOverride, "ssh-port", 0, spusage)
	adusage := "enable AI development mode with an auth token"
	rootFlags.StringVar(&aidevFlagToken, "aidev", "", adusage)
	remainingArgs, err := func() ([]string, error) {
		if err := rootFlags.Parse(args); err != nil {
			return nil, err
		}
		return rootFlags.Args(), nil
	}()
	if err != nil {
		fmt.Printf("Error: %v\n\n", err)
		printCLIUsage()
		os.Exit(2)
	}
	if len(remainingArgs) > 0 {
		var code int
		switch remainingArgs[0] {
		case pipelineChildExecCommand:
			code = runPipelineChildExec()
		case pipelineChildRPCCommand:
			code = runPipelineChildRPC()
		default:
			code = -1
		}
		if code >= 0 {
			if code != 0 {
				os.Exit(code)
			}
			return
		}
	}
	if helpRequested {
		printCLIUsage()
		return
	}
	if len(remainingArgs) > 0 {
		command := remainingArgs[0]
		commandArgs := remainingArgs[1:]
		if command == "help" || command == "version" || command == "init" || !cliCommandKnown(command) || shouldShowCLICommandHelp(command, commandArgs) || (command == "run" && len(commandArgs) > 0) {
			os.Exit(processCLI(command, commandArgs))
		}
	}
	cliOp = len(remainingArgs) > 0 && remainingArgs[0] != "run"

	setAIDevToken(aidevFlagToken)

	startMode = detectStartupMode()
	if startMode == "test-dev" {
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
		// Available to startup/bootstrap flows running in container mode
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
	// Re-evaluate startup mode after private environment loading so
	// bootstrap decisions include values from process env or .env.
	startMode = detectStartupMode()

	var logger *log.Logger
	var logOut *os.File

	var cliCommand string
	var cliCommandArgs []string

	// Get CLI command and set up pre-initBot logging
	if cliOp {
		cliCommand = remainingArgs[0]
		cliCommandArgs = remainingArgs[1:]
		logOut = os.Stderr
	} else {
		logOut = os.Stdout
	}
	logger = log.New(logOut, "", logFlags)
	botLogger.logger = logger
	if cliOp {
		setLogLevel(robot.Warn)
	} else if elle, ok := lookupEnv("GOPHER_LOGLEVEL"); ok {
		ele := logStrToLevel(elle)
		setLogLevel(ele)
	}
	var shortDesc string
	switch startMode {
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

	// Process CLI commands that don't need/want full initBot + brain.
	if cliOp && cliCommandRunsBeforeInit(cliCommand) {
		os.Exit(processCLI(cliCommand, cliCommandArgs))
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
		logDest = "stderr"
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
	if isAIDevMode() && !cliOp {
		logDest = defaultLogFile
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

	if cliOp {
		setLogLevel(robot.Warn)
	} else {
		setLogLevel(currentCfg.logLevel)
	}

	if cliOp {
		go runBrain()
		code := processCLI(cliCommand, cliCommandArgs)
		brainQuit()
		os.Exit(code)
	}
	if currentCfg.protocol == "terminal" {
		localTerm = true
	}
	if currentCfg.protocol == "nullconn" {
		nullConn = true
	}
	if err := initializeConnectorRuntime(logger); err != nil {
		logger.Fatalf("Initializing connector runtime: %v", err)
	}

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
