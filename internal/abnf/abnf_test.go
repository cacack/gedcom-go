package abnf

import (
	"strings"
	"testing"
)

func mustParse(t *testing.T, src string) *Grammar {
	t.Helper()
	g, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func TestMatch(t *testing.T) {
	g := mustParse(t, `; a comment line
greeting  = "Hi" / %s"YO"  ; case-insensitive, then case-sensitive
greeting  =/ %i"hey"
counted   = 2*3"a" 1"b" *1"c" 2DIGIT
ranges    = %x41-43 %d68 %b1000101 %x46.47
grouped   = ("x" / "y") ["z" "z"]
Mixed-Case = digit
semicolon = ";" %x3B
spread    = "a"
            "b"
                       ; a comment inside a rule
            "c"
backtrack = *"a" "a"
unicode   = %x4E2D-4E2E %x1F600
`)
	tests := []struct {
		rule, in string
		want     bool
	}{
		{"greeting", "hi", true},
		{"greeting", "HI", true},
		{"greeting", "YO", true},
		{"greeting", "yo", false},
		{"greeting", "HEY", true},
		{"greeting", "hih", false},
		{"counted", "aab12", true},
		{"counted", "aaabc12", true},
		{"counted", "ab12", false},
		{"counted", "aaaab12", false},
		{"counted", "aabcc12", false},
		{"counted", "aab1", false},
		{"ranges", "ADEFG", true},
		{"ranges", "CDEFG", true},
		{"ranges", "DDEFG", false},
		{"ranges", "ADEF", false},
		{"grouped", "x", true},
		{"grouped", "yzz", true},
		{"grouped", "xz", false},
		{"GROUPED", "y", true},
		{"mixed-case", "7", true},
		{"semicolon", ";;", true},
		{"spread", "abc", true},
		{"backtrack", "aaa", true},
		{"backtrack", "", false},
		{"unicode", "中\U0001F600", true},
		{"unicode", "中", false},
		{"ALPHA", "q", true},
		{"CRLF", "\r\n", true},
		{"WSP", "\t", true},
	}
	for _, tt := range tests {
		got, err := g.Match(tt.rule, tt.in)
		if err != nil {
			t.Fatalf("Match(%s, %q): %v", tt.rule, tt.in, err)
		}
		if got != tt.want {
			t.Errorf("Match(%s, %q) = %v, want %v", tt.rule, tt.in, got, tt.want)
		}
	}
}

func TestCoreRulesYieldToTheGrammar(t *testing.T) {
	g := mustParse(t, "DIGIT = \"x\"\n")
	if ok, _ := g.Match("DIGIT", "x"); !ok {
		t.Error("the grammar's own DIGIT was replaced by the core rule")
	}
	if !g.Has("digit") || !g.Has("HEXDIG") {
		t.Error("Has does not fold case or lacks a core rule")
	}
	if rules := g.Rules(); len(rules) == 0 || !strings.Contains(strings.Join(rules, " "), "DIGIT") {
		t.Errorf("Rules() = %v", rules)
	}
}

func TestLeftRecursionMatchesNothing(t *testing.T) {
	g := mustParse(t, "list = list \",\" \"a\" / \"a\"\n")
	if ok, err := g.Match("list", "a"); err != nil || !ok {
		t.Errorf("Match(list, a) = %v, %v", ok, err)
	}
}

func TestMatchUndefined(t *testing.T) {
	g := mustParse(t, "a = b\n")
	if _, err := g.Match("a", "x"); err == nil {
		t.Error("a reference to an undefined rule matched")
	}
	if _, err := g.Match("nope", "x"); err == nil {
		t.Error("an undefined rule matched")
	}
	for _, src := range []string{"a = b / c\n", "a = \"x\" b\n", "a = *b\n"} {
		g := mustParse(t, src+"c = \"y\"\n")
		if _, err := g.Match("a", "x"); err == nil {
			t.Errorf("%q: a reference to an undefined rule matched", src)
		}
	}
}

func TestParseErrors(t *testing.T) {
	for name, src := range map[string]string{
		"continuation first": "  \"a\"\n",
		"no equals":          "rule \"a\"\n",
		"bad name":           "1rule = \"a\"\n",
		"unterminated":       "a = \"x\n",
		"incremental first":  "a =/ \"x\"\n",
		"twice":              "a = \"x\"\na = \"y\"\n",
		"prose":              "a = <prose>\n",
		"empty alternative":  "a = \"x\" / \n",
		"empty group":        "a = ()\n",
		"unclosed group":     "a = (\"x\"\n",
		"unclosed option":    "a = [\"x\"\n",
		"stray":              "a = \"x\" )\n",
		"unknown percent":    "a = %q41\n",
		"short percent":      "a = %x\n",
		"no number":          "a = %xZZ\n",
		"bad range":          "a = %x42-41\n",
		"dangling dot":       "a = %x41.\n",
		"bare %s":            "a = %sx\n",
		"bad character":      "a = !\n",
		"missing element":    "a = 2*\n",
	} {
		if _, err := Parse(src); err == nil {
			t.Errorf("%s: Parse(%q) succeeded", name, src)
		}
	}
}

func TestGenerate(t *testing.T) {
	g := mustParse(t, `top = 1*3word *(" " word) [%s"Z"] range
word = "ab" / %x61-7A / DIGIT
range = %x41-10FFFF
`)
	gen := NewGenerator(g, 7)
	again := NewGenerator(g, 7)
	for range 500 {
		s, err := gen.Generate("top")
		if err != nil {
			t.Fatal(err)
		}
		if ok, _ := g.Match("top", s); !ok {
			t.Fatalf("generated %q does not match", s)
		}
		if s2, _ := again.Generate("top"); s2 != s {
			t.Fatalf("the same seed generated %q then %q", s, s2)
		}
	}
}

func TestGenerateOverrideAndExclude(t *testing.T) {
	g := mustParse(t, "top = word \" \" letter\nword = 1*ALPHA\nletter = %x61-63\n")
	gen := NewGenerator(g, 1)
	gen.Override = map[string][]string{"WORD": {"JAN"}}
	gen.Exclude = func(r rune) bool { return r == 'a' || r == 'c' }
	gen.MaxExtra = 1
	overridden := 0
	for range 200 {
		s, err := gen.Generate("top")
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasSuffix(s, " b") {
			t.Fatalf("%q: an excluded letter was generated", s)
		}
		if strings.HasPrefix(s, "JAN ") {
			overridden++
		}
	}
	if overridden < 100 || overridden == 200 {
		t.Errorf("override used %d times in 200", overridden)
	}
}

func TestGenerateErrors(t *testing.T) {
	g := mustParse(t, "a = b\nloop = \"x\" loop\nnone = %x41-42\n")
	gen := NewGenerator(g, 1)
	if _, err := gen.Generate("a"); err == nil {
		t.Error("generated from an undefined rule")
	}
	if _, err := gen.Generate("loop"); err == nil {
		t.Error("an endless recursion generated")
	}
	gen.Exclude = func(rune) bool { return true }
	if _, err := gen.Generate("none"); err == nil {
		t.Error("generated an excluded character")
	}
	g = mustParse(t, "s = \"x\" b\nr = 1*b\n")
	for _, rule := range []string{"s", "r"} {
		if _, err := NewGenerator(g, 1).Generate(rule); err == nil {
			t.Errorf("%s: generated through an undefined rule", rule)
		}
	}
}

func TestMutate(t *testing.T) {
	gen := NewGenerator(mustParse(t, "a = \"x\"\n"), 3)
	changed := 0
	for range 200 {
		if gen.Mutate("Ab1 Cd") != "Ab1 Cd" {
			changed++
		}
	}
	if changed < 150 {
		t.Errorf("Mutate changed its input %d times in 200", changed)
	}
	if gen.Mutate("") == "" {
		t.Error("Mutate left the empty string empty")
	}
}
