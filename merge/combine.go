package merge

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/cacack/gedcom-go/gedcom"
)

// CollisionStrategy chooses how to handle XRef collisions between
// doc1 and doc2 during Combine.
type CollisionStrategy int

const (
	// ErrorOnCollision (zero value) returns an error if any XRef in
	// doc2 already exists in doc1. Safe default: callers must
	// explicitly opt in to a remapping strategy.
	ErrorOnCollision CollisionStrategy = iota
	// PrefixDoc2 prefixes every colliding XRef in doc2 with
	// CombineOptions.Prefix. Non-colliding XRefs in doc2 are kept as-is.
	// Requires Prefix to be set (non-empty, body-character-safe).
	PrefixDoc2
	// RenumberDoc2 renumbers every colliding XRef in doc2 to a fresh
	// sequential ID per record type, starting after the highest
	// numeric ID in doc1 for that type. Non-colliding XRefs are kept.
	RenumberDoc2
)

// CombineOptions configures Combine behavior.
type CombineOptions struct {
	// CollisionStrategy chooses how XRef collisions between doc1 and
	// doc2 are resolved. The zero value (ErrorOnCollision) refuses any
	// collision, requiring the caller to opt in to mutation.
	CollisionStrategy CollisionStrategy
	// Prefix is the string prepended to colliding XRef bodies when
	// CollisionStrategy == PrefixDoc2. Required for that strategy;
	// must contain no whitespace and must not be empty.
	Prefix string
}

// CombineReport describes what Combine did on success. Combine returns
// a nil report on every error path; callers should check err before
// dereferencing the report. The report records both the remapping
// applied to doc2 and any non-fatal header field differences that were
// discarded in favor of doc1.
type CombineReport struct {
	// RemappedXRefs maps doc2's original XRefs to their new XRefs in
	// the combined document. Only XRefs that actually changed are
	// recorded; no-op entries are filtered. Empty if no remap was
	// needed.
	RemappedXRefs map[string]string
	// HeaderConflicts records header fields where doc2 had a
	// different non-empty value that was dropped in favor of doc1.
	HeaderConflicts []HeaderConflict
}

// HeaderConflict records a single header field where doc1 and doc2
// disagreed and doc1's value was kept.
type HeaderConflict struct {
	// Field is the conflicting field name, e.g. "SourceSystem",
	// "Language", "Copyright", "Schema", "Vendor".
	Field string
	// Doc1 is the value kept in the combined header.
	Doc1 string
	// Doc2 is the value dropped from doc2.
	Doc2 string
}

