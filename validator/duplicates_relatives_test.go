package validator

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// decodeLines decodes a GEDCOM document from its lines, header and trailer
// added.
func decodeLines(t *testing.T, lines ...string) *gedcom.Document {
	t.Helper()
	data := "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n1 CHAR UTF-8\n" + strings.Join(lines, "\n") + "\n0 TRLR\n"
	doc, err := decoder.Decode(strings.NewReader(data))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	return doc
}

// marriedFamily returns the lines of a wife named wifeName, her husband John
// /Smith/ and their children Ann and Thomas /Smith/, with XRefs prefixed by p.
func marriedFamily(p, wifeName string) []string {
	return []string{
		"0 @" + p + "W@ INDI", "1 NAME " + wifeName, "1 SEX F", "1 FAMS @" + p + "F@",
		"0 @" + p + "H@ INDI", "1 NAME John /Smith/", "1 SEX M", "1 FAMS @" + p + "F@",
		"0 @" + p + "C1@ INDI", "1 NAME Ann /Smith/", "1 FAMC @" + p + "F@",
		"0 @" + p + "C2@ INDI", "1 NAME Thomas /Smith/", "1 FAMC @" + p + "F@",
		"0 @" + p + "F@ FAM", "1 HUSB @" + p + "H@", "1 WIFE @" + p + "W@",
		"1 CHIL @" + p + "C1@", "1 CHIL @" + p + "C2@",
	}
}

// findPair returns the pair of the two XRefs, in either order.
func findPair(pairs []DuplicatePair, x1, x2 string) (DuplicatePair, bool) {
	for _, p := range pairs {
		a, b := p.Individual1.XRef, p.Individual2.XRef
		if (a == x1 && b == x2) || (a == x2 && b == x1) {
			return p, true
		}
	}
	return DuplicatePair{}, false
}

var relativesConfig = &DuplicateConfig{CompareRelatives: true}

// wantMarriedMaidenReasons are the reasons of the #932 pair: no surname
// match, so no "exact surname match".
var wantMarriedMaidenReasons = []string{
	"exact given name match", "same sex", "same spouse name", "2 shared child names",
}

// TestFindDuplicates_CompareRelatives_MarriedAndMaidenName: the #932 fixture,
// one woman under her married name and her maiden name, each with the same
// husband and two children, is reported at the default MinConfidence only
// when CompareRelatives is on.
func TestFindDuplicates_CompareRelatives_MarriedAndMaidenName(t *testing.T) {
	doc := decodeLines(t, slices.Concat(marriedFamily("A", "Mary /Smith/"), marriedFamily("B", "Mary /Jones/"))...)

	if _, ok := findPair(NewDuplicateDetector(nil).FindDuplicates(doc), "@AW@", "@BW@"); ok {
		t.Error("default config reported the married/maiden pair; relatives must be opt-in")
	}

	pairs := NewDuplicateDetector(relativesConfig).FindDuplicates(doc)
	pair, ok := findPair(pairs, "@AW@", "@BW@")
	if !ok {
		t.Fatalf("married/maiden pair not reported; got %d pairs", len(pairs))
	}
	assertConfidence(t, pair, 0.8)
	if !slices.Equal(pair.MatchReasons, wantMarriedMaidenReasons) {
		t.Errorf("reasons = %v, want %v", pair.MatchReasons, wantMarriedMaidenReasons)
	}

	// The husbands share a surname and two children; the option adds the
	// children to their score, capped at 1.0.
	husbands, ok := findPair(pairs, "@AH@", "@BH@")
	if !ok {
		t.Fatal("husband pair not reported")
	}
	assertConfidence(t, husbands, 0.9)
	if got := husbands.MatchReasons[len(husbands.MatchReasons)-1]; got != "2 shared child names" {
		t.Errorf("last husband reason = %q, want %q", got, "2 shared child names")
	}
}

