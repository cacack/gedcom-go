package validator

import (
	"testing"

	"github.com/cacack/gedcom-go/gedcom"
)

// Helper to create a parsed date from a simple year (for testing)
func makeYearDate(year int) *gedcom.Date {
	return &gedcom.Date{
		Original: string(rune('0'+year/1000)) + string(rune('0'+(year/100)%10)) + string(rune('0'+(year/10)%10)) + string(rune('0'+year%10)),
		Year:     year,
	}
}

// Helper to create an individual with birth and death dates
func makeIndividual(xref string, birthYear, deathYear int) *gedcom.Individual {
	ind := &gedcom.Individual{XRef: xref}
	if birthYear > 0 {
		ind.Events = append(ind.Events, &gedcom.Event{
			Type:       gedcom.EventBirth,
			ParsedDate: makeYearDate(birthYear),
		})
	}
	if deathYear > 0 {
		ind.Events = append(ind.Events, &gedcom.Event{
			Type:       gedcom.EventDeath,
			ParsedDate: makeYearDate(deathYear),
		})
	}
	return ind
}

// Helper to create a simple document with individuals and families
func makeDocument(individuals []*gedcom.Individual, families []*gedcom.Family) *gedcom.Document {
	doc := &gedcom.Document{
		Records: make([]*gedcom.Record, 0),
		XRefMap: make(map[string]*gedcom.Record),
	}

	for _, ind := range individuals {
		rec := &gedcom.Record{XRef: ind.XRef, Type: gedcom.RecordTypeIndividual, Entity: ind}
		doc.Records = append(doc.Records, rec)
		doc.XRefMap[ind.XRef] = rec
	}

	for _, fam := range families {
		rec := &gedcom.Record{XRef: fam.XRef, Type: gedcom.RecordTypeFamily, Entity: fam}
		doc.Records = append(doc.Records, rec)
		doc.XRefMap[fam.XRef] = rec
	}

	return doc
}

func TestDefaultDateLogicConfig(t *testing.T) {
	config := DefaultDateLogicConfig()

	if config.MaxReasonableAge != 120 {
		t.Errorf("MaxReasonableAge = %d, want 120", config.MaxReasonableAge)
	}
	if config.MinParentAge != 12 {
		t.Errorf("MinParentAge = %d, want 12", config.MinParentAge)
	}
	if config.MaxMotherAge != 55 {
		t.Errorf("MaxMotherAge = %d, want 55", config.MaxMotherAge)
	}
	if config.MaxFatherAge != 90 {
		t.Errorf("MaxFatherAge = %d, want 90", config.MaxFatherAge)
	}
}

func TestNewDateLogicValidator(t *testing.T) {
	tests := []struct {
		name          string
		config        *DateLogicConfig
		wantMaxAge    int
		wantMinParent int
		wantMaxMother int
		wantMaxFather int
	}{
		{
			name:          "nil config uses defaults",
			config:        nil,
			wantMaxAge:    120,
			wantMinParent: 12,
			wantMaxMother: 55,
			wantMaxFather: 90,
		},
		{
			name: "custom config",
			config: &DateLogicConfig{
				MaxReasonableAge: 110,
				MinParentAge:     14,
				MaxMotherAge:     50,
				MaxFatherAge:     80,
			},
			wantMaxAge:    110,
			wantMinParent: 14,
			wantMaxMother: 50,
			wantMaxFather: 80,
		},
		{
			name:          "zero values get defaults",
			config:        &DateLogicConfig{},
			wantMaxAge:    120,
			wantMinParent: 12,
			wantMaxMother: 55,
			wantMaxFather: 90,
		},
		{
			name: "partial config fills defaults",
			config: &DateLogicConfig{
				MaxReasonableAge: 100,
				// Other fields left at zero
			},
			wantMaxAge:    100,
			wantMinParent: 12,
			wantMaxMother: 55,
			wantMaxFather: 90,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewDateLogicValidator(tt.config)

			if v.config.MaxReasonableAge != tt.wantMaxAge {
				t.Errorf("MaxReasonableAge = %d, want %d", v.config.MaxReasonableAge, tt.wantMaxAge)
			}
			if v.config.MinParentAge != tt.wantMinParent {
				t.Errorf("MinParentAge = %d, want %d", v.config.MinParentAge, tt.wantMinParent)
			}
			if v.config.MaxMotherAge != tt.wantMaxMother {
				t.Errorf("MaxMotherAge = %d, want %d", v.config.MaxMotherAge, tt.wantMaxMother)
			}
			if v.config.MaxFatherAge != tt.wantMaxFather {
				t.Errorf("MaxFatherAge = %d, want %d", v.config.MaxFatherAge, tt.wantMaxFather)
			}
		})
	}
}