// Combine merges doc2 into doc1 and returns the combined document.
//
// Combine never mutates its inputs. Both documents are deep-copied;
// references inside doc2 are rewritten according to opts.CollisionStrategy
// so the combined document keeps referential integrity.
//
// Header merge policy:
//
//   - Version and Encoding must be compatible. If both headers have
//     a non-empty value and they differ, Combine returns an error.
//     Equal or one-empty is OK.
//   - For the other header scalar fields (SourceSystem, Date,
//     Language, Copyright, AncestryTreeID, Vendor, Schema) doc1's
//     value is kept. If doc2's value is non-empty AND differs,
//     a HeaderConflict is recorded in the report.
//   - Submitter: doc1's submitter wins if set. If doc1 has none,
//     doc2's submitter is used and is looked up in the remapping
//     (it may have been renamed by collision resolution).
//   - Header tags are concatenated (doc1's first, then doc2's) so
//     custom tags from both documents are preserved.
//   - Trailer: doc1's wins; if doc1 has none, doc2's is used.
//
// Returns an error if either doc is nil, if the opts are invalid for
// the chosen strategy, or if any header compatibility check fails.
// The error wraps ErrIncompatibleHeader for compat failures and
// ErrXRefCollision for collision failures.
func Combine(doc1, doc2 *gedcom.Document, opts CombineOptions) (*gedcom.Document, *CombineReport, error) {
	if doc1 == nil {
		return nil, nil, errors.New("merge: doc1 is nil")
	}
	if doc2 == nil {
		return nil, nil, errors.New("merge: doc2 is nil")
	}

	// Check version/encoding compatibility BEFORE doing any work.
	if err := checkHeaderCompat(doc1.Header, doc2.Header); err != nil {
		return nil, nil, err
	}

	// Detect collisions between doc1 and doc2 XRefs.
	collisions := findCollisions(doc1, doc2)

	// Apply the requested collision strategy to produce a remapped
	// version of doc2. RemapXRefs clones doc2 internally, so doc2
	// itself is never mutated.
	remappedDoc2, fullMapping, err := applyStrategy(doc1, doc2, collisions, opts)
	if err != nil {
		return nil, nil, err
	}

	// Filter mapping to keep only the entries that actually changed.
	changedMapping := make(map[string]string, len(fullMapping))
	for old, newXRef := range fullMapping {
		if old != newXRef {
			changedMapping[old] = newXRef
		}
	}

	// Clone doc1 so we never mutate the input. Records will be
	// concatenated into this output.
	out := doc1.Clone()
	if out.XRefMap == nil {
		out.XRefMap = make(map[string]*gedcom.Record)
	}

	// Append doc2's (already remapped) records.
	for _, r := range remappedDoc2.Records {
		out.Records = append(out.Records, r)
		if r != nil && r.XRef != "" {
			out.XRefMap[r.XRef] = r
		}
	}

	// Merge headers and document-level metadata.
	mergedHeader, headerConflicts := mergeHeaders(doc1.Header, remappedDoc2.Header)
	out.Header = mergedHeader
	headerConflicts = mergeDocMetadata(out, doc1, remappedDoc2, headerConflicts)

	report := &CombineReport{
		RemappedXRefs:   changedMapping,
		HeaderConflicts: headerConflicts,
	}
	return out, report, nil
}

// mergeDocMetadata merges Schema, Vendor, and Trailer from doc1 and
// the remapped doc2 into out, appending any conflicts to conflicts
// and returning the updated slice. Doc1's values win; doc2's values
// are only adopted when doc1's are empty.
func mergeDocMetadata(out, doc1, remappedDoc2 *gedcom.Document, conflicts []HeaderConflict) []HeaderConflict {
	// Schema: prefer doc1's. Record a conflict if doc2 had a
	// distinct non-empty schema we are discarding.
	switch {
	case doc1.Schema == nil && remappedDoc2.Schema != nil:
		out.Schema = cloneSchema(remappedDoc2.Schema)
	case doc1.Schema != nil && remappedDoc2.Schema != nil && !schemasEqual(doc1.Schema, remappedDoc2.Schema):
		conflicts = append(conflicts, HeaderConflict{
			Field: "Schema",
			Doc1:  describeSchema(doc1.Schema),
			Doc2:  describeSchema(remappedDoc2.Schema),
		})
	}

	// Vendor: prefer doc1's. Record a conflict if doc2's non-empty
	// value differs.
	switch {
	case doc1.Vendor == "" && remappedDoc2.Vendor != "":
		out.Vendor = remappedDoc2.Vendor
	case doc1.Vendor != "" && remappedDoc2.Vendor != "" && doc1.Vendor != remappedDoc2.Vendor:
		conflicts = append(conflicts, HeaderConflict{
			Field: "Vendor",
			Doc1:  string(doc1.Vendor),
			Doc2:  string(remappedDoc2.Vendor),
		})
	}

	// Trailer: prefer doc1's, fall back to doc2's.
	if out.Trailer == nil && remappedDoc2.Trailer != nil {
		out.Trailer = remappedDoc2.Trailer.Clone()
	}

	return conflicts
}

// ErrIncompatibleHeader is returned by Combine when doc1 and doc2 have
// header fields that cannot be safely merged (e.g., different GEDCOM
// versions or different character encodings).
var ErrIncompatibleHeader = errors.New("incompatible headers")

// ErrXRefCollision is returned by Combine with CollisionStrategy ==
// ErrorOnCollision when doc1 and doc2 share one or more XRef
// identifiers. Use the listed XRefs to choose a remapping strategy.
var ErrXRefCollision = errors.New("xref collision")