// TestFindDuplicatesBetween_CompareRelatives: the same fixture split across
// two files, as gedcom-cli's duplicates command sees it.
func TestFindDuplicatesBetween_CompareRelatives(t *testing.T) {
	a := decodeLines(t, marriedFamily("A", "Mary /Smith/")...)
	b := decodeLines(t, marriedFamily("B", "Mary /Jones/")...)

	if _, ok := findPair(NewDuplicateDetector(nil).FindDuplicatesBetween(a, b), "@AW@", "@BW@"); ok {
		t.Error("default config reported the married/maiden pair")
	}

	pairs := NewDuplicateDetector(relativesConfig).FindDuplicatesBetween(a, b)
	pair, ok := findPair(pairs, "@AW@", "@BW@")
	if !ok {
		t.Fatalf("married/maiden pair not reported; got %d pairs", len(pairs))
	}
	if pair.Individual1.XRef != "@AW@" {
		t.Errorf("Individual1 = %s, want the individual from a", pair.Individual1.XRef)
	}
	assertConfidence(t, pair, 0.8)
	if !slices.Equal(pair.MatchReasons, wantMarriedMaidenReasons) {
		t.Errorf("reasons = %v, want %v", pair.MatchReasons, wantMarriedMaidenReasons)
	}

	// Each pair is reported once: wives through the spouse pass, husbands
	// and children through the surname pass.
	if len(pairs) != 4 {
		t.Errorf("got %d pairs, want 4 (wives, husbands, two children)", len(pairs))
	}
}

// TestFindDuplicates_CompareRelatives_Gates: relatives let a pair past the
// surname gate only through a shared spouse's name, and no other gate.
func TestFindDuplicates_CompareRelatives_Gates(t *testing.T) {
	tests := []struct {
		name     string
		lines    []string
		wantPair bool
		want     float64
	}{
		{
			name:  "given names too far apart",
			lines: slices.Concat(marriedFamily("A", "Mary /Smith/"), marriedFamily("B", "Margaret /Jones/")),
		},
		{
			name: "known sexes differ",
			lines: slices.Concat(marriedFamily("A", "Mary /Smith/"),
				replaceLine(marriedFamily("B", "Mary /Jones/"), "1 SEX F", "1 SEX M")),
		},
		{
			name: "shared children but no shared spouse",
			lines: slices.Concat(marriedFamily("A", "Mary /Smith/"),
				replaceLine(marriedFamily("B", "Mary /Jones/"), "1 NAME John /Smith/", "1 NAME Peter /Smith/")),
		},
		{
			name: "spouse without a given name says nothing",
			lines: slices.Concat(
				replaceLine(marriedFamily("A", "Mary /Smith/"), "1 NAME John /Smith/", "1 NAME /Smith/"),
				replaceLine(marriedFamily("B", "Mary /Jones/"), "1 NAME John /Smith/", "1 NAME /Smith/")),
		},
		{
			name: "same spouse alone",
			lines: slices.Concat(
				replaceLine(marriedFamily("A", "Mary /Smith/"), "1 CHIL @AC2@", ""),
				replaceLine(replaceLine(marriedFamily("B", "Mary /Jones/"), "1 CHIL @BC1@", ""), "1 CHIL @BC2@", "")),
			wantPair: false, // 0.3 + 0.1 + 0.2 = 0.6, below 0.7
		},
		{
			name: "same spouse and one child",
			lines: slices.Concat(
				replaceLine(marriedFamily("A", "Mary /Smith/"), "1 CHIL @AC2@", ""),
				replaceLine(marriedFamily("B", "Mary /Jones/"), "1 CHIL @BC2@", "")),
			wantPair: true, want: 0.7,
		},
		{
			name:     "no surname on one side",
			lines:    slices.Concat(marriedFamily("A", "Mary /Smith/"), marriedFamily("B", "Mary")),
			wantPair: true, want: 0.8,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := decodeLines(t, dropEmpty(tt.lines)...)
			pair, ok := findPair(NewDuplicateDetector(relativesConfig).FindDuplicates(doc), "@AW@", "@BW@")
			if ok != tt.wantPair {
				t.Fatalf("pair reported = %v, want %v (reasons %v)", ok, tt.wantPair, pair.MatchReasons)
			}
			if ok {
				assertConfidence(t, pair, tt.want)
			}
		})
	}
}

// TestFindDuplicates_CompareRelatives_ScoreCapped: a pair agreeing on
// everything, relatives included, scores the documented maximum 1.0.
func TestFindDuplicates_CompareRelatives_ScoreCapped(t *testing.T) {
	doc := decodeLines(t, slices.Concat(marriedFamily("A", "Mary /Smith/"), marriedFamily("B", "Mary /Smith/"))...)
	for _, xref := range []string{"@AW@", "@BW@"} {
		ind := doc.GetIndividual(xref)
		ind.Events = append(ind.Events, lifeEvent(t, gedcom.EventBirth, "3 MAR 1850", "Springfield"))
	}

	pair, ok := findPair(NewDuplicateDetector(relativesConfig).FindDuplicates(doc), "@AW@", "@BW@")
	if !ok {
		t.Fatal("pair not reported")
	}
	assertConfidence(t, pair, 1.0)
}

