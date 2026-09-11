package validator

import (
	"strconv"
	"testing"

	"github.com/cacack/gedcom-go/v2/gedcom"
)

// BenchmarkValidateMinimal benchmarks validating a minimal document
func BenchmarkValidateMinimal(b *testing.B) {
	doc := &gedcom.Document{
		Header: &gedcom.Header{Version: "5.5"},
		Records: []*gedcom.Record{
			{
				XRef: "@I1@",
				Type: gedcom.RecordTypeIndividual,
				Tags: []*gedcom.Tag{
					{Level: 1, Tag: "NAME", Value: "John /Doe/"},
				},
			},
		},
		XRefMap: map[string]*gedcom.Record{
			"@I1@": {XRef: "@I1@", Type: gedcom.RecordTypeIndividual},
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		v := New()
		_ = v.Validate(doc)
	}
}

// BenchmarkValidateSmall benchmarks validating a small document (10 individuals)
func BenchmarkValidateSmall(b *testing.B) {
	doc := generateValidDocument(10)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		v := New()
		_ = v.Validate(doc)
	}
}

// BenchmarkValidateMedium benchmarks validating a medium document (100 individuals)
func BenchmarkValidateMedium(b *testing.B) {
	doc := generateValidDocument(100)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		v := New()
		_ = v.Validate(doc)
	}
}

// BenchmarkValidateLarge benchmarks validating a large document (1000 individuals)
func BenchmarkValidateLarge(b *testing.B) {
	doc := generateValidDocument(1000)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		v := New()
		_ = v.Validate(doc)
	}
}

// BenchmarkValidateWithErrors benchmarks validating a document with broken references
func BenchmarkValidateWithErrors(b *testing.B) {
	doc := generateInvalidDocument(100)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		v := New()
		_ = v.Validate(doc)
	}
}

// Helper to generate a valid document with N individuals
func generateValidDocument(numIndividuals int) *gedcom.Document {
	records := make([]*gedcom.Record, 0, numIndividuals)
	xrefMap := make(map[string]*gedcom.Record)

	for i := 0; i < numIndividuals; i++ {
		xref := "@I" + string(rune('0'+i%10)) + "@"
		rec := &gedcom.Record{
			XRef: xref,
			Type: gedcom.RecordTypeIndividual,
			Tags: []*gedcom.Tag{
				{Level: 1, Tag: "NAME", Value: "Person " + string(rune('0'+i%10)) + " /Surname/"},
				{Level: 1, Tag: "SEX", Value: "M"},
			},
			Entity: &gedcom.Individual{
				XRef: xref,
				Names: []*gedcom.PersonalName{
					{Full: "Person " + string(rune('0'+i%10)) + " /Surname/"},
				},
				Sex: "M",
			},
		}
		records = append(records, rec)
		xrefMap[xref] = rec
	}

	return &gedcom.Document{
		Header:  &gedcom.Header{Version: "5.5"},
		Records: records,
		XRefMap: xrefMap,
	}
}

// Helper to generate an invalid document with broken references
func generateInvalidDocument(numIndividuals int) *gedcom.Document {
	records := make([]*gedcom.Record, 0, numIndividuals+numIndividuals/10)
	xrefMap := make(map[string]*gedcom.Record)

	// Add valid individuals
	for i := 0; i < numIndividuals; i++ {
		xref := "@I" + string(rune('0'+i%10)) + "@"
		rec := &gedcom.Record{
			XRef: xref,
			Type: gedcom.RecordTypeIndividual,
			Tags: []*gedcom.Tag{
				{Level: 1, Tag: "NAME", Value: "Person " + string(rune('0'+i%10)) + " /Surname/"},
				{Level: 1, Tag: "SEX", Value: "M"},
			},
			Entity: &gedcom.Individual{
				XRef: xref,
				Names: []*gedcom.PersonalName{
					{Full: "Person " + string(rune('0'+i%10)) + " /Surname/"},
				},
				Sex: "M",
			},
		}
		records = append(records, rec)
		xrefMap[xref] = rec
	}

	// Add families with broken references (10%)
	for i := 0; i < numIndividuals/10; i++ {
		xref := "@F" + string(rune('0'+i%10)) + "@"
		rec := &gedcom.Record{
			XRef: xref,
			Type: gedcom.RecordTypeFamily,
			Tags: []*gedcom.Tag{
				{Level: 1, Tag: "HUSB", Value: "@NONEXISTENT@"},  // Broken reference
				{Level: 1, Tag: "WIFE", Value: "@ALSO_MISSING@"}, // Broken reference
			},
			Entity: &gedcom.Family{
				XRef:    xref,
				Husband: "@NONEXISTENT@",
				Wife:    "@ALSO_MISSING@",
			},
		}
		records = append(records, rec)
		xrefMap[xref] = rec
	}

	return &gedcom.Document{
		Header:  &gedcom.Header{Version: "5.5"},
		Records: records,
		XRefMap: xrefMap,
	}
}

