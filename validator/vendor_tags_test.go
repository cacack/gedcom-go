package validator

import (
	"testing"

	"github.com/cacack/gedcom-go/gedcom"
)

func TestAncestryRegistry(t *testing.T) {
	r := AncestryRegistry()

	if r == nil {
		t.Fatal("AncestryRegistry returned nil")
	}

	// Expected Ancestry tags
	expectedTags := []struct {
		tag            string
		allowedParents []string
		hasPattern     bool
	}{
		{"_APID", []string{"SOUR"}, false},
		{"_TREE", []string{"HEAD"}, false},
		{"_MILT", []string{"INDI"}, false},
		{"_DEST", []string{"EMIG", "IMMI"}, false},
		{"_PRIM", []string{"OBJE"}, true}, // Has YesNoPattern
		{"_PHOTO", []string{"INDI"}, false},
	}

	for _, expected := range expectedTags {
		t.Run(expected.tag, func(t *testing.T) {
			if !r.IsKnown(expected.tag) {
				t.Errorf("AncestryRegistry should contain %s", expected.tag)
				return
			}

			def, _ := r.Get(expected.tag)

			// Check allowed parents
			if len(expected.allowedParents) != len(def.AllowedParents) {
				t.Errorf("%s: AllowedParents count = %d, want %d",
					expected.tag, len(def.AllowedParents), len(expected.allowedParents))
			}

			for _, wantParent := range expected.allowedParents {
				found := false
				for _, gotParent := range def.AllowedParents {
					if gotParent == wantParent {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("%s: missing expected parent %s", expected.tag, wantParent)
				}
			}

			// Check pattern
			if expected.hasPattern && def.ValuePattern == nil {
				t.Errorf("%s: expected ValuePattern, got nil", expected.tag)
			}
			if !expected.hasPattern && def.ValuePattern != nil {
				t.Errorf("%s: expected no ValuePattern, got %v", expected.tag, def.ValuePattern)
			}

			// All tags should have descriptions
			if def.Description == "" {
				t.Errorf("%s: expected non-empty Description", expected.tag)
			}
		})
	}

	// Verify count
	if r.Len() != len(expectedTags) {
		t.Errorf("AncestryRegistry should have %d tags, got %d", len(expectedTags), r.Len())
	}
}

func TestFamilySearchRegistry(t *testing.T) {
	r := FamilySearchRegistry()

	if r == nil {
		t.Fatal("FamilySearchRegistry returned nil")
	}

	// Expected FamilySearch tags
	expectedTags := []struct {
		tag            string
		allowedParents []string
	}{
		{"_FSFTID", []string{"INDI"}},
		{"_FSORD", []string{"INDI"}},
		{"_FSTAG", []string{}}, // Empty = allowed anywhere
	}

	for _, expected := range expectedTags {
		t.Run(expected.tag, func(t *testing.T) {
			if !r.IsKnown(expected.tag) {
				t.Errorf("FamilySearchRegistry should contain %s", expected.tag)
				return
			}

			def, _ := r.Get(expected.tag)

			// Check allowed parents
			if len(expected.allowedParents) != len(def.AllowedParents) {
				t.Errorf("%s: AllowedParents count = %d, want %d",
					expected.tag, len(def.AllowedParents), len(expected.allowedParents))
			}

			// All tags should have descriptions
			if def.Description == "" {
				t.Errorf("%s: expected non-empty Description", expected.tag)
			}
		})
	}

	// Verify count
	if r.Len() != len(expectedTags) {
		t.Errorf("FamilySearchRegistry should have %d tags, got %d", len(expectedTags), r.Len())
	}
}

func TestRootsMagicRegistry(t *testing.T) {
	r := RootsMagicRegistry()

	if r == nil {
		t.Fatal("RootsMagicRegistry returned nil")
	}

	// Expected RootsMagic tags
	expectedTags := []struct {
		tag            string
		allowedParents []string
		hasPattern     bool
	}{
		{"_PRIM", []string{}, true}, // Empty parents (allowed anywhere), has YesNoPattern
		{"_SDATE", []string{}, false},
		{"_TMPLT", []string{"SOUR"}, false},
	}

	for _, expected := range expectedTags {
		t.Run(expected.tag, func(t *testing.T) {
			if !r.IsKnown(expected.tag) {
				t.Errorf("RootsMagicRegistry should contain %s", expected.tag)
				return
			}

			def, _ := r.Get(expected.tag)

			// Check allowed parents
			if len(expected.allowedParents) != len(def.AllowedParents) {
				t.Errorf("%s: AllowedParents count = %d, want %d",
					expected.tag, len(def.AllowedParents), len(expected.allowedParents))
			}

			// Check pattern
			if expected.hasPattern && def.ValuePattern == nil {
				t.Errorf("%s: expected ValuePattern, got nil", expected.tag)
			}

			// All tags should have descriptions
			if def.Description == "" {
				t.Errorf("%s: expected non-empty Description", expected.tag)
			}
		})
	}

	// Verify count
	if r.Len() != len(expectedTags) {
		t.Errorf("RootsMagicRegistry should have %d tags, got %d", len(expectedTags), r.Len())
	}
}

func TestMergeRegistries(t *testing.T) {
	t.Run("merge multiple registries", func(t *testing.T) {
		r1 := NewTagRegistry()
		_ = r1.Register("_TAG1", TagDefinition{Description: "Tag 1"})
		_ = r1.Register("_TAG2", TagDefinition{Description: "Tag 2"})

		r2 := NewTagRegistry()
		_ = r2.Register("_TAG3", TagDefinition{Description: "Tag 3"})
		_ = r2.Register("_TAG4", TagDefinition{Description: "Tag 4"})

		merged := MergeRegistries(r1, r2)

		if merged.Len() != 4 {
			t.Errorf("merged registry should have 4 tags, got %d", merged.Len())
		}

		// Verify all tags present
		for _, tag := range []string{"_TAG1", "_TAG2", "_TAG3", "_TAG4"} {
			if !merged.IsKnown(tag) {
				t.Errorf("merged registry should contain %s", tag)
			}
		}
	})

	t.Run("first definition wins on conflict", func(t *testing.T) {
		r1 := NewTagRegistry()
		_ = r1.Register("_PRIM", TagDefinition{
			AllowedParents: []string{"OBJE"},
			Description:    "First definition",
		})

		r2 := NewTagRegistry()
		_ = r2.Register("_PRIM", TagDefinition{
			AllowedParents: []string{}, // Different
			Description:    "Second definition",
		})

		merged := MergeRegistries(r1, r2)

		def, _ := merged.Get("_PRIM")
		if def.Description != "First definition" {
			t.Errorf("expected first definition to win, got %q", def.Description)
		}
		if len(def.AllowedParents) != 1 || def.AllowedParents[0] != "OBJE" {
			t.Errorf("AllowedParents should be from first definition, got %v", def.AllowedParents)
		}
	})

	t.Run("handles nil registries", func(t *testing.T) {
		r1 := NewTagRegistry()
		_ = r1.Register("_TAG1", TagDefinition{Description: "Tag 1"})

		merged := MergeRegistries(r1, nil, nil)

		if merged.Len() != 1 {
			t.Errorf("merged registry should have 1 tag, got %d", merged.Len())
		}
		if !merged.IsKnown("_TAG1") {
			t.Error("merged registry should contain _TAG1")
		}
	})
}

func TestMergeRegistries_Empty(t *testing.T) {
	t.Run("all empty registries", func(t *testing.T) {
		merged := MergeRegistries(NewTagRegistry(), NewTagRegistry())
		if merged.Len() != 0 {
			t.Errorf("merged empty registries should be empty, got %d tags", merged.Len())
		}
	})

	t.Run("no registries", func(t *testing.T) {
		merged := MergeRegistries()
		if merged.Len() != 0 {
			t.Errorf("MergeRegistries with no args should be empty, got %d tags", merged.Len())
		}
	})

	t.Run("all nil registries", func(t *testing.T) {
		merged := MergeRegistries(nil, nil, nil)
		if merged.Len() != 0 {
			t.Errorf("MergeRegistries with all nil should be empty, got %d tags", merged.Len())
		}
	})
}

func TestDefaultVendorRegistry(t *testing.T) {
	r := DefaultVendorRegistry()

	if r == nil {
		t.Fatal("DefaultVendorRegistry returned nil")
	}

	// Should include tags from all vendors
	ancestryTags := []string{"_APID", "_TREE", "_MILT"}
	familySearchTags := []string{"_FSFTID", "_FSORD", "_FSTAG"}
	rootsMagicTags := []string{"_SDATE", "_TMPLT"}

	for _, tag := range ancestryTags {
		if !r.IsKnown(tag) {
			t.Errorf("DefaultVendorRegistry should contain Ancestry tag %s", tag)
		}
	}

	for _, tag := range familySearchTags {
		if !r.IsKnown(tag) {
			t.Errorf("DefaultVendorRegistry should contain FamilySearch tag %s", tag)
		}
	}

	for _, tag := range rootsMagicTags {
		if !r.IsKnown(tag) {
			t.Errorf("DefaultVendorRegistry should contain RootsMagic tag %s", tag)
		}
	}

	// _PRIM should use Ancestry definition (listed first in merge)
	def, _ := r.Get("_PRIM")
	if len(def.AllowedParents) != 1 || def.AllowedParents[0] != "OBJE" {
		t.Errorf("_PRIM should use Ancestry definition (OBJE parent), got %v", def.AllowedParents)
	}
}

func TestRegistryForVendor(t *testing.T) {
	tests := []struct {
		name           string
		vendor         gedcom.Vendor
		expectedTags   []string
		unexpectedTags []string
	}{
		{
			name:           "Ancestry vendor",
			vendor:         gedcom.VendorAncestry,
			expectedTags:   []string{"_APID", "_TREE", "_MILT", "_PRIM"},
			unexpectedTags: []string{"_FSFTID", "_SDATE"},
		},
		{
			name:           "FamilySearch vendor",
			vendor:         gedcom.VendorFamilySearch,
			expectedTags:   []string{"_FSFTID", "_FSORD", "_FSTAG"},
			unexpectedTags: []string{"_APID", "_SDATE"},
		},
		{
			name:           "RootsMagic vendor",
			vendor:         gedcom.VendorRootsMagic,
			expectedTags:   []string{"_PRIM", "_SDATE", "_TMPLT"},
			unexpectedTags: []string{"_APID", "_FSFTID"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := RegistryForVendor(tt.vendor)

			if r == nil {
				t.Fatal("RegistryForVendor returned nil")
			}

			for _, tag := range tt.expectedTags {
				if !r.IsKnown(tag) {
					t.Errorf("registry for %s should contain %s", tt.vendor, tag)
				}
			}

			for _, tag := range tt.unexpectedTags {
				if r.IsKnown(tag) {
					t.Errorf("registry for %s should not contain %s", tt.vendor, tag)
				}
			}
		})
	}
}

func TestRegistryForVendor_Unknown(t *testing.T) {
	tests := []struct {
		name   string
		vendor gedcom.Vendor
	}{
		{"Unknown vendor", gedcom.VendorUnknown},
		{"Legacy vendor (no registry defined)", gedcom.VendorLegacy},
		{"Gramps vendor (no registry defined)", gedcom.VendorGramps},
		{"MyHeritage vendor (no registry defined)", gedcom.VendorMyHeritage},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := RegistryForVendor(tt.vendor)

			if r == nil {
				t.Fatal("RegistryForVendor should return empty registry, not nil")
			}

			if r.Len() != 0 {
				t.Errorf("registry for %s should be empty, got %d tags", tt.vendor, r.Len())
			}
		})
	}
}

