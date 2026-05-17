package gedcom

import (
	"strings"
)

// refCallback is invoked for every string field that may hold an XRef
// reference. Visit reads the pointee; Apply rewrites it in place.
type refCallback func(*string)

// IsPointerXRef reports whether s is shaped like a GEDCOM XRef pointer
// (@xref@) and is not the GEDCOM 7.0 "@VOID@" sentinel (intentionally
// null pointer). Fields like Individual.Notes and
// SourceCitation.SourceXRef may carry either an XRef pointer or inline
// text; this distinguishes the two so callers only follow actual pointers.
//
// The body (between the delimiting @s) must contain no whitespace and no
// interior @ characters. Per the GEDCOM spec, a literal @ inside a value
// is escaped as @@; an un-escaped interior @ means the value either
// straddles two XRefs or is malformed, so it is not a valid pointer.
func IsPointerXRef(s string) bool {
	if len(s) < 3 || s[0] != '@' || s[len(s)-1] != '@' {
		return false
	}
	if s == "@VOID@" {
		return false
	}
	// XRefs do not contain whitespace or interior @ by spec; inline text
	// often contains both.
	return !strings.ContainsAny(s[1:len(s)-1], " \t\n\r@")
}

// Visit invokes visit for every pointer-shaped XRef reachable from r's
// Entity and raw Tags. Definition sites (Record.XRef and entity XRef
// fields) are not visited. Non-pointer-shaped values and the @VOID@
// sentinel are filtered before reaching the callback.
func Visit(r *Record, visit func(string)) {
	if r == nil || visit == nil {
		return
	}
	cb := func(p *string) {
		if p != nil && IsPointerXRef(*p) {
			visit(*p)
		}
	}
	walkRecord(r, cb)
}

// Apply rewrites every XRef occurrence in d using mapping. References
// with no mapping entry are left unchanged. A nil document or empty
// mapping is a no-op.
//
// Apply updates: Record.XRef, entity XRef fields, every pointer-reference
// field on every typed entity, XRef-shaped values in raw Tags (both
// Tag.XRef and Tag.Value), the Header.Submitter and Header tags, and the
// keys of Document.XRefMap.
//
// Apply mutates d in place. Most callers should reach for the higher-
// level merge.RemapXRefs instead, which clones the document, validates
// that the transform produces well-formed non-colliding XRefs, and
// returns the applied mapping. Apply is the low-level primitive Apply
// is exposed because it must be — but it is unsafe in the sense that
// nothing here verifies the mapping is collision-free or shape-correct.
func Apply(d *Document, mapping map[string]string) {
	if d == nil || len(mapping) == 0 {
		return
	}
	rewrite := func(p *string) {
		if p == nil || *p == "" {
			return
		}
		if newRef, ok := mapping[*p]; ok {
			*p = newRef
		}
	}

	// Update definition sites and walk references inside each record.
	for _, r := range d.Records {
		if r == nil {
			continue
		}
		rewrite(&r.XRef)
		if newXRef, ok := mapping[entityXRef(r.Entity)]; ok {
			setEntityXRef(r.Entity, newXRef)
		}
		walkRecord(r, rewrite)
	}

	// Update header references.
	if d.Header != nil {
		rewrite(&d.Header.Submitter)
		for _, t := range d.Header.Tags {
			walkTag(t, rewrite)
		}
	}

	// Rebuild XRefMap with remapped keys.
	if d.XRefMap != nil {
		newMap := make(map[string]*Record, len(d.XRefMap))
		for k, v := range d.XRefMap {
			if newK, ok := mapping[k]; ok {
				newMap[newK] = v
			} else {
				newMap[k] = v
			}
		}
		d.XRefMap = newMap
	}
}

