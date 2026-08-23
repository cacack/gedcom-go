// Package encoder provides functionality to write GEDCOM documents to files.
//
// The encoder package converts structured gedcom.Document objects back into
// the GEDCOM file format. It supports customizable line endings and ensures
// proper GEDCOM structure is maintained.
//
// # Basic Usage
//
//	doc := &gedcom.Document{
//	    Header: &gedcom.Header{Version: "5.5", Encoding: "UTF-8"},
//	    Records: records,
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
//   - DisableLineWrap     — disable CONC splitting entirely
//   - TargetVersion       — override the document's GEDCOM version in output
//   - PreserveUnknownTags — true (default) keeps custom _UNDERSCORE tags. False
//     drops each one with its whole subtree, and drops a record whose own type
//     is a custom tag ("0 _EVDEF") entirely rather than leaving a stub.
//
// Example with CRLF line endings:
//
//	opts := &encoder.EncodeOptions{LineEnding: "\r\n"}
//	if err := encoder.EncodeWithOptions(f, doc, opts); err != nil {
//	    log.Fatal(err)
//	}
//
// # How the Header Is Written
//
// The header has two write paths, chosen by whether [gedcom.Header.Tags] holds
// anything.
//
// A decoded document carries every header sub-tag in Tags, and those tags are
// what gets written — so SCHMA, SUBM, DEST, COPR, the SOUR subtree and header
// notes all survive a round-trip. The typed scalar fields (Version, Encoding,
// SourceSystem, Language) are NOT consulted on this path. Setting
// doc.Header.SourceSystem on a decoded document and encoding it does not change
// the output; edit the corresponding entry in doc.Header.Tags instead. This is
// the trade the library makes for a lossless header: what was read is what is
// written.
//
// A document assembled in memory has no tags, so its header is built from those
// typed fields instead.
//
// Two values are the encoder's rather than the document's, on both paths:
//
//   - CHAR always declares UTF-8, because Encode always writes UTF-8 and
//     converts nothing on the way out. A header echoing a source charset it no
//     longer contains cannot be re-decoded.
//   - GEDC.VERS follows TargetVersion when that option is set.
//
// Version conversion belongs to the converter package, which updates the raw
// header tags alongside the typed fields for exactly this reason.
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
// nil-free; callers that need to know should check before encoding. The
// library-wide policy, and why a nil is skipped rather than reported, is in
// docs/decisions/0007-error-transparency.md.
package encoder
