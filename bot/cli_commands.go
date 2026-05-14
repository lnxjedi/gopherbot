package bot

import (
	"bytes"
	crand "crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"image/png"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/lnxjedi/gopherbot/robot"
	"github.com/pquerna/otp/totp"
)

type cliCommandSpec struct {
	Name           string
	SummaryUsage   string
	Summary        string
	HelpLines      []string
	RunsBeforeInit bool
}

func cliCommands() []cliCommandSpec {
	protocols := availableInitProtocols()
	protocolLine := "Available protocols: (none found)"
	if len(protocols) > 0 {
		protocolLine = fmt.Sprintf("Available protocols: %s", strings.Join(protocols, ", "))
	}
	return []cliCommandSpec{
		{
			Name:         "help",
			SummaryUsage: "help [command]",
			Summary:      "show general or subcommand help",
			HelpLines: []string{
				"Usage: gopherbot help [command]",
				"",
				"Shows general help, or detailed help for a specific subcommand.",
				"",
				"Examples:",
				"  gopherbot help",
				"  gopherbot help encrypt",
			},
			RunsBeforeInit: true,
		},
		{
			Name:         "encrypt",
			SummaryUsage: "encrypt [options] <string>",
			Summary:      "encrypt a string or file",
			HelpLines: []string{
				"Usage: gopherbot encrypt [options] <string>",
				"   or: gopherbot encrypt -file <path|->",
				"",
				"Encrypts a literal string argument or the contents of a file/stdin.",
				"",
				"Options:",
				"  -f, -file <path|->   file to encrypt; use - for stdin",
				"  -b, -binary          write raw ciphertext instead of base64",
				"",
				"Notes:",
				"  Requires robot encryption to be initialized from GOPHER_ENCRYPTION_KEY",
				"  or a loaded .env/private environment file.",
			},
			RunsBeforeInit: true,
		},
		{
			Name:         "decrypt",
			SummaryUsage: "decrypt [options] <base64>",
			Summary:      "decrypt a base64 string or file",
			HelpLines: []string{
				"Usage: gopherbot decrypt [options] <base64>",
				"   or: gopherbot decrypt -file <path|->",
				"",
				"Decrypts a base64 string argument or raw encrypted bytes from a file/stdin.",
				"",
				"Options:",
				"  -f, -file <path|->   file to decrypt; use - for stdin",
				"",
				"Notes:",
				"  Requires robot encryption to be initialized from GOPHER_ENCRYPTION_KEY",
				"  or a loaded .env/private environment file.",
			},
			RunsBeforeInit: true,
		},
		{
			Name:         "gentotp",
			SummaryUsage: "gentotp <username>",
			Summary:      "generate a user TOTP secret and QR image",
			HelpLines: []string{
				"Usage: gopherbot gentotp <username>",
				"",
				"Generates a TOTP secret for the named user, prints the secret plus an",
				"encrypted config snippet, and writes <username>.png for QR enrollment.",
			},
			RunsBeforeInit: true,
		},
		{
			Name:         "genkey",
			SummaryUsage: "genkey [options]",
			Summary:      "generate an encrypted binary key for an environment",
			HelpLines: []string{
				"Usage: gopherbot genkey [options]",
				"",
				"Generates a fresh robot data key encrypted by GOPHER_ENCRYPTION_KEY.",
				"By default the encrypted key is printed to stdout.",
				"",
				"Options:",
				"  -e, -environment <name>  environment name; defaults to GOPHER_ENVIRONMENT or production",
				"  -w, -write               write binary-encrypted-key[.<environment>] under the custom config dir",
				"  -force                   allow -write to replace an existing key file",
				"",
				"Notes:",
				"  For non-production environments, -write targets binary-encrypted-key.<environment>.",
				"  Replacing an existing key makes secrets encrypted by the old data key unreadable.",
			},
			RunsBeforeInit: true,
		},
		{
			Name:         "uuid",
			SummaryUsage: "uuid",
			Summary:      "generate and encrypt a random UUID",
			HelpLines: []string{
				"Usage: gopherbot uuid",
				"",
				"Generates a random UUID and prints both the plaintext value and an",
				"encrypted value suitable for custom/conf/variables/<environment>.yaml",
				"Secrets entries.",
				"",
				"Notes:",
				"  Requires robot encryption to be initialized from GOPHER_ENCRYPTION_KEY",
				"  or a loaded .env/private environment file.",
			},
			RunsBeforeInit: true,
		},
		{
			Name:         "delete",
			SummaryUsage: "delete <key>",
			Summary:      "delete a memory",
			HelpLines: []string{
				"Usage: gopherbot delete <key>",
				"",
				"Deletes the named brain memory key.",
			},
			RunsBeforeInit: true,
		},
		{
			Name:         "dump",
			SummaryUsage: "dump <installed|configured> <path>",
			Summary:      "expand and print a raw config file",
			HelpLines: []string{
				"Usage: gopherbot dump <installed|configured> <path>",
				"",
				"Reads conf/<path>, expands templates/includes, and prints the raw YAML.",
				"",
				"Examples:",
				"  gopherbot dump installed robot.yaml",
				"  gopherbot dump configured plugins/help.yaml",
			},
			RunsBeforeInit: true,
		},
		{
			Name:         "fetch",
			SummaryUsage: "fetch [options] <key>",
			Summary:      "fetch the contents of a memory",
			HelpLines: []string{
				"Usage: gopherbot fetch [options] <key>",
				"",
				"Reads a brain memory key and writes it to stdout.",
				"",
				"Options:",
				"  -b, -base64          encode the fetched value as base64",
			},
			RunsBeforeInit: true,
		},
		{
			Name:         "init",
			SummaryUsage: "init <protocol>",
			Summary:      "create a new robot answerfile in the current directory",
			HelpLines: []string{
				"Usage: gopherbot init <protocol>",
				"",
				"Creates answerfile.txt in the current directory from an installed template.",
				"If a local ./gopherbot symlink does not exist, gopherbot also tries to create it.",
				"",
				protocolLine,
			},
			RunsBeforeInit: true,
		},
		{
			Name:         "list",
			SummaryUsage: "list",
			Summary:      "list robot memories",
			HelpLines: []string{
				"Usage: gopherbot list",
				"",
				"Lists all stored brain memory keys.",
			},
			RunsBeforeInit: true,
		},
		{
			Name:         "run",
			SummaryUsage: "run",
			Summary:      "run the robot (same as no subcommand)",
			HelpLines: []string{
				"Usage: gopherbot run",
				"",
				"Starts the robot using the normal startup flow. This is the default when",
				"you invoke gopherbot without a subcommand.",
				"",
				"Use top-level options before 'run', for example:",
				"  gopherbot -log stderr run",
			},
		},
		{
			Name:         "store",
			SummaryUsage: "store <key> [file]",
			Summary:      "store a memory",
			HelpLines: []string{
				"Usage: gopherbot store <key> [file]",
				"",
				"Stores file contents in the named brain memory key.",
				"If [file] is omitted, stdin is used.",
			},
			RunsBeforeInit: true,
		},
		{
			Name:         "validate",
			SummaryUsage: "validate <path>",
			Summary:      "syntax-check a robot repository",
			HelpLines: []string{
				"Usage: gopherbot validate <path>",
				"",
				"Loads the target robot repository and validates its startup configuration",
				"without starting connectors.",
			},
			RunsBeforeInit: true,
		},
		{
			Name:         "version",
			SummaryUsage: "version",
			Summary:      "display the gopherbot version",
			HelpLines: []string{
				"Usage: gopherbot version",
				"",
				"Prints the linked version and commit.",
			},
			RunsBeforeInit: true,
		},
	}
}

