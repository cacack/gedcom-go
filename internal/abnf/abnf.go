// Package abnf reads a grammar written in ABNF (RFC 5234, with the
// case-sensitive strings of RFC 7405) and matches strings against its rules
// and generates strings from them.
//
// It is for tests only: internal/payloadgrammar uses it to hold the library's
// payload parsers to the GEDCOM 7.0 grammar (ADR 0013). It supports every
// construct RFC 5234 defines except prose values (<...>): alternation,
// incremental alternatives (=/), concatenation, repetition, optional
// sequences, groups, case-insensitive and case-sensitive strings, and
// numeric values as single characters, ranges and concatenations. Rule names
// are case-insensitive, and the RFC 5234 core rules are defined for a grammar
// that does not define them itself.
//
// Matching works on Unicode code points and explores every alternative, so a
// match is exact whatever order a rule lists its alternatives in.
package abnf

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// coreRules are the RFC 5234 appendix B.1 rules, added for any a grammar
// does not define.
const coreRules = `ALPHA = %x41-5A / %x61-7A
BIT = "0" / "1"
CHAR = %x01-7F
CR = %x0D
CRLF = CR LF
CTL = %x00-1F / %x7F
DIGIT = %x30-39
DQUOTE = %x22
HEXDIG = DIGIT / "A" / "B" / "C" / "D" / "E" / "F"
HTAB = %x09
LF = %x0A
LWSP = *(WSP / CRLF WSP)
OCTET = %x00-FF
SP = %x20
VCHAR = %x21-7E
WSP = SP / HTAB
`

// node is one element of a rule's definition.
type node struct {
	kind     nodeKind
	children []*node // alternatives, or a concatenation's elements
	min, max int     // repetition bounds; max -1 is unbounded
	name     string  // a rule reference, lower-cased
	text     []rune  // a string's characters
	fold     bool    // the string is case-insensitive
	lo, hi   rune    // a character range
}

type nodeKind int

const (
	kindAlt nodeKind = iota
	kindSeq
	kindRep
	kindRef
	kindString
	kindRange
)

// Grammar is a parsed ABNF grammar.
type Grammar struct {
	rules map[string]*node  // by lower-cased name
	names map[string]string // lower-cased name -> name as first written
}

// Parse reads an ABNF grammar. A rule a definition references but no rule
// defines is an error only when Match or Generate reaches it.
func Parse(src string) (*Grammar, error) {
	g := &Grammar{rules: map[string]*node{}, names: map[string]string{}}
	if err := g.add(src, false); err != nil {
		return nil, err
	}
	if err := g.add(coreRules, true); err != nil {
		return nil, fmt.Errorf("core rules: %w", err)
	}
	return g, nil
}

// add parses the rules of src. With core set, a rule src defines that g
// already has is skipped instead of being an error.
func (g *Grammar) add(src string, core bool) error {
	rules, err := logicalRules(src)
	if err != nil {
		return err
	}
	for _, r := range rules {
		name, incremental, body, err := splitRule(r.text)
		if err != nil {
			return fmt.Errorf("line %d: %w", r.line, err)
		}
		key := strings.ToLower(name)
		if core && g.rules[key] != nil {
			continue
		}
		p := &parser{src: []rune(body)}
		n, err := p.alternation()
		if err == nil && p.skipSpace() < len(p.src) {
			err = fmt.Errorf("unexpected %q", string(p.src[p.pos:]))
		}
		if err != nil {
			return fmt.Errorf("line %d: rule %s: %w", r.line, name, err)
		}
		switch existing := g.rules[key]; {
		case incremental && existing == nil:
			return fmt.Errorf("line %d: %s =/ before %s is defined", r.line, name, name)
		case incremental:
			if existing.kind != kindAlt {
				existing = &node{kind: kindAlt, children: []*node{existing}}
			}
			existing.children = append(existing.children, n)
			g.rules[key] = existing
		case existing != nil:
			return fmt.Errorf("line %d: %s is defined twice", r.line, name)
		default:
			g.rules[key] = n
			g.names[key] = name
		}
	}
	return nil
}

// Has reports whether the grammar defines a rule, named in any case.
func (g *Grammar) Has(rule string) bool {
	return g.rules[strings.ToLower(rule)] != nil
}

