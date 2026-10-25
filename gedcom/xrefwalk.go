package gedcom

import (
	"strings"
)

// nullXRef is the GEDCOM 7.0 null pointer. GEDCOM 5.5 and 5.5.1 do not
// reserve it, so there it is an ordinary identifier.
const nullXRef = "@VOID@"

// refCallback is invoked for every string field that may hold an XRef
// reference. Visit reads the pointee; Apply rewrites it in place.
type refCallback func(*string)

// IsPointerXRef reports whether s is shaped like a GEDCOM XRef pointer
// (@xref@) and is not the GEDCOM 7.0 "@VOID@" sentinel (intentionally
// null pointer). Fields like Individual.NoteXRefs and
// SourceCitation.SourceXRef are pointer-typed by contract but hold plain
// strings; this distinguishes a real pointer from anything else a caller
// may have put there, so only actual pointers are followed.
//
// The body (between the delimiting @s) must contain no whitespace and no
// interior @ characters. Per the GEDCOM spec, a literal @ inside a value
// is escaped as @@; an un-escaped interior @ means the value either
// straddles two XRefs or is malformed, so it is not a valid pointer.
// This is the GEDCOM 7.0 grammar; [IsPointerXRefForVersion] also accepts the
// spaced identifiers GEDCOM 5.5 and 5.5.1 allow.
func IsPointerXRef(s string) bool {
	if len(s) < 3 || s[0] != '@' || s[len(s)-1] != '@' {
		return false
	}
	if s == nullXRef {
		return false
	}
	// XRefs do not contain whitespace or interior @ by spec; inline text
	// often contains both.
	return !strings.ContainsAny(s[1:len(s)-1], " \t\n\r@")
}

// IsPointerXRefForVersion reports whether s is an XRef pointer under the
// identifier grammar of GEDCOM version v.
//
// GEDCOM 5.5 and 5.5.1 list the space (0x20) among the pointer characters, so
// for those versions s may also contain spaces after its first character
// ("@N 1@"); every other rule of [IsPointerXRef] still applies, and s must be
// the whole pointer, so "@N 1@ see" is not one. For GEDCOM 7.0, whose Xref
// grammar has no space, and for any other version it is [IsPointerXRef].
//
// Pass the version the file's header declares, as the decoder does (see
// version.DeclaredVersion). Document.Header.Version may instead be a guessed
// version — version detection falls back to tag heuristics and then 5.5 — so
// using it would apply the 5.5 grammar to files that never claimed it.
func IsPointerXRefForVersion(s string, v Version) bool {
	if v != Version55 && v != Version551 {
		return IsPointerXRef(s)
	}
	if len(s) < 3 || s[0] != '@' || s[len(s)-1] != '@' || s[1] == ' ' {
		return false
	}
	if s == nullXRef {
		return false
	}
	// A space is an ordinary identifier character here, so it cannot make s
	// ambiguous; every other XRef exclusion still applies.
	return !strings.ContainsAny(s[1:len(s)-1], "\t\n\r@")
}

// IsPointerXRef reports whether s is an XRef pointer under the identifier
// grammar of the version d's header declares (Header.DeclaredVersion); see
// [IsPointerXRefForVersion]. A document that declares no version -- one built
// in code, or decoded from a file without a recognized GEDC.VERS -- uses the
// GEDCOM 7.0 grammar of [IsPointerXRef], as does a nil d or Header.
func (d *Document) IsPointerXRef(s string) bool {
	return IsPointerXRefForVersion(s, d.declaredVersion())
}

// IsRecordPointer reports whether s, read as a pointer value in d, names a
// record identifier: [Document.IsPointerXRef], or "@VOID@" when d is not
// GEDCOM 7.0. @VOID@ is the GEDCOM 7.0 null pointer but an ordinary
// identifier in GEDCOM 5.5 and 5.5.1, and this uses the same version rule as
// [Apply]: @VOID@ is the null pointer only when Header.Version is 7.0, so in
// a document with no Header or no Header.Version it is an ordinary identifier.
// [Document.Subset] and the validator's broken-pointer check use it.
func (d *Document) IsRecordPointer(s string) bool {
	if s == nullXRef {
		return !d.voidIsNull()
	}
	return d.IsPointerXRef(s)
}

// voidIsNull reports whether @VOID@ is the null pointer in d: Header.Version
// is GEDCOM 7.0. It keys on Header.Version, not Header.DeclaredVersion; see
// [Apply].
func (d *Document) voidIsNull() bool {
	return d != nil && d.Header != nil && d.Header.Version == Version70
}

