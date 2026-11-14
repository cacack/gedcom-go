# GEDZIP Archives

How to read and write GEDZIP (`.gdz`) files with the `gedzip` package, and what
to set before reading one you did not make.

GEDZIP is the GEDCOM 7 packaging format: a zip archive holding the dataset in
an entry named exactly `gedcom.ged`, plus the local files that the dataset's
`FILE` and `FILE.TRAN` payloads name. Entry names are case-sensitive UTF-8.

The package keeps the archive apart from the document. Decoding gives an
ordinary `gedcom.Document`; media stays in the archive until it is asked for,
one entry at a time. Nothing is extracted to disk. The reasons are in
[ADR 0012](../decisions/0012-gedzip-archive-model.md), and a runnable program
is in [`examples/gedzip`](../../examples/gedzip).

## Reading

```go
zr, err := gedzip.OpenReader("family.gdz", nil) // nil: default limits
if err != nil {
    return err
}
defer zr.Close()

doc, err := zr.Decode(nil) // nil: the decoder's defaults
if err != nil {
    return err
}
```

`OpenReader` opens a file and keeps it open until `Close`. `NewReader(r, size,
opts)` reads from any `io.ReaderAt`, such as a `bytes.Reader`. Both read only
the archive's directory; nothing is inflated until `Decode` or `Open`.

`Decode` streams `gedcom.ged` into `decoder.DecodeWithOptions`, and
`DecodeWithDiagnostics` into the decoder function of the same name, so
everything in the [decoding guide](decoding.md) applies unchanged: strict and
lenient mode, diagnostics, `Context`, character-set and BOM handling.

Opening fails when:

