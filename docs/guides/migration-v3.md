# Migrating from v2 to v3

v3.0.0 removes API that v2 had already marked superseded, and fixes three
option fields whose zero value did the opposite of what their documentation
promised. Most migrations are a deletion.

This guide is written as v3 is assembled, so it grows with each breaking change
that lands. Every entry gives the old form, the new form, and — where the
compiler cannot tell you — the value mapping.

**Upgrade to `v2.5.0` before you start.** It is the last v2 minor, and it exists
to make this migration a staged one: it carries the replacements that postdate
`v2.4.0` and marks every symbol v3 removes as `// Deprecated:`, so your tooling
lists the call sites and each one can be converted, tested and shipped while
still on v2. What is left at the v3 upgrade is then the module path and the
changes that genuinely have no v2 form.

> **Read the value mappings, not just the names.** Several changes on this page
> alter what a value *means* — a boolean inverts, a constant is renumbered, an
> empty string stops meaning "empty", and an `int` becomes a `*int` so that zero
> stops meaning absent. Where the change is a rename, the mechanical fix (rename
> the identifier, keep the value) can produce working code with the opposite
> behaviour and no error.

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


### `converter.ConvertOptions.PreserveUnknownTags` split in two

The old field's name was wrong: it never preserved anything. Nothing is dropped
by the converter either way. It gated two unrelated things, which are now
separate fields that each say what they do.

| v2 | v3 |
|----|----|
| `PreserveUnknownTags: true` | `ReportPreservedTags: true` **and** `MapEXIDToVendorTags: true` |
| `PreserveUnknownTags: false` | omit both — `false` is each field's zero value |

- `ReportPreservedTags` — itemises each preserved vendor/unknown tag in the
  conversion report. Report-only; no output byte depends on it.
- `MapEXIDToVendorTags` — on a 7.0 downgrade, maps a FamilySearch ARK EXID to
  `_FSFTID` rather than recording it as data loss.

Note that this field shared a name with `encoder.EncodeOptions.PreserveUnknownTags`
while meaning something entirely different — the encoder's really did drop tags.
After v3 no identifier by that name remains anywhere in the module.

`nil` still means `DefaultOptions()`. Any non-nil pointer is taken exactly as
written, so a literal that omits a field gets that field's zero value — start
from `DefaultOptions()` when changing one setting:

```go
opts := converter.DefaultOptions()
opts.StrictDataLoss = true
```

This is deliberately *not* smart about a wholly zero `&ConvertOptions{}`. An
earlier draft substituted the defaults for it, which read as friendlier but made
"every option off" impossible to express — the literal for it is the zero
struct.

### `SourceCitation.Quality` is now `*int`

| v2 | v3 |
|----|----|
| `cite.Quality` (an `int`) | `cite.Quality` (a `*int`) — nil check, then dereference |
| `Quality: 3` | `q := 3; Quality: &q` |
| absent, or `Quality: 0` | `nil` means absent; `&zero` means a real `QUAY 0` |

`QUAY` is an enumeration whose `0` is a meaningful assertion — "unreliable
evidence or estimated data" — not an absence. As an `int` its Go zero value
collided with that, and the encoder resolved the ambiguity in favour of absent
by emitting the tag only when `Quality > 0`. So `1 QUAY 0` decoded fine and then
vanished on re-encode, and no caller could assert "unreliable" at all.

```go
// v2
if cite.Quality > 0 {
    fmt.Println("quality:", cite.Quality)
}

// v3
if cite.Quality != nil {
    fmt.Println("quality:", *cite.Quality)
}
```

Byte round-trips were never affected, because `Record.Tags` is authoritative on
encode. The loss showed on hand-built documents, the converter, and anything
that clears `Tags`.

> **The compile error has a wrong mechanical fix.** If the value comes from a
> function that returns `0` to mean "no quality recorded", taking its address
> turns that into a *positive* `QUAY 0` — "unreliable evidence or estimated
> data" — on every such citation. v2's `> 0` guard hid the conflation; v3 emits
> it. Keep the pointer nil in the unset branch:
>
> ```go
> // WRONG: 0 might mean "unset", and this now asserts "unreliable".
> q := mapQuality(x)
> cite.Quality = &q
>
> // RIGHT: only take the address when a rating was actually determined.
> if q, ok := mapQuality(x); ok {
>     cite.Quality = &q
> }
> ```
>
> Symmetrically on read, `cite.Quality` is nil for a citation with no `QUAY`,
> so a helper taking a bare `int` must become `*int` and handle nil rather than
> dereferencing.

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

