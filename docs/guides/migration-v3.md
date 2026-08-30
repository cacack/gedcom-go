# Migrating from v2 to v3

v3.0.0 removes API that v2 had already marked superseded, and fixes three
option fields whose zero value did the opposite of what their documentation
promised. Most migrations are a deletion.

This guide is written as v3 is assembled, so it grows with each breaking change
that lands. Every entry gives the old form, the new form, and — where the
compiler cannot tell you — the value mapping.

> **Read the value mappings, not just the names.** Three changes on this page
> alter what a value *means* — a boolean inverts, a constant is renumbered, and
> an empty string stops meaning "empty". For the first two the mechanical fix
> (rename the identifier, keep the value) produces working code with the
> opposite behaviour and no error.

## Module path

v3 changes the import path, as Go's Semantic Import Versioning requires:

```go
import "github.com/cacack/gedcom-go/v3/decoder"   // was .../v2/decoder
```

`go get github.com/cacack/gedcom-go/v2@latest` will never resolve to v3 — the
`/v2` and `/v3` paths are different modules to the toolchain. Update the import
path, then `go get github.com/cacack/gedcom-go/v3@latest`.

## Changes that invert or remap a value

These are the ones to read carefully. The compiler flags the identifier; it
says nothing about the value.

### `encoder.EncodeOptions.PreserveUnknownTags` → `DropUnknownTags`

The field's polarity is inverted so that the zero value preserves data.

| v2 | v3 |
|----|----|
| `PreserveUnknownTags: true` | **omit the field** — it is now the default |
| `PreserveUnknownTags: false` | `DropUnknownTags: true` |

```go
// v2
opts := &encoder.EncodeOptions{LineEnding: "\n", PreserveUnknownTags: true}

// v3 — the term is gone, not renamed
opts := &encoder.EncodeOptions{LineEnding: "\n"}
```

Renaming the identifier while keeping `true` silently strips every custom tag
and every custom-tag-typed record from your output. There is no error and no
diagnostic; the file just comes back smaller.

`converter.ConvertOptions.PreserveUnknownTags` is a **different field** and is
unchanged in v3 — it keeps its original polarity, so a bare
`&converter.ConvertOptions{}` still drops unknown tags. Set it explicitly, or
use `converter.DefaultOptions()`.

### `encoder.EncodeOptions.LineEnding` defaults when empty

| v2 | v3 |
|----|----|
| `LineEnding: ""` wrote every line with no separator | `LineEnding: ""` writes `"\n"` |

In v2 an empty `LineEnding` produced a single unparseable line — the whole
document concatenated — so no correct program relied on it. If you set it
explicitly to `"\r\n"` or `"\n"`, nothing changes.

Together with `DropUnknownTags` above and `MaxLineLength` (which already
defaulted to 248), this makes a bare `&encoder.EncodeOptions{}` lossless in
full: every field's zero value is now the safe one.


### `validator.Strictness` renumbered

`StrictnessNormal` is now the zero value, so an options struct that never
mentions `Strictness` reports errors *and* warnings — which is what the
documentation always promised.

| Constant | v2 value | v3 value |
|----------|----------|----------|
| `StrictnessNormal` | 1 | **0** |
| `StrictnessRelaxed` | 0 | **1** |
| `StrictnessStrict` | 2 | 2 |

Code that names the constants needs no change. Code that **persisted the
integer** — a config file, a database column, a JSON payload — must remap it,
because a stored `0` meant Relaxed under v2 and means Normal under v3. There is
no compiler signal for this.

## Straight removals

Each of these is superseded by something that already exists in v2, so you can
migrate before upgrading.

| Removed | Replacement |
|---------|-------------|
| `gedcom.Source.RepositoryRef` | `Source.RepositoryLink.XRef` |
| `gedcom.Source.Repository` | `Source.RepositoryLink.Inline` |
| `decoder.DecodeOptions.MaxNestingDepth` | none needed — the field was never read. The ceiling is fixed at `parser.MaxNestingDepth-1` (99) by the grammar's two-digit level field |
| `gedcom/testing.WithHeaderTagComparison()` | none needed — delete the argument. Header tags have been compared unconditionally since v2 |

```go
// v2
src.RepositoryRef = "@R1@"
src.Repository = &gedcom.InlineRepository{Name: "State Archives"}

// v3
src.RepositoryLink = &gedcom.SourceRepositoryLink{XRef: "@R1@"}
src.RepositoryLink = &gedcom.SourceRepositoryLink{
    Inline: &gedcom.InlineRepository{Name: "State Archives"},
}
```

`SourceRepositoryLink` also carries the call numbers, media type, and per-link
notes that the flat fields could not represent, so it is a strict superset.

One behaviour improves silently: `Visit` and `Document.Subset` no longer write
to the `Source` they traverse. In v2 they re-synced `RepositoryRef` from
`RepositoryLink.XRef` on every walk, which blanked a caller-set value on a
document with an inline repository.

## Checking your upgrade

`make api-check` in this repository reports the full apidiff between the last
release and `main`, including constant value changes. For your own code, the
compiler catches every removal and rename on this page; the three value
changes above are the ones it cannot.

See [`docs/governance/policies/api-stability.md`](../governance/policies/api-stability.md)
for what the project treats as a breaking change, including the semantic breaks
that carry no signature change at all.