func cliCommandByName(name string) (cliCommandSpec, bool) {
	for _, spec := range cliCommands() {
		if spec.Name == name {
			return spec, true
		}
	}
	return cliCommandSpec{}, false
}

func cliCommandKnown(name string) bool {
	_, ok := cliCommandByName(name)
	return ok
}

func cliCommandRunsBeforeInit(name string) bool {
	spec, ok := cliCommandByName(name)
	return ok && spec.RunsBeforeInit
}

func isCLIHelpArg(arg string) bool {
	switch arg {
	case "-h", "-help", "--help", "help":
		return true
	default:
		return false
	}
}

func shouldShowCLICommandHelp(command string, args []string) bool {
	if !cliCommandKnown(command) || len(args) == 0 {
		return false
	}
	return isCLIHelpArg(args[0])
}

func availableInitProtocols() []string {
	pattern := filepath.Join(installPath, "resources", "answerfiles", "*.txt")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil
	}
	protocols := make([]string, 0, len(matches))
	for _, match := range matches {
		base := filepath.Base(match)
		protocols = append(protocols, strings.TrimSuffix(base, filepath.Ext(base)))
	}
	sort.Strings(protocols)
	return protocols
}

func printCLIUsage() {
	fmt.Println("Usage: gopherbot [options] [command [command options] [command args]]")
	fmt.Println()
	fmt.Println("Commands:")
	for _, spec := range cliCommands() {
		fmt.Printf("  %-34s %s\n", spec.SummaryUsage, spec.Summary)
	}
	fmt.Println()
	fmt.Println("Help:")
	fmt.Println("  gopherbot -h")
	fmt.Println("  gopherbot help <command>")
	fmt.Println("  gopherbot <command> -h")
	fmt.Println()
	fmt.Println("Common options:")
	fmt.Println("  -h, -help                 show general help")
	fmt.Println("  -l, -level <level>        set the log level (trace, debug, info, audit, warn, error)")
	fmt.Println("  -L, -log <path>           path to robot's log file (or 'stdout' or 'stderr')")
	fmt.Println("  -p, -plainlog             omit timestamps from the log")
	fmt.Println("  -ssh-port <port>          override SSH listen port for the local connector")
	fmt.Println("  -aidev <token>            enable AI development mode with an auth token")
}

