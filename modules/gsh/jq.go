package gsh

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/itchyny/gojq"
	"gopkg.in/yaml.v3"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
)

const (
	jqExitOK = iota
	jqExitFalsy
	jqExitFlagParse
	jqExitCompile
	jqExitNoValue
	jqExitError
)

// The GSH jq builtin is a local adapter modeled on github.com/itchyny/gojq/cli
// v0.12.17. Keep this file in sync when the gojq dependency is upgraded, while
// preserving GSH-local stdin/stdout/stderr, working-directory, and environment
// semantics.
type jqOptions struct {
	outputRaw     bool
	outputRaw0    bool
	outputJoin    bool
	outputCompact bool
	outputIndent  *int
	outputTab     bool
	outputYAML    bool
	inputNull     bool
	inputRaw      bool
	inputStream   bool
	inputYAML     bool
	inputSlurp    bool
	fromFile      bool
	modulePaths   []string
	exitStatus    bool
	help          bool
	version       bool

	argNames   []string
	argValues  []any
	args       []any
	jsonArgs   []any
	rawFiles   map[string]string
	slurpFiles map[string]any
}

type jqInputValue struct {
	name  string
	value any
}

type jqSliceIter struct {
	values []jqInputValue
	idx    int
	name   string
}

func (i *jqSliceIter) Next() (any, bool) {
	if i.idx >= len(i.values) {
		i.name = ""
		return nil, false
	}
	item := i.values[i.idx]
	i.idx++
	i.name = item.name
	return item.value, true
}

func (i *jqSliceIter) Name() string {
	return i.name
}

func (c *shellContext) runJq(ctx context.Context, args []string) error {
	hc := interp.HandlerCtx(ctx)
	opts, rest, err := parseJqArgs(hc, args)
	if err != nil {
		fmt.Fprintf(hc.Stderr, "jq: %v\n", err)
		return interp.ExitStatus(jqExitFlagParse)
	}
	if opts.help {
		fmt.Fprint(hc.Stdout, jqHelpText())
		return nil
	}
	if opts.version {
		fmt.Fprintln(hc.Stdout, "gojq 0.12.17 (gopherbot gsh builtin)")
		return nil
	}

	queryText := "."
	queryName := "<arg>"
	if opts.fromFile {
		if len(rest) == 0 {
			fmt.Fprintln(hc.Stderr, "jq: expected a query file for -f")
			return interp.ExitStatus(jqExitFlagParse)
		}
		path := resolvePath(hc.Dir, rest[0])
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(hc.Stderr, "jq: %v\n", err)
			return interp.ExitStatus(jqExitError)
		}
		queryText = string(data)
		queryName = rest[0]
		rest = rest[1:]
	} else if len(rest) > 0 {
		queryText = strings.TrimSpace(rest[0])
		rest = rest[1:]
	}

	query, err := gojq.Parse(queryText)
	if err != nil {
		fmt.Fprintf(hc.Stderr, "jq: compile error in %s: %v\n", queryName, err)
		return interp.ExitStatus(jqExitCompile)
	}

	if err := c.finishJqVariables(hc, opts); err != nil {
		fmt.Fprintf(hc.Stderr, "jq: %v\n", err)
		return interp.ExitStatus(jqExitError)
	}

	inputValues, err := c.jqInputValues(hc, rest, opts)
	if err != nil {
		fmt.Fprintf(hc.Stderr, "jq: %v\n", err)
		return interp.ExitStatus(jqExitError)
	}
	inputIter := &jqSliceIter{values: inputValues}

	modulePaths := opts.modulePaths
	if len(modulePaths) == 0 {
		modulePaths = []string{"~/.jq", "$ORIGIN/../lib/gojq", "$ORIGIN/../lib"}
	}
	code, err := gojq.Compile(query,
		gojq.WithModuleLoader(gojq.NewModuleLoader(modulePaths)),
		gojq.WithEnvironLoader(func() []string { return jqEnvironment(hc.Env) }),
		gojq.WithVariables(opts.argNames),
		gojq.WithInputIter(inputIter),
		gojq.WithFunction("debug", 0, 0, jqDebugFunc(hc.Stderr)),
		gojq.WithFunction("stderr", 0, 0, jqStderrFunc(hc.Stderr)),
		gojq.WithFunction("input_filename", 0, 0, func(any, []any) any {
			if name := inputIter.Name(); name != "" && (len(rest) > 0 || !opts.inputNull) {
				return name
			}
			return nil
		}),
	)
	if err != nil {
		fmt.Fprintf(hc.Stderr, "jq: compile error: %v\n", err)
		return interp.ExitStatus(jqExitCompile)
	}

	mainValues := inputValues
	if opts.inputNull {
		mainValues = []jqInputValue{{value: nil}}
	}
	mainIter := inputIter
	if opts.inputNull {
		mainIter = &jqSliceIter{values: mainValues}
	}

	seenOutput := false
	var lastOutput any
	runtimeStatus := 0
	for {
		value, ok := mainIter.Next()
		if !ok {
			break
		}
		iter := code.RunWithContext(ctx, value, opts.argValues...)
		for {
			out, ok := iter.Next()
			if !ok {
				break
			}
			if err, ok := out.(error); ok {
				status := jqHandleRuntimeError(hc.Stderr, err)
				if status > runtimeStatus {
					runtimeStatus = status
				}
				continue
			}
			if err := jqPrintValue(hc.Stdout, out, opts); err != nil {
				fmt.Fprintf(hc.Stderr, "jq: %v\n", err)
				return interp.ExitStatus(jqExitError)
			}
			seenOutput = true
			lastOutput = out
		}
	}
	if runtimeStatus != 0 {
		return interp.ExitStatus(runtimeStatus)
	}
	if opts.exitStatus {
		if !seenOutput {
			return interp.ExitStatus(jqExitNoValue)
		}
		if lastOutput == nil || lastOutput == false {
			return interp.ExitStatus(jqExitFalsy)
		}
	}
	return nil
}

