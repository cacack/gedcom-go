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
//   - PreserveUnknownTags — true (default) keeps custom _UNDERSCORE tags
//
// Example with CRLF line endings:
//
//	opts := &encoder.EncodeOptions{LineEnding: "\r\n"}
//	if err := encoder.EncodeWithOptions(f, doc, opts); err != nil {
//	    log.Fatal(err)
//	}
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
// A nil element carries no data, so skipping it loses nothing that was there —
// but it usually means the code that built the document has a bug, and the
// encoder does not report one. Callers that need to know should check for nils
// before encoding.
//
// This policy covers the encoder, not the whole library (issue #458). A
// document holding nils encodes cleanly, but the same document can still panic
// elsewhere: validator, converter and the [gedcom.Document] collection
// accessors read Document.Records without a nil check. Traversal in the gedcom
// package (Visit, Apply, Clone, subset extraction) and XRef remapping in merge
// do skip nils, so the split is per package. Do not read a clean encode as
// proof that a document is nil-free.
package encoder