func printCLICommandHelp(command string) {
	spec, ok := cliCommandByName(command)
	if !ok {
		fmt.Printf("Error: unknown command %q\n\n", command)
		printCLIUsage()
		return
	}
	for _, line := range spec.HelpLines {
		fmt.Println(line)
	}
}

func newCLIFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

func processCLI(command string, args []string) int {
	if command != "help" && shouldShowCLICommandHelp(command, args) {
		printCLICommandHelp(command)
		return 0
	}

	var fileName string
	var encodeBinary bool
	var encodeBase64 bool
	var genkeyEnvironment string
	var genkeyWrite bool
	var genkeyForce bool

	encFlags := newCLIFlagSet("encrypt")
	encFlags.StringVar(&fileName, "file", "", "file to encrypt (or - for stdin)")
	encFlags.StringVar(&fileName, "f", "", "")
	encFlags.BoolVar(&encodeBinary, "binary", false, "binary dump (defauts to base64 encoded)")
	encFlags.BoolVar(&encodeBinary, "b", false, "")

	decFlags := newCLIFlagSet("decrypt")
	decFlags.StringVar(&fileName, "file", "", "file to decrypt (or - for stdin)")
	decFlags.StringVar(&fileName, "f", "", "")
	decFlags.BoolVar(&encodeBinary, "binary", false, "")
	decFlags.BoolVar(&encodeBinary, "b", false, "")

	totpFlags := newCLIFlagSet("gentotp")

	genkeyFlags := newCLIFlagSet("genkey")
	genkeyFlags.StringVar(&genkeyEnvironment, "environment", "", "environment name")
	genkeyFlags.StringVar(&genkeyEnvironment, "e", "", "")
	genkeyFlags.BoolVar(&genkeyWrite, "write", false, "write encrypted key file")
	genkeyFlags.BoolVar(&genkeyWrite, "w", false, "")
	genkeyFlags.BoolVar(&genkeyForce, "force", false, "replace existing encrypted key file")

	fetchFlags := newCLIFlagSet("fetch")
	fetchFlags.BoolVar(&encodeBase64, "base64", false, "encode memory as base64")
	fetchFlags.BoolVar(&encodeBase64, "b", false, "")

	switch command {
	case "help":
		switch len(args) {
		case 0:
			printCLIUsage()
			return 0
		case 1:
			printCLICommandHelp(args[0])
			if cliCommandKnown(args[0]) {
				return 0
			}
			return 2
		default:
			fmt.Println("Error: help accepts at most one command name")
			fmt.Println()
			printCLICommandHelp("help")
			return 2
		}
	case "encrypt":
		if err := encFlags.Parse(args); err != nil {
			if err == flag.ErrHelp {
				printCLICommandHelp(command)
				return 0
			}
			fmt.Printf("Error: %v\n\n", err)
			printCLICommandHelp(command)
			return 2
		}
		if len(fileName) == 0 && len(encFlags.Args()) != 1 {
			fmt.Println("Error: encrypt requires either a string argument or -file")
			fmt.Println()
			printCLICommandHelp(command)
			return 2
		}
		cliEncrypt(encFlags.Arg(0), fileName, encodeBinary)
	case "decrypt":
		if err := decFlags.Parse(args); err != nil {
			if err == flag.ErrHelp {
				printCLICommandHelp(command)
				return 0
			}
			fmt.Printf("Error: %v\n\n", err)
			printCLICommandHelp(command)
			return 2
		}
		if len(fileName) == 0 && len(decFlags.Args()) != 1 {
			fmt.Println("Error: decrypt requires either a base64 argument or -file")
			fmt.Println()
			printCLICommandHelp(command)
			return 2
		}
		cliDecrypt(decFlags.Arg(0), fileName)
	case "dump":
		setLogLevel(robot.Warn)
		if len(args) != 2 {
			fmt.Println("Error: dump requires a source and a path")
			fmt.Println()
			printCLICommandHelp(command)
			return 2
		}
		switch args[0] {
		case "installed", "configured":
			initCrypt()
			cliDump(args[0], args[1])
			return 0
		default:
			fmt.Printf("Error: dump source must be \"installed\" or \"configured\", got %q\n\n", args[0])
			printCLICommandHelp(command)
			return 2
		}
	case "gentotp":
		if err := totpFlags.Parse(args); err != nil {
			if err == flag.ErrHelp {
				printCLICommandHelp(command)
				return 0
			}
			fmt.Printf("Error: %v\n\n", err)
			printCLICommandHelp(command)
			return 2
		}
		if len(totpFlags.Args()) == 0 || len(totpFlags.Arg(0)) == 0 {
			fmt.Println("Error: gentotp requires a username")
			fmt.Println()
			printCLICommandHelp(command)
			return 2
		}
		cliTOTPgen(totpFlags.Arg(0))
	case "genkey":
		if err := genkeyFlags.Parse(args); err != nil {
			if err == flag.ErrHelp {
				printCLICommandHelp(command)
				return 0
			}
			fmt.Printf("Error: %v\n\n", err)
			printCLICommandHelp(command)
			return 2
		}
		if len(genkeyFlags.Args()) > 0 {
			fmt.Println("Error: genkey does not take positional arguments")
			fmt.Println()
			printCLICommandHelp(command)
			return 2
		}
		if err := cliGenKey(genkeyEnvironment, genkeyWrite, genkeyForce); err != nil {
			fmt.Printf("Error: %v\n", err)
			return 1
		}
	case "uuid":
		if len(args) > 0 {
			fmt.Println("Error: uuid does not take arguments")
			fmt.Println()
			printCLICommandHelp(command)
			return 2
		}
		if err := cliUUID(); err != nil {
			fmt.Printf("Error: %v\n", err)
			return 1
		}
	case "fetch":
		if err := fetchFlags.Parse(args); err != nil {
			if err == flag.ErrHelp {
				printCLICommandHelp(command)
				return 0
			}
			fmt.Printf("Error: %v\n\n", err)
			printCLICommandHelp(command)
			return 2
		}
		if len(fetchFlags.Args()) == 0 || len(fetchFlags.Arg(0)) == 0 {
			fmt.Println("Error: fetch requires a memory key")
			fmt.Println()
			printCLICommandHelp(command)
			return 2
		}
		initCLIBrainProvider()
		defer shutdownCLIBrainProvider()
		cliFetch(fetchFlags.Arg(0), encodeBase64)
	case "init":
		if len(args) != 1 {
			if len(args) == 0 {
				fmt.Println("Error: init requires a protocol name")
			} else {
				fmt.Println("Error: init accepts exactly one protocol name")
			}
			fmt.Println()
			printCLICommandHelp(command)
			return 2
		}
		if _, err := os.Stat("answerfile.txt"); err == nil {
			fmt.Println("Not over-writing existing 'answerfile.txt'")
			return 1
		}
		ansFile := filepath.Join(installPath, "resources", "answerfiles", args[0]+".txt")
		if _, err := os.Stat(ansFile); err != nil {
			fmt.Printf("Error: no answerfile template found for protocol %q\n", args[0])
			if protocols := availableInitProtocols(); len(protocols) > 0 {
				fmt.Printf("Available protocols: %s\n", strings.Join(protocols, ", "))
			}
			return 1
		}
		var ansBytes []byte
		var err error
		if ansBytes, err = os.ReadFile(ansFile); err != nil {
			fmt.Printf("Reading '%s': %v", ansFile, err)
			return 1
		}
		if err = os.WriteFile("answerfile.txt", ansBytes, 0600); err != nil {
			fmt.Printf("Writing 'answerfile.txt': %v", err)
			return 1
		}
		if _, err := os.Stat("gopherbot"); err == nil {
			fmt.Println("Edit 'answerfile.txt' and re-run gopherbot with no arguments to generate your robot.")
		} else {
			exeFile := filepath.Join(installPath, "gopherbot")
			err := os.Symlink(exeFile, "gopherbot")
			if err != nil {
				fmt.Println("Unable to create symlink for 'gopherbot'")
				fmt.Println("Edit 'answerfile.txt' and re-run gopherbot with no arguments to generate your robot.")
			} else {
				fmt.Println("Edit 'answerfile.txt' and run './gopherbot' with no arguments to generate your robot.")
			}
		}
		return 0
	case "store":
		if len(args) == 0 || len(args) > 2 {
			if len(args) == 0 {
				fmt.Println("Error: store requires a memory key")
			} else {
				fmt.Println("Error: store accepts at most a key and optional file")
			}
			fmt.Println()
			printCLICommandHelp(command)
			return 2
		}
		file := "-"
		if len(args) == 2 {
			file = args[1]
		}
		initCLIBrainProvider()
		defer shutdownCLIBrainProvider()
		cliStore(args[0], file)
	case "list":
		if len(args) > 0 {
			fmt.Println("Error: list does not take arguments")
			fmt.Println()
			printCLICommandHelp(command)
			return 2
		}
		initCLIBrainProvider()
		defer shutdownCLIBrainProvider()
		cliList()
	case "delete":
		if len(args) != 1 {
			fmt.Println("Error: delete requires exactly one memory key")
			fmt.Println()
			printCLICommandHelp(command)
			return 2
		}
		initCLIBrainProvider()
		defer shutdownCLIBrainProvider()
		cliDelete(args[0])
	case "validate":
		if len(args) != 1 {
			fmt.Println("Error: validate requires a path to a robot repository")
			fmt.Println()
			printCLICommandHelp(command)
			return 2
		}
		cliValidate(args[0])
	case "version":
		if len(args) > 0 {
			fmt.Println("Error: version does not take arguments")
			fmt.Println()
			printCLICommandHelp(command)
			return 2
		}
		fmt.Printf("Version %s, commit: %s\n", botVersion.Version, botVersion.Commit)
		return 0
	case "run":
		if len(args) > 0 && !shouldShowCLICommandHelp(command, args) {
			fmt.Println("Error: run does not take subcommand arguments")
			fmt.Println()
		}
		printCLICommandHelp(command)
		if len(args) > 0 && !shouldShowCLICommandHelp(command, args) {
			return 2
		}
		return 0
	default:
		fmt.Printf("Error: unknown command %q\n\n", command)
		printCLIUsage()
		return 2
	}
	return 0
}

