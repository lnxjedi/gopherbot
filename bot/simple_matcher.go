package bot

import (
	"fmt"
	"regexp"
	"strings"
)

const simpleMatcherSeparatorRegex = `(?:\s+|-+)`

var simpleMatcherIdentifierRe = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*$`)

var simpleMatcherTypePatterns = map[string]string{
	"base64":   `[A-Za-z0-9+/=]+`,
	"bool":     `(?:true|false|yes|no|on|off|1|0)`,
	"cidr":     `(?:\d{1,3}\.){3}\d{1,3}/\d{1,2}`,
	"decimal":  `[+-]?(?:\d+(?:\.\d+)?|\.\d+)`,
	"dnsname":  `(?:[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?)(?:\.(?:[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?))*`,
	"duration": `(?:\d+(?:ns|us|ms|s|m|h))+`,
	"email":    `[^\s@]+@[^\s@]+\.[^\s@]+`,
	"ident":    `[A-Za-z][\w-]*`,
	"ip":       `(?:\d{1,3}\.){3}\d{1,3}|(?:[0-9A-Fa-f:]+:+)+[0-9A-Fa-f]+`,
	"ipv4":     `(?:\d{1,3}\.){3}\d{1,3}`,
	"ipv6":     `(?:[0-9A-Fa-f:]+:+)+[0-9A-Fa-f]+`,
	"number":   `[+-]?\d+`,
	"rest":     `.+`,
	"slug":     `[\w.*-]+`,
	"token":    `[^\s]+`,
	"url":      `[A-Za-z][A-Za-z0-9+.-]*://[^\s]+`,
}

var simpleMatcherTypeDescriptions = map[string]string{
	"base64":   "base64 text.",
	"bool":     "a boolean value: true, false, yes, no, on, off, 1, or 0.",
	"cidr":     "a CIDR block like 10.0.0.0/24.",
	"decimal":  "a decimal number.",
	"dnsname":  "a DNS hostname.",
	"duration": "a Go-style duration like 5m30s.",
	"email":    "an email address.",
	"ident":    "an identifier starting with a letter, followed by letters, numbers, '_' or '-'.",
	"ip":       "an IP address.",
	"ipv4":     "an IPv4 address.",
	"ipv6":     "an IPv6 address.",
	"number":   "an integer.",
	"slug":     "a slug-like identifier.",
	"token":    "a non-whitespace token.",
	"url":      "a full URL.",
}

type simpleMatcherParser struct {
	spec string
	pos  int
}

type inputMatchKind int

const (
	inputNoMatch inputMatchKind = iota
	inputSyntaxMatch
	inputExactMatch
)

type inputMatchResult struct {
	kind       inputMatchKind
	args       []string
	diagnostic string
}

type simpleMatcher struct {
	spec  string
	expr  simpleMatcherExpr
	regex string
}

type simpleMatcherExpr struct {
	alternatives []simpleMatcherSequence
}

type simpleMatcherSequence struct {
	terms []simpleMatcherTerm
}

type simpleMatcherTerm interface {
	compileBare() (string, error)
	isOptional() bool
	containsSlot() bool
	matchTokens([]simpleMatcherToken, simpleMatcherMatchState) []simpleMatcherMatchState
}

type simpleMatcherLiteral struct {
	value string
}

type simpleMatcherSlot struct {
	name string
	kind string
}

type simpleMatcherGroup struct {
	expr      simpleMatcherExpr
	label     string
	optional  bool
	capturing bool
}

func compileInputMatcher(matcher *InputMatcher, allowSimple bool) error {
	regex := strings.TrimSpace(matcher.Regex)
	simple := strings.TrimSpace(matcher.SimpleMatcher)

	switch {
	case regex != "" && simple != "":
		return fmt.Errorf("matcher specifies both Regex and SimpleMatcher")
	case simple != "" && !allowSimple:
		return fmt.Errorf("SimpleMatcher is only supported for directed Commands")
	case regex == "" && simple == "":
		if allowSimple {
			return fmt.Errorf("matcher must specify either Regex or SimpleMatcher")
		}
		return fmt.Errorf("matcher must specify Regex")
	}

	if simple != "" {
		compiled, err := compileSimpleMatcherObject(simple)
		if err != nil {
			return err
		}
		matcher.simple = compiled
		regex = compiled.regex
	}

	wrapped := `^(?s:\s*` + regex + `\s*)$`
	re, err := regexp.Compile(wrapped)
	if err != nil {
		return fmt.Errorf("couldn't compile matcher regular expression '%s': %w", wrapped, err)
	}
	matcher.Regex = wrapped
	matcher.re = re
	return nil
}

func compileSimpleMatcher(spec string) (string, error) {
	compiled, err := compileSimpleMatcherObject(spec)
	if err != nil {
		return "", err
	}
	return compiled.regex, nil
}

func compileSimpleMatcherObject(spec string) (*simpleMatcher, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return nil, fmt.Errorf("SimpleMatcher cannot be empty")
	}
	parser := simpleMatcherParser{spec: spec}
	expr, err := parser.parseExpr(0)
	if err != nil {
		return nil, err
	}
	parser.skipSpaces()
	if !parser.eof() {
		return nil, fmt.Errorf("unexpected trailing character %q in SimpleMatcher", parser.peek())
	}
	regex, err := expr.compileBare()
	if err != nil {
		return nil, err
	}
	if regex == "" {
		return nil, fmt.Errorf("SimpleMatcher cannot compile to an empty pattern")
	}
	return &simpleMatcher{
		spec:  spec,
		expr:  expr,
		regex: `(?i:` + regex + `)`,
	}, nil
}