func TestDateLogicValidator_Validate_NilDocument(t *testing.T) {
	v := NewDateLogicValidator(nil)
	issues := v.Validate(nil)

	if issues != nil {
		t.Errorf("Validate(nil) should return nil, got %v", issues)
	}
}

func TestDateLogicValidator_ValidateIndividual_NilIndividual(t *testing.T) {
	v := NewDateLogicValidator(nil)
	issues := v.ValidateIndividual(nil, nil)

	if issues != nil {
		t.Errorf("ValidateIndividual(nil, nil) should return nil, got %v", issues)
	}
}

func TestDateLogicValidator_CheckDeathBeforeBirth(t *testing.T) {
	v := NewDateLogicValidator(nil)

	tests := []struct {
		name      string
		birthYear int
		deathYear int
		wantIssue bool
	}{
		{
			name:      "death before birth detected",
			birthYear: 1950,
			deathYear: 1940,
			wantIssue: true,
		},
		{
			name:      "normal lifespan no issue",
			birthYear: 1900,
			deathYear: 1980,
			wantIssue: false,
		},
		{
			name:      "same year no issue",
			birthYear: 1950,
			deathYear: 1950,
			wantIssue: false,
		},
		{
			name:      "no birth date no issue",
			birthYear: 0,
			deathYear: 1980,
			wantIssue: false,
		},
		{
			name:      "no death date no issue",
			birthYear: 1900,
			deathYear: 0,
			wantIssue: false,
		},
		{
			name:      "no dates no issue",
			birthYear: 0,
			deathYear: 0,
			wantIssue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ind := makeIndividual("@I1@", tt.birthYear, tt.deathYear)
			issue := v.checkDeathBeforeBirth(ind)

			if tt.wantIssue {
				if issue == nil {
					t.Error("expected issue, got nil")
					return
				}
				if issue.Code != CodeDeathBeforeBirth {
					t.Errorf("Code = %q, want %q", issue.Code, CodeDeathBeforeBirth)
				}
				if issue.Severity != SeverityError {
					t.Errorf("Severity = %v, want %v", issue.Severity, SeverityError)
				}
				if issue.RecordXRef != "@I1@" {
					t.Errorf("RecordXRef = %q, want %q", issue.RecordXRef, "@I1@")
				}
			} else if issue != nil {
				t.Errorf("expected no issue, got %v", issue)
			}
		})
	}
}

func TestDateLogicValidator_CheckChildBeforeParent(t *testing.T) {
	v := NewDateLogicValidator(nil)

	tests := []struct {
		name           string
		childBirth     int
		parentBirths   []int
		wantIssueCount int
	}{
		{
			name:           "child born before parent detected",
			childBirth:     1900,
			parentBirths:   []int{1920},
			wantIssueCount: 1,
		},
		{
			name:           "child born before both parents",
			childBirth:     1900,
			parentBirths:   []int{1920, 1925},
			wantIssueCount: 2,
		},
		{
			name:           "normal relationship no issue",
			childBirth:     1950,
			parentBirths:   []int{1920, 1925},
			wantIssueCount: 0,
		},
		{
			name:           "child has no birth date",
			childBirth:     0,
			parentBirths:   []int{1920},
			wantIssueCount: 0,
		},
		{
			name:           "parent has no birth date",
			childBirth:     1950,
			parentBirths:   []int{0},
			wantIssueCount: 0,
		},
		{
			name:           "no parents no issue",
			childBirth:     1950,
			parentBirths:   []int{},
			wantIssueCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create child
			child := makeIndividual("@I1@", tt.childBirth, 0)

			// Create parents and family
			var parents []*gedcom.Individual
			family := &gedcom.Family{XRef: "@F1@", Children: []string{"@I1@"}}

			for i, birthYear := range tt.parentBirths {
				parentXRef := "@I" + string(rune('2'+i)) + "@"
				parent := makeIndividual(parentXRef, birthYear, 0)
				if i == 0 {
					family.Husband = parentXRef
					parent.Sex = "M"
				} else {
					family.Wife = parentXRef
					parent.Sex = "F"
				}
				parents = append(parents, parent)
			}

			// Link child to family
			child.ChildInFamilies = []gedcom.FamilyLink{{FamilyXRef: "@F1@"}}

			// Create document
			allInds := append([]*gedcom.Individual{child}, parents...)
			doc := makeDocument(allInds, []*gedcom.Family{family})

			issues := v.checkChildBeforeParent(doc, child)

			if len(issues) != tt.wantIssueCount {
				t.Errorf("got %d issues, want %d", len(issues), tt.wantIssueCount)
			}

			for _, issue := range issues {
				if issue.Code != CodeChildBeforeParent {
					t.Errorf("Code = %q, want %q", issue.Code, CodeChildBeforeParent)
				}
				if issue.Severity != SeverityError {
					t.Errorf("Severity = %v, want %v", issue.Severity, SeverityError)
				}
			}
		})
	}
}