var cliConfigInitialized bool

func initCLIConfigDirectory() {
	var err error
	homePath, err = os.Getwd()
	if err != nil {
		Log(robot.Warn, "Unable to get cwd")
	}
	h := handler{}
	if err := h.GetDirectory(configPath); err != nil {
		Log(robot.Fatal, "Unable to get/create config path: %s", configPath)
	}
	if filepath.IsAbs(configPath) {
		configFull = configPath
	} else {
		configFull = filepath.Join(homePath, configPath)
	}
}

func initCLIConfigOnly() {
	if cliConfigInitialized {
		return
	}
	currentCfg.configuration = &configuration{}
	initCLIConfigDirectory()

	encryptionInitialized := initCrypt()
	if encryptionInitialized {
		setEnv("GOPHER_ENCRYPTION_INITIALIZED", "initialized")
	} else {
		mode := detectStartupMode()
		switch mode {
		case "cli", "bootstrap", "production":
			Log(robot.Fatal, "unable to initialize encryption for startup mode '%s', no GOPHER_ENCRYPTION_KEY set in environment (or .env)", mode)
		default:
			cryptKey.Lock()
			cryptKey.key = make([]byte, 32)
			if _, err := crand.Read(cryptKey.key); err != nil {
				cryptKey.Unlock()
				Log(robot.Fatal, "Generating temporary encryption key: %v", err)
			}
			cryptKey.initialized = true
			cryptKey.Unlock()
			Log(robot.Info, "Initialized temporary encryption key for '%s' mode", mode)
		}
	}

	if err := loadConfig(true); err != nil {
		Log(robot.Fatal, "Loading initial configuration: %v", err)
	}
	if err := validatePrivsepStartupPolicy(currentCfg.privsepSupplementaryGroups); err != nil {
		Log(robot.Fatal, "Privilege separation startup validation failed: %v", err)
	}
	cliConfigInitialized = true
}

