package validator

import (
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"
)

// Issue #949: duplicate detection reuses its Levenshtein buffers across the
// pairs of one sweep and spells out MatchReasons only for accepted pairs, so a
// compared pair allocates next to nothing.

// TestLevenshteinScratchReuse checks that a scratch reused across pairs of
// varying length, longest first so later pairs run in oversized buffers, gives
// the distance a fresh scratch gives every pair.
func TestLevenshteinScratchReuse(t *testing.T) {
	pairs := [][2]string{
		{strings.Repeat("abcdefgh", 8), strings.Repeat("abcdefgi", 8)},
		{"kitten", "sitting"},
		{"", "abc"},
		{"abc", ""},
		{"josé", "jose"},
		{"müller", "mueller"},
		{"a", "b"},
		{"same", "same"},
		{"\xff\xfe", "\xff"},
		{"Bartholomew0001Fitzwilliam", "Bartholomew0002Fitzwilliam"},
		{"jon", "john"},
	}
	reused := new(levenshteinScratch)
	for _, p := range pairs {
		want := levenshteinDistance(p[0], p[1], new(levenshteinScratch))
		if got := levenshteinDistance(p[0], p[1], reused); got != want {
			t.Errorf("levenshteinDistance(%q, %q) with a reused scratch = %d, fresh = %d",
				p[0], p[1], got, want)
		}
	}
}

// TestFindDuplicatesAllocsPerPair pins the issue's bound: at most one
// allocation per compared pair on the near-miss fixture, where every pair
// runs the full DP and is then rejected.
func TestFindDuplicatesAllocsPerPair(t *testing.T) {
	if raceDetectorEnabled {
		t.Skip("allocation counts are not meaningful under the race detector")
	}
	const n = 200
	pairs := n * (n - 1) / 2
	doc := generateAdversarialDuplicateDocument(n)
	d := NewDuplicateDetector(nil)

	allocs := testing.AllocsPerRun(5, func() { _ = d.FindDuplicates(doc) })
	if perPair := allocs / float64(pairs); perPair > 1 {
		t.Errorf("FindDuplicates made %.0f allocations over %d compared pairs, %.2f per pair; want at most 1",
			allocs, pairs, perPair)
	}
}

// TestFindDuplicatesConcurrent runs every sweep on one DuplicateDetector from
// several goroutines at once. Under -race it shows the per-sweep buffers and
// the pooled normalization chains are not shared between calls; in any mode
// it checks each call gets the result a lone call gets.
func TestFindDuplicatesConcurrent(t *testing.T) {
	a := generateDuplicateHeavyDocument(400)
	b := generateDuplicateHeavyDocument(300)
	config := DefaultDuplicateConfig()
	config.CompareRelatives = true
	d := NewDuplicateDetector(config)

	wantOne := pairLines(d.FindDuplicates(a))
	wantBetween := pairLines(d.FindDuplicatesBetween(a, b))
	if len(wantOne) == 0 || len(wantBetween) == 0 {
		t.Fatalf("fixture found %d and %d pairs; the comparison is vacuous",
			len(wantOne), len(wantBetween))
	}

	const workers = 8
	var wg sync.WaitGroup
	errs := make(chan string, workers)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			if w%2 == 0 {
				if got := pairLines(d.FindDuplicates(a)); !slices.Equal(got, wantOne) {
					errs <- fmt.Sprintf("worker %d: FindDuplicates gave %d pairs, want %d", w, len(got), len(wantOne))
				}
				return
			}
			if got := pairLines(d.FindDuplicatesBetween(a, b)); !slices.Equal(got, wantBetween) {
				errs <- fmt.Sprintf("worker %d: FindDuplicatesBetween gave %d pairs, want %d", w, len(got), len(wantBetween))
			}
		}(w)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}

// pairLines renders pairs, reasons included, sorted, since pairs come in no
// particular order.
func pairLines(pairs []DuplicatePair) []string {
	lines := make([]string, len(pairs))
	for i, p := range pairs {
		lines[i] = fmt.Sprintf("%s|%s|%v|%s", p.Individual1.XRef, p.Individual2.XRef,
			p.Confidence, strings.Join(p.MatchReasons, ";"))
	}
	slices.Sort(lines)
	return lines
}

// TestPairSignalsReasons pins the wording and order of every reason, which
// comparePair used to append as it went and now spells out on acceptance.
func TestPairSignalsReasons(t *testing.T) {
	tests := []struct {
		name    string
		signals pairSignals
		want    []string
	}{
		{
			name:    "zero value credits nothing",
			signals: pairSignals{},
			want:    nil,
		},
		{
			name:    "minimal",
			signals: pairSignals{givenMatched: true, givenSimilarity: 1, sexUnrecorded: true},
			want:    []string{"exact given name match", "sex not recorded on both"},
		},
		{
			name:    "given names 0% similar under ZeroThreshold",
			signals: pairSignals{givenMatched: true, sexUnrecorded: true},
			want:    []string{"similar given name (0%)", "sex not recorded on both"},
		},
		{
			name: "every signal",
			signals: pairSignals{
				sameSurname: true, givenMatched: true, givenSimilarity: 0.875, sameBirthYear: true, sameSex: true,
				sameBirthDate: true, sameDeathYear: true, sameBirthPlace: true,
				sameDeathPlace: true, sameSpouse: true, sharedChildren: 1,
			},
			want: []string{
				"exact surname match", "similar given name (88%)", "same birth year", "same sex",
				"same birth date", "same death year", "same birth place", "same death place",
				"same spouse name", "1 shared child name",
			},
		},
		{
			name:    "years apart and several children",
			signals: pairSignals{sameSurname: true, givenMatched: true, givenSimilarity: 1, birthYearsApart: 3, sexUnrecorded: true, sharedChildren: 4},
			want: []string{
				"exact surname match", "exact given name match", "birth year within 3 years",
				"sex not recorded on both", "4 shared child names",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.signals.reasons(); !slices.Equal(got, tt.want) {
				t.Errorf("reasons() = %q, want %q", got, tt.want)
			}
		})
	}
}
