# ADR-012: GEDZIP Archive Model

**Status**: Accepted
**Date**: 2026-11-13 ([#127](https://github.com/cacack/gedcom-go/issues/127))
**Context**: Applies the error and no-panic rules of [ADR-007](0007-error-transparency.md) to an archive format

## Decision

Support GEDZIP, the GEDCOM 7 packaging format, in a new package `gedzip`, on five rules:

- **The archive is separate from the document, and media is lazy.** A `Reader` decodes
  `gedcom.ged` into an ordinary `gedcom.Document` and opens one media entry per call
  (`Reader.Open`, `Reader.FS`). `gedcom.Document` gains no media field, and nothing is
  extracted to disk. `Write` takes the document and an `fs.FS` and streams each file.
- **References are resolved strictly, with no leniency tiers.** The dataset is the entry
  named exactly `gedcom.ged`. A local `FILE` or `FILE.TRAN` payload names the entry that is
  the payload percent-decoded, matched byte for byte. `ClassifyFileRef` decides local, web,
  `file:` URL or invalid before any lookup. There is no case folding, Unicode
  normalisation, backslash repair, suffix matching or fallback to another `.ged` entry.
- **An unusable entry is unreachable, not fatal.** An entry that is encrypted, not a
  regular file, compressed with anything but Store or Deflate, duplicated, or not named
  with a clean relative UTF-8 path is listed with its reason and can never be opened. It
  does not prevent reading `gedcom.ged` or other media. The exception is `gedcom.ged`
  itself: unusable or duplicated, it fails `NewReader`.
- **Reads are bounded by default.** `ReadOptions` caps entry count, dataset size,
  per-entry size, declared total and compression ratio; zero is the default and negative
  is unlimited, the `DecodeOptions.MaxDiagnostics` convention. Unlike a usability problem,
  **a limit breach fails `NewReader` for the whole archive**, whichever entry caused it.
- **The writer is strict and rewrites nothing.** `Write` refuses a document that would not
  be written as GEDCOM 7.x, a `file:` URL, an invalid payload, a reference to `gedcom.ged`
  and a local reference with no file in the media source, all before the first byte. It
  does not convert the document or alter a payload.

## Context

The specification (7.0.18, the GEDZIP chapter and the File Path datatype) fixes the
writer's side: the dataset entry's name, how a payload maps to an entry name, that `file:`
URLs must not be used, and that a local file named `gedcom.ged` must be renamed. Every
MUST is on the writer. What a reader does with a missing entry, an extra one, a duplicate,
an escaped separator or an oversized entry is unspecified, so those are this library's
decisions.

Issue #127 sketched an eager API, `MediaFiles map[string][]byte`. Its own security note
describes the problem with that: a small archive that inflates to gigabytes. A server
decoding uploads is a stated use of this library
([untrusted input](../guides/untrusted-input.md)).

Four other implementations were read in source as a cross-check: oxidged (Rust),
GedcomSharp and ArmidaleSoftware/gedcom7 (C#), and gedcomkit. What was taken from them:

- an archive object apart from the document, with lazy media (oxidged, gedcomkit);
- limits checked from the central directory at open, then again while inflating
  (gedcomkit, oxidged);
- classifying a reference before looking it up (ArmidaleSoftware/gedcom7);
- unresolved references as a first-class query (GedcomSharp), here `CheckRefs`;
- a lazy media source on write (GedcomSharp), here `fs.FS`;
- refusing reserved names on write, a deterministic entry order, and deflating text while
  storing compressed media (gedcomkit).

What was seen there and avoided: no default bound; buffering the whole archive;
percent-decoding entry names as well as references, so a literal `%20` in a name never
matches; case-insensitive and suffix matching that returns the wrong file; decoding before
separator handling, so `%2F` becomes a separator; `..` surviving into written entry names;
an exact `VERS == "7.0"` test that rejects 7.1; one bad unrelated entry making the dataset
unreadable; guessing among several `.ged` entries.

## Considered Options

### Option A: Eager media on the document (`MediaFiles map[string][]byte`)

- **Pros**: One call returns everything; the shape the issue proposed.
- **Cons**: Memory is the archive's whole inflated size, chosen by whoever made the file.
  Puts a field on `gedcom.Document` that only one container format fills.
- **Verdict**: Rejected.

### Option B: Extract to a directory

- **Pros**: Familiar; media ends up as files.
- **Cons**: Makes the library responsible for path traversal, symlinks and overwrite
  policy on the caller's disk. A caller who wants files can copy from `Reader.Open` to a
  path of its own choosing.
- **Verdict**: Rejected; out of scope.

### Option C: Lenient resolution tiers (exact, then case-insensitive, then by file name)

- **Pros**: Opens archives whose names differ from their references in case or directory.
- **Cons**: No corpus file needs it, and each tier was a wrong-file or bypass bug in an
  implementation that had it. Entry names are case-sensitive by specification, so two
  entries may legally differ only in case.
- **Verdict**: Rejected. A missing-dataset error names any other root `.ged` entries, so a
  misnamed real-world archive produces an actionable report, and evidence for revisiting.

### Option D: Fail the archive on any unsafe entry

- **Pros**: Simplest rule.
- **Cons**: One stray symlink or oddly named entry, unrelated to the dataset, makes the
  whole archive unreadable.
- **Verdict**: Rejected for usability problems. Kept for limits: see below.

### Option E: Separate archive, lazy media, strict resolution, bounded (Selected)

- **Pros**: Memory is bounded by the caller's limits rather than by the file; one
  resolution rule that can be stated in a sentence; `fs.FS` covers a directory, memory and
  another archive with one parameter.
- **Cons**: More calls than a single eager decode; archives that depend on lenient
  matching report missing references instead of opening.
- **Verdict**: Accepted.

## Consequences

### Positive

- A hostile archive is refused from its directory, before anything is inflated.
- Read, edit and write back is `Write(w, doc, reader.FS(), nil)`.
- No new dependency: the container is `archive/zip`.
- The same document and media produce the same bytes.

### Negative

- **A limit breach is all-or-nothing.** An archive with one media entry over
  `MaxEntryBytes` cannot be opened even to read its dataset; the caller must raise the
  limit. Limits apply to every directory entry, referenced or not, and `MaxEntries` counts
  directories. This was chosen because an archive over a limit is presumed hostile, and
  refusing it whole keeps "passed `NewReader`" equal to "within every limit".
- **The default limits are judgement, not derivation.** Nothing in the specification or in
  measurement dictates them. `MaxDocumentBytes` is the tight one because decoding takes
  many times the dataset's size in memory; a service should set it from its own budget.
- **Stricter than the specification in one place.** A tab, carriage return or line feed in
  a local path is rejected, literal or escaped, though the File Path datatype's banned
  list omits them, so that an entry name never holds a control character.
- **`Write` decodes the dataset it has just encoded**, one extra decode per call.

## Implementation notes

As built, where it differs from the plan or is not obvious from the API:

- **The limit+1 read wrapper is defence in depth.** `Open` and `Decode` wrap each entry in
  a reader that returns `ErrLimitExceeded` one byte past the limit. Through `archive/zip`
  it cannot currently fire: the declared size has already passed the same limit in
  `NewReader`, and `archive/zip` fails a read that does not match the declared size. The
  wrapper is kept, and tested directly, so the limit does not rest on that behaviour.
- **A declared compressed size is capped at the archive's length** before the ratio check,
  so overstating it cannot hide a high ratio.
- **The writer takes references from the encoded dataset, read back**, not from a walk of
  the input document's typed media objects. The encoder writes a record from `Record.Tags`
  when it has them ([ADR-003](0003-lossless-dual-storage.md)), so a typed edit not yet
  synced is not what gets written; and for 7.0 output it promotes the inline media links
  of a record built from typed fields into new `OBJE` records. A walk of the input would
  miss both, and the archive would disagree with its own `gedcom.ged`. A decoded 5.5.1
  record written with a 7.0 target keeps its inline `OBJE` structure, so the walk of the
  read-back dataset covers inline `FILE` payloads in record tags as well as `OBJE` records.
- **The writer checks the version twice**: the version the encoder writes by
  (`TargetVersion`, else `Header.Version`) and the one the written header declares, read
  back. The encoder writes a decoded header from its tags, so the two can disagree, and a
  dataset on which they do is not GEDCOM 7 either way round.
- **`CheckRefs` reports what `Write` refuses**, including a reference to `gedcom.ged`
  (`ErrReservedName`), so a document it passes can be written back from `Reader.FS()`.
- **Entries are marked made-on-Unix, mode 0644.** With `archive/zip`'s default creator
  (FAT), Info-ZIP `unzip` on macOS garbled a non-ASCII entry name despite the UTF-8 flag;
  with the Unix marking, `unzip` and Python's `zipfile` both listed it intact.
- **The zip UTF-8 name flag is not consulted on read.** GEDZIP names are UTF-8 by
  definition, so a name is taken as UTF-8 whenever it is valid UTF-8.
- **`%2F` and `%5C` are rejected on the raw payload**, so decoding can neither create nor
  hide a path separator.
- **7.x, not 7.0.** `Write` accepts any version the library reads as GEDCOM 7.
- **ISO/IEC 21320-1 was not consulted directly.** The GEDCOM specification cites it for the
  container; its constraints as applied here (Store or Deflate only, no encryption) are
  taken from secondary knowledge. Encrypted entries are neither read nor written, and
  reading one reports `ErrEncrypted`.
- Not covered: `CheckRefs` does not flag a payload of `gedcom.ged` (it resolves to the
  dataset entry; `Write` refuses it), and it reports each occurrence of a repeated payload.
  `MANIFEST.MF` and `META-INF/`, which the specification discourages, are neither refused
  nor written specially.

## References

- `gedzip/fileref.go` — File Path classification; `gedzip/reader.go` — limits, usability,
  lazy access; `gedzip/writer.go` — the writer
- [GEDZIP guide](../guides/gedzip.md) and
  [untrusted input](../guides/untrusted-input.md#gedzip-archives)
- [FamilySearch GEDCOM 7 specification](https://gedcom.io/specifications/FamilySearchGEDCOMv7.html),
  GEDZIP chapter and File Path datatype
