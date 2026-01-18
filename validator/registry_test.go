package validator

import (
	"regexp"
	"testing"
)

func TestNewTagRegistry(t *testing.T) {
	r := NewTagRegistry()
	if r == nil {
		t.Fatal("NewTagRegistry returned nil")
	}
	if r.tags == nil {
		t.Error("tags map should be initialized")
	}
	if len(r.tags) != 0 {
		t.Errorf("new registry should be empty, got %d tags", len(r.tags))
	}
}

func TestTagRegistry_Register(t *testing.T) {
	tests := []struct {
		name    string
		tag     string
		def     TagDefinition
		wantErr bool
	}{
		{
			name: "register simple tag",
			tag:  "_MILT",
			def: TagDefinition{
				AllowedParents: []string{"INDI"},
				Description:    "Military service",
			},
			wantErr: false,
		},
		{
			name: "register tag with pattern",
			tag:  "_PRIM",
			def: TagDefinition{
				AllowedParents: []string{"OBJE"},
				ValuePattern:   YesNoPattern,
				Description:    "Primary flag",
			},
			wantErr: false,
		},
		{
			name: "register tag without parents (allowed anywhere)",
			tag:  "_CUSTOM",
			def: TagDefinition{
				Description: "Custom tag allowed anywhere",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewTagRegistry()
			err := r.Register(tt.tag, tt.def)

			if (err != nil) != tt.wantErr {
				t.Errorf("Register() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify tag was registered
				got, ok := r.Get(tt.tag)
				if !ok {
					t.Error("registered tag not found")
					return
				}
				// Tag field should be set to the key
				if got.Tag != tt.tag {
					t.Errorf("Tag field = %q, want %q", got.Tag, tt.tag)
				}
				if got.Description != tt.def.Description {
					t.Errorf("Description = %q, want %q", got.Description, tt.def.Description)
				}
			}
		})
	}
}

func TestTagRegistry_RegisterDuplicate(t *testing.T) {
	r := NewTagRegistry()

	// Register first time
	err := r.Register("_MILT", TagDefinition{Description: "First"})
	if err != nil {
		t.Fatalf("first Register() unexpected error: %v", err)
	}

	// Register same tag again
	err = r.Register("_MILT", TagDefinition{Description: "Second"})
	if err == nil {
		t.Error("expected error for duplicate registration, got nil")
	}

	// Verify original definition is preserved
	def, _ := r.Get("_MILT")
	if def.Description != "First" {
		t.Errorf("original definition should be preserved, got %q", def.Description)
	}
}

func TestTagRegistry_RegisterBatch(t *testing.T) {
	t.Run("successful batch registration", func(t *testing.T) {
		r := NewTagRegistry()
		defs := map[string]TagDefinition{
			"_TAG1": {Description: "Tag 1"},
			"_TAG2": {Description: "Tag 2"},
			"_TAG3": {Description: "Tag 3"},
		}

		err := r.RegisterBatch(defs)
		if err != nil {
			t.Fatalf("RegisterBatch() unexpected error: %v", err)
		}

		if r.Len() != 3 {
			t.Errorf("expected 3 tags, got %d", r.Len())
		}

		for tag, expected := range defs {
			got, ok := r.Get(tag)
			if !ok {
				t.Errorf("tag %s not found", tag)
				continue
			}
			if got.Tag != tag {
				t.Errorf("tag %s: Tag field = %q, want %q", tag, got.Tag, tag)
			}
			if got.Description != expected.Description {
				t.Errorf("tag %s: Description = %q, want %q", tag, got.Description, expected.Description)
			}
		}
	})

	t.Run("batch registration with duplicate fails atomically", func(t *testing.T) {
		r := NewTagRegistry()

		// Pre-register one tag
		err := r.Register("_EXISTING", TagDefinition{Description: "Existing"})
		if err != nil {
			t.Fatalf("pre-register error: %v", err)
		}

		// Try to batch register including the existing tag
		defs := map[string]TagDefinition{
			"_NEW1":     {Description: "New 1"},
			"_EXISTING": {Description: "Duplicate"},
			"_NEW2":     {Description: "New 2"},
		}

		err = r.RegisterBatch(defs)
		if err == nil {
			t.Error("expected error for batch with duplicate, got nil")
		}

		// None of the new tags should be registered (atomic)
		if r.Len() != 1 {
			t.Errorf("expected 1 tag (only pre-existing), got %d", r.Len())
		}
		if r.IsKnown("_NEW1") {
			t.Error("_NEW1 should not be registered due to atomic failure")
		}
		if r.IsKnown("_NEW2") {
			t.Error("_NEW2 should not be registered due to atomic failure")
		}
	})

	t.Run("empty batch", func(t *testing.T) {
		r := NewTagRegistry()
		err := r.RegisterBatch(map[string]TagDefinition{})
		if err != nil {
			t.Errorf("empty batch should succeed, got error: %v", err)
		}
		if r.Len() != 0 {
			t.Errorf("registry should remain empty, got %d tags", r.Len())
		}
	})
}

func TestTagRegistry_Get(t *testing.T) {
	r := NewTagRegistry()
	_ = r.Register("_MILT", TagDefinition{
		AllowedParents: []string{"INDI"},
		Description:    "Military service",
	})

	t.Run("found", func(t *testing.T) {
		def, ok := r.Get("_MILT")
		if !ok {
			t.Error("expected to find registered tag")
		}
		if def.Tag != "_MILT" {
			t.Errorf("Tag = %q, want %q", def.Tag, "_MILT")
		}
		if def.Description != "Military service" {
			t.Errorf("Description = %q, want %q", def.Description, "Military service")
		}
	})

	t.Run("not found", func(t *testing.T) {
		def, ok := r.Get("_NONEXISTENT")
		if ok {
			t.Error("expected not to find unregistered tag")
		}
		if def.Tag != "" {
			t.Errorf("expected empty TagDefinition, got Tag = %q", def.Tag)
		}
	})
}

func TestTagRegistry_IsKnown(t *testing.T) {
	r := NewTagRegistry()
	_ = r.Register("_MILT", TagDefinition{Description: "Military"})

	tests := []struct {
		tag  string
		want bool
	}{
		{"_MILT", true},
		{"_NONEXISTENT", false},
		{"MILT", false}, // Without underscore
		{"", false},     // Empty string
	}

	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			got := r.IsKnown(tt.tag)
			if got != tt.want {
				t.Errorf("IsKnown(%q) = %v, want %v", tt.tag, got, tt.want)
			}
		})
	}
}

