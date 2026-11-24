# Editing and Merging Documents

How to change a decoded document record by record, see what changed between
two documents, combine files, merge duplicate records, and cut a branch out
and put it back.

Six tasks, each a section below:

| Task | Use |
|------|-----|
| [Edit one record as text](#editing-one-record-as-text) | `encoder.EncodeRecord`, `decoder.DecodeRecords`, `Document.ReplaceRecord` |
| [Add a record](#adding-a-record) | `Document.FreeXRef`, `Document.AddRecord`, `Record.AddMediaLink` |
| [See what changed](#seeing-what-changed) | `gedcom.Diff` |
| [Combine two files](#combining-two-files) | `merge.Combine` |
| [Merge duplicate records](#merging-duplicate-records) | `merge.Individuals`, `merge.Families` |
| [Cut a branch and zip it back](#cutting-a-branch-and-zipping-it-back) | `Document.BoundedSubset`, `merge.ZipSharedXRefs` |

Two rules hold throughout:

- **`Record.Tags` is what gets written and compared.** The encoder, `Diff`,
  the record merges and the zip all work on a record's raw tags. After editing
  typed fields, call `encoder.SyncTags(doc)` first; see
  [Modifying a decoded document](decoding.md#modifying-a-decoded-document).
- **The library does the bookkeeping, not the policy.** Nothing here decides
  that two records are the same person or which of two values is right. That
  stays with the caller.

`EncodeRecord`, `DecodeRecords`, `ReplaceRecord`, `RemoveRecord`, `Diff`,
`Individuals`, `Families`, `BoundedSubset` and `ZipSharedXRefs` are available
since v3.11.0, and `merge.ZipOptions` since v3.13.0.

## Editing One Record as Text

Use this to let a person, or another program, edit a record as GEDCOM text:
write the record out, take the edited text back, and put the result in the
record's place.

```go
var buf bytes.Buffer
if err := encoder.EncodeRecord(&buf, doc, doc.GetRecord("@I1@"), nil); err != nil {
    return err
}

edited := edit(buf.String()) // your editor, form or patch

records, diags, err := decoder.DecodeRecords(strings.NewReader(edited), doc, nil)
if err != nil {
    return err
}
if diags.HasErrors() || len(records) != 1 {
    return fmt.Errorf("edit rejected:\n%s", diags)
}
if err := doc.ReplaceRecord(records[0]); err != nil {
    return err
}
```

`EncodeRecord` writes the record's level-0 line and everything beneath it,
with no `HEAD` and no `TRLR`. The document argument supplies what a whole-file
encode would know, chiefly the GEDCOM version, so a decoded record passed with
its document is written as `Encode` writes it there under the same options.
The document may be nil; the version then comes from
`EncodeOptions.TargetVersion` alone. A nil record writes nothing.

`DecodeRecords` reads a fragment of level-0 records as if its lines were in
the document: under its version, identifier grammar and encoding. The document
is only read. Things to know:

- **The fragment may hold any number of records**, including none. Check the
  count when you expect one.
- **A `HEAD` or `TRLR` line, or a first line that is not level 0**, is an
  error wrapping `decoder.ErrNotRecords`.
- **Options mean what they mean for `DecodeWithDiagnostics`**: lenient by
  default, so a bad line is skipped and reported in the diagnostics with a nil
  error. Line numbers count from the fragment's first line. Set `StrictMode`
  to get an error instead. `PreserveLayout` is ignored.
- **The records carry no line numbers**, as records built in code do not.

`ReplaceRecord` puts the record where the one with the same XRef was, in
`Records` and in `XRefMap`. Nothing else changes:

- An XRef the document does not hold is an error wrapping
  `gedcom.ErrUnknownXRef`, and the document is unchanged. That is what happens
  when the edit renamed the record. Use `Document.AddRecord` for a new record;
  it wraps `gedcom.ErrDuplicateXRef` when the XRef is taken.
- Pointers are not checked. An edit that adds `1 FAMS @F9@` is accepted
  whether or not `@F9@` exists; run the validator afterwards.

`Document.RemoveRecord(xref)` removes a record and returns every pointer still
naming it, as `gedcom.Reference` values (the holding record's XRef and the tag
path, such as `FAM`, `CHIL`). The pointers are reported, not changed, so they
dangle until you edit the records that hold them.

With `PreserveLayout` on both sides, every record that was not replaced or
removed is written byte for byte as read. A record from `DecodeRecords` is
written as new lines. To keep the layout of the lines an edit did not touch,
edit a `Clone` of the old record instead of decoding text; see
[What Changes](preserving-layout.md#what-changes).

## Adding a Record

A new record needs an XRef nothing in the document uses.
`Document.FreeXRef(prefix)` returns the first free `@<prefix><n>@`:

```go
xref := doc.FreeXRef("O")
err := doc.AddRecord(&gedcom.Record{XRef: xref, Type: gedcom.RecordTypeMedia,
    Entity: &gedcom.MediaObject{XRef: xref,
        Files: []*gedcom.MediaFile{{FileRef: "photos/ada.jpg", Form: "image/jpeg"}}}})
if err != nil {
    return err
}
```

- **An XRef that only a pointer names is in use.** A file can hold
  `1 OBJE @O3@` with no `@O3@` record; a new record given that XRef would
  become the pointer's target. `FreeXRef` counts every XRef a record defines
  and every pointer in the records and the header, read under the version the
  header declares.
- **`FreeXRef` reserves nothing.** Two calls return the same XRef until a
  record takes it. To add several records, take one
  `doc.NewXRefAllocator()` and call `Next(prefix)` for each; it never returns
  an XRef twice. An allocator is a snapshot, so make a new one after changing
  the document some other way.
- **The prefix is not checked.** Use one the document's version allows; for
  GEDCOM 7.0 that is uppercase letters, digits and `_`.

To link the new media record from an individual, family or source record,
append to its record-level media links and bring the raw tags in step:

```go
person := doc.GetRecord("@I1@")
if err := person.AddMediaLink(&gedcom.MediaLink{MediaXRef: xref}); err != nil {
    return err
}
if err := encoder.SyncTags(doc); err != nil {
    return err
}
```

`Record.MediaLinks()` reads the same links, for any of the three record types,
and reports with its second result whether the record type can carry them at
all. Links deeper in a record, on an event or a source citation, are not
included.

| Error | When |
|-------|------|
| wraps `gedcom.ErrNoMediaLinks` | The record's type cannot carry record-level media links (a `NOTE` record, say), or the record has no `Entity` |
| `gedcom.ErrNilMediaLink` | The link is nil |

`AddMediaLink` does not look for an equal link already present: adding one
twice links it twice. Check `MediaLinks()` first when that matters.

## Seeing What Changed

`gedcom.Diff` compares two documents record by record. Use it to review an
edit before saving, or to compare two exports of one tree, where a line diff
is mostly noise.

```go
before := doc.Clone()
// ... edit doc ...

d := gedcom.Diff(before, doc, &gedcom.DiffOptions{
    IgnoreTags: []string{"CHAN"}, // skipped with everything under them
})
if d.IsEmpty() {
    return nil
}
for _, r := range d.Added {
    fmt.Println("added", r.XRef)
}
for _, r := range d.Removed {
    fmt.Println("removed", r.XRef)
}
for _, m := range d.Modified {
    for _, c := range m.Changes {
        for _, t := range c.Removed {
            fmt.Printf("%s %v - %d %s %s\n", m.XRef, c.Path, t.Level, t.Tag, t.Value)
        }
        for _, t := range c.Added {
            fmt.Printf("%s %v + %d %s %s\n", m.XRef, c.Path, t.Level, t.Tag, t.Value)
        }
    }
}
```

Records are matched by XRef and type. A record that was renumbered, or whose
type changed, is reported as removed and added; finding the same person under
two XRefs is duplicate detection, not diffing. Each modified record lists its
changes as hunks: the old lines, the new lines, and the tag path of their
parent (`["BIRT"]` for a changed `DATE` under a level-1 `BIRT`).

| Not a change | A change |
|--------------|----------|
| The order of records in the file | The order of siblings with the same tag (`NAME`, `CHIL`) |
| A substructure moved past siblings with a different tag | Case or whitespace in a value |
| A value split differently across `CONT`/`CONC` lines | Anything else |
| Line numbers | |

Caveats:

- **The header is left out** unless `DiffOptions.IncludeHeader` is set, since
  it changes on every export.
- **Typed edits are invisible until `encoder.SyncTags` has run**, and a record
  with no `Tags` at all (built in code) compares only its level-0 line. To
  compare what such a document would write, encode it and decode it again.
- **The result points into the two documents.** Its records and tags are not
  copies, so `Tag.LineNumber` in a removed line is the line in the old file.
- Neither document is modified, a nil document compares as an empty one, and
  nil options are the defaults. `Diff` takes no context.

[Record Diff](../../FEATURES.md#record-diff) in FEATURES.md has the full
rules.

## Combining Two Files

`merge.Combine` puts the records of two documents into one. Use it for two
files that describe different people, or to bring a relative's file in before
deduplicating. It returns a new document and never modifies its inputs.

Two unrelated files usually both have an `@I1@`. The collision strategy says
what to do about that:

| `CollisionStrategy` | A doc2 XRef that doc1 also holds |
|---------------------|----------------------------------|
| `ErrorOnCollision` (zero value) | `Combine` fails with `ErrXRefCollision` |
| `PrefixDoc2` | Gets `CombineOptions.Prefix` in front: `@I1@` becomes `@b_I1@` |
| `PrefixAllDoc2` | Gets the prefix too, and so does every other doc2 XRef |
| `RenumberDoc2` | Gets the next free number for its record type: `@I1@` becomes `@I6@` |
| `ZipSharedXRefs` | Is the same record; see [below](#cutting-a-branch-and-zipping-it-back) |

```go
combined, report, err := merge.Combine(doc1, doc2, merge.CombineOptions{
    CollisionStrategy: merge.RenumberDoc2,
    Context:           ctx,
})
if err != nil {
    return err
}
for old, renamed := range report.RemappedXRefs {
    fmt.Printf("%s -> %s\n", old, renamed)
}
for _, c := range report.HeaderConflicts {
    fmt.Printf("header %s: kept %q, dropped %q\n", c.Field, c.Doc1, c.Doc2)
}
```

What to expect:

- **Only colliding XRefs are renamed**, and every pointer to a renamed record
  is rewritten with it. The exception is `PrefixAllDoc2`, which renames every
  doc2 record with a pointer-shaped XRef so that an XRef shows which file its
  record came from. doc1's XRefs never change. `report.RemappedXRefs` maps
  each doc2 XRef that changed to its new one; keep it if anything outside the
  document refers to doc2's records.
- **A broken pointer in doc2 stays broken.** An XRef doc2 points at but does
  not define is renamed with doc2's records when doc1 defines it (always,
  under `PrefixAllDoc2`), so it cannot resolve to a doc1 record and the
  validator still reports it. `RemapXRefs` renames such XRefs too, so
  prefixing doc2 with it before an `ErrorOnCollision` combine is just as safe.
  `ErrorOnCollision` alone leaves them as they are.
- **Records are doc1's, then doc2's.** Nothing is deduplicated: a person in
  both files is in the result twice. Find candidates with the validator's
  `DuplicateDetector.FindDuplicatesBetween` and
  [merge them](#merging-duplicate-records) afterwards.
- **doc1's header wins.** A header field or structure that doc2 also has is
  dropped and listed in `report.HeaderConflicts`. Expect entries such as
  `Tags.GEDC` and `Tags.CHAR` for any two decoded files, even when the two
  lines are equal, because a header may declare each structure once. doc2's
  submitter record stays in `Records`; the header points at doc1's.
- **Versions and encodings must agree.** Two different non-empty values fail
  with `ErrIncompatibleHeader`. Convert one document first with the
  [converter](converter.md). `IgnoreEncodingMismatch` waives the encoding
  check, since decoded text is UTF-8 in memory either way.
- **Encode the result without `PreserveLayout`.** Its lines come from two
  files, so there is no one source layout to keep.

Errors, all for `errors.Is`:

| Error wraps | When |
|-------------|------|
| `merge.ErrNilDocument` | A document is nil |
| `merge.ErrInvalidOptions` | An unknown strategy, or `PrefixDoc2` or `PrefixAllDoc2` with no `Prefix` |
| `merge.ErrIncompatibleHeader` | Versions or encodings differ |
| `merge.ErrXRefCollision` | `ErrorOnCollision` and a shared XRef, or a rename that lands on an XRef already in use |
| `merge.ErrRecordConflict` | `ZipSharedXRefs` and a shared record that differs |
| `context.Canceled`, `context.DeadlineExceeded` | `CombineOptions.Context` is done |

On every error the document and the report are nil. `errors.As` with
`*merge.CombineError` gives the XRefs involved in `Colliding`.

To rename XRefs without combining, for example to move a document into its
own namespace beforehand, `merge.RemapXRefs(doc, transform)` rewrites every
XRef and every pointer through a function and returns the new document with
the mapping. [`examples/merge`](../../examples/merge) is a runnable combine,
and [Merge Primitives](../../FEATURES.md#merge-primitives) lists the header
rules in full.

## Merging Duplicate Records

`merge.Individuals` folds one individual into another in the same document;
`merge.Families` does the same for two families. You name the survivor
(`keep`) and the record to absorb (`drop`).

```go
merged, report, err := merge.Individuals(doc, "@I1@", "@I2@", merge.RecordMergeOptions{})
if err != nil {
    return err
}
fmt.Printf("%s absorbed %s: %d structures moved, %d already held, %d pointers repointed\n",
    report.Keep, report.Drop, len(report.Moved), len(report.Deduplicated), len(report.Repointed))
```

What happens:

- **Every level-1 structure of `drop` moves under `keep`**, after `keep`'s
  last structure with the same tag, unless `keep` already holds one that is
  identical line for line. By default that is the whole policy. A second `NAME` is added,
  and so is a differing `SEX` or a second `BIRT`: both claims are kept for you
  or the validator to resolve.
- **With `FoldEvents`, the same event is written once.** Set
  `RecordMergeOptions.FoldEvents` and a structure of `drop` with the tag, value
  and date of one of `keep`'s (dates compared as parsed, so `7 DEC 1977` is
  `07 Dec 1977`), the same `TYPE` if any, and the same place or a place only
  one of them has, is folded
  into it: `keep`'s lines stay, and `drop`'s citations, notes and other lines
  `keep`'s lacks are added. A structure without a valid `DATE`, or whose date,
  time, place or `TYPE` differs, is added as before. `report.Folded` lists each fold.
- **Every pointer to `drop` now names `keep`**, in families, associations and
  vendor tags alike. A level-1 line that would then repeat a sibling exactly,
  such as the second `CHIL` of a family that listed both individuals, is
  removed and listed in `report.Collapsed`.
- **`drop` is gone** from `Records` and `XRefMap`.
- **The input is not modified.** The result is a new document.

Merging two individuals often leaves two families for one couple. Merge those
with `merge.Families(doc, keepFam, dropFam, opts)`; a child both families list
is listed once.

Caveats:

- The merge works on raw tags and rebuilds the typed fields of the records it
  changed from them. Typed edits not yet synced are lost on those records, so
  call `encoder.SyncTags` first.
- A record built in code with no `Tags` cannot take part: if `keep`, `drop`
  or a record pointing at `drop` has none, the merge fails.
- Every failure wraps `merge.ErrInvalidMerge`: merging a record into itself,
  a missing record (which also wraps `gedcom.ErrUnknownXRef`), one of the
  wrong type, one defined twice, or one without `Tags`. A nil document wraps
  `merge.ErrNilDocument`. `RecordMergeOptions.Context` cancels as it does for
  `Combine`.

## Cutting a Branch and Zipping It Back

Use this to hand part of a tree to someone else, or to another tool, and take
their edits back without the rest of the tree leaving your hands.

`Document.Subset(xrefs)` is the simpler cut: the named records plus everything
they reference, transitively. Because individuals reference families and
families reference individuals, one seed usually pulls in the whole connected
tree. It suits `doc.Subset(doc.Descendants("@I1@"))`-style extracts that are
not coming back.

`Document.BoundedSubset(individuals)` stops at people. The result holds the
named individuals, every family any of them belongs to, and the sources,
notes, media and other records those reference. A pointer to a person or
family outside the set is removed, and each removal is reported as a
`gedcom.DroppedPointer`.

```go
branch, droppedFromBranch, err := master.BoundedSubset(branchPeople)
if err != nil {
    return err
}
rest, _, err := master.BoundedSubset(everyoneElse)
if err != nil {
    return err
}
// Write branch to a file and send it out. Keep droppedFromBranch (it
// marshals as JSON) and rest, or the master to cut rest from again.

// Later: edited is the branch file, decoded.
whole, report, err := merge.Combine(rest, edited, merge.CombineOptions{
    CollisionStrategy: merge.ZipSharedXRefs,
    Zip:               &merge.ZipOptions{DroppedFromDoc2: droppedFromBranch},
})
if errors.Is(err, merge.ErrRecordConflict) {
    var ce *merge.CombineError
    errors.As(err, &ce)
    for _, c := range ce.Conflicts {
        fmt.Printf("%s differs on both sides (%s)\n", c.XRef, c.Reason)
    }
}
if err != nil {
    return err
}
```

Zip the edited branch with **the rest**, not with the master. The master
still holds the branch's records as they were, and every edited one would
conflict.

With `ZipSharedXRefs` a shared XRef means the same record, and nothing is
renamed. For each record both documents hold:

| The two copies | Result |
|----------------|--------|
| Are identical | One copy |
| Differ only in pointer lines naming an individual or family | The union of both, which is how a boundary family gets back the `HUSB`, `WIFE` and `CHIL` lines each cut dropped |
| Differ in any other way | A conflict |

So an edit to a record only the branch holds (its people, and a family whose
members are all in it) comes back untouched. An edit to a record both sides
hold is a conflict: a boundary family's own events, a source or note cited
from both sides, the submitter. `Combine` then fails with `ErrRecordConflict`
and returns nothing. To take one side's version, set `KeepFirstOnConflict` and
pass that side as doc1; the conflicts are then listed in
`report.RecordConflicts` instead:

```go
whole, report, err := merge.Combine(edited, rest, merge.CombineOptions{
    CollisionStrategy:   merge.ZipSharedXRefs,
    KeepFirstOnConflict: true,
    Zip:                 &merge.ZipOptions{DroppedFromDoc2: droppedFromRest},
})
```

Caveats:

- **Pass the report of whichever subset is doc2.** A subset decoded from a
  file has that file's line numbers, and without the report a family split
  between the two can get its `CHIL` lines back in a different order. The
  report belongs to the subset as it was cut; one that no longer fits a record
  is ignored for that record. It affects the order of pointer lines only,
  never which lines the result holds.
- **Not every dropped pointer comes back.** `DroppedPointer.Restorable` is
  true for a family's `HUSB`, `WIFE` and `CHIL`. Another pointer across the
  boundary, such as an `ASSO` naming someone outside the set, is restored only
  if the other subset holds the record it sat in. Check the report at cut
  time, and keep both ends in one set to be sure.
- **Record order is not the master's.** The result holds doc1's records in
  order, then the records only doc2 has. `Diff` ignores record order, so
  `gedcom.Diff(master, whole, nil)` is a direct check that the zip changed
  only what was edited.
- **A record added in the branch needs an XRef the master does not use.** If
  it reuses one that the rest holds, it is taken for that record.
- **Seeds must be individuals** (`gedcom.ErrNotIndividual` otherwise) that
  exist (`gedcom.ErrUnknownXRef`). A dangling pointer in an included record is
  also `ErrUnknownXRef`, for `Subset` and `BoundedSubset` alike.
- **Every subset carries the header's submitter record**, with whatever
  address, phone and email the source file recorded. Clear or replace it
  before sharing a branch if that is not wanted.
- **Records need raw `Tags`** to be unioned, as every decoded record has. A
  document built in code without them does not round-trip.
- A subset has no layout of its own. Encode it, and the zipped result, without
  `PreserveLayout`.

More than two subsets zip as a chain: combine two, then combine the result
with the third as doc2, passing the third's report. The full rules are in
[Bounded Subset](../../FEATURES.md#bounded-subset) and
[Merge Primitives](../../FEATURES.md#merge-primitives).

## Related Documentation

- [Preserving Layout](preserving-layout.md): writing an edited file back so it
  differs only where it was edited
- [Decoding Contract](decoding.md): diagnostics, strict and lenient mode, and
  `encoder.SyncTags`
- [Converter](converter.md): bringing two documents to one GEDCOM version
  before combining them
- [Context Support](../../FEATURES.md#context-support): where each operation
  checks for cancellation
