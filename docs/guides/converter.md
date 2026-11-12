# GEDCOM Version Converter

The converter package provides bidirectional conversion between GEDCOM versions 5.5, 5.5.1, and 7.0.

## Overview

The GEDCOM specification has evolved through multiple versions, each with different features and requirements. This converter enables:

- **Upgrading** documents to newer versions (5.5 -> 5.5.1 -> 7.0)
- **Downgrading** documents to older versions (7.0 -> 5.5.1 -> 5.5)
- **Tracking** all transformations and data loss in a detailed report

## Quick Start

```go
import (
    "github.com/cacack/gedcom-go/v3/converter"
    "github.com/cacack/gedcom-go/v3/decoder"
    "github.com/cacack/gedcom-go/v3/gedcom"
)

// Load a GEDCOM 5.5 file
doc, _ := decoder.Decode(reader)

// Convert to GEDCOM 7.0
converted, report, err := converter.Convert(doc, gedcom.Version70)
if err != nil {
    log.Fatal(err)
}

// Check what changed
fmt.Println(report)
```

## API Reference

### Convert

```go
func Convert(doc *gedcom.Document, targetVersion gedcom.Version) (*gedcom.Document, *gedcom.ConversionReport, error)
```

Converts a document to the target version using default options. The original document is not modified; a deep copy is created for conversion.

