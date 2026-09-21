package validator

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/cacack/gedcom-go/v2/decoder"
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

// duplicateBenchmarkSizes are the document sizes BenchmarkFindDuplicates runs.
// TestBenchmarkFixtureShape validates exactly this list rather than a size of
// its own, so the guard cannot fall out of step with what is measured.
var duplicateBenchmarkSizes = []int{100, 1000, 5000}

// TestBenchmarkFixtureShape pins the properties BenchmarkFindDuplicates depends
// on. A benchmark is a silent test: if the fixture drifts into a shape where
// most pairs match, the run still succeeds and still prints a number, but that
// number is dominated by DuplicatePair allocation rather than the comparison
// path -- and a returning #529 regression would be masked by the noise. The
// original fixture had exactly that defect, so it is asserted rather than
// assumed.
//
// Since #530 the fixture has a second way to go quietly wrong. The benchmark
// builds its detector with NewDuplicateDetector(nil), which now caps groups at
// DefaultMaxGroupSize. generateDuplicateHeavyDocument puts 4/5 of its
// individuals into 5 surname groups, so the largest group is n*4/25 -- 800 at
// n=5000, but past n≈6250 it crosses the cap and every group is skipped. The
// benchmark would then measure grouping alone, report a fast number, and stop
// guarding #529 entirely. Asserting the ceiling at every benchmarked size is
// what makes a future size bump fail loudly instead.
func TestBenchmarkFixtureShape(t *testing.T) {
	// Summed across sizes, not asserted per size: the smallest benchmarked
	// document has only 16 individuals per group, too few for the given-name
	// cycle to bring any pair within the confidence threshold. Requiring a
	// match at every size would fail on a fixture that is behaving correctly.
	// What must hold is that the benchmark suite exercises DuplicatePair
	// construction somewhere.
	var totalMatched int

	for _, size := range duplicateBenchmarkSizes {
		t.Run(strconv.Itoa(size), func(t *testing.T) {
			doc := generateDuplicateHeavyDocument(size)
			d := NewDuplicateDetector(nil)

			groups := d.buildSurnameGroups(doc.Individuals())

			// Every surname must actually reach an individual. An index cycle that
			// shares a period with the surname-less rule silently strands one.
			if len(groups) != 5 {
				t.Errorf("got %d surname groups, want 5 -- a surname is unreachable", len(groups))
			}

			var pairs, largest int
			for _, g := range groups {
				pairs += len(g) * (len(g) - 1) / 2
				largest = max(largest, len(g))
			}

			// The benchmark must actually compare pairs. A group above the cap
			// is skipped whole, so the benchmark would measure nothing.
			if largest > DefaultMaxGroupSize {
				t.Errorf("largest surname group is %d, above DefaultMaxGroupSize=%d: "+
					"BenchmarkFindDuplicates at n=%d skips every group and no longer "+
					"measures the comparison path. Lower the size, or give the "+
					"benchmark an explicit MaxGroupSize: UnlimitedGroupSize.",
					largest, DefaultMaxGroupSize, size)
			}

			report := d.FindDuplicatesReport(doc)
			if len(report.LimitIssues) != 0 {
				t.Errorf("fixture tripped the group cap: %s", report.LimitIssues[0].Message)
			}

			matched := len(report.Pairs)
			totalMatched += matched
			rate := float64(matched) / float64(pairs)
			t.Logf("n=%d: %d groups, largest %d, %d pairs compared, %d matched (%.1f%%)",
				size, len(groups), largest, pairs, matched, rate*100)

			// The benchmark must spend its time comparing, not allocating results.
			// The pre-fix fixture sat at 27.9%.
			if rate > 0.05 {
				t.Errorf("match rate %.1f%% (%d of %d pairs) exceeds 5%%: the benchmark is "+
					"measuring DuplicatePair allocation rather than comparison cost",
					rate*100, matched, pairs)
			}

		})
	}

	// ...but not zero across the whole suite, or the result-construction path
	// goes unexercised by every benchmark that runs.
	if totalMatched == 0 {
		t.Error("no benchmarked size produces any match; DuplicatePair construction is never exercised")
	}
}

