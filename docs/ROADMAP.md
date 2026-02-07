# Roadmap

The phased feature plan for gedcom-go. For the project vision and guiding principles, see [ETHOS.md](./ETHOS.md).

---

## Phase Guide

Features are organized into phases to focus effort. **Phase 1 is the current priority** — resist the temptation to jump ahead. Features are driven by real downstream usage ([my-family](https://github.com/cacack/my-family)), not speculation.

| Phase | Focus | Principle |
|-------|-------|-----------|
| **Phase 1 (Now)** | API polish & stability | Nail the Basics First |
| **Phase 2 (Near-term)** | Document manipulation | Dogfood Relentlessly |
| **Phase 3 (Future)** | Advanced features & formats | Start Small, Ship Often |

---

## Phase 1 — API Polish & Stability

Get the existing API right before adding new capabilities.

| Issue | Description | Effort | Value |
|-------|-------------|--------|-------|
| [#156](https://github.com/cacack/gedcom-go/issues/156) | Add Options types for Encode and Validate | Medium | Medium |
| [#135](https://github.com/cacack/gedcom-go/issues/135) | Add place parsing helpers | Low | Low |
| [#141](https://github.com/cacack/gedcom-go/issues/141) | Add vendor test data (Legacy Family Tree) | Low | — |
| [#44](https://github.com/cacack/gedcom-go/issues/44) | CLI tool for GEDCOM validation and info | Medium | Medium |

**Exit criteria**: All Phase 1 issues closed, API stability documented, downstream consumer stable.

---

## Phase 2 — Document Manipulation

Enable programmatic modification and comparison of GEDCOM data.

| Issue | Description | Effort | Value |
|-------|-------------|--------|-------|
| [#132](https://github.com/cacack/gedcom-go/issues/132) | Merge utilities and XRef remapping | High | Medium |
| [#36](https://github.com/cacack/gedcom-go/issues/36) | Merge and diff capabilities | High | Low |
| [#37](https://github.com/cacack/gedcom-go/issues/37) | Export to JSON and XML formats | Medium | Low |

**Exit criteria**: Can merge two GEDCOM files, diff documents, export to JSON/XML.

---

## Phase 3 — Advanced Features

Extend the library for specialized use cases.

| Feature | Description | Source |
|---------|-------------|--------|
| GEDZip support | [#127](https://github.com/cacack/gedcom-go/issues/127) — GEDCOM 7.0 archive format | Issue |
| Fluent builder API | Method-chaining document construction | IDEAS.md |
| JSON struct tags | `json` tags on types for easy serialization | IDEAS.md |
| BOM output option | UTF-8 BOM for Windows compatibility | IDEAS.md |

**Exit criteria**: GEDZip implemented, builder API usable for downstream consumers.

---

## Related

- [Project Ethos](./ETHOS.md) — Vision, principles, and differentiators
- [GitHub Issues](https://github.com/cacack/gedcom-go/issues) — Single source of truth for planned work
- [IDEAS.md](../IDEAS.md) — Unvetted ideas and rough concepts
- [CONTRIBUTING.md](../CONTRIBUTING.md) — How to contribute