func TestTagRegistry_Tags(t *testing.T) {
	r := NewTagRegistry()

	t.Run("empty registry", func(t *testing.T) {
		tags := r.Tags()
		if len(tags) != 0 {
			t.Errorf("expected empty slice, got %v", tags)
		}
	})

	// Register tags in non-alphabetical order
	_ = r.Register("_ZEBRA", TagDefinition{})
	_ = r.Register("_ALPHA", TagDefinition{})
	_ = r.Register("_MIDDLE", TagDefinition{})

	t.Run("returns sorted list", func(t *testing.T) {
		tags := r.Tags()
		if len(tags) != 3 {
			t.Fatalf("expected 3 tags, got %d", len(tags))
		}

		expected := []string{"_ALPHA", "_MIDDLE", "_ZEBRA"}
		for i, want := range expected {
			if tags[i] != want {
				t.Errorf("tags[%d] = %q, want %q", i, tags[i], want)
			}
		}
	})
}

func TestTagRegistry_Len(t *testing.T) {
	r := NewTagRegistry()

	if r.Len() != 0 {
		t.Errorf("empty registry Len() = %d, want 0", r.Len())
	}

	_ = r.Register("_TAG1", TagDefinition{})
	if r.Len() != 1 {
		t.Errorf("after 1 register, Len() = %d, want 1", r.Len())
	}

	_ = r.Register("_TAG2", TagDefinition{})
	if r.Len() != 2 {
		t.Errorf("after 2 registers, Len() = %d, want 2", r.Len())
	}

	// Duplicate should not increase length
	_ = r.Register("_TAG1", TagDefinition{})
	if r.Len() != 2 {
		t.Errorf("after duplicate register, Len() = %d, want 2", r.Len())
	}
}

