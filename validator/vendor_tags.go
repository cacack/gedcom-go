// vendor_tags.go provides pre-built tag registries for common genealogy software vendors.
//
// These registries provide out-of-the-box validation for the most common vendor-specific
// GEDCOM extensions, saving users from having to define these tags themselves.
//
// # Usage
//
// Use a vendor-specific registry:
//
//	registry := validator.AncestryRegistry()
//	tv := validator.NewTagValidator(registry, true)
//	issues := tv.Validate(doc)
//
// Or use a combined registry with all vendors:
//
//	registry := validator.DefaultVendorRegistry()
//	tv := validator.NewTagValidator(registry, true)
//	issues := tv.Validate(doc)
//
// Select registry based on detected vendor:
//
//	registry := validator.RegistryForVendor(doc.Vendor)
//	tv := validator.NewTagValidator(registry, true)

package validator

import "github.com/cacack/gedcom-go/gedcom"

// AncestryRegistry returns a TagRegistry containing common Ancestry.com custom tags.
//
// Ancestry.com (and FamilyTreeMaker) use various custom tags to track proprietary
// identifiers and metadata. This registry covers the most commonly encountered tags.
func AncestryRegistry() *TagRegistry {
	r := NewTagRegistry()

	// _APID - Ancestry Permanent ID
	// Format: "database_id,record_id" (e.g., "7602,2771226")
	// Used to link source citations back to specific records in Ancestry databases
	_ = r.Register("_APID", TagDefinition{
		Tag:            "_APID",
		AllowedParents: []string{"SOUR"},
		Description:    "Ancestry Permanent ID linking to database record (format: database_id,record_id)",
	})

	// _TREE - Ancestry tree reference
	// Found in the header, references the Ancestry tree the GEDCOM was exported from
	_ = r.Register("_TREE", TagDefinition{
		Tag:            "_TREE",
		AllowedParents: []string{"HEAD"},
		Description:    "Ancestry tree reference identifier",
	})

	// _MILT - Military service
	// Used to record military service information for individuals
	_ = r.Register("_MILT", TagDefinition{
		Tag:            "_MILT",
		AllowedParents: []string{"INDI"},
		Description:    "Military service record",
	})

	// _DEST - Destination
	// Used as a subordinate to emigration/immigration events
	_ = r.Register("_DEST", TagDefinition{
		Tag:            "_DEST",
		AllowedParents: []string{"EMIG", "IMMI"},
		Description:    "Destination location for emigration/immigration events",
	})

	// _PRIM - Primary flag
	// Indicates whether a media object is the primary/preferred one
	_ = r.Register("_PRIM", TagDefinition{
		Tag:            "_PRIM",
		AllowedParents: []string{"OBJE"},
		ValuePattern:   YesNoPattern,
		Description:    "Primary flag for photos (Y/N)",
	})

	// _PHOTO - Photo indicator
	// Indicates individual has an associated photo
	_ = r.Register("_PHOTO", TagDefinition{
		Tag:            "_PHOTO",
		AllowedParents: []string{"INDI"},
		Description:    "Photo indicator for individual",
	})

	return r
}

// FamilySearchRegistry returns a TagRegistry containing common FamilySearch custom tags.
//
// FamilySearch uses custom tags to link GEDCOM data back to their Family Tree
// and to record LDS-specific ordinance information.
func FamilySearchRegistry() *TagRegistry {
	r := NewTagRegistry()

	// _FSFTID - FamilySearch Family Tree ID
	// Alphanumeric identifier like "KWCJ-QN7" linking to a person in FamilySearch
	_ = r.Register("_FSFTID", TagDefinition{
		Tag:            "_FSFTID",
		AllowedParents: []string{"INDI"},
		Description:    "FamilySearch Family Tree ID (e.g., KWCJ-QN7)",
	})

	// _FSORD - FamilySearch ordinance
	// Used to record FamilySearch-specific ordinance information
	_ = r.Register("_FSORD", TagDefinition{
		Tag:            "_FSORD",
		AllowedParents: []string{"INDI"},
		Description:    "FamilySearch ordinance information",
	})

	// _FSTAG - FamilySearch tag
	// General-purpose tag used by FamilySearch for various metadata
	_ = r.Register("_FSTAG", TagDefinition{
		Tag:            "_FSTAG",
		AllowedParents: []string{}, // Empty = allowed anywhere
		Description:    "FamilySearch general-purpose tag",
	})

	return r
}

