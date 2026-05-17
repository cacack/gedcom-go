package merge

import (
	"errors"
	"fmt"

	"github.com/cacack/gedcom-go/v2/gedcom"
)

// RemapXRefs returns a new document with every XRef remapped by the
// transform function. The input document is not modified.
//
// For each record in doc with a non-empty XRef, transform is called
// with the current XRef (e.g., "@I1@") and must return the new XRef
// (e.g., "@A_I1@"). The returned XRef must:
//
//   - have the @xref@ pointer shape (gedcom.IsPointerXRef returns true)
//   - not collide with another transform output in the same call
//
// Records with an empty XRef (records that have no definition site,
// such as the implicit header) are skipped and contribute no mapping
// entry.
//
// On success, RemapXRefs returns:
//
//   - the remapped document (a deep copy of doc with all references
//     rewritten, including header references, raw tag references, and
//     entries in Document.XRefMap)
//   - the old → new mapping that was applied (a fresh map; caller
//     mutations do not affect internal state)
//   - nil error
//
// On a transform failure, RemapXRefs returns a nil document, a nil
// mapping, and an error wrapping ErrInvalidRemap. Use errors.As with
// *RemapError to recover the offending input/output pair. RemapXRefs
// does not return a partially-remapped document.
//
// If doc is nil or transform is nil, RemapXRefs returns a plain error
// (errors.New) — these conditions are programmer mistakes rather than
// data problems and intentionally do NOT wrap ErrInvalidRemap. Callers
// using errors.Is(err, ErrInvalidRemap) to distinguish data errors from
// programmer errors will get the expected false on nil-input.
//
// Example transform that prefixes every XRef:
//
//	doc2, mapping, err := merge.RemapXRefs(doc, func(old string) string {
//	    // old is shaped like "@I1@"; produce "@A_I1@" (valid xref shape).
//	    return "@A_" + old[1:]
//	})
func RemapXRefs(doc *gedcom.Document, transform func(old string) string) (*gedcom.Document, map[string]string, error) {
	if doc == nil {
		return nil, nil, errors.New("merge: doc is nil")
	}
	if transform == nil {
		return nil, nil, errors.New("merge: transform is nil")
	}

	mapping := make(map[string]string)
	// seenNew maps a newly produced XRef back to the old XRef that
	// produced it, so we can name the conflicting pair in the error.
	seenNew := make(map[string]string)

	for _, r := range doc.Records {
		if r == nil || r.XRef == "" {
			continue
		}
		newXRef := transform(r.XRef)
		if !gedcom.IsPointerXRef(newXRef) {
			return nil, nil, &RemapError{
				Old:    r.XRef,
				New:    newXRef,
				Reason: "transform produced a malformed XRef (must satisfy gedcom.IsPointerXRef)",
			}
		}
		if prevOld, dup := seenNew[newXRef]; dup {
			return nil, nil, &RemapError{
				Old:    r.XRef,
				New:    newXRef,
				Reason: fmt.Sprintf("collides with mapping for %q", prevOld),
			}
		}
		seenNew[newXRef] = r.XRef
		mapping[r.XRef] = newXRef
	}

	out := doc.Clone()
	gedcom.Apply(out, mapping)

	// Return a defensive copy so caller mutations don't surprise us or
	// the document we just produced.
	result := make(map[string]string, len(mapping))
	for k, v := range mapping {
		result[k] = v
	}
	return out, result, nil
}