// CombineError reports a header-compatibility or collision failure
// from Combine.
type CombineError struct {
	// Kind names the failure: "version", "encoding", "collision",
	// "prefix-collision", "renumber-collision", "missing-prefix".
	Kind string
	// Doc1 is doc1's value (for version/encoding) or empty.
	Doc1 string
	// Doc2 is doc2's value (for version/encoding) or empty.
	Doc2 string
	// Colliding is the list of colliding XRefs (for collision kinds);
	// truncated for display in Error().
	Colliding []string
}

func (e *CombineError) Error() string {
	switch e.Kind {
	case "version":
		return fmt.Sprintf("merge: incompatible versions: %s vs %s", e.Doc1, e.Doc2)
	case "encoding":
		return fmt.Sprintf("merge: incompatible encodings: %s vs %s", e.Doc1, e.Doc2)
	case "collision":
		return fmt.Sprintf("merge: xref collision: %s", summarizeXRefs(e.Colliding))
	case "prefix-collision":
		return fmt.Sprintf("merge: prefix produced new collision: %s", summarizeXRefs(e.Colliding))
	case "renumber-collision":
		return fmt.Sprintf("merge: renumber produced new collision: %s", summarizeXRefs(e.Colliding))
	case "missing-prefix":
		return "merge: PrefixDoc2 strategy requires non-empty Prefix"
	}
	return "merge: " + e.Kind
}

func (e *CombineError) Is(target error) bool {
	switch target {
	case ErrIncompatibleHeader:
		return e.Kind == "version" || e.Kind == "encoding"
	case ErrXRefCollision:
		return e.Kind == "collision" || e.Kind == "prefix-collision" || e.Kind == "renumber-collision"
	}
	return false
}

// summarizeXRefs formats up to 10 XRefs for display in an error.
func summarizeXRefs(xrefs []string) string {
	const limit = 10
	if len(xrefs) <= limit {
		return strings.Join(xrefs, ", ")
	}
	return strings.Join(xrefs[:limit], ", ") + fmt.Sprintf(" (and %d more)", len(xrefs)-limit)
}

// checkHeaderCompat returns an error if doc1 and doc2 have
// incompatible Version or Encoding fields. Empty values are treated as
// "unknown" and never conflict.
func checkHeaderCompat(h1, h2 *gedcom.Header) error {
	if h1 == nil || h2 == nil {
		return nil
	}
	if h1.Version != "" && h2.Version != "" && h1.Version != h2.Version {
		return &CombineError{
			Kind: "version",
			Doc1: string(h1.Version),
			Doc2: string(h2.Version),
		}
	}
	if h1.Encoding != "" && h2.Encoding != "" && h1.Encoding != h2.Encoding {
		return &CombineError{
			Kind: "encoding",
			Doc1: string(h1.Encoding),
			Doc2: string(h2.Encoding),
		}
	}
	return nil
}

// findCollisions returns the sorted list of XRefs present in both
// doc1 and doc2.
func findCollisions(doc1, doc2 *gedcom.Document) []string {
	set1 := make(map[string]bool)
	for _, r := range doc1.Records {
		if r != nil && r.XRef != "" {
			set1[r.XRef] = true
		}
	}
	var collisions []string
	for _, r := range doc2.Records {
		if r == nil || r.XRef == "" {
			continue
		}
		if set1[r.XRef] {
			collisions = append(collisions, r.XRef)
		}
	}
	return collisions
}

