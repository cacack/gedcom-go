# Project Ethos

The guiding philosophy and strategic vision for gedcom-go. For the phased feature plan, see [ROADMAP.md](./ROADMAP.md).

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

### 4. Zero Dependencies
Standard library only. No dependency trees to audit, no version conflicts for consumers.

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

## Anti-Patterns to Avoid

- **Feature bloat** — Do fewer things well; a library is not an application
- **Speculative features** — Build what downstream consumers need, not what might be cool
- **Vendor lock-in** — Data must always be exportable; never add proprietary requirements
- **Breaking changes** — Follow API stability guarantees; prefer additive changes
- **Ignoring real-world files** — Support what vendors actually produce, not just the spec
- **Panicking in library code** — Return errors; let callers decide how to handle them

---

## Inspirations

- **GEDCOM Specification** — The standard itself (5.5, 5.5.1, 7.0)
- **Go Standard Library** — API design patterns, documentation style, zero-dependency philosophy
- **Evidence Explained** — Citation standards and genealogical methodology
- **encoding/json, encoding/xml** — Go's approach to serialization libraries

---

## Related

- [Roadmap](./ROADMAP.md) — Phased feature plan
- [Architecture Decision Records](./adr/) — Key design decisions
- [CONTRIBUTING.md](../CONTRIBUTING.md) — How to contribute
- [API Stability](./API_STABILITY.md) — Compatibility guarantees
