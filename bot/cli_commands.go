package bot

import (
	"bytes"
	"context"
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
	"time"

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
			Name:         "check",
			SummaryUsage: "check [options] <simple matcher> <command text>",
			Summary:      "test one SimpleMatcher against example command text",
			HelpLines: []string{
				"Usage: gopherbot check [options] <simple matcher> <command text>",
				"",
				"Compiles one SimpleMatcher and tests it against the command text.",
				"This command does not load robot configuration and can run outside",
				"a configured robot repository.",
				"",
				"Options:",
				"  -json, -j   write structured JSON output",
				"",
				"Examples:",
				"  gopherbot check 'spot (type:rails|devops) up [<branch:token>]' spot-devops-up",
				"  gopherbot check -json 'set loglevel {to} (level:trace|debug|info|warn|error)' set-loglevel-info",
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
			Name:         "match",
			SummaryUsage: "match [options] <command text>",
			Summary:      "match command text against configured plugin commands",
			HelpLines: []string{
				"Usage: gopherbot match [options] <command text>",
				"   or: gopherbot match -interactive [options]",
				"",
				"Loads robot configuration, inspects directed plugin Commands, and",
				"reports exact matches or SimpleMatcher syntax diagnostics. It does",
				"not start connectors or execute matching pipelines.",
				"",
				"Options:",
				"  -interactive   prompt for command text until EOF",
				"  -json, -j      write structured JSON output",
				"",
				"Examples:",
				"  gopherbot match spot-devops-up",
				"  gopherbot match -interactive",
				"  gopherbot match -json get-console qa",
			},
			RunsBeforeInit: true,
		},
		{
			Name:         "syntax",
			SummaryUsage: "syntax [options] <script> [script...]",
			Summary:      "syntax-check built-in interpreter scripts",
			HelpLines: []string{
				"Usage: gopherbot syntax [options] <script> [script...]",
				"",
				"Checks Lua, JavaScript, GSH, and interpreted Go scripts without",
				"loading robot configuration or starting connectors.",
				"",
				"Options:",
				"  -language <lua|js|gsh|go>   override language detection by extension",
				"  -json, -j                    write structured JSON output",
				"",
				"Examples:",
				"  gopherbot syntax plugins/foo.lua",
				"  gopherbot syntax -json plugins/foo.lua plugins/bar.js",
				"  gopherbot syntax -language gsh scripts/local-check",
			},
			RunsBeforeInit: true,
		},
		{
			Name:         "script",
			SummaryUsage: "script [options] <script> [--] <command> [args...]",
			Summary:      "run a built-in interpreter script locally",
			HelpLines: []string{
				"Usage: gopherbot script [options] <script> [--] <command> [args...]",
				"   or: gopherbot script [options] -c <source> [--] <command> [args...]",
				"   or: gopherbot script -new-fixture <path>",
				"",
				"Runs a Lua, JavaScript, GSH, or interpreted Go script from the filesystem",
				"using a local fixture-backed robot API. The script path is explicit and",
				"resolved from the current working directory when relative.",
				"",
				"Plugin mode prepends <command> before capture args, matching configured",
				"plugin execution. Use -- before the command when args may look like flags.",
				"Task and job modes pass args as-is.",
				"",
				"Options:",
				"  -c <source>                  run inline source; requires -language",
				"  -fixture <path>              load YAML or JSON fixture data",
				"                               defaults to installed conf/default-fixture.yaml",
				"  -new-fixture <path>          copy installed conf/default-fixture.yaml to path",
				"  -force                       allow -new-fixture to overwrite an existing file",
				"  -kind <plugin|job|task>      execution kind; default plugin",
				"  -language <lua|js|gsh|go>   override language detection by extension",
				"  -no-interactive             fail prompts when fixture replies run out",
				"  -workdir <path>              child process working directory",
				"  -json, -j                    write structured JSON output",
				"",
				"Examples:",
				"  gopherbot script -new-fixture local-fixture.yaml",
				"  gopherbot script plugins/foo.lua -- console qa",
				"  gopherbot script -fixture test-scripts/fixtures/cat.json test-scripts/lua/demo.lua -- prompt",
				"  gopherbot script -language lua -c 'local g=require(\"gopherbot_v1\"); return g.task.Normal' -- _init",
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
				"Deletes the named brain memory key from the local cache and, for",
				"cloud-backed brains, flushes the delete tombstone to cloud before exiting.",
			},
			RunsBeforeInit: true,
		},
		{
			Name:         "flush-brain",
			SummaryUsage: "flush-brain",
			Summary:      "flush queued brain writes to cloud",
			HelpLines: []string{
				"Usage: gopherbot flush-brain",
				"",
				"Flushes any queued local brain cache writes to the configured cloud brain.",
				"For file and mem brains this is a no-op.",
			},
			RunsBeforeInit: true,
		},
		{
			Name:         "dump",
			SummaryUsage: "dump [options] <installed|configured> <path>",
			Summary:      "expand and print a raw config file",
			HelpLines: []string{
				"Usage: gopherbot dump [options] <installed|configured> <path>",
				"",
				"Reads conf/<path>, expands templates/includes, and prints the raw YAML.",
				"Secret template values are redacted by default.",
				"",
				"Options:",
				"  -unredacted-secrets   print decrypted secret template values",
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
				"By default, fetch reads the local cache only.",
				"",
				"Options:",
				"  -b, -base64          encode the fetched value as base64",
				"      -validate-cloud  verify local checksum/version against v3 cloud before printing",
				"      -cloud           read directly from v3 cloud instead of local cache",
				"      -update-cache    with -cloud, update the existing local cache with the cloud value",
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
				"Usage: gopherbot list [options]",
				"",
				"Lists local brain cache memory keys by default.",
				"",
				"Options:",
				"      -cloud           list keys from the configured cloud brain",
			},
			RunsBeforeInit: true,
		},
		{
			Name:         "pull-brain",
			SummaryUsage: "pull-brain [options]",
			Summary:      "pull remote or legacy brain memories into the local cache",
			HelpLines: []string{
				"Usage: gopherbot pull-brain [options]",
				"",
				"Imports remote brain memories into the local v3 cache.",
				"By default it does not modify the remote brain.",
				"",
				"Options:",
				"  -dry-run             report planned work without writing",
				"  -force               replace existing local cache",
				"  -upgrade-cloud-v3    also write upgraded v3 records to cloud",
				"  -budget <n>          maximum cloud writes for this run",
			},
			RunsBeforeInit: true,
		},
		{
			Name:         "restore-brain",
			SummaryUsage: "restore-brain [-v2] [options]",
			Summary:      "write the local brain cache to a remote provider",
			HelpLines: []string{
				"Usage: gopherbot restore-brain [-v2] [options]",
				"",
				"Writes the local v3 cache back to the configured remote brain.",
				"Defaults to v3 output before starting the v3 runtime.",
				"Use -v2 only for rollback export to v2-compatible cloud data.",
				"",
				"Options:",
				"  -dry-run              report planned work without writing",
				"  -force                allow initializing/replacing remote state",
				"  -v2                   write v2-compatible cloud data instead of v3",
				"  -budget <n>           maximum cloud writes for this run",
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
				"For cloud-backed brains, writes the local cache and flushes cloud sync",
				"before exiting.",
			},
			RunsBeforeInit: true,
		},
		{
			Name:         "validate",
			SummaryUsage: "validate [options] <path>",
			Summary:      "syntax-check a robot repository",
			HelpLines: []string{
				"Usage: gopherbot validate [options] <path>",
				"",
				"Loads the target robot repository and validates its startup configuration",
				"without starting connectors.",
				"",
				"Options:",
				"  -redacted-secrets   substitute secret template values with redacted placeholders",
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
	var genkeyEnvironment string
	var genkeyWrite bool
	var genkeyForce bool
	var fetchOpts cliFetchOptions
	var listCloud bool
	var checkJSON bool
	var matchJSON bool
	var matchInteractive bool
	var dumpUnredactedSecrets bool
	var validateRedactedSecrets bool
	var syntaxLanguage string
	var syntaxJSON bool
	var scriptOpts cliScriptOptions

	encFlags := newCLIFlagSet("encrypt")
	encFlags.StringVar(&fileName, "file", "", "file to encrypt (or - for stdin)")
	encFlags.StringVar(&fileName, "f", "", "")
	encFlags.BoolVar(&encodeBinary, "binary", false, "binary dump (defauts to base64 encoded)")
	encFlags.BoolVar(&encodeBinary, "b", false, "")

	checkFlags := newCLIFlagSet("check")
	checkFlags.BoolVar(&checkJSON, "json", false, "write structured JSON output")
	checkFlags.BoolVar(&checkJSON, "j", false, "")

	decFlags := newCLIFlagSet("decrypt")
	decFlags.StringVar(&fileName, "file", "", "file to decrypt (or - for stdin)")
	decFlags.StringVar(&fileName, "f", "", "")
	decFlags.BoolVar(&encodeBinary, "binary", false, "")
	decFlags.BoolVar(&encodeBinary, "b", false, "")

	dumpFlags := newCLIFlagSet("dump")
	dumpFlags.BoolVar(&dumpUnredactedSecrets, "unredacted-secrets", false, "print decrypted secret template values")

	validateFlags := newCLIFlagSet("validate")
	validateFlags.BoolVar(&validateRedactedSecrets, "redacted-secrets", false, "substitute secret template values with redacted placeholders")

	totpFlags := newCLIFlagSet("gentotp")

	genkeyFlags := newCLIFlagSet("genkey")
	genkeyFlags.StringVar(&genkeyEnvironment, "environment", "", "environment name")
	genkeyFlags.StringVar(&genkeyEnvironment, "e", "", "")
	genkeyFlags.BoolVar(&genkeyWrite, "write", false, "write encrypted key file")
	genkeyFlags.BoolVar(&genkeyWrite, "w", false, "")
	genkeyFlags.BoolVar(&genkeyForce, "force", false, "replace existing encrypted key file")

	fetchFlags := newCLIFlagSet("fetch")
	fetchFlags.BoolVar(&fetchOpts.base64, "base64", false, "encode memory as base64")
	fetchFlags.BoolVar(&fetchOpts.base64, "b", false, "")
	fetchFlags.BoolVar(&fetchOpts.validateCloud, "validate-cloud", false, "verify local checksum/version against cloud")
	fetchFlags.BoolVar(&fetchOpts.cloud, "cloud", false, "read directly from cloud")
	fetchFlags.BoolVar(&fetchOpts.updateCache, "update-cache", false, "with -cloud, update local cache")

	listFlags := newCLIFlagSet("list")
	listFlags.BoolVar(&listCloud, "cloud", false, "list remote cloud keys")

	matchFlags := newCLIFlagSet("match")
	matchFlags.BoolVar(&matchInteractive, "interactive", false, "prompt for command text until EOF")
	matchFlags.BoolVar(&matchJSON, "json", false, "write structured JSON output")
	matchFlags.BoolVar(&matchJSON, "j", false, "")

	syntaxFlags := newCLIFlagSet("syntax")
	syntaxFlags.StringVar(&syntaxLanguage, "language", "", "override language detection")
	syntaxFlags.BoolVar(&syntaxJSON, "json", false, "write structured JSON output")
	syntaxFlags.BoolVar(&syntaxJSON, "j", false, "")

	scriptFlags := newCLIFlagSet("script")
	scriptFlags.StringVar(&scriptOpts.inlineSource, "c", "", "inline source to run")
	scriptFlags.StringVar(&scriptOpts.fixturePath, "fixture", "", "YAML or JSON fixture path")
	scriptFlags.StringVar(&scriptOpts.newFixture, "new-fixture", "", "copy installed default fixture to path")
	scriptFlags.BoolVar(&scriptOpts.force, "force", false, "overwrite destination for -new-fixture")
	scriptFlags.StringVar(&scriptOpts.kind, "kind", "", "execution kind")
	scriptFlags.StringVar(&scriptOpts.language, "language", "", "override language detection")
	scriptFlags.BoolVar(&scriptOpts.noInteractive, "no-interactive", false, "fail prompts when fixture replies run out")
	scriptFlags.StringVar(&scriptOpts.workDir, "workdir", "", "child process working directory")
	scriptFlags.BoolVar(&scriptOpts.jsonOutput, "json", false, "write structured JSON output")
	scriptFlags.BoolVar(&scriptOpts.jsonOutput, "j", false, "")

	pullBrainFlags := newCLIFlagSet("pull-brain")
	var pullBrainOpts brainPullOptions
	pullBrainFlags.BoolVar(&pullBrainOpts.force, "force", false, "replace existing local cache")
	pullBrainFlags.BoolVar(&pullBrainOpts.dryRun, "dry-run", false, "report planned work without writing")
	pullBrainFlags.BoolVar(&pullBrainOpts.upgradeCloudV3, "upgrade-cloud-v3", false, "write upgraded v3 records to cloud")
	pullBrainFlags.IntVar(&pullBrainOpts.budget, "budget", 0, "maximum cloud writes")

	restoreBrainFlags := newCLIFlagSet("restore-brain")
	var restoreBrainOpts brainRestoreOptions
	restoreBrainFlags.BoolVar(&restoreBrainOpts.force, "force", false, "allow initializing/replacing remote state")
	restoreBrainFlags.BoolVar(&restoreBrainOpts.dryRun, "dry-run", false, "report planned work without writing")
	restoreBrainFlags.BoolVar(&restoreBrainOpts.v2, "v2", false, "write v2-compatible cloud data instead of v3")
	restoreBrainFlags.IntVar(&restoreBrainOpts.budget, "budget", 0, "maximum cloud writes")

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
	case "check":
		if err := checkFlags.Parse(args); err != nil {
			if err == flag.ErrHelp {
				printCLICommandHelp(command)
				return 0
			}
			fmt.Printf("Error: %v\n\n", err)
			printCLICommandHelp(command)
			return 2
		}
		return processCLICheckCommand(checkFlags.Args(), checkJSON)
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
		if err := dumpFlags.Parse(args); err != nil {
			if err == flag.ErrHelp {
				printCLICommandHelp(command)
				return 0
			}
			fmt.Printf("Error: %v\n\n", err)
			printCLICommandHelp(command)
			return 2
		}
		dumpArgs := dumpFlags.Args()
		if len(dumpArgs) != 2 {
			fmt.Println("Error: dump requires a source and a path")
			fmt.Println()
			printCLICommandHelp(command)
			return 2
		}
		switch dumpArgs[0] {
		case "installed", "configured":
			if dumpUnredactedSecrets {
				initCrypt()
			}
			cliDump(dumpArgs[0], dumpArgs[1], dumpUnredactedSecrets)
			return 0
		default:
			fmt.Printf("Error: dump source must be \"installed\" or \"configured\", got %q\n\n", dumpArgs[0])
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
		if len(fetchFlags.Args()) > 1 {
			fmt.Println("Error: fetch accepts exactly one memory key")
			fmt.Println()
			printCLICommandHelp(command)
			return 2
		}
		if fetchOpts.updateCache && !fetchOpts.cloud {
			fmt.Println("Error: fetch -update-cache requires -cloud")
			fmt.Println()
			printCLICommandHelp(command)
			return 2
		}
		if fetchOpts.cloud && fetchOpts.validateCloud {
			fmt.Println("Error: fetch -cloud and -validate-cloud are mutually exclusive")
			fmt.Println()
			printCLICommandHelp(command)
			return 2
		}
		fetchOpts.key = fetchFlags.Arg(0)
		if err := cliFetch(fetchOpts); err != nil {
			fmt.Printf("Error: %v\n", err)
			return 1
		}
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
		if err := cliStore(args[0], file); err != nil {
			fmt.Printf("Error: %v\n", err)
			shutdownCLIBrainProvider(false)
			return 1
		}
		shutdownCLIBrainProvider(true)
		reportLocalCloudOutboxStatus()
		fmt.Println("Stored")
	case "list":
		if err := listFlags.Parse(args); err != nil {
			if err == flag.ErrHelp {
				printCLICommandHelp(command)
				return 0
			}
			fmt.Printf("Error: %v\n\n", err)
			printCLICommandHelp(command)
			return 2
		}
		if len(listFlags.Args()) > 0 {
			fmt.Println("Error: list does not take positional arguments")
			fmt.Println()
			printCLICommandHelp(command)
			return 2
		}
		if err := cliList(listCloud); err != nil {
			fmt.Printf("Error: %v\n", err)
			return 1
		}
	case "match":
		if err := matchFlags.Parse(args); err != nil {
			if err == flag.ErrHelp {
				printCLICommandHelp(command)
				return 0
			}
			fmt.Printf("Error: %v\n\n", err)
			printCLICommandHelp(command)
			return 2
		}
		return processCLIMatchCommand(matchFlags.Args(), matchJSON, matchInteractive)
	case "syntax":
		if err := syntaxFlags.Parse(args); err != nil {
			if err == flag.ErrHelp {
				printCLICommandHelp(command)
				return 0
			}
			fmt.Printf("Error: %v\n\n", err)
			printCLICommandHelp(command)
			return 2
		}
		return processCLISyntaxCommand(syntaxFlags.Args(), syntaxLanguage, syntaxJSON)
	case "script":
		if err := scriptFlags.Parse(args); err != nil {
			if err == flag.ErrHelp {
				printCLICommandHelp(command)
				return 0
			}
			fmt.Printf("Error: %v\n\n", err)
			printCLICommandHelp(command)
			return 2
		}
		return processCLIScriptCommand(scriptFlags.Args(), scriptOpts)
	case "delete":
		if len(args) != 1 {
			fmt.Println("Error: delete requires exactly one memory key")
			fmt.Println()
			printCLICommandHelp(command)
			return 2
		}
		initCLIBrainProvider()
		if err := cliDelete(args[0]); err != nil {
			fmt.Printf("Error: %v\n", err)
			shutdownCLIBrainProvider(false)
			return 1
		}
		shutdownCLIBrainProvider(true)
		reportLocalCloudOutboxStatus()
		fmt.Println("Deleted")
	case "flush-brain":
		if len(args) > 0 {
			fmt.Println("Error: flush-brain does not take arguments")
			fmt.Println()
			printCLICommandHelp(command)
			return 2
		}
		initCLIBrainProvider()
		if err := interfaces.brain.Flush(); err != nil {
			fmt.Printf("Error: %v\n", err)
			shutdownCLIBrainProvider(false)
			return 1
		}
		reportLocalCloudOutboxStatus()
		shutdownCLIBrainProvider(false)
		fmt.Println("Brain flushed")
	case "pull-brain":
		if err := pullBrainFlags.Parse(args); err != nil {
			if err == flag.ErrHelp {
				printCLICommandHelp(command)
				return 0
			}
			fmt.Printf("Error: %v\n\n", err)
			printCLICommandHelp(command)
			return 2
		}
		if len(pullBrainFlags.Args()) > 0 {
			fmt.Println("Error: pull-brain does not take positional arguments")
			fmt.Println()
			printCLICommandHelp(command)
			return 2
		}
		if err := cliPullBrain(pullBrainOpts); err != nil {
			fmt.Printf("Error: %v\n", err)
			return 1
		}
	case "restore-brain":
		if err := restoreBrainFlags.Parse(args); err != nil {
			if err == flag.ErrHelp {
				printCLICommandHelp(command)
				return 0
			}
			fmt.Printf("Error: %v\n\n", err)
			printCLICommandHelp(command)
			return 2
		}
		if len(restoreBrainFlags.Args()) > 0 {
			fmt.Println("Error: restore-brain does not take positional arguments")
			fmt.Println()
			printCLICommandHelp(command)
			return 2
		}
		if err := cliRestoreBrain(restoreBrainOpts); err != nil {
			fmt.Printf("Error: %v\n", err)
			return 1
		}
	case "validate":
		if err := validateFlags.Parse(args); err != nil {
			if err == flag.ErrHelp {
				printCLICommandHelp(command)
				return 0
			}
			fmt.Printf("Error: %v\n\n", err)
			printCLICommandHelp(command)
			return 2
		}
		validateArgs := validateFlags.Args()
		if len(validateArgs) != 1 {
			fmt.Println("Error: validate requires a path to a robot repository")
			fmt.Println()
			printCLICommandHelp(command)
			return 2
		}
		if err := cliValidate(validateArgs[0], validateRedactedSecrets); err != nil {
			fmt.Printf("Error: %v\n", err)
			return 1
		}
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
var cliMatcherConfigLoad bool
var cliMatcherConfigInitialized bool
var cliMatcherConfigRedacted bool

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
	initCLIConfigOnlyWithSecrets(configSecretRequire, false)
}

func initCLIConfigOnlyWithSecrets(secretMode configSecretResolutionMode, allowMissingEncryption bool) bool {
	redacted, err := loadCLIConfigOnlyWithSecrets(secretMode, allowMissingEncryption)
	if err != nil {
		Log(robot.Fatal, "%v", err)
	}
	return redacted
}

func loadCLIConfigOnlyWithSecrets(secretMode configSecretResolutionMode, allowMissingEncryption bool) (bool, error) {
	if cliConfigInitialized {
		return false, nil
	}
	currentCfg.configuration = &configuration{}
	initCLIConfigDirectory()

	if secretMode != configSecretRedact {
		encryptionInitialized := initCrypt()
		if encryptionInitialized {
			setEnv("GOPHER_ENCRYPTION_INITIALIZED", "initialized")
		} else if !allowMissingEncryption {
			mode := detectStartupMode()
			switch mode {
			case "cli", "bootstrap", "production":
				return false, fmt.Errorf("unable to initialize encryption for startup mode '%s', no GOPHER_ENCRYPTION_KEY set in environment (or .env)", mode)
			default:
				cryptKey.Lock()
				cryptKey.key = make([]byte, 32)
				if _, err := crand.Read(cryptKey.key); err != nil {
					cryptKey.Unlock()
					return false, fmt.Errorf("generating temporary encryption key: %w", err)
				}
				cryptKey.initialized = true
				cryptKey.Unlock()
				Log(robot.Info, "Initialized temporary encryption key for '%s' mode", mode)
			}
		}
	}

	restoreSecretMode := beginConfigSecretResolution(secretMode)
	err := loadConfig(true)
	redacted := configSecretRedactionUsed()
	restoreSecretMode()
	if err != nil {
		return redacted, fmt.Errorf("loading initial configuration: %w", err)
	}
	cliConfigInitialized = true
	return redacted, nil
}

func initCLICommandMatcherConfig() {
	if cliMatcherConfigInitialized {
		return
	}
	currentCfg.configuration = &configuration{}
	initCLIConfigDirectory()

	encryptionInitialized := initCrypt()
	if encryptionInitialized {
		setEnv("GOPHER_ENCRYPTION_INITIALIZED", "initialized")
	}

	restoreSecretMode := beginConfigSecretResolution(configSecretPrefer)
	cliMatcherConfigLoad = true
	err := loadConfig(false)
	cliMatcherConfigLoad = false
	cliMatcherConfigRedacted = configSecretRedactionUsed()
	restoreSecretMode()
	if err != nil {
		Log(robot.Fatal, "Loading command matcher configuration: %v", err)
	}
	cliMatcherConfigInitialized = true
}

func initCLIBrainProvider() {
	initCLIConfigOnly()
	if interfaces.brain != nil {
		return
	}
	brain, providerName, err := initializeConfiguredBrain()
	if err != nil {
		Log(robot.Fatal, "Initializing brain provider '%s': %v", providerName, err)
	}
	interfaces.brain = brain
	Log(robot.Info, "Initialized brain provider '%s'", providerName)
}

func shutdownCLIBrainProvider(flush bool) {
	if interfaces.brain == nil {
		return
	}
	if flush {
		interfaces.brain.Shutdown()
	} else if brain, ok := interfaces.brain.(interface{ ShutdownWithoutFlush() }); ok {
		brain.ShutdownWithoutFlush()
	} else {
		interfaces.brain.Shutdown()
	}
	interfaces.brain = nil
}

func initCLILocalBrainProviderForRead() error {
	initCLIConfigOnly()
	if interfaces.brain != nil {
		return nil
	}
	provider := currentCfg.brainProvider
	if provider == "" {
		provider = "mem"
	}
	if provider == "mem" || provider == "file" {
		brain, _, err := initializeConfiguredBrain()
		if err != nil {
			return err
		}
		interfaces.brain = brain
		return nil
	}
	cache, err := openExistingBrainCacheAny(currentCfg.brainCache)
	if err != nil {
		return fmt.Errorf("opening local brain cache: %w; run gopherbot pull-brain or use fetch -cloud", err)
	}
	interfaces.brain = cache
	return nil
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

type cliFetchOptions struct {
	key           string
	base64        bool
	validateCloud bool
	cloud         bool
	updateCache   bool
}

func cliFetch(opts cliFetchOptions) error {
	if opts.cloud {
		return cliFetchCloud(opts)
	}
	if err := initCLILocalBrainProviderForRead(); err != nil {
		return err
	}
	defer shutdownCLIBrainProvider(false)
	if opts.validateCloud {
		cache, ok := interfaces.brain.(*cachedBrain)
		if !ok {
			return fmt.Errorf("configured brain is not cloud-backed")
		}
		remote, _, _, err := initRemoteBrainForCLI()
		if err != nil {
			return err
		}
		defer remote.Shutdown()
		if err := validateLocalMemoryAgainstCloud(cache, remote, opts.key); err != nil {
			fmt.Fprintf(os.Stderr, "Brain cache sync: %v\n", err)
			return err
		}
		fmt.Fprintf(os.Stderr, "Brain cache sync: local memory %s matches cloud\n", opts.key)
	}
	return cliFetchLocal(opts.key, opts.base64)
}

func cliFetchLocal(item string, b64 bool) error {
	_, datum, exists, ret := getDatum(item, false)
	if ret != robot.Ok {
		return fmt.Errorf("retrieving datum: %v", ret)
	}
	if !exists {
		return fmt.Errorf("item not found")
	}
	writeFetchedMemory(*datum, b64)
	return nil
}

func cliFetchCloud(opts cliFetchOptions) error {
	initCLIConfigOnly()
	remote, _, providerName, err := initRemoteBrainForCLI()
	if err != nil {
		return err
	}
	defer remote.Shutdown()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	record, exists, err := remote.Get(ctx, opts.key)
	if err != nil {
		return err
	}
	if !exists || record.Deleted {
		return fmt.Errorf("item not found in cloud brain %q", providerName)
	}
	if err := validateRemoteMemoryRecord(record); err != nil {
		return err
	}
	if opts.updateCache {
		cache, err := openExistingBrainCacheComplete(currentCfg.brainCache)
		if err != nil {
			return err
		}
		if err := cache.importV3Record(record); err != nil {
			return err
		}
	}
	reportCloudMemorySyncStatus(record)
	payload, err := decryptMemoryPayload(record.Payload)
	if err != nil {
		return err
	}
	writeFetchedMemory(payload, opts.base64)
	return nil
}

func validateLocalMemoryAgainstCloud(cache *cachedBrain, remote robot.RemoteBrainBackend, key string) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	outbox, pending, err := cache.readOutboxEntry(key)
	if err != nil {
		return err
	}
	if pending {
		return fmt.Errorf("local memory %s has pending cloud sync at version %d; run gopherbot flush-brain before validating", key, outbox.Version)
	}
	meta, exists, err := cache.readMeta(key)
	if err != nil {
		return err
	}
	if !exists || meta.Deleted {
		return fmt.Errorf("item not found in local cache")
	}
	payload, err := os.ReadFile(cache.payloadPath(key))
	if err != nil {
		return err
	}
	localChecksum := checksumBytes(payload)
	if meta.Checksum != localChecksum {
		return fmt.Errorf("local memory %s checksum mismatch: metadata=%s payload=%s", key, meta.Checksum, localChecksum)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	record, exists, err := remote.Get(ctx, key)
	if err != nil {
		return err
	}
	if !exists || record.Deleted {
		return fmt.Errorf("cloud memory %s is missing or deleted", key)
	}
	if err := validateRemoteMemoryRecord(record); err != nil {
		return err
	}
	if record.Version != meta.Version {
		return fmt.Errorf("cloud memory %s version mismatch: local=%d cloud=%d", key, meta.Version, record.Version)
	}
	if record.Checksum != localChecksum {
		return fmt.Errorf("cloud memory %s checksum mismatch: local=%s cloud=%s", key, localChecksum, record.Checksum)
	}
	return nil
}

func validateRemoteMemoryRecord(record robot.RemoteBrainRecord) error {
	if record.Format != brainCacheFormat {
		return fmt.Errorf("cloud memory %s is not a v3 brain record", record.Key)
	}
	if record.Checksum == "" {
		return fmt.Errorf("cloud memory %s is missing checksum metadata", record.Key)
	}
	payloadChecksum := checksumBytes(record.Payload)
	if record.Checksum != payloadChecksum {
		return fmt.Errorf("cloud memory %s checksum mismatch: metadata=%s payload=%s", record.Key, record.Checksum, payloadChecksum)
	}
	return nil
}

func decryptMemoryPayload(payload []byte) ([]byte, error) {
	cryptKey.RLock()
	initialized := cryptKey.initialized
	key := cryptKey.key
	cryptKey.RUnlock()
	if !initialized {
		return nil, fmt.Errorf("brain encryption is not initialized")
	}
	plain, err := decrypt(payload, key)
	if err != nil {
		return nil, fmt.Errorf("decrypting cloud memory: %w", err)
	}
	return plain, nil
}

func writeFetchedMemory(payload []byte, b64 bool) {
	if b64 {
		encoder := base64.NewEncoder(base64.StdEncoding, os.Stdout)
		encoder.Write(payload)
		encoder.Close()
		os.Stdout.Write([]byte("\n"))
		return
	}
	os.Stdout.Write(payload)
	os.Stdout.Write([]byte("\n"))
}

func reportCloudMemorySyncStatus(record robot.RemoteBrainRecord) {
	cache, err := openExistingBrainCacheComplete(currentCfg.brainCache)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Brain cache sync: local cache unavailable (%v)\n", err)
		return
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if outbox, pending, err := cache.readOutboxEntry(record.Key); err != nil {
		fmt.Fprintf(os.Stderr, "Brain cache sync: unable to inspect local outbox for %s: %v\n", record.Key, err)
	} else if pending {
		fmt.Fprintf(os.Stderr, "Brain cache sync: local memory %s has pending cloud sync at version %d\n", record.Key, outbox.Version)
		return
	}
	meta, exists, err := cache.readMeta(record.Key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Brain cache sync: unable to inspect local metadata for %s: %v\n", record.Key, err)
		return
	}
	if record.Deleted {
		if !exists || meta.Deleted {
			fmt.Fprintf(os.Stderr, "Brain cache sync: local memory %s matches cloud tombstone\n", record.Key)
			return
		}
		fmt.Fprintf(os.Stderr, "Brain cache sync: local memory %s is active but cloud has a tombstone\n", record.Key)
		return
	}
	if !exists {
		fmt.Fprintf(os.Stderr, "Brain cache sync: local cache is missing cloud memory %s\n", record.Key)
		return
	}
	if meta.Deleted {
		fmt.Fprintf(os.Stderr, "Brain cache sync: local cache has a tombstone for active cloud memory %s\n", record.Key)
		return
	}
	payload, err := os.ReadFile(cache.payloadPath(record.Key))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Brain cache sync: unable to read local payload for %s: %v\n", record.Key, err)
		return
	}
	localChecksum := checksumBytes(payload)
	if meta.Version == record.Version && localChecksum == record.Checksum {
		fmt.Fprintf(os.Stderr, "Brain cache sync: local memory %s matches cloud version %d\n", record.Key, record.Version)
		return
	}
	fmt.Fprintf(os.Stderr, "Brain cache sync: local memory %s differs from cloud (local version %d checksum %s, cloud version %d checksum %s)\n",
		record.Key, meta.Version, localChecksum, record.Version, record.Checksum)
}

func reportCloudListSyncStatus(records []robot.RemoteBrainRecord) {
	cache, err := openExistingBrainCacheComplete(currentCfg.brainCache)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Brain cache sync: local cache unavailable (%v)\n", err)
		return
	}
	localKeys, err := cache.List()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Brain cache sync: unable to list local cache: %v\n", err)
		return
	}
	pending, err := cache.outboxEntries()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Brain cache sync: unable to inspect local outbox: %v\n", err)
		return
	}
	cloudSet := make(map[string]bool, len(records))
	var missingLocal, mismatched, v2OrInvalid int
	for _, record := range records {
		cloudSet[record.Key] = true
		if record.Format != brainCacheFormat {
			v2OrInvalid++
			continue
		}
		meta, exists, err := cache.readMeta(record.Key)
		if err != nil || !exists || meta.Deleted {
			missingLocal++
			continue
		}
		payload, err := os.ReadFile(cache.payloadPath(record.Key))
		if err != nil {
			mismatched++
			continue
		}
		if meta.Version != record.Version || checksumBytes(payload) != record.Checksum {
			mismatched++
		}
	}
	var extraLocal int
	for _, key := range localKeys {
		if !cloudSet[key] {
			extraLocal++
		}
	}
	switch {
	case len(pending) == 0 && missingLocal == 0 && extraLocal == 0 && mismatched == 0 && v2OrInvalid == 0:
		fmt.Fprintf(os.Stderr, "Brain cache sync: local cache matches listed cloud memories (%d key(s))\n", len(records))
	default:
		fmt.Fprintf(os.Stderr, "Brain cache sync: %d pending local write(s), %d missing local key(s), %d extra local key(s), %d mismatched key(s), %d non-v3 cloud key(s)\n",
			len(pending), missingLocal, extraLocal, mismatched, v2OrInvalid)
	}
}