// TestFindDuplicates_CompareRelatives_PairComparedOnce: a pair sharing two
// spouses' names is in two spouse groups but reported once, and a pair the
// surname pass reports is not reported again by the spouse pass.
func TestFindDuplicates_CompareRelatives_PairComparedOnce(t *testing.T) {
	second := func(p string) []string {
		return []string{
			"0 @" + p + "H2@ INDI", "1 NAME Peter /Brown/", "1 SEX M", "1 FAMS @" + p + "F2@",
			"0 @" + p + "F2@ FAM", "1 HUSB @" + p + "H2@", "1 WIFE @" + p + "W@",
		}
	}
	for _, names := range [][2]string{{"Mary /Smith/", "Mary /Jones/"}, {"Mary /Smith/", "Mary /Smith/"}} {
		lines := slices.Concat(
			marriedFamily("A", names[0]), second("A"),
			marriedFamily("B", names[1]), second("B"),
		)
		lines = replaceLine(lines, "1 FAMS @AF@", "1 FAMS @AF@\n1 FAMS @AF2@")
		lines = replaceLine(lines, "1 FAMS @BF@", "1 FAMS @BF@\n1 FAMS @BF2@")
		doc := decodeLines(t, lines...)

		count := 0
		for _, p := range NewDuplicateDetector(relativesConfig).FindDuplicates(doc) {
			if _, ok := findPair([]DuplicatePair{p}, "@AW@", "@BW@"); ok {
				count++
			}
		}
		if count != 1 {
			t.Errorf("%v: pair reported %d times, want 1", names, count)
		}
	}
}

// TestFindDuplicates_CompareRelatives_Limits: a spouse-name group larger
// than MaxGroupSize is skipped whole and named on the notice, and relatives
// past maxRelativesRead links are counted there.
func TestFindDuplicates_CompareRelatives_Limits(t *testing.T) {
	t.Run("oversized spouse group", func(t *testing.T) {
		doc := decodeLines(t, slices.Concat(marriedFamily("A", "Mary /Smith/"), marriedFamily("B", "Mary /Jones/"))...)
		report := NewDuplicateDetector(&DuplicateConfig{CompareRelatives: true, MaxGroupSize: 1}).FindDuplicatesReport(doc)
		if len(report.LimitIssues) != 1 {
			t.Fatalf("got %d limit issues, want 1", len(report.LimitIssues))
		}
		if got := report.LimitIssues[0].Details["surnames"]; !strings.Contains(got, "spouse:john /smith/") {
			t.Errorf("surnames = %q, want it to name the spouse group", got)
		}
		if _, ok := findPair(report.Pairs, "@AW@", "@BW@"); ok {
			t.Error("pair from a skipped spouse group was reported")
		}

		a := decodeLines(t, marriedFamily("A", "Mary /Smith/")...)
		b := decodeLines(t, marriedFamily("B", "Mary /Jones/")...)
		between := NewDuplicateDetector(&DuplicateConfig{CompareRelatives: true, MaxGroupSize: 1}).FindDuplicatesBetweenReport(a, b)
		if len(between.LimitIssues) != 1 || !strings.Contains(between.LimitIssues[0].Details["surnames"], "spouse:john /smith/") {
			t.Errorf("between limit issues = %v, want the spouse group named", between.LimitIssues)
		}
	})

	t.Run("too many relatives", func(t *testing.T) {
		tests := []struct {
			name          string
			wantTruncated string
			family        func() []string
		}{
			// replaceLine extends both partners' FAMS lines.
			{name: "spouse links", wantTruncated: "2", family: func() []string {
				lines := marriedFamily("A", "Mary /Smith/")
				for i := range maxRelativesRead {
					lines = append(lines, fmt.Sprintf("0 @X%d@ FAM", i), "1 WIFE @AW@")
				}
				fams := "1 FAMS @AF@"
				for i := range maxRelativesRead {
					fams += fmt.Sprintf("\n1 FAMS @X%d@", i)
				}
				return replaceLine(lines, "1 FAMS @AF@", fams)
			}},
			// Both parents read the family's children.
			{name: "child links", wantTruncated: "2", family: func() []string {
				lines := marriedFamily("A", "Mary /Smith/")
				chil := "1 CHIL @AC2@"
				for i := range maxRelativesRead {
					lines = append(lines, fmt.Sprintf("0 @K%d@ INDI", i), fmt.Sprintf("1 NAME Kid%d /Smith/", i))
					chil += fmt.Sprintf("\n1 CHIL @K%d@", i)
				}
				return replaceLine(lines, "1 CHIL @AC2@", chil)
			}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				doc := decodeLines(t, slices.Concat(tt.family(), marriedFamily("B", "Mary /Jones/"))...)
				report := NewDuplicateDetector(relativesConfig).FindDuplicatesReport(doc)
				if len(report.LimitIssues) != 1 {
					t.Fatalf("got %d limit issues, want 1", len(report.LimitIssues))
				}
				issue := report.LimitIssues[0]
				if got := issue.Details["truncated_relatives"]; got != tt.wantTruncated {
					t.Errorf("truncated_relatives = %q, want %s (message %q)", got, tt.wantTruncated, issue.Message)
				}
				if !strings.Contains(issue.Message, "spouse and child links") {
					t.Errorf("message = %q, want it to describe the relatives cut", issue.Message)
				}
				if _, ok := findPair(report.Pairs, "@AW@", "@BW@"); !ok {
					t.Error("pair lost: the first links read still hold the spouse and children")
				}
			})
		}
	})
}