### `MediaObject.NoteXRefs` and `.SharedNoteXRefs` no longer overlap

**Nothing announces this one.** Both fields keep their name and their
`[]string` type, so the compiler is silent and `make api-check` reports no
entry at all. Unlike the `Notes` removal below, which is a compile error
everywhere it bites, a clean build tells you nothing about whether this one
affects you — read the call sites.

| v2 | v3 |
|----|----|
| `NoteXRefs` held `NOTE` **and** `SNOTE` pointers | `NoteXRefs` holds `NOTE` pointers only |
| `SharedNoteXRefs` repeated the `SNOTE` pointers already in `NoteXRefs` | `SharedNoteXRefs` holds the `SNOTE` pointers, and is now the only field that does |

The decoder used to append every `SNOTE` pointer to *both* slices, so the two
overlapped and their concatenation listed every shared note twice. In v3 they
partition a media object's note pointers: disjoint, and together complete.
`MediaObject` is the only type with a `SharedNoteXRefs` field, so nothing else
is affected.

Two caller shapes need attention. Reading `NoteXRefs` alone:

```go
// v2 — this saw shared-note pointers too
for _, x := range obj.NoteXRefs { /* NOTE and SNOTE pointers */ }

// v3 — read both, or the SNOTE pointers are silently lost
for _, x := range obj.NoteXRefs       { use(doc.GetNote(x)) }
for _, x := range obj.SharedNoteXRefs { use(doc.GetSharedNote(x)) }
```

And concatenating the two:

```go
// v2 — the concatenation double-counted every shared note, so callers deduped
all := dedupe(append(slices.Clone(obj.NoteXRefs), obj.SharedNoteXRefs...))

// v3 — keep the concatenation, delete the dedupe
all := append(slices.Clone(obj.NoteXRefs), obj.SharedNoteXRefs...)
```

**Delete the dedupe rather than changing which slices you read.** The
duplication it removed no longer exists, so all it can discard now is a
*genuine* repeat — the same note pointed at twice by one media object.

`obj.AllNotes(doc)` returns the same notes as before, once each. The dedupe it
used to run internally existed only to undo the duplication, and is gone for
the same reason.

Encoder output is unchanged **for a decoded document re-encoded from its
`Record.Tags`**, which is the common case — the raw tags win when non-empty, so
the typed writers are not consulted at all and decode-then-encode still writes
the same bytes. Two qualifications apply on the typed path (a hand-built
document, or one whose `Tags` you cleared):

- The encoder writes `NoteXRefs` before `SharedNoteXRefs`, so a record whose
  source had `SNOTE` ahead of `NOTE` re-encodes with the two reordered.
- A `MediaObject` you carried across from v2 — persisted, serialized to JSON, or
  copied out of a v2 decode — holds each SNOTE pointer in *both* slices, because
  that is what the v2 decoder produced. Nothing detects this, and the dedupe that
  used to absorb it is gone, so it now yields a duplicate note. **Drop the
  duplicates once when you migrate such a value**; a freshly decoded document is
  already correct.

Only code that reads the two fields directly needs a change.

## Retypes

A field keeps its name and its meaning but changes its type. The build fails
wherever a call site names a field of the retyped value, so the compiler hands
you most of the list — apply the mapping below rather than the first fix that
compiles. It does not hand you the whole list; see the caveat in each entry.

### `Individual.SpouseInFamilies` is now `[]FamilyLink`

| v2 | v3 |
|----|----|
| `SpouseInFamilies []string` — family XRefs | `SpouseInFamilies []FamilyLink` — the XRef plus the link's own subordinates |