func TestDateLogicValidator_CheckMarriageBeforeBirth(t *testing.T) {
	v := NewDateLogicValidator(nil)

	tests := []struct {
		name         string
		birthYear    int
		marriageYear int
		wantIssue    bool
	}{
		{
			name:         "marriage before birth detected",
			birthYear:    1950,
			marriageYear: 1940,
			wantIssue:    true,
		},
		{
			name:         "normal marriage no issue",
			birthYear:    1950,
			marriageYear: 1975,
			wantIssue:    false,
		},
		{
			name:         "no birth date no issue",
			birthYear:    0,
			marriageYear: 1975,
			wantIssue:    false,
		},
		{
			name:         "no marriage date no issue",
			birthYear:    1950,
			marriageYear: 0,
			wantIssue:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create individual
			ind := makeIndividual("@I1@", tt.birthYear, 0)

			// Create spouse
			spouse := makeIndividual("@I2@", 1945, 0)

			// Create family with marriage event
			family := &gedcom.Family{
				XRef:    "@F1@",
				Husband: "@I1@",
				Wife:    "@I2@",
			}

			if tt.marriageYear > 0 {
				family.Events = []*gedcom.Event{{
					Type:       gedcom.EventMarriage,
					ParsedDate: makeYearDate(tt.marriageYear),
				}}
			}

			// Link individual to family as spouse
			ind.SpouseInFamilies = []string{"@F1@"}

			// Create document
			doc := makeDocument([]*gedcom.Individual{ind, spouse}, []*gedcom.Family{family})

			issues := v.checkMarriageBeforeBirth(doc, ind)

			if tt.wantIssue {
				if len(issues) != 1 {
					t.Errorf("got %d issues, want 1", len(issues))
					return
				}
				if issues[0].Code != CodeMarriageBeforeBirth {
					t.Errorf("Code = %q, want %q", issues[0].Code, CodeMarriageBeforeBirth)
				}
				if issues[0].Severity != SeverityError {
					t.Errorf("Severity = %v, want %v", issues[0].Severity, SeverityError)
				}
			} else if len(issues) != 0 {
				t.Errorf("expected no issues, got %d", len(issues))
			}
		})
	}
}

func TestDateLogicValidator_CheckReasonableAge(t *testing.T) {
	v := NewDateLogicValidator(nil) // Default max age is 120

	tests := []struct {
		name      string
		birthYear int
		deathYear int
		wantIssue bool
	}{
		{
			name:      "impossible age detected",
			birthYear: 1800,
			deathYear: 1950, // 150 years old
			wantIssue: true,
		},
		{
			name:      "reasonable age no issue",
			birthYear: 1900,
			deathYear: 1990, // 90 years old
			wantIssue: false,
		},
		{
			name:      "exactly 120 no issue",
			birthYear: 1800,
			deathYear: 1920, // 120 years old
			wantIssue: false,
		},
		{
			name:      "121 years flagged",
			birthYear: 1800,
			deathYear: 1921, // 121 years old
			wantIssue: true,
		},
		{
			name:      "no birth date no issue",
			birthYear: 0,
			deathYear: 1950,
			wantIssue: false,
		},
		{
			name:      "no death date no issue",
			birthYear: 1900,
			deathYear: 0,
			wantIssue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ind := makeIndividual("@I1@", tt.birthYear, tt.deathYear)
			issue := v.checkReasonableAge(ind)

			if tt.wantIssue {
				if issue == nil {
					t.Error("expected issue, got nil")
					return
				}
				if issue.Code != CodeImpossibleAge {
					t.Errorf("Code = %q, want %q", issue.Code, CodeImpossibleAge)
				}
				if issue.Severity != SeverityWarning {
					t.Errorf("Severity = %v, want %v", issue.Severity, SeverityWarning)
				}
			} else if issue != nil {
				t.Errorf("expected no issue, got %v", issue)
			}
		})
	}
}

