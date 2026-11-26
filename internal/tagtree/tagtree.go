// Package tagtree decides how a flat list of GEDCOM tags nests (SI-023).
//
// Record.Tags and Header.Tags store a structure as a run of lines, each with
// a level. A lenient decode clamps a level that jumps more than one past the
// previous line's, but a strict decode keeps it (SI-006) and code can build
// any levels, so a tag's parent is not always the tag one level up. Every
// package that reads that nesting (the gedcom package's path lookup,
// references, substructure moves and diff, and the encoder's layout-keeping
// sync) must agree on it, so the rule lives here once (issue #968).
package tagtree

// Parents returns the index in tags of each tag's parent: the nearest earlier
// tag at a shallower level, so a level-3 tag directly after a level-1 tag is
// that tag's child. It is -1 for a tag with no shallower tag before it, which
// is a top-level structure, and for a tag that level reports as absent (a nil
// tag); an absent tag is never a parent. level returns a tag's level and
// whether it is present.
//
// The rule is SI-023.
func Parents[T any](tags []T, level func(T) (int, bool)) []int {
	type open struct{ index, level int }
	var chain []open // the tags that can still be a parent, shallowest first
	parents := make([]int, len(tags))
	for i, t := range tags {
		parents[i] = -1
		l, ok := level(t)
		if !ok {
			continue
		}
		for len(chain) > 0 && chain[len(chain)-1].level >= l {
			chain = chain[:len(chain)-1]
		}
		if len(chain) > 0 {
			parents[i] = chain[len(chain)-1].index
		}
		chain = append(chain, open{i, l})
	}
	return parents
}