// declaredVersion returns d's Header.DeclaredVersion, or "" for a nil d or
// Header.
func (d *Document) declaredVersion() Version {
	if d == nil || d.Header == nil {
		return ""
	}
	return d.Header.DeclaredVersion
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
// fields) are not visited. Non-pointer-shaped values and the GEDCOM 7.0
// @VOID@ null pointer are filtered before reaching the callback. Pointers are
// matched with the GEDCOM 7.0 grammar ([IsPointerXRef]); use
// [VisitForVersion] to also visit the spaced pointers GEDCOM 5.5 and 5.5.1
// allow ("@N 1@") and @VOID@, an ordinary identifier in those versions.
func Visit(r *Record, visit func(string)) {
	VisitForVersion(r, "", visit)
}

// VisitForVersion is [Visit] with pointers matched under the identifier
// grammar of GEDCOM version v ([IsPointerXRefForVersion]). Pass the version
// the document declares, Document.Header.DeclaredVersion, not the possibly
// guessed Header.Version.
//
// @VOID@ is visited like any other pointer when v is GEDCOM 5.5 or 5.5.1,
// where it is an ordinary identifier. For 7.0, and for any other v (which
// selects the 7.0 grammar), it is the null pointer and is not visited.
//
// This differs from [Document.IsRecordPointer], the document-level rule
// [Apply] also uses, when no version is known: v == "" (as [Visit] passes)
// skips @VOID@, while a document with no Header.Version treats it as an
// ordinary identifier. The two also read different fields: v is the declared
// version, IsRecordPointer keys @VOID@ on Header.Version. To classify pointers
// the way [Document.Subset] and the validator do, use
// [Document.IsRecordPointer].
func VisitForVersion(r *Record, v Version, visit func(string)) {
	if r == nil || visit == nil {
		return
	}
	voidIsOrdinary := v == Version55 || v == Version551
	cb := func(p *string) {
		if p == nil {
			return
		}
		if IsPointerXRefForVersion(*p, v) || (voidIsOrdinary && *p == nullXRef) {
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
// "@VOID@" is the GEDCOM 7.0 null pointer, so in a document whose
// Header.Version is 7.0 a @VOID@ reference is never rewritten. In any other
// document, such as a 5.5/5.5.1 one where @VOID@ is an ordinary identifier,
// a mapping entry for "@VOID@" rewrites its references like any other key.
// This keys on Header.Version, not Header.DeclaredVersion, because the
// converter renames a 5.5/5.5.1 record named @VOID@ before it stamps the
// target version 7.0, whatever version the source file declared.
// [Document.IsRecordPointer] applies the same rule.
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
	// rewriteRef is used by walkRecord, which traverses string fields
	// like Individual.NoteXRefs and SourceCitation.SourceXRef that are
	// pointer-typed by contract but hold plain strings, and raw tag
	// values. Only a whole value that is pointer-shaped under the most
	// permissive grammar (GEDCOM 5.5.1, which allows spaces such as
	// "@N 1@") is looked up, so inline text is never rewritten.
	//
	// @VOID@ is not pointer-shaped, so it is let through separately when the
	// document is not GEDCOM 7.0, where it is not the null pointer.
	voidIsNull := d.voidIsNull()
	rewriteRef := func(p *string) {
		if p == nil {
			return
		}
		if *p == nullXRef {
			if voidIsNull {
				return
			}
		} else if !IsPointerXRefForVersion(*p, Version551) {
			return
		}
		rewrite(p)
	}

	applyToRecords(d.Records, mapping, rewrite, rewriteRef)
	applyToHeader(d.Header, rewrite, rewriteRef)
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
// tag with rewriteRef, which filters out non-pointer tag values.
func applyToHeader(h *Header, rewrite, rewriteRef refCallback) {
	if h == nil {
		return
	}
	rewrite(&h.Submitter)
	for _, t := range h.Tags {
		walkTag(t, rewriteRef)
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
// Both fields are passed unfiltered, like every other walked field: the
// callback decides which values are pointers (VisitForVersion uses
// IsPointerXRefForVersion; Apply also accepts spaced GEDCOM 5.5/5.5.1
// pointers that are mapping keys), so plain text like "John Smith" is never
// acted on.
func walkTag(t *Tag, cb refCallback) {
	if t == nil {
		return
	}
	cb(&t.XRef)
	cb(&t.Value)
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
		walkStrings(i.ChildInFamilies[k].NoteXRefs, cb)
	}
	for k := range i.SpouseInFamilies {
		cb(&i.SpouseInFamilies[k].FamilyXRef)
		walkStrings(i.SpouseInFamilies[k].NoteXRefs, cb)
	}
	for _, n := range i.Names {
		walkPersonalName(n, cb)
	}
	walkNotes(i.NoteXRefs, cb)
	walkAssociations(i.Associations, cb)
	walkCitations(i.SourceCitations, cb)
	walkMediaLinks(i.Media, cb)
	for _, ev := range i.Events {
		walkEvent(ev, cb)
	}
	for _, at := range i.Attributes {
		walkAttribute(at, cb)
	}
	walkLDSOrdinances(i.LDSOrdinances, cb)
	walkChangeDate(i.ChangeDate, cb)
	walkChangeDate(i.CreationDate, cb)
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
	walkNotes(f.NoteXRefs, cb)
	walkCitations(f.SourceCitations, cb)
	walkMediaLinks(f.Media, cb)
	for _, ev := range f.Events {
		walkEvent(ev, cb)
	}
	for _, at := range f.Attributes {
		walkAttribute(at, cb)
	}
	walkLDSOrdinances(f.LDSOrdinances, cb)
	walkChangeDate(f.ChangeDate, cb)
	walkChangeDate(f.CreationDate, cb)
	for _, t := range f.Tags {
		walkTag(t, cb)
	}
}

func walkSource(s *Source, cb refCallback) {
	if s == nil {
		return
	}
	for _, link := range s.RepositoryLinks {
		if link != nil {
			cb(&link.XRef)
		}
	}
	walkNotes(s.NoteXRefs, cb)
	for _, link := range s.RepositoryLinks {
		if link != nil {
			walkNotes(link.NoteXRefs, cb)
		}
	}
	walkMediaLinks(s.Media, cb)
	walkChangeDate(s.ChangeDate, cb)
	walkChangeDate(s.CreationDate, cb)
	for _, t := range s.Tags {
		walkTag(t, cb)
	}
}

func walkRepository(r *Repository, cb refCallback) {
	if r == nil {
		return
	}
	walkNotes(r.NoteXRefs, cb)
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
	walkNotes(m.NoteXRefs, cb)
	walkStrings(m.SharedNoteXRefs, cb)
	walkCitations(m.SourceCitations, cb)
	walkChangeDate(m.ChangeDate, cb)
	walkChangeDate(m.CreationDate, cb)
	for _, t := range m.Tags {
		walkTag(t, cb)
	}
}

func walkSubmitter(s *Submitter, cb refCallback) {
	if s == nil {
		return
	}
	walkNotes(s.NoteXRefs, cb)
	for _, t := range s.Tags {
		walkTag(t, cb)
	}
}

func walkSharedNote(s *SharedNote, cb refCallback) {
	if s == nil {
		return
	}
	walkCitations(s.SourceCitations, cb)
	walkChangeDate(s.ChangeDate, cb)
	for _, t := range s.Tags {
		walkTag(t, cb)
	}
}

func walkPersonalName(n *PersonalName, cb refCallback) {
	if n == nil {
		return
	}
	walkNotes(n.NoteXRefs, cb)
	walkCitations(n.SourceCitations, cb)
}

func walkEvent(e *Event, cb refCallback) {
	if e == nil {
		return
	}
	walkNotes(e.NoteXRefs, cb)
	walkPlaceDetail(e.PlaceDetail, cb)
	walkCitations(e.SourceCitations, cb)
	walkMediaLinks(e.Media, cb)
	walkAssociations(e.Associations, cb)
}

func walkAttribute(a *Attribute, cb refCallback) {
	if a == nil {
		return
	}
	walkNotes(a.NoteXRefs, cb)
	walkPlaceDetail(a.PlaceDetail, cb)
	walkCitations(a.SourceCitations, cb)
	walkMediaLinks(a.Media, cb)
	walkAssociations(a.Associations, cb)
}

func walkCitations(citations []*SourceCitation, cb refCallback) {
	for _, sc := range citations {
		if sc == nil {
			continue
		}
		cb(&sc.SourceXRef)
		walkNotes(sc.NoteXRefs, cb)
	}
}

// walkStrings visits every element of a string slice in place, so a
// rewriting callback can replace the values it is given.
func walkStrings(ss []string, cb refCallback) {
	for k := range ss {
		cb(&ss[k])
	}
}

// walkNotes visits a structure's note pointers: the NoteXRefs slice,
// which holds shared-note pointers and nothing else.
//
// InlineNotes is deliberately absent. It holds note text, never a
// pointer, and an XRef-shaped string there is payload rather than a
// reference.
func walkNotes(noteXRefs []string, cb refCallback) {
	walkStrings(noteXRefs, cb)
}

// walkAssociations visits an ASSO structure's pointer to the associated
// individual, plus the notes and citations hanging off it.
func walkAssociations(assocs []*Association, cb refCallback) {
	for _, a := range assocs {
		if a == nil {
			continue
		}
		cb(&a.IndividualXRef)
		walkNotes(a.NoteXRefs, cb)
		walkCitations(a.SourceCitations, cb)
	}
}

// walkLDSOrdinances visits an ordinance's FAMC pointer and its notes.
func walkLDSOrdinances(ords []*LDSOrdinance, cb refCallback) {
	for _, ord := range ords {
		if ord == nil {
			continue
		}
		cb(&ord.FamilyXRef)
		walkNotes(ord.NoteXRefs, cb)
	}
}

// walkChangeDate visits the notes on a CHAN or CREA structure.
func walkChangeDate(cd *ChangeDate, cb refCallback) {
	if cd == nil {
		return
	}
	walkNotes(cd.NoteXRefs, cb)
}

func walkMediaLinks(links []*MediaLink, cb refCallback) {
	for _, ml := range links {
		if ml == nil {
			continue
		}
		cb(&ml.MediaXRef)
		walkStrings(ml.NoteXRefs, cb)
	}
}

// walkPlaceDetail visits the note pointers on a PLAC structure. A place holds
// no other reference, but its NOTE subordinates can point at shared notes.
func walkPlaceDetail(p *PlaceDetail, cb refCallback) {
	if p == nil {
		return
	}
	walkStrings(p.NoteXRefs, cb)
}
