# Decisions

Architecture Decision Records — one accepted decision each, with its rationale, alternatives, and trade-offs.
A record is immutable once accepted; supersede it with a new record rather than editing history.

| ADR | Decision |
|-----|----------|
| [0001](0001-custom-date-struct.md) | Custom Date struct for lossless GEDCOM dates |
| [0002](0002-xref-resolution-strategy.md) | XRef resolution via strings + O(1) map lookup |
| [0003](0003-lossless-dual-storage.md) | Lossless dual storage (raw tags + typed entity) |
| [0004](0004-encoding-detection-cascade.md) | Encoding detection cascade (BOM → Header → UTF-8) |
| [0005](0005-version-detection-strategy.md) | Version detection (header-first with tag fallback) |
| [0006](0006-line-continuation-handling.md) | Line continuation handling (CONT/CONC) |
| [0007](0007-error-transparency.md) | Error transparency (line numbers, context, never panic) |
| [0008](0008-validator-architecture.md) | Validator architecture (pluggable, configurable) |