// entityXRef returns the XRef field on the typed entity for an
// entity-level definition site, or "" if the entity is nil or an
// unknown type. The per-case switch exists because the entity types
// do not share a common interface for their XRef field. Each case
// guards against a typed-nil pointer so reading the field never panics.
//
//nolint:gocyclo // 8 entity types × per-case nil guard; intrinsic shape
func entityXRef(entity interface{}) string {
	switch e := entity.(type) {
	case *Individual:
		if e == nil {
			return ""
		}
		return e.XRef
	case *Family:
		if e == nil {
			return ""
		}
		return e.XRef
	case *Source:
		if e == nil {
			return ""
		}
		return e.XRef
	case *Repository:
		if e == nil {
			return ""
		}
		return e.XRef
	case *Note:
		if e == nil {
			return ""
		}
		return e.XRef
	case *MediaObject:
		if e == nil {
			return ""
		}
		return e.XRef
	case *Submitter:
		if e == nil {
			return ""
		}
		return e.XRef
	case *SharedNote:
		if e == nil {
			return ""
		}
		return e.XRef
	}
	return ""
}

// setEntityXRef writes newXRef into the typed entity's XRef field.
// Unknown types and typed-nil entities are ignored.
//
//nolint:gocyclo // 8 entity types × per-case nil guard; intrinsic shape
func setEntityXRef(entity interface{}, newXRef string) {
	switch e := entity.(type) {
	case *Individual:
		if e != nil {
			e.XRef = newXRef
		}
	case *Family:
		if e != nil {
			e.XRef = newXRef
		}
	case *Source:
		if e != nil {
			e.XRef = newXRef
		}
	case *Repository:
		if e != nil {
			e.XRef = newXRef
		}
	case *Note:
		if e != nil {
			e.XRef = newXRef
		}
	case *MediaObject:
		if e != nil {
			e.XRef = newXRef
		}
	case *Submitter:
		if e != nil {
			e.XRef = newXRef
		}
	case *SharedNote:
		if e != nil {
			e.XRef = newXRef
		}
	}
}

// walkRecord invokes cb for every XRef-bearing field reachable from r's
// Entity and raw Tags. Definition sites (Record.XRef, entity XRef
// fields) are NOT walked here; Apply handles those separately so Visit
// can ignore them.
func walkRecord(r *Record, cb refCallback) {
	if r == nil {
		return
	}
	walkEntity(r.Entity, cb)
	for _, t := range r.Tags {
		walkTag(t, cb)
	}
}

// walkTag invokes cb for both Tag.XRef (parser-provided pointer field)
// and Tag.Value when Value is XRef-shaped. The dual coverage unifies
// the two existing walkers: converter inspects tag.Value, subset reads
// tag.XRef. Centralizing both here means raw closure references survive
// remap, no matter which field the parser populated.
//
// Both fields are filtered through IsPointerXRef before reaching cb so
// non-pointer values (empty strings, plain text like "John Smith",
// @VOID@ sentinels) are never passed to the callback. This protects
// Apply from accidentally rewriting non-pointer text that coincidentally
// matches a mapping key, and short-circuits the empty-string case that
// dominates real GEDCOM files.
func walkTag(t *Tag, cb refCallback) {
	if t == nil {
		return
	}
	if IsPointerXRef(t.XRef) {
		cb(&t.XRef)
	}
	if IsPointerXRef(t.Value) {
		cb(&t.Value)
	}
}

func walkEntity(entity interface{}, cb refCallback) {
	switch e := entity.(type) {
	case *Individual:
		walkIndividual(e, cb)
	case *Family:
		walkFamily(e, cb)
	case *Source:
		walkSource(e, cb)
	case *Repository:
		walkRepository(e, cb)
	case *Note:
		walkNote(e, cb)
	case *MediaObject:
		walkMediaObject(e, cb)
	case *Submitter:
		walkSubmitter(e, cb)
	case *SharedNote:
		walkSharedNote(e, cb)
	}
}