func parseJqArgs(hc interp.HandlerContext, args []string) (*jqOptions, []string, error) {
	opts := &jqOptions{
		rawFiles:   map[string]string{},
		slurpFiles: map[string]any{},
	}
	rest := []string{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			rest = append(rest, args[i+1:]...)
			break
		}
		if arg == "" || arg == "-" || !strings.HasPrefix(arg, "-") {
			rest = append(rest, arg)
			continue
		}
		if strings.HasPrefix(arg, "--") {
			name, val, hasVal := strings.Cut(arg[2:], "=")
			take := func(flag string) (string, error) {
				if hasVal {
					hasVal = false
					return val, nil
				}
				i++
				if i >= len(args) {
					return "", fmt.Errorf("expected argument for flag `%s'", flag)
				}
				return args[i], nil
			}
			switch name {
			case "raw-output":
				opts.outputRaw = true
			case "raw-output0":
				opts.outputRaw = true
				opts.outputRaw0 = true
			case "join-output":
				opts.outputRaw = true
				opts.outputJoin = true
			case "compact-output":
				opts.outputCompact = true
			case "null-input":
				opts.inputNull = true
			case "raw-input":
				opts.inputRaw = true
			case "stream":
				opts.inputStream = true
			case "yaml-input":
				opts.inputYAML = true
			case "slurp":
				opts.inputSlurp = true
			case "yaml-output":
				opts.outputYAML = true
			case "tab":
				opts.outputTab = true
			case "indent":
				value, err := take("--indent")
				if err != nil {
					return nil, nil, err
				}
				n, err := strconv.Atoi(value)
				if err != nil || n < 0 || n > 9 {
					return nil, nil, fmt.Errorf("invalid argument for flag `--indent': %s", value)
				}
				opts.outputIndent = &n
			case "from-file":
				opts.fromFile = true
			case "library-path":
				value, err := take("--library-path")
				if err != nil {
					return nil, nil, err
				}
				opts.modulePaths = append(opts.modulePaths, resolvePath(hc.Dir, value))
			case "arg", "argjson", "slurpfile", "rawfile":
				key, err := take("--" + name)
				if err != nil {
					return nil, nil, err
				}
				value, err := take("--" + name)
				if err != nil {
					return nil, nil, err
				}
				if err := addJqBinding(hc, opts, name, key, value); err != nil {
					return nil, nil, err
				}
			case "args":
				for _, value := range args[i+1:] {
					opts.args = append(opts.args, value)
				}
				return opts, rest, nil
			case "jsonargs":
				for _, value := range args[i+1:] {
					parsed, err := jqParseJSONValue(strings.NewReader(value), "--jsonargs")
					if err != nil {
						return nil, nil, err
					}
					opts.jsonArgs = append(opts.jsonArgs, parsed)
				}
				return opts, rest, nil
			case "exit-status":
				opts.exitStatus = true
			case "version":
				opts.version = true
			case "help":
				opts.help = true
			case "color-output", "monochrome-output":
				// GSH command output is log/chat oriented; color flags are accepted
				// for CLI compatibility but intentionally ignored.
			default:
				return nil, nil, fmt.Errorf("unknown flag `--%s'", name)
			}
			if hasVal {
				return nil, nil, fmt.Errorf("boolean flag `--%s' cannot have an argument", name)
			}
			continue
		}

		shorts := arg[1:]
		for len(shorts) > 0 {
			ch := shorts[0]
			shorts = shorts[1:]
			takeShort := func(flag string) (string, error) {
				if len(shorts) > 0 {
					value := shorts
					if strings.HasPrefix(value, "=") {
						value = value[1:]
					}
					shorts = ""
					return value, nil
				}
				i++
				if i >= len(args) {
					return "", fmt.Errorf("expected argument for flag `%s'", flag)
				}
				return args[i], nil
			}
			switch ch {
			case 'r':
				opts.outputRaw = true
			case 'j':
				opts.outputRaw = true
				opts.outputJoin = true
			case 'c':
				opts.outputCompact = true
			case 'n':
				opts.inputNull = true
			case 'R':
				opts.inputRaw = true
			case 's':
				opts.inputSlurp = true
			case 'e':
				opts.exitStatus = true
			case 'f':
				opts.fromFile = true
			case 'L':
				value, err := takeShort("-L")
				if err != nil {
					return nil, nil, err
				}
				opts.modulePaths = append(opts.modulePaths, resolvePath(hc.Dir, value))
			case 'C', 'M':
				// Accepted for compatibility; GSH does not emit colorized jq.
			case 'v':
				opts.version = true
			case 'h':
				opts.help = true
			default:
				return nil, nil, fmt.Errorf("unknown flag `-%c'", ch)
			}
		}
	}
	return opts, rest, nil
}