// Rules returns the names of the rules the grammar defines, as first
// written, sorted.
func (g *Grammar) Rules() []string {
	var names []string
	for _, name := range g.names {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// Match reports whether s, as a whole, matches rule. It returns an error when
// the rule, or a rule it references, is not defined.
func (g *Grammar) Match(rule, s string) (bool, error) {
	n := g.rules[strings.ToLower(rule)]
	if n == nil {
		return false, fmt.Errorf("no rule %s", rule)
	}
	m := &matcher{g: g, in: []rune(s), memo: map[memoKey][]int{}}
	ends, err := m.ends(n, 0)
	if err != nil {
		return false, err
	}
	return slices.Contains(ends, len(m.in)), nil
}

// logicalRule is one rule's text, its continuation lines joined, comments
// removed.
type logicalRule struct {
	line int
	text string
}

// logicalRules splits src into rules: a rule starts on a line that does not
// begin with whitespace and continues on the lines that do. Comments run
// from a ";" outside a string to the end of the line.
func logicalRules(src string) ([]logicalRule, error) {
	var rules []logicalRule
	for i, line := range strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n") {
		text, err := stripComment(line)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", i+1, err)
		}
		if strings.TrimSpace(text) == "" {
			continue
		}
		if text[0] == ' ' || text[0] == '\t' {
			if len(rules) == 0 {
				return nil, fmt.Errorf("line %d: continuation line before any rule", i+1)
			}
			rules[len(rules)-1].text += " " + text
			continue
		}
		rules = append(rules, logicalRule{line: i + 1, text: text})
	}
	return rules, nil
}

func stripComment(line string) (string, error) {
	inString := false
	for i, r := range line {
		switch {
		case r == '"':
			inString = !inString
		case r == ';' && !inString:
			return line[:i], nil
		}
	}
	if inString {
		return "", fmt.Errorf("unterminated string")
	}
	return line, nil
}

// splitRule splits "name = body" or "name =/ body".
func splitRule(text string) (name string, incremental bool, body string, err error) {
	eq := strings.Index(text, "=")
	if eq < 0 {
		return "", false, "", fmt.Errorf("no \"=\" in %q", text)
	}
	name = strings.TrimSpace(text[:eq])
	if !isRuleName(name) {
		return "", false, "", fmt.Errorf("%q is not a rule name", name)
	}
	body = text[eq+1:]
	if strings.HasPrefix(body, "/") {
		incremental, body = true, body[1:]
	}
	return name, incremental, body, nil
}

func isRuleName(s string) bool {
	for i, r := range s {
		if !isASCIILetter(r) && (i == 0 || r != '-' && (r < '0' || r > '9')) {
			return false
		}
	}
	return s != ""
}

func isASCIILetter(r rune) bool {
	return r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z'
}

// parser reads one rule's definition.
type parser struct {
	src []rune
	pos int
}

func (p *parser) skipSpace() int {
	for p.pos < len(p.src) && unicode.IsSpace(p.src[p.pos]) {
		p.pos++
	}
	return p.pos
}

func (p *parser) peek() rune {
	if p.skipSpace() >= len(p.src) {
		return 0
	}
	return p.src[p.pos]
}

func (p *parser) alternation() (*node, error) {
	var alts []*node
	for {
		seq, err := p.concatenation()
		if err != nil {
			return nil, err
		}
		alts = append(alts, seq)
		if p.peek() != '/' {
			break
		}
		p.pos++
	}
	if len(alts) == 1 {
		return alts[0], nil
	}
	return &node{kind: kindAlt, children: alts}, nil
}

func (p *parser) concatenation() (*node, error) {
	var elems []*node
	for {
		switch p.peek() {
		case 0, '/', ')', ']':
			if len(elems) == 0 {
				return nil, fmt.Errorf("empty alternative at %d", p.pos)
			}
			if len(elems) == 1 {
				return elems[0], nil
			}
			return &node{kind: kindSeq, children: elems}, nil
		}
		elem, err := p.repetition()
		if err != nil {
			return nil, err
		}
		elems = append(elems, elem)
	}
}

func (p *parser) repetition() (*node, error) {
	p.skipSpace()
	lo, hasLo := p.number(10)
	hasStar := p.pos < len(p.src) && p.src[p.pos] == '*'
	if !hasLo && !hasStar {
		return p.element()
	}
	minimum, maximum := lo, lo
	if hasStar {
		p.pos++
		maximum = -1
		if hi, ok := p.number(10); ok {
			maximum = hi
		}
		if !hasLo {
			minimum = 0
		}
	}
	elem, err := p.element()
	if err != nil {
		return nil, err
	}
	return &node{kind: kindRep, children: []*node{elem}, min: minimum, max: maximum}, nil
}

