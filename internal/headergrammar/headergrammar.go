// Package headergrammar holds the order each GEDCOM version's grammar gives the
// level-1 substructures of HEAD.
//
// The encoder and the converter both add header structures the source did not
// have (GEDC, CHAR), and the converter retargets a whole header to another
// version. Both must agree on where a structure goes, so the order lives here
// once rather than in each (issue #465).
package headergrammar

import (
	"cmp"
	"slices"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// order55 is the HEAD grammar of GEDCOM 5.5 and 5.5.1, which agree.
var order55 = []string{
	"SOUR", "DEST", "DATE", "SUBM", "SUBN", "FILE", "COPR", "GEDC", "CHAR", "LANG", "PLAC", "NOTE",
}

// order70 is the HEAD grammar of GEDCOM 7.0: GEDC first, so a reader learns the
// version before anything else. NOTE and SNOTE are the two forms of its note
// structure and share the last position.
var order70 = []string{
	"GEDC", "SCHMA", "SOUR", "DEST", "DATE", "SUBM", "COPR", "LANG", "PLAC", "NOTE", "SNOTE",
}

// Rank returns the position of the level-1 HEAD structure tag in version v's
// grammar. A tag the grammar does not define -- an extension, or a structure
// another version has -- ranks after every defined one. Any version other than
// 7.0 takes the 5.5 order, as the encoder's hand-built header path does.
func Rank(v gedcom.Version, tag string) int {
	order := order55
	if v == gedcom.Version70 {
		order = order70
	}
	if tag == "SNOTE" && v == gedcom.Version70 {
		tag = "NOTE"
	}
	if i := slices.Index(order, tag); i >= 0 {
		return i
	}
	return len(order)
}

// Sort returns tags, a flat level-encoded HEAD tag list, with its level-1
// structures (each carrying its subtree) in version v's grammar order. The sort
// is stable, so structures of equal rank -- undefined ones in particular --
// keep their relative order. Anything before the first level-1 tag stays
// first. The tags themselves are not modified, and neither is the input slice.
func Sort(tags []*gedcom.Tag, v gedcom.Version) []*gedcom.Tag {
	type group struct {
		rank int
		tags []*gedcom.Tag
	}

	var groups []group
	for _, tag := range tags {
		// Level <= 1, as elsewhere in the header code: a hand-built header may
		// leave Level unset on its top-level structures.
		topLevel := tag != nil && tag.Level <= 1
		switch {
		case topLevel:
			groups = append(groups, group{rank: Rank(v, tag.Tag)})
		case len(groups) == 0:
			groups = append(groups, group{rank: -1})
		}
		last := &groups[len(groups)-1]
		last.tags = append(last.tags, tag)
	}

	slices.SortStableFunc(groups, func(a, b group) int { return cmp.Compare(a.rank, b.rank) })

	sorted := make([]*gedcom.Tag, 0, len(tags))
	for _, g := range groups {
		sorted = append(sorted, g.tags...)
	}
	return sorted
}
