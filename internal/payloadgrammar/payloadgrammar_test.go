package payloadgrammar

import (
	"crypto/sha1" // #nosec G505 -- git's blob hash, not a security use
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/internal/corpora"
)

// TestVendoredGrammarIsPinned fails when grammar.abnf differs by a byte from
// the upstream file SOURCE pins: its git blob hash is the one SOURCE records,
// which is the hash GitHub reports for the file at the pinned commit.
func TestVendoredGrammarIsPinned(t *testing.T) {
	root, err := corpora.RepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(Path)))
	if err != nil {
		t.Fatal(err)
	}
	h := sha1.New() // #nosec G401 -- git's blob hash, not a security use
	fmt.Fprintf(h, "blob %d\x00", len(data))
	h.Write(data)
	got := hex.EncodeToString(h.Sum(nil))

	source, err := os.ReadFile(filepath.Join(root, "testdata", "spec", "gedcom-7.0", "SOURCE"))
	if err != nil {
		t.Fatal(err)
	}
	want := ""
	for _, line := range strings.Split(string(source), "\n") {
		if key, value, ok := strings.Cut(line, ":"); ok && key == "grammar.abnf blob" {
			want = strings.TrimSpace(value)
		}
	}
	if want == "" {
		t.Fatal("SOURCE records no \"grammar.abnf blob\"")
	}
	if got != want {
		t.Errorf("grammar.abnf has git blob hash %s; SOURCE pins %s", got, want)
	}
}

// TestRequiredRules fails when the vendored grammar lacks a rule the oracle
// depends on, and checks that a missing or renamed rule is caught.
func TestRequiredRules(t *testing.T) {
	o := Load(t)
	if missing := Missing(o.Grammar()); len(missing) > 0 {
		t.Fatalf("grammar.abnf has no rule %s", strings.Join(missing, ", "))
	}
	root, err := corpora.RepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	src, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(Path)))
	if err != nil {
		t.Fatal(err)
	}
	for _, rename := range [][2]string{
		{"\nDateValue   =", "\nDateValue2  ="},
		{"\nLatitude =", "\nLatitude-Value ="},
		{"\nbanned =", "\nforbidden ="},
	} {
		if !strings.Contains(string(src), rename[0]) {
			t.Fatalf("grammar.abnf has no %q to rename", rename[0])
		}
		_, err := New(strings.Replace(string(src), rename[0], rename[1], 1))
		if err == nil || !strings.Contains(err.Error(), strings.Fields(rename[0])[0]) {
			t.Errorf("renaming %q: New error = %v, want one naming the rule", rename[0], err)
		}
	}
	if _, err := New("a = <prose>\n"); err == nil {
		t.Error("New accepted a grammar it cannot parse")
	}
}

func TestMatchPanicsOnAnUnlistedRule(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("Match of a rule not in Required did not panic")
		}
	}()
	Load(t).Match("Enum", "X")
}

func TestAccepts(t *testing.T) {
	o := Load(t)
	tests := []struct {
		t    Type
		in   string
		want bool
	}{
		{DateValue, "", true},
		{DateValue, "1 JAN 2000", true},
		{DateValue, "29 FEB 1900", true},
		{DateValue, "30 FEB 2000", false},
		{DateValue, "0 JAN 2000", false},
		{DateValue, "2000 BCE", true},
		{DateValue, "JULIAN 44 BCE", true},
		{DateValue, "HEBREW 5000 BCE", false},
		{DateValue, "FRENCH_R 6 COMP 5", true},
		{DateValue, "FRENCH_R 7 COMP 5", false},
		{DateValue, "FRENCH_R 1 JAN 5", false},
		{DateValue, "1 TSH 5000", false},
		{DateValue, "HEBREW 30 ELL 5000", false},
		{DateValue, "JAN 0", false},
		{DateValue, "1 _MON 2000", true},
		{DateValue, "2000 _EP", true},
		{DateValue, "_CAL 99 MAR 0", true},
		{DateValue, "BET HEBREW 1 TSH 5000 AND 1 JAN 2000", true},
		{DateValue, "FROM 1 JAN 2000 TO 31 FEB 2000", false},
		{DateValue, "1 FOO 2000", false},
		{DateValue, "AND 1", false},
		{DateValue, "1 JAN AND", false},
		{DateValue, "TO TO 2000", false},
		{DateValue, "abt 2000", false},
		{DateExact, "31 DEC 2000", true},
		{DateExact, "1 VEND 2000", false},
		{DateExact, "1 JAN 0", false},
		{Time, "23:59:59.999Z", true},
		{Time, "24:00", false},
		{Age, "> 1y 2m 3w 4d", true},
		{Age, ">1y", false},
		{PersonalName, "John /Doe/ Jr.", true},
		{PersonalName, "John /Doe", false},
		{Latitude, "N90", true},
		{Latitude, "N90.5", false},
		{Latitude, "n18.150944", true},
		{Longitude, "W180", true},
		{Longitude, "W180.1", false},
		{Xref, "@I1@", true},
		{Xref, "@VOID@", false},
		{Xref, "@i1@", false},
	}
	for _, tt := range tests {
		if got := o.Accepts(tt.t, tt.in); got != tt.want {
			t.Errorf("Accepts(%s, %q) = %v, want %v", tt.t, tt.in, got, tt.want)
		}
	}
}