// number reads an unsigned number in base at the current position.
func (p *parser) number(base int) (int, bool) {
	start := p.pos
	for p.pos < len(p.src) && digitValue(p.src[p.pos]) < base {
		p.pos++
	}
	if p.pos == start {
		return 0, false
	}
	n, err := strconv.ParseInt(string(p.src[start:p.pos]), base, 32)
	if err != nil {
		p.pos = start
		return 0, false
	}
	return int(n), true
}

func digitValue(r rune) int {
	switch {
	case r >= '0' && r <= '9':
		return int(r - '0')
	case r >= 'A' && r <= 'F':
		return int(r-'A') + 10
	case r >= 'a' && r <= 'f':
		return int(r-'a') + 10
	}
	return 99
}

func (p *parser) element() (*node, error) {
	switch r := p.peek(); {
	case r == '(' || r == '[':
		return p.group(r)
	case r == '"':
		return p.quoted(true)
	case r == '%':
		return p.percent()
	case isASCIILetter(r):
		return p.ruleName(), nil
	case r == '<':
		return nil, fmt.Errorf("prose values are not supported (at %d)", p.pos)
	case r == 0:
		return nil, fmt.Errorf("missing element at end")
	default:
		return nil, fmt.Errorf("unexpected %q at %d", r, p.pos)
	}
}

// group reads a group or an optional sequence at its opening bracket.
func (p *parser) group(opener rune) (*node, error) {
	p.pos++
	inner, err := p.alternation()
	if err != nil {
		return nil, err
	}
	closer := map[rune]rune{'(': ')', '[': ']'}[opener]
	if p.peek() != closer {
		return nil, fmt.Errorf("missing %q at %d", closer, p.pos)
	}
	p.pos++
	if opener == '[' {
		return &node{kind: kindRep, children: []*node{inner}, min: 0, max: 1}, nil
	}
	return inner, nil
}

// ruleName reads a reference to a rule.
func (p *parser) ruleName() *node {
	start := p.pos
	for p.pos < len(p.src) && (isASCIILetter(p.src[p.pos]) || p.src[p.pos] == '-' ||
		p.src[p.pos] >= '0' && p.src[p.pos] <= '9') {
		p.pos++
	}
	return &node{kind: kindRef, name: strings.ToLower(string(p.src[start:p.pos]))}
}

// quoted reads a string at the opening quote.
func (p *parser) quoted(fold bool) (*node, error) {
	p.pos++ // the opening quote
	start := p.pos
	for p.pos < len(p.src) && p.src[p.pos] != '"' {
		p.pos++
	}
	if p.pos == len(p.src) {
		return nil, fmt.Errorf("unterminated string at %d", start)
	}
	text := slices.Clone(p.src[start:p.pos])
	p.pos++
	return &node{kind: kindString, text: text, fold: fold}, nil
}

// percent reads %s"..." and %i"..." strings and %x, %d and %b numeric values.
func (p *parser) percent() (*node, error) {
	p.pos++
	if p.pos+1 >= len(p.src) {
		return nil, fmt.Errorf("incomplete %% value")
	}
	marker := unicode.ToLower(p.src[p.pos])
	p.pos++
	switch marker {
	case 's', 'i':
		if p.src[p.pos] != '"' {
			return nil, fmt.Errorf("%%%c without a string at %d", marker, p.pos)
		}
		return p.quoted(marker == 'i')
	}
	base := map[rune]int{'x': 16, 'd': 10, 'b': 2}[marker]
	if base == 0 {
		return nil, fmt.Errorf("unknown %%%c at %d", marker, p.pos)
	}
	first, ok := p.number(base)
	if !ok {
		return nil, fmt.Errorf("missing number at %d", p.pos)
	}
	if p.pos < len(p.src) && p.src[p.pos] == '-' {
		p.pos++
		last, ok := p.number(base)
		if !ok || last < first {
			return nil, fmt.Errorf("bad range at %d", p.pos)
		}
		return &node{kind: kindRange, lo: rune(first), hi: rune(last)}, nil // #nosec G115 -- number parses at most 32 bits
	}
	text := []rune{rune(first)} // #nosec G115 -- number parses at most 32 bits
	for p.pos < len(p.src) && p.src[p.pos] == '.' {
		p.pos++
		next, ok := p.number(base)
		if !ok {
			return nil, fmt.Errorf("missing number at %d", p.pos)
		}
		text = append(text, rune(next)) // #nosec G115 -- number parses at most 32 bits
	}
	return &node{kind: kindString, text: text}, nil
}

