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

// EscapeLeadingAt escapes a leading "@" in a line value as "@@", per the GEDCOM
// convention that a literal leading "@" in a value must be doubled so it is not
// read as the start of a cross-reference pointer or escape token. Only the
// leading "@" is doubled; the rest of the value is unchanged. A value that does
// not begin with "@" is returned as-is.
//
// Use this when writing a literal value (e.g. a synthesized vendor-tag
// identifier) that could otherwise be pointer-shaped; UnescapeLeadingAt is the
// inverse. Note this deliberately covers any leading "@", not only well-formed
// "@xref@" pointers — a value like "@foo" still needs escaping per the spec.
func EscapeLeadingAt(s string) string {
	if strings.HasPrefix(s, "@") {
		return "@" + s
	}
	return s
}

// UnescapeLeadingAt reverses EscapeLeadingAt: a leading "@@" collapses to a
// single literal "@". A value that does not begin with "@@" is returned as-is.
func UnescapeLeadingAt(s string) string {
	if strings.HasPrefix(s, "@@") {
		return s[1:]
	}
	return s
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
	// rewrite is used at known-XRef definition sites (Record.XRef,
	// Header.Submitter, XRefMap keys). It looks up the mapping
	// directly because the caller has already established the field
	// holds an XRef.
	rewrite := func(p *string) {
		if p == nil || *p == "" {
			return
		}
		if newRef, ok := mapping[*p]; ok {
			*p = newRef
		}
	}
	// rewriteRef is used by walkRecord, which traverses union-type
	// fields like Individual.Notes and SourceCitation.SourceXRef that
	// may hold either an XRef pointer or inline text. Guarding with
	// IsPointerXRef ensures Apply only ever rewrites pointer-shaped
	// values, never inline text that happens to match a mapping key.
	rewriteRef := func(p *string) {
		if p == nil || !IsPointerXRef(*p) {
			return
		}
		rewrite(p)
	}

	applyToRecords(d.Records, mapping, rewrite, rewriteRef)
	applyToHeader(d.Header, rewrite)
	d.XRefMap = remapXRefMap(d.XRefMap, mapping)
}

// applyToRecords rewrites definition sites and walks references on
// every record. Extracted from Apply to keep its cyclomatic complexity
// in check; the two closures are passed in because they capture the
// mapping.
func applyToRecords(records []*Record, mapping map[string]string, rewrite, rewriteRef refCallback) {
	for _, r := range records {
		if r == nil {
			continue
		}
		rewrite(&r.XRef)
		if newXRef, ok := mapping[entityXRef(r.Entity)]; ok {
			setEntityXRef(r.Entity, newXRef)
		}
		walkRecord(r, rewriteRef)
	}
}

// applyToHeader rewrites the Submitter pointer and walks every header
// tag. Header tags go through walkTag which already filters by
// IsPointerXRef, so the bare rewrite closure is sufficient.
func applyToHeader(h *Header, rewrite refCallback) {
	if h == nil {
		return
	}
	rewrite(&h.Submitter)
	for _, t := range h.Tags {
		walkTag(t, rewrite)
	}
}

// remapXRefMap returns a new XRefMap with keys rewritten per mapping.
// Records whose key has no mapping entry retain their original key.
// Returns nil if the input is nil.
func remapXRefMap(in map[string]*Record, mapping map[string]string) map[string]*Record {
	if in == nil {
		return nil
	}
	out := make(map[string]*Record, len(in))
	for k, v := range in {
		if newK, ok := mapping[k]; ok {
			out[newK] = v
		} else {
			out[k] = v
		}
	}
	return out
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
	// RepositoryRef is the legacy alias of RepositoryLink.XRef (the decoder
	// populates both from the same REPO pointer). Walk only the canonical
	// field to avoid visiting/rewriting the same logical pointer twice, then
	// re-sync the alias so an Apply rewrite propagates to it. When there is no
	// RepositoryLink (legacy-only Source), fall back to walking RepositoryRef.
	if s.RepositoryLink != nil {
		cb(&s.RepositoryLink.XRef)
		s.RepositoryRef = s.RepositoryLink.XRef
	} else {
		cb(&s.RepositoryRef)
	}
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
