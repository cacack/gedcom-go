package decoder

//go:generate go run ../internal/cmd/genplacement

import (
	"slices"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// tagPlacement is where one GEDCOM version allows each standard tag
// (placement_gen.go), by the tag of the structure it sits under. It tells a
// standard tag in the wrong place (CodeMisplacedTag) from a tag the version
// does not define (CodeUnknownTag), issue #884.
type tagPlacement struct {
	// tags is every tag the version defines anywhere, sorted.
	tags []string
	// records maps a record type (and HEAD) to the tags allowed at level 1.
	records map[string][]string
	// children maps a tag to the tags allowed under any structure below
	// level 0 that has it.
	children map[string][]string
}

// placementFor returns the placement tables of v, or nil for a version this
// package has none for.
func placementFor(v gedcom.Version) *tagPlacement {
	switch v {
	case gedcom.Version55:
		return placement55
	case gedcom.Version551:
		return placement551
	case gedcom.Version70:
		return placement70
	}
	return nil
}

// misplaced reports whether tag is a standard tag of p's version that no
// structure with parent's tag allows. parent is a record type when top is
// set (tag is at level 1). A tag the version does not define, or one under a
// parent it does not define (an extension, an unknown tag), is not misplaced.
// Because parents are matched by tag, a tag one structure with parent's tag
// allows is never misplaced under another with the same tag.
func (p *tagPlacement) misplaced(parent string, top bool, tag string) bool {
	if p == nil || !p.defines(tag) {
		return false
	}
	allowed := p.children
	if top {
		allowed = p.records
	}
	children, ok := allowed[parent]
	if !ok && !p.defines(parent) {
		return false
	}
	_, found := slices.BinarySearch(children, tag)
	return !found
}

// defines reports whether the version defines tag anywhere.
func (p *tagPlacement) defines(tag string) bool {
	_, found := slices.BinarySearch(p.tags, tag)
	return found
}

// lineParent is the parent of one line in a diagnosticCollector's scope.
type lineParent struct {
	tag string
	top bool
}

// enterScope tells c that the lines it is told about next belong to the
// record (or header) of type root, whose subordinate lines are tags, so that
// addUnknownTag can find a line's parent. The parents are indexed only when a
// tag is first reported.
func (c *diagnosticCollector) enterScope(root string, tags []*gedcom.Tag) {
	if c.collecting() {
		c.scopeRoot, c.scopeTags, c.parents = root, tags, nil
	}
}

// parentOf returns the parent of the line at lineNumber in c's scope.
func (c *diagnosticCollector) parentOf(lineNumber int) (lineParent, bool) {
	if c.parents == nil {
		c.parents = make(map[int]lineParent, len(c.scopeTags))
		// path[i] is the tag at level i on the way to the current line.
		path := []string{c.scopeRoot}
		for _, tag := range c.scopeTags {
			if tag == nil || tag.Level < 1 || tag.Level > len(path) {
				continue
			}
			c.parents[tag.LineNumber] = lineParent{path[tag.Level-1], tag.Level == 1}
			path = append(path[:tag.Level], tag.Tag)
		}
	}
	p, ok := c.parents[lineNumber]
	return p, ok
}
