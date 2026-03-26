package bot

import (
	"fmt"
	"regexp"
	"strings"
)

const simpleMatcherSeparatorRegex = `(?:[ -]+)`

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

type simpleMatcherParser struct {
	spec string
	pos  int
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
}

type simpleMatcherLiteral struct {
	value string
}

type simpleMatcherSlot struct {
	name string
	kind string
}

type simpleMatcherGroup struct {
	expr     simpleMatcherExpr
	optional bool
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
		compiled, err := compileSimpleMatcher(simple)
		if err != nil {
			return err
		}
		regex = compiled
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
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return "", fmt.Errorf("SimpleMatcher cannot be empty")
	}
	parser := simpleMatcherParser{spec: spec}
	expr, err := parser.parseExpr(0)
	if err != nil {
		return "", err
	}
	parser.skipSpaces()
	if !parser.eof() {
		return "", fmt.Errorf("unexpected trailing character %q in SimpleMatcher", parser.peek())
	}
	regex, err := expr.compileBare()
	if err != nil {
		return "", err
	}
	if regex == "" {
		return "", fmt.Errorf("SimpleMatcher cannot compile to an empty pattern")
	}
	return `(?i:` + regex + `)`, nil
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
			p.pos++
			continue
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
		expr, err := p.parseExpr(']')
		if err != nil {
			return nil, err
		}
		return simpleMatcherGroup{expr: expr, optional: true}, nil
	case '(':
		p.pos++
		expr, err := p.parseExpr(')')
		if err != nil {
			return nil, err
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
			if strings.ContainsRune("[]()|<>", ch) || isSimpleMatcherSpace(ch) {
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

func (g simpleMatcherGroup) compileBare() (string, error) {
	return g.expr.compileBare()
}

func (g simpleMatcherGroup) isOptional() bool {
	return g.optional
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