// BenchmarkFindDuplicates benchmarks duplicate detection, whose cost is
// quadratic in the size of each surname group. Name normalization used to run
// inside that pair loop, which made large documents pathologically slow (#529);
// this benchmark is the standing guard against that regression returning.
func BenchmarkFindDuplicates(b *testing.B) {
	for _, size := range duplicateBenchmarkSizes {
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

// TestDuplicateCapPreservesCorpusResults is the standing guard on the VALUE of
// DefaultMaxGroupSize (#530).
//
// The cap only pays for itself if it never fires on real genealogical data. The
// largest normalized-surname group anywhere in this repo's corpus is 519, in
// this fixture (203,154 individuals across 55,801 groups); the next largest
// anywhere is 70. The default of 1000 therefore leaves roughly 2x headroom over
// the worst real-world observation.
//
// If someone lowers the default below ~519 to "be safer", real-world duplicate
// detection starts silently losing its largest and most interesting surname
// group, and this test is what says so: it asserts the capped sweep finds
// exactly what an uncapped one finds, and that no limit issue was emitted at
// all. Both matter -- equal pair counts alone would also hold if the cap fired
// on a group that happened to contain no duplicates.
func TestDuplicateCapPreservesCorpusResults(t *testing.T) {
	// The fixture is ~48MB and the two sweeps compare ~3M pairs each, so this
	// is a poor trade on every local iteration. CI runs the full suite.
	if testing.Short() {
		t.Skip("skipping full-corpus duplicate sweep in -short mode")
	}
	// decoder/scale_test.go refuses this same fixture under -race because the
	// decode's ~930MB peak RSS is amplified several-fold by race instrumentation
	// and risks OOM on CI runners. This test decodes it and then sweeps it three
	// times over, so the same refusal applies. CI's non-race coverage job still
	// exercises it.
	if raceDetectorEnabled {
		t.Skip("skipping full-corpus duplicate sweep under the race detector (memory amplification)")
	}

	const path = "../testdata/gedcom-5.5.1/longsword.ged"
	f, err := os.Open(path) // #nosec G304 -- path is a constant in the repo's own testdata
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer func() { _ = f.Close() }()

	doc, err := decoder.Decode(f)
	if err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}

	capped := NewDuplicateDetector(nil).FindDuplicatesReport(doc)

	unlimited := DefaultDuplicateConfig()
	unlimited.MaxGroupSize = -1
	uncapped := NewDuplicateDetector(&unlimited).FindDuplicatesReport(doc)

	if len(capped.LimitIssues) != 0 {
		t.Errorf("the default cap of %d fired on real data: %s",
			DefaultMaxGroupSize, capped.LimitIssues[0].Message)
	}
	if len(capped.Pairs) != len(uncapped.Pairs) {
		t.Errorf("capped detection found %d pairs, uncapped %d: the default cap is losing "+
			"real duplicates", len(capped.Pairs), len(uncapped.Pairs))
	}
	if len(capped.Pairs) == 0 {
		t.Fatal("the fixture produced no duplicate pairs at all; the comparison is vacuous")
	}

	var largest int
	groups := NewDuplicateDetector(nil).buildSurnameGroups(doc.Individuals())
	for _, group := range groups {
		largest = max(largest, len(group))
	}
	if largest > DefaultMaxGroupSize {
		t.Errorf("largest surname group is %d, above DefaultMaxGroupSize=%d",
			largest, DefaultMaxGroupSize)
	}
	t.Logf("%d individuals, %d surname groups, largest %d (cap %d), %d pairs",
		len(doc.Individuals()), len(groups), largest, DefaultMaxGroupSize, len(capped.Pairs))
}

// adversarialGroupSize is the size of the single surname group in
// BenchmarkFindDuplicatesAdversarial: one past DefaultMaxGroupSize, so the
// default-capped variant skips the group whole while the uncapped variant pays
// all C(1001,2) = 500,500 comparisons. At roughly 1.4 us per near-miss
// comparison that is about 0.7 s per uncapped iteration -- big enough to make
// the cap's effect unmistakable, small enough that `make bench` still finishes.
const adversarialGroupSize = DefaultMaxGroupSize + 1

// generateAdversarialDuplicateDocument builds the document shape #530 is about:
// every individual in ONE surname group, with long given names that are all
// mutual near misses.
//
// Near misses are deliberately the expensive case, and three properties make
// them so:
//
//   - All given names are the same length, so the length-based early exit in
//     compareGivenNames can never fire and every pair runs the full 26x26
//     Levenshtein DP.
//   - The resulting similarity is at worst 1-4/26 = 0.85, above
//     MinNameSimilarity, so the DP's result is not merely computed but accepted
//     -- the pair proceeds through the rest of comparePair.
//   - Confidence then tops out at 0.3 + 0.3*0.96 + 0.1 = 0.69 (surname, given
//     name, same sex; no birth dates), just under MinConfidence, so not one
//     DuplicatePair is ever constructed.
//
// The benchmark therefore measures comparison cost and nothing else: no
// allocation of results, no early exits. TestAdversarialFixtureShape pins that.
func generateAdversarialDuplicateDocument(n int) *gedcom.Document {
	records := make([]*gedcom.Record, 0, n)
	xrefMap := make(map[string]*gedcom.Record, n)

	for i := 0; i < n; i++ {
		xref := "@I" + strconv.Itoa(i) + "@"
		rec := &gedcom.Record{
			XRef: xref,
			Type: gedcom.RecordTypeIndividual,
			Entity: &gedcom.Individual{
				XRef: xref,
				Names: []*gedcom.PersonalName{{
					Given:   fmt.Sprintf("Bartholomew%04dFitzwilliam", i),
					Surname: "Featherstonehaugh",
				}},
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

// TestAdversarialFixtureShape pins what BenchmarkFindDuplicatesAdversarial
// silently depends on, in the spirit of TestBenchmarkFixtureShape above.
//
// Note the inverted expectation from that test: here ZERO matches is the
// desired property, because the point of this benchmark is the cost of
// comparing pairs that lose, which is the worst case an attacker can arrange.
// A fixture that started matching would be timing DuplicatePair allocation
// instead, and the capped-vs-uncapped contrast would blur.
func TestAdversarialFixtureShape(t *testing.T) {
	// The match-rate property holds for every pair in the fixture, and a small
	// n proves it over 19,900 of them in milliseconds where the benchmark's own
	// n would cost the better part of a second.
	const probe = 200
	doc := generateAdversarialDuplicateDocument(probe)
	d := NewDuplicateDetector(nil)

	groups := d.buildSurnameGroups(doc.Individuals())
	if len(groups) != 1 {
		t.Fatalf("got %d surname groups, want 1 -- the fixture must be a single group",
			len(groups))
	}
	for _, group := range groups {
		if len(group) != probe {
			t.Errorf("group holds %d of %d individuals; some were dropped before the sweep",
				len(group), probe)
		}
	}
	if matched := len(d.FindDuplicates(doc)); matched != 0 {
		t.Errorf("fixture produced %d matching pairs, want 0: the benchmark would be "+
			"measuring DuplicatePair allocation rather than comparison cost", matched)
	}

	// The benchmark's own n must genuinely cross the default cap, or the
	// "capped" variant would measure the same sweep as the uncapped one and the
	// comparison would be meaningless. Grouping is linear, so this costs
	// nothing to check.
	big := generateAdversarialDuplicateDocument(adversarialGroupSize)
	report := NewDuplicateDetector(nil).FindDuplicatesReport(big)
	if len(report.LimitIssues) != 1 {
		t.Fatalf("n=%d produced %d limit issues, want 1: the capped benchmark variant is "+
			"not actually being capped", adversarialGroupSize, len(report.LimitIssues))
	}
	if got, want := report.LimitIssues[0].Details["largest_group"], strconv.Itoa(adversarialGroupSize); got != want {
		t.Errorf("largest_group = %q, want %q", got, want)
	}
}

// BenchmarkFindDuplicatesAdversarial measures the worst case #530 describes: a
// document whose individuals all share one surname and whose given names are
// mutual near misses, so every pair pays the full Levenshtein DP and then
// loses. The uncapped variant is the pre-#530 behaviour; the capped variant is
// what a caller gets by default. See generateAdversarialDuplicateDocument for
// why the fixture is expensive on purpose, and adversarialGroupSize for the
// sizing.
func BenchmarkFindDuplicatesAdversarial(b *testing.B) {
	doc := generateAdversarialDuplicateDocument(adversarialGroupSize)

	unlimited := DefaultDuplicateConfig()
	unlimited.MaxGroupSize = -1

	variants := []struct {
		name   string
		config *DuplicateConfig
	}{
		{"capped", nil}, // nil -> DefaultDuplicateConfig, i.e. what callers get
		{"uncapped", &unlimited},
	}

	for _, variant := range variants {
		b.Run(variant.name, func(b *testing.B) {
			detector := NewDuplicateDetector(variant.config)
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_ = detector.FindDuplicates(doc)
			}
		})
	}
}