`ChildInFamilies` made this move in v2; the two halves now match. `FAMC` and
`FAMS` are one `FAM-LINK` structure in both 5.5.1 and 7.0, but only the child
half carried the `NoteXRefs`/`InlineNotes` fields, so a `NOTE` or `SNOTE` under
a `FAMS` line was dropped on the typed path. A spouse link's notes now survive a
decode and re-encode exactly as a child link's do.

Ranging over the slice needs the XRef taken off the link:

```go
// v2
for _, famXRef := range ind.SpouseInFamilies { use(doc.GetFamily(famXRef)) }

// v3
for _, link := range ind.SpouseInFamilies { use(doc.GetFamily(link.FamilyXRef)) }
```

**Every site that names a field is a compile error** — `link.FamilyXRef` has no
`[]string` form, so direct field access cannot survive the retype silently.

**A clean build is still not proof you found them all.** A site that passes the
slice somewhere as a whole — rather than naming a field of it — never stops
compiling, and its output changes underneath:

```go
// Compiles before and after. Was "[@F1@]", now "[{@F1@  [] []}]".
fmt.Printf("%v\n", ind.SpouseInFamilies)
```

The same silence covers any `any`/`interface{}` hand-off and a `text/template`
`{{range .SpouseInFamilies}}` whose body prints the element. `Individual`
carries no struct tags, so anything serialising it with `encoding/json` or
`encoding/gob` also changes wire shape with no build error:

```text
v2: ["@F1@"]
v3: [{"FamilyXRef":"@F1@","Pedigree":"","NoteXRefs":null,"InlineNotes":null}]
```

`len(ind.SpouseInFamilies)` is the reassuring case: still one entry per `FAMS`
line, so a count does not change.

This repository had to hand-fix its own `examples/query` for exactly this — the
compiler flagged nothing. **Grep for `SpouseInFamilies` as well as building.**

`FamiliesAsSpouse(doc)` is unaffected by the retype — it still takes a
`*Document` and returns `[]*Family`. The method is renamed, though; see below.

`FamilyLink.Pedigree` is meaningless on a spouse link: neither 5.5.1 nor 7.0
defines `PEDI` under `FAMS`. It is present only because both link kinds share
one struct, and the encoder still writes it when set, so that malformed input
round-trips. Do not read it off a spouse link. Decoding one now also records an
`INVALID_VALUE` diagnostic, so the anomaly is reportable rather than merely
preserved.

**Expect more decode diagnostics on some files.** In v2 nothing subordinate to a
`FAMS` line was ever visited, so nothing under it could be reported. v3 parses
those subordinates, and any tag there that is not `PEDI`, `NOTE`, `SNOTE`,
`STAT`, or a `_`-prefixed vendor extension now yields an `UNKNOWN_TAG` warning —
once per occurrence. A vendor export carrying, say, `2 SOUR` under every `1 FAMS`
can gain a large number of warnings on input that decoded quietly before. No
data is lost either way: the raw tags were preserved in v2 and still are. If you
surface diagnostics to end users, check the volume before shipping the upgrade.

## Renames

### `Individual.ParentalFamilies` / `SpouseFamilies`

| v2 | v3 |
|----|----|
| `ind.ParentalFamilies(doc)` | `ind.FamiliesAsChild(doc)` |
| `ind.SpouseFamilies(doc)` | `ind.FamiliesAsSpouse(doc)` |

The old pair used opposite conventions for the same kind of relation.
`SpouseFamilies` meant "families where I am a spouse", but `ParentalFamilies`
meant "families where I am a **child**" — under the first name's own rule it
reads as the opposite, and "families where I am a parent" is exactly what
`SpouseFamilies` returned.

Both take a `*Document` and return `[]*Family`, so transposing them produced a
wrong-but-plausible tree with no type error and no panic. **If your code had
them swapped, the compiler will not tell you — but your results change.** Check
each call site against the role you meant, rather than mapping the names
mechanically.

### `validator.Issue.Details["line_number"]` removed

| v2 | v3 |
|----|----|
| `strconv.Atoi(issue.Details["line_number"])` | `issue.LineNumber` |

`Issue.LineNumber` is a real field now. It is populated by the checks that walk
raw tags — the custom-tag checks, the control-character checks, and the
XRef-length check — and is `0` elsewhere.

