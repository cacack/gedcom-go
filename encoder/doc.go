// Package encoder provides functionality to write GEDCOM documents to files.
//
// The encoder package converts structured gedcom.Document objects back into
// the GEDCOM file format. It supports customizable line endings and ensures
// proper GEDCOM structure is maintained.
//
// # Basic Usage
//
//	doc := gedcom.NewDocument(gedcom.Version55)
//	for _, r := range records {
//	    if err := doc.AddRecord(r); err != nil {
//	        log.Fatal(err)
//	    }
//	}
//
//	f, err := os.Create("output.ged")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer f.Close()
//
//	if err := encoder.Encode(f, doc); err != nil {
//	    log.Fatal(err)
//	}
//
// # Options
//
// Use [EncodeWithOptions] together with [EncodeOptions] to customize output.
// Call [DefaultOptions] for a populated starting point.
//
//   - LineEnding          — "\n" (default) or "\r\n" (CRLF for legacy tooling)
//   - MaxLineLength       — split long values with CONC when writing from typed
//     entities (default: 248). Pre-built [gedcom.Tag] values are written verbatim.
//     A split falls between two non-space characters, never inside a UTF-8
//     character. GEDCOM 7.0 output is never split: 7.0 has no CONC.
//   - DisableLineWrap     — disable CONC splitting entirely
//   - TargetVersion       — override the document's GEDCOM version in output.
//     A record written from typed entities gets only the name variations the
//     output version defines: FONE/ROMN (PersonalName.Phonetic, Romanized)
//     only for 5.5.1 output, TRAN (Transliterations) only for 7.0 output.
//     The drop is not reported; the converter package reports data loss on a
//     conversion. Likewise a 7.0 DATE PHRASE (gedcom.Date.PhraseSubstructure
//     on an event, attribute or LDS ordinance date) is written beneath its
//     DATE for 7.0 output only; 5.5/5.5.1 output drops it without folding it
//     into the value, as the converter does. A 5.5.1 "(text)" date phrase is
//     part of the DATE value: it is never written as a PHRASE, and it is not
//     rewritten for 7.0 output either, so a typed 5.5.1 document retargeted to
//     7.0 keeps "DATE (text)", which 7.0 does not permit -- convert it with
//     the converter package for valid 7.0. An association's role is written
//     as RELA for 5.5/5.5.1 output and ROLE otherwise, and its TYPE
//     (Association.RecordType) only for 5.5 output, as INDI when unset. An
//     LDS STAT is written with its DATE (StatusDate) for 5.5.1 and 7.0
//     output, which both require it; a Status with no StatusDate is written
//     alone rather than dropped, as is every STAT in 5.5 output, and TIME
//     (StatusTime) is 7.0 only. An event's FAMC (Event.FamilyXRef) is written
//     only on a BIRT, CHR or ADOP event, and its ADOP only on an ADOP event.
//     An inline media link (gedcom.MediaLink with Files and no MediaXRef) is
//     written with FORM beside FILE for 5.5 output and FORM under FILE
//     otherwise; 7.0 has no inline link, so [EncodeWithOptions] and
//     [EncodeStreamingWithOptions] write it as a pointer to an OBJE record
//     they append before TRLR, identified "@INLINE_OBJEn@" (the first n no
//     record uses). A bare [StreamEncoder] cannot see the whole document and
//     writes it inline.
//     With no known output version (no TargetVersion and no header version)
//     none of these is dropped, and the role is ROLE.
//     Every version-specific rule (these, CONC splitting, SNOTE pointers,
//     header order) reads a GEDCOM 7 patch or minor version such as "7.0.14"
//     or "7.1" as 7.0, and a vendor spelling such as "5.51" as the version it
//     means, as the decoder reads VERS; GEDC.VERS is still written as given.
//     A value the decoder does not recognize either, such as "7.0.x", is not
//     7.0.
//   - DropUnknownTags     — false (the zero value) keeps custom _UNDERSCORE
//     tags. True drops each one with its whole subtree, and drops a record
//     whose own type is a custom tag ("0 _EVDEF") entirely rather than
//     leaving a stub.
//
// Example with CRLF line endings:
//
//	opts := &encoder.EncodeOptions{LineEnding: "\r\n"}
//	if err := encoder.EncodeWithOptions(f, doc, opts); err != nil {
//	    log.Fatal(err)
//	}
//
// # Raw Tags Are Authoritative
//
// Every record has two write paths, chosen by whether its raw tags hold
// anything. [gedcom.Record.Tags] is written verbatim when non-empty, and the
// typed Entity is consulted only for a record that has none.
//
// The consequence for a decoded document is the single most important rule
// here: editing a typed field does not change encoded output. Setting
// indi.Name().Full, fam.Attributes[0].Value or any other typed field on a
// decoded record succeeds in memory, survives a round of reads, and is silently
// absent from the file. Call [SyncTags] after the edit: it applies the change
// to Record.Tags and Header.Tags one structure at a time and keeps every raw
// tag the edit did not touch, and validator.CodeEntityTagsDiverged reports a
// document that still needs it. Otherwise edit the matching entry in
// Record.Tags, or set Record.Tags to nil to have the record rebuilt from
// Entity. Use one route per document: SyncTags makes the typed fields win, so
// it reverts an edit made to Tags alone. The converter edits Tags without
// keeping every Entity field in step, so encode and decode its output before
// making typed edits on it.
//
// A record rebuilt from its Entity is written in two parts. The typed fields
// come first, as tags; for a NOTE or SNOTE the entity's Text supplies the
// level-0 value and its CONT/CONC lines when it is non-empty, and Record.Value
// is written otherwise. Then come the extension tags in the entity's own Tags
// (Individual.Tags, Family.Tags and so on): each level-1 tag with a leading
// underscore, with its subordinates, except _UID and _FSFTID, which the typed
// model holds in fields and writes from them. This is how a hand-built record
// carries a vendor or application tag beside its typed structures. Standard
// tags are never written from an entity's Tags, because for a decoded record
// that field is the whole raw record and a standard tag the typed fields did
// not produce is usually one the caller deleted; the typed fields are the
// only source of standard structures on this path, so a decoded record whose
// Record.Tags were cleared drops the standard tags the typed model has no
// field for, as it always did.
//
// Values written from typed fields are escaped for the output version so a
// literal "@" is not read as a pointer: a leading "@" is doubled for 7.0 (and an
// unknown version), every "@" outside an escape such as "@#DJULIAN@" for 5.5
// and 5.5.1. Escaping comes before CONC wrapping, so the escaped value is what
// fits MaxLineLength. Pointer fields are written as they are, and so are raw Tags, which
// must already hold the wire form. A NOTE or SNOTE whose Record.Value already
// reads back as its entity's one-line Text is written with that Value.
//
// The header follows the same rule with its own field, described next.
//
// # How the Header Is Written
//
// The header has two write paths, chosen by whether [gedcom.Header.Tags] holds
// anything.
//
// A decoded document carries every header sub-tag in Tags, and those tags are
// what gets written — so SCHMA, SUBM, DEST, DATE (with its TIME), COPR, the
// SOUR subtree and header notes all survive a round-trip. The typed scalar
// fields (Version, Encoding, SourceSystem, Date, Language, Submitter) are NOT
// consulted on this path, and the header-substructure views (SourceVersion,
// SourceName, SourceCorporation, SourceData, Destination, Time, PlaceForm) are never
// consulted at all. Setting doc.Header.SourceSystem on a decoded
// document and encoding it does not change the output; call [SyncTags] first,
// which carries the scalar fields into Tags, or edit the corresponding entry
// in doc.Header.Tags. SyncTags does not carry the substructure views.
//
// This is the trade the library makes for a lossless header: what was read is
// what is written.
//
// A document assembled in memory has no tags, so its header is built from those
// typed fields instead.
//
// Two values are the encoder's rather than the document's, on both paths:
//
//   - CHAR always declares UTF-8, because Encode always writes UTF-8 and
//     converts nothing on the way out. A header echoing a source charset it no
//     longer contains cannot be re-decoded. A hand-built header declares it
//     even with Encoding unset, so a 5.x reader does not assume ANSEL; only
//     7.0 output, which removed CHAR, omits it unless Encoding is set.
//   - GEDC.VERS follows TargetVersion when that option is set. A header with
//     no GEDC gets one, placed where the target version's grammar puts it
//     rather than at the end; for 5.5 and 5.5.1 it carries FORM
//     LINEAGE-LINKED, as does the GEDC of a hand-built 5.x header. A GEDC the
//     header already has gets that FORM too when it lacks one, as a 7.0
//     header retargeted to 5.x without the converter does.
//
// Version conversion belongs to the converter package, which updates the raw
// header tags alongside the typed fields for exactly this reason.
//
// # Line Breaks
//
// A line break (CR, LF or CRLF) in a value is written as CONT lines, because
// that is how GEDCOM spells one. This includes the hand-built header fields
// written as values ([gedcom.Header].Version, SourceSystem, Date and
// Language). The grammar allows no CONT under HEAD.SOUR, HEAD.LANG or
// GEDC.VERS, so folding there only keeps the line break from starting a new
// record: decoding the output again fills the typed field from the first line
// alone, with the CONT kept in [gedcom.Header].Tags. Those fields hold
// single-line tokens, so a caller filling them from external strings should
// strip line breaks first.
//
// A tag name or cross-reference identifier has no CONT form, so a line break
// in [gedcom.Tag].Tag, [gedcom.Tag].XRef, [gedcom.Record].Type,
// [gedcom.Record].XRef or the [gedcom.Header].Submitter pointer -- or in a
// typed entity field that becomes a tag name, such as [gedcom.Event].Type --
// makes Encode return [ErrLineBreak] instead of writing a line a reader could
// take for a record the caller never wrote.
//
// The encoder does not write what its own decoder cannot read back, so the
// rest of a line a hand-built document controls is checked too. Encode returns
// [ErrMalformedLine] for a [gedcom.Tag] below a record or the header whose
// Level is outside 1-99 (0 is the zero value, and would open a record
// mid-record) or is 99 with a multi-line Value, for an empty tag name or one
// containing whitespace, for a [gedcom.Record].XRef that does not open with
// "@" or a [gedcom.Tag].XRef not closed by "@", for an identifier whose
// whitespace the parser would split differently (see [ErrMalformedLine]), and
// for a record typed HEAD or TRLR with no XRef. It returns [ErrInvalidUTF8] for a value, tag name
// or identifier that is not valid UTF-8, the charset the output declares.
// Each error names the record (or HEAD) and the line. A decoded document never
// trips any of them: a record identifier with no closing "@" ("@I1") or a
// spaced one with no tag after it ("0 @I 1@"), which a lenient decode
// recovers, is accepted and written back verbatim. Other
// characters are written verbatim.
//
// Lines already written when the error is found stay written, as with a write
// error, so encode into a buffer when partial output must not reach the
// destination.
//
// # Nil Values
//
// A document assembled in memory can hold nils that a decoded one never does.
// The encoder never panics on them (see ADR 0007); it writes them as nothing:
//
//   - A nil element in any slice — a record in Document.Records, a tag in
//     Record.Tags, or an entity's Names, Events, SourceCitations and so on — is
//     skipped. The surrounding record and its remaining elements encode normally.
//   - A nil entity field, and a Record.Entity holding a typed nil pointer,
//     contribute no tags.
//   - A nil Document.Header is written as an empty header, exactly as
//     &gedcom.Header{} is, so options such as TargetVersion still apply.
//   - A nil [gedcom.Document] returns [ErrNilDocument], because there is no
//     document to write at all.
//
// The skip is silent, so a clean encode is not proof that a document is
// nil-free; callers that need to know should validate before encoding, which
// reports each nil as validator.CodeNilElement. The
// library-wide policy, and why a nil is skipped rather than reported, is in
// docs/decisions/0007-error-transparency.md.
package encoder