func initCLIBrainProvider() {
	initCLIConfigOnly()
	if interfaces.brain != nil {
		return
	}
	if len(currentCfg.brainProvider) > 0 {
		registration, ok := brainProviderRegistration(currentCfg.brainProvider)
		if !ok {
			Log(robot.Fatal, "No provider registered for brain: \"%s\"", currentCfg.brainProvider)
		}
		interfaces.brain = registration.Provider(handle)
		Log(robot.Info, "Initialized brain provider '%s'", currentCfg.brainProvider)
		return
	}
	registration, ok := brainProviderRegistration("mem")
	if !ok {
		Log(robot.Fatal, "No provider registered for default brain: \"mem\"")
	}
	interfaces.brain = registration.Provider(handle)
	Log(robot.Error, "No brain configured, falling back to default 'mem' brain - no memories will persist")
}

func shutdownCLIBrainProvider() {
	if interfaces.brain != nil {
		interfaces.brain.Shutdown()
	}
}

func generateEncryptedUUID() (string, string, error) {
	cryptKey.RLock()
	initialized := cryptKey.initialized
	key := cryptKey.key
	cryptKey.RUnlock()
	if !initialized {
		return "", "", fmt.Errorf("encryption not initialized; set GOPHER_ENCRYPTION_KEY or load a .env file first")
	}
	plain := uuid.NewString()
	ct, err := encrypt([]byte(plain), key)
	if err != nil {
		return "", "", fmt.Errorf("encrypting generated UUID: %w", err)
	}
	return plain, base64.StdEncoding.EncodeToString(ct), nil
}