`0` is common and does not mean line 1. Checks that compare whole entities
(date logic, duplicate detection) or the document as a whole (a missing header
`SUBM`) have no single line to point at. Broader population is follow-up work,
not something this release completed.

The compiler cannot help here: a map lookup on a removed key returns the zero
value, so the old code keeps compiling and silently reads `""`. Nor can a
`// Deprecated:` marker, since a map key cannot carry one — the `Details` godoc
says so instead.

`Issue.LineNumber` ships in `v2.5.0`, where both it and the legacy key exist and
agree, so migrate these reads there rather than at upgrade time.

`Details["position"]` is **not** removed. It is a byte offset within a field's
value, not a source line, and `CodeBannedControlCharacter` now carries both:
`LineNumber` for the line, `Details["position"]` for the offset within it.

### `StreamingValidator` reports fewer `ORPHANED_NOTE` issues

No code change is needed for this one. It is listed because it changes the
issues a document produces, which a caller may have snapshotted or suppressed.

`StreamingValidator` collected note references from the deprecated `Notes`
slice. That slice interleaved shared-note pointers with inline note *text*, and
collection applied no pointer test — so every non-empty entry became a
reference to look up. A record carrying ordinary note prose was therefore
reported as an orphaned reference to a record whose XRef was the prose itself:

```
0 @I1@ INDI
1 NOTE Born at home, per the family bible.
```

In v2 that produces an `ORPHANED_NOTE` issue. In v3 it produces none, because
collection reads `NoteXRefs`, which the decoder fills only with pointer-shaped
values. Genuine dangling pointers are still reported exactly as before.

Two consequences:

- A caller suppressing `ORPHANED_NOTE` wholesale to work around the noise can
  stop, and will start seeing the real ones.
- A test asserting an exact issue count on a document with inline notes will
  need its expected value lowered.

`Issue.Details["field"]` also now reads `NoteXRefs[N]` rather than `Notes[N]`
for these issues. Code matching that string on a `"Notes["` prefix should match
`"NoteXRefs["` instead.

## Straight removals

Each of these is superseded by something that already exists in v2, so you can
migrate before upgrading. **Be on `v2.5.0` first.** The place accessors and the
encoder fix that goes with them postdate `v2.4.0` and were released in `v2.5.0`
specifically so these migrations can be staged; against `v2.4.0` or earlier the
replacement does not exist, and the compile break and the behaviour change arrive
together. `v2.5.0` also carries a `// Deprecated:` marker on almost every symbol listed
here, so your tooling will point at the call sites that still need attention.
Three note fields are the exception — `Event.Notes`, `Association.Notes` and
`SourceRepositoryLink.Notes`, whose replacement fields exist only in v3. See
below.

| Removed | Replacement |
|---------|-------------|
| `gedcom.Source.RepositoryRef` | `Source.RepositoryLink.XRef` |
| `gedcom.Source.Repository` | `Source.RepositoryLink.Inline` |
| `gedcom.Note.Continuation` | put the whole body in `Note.Text`, newlines included |
| `gedcom.Note.FullText()` | `Note.Text` |
| `decoder.DecodeOptions.MaxNestingDepth` | none needed — the field was never read. The ceiling is fixed at `parser.MaxNestingDepth-1` (99) by the grammar's two-digit level field |
| `gedcom/testing.WithHeaderTagComparison()` | none needed — delete the argument. Header tags have been compared unconditionally since v2 |
| `version.IsValidVersion(v)` | `v.IsValid()` — the same switch, as a method on `gedcom.Version` |
| `gedcom.Event.Tags` | `Record.Tags` — the single store for an event's raw tags |
| The deprecated `Notes []string` on 13 types | `NoteXRefs` + `InlineNotes`, or `AllNotes(doc)` where it exists — see below, the ordering changes |
| `gedcom.Event.Place` | `Event.PlaceName()` to read, `Event.SetPlaceName(name)` to write |
| `gedcom.Attribute.Place` | `Attribute.PlaceName()` to read, `Attribute.SetPlaceName(name)` to write |
| `validator.PlaceConsistencyValidator`, `validator.NewPlaceConsistencyValidator()`, `validator.CodePlaceCarrierMismatch` | none needed — the check compared two place carriers and there is now one. Never shipped in a tagged release |

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

