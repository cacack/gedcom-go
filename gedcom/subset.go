package gedcom

import (
	"errors"
	"fmt"
)

// ErrUnknownXRef is returned by Subset when a seed XRef or a transitively
// referenced XRef cannot be resolved in the source document. Callers can
// use errors.Is to detect this case; use errors.As with *UnknownXRefError
// to recover the specific missing XRef.
var ErrUnknownXRef = errors.New("unknown xref")

// UnknownXRefError records a specific XRef that could not be resolved.
// It is returned by Subset wrapped with ErrUnknownXRef.
type UnknownXRefError struct {
	// XRef is the missing cross-reference identifier.
	XRef string
	// IsSeed is true when XRef was in the seed list passed to Subset,
	// false when it was discovered while walking the reference graph
	// from an included record.
	IsSeed bool
}

func (e *UnknownXRefError) Error() string {
	if e.IsSeed {
		return fmt.Sprintf("unknown xref %q in seed list", e.XRef)
	}
	return fmt.Sprintf("unknown xref %q reachable from included records", e.XRef)
}

func (e *UnknownXRefError) Is(target error) bool {
	return target == ErrUnknownXRef
}

// Subset returns a new self-contained document containing the records
// identified by the seed XRefs plus everything transitively referenced
// by them (families, sources, notes, media, repositories, submitters,
// shared notes). The source document is not mutated.
//
// XRefs in the result are preserved exactly. Callers who need fresh
// IDs to avoid collisions when later combining documents can apply an
// XRef remap as a separate step.
//
// Header policy: the returned document always has a non-nil Header.
// When the source has a Header, the new one carries Version, Encoding,
// SourceSystem, Date, Language, Copyright, AncestryTreeID, and Schema
// from it. When the source's Header is nil, an empty *Header is
// returned so callers can safely access sub.Header.Version without a
// nil check. The Submitter pointer is preserved only when the
// referenced submitter record is in the closure; otherwise it is
// cleared. Raw header Tags are copied except for any tag whose XRef
// field points at a record not in the closure, which is dropped to
// keep the result self-contained.
//
// Subset operates in strict mode for seeds: each seed must have the
// @xref@ pointer shape and must resolve to a record in the source,
// otherwise an error wrapping ErrUnknownXRef is returned. Empty
// strings, the @VOID@ sentinel, and malformed seeds all error rather
// than being silently dropped, so caller mistakes surface immediately.
// Duplicate seed XRefs are deduplicated silently. A nil or empty
// seeds slice produces an empty document with the carried-over header.
//
// Strict mode also applies during the closure walk: any reference
// followed from an included record that does not resolve returns an
// error wrapping ErrUnknownXRef. The GEDCOM 7.0 "@VOID@" sentinel is
// silently skipped during the walk (it does not pull anything into
// the closure and does not error). Inline note text and inline
// citation values that do not have the @xref@ pointer shape are
// likewise ignored during the walk.
//
// Raw tag closure: when walking raw Tags on records, both Tag.XRef
// (the parser-populated pointer field) and Tag.Value are inspected for
// pointer-shaped references. Depending on the parser, custom or vendor
// tags may surface an XRef in either field; following both ensures that
// vendor-extension references are not silently dropped from the subset.
func (d *Document) Subset(xrefs []string) (*Document, error) {
	if d == nil {
		return nil, errors.New("subset: source document is nil")
	}

	closure, err := d.subsetClosure(xrefs)
	if err != nil {
		return nil, err
	}

	out := &Document{
		Header:  subsetHeader(d, closure),
		Trailer: d.Trailer.Clone(),
		XRefMap: make(map[string]*Record, len(closure)),
		Vendor:  d.Vendor,
		Schema:  cloneSchemaDefinition(d.Schema),
	}

	out.Records = make([]*Record, 0, len(closure))
	for _, record := range d.Records {
		if record == nil || record.XRef == "" {
			continue
		}
		if !closure[record.XRef] {
			continue
		}
		copied := record.Clone()
		out.Records = append(out.Records, copied)
		out.XRefMap[copied.XRef] = copied
	}

	return out, nil
}

// subsetClosure computes the transitive set of XRefs reachable from
// xrefs. Returns an error if any seed or transitive reference cannot
// be resolved.
func (d *Document) subsetClosure(xrefs []string) (map[string]bool, error) {
	closure := make(map[string]bool, len(xrefs))
	queue := make([]string, 0, len(xrefs))

	for _, seed := range xrefs {
		if closure[seed] {
			continue
		}
		if !IsPointerXRef(seed) || d.GetRecord(seed) == nil {
			return nil, &UnknownXRefError{XRef: seed, IsSeed: true}
		}
		closure[seed] = true
		queue = append(queue, seed)
	}

	var walkErr error
	visit := func(ref string) {
		if walkErr != nil {
			return
		}
		if !IsPointerXRef(ref) || closure[ref] {
			return
		}
		if d.GetRecord(ref) == nil {
			walkErr = &UnknownXRefError{XRef: ref}
			return
		}
		closure[ref] = true
		queue = append(queue, ref)
	}

	// Every XRef in queue is already validated in closure, so the loop
	// can fetch records directly without re-checking for nil.
	for head := 0; head < len(queue); head++ {
		Visit(d.XRefMap[queue[head]], visit)
		if walkErr != nil {
			return nil, walkErr
		}
	}

	return closure, nil
}

// subsetHeader builds the header for a subset document. Version,
// encoding, and similar file-level metadata are preserved. The
// Submitter pointer is kept only when the referenced submitter is in
// the closure. When the source has no header, an empty *Header is
// returned (never nil) so callers can rely on sub.Header being usable.
func subsetHeader(src *Document, closure map[string]bool) *Header {
	if src.Header == nil {
		return &Header{}
	}
	h := &Header{
		Version:        src.Header.Version,
		Encoding:       src.Header.Encoding,
		SourceSystem:   src.Header.SourceSystem,
		Date:           src.Header.Date,
		Language:       src.Header.Language,
		Copyright:      src.Header.Copyright,
		AncestryTreeID: src.Header.AncestryTreeID,
	}
	if src.Header.Submitter != "" && closure[src.Header.Submitter] {
		h.Submitter = src.Header.Submitter
	}
	for _, tag := range src.Header.Tags {
		if tag == nil {
			continue
		}
		if tag.XRef != "" && !closure[tag.XRef] {
			continue
		}
		// Tag.Value can also carry a pointer (parsers differ on which
		// field they populate for vendor tags). Drop the tag if its
		// Value resolves to a record outside the closure so the subset
		// stays self-contained.
		if IsPointerXRef(tag.Value) && !closure[tag.Value] {
			continue
		}
		h.Tags = append(h.Tags, tag.Clone())
	}
	return h
}
