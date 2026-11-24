package encoder_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/encoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// TestAddMediaLinkIsWrittenAfterSyncTags pins that a link appended through
// Record.AddMediaLink is written as an OBJE pointer under its record once
// SyncTags has run, for each record type that carries record-level links.
func TestAddMediaLinkIsWrittenAfterSyncTags(t *testing.T) {
	const input = "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n2 FORM LINEAGE-LINKED\n1 CHAR UTF-8\n" +
		"0 @I1@ INDI\n1 NAME John /Doe/\n1 FAMS @F1@\n" +
		"0 @F1@ FAM\n1 HUSB @I1@\n" +
		"0 @S1@ SOUR\n1 TITL Parish register\n" +
		"0 @O1@ OBJE\n1 FILE photo.jpg\n2 FORM jpeg\n" +
		"0 TRLR\n"

	for _, xref := range []string{"@I1@", "@F1@", "@S1@"} {
		t.Run(xref, func(t *testing.T) {
			doc, err := decoder.Decode(strings.NewReader(input))
			if err != nil {
				t.Fatalf("Decode() error = %v", err)
			}
			record := doc.GetRecord(xref)
			if record == nil {
				t.Fatalf("record %s not found", xref)
			}
			if err := record.AddMediaLink(&gedcom.MediaLink{MediaXRef: "@O1@"}); err != nil {
				t.Fatalf("AddMediaLink() error = %v", err)
			}
			if err := encoder.SyncTags(doc); err != nil {
				t.Fatalf("SyncTags() error = %v", err)
			}
			var buf bytes.Buffer
			if err := encoder.Encode(&buf, doc); err != nil {
				t.Fatalf("Encode() error = %v", err)
			}

			// The link must sit in the edited record's own block, and nowhere else.
			out := strings.ReplaceAll(buf.String(), "\r\n", "\n")
			for _, block := range strings.Split(out, "\n0 ")[1:] {
				header, _, _ := strings.Cut(block, "\n")
				has := strings.Contains(block+"\n", "\n1 OBJE @O1@\n")
				if want := strings.HasPrefix(header, xref+" "); has != want {
					t.Errorf("record %q: has 1 OBJE @O1@ = %v, want %v\n%s", header, has, want, out)
				}
			}

			// And it reads back as a typed link.
			back, err := decoder.Decode(bytes.NewReader(buf.Bytes()))
			if err != nil {
				t.Fatalf("Decode(encoded) error = %v", err)
			}
			links, ok := back.GetRecord(xref).MediaLinks()
			if !ok || len(links) != 1 || links[0].MediaXRef != "@O1@" {
				t.Errorf("after round trip MediaLinks() = %v, %v, want one link to @O1@", links, ok)
			}
		})
	}
}

// TestAddMediaLinkKeepsExistingLinks pins that appending a link to a record
// that already has OBJE lines, a pointer with a subordinate and an inline
// 5.5.1 link, writes the new pointer and leaves the existing ones as they were.
func TestAddMediaLinkKeepsExistingLinks(t *testing.T) {
	const input = "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n2 FORM LINEAGE-LINKED\n1 CHAR UTF-8\n" +
		"0 @I1@ INDI\n1 NAME John /Doe/\n" +
		"1 OBJE @O1@\n2 NOTE on the left\n" +
		"1 OBJE\n2 FILE scan.png\n3 FORM png\n2 TITL Scan\n" +
		"0 @O1@ OBJE\n1 FILE photo.jpg\n2 FORM jpeg\n" +
		"0 @O2@ OBJE\n1 FILE other.jpg\n2 FORM jpeg\n" +
		"0 TRLR\n"

	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	record := doc.GetRecord("@I1@")
	if err := record.AddMediaLink(&gedcom.MediaLink{MediaXRef: "@O2@"}); err != nil {
		t.Fatalf("AddMediaLink() error = %v", err)
	}

	// Before SyncTags the link is in the typed entity only, so it is listed
	// but not written.
	if links, _ := record.MediaLinks(); len(links) != 3 {
		t.Fatalf("MediaLinks() before sync = %d links, want 3", len(links))
	}
	var before bytes.Buffer
	if err := encoder.Encode(&before, doc); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	if strings.Contains(before.String(), "1 OBJE @O2@") {
		t.Error("link written before SyncTags; the AddMediaLink doc comment says it is not")
	}

	if err := encoder.SyncTags(doc); err != nil {
		t.Fatalf("SyncTags() error = %v", err)
	}
	var buf bytes.Buffer
	if err := encoder.Encode(&buf, doc); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	out := strings.ReplaceAll(buf.String(), "\r\n", "\n")
	want := "0 @I1@ INDI\n1 NAME John /Doe/\n" +
		"1 OBJE @O1@\n2 NOTE on the left\n" +
		"1 OBJE\n2 FILE scan.png\n3 FORM png\n2 TITL Scan\n" +
		"1 OBJE @O2@\n"
	if !strings.Contains(out, want) {
		t.Errorf("record not written as expected.\nwant block:\n%s\ngot:\n%s", want, out)
	}
}