func walkIndividual(i *Individual, cb refCallback) {
	if i == nil {
		return
	}
	for k := range i.ChildInFamilies {
		cb(&i.ChildInFamilies[k].FamilyXRef)
	}
	for k := range i.SpouseInFamilies {
		cb(&i.SpouseInFamilies[k])
	}
	for k := range i.Notes {
		cb(&i.Notes[k])
	}
	for _, a := range i.Associations {
		if a == nil {
			continue
		}
		cb(&a.IndividualXRef)
		for k := range a.Notes {
			cb(&a.Notes[k])
		}
		walkCitations(a.SourceCitations, cb)
	}
	walkCitations(i.SourceCitations, cb)
	walkMediaLinks(i.Media, cb)
	for _, ev := range i.Events {
		walkEvent(ev, cb)
	}
	for _, at := range i.Attributes {
		walkAttribute(at, cb)
	}
	for _, ord := range i.LDSOrdinances {
		if ord == nil {
			continue
		}
		cb(&ord.FamilyXRef)
	}
	for _, t := range i.Tags {
		walkTag(t, cb)
	}
}

func walkFamily(f *Family, cb refCallback) {
	if f == nil {
		return
	}
	cb(&f.Husband)
	cb(&f.Wife)
	for k := range f.Children {
		cb(&f.Children[k])
	}
	for k := range f.Notes {
		cb(&f.Notes[k])
	}
	walkCitations(f.SourceCitations, cb)
	walkMediaLinks(f.Media, cb)
	for _, ev := range f.Events {
		walkEvent(ev, cb)
	}
	for _, ord := range f.LDSOrdinances {
		if ord == nil {
			continue
		}
		cb(&ord.FamilyXRef)
	}
	for _, t := range f.Tags {
		walkTag(t, cb)
	}
}

func walkSource(s *Source, cb refCallback) {
	if s == nil {
		return
	}
	cb(&s.RepositoryRef)
	for k := range s.Notes {
		cb(&s.Notes[k])
	}
	walkMediaLinks(s.Media, cb)
	for _, t := range s.Tags {
		walkTag(t, cb)
	}
}

func walkRepository(r *Repository, cb refCallback) {
	if r == nil {
		return
	}
	for k := range r.Notes {
		cb(&r.Notes[k])
	}
	for _, t := range r.Tags {
		walkTag(t, cb)
	}
}

func walkNote(n *Note, cb refCallback) {
	if n == nil {
		return
	}
	for _, t := range n.Tags {
		walkTag(t, cb)
	}
}

func walkMediaObject(m *MediaObject, cb refCallback) {
	if m == nil {
		return
	}
	for k := range m.Notes {
		cb(&m.Notes[k])
	}
	walkCitations(m.SourceCitations, cb)
	for _, t := range m.Tags {
		walkTag(t, cb)
	}
}

func walkSubmitter(s *Submitter, cb refCallback) {
	if s == nil {
		return
	}
	for k := range s.Notes {
		cb(&s.Notes[k])
	}
	for _, t := range s.Tags {
		walkTag(t, cb)
	}
}

func walkSharedNote(s *SharedNote, cb refCallback) {
	if s == nil {
		return
	}
	walkCitations(s.SourceCitations, cb)
	for _, t := range s.Tags {
		walkTag(t, cb)
	}
}

func walkEvent(e *Event, cb refCallback) {
	if e == nil {
		return
	}
	for k := range e.Notes {
		cb(&e.Notes[k])
	}
	walkCitations(e.SourceCitations, cb)
	walkMediaLinks(e.Media, cb)
	for _, t := range e.Tags {
		walkTag(t, cb)
	}
}

func walkAttribute(a *Attribute, cb refCallback) {
	if a == nil {
		return
	}
	walkCitations(a.SourceCitations, cb)
}

func walkCitations(citations []*SourceCitation, cb refCallback) {
	for _, sc := range citations {
		if sc == nil {
			continue
		}
		cb(&sc.SourceXRef)
	}
}

func walkMediaLinks(links []*MediaLink, cb refCallback) {
	for _, ml := range links {
		if ml == nil {
			continue
		}
		cb(&ml.MediaXRef)
	}
}