func (p *simpleMatcherParser) parseExpr(end rune) (simpleMatcherExpr, error) {
	alternatives := make([]simpleMatcherSequence, 0, 1)
	for {
		seq, err := p.parseSequence(end)
		if err != nil {
			return simpleMatcherExpr{}, err
		}
		alternatives = append(alternatives, seq)
		p.skipSpaces()

		if end != 0 {
			if p.eof() {
				return simpleMatcherExpr{}, fmt.Errorf("unterminated SimpleMatcher group, missing %q", string(end))
			}
			switch p.peek() {
			case '|':
				p.pos++
				continue
			case end:
				p.pos++
				return simpleMatcherExpr{alternatives: alternatives}, nil
			default:
				return simpleMatcherExpr{}, fmt.Errorf("unexpected character %q in SimpleMatcher group", p.peek())
			}
		}

		if p.eof() {
			return simpleMatcherExpr{alternatives: alternatives}, nil
		}
		if p.peek() == '|' {
			return simpleMatcherExpr{}, fmt.Errorf("unexpected character %q in SimpleMatcher", p.peek())
		}
		return simpleMatcherExpr{}, fmt.Errorf("unexpected character %q in SimpleMatcher", p.peek())
	}
}

func (p *simpleMatcherParser) parseSequence(end rune) (simpleMatcherSequence, error) {
	terms := make([]simpleMatcherTerm, 0, 4)
	for {
		p.skipSpaces()
		if p.eof() {
			break
		}
		ch := p.peek()
		if ch == '|' || (end != 0 && ch == end) {
			break
		}
		term, err := p.parseTerm()
		if err != nil {
			return simpleMatcherSequence{}, err
		}
		terms = append(terms, term)
	}
	return simpleMatcherSequence{terms: terms}, nil
}

