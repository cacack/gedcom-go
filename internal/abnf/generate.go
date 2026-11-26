package abnf

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"unicode"
)

// Generator produces strings from a grammar's rules by a seeded random walk,
// so the same seed always yields the same strings.
type Generator struct {
	g *Grammar
	r *rand.Rand
	// MaxExtra caps how many repetitions beyond its minimum an unbounded
	// repetition makes. Zero means 3.
	MaxExtra int
	// Exclude, when set, rejects a character a range would otherwise
	// produce, such as one the grammar bans in prose.
	Exclude func(rune) bool
	// Override maps a rule name, in any case, to strings a reference to the
	// rule produces instead of a walk of its definition, four times in five.
	// It steers a rule the grammar leaves open, such as a month that is any
	// tag, toward the values its prose allows.
	Override map[string][]string
}

// NewGenerator returns a generator for g seeded with seed.
func NewGenerator(g *Grammar, seed uint64) *Generator {
	return &Generator{g: g, r: rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))} // #nosec G404 -- deterministic test input, not security
}

// Generate returns a string matching rule.
func (gen *Generator) Generate(rule string) (string, error) {
	var b strings.Builder
	if err := gen.walk(&b, &node{kind: kindRef, name: strings.ToLower(rule)}, 0); err != nil {
		return "", err
	}
	return b.String(), nil
}

// maxDepth bounds the walk of a recursive grammar.
const maxDepth = 64

func (gen *Generator) walk(b *strings.Builder, n *node, depth int) error {
	if depth > maxDepth {
		return fmt.Errorf("generation deeper than %d", maxDepth)
	}
	switch n.kind {
	case kindString:
		gen.literal(b, n)
	case kindRange:
		r, err := gen.pick(n.lo, n.hi)
		if err != nil {
			return err
		}
		b.WriteRune(r)
	case kindRef:
		if values := gen.override(n.name); len(values) > 0 && gen.r.IntN(5) > 0 {
			b.WriteString(values[gen.r.IntN(len(values))])
			return nil
		}
		rule := gen.g.rules[n.name]
		if rule == nil {
			return fmt.Errorf("no rule %s", n.name)
		}
		return gen.walk(b, rule, depth+1)
	case kindAlt:
		return gen.walk(b, n.children[gen.r.IntN(len(n.children))], depth+1)
	case kindSeq:
		for _, elem := range n.children {
			if err := gen.walk(b, elem, depth+1); err != nil {
				return err
			}
		}
	default: // kindRep
		for range gen.count(n) {
			if err := gen.walk(b, n.children[0], depth+1); err != nil {
				return err
			}
		}
	}
	return nil
}

// literal writes a string, flipping the case of a case-insensitive one's
// letters at random.
func (gen *Generator) literal(b *strings.Builder, n *node) {
	for _, r := range n.text {
		if n.fold && gen.r.IntN(2) == 0 {
			r = flipCase(r)
		}
		b.WriteRune(r)
	}
}

// count picks how many times a repetition repeats.
func (gen *Generator) count(n *node) int {
	extra := gen.MaxExtra
	if extra == 0 {
		extra = 3
	}
	switch {
	case n.max < 0:
		return n.min + gen.r.IntN(extra+1)
	case n.max > n.min:
		return n.min + gen.r.IntN(min(n.max-n.min, extra)+1)
	}
	return n.min
}

func (gen *Generator) override(name string) []string {
	for rule, values := range gen.Override {
		if strings.EqualFold(rule, name) {
			return values
		}
	}
	return nil
}

// samples are characters beyond ASCII a wide range draws from now and then.
var samples = []rune{0xA0, 0xE9, 0x416, 0x5D0, 0x4E2D, 0x1F600}

// pick returns a character in [lo, hi], usually ASCII when the range
// reaches it, never one Exclude rejects or a Go string cannot hold.
func (gen *Generator) pick(lo, hi rune) (rune, error) {
	for range 100 {
		var r rune
		switch c := gen.r.IntN(8); {
		case c == 0:
			r = lo
		case c == 1:
			r = hi
		case c == 2:
			r = samples[gen.r.IntN(len(samples))]
		case lo < 0x7F:
			r = lo + gen.r.Int32N(min(hi, 0x7E)-lo+1)
		default:
			r = lo + gen.r.Int32N(hi-lo+1)
		}
		if r >= lo && r <= hi && validRune(r) && (gen.Exclude == nil || !gen.Exclude(r)) {
			return r, nil
		}
	}
	return 0, fmt.Errorf("no character in %%x%X-%X that may be generated", lo, hi)
}

func flipCase(r rune) rune {
	if r < 0x80 && unicode.IsUpper(r) {
		return unicode.ToLower(r)
	}
	if r < 0x80 && unicode.IsLower(r) {
		return unicode.ToUpper(r)
	}
	return r
}

// mutationAlphabet is what Mutate inserts and substitutes: the characters
// GEDCOM payload grammars turn on.
const mutationAlphabet = " \t/@.,:<>_-+09azAZyYmMwWdDnNsSeE"

// Mutate returns a near miss of s: one character deleted, doubled,
// substituted, inserted or swapped with its neighbor, a letter's case
// flipped, or a space added at either end. The result may still match the
// rule s came from; the matcher, not Mutate, decides.
func (gen *Generator) Mutate(s string) string {
	rs := []rune(s)
	char := func() rune { return rune(mutationAlphabet[gen.r.IntN(len(mutationAlphabet))]) }
	if len(rs) == 0 {
		return string(char())
	}
	i := gen.r.IntN(len(rs))
	switch gen.r.IntN(8) {
	case 0:
		rs = append(rs[:i:i], rs[i+1:]...)
	case 1:
		rs = append(rs[:i+1:i+1], rs[i:]...)
	case 2:
		rs[i] = char()
	case 3:
		rs = append(rs[:i:i], append([]rune{char()}, rs[i:]...)...)
	case 4:
		if i+1 < len(rs) {
			rs[i], rs[i+1] = rs[i+1], rs[i]
		}
	case 5:
		rs[i] = flipCase(rs[i])
	case 6:
		return " " + s
	default:
		return s + " "
	}
	return string(rs)
}
