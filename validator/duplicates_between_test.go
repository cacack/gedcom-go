package validator

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
	"github.com/cacack/gedcom-go/v3/merge"
)

// betweenPerson builds an individual for the cross-document tests: a given
// name and surname, a sex, and an exact birth year (0 for none).
func betweenPerson(xref, given, surname, sex string, birthYear int) *gedcom.Individual {
	ind := &gedcom.Individual{
		XRef:  xref,
		Names: []*gedcom.PersonalName{{Given: given, Surname: surname}},
		Sex:   sex,
	}
	if birthYear > 0 {
		ind.Events = []*gedcom.Event{{Type: gedcom.EventBirth, ParsedDate: makeYearDate(birthYear)}}
	}
	return ind
}

// pairKey renders a pair as "xref1|xref2" for order-independent comparison.
func pairKey(p DuplicatePair) string {
	return p.Individual1.XRef + "|" + p.Individual2.XRef
}

func sortedPairKeys(pairs []DuplicatePair) []string {
	keys := make([]string, 0, len(pairs))
	for _, p := range pairs {
		keys = append(keys, pairKey(p))
	}
	slices.Sort(keys)
	return keys
}

// TestFindDuplicatesBetween_OnlyCrossDocumentPairs: a duplicate within one
// document is not reported, and every reported pair takes Individual1 from the
// first document and Individual2 from the second.
func TestFindDuplicatesBetween_OnlyCrossDocumentPairs(t *testing.T) {
	a := makeDocument([]*gedcom.Individual{
		betweenPerson("@A1@", "John", "Example", "M", 1850),
		betweenPerson("@A2@", "John", "Example", "M", 1850), // duplicate within a
		betweenPerson("@A3@", "Mary", "Sample", "F", 1855),
	}, nil)
	b := makeDocument([]*gedcom.Individual{
		betweenPerson("@B1@", "John", "Example", "M", 1851),
		betweenPerson("@B2@", "Mary", "Sample", "F", 1855),
		betweenPerson("@B3@", "Mary", "Sample", "F", 1855), // duplicate within b
		betweenPerson("@B4@", "Ann", "Other", "F", 1860),   // surname only in b
	}, nil)

	d := NewDuplicateDetector(nil)
	pairs := d.FindDuplicatesBetween(a, b)

	want := []string{"@A1@|@B1@", "@A2@|@B1@", "@A3@|@B2@", "@A3@|@B3@"}
	if got := sortedPairKeys(pairs); !slices.Equal(got, want) {
		t.Fatalf("pairs = %v, want %v", got, want)
	}
	for _, p := range pairs {
		if a.GetIndividual(p.Individual1.XRef) != p.Individual1 {
			t.Errorf("Individual1 %s is not a's record", p.Individual1.XRef)
		}
		if b.GetIndividual(p.Individual2.XRef) != p.Individual2 {
			t.Errorf("Individual2 %s is not b's record", p.Individual2.XRef)
		}
	}

	// Swapping the arguments swaps the sides and keeps the scores.
	swapped := d.FindDuplicatesBetween(b, a)
	if len(swapped) != len(pairs) {
		t.Fatalf("swapped run found %d pairs, want %d", len(swapped), len(pairs))
	}
	conf := map[string]float64{}
	for _, p := range pairs {
		conf[pairKey(p)] = p.Confidence
	}
	for _, p := range swapped {
		key := p.Individual2.XRef + "|" + p.Individual1.XRef
		if c, ok := conf[key]; !ok || c != p.Confidence {
			t.Errorf("swapped pair %s confidence %v, want %v (present %v)", key, p.Confidence, c, ok)
		}
	}
}

// TestFindDuplicatesBetween_CollidingXRefs: both documents use @I1@ and @I2@
// for different people. Pairs must join the right records, never merge or
// look one document's XRef up in the other.
func TestFindDuplicatesBetween_CollidingXRefs(t *testing.T) {
	aJohn := betweenPerson("@I1@", "John", "Example", "M", 1850)
	aMary := betweenPerson("@I2@", "Mary", "Sample", "F", 1855)
	bMary := betweenPerson("@I1@", "Mary", "Sample", "F", 1855)
	bJohn := betweenPerson("@I2@", "John", "Example", "M", 1850)
	a := makeDocument([]*gedcom.Individual{aJohn, aMary}, nil)
	b := makeDocument([]*gedcom.Individual{bMary, bJohn}, nil)

	pairs := NewDuplicateDetector(nil).FindDuplicatesBetween(a, b)
	if len(pairs) != 2 {
		t.Fatalf("got %d pairs, want 2: %v", len(pairs), sortedPairKeys(pairs))
	}
	got := map[*gedcom.Individual]*gedcom.Individual{}
	for _, p := range pairs {
		got[p.Individual1] = p.Individual2
	}
	if got[aJohn] != bJohn {
		t.Errorf("a's @I1@ (John) paired with %v, want b's @I2@ (John)", got[aJohn])
	}
	if got[aMary] != bMary {
		t.Errorf("a's @I2@ (Mary) paired with %v, want b's @I1@ (Mary)", got[aMary])
	}

	// Same XRef, different people: no pair, however the XRefs line up.
	c := makeDocument([]*gedcom.Individual{betweenPerson("@I1@", "Peter", "Example", "M", 1850)}, nil)
	if pairs := NewDuplicateDetector(nil).FindDuplicatesBetween(a, c); len(pairs) != 0 {
		t.Errorf("got %v, want no pairs for different people sharing @I1@", sortedPairKeys(pairs))
	}
}

