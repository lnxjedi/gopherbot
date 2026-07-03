package bot

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/chzyer/readline"
)

type cliCommandMatchReport struct {
	Input             string                       `json:"input"`
	SimpleMatcher     string                       `json:"simple_matcher,omitempty"`
	Status            string                       `json:"status"`
	Ambiguous         bool                         `json:"ambiguous"`
	RedactedSecrets   bool                         `json:"redacted_secrets,omitempty"`
	Matches           []cliCommandMatch            `json:"matches"`
	SyntaxDiagnostics []cliCommandSyntaxDiagnostic `json:"syntax_diagnostics"`
}

type cliCommandMatch struct {
	Plugin        string   `json:"plugin,omitempty"`
	Command       string   `json:"command,omitempty"`
	Args          []string `json:"args"`
	SimpleMatcher string   `json:"simple_matcher,omitempty"`
	Regex         string   `json:"regex,omitempty"`
	Usage         string   `json:"usage,omitempty"`
	Summary       string   `json:"summary,omitempty"`
}

type cliCommandSyntaxDiagnostic struct {
	Plugin        string `json:"plugin,omitempty"`
	Command       string `json:"command,omitempty"`
	Diagnostic    string `json:"diagnostic"`
	SimpleMatcher string `json:"simple_matcher,omitempty"`
	Regex         string `json:"regex,omitempty"`
	Usage         string `json:"usage,omitempty"`
	Summary       string `json:"summary,omitempty"`
}

func cliCheckSimpleMatcher(spec, input string) (cliCommandMatchReport, error) {
	report := cliCommandMatchReport{
		Input:         input,
		SimpleMatcher: spec,
		Status:        "no_match",
	}
	matcher := InputMatcher{Command: "check", SimpleMatcher: spec}
	if err := compileInputMatcher(&matcher, true); err != nil {
		return report, err
	}
	result := matcher.matchCommandInput(input)
	switch result.kind {
	case inputExactMatch:
		report.Status = "match"
		report.Matches = append(report.Matches, cliCommandMatch{
			Args:          result.args,
			SimpleMatcher: spec,
		})
	case inputSyntaxMatch:
		report.Status = "syntax"
		report.SyntaxDiagnostics = append(report.SyntaxDiagnostics, cliCommandSyntaxDiagnostic{
			Diagnostic:    result.diagnostic,
			SimpleMatcher: spec,
		})
	}
	return report, nil
}

func cliMatchConfiguredCommand(input string) cliCommandMatchReport {
	matcher := configuredCommandMatcher()
	return matcher(input)
}

func configuredCommandMatcher() func(string) cliCommandMatchReport {
	initCLICommandMatcherConfig()
	currentCfg.RLock()
	tasks := currentCfg.taskList
	currentCfg.RUnlock()
	redactedSecrets := cliMatcherConfigRedacted
	return func(input string) cliCommandMatchReport {
		report := matchConfiguredCommand(input, tasks)
		report.RedactedSecrets = redactedSecrets
		return report
	}
}

func matchConfiguredCommand(input string, tasks *taskList) cliCommandMatchReport {
	report := cliCommandMatchReport{
		Input:  input,
		Status: "no_match",
	}
	if tasks == nil {
		return report
	}
	for _, item := range tasks.t[1:] {
		task, plugin, _ := getTask(item)
		if task == nil || plugin == nil || task.Disabled {
			continue
		}
		for _, matcher := range plugin.Commands {
			result := matcher.matchCommandInput(input)
			switch result.kind {
			case inputExactMatch:
				report.Matches = append(report.Matches, cliCommandMatch{
					Plugin:        task.name,
					Command:       matcher.Command,
					Args:          result.args,
					SimpleMatcher: strings.TrimSpace(matcher.SimpleMatcher),
					Regex:         strings.TrimSpace(matcher.Regex),
					Usage:         strings.TrimSpace(matcher.Usage),
					Summary:       strings.TrimSpace(matcher.Summary),
				})
			case inputSyntaxMatch:
				report.SyntaxDiagnostics = append(report.SyntaxDiagnostics, cliCommandSyntaxDiagnostic{
					Plugin:        task.name,
					Command:       matcher.Command,
					Diagnostic:    result.diagnostic,
					SimpleMatcher: strings.TrimSpace(matcher.SimpleMatcher),
					Regex:         strings.TrimSpace(matcher.Regex),
					Usage:         strings.TrimSpace(matcher.Usage),
					Summary:       strings.TrimSpace(matcher.Summary),
				})
			}
		}
	}
	sort.Slice(report.Matches, func(i, j int) bool {
		if report.Matches[i].Plugin == report.Matches[j].Plugin {
			return report.Matches[i].Command < report.Matches[j].Command
		}
		return report.Matches[i].Plugin < report.Matches[j].Plugin
	})
	sort.Slice(report.SyntaxDiagnostics, func(i, j int) bool {
		if report.SyntaxDiagnostics[i].Plugin == report.SyntaxDiagnostics[j].Plugin {
			return report.SyntaxDiagnostics[i].Command < report.SyntaxDiagnostics[j].Command
		}
		return report.SyntaxDiagnostics[i].Plugin < report.SyntaxDiagnostics[j].Plugin
	})
	if len(report.Matches) > 0 {
		report.Status = "match"
		report.Ambiguous = len(report.Matches) > 1
		return report
	}
	if len(report.SyntaxDiagnostics) > 0 {
		report.Status = "syntax"
		report.Ambiguous = len(report.SyntaxDiagnostics) > 1
	}
	return report
}