func TestCorpusHasEveryType(t *testing.T) {
	corpus := Corpus(t)
	for _, typ := range Types {
		if len(corpus[typ]) == 0 {
			t.Errorf("no %s value in %s", typ, strings.Join(CorpusDirs, ", "))
		}
	}
	if !slices.Contains(corpus[DateExact], "1 JAN 2020") || slices.Contains(corpus[DateValue], "1 JAN 2020 ") {
		t.Errorf("DateExact corpus = %q", corpus[DateExact])
	}
}

func TestCorpusReadsOnly70Files(t *testing.T) {
	c := &collector{values: map[Type][]string{}, seen: map[Type]map[string]bool{}}
	c.file([]byte("0 HEAD\n1 GEDC\n2 VERS 5.5.1\n0 @I1@ INDI\n0 TRLR\n"))
	c.file([]byte("0 @I1@ INDI\n"))
	if len(c.values) != 0 {
		t.Errorf("read %v from files that do not declare 7.0", c.values)
	}
	if _, _, _, _, ok := splitLine("x HEAD"); ok {
		t.Error("splitLine read a line with no level")
	}
}

func TestInputsAreDeterministic(t *testing.T) {
	for _, typ := range Types {
		a, b := Inputs(t, typ), Inputs(t, typ)
		if !slices.Equal(a, b) {
			t.Errorf("%s: two calls gave different inputs", typ)
		}
		// Generated strings repeat, the more so for a small language
		// like Age's, but most are distinct.
		if len(a) < Generated/2 {
			t.Errorf("%s: %d distinct inputs, want at least %d", typ, len(a), Generated/2)
		}
	}
}

// recorder is a testing.TB that records failures instead of failing.
type recorder struct {
	testing.TB
	errors []string
}

func (r *recorder) Helper() {}

func (r *recorder) Errorf(format string, args ...any) {
	r.errors = append(r.errors, fmt.Sprintf(format, args...))
}

func (r *recorder) Logf(string, ...any) {}

func TestCompareFailsOnAnUnlistedDisagreement(t *testing.T) {
	r := &recorder{TB: t}
	Compare(r, Time, func(string) bool { return true }, func(string, bool) []string { return nil })
	if len(r.errors) != maxReported+1 || !strings.Contains(r.errors[maxReported], "more disagreements") {
		t.Errorf("Compare reported %d failures: %q", len(r.errors), r.errors)
	}

	r = &recorder{TB: t}
	Compare(r, Time, func(string) bool { return true }, func(string, bool) []string { return []string{"SI-001"} })
	if len(r.errors) != 0 {
		t.Errorf("Compare failed on disagreements a row allows: %q", r.errors)
	}

	r = &recorder{TB: t}
	Check(r, Time, "25:00", func(string) bool { return true }, func(string, bool) []string { return nil })
	Check(r, Time, "23:00", func(string) bool { return true }, func(string, bool) []string { return nil })
	if len(r.errors) != 1 {
		t.Errorf("Check reported %q, want one failure", r.errors)
	}
}