```go
// v2 — two carriers for one fact, written apart
ev.Place = "Boston, MA"
ev.PlaceDetail = &gedcom.PlaceDetail{
    Name:        "Boston, MA",
    Coordinates: &gedcom.Coordinates{Latitude: "N42.3601", Longitude: "W71.0589"},
}

// v3 — one carrier
ev.PlaceDetail = &gedcom.PlaceDetail{
    Name:        "Boston, MA",
    Coordinates: &gedcom.Coordinates{Latitude: "N42.3601", Longitude: "W71.0589"},
}
fmt.Println(ev.PlaceName()) // nil-safe read on the receiver and on PlaceDetail
```

#### Migrating a write

**`PlaceDetail` is a pointer, and in v3 a nil one means "no place" — the encoder
writes no `PLAC` line at all.** That makes two otherwise-reasonable repairs of a
`Place` assignment silently wrong, so prefer `SetPlaceName`, which allocates
when needed and leaves any existing `Form` and `Coordinates` untouched:

```go
// v2
ev.Place = name

// v3 — allocates PlaceDetail if absent, nil-safe on the receiver
ev.SetPlaceName(name)
```

Both of these are traps:

```go
ev.PlaceDetail.Name = name   // panics: a freshly built Event has no PlaceDetail
// (deleting the `ev.Place = name` line)  // place silently vanishes on encode
```

The second is the one to watch for, because it compiles. Code that allocated
`PlaceDetail` only inside a conditional -- a coordinates branch, say -- keeps
compiling once the `Place` assignment is deleted, and every place that misses
that branch stops being written. The loss is invisible in memory, since reads
still resolve, and it produces no error and no diagnostic; only a diff of the
exported file shows it. If you assign `PlaceDetail` wholesale instead of calling
`SetPlaceName`, hoist the allocation out of any conditional.

`SetPlaceName` is a safe substitution for an optional value: an empty name on an
event that has no `PlaceDetail` yet does nothing, matching what `ev.Place = ""`
did in v2, so an existing `if place != ""` guard around the call is redundant
rather than load-bearing. Clearing a place that *is* already recorded is the one
thing the setter does not do -- assign `ev.PlaceDetail = nil` for that, since
calling it with an empty name blanks the name but keeps the carrier, and a
carrier with an empty `Name` still encodes as a valueless `PLAC` line.

In a composite literal, where no method call is possible, assign the structured
field directly and keep it out of any conditional:

```go
ev := &gedcom.Event{
    Type:        gedcom.EventBirth,
    PlaceDetail: &gedcom.PlaceDetail{Name: p.BirthPlace},
}
```

`PlaceName()` resolves to `PlaceDetail.Name`, so a read migrated ahead of the
upgrade needs no further change once you are on a release that carries the
accessor (see the note below). A write behaves differently than it did: the
encoder used to prefer the scalar, so code that set only `PlaceDetail` could
lose the place -- name and coordinates both, since `MAP` hangs off the `PLAC`
line. In v3 a non-nil `PlaceDetail` is the whole gate, and an empty `Name` still
writes a valueless `PLAC` line.

> **Minimum version for a staged migration: `v2.5.0`.** `PlaceName()`,
> `SetPlaceName()` and the encoder fix that emits `PLAC` from `PlaceDetail` alone
> all postdate `v2.4.0` and were released together in `v2.5.0`. Against `v2.4.0`
> or earlier the read accessor does not exist, and the compile break and the
> behaviour change arrive together.
>
> One difference to know about while you are still on v2: `SetPlaceName` there
> writes both carriers, because the v2 encoder prefers the scalar and would
> otherwise re-emit the name you just replaced. v3 has one carrier and writes only
> that. Either way the encoded output is the name you set, which is what lets the
> same call site work before and after the upgrade.

#### Known downstream call sites

`my-family` is the documented consumer of this library, and these are the sites
that will not compile against v3. They are listed here so the migration is not
rediscovered at `go get` time:

