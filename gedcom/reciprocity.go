package gedcom

import "slices"

// IsSpouseIn reports whether i is a spouse in fam under the reciprocal-link
// rule: i has a FAMS pointer to fam and fam names i as HUSB or WIFE. It is
// false when either side is missing, or when i or fam is nil or has no XRef.
//
// The reciprocal-link rule: GEDCOM records a family membership on both sides,
// as a FAMS or FAMC pointer on the individual and as a HUSB, WIFE or CHIL
// pointer on the family, and a membership counts only when both sides record
// it. A link only one side records (a FAMS to a family that names other
// people as HUSB and WIFE, or a CHIL naming a person with no matching FAMC)
// is ignored by every relationship helper in this package: Parents,
// ParentLinks, Spouses, Children, Siblings, FamiliesAsSpouse,
// FamiliesAsChild, Family.HusbandIndividual, WifeIndividual,
// ChildrenIndividuals and AllMembers, and Ancestors and Descendants with
// their WithGenerations forms. IsSpouseIn and IsChildIn are the rule's one
// definition; the validator reports a link the rule does not count.
func (i *Individual) IsSpouseIn(fam *Family) bool {
	if !linkable(i, fam) || (fam.Husband != i.XRef && fam.Wife != i.XRef) {
		return false
	}
	return hasFamilyLink(i.SpouseInFamilies, fam.XRef)
}

// IsChildIn reports whether i is a child in fam under the reciprocal-link
// rule: i has a FAMC pointer to fam and fam names i as CHIL. It is false when
// either side is missing, or when i or fam is nil or has no XRef.
func (i *Individual) IsChildIn(fam *Family) bool {
	return hasChildSide(i, fam) && slices.Contains(fam.Children, i.XRef)
}

// hasChildSide reports whether i records its side of being a child in fam: a
// FAMC pointer to it. A walk over fam.Children already has the family's side,
// so it checks only this one and stays linear in the number of children,
// where IsChildIn per child would rescan fam.Children each time.
func hasChildSide(i *Individual, fam *Family) bool {
	return linkable(i, fam) && hasFamilyLink(i.ChildInFamilies, fam.XRef)
}

// linkable reports whether i and fam can be linked at all: both exist and
// have an XRef, so an empty pointer never matches an empty XRef.
func linkable(i *Individual, fam *Family) bool {
	return i != nil && fam != nil && i.XRef != "" && fam.XRef != ""
}

// hasFamilyLink reports whether links holds a pointer to famXRef.
func hasFamilyLink(links []FamilyLink, famXRef string) bool {
	return slices.ContainsFunc(links, func(l FamilyLink) bool { return l.FamilyXRef == famXRef })
}