func TestTagRegistry_ValidateTag_ValidParent(t *testing.T) {
	r := NewTagRegistry()
	_ = r.Register("_MILT", TagDefinition{
		AllowedParents: []string{"INDI", "FAM"},
		Description:    "Military service",
	})

	tests := []struct {
		name   string
		parent string
	}{
		{"INDI parent", "INDI"},
		{"FAM parent", "FAM"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issue := r.ValidateTag("_MILT", tt.parent, "Army")
			if issue != nil {
				t.Errorf("expected no issue for valid parent %s, got %v", tt.parent, issue)
			}
		})
	}
}

func TestTagRegistry_ValidateTag_InvalidParent(t *testing.T) {
	r := NewTagRegistry()
	_ = r.Register("_MILT", TagDefinition{
		AllowedParents: []string{"INDI"},
		Description:    "Military service",
	})

	issue := r.ValidateTag("_MILT", "FAM", "Army")
	if issue == nil {
		t.Fatal("expected issue for invalid parent, got nil")
	}

	if issue.Code != CodeInvalidTagParent {
		t.Errorf("Code = %q, want %q", issue.Code, CodeInvalidTagParent)
	}
	if issue.Severity != SeverityWarning {
		t.Errorf("Severity = %v, want %v", issue.Severity, SeverityWarning)
	}
	if issue.Details["tag"] != "_MILT" {
		t.Errorf("tag detail = %q, want %q", issue.Details["tag"], "_MILT")
	}
	if issue.Details["parent"] != "FAM" {
		t.Errorf("parent detail = %q, want %q", issue.Details["parent"], "FAM")
	}
}

func TestTagRegistry_ValidateTag_ValuePattern(t *testing.T) {
	r := NewTagRegistry()
	_ = r.Register("_PRIM", TagDefinition{
		ValuePattern: YesNoPattern,
		Description:  "Primary flag",
	})

	t.Run("valid values", func(t *testing.T) {
		for _, val := range []string{"Y", "N"} {
			issue := r.ValidateTag("_PRIM", "OBJE", val)
			if issue != nil {
				t.Errorf("expected no issue for value %q, got %v", val, issue)
			}
		}
	})

	t.Run("invalid values", func(t *testing.T) {
		for _, val := range []string{"YES", "NO", "y", "n", "1", "true"} {
			issue := r.ValidateTag("_PRIM", "OBJE", val)
			if issue == nil {
				t.Errorf("expected issue for invalid value %q, got nil", val)
				continue
			}
			if issue.Code != CodeInvalidTagValue {
				t.Errorf("value %q: Code = %q, want %q", val, issue.Code, CodeInvalidTagValue)
			}
			if issue.Details["value"] != val {
				t.Errorf("value detail = %q, want %q", issue.Details["value"], val)
			}
		}
	})

	t.Run("empty value skips pattern check", func(t *testing.T) {
		issue := r.ValidateTag("_PRIM", "OBJE", "")
		if issue != nil {
			t.Errorf("empty value should not be validated against pattern, got %v", issue)
		}
	})
}

func TestTagRegistry_ValidateTag_EmptyAllowedParents(t *testing.T) {
	r := NewTagRegistry()
	_ = r.Register("_ANYWHERE", TagDefinition{
		AllowedParents: []string{}, // Empty = allowed anywhere
		Description:    "Tag allowed anywhere",
	})

	// Should be valid under any parent
	parents := []string{"INDI", "FAM", "SOUR", "NOTE", "BIRT", "DEAT"}
	for _, parent := range parents {
		issue := r.ValidateTag("_ANYWHERE", parent, "value")
		if issue != nil {
			t.Errorf("expected no issue for parent %s with empty AllowedParents, got %v", parent, issue)
		}
	}
}

func TestTagRegistry_ValidateTag_NilPattern(t *testing.T) {
	r := NewTagRegistry()
	_ = r.Register("_NOPATTERN", TagDefinition{
		AllowedParents: []string{"INDI"},
		ValuePattern:   nil, // No pattern = any value accepted
		Description:    "Tag with no value restriction",
	})

	// Should accept any value
	values := []string{"anything", "123", "Y", "special!@#$", "multi word value"}
	for _, val := range values {
		issue := r.ValidateTag("_NOPATTERN", "INDI", val)
		if issue != nil {
			t.Errorf("expected no issue for value %q with nil pattern, got %v", val, issue)
		}
	}
}