Counts below are as of `v2.4.0-37-g4369558`; re-grep before relying on them.

| Site | v2 | v3 |
|------|----|----|
| `internal/gedcom/importer.go` (9 read sites) | `event.Place` | `event.PlaceName()` |
| `internal/gedcom/exporter.go` (4 event write sites) | `ev.Place = name` | `ev.SetPlaceName(name)` |
| `internal/gedcom/exporter.go` (attribute write) | `ga.Place = attr.Place` | `ga.SetPlaceName(attr.Place)` |
| `internal/query/validation_service.go` (3 event literals) | `Place: p.BirthPlace` | `PlaceDetail: &gedcom.PlaceDetail{Name: p.BirthPlace}` |

The exporter's event write sites need the closest reading. Each one assigns
`Place` unconditionally but allocates `PlaceDetail` only inside a nested
coordinates branch, so deleting the `Place` line compiles and drops every place
that has no latitude and longitude. Replacing the line with `SetPlaceName` keeps
the coordinates branch working unchanged -- it allocates only when `PlaceDetail`
is still nil, and never clears `Coordinates`. The attribute write site has no
coordinates branch at all, so deleting its line drops the place unconditionally.
Note that the value there comes from a read model with a plain `Place` field and
no accessor, so it stays `attr.Place` on the right-hand side.

`validation_service.go` is the easiest one to get wrong. Its three sites are
**composite literals**, so `SetPlaceName` cannot be used inline and the
structured field has to be assigned directly. The events exist only to be handed
to `ValidateAll`, so silently dropping the field there does not corrupt an
exported file -- it makes the quality analyzer report missing places for records
that have them.

`LDSOrdinance.Place` is a different field with no `PlaceDetail` twin. It is
unchanged in v3, so leave those call sites alone.

### `Family.NumberOfChildren` is now a method

| v2 | v3 |
|----|----|
| `fam.NumberOfChildren` (read) | `fam.NumberOfChildren()` |
| `fam.NumberOfChildren = "4"` | `fam.SetNumberOfChildren("4")` |

The field was a second store for a fact `Family.Attributes` already held, and
the attribute is strictly richer — it carries the `NCHI` line's subordinates as
well as its value. The encoder preferred the attribute, so a write to the field
appeared to succeed, survived in memory, and vanished on encode.

`SetNumberOfChildren` updates an existing `NCHI` attribute in place (keeping its
subordinates) or appends one, so the write stays typed and you never need to
know the raw tag name.

**On a decoded family this changes the typed model only.** `Record.Tags` is
authoritative on encode (see the `Record.Tags` godoc), so to change what a
decoded family writes, edit the `NCHI` tag in `Record.Tags`, or clear `Tags` to
rebuild from the typed model.

### `version.DetectVersion` no longer returns an error

| v2 | v3 |
|----|----|
| `ver, err := version.DetectVersion(lines)` | `ver := version.DetectVersion(lines)` |

The error was always nil. Detection cannot fail: a file with no recognisable
version falls back to tag heuristics and then to `gedcom.Version55`, by design
(see [ADR 0005](../decisions/0005-version-detection-strategy.md)). Every caller wrote a
branch that could not be taken.

### `Event.Tags` in detail

`Event.Tags` was dead in both directions. The decoder never assigned it —
unrecognised event subtags go to `Record.Tags` — and the encoder built its
output purely from typed fields, so a tag placed there was cloned and walked but
never written.

There is no mechanical one-line replacement, because `Record.Tags` is not a
side channel the way `Event.Tags` looked like one. Two rules govern it, and the
naive `append` translation breaks both:

**`Record.Tags` is all-or-nothing.** The encoder writes `Tags` verbatim whenever
it is non-empty and derives tags from `Entity` only for a record that has none
(`encoder/encoder.go`, and the `Record.Tags` godoc). So appending a single tag
to a *hand-built* record — one whose `Tags` is empty because you built it from
the typed model — suppresses the whole derivation, and the record encodes to its
level-0 line plus your one tag. Everything else is silently gone.

**Tags are written in slice order, at the level each one stores.** Nothing
renumbers or re-nests them. On a *decoded* record, appending a `Level: 2` tag
puts it at the end of the record, where it nests under whichever level-1
structure happens to be last — not under the event you meant.

