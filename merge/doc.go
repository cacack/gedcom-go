// Package merge provides primitives for combining GEDCOM documents.
//
// This package handles the mechanical bookkeeping of merging — XRef
// remapping and document combination. It deliberately does NOT make
// opinionated decisions about record-level merge policy (which field
// wins, how to score fuzzy duplicates, how to reconcile conflicting
// citations, how to detect "the same person"). That policy belongs in
// the consuming application; see docs/ETHOS.md for the rationale
// ("a library is not an application").
//
// What this package does:
//
//   - RemapXRefs: rewrite every XRef in a document according to a
//     caller-supplied transform, preserving referential integrity.
//   - Combine: merge two documents with a configurable collision
//     strategy (ErrorOnCollision, PrefixDoc2, RenumberDoc2), returning
//     a fresh document plus a report describing what was remapped and
//     which header fields conflicted.
//
// What this package does NOT do:
//
//   - Fuzzy matching of individuals, families, or events across
//     documents.
//   - Field-level merge policy (last-write-wins, conflict resolution,
//     evidence weighting).
//   - Deduplication beyond what an XRef collision check provides.
//
// All operations return a new document; inputs are never mutated.
package merge