// TestBenchmarkFixtureShape pins the properties BenchmarkFindDuplicates depends
// on. A benchmark is a silent test: if the fixture drifts into a shape where
// most pairs match, the run still succeeds and still prints a number, but that
// number is dominated by DuplicatePair allocation rather than the comparison
// path -- and a returning #529 regression would be masked by the noise. The
// original fixture had exactly that defect, so it is asserted rather than
// assumed.
func TestBenchmarkFixtureShape(t *testing.T) {
	const size = 5000
	doc := generateDuplicateHeavyDocument(size)
	d := NewDuplicateDetector(nil)

	groups := d.buildSurnameGroups(doc.Individuals())

	// Every surname must actually reach an individual. An index cycle that
	// shares a period with the surname-less rule silently strands one.
	if len(groups) != 5 {
		t.Errorf("got %d surname groups, want 5 -- a surname is unreachable", len(groups))
	}

	var pairs int
	for _, g := range groups {
		pairs += len(g) * (len(g) - 1) / 2
	}

	matched := len(d.FindDuplicates(doc))
	rate := float64(matched) / float64(pairs)
	t.Logf("n=%d: %d groups, %d pairs compared, %d matched (%.1f%%)",
		size, len(groups), pairs, matched, rate*100)

	// The benchmark must spend its time comparing, not allocating results.
	// The pre-fix fixture sat at 27.9%.
	if rate > 0.05 {
		t.Errorf("match rate %.1f%% (%d of %d pairs) exceeds 5%%: the benchmark is "+
			"measuring DuplicatePair allocation rather than comparison cost",
			rate*100, matched, pairs)
	}

	// ...but not zero, or the result-construction path goes unexercised.
	if matched == 0 {
		t.Error("fixture produces no matches at all; DuplicatePair construction is never exercised")
	}
}

// BenchmarkFindDuplicates benchmarks duplicate detection, whose cost is
// quadratic in the size of each surname group. Name normalization used to run
// inside that pair loop, which made large documents pathologically slow (#529);
// this benchmark is the standing guard against that regression returning.
func BenchmarkFindDuplicates(b *testing.B) {
	for _, size := range []int{100, 1000, 5000} {
		doc := generateDuplicateHeavyDocument(size)
		b.Run(strconv.Itoa(size), func(b *testing.B) {
			detector := NewDuplicateDetector(nil)
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_ = detector.FindDuplicates(doc)
			}
		})
	}
}

// generateDuplicateHeavyDocument builds a document shaped like the corpus that
// exposed #529: a few large surname groups driving the pair loop, plus a block
// of surname-less individuals. Accented characters keep normalizeName doing real
// Unicode work rather than hitting its empty-string fast path.
//
// The fixture is tuned so that comparison cost, not result construction,
// dominates the measurement. Three properties matter and are easy to break:
//
//   - Given names must stay distinct *after normalization*. "Élise" and "Elise"
//     both fold to "elise", which would make them exact matches rather than the
//     near-misses intended.
//   - The surname, given-name and sex cycles must not share a period with the
//     every-fifth surname-less rule, or they alias: names collide inside a
//     bucket and most pairs match, turning the benchmark into a measure of
//     DuplicatePair allocation.
//   - Near-miss pairs (John/Jon, Margaret/Margarethe) are deliberate. They force
//     the full Levenshtein DP in compareGivenNames and then fall under
//     MinConfidence, which is the comparison work this benchmark exists to time.
//
// See TestBenchmarkFixtureShape, which pins all three.
func generateDuplicateHeavyDocument(numIndividuals int) *gedcom.Document {
	surnames := []string{"Beaumont", "Bélanger", "Neville", "Stewart", "Dafydd"}
	// An odd count, so that stepping by len(givenNames) flips i's parity and the
	// sex cycle below stays independent of the given-name cycle.
	givenNames := []string{
		"John", "Jon", "Jonathan", "Margaret", "Margarethe", "Wilhelmina",
		"Élise", "Aoife", "Caoimhe", "Bartholomew", "Seren", "Rhiannon",
		"Gwenllian", "Aneirin", "Tegwyn",
	}

	records := make([]*gedcom.Record, 0, numIndividuals)
	xrefMap := make(map[string]*gedcom.Record)

	for i := 0; i < numIndividuals; i++ {
		xref := "@I" + strconv.Itoa(i) + "@"

		// Every fifth individual carries no surname. Such records are dropped
		// before the pair loop, so this block guards the skip rather than
		// driving the cost -- it would catch a regression that started pairing
		// surname-less individuals.
		//
		// Each attribute advances on its own stride, chosen coprime to the
		// others and to the every-fifth rule, so surnames, given names and sex
		// all vary independently within a bucket.
		full := givenNames[i%len(givenNames)]
		if i%5 != 0 {
			full += " /" + surnames[(i/5)%len(surnames)] + "/"
		}

		sex := "M"
		if i%2 == 1 {
			sex = "F"
		}

		rec := &gedcom.Record{
			XRef: xref,
			Type: gedcom.RecordTypeIndividual,
			Entity: &gedcom.Individual{
				XRef:  xref,
				Names: []*gedcom.PersonalName{{Full: full}},
				Sex:   sex,
			},
		}
		records = append(records, rec)
		xrefMap[xref] = rec
	}

	return &gedcom.Document{
		Header:  &gedcom.Header{Version: "5.5"},
		Records: records,
		XRefMap: xrefMap,
	}
}