When the document is already at the target version, it is returned as-is and the report is empty. The exception is 7.0 → 7.0: XRefs the 7.0 grammar rejects (`0 @i 2@ INDI`) are normalized on a copy exactly as in [XRef normalization](#upgrade-to-gedcom-70), pointers included, and each rename is a `Normalized` note, so converting to 7.0 always yields valid 7.0 identifiers. No other transformation runs. A dangling pointer containing a space (`1 NOTE @X 9@`) is not a pointer under the 7.0 grammar a 7.0 header declares, so it is left as-is. A 7.0 → 7.0 call therefore returns the input itself when nothing needs renaming and a copy otherwise, so do not rely on the result's pointer identity.

### ConvertWithOptions

```go
func ConvertWithOptions(doc *gedcom.Document, targetVersion gedcom.Version, opts *ConvertOptions) (*gedcom.Document, *gedcom.ConversionReport, error)
```

Converts a document with custom options for fine-grained control over the conversion process.

A document already at the target version is handled as described under [Convert](#convert); the options do not apply on that path.

### ConvertOptions

| Option | `DefaultOptions` | Description |
|--------|------------------|-------------|
| `Validate` | `true` | Run validation on converted document |
| `StrictDataLoss` | `false` | Fail if any data would be lost |
| `ReportPreservedTags` | `true` | Itemise each preserved vendor/unknown tag in the report. Report-only — tags are preserved either way |
| `MapEXIDToVendorTags` | `true` | On a 7.0 downgrade, map a FamilySearch ARK EXID to `_FSFTID` instead of recording it as data loss |

Nothing the converter does drops a tag: preservation is unconditional, and
`ReportPreservedTags` only controls whether the report lists what was kept.

`nil` and a wholly zero `&converter.ConvertOptions{}` both mean "use the
defaults". A **partially** populated literal does not — each field you leave
out takes its own zero value, so start from `DefaultOptions()` to change one
setting:

```go
opts := converter.DefaultOptions()
opts.StrictDataLoss = true  // Fail on any data loss
converted, report, err := converter.ConvertWithOptions(doc, gedcom.Version70, opts)
```

## Version-Specific Transformations

### Upgrade to GEDCOM 7.0

| Transformation | Type | Description |
|---------------|------|-------------|
| CONC removal | `CONC_REMOVED` | Line continuations merged into single values, at any depth, including those continuing a level-0 line such as a 5.x `NOTE` record's text |
| XRef normalization | `XREF_UPPERCASE` | Cross-references rewritten to the 7.0 form `@[A-Z0-9_]+@`: uppercased, other characters (such as 5.5.1 spaces) replaced by `_`, and a `_2`, `_3`, ... suffix added on collision with an existing record or pointer (never `@VOID@`); pointers follow, and each rewrite is a `Normalized` note. A record named `@VOID@`, an ordinary 5.5/5.5.1 identifier but the 7.0 null pointer, is renamed the same way (`@VOID_2@`) so its pointers do not become null; a `@VOID@` pointer no record defines stays the null pointer. A pointer to an undefined XRef joins a record only when they differ by ASCII case alone, otherwise it gets its own identifier. A dangling spaced pointer (`1 NOTE @X 9@`) is rewritten only when the source header declares 5.5 or 5.5.1 in `GEDC.VERS`; without that declaration it is left as-is. `Count` includes rewritten dangling pointers |
| Note records | `NOTE_RECORD_TO_SNOTE` | `0 @N1@ NOTE` records become `0 @N1@ SNOTE` shared note records (the typed `Note` becomes a `SharedNote`); each is a `Normalized` note |
| Note pointers | `NOTE_POINTER_TO_SNOTE` | `NOTE @N1@` pointers to a converted note record become `SNOTE @N1@`, at any level; inline note text and pointers no record defines stay `NOTE` |
| Media types | `MEDIA_TYPE_MAPPED` | Legacy formats (JPG) converted to IANA (image/jpeg) |
| Date calendars | `DATE_CALENDAR_CONVERTED` | Calendar escapes (`@#DJULIAN@`) converted to keywords (`JULIAN`) |
| Date epochs | `DATE_EPOCH_CONVERTED` | `B.C.`, `BC` and `B.C.E.` spelled `BCE`; see [Date epochs and phrases](#date-epochs-and-phrases) |
| Date phrases | `DATE_PHRASE_CONVERTED` | `INT <date> (<phrase>)` becomes `DATE <date>` + `PHRASE`, `(<phrase>)` an empty `DATE` + `PHRASE` |
| Enumerations | `ENUMERATION_CONVERTED` | Enumerated payloads translated to the 7.0 vocabulary (`PEDI birth` → `PEDI BIRTH`, `ASSO.RELA` → `ROLE`, `AGE CHILD` → `AGE < 8y` + `PHRASE`); see [Enumeration Mappings](#enumeration-mappings) |
| Vendor `_UID` | `UID_FROM_VENDOR_TAG` | A `_UID` tag becomes the 7.0 `UID` it standardised wherever 7.0 defines `UID`: at level 1 of an `INDI`, `FAM`, `SOUR`, `REPO`, `OBJE`, `SNOTE` or `SUBM` record, and at level 2 under an individual or family event or attribute. A `_UID` elsewhere (under `NAME`, say) stays an extension. The value is carried unchanged, as 7.0 asks for an existing identifier; each rename is a `Normalized` note, naming the enclosing event for a nested one |
| `RESN` | `RESTRICTION_VALUES_MAPPED` | Restriction notice keywords at any level become upper case (`privacy` -> `PRIVACY`); a `Normalized` note each. A value that is not `confidential`, `locked` or `privacy` is left as written |
| Vendor `_FREL`/`_MREL` | `PEDIGREE_FROM_VENDOR_TAG` | A child's `FAMC` gains a `PEDI` from the `_FREL`/`_MREL` under the family's `CHIL` when the mapping is unambiguous: each parent the family names has a relationship, all map to one value (`Adopted`, `Foster`, `Sealing`), and it is not `Natural`/`Birth`. `Step`, `Guardian` and other values have no `PEDI` and add none; a `FAMC` that already has a `PEDI` is kept. The vendor tags stay; each `PEDI` added is a `Normalized` note |
| Vendor `_MARNM` | `MARRIED_NAME_FROM_VENDOR_TAG` | A `NAME` with `_MARNM` is followed by a new `NAME` with `TYPE MARRIED`: the original name with the `_MARNM` surname in place of its own, or the `_MARNM` value itself when it is a full name (has a slash). A matching `TYPE MARRIED` name already present is not added again. `_MARNM` stays; each name added is a `Normalized` note |
| Inline media links | `INLINE_OBJE_TO_RECORD` | An `OBJE` link with no pointer, carrying its `FILE` in place (5.5.1 `FILE`/`FORM`/`TITL`, or 5.5 `FORM` beside `FILE`), becomes a new `OBJE` record at the first free `@O<n>@` and a pointer to it, as the 7.0 migration guide gives; a link-level `TITL` and `FORM` go under the first `FILE`, `NOTE` and extensions stay at record level. A link with no `FILE` is left as it is. Each is a `Normalized` note at the link's path |
| `AFN`, `RFN`, `RIN` | `IDENTIFIER_TO_EXID` | A record's level-1 `AFN`, `RFN` or `RIN` becomes an `EXID` with the registered `TYPE`: `https://gedcom.io/terms/v7/AFN`; `.../RIN#<HEAD.SOUR>` (no fragment when the header names no source system); `.../RFN#<resource>` with the value after the resource's colon (`RFN Resource:5431` -> `EXID 5431`), or the bare `RFN` URI and the whole value when there is no colon. Each is a `Normalized` note |
| `LANG` | -- | `HEAD.LANG` and `SUBM.LANG` language names become BCP 47 tags (`English` -> `en`, case-insensitive), a `Normalized` note each. A value already tag-shaped is left alone; any other name is kept verbatim with an `Approximated` note |
| Version-foreign structures | `FOREIGN_STRUCTURES_DROPPED` | From 5.5.1 only: see [Version-foreign structures](#version-foreign-structures) |
| `@` escaping | `AT_SIGN_REESCAPED` | 5.x doubles every literal `@` in text, 7.0 only a leading one, so every raw text value is rewritten for 7.0 (`a@@b` -> `a@b`; `@@lead` and a leading `@@` on a `CONT` line stay). Pointers, tag names, XRefs and 5.x escape sequences such as an unmapped `@#DROMAN@` are left as they are; each rewrite is a `Normalized` note |
| Header update | `VERSION_UPGRADE` | Header version updated to 7.0 |

### Upgrade 5.5 to 5.5.1

| Transformation | Type | Description |
|---------------|------|-------------|
| Header update | `VERSION_UPGRADE` | Backward compatible upgrade |
| Vendor `_FREL`/`_MREL` | `PEDIGREE_FROM_VENDOR_TAG` | As for the upgrade to 7.0, with the `PEDI` value in lower case (`adopted`) |
| GEDC.FORM | `GEDC_FORM_ADDED` | `2 FORM LINEAGE-LINKED` added under `GEDC` when missing |

### Downgrade from GEDCOM 7.0

| Transformation | Type | Description |
|---------------|------|-------------|
| Newlines to CONT | `CONT_EXPANDED` | Embedded newlines expanded to CONT tags |
| `@` escaping | `AT_SIGN_REESCAPED` | Every literal `@` in a raw text value doubled for 5.x (`a@b` -> `a@@b`, `@@lead` stays), before `CONC` splitting so no pair is split; pointers are left as they are. Each rewrite is a `Normalized` note |
| Long lines to CONC | `CONC_ADDED` | A value line too long for 5.x's 255-character limit split onto CONC tags, between two non-spaces, including a level-0 `SNOTE` record's text; each split value is a `Normalized` note |
| Media types | `MEDIA_TYPE_MAPPED` | IANA formats converted to legacy |
| Date calendars | `DATE_CALENDAR_CONVERTED` | Calendar keywords (`JULIAN`) converted to escapes (`@#DJULIAN@`) |
| Date epochs | `DATE_EPOCH_CONVERTED` | `BCE` spelled `B.C.` |
| Date phrases | `DATE_PHRASE_CONVERTED` | `DATE <date>` + `PHRASE` becomes `INT <date> (<phrase>)`, an empty `DATE` + `PHRASE` becomes `(<phrase>)` |
| `RESN` | `RESTRICTION_VALUES_MAPPED` | Restriction notice keywords at any level become lower case (`LOCKED` -> `locked`); a `Normalized` note each. 5.x takes one value, so a 7.0 list (`CONFIDENTIAL, LOCKED`) keeps its most protective keyword (privacy, then confidential, then locked) and the rest is reported as `DataLoss` (`RESN restriction lists`) |
| Enumerations | `ENUMERATION_CONVERTED` | Enumerated payloads translated to the 5.x vocabulary (`PEDI BIRTH` → `PEDI birth`, `ASSO.ROLE` → `RELA`); see [Enumeration Mappings](#enumeration-mappings) |
| `UID` to vendor `_UID` | `UID_TO_VENDOR_TAG` | Every `UID` tag, at any level, becomes `_UID`, the extension 5.5.x producers use for the same identifier; each is a `Normalized` note, and `UID` is not data loss |
| `LANG` | -- | `HEAD.LANG` and `SUBM.LANG` BCP 47 tags become 5.x language names by primary subtag: `de` -> `German` is `Normalized`, `en-US` -> `English` (region lost) is `Approximated`. A value already a 5.x name is left alone; a tag with no name is kept verbatim with an `Approximated` note |
| `HEAD.SOUR` | `HEAD_SOUR_ADDED` | Added as `gedcom-go` when the header names no source system (required by 5.5 and 5.5.1) |
| `HEAD.SUBM` | `HEAD_SUBM_ADDED`, `SUBM_RECORD_ADDED` | Added when missing (required by 5.5 and 5.5.1), pointing at the first `SUBM` record; when there is none, a `SUBM` record with `NAME Unknown` is added at the first free `@SUBM<n>@` |
| Version-foreign structures | `FOREIGN_STRUCTURES_DROPPED` | To 5.5.1 only: see [Version-foreign structures](#version-foreign-structures) |
| Header update | `VERSION_DOWNGRADE` | Header version updated |
| GEDC.FORM | `GEDC_FORM_ADDED` | `2 FORM LINEAGE-LINKED` added under `GEDC` when missing (required by 5.5 and 5.5.1) |

### Version-foreign structures

Between 7.0 and 5.5.1, after every mapping above, the converter removes each
structure the source version defines and the target does not, with its
subordinate lines, from the header and every record's `Tags`: 7.0's `SDATE`,
`INIL`, `CROP`, `FAM.FACT`, `ASSO` under an event or a family, `TIME` on an
event date, `NOTE.LANG`/`MIME`/`SOUR`, `TEXT.LANG`/`MIME`, `PLAC.LANG`,
`SUBM.REFN`, `OBJE.RESN`, `NCHI` substructures and the header note's
`TRAN`, `MIME` and `LANG` on a downgrade; 5.5.1's `HEAD.FILE` on an upgrade.
Each is a `Dropped` note at its full path (`Family @F1@ > MARR > SDATE`), and
each parent.tag pair one data loss entry (`MARR.SDATE structures`) naming the
records it was dropped from.

What counts as foreign is generated from the vendored specification files
(`testdata/spec/*/substructures.tsv`) into `converter/structures_gen.go` by
`go generate ./converter`: the pairs (superstructure, tag) the target has no
structure for wherever their superstructure is reached, and, where that
depends on where the superstructure sits (7.0 allows `TIME` on every exact
date, 5.5.1 only on a `CHAN` date), the triples with the superstructure's
parent. A tag is read in its source context, so an extension tag, a
nonconforming tag and everything under either are kept, as are `CONT` and
`CONC`. Two pairs are left to the enumeration translation, which maps rather
than drops them: `ASSO.RELA`/`ROLE` and the record `FILE.FORM.TYPE`/`MEDI`.
An inline source citation's `TEXT` (5.5.1 `SOUR <text>` / `TEXT`) is kept,
since 7.0 has no inline citation and nothing converts one to a record yet ([#793](https://github.com/cacack/gedcom-go/issues/793)). A
record with no `Tags` is written from its entity and not walked. Conversions
to and from 5.5 do not drop version-foreign structures yet ([#794](https://github.com/cacack/gedcom-go/issues/794)).

Every conversion sets the header's `GEDC.VERS` to the target version, rewriting
any existing `VERS` under any `GEDC`. If no `GEDC` has a `VERS`, one is added
under the first `GEDC`, keeping its other children (`FORM` is kept for 5.5 and
5.5.1 targets but dropped for 7.0, which removed it). A header with no `GEDC`
gets one added. A 5.5 or 5.5.1 target's `GEDC` always carries
`FORM LINEAGE-LINKED`, after `VERS`.

The header's level-1 structures are then put in the target version's grammar
order -- `SOUR, DEST, DATE, SUBM, SUBN, FILE, COPR, GEDC, CHAR, LANG, PLAC,
NOTE` for 5.5/5.5.1, `GEDC, SCHMA, SOUR, DEST, DATE, SUBM, COPR, LANG, PLAC,
NOTE` for 7.0 -- each moving with its subordinate lines. Structures the target
grammar does not define (extensions, or `FILE` in 7.0) go last in source order.
A same-version conversion returns the document unchanged, so its header keeps
the source's order. The one header repair it makes is `VERS`: a `GEDC` with no
`VERS` (such a header is detected by tag fallback) gets `2 VERS <target>` added
as its first child, on a copy. A header with no `GEDC`, or one that already
declares `VERS`, is left as it is.

## Media Type Mappings

When converting between versions, media types are automatically mapped:

| Legacy (5.5/5.5.1) | IANA (7.0) |
|-------------------|------------|
| JPG, JPEG | image/jpeg |
| PNG | image/png |
| GIF | image/gif |
| TIFF, TIF | image/tiff |
| BMP | image/bmp |
| MP3 | audio/mpeg |
| WAV | audio/wav |
| MP4 | video/mp4 |
| MPEG, MPG | video/mpeg |
| AVI | video/x-msvideo |
| PDF | application/pdf |
| TXT, TEXT | text/plain |

## Date Calendar Mappings

GEDCOM 5.5/5.5.1 marks a date's calendar with an escape; GEDCOM 7.0 uses a
keyword. Converting to or from 7.0 rewrites each one:

| 5.5/5.5.1 escape | 7.0 keyword |
|------------------|-------------|
| `@#DGREGORIAN@` | `GREGORIAN` |
| `@#DJULIAN@` | `JULIAN` |
| `@#DHEBREW@` | `HEBREW` |
| `@#DFRENCH R@` | `FRENCH_R` |

A calendar belongs to each date, so both endpoints of a range or period are
translated (`BET @#DJULIAN@ 1700 AND @#DGREGORIAN@ 1710` becomes
`BET JULIAN 1700 AND GREGORIAN 1710`); date phrase text in parentheses is left
alone. Every `DATE` and `SDATE` value in `Record.Tags` and `Header.Tags` (such
as `HEAD.SOUR.DATA.DATE`) is rewritten, as are `Header.Date` and
`Date`/`SortDate` on individual and family events, attributes and LDS
ordinances (with `ParsedDate.Original` kept in step), and the `Data.Date` of
source citations on individual, family, media and shared note records and on
their events, attributes and associations. Each rewrite is a Normalized note.

A calendar with no standard equivalent -- a 5.x escape such as `@#DROMAN@` or
`@#DUNKNOWN@`, or a 7.0 extension calendar such as `_MAYAN` -- is left
unchanged and reported as a Preserved note, so the output may not be valid in
the target version for that date. This is not data loss, so `StrictDataLoss`
does not reject it: check `report.Preserved` for these notes when the output
must conform to the target version.

## Date Epochs and Phrases

GEDCOM 7.0 spells the B.C. epoch only `BCE`, and has neither the `INT`
modifier nor a parenthesized date phrase: a phrase goes in a `PHRASE`
substructure of the `DATE`. Converting to or from 7.0 rewrites each date:

| 5.5/5.5.1 | 7.0 |
|-----------|-----|
| `44 B.C.` (also `BC`, `B.C.E.`) | `44 BCE` |
| `INT 1850 (about eighteen fifty)` | `DATE 1850` + `PHRASE about eighteen fifty` |
| `INT 1850` | `DATE 1850` |
| `(unknown)` | empty `DATE` + `PHRASE unknown` |

The downgrade writes `B.C.` and folds a `PHRASE` back into the value, so both
lossless 5.x forms survive a round trip. Each rewrite is a Normalized note
whose `Original` or `Result` shows the phrase as `1850 + PHRASE about
eighteen fifty`. Typed dates follow: `Event.Date`, `Attribute.Date` and
`LDSOrdinance.Date` change with the raw tag, and their `ParsedDate` is
re-parsed as decoding the output would give it, with `Phrase` holding a 7.0
`PHRASE`. An `SDATE` phrase moves the same way, between `SortDate` and
`SortDatePhrase`.

A phrase with no target form is reported rather than guessed at:

- To 7.0, a phrase that is neither `INT <date> (<phrase>)` nor the whole value
  (`ABT 1850 (circa)`), or that sits where 7.0 allows no `PHRASE` (an exact
  date such as `CHAN.DATE` or `HEAD.DATE`, a `DATE` that already has a
  `PHRASE`, or a typed citation or header date on a record built in code), is
  left in the value. The output is not valid 7.0 for that date: it is a
  `Preserved` note and part of the `Date phrases` DataLoss item.
- To 5.x, a `PHRASE` beside a value `INT` cannot qualify (a range, period or
  approximation such as `ABT 1850`), one whose text holds a parenthesis or a
  line break, or one that would push the `DATE` line past 255 characters is
  dropped and reported under `PHRASE tags`.

## Enumeration Mappings

Converting to or from 7.0 translates every enumerated payload in
`Record.Tags` between the 5.5/5.5.1 spelling and the 7.0 enumeration
(`testdata/spec/gedcom-7.0/enumerationsets.tsv`). Each rewrite is a
`Normalized` note. `RESN` is translated by its own step (`RESTRICTION_VALUES_MAPPED`, above).

| Structure | 5.5/5.5.1 | 7.0 |
|-----------|-----------|-----|
| `INDI.SEX` | `M`, `F`, `U` | `M`, `F`, `U`, `X` |
| `INDI.NAME.TYPE` | `aka`, `birth`, `immigrant`, `maiden`, `married`, free text | `AKA` ... `MARRIED`, `PROFESSIONAL`, `OTHER` + `PHRASE` |
| `FAMC.PEDI` | `adopted`, `birth`, `foster`, `sealing` | `ADOPTED` ... `SEALING`, `OTHER` + `PHRASE` |
| `FAMC.STAT` | `challenged`, `disproven`, `proven` | `CHALLENGED`, `DISPROVEN`, `PROVEN` |
| LDS ordinance `STAT` | `PRE-1970`, `DNS/CAN`, ... | `PRE_1970`, `DNS_CAN`, ... |
| `MEDI` (and 5.5.1 `OBJE.FILE.FORM.TYPE`) | `audio`, `photo`, ..., free text | `AUDIO`, `PHOTO`, ..., `OTHER` + `PHRASE` (`OBJE.FILE.FORM.MEDI`) |
| `ASSO` role | `RELA <text>` | `ROLE <enum>` |
| `SOUR.EVEN.ROLE` | `CHIL` ... `SPOU`, `(<descriptor>)` | `ROLE <enum>` |
| `AGE` | `CHILD`, `INFANT`, `STILLBORN`, `<8`, `11m99y` | `< 8y`, `< 1y`, `0y` + `PHRASE <keyword>`; `< 8y`, `99y 11m` |

To 7.0, a value matching an enumeration case-insensitively takes its 7.0
spelling; a value an open set (one with `OTHER`) does not list becomes `OTHER`
with the value as its `PHRASE`. A `RELA` or role descriptor that is a role
name (`friend` → `FRIEND`) maps to it directly; one that is a role's word
(`neighbor`, `godparent`, `witness`) maps to that role with the word kept as
`PHRASE` (`NGHBR` + `PHRASE neighbor`), as the gedcom7code `rela_1` test pair
does; anything else is `OTHER` + `PHRASE`. A 5.x age is rewritten in 7.0
syntax: a bare count is years, units are lowercased and ordered, and the bound
is followed by a space.

To 5.x, a 7.0 value takes the 5.x spelling. `OTHER` + `PHRASE` in a payload
that admits free text (`NAME.TYPE`, `MEDI`) becomes the phrase; a 7.0 value
5.x does not define but whose payload admits free text (`NAME.TYPE
PROFESSIONAL`) is kept as that text. `ASSO.ROLE` becomes `RELA`, written as
its `PHRASE` when it has one and the role's word (`WITN` → `witness`)
otherwise; a `SOUR.EVEN.ROLE` other than `CHIL`, `HUSB`, `WIFE`, `MOTH`,
`FATH`, `SPOU` becomes a parenthesised descriptor (`CLERGY` → `(clergy)`). An
`AGE` whose `PHRASE` is `CHILD`, `INFANT` or `STILLBORN` (any case) becomes
that keyword; any other age loses the space after its bound and has its weeks
counted as days. A `PHRASE` folded in this way is removed; any other is
removed and reported as `PHRASE tags` data loss.

A valid source value with no equivalent in the target -- `SEX X` or `PEDI
OTHER` to 5.x -- is left unchanged and reported as a `Preserved` note and,
once per conversion, as `Enumerated values` data loss naming the affected
records, so `StrictDataLoss` rejects the conversion. A value the source version does not define either -- an unknown value in a
closed set, or an age that does not parse, such as a 5.x `AGE Abt 45` -- is
left unchanged and reported as a `Preserved` note only: the conversion loses
nothing, so `StrictDataLoss` does not reject it. Payloads in an extension
record or beneath an extension tag (`_FOO.AGE`) are not translated, and
records without `Tags` (built in code) are written from their entity and are
not translated either.

## Data Loss Reference

### 5.5 / 5.5.1 -> 7.0

| Feature | Reason |
|---------|--------|
| Date phrases | A 5.x date phrase with no 7.0 form (`ABT 1850 (circa)`, or one where no `PHRASE` can go) is left in the value, so the output is not valid 7.0 for it; each is also a `Preserved` note. See [Date epochs and phrases](#date-epochs-and-phrases) |
| Dual-year dates | 7.0 has no dual-year syntax (`11 FEB 1731/32`); the date is left unchanged, so the output is not valid 7.0 for it. Each such date is also a `Preserved` note at its path, and `AffectedRecords` lists each record holding one. Dual years inside a date phrase (`(born 1799/00)`) are not flagged |
| `SUBN records` | 7.0 has no submission record. Every `0 @S1@ SUBN` record and the `HEAD.SUBN` pointer are removed before any other step; `AffectedRecords` lists the removed records, and each record (`Submission @S1@`) and the pointer (`Header > SUBN`) is also a `Dropped` note. The submitter (`SUBM`) records stay |
| `HEAD.FILE structures` (and any other 5.5.1 structure 7.0 does not define) | Removed with its subordinates; see [Version-foreign structures](#version-foreign-structures) |
| `FONE tags`, `ROMN tags` | 5.5.1 phonetic and romanized variations (under `NAME` or `PLAC`); 7.0 replaced them with `TRAN`, which names a language rather than the `TYPE` scheme, so they are not mapped. See [Name variations](#name-variations-fone-romn) |
| Enumerated values | Not data loss: every valid 5.x value has a 7.0 equivalent. A value 5.x does not define either (an unknown `SEX` or `FAMC.STAT` value, an `AGE` that does not parse) is kept unchanged, is not valid 7.0, and is a `Preserved` note only. See [Enumeration Mappings](#enumeration-mappings) |

The `SNOTE` records this direction produces become `NOTE` records again when
converted back down to 5.x (see below).

### 7.0 -> 5.5.1 / 5.5

| Feature | Reason |
|---------|--------|
| NO tags | Negative assertions not supported; removed from the output (the `NO` line and its subordinates, and the negated event from `Individual.Events` / `Family.Events`) |
| EXID tags | External identifiers not supported |
| TRAN tags | Translation records not supported |
| PHRASE tags | Phrase annotations not supported. A date `PHRASE` is folded into the value instead where 5.x has a form for it (see [Date epochs and phrases](#date-epochs-and-phrases)) |
| CREA tags | Creation date not supported |
| `@VOID@` null pointers | 5.x has no null pointer; a kept `@VOID@` would dangle |
| `<parent>.<tag> structures` (`MARR.SDATE`, `INDI.INIL`, `NOTE.LANG`, ...) | To 5.5.1: a structure 5.5.1 does not define, removed with its subordinates; see [Version-foreign structures](#version-foreign-structures) |
| Enumerated values | A valid 7.0 value with no 5.x equivalent (`SEX X`, `PEDI OTHER`) is kept unchanged and is not valid 5.x; each is a `Preserved` note. One 7.0 does not define either (an age that does not parse) is a `Preserved` note only, not data loss. See [Enumeration Mappings](#enumeration-mappings) |

Each `EXID`, `TRAN`, `PHRASE` and `CREA` is removed from the record's
`Record.Tags` with its subordinate lines, at any level, so the output matches
the report; a match nested inside a removed block is covered by the outer
entry. The typed entity's record-level fields for them (`CreationDate`,
`ExternalIDs`, and a shared note's `Translations`) are cleared too, because
the encoder writes a record without `Tags` from its entity; a record built in
code without `Tags` is reported from those fields. Nested entity fields are
not cleared, so a record built in code can still write them. A FamilySearch
ARK `EXID` mapped to `_FSFTID` (`MapEXIDToVendorTags`) is not data loss, and
neither is `UID`, which is renamed to the vendor tag `_UID`
(`UID_TO_VENDOR_TAG`) and written as such by the encoder for a 5.x target.

Each `@VOID@` pointer is removed with the structure that holds it: a tag whose
value is `@VOID@` (`1 FAMC @VOID@`, `2 SOUR @VOID@`, ...) goes with its
subordinate lines, and the matching typed structure (a `FamilyLink`,
citation, media link, association, repository link or note pointer) is
removed from the entity; `HUSB`, `WIFE`, an LDS `FAMC` and the header `SUBM`
are cleared. Each is a `Dropped` note, and the conversion reports one
`@VOID@ null pointers` data loss entry, so `Subset` and the validator never
meet a dangling `@VOID@` in the output.

Shared notes are not lost: `0 @N1@ SNOTE` records become `0 @N1@ NOTE`
records (the typed `SharedNote` becomes a `Note`) and every `SNOTE` tag becomes
`NOTE`, reported as a `Normalized` note per record and the
`SNOTE_RECORD_TO_NOTE` and `SNOTE_POINTER_TO_NOTE` transformations. A shared
note's level-1 `MIME` and `LANG`, which a 5.x `NOTE` record does not define,
are removed and reported as `Shared note MIME` and `Shared note LANG` data
loss.

Only the converter's 7.0 downgrade removes negative assertions. The encoder
writes a negated event as `1 NO <EVENT>` whatever the header version, so a
document built in code with `IsNegative` events and a 5.5/5.5.1 header, or a
5.5.1 -> 5.5 conversion of a document that carries `NO`, still produces `NO`
lines. For a document built in code, set `Events` to `OccurredEvents()` on
each individual and family before encoding for a 5.x target; a decoded 5.x
document encodes from `Record.Tags`, so remove its level-1 `NO` tags there.

### 5.5.1 -> 5.5

| Feature | Reason |
|---------|--------|
| EMAIL | Tag introduced in 5.5.1 |
| FAX | Tag introduced in 5.5.1 |
| WWW | Tag introduced in 5.5.1 |
| FACT | Generic fact tag introduced in 5.5.1 |
| MAP, LATI, LONG | Geographic coordinates introduced in 5.5.1 |
| `FONE tags`, `ROMN tags` | Phonetic and romanized variations introduced in 5.5.1; removed (see below) |

Note: The tags other than `FONE` and `ROMN` are preserved but may not be
recognized by strict 5.5 parsers.

### Name variations (FONE, ROMN)

On a conversion to 7.0 or to 5.5, every `FONE` and `ROMN` is removed from
`Record.Tags` with its subordinate lines, at any level (under `NAME` or
`PLAC`), and each individual's typed `PersonalName.Phonetic` and `Romanized`
are cleared, because the encoder writes a record without `Tags` from its
entity. Each tag is a `FONE tags` / `ROMN tags` data loss entry listing the
affected records, with a `Dropped` note per record; a record built in code
without `Tags` is reported from those fields. This covers a nonconforming 5.5
source on a conversion to 7.0, and a nonconforming 7.0 source on a conversion
to 5.5, as well as 5.5.1. They are removed before the text and pointer steps,
so no other report entry describes a line inside a removed block. A conversion
to 5.5.1 keeps them.

## ConversionReport Structure

The `ConversionReport` provides detailed information about what changed during conversion:

```go
type ConversionReport struct {
    SourceVersion    Version          // Original GEDCOM version
    TargetVersion    Version          // Converted GEDCOM version
    Transformations  []Transformation // All changes made
    DataLoss         []DataLossItem   // Features lost (typically downgrades)
    ValidationIssues []string         // Problems found after conversion
    Success          bool             // Whether conversion completed successfully
}
```

### Transformation

```go
type Transformation struct {
    Type        string   // Short identifier (e.g., "XREF_UPPERCASE")
    Description string   // Human-readable explanation
    Count       int      // Number of instances transformed
    Details     []string // Optional specific information
}
```

### DataLossItem

```go
type DataLossItem struct {
    Feature         string   // Name of lost feature
    Reason          string   // Why the feature was lost
    AffectedRecords []string // XRefs of affected records
}
```

### Report Methods

| Method | Description |
|--------|-------------|
| `HasDataLoss()` | Returns true if any data was lost |
| `String()` | Human-readable summary of the conversion |

Example output from `report.String()`:

```
Conversion: 5.5 -> 7.0
Success: true
Transformations: 2
  - CONC_REMOVED: Consolidated CONC continuation tags into parent values (15 instances)
  - XREF_UPPERCASE: Normalized XRefs to uppercase letters, digits and underscore (required for GEDCOM 7.0) (8 instances)
```

## Strict Mode

When `StrictDataLoss` is enabled, the conversion fails if any data would be lost:

```go
opts := converter.DefaultOptions()
opts.StrictDataLoss = true
converted, report, err := converter.ConvertWithOptions(doc, gedcom.Version55, opts)
if errors.Is(err, converter.ErrDataLoss) {
    // Refused: report.DataLoss lists what would have been lost
}
```

## Related Documentation

- [GEDCOM Versions](../reference/gedcom-versions.md) - Detailed version specifications
- [Encoding Implementation](../archive/encoding-implementation-plan.md) - Character encoding details (archived)
