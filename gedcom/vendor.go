package gedcom

import (
	"slices"
	"strings"
	"unicode"
)

// Vendor represents the software that created a GEDCOM file.
// It is detected from the HEAD.SOUR subtree of the GEDCOM header; see
// DetectVendorFromHeader.
type Vendor string

// Known vendor constants. These represent common genealogy software that
// produces GEDCOM files. VendorUnknown is returned when the source system
// cannot be identified.
const (
	// VendorUnknown indicates the GEDCOM source could not be identified.
	VendorUnknown Vendor = ""

	// VendorAncestry represents Ancestry.com products including Family Tree
	// Maker (HEAD.SOUR "FTM" or "FamilyTreeMaker").
	VendorAncestry Vendor = "ancestry"

	// VendorFamilySearch represents FamilySearch.org.
	VendorFamilySearch Vendor = "familysearch"

	// VendorRootsMagic represents RootsMagic genealogy software.
	VendorRootsMagic Vendor = "rootsmagic"

	// VendorLegacy represents Legacy Family Tree software.
	VendorLegacy Vendor = "legacy"

	// VendorGramps represents the Gramps open-source genealogy software.
	VendorGramps Vendor = "gramps"

	// VendorMyHeritage represents MyHeritage genealogy service and its
	// Family Tree Builder desktop software.
	VendorMyHeritage Vendor = "myheritage"

	// VendorPAF represents Personal Ancestral File.
	VendorPAF Vendor = "paf"

	// VendorAncestris represents the Ancestris open-source genealogy software.
	VendorAncestris Vendor = "ancestris"

	// VendorHeredis represents Heredis genealogy software.
	VendorHeredis Vendor = "heredis"

	// VendorFamilyHistorian represents Family Historian (Calico Pie).
	VendorFamilyHistorian Vendor = "familyhistorian"

	// VendorBrothersKeeper represents Brother's Keeper.
	VendorBrothersKeeper Vendor = "brotherskeeper"

	// VendorTMG represents The Master Genealogist.
	VendorTMG Vendor = "tmg"

	// VendorFamilyOrigins represents Family Origins (FormalSoft).
	VendorFamilyOrigins Vendor = "familyorigins"

	// VendorEasyTree represents EasyTree.
	VendorEasyTree Vendor = "easytree"

	// VendorGEDitCOM represents GEDitCOM.
	VendorGEDitCOM Vendor = "geditcom"

	// VendorWebtrees represents the webtrees online genealogy application.
	VendorWebtrees Vendor = "webtrees"

	// VendorGeneanet represents Geneanet.
	VendorGeneanet Vendor = "geneanet"

	// VendorMyRoots represents My Roots for Palm OS.
	VendorMyRoots Vendor = "myroots"

	// VendorWebTreePrint represents the webtreeprint.com web generator.
	VendorWebTreePrint Vendor = "webtreeprint"

	// VendorReunion represents Reunion (Leister Productions).
	VendorReunion Vendor = "reunion"

	// VendorMacFamilyTree represents MacFamilyTree (Synium Software).
	VendorMacFamilyTree Vendor = "macfamilytree"
)

// vendorRule recognizes one vendor. names match as substrings of the input
// lowercased with every non-letter, non-digit removed, so "Family Tree
// Maker", "FamilyTreeMaker" and "FAMILY_TREE_MAKER" all match
// "familytreemaker". codes are short abbreviations that match only as a
// whole word, so "FTM 2019" matches "ftm" but "GiftMaker" does not.
type vendorRule struct {
	vendor Vendor
	names  []string
	codes  []string
}