// RootsMagicRegistry returns a TagRegistry containing common RootsMagic custom tags.
//
// RootsMagic uses various custom tags for features like sort dates, templates,
// and primary indicators.
func RootsMagicRegistry() *TagRegistry {
	r := NewTagRegistry()

	// _PRIM - Primary indicator
	// RootsMagic uses this for various "primary" designations
	// Note: More permissive AllowedParents than Ancestry's _PRIM
	_ = r.Register("_PRIM", TagDefinition{
		Tag:            "_PRIM",
		AllowedParents: []string{}, // Empty = allowed anywhere (events, names, etc.)
		ValuePattern:   YesNoPattern,
		Description:    "Primary indicator (Y/N)",
	})

	// _SDATE - Sort date
	// Used to provide a sort order date separate from the display date
	_ = r.Register("_SDATE", TagDefinition{
		Tag:            "_SDATE",
		AllowedParents: []string{}, // Can appear under various event tags
		Description:    "Sort date for ordering events",
	})

	// _TMPLT - Template reference
	// Used to reference source templates in RootsMagic
	_ = r.Register("_TMPLT", TagDefinition{
		Tag:            "_TMPLT",
		AllowedParents: []string{"SOUR"},
		Description:    "Source template reference",
	})

	return r
}

// MergeRegistries combines multiple TagRegistries into a single registry.
//
// If the same tag is defined in multiple registries, the first definition wins.
// Nil registries are safely skipped.
//
// This is useful for creating a combined registry for files that may contain
// tags from multiple vendors (e.g., an Ancestry file imported into RootsMagic).
func MergeRegistries(registries ...*TagRegistry) *TagRegistry {
	merged := NewTagRegistry()

	for _, reg := range registries {
		if reg == nil {
			continue
		}

		for _, tagName := range reg.Tags() {
			def, ok := reg.Get(tagName)
			if !ok {
				continue
			}

			// Attempt to register; ignore error if tag already exists
			// (first definition wins)
			_ = merged.Register(tagName, def)
		}
	}

	return merged
}

// DefaultVendorRegistry returns a merged registry containing tags from all
// known vendors (Ancestry, FamilySearch, RootsMagic).
//
// This provides convenient out-of-the-box support for validating GEDCOM files
// from any common source without needing to know the vendor in advance.
//
// Note: Where vendors use the same tag differently (e.g., _PRIM), the Ancestry
// definition takes precedence since Ancestry is listed first.
func DefaultVendorRegistry() *TagRegistry {
	return MergeRegistries(
		AncestryRegistry(),
		FamilySearchRegistry(),
		RootsMagicRegistry(),
	)
}

// RegistryForVendor returns the appropriate TagRegistry for a detected vendor.
//
// For known vendors (Ancestry, FamilySearch, RootsMagic), returns the
// vendor-specific registry. For unknown vendors or VendorUnknown, returns
// an empty registry.
//
// Use this when you want strict validation limited to tags from a specific
// vendor. For more permissive validation, use DefaultVendorRegistry() instead.
func RegistryForVendor(vendor gedcom.Vendor) *TagRegistry {
	switch vendor {
	case gedcom.VendorAncestry:
		return AncestryRegistry()
	case gedcom.VendorFamilySearch:
		return FamilySearchRegistry()
	case gedcom.VendorRootsMagic:
		return RootsMagicRegistry()
	case gedcom.VendorLegacy, gedcom.VendorGramps, gedcom.VendorMyHeritage:
		// These vendors are detected but don't have registry definitions yet
		// Return empty registry rather than nil for safe use
		return NewTagRegistry()
	default:
		// Unknown vendor - return empty registry
		return NewTagRegistry()
	}
}