func TestVendorRegistries_Validate(t *testing.T) {
	// Test that vendor registries actually validate correctly
	t.Run("Ancestry _APID under valid parent", func(t *testing.T) {
		r := AncestryRegistry()
		issue := r.ValidateTag("_APID", "SOUR", "7602,2771226")
		if issue != nil {
			t.Errorf("_APID under SOUR should be valid, got %v", issue)
		}
	})

	t.Run("Ancestry _APID under invalid parent", func(t *testing.T) {
		r := AncestryRegistry()
		issue := r.ValidateTag("_APID", "INDI", "7602,2771226")
		if issue == nil {
			t.Error("_APID under INDI should be invalid")
		}
		if issue != nil && issue.Code != CodeInvalidTagParent {
			t.Errorf("expected CodeInvalidTagParent, got %s", issue.Code)
		}
	})

	t.Run("Ancestry _PRIM with valid value", func(t *testing.T) {
		r := AncestryRegistry()
		for _, val := range []string{"Y", "N"} {
			issue := r.ValidateTag("_PRIM", "OBJE", val)
			if issue != nil {
				t.Errorf("_PRIM with value %q should be valid, got %v", val, issue)
			}
		}
	})

	t.Run("Ancestry _PRIM with invalid value", func(t *testing.T) {
		r := AncestryRegistry()
		issue := r.ValidateTag("_PRIM", "OBJE", "YES")
		if issue == nil {
			t.Error("_PRIM with value 'YES' should be invalid")
		}
		if issue != nil && issue.Code != CodeInvalidTagValue {
			t.Errorf("expected CodeInvalidTagValue, got %s", issue.Code)
		}
	})

	t.Run("FamilySearch _FSTAG allowed anywhere", func(t *testing.T) {
		r := FamilySearchRegistry()
		// _FSTAG has empty AllowedParents, should be valid under any parent
		for _, parent := range []string{"INDI", "FAM", "SOUR", "NOTE", "HEAD"} {
			issue := r.ValidateTag("_FSTAG", parent, "value")
			if issue != nil {
				t.Errorf("_FSTAG under %s should be valid, got %v", parent, issue)
			}
		}
	})

	t.Run("RootsMagic _PRIM allowed anywhere", func(t *testing.T) {
		r := RootsMagicRegistry()
		// RootsMagic _PRIM has empty AllowedParents
		for _, parent := range []string{"INDI", "FAM", "OBJE", "NAME", "BIRT"} {
			issue := r.ValidateTag("_PRIM", parent, "Y")
			if issue != nil {
				t.Errorf("RootsMagic _PRIM under %s should be valid, got %v", parent, issue)
			}
		}
	})
}

func TestVendorRegistries_Independence(t *testing.T) {
	// Verify that vendor registries are independent (not shared state)
	ancestry1 := AncestryRegistry()
	ancestry2 := AncestryRegistry()

	// They should have the same content
	if ancestry1.Len() != ancestry2.Len() {
		t.Error("two AncestryRegistry calls should return same content")
	}

	// But modifying one shouldn't affect the other
	_ = ancestry1.Register("_NEW_TAG", TagDefinition{Description: "New"})

	if ancestry2.IsKnown("_NEW_TAG") {
		t.Error("registries should be independent; modifying one affected another")
	}
}
