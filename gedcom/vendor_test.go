package gedcom

import "testing"

func TestDetectVendor(t *testing.T) {
	tests := []struct {
		name         string
		sourceSystem string
		want         Vendor
	}{
		// Empty and unknown cases
		{
			name:         "empty string returns unknown",
			sourceSystem: "",
			want:         VendorUnknown,
		},
		{
			name:         "unrecognized vendor returns unknown",
			sourceSystem: "SomeRandomApp",
			want:         VendorUnknown,
		},
		{
			name:         "GS returns unknown",
			sourceSystem: "GS",
			want:         VendorUnknown,
		},
		{
			name:         "PAF returns unknown",
			sourceSystem: "PAF 2.2",
			want:         VendorUnknown,
		},
		{
			name:         "GEDitCOM returns unknown",
			sourceSystem: "GEDitCOM",
			want:         VendorUnknown,
		},

		// Ancestry variants
		{
			name:         "ancestry lowercase",
			sourceSystem: "ancestry",
			want:         VendorAncestry,
		},
		{
			name:         "Ancestry uppercase",
			sourceSystem: "ANCESTRY",
			want:         VendorAncestry,
		},
		{
			name:         "Ancestry mixed case",
			sourceSystem: "Ancestry.com",
			want:         VendorAncestry,
		},
		{
			name:         "FamilyTreeMaker lowercase",
			sourceSystem: "familytreemaker",
			want:         VendorAncestry,
		},
		{
			name:         "FamilyTreeMaker mixed case",
			sourceSystem: "FamilyTreeMaker",
			want:         VendorAncestry,
		},
		{
			name:         "FamilyTreeMaker with version",
			sourceSystem: "FamilyTreeMaker 2019",
			want:         VendorAncestry,
		},

		// FamilySearch variants
		{
			name:         "familysearch lowercase",
			sourceSystem: "familysearch",
			want:         VendorFamilySearch,
		},
		{
			name:         "FamilySearch mixed case",
			sourceSystem: "FamilySearch",
			want:         VendorFamilySearch,
		},
		{
			name:         "FamilySearch with org",
			sourceSystem: "FamilySearch.org",
			want:         VendorFamilySearch,
		},

		// RootsMagic variants
		{
			name:         "rootsmagic lowercase",
			sourceSystem: "rootsmagic",
			want:         VendorRootsMagic,
		},
		{
			name:         "RootsMagic mixed case",
			sourceSystem: "RootsMagic",
			want:         VendorRootsMagic,
		},
		{
			name:         "RootsMagic with version",
			sourceSystem: "RootsMagic 8",
			want:         VendorRootsMagic,
		},

		// Legacy variants
		{
			name:         "legacy lowercase",
			sourceSystem: "legacy",
			want:         VendorLegacy,
		},
		{
			name:         "Legacy mixed case",
			sourceSystem: "Legacy Family Tree",
			want:         VendorLegacy,
		},
		{
			name:         "Legacy with version",
			sourceSystem: "Legacy 9.0",
			want:         VendorLegacy,
		},

		// Gramps variants
		{
			name:         "gramps lowercase",
			sourceSystem: "gramps",
			want:         VendorGramps,
		},
		{
			name:         "Gramps uppercase",
			sourceSystem: "GRAMPS",
			want:         VendorGramps,
		},
		{
			name:         "Gramps with version",
			sourceSystem: "Gramps 5.1.4",
			want:         VendorGramps,
		},

		// MyHeritage variants
		{
			name:         "myheritage lowercase",
			sourceSystem: "myheritage",
			want:         VendorMyHeritage,
		},
		{
			name:         "MyHeritage mixed case",
			sourceSystem: "MyHeritage",
			want:         VendorMyHeritage,
		},
		{
			name:         "MyHeritage with description",
			sourceSystem: "MyHeritage Family Tree Builder",
			want:         VendorMyHeritage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectVendor(tt.sourceSystem)
			if got != tt.want {
				t.Errorf("DetectVendor(%q) = %v, want %v", tt.sourceSystem, got, tt.want)
			}
		})
	}
}

func TestVendor_String(t *testing.T) {
	tests := []struct {
		vendor Vendor
		want   string
	}{
		{VendorUnknown, "unknown"},
		{VendorAncestry, "ancestry"},
		{VendorFamilySearch, "familysearch"},
		{VendorRootsMagic, "rootsmagic"},
		{VendorLegacy, "legacy"},
		{VendorGramps, "gramps"},
		{VendorMyHeritage, "myheritage"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.vendor.String()
			if got != tt.want {
				t.Errorf("Vendor(%q).String() = %q, want %q", tt.vendor, got, tt.want)
			}
		})
	}
}

func TestVendor_IsKnown(t *testing.T) {
	tests := []struct {
		vendor Vendor
		want   bool
	}{
		{VendorUnknown, false},
		{VendorAncestry, true},
		{VendorFamilySearch, true},
		{VendorRootsMagic, true},
		{VendorLegacy, true},
		{VendorGramps, true},
		{VendorMyHeritage, true},
	}

	for _, tt := range tests {
		t.Run(tt.vendor.String(), func(t *testing.T) {
			got := tt.vendor.IsKnown()
			if got != tt.want {
				t.Errorf("Vendor(%q).IsKnown() = %v, want %v", tt.vendor, got, tt.want)
			}
		})
	}
}

func TestVendorConstants(t *testing.T) {
	// Verify all vendor constants have expected values
	tests := []struct {
		vendor Vendor
		value  string
	}{
		{VendorUnknown, ""},
		{VendorAncestry, "ancestry"},
		{VendorFamilySearch, "familysearch"},
		{VendorRootsMagic, "rootsmagic"},
		{VendorLegacy, "legacy"},
		{VendorGramps, "gramps"},
		{VendorMyHeritage, "myheritage"},
	}

	for _, tt := range tests {
		if string(tt.vendor) != tt.value {
			t.Errorf("Vendor constant %q has value %q, want %q", tt.vendor, string(tt.vendor), tt.value)
		}
	}
}