// applyStrategy resolves collisions per opts.CollisionStrategy and
// returns the remapped doc2 plus the full transform mapping (including
// no-op entries — caller filters).
func applyStrategy(doc1, doc2 *gedcom.Document, collisions []string, opts CombineOptions) (*gedcom.Document, map[string]string, error) {
	switch opts.CollisionStrategy {
	case ErrorOnCollision:
		if len(collisions) > 0 {
			return nil, nil, &CombineError{Kind: "collision", Colliding: collisions}
		}
		// No remap needed; just clone doc2 so the caller can safely
		// concatenate without aliasing the input.
		return doc2.Clone(), nil, nil

	case PrefixDoc2:
		return applyPrefixStrategy(doc1, doc2, collisions, opts.Prefix)

	case RenumberDoc2:
		transform := renumberTransform(doc1, doc2, collisions)
		out, mapping, err := RemapXRefs(doc2, transform)
		if err != nil {
			return nil, nil, err
		}
		// Defensive: renumber arithmetic seeds counters above both
		// doc1's and doc2's existing numeric XRefs, so a fresh
		// collision should be impossible. Verify anyway — if a future
		// edit to renumberTransform introduces an off-by-one, this
		// check surfaces it instead of letting silent referential
		// corruption into the combined document.
		if newCollisions := postRemapCollisions(doc1, out); len(newCollisions) > 0 {
			return nil, nil, &CombineError{Kind: "renumber-collision", Colliding: newCollisions}
		}
		return out, mapping, nil
	}
	return nil, nil, fmt.Errorf("merge: unknown CollisionStrategy %d", opts.CollisionStrategy)
}

// applyPrefixStrategy implements the PrefixDoc2 path: validate the
// prefix, remap only colliding XRefs, then verify the remap didn't
// create a fresh collision with doc1 (e.g. doc1 already had the
// prefixed name).
func applyPrefixStrategy(doc1, doc2 *gedcom.Document, collisions []string, prefix string) (*gedcom.Document, map[string]string, error) {
	if prefix == "" {
		return nil, nil, &CombineError{Kind: "missing-prefix"}
	}
	collisionSet := make(map[string]bool, len(collisions))
	for _, c := range collisions {
		collisionSet[c] = true
	}
	transform := prefixTransform(prefix, collisionSet)
	out, mapping, err := RemapXRefs(doc2, transform)
	if err != nil {
		return nil, nil, err
	}
	if newCollisions := postRemapCollisions(doc1, out); len(newCollisions) > 0 {
		return nil, nil, &CombineError{Kind: "prefix-collision", Colliding: newCollisions}
	}
	return out, mapping, nil
}

// postRemapCollisions returns any XRefs in remapped that now collide
// with doc1's XRefs. Used by PrefixDoc2 to detect e.g. doc1 having
// @z_I1@ already when doc2's @I1@ gets prefixed with "z_".
func postRemapCollisions(doc1, remapped *gedcom.Document) []string {
	set1 := make(map[string]bool)
	for _, r := range doc1.Records {
		if r != nil && r.XRef != "" {
			set1[r.XRef] = true
		}
	}
	var collisions []string
	for _, r := range remapped.Records {
		if r == nil || r.XRef == "" {
			continue
		}
		if set1[r.XRef] {
			collisions = append(collisions, r.XRef)
		}
	}
	return collisions
}

// prefixTransform returns a transform that applies prefix only to
// XRefs present in the collisions set. Non-colliding XRefs are
// returned unchanged.
//
// Input is shaped like "@I1@"; output is "@<prefix>I1@" so the
// pointer shape is preserved.
func prefixTransform(prefix string, collisions map[string]bool) func(string) string {
	return func(old string) string {
		if !collisions[old] {
			return old
		}
		return "@" + prefix + old[1:]
	}
}