func addJqBinding(hc interp.HandlerContext, opts *jqOptions, kind, key, value string) error {
	if key == "" {
		return fmt.Errorf("--%s requires a non-empty name", kind)
	}
	switch kind {
	case "arg":
		opts.argNames = append(opts.argNames, "$"+key)
		opts.argValues = append(opts.argValues, value)
	case "argjson":
		parsed, err := jqParseJSONValue(strings.NewReader(value), "$"+key)
		if err != nil {
			return err
		}
		opts.argNames = append(opts.argNames, "$"+key)
		opts.argValues = append(opts.argValues, parsed)
	case "slurpfile":
		parsed, err := jqReadJSONFile(resolvePath(hc.Dir, value), true)
		if err != nil {
			return err
		}
		opts.slurpFiles[key] = parsed
	case "rawfile":
		data, err := os.ReadFile(resolvePath(hc.Dir, value))
		if err != nil {
			return err
		}
		opts.rawFiles[key] = string(data)
	}
	return nil
}

func (c *shellContext) finishJqVariables(hc interp.HandlerContext, opts *jqOptions) error {
	for key, value := range opts.slurpFiles {
		opts.argNames = append(opts.argNames, "$"+key)
		opts.argValues = append(opts.argValues, value)
	}
	for key, value := range opts.rawFiles {
		opts.argNames = append(opts.argNames, "$"+key)
		opts.argValues = append(opts.argValues, value)
	}
	positional := append([]any{}, opts.args...)
	positional = append(positional, opts.jsonArgs...)
	named := make(map[string]any)
	for i, name := range opts.argNames {
		if strings.HasPrefix(name, "$") {
			named[name[1:]] = opts.argValues[i]
		}
	}
	opts.argNames = append(opts.argNames, "$ARGS")
	opts.argValues = append(opts.argValues, map[string]any{
		"named":      named,
		"positional": positional,
	})
	return nil
}