func TestDateLogicValidator_CheckReasonableAge_CustomConfig(t *testing.T) {
	v := NewDateLogicValidator(&DateLogicConfig{MaxReasonableAge: 100})

	// 101 years old should be flagged with custom config
	ind := makeIndividual("@I1@", 1800, 1901)
	issue := v.checkReasonableAge(ind)

	if issue == nil {
		t.Error("expected issue for age 101 with max 100, got nil")
		return
	}

	if issue.Details["max_age"] != "100" {
		t.Errorf("max_age detail = %q, want %q", issue.Details["max_age"], "100")
	}
}

func TestDateLogicValidator_CheckReasonableParentAge(t *testing.T) {
	v := NewDateLogicValidator(nil)

	tests := []struct {
		name           string
		parentBirth    int
		parentSex      string
		childBirth     int
		wantIssueCount int
		wantCode       string
	}{
		{
			name:           "father too young",
			parentBirth:    1950,
			parentSex:      "M",
			childBirth:     1960, // Father is 10
			wantIssueCount: 1,
			wantCode:       CodeUnreasonableParentAge,
		},
		{
			name:           "mother too young",
			parentBirth:    1950,
			parentSex:      "F",
			childBirth:     1960, // Mother is 10
			wantIssueCount: 1,
			wantCode:       CodeUnreasonableParentAge,
		},
		{
			name:           "father too old",
			parentBirth:    1900,
			parentSex:      "M",
			childBirth:     1995, // Father is 95
			wantIssueCount: 1,
			wantCode:       CodeUnreasonableParentAge,
		},
		{
			name:           "mother too old",
			parentBirth:    1900,
			parentSex:      "F",
			childBirth:     1960, // Mother is 60
			wantIssueCount: 1,
			wantCode:       CodeUnreasonableParentAge,
		},
		{
			name:           "father at reasonable age",
			parentBirth:    1950,
			parentSex:      "M",
			childBirth:     1980, // Father is 30
			wantIssueCount: 0,
		},
		{
			name:           "mother at reasonable age",
			parentBirth:    1950,
			parentSex:      "F",
			childBirth:     1980, // Mother is 30
			wantIssueCount: 0,
		},
		{
			name:           "father at max age (90)",
			parentBirth:    1900,
			parentSex:      "M",
			childBirth:     1990, // Father is 90
			wantIssueCount: 0,
		},
		{
			name:           "mother at max age (55)",
			parentBirth:    1900,
			parentSex:      "F",
			childBirth:     1955, // Mother is 55
			wantIssueCount: 0,
		},
		{
			name:           "parent at minimum age (12)",
			parentBirth:    1950,
			parentSex:      "F",
			childBirth:     1962, // Mother is 12
			wantIssueCount: 0,
		},
		{
			name:           "unknown sex uses father limits",
			parentBirth:    1900,
			parentSex:      "U",
			childBirth:     1995, // Age 95 (exceeds father max of 90)
			wantIssueCount: 1,
			wantCode:       CodeUnreasonableParentAge,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create parent
			parent := makeIndividual("@I1@", tt.parentBirth, 0)
			parent.Sex = tt.parentSex

			// Create child
			child := makeIndividual("@I2@", tt.childBirth, 0)

			// Create family
			family := &gedcom.Family{
				XRef:     "@F1@",
				Children: []string{"@I2@"},
			}
			if tt.parentSex == "M" || tt.parentSex == "U" {
				family.Husband = "@I1@"
			} else {
				family.Wife = "@I1@"
			}

			// Link parent to family as spouse
			parent.SpouseInFamilies = []string{"@F1@"}

			// Create document
			doc := makeDocument([]*gedcom.Individual{parent, child}, []*gedcom.Family{family})

			issues := v.checkReasonableParentAge(doc, parent)

			if len(issues) != tt.wantIssueCount {
				t.Errorf("got %d issues, want %d", len(issues), tt.wantIssueCount)
			}

			if tt.wantIssueCount > 0 && len(issues) > 0 {
				if issues[0].Code != tt.wantCode {
					t.Errorf("Code = %q, want %q", issues[0].Code, tt.wantCode)
				}
				if issues[0].Severity != SeverityWarning {
					t.Errorf("Severity = %v, want %v", issues[0].Severity, SeverityWarning)
				}
			}
		})
	}
}

