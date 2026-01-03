package gedcom

import "strings"

// Vendor represents the software that created a GEDCOM file.
// It is detected from the HEAD.SOUR tag in the GEDCOM header.
type Vendor string

// Known vendor constants. These represent common genealogy software that
// produces GEDCOM files. VendorUnknown is returned when the source system
// cannot be identified.
const (
	// VendorUnknown indicates the GEDCOM source could not be identified.
	VendorUnknown Vendor = ""

	// VendorAncestry represents Ancestry.com products including FamilyTreeMaker.
	VendorAncestry Vendor = "ancestry"

	// VendorFamilySearch represents FamilySearch.org.
	VendorFamilySearch Vendor = "familysearch"

	// VendorRootsMagic represents RootsMagic genealogy software.
	VendorRootsMagic Vendor = "rootsmagic"

	// VendorLegacy represents Legacy Family Tree software.
	VendorLegacy Vendor = "legacy"

	// VendorGramps represents the Gramps open-source genealogy software.
	VendorGramps Vendor = "gramps"

	// VendorMyHeritage represents MyHeritage genealogy service.
	VendorMyHeritage Vendor = "myheritage"
)

// DetectVendor identifies the GEDCOM vendor from the source system string.
// The source system is typically found in the HEAD.SOUR tag of a GEDCOM file.
// Detection is case-insensitive and uses substring matching.
// Returns VendorUnknown if the source system is not recognized.
func DetectVendor(sourceSystem string) Vendor {
	if sourceSystem == "" {
		return VendorUnknown
	}

	lower := strings.ToLower(sourceSystem)

	// Check for Ancestry products (including FamilyTreeMaker)
	if strings.Contains(lower, "ancestry") || strings.Contains(lower, "familytreemaker") {
		return VendorAncestry
	}

	// Check for FamilySearch
	if strings.Contains(lower, "familysearch") {
		return VendorFamilySearch
	}

	// Check for RootsMagic
	if strings.Contains(lower, "rootsmagic") {
		return VendorRootsMagic
	}

	// Check for Legacy
	if strings.Contains(lower, "legacy") {
		return VendorLegacy
	}

	// Check for Gramps
	if strings.Contains(lower, "gramps") {
		return VendorGramps
	}

	// Check for MyHeritage
	if strings.Contains(lower, "myheritage") {
		return VendorMyHeritage
	}

	return VendorUnknown
}

// String returns the vendor name as a string.
// For VendorUnknown, it returns "unknown".
func (v Vendor) String() string {
	if v == VendorUnknown {
		return "unknown"
	}
	return string(v)
}

// IsKnown returns true if the vendor is not VendorUnknown.
func (v Vendor) IsKnown() bool {
	return v != VendorUnknown
}