// TestFindDuplicatesBetween_MatchesCombinedRun pins the scoring contract: the
// pairs and confidences are exactly the cross-document pairs FindDuplicates
// reports on the two documents combined, under every config tried.
func TestFindDuplicatesBetween_MatchesCombinedRun(t *testing.T) {
	mk := func(prefix string, people [][4]any) *gedcom.Document {
		inds := make([]*gedcom.Individual, 0, len(people))
		for i, p := range people {
			inds = append(inds, betweenPerson(fmt.Sprintf("@%s%d@", prefix, i+1),
				p[0].(string), p[1].(string), p[2].(string), p[3].(int)))
		}
		return makeDocument(inds, nil)
	}
	// Both documents use the I prefix, so every XRef collides and Combine
	// has to rename b's.
	a := mk("I", [][4]any{
		{"John", "Example", "M", 1850}, {"Johann", "Example", "M", 1850},
		{"Mary", "Sample", "F", 1855}, {"Mary", "Sample", "", 0},
		{"Peter", "Example", "M", 1820}, {"Anne", "Éxample", "F", 1870},
		{"William", "Test", "M", 1890},
	})
	b := mk("I", [][4]any{
		{"Jon", "Example", "M", 1851}, {"John", "Example", "M", 1850},
		{"Mary", "Sample", "F", 1856}, {"Marie", "Sample", "F", 0},
		{"Peter", "Example", "F", 1820}, {"Anne", "Example", "", 1870},
		{"Willem", "Test", "U", 1891},
	})

	combined, report, err := merge.Combine(a, b, merge.CombineOptions{
		CollisionStrategy: merge.PrefixDoc2, Prefix: "B"})
	if err != nil {
		t.Fatal(err)
	}
	fromB := map[string]string{} // combined XRef -> b's XRef
	for _, ind := range b.Individuals() {
		x := ind.XRef
		if renamed, ok := report.RemappedXRefs[x]; ok {
			x = renamed
		}
		fromB[x] = ind.XRef
	}

	configs := map[string]*DuplicateConfig{
		"default":        nil,
		"zero threshold": {MinNameSimilarity: ZeroThreshold, MinConfidence: ZeroThreshold},
		"strict":         {MinConfidence: 0.9, RequireBirthDate: true},
		"no normalize":   {DisableNameNormalization: true, MaxBirthYearDiff: 40},
	}
	for name, cfg := range configs {
		t.Run(name, func(t *testing.T) {
			d := NewDuplicateDetector(cfg)
			want := map[string]float64{}
			for _, p := range d.FindDuplicates(combined) {
				x1, in1 := fromB[p.Individual1.XRef]
				x2, in2 := fromB[p.Individual2.XRef]
				switch {
				case !in1 && in2:
					want[p.Individual1.XRef+"|"+x2] = p.Confidence
				case in1 && !in2:
					want[p.Individual2.XRef+"|"+x1] = p.Confidence
				}
			}
			got := map[string]float64{}
			for _, p := range d.FindDuplicatesBetween(a, b) {
				got[pairKey(p)] = p.Confidence
			}
			if len(want) == 0 {
				t.Fatal("combined run found no cross-document pairs; the fixture tests nothing")
			}
			if fmt.Sprint(got) != fmt.Sprint(want) {
				t.Errorf("between = %v\ncombined = %v", got, want)
			}
		})
	}
}

