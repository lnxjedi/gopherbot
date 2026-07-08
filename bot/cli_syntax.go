package bot

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/parser"
	"go/scanner"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/dop251/goja"
	gshmod "github.com/lnxjedi/gopherbot/v2/modules/gsh"
	luaparse "github.com/yuin/gopher-lua/parse"
	"mvdan.cc/sh/v3/syntax"
)

type cliScriptSyntaxReport struct {
	Status string                      `json:"status"`
	Files  []cliScriptSyntaxFileReport `json:"files"`
}

type cliScriptSyntaxFileReport struct {
	Path        string                `json:"path"`
	Language    string                `json:"language,omitempty"`
	OK          bool                  `json:"ok"`
	Diagnostics []cliScriptDiagnostic `json:"diagnostics,omitempty"`
}

type cliScriptDiagnostic struct {
	Severity string `json:"severity"`
	Stage    string `json:"stage"`
	Message  string `json:"message"`
	Line     int    `json:"line,omitempty"`
	Column   int    `json:"column,omitempty"`
}

func processCLISyntaxCommand(paths []string, langOverride string, jsonOutput bool) int {
	if len(paths) == 0 {
		fmt.Println("Error: syntax requires at least one script path")
		fmt.Println()
		printCLICommandHelp("syntax")
		return 2
	}
	report := cliScriptSyntaxReport{Status: "ok"}
	for _, path := range paths {
		fileReport := cliCheckScriptSyntax(path, langOverride)
		if !fileReport.OK {
			report.Status = "error"
		}
		report.Files = append(report.Files, fileReport)
	}
	if err := printCLISyntaxReport(report, jsonOutput); err != nil {
		fmt.Printf("Error: writing syntax report: %v\n", err)
		return 1
	}
	if report.Status == "ok" {
		return 0
	}
	return 1
}

func cliCheckScriptSyntax(path, langOverride string) cliScriptSyntaxFileReport {
	language, langErr := normalizeCLIScriptLanguage(path, langOverride)
	report := cliScriptSyntaxFileReport{
		Path:     path,
		Language: language,
	}
	if langErr != nil {
		report.Diagnostics = append(report.Diagnostics, cliScriptDiagnostic{
			Severity: "error",
			Stage:    "language",
			Message:  langErr.Error(),
		})
		return report
	}

	diags := checkScriptSyntaxByLanguage(path, language)
	if len(diags) == 0 {
		report.OK = true
		return report
	}
	report.Diagnostics = diags
	return report
}

func normalizeCLIScriptLanguage(path, override string) (string, error) {
	language := strings.TrimSpace(strings.ToLower(override))
	switch language {
	case "":
		switch strings.ToLower(filepath.Ext(path)) {
		case ".lua":
			return "lua", nil
		case ".js", ".mjs", ".cjs":
			return "js", nil
		case ".gsh":
			return "gsh", nil
		case ".go":
			return "go", nil
		default:
			return "", fmt.Errorf("could not infer language for %q; use -language lua|js|gsh|go", path)
		}
	case "javascript", "node":
		return "js", nil
	case "shell", "sh", "mvdan":
		return "gsh", nil
	case "golang", "yaegi":
		return "go", nil
	case "lua", "js", "gsh", "go":
		return language, nil
	default:
		return "", fmt.Errorf("unsupported language %q; use lua, js, gsh, or go", override)
	}
}

func checkScriptSyntaxByLanguage(path, language string) []cliScriptDiagnostic {
	switch language {
	case "lua":
		return checkLuaScriptSyntax(path)
	case "js":
		return checkJSScriptSyntax(path)
	case "gsh":
		return checkGSHScriptSyntax(path)
	case "go":
		return checkGoScriptSyntax(path)
	default:
		return []cliScriptDiagnostic{{
			Severity: "error",
			Stage:    "language",
			Message:  fmt.Sprintf("unsupported language %q", language),
		}}
	}
}

func checkLuaScriptSyntax(path string) []cliScriptDiagnostic {
	source, err := os.ReadFile(path)
	if err != nil {
		return []cliScriptDiagnostic{readScriptDiagnostic(err)}
	}
	if _, err := luaparse.Parse(bytes.NewReader(source), path); err != nil {
		return []cliScriptDiagnostic{diagnosticFromError("parse", err, 0)}
	}
	return nil
}