```go
// v2 — accepted, and silently dropped on encode
ev.Tags = append(ev.Tags, &gedcom.Tag{Level: 2, Tag: "_EVCUST", Value: "e"})

// v3, decoded record — insert at the event's position, not at the end.
// Find the event's level-1 line, then insert after its subordinates.
i := indexAfterEventSubordinates(rec.Tags, eventIdx) // your own helper
rec.Tags = slices.Insert(rec.Tags, i, &gedcom.Tag{Level: 2, Tag: "_EVCUST", Value: "e"})
```

On a hand-built record there is no typed-model-plus-one-raw-tag mode: either
build the whole record as `Tags`, or accept that the typed model is what gets
written. This is the same rule the `Family.NumberOfChildren` note above
describes, and it is worth reading once — `Record.Tags` reports no error and no
diagnostic when it drops something.

### `Note.Continuation` in detail

`Note` now matches `SharedNote`: one `Text` field holding the whole body.

```go
// v2 — a hand-built multi-line note
note := &gedcom.Note{
    Text:         lines[0],
    Continuation: lines[1:],
}
text := note.FullText()

// v3 — the encoder does the CONT/CONC split for you
note := &gedcom.Note{Text: strings.Join(lines, "\n")}
text := note.Text
```

The old field was broken in both directions. The decoder never populated it, so
reading it on a decoded document always gave `nil` and made every multi-line
note look single-line. And the encoder emitted it *in addition to* `Text`, so a
hand-built note that set both wrote its body twice.

Letting the encoder split `Text` is also safer than splitting it yourself: it
enforces `MaxLineLength` and stops an embedded newline from forging a new
GEDCOM record.

#### Known downstream call sites

`my-family` is the documented consumer of this library, and these are the sites
that will not compile against v3. They are listed here so the migration is not
rediscovered at `go get` time:

| Site | v2 | v3 |
|------|----|----|
| `internal/gedcom/exporter.go` (`toGedcomNoteRecord`) | `&gedcom.Note{Text: first, Continuation: rest}` | `&gedcom.Note{Text: n.Text}` — drop the hand split entirely |
| `internal/gedcom/importer.go` | `note.FullText()` | `note.Text` |
| `internal/gedcom/exporter.go` (citation export) | `citation.Quality = mapGPSToGedcomQuality(...)` | see the `Quality` warning above — the unset branch must leave the pointer nil |
| `internal/gedcom/importer.go` (citation import) | `mapGedcomQualityToGPS(srcCit.Quality)` | take `*int` and return the empty rating for nil |

The exporter's hand split is a net loss to keep: the encoder's split additionally
enforces `MaxLineLength` and guards against newline injection, neither of which
a manual `strings.Split` does.

One behaviour improves silently: `Visit` and `Document.Subset` no longer write
to the `Source` they traverse. In v2 they re-synced `RepositoryRef` from
`RepositoryLink.XRef` on every walk, which blanked a caller-set value on a
document with an inline repository.

### The deprecated `Notes` slices in detail

Thirteen types carried a `Notes []string` next to the `NoteXRefs` /
`InlineNotes` pair that superseded it: `Individual`, `Family`, `Source`,
`Repository`, `Submitter`, `MediaObject`, `Event`, `Attribute`,
`SourceCitation`, `LDSOrdinance`, `ChangeDate`, `Association` and
`SourceRepositoryLink`. All thirteen are gone.

```go
// v2 — one slice, and no way to tell a pointer from prose without re-testing
for _, n := range indi.Notes {
    if gedcom.IsPointerXRef(n) { /* shared note */ } else { /* inline text */ }
}

// v3 — the kinds are separate fields
for _, x := range indi.NoteXRefs { /* pointers to NOTE/SNOTE records */ }
for _, t := range indi.InlineNotes { /* note text, CONT/CONC already folded */ }

// v3 — or let the library resolve the pointers for you
for _, text := range indi.AllNotes(doc) { /* inline text + resolved shared notes */ }
```