// TestFindDuplicatesBetweenReport_CapOnCombinedGroup: MaxGroupSize applies to
// a surname's individuals in both documents together, as on the combined
// documents; an oversized group found in only one document cannot yield a
// cross pair and is not reported.
func TestFindDuplicatesBetweenReport_CapOnCombinedGroup(t *testing.T) {
	a := makeDocument([]*gedcom.Individual{
		betweenPerson("@A1@", "John", "Example", "M", 1850),
		betweenPerson("@A2@", "P2", "Example", "M", 0),
		betweenPerson("@A3@", "Mary", "Sample", "F", 1855),
		betweenPerson("@A4@", "P4", "Lonely", "M", 0),
		betweenPerson("@A5@", "P5", "Lonely", "M", 0),
		betweenPerson("@A6@", "P6", "Lonely", "M", 0),
		betweenPerson("@A7@", "P7", "Lonely", "M", 0),
	}, nil)
	b := makeDocument([]*gedcom.Individual{
		betweenPerson("@B1@", "John", "Example", "M", 1850),
		betweenPerson("@B2@", "P9", "Example", "M", 0),
		betweenPerson("@B3@", "Mary", "Sample", "F", 1855),
	}, nil)

	// Example has 2+2 = 4 > 3 and is skipped; Sample has 1+1 and is compared;
	// Lonely has 4 but only in a.
	report := NewDuplicateDetector(&DuplicateConfig{MaxGroupSize: 3}).FindDuplicatesBetweenReport(a, b)
	if got, want := sortedPairKeys(report.Pairs), []string{"@A3@|@B3@"}; !slices.Equal(got, want) {
		t.Errorf("pairs = %v, want %v", got, want)
	}
	if len(report.LimitIssues) != 1 {
		t.Fatalf("got %d limit issues, want 1", len(report.LimitIssues))
	}
	issue := report.LimitIssues[0]
	if issue.Code != CodeDuplicateDetectionLimited || issue.Severity != SeverityWarning {
		t.Errorf("issue = %s/%v, want %s/Warning", issue.Code, issue.Severity, CodeDuplicateDetectionLimited)
	}
	for key, want := range map[string]string{
		"skipped_groups": "1", "skipped_individuals": "4", "largest_group": "4",
		"max_group_size": "3", "surnames": "example",
	} {
		if got := issue.Details[key]; got != want {
			t.Errorf("Details[%q] = %q, want %q", key, got, want)
		}
	}

	// At the cap the group is compared in full.
	report = NewDuplicateDetector(&DuplicateConfig{MaxGroupSize: 4}).FindDuplicatesBetweenReport(a, b)
	if got, want := sortedPairKeys(report.Pairs), []string{"@A1@|@B1@", "@A3@|@B3@"}; !slices.Equal(got, want) {
		t.Errorf("at the cap: pairs = %v, want %v", got, want)
	}
	if len(report.LimitIssues) != 0 {
		t.Errorf("at the cap: got %d limit issues, want 0", len(report.LimitIssues))
	}
	if got := NewDuplicateDetector(&DuplicateConfig{MaxGroupSize: 3}).FindDuplicatesBetween(a, b); len(got) != 1 {
		t.Errorf("FindDuplicatesBetween returned %d pairs, want the report's 1", len(got))
	}
}

// TestFindDuplicatesBetweenReport_TruncatedNames: given names cut to
// MaxNameRunes are counted on the limit issue from both sides.
func TestFindDuplicatesBetweenReport_TruncatedNames(t *testing.T) {
	long := strings.Repeat("x", 10)
	a := makeDocument([]*gedcom.Individual{betweenPerson("@I1@", long, "Example", "M", 1850)}, nil)
	b := makeDocument([]*gedcom.Individual{betweenPerson("@I1@", long+"y", "Example", "M", 1850)}, nil)

	report := NewDuplicateDetector(&DuplicateConfig{MaxNameRunes: 5}).FindDuplicatesBetweenReport(a, b)
	if len(report.Pairs) != 1 {
		t.Errorf("got %d pairs, want 1", len(report.Pairs))
	}
	if len(report.LimitIssues) != 1 {
		t.Fatalf("got %d limit issues, want 1", len(report.LimitIssues))
	}
	if got := report.LimitIssues[0].Details["truncated_names"]; got != "2" {
		t.Errorf("truncated_names = %q, want 2", got)
	}
}

func TestFindDuplicatesBetween_NilOrEmpty(t *testing.T) {
	d := NewDuplicateDetector(nil)
	doc := makeDocument([]*gedcom.Individual{betweenPerson("@I1@", "John", "Example", "M", 1850)}, nil)
	empty := makeDocument(nil, nil)

	for name, docs := range map[string][2]*gedcom.Document{
		"nil a": {nil, doc}, "nil b": {doc, nil}, "empty b": {doc, empty},
	} {
		report := d.FindDuplicatesBetweenReport(docs[0], docs[1])
		if len(report.Pairs) != 0 || len(report.LimitIssues) != 0 {
			t.Errorf("%s: got %+v, want an empty report", name, report)
		}
	}
}