func encryptPlaintextBase64(plaintext string) (string, error) {
	cryptKey.RLock()
	initialized := cryptKey.initialized
	key := cryptKey.key
	cryptKey.RUnlock()
	if !initialized {
		return "", fmt.Errorf("encryption not initialized; set GOPHER_ENCRYPTION_KEY or load a .env file first")
	}
	ct, err := encrypt([]byte(plaintext), key)
	if err != nil {
		return "", fmt.Errorf("encrypting secret: %w", err)
	}
	return base64.StdEncoding.EncodeToString(ct), nil
}

func ensureCLIEncryptionInitialized() error {
	cryptKey.RLock()
	initialized := cryptKey.initialized
	cryptKey.RUnlock()
	if initialized {
		return nil
	}
	initCLIConfigDirectory()
	if initCrypt() {
		return nil
	}
	return fmt.Errorf("encryption not initialized; set GOPHER_ENCRYPTION_KEY or load a .env file first")
}

func cliUUID() error {
	if err := ensureCLIEncryptionInitialized(); err != nil {
		return err
	}
	plain, encrypted, err := generateEncryptedUUID()
	if err != nil {
		return err
	}
	fmt.Printf("UUID: %s\n", plain)
	fmt.Printf("Encrypted: %s\n", encrypted)
	return nil
}