func reportLocalCloudOutboxStatus() {
	provider := currentCfg.brainProvider
	if provider == "" || provider == "mem" || provider == "file" {
		return
	}
	cache, err := openExistingBrainCacheAny(currentCfg.brainCache)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Brain cache sync: unable to inspect local cache (%v)\n", err)
		return
	}
	pending, err := cache.outboxEntries()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Brain cache sync: unable to inspect local outbox (%v)\n", err)
		return
	}
	if len(pending) == 0 {
		fmt.Fprintln(os.Stderr, "Brain cache sync: no pending local cloud writes")
		return
	}
	fmt.Fprintf(os.Stderr, "Brain cache sync: %d pending local cloud write(s)\n", len(pending))
}

func cliStore(key, file string) error {
	var fc []byte
	var err error
	if file == "-" {
		fc, err = io.ReadAll(os.Stdin)
	} else {
		fc, err = os.ReadFile(file)
	}
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}
	ret := storeDatum(key, &fc)
	if ret != robot.Ok {
		return fmt.Errorf("storing datum: %s", ret)
	}
	return nil
}

func cliList(cloud bool) error {
	if cloud {
		return cliListCloud()
	}
	if err := initCLILocalBrainProviderForRead(); err != nil {
		return err
	}
	defer shutdownCLIBrainProvider(false)
	brain := interfaces.brain
	list, err := brain.List()
	if err != nil {
		return fmt.Errorf("listing memories: %w", err)
	}
	if len(list) > 0 {
		for _, memory := range list {
			fmt.Println(memory)
		}
		return nil
	}
	fmt.Println("No memories found")
	return nil
}