func (c *shellContext) jqInputValues(hc interp.HandlerContext, files []string, opts *jqOptions) ([]jqInputValue, error) {
	if opts.inputNull && len(files) == 0 {
		// Keep stdin available to input/inputs under -n.
		files = []string{"-"}
	}
	readers, err := inputReaders(hc, files)
	if err != nil {
		return nil, err
	}
	defer closeReaders(readers)
	values := []jqInputValue{}
	for _, reader := range readers {
		name := reader.name
		if name == "" {
			name = "<stdin>"
		}
		items, err := jqReadValues(reader.reader, name, opts)
		if err != nil {
			return nil, err
		}
		values = append(values, items...)
	}
	if len(values) == 0 && !opts.inputSlurp && !opts.inputRaw && !opts.inputStream && !opts.inputYAML {
		return []jqInputValue{}, nil
	}
	return values, nil
}

func jqReadValues(r io.Reader, name string, opts *jqOptions) ([]jqInputValue, error) {
	switch {
	case opts.inputRaw && opts.inputSlurp:
		data, err := io.ReadAll(r)
		if err != nil {
			return nil, err
		}
		return []jqInputValue{{name: name, value: string(data)}}, nil
	case opts.inputRaw:
		scanner := bufio.NewScanner(r)
		values := []jqInputValue{}
		for scanner.Scan() {
			values = append(values, jqInputValue{name: name, value: scanner.Text()})
		}
		return values, scanner.Err()
	case opts.inputStream:
		dec := json.NewDecoder(r)
		dec.UseNumber()
		stream := newGSHJqJSONStream(dec)
		values := []jqInputValue{}
		for {
			value, err := stream.next()
			if err != nil {
				if err == io.EOF {
					break
				}
				return nil, fmt.Errorf("invalid json: %s: %w", name, err)
			}
			values = append(values, jqInputValue{name: name, value: value})
		}
		return values, nil
	case opts.inputYAML:
		dec := yaml.NewDecoder(r)
		values := []jqInputValue{}
		for {
			var value any
			if err := dec.Decode(&value); err != nil {
				if err == io.EOF {
					break
				}
				return nil, fmt.Errorf("invalid yaml: %s: %w", name, err)
			}
			values = append(values, jqInputValue{name: name, value: normalizeGSHJqYAML(value)})
		}
		if opts.inputSlurp {
			return []jqInputValue{{name: name, value: jqOnlyValues(values)}}, nil
		}
		return values, nil
	default:
		values, err := jqReadJSONValues(r, name)
		if err != nil {
			return nil, err
		}
		if opts.inputSlurp {
			return []jqInputValue{{name: name, value: jqOnlyValues(values)}}, nil
		}
		return values, nil
	}
}

func jqReadJSONValues(r io.Reader, name string) ([]jqInputValue, error) {
	dec := json.NewDecoder(r)
	dec.UseNumber()
	values := []jqInputValue{}
	for {
		var value any
		if err := dec.Decode(&value); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("invalid json: %s: %w", name, err)
		}
		values = append(values, jqInputValue{name: name, value: value})
	}
	return values, nil
}

func jqReadJSONFile(path string, slurp bool) (any, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	values, err := jqReadJSONValues(f, path)
	if err != nil {
		return nil, err
	}
	if slurp {
		return jqOnlyValues(values), nil
	}
	if len(values) == 0 {
		return nil, nil
	}
	return values[0].value, nil
}

func jqParseJSONValue(r io.Reader, name string) (any, error) {
	values, err := jqReadJSONValues(r, name)
	if err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("invalid json: %s: empty input", name)
	}
	return values[0].value, nil
}

func jqOnlyValues(values []jqInputValue) []any {
	out := make([]any, 0, len(values))
	for _, value := range values {
		out = append(out, value.value)
	}
	return out
}