`AllNotes(doc)` is defined on the six record types plus `Event` and
`Attribute`. The other five are substructures; read their two fields directly,
and resolve a pointer with `doc.GetNote(x)` or `doc.GetSharedNote(x)`.

**The order changes.** `Notes` interleaved pointers and text in original GEDCOM
order; the split fields group by kind, and the encoder writes every pointer
ahead of every inline note. A structure whose source file read `NOTE text` then
`NOTE @N1@` re-encodes from the typed model with the pointer first. Nothing is
lost, and a decoded record still re-encodes byte-for-byte from `Record.Tags`;
it is the typed-model path — a hand-built document, or one whose `Tags` you
cleared — where the `NOTE` lines come out grouped.

#### Which of the thirteen you can migrate before upgrading

Only nine of the thirteen `Notes` fields ever shipped in a v2 release, so those
are the only ones a v2 codebase can be reading. `Attribute`, `SourceCitation`,
`LDSOrdinance` and `ChangeDate` never carried one in v2 — their `Notes` existed
only inside the v3 line, alongside the split pair that replaced it.

Of the nine, six can be converted while still on `v2.5.0`, which carries both
the replacement pair and a `// Deprecated:` marker on the old field:
`Individual`, `Family`, `Source`, `Repository`, `Submitter` and `MediaObject`.

The other three — **`Event.Notes`, `Association.Notes` and
`SourceRepositoryLink.Notes`** — have no staged path. `v2.5.0` has the field
but not the `NoteXRefs` / `InlineNotes` pair that supersedes it, so the field
carries no deprecation marker there and your tooling will not flag it. For
these three the compile break and its replacement arrive together at the
upgrade.

`Event` at least gains `AllNotes(doc)` in v3; `Association` and
`SourceRepositoryLink` do not.

#### `Association.Notes` in particular

`Association` is the likeliest of the three to bite. It is a substructure, so
nothing about it changed in v2 and no call site has had reason to move; it has
no `AllNotes` to fall back on; and the field it loses is the only note accessor
it ever had.

```go
// v2
for _, n := range assoc.Notes { /* pointers and text, interleaved */ }

// v3
for _, x := range assoc.NoteXRefs   { /* pointers to NOTE/SNOTE records */ }
for _, t := range assoc.InlineNotes { /* note text */ }
```

`Association` and `SourceRepositoryLink` have no `AllNotes(doc)`. Both are
substructures, and the line the API draws is that record-level types plus
`Event` and `Attribute` carry the resolver while substructures do not — v3
does not add one to either. Read the two fields and resolve a pointer with
`doc.GetNote(x)` or `doc.GetSharedNote(x)`.

#### Known downstream call sites

`my-family` has already moved its `Source`, `Individual` and `Repository` reads
to `InlineNotes`, but `Association` was never split before v3, so its two sites
are untouched and will not compile:

| Site | v2 | v3 |
|------|----|----|
| `internal/gedcom/importer.go:1362` (read) | `assoc.Notes` | `assoc.NoteXRefs` and `assoc.InlineNotes` |
| `internal/gedcom/exporter.go:772` (write) | `Notes: notes` | `InlineNotes:` for text, `NoteXRefs:` for pointers |

Line numbers are as of the survey that produced this guide; re-grep before
relying on them.

## Checking your upgrade

`make api-check` in this repository reports the full apidiff between the last
release and `main`, including constant value changes. For your own code, the
compiler catches every removal and rename on this page, and every site that
*names a field* of a retyped value. It does **not** catch the value changes —
the inverted boolean, the renumbered constant, the `*int` retype (whose compile
error has a mechanical fix that can be wrong), or the `MediaObject`
note-pointer partition, which changes no signature at all and so produces no
build error anywhere. A clean build is not evidence that those four are done.

Nor does it catch a retyped slice that a call site only prints, marshals, or
measures: `SpouseInFamilies` changes shape for `%v`, `encoding/json` and
`text/template` without ever failing to compile. Grep for the field name
alongside building; the Retypes entry above lists the shapes to look for.

See [`docs/governance/policies/api-stability.md`](../governance/policies/api-stability.md)
for what the project treats as a breaking change, including the semantic breaks
that carry no signature change at all.