func cliListCloud() error {
	initCLIConfigOnly()
	remote, _, providerName, err := initRemoteBrainForCLI()
	if err != nil {
		return err
	}
	defer remote.Shutdown()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cursor := ""
	var keys []string
	var records []robot.RemoteBrainRecord
	for {
		page, err := remote.ListMetadata(ctx, cursor, 1000)
		if err != nil {
			return fmt.Errorf("listing cloud memories from %s: %w", providerName, err)
		}
		for _, record := range page.Records {
			if record.Deleted {
				continue
			}
			records = append(records, record)
			keys = append(keys, record.Key)
		}
		if page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
	}
	reportCloudListSyncStatus(records)
	sort.Strings(keys)
	if len(keys) == 0 {
		fmt.Println("No cloud memories found")
		return nil
	}
	for _, key := range keys {
		fmt.Println(key)
	}
	return nil
}

func cliDelete(key string) error {
	brain := interfaces.brain
	err := brain.Delete(key)
	if err != nil {
		return fmt.Errorf("deleting memory: %w", err)
	}
	return nil
}

func cliValidate(path string, redactedSecrets bool) error {
	configPath = path
	testpath := filepath.Join(configPath, "conf", robotConfigFileName)
	_, err := os.Stat(testpath)
	if err != nil {
		return fmt.Errorf("robot repository not found at %q (expected %s)", path, testpath)
	}
	botLogger.logger = log.New(os.Stdout, "", 0)
	fmt.Println("Validating configuration")
	secretMode := configSecretRequire
	allowMissingEncryption := false
	if redactedSecrets {
		secretMode = configSecretRedact
		allowMissingEncryption = true
	}
	redacted, err := loadCLIConfigOnlyWithSecrets(secretMode, allowMissingEncryption)
	if err != nil {
		return err
	}
	if redacted {
		fmt.Fprintln(os.Stderr, "Info: secret template values redacted for validation")
	}
	fmt.Println("Configuration valid")
	return nil
}
