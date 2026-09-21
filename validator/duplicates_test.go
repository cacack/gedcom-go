package validator

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v2/gedcom"
)

func TestDefaultDuplicateConfig(t *testing.T) {
	config := DefaultDuplicateConfig()

	if !config.RequireExactSurname {
		t.Error("RequireExactSurname should default to true")
	}
	if !config.NormalizeNames {
		t.Error("NormalizeNames should default to true")
	}
	if config.MinNameSimilarity != 0.8 {
		t.Errorf("MinNameSimilarity = %v, want 0.8", config.MinNameSimilarity)
	}
	if config.MaxBirthYearDiff != 2 {
		t.Errorf("MaxBirthYearDiff = %d, want 2", config.MaxBirthYearDiff)
	}
	if config.RequireBirthDate {
		t.Error("RequireBirthDate should default to false")
	}
	if config.MinConfidence != 0.7 {
		t.Errorf("MinConfidence = %v, want 0.7", config.MinConfidence)
	}
	// The default is spelled out rather than left at the zero value so that a
	// config built from this function and one built as a partial literal are
	// bounded by the same number (#530).
	if config.MaxGroupSize != DefaultMaxGroupSize {
		t.Errorf("MaxGroupSize = %d, want DefaultMaxGroupSize (%d)",
			config.MaxGroupSize, DefaultMaxGroupSize)
	}
}

func TestNewDuplicateDetector(t *testing.T) {
	// Test with nil config (should use defaults)
	detector := NewDuplicateDetector(nil)
	if detector == nil {
		t.Fatal("NewDuplicateDetector(nil) returned nil")
	}
	if detector.config.MinNameSimilarity != 0.8 {
		t.Error("Nil config should result in default values")
	}

	// Test with custom config
	customConfig := &DuplicateConfig{
		MinNameSimilarity: 0.9,
		MinConfidence:     0.8,
	}
	detector = NewDuplicateDetector(customConfig)
	if detector.config.MinNameSimilarity != 0.9 {
		t.Errorf("Custom config not applied, MinNameSimilarity = %v", detector.config.MinNameSimilarity)
	}
}

func TestNormalizeName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple lowercase",
			input: "John",
			want:  "john",
		},
		{
			name:  "already lowercase",
			input: "john",
			want:  "john",
		},
		{
			name:  "with spaces",
			input: "  John Doe  ",
			want:  "john doe",
		},
		{
			name:  "acute accent",
			input: "Jose",
			want:  "jose",
		},
		{
			name:  "e with acute",
			input: "Jose\u0301", // Jose + combining acute
			want:  "jose",
		},
		{
			name:  "precomposed e-acute",
			input: "Jos\u00e9", // precomposed e-acute
			want:  "jose",
		},
		{
			name:  "umlaut",
			input: "M\u00fcller", // Muller with u-umlaut
			want:  "muller",
		},
		{
			name:  "multiple accents",
			input: "Caf\u00e9", // Cafe with e-acute
			want:  "cafe",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "cedilla",
			input: "Fran\u00e7ois", // Francois with c-cedilla
			want:  "francois",
		},
		{
			name:  "tilde",
			input: "Espa\u00f1a", // Espana with n-tilde
			want:  "espana",
		},
		{
			name:  "circumflex",
			input: "Beno\u00eet", // Benoit with i-circumflex
			want:  "benoit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeName(tt.input)
			if got != tt.want {
				t.Errorf("normalizeName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestStringSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		s1       string
		s2       string
		want     float64
		minValue float64 // For approximate comparisons
		maxValue float64
	}{
		{
			name:     "identical strings",
			s1:       "john",
			s2:       "john",
			want:     1.0,
			minValue: 1.0,
			maxValue: 1.0,
		},
		{
			name:     "empty strings",
			s1:       "",
			s2:       "",
			want:     1.0,
			minValue: 1.0,
			maxValue: 1.0,
		},
		{
			name:     "one empty string",
			s1:       "john",
			s2:       "",
			want:     0.0,
			minValue: 0.0,
			maxValue: 0.0,
		},
		{
			name:     "other empty string",
			s1:       "",
			s2:       "john",
			want:     0.0,
			minValue: 0.0,
			maxValue: 0.0,
		},
		{
			name:     "john vs jon (similar)",
			s1:       "john",
			s2:       "jon",
			minValue: 0.7,
			maxValue: 0.85,
		},
		{
			name:     "william vs bill (different)",
			s1:       "william",
			s2:       "bill",
			minValue: 0.0,
			maxValue: 0.5,
		},
		{
			name:     "robert vs bob",
			s1:       "robert",
			s2:       "bob",
			minValue: 0.0,
			maxValue: 0.5,
		},
		{
			name:     "elizabeth vs beth",
			s1:       "elizabeth",
			s2:       "beth",
			minValue: 0.3,
			maxValue: 0.6,
		},
		{
			name:     "completely different",
			s1:       "john",
			s2:       "mary",
			minValue: 0.0,
			maxValue: 0.3,
		},
		{
			name:     "one character difference",
			s1:       "smith",
			s2:       "smyth",
			minValue: 0.7,
			maxValue: 0.9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stringSimilarity(tt.s1, tt.s2)

			if tt.want != 0 {
				// Exact comparison
				if got != tt.want {
					t.Errorf("stringSimilarity(%q, %q) = %v, want %v", tt.s1, tt.s2, got, tt.want)
				}
			} else {
				// Range comparison
				if got < tt.minValue || got > tt.maxValue {
					t.Errorf("stringSimilarity(%q, %q) = %v, want in range [%v, %v]",
						tt.s1, tt.s2, got, tt.minValue, tt.maxValue)
				}
			}
		})
	}
}

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		name string
		s1   string
		s2   string
		want int
	}{
		{
			name: "identical",
			s1:   "john",
			s2:   "john",
			want: 0,
		},
		{
			name: "one substitution",
			s1:   "john",
			s2:   "jonn",
			want: 1,
		},
		{
			name: "one deletion",
			s1:   "john",
			s2:   "jon",
			want: 1,
		},
		{
			name: "one insertion",
			s1:   "jon",
			s2:   "john",
			want: 1,
		},
		{
			name: "empty first",
			s1:   "",
			s2:   "john",
			want: 4,
		},
		{
			name: "empty second",
			s1:   "john",
			s2:   "",
			want: 4,
		},
		{
			name: "both empty",
			s1:   "",
			s2:   "",
			want: 0,
		},
		{
			name: "kitten to sitting",
			s1:   "kitten",
			s2:   "sitting",
			want: 3,
		},
		{
			name: "unicode characters",
			s1:   "cafe",
			s2:   "cafe",
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := levenshteinDistance(tt.s1, tt.s2)
			if got != tt.want {
				t.Errorf("levenshteinDistance(%q, %q) = %d, want %d", tt.s1, tt.s2, got, tt.want)
			}
		})
	}
}