func TestDateLogicValidator_Validate_Integration(t *testing.T) {
	v := NewDateLogicValidator(nil)

	// Create a complex family with multiple issues
	// Parent born 1950, child born 1940 (child before parent)
	// Child has death before birth (died 1935, born 1940)

	parent := makeIndividual("@I1@", 1950, 0)
	parent.Sex = "M"

	child := &gedcom.Individual{
		XRef: "@I2@",
		Events: []*gedcom.Event{
			{Type: gedcom.EventBirth, ParsedDate: makeYearDate(1940)},
			{Type: gedcom.EventDeath, ParsedDate: makeYearDate(1935)},
		},
		ChildInFamilies: []gedcom.FamilyLink{{FamilyXRef: "@F1@"}},
	}

	family := &gedcom.Family{
		XRef:     "@F1@",
		Husband:  "@I1@",
		Children: []string{"@I2@"},
	}
	parent.SpouseInFamilies = []string{"@F1@"}

	doc := makeDocument([]*gedcom.Individual{parent, child}, []*gedcom.Family{family})

	issues := v.Validate(doc)

	// Should find:
	// 1. Child born before parent (from parent's perspective via checkReasonableParentAge - too young)
	// 2. Death before birth for child
	// 3. Child before parent (from child's perspective)

	if len(issues) < 2 {
		t.Errorf("expected at least 2 issues, got %d: %v", len(issues), issues)
	}

	// Verify we got the expected error codes
	codes := make(map[string]bool)
	for _, issue := range issues {
		codes[issue.Code] = true
	}

	if !codes[CodeDeathBeforeBirth] {
		t.Error("expected DEATH_BEFORE_BIRTH issue")
	}
	if !codes[CodeChildBeforeParent] {
		t.Error("expected CHILD_BEFORE_PARENT issue")
	}
}

func TestDateLogicValidator_PartialDates(t *testing.T) {
	v := NewDateLogicValidator(nil)

	// Test that year-only dates still work for comparisons
	ind := &gedcom.Individual{
		XRef: "@I1@",
		Events: []*gedcom.Event{
			{
				Type: gedcom.EventBirth,
				ParsedDate: &gedcom.Date{
					Original: "1950",
					Year:     1950,
					// Month and Day are 0 (partial date)
				},
			},
			{
				Type: gedcom.EventDeath,
				ParsedDate: &gedcom.Date{
					Original: "1940",
					Year:     1940,
					// Month and Day are 0 (partial date)
				},
			},
		},
	}

	issue := v.checkDeathBeforeBirth(ind)

	if issue == nil {
		t.Error("expected death before birth issue for partial dates")
		return
	}

	if issue.Code != CodeDeathBeforeBirth {
		t.Errorf("Code = %q, want %q", issue.Code, CodeDeathBeforeBirth)
	}
}

func TestDateLogicValidator_MissingDates(t *testing.T) {
	v := NewDateLogicValidator(nil)

	// Individual with no dates at all
	ind := &gedcom.Individual{XRef: "@I1@"}

	doc := makeDocument([]*gedcom.Individual{ind}, nil)

	issues := v.ValidateIndividual(doc, ind)

	if len(issues) != 0 {
		t.Errorf("expected no issues for individual without dates, got %d: %v", len(issues), issues)
	}
}

func TestDateLogicValidator_NilDocument(t *testing.T) {
	v := NewDateLogicValidator(nil)

	ind := makeIndividual("@I1@", 1950, 0)

	// These methods should handle nil doc gracefully
	issues := v.checkChildBeforeParent(nil, ind)
	if len(issues) != 0 {
		t.Errorf("checkChildBeforeParent with nil doc should return empty, got %d issues", len(issues))
	}

	issues = v.checkMarriageBeforeBirth(nil, ind)
	if len(issues) != 0 {
		t.Errorf("checkMarriageBeforeBirth with nil doc should return empty, got %d issues", len(issues))
	}

	issues = v.checkReasonableParentAge(nil, ind)
	if len(issues) != 0 {
		t.Errorf("checkReasonableParentAge with nil doc should return empty, got %d issues", len(issues))
	}
}

