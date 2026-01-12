package validator

import (
	"testing"

	"github.com/cacack/gedcom-go/gedcom"
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