func (p *simpleMatcherParser) parseTerm() (simpleMatcherTerm, error) {
	switch p.peek() {
	case '[':
		p.pos++
		body, err := p.readDelimitedBody(']')
		if err != nil {
			return nil, err
		}
		expr, label, err := parseSimpleMatcherBracketBody(body, true)
		if err != nil {
			return nil, err
		}
		hasSlots := expr.containsSlot()
		return simpleMatcherGroup{expr: expr, label: label, optional: true, capturing: !hasSlots}, nil
	case '{':
		p.pos++
		expr, err := p.parseExpr('}')
		if err != nil {
			return nil, err
		}
		if expr.containsSlot() {
			return nil, fmt.Errorf("non-capturing SimpleMatcher group cannot contain typed captures")
		}
		return simpleMatcherGroup{expr: expr, optional: true, capturing: false}, nil
	case '(':
		p.pos++
		body, err := p.readDelimitedBody(')')
		if err != nil {
			return nil, err
		}
		expr, label, err := parseSimpleMatcherBracketBody(body, false)
		if err != nil {
			return nil, err
		}
		if expr.containsSlot() {
			return nil, fmt.Errorf("capturing SimpleMatcher choice cannot contain typed captures")
		}
		return simpleMatcherGroup{expr: expr, label: label, capturing: true}, nil
	case '/':
		p.pos++
		expr, err := p.parseExpr('/')
		if err != nil {
			return nil, err
		}
		if expr.containsSlot() {
			return nil, fmt.Errorf("non-capturing SimpleMatcher synonym group cannot contain typed captures")
		}
		return simpleMatcherGroup{expr: expr}, nil
	case '<':
		p.pos++
		start := p.pos
		for !p.eof() && p.peek() != '>' {
			p.pos++
		}
		if p.eof() {
			return nil, fmt.Errorf("unterminated capture in SimpleMatcher")
		}
		raw := strings.TrimSpace(p.spec[start:p.pos])
		p.pos++
		return parseSimpleMatcherSlot(raw)
	default:
		start := p.pos
		for !p.eof() {
			ch := p.peek()
			if strings.ContainsRune("[]{}()/|<>", ch) || isSimpleMatcherSpace(ch) {
				break
			}
			p.pos++
		}
		literal := strings.TrimSpace(p.spec[start:p.pos])
		if literal == "" {
			return nil, fmt.Errorf("expected literal token in SimpleMatcher")
		}
		return simpleMatcherLiteral{value: literal}, nil
	}
}

func parseSimpleMatcherBracketBody(body string, optional bool) (simpleMatcherExpr, string, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return simpleMatcherExpr{}, "", fmt.Errorf("capturing SimpleMatcher choice cannot be empty")
	}
	if strings.Contains(body, "<") {
		expr, err := parseSimpleMatcherInnerExpr(body)
		if err != nil {
			return simpleMatcherExpr{}, "", err
		}
		if !optional {
			return simpleMatcherExpr{}, "", fmt.Errorf("capturing SimpleMatcher choice cannot contain typed captures")
		}
		return expr, "", nil
	}
	colon := strings.IndexRune(body, ':')
	if colon < 0 {
		return simpleMatcherExpr{}, "", fmt.Errorf("capturing SimpleMatcher choice must include a label prefix")
	}
	label := strings.TrimSpace(body[:colon])
	if label != "" && !simpleMatcherIdentifierRe.MatchString(label) {
		return simpleMatcherExpr{}, "", fmt.Errorf("invalid SimpleMatcher choice label %q", label)
	}
	choices := strings.TrimSpace(body[colon+1:])
	if choices == "" {
		return simpleMatcherExpr{}, "", fmt.Errorf("capturing SimpleMatcher choice must include at least one value")
	}
	expr, err := parseSimpleMatcherInnerExpr(choices)
	if err != nil {
		return simpleMatcherExpr{}, "", err
	}
	return expr, label, nil
}

func parseSimpleMatcherInnerExpr(spec string) (simpleMatcherExpr, error) {
	parser := simpleMatcherParser{spec: spec}
	alternatives := make([]simpleMatcherSequence, 0, 1)
	for {
		seq, err := parser.parseSequence(0)
		if err != nil {
			return simpleMatcherExpr{}, err
		}
		alternatives = append(alternatives, seq)
		parser.skipSpaces()
		if parser.eof() {
			return simpleMatcherExpr{alternatives: alternatives}, nil
		}
		if parser.peek() != '|' {
			return simpleMatcherExpr{}, fmt.Errorf("unexpected character %q in SimpleMatcher choice", parser.peek())
		}
		parser.pos++
	}
}

