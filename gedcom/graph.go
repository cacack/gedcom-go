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
	if d == nil || xref == "" {
		return nil
	}
	if d.GetIndividual(xref) == nil {
		return nil
	}

	visited := map[string]bool{xref: true}
	result := []string{}
	queue := []string{xref}

	for head := 0; head < len(queue); head++ {
		ind := d.GetIndividual(queue[head])
		if ind == nil {
			continue
		}

		for _, famXRef := range ind.SpouseInFamilies {
			fam := d.GetFamily(famXRef)
			if fam == nil {
				continue
			}
			for _, childXRef := range fam.Children {
				if childXRef == "" || visited[childXRef] {
					continue
				}
				visited[childXRef] = true
				result = append(result, childXRef)
				queue = append(queue, childXRef)
			}
		}
	}

	return result
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
	if d == nil || xref == "" {
		return nil
	}
	if d.GetIndividual(xref) == nil {
		return nil
	}

	visited := map[string]bool{xref: true}
	result := []string{}
	queue := []string{xref}

	for head := 0; head < len(queue); head++ {
		ind := d.GetIndividual(queue[head])
		if ind == nil {
			continue
		}

		for _, link := range ind.ChildInFamilies {
			fam := d.GetFamily(link.FamilyXRef)
			if fam == nil {
				continue
			}
			for _, parent := range []string{fam.Husband, fam.Wife} {
				if parent == "" || visited[parent] {
					continue
				}
				visited[parent] = true
				result = append(result, parent)
				queue = append(queue, parent)
			}
		}
	}

	return result
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
