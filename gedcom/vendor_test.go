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
			name:         "short code inside a word is not a match",
			sourceSystem: "GiftMaker",
			want:         VendorUnknown,
		},
		{
			name:         "webtreeprint does not contain webtrees",
			sourceSystem: "webtreeprint.com",
			want:         VendorWebTreePrint,
		},

		// Issue #731: HEAD.SOUR, SOUR.NAME and SOUR.CORP values from real
		// exports in testdata/.
		{name: "FTM", sourceSystem: "FTM", want: VendorAncestry},
		{name: "Family Tree Maker name", sourceSystem: "Family Tree Maker for Windows", want: VendorAncestry},
		{name: "PAF with version", sourceSystem: "PAF 2.2", want: VendorPAF},
		{name: "PAF name", sourceSystem: "Personal Ancestral File", want: VendorPAF},
		{name: "Ancestris", sourceSystem: "ANCESTRIS", want: VendorAncestris},
		{name: "Heredis", sourceSystem: "HEREDIS 14 PC", want: VendorHeredis},
		{name: "Family Historian", sourceSystem: "FAMILY_HISTORIAN", want: VendorFamilyHistorian},
		{name: "Brother's Keeper", sourceSystem: "BROSKEEP", want: VendorBrothersKeeper},
		{name: "Brother's Keeper name", sourceSystem: "Brother's Keeper", want: VendorBrothersKeeper},
		{name: "TMG", sourceSystem: "TMG 1.2a", want: VendorTMG},
		{name: "Family Origins", sourceSystem: "FamilyOrigins", want: VendorFamilyOrigins},
		{name: "EasyTree", sourceSystem: "EasyTree", want: VendorEasyTree},
		{name: "GEDitCOM", sourceSystem: "GEDitCOM", want: VendorGEDitCOM},
		{name: "webtrees", sourceSystem: "webtrees", want: VendorWebtrees},
		{name: "Geneanet", sourceSystem: "Geneanet", want: VendorGeneanet},
		{name: "My Roots", sourceSystem: "My_Roots", want: VendorMyRoots},
		{name: "Reunion", sourceSystem: "Reunion", want: VendorReunion},
		{name: "MacFamilyTree", sourceSystem: "SyniumFamilyTree", want: VendorMacFamilyTree},

		// Rule order: a string naming two vendors goes to the earlier rule.
		{name: "earlier rule wins over legacy", sourceSystem: "Ancestry Legacy", want: VendorAncestry},
		{name: "legacy wins over later reunion", sourceSystem: "Legacy Reunion", want: VendorLegacy},
		{name: "myheritage wins over later myroots", sourceSystem: "MyHeritage My Roots", want: VendorMyHeritage},

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
		{VendorPAF, "paf"},
		{VendorAncestris, "ancestris"},
		{VendorHeredis, "heredis"},
		{VendorFamilyHistorian, "familyhistorian"},
		{VendorBrothersKeeper, "brotherskeeper"},
		{VendorTMG, "tmg"},
		{VendorFamilyOrigins, "familyorigins"},
		{VendorEasyTree, "easytree"},
		{VendorGEDitCOM, "geditcom"},
		{VendorWebtrees, "webtrees"},
		{VendorGeneanet, "geneanet"},
		{VendorMyRoots, "myroots"},
		{VendorWebTreePrint, "webtreeprint"},
		{VendorReunion, "reunion"},
		{VendorMacFamilyTree, "macfamilytree"},
	}

	for _, tt := range tests {
		if string(tt.vendor) != tt.value {
			t.Errorf("Vendor constant %q has value %q, want %q", tt.vendor, string(tt.vendor), tt.value)
		}
	}
}

func TestDetectVendorFromHeader(t *testing.T) {
	tests := []struct {
		name   string
		header *Header
		want   Vendor
	}{
		{"nil header", nil, VendorUnknown},
		{"empty header", &Header{}, VendorUnknown},
		{"SOUR wins over NAME and CORP", &Header{
			SourceSystem:      "Legacy",
			SourceName:        "Legacy (R)",
			SourceCorporation: &Corporation{Name: "MyHeritage USA Inc."},
		}, VendorLegacy},
		{"NAME when SOUR is unrecognized", &Header{
			SourceSystem: "XYZ",
			SourceName:   "Family Origins(R) for Windows",
		}, VendorFamilyOrigins},
		{"CORP when SOUR and NAME are unrecognized", &Header{
			SourceSystem:      "XYZ",
			SourceName:        "Some Product",
			SourceCorporation: &Corporation{Name: "Ancestry.com"},
		}, VendorAncestry},
		{"HEAD.SOUR._RTLSAVE identifies MyHeritage", &Header{
			Tags: []*Tag{
				{Level: 1, Tag: "SOUR"},
				{Level: 2, Tag: "NAME"},
				{Level: 2, Tag: "_RTLSAVE", Value: "RTL"},
			},
		}, VendorMyHeritage},
		{"_RTLSAVE outside HEAD.SOUR is ignored", &Header{
			Tags: []*Tag{
				{Level: 1, Tag: "SOUR"},
				nil,
				{Level: 1, Tag: "GEDC"},
				{Level: 2, Tag: "_RTLSAVE", Value: "RTL"},
			},
		}, VendorUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DetectVendorFromHeader(tt.header); got != tt.want {
				t.Errorf("DetectVendorFromHeader() = %q, want %q", got, tt.want)
			}
		})
	}
}