func (p *simpleMatcherParser) readDelimitedBody(end rune) (string, error) {
	start := p.pos
	for !p.eof() && p.peek() != end {
		p.pos++
	}
	if p.eof() {
		return "", fmt.Errorf("unterminated SimpleMatcher group, missing %q", string(end))
	}
	body := p.spec[start:p.pos]
	p.pos++
	return body, nil
}

func parseSimpleMatcherSlot(raw string) (simpleMatcherTerm, error) {
	parts := strings.Split(raw, ":")
	switch len(parts) {
	case 1:
		kind := strings.TrimSpace(parts[0])
		if !simpleMatcherIdentifierRe.MatchString(kind) {
			return nil, fmt.Errorf("invalid SimpleMatcher capture type %q", kind)
		}
		return simpleMatcherSlot{kind: kind}, nil
	case 2:
		name := strings.TrimSpace(parts[0])
		kind := strings.TrimSpace(parts[1])
		if !simpleMatcherIdentifierRe.MatchString(name) {
			return nil, fmt.Errorf("invalid SimpleMatcher capture name %q", name)
		}
		if !simpleMatcherIdentifierRe.MatchString(kind) {
			return nil, fmt.Errorf("invalid SimpleMatcher capture type %q", kind)
		}
		return simpleMatcherSlot{name: name, kind: kind}, nil
	default:
		return nil, fmt.Errorf("invalid capture %q in SimpleMatcher", raw)
	}
}

func (e simpleMatcherExpr) compileBare() (string, error) {
	switch len(e.alternatives) {
	case 0:
		return "", nil
	case 1:
		return e.alternatives[0].compileBare()
	default:
		parts := make([]string, 0, len(e.alternatives))
		for _, alt := range e.alternatives {
			part, err := alt.compileBare()
			if err != nil {
				return "", err
			}
			parts = append(parts, part)
		}
		return `(?:` + strings.Join(parts, `|`) + `)`, nil
	}
}

func (s simpleMatcherSequence) compileBare() (string, error) {
	if len(s.terms) == 0 {
		return "", nil
	}

	var out strings.Builder
	i := 0
	for i < len(s.terms) && s.terms[i].isOptional() {
		bare, err := s.terms[i].compileBare()
		if err != nil {
			return "", err
		}
		if i < len(s.terms)-1 {
			out.WriteString(`(?:`)
			out.WriteString(bare)
			out.WriteString(simpleMatcherSeparatorRegex)
			out.WriteString(`)?`)
		} else {
			out.WriteString(`(?:`)
			out.WriteString(bare)
			out.WriteString(`)?`)
		}
		i++
	}
	firstRequiredIdx := i

	for ; i < len(s.terms); i++ {
		bare, err := s.terms[i].compileBare()
		if err != nil {
			return "", err
		}
		if s.terms[i].isOptional() {
			out.WriteString(`(?:`)
			out.WriteString(simpleMatcherSeparatorRegex)
			out.WriteString(bare)
			out.WriteString(`)?`)
			continue
		}
		if i > firstRequiredIdx {
			out.WriteString(simpleMatcherSeparatorRegex)
		}
		out.WriteString(bare)
	}

	return out.String(), nil
}

func (l simpleMatcherLiteral) compileBare() (string, error) {
	if l.value == "" {
		return "", fmt.Errorf("empty literal in SimpleMatcher")
	}
	parts := strings.Split(l.value, "-")
	if len(parts) > 1 {
		allNonEmpty := true
		for _, part := range parts {
			if part == "" {
				allNonEmpty = false
				break
			}
		}
		if allNonEmpty {
			escaped := make([]string, 0, len(parts))
			for _, part := range parts {
				escaped = append(escaped, regexp.QuoteMeta(part))
			}
			return strings.Join(escaped, simpleMatcherSeparatorRegex), nil
		}
	}
	return regexp.QuoteMeta(l.value), nil
}

func (l simpleMatcherLiteral) isOptional() bool {
	return false
}

