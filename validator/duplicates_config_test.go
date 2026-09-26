package validator

import (
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// TestNewDuplicateDetector_PartialLiteralGetsDefaults pins #555: a struct
// literal naming one field must leave every other field at its documented
// default, rather than at a Go zero value that disables it. In v2 the literal
// below ran with name normalization off and no given-name gate.
func TestNewDuplicateDetector_PartialLiteralGetsDefaults(t *testing.T) {
	detector := NewDuplicateDetector(&DuplicateConfig{MinConfidence: 0.9})
	got := detector.config

	if got.MinConfidence != 0.9 {
		t.Errorf("MinConfidence = %v, want 0.9 (the field that was set)", got.MinConfidence)
	}
	if got.MinNameSimilarity != defaultMinNameSimilarity {
		t.Errorf("MinNameSimilarity = %v, want default %v", got.MinNameSimilarity, defaultMinNameSimilarity)
	}
	if got.MaxBirthYearDiff != defaultMaxBirthYearDiff {
		t.Errorf("MaxBirthYearDiff = %d, want default %d", got.MaxBirthYearDiff, defaultMaxBirthYearDiff)
	}
	if got.DisableNameNormalization {
		t.Error("DisableNameNormalization = true, want false (normalization on by default)")
	}
	if detector.maxGroupSize() != DefaultMaxGroupSize {
		t.Errorf("maxGroupSize() = %d, want DefaultMaxGroupSize", detector.maxGroupSize())
	}
}

// TestNewDuplicateDetector_ZeroConfigEqualsDefault checks that an empty
// literal, nil, and DefaultDuplicateConfig all resolve to the same detector
// configuration.
func TestNewDuplicateDetector_ZeroConfigEqualsDefault(t *testing.T) {
	zero := NewDuplicateDetector(&DuplicateConfig{})
	def := NewDuplicateDetector(DefaultDuplicateConfig())
	fromNil := NewDuplicateDetector(nil)

	for name, d := range map[string]*DuplicateDetector{"&DuplicateConfig{}": zero, "nil": fromNil} {
		if d.maxGroupSize() != def.maxGroupSize() {
			t.Errorf("%s: maxGroupSize() = %d, want %d", name, d.maxGroupSize(), def.maxGroupSize())
		}
		// MaxGroupSize is resolved on use rather than at construction, so it
		// is compared above and blanked here.
		gotCfg, wantCfg := d.config, def.config
		gotCfg.MaxGroupSize, wantCfg.MaxGroupSize = 0, 0
		if gotCfg != wantCfg {
			t.Errorf("%s resolved to %+v, DefaultDuplicateConfig() to %+v", name, gotCfg, wantCfg)
		}
	}
}

func TestNewDuplicateDetector_ThresholdResolution(t *testing.T) {
	tests := []struct {
		name           string
		config         DuplicateConfig
		wantSimilarity float64
		wantYears      int
		wantConfidence float64
	}{
		{
			name:           "zero selects defaults",
			config:         DuplicateConfig{},
			wantSimilarity: 0.8,
			wantYears:      2,
			wantConfidence: 0.7,
		},
		{
			name: "ZeroThreshold selects an explicit zero",
			config: DuplicateConfig{
				MinNameSimilarity: ZeroThreshold,
				MaxBirthYearDiff:  ZeroThreshold,
				MinConfidence:     ZeroThreshold,
			},
		},
		{
			name:   "any negative value selects an explicit zero",
			config: DuplicateConfig{MinNameSimilarity: -0.5, MaxBirthYearDiff: -7, MinConfidence: -2},
		},
		{
			name:           "positive values are used as given",
			config:         DuplicateConfig{MinNameSimilarity: 0.5, MaxBirthYearDiff: 10, MinConfidence: 0.95},
			wantSimilarity: 0.5,
			wantYears:      10,
			wantConfidence: 0.95,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.config
			got := NewDuplicateDetector(&cfg).config
			if got.MinNameSimilarity != tt.wantSimilarity {
				t.Errorf("MinNameSimilarity = %v, want %v", got.MinNameSimilarity, tt.wantSimilarity)
			}
			if got.MaxBirthYearDiff != tt.wantYears {
				t.Errorf("MaxBirthYearDiff = %d, want %d", got.MaxBirthYearDiff, tt.wantYears)
			}
			if got.MinConfidence != tt.wantConfidence {
				t.Errorf("MinConfidence = %v, want %v", got.MinConfidence, tt.wantConfidence)
			}
			// The caller's struct is copied, never rewritten in place.
			if cfg != tt.config {
				t.Errorf("caller's config was mutated: got %+v, want %+v", cfg, tt.config)
			}
		})
	}
}

// twoIndividualDoc builds a document holding exactly the two given individuals.
func twoIndividualDoc(ind1, ind2 *gedcom.Individual) *gedcom.Document {
	return &gedcom.Document{
		Records: []*gedcom.Record{
			{XRef: ind1.XRef, Type: gedcom.RecordTypeIndividual, Entity: ind1},
			{XRef: ind2.XRef, Type: gedcom.RecordTypeIndividual, Entity: ind2},
		},
	}
}