func TestCompareSurnames(t *testing.T) {
	tests := []struct {
		name  string
		s1    string
		s2    string
		exact bool
		want  bool
	}{
		{
			name:  "exact match - exact mode",
			s1:    "smith",
			s2:    "smith",
			exact: true,
			want:  true,
		},
		{
			name:  "different - exact mode",
			s1:    "smith",
			s2:    "jones",
			exact: true,
			want:  false,
		},
		{
			name:  "exact match - non-exact mode",
			s1:    "smith",
			s2:    "smith",
			exact: false,
			want:  true,
		},
		{
			name:  "empty first",
			s1:    "",
			s2:    "smith",
			exact: true,
			want:  false,
		},
		{
			name:  "empty second",
			s1:    "smith",
			s2:    "",
			exact: true,
			want:  false,
		},
		{
			name:  "both empty",
			s1:    "",
			s2:    "",
			exact: true,
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := compareSurnames(tt.s1, tt.s2, tt.exact)
			if got != tt.want {
				t.Errorf("compareSurnames(%q, %q, %v) = %v, want %v",
					tt.s1, tt.s2, tt.exact, got, tt.want)
			}
		})
	}
}

func TestCompareGivenNames(t *testing.T) {
	tests := []struct {
		name          string
		g1            string
		g2            string
		minSimilarity float64
		wantMin       float64
		wantMax       float64
	}{
		{
			name:          "exact match",
			g1:            "john",
			g2:            "john",
			minSimilarity: 0.8,
			wantMin:       1.0,
			wantMax:       1.0,
		},
		{
			name:          "empty first",
			g1:            "",
			g2:            "john",
			minSimilarity: 0.8,
			wantMin:       0.0,
			wantMax:       0.0,
		},
		{
			name:          "empty second",
			g1:            "john",
			g2:            "",
			minSimilarity: 0.8,
			wantMin:       0.0,
			wantMax:       0.0,
		},
		{
			name:          "similar names",
			g1:            "john",
			g2:            "jon",
			minSimilarity: 0.7,
			wantMin:       0.7,
			wantMax:       0.85,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := compareGivenNames(tt.g1, tt.g2, tt.minSimilarity)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("compareGivenNames(%q, %q, %v) = %v, want in [%v, %v]",
					tt.g1, tt.g2, tt.minSimilarity, got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestExtractSurnameFromFull(t *testing.T) {
	tests := []struct {
		name     string
		fullName string
		want     string
	}{
		{
			name:     "standard format",
			fullName: "John /Doe/",
			want:     "Doe",
		},
		{
			name:     "no surname",
			fullName: "John",
			want:     "",
		},
		{
			name:     "empty string",
			fullName: "",
			want:     "",
		},
		{
			name:     "only surname",
			fullName: "/Doe/",
			want:     "Doe",
		},
		{
			name:     "multiple names",
			fullName: "John Paul /Doe Smith/",
			want:     "Doe Smith",
		},
		{
			name:     "single slash",
			fullName: "John /Doe",
			want:     "",
		},
		{
			name:     "adjacent slashes",
			fullName: "John //",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractSurnameFromFull(tt.fullName)
			if got != tt.want {
				t.Errorf("extractSurnameFromFull(%q) = %q, want %q", tt.fullName, got, tt.want)
			}
		})
	}
}

func TestExtractGivenFromFull(t *testing.T) {
	tests := []struct {
		name     string
		fullName string
		want     string
	}{
		{
			name:     "standard format",
			fullName: "John /Doe/",
			want:     "John",
		},
		{
			name:     "no surname",
			fullName: "John",
			want:     "John",
		},
		{
			name:     "empty string",
			fullName: "",
			want:     "",
		},
		{
			name:     "only surname",
			fullName: "/Doe/",
			want:     "",
		},
		{
			name:     "multiple given names",
			fullName: "John Paul /Doe/",
			want:     "John Paul",
		},
		{
			name:     "with leading space",
			fullName: "  John /Doe/",
			want:     "John",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractGivenFromFull(tt.fullName)
			if got != tt.want {
				t.Errorf("extractGivenFromFull(%q) = %q, want %q", tt.fullName, got, tt.want)
			}
		})
	}
}

func TestFindDuplicates_ExactMatch(t *testing.T) {
	// Create two individuals with exact same name
	ind1 := &gedcom.Individual{
		XRef:  "@I1@",
		Names: []*gedcom.PersonalName{{Full: "John /Doe/"}},
		Sex:   "M",
	}
	ind2 := &gedcom.Individual{
		XRef:  "@I2@",
		Names: []*gedcom.PersonalName{{Full: "John /Doe/"}},
		Sex:   "M",
	}

	doc := &gedcom.Document{
		Records: []*gedcom.Record{
			{XRef: ind1.XRef, Type: gedcom.RecordTypeIndividual, Entity: ind1},
			{XRef: ind2.XRef, Type: gedcom.RecordTypeIndividual, Entity: ind2},
		},
	}

	detector := NewDuplicateDetector(nil)
	duplicates := detector.FindDuplicates(doc)

	if len(duplicates) != 1 {
		t.Fatalf("Expected 1 duplicate pair, got %d", len(duplicates))
	}

	pair := duplicates[0]
	if pair.Individual1.XRef != "@I1@" || pair.Individual2.XRef != "@I2@" {
		t.Errorf("Wrong individuals in pair: %s, %s", pair.Individual1.XRef, pair.Individual2.XRef)
	}

	if pair.Confidence < 0.7 {
		t.Errorf("Confidence too low: %v", pair.Confidence)
	}

	if len(pair.MatchReasons) == 0 {
		t.Error("Expected match reasons to be populated")
	}
}

func TestFindDuplicates_SimilarGivenName(t *testing.T) {
	// Create two individuals with similar given names (John vs Jon)
	ind1 := &gedcom.Individual{
		XRef:  "@I1@",
		Names: []*gedcom.PersonalName{{Full: "John /Smith/"}},
		Sex:   "M",
	}
	ind2 := &gedcom.Individual{
		XRef:  "@I2@",
		Names: []*gedcom.PersonalName{{Full: "Jon /Smith/"}},
		Sex:   "M",
	}

	doc := &gedcom.Document{
		Records: []*gedcom.Record{
			{XRef: ind1.XRef, Type: gedcom.RecordTypeIndividual, Entity: ind1},
			{XRef: ind2.XRef, Type: gedcom.RecordTypeIndividual, Entity: ind2},
		},
	}

	// Use lower similarity threshold to catch John/Jon
	config := DefaultDuplicateConfig()
	config.MinNameSimilarity = 0.7
	config.MinConfidence = 0.6

	detector := NewDuplicateDetector(&config)
	duplicates := detector.FindDuplicates(doc)

	if len(duplicates) != 1 {
		t.Fatalf("Expected 1 duplicate pair for similar names, got %d", len(duplicates))
	}

	// Verify match reasons contain "similar given name"
	foundSimilar := false
	for _, reason := range duplicates[0].MatchReasons {
		if containsSubstring(reason, "similar given name") {
			foundSimilar = true
			break
		}
	}
	if !foundSimilar {
		t.Errorf("Expected 'similar given name' in match reasons, got %v", duplicates[0].MatchReasons)
	}
}

func TestFindDuplicates_DifferentSurnames(t *testing.T) {
	// Create two individuals with different surnames
	ind1 := &gedcom.Individual{
		XRef:  "@I1@",
		Names: []*gedcom.PersonalName{{Full: "John /Doe/"}},
	}
	ind2 := &gedcom.Individual{
		XRef:  "@I2@",
		Names: []*gedcom.PersonalName{{Full: "John /Smith/"}},
	}

	doc := &gedcom.Document{
		Records: []*gedcom.Record{
			{XRef: ind1.XRef, Type: gedcom.RecordTypeIndividual, Entity: ind1},
			{XRef: ind2.XRef, Type: gedcom.RecordTypeIndividual, Entity: ind2},
		},
	}

	detector := NewDuplicateDetector(nil)
	duplicates := detector.FindDuplicates(doc)

	if len(duplicates) != 0 {
		t.Errorf("Expected 0 duplicate pairs for different surnames, got %d", len(duplicates))
	}
}

func TestFindDuplicates_BirthDateProximity(t *testing.T) {
	birthDate1, _ := gedcom.ParseDate("1 JAN 1900")
	birthDate2, _ := gedcom.ParseDate("1 JAN 1901")

	ind1 := &gedcom.Individual{
		XRef:  "@I1@",
		Names: []*gedcom.PersonalName{{Full: "John /Doe/"}},
		Sex:   "M",
		Events: []*gedcom.Event{
			{Type: gedcom.EventBirth, ParsedDate: birthDate1},
		},
	}
	ind2 := &gedcom.Individual{
		XRef:  "@I2@",
		Names: []*gedcom.PersonalName{{Full: "John /Doe/"}},
		Sex:   "M",
		Events: []*gedcom.Event{
			{Type: gedcom.EventBirth, ParsedDate: birthDate2},
		},
	}

	doc := &gedcom.Document{
		Records: []*gedcom.Record{
			{XRef: ind1.XRef, Type: gedcom.RecordTypeIndividual, Entity: ind1},
			{XRef: ind2.XRef, Type: gedcom.RecordTypeIndividual, Entity: ind2},
		},
	}

	detector := NewDuplicateDetector(nil)
	duplicates := detector.FindDuplicates(doc)

	if len(duplicates) != 1 {
		t.Fatalf("Expected 1 duplicate pair, got %d", len(duplicates))
	}

	// Check that birth year proximity is in match reasons
	foundBirthYear := false
	for _, reason := range duplicates[0].MatchReasons {
		if containsSubstring(reason, "birth year") {
			foundBirthYear = true
			break
		}
	}
	if !foundBirthYear {
		t.Errorf("Expected birth year in match reasons, got %v", duplicates[0].MatchReasons)
	}
}

func TestFindDuplicates_SameBirthYear(t *testing.T) {
	birthDate1, _ := gedcom.ParseDate("1 JAN 1900")
	birthDate2, _ := gedcom.ParseDate("15 MAR 1900")

	ind1 := &gedcom.Individual{
		XRef:  "@I1@",
		Names: []*gedcom.PersonalName{{Full: "John /Doe/"}},
		Sex:   "M",
		Events: []*gedcom.Event{
			{Type: gedcom.EventBirth, ParsedDate: birthDate1},
		},
	}
	ind2 := &gedcom.Individual{
		XRef:  "@I2@",
		Names: []*gedcom.PersonalName{{Full: "John /Doe/"}},
		Sex:   "M",
		Events: []*gedcom.Event{
			{Type: gedcom.EventBirth, ParsedDate: birthDate2},
		},
	}

	doc := &gedcom.Document{
		Records: []*gedcom.Record{
			{XRef: ind1.XRef, Type: gedcom.RecordTypeIndividual, Entity: ind1},
			{XRef: ind2.XRef, Type: gedcom.RecordTypeIndividual, Entity: ind2},
		},
	}

	detector := NewDuplicateDetector(nil)
	duplicates := detector.FindDuplicates(doc)

	if len(duplicates) != 1 {
		t.Fatalf("Expected 1 duplicate pair, got %d", len(duplicates))
	}

	// Check that "same birth year" is in match reasons
	foundSameBirthYear := false
	for _, reason := range duplicates[0].MatchReasons {
		if reason == "same birth year" {
			foundSameBirthYear = true
			break
		}
	}
	if !foundSameBirthYear {
		t.Errorf("Expected 'same birth year' in match reasons, got %v", duplicates[0].MatchReasons)
	}
}

func TestFindDuplicates_MissingBirthDates(t *testing.T) {
	ind1 := &gedcom.Individual{
		XRef:  "@I1@",
		Names: []*gedcom.PersonalName{{Full: "John /Doe/"}},
		Sex:   "M",
	}
	ind2 := &gedcom.Individual{
		XRef:  "@I2@",
		Names: []*gedcom.PersonalName{{Full: "John /Doe/"}},
		Sex:   "M",
	}

	doc := &gedcom.Document{
		Records: []*gedcom.Record{
			{XRef: ind1.XRef, Type: gedcom.RecordTypeIndividual, Entity: ind1},
			{XRef: ind2.XRef, Type: gedcom.RecordTypeIndividual, Entity: ind2},
		},
	}

	// Test with RequireBirthDate = false (default)
	detector := NewDuplicateDetector(nil)
	duplicates := detector.FindDuplicates(doc)

	if len(duplicates) != 1 {
		t.Fatalf("Expected 1 duplicate pair with RequireBirthDate=false, got %d", len(duplicates))
	}

	// Test with RequireBirthDate = true
	config := DefaultDuplicateConfig()
	config.RequireBirthDate = true
	detector = NewDuplicateDetector(&config)
	duplicates = detector.FindDuplicates(doc)

	if len(duplicates) != 0 {
		t.Errorf("Expected 0 duplicate pairs with RequireBirthDate=true, got %d", len(duplicates))
	}
}

func TestFindDuplicates_ConfigThresholds(t *testing.T) {
	ind1 := &gedcom.Individual{
		XRef:  "@I1@",
		Names: []*gedcom.PersonalName{{Full: "John /Doe/"}},
	}
	ind2 := &gedcom.Individual{
		XRef:  "@I2@",
		Names: []*gedcom.PersonalName{{Full: "John /Doe/"}},
	}

	doc := &gedcom.Document{
		Records: []*gedcom.Record{
			{XRef: ind1.XRef, Type: gedcom.RecordTypeIndividual, Entity: ind1},
			{XRef: ind2.XRef, Type: gedcom.RecordTypeIndividual, Entity: ind2},
		},
	}

	// High confidence threshold should reject matches
	config := DefaultDuplicateConfig()
	config.MinConfidence = 0.99
	detector := NewDuplicateDetector(&config)
	duplicates := detector.FindDuplicates(doc)

	if len(duplicates) != 0 {
		t.Errorf("Expected 0 duplicates with high confidence threshold, got %d", len(duplicates))
	}

	// Low confidence threshold should accept matches
	config.MinConfidence = 0.3
	detector = NewDuplicateDetector(&config)
	duplicates = detector.FindDuplicates(doc)

	if len(duplicates) != 1 {
		t.Errorf("Expected 1 duplicate with low confidence threshold, got %d", len(duplicates))
	}
}

func TestFindDuplicates_NilDocument(t *testing.T) {
	detector := NewDuplicateDetector(nil)
	duplicates := detector.FindDuplicates(nil)

	if duplicates != nil {
		t.Errorf("Expected nil for nil document, got %v", duplicates)
	}
}

func TestFindDuplicates_SingleIndividual(t *testing.T) {
	ind := &gedcom.Individual{
		XRef:  "@I1@",
		Names: []*gedcom.PersonalName{{Full: "John /Doe/"}},
	}

	doc := &gedcom.Document{
		Records: []*gedcom.Record{
			{XRef: ind.XRef, Type: gedcom.RecordTypeIndividual, Entity: ind},
		},
	}

	detector := NewDuplicateDetector(nil)
	duplicates := detector.FindDuplicates(doc)

	if duplicates != nil {
		t.Errorf("Expected nil for single individual, got %v", duplicates)
	}
}

func TestFindDuplicates_EmptyDocument(t *testing.T) {
	doc := &gedcom.Document{
		Records: []*gedcom.Record{},
	}

	detector := NewDuplicateDetector(nil)
	duplicates := detector.FindDuplicates(doc)

	if duplicates != nil {
		t.Errorf("Expected nil for empty document, got %v", duplicates)
	}
}

func TestFindDuplicates_WithExplicitNameFields(t *testing.T) {
	// Use explicit Given and Surname fields instead of Full
	ind1 := &gedcom.Individual{
		XRef: "@I1@",
		Names: []*gedcom.PersonalName{{
			Given:   "John",
			Surname: "Doe",
		}},
		Sex: "M",
	}
	ind2 := &gedcom.Individual{
		XRef: "@I2@",
		Names: []*gedcom.PersonalName{{
			Given:   "John",
			Surname: "Doe",
		}},
		Sex: "M",
	}

	doc := &gedcom.Document{
		Records: []*gedcom.Record{
			{XRef: ind1.XRef, Type: gedcom.RecordTypeIndividual, Entity: ind1},
			{XRef: ind2.XRef, Type: gedcom.RecordTypeIndividual, Entity: ind2},
		},
	}

	detector := NewDuplicateDetector(nil)
	duplicates := detector.FindDuplicates(doc)

	if len(duplicates) != 1 {
		t.Fatalf("Expected 1 duplicate pair with explicit name fields, got %d", len(duplicates))
	}
}

func TestFindDuplicates_NormalizeAccents(t *testing.T) {
	// Create individuals with accented names that should match
	ind1 := &gedcom.Individual{
		XRef:  "@I1@",
		Names: []*gedcom.PersonalName{{Full: "Jos\u00e9 /M\u00fcller/"}}, // Jose with accent, Muller with umlaut
		Sex:   "M",
	}
	ind2 := &gedcom.Individual{
		XRef:  "@I2@",
		Names: []*gedcom.PersonalName{{Full: "Jose /Muller/"}}, // Without accents
		Sex:   "M",
	}

	doc := &gedcom.Document{
		Records: []*gedcom.Record{
			{XRef: ind1.XRef, Type: gedcom.RecordTypeIndividual, Entity: ind1},
			{XRef: ind2.XRef, Type: gedcom.RecordTypeIndividual, Entity: ind2},
		},
	}

	detector := NewDuplicateDetector(nil)
	duplicates := detector.FindDuplicates(doc)

	if len(duplicates) != 1 {
		t.Fatalf("Expected 1 duplicate pair with accented names, got %d", len(duplicates))
	}
}

func TestFindDuplicates_NoNormalization(t *testing.T) {
	// With normalization disabled, case matters
	ind1 := &gedcom.Individual{
		XRef:  "@I1@",
		Names: []*gedcom.PersonalName{{Full: "John /DOE/"}},
	}
	ind2 := &gedcom.Individual{
		XRef:  "@I2@",
		Names: []*gedcom.PersonalName{{Full: "John /Doe/"}},
	}

	doc := &gedcom.Document{
		Records: []*gedcom.Record{
			{XRef: ind1.XRef, Type: gedcom.RecordTypeIndividual, Entity: ind1},
			{XRef: ind2.XRef, Type: gedcom.RecordTypeIndividual, Entity: ind2},
		},
	}

	config := DefaultDuplicateConfig()
	config.NormalizeNames = false
	detector := NewDuplicateDetector(&config)
	duplicates := detector.FindDuplicates(doc)

	// Without normalization, DOE != Doe so they're in different groups
	if len(duplicates) != 0 {
		t.Errorf("Expected 0 duplicates without normalization (case differs), got %d", len(duplicates))
	}
}

func TestFindDuplicates_NoNormalizationStillMatches(t *testing.T) {
	// The companion to TestFindDuplicates_NoNormalization: with normalization
	// off, names that already agree must still match. Grouping and comparison
	// read the same precomputed keys, so this pins both halves to the same
	// NormalizeNames setting -- a match here proves the flag is honoured on the
	// matching path, not just the rejecting one.
	// Sex matches too: surname (0.3) plus given name (0.3) alone total 0.6,
	// which sits under the default MinConfidence of 0.7.
	ind1 := &gedcom.Individual{
		XRef:  "@I1@",
		Names: []*gedcom.PersonalName{{Full: "John /Doe/"}},
		Sex:   "M",
	}
	ind2 := &gedcom.Individual{
		XRef:  "@I2@",
		Names: []*gedcom.PersonalName{{Full: "John /Doe/"}},
		Sex:   "M",
	}

	doc := &gedcom.Document{
		Records: []*gedcom.Record{
			{XRef: ind1.XRef, Type: gedcom.RecordTypeIndividual, Entity: ind1},
			{XRef: ind2.XRef, Type: gedcom.RecordTypeIndividual, Entity: ind2},
		},
	}

	config := DefaultDuplicateConfig()
	config.NormalizeNames = false
	detector := NewDuplicateDetector(&config)
	duplicates := detector.FindDuplicates(doc)

	if len(duplicates) != 1 {
		t.Fatalf("Expected 1 duplicate without normalization (names identical), got %d", len(duplicates))
	}
}

func TestFindDuplicates_SurnamelessNeverMatch(t *testing.T) {
	// Individuals with no surname share the empty-string bucket without sharing
	// a surname. Every other signal agrees here -- same given name, same birth
	// year, same sex -- so only the missing surname can keep them apart. The
	// bucket is skipped outright for speed (#529); this pins the behaviour that
	// skip relies on, so a future change that starts pairing them fails loudly.
	birthDate, _ := gedcom.ParseDate("1 JAN 1850")
	newInd := func(xref string) *gedcom.Individual {
		return &gedcom.Individual{
			XRef:  xref,
			Names: []*gedcom.PersonalName{{Full: "John"}},
			Sex:   "M",
			Events: []*gedcom.Event{
				{Type: gedcom.EventBirth, ParsedDate: birthDate},
			},
		}
	}
	ind1 := newInd("@I1@")
	ind2 := newInd("@I2@")

	doc := &gedcom.Document{
		Records: []*gedcom.Record{
			{XRef: ind1.XRef, Type: gedcom.RecordTypeIndividual, Entity: ind1},
			{XRef: ind2.XRef, Type: gedcom.RecordTypeIndividual, Entity: ind2},
		},
	}

	detector := NewDuplicateDetector(nil)
	if duplicates := detector.FindDuplicates(doc); len(duplicates) != 0 {
		t.Errorf("Expected 0 duplicates for surname-less individuals, got %d", len(duplicates))
	}
}

func TestDuplicatePairToIssue(t *testing.T) {
	ind1 := &gedcom.Individual{
		XRef:  "@I1@",
		Names: []*gedcom.PersonalName{{Full: "John /Doe/"}},
	}
	ind2 := &gedcom.Individual{
		XRef:  "@I2@",
		Names: []*gedcom.PersonalName{{Full: "John /Doe/"}},
	}

	pair := DuplicatePair{
		Individual1:  ind1,
		Individual2:  ind2,
		Confidence:   0.85,
		MatchReasons: []string{"exact surname match", "exact given name match"},
	}

	issue := pair.ToIssue()

	if issue.Severity != SeverityInfo {
		t.Errorf("Expected SeverityInfo, got %v", issue.Severity)
	}
	if issue.Code != CodePotentialDuplicate {
		t.Errorf("Expected CodePotentialDuplicate, got %s", issue.Code)
	}
	if issue.RecordXRef != "@I1@" {
		t.Errorf("Expected RecordXRef @I1@, got %s", issue.RecordXRef)
	}
	if issue.RelatedXRef != "@I2@" {
		t.Errorf("Expected RelatedXRef @I2@, got %s", issue.RelatedXRef)
	}
	if issue.Details["confidence"] != "0.85" {
		t.Errorf("Expected confidence 0.85, got %s", issue.Details["confidence"])
	}
	if issue.Details["reason_1"] != "exact surname match" {
		t.Errorf("Expected reason_1 'exact surname match', got %s", issue.Details["reason_1"])
	}
}

func TestGetDisplayName(t *testing.T) {
	tests := []struct {
		name string
		ind  *gedcom.Individual
		want string
	}{
		{
			name: "full name with slashes",
			ind: &gedcom.Individual{
				XRef:  "@I1@",
				Names: []*gedcom.PersonalName{{Full: "John /Doe/"}},
			},
			want: "John Doe",
		},
		{
			name: "explicit given and surname",
			ind: &gedcom.Individual{
				XRef: "@I1@",
				Names: []*gedcom.PersonalName{{
					Given:   "John",
					Surname: "Doe",
				}},
			},
			want: "John Doe",
		},
		{
			name: "no names",
			ind: &gedcom.Individual{
				XRef:  "@I1@",
				Names: []*gedcom.PersonalName{},
			},
			want: "@I1@",
		},
		{
			name: "nil individual",
			ind:  nil,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getDisplayName(tt.ind)
			if got != tt.want {
				t.Errorf("getDisplayName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFindDuplicates_MultiplePairs(t *testing.T) {
	// Create 3 individuals that could all be duplicates of each other
	ind1 := &gedcom.Individual{
		XRef:  "@I1@",
		Names: []*gedcom.PersonalName{{Full: "John /Doe/"}},
		Sex:   "M",
	}
	ind2 := &gedcom.Individual{
		XRef:  "@I2@",
		Names: []*gedcom.PersonalName{{Full: "John /Doe/"}},
		Sex:   "M",
	}
	ind3 := &gedcom.Individual{
		XRef:  "@I3@",
		Names: []*gedcom.PersonalName{{Full: "John /Doe/"}},
		Sex:   "M",
	}

	doc := &gedcom.Document{
		Records: []*gedcom.Record{
			{XRef: ind1.XRef, Type: gedcom.RecordTypeIndividual, Entity: ind1},
			{XRef: ind2.XRef, Type: gedcom.RecordTypeIndividual, Entity: ind2},
			{XRef: ind3.XRef, Type: gedcom.RecordTypeIndividual, Entity: ind3},
		},
	}

	detector := NewDuplicateDetector(nil)
	duplicates := detector.FindDuplicates(doc)

	// Should find 3 pairs: (1,2), (1,3), (2,3)
	if len(duplicates) != 3 {
		t.Errorf("Expected 3 duplicate pairs, got %d", len(duplicates))
	}
}

func TestFindDuplicates_DifferentSurnameGroups(t *testing.T) {
	// Individuals from different surname groups should not be compared
	ind1 := &gedcom.Individual{
		XRef:  "@I1@",
		Names: []*gedcom.PersonalName{{Full: "John /Doe/"}},
		Sex:   "M",
	}
	ind2 := &gedcom.Individual{
		XRef:  "@I2@",
		Names: []*gedcom.PersonalName{{Full: "John /Smith/"}},
		Sex:   "M",
	}
	ind3 := &gedcom.Individual{
		XRef:  "@I3@",
		Names: []*gedcom.PersonalName{{Full: "John /Jones/"}},
		Sex:   "M",
	}
	// This one should match with ind1
	ind4 := &gedcom.Individual{
		XRef:  "@I4@",
		Names: []*gedcom.PersonalName{{Full: "John /Doe/"}},
		Sex:   "M",
	}

	doc := &gedcom.Document{
		Records: []*gedcom.Record{
			{XRef: ind1.XRef, Type: gedcom.RecordTypeIndividual, Entity: ind1},
			{XRef: ind2.XRef, Type: gedcom.RecordTypeIndividual, Entity: ind2},
			{XRef: ind3.XRef, Type: gedcom.RecordTypeIndividual, Entity: ind3},
			{XRef: ind4.XRef, Type: gedcom.RecordTypeIndividual, Entity: ind4},
		},
	}

	detector := NewDuplicateDetector(nil)
	duplicates := detector.FindDuplicates(doc)

	// Should only find 1 pair: ind1 and ind4 (both Doe)
	if len(duplicates) != 1 {
		t.Errorf("Expected 1 duplicate pair, got %d", len(duplicates))
	}

	if duplicates[0].Individual1.XRef != "@I1@" || duplicates[0].Individual2.XRef != "@I4@" {
		t.Errorf("Wrong individuals matched: %s and %s",
			duplicates[0].Individual1.XRef, duplicates[0].Individual2.XRef)
	}
}

func TestFindDuplicates_NoNameIndividuals(t *testing.T) {
	// Individuals without names should be handled gracefully
	ind1 := &gedcom.Individual{
		XRef:  "@I1@",
		Names: nil,
	}
	ind2 := &gedcom.Individual{
		XRef:  "@I2@",
		Names: []*gedcom.PersonalName{{Full: "John /Doe/"}},
	}

	doc := &gedcom.Document{
		Records: []*gedcom.Record{
			{XRef: ind1.XRef, Type: gedcom.RecordTypeIndividual, Entity: ind1},
			{XRef: ind2.XRef, Type: gedcom.RecordTypeIndividual, Entity: ind2},
		},
	}

	detector := NewDuplicateDetector(nil)
	duplicates := detector.FindDuplicates(doc)

	// No match should occur with nameless individual
	if len(duplicates) != 0 {
		t.Errorf("Expected 0 duplicates with nameless individual, got %d", len(duplicates))
	}
}

func TestAbsInt(t *testing.T) {
	tests := []struct {
		input int
		want  int
	}{
		{0, 0},
		{1, 1},
		{-1, 1},
		{100, 100},
		{-100, 100},
	}

	for _, tt := range tests {
		got := absInt(tt.input)
		if got != tt.want {
			t.Errorf("absInt(%d) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestMin(t *testing.T) {
	tests := []struct {
		a, b, c int
		want    int
	}{
		{1, 2, 3, 1},
		{3, 2, 1, 1},
		{2, 1, 3, 1},
		{5, 5, 5, 5},
		{0, 0, 0, 0},
		{-1, 0, 1, -1},
	}

	for _, tt := range tests {
		got := min(tt.a, tt.b, tt.c)
		if got != tt.want {
			t.Errorf("min(%d, %d, %d) = %d, want %d", tt.a, tt.b, tt.c, got, tt.want)
		}
	}
}

func TestMax(t *testing.T) {
	tests := []struct {
		a, b int
		want int
	}{
		{1, 2, 2},
		{2, 1, 2},
		{5, 5, 5},
		{0, 0, 0},
		{-1, 0, 0},
	}

	for _, tt := range tests {
		got := max(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("max(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

// Helper function for checking substrings in test assertions
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || substr == "" ||
		(s != "" && substr != "" && searchSubstring(s, substr)))
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// --- Bounded duplicate detection (#530) --------------------------------------

// singleSurnameSurname is the one surname singleSurnameDocument uses. The limit
// Issue reports the NORMALIZED group key, so tests assert on its lowercase form.
const singleSurnameSurname = "Ashworth"

// singleSurnameDocument builds a document whose individuals all share one
// surname: a single normalized-surname group of size n, which is exactly the
// shape DuplicateConfig.MaxGroupSize exists to bound (#530).
//
// Two fixture properties are load-bearing and easy to break:
//
//   - Given names are "P" plus the decimal index, so none of the incidental
//     pairs can match. Two four-character names differing in one character
//     score 1-1/4 = 0.75, below MinNameSimilarity; even the 0.8 that a
//     five-character pair can reach yields confidence 0.3+0.24+0.1 = 0.64,
//     below MinConfidence. Without that guarantee an accidental match would
//     make the pair counts below meaningless.
//   - The final two individuals share a given name, a birth year and a sex,
//     planting exactly ONE matching pair (confidence 0.9). A "the cap did not
//     fire" assertion needs a nonzero pair count, or it would pass just as
//     happily against a detector that compared nothing at all.
func singleSurnameDocument(n int) *gedcom.Document {
	individuals := make([]*gedcom.Individual, 0, n)

	for i := 0; i < n; i++ {
		ind := &gedcom.Individual{
			XRef:  "@I" + strconv.Itoa(i) + "@",
			Names: []*gedcom.PersonalName{{Given: "P" + strconv.Itoa(i), Surname: singleSurnameSurname}},
			Sex:   "M",
		}
		if n >= 2 && i >= n-2 {
			ind.Names[0].Given = "P" + strconv.Itoa(n-2)
			ind.Events = []*gedcom.Event{{Type: gedcom.EventBirth, ParsedDate: makeYearDate(1850)}}
		}
		individuals = append(individuals, ind)
	}

	return makeDocument(individuals, nil)
}

// oversizedGroupsDocument builds `groups` surname groups of `perGroup`
// individuals each, so any cap below perGroup skips every one of them. It
// exercises the aggregate limit Issue over more groups than that Issue is
// willing to name.
func oversizedGroupsDocument(groups, perGroup int) *gedcom.Document {
	individuals := make([]*gedcom.Individual, 0, groups*perGroup)

	for g := 0; g < groups; g++ {
		surname := "Surname" + string(rune('A'+g))
		for i := 0; i < perGroup; i++ {
			individuals = append(individuals, &gedcom.Individual{
				XRef:  fmt.Sprintf("@I%d_%d@", g, i),
				Names: []*gedcom.PersonalName{{Given: "P" + strconv.Itoa(i), Surname: surname}},
				Sex:   "M",
			})
		}
	}

	return makeDocument(individuals, nil)
}

// issueFingerprint renders an Issue's code, message and details in a fixed
// order so two runs over the same input can be compared byte for byte.
func issueFingerprint(issue *Issue) string {
	keys := make([]string, 0, len(issue.Details))
	for k := range issue.Details {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	b.WriteString(issue.Code)
	b.WriteString("|")
	b.WriteString(issue.Message)
	for _, k := range keys {
		b.WriteString("|" + k + "=" + issue.Details[k])
	}
	return b.String()
}

// TestMaxGroupSize_TriState pins the resolver's three cases. The rule is
// deliberately NOT the Go zero-value convention, and the deviation is the whole
// protection (see the MaxGroupSize field comment and
// docs/decisions/0009-bounded-duplicate-detection.md): opting out of the cap has
// to be explicit, so only a negative value means unlimited.
func TestMaxGroupSize_TriState(t *testing.T) {
	tests := []struct {
		name       string
		configured int
		want       int
	}{
		{"zero resolves to the default, not to unlimited", 0, DefaultMaxGroupSize},
		{"negative means unlimited", -1, math.MaxInt},
		{"a large negative is still just unlimited", -9999, math.MaxInt},
		{"positive is taken as given", 7, 7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDuplicateDetector(&DuplicateConfig{MaxGroupSize: tt.configured})
			if got := d.maxGroupSize(); got != tt.want {
				t.Errorf("maxGroupSize() = %d for MaxGroupSize=%d, want %d",
					got, tt.configured, tt.want)
			}
		})
	}

	// A nil config is the documented "give me the defaults" path, so it must
	// land on the cap rather than on the zero value's raw meaning.
	if got := NewDuplicateDetector(nil).maxGroupSize(); got != DefaultMaxGroupSize {
		t.Errorf("maxGroupSize() = %d for a nil config, want %d", got, DefaultMaxGroupSize)
	}
}

// TestFindDuplicatesReport_CapFires covers the bound doing its job: a surname
// group past the cap is skipped whole and reported.
//
// There is deliberately no timing assertion here. The saving is real -- nothing
// in the group is compared at all -- but wall-clock thresholds are flaky on
// shared CI, and the observable contract is the empty Pairs plus the Issue, not
// the clock.
func TestFindDuplicatesReport_CapFires(t *testing.T) {
	const n = DefaultMaxGroupSize + 1
	doc := singleSurnameDocument(n)

	report := NewDuplicateDetector(nil).FindDuplicatesReport(doc)

	// The group is skipped whole, not truncated, so even the planted pair at
	// the end of the fixture goes unreported. Truncating would depend on record
	// order and make a partial result indistinguishable from a complete one.
	if len(report.Pairs) != 0 {
		t.Errorf("got %d pairs from a skipped group, want 0", len(report.Pairs))
	}

	// One aggregate issue, never one per group: the output has to stay bounded
	// however many oversized groups a hostile document contains.
	if len(report.LimitIssues) != 1 {
		t.Fatalf("got %d limit issues, want exactly 1", len(report.LimitIssues))
	}
	issue := report.LimitIssues[0]

	if issue.Code != CodeDuplicateDetectionLimited {
		t.Errorf("Code = %q, want %q", issue.Code, CodeDuplicateDetectionLimited)
	}
	// Warning, not Info: at the default strictness a caller must learn that the
	// sweep was incomplete, or "no duplicates found" reads as "none exist".
	if issue.Severity != SeverityWarning {
		t.Errorf("Severity = %v, want SeverityWarning", issue.Severity)
	}

	wantDetails := map[string]string{
		"skipped_groups":      "1",
		"skipped_individuals": strconv.Itoa(n),
		"largest_group":       strconv.Itoa(n),
		// The RESOLVED ceiling, not the raw config field, because that is the
		// number that actually governed the decision.
		"max_group_size": strconv.Itoa(DefaultMaxGroupSize),
		// Normalized, because the normalized surname is the group key that was
		// skipped.
		"surnames": strings.ToLower(singleSurnameSurname),
	}
	for key, want := range wantDetails {
		if got := issue.Details[key]; got != want {
			t.Errorf("Details[%q] = %q, want %q", key, got, want)
		}
	}
}

// TestFindDuplicatesReport_CapDoesNotFire covers the other side of the
// boundary: a group exactly at the cap is compared in full, exactly as it was
// before #530. The comparison is `>`, not `>=`, and this is what pins that.
func TestFindDuplicatesReport_CapDoesNotFire(t *testing.T) {
	doc := singleSurnameDocument(DefaultMaxGroupSize)

	report := NewDuplicateDetector(nil).FindDuplicatesReport(doc)

	if len(report.LimitIssues) != 0 {
		t.Errorf("a group exactly at the cap was skipped: %d limit issue(s), want 0",
			len(report.LimitIssues))
	}
	if len(report.Pairs) != 1 {
		t.Errorf("got %d pairs, want the 1 planted duplicate", len(report.Pairs))
	}

	// FindDuplicates is now a thin wrapper over FindDuplicatesReport. Its
	// signature did not change, so callers predating #530 must still see the
	// same slice.
	if pairs := NewDuplicateDetector(nil).FindDuplicates(doc); len(pairs) != len(report.Pairs) {
		t.Errorf("FindDuplicates returned %d pairs, FindDuplicatesReport %d; they must agree",
			len(pairs), len(report.Pairs))
	}
}

// TestFindDuplicatesReport_NegativeIsUnlimited covers the explicit opt-out: a
// caller who asks for an unbounded sweep gets one, with no limit issue, and
// finds the duplicate the capped run above could not see.
func TestFindDuplicatesReport_NegativeIsUnlimited(t *testing.T) {
	doc := singleSurnameDocument(DefaultMaxGroupSize + 1)

	config := DefaultDuplicateConfig()
	config.MaxGroupSize = -1
	report := NewDuplicateDetector(&config).FindDuplicatesReport(doc)

	if len(report.LimitIssues) != 0 {
		t.Errorf("unlimited detection still reported %d limit issue(s), want 0",
			len(report.LimitIssues))
	}
	if len(report.Pairs) != 1 {
		t.Errorf("got %d pairs, want the 1 planted duplicate -- an unlimited sweep "+
			"must compare the oversized group", len(report.Pairs))
	}
}

// TestFindDuplicatesReport_PartialLiteralIsStillCapped guards the deliberate
// zero-value decision.
//
// &DuplicateConfig{MinConfidence: 0.7} leaves MaxGroupSize at zero without the
// caller ever thinking about it. Zero therefore resolves to DefaultMaxGroupSize
// and NOT to unlimited -- a departure from the Go convention, and from the
// MaxErrors precedent elsewhere in this package, made on purpose. If someone
// later "fixes" zero to mean unlimited, THIS TEST MUST FAIL LOUDLY: that change
// would silently reopen the unbounded quadratic sweep for every caller who
// writes a partial struct literal, which is the whole of #530.
func TestFindDuplicatesReport_PartialLiteralIsStillCapped(t *testing.T) {
	doc := singleSurnameDocument(DefaultMaxGroupSize + 1)

	report := NewDuplicateDetector(&DuplicateConfig{MinConfidence: 0.7}).FindDuplicatesReport(doc)

	if len(report.LimitIssues) != 1 {
		t.Fatalf("a partial config literal left detection uncapped: %d limit issues, want 1",
			len(report.LimitIssues))
	}
	want := strconv.Itoa(DefaultMaxGroupSize)
	if got := report.LimitIssues[0].Details["max_group_size"]; got != want {
		t.Errorf("Details[max_group_size] = %q, want %q: an unset field must report the "+
			"default that actually governed the skip", got, want)
	}
}

// TestFindDuplicatesReport_LimitIssueIsDeterministic pins the surname sample's
// ordering and truncation.
//
// The sample is drawn from a map, and Go randomizes map iteration order, so an
// unsorted sample would make the same document emit a different Issue on every
// run -- a real regression class that breaks anyone diffing or golden-testing
// validation output, and one a single-run assertion cannot see. The cap is set
// small so nothing is compared: the behavior under test is the sample, not the
// sweep.
func TestFindDuplicatesReport_LimitIssueIsDeterministic(t *testing.T) {
	const groups = 8 // more than maxSkippedSurnamesReported, so truncation runs
	const perGroup = 4

	doc := oversizedGroupsDocument(groups, perGroup)
	config := DefaultDuplicateConfig()
	config.MaxGroupSize = perGroup - 1

	var first string
	for run := 0; run < 20; run++ {
		report := NewDuplicateDetector(&config).FindDuplicatesReport(doc)
		if len(report.LimitIssues) != 1 {
			t.Fatalf("run %d: got %d limit issues, want 1", run, len(report.LimitIssues))
		}

		got := issueFingerprint(&report.LimitIssues[0])
		if run == 0 {
			first = got
			continue
		}
		if got != first {
			t.Fatalf("run %d emitted a different issue:\n got %s\nwant %s", run, got, first)
		}
	}

	issue := NewDuplicateDetector(&config).FindDuplicatesReport(doc).LimitIssues[0]

	// Every group is skipped and accounted for...
	if got, want := issue.Details["skipped_groups"], strconv.Itoa(groups); got != want {
		t.Errorf("Details[skipped_groups] = %q, want %q", got, want)
	}
	if got, want := issue.Details["skipped_individuals"], strconv.Itoa(groups*perGroup); got != want {
		t.Errorf("Details[skipped_individuals] = %q, want %q", got, want)
	}
	// ...but only the first five surnames are named. The detail is a sample for
	// diagnosis, not an inventory: an adversary can manufacture one oversized
	// group per MaxGroupSize individuals, so the message must not grow with them.
	wantSample := "surnamea, surnameb, surnamec, surnamed, surnamee"
	if got := issue.Details["surnames"]; got != wantSample {
		t.Errorf("Details[surnames] = %q, want %q (sorted, truncated to %d)",
			got, wantSample, maxSkippedSurnamesReported)
	}
}

// TestCompareGivenNames_EarlyExitEquivalence is the standing proof that the
// length-based early exit in compareGivenNames (#530) cannot change any
// duplicate the library reports.
//
// The exit skips the Levenshtein DP when an upper bound on the achievable
// similarity already falls below the caller's threshold, and returns that bound
// instead of the exact value. That is only legitimate because of two
// properties, both asserted here for every case:
//
//   - The returned value is either exactly what the full computation returns,
//     or it is below minSimilarity while the exact value is below it too. Since
//     the only consumer, comparePair, asks `givenSimilarity < MinNameSimilarity`,
//     the accept/reject verdict is identical either way.
//   - The returned value never UNDERSTATES the exact similarity. A bound that
//     dipped below the true value could reject a pair that ought to match.
//
// Passing minSimilarity <= 0 disables the exit, which is how the exact value is
// obtained here without reaching around the function under test.
func TestCompareGivenNames_EarlyExitEquivalence(t *testing.T) {
	tests := []struct {
		name          string
		g1, g2        string
		minSimilarity float64
	}{
		{"identical", "john", "john", 0.8},
		{"empty operand", "", "john", 0.8},
		{"equal lengths, clears the threshold", "john", "joan", 0.5},
		{"equal lengths, fails the threshold", "john", "mary", 0.8},
		{"wildly different lengths", "jo", "bartholomew", 0.8},
		{"wildly different lengths, reversed", "bartholomew", "jo", 0.8},
		// Rune count and byte count diverge from here down: the bound counts
		// runes to match the DP, over stringSimilarity's own byte denominator.
		{"accented, same rune count", "jose", "josé", 0.8},
		{"accented, different rune count", "josé", "jo", 0.8},
		{"multibyte, exit does not fire", "日本語", "京都", 0.8},
		{"multibyte, exit fires", "日本語である", "京", 0.8},
		{"emoji, rune and byte counts diverge", "jo🙂", "jo", 0.8},
		// 1 - |5-3|/5 = 0.6 exactly, and the exit tests `<`, so this one must
		// NOT fire and the exact value must come back...
		{"threshold boundary, exit must not fire", "abcde", "abc", 0.6},
		// ...while a hair above the same bound it must.
		{"threshold boundary, exit fires", "abcde", "abc", 0.61},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := compareGivenNames(tt.g1, tt.g2, tt.minSimilarity)
			exact := compareGivenNames(tt.g1, tt.g2, 0)

			if got != exact && !(got < tt.minSimilarity && exact < tt.minSimilarity) {
				t.Errorf("compareGivenNames(%q, %q, %v) = %v but the exact similarity is %v: "+
					"the early exit changed a reported value",
					tt.g1, tt.g2, tt.minSimilarity, got, exact)
			}
			if got < exact {
				t.Errorf("compareGivenNames(%q, %q, %v) = %v, below the exact similarity %v: "+
					"the bound must never understate", tt.g1, tt.g2, tt.minSimilarity, got, exact)
			}
			// Whatever path was taken, the accept/reject decision comparePair
			// makes must be the one the exact value would have produced.
			if (got < tt.minSimilarity) != (exact < tt.minSimilarity) {
				t.Errorf("compareGivenNames(%q, %q, %v) = %v flips the accept/reject verdict "+
					"against the exact %v", tt.g1, tt.g2, tt.minSimilarity, got, exact)
			}
		})
	}
}