func (l simpleMatcherLiteral) containsSlot() bool {
	return false
}

func (s simpleMatcherSlot) compileBare() (string, error) {
	pattern, ok := simpleMatcherTypePatterns[s.kind]
	if !ok {
		return "", fmt.Errorf("unknown SimpleMatcher capture type %q", s.kind)
	}
	return `(` + pattern + `)`, nil
}

func (s simpleMatcherSlot) isOptional() bool {
	return false
}

func (s simpleMatcherSlot) containsSlot() bool {
	return true
}

func (g simpleMatcherGroup) compileBare() (string, error) {
	bare, err := g.expr.compileBare()
	if err != nil {
		return "", err
	}
	if g.capturing {
		return "(" + bare + ")", nil
	}
	return bare, nil
}

func (g simpleMatcherGroup) isOptional() bool {
	return g.optional
}

func (g simpleMatcherGroup) containsSlot() bool {
	return g.expr.containsSlot()
}

func (e simpleMatcherExpr) containsSlot() bool {
	for _, alt := range e.alternatives {
		if alt.containsSlot() {
			return true
		}
	}
	return false
}

func (s simpleMatcherSequence) containsSlot() bool {
	for _, term := range s.terms {
		if term.containsSlot() {
			return true
		}
	}
	return false
}

func simpleMatcherLiteralSequences(spec string) ([][]string, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return nil, fmt.Errorf("SimpleMatcher cannot be empty")
	}
	parser := simpleMatcherParser{spec: spec}
	expr, err := parser.parseExpr(0)
	if err != nil {
		return nil, err
	}
	parser.skipSpaces()
	if !parser.eof() {
		return nil, fmt.Errorf("unexpected trailing character %q in SimpleMatcher", parser.peek())
	}
	return expr.literalSequences(), nil
}

func (e simpleMatcherExpr) literalSequences() [][]string {
	sequences := make([][]string, 0, len(e.alternatives))
	for _, alt := range e.alternatives {
		sequences = append(sequences, alt.literalSequences()...)
	}
	return sequences
}

func (s simpleMatcherSequence) literalSequences() [][]string {
	sequences := [][]string{{}}
	for _, term := range s.terms {
		var options [][]string
		switch t := term.(type) {
		case simpleMatcherLiteral:
			options = [][]string{fallbackLiteralTokens(t.value)}
		case simpleMatcherSlot:
			options = [][]string{{}}
		case simpleMatcherGroup:
			options = t.literalSequences()
		default:
			options = [][]string{{}}
		}
		next := make([][]string, 0, len(sequences)*maxInt(1, len(options)))
		for _, existing := range sequences {
			for _, option := range options {
				combined := append([]string(nil), existing...)
				combined = append(combined, option...)
				next = append(next, combined)
			}
		}
		sequences = next
	}
	return sequences
}

func (g simpleMatcherGroup) literalSequences() [][]string {
	options := g.expr.literalSequences()
	if !g.optional {
		return options
	}
	return append([][]string{{}}, options...)
}

func (m InputMatcher) matchInput(input string) inputMatchResult {
	tryExact := func(candidate string) inputMatchResult {
		if m.re == nil {
			return inputMatchResult{kind: inputNoMatch}
		}
		matches := m.re.FindStringSubmatch(candidate)
		if matches == nil {
			return inputMatchResult{kind: inputNoMatch}
		}
		return inputMatchResult{kind: inputExactMatch, args: matches[1:]}
	}

	if exact := tryExact(input); exact.kind == inputExactMatch {
		return exact
	}
	collapsed := spaceRe.ReplaceAllString(input, " ")
	if collapsed != input {
		if exact := tryExact(collapsed); exact.kind == inputExactMatch {
			return exact
		}
	}

	if m.simple == nil {
		return inputMatchResult{kind: inputNoMatch}
	}
	return m.simple.syntaxMatch(input)
}

type simpleMatcherToken struct {
	value string
}

type simpleMatcherMatchState struct {
	pos        int
	args       []string
	diagnostic string
}