// renumberTransform builds a transform that renumbers colliding XRefs
// using the next free per-type sequential number above doc1's highest
// numeric XRef for that record type. Non-colliding XRefs in doc2 are
// returned unchanged.
//
// Record-type letter prefixes follow GEDCOM convention:
//
//	I  → Individual
//	F  → Family
//	S  → Source
//	R  → Repository
//	N  → Note
//	M  → Media object
//	U  → Submitter  (GEDCOM 5.5/5.5.1 commonly write @SUBM1@; we use the first letter)
//	SNOTE → Shared note (GEDCOM 7.0)
//
// For each colliding record we look up its RecordType in doc2 and
// pick the next sequential number for that letter, starting at
// max(doc1ID, doc2ID) + 1 to avoid producing another collision.
func renumberTransform(doc1, doc2 *gedcom.Document, collisions []string) func(string) string {
	// Map RecordType → letter prefix used in XRef bodies.
	typePrefix := map[gedcom.RecordType]string{
		gedcom.RecordTypeIndividual: "I",
		gedcom.RecordTypeFamily:     "F",
		gedcom.RecordTypeSource:     "S",
		gedcom.RecordTypeRepository: "R",
		gedcom.RecordTypeNote:       "N",
		gedcom.RecordTypeMedia:      "M",
		gedcom.RecordTypeSubmitter:  "U",
		gedcom.RecordTypeSharedNote: "SNOTE",
	}

	// counters[prefix] holds the next free numeric suffix for that
	// prefix. Seed from doc1 AND doc2's existing numeric XRefs so we
	// never collide with anything already present.
	counters := make(map[string]int)
	seedCounters := func(d *gedcom.Document) {
		for _, r := range d.Records {
			if r == nil || r.XRef == "" {
				continue
			}
			p, n, ok := extractIDNumber(r.XRef)
			if !ok {
				continue
			}
			if n >= counters[p] {
				counters[p] = n + 1
			}
		}
	}
	seedCounters(doc1)
	seedCounters(doc2)

	// Build a record-type lookup keyed by doc2 XRef so the transform
	// closure can pick the right prefix per colliding XRef.
	doc2Type := make(map[string]gedcom.RecordType, len(doc2.Records))
	for _, r := range doc2.Records {
		if r == nil || r.XRef == "" {
			continue
		}
		doc2Type[r.XRef] = r.Type
	}

	collisionSet := make(map[string]bool, len(collisions))
	for _, c := range collisions {
		collisionSet[c] = true
	}

	// Pre-compute the new XRef for every colliding record so the
	// transform closure is cheap and deterministic.
	resolved := make(map[string]string, len(collisions))
	for _, old := range collisions {
		typ := doc2Type[old]
		prefix, ok := typePrefix[typ]
		if !ok {
			// Unknown record type — fall back to extracting the
			// existing XRef's letter prefix, or "X" if that fails.
			if p, _, ok := extractIDNumber(old); ok {
				prefix = p
			} else {
				prefix = "X"
			}
		}
		n := counters[prefix]
		counters[prefix] = n + 1
		resolved[old] = "@" + prefix + strconv.Itoa(n) + "@"
	}

	return func(old string) string {
		if !collisionSet[old] {
			return old
		}
		return resolved[old]
	}
}

// extractIDNumber parses "@I5@" into ("I", 5, true). Returns ok=false
// for XRefs that don't fit the letter-prefix-then-digits shape (e.g.
// "@MYID@", "@SUBM_A@").
func extractIDNumber(xref string) (prefix string, num int, ok bool) {
	if len(xref) < 3 || xref[0] != '@' || xref[len(xref)-1] != '@' {
		return "", 0, false
	}
	body := xref[1 : len(xref)-1]
	// Find the first digit.
	split := -1
	for i, c := range body {
		if c >= '0' && c <= '9' {
			split = i
			break
		}
	}
	if split <= 0 {
		return "", 0, false
	}
	prefix = body[:split]
	digits := body[split:]
	// Body must be letters-then-digits with nothing else after.
	for _, c := range digits {
		if c < '0' || c > '9' {
			return "", 0, false
		}
	}
	n, err := strconv.Atoi(digits)
	if err != nil {
		return "", 0, false
	}
	return prefix, n, true
}