func checkJSScriptSyntax(path string) []cliScriptDiagnostic {
	source, err := os.ReadFile(path)
	if err != nil {
		return []cliScriptDiagnostic{readScriptDiagnostic(err)}
	}
	if _, err := goja.Compile(path, string(source), true); err != nil {
		return []cliScriptDiagnostic{diagnosticFromError("compile", err, 0)}
	}
	return nil
}

func checkGSHScriptSyntax(path string) []cliScriptDiagnostic {
	if err := gshmod.CheckSyntax(path); err != nil {
		return []cliScriptDiagnostic{diagnosticFromError("parse", err, gshmod.SyntaxLineOffset())}
	}
	return nil
}

func checkGoScriptSyntax(path string) []cliScriptDiagnostic {
	fset := token.NewFileSet()
	if _, err := parser.ParseFile(fset, path, nil, parser.AllErrors); err != nil {
		var list scanner.ErrorList
		if errors.As(err, &list) && len(list) > 0 {
			diags := make([]cliScriptDiagnostic, 0, len(list))
			for _, item := range list {
				diags = append(diags, cliScriptDiagnostic{
					Severity: "error",
					Stage:    "parse",
					Message:  item.Msg,
					Line:     item.Pos.Line,
					Column:   item.Pos.Column,
				})
			}
			return diags
		}
		return []cliScriptDiagnostic{diagnosticFromError("parse", err, 0)}
	}
	return nil
}

func readScriptDiagnostic(err error) cliScriptDiagnostic {
	return cliScriptDiagnostic{
		Severity: "error",
		Stage:    "read",
		Message:  err.Error(),
	}
}

func diagnosticFromError(stage string, err error, lineOffset int) cliScriptDiagnostic {
	diag := cliScriptDiagnostic{
		Severity: "error",
		Stage:    stage,
		Message:  err.Error(),
	}
	var parseErr syntax.ParseError
	if errors.As(err, &parseErr) {
		diag.Line = int(parseErr.Pos.Line())
		diag.Column = int(parseErr.Pos.Col())
		adjustDiagnosticLine(&diag, lineOffset)
		return diag
	}
	var langErr syntax.LangError
	if errors.As(err, &langErr) {
		diag.Line = int(langErr.Pos.Line())
		diag.Column = int(langErr.Pos.Col())
		adjustDiagnosticLine(&diag, lineOffset)
		return diag
	}
	if line, col, ok := parseLineColumn(err.Error()); ok {
		diag.Line = line
		diag.Column = col
		adjustDiagnosticLine(&diag, lineOffset)
	}
	return diag
}

func adjustDiagnosticLine(diag *cliScriptDiagnostic, lineOffset int) {
	if diag.Line > lineOffset {
		diag.Line -= lineOffset
	}
}

var cliLineColumnPattern = regexp.MustCompile(`:(\d+):(\d+):`)

func parseLineColumn(message string) (int, int, bool) {
	matches := cliLineColumnPattern.FindStringSubmatch(message)
	if len(matches) != 3 {
		return 0, 0, false
	}
	var line, column int
	if _, err := fmt.Sscanf(matches[1], "%d", &line); err != nil {
		return 0, 0, false
	}
	if _, err := fmt.Sscanf(matches[2], "%d", &column); err != nil {
		return 0, 0, false
	}
	return line, column, true
}

func printCLISyntaxReport(report cliScriptSyntaxReport, jsonOutput bool) error {
	if jsonOutput {
		var out bytes.Buffer
		encoder := json.NewEncoder(&out)
		encoder.SetEscapeHTML(false)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(report); err != nil {
			return err
		}
		_, err := fmt.Print(out.String())
		return err
	}
	for _, file := range report.Files {
		if file.OK {
			fmt.Printf("OK %s %s\n", file.Language, file.Path)
			continue
		}
		if len(file.Diagnostics) == 0 {
			fmt.Printf("ERROR %s %s\n", file.Language, file.Path)
			continue
		}
		for _, diag := range file.Diagnostics {
			location := ""
			if diag.Line > 0 {
				location = fmt.Sprintf(":%d", diag.Line)
				if diag.Column > 0 {
					location += fmt.Sprintf(":%d", diag.Column)
				}
			}
			fmt.Printf("ERROR %s %s%s: %s\n", file.Language, file.Path, location, diag.Message)
		}
	}
	return nil
}