func (s *simpleMatcher) syntaxMatch(input string) inputMatchResult {
	tokens := simpleMatcherTokenize(input)
	if len(tokens) == 0 {
		return inputMatchResult{kind: inputNoMatch}
	}
	states := s.expr.matchTokens(tokens, simpleMatcherMatchState{})
	var diagnostics []string
	for _, state := range states {
		if state.pos != len(tokens) {
			continue
		}
		if state.diagnostic == "" {
			return inputMatchResult{kind: inputExactMatch, args: state.args}
		}
		diagnostics = appendUniquePreserveOrder(diagnostics, state.diagnostic)
	}
	if len(diagnostics) == 1 {
		return inputMatchResult{kind: inputSyntaxMatch, diagnostic: diagnostics[0]}
	}
	return inputMatchResult{kind: inputNoMatch}
}

func simpleMatcherTokenize(input string) []simpleMatcherToken {
	parts := strings.FieldsFunc(strings.TrimSpace(input), func(ch rune) bool {
		return isSimpleMatcherSpace(ch) || ch == '-'
	})
	tokens := make([]simpleMatcherToken, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		tokens = append(tokens, simpleMatcherToken{value: part})
	}
	return tokens
}

func (e simpleMatcherExpr) matchTokens(tokens []simpleMatcherToken, state simpleMatcherMatchState) []simpleMatcherMatchState {
	var out []simpleMatcherMatchState
	for _, alt := range e.alternatives {
		out = append(out, alt.matchTokens(tokens, state)...)
	}
	return out
}

func (s simpleMatcherSequence) matchTokens(tokens []simpleMatcherToken, initial simpleMatcherMatchState) []simpleMatcherMatchState {
	states := []simpleMatcherMatchState{initial}
	for _, term := range s.terms {
		var next []simpleMatcherMatchState
		for _, state := range states {
			next = append(next, term.matchTokens(tokens, state)...)
		}
		states = next
		if len(states) == 0 {
			break
		}
	}
	return states
}

func (l simpleMatcherLiteral) matchTokens(tokens []simpleMatcherToken, state simpleMatcherMatchState) []simpleMatcherMatchState {
	parts := simpleMatcherLiteralParts(l.value)
	if len(parts) == 0 || state.pos+len(parts) > len(tokens) {
		return nil
	}
	for i, part := range parts {
		if !strings.EqualFold(tokens[state.pos+i].value, part) {
			return nil
		}
	}
	state.pos += len(parts)
	return []simpleMatcherMatchState{state}
}

func (s simpleMatcherSlot) matchTokens(tokens []simpleMatcherToken, state simpleMatcherMatchState) []simpleMatcherMatchState {
	if state.pos >= len(tokens) {
		return nil
	}
	value := tokens[state.pos].value
	if s.kind == "rest" {
		value = joinSimpleMatcherTokens(tokens[state.pos:])
		state.pos = len(tokens)
		state.args = append(state.args, value)
		return []simpleMatcherMatchState{state}
	}
	if simpleMatcherTypeMatches(s.kind, value) {
		state.pos++
		state.args = append(state.args, value)
		return []simpleMatcherMatchState{state}
	}
	if state.diagnostic != "" {
		return nil
	}
	state.pos++
	state.diagnostic = invalidSimpleMatcherTypeDiagnostic(s.name, s.kind, value)
	return []simpleMatcherMatchState{state}
}

func (g simpleMatcherGroup) matchTokens(tokens []simpleMatcherToken, state simpleMatcherMatchState) []simpleMatcherMatchState {
	var out []simpleMatcherMatchState
	if g.optional {
		out = append(out, state)
	}
	if g.capturing && !g.containsSlot() {
		out = append(out, g.matchChoiceTokens(tokens, state)...)
		return out
	}
	return append(out, g.expr.matchTokens(tokens, state)...)
}