| Condition | Error wraps |
|-----------|-------------|
| Not a zip archive | `archive/zip`'s error |
| Over a [limit](#untrusted-uploads) | `gedzip.ErrLimitExceeded` |
| No entry named exactly `gedcom.ged` | `gedzip.ErrNoDataset`. The message names up to five other `.ged` entries at the archive's root, since a dataset stored under another name is the likely mistake. They are named, never read. |
| `gedcom.ged` is unusable or appears twice | `gedzip.ErrUnusableEntry`, and `gedzip.ErrEncrypted` when encryption is why |

## Resolving `FILE` References

`Reader.Open` takes a `FILE` or `FILE.TRAN` payload exactly as the document has
it and returns that entry's bytes:

```go
for _, media := range doc.MediaObjects() {
    for _, file := range media.Files {
        rc, err := zr.Open(file.FileRef)
        if errors.Is(err, gedzip.ErrNotLocal) {
            continue // not a file in the archive; see the table below
        }
        if err != nil {
            return err
        }
        _, err = io.Copy(dst, rc)
        rc.Close()
        if err != nil {
            return err
        }
    }
}
```

A payload is classified first, by the GEDCOM 7 File Path rules.
`gedzip.ClassifyFileRef(ref)` returns the kind and, for a local file, the
entry name:

| Payload | Kind | `Open` |
|---------|------|--------|
| `media/photo.jpg` | `RefLocal`, entry `media/photo.jpg` | The entry, or `ErrEntryNotFound`, or `ErrUnusableEntry` |
| `media/CharlotteBront%C3%AB.jpg` | `RefLocal`, entry `media/CharlotteBrontë.jpg` | As above |
| `https://example.com/photo.jpg` (also `http`, `ftp`) | `RefWeb` | `ErrNotLocal`: a web file has no entry |
| `file:///photos/photo.jpg` | `RefFileURL` | `ErrNotLocal`: the specification says `file:` URLs must not be used in a GEDZIP |
| `/photos/photo.jpg`, `../photo.jpg`, `C:\photos\photo.jpg`, `a%2Fb.jpg` | `RefInvalid` | `ErrNotLocal` |

The entry name is the payload percent-decoded once, and it is matched byte for
byte. Entry names themselves are never decoded, so an entry literally named
`50%20off.jpg` is reached by the payload `50%2520off.jpg`. There is no case
folding, no Unicode normalisation, no backslash repair and no trimming: a
payload of `Photo.JPG` does not open `photo.jpg`.

Where the File Path datatype is silent, `ClassifyFileRef` takes the stricter
reading. Beyond what the specification bans, it treats as invalid an escaped
separator (`%2F`, `%5C`), an empty or `.` segment, a trailing `/`, a malformed
percent escape, invalid UTF-8, a `:` in the first segment that does not end a
scheme, and a tab, carriage return or line feed (literal or escaped). The last
three are not on the specification's list of banned characters; they are
rejected so that an entry name never holds a control character.

`Open("gedcom.ged")` returns the dataset's raw bytes.

### Listing and walking

`Reader.Entries()` lists every file entry in archive order, `gedcom.ged`
included, with its declared size and, for an entry that cannot be read, the
reason in `Entry.Unusable`. An unusable entry's `Name` may be anything; never
use it as a filesystem path.

`Reader.FS()` exposes the usable media entries as a read-only `fs.FS`, named by
entry name rather than by payload. `gedcom.ged` and unusable entries are not in
it. Use it with `fs.WalkDir`, or hand it to [`Write`](#writing).

## Checking References

`Reader.CheckRefs(doc)` reports every `FILE` and `FILE.TRAN` payload of the
document that the archive does not satisfy. It reads no entry. The payloads
are those of `OBJE` records and of inline `OBJE` structures in the tags of
other records; GEDCOM 7 has no inline structure, but a dataset written from
older records can carry one.

```go
for _, p := range zr.CheckRefs(doc) {
    fmt.Printf("%s: %q (%s): %v\n", p.XRef, p.FileRef, p.Kind, p.Err)
}
```

| Problem | `Kind` | `Err` wraps |
|---------|--------|-------------|
| Local file with no entry | `RefLocal` | `ErrEntryNotFound` |
| Local file whose entry cannot be read | `RefLocal` | `ErrUnusableEntry` |
| Local reference to `gedcom.ged`, the dataset itself | `RefLocal` | `ErrReservedName` |
| `file:` URL | `RefFileURL` | `ErrNotLocal` |
| Invalid payload | `RefInvalid` | `ErrNotLocal` |

Web references are never a problem. Problems come in document order, one per
reference, so a payload used twice is reported twice. `Translation` is true for
a `FILE.TRAN` payload. Entries the dataset never references are allowed and
not reported. These are the references `Write` refuses, so a document with no
problem can be written back with `Reader.FS()` as its media.

`Reader.Open("gedcom.ged")` still opens the dataset's raw bytes: check
references with `CheckRefs` before serving them as media.

## Writing

`gedzip.Write` packages a GEDCOM 7 document and the files it references. The
media comes from any `fs.FS`; for a directory, use `os.DirFS`:

```go
out, err := os.Create("family.gdz")
if err != nil {
    return err
}
defer out.Close()

// A FILE payload of media/photo.jpg is read from family-media/media/photo.jpg.
if err := gedzip.Write(out, doc, os.DirFS("family-media"), nil); err != nil {
    return err
}
```

Another archive's `Reader.FS()` is a media source too, which makes read, edit
and write back short:

```go
doc, err := zr.Decode(nil)
// ... change doc ...
err = gedzip.Write(out, doc, zr.FS(), nil)
```

The media source may be nil when the dataset has no local reference.
`WriteOptions.Encode` passes `encoder.EncodeOptions` to the dataset's encoding.

What is written:

- `gedcom.ged` first, then one entry per distinct local file the dataset
  names, sorted by entry name. A file referenced twice is written once. A file
  in the source that nothing references is not written. There are no directory
  entries.
- Entry names are the percent-decoded payloads, in UTF-8. Payloads in the
  dataset are written as they are.
- The dataset and media with a text extension (`csv`, `ged`, `htm`, `html`,
  `json`, `md`, `svg`, `txt`, `vtt`, `xml`) are deflated. Everything else is
  taken to be compressed already and is stored.
- Every entry has the same fixed timestamp and mode `0644`, whatever the source
  reports, so the same document and media give the same bytes.
- Entries are marked as made on Unix. With `archive/zip`'s default (FAT),
  Info-ZIP `unzip` reads a non-ASCII name as a DOS code page and garbles it,
  even though the entry's UTF-8 flag is set.

### What `Write` refuses

`Write` validates everything first and writes nothing if any check fails:

| Refused | Error wraps |
|---------|-------------|
| A nil document | `encoder.ErrNilDocument` |
| A dataset that is not written as GEDCOM 7.x | `gedzip.ErrNotGEDCOM7` |
| A `file:` URL, or an invalid payload | `gedzip.ErrNotLocal` |
| A local reference to `gedcom.ged` | `gedzip.ErrReservedName`; the specification requires such a file to be renamed |
| A local reference with no regular file in the media source | `gedzip.ErrMissingMedia` |

Every refused reference is reported, once per distinct payload, joined into one
error and prefixed with the XRef of the record holding it. Test with
`errors.Is`.

Two versions must both be 7.x: the one the encoder writes by
(`WriteOptions.Encode.TargetVersion` when set, otherwise the document's
`Header.Version`) and the one the written header declares. They differ when
`Header.Version` is changed on a decoded document, because the encoder writes
a decoded header from its tags. `Write` converts nothing. Convert an older
document first, and expect to fix its paths, because `Write` does not rewrite
payloads either:

```go
doc7, _, err := converter.Convert(doc, gedcom.Version70)
if err != nil {
    return err
}
err = gedzip.Write(out, doc7, media, nil)
```

### References come from the written dataset

`Write` encodes the dataset, reads it back, and takes the references from that
rather than from the typed media objects of the document it was given. The two
can differ:

- The encoder writes a record from its `Record.Tags` when it has them. A change
  made only to a typed field of a decoded record, such as `MediaFile.FileRef`,
  is not written until `encoder.SyncTags(doc)` has carried it into the tags.
- For GEDCOM 7 output the encoder moves the inline media links (an `OBJE` with
  no pointer, from 5.5 or 5.5.1) of a record built from typed fields into new
  `OBJE` records.
- A decoded 5.5.1 record written with a 7.0 `TargetVersion` keeps its inline
  `OBJE` structure, since its tags are written as they are. `Write` treats the
  `FILE` payloads beneath it as references like any other: they are checked
  and their files archived. `converter.Convert` turns such links into proper
  `OBJE` records and is the better route.

Taking references from the written dataset means the archive always matches
the `gedcom.ged` inside it. The XRefs in a refusal are those of the written
dataset. The cost is one extra decode of the dataset per `Write`, so the
document, its encoded bytes and a second decoded document are in memory
together.

An error after validation, such as the destination failing or a media file
that cannot be read in full, leaves an incomplete archive in the destination.
Write to a temporary file and rename it if that matters.

### Writing from an untrusted document

`Write` copies whatever regular file a reference names. When the document
comes from someone else, its author chooses which files of the media source
end up in the archive, so give `Write` a source that holds nothing but files
you are willing to publish: a dedicated directory, an allowlisting `fs.FS`, or
another archive's `Reader.FS()`. Paths cannot leave the source, but `os.DirFS`
follows symbolic links, including ones that point outside its directory.

### Size

`Write` applies no size limit. An archive holding more or larger media than the
[`ReadOptions` defaults](#untrusted-uploads) allow, such as a single file over
1 GiB, is written as asked; `NewReader` then refuses it until the reader raises
those limits.

## Untrusted Uploads

`ReadOptions` bounds what a `Reader` accepts. In every field zero means the
default and a negative value means unlimited; a nil `*ReadOptions` is all
defaults.

| Field | Caps | Default |
|-------|------|---------|
| `MaxEntries` | Entries in the archive's directory, directories included | `DefaultMaxEntries` (100,000) |
| `MaxDocumentBytes` | Uncompressed size of `gedcom.ged` | `DefaultMaxDocumentBytes` (256 MiB) |
| `MaxEntryBytes` | Uncompressed size of each other entry | `DefaultMaxEntryBytes` (1 GiB) |
| `MaxTotalBytes` | Sum of the declared uncompressed sizes | `DefaultMaxTotalBytes` (4 GiB) |
| `MaxCompressionRatio` | Uncompressed ÷ compressed size, for each entry declaring more than `CompressionRatioMinBytes` (1 MiB) | `DefaultMaxCompressionRatio` (200) |

Nothing in the specification dictates these values. The defaults are sized to
refuse a hostile archive before it is inflated; lower them to what your
service expects:

```go
zr, err := gedzip.NewReader(upload, size, &gedzip.ReadOptions{
    MaxEntries:       10_000,
    MaxDocumentBytes: 16 << 20,
    MaxEntryBytes:    64 << 20,
    MaxTotalBytes:    512 << 20,
})
if errors.Is(err, gedzip.ErrLimitExceeded) {
    return errors.New("archive too large")
}
```

How the limits behave:

- **A breach refuses the whole archive.** The limits are checked against the
  sizes the directory declares when the archive is opened, for every entry,
  whether or not the dataset references it. `NewReader` fails with
  `ErrLimitExceeded`, naming the entry and the option. This differs from the
  usability rules below, which only make one entry unreachable.
- **Declared sizes cannot be understated.** `archive/zip` fails a read whose
  bytes do not match the declared size or checksum, and the declared size has
  already passed the limit. As defence in depth, `Open` and `Decode` also count
  the bytes inflated and stop with `ErrLimitExceeded` one byte past the limit,
  so the limit holds without relying on that check, and an entry is never
  returned silently short.
- **The ratio cannot be dodged by overstating the compressed size**: a
  declared compressed size is capped at the archive's length before the ratio
  is taken.

Two things remain yours to bound:

- **The archive's own size.** `NewReader` needs an `io.ReaderAt` and a length,
  so an upload is held in memory or spooled to a file before the package sees
  it. Cap the request body first.
- **Decode memory.** Decoding takes many times the dataset's size in memory.
  Set `MaxDocumentBytes` from your memory budget using
  [Sizing `io.LimitReader`](untrusted-input.md#sizing-iolimitreader); the
  default is far above what a small service should accept.

### Entries that cannot be reached

Apart from the limits, each entry is judged on its own. An entry is usable only
if it:

- is not encrypted;
- is a regular file, not a symbolic link or other special file;
- is stored or deflated;
- has a name that is valid UTF-8 and a clean relative slash-separated path: no
  leading `/`, no backslash, no empty, `.` or `..` segment, no control
  character;
- is the only entry with that name.

An unusable entry is listed by `Entries()` with the reason, is absent from
`FS()`, and `Open` returns its reason wrapping `ErrUnusableEntry`. It does not
stop `gedcom.ged` or other media from being read. The exception is `gedcom.ged`
itself: if it is unusable or duplicated, opening the archive fails.

The package never writes to the filesystem, and never opens a path derived
from an archive entry or a payload. If you copy media out yourself, build the
destination path from a name you chose, or from a `ClassifyFileRef` entry name
joined under a directory you control, never from `Entry.Name`.

## Out of Scope

- **Extracting to a directory.** Read entries with `Open` or `FS()` and write
  them where you choose.
- **Encryption.** Encrypted entries are neither read nor written.
- **Lenient matching.** No case-insensitive or suffix matching of entry names,
  no backslash repair, and no fallback to a dataset under another name.
- **Rewriting `FILE` payloads**, on read or on write.
- **Converting to GEDCOM 7** inside `Write`; use the
  [converter](converter.md) first.
- **Compression methods other than Store and Deflate.**

The zip container rules come from ISO/IEC 21320-1, which the GEDCOM 7
specification cites and which was not consulted directly for this package;
Store and Deflate only, and no encryption, are what the package holds to.