// TestFindDuplicates_CompareRelatives_ContextCanceled: the spouse pass stops
// with the surname pass when the context is done.
func TestFindDuplicates_CompareRelatives_ContextCanceled(t *testing.T) {
	doc := decodeLines(t, slices.Concat(marriedFamily("A", "Mary /Smith/"), marriedFamily("B", "Mary /Jones/"))...)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	report := NewDuplicateDetector(relativesConfig).findDuplicatesReport(ctx, doc)
	if len(report.Pairs) != 0 {
		t.Errorf("got %d pairs from a canceled run, want 0", len(report.Pairs))
	}
}

// replaceLine replaces every line equal to old with replacement, which may
// hold several lines or be empty.
func replaceLine(lines []string, old, replacement string) []string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if line == old {
			line = replacement
		}
		out = append(out, line)
	}
	return out
}

// dropEmpty drops the empty lines replaceLine leaves.
func dropEmpty(lines []string) []string {
	return slices.DeleteFunc(slices.Clone(lines), func(s string) bool { return s == "" })
}

// TestFindDuplicates_CompareRelatives_DanglingPointers: a FAMS, HUSB or CHIL
// pointing at nothing names no relative and does not panic.
func TestFindDuplicates_CompareRelatives_DanglingPointers(t *testing.T) {
	doc := decodeLines(t,
		"0 @AW@ INDI", "1 NAME Mary /Smith/", "1 FAMS @NONE@", "1 FAMS @AF@",
		"0 @BW@ INDI", "1 NAME Mary /Jones/", "1 FAMS @BF@",
		"0 @AF@ FAM", "1 HUSB @GONE@", "1 WIFE @AW@", "1 CHIL @GONE@",
		"0 @BF@ FAM", "1 HUSB @GONE@", "1 WIFE @BW@", "1 CHIL @GONE@",
	)
	if _, ok := findPair(NewDuplicateDetector(relativesConfig).FindDuplicates(doc), "@AW@", "@BW@"); ok {
		t.Error("pair reported through relatives that do not exist")
	}
}

// TestFindDuplicatesBetween_CompareRelatives_SharedIndividual: an individual
// both documents share, met in a spouse group, is not paired with itself.
func TestFindDuplicatesBetween_CompareRelatives_SharedIndividual(t *testing.T) {
	a := decodeLines(t, marriedFamily("A", "Mary /Smith/")...)
	b := &gedcom.Document{Header: a.Header, Records: slices.Clone(a.Records), XRefMap: a.XRefMap}

	for _, p := range NewDuplicateDetector(relativesConfig).FindDuplicatesBetween(a, b) {
		if p.Individual1 == p.Individual2 {
			t.Errorf("%s paired with itself", p.Individual1.XRef)
		}
	}
}