func (g simpleMatcherGroup) matchChoiceTokens(tokens []simpleMatcherToken, state simpleMatcherMatchState) []simpleMatcherMatchState {
	var out []simpleMatcherMatchState
	for _, alt := range g.expr.alternatives {
		if value, ok := alt.literalValueAt(tokens, state.pos); ok {
			matched := state
			matched.pos += len(simpleMatcherSequenceLiteralParts(alt))
			matched.args = append(matched.args, value)
			out = append(out, matched)
		}
	}
	if len(out) > 0 || state.pos >= len(tokens) || state.diagnostic != "" {
		return out
	}
	invalid := state
	invalid.pos++
	invalid.diagnostic = invalidSimpleMatcherChoiceDiagnostic(g.label, tokens[state.pos].value, g.choiceValues())
	return append(out, invalid)
}

func (s simpleMatcherSequence) literalValueAt(tokens []simpleMatcherToken, pos int) (string, bool) {
	parts := simpleMatcherSequenceLiteralParts(s)
	if len(parts) == 0 || pos+len(parts) > len(tokens) {
		return "", false
	}
	for i, part := range parts {
		if !strings.EqualFold(tokens[pos+i].value, part) {
			return "", false
		}
	}
	values := make([]simpleMatcherToken, len(parts))
	copy(values, tokens[pos:pos+len(parts)])
	return joinSimpleMatcherTokens(values), true
}

func simpleMatcherSequenceLiteralParts(s simpleMatcherSequence) []string {
	var parts []string
	for _, term := range s.terms {
		lit, ok := term.(simpleMatcherLiteral)
		if !ok {
			return nil
		}
		parts = append(parts, simpleMatcherLiteralParts(lit.value)...)
	}
	return parts
}

func (g simpleMatcherGroup) choiceValues() []string {
	values := make([]string, 0, len(g.expr.alternatives))
	for _, alt := range g.expr.alternatives {
		parts := simpleMatcherSequenceLiteralParts(alt)
		if len(parts) == 0 {
			continue
		}
		values = append(values, strings.Join(parts, " "))
	}
	return values
}

func simpleMatcherLiteralParts(value string) []string {
	raw := strings.FieldsFunc(value, func(ch rune) bool {
		return isSimpleMatcherSpace(ch) || ch == '-'
	})
	parts := make([]string, 0, len(raw))
	for _, part := range raw {
		if part == "" {
			continue
		}
		parts = append(parts, part)
	}
	return parts
}

func joinSimpleMatcherTokens(tokens []simpleMatcherToken) string {
	values := make([]string, 0, len(tokens))
	for _, token := range tokens {
		values = append(values, token.value)
	}
	return strings.Join(values, " ")
}

func simpleMatcherTypeMatches(kind, value string) bool {
	pattern, ok := simpleMatcherTypePatterns[kind]
	if !ok {
		return false
	}
	re, err := regexp.Compile(`(?i:^(?:` + pattern + `)$)`)
	if err != nil {
		return false
	}
	return re.MatchString(value)
}

func invalidSimpleMatcherTypeDiagnostic(label, kind, value string) string {
	name := strings.TrimSpace(label)
	if name == "" {
		name = kind
	}
	description := simpleMatcherTypeDescriptions[kind]
	if description == "" {
		description = "a valid " + kind + " value."
	}
	return fmt.Sprintf("Invalid value '%s' for '%s'; expected %s", value, name, description)
}

func invalidSimpleMatcherChoiceDiagnostic(label, value string, choices []string) string {
	if strings.TrimSpace(label) == "" {
		return fmt.Sprintf("Invalid value '%s'; valid values: %s.", value, strings.Join(choices, ", "))
	}
	return fmt.Sprintf("Invalid value '%s' for '%s'; valid values: %s.", value, label, strings.Join(choices, ", "))
}

func (p *simpleMatcherParser) skipSpaces() {
	for !p.eof() && isSimpleMatcherSpace(p.peek()) {
		p.pos++
	}
}

func (p *simpleMatcherParser) eof() bool {
	return p.pos >= len(p.spec)
}

func (p *simpleMatcherParser) peek() rune {
	return rune(p.spec[p.pos])
}

func isSimpleMatcherSpace(ch rune) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r'
}