func TestDateLogicValidator_IssueDetails(t *testing.T) {
	v := NewDateLogicValidator(nil)

	// Test that issue details contain expected information
	ind := &gedcom.Individual{
		XRef: "@I1@",
		Events: []*gedcom.Event{
			{
				Type: gedcom.EventBirth,
				ParsedDate: &gedcom.Date{
					Original: "15 JAN 1950",
					Year:     1950,
					Month:    1,
					Day:      15,
				},
			},
			{
				Type: gedcom.EventDeath,
				ParsedDate: &gedcom.Date{
					Original: "10 DEC 1940",
					Year:     1940,
					Month:    12,
					Day:      10,
				},
			},
		},
	}

	issue := v.checkDeathBeforeBirth(ind)

	if issue == nil {
		t.Fatal("expected issue, got nil")
	}

	// Check details are populated
	if issue.Details["birth_date"] != "15 JAN 1950" {
		t.Errorf("birth_date detail = %q, want %q", issue.Details["birth_date"], "15 JAN 1950")
	}
	if issue.Details["death_date"] != "10 DEC 1940" {
		t.Errorf("death_date detail = %q, want %q", issue.Details["death_date"], "10 DEC 1940")
	}

	// Check message contains the dates
	if issue.Message == "" {
		t.Error("Message should not be empty")
	}
}

func TestDateLogicValidator_EmptyDocument(t *testing.T) {
	v := NewDateLogicValidator(nil)

	doc := &gedcom.Document{
		Records: []*gedcom.Record{},
		XRefMap: make(map[string]*gedcom.Record),
	}

	issues := v.Validate(doc)

	if len(issues) != 0 {
		t.Errorf("expected no issues for empty document, got %d", len(issues))
	}
}

func TestDateLogicValidator_MultipleMarriages(t *testing.T) {
	v := NewDateLogicValidator(nil)

	// Individual with two marriages, one before birth
	ind := makeIndividual("@I1@", 1950, 0)
	spouse1 := makeIndividual("@I2@", 1948, 0)
	spouse2 := makeIndividual("@I3@", 1952, 0)

	family1 := &gedcom.Family{
		XRef:    "@F1@",
		Husband: "@I1@",
		Wife:    "@I2@",
		Events: []*gedcom.Event{{
			Type:       gedcom.EventMarriage,
			ParsedDate: makeYearDate(1940), // Before birth
		}},
	}

	family2 := &gedcom.Family{
		XRef:    "@F2@",
		Husband: "@I1@",
		Wife:    "@I3@",
		Events: []*gedcom.Event{{
			Type:       gedcom.EventMarriage,
			ParsedDate: makeYearDate(1975), // Normal
		}},
	}

	ind.SpouseInFamilies = []string{"@F1@", "@F2@"}

	doc := makeDocument(
		[]*gedcom.Individual{ind, spouse1, spouse2},
		[]*gedcom.Family{family1, family2},
	)

	issues := v.checkMarriageBeforeBirth(doc, ind)

	// Should only flag the first marriage
	if len(issues) != 1 {
		t.Errorf("expected 1 issue, got %d", len(issues))
	}

	if len(issues) > 0 && issues[0].RelatedXRef != "@F1@" {
		t.Errorf("RelatedXRef = %q, want %q", issues[0].RelatedXRef, "@F1@")
	}
}

func TestDateLogicValidator_MultipleChildren(t *testing.T) {
	v := NewDateLogicValidator(nil)

	// Parent with multiple children, some at unreasonable ages
	parent := makeIndividual("@I1@", 1950, 0)
	parent.Sex = "F"

	child1 := makeIndividual("@I2@", 1960, 0) // Parent age 10 - too young
	child2 := makeIndividual("@I3@", 1980, 0) // Parent age 30 - ok
	child3 := makeIndividual("@I4@", 2010, 0) // Parent age 60 - too old for mother

	family := &gedcom.Family{
		XRef:     "@F1@",
		Wife:     "@I1@",
		Children: []string{"@I2@", "@I3@", "@I4@"},
	}

	parent.SpouseInFamilies = []string{"@F1@"}

	doc := makeDocument(
		[]*gedcom.Individual{parent, child1, child2, child3},
		[]*gedcom.Family{family},
	)

	issues := v.checkReasonableParentAge(doc, parent)

	// Should flag child1 (too young) and child3 (too old)
	if len(issues) != 2 {
		t.Errorf("expected 2 issues, got %d: %v", len(issues), issues)
	}
}

// Test to ensure new error code is included in tests
func TestErrorCodeUnreasonableParentAge(t *testing.T) {
	if CodeUnreasonableParentAge == "" {
		t.Error("CodeUnreasonableParentAge should not be empty")
	}
	if CodeUnreasonableParentAge != "UNREASONABLE_PARENT_AGE" {
		t.Errorf("CodeUnreasonableParentAge = %q, want %q", CodeUnreasonableParentAge, "UNREASONABLE_PARENT_AGE")
	}
}