func printCLICommandMatchReport(report cliCommandMatchReport, jsonOutput bool) error {
	return printCLICommandMatchReportWithOptions(report, jsonOutput, false)
}

func printCLICommandMatchReportWithOptions(report cliCommandMatchReport, jsonOutput, suppressRedactedNotice bool) error {
	if report.RedactedSecrets && !suppressRedactedNotice {
		fmt.Fprintln(os.Stderr, "Info: encryption unavailable; using redacted secret placeholders for command metadata")
	}
	if jsonOutput {
		var out bytes.Buffer
		encoder := json.NewEncoder(&out)
		encoder.SetEscapeHTML(false)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(report); err != nil {
			return err
		}
		fmt.Print(out.String())
		return nil
	}
	switch {
	case len(report.Matches) > 0:
		if len(report.Matches) > 1 {
			fmt.Println("MULTIPLE MATCHES")
		}
		for _, match := range report.Matches {
			if match.Plugin == "" && match.Command == "" {
				fmt.Printf("MATCH %s\n", formatCLIArgsList(match.Args))
				continue
			}
			fmt.Printf("MATCH %s/%s %s\n", match.Plugin, match.Command, formatCLIArgsList(match.Args))
		}
	case len(report.SyntaxDiagnostics) > 0:
		if len(report.SyntaxDiagnostics) > 1 {
			fmt.Println("MULTIPLE SYNTAX DIAGNOSTICS")
		}
		for _, diag := range report.SyntaxDiagnostics {
			if diag.Plugin == "" && diag.Command == "" {
				fmt.Printf("SYNTAX %s\n", diag.Diagnostic)
				continue
			}
			fmt.Printf("SYNTAX %s/%s: %s\n", diag.Plugin, diag.Command, diag.Diagnostic)
		}
	default:
		fmt.Println("NO MATCH")
	}
	return nil
}

func cliCommandMatchExitCode(report cliCommandMatchReport) int {
	if len(report.Matches) > 0 {
		return 0
	}
	return 1
}

func formatCLIArgsList(args []string) string {
	if len(args) == 0 {
		return "[]"
	}
	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		quoted = append(quoted, strconv.Quote(arg))
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

func processCLICheckCommand(args []string, jsonOutput bool) int {
	if len(args) < 2 {
		fmt.Println("Error: check requires a SimpleMatcher and command text")
		fmt.Println()
		printCLICommandHelp("check")
		return 2
	}
	report, err := cliCheckSimpleMatcher(args[0], strings.Join(args[1:], " "))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return 2
	}
	if err := printCLICommandMatchReport(report, jsonOutput); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	return cliCommandMatchExitCode(report)
}

func processCLIMatchCommand(args []string, jsonOutput, interactive bool) int {
	if interactive {
		if len(args) > 0 {
			fmt.Println("Error: match -interactive does not accept command text arguments")
			fmt.Println()
			printCLICommandHelp("match")
			return 2
		}
		prompt := io.Writer(os.Stdout)
		if jsonOutput {
			prompt = os.Stderr
		}
		matcher := configuredCommandMatcher()
		if readline.DefaultIsTerminal() {
			return processCLIMatchInteractiveReadline(prompt, jsonOutput, matcher)
		}
		return processCLIMatchInteractive(os.Stdin, prompt, jsonOutput, matcher)
	}
	if len(args) == 0 {
		fmt.Println("Error: match requires command text")
		fmt.Println()
		printCLICommandHelp("match")
		return 2
	}
	report := cliMatchConfiguredCommand(strings.Join(args, " "))
	if err := printCLICommandMatchReport(report, jsonOutput); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	return cliCommandMatchExitCode(report)
}

func processCLIMatchInteractive(input io.Reader, prompt io.Writer, jsonOutput bool, match func(string) cliCommandMatchReport) int {
	scanner := bufio.NewScanner(input)
	redactedNoticeShown := false
	for {
		fmt.Fprint(prompt, "Command?: ")
		if !scanner.Scan() {
			break
		}
		if err := handleCLIMatchInteractiveCommand(scanner.Text(), jsonOutput, &redactedNoticeShown, match); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading command text: %v\n", err)
		return 1
	}
	return 0
}

func processCLIMatchInteractiveReadline(prompt io.Writer, jsonOutput bool, match func(string) cliCommandMatchReport) int {
	rl, err := readline.NewEx(&readline.Config{
		Prompt:            "Command?: ",
		HistorySearchFold: true,
		EOFPrompt:         "\n",
		Stdin:             os.Stdin,
		Stdout:            prompt,
		Stderr:            os.Stderr,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating command prompt: %v\n", err)
		return 1
	}
	defer rl.Close()

	redactedNoticeShown := false
	for {
		command, err := rl.Readline()
		if err == io.EOF {
			break
		}
		if err == readline.ErrInterrupt {
			continue
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading command text: %v\n", err)
			return 1
		}
		if err := handleCLIMatchInteractiveCommand(command, jsonOutput, &redactedNoticeShown, match); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
	}
	return 0
}

func handleCLIMatchInteractiveCommand(command string, jsonOutput bool, redactedNoticeShown *bool, match func(string) cliCommandMatchReport) error {
	command = strings.TrimSpace(command)
	if command == "" {
		return nil
	}
	report := match(command)
	if report.RedactedSecrets && !*redactedNoticeShown {
		fmt.Fprintln(os.Stderr, "Info: encryption unavailable; using redacted secret placeholders for command metadata")
		*redactedNoticeShown = true
	}
	return printCLICommandMatchReportWithOptions(report, jsonOutput, true)
}