// linkedToLongNamedSpouse returns a document in which n individuals each
// link family @F@ maxRelativesRead times, and @F@'s husband has a given name
// of nameRunes runes: the shape on which deriving a relative's key per link
// cost the document's size times the length of the name.
func linkedToLongNamedSpouse(t testing.TB, n, nameRunes int) *gedcom.Document {
	t.Helper()
	fams := strings.Repeat("1 FAMS @F@\n", maxRelativesRead)
	var sb strings.Builder
	sb.WriteString("0 HEAD\n1 GEDC\n2 VERS 5.5.1\n1 CHAR UTF-8\n")
	sb.WriteString("0 @H@ INDI\n1 NAME " + strings.Repeat("a", nameRunes) + " /" + strings.Repeat("b", nameRunes) + "/\n")
	for i := range n {
		fmt.Fprintf(&sb, "0 @I%d@ INDI\n1 NAME Mary /Smith%d/\n%s", i, i, fams)
	}
	sb.WriteString("0 @F@ FAM\n1 HUSB @H@\n0 TRLR\n")
	doc, err := decoder.Decode(strings.NewReader(sb.String()))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	return doc
}

// TestFindDuplicates_CompareRelatives_RelativeKeyDerivedOnce: a relative
// named by many links is keyed once per document, its key cut to MaxNameRunes
// runes per part, and every individual shares that one key.
func TestFindDuplicates_CompareRelatives_RelativeKeyDerivedOnce(t *testing.T) {
	const n = 50
	doc := linkedToLongNamedSpouse(t, n, 10_000)
	d := NewDuplicateDetector(relativesConfig)
	var skipped skippedGroups
	b := d.newCandidateBuilder(doc, d.maxNameRunes(), &skipped)

	groups := b.spouseGroups(context.Background(), doc.Individuals())
	if len(b.relatives) != 1 {
		t.Errorf("relative keys derived = %d, want 1", len(b.relatives))
	}
	want := strings.Repeat("a", DefaultMaxNameRunes) + " /" + strings.Repeat("b", DefaultMaxNameRunes) + "/"
	if got := b.relatives["@H@"]; got != want {
		t.Errorf("key has %d bytes, want %d", len(got), len(want))
	}
	if len(groups) != 1 || len(groups[want]) != n {
		t.Errorf("groups = %d (group size %d), want 1 group of %d", len(groups), len(groups[want]), n)
	}

	// The candidate reuses the spouse keys the groups were built from.
	c := b.candidate(doc.GetIndividual("@I0@"))
	if !slices.Equal(c.spouses, []string{want}) {
		t.Errorf("candidate spouses = %v, want the one cut key", c.spouses)
	}
	if len(b.relatives) != 1 {
		t.Errorf("relative keys derived after candidate = %d, want 1", len(b.relatives))
	}

	// Sharing only a given name and a spouse scores 0.5, below the default
	// MinConfidence.
	if pairs := d.FindDuplicates(doc); len(pairs) != 0 {
		t.Errorf("got %d pairs, want 0", len(pairs))
	}
}

// TestFindDuplicates_CompareRelatives_SpouseGroupsCanceled: building the
// spouse groups stops once the context is done.
func TestFindDuplicates_CompareRelatives_SpouseGroupsCanceled(t *testing.T) {
	doc := linkedToLongNamedSpouse(t, 10, 10)
	d := NewDuplicateDetector(relativesConfig)
	var skipped skippedGroups
	b := d.newCandidateBuilder(doc, d.maxNameRunes(), &skipped)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if groups := b.spouseGroups(ctx, doc.Individuals()); len(groups) != 0 {
		t.Errorf("got %d groups from a canceled build, want 0", len(groups))
	}
}

// BenchmarkFindDuplicates_CompareRelatives_LongNamedSpouse: the crafted
// shape of 2,000 individuals each linking, 64 times, one family whose husband
// has a 100,000-rune given name and surname.
func BenchmarkFindDuplicates_CompareRelatives_LongNamedSpouse(b *testing.B) {
	doc := linkedToLongNamedSpouse(b, 2000, 100_000)
	d := NewDuplicateDetector(relativesConfig)
	b.ResetTimer()
	for b.Loop() {
		d.FindDuplicatesReport(doc)
	}
}
