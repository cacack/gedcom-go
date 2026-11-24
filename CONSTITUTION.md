# Constitution

The guiding philosophy and strategic vision for gedcom-go. This document covers the *why*: principles and foundational guidance only. What is planned, in what order and by when lives in [GitHub Issues and Milestones](https://github.com/cacack/gedcom-go/milestones), the single source of truth for execution.

---

## Vision Statement

The reference Go library for GEDCOM processing — **correct**, **complete**, and **pleasant to use**.

---

## Core Differentiators

### 1. Lossless Representation
Preserve every detail from the original file: partial dates, calendar-specific formats, vendor extensions, unknown tags. Data in equals data out.

### 2. Multi-Version Support
Auto-detect and handle GEDCOM 5.5, 5.5.1, and 7.0 transparently. Roundtrip fidelity across versions.

### 3. Real-World Compatibility
Handle files from Ancestry, FamilySearch, RootsMagic, Gramps, and other vendors — including their quirks and extensions.

### 4. Minimal Dependencies
One dependency, `golang.org/x/text`, and no third-party ones: almost nothing to audit, no version conflicts for consumers. [FEATURES.md](FEATURES.md#api-design) carries the authoritative statement.

### 5. Streaming & Performance
Process multi-million record files with constant memory. Indexed random access without loading everything.

---

## Core Principles

These six principles guide all development decisions:

1. **Library-First Design**: Every feature as a well-defined, independently testable library component
2. **API Clarity**: Public APIs prioritize simplicity with comprehensive godoc, io.Reader/Writer interfaces
3. **Test Coverage (NON-NEGOTIABLE)**: Minimum 85% coverage, TDD approach, table-driven tests
4. **Version Support**: Auto-detect and support GEDCOM 5.5, 5.5.1, and 7.0 with roundtrip fidelity
5. **Error Transparency**: All errors include line numbers, context, and never panic
6. **Lossless Representation (NON-NEGOTIABLE)**: Preserve original values, partial data, calendar-specific dates

---

## Strategic Principles

### 1. Nail the Basics First
Parsing, encoding, and validation must be rock-solid before adding convenience features. No fancy APIs matter if the core is broken.

### 2. Start Small, Ship Often
One polished feature beats ten half-done ones. Each enhancement should be a complete, testable unit.

### 3. Document as You Go
Good documentation is a feature. Godoc, examples, ADRs — lack of docs kills adoption.

### 4. Respect the Data
Genealogy data is irreplaceable. Never lose it, never corrupt it, never silently drop information.

### 5. Honor the Standards
GEDCOM is a real specification. Support it correctly, including the parts that are awkward or annoying.

---

## Evidence Over Speculation

Features are driven by evidence, not speculation. **Resist jumping ahead.**

### What counts as evidence

Three signals, any of which can justify work. They are listed in the order that has proven most reliable, not in order of importance:

1. **The test corpus.** `testdata/` holds real exports from 20+ vendors. A construct that appears in real files and that the library mishandles is a defect against Differentiator 3, whether or not anyone has complained.
2. **Spec coverage.** A tag defined by 5.5, 5.5.1 or 7.0 that the library does not handle is a gap against Strategic Principle 5. A library calling itself *the reference* must be able to state its own coverage.
3. **Downstream need.** A concrete request from a consumer, [my-family](https://github.com/cacack/my-family) or otherwise.

Downstream need was previously the *only* signal, and that was a mistake: because `Record.Tags` always provides a lossless fallback, a consumer that needs an unmodelled structure simply reads the raw tags and never files a request. The gap then registers as zero demand forever. An escape hatch that works is not evidence of a need being met.

This does **not** relax the bar against speculation. "Someone might want this" is still not a reason. The change is what may serve as proof — measured corpus behaviour and spec coverage now count, alongside asking.

---

## Anti-Patterns to Avoid

- **Feature bloat** — Do fewer things well; a library is not an application (the command-line tool is a separate project, [gedcom-cli](https://github.com/cacack/gedcom-cli))
- **Speculative features** — Build what the evidence supports (see [What counts as evidence](#what-counts-as-evidence)), not what might be cool
- **Mistaking a working fallback for a met need** — `Record.Tags` means no gap ever blocks a consumer, so silence is not a signal
- **Vendor lock-in** — Data must always be exportable; never add proprietary requirements
- **Breaking changes** — Follow API stability guarantees; prefer additive changes
- **Ignoring real-world files** — Support what vendors actually produce, not just the spec
- **Panicking in library code** — Return errors; let callers decide how to handle them

---

## Inspirations

- **GEDCOM Specification** — The standard itself (5.5, 5.5.1, 7.0)
- **Go Standard Library** — API design patterns, documentation style, minimal-dependency philosophy
- **Evidence Explained** — Citation standards and genealogical methodology
- **encoding/json, encoding/xml** — Go's approach to serialization libraries

---

## Related

- [GitHub Milestones](https://github.com/cacack/gedcom-go/milestones) — Planned work and exit criteria
- [Architecture Decision Records](docs/decisions/) — Key design decisions
- [CONTRIBUTING.md](CONTRIBUTING.md) — How to contribute
- [API Stability](docs/governance/policies/api-stability.md) — Compatibility guarantees