func TestTagRegistry_ValidateTag_UnknownTag(t *testing.T) {
	r := NewTagRegistry()

	// Validating an unknown tag should return nil (no issue)
	// The registry only validates registered tags
	issue := r.ValidateTag("_UNKNOWN", "INDI", "value")
	if issue != nil {
		t.Errorf("unknown tag should return nil (not validated by registry), got %v", issue)
	}
}

func TestXRefPattern(t *testing.T) {
	validXRefs := []string{
		"@I1@",
		"@F123@",
		"@S1@",
		"@R_1@",
		"@ABC123@",
		"@a1@",
		"@LONG_XREF_123@",
	}

	invalidXRefs := []string{
		"I1",       // No @ signs
		"@I1",      // Missing closing @
		"I1@",      // Missing opening @
		"@@",       // Empty content
		"@ @",      // Space in content
		"@I 1@",    // Space in content
		"@I-1@",    // Hyphen not allowed
		"@I.1@",    // Period not allowed
		"",         // Empty string
		"@I1@ ",    // Trailing space
		" @I1@",    // Leading space
		"@I1@@I2@", // Multiple xrefs
	}

	for _, xref := range validXRefs {
		t.Run("valid_"+xref, func(t *testing.T) {
			if !XRefPattern.MatchString(xref) {
				t.Errorf("XRefPattern should match %q", xref)
			}
		})
	}

	for _, xref := range invalidXRefs {
		name := xref
		if name == "" {
			name = "empty"
		}
		t.Run("invalid_"+name, func(t *testing.T) {
			if XRefPattern.MatchString(xref) {
				t.Errorf("XRefPattern should not match %q", xref)
			}
		})
	}
}

func TestYesNoPattern(t *testing.T) {
	validValues := []string{"Y", "N"}

	invalidValues := []string{
		"y",
		"n",
		"YES",
		"NO",
		"yes",
		"no",
		"1",
		"0",
		"true",
		"false",
		"T",
		"F",
		"",
		" Y",
		"Y ",
		"YN",
	}

	for _, val := range validValues {
		t.Run("valid_"+val, func(t *testing.T) {
			if !YesNoPattern.MatchString(val) {
				t.Errorf("YesNoPattern should match %q", val)
			}
		})
	}

	for _, val := range invalidValues {
		name := val
		if name == "" {
			name = "empty"
		}
		t.Run("invalid_"+name, func(t *testing.T) {
			if YesNoPattern.MatchString(val) {
				t.Errorf("YesNoPattern should not match %q", val)
			}
		})
	}
}

func TestTagRegistry_ValidateTag_CombinedRestrictions(t *testing.T) {
	r := NewTagRegistry()
	_ = r.Register("_STRICT", TagDefinition{
		AllowedParents: []string{"INDI"},
		ValuePattern:   regexp.MustCompile(`^\d{4}$`), // Year only
		Description:    "Strict tag with both restrictions",
	})

	tests := []struct {
		name      string
		parent    string
		value     string
		wantCode  string
		wantIssue bool
	}{
		{"valid parent and value", "INDI", "1985", "", false},
		{"invalid parent, valid value", "FAM", "1985", CodeInvalidTagParent, true},
		{"valid parent, invalid value", "INDI", "85", CodeInvalidTagValue, true},
		{"invalid parent, invalid value", "FAM", "invalid", CodeInvalidTagParent, true}, // Parent checked first
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issue := r.ValidateTag("_STRICT", tt.parent, tt.value)

			if tt.wantIssue {
				if issue == nil {
					t.Error("expected issue, got nil")
					return
				}
				if issue.Code != tt.wantCode {
					t.Errorf("Code = %q, want %q", issue.Code, tt.wantCode)
				}
			} else if issue != nil {
				t.Errorf("expected no issue, got %v", issue)
			}
		})
	}
}