func jqPrintValue(w io.Writer, value any, opts *jqOptions) error {
	if opts.outputYAML {
		enc := yaml.NewEncoder(w)
		if opts.outputIndent != nil {
			enc.SetIndent(*opts.outputIndent)
		}
		if err := enc.Encode(value); err != nil {
			enc.Close()
			return err
		}
		return enc.Close()
	}
	if opts.outputRaw {
		if s, ok := value.(string); ok {
			if opts.outputRaw0 && strings.ContainsRune(s, '\x00') {
				return fmt.Errorf("cannot output a string containing NUL character")
			}
			if _, err := io.WriteString(w, s); err != nil {
				return err
			}
			return jqWriteDelimiter(w, opts)
		}
	}
	data, err := jqMarshalJSON(value, opts)
	if err != nil {
		return err
	}
	if _, err := w.Write(data); err != nil {
		return err
	}
	return jqWriteDelimiter(w, opts)
}

func jqMarshalJSON(value any, opts *jqOptions) ([]byte, error) {
	if opts.outputCompact {
		return gojq.Marshal(value)
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	switch {
	case opts.outputTab:
		enc.SetIndent("", "\t")
	case opts.outputIndent != nil:
		enc.SetIndent("", strings.Repeat(" ", *opts.outputIndent))
	default:
		enc.SetIndent("", "  ")
	}
	if err := enc.Encode(value); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(buf.Bytes(), []byte{'\n'}), nil
}

func jqWriteDelimiter(w io.Writer, opts *jqOptions) error {
	switch {
	case opts.outputRaw0:
		_, err := w.Write([]byte{0})
		return err
	case opts.outputJoin:
		return nil
	default:
		_, err := io.WriteString(w, "\n")
		return err
	}
}

func jqHandleRuntimeError(w io.Writer, err error) int {
	var halt *gojq.HaltError
	if errors.As(err, &halt) {
		if value := halt.Value(); value != nil {
			if s, ok := value.(string); ok {
				fmt.Fprint(w, s)
			} else if data, marshalErr := gojq.Marshal(value); marshalErr == nil {
				w.Write(data)
				w.Write([]byte{'\n'})
			}
		}
		return jqExitError
	}
	fmt.Fprintf(w, "jq: %v\n", err)
	return jqExitError
}

func jqEnvironment(env expand.Environ) []string {
	values := []string{}
	env.Each(func(name string, vr expand.Variable) bool {
		if vr.IsSet() && vr.Exported {
			values = append(values, name+"="+vr.String())
		}
		return true
	})
	return values
}

func jqDebugFunc(w io.Writer) func(any, []any) any {
	return func(value any, _ []any) any {
		if data, err := gojq.Marshal([]any{"DEBUG:", value}); err == nil {
			w.Write(data)
			w.Write([]byte{'\n'})
		}
		return value
	}
}

func jqStderrFunc(w io.Writer) func(any, []any) any {
	return func(value any, _ []any) any {
		if s, ok := value.(string); ok {
			w.Write([]byte(s))
			return value
		}
		if data, err := gojq.Marshal(value); err == nil {
			w.Write(data)
		}
		return value
	}
}

func jqHelpText() string {
	return `gojq - Go implementation of jq (Gopherbot shell builtin)

Usage:
  jq [OPTIONS] [FILTER] [FILES...]

The GSH builtin is modeled on gojq CLI v0.12.17. It runs inside the GSH
interpreter, so stdin/stdout/stderr, file paths, and environment variables come
from the active GSH command context.
`
}

func normalizeGSHJqYAML(value any) any {
	switch value := value.(type) {
	case map[any]any:
		out := make(map[string]any, len(value))
		for k, v := range value {
			out[fmt.Sprint(k)] = normalizeGSHJqYAML(v)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(value))
		for k, v := range value {
			out[k] = normalizeGSHJqYAML(v)
		}
		return out
	case []any:
		for i, item := range value {
			value[i] = normalizeGSHJqYAML(item)
		}
		return value
	case time.Time:
		return value.Format(time.RFC3339)
	default:
		return value
	}
}

type gshJqJSONStream struct {
	dec    *json.Decoder
	path   []any
	states []int
}

const (
	gshJqJSONStateTopValue = iota
	gshJqJSONStateArrayStart
	gshJqJSONStateArrayValue
	gshJqJSONStateArrayEnd
	gshJqJSONStateArrayEmptyEnd
	gshJqJSONStateObjectStart
	gshJqJSONStateObjectKey
	gshJqJSONStateObjectValue
	gshJqJSONStateObjectEnd
	gshJqJSONStateObjectEmptyEnd
)

func newGSHJqJSONStream(dec *json.Decoder) *gshJqJSONStream {
	return &gshJqJSONStream{dec: dec, states: []int{gshJqJSONStateTopValue}, path: []any{}}
}

func (s *gshJqJSONStream) next() (any, error) {
	switch s.states[len(s.states)-1] {
	case gshJqJSONStateArrayEnd, gshJqJSONStateObjectEnd:
		s.path = s.path[:len(s.path)-1]
		fallthrough
	case gshJqJSONStateArrayEmptyEnd, gshJqJSONStateObjectEmptyEnd:
		s.states = s.states[:len(s.states)-1]
	}
	if s.dec.More() {
		switch s.states[len(s.states)-1] {
		case gshJqJSONStateArrayValue:
			s.path[len(s.path)-1] = s.path[len(s.path)-1].(int) + 1
		case gshJqJSONStateObjectValue:
			s.path = s.path[:len(s.path)-1]
		}
	}
	for {
		token, err := s.dec.Token()
		if err != nil {
			if err == io.EOF && s.states[len(s.states)-1] != gshJqJSONStateTopValue {
				err = io.ErrUnexpectedEOF
			}
			return nil, err
		}
		if delim, ok := token.(json.Delim); ok {
			switch delim {
			case '[', '{':
				switch s.states[len(s.states)-1] {
				case gshJqJSONStateArrayStart:
					s.states[len(s.states)-1] = gshJqJSONStateArrayValue
				case gshJqJSONStateObjectKey:
					s.states[len(s.states)-1] = gshJqJSONStateObjectValue
				}
				if delim == '[' {
					s.states = append(s.states, gshJqJSONStateArrayStart)
					s.path = append(s.path, 0)
				} else {
					s.states = append(s.states, gshJqJSONStateObjectStart)
				}
			case ']':
				if s.states[len(s.states)-1] == gshJqJSONStateArrayStart {
					s.states[len(s.states)-1] = gshJqJSONStateArrayEmptyEnd
					s.path = s.path[:len(s.path)-1]
					return []any{s.copyPath(), []any{}}, nil
				}
				s.states[len(s.states)-1] = gshJqJSONStateArrayEnd
				return []any{s.copyPath()}, nil
			case '}':
				if s.states[len(s.states)-1] == gshJqJSONStateObjectStart {
					s.states[len(s.states)-1] = gshJqJSONStateObjectEmptyEnd
					return []any{s.copyPath(), map[string]any{}}, nil
				}
				s.states[len(s.states)-1] = gshJqJSONStateObjectEnd
				return []any{s.copyPath()}, nil
			default:
				panic(delim)
			}
			continue
		}
		switch s.states[len(s.states)-1] {
		case gshJqJSONStateArrayStart:
			s.states[len(s.states)-1] = gshJqJSONStateArrayValue
			fallthrough
		case gshJqJSONStateArrayValue:
			return []any{s.copyPath(), token}, nil
		case gshJqJSONStateObjectStart, gshJqJSONStateObjectValue:
			s.states[len(s.states)-1] = gshJqJSONStateObjectKey
			s.path = append(s.path, token)
		case gshJqJSONStateObjectKey:
			s.states[len(s.states)-1] = gshJqJSONStateObjectValue
			return []any{s.copyPath(), token}, nil
		default:
			s.states[len(s.states)-1] = gshJqJSONStateTopValue
			return []any{s.copyPath(), token}, nil
		}
	}
}

func (s *gshJqJSONStream) copyPath() []any {
	path := make([]any, len(s.path))
	copy(path, s.path)
	return path
}
