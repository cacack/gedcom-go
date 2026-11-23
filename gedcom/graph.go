package gedcom

// Descendants returns the XRefs of all transitive descendants of the
// individual identified by xref. Walks family links via SpouseInFamilies
// → Children → Spouse families, breadth-first, with cycle detection.
//
// The result does not include the seed xref itself. Spouses of
// descendants are not included; callers who want them should union the
// result with the descendants' Spouses lookups.
//
// Return contract:
//   - nil — invalid input (doc is nil, xref is empty, the record does
//     not exist, or the record is not an individual)
//   - non-nil []string (possibly empty) — input was valid; the slice
//     holds the descendant XRefs, or is empty if there are none
//
// Ordering: descendants are returned in BFS order (closer descendants
// before more distant ones). Ties within a generation follow the source
// document order.
func (d *Document) Descendants(xref string) []string {
	return relativeXRefs(d.DescendantsWithGenerations(xref, 0))
}

// Ancestors returns the XRefs of all transitive ancestors of the
// individual identified by xref. Walks family links via ChildInFamilies
// → husband/wife → that individual's ChildInFamilies, breadth-first,
// with cycle detection.
//
// Both the husband and wife of each parent family are included, so
// step-parents and adoptive parents recorded via parent family links
// appear in the result alongside biological parents. Callers who want
// only one side should filter the result.
//
// The result does not include the seed xref itself.
//
// Return contract:
//   - nil — invalid input (doc is nil, xref is empty, the record does
//     not exist, or the record is not an individual)
//   - non-nil []string (possibly empty) — input was valid; the slice
//     holds the ancestor XRefs, or is empty if there are none
//
// Ordering: ancestors are returned in BFS order (parents before
// grandparents). Within a generation, husband precedes wife and order
// across multiple parent families follows source document order.
func (d *Document) Ancestors(xref string) []string {
	return relativeXRefs(d.AncestorsWithGenerations(xref, 0))
}

// Relative is one person reached by a generational walk, with the
// generation it was reached in: 1 for parents or children, 2 for
// grandparents or grandchildren, and so on.
type Relative struct {
	XRef       string
	Generation int
}

// DescendantsWithGenerations is Descendants with each descendant's
// generation number and an optional depth limit. It walks the same
// links, in the same order, and returns the same people as
// Descendants when maxGenerations is 0.
//
// maxGenerations limits the walk: the result holds no one beyond that
// generation. Zero or a negative value means unlimited.
//
// A person reachable by more than one path (pedigree collapse, such as
// a child of a cousin marriage) appears once, with the generation of
// the shortest path.
//
// The return contract and ordering match Descendants.
func (d *Document) DescendantsWithGenerations(xref string, maxGenerations int) []Relative {
	return d.walkGenerations(xref, maxGenerations, func(ind *Individual, visit func(string)) {
		for _, link := range ind.SpouseInFamilies {
			if fam := d.GetFamily(link.FamilyXRef); fam != nil {
				for _, child := range fam.Children {
					visit(child)
				}
			}
		}
	})
}

// AncestorsWithGenerations is Ancestors with each ancestor's generation
// number and an optional depth limit. It walks the same links, in the
// same order, and returns the same people as Ancestors when
// maxGenerations is 0.
//
// maxGenerations limits the walk: the result holds no one beyond that
// generation. Zero or a negative value means unlimited.
//
// A person reachable by more than one path (pedigree collapse, such as
// a common ancestor of two parents who are cousins) appears once, with
// the generation of the shortest path.
//
// The return contract and ordering match Ancestors.
func (d *Document) AncestorsWithGenerations(xref string, maxGenerations int) []Relative {
	return d.walkGenerations(xref, maxGenerations, func(ind *Individual, visit func(string)) {
		for _, link := range ind.ChildInFamilies {
			if fam := d.GetFamily(link.FamilyXRef); fam != nil {
				visit(fam.Husband)
				visit(fam.Wife)
			}
		}
	})
}

// walkGenerations is the breadth-first walk behind Ancestors and
// Descendants. next calls visit with each neighbor of an individual in
// order; empty, already-visited and non-individual XRefs are skipped.
// Breadth-first order means each person is first reached by a shortest
// path, so the generation recorded is the shortest.
func (d *Document) walkGenerations(xref string, maxGenerations int, next func(*Individual, func(string))) []Relative {
	if d == nil || xref == "" {
		return nil
	}
	seed := d.GetIndividual(xref)
	if seed == nil {
		return nil
	}

	visited := map[string]bool{xref: true}
	result := []Relative{}
	type step struct {
		ind        *Individual
		generation int
	}
	queue := []step{{seed, 0}}

	for head := 0; head < len(queue); head++ {
		cur := queue[head]
		if maxGenerations > 0 && cur.generation >= maxGenerations {
			continue
		}
		generation := cur.generation + 1
		next(cur.ind, func(x string) {
			if x == "" || visited[x] {
				return
			}
			ind := d.GetIndividual(x)
			if ind == nil {
				return
			}
			visited[x] = true
			result = append(result, Relative{XRef: x, Generation: generation})
			queue = append(queue, step{ind, generation})
		})
	}

	return result
}

// relativeXRefs projects a walk result onto its XRefs, keeping nil as nil.
func relativeXRefs(rels []Relative) []string {
	if rels == nil {
		return nil
	}
	xrefs := make([]string, len(rels))
	for i, r := range rels {
		xrefs[i] = r.XRef
	}
	return xrefs
}

// Descendants returns the Individuals that are transitive descendants
// of this individual. Convenience wrapper over Document.Descendants
// that resolves XRefs to Individual records.
//
// Return contract matches Document.Descendants:
//   - nil — receiver is nil, doc is nil, or this individual is not in
//     the document
//   - non-nil []*Individual (possibly empty) — otherwise; the slice
//     holds the resolved descendants in BFS order
func (i *Individual) Descendants(doc *Document) []*Individual {
	if i == nil || doc == nil || doc.GetIndividual(i.XRef) == nil {
		return nil
	}
	xrefs := doc.Descendants(i.XRef)
	result := make([]*Individual, 0, len(xrefs))
	for _, x := range xrefs {
		if ind := doc.GetIndividual(x); ind != nil {
			result = append(result, ind)
		}
	}
	return result
}

// Ancestors returns the Individuals that are transitive ancestors of
// this individual. Convenience wrapper over Document.Ancestors that
// resolves XRefs to Individual records.
//
// Return contract matches Document.Ancestors:
//   - nil — receiver is nil, doc is nil, or this individual is not in
//     the document
//   - non-nil []*Individual (possibly empty) — otherwise; the slice
//     holds the resolved ancestors in BFS order
func (i *Individual) Ancestors(doc *Document) []*Individual {
	if i == nil || doc == nil || doc.GetIndividual(i.XRef) == nil {
		return nil
	}
	xrefs := doc.Ancestors(i.XRef)
	result := make([]*Individual, 0, len(xrefs))
	for _, x := range xrefs {
		if ind := doc.GetIndividual(x); ind != nil {
			result = append(result, ind)
		}
	}
	return result
}