// matcher matches one input. It memoizes the end positions of each rule
// reference by start position.
type matcher struct {
	g    *Grammar
	in   []rune
	memo map[memoKey][]int
}

type memoKey struct {
	rule string
	pos  int
}

// ends returns, sorted, every position at which n can stop matching when it
// starts at pos.
func (m *matcher) ends(n *node, pos int) ([]int, error) {
	switch n.kind {
	case kindString:
		return m.literal(n, pos), nil
	case kindRange:
		if pos < len(m.in) && m.in[pos] >= n.lo && m.in[pos] <= n.hi {
			return []int{pos + 1}, nil
		}
		return nil, nil
	case kindRef:
		return m.reference(n, pos)
	case kindAlt:
		var all []int
		for _, alt := range n.children {
			ends, err := m.ends(alt, pos)
			if err != nil {
				return nil, err
			}
			all = append(all, ends...)
		}
		return sortedSet(all), nil
	case kindSeq:
		frontier := []int{pos}
		for _, elem := range n.children {
			next, err := m.endsFrom(elem, frontier)
			if err != nil || len(next) == 0 {
				return nil, err
			}
			frontier = next
		}
		return frontier, nil
	default: // kindRep
		return m.repeat(n, pos)
	}
}

// literal matches a string.
func (m *matcher) literal(n *node, pos int) []int {
	if pos+len(n.text) > len(m.in) {
		return nil
	}
	for i, want := range n.text {
		if got := m.in[pos+i]; got != want && (!n.fold || foldASCII(got) != foldASCII(want)) {
			return nil
		}
	}
	return []int{pos + len(n.text)}
}

// reference matches a rule reference, memoized by start position.
func (m *matcher) reference(n *node, pos int) ([]int, error) {
	key := memoKey{n.name, pos}
	if ends, ok := m.memo[key]; ok {
		return ends, nil
	}
	rule := m.g.rules[n.name]
	if rule == nil {
		return nil, fmt.Errorf("no rule %s", n.name)
	}
	m.memo[key] = nil // a left-recursive reference matches nothing
	ends, err := m.ends(rule, pos)
	m.memo[key] = ends
	return ends, err
}

// endsFrom returns the end positions of n started at each of starts.
func (m *matcher) endsFrom(n *node, starts []int) ([]int, error) {
	var all []int
	for _, start := range starts {
		ends, err := m.ends(n, start)
		if err != nil {
			return nil, err
		}
		all = append(all, ends...)
	}
	return sortedSet(all), nil
}

// repeat matches a repetition. Once the minimum count is reached, a position
// reached again after more repetitions is dropped: the fewer repetitions
// that reached it first allow at least as many continuations.
func (m *matcher) repeat(n *node, pos int) ([]int, error) {
	seen := map[int]bool{}
	var out []int
	frontier := []int{pos}
	for count := 0; len(frontier) > 0; count++ {
		if count >= n.min {
			for _, p := range frontier {
				seen[p] = true
			}
			out = append(out, frontier...)
		}
		if count == n.max {
			break
		}
		next, err := m.endsFrom(n.children[0], frontier)
		if err != nil {
			return nil, err
		}
		if count+1 >= n.min {
			next = slices.DeleteFunc(next, func(p int) bool { return seen[p] })
		}
		frontier = next
	}
	return sortedSet(out), nil
}

func sortedSet(positions []int) []int {
	slices.Sort(positions)
	return slices.Compact(positions)
}

// foldASCII lower-cases an ASCII letter: RFC 5234 strings are
// case-insensitive for ASCII letters only.
func foldASCII(r rune) rune {
	if r >= 'A' && r <= 'Z' {
		return r + 'a' - 'A'
	}
	return r
}

// validRune reports whether r can be held in a Go string as itself.
func validRune(r rune) bool {
	return utf8.ValidRune(r)
}