// vendorRules is checked in order; the first match wins, so order is part of
// the behaviour. Names are unanchored substrings, so a generic word such as
// "legacy" or "reunion" also matches inside unrelated text; keep earlier
// rules the more specific ones (pinned by TestDetectVendor).
var vendorRules = []vendorRule{
	{VendorAncestry, []string{"ancestry", "familytreemaker"}, []string{"ftm"}},
	{VendorFamilySearch, []string{"familysearch"}, nil},
	{VendorRootsMagic, []string{"rootsmagic"}, nil},
	{VendorLegacy, []string{"legacy"}, nil},
	{VendorGramps, []string{"gramps"}, nil},
	{VendorMyHeritage, []string{"myheritage"}, nil},
	{VendorPAF, []string{"personalancestralfile"}, []string{"paf"}},
	{VendorAncestris, []string{"ancestris"}, nil},
	{VendorHeredis, []string{"heredis"}, nil},
	{VendorFamilyHistorian, []string{"familyhistorian"}, nil},
	{VendorBrothersKeeper, []string{"broskeep", "brotherskeeper"}, nil},
	{VendorTMG, []string{"mastergenealogist"}, []string{"tmg"}},
	{VendorFamilyOrigins, []string{"familyorigins"}, nil},
	{VendorEasyTree, []string{"easytree"}, nil},
	{VendorGEDitCOM, []string{"geditcom"}, nil},
	{VendorWebtrees, []string{"webtrees"}, nil},
	{VendorGeneanet, []string{"geneanet"}, nil},
	{VendorMyRoots, []string{"myroots"}, nil},
	{VendorWebTreePrint, []string{"webtreeprint"}, nil},
	{VendorReunion, []string{"reunion"}, nil},
	{VendorMacFamilyTree, []string{"macfamilytree", "syniumfamilytree"}, nil},
}

// DetectVendor identifies the GEDCOM vendor from one source string, such as
// the HEAD.SOUR value. Detection is case-insensitive and ignores spacing and
// punctuation; see the Vendor constants for what each one recognizes.
// Returns VendorUnknown if the string is not recognized.
//
// To detect from a decoded header, which also consults HEAD.SOUR.NAME and
// HEAD.SOUR.CORP, use DetectVendorFromHeader.
func DetectVendor(sourceSystem string) Vendor {
	if sourceSystem == "" {
		return VendorUnknown
	}

	lower := strings.ToLower(sourceSystem)
	notAlnum := func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }
	words := strings.FieldsFunc(lower, notAlnum)
	compact := strings.Join(words, "")

	for _, rule := range vendorRules {
		for _, name := range rule.names {
			if strings.Contains(compact, name) {
				return rule.vendor
			}
		}
		for _, code := range rule.codes {
			if slices.Contains(words, code) {
				return rule.vendor
			}
		}
	}

	return VendorUnknown
}

// DetectVendorFromHeader identifies the GEDCOM vendor from a header. It tries
// HEAD.SOUR (SourceSystem), then HEAD.SOUR.NAME (SourceName), then
// HEAD.SOUR.CORP (SourceCorporation.Name) with DetectVendor, and returns the
// first vendor found. When none of them names a vendor, a HEAD.SOUR._RTLSAVE
// subordinate identifies MyHeritage, whose Family Tree Builder can export with
// HEAD.SOUR, NAME and CORP all empty. Returns VendorUnknown for a nil header
// or when nothing matches. The decoder sets Document.Vendor from this.
func DetectVendorFromHeader(h *Header) Vendor {
	if h == nil {
		return VendorUnknown
	}
	candidates := []string{h.SourceSystem, h.SourceName}
	if h.SourceCorporation != nil {
		candidates = append(candidates, h.SourceCorporation.Name)
	}
	for _, s := range candidates {
		if v := DetectVendor(s); v.IsKnown() {
			return v
		}
	}
	if hasSourceSubordinate(h.Tags, "_RTLSAVE") {
		return VendorMyHeritage
	}
	return VendorUnknown
}

// hasSourceSubordinate reports whether a level-2 tag sits directly under the
// level-1 SOUR in header tags (whose levels start at 1).
func hasSourceSubordinate(tags []*Tag, tag string) bool {
	inSour := false
	for _, t := range tags {
		if t == nil {
			continue
		}
		switch t.Level {
		case 1:
			inSour = t.Tag == "SOUR"
		case 2:
			if inSour && t.Tag == tag {
				return true
			}
		}
	}
	return false
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
