# ADR-009: Bounded Duplicate Detection

**Status**: Accepted
**Date**: 2026-09-20
**Context**: Duplicate detection work bounds in the validator package

## Decision

Bound duplicate detection with a **per-surname-group cap** (`DuplicateConfig.MaxGroupSize`,
default 1000). A group larger than the cap is skipped whole, never truncated. The cap is
tri-state: zero means "use the default", negative means "unlimited". When any group is
skipped, the detector emits one aggregate `DUPLICATE_DETECTION_LIMITED` issue at
**Warning** severity so the caller learns the analysis was incomplete.

## Context

`DuplicateDetector.FindDuplicates` buckets individuals by normalized surname and then
compares every pair within a bucket. That inner comparison is O(k²) in the bucket size k,
with no bound on k, and `Validator.ValidateAll` calls it unconditionally.

[#529](https://github.com/cacack/gedcom-go/issues/529) cut the constant factor by
pre-normalizing names once per individual instead of once per comparison. It did not change
the shape of the curve, which is what [#530](https://github.com/cacack/gedcom-go/issues/530)
is about: the input controls the surnames, so the input controls the group sizes.

Measured on the corpus:

| Observation | Value |
|-------------|-------|
| Largest normalized-surname group in all of `testdata/` | 519 (`testdata/gedcom-5.5.1/longsword.ged` — 203,154 individuals, 55,801 groups, 48 MB) |
| Next largest group anywhere in the corpus | 70 |
| That fixture's detection run | ~1.1s over ~3.06M candidate pairs (~0.36 µs/pair) |

Measured on crafted adversarial input (#530), where every individual shares one surname and
given names are long and mutually near-miss — the expensive case, because a near-miss forces
the full Levenshtein DP and then fails the confidence threshold, so the cost is paid with no
early exit:

| n | Candidate pairs | Elapsed | Cost per pair |
|---|-----------------|---------|---------------|
| 2,000 | 1,999,000 | 2.76s | 1.38 µs |
| 4,000 | 7,998,000 | 11.06s | 1.38 µs |
| 8,000 | 31,996,000 | 44.52s | 1.39 µs |

Per-pair cost is flat across the three sizes, which confirms the curve is cleanly quadratic
within a group — so the pair count is the whole story. Note the two per-pair figures differ
by ~4x: the corpus is cheaper per pair because its given names are short and mostly
dissimilar, while the adversarial shape is chosen to maximise DP work. The attacker picks
which of those they hand you. Crafted input exploits this directly: ~3 MB buys ~29 minutes
of single-core CPU, and ~6 MB buys ~2 hours.

The question: how do we put a ceiling on that work without making results depend on
something the caller cannot see?

## Decision Drivers

1. **Deterministic results** - the same document must produce the same findings
2. **Explainable omissions** - a caller can say exactly what was not compared, and why
3. **Safe by default** - a partially-filled config must not opt back into unbounded work
4. **One rule to document** - a second, overlapping limit is cost without capability

## Considered Options

### Option A: No Bound (Status Quo)

- **Pros**: nothing to configure, nothing to explain
- **Cons**: a ~3 MB file is a CPU exhaustion vector in any caller that validates untrusted uploads
- **Verdict**: Rejected - this is the defect

### Option B: Global Pair Budget (`MaxComparisons`)

Cap total pair comparisons across the whole document.

- **Pros**: caps total work directly, and is arguably the quantity a caller most wants to reason about
- **Cons**: to stay deterministic it must fix an iteration order over the surname groups and stop mid-sweep, so *which* groups got compared depends on that ordering. Go map iteration is randomized, so the order would have to be invented and maintained purely to make the budget reproducible — and results end up partial in a way that is hard to explain
- **Verdict**: Rejected - deterministic only by accident of ordering, and not explainable

### Option C: Truncate Oversized Groups to the First N

- **Pros**: every group still contributes some findings
- **Cons**: "the first N" is record-order dependent, so the same document reordered gives different results; worse, the output looks complete while silently being partial
- **Verdict**: Rejected - non-deterministic and quietly misleading

### Option D: Per-Group Cap, Oversized Groups Skipped Whole (Selected)

- **Pros**: order-independent (a group's size does not depend on iteration order), and crisply explainable — "the 'smith' bucket had 20,000 members and was skipped". Gives a derived bound of **O(n · MaxGroupSize / 2)** pair comparisons, i.e. linear in document size
- **Verdict**: Accepted

## Consequences

### Positive

- Worst-case work is linear in document size, with a constant the caller chooses
- Results are reproducible regardless of record order or map iteration order
- Omissions are reported, with the surnames and counts involved

### Negative

- A legitimately skewed real dataset — a tree where one very common surname dominates, or a
  culture with a small surname pool — can exceed the cap and get reduced detection. This is
  an **accepted trade-off**, not a defect: it is why the cap is configurable (raise it, or
  set it negative for the old unbounded behaviour) and why the notice is a Warning rather
  than silence.
- Duplicate detection now has a knob whose semantics must be read, because they are not the
  conventional ones (below).

## Implementation

### Configuration

```go
// MaxGroupSize caps the number of individuals in one normalized-surname group
// that will be compared pairwise. Groups larger than this are skipped whole.
//   0        → DefaultMaxGroupSize (NOT unlimited)
//   negative → unlimited
//   positive → that value
// Default: DefaultMaxGroupSize (1000)
MaxGroupSize int
```

The tri-state rule lives in exactly one place, an unexported `maxGroupSize()` resolver on
the detector, so no call site special-cases it.

### Why Zero Means the Default, Not Unlimited

This breaks the Go zero-value convention, and it is inconsistent with
`ValidatorConfig.MaxErrors` in the same package, where 0 does mean unlimited. The
inconsistency is deliberate and is recorded here rather than hidden.

`ValidatorConfig.Duplicates` is a `*DuplicateConfig`, and callers routinely pass a partial
literal:

```go
&validator.ValidatorConfig{
    Duplicates: &validator.DuplicateConfig{MinConfidence: 0.7},
}
```

Under the conventional reading, `MaxGroupSize` is 0 there and that caller silently reverts
to the unbounded path — reopening exactly the hole this ADR closes, for the callers most
likely to be tuning duplicate detection. The rejected alternative (0 = unlimited, matching
`MaxErrors`) buys convention consistency and pays for it with a silent loss of the bound.
Safety wins; the cost is one documented rule, and "unlimited" stays reachable by asking for
it explicitly with a negative value.

### Why the Limit Notice Is a Warning

`POTENTIAL_DUPLICATE` is `SeverityInfo`, so duplicate findings only surface at
`StrictnessStrict`. The rejected alternative was to match it and file the limit notice as
Info too. But a notice that the analysis was *incomplete* is different in kind from a
finding: a caller running at the default `StrictnessNormal` is not reading duplicate
findings, yet must still learn that the detector gave up on part of the document. As #530
puts it, silently returning fewer duplicates than exist is its own defect. The notice is
therefore `SeverityWarning`, and one aggregate issue rather than one per group, so an
adversary cannot turn it into its own output amplifier.

### Why 1000

The largest normalized-surname group anywhere in `testdata/` is 519, in a 48 MB, 203,154-
individual real-world file; the next largest is 70. A default of 1000 gives ~2x headroom
over the worst real observation, so the cap does not fire on realistic genealogical data
while still holding the adversarial case to a linear bound.

### What Was Left Out

A document-level `MaxIndividuals` gate was considered and deliberately omitted. That gate is
already expressible — `ValidatorConfig.SkipDuplicateDetection` turns the whole pass off, and
a caller who wants a size threshold can check `len(doc.Individuals())` itself. A second cap
would be a second rule to document, test, and reconcile with the first, for no capability
the caller does not already have.

## References

- [#530](https://github.com/cacack/gedcom-go/issues/530) - unbounded duplicate detection
- [#529](https://github.com/cacack/gedcom-go/issues/529) - the prior constant-factor fix
- `validator/duplicates.go` - `MaxGroupSize`, `maxGroupSize()`, `FindDuplicatesReport`
- `validator/issue.go` - `CodeDuplicateDetectionLimited`
- `validator/validator.go` - `SkipDuplicateDetection`, limit issues in `ValidateAll`
- [ADR-008](0008-validator-architecture.md) - validator architecture, strictness levels