// bornIn returns an individual with one name, sex M, and a birth in year.
func bornIn(t *testing.T, xref, name, year string) *gedcom.Individual {
	t.Helper()
	date, err := gedcom.ParseDate(year)
	if err != nil {
		t.Fatalf("ParseDate(%q): %v", year, err)
	}
	return &gedcom.Individual{
		XRef:   xref,
		Names:  []*gedcom.PersonalName{{Full: name}},
		Sex:    "M",
		Events: []*gedcom.Event{{Type: gedcom.EventBirth, ParsedDate: date}},
	}
}

// TestFindDuplicates_PartialLiteralKeepsGivenNameGate is the behavioural half
// of #555: with only MinConfidence set, "John" and "Mary" must still be kept
// apart by the default given-name gate. In v2 MinNameSimilarity silently
// became 0 here and the pair was reported.
func TestFindDuplicates_PartialLiteralKeepsGivenNameGate(t *testing.T) {
	doc := twoIndividualDoc(
		bornIn(t, "@I1@", "John /Doe/", "1850"),
		bornIn(t, "@I2@", "Mary /Doe/", "1850"),
	)

	if got := NewDuplicateDetector(&DuplicateConfig{MinConfidence: 0.5}).FindDuplicates(doc); len(got) != 0 {
		t.Errorf("partial literal: got %d pairs, want 0 (given-name gate must stay on)", len(got))
	}

	// Opting out is explicit and still works.
	cfg := &DuplicateConfig{MinConfidence: 0.5, MinNameSimilarity: ZeroThreshold}
	if got := NewDuplicateDetector(cfg).FindDuplicates(doc); len(got) != 1 {
		t.Errorf("MinNameSimilarity: ZeroThreshold: got %d pairs, want 1", len(got))
	}
}

// TestFindDuplicates_PartialLiteralNormalizes pins the other half of #555: a
// partial literal must not turn off case folding.
func TestFindDuplicates_PartialLiteralNormalizes(t *testing.T) {
	doc := twoIndividualDoc(
		&gedcom.Individual{XRef: "@I1@", Names: []*gedcom.PersonalName{{Full: "John /DOE/"}}, Sex: "M"},
		&gedcom.Individual{XRef: "@I2@", Names: []*gedcom.PersonalName{{Full: "john /Doe/"}}, Sex: "M"},
	)

	if got := NewDuplicateDetector(&DuplicateConfig{MinConfidence: 0.6}).FindDuplicates(doc); len(got) != 1 {
		t.Errorf("partial literal: got %d pairs, want 1 (names normalized by default)", len(got))
	}

	cfg := &DuplicateConfig{MinConfidence: 0.6, DisableNameNormalization: true}
	if got := NewDuplicateDetector(cfg).FindDuplicates(doc); len(got) != 0 {
		t.Errorf("DisableNameNormalization: got %d pairs, want 0", len(got))
	}
}

// TestFindDuplicates_MaxBirthYearDiffResolution checks the birth-year window
// through its observable effect: a one-year gap earns partial credit under the
// default window and none under ZeroThreshold.
func TestFindDuplicates_MaxBirthYearDiffResolution(t *testing.T) {
	ind1 := bornIn(t, "@I1@", "John /Doe/", "1850")
	ind2 := bornIn(t, "@I2@", "John /Doe/", "1851")
	ind1.Sex, ind2.Sex = "", ""
	doc := twoIndividualDoc(ind1, ind2)

	// Surname 0.3 + given 0.3 + near birth year 0.1 = 0.7, exactly the default
	// MinConfidence; without the birth-year credit the pair falls short.
	if got := NewDuplicateDetector(&DuplicateConfig{}).FindDuplicates(doc); len(got) != 1 {
		t.Errorf("MaxBirthYearDiff zero (default 2): got %d pairs, want 1", len(got))
	}
	cfg := &DuplicateConfig{MaxBirthYearDiff: ZeroThreshold}
	if got := NewDuplicateDetector(cfg).FindDuplicates(doc); len(got) != 0 {
		t.Errorf("MaxBirthYearDiff: ZeroThreshold: got %d pairs, want 0", len(got))
	}
}

// TestValidator_PartialDuplicateLiteralGetsDefaults drives the same rule
// through the public ValidatorConfig.Duplicates entry point.
func TestValidator_PartialDuplicateLiteralGetsDefaults(t *testing.T) {
	doc := twoIndividualDoc(
		bornIn(t, "@I1@", "John /Doe/", "1850"),
		bornIn(t, "@I2@", "Mary /Doe/", "1850"),
	)

	v := NewWithConfig(&ValidatorConfig{Duplicates: &DuplicateConfig{MinConfidence: 0.5}})
	if got := v.FindPotentialDuplicates(doc); len(got) != 0 {
		t.Errorf("got %d pairs, want 0: the partial literal must keep the default given-name gate", len(got))
	}
}