func cliTOTPgen(user string) {
	initCLIConfigOnly()
	if !cryptKey.initialized {
		fmt.Println("Error: encryption not initialized; set GOPHER_ENCRYPTION_KEY or load a .env file first")
		os.Exit(1)
	}
	issuer := currentCfg.botinfo.FullName
	if issuer == "" {
		issuer = "Gopherbot"
	}
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: user,
	})
	if err != nil {
		fmt.Printf("Error generating TOTP: %v\n", err)
		os.Exit(1)
	}
	secStr := key.Secret()
	fmt.Printf("Secret for %s: %s\n", user, secStr)
	ct, err := encrypt([]byte(secStr), cryptKey.key)
	if err != nil {
		fmt.Printf("Error encrypting: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Encrypted secret for custom/conf/variables/<environment>.yaml:\n")
	fmt.Printf("Secrets:\n  TOTP_%s: \"%s\"\n", strings.ToUpper(user), base64.StdEncoding.EncodeToString(ct))
	fmt.Printf("Reference it from configuration with: {{ secret \"TOTP_%s\" }}\n", strings.ToUpper(user))
	var buf bytes.Buffer
	img, imgerr := key.Image(400, 400)
	if imgerr != nil {
		fmt.Printf("Error generating image: %v\n", imgerr)
		os.Exit(1)
	}
	png.Encode(&buf, img)
	ferr := os.WriteFile(fmt.Sprintf("%s.png", user), buf.Bytes(), 0644)
	if ferr != nil {
		fmt.Printf("Error writing '%s.png': %v\n", user, imgerr)
		os.Exit(1)
	}
	fmt.Printf("Wrote '%s.png'\n", user)
}

func cliGenKey(environment string, writeFile, force bool) error {
	wrappingKey, ok := lookupEnv(keyEnv)
	if !ok || len(wrappingKey) < 32 {
		return fmt.Errorf("%s must be set and at least 32 bytes long", keyEnv)
	}
	env := strings.TrimSpace(environment)
	if env == "" {
		env = currentConfigTemplateEnvironment()
	}
	if err := validateConfigTemplateEnvironment(env); err != nil {
		return err
	}
	dataKey := make([]byte, 32)
	if _, err := crand.Read(dataKey); err != nil {
		return fmt.Errorf("generating random data key: %w", err)
	}
	encrypted, err := encrypt(dataKey, []byte(wrappingKey)[:32])
	if err != nil {
		return fmt.Errorf("encrypting generated data key: %w", err)
	}
	encoded := base64.StdEncoding.EncodeToString(encrypted)
	if !writeFile {
		fmt.Println(encoded)
		return nil
	}
	target := filepath.Join(configPath, encryptedKeyFile)
	if env != "production" {
		target = filepath.Join(configPath, encryptedKeyFile+"."+env)
	}
	if _, err := os.Stat(target); err == nil && !force {
		return fmt.Errorf("%s already exists; rerun with -force to replace it", target)
	} else if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("checking existing key file %q: %w", target, err)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
		return fmt.Errorf("creating key directory: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(target), filepath.Base(target)+".tmp-*")
	if err != nil {
		return fmt.Errorf("creating temporary key file: %w", err)
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			os.Remove(tmpName)
		}
	}()
	if _, err := tmp.WriteString(encoded); err != nil {
		tmp.Close()
		return fmt.Errorf("writing temporary key file: %w", err)
	}
	if err := tmp.Chmod(encryptedKeyFileMode); err != nil {
		tmp.Close()
		return fmt.Errorf("setting temporary key file permissions: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temporary key file: %w", err)
	}
	raiseThreadPriv("writing generated encrypted key")
	if err := os.Rename(tmpName, target); err != nil {
		return fmt.Errorf("installing generated key file: %w", err)
	}
	cleanup = false
	if err := enforceEncryptedKeyFilePermissions(target); err != nil {
		return fmt.Errorf("securing generated key file: %w", err)
	}
	fmt.Printf("Wrote %s\n", target)
	return nil
}