// mergeHeaders combines two headers preferring h1's values for scalar
// fields. Non-empty h2 values that differ from h1 are recorded as
// HeaderConflict entries.
//
// h1 is expected to already be a fresh clone (Combine clones doc1
// via doc1.Clone() before calling). h2 is the (already-remapped) doc2
// header — its Submitter has already been rewritten if needed.
//
// Header.Tags are concatenated rather than merged: doc1's first, then
// doc2's. This preserves any custom tags from either document.
func mergeHeaders(h1, h2 *gedcom.Header) (*gedcom.Header, []HeaderConflict) {
	if h1 == nil && h2 == nil {
		return nil, nil
	}
	if h1 == nil {
		// Take h2 wholesale. Caller already cloned the input docs.
		return h2.Clone(), nil
	}
	if h2 == nil {
		return h1.Clone(), nil
	}

	out := h1.Clone()
	conflicts := mergeScalarHeaderFields(out, h1, h2)

	// Date: prefer h1's. If h1 has a zero Date and h2 has one,
	// adopt h2's silently. (We don't record a conflict for Date
	// because the timestamps will essentially always differ; it
	// would be noise.)
	if h1.Date.IsZero() && !h2.Date.IsZero() {
		out.Date = h2.Date
	}

	// Submitter: doc1 wins if set; otherwise doc2's (already-remapped)
	// submitter is adopted. We do NOT report a conflict here because
	// recording every dropped doc2 submitter would create noise.
	if h1.Submitter == "" && h2.Submitter != "" {
		out.Submitter = h2.Submitter
	}

	// Tags: append h2's after h1's. h1's clone already has the doc1
	// tags; we append clones of doc2's tags.
	for _, t := range h2.Tags {
		out.Tags = append(out.Tags, t.Clone())
	}

	return out, conflicts
}

// mergeScalarHeaderFields applies h1-wins-promote-h2 logic to the
// scalar string fields on Header. It writes promoted values into out
// (which is expected to be a clone of h1) and returns the list of
// recorded conflicts. Extracted from mergeHeaders to keep cyclomatic
// complexity manageable.
func mergeScalarHeaderFields(out, h1, h2 *gedcom.Header) []HeaderConflict {
	var conflicts []HeaderConflict

	// Each entry knows its name, the two string values to compare,
	// and a setter that promotes h2's typed value into out when h1
	// is empty. The setter is closure-bound so the typed Version /
	// Encoding aliases survive without a string switch.
	fields := []struct {
		name string
		v1   string
		v2   string
		set  func()
	}{
		{"Version", string(h1.Version), string(h2.Version), func() { out.Version = h2.Version }},
		{"Encoding", string(h1.Encoding), string(h2.Encoding), func() { out.Encoding = h2.Encoding }},
		{"SourceSystem", h1.SourceSystem, h2.SourceSystem, func() { out.SourceSystem = h2.SourceSystem }},
		{"Language", h1.Language, h2.Language, func() { out.Language = h2.Language }},
		{"Copyright", h1.Copyright, h2.Copyright, func() { out.Copyright = h2.Copyright }},
		{"AncestryTreeID", h1.AncestryTreeID, h2.AncestryTreeID, func() { out.AncestryTreeID = h2.AncestryTreeID }},
	}
	for _, f := range fields {
		// If h1 is empty and h2 is non-empty, promote h2's value
		// silently — there's no real conflict.
		if f.v1 == "" && f.v2 != "" {
			f.set()
			continue
		}
		if f.v2 != "" && f.v1 != f.v2 {
			conflicts = append(conflicts, HeaderConflict{
				Field: f.name,
				Doc1:  f.v1,
				Doc2:  f.v2,
			})
		}
	}
	return conflicts
}

// cloneSchema returns a deep copy of a SchemaDefinition.
func cloneSchema(s *gedcom.SchemaDefinition) *gedcom.SchemaDefinition {
	if s == nil {
		return nil
	}
	out := &gedcom.SchemaDefinition{}
	if s.TagMappings != nil {
		out.TagMappings = make(map[string]string, len(s.TagMappings))
		for k, v := range s.TagMappings {
			out.TagMappings[k] = v
		}
	}
	return out
}

// schemasEqual reports whether two SchemaDefinitions have the same
// tag mappings.
func schemasEqual(a, b *gedcom.SchemaDefinition) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if len(a.TagMappings) != len(b.TagMappings) {
		return false
	}
	for k, v := range a.TagMappings {
		if b.TagMappings[k] != v {
			return false
		}
	}
	return true
}

// describeSchema returns a short string describing a schema for
// reporting in HeaderConflict.
func describeSchema(s *gedcom.SchemaDefinition) string {
	if s == nil || len(s.TagMappings) == 0 {
		return ""
	}
	return fmt.Sprintf("%d tag mapping(s)", len(s.TagMappings))
}
