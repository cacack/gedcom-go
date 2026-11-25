# Preserving Layout

How to write an edited GEDCOM file back so that it differs from its source only
where it was edited.

By default the encoder normalizes layout: it writes one line terminator (LF,
or CRLF when asked), no byte order mark, no indentation, no blank lines, and no
delimiter space on a valueless line. That is the right output for cleaning a
file up. It is the wrong output for a file kept under version control, because
the first edit rewrites lines nobody touched and the diff cannot be reviewed.
Real exports differ from the normalized form in exactly those ways: a UTF-8 BOM
with CRLF terminators, CRLF lines inside an otherwise LF file, indentation by
level, blank lines between records, `2 CONT ` with a trailing space.

`PreserveLayout` writes every line the caller did not change exactly as it was
read.

## Usage

Set it on both sides: the decoder records the layout, and the encoder writes it
back.

```go
doc, err := gedcomgo.DecodeWithOptions(reader, &gedcomgo.DecodeOptions{PreserveLayout: true})
if err != nil {
    log.Fatal(err)
}

// Edit doc: Record.Tags directly, or typed fields followed by encoder.SyncTags.

if err := gedcomgo.EncodeWithOptions(writer, doc, &gedcomgo.EncodeOptions{PreserveLayout: true}); err != nil {
    log.Fatal(err)
}
```

The decoder stores the layout in `Document.Layout`. The encoder option does
nothing on a document without one, and a document with one encodes exactly as
before unless the encoder option is set. Both options default to off, so
existing callers are unaffected.

`EncodeStreamingWithOptions` honors it too. A bare `StreamEncoder` has no
document to take a layout from.

## What Is Kept

The rule is per line, not per file. A line is written from its source when it
still reads as that source line: the same level, cross-reference identifier,
tag and value. With such a line the encoder keeps:

- its indentation, and any extra spacing inside it (`1  NAME`);
- its terminator, so a file mixing LF and CRLF keeps both, and a bare-CR file
  stays bare CR;
- the blank or skipped lines immediately before it;
- spacing the decoded value does not hold: the delimiter space on a valueless
  line (`2 CONT `), and a value of nothing but spaces.

Also kept: a UTF-8 byte order mark, everything after the last line (trailing
blank lines), and a missing final terminator.

## What Changes

**A changed line** is written as the encoder writes it without the option, but
keeps its source line's indentation, terminator and preceding blank lines. A
removed line takes its preceding blank lines with it. A line moved elsewhere
keeps its own layout. A copied line (a cloned tag or record, which keeps its
`LineNumber`) is written from its source once, at its first occurrence; every
other copy is a new line.

**A new line** (one the caller added, a CONT for a line break added to a value,
or a line the encoder supplies) has no source line. It is written unindented,
with the terminator of the source's first line, so an edit to a CRLF file does
not introduce LF lines. `LineEnding` applies only when the source had no
terminator at all.

**A replaced or removed record** (`Document.ReplaceRecord`,
`Document.RemoveRecord`) leaves every other record written as read. Records
from `decoder.DecodeRecords` carry no line numbers, so all their lines are new
lines; a replacement made by editing a `Clone` of the old record keeps the
layout of the lines it did not change.

**A raw tag's whole value** set with `Record.SetFullValue` rewrites that tag
and its CONT/CONC lines only: the tag keeps its source line's layout, its new
continuation lines are new lines, and its other children keep theirs.

**Typed edits** work through `encoder.SyncTags`, which keeps every unchanged
line's `Tag`, so only the edited structure differs. Lines that `SyncTags`
rewrites are new lines.

**The encoder's own rules still apply**, and what they change counts as a
change:

- The header's `CHAR` is written as `UTF-8` whatever the source declared,
  because the output is UTF-8 (see below). A source declaring `ASCII`, `ANSEL`
  or `ANSI` over ASCII bytes therefore differs on that one line.
- A level that lenient decoding clamped (`1 BIRT` then `4 DATE`) is written
  clamped. The layout records each line as the parser read it, before recovery,
  so a recovered line never reads as unchanged.
- `TargetVersion`, `DropUnknownTags` and the header the encoder supplies for a
  headerless file work as they do without the option.

**A combined document** (`merge.Combine`) has no layout, because its lines
come from two files, and is written in the normalized form.

## Non-UTF-8 Sources

The encoder writes UTF-8 regardless of the source encoding. A source in ANSEL,
UTF-16 or a code page keeps its layout (indentation, terminators, blank lines)
but not its bytes: every line holding a non-ASCII character differs, as does
`CHAR`. A UTF-16 byte order mark is not written; a UTF-8 one is written only
when the source had one. Writing the source's own encoding back is not
supported.

## Whitespace-Only Values

The parser reads a value of nothing but spaces (`2 CONT` followed by three
spaces) as no value, as it reads `2 CONT ` with its single delimiter space.
That is unchanged: the document holds an empty value, and without
`PreserveLayout` the line is written as a bare `2 CONT`. With `PreserveLayout`
the line is written exactly as read, spaces included, until it is edited.

## Cost

The layout holds a second copy of the decoded input and a record per line.
Decoding a 2.4 MB export took 38 MB rather than 25 MB. Set
`DecodeOptions.PreserveLayout` only for a document that will be written back.

## How It Is Tested

`layout_roundtrip_test.go` decodes and encodes every fixture in `testdata/`
with `PreserveLayout` and compares the output with the input byte for byte,
with no normalization. Each fixture that cannot match is listed with its
reason, and the list is self-cleaning: a listed fixture that starts matching
fails the test. `encoder/layout_test.go` covers each layout above on synthetic
input, and edits that must change only their own lines.