func cliEncrypt(item, file string, binary bool) {
	if err := ensureCLIEncryptionInitialized(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	if len(file) > 0 {
		var fc []byte
		var err error
		if file == "-" {
			fc, err = io.ReadAll(os.Stdin)
		} else {
			fc, err = os.ReadFile(file)
		}
		if err != nil {
			fmt.Printf("Error reading file: %v\n", err)
			os.Exit(1)
		}
		ct, err := encrypt(fc, cryptKey.key)
		if err != nil {
			fmt.Printf("Error encrypting: %v\n", err)
			os.Exit(1)
		}
		if binary {
			os.Stdout.Write(ct)
		} else {
			WriteBase64(os.Stdout, &ct)
		}
		return
	}
	if len(item) > 0 {
		encrypted, err := encryptPlaintextBase64(item)
		if err != nil {
			fmt.Printf("Error encrypting: %v\n", err)
			os.Exit(1)
		}
		if binary {
			ct, err := base64.StdEncoding.DecodeString(encrypted)
			if err != nil {
				fmt.Printf("Error encoding ciphertext: %v\n", err)
				os.Exit(1)
			}
			os.Stdout.Write(ct)
		} else {
			fmt.Println(encrypted)
		}
		return
	}
	os.Stderr.Write([]byte("Ingoring zero-length item\n"))
	os.Exit(1)
}

func cliDecrypt(item, file string) {
	if err := ensureCLIEncryptionInitialized(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	if len(file) > 0 {
		var ct *[]byte
		var err error
		if file == "-" {
			ct, err = ReadBinary(os.Stdin)
		} else {
			ct, err = ReadBinaryFile(file)
		}
		if err != nil {
			fmt.Printf("Error reading file: %v\n", err)
			os.Exit(1)
		}
		pt, err := decrypt(*ct, cryptKey.key)
		if err != nil {
			fmt.Printf("Error decrypting: %v\n", err)
		}
		os.Stdout.Write(pt)
		return
	}
	if len(item) > 0 {
		eb, err := base64.StdEncoding.DecodeString(item)
		if err != nil {
			fmt.Printf("Decoding base64: %v\n", err)
			os.Exit(1)
		}
		value, err := decrypt(eb, cryptKey.key)
		if err != nil {
			fmt.Printf("Error decrypting: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(value))
		return
	}
	os.Stderr.Write([]byte("Ingoring zero-length item\n"))
	os.Exit(1)
}

func cliFetch(item string, b64 bool) {
	_, datum, exists, ret := getDatum(item, false)
	if ret != robot.Ok {
		fmt.Printf("Retrieving datum: %v\n", ret)
		os.Exit(1)
	}
	if !exists {
		fmt.Println("Item not found")
		os.Exit(1)
	}
	if b64 {
		encoder := base64.NewEncoder(base64.StdEncoding, os.Stdout)
		encoder.Write(*datum)
		os.Stdout.Write([]byte("\n"))
		return
	}
	os.Stdout.Write(*datum)
	os.Stdout.Write([]byte("\n"))
}

func cliStore(key, file string) {
	var fc []byte
	var err error
	if file == "-" {
		fc, err = io.ReadAll(os.Stdin)
	} else {
		fc, err = os.ReadFile(file)
	}
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}
	ret := storeDatum(key, &fc)
	if ret != robot.Ok {
		fmt.Printf("Storing datum: %s\n", ret)
		return
	}
	fmt.Println("Stored")
}

func cliList() {
	brain := interfaces.brain
	list, err := brain.List()
	if err != nil {
		fmt.Printf("Listing memories: %v\n", err)
		return
	}
	if len(list) > 0 {
		for _, memory := range list {
			fmt.Println(memory)
		}
		return
	}
	fmt.Println("No memories found")
}

func cliDelete(key string) {
	brain := interfaces.brain
	err := brain.Delete(key)
	if err != nil {
		fmt.Printf("Deleting memory: %v\n", err)
		return
	}
	fmt.Println("Deleted")
}

func cliValidate(path string) {
	configPath = path
	testpath := filepath.Join(configPath, "conf", robotConfigFileName)
	_, err := os.Stat(testpath)
	if err != nil {
		fmt.Printf("Error: robot repository not found at %q (expected %s)\n", path, testpath)
		os.Exit(1)
	}
	botLogger.logger = log.New(os.Stdout, "", 0)
	fmt.Println("Validating configuration")
	initCLIConfigOnly()
	fmt.Println("Configuration valid")
}
