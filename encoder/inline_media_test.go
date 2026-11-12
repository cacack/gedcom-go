package encoder

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// Issue #733: an inline media link (Files, no MediaXRef) is written in the
// layout of the output version, and promoted to an OBJE record for 7.0.

func inlinePhotoLink() *gedcom.MediaLink {
	return &gedcom.MediaLink{
		Title: "Pic",
		Files: []*gedcom.MediaFile{{FileRef: `C:\photos\a.jpg`, Form: "jpg", MediaType: "photo"}},
	}
}

func TestInlineMediaLinkWrittenForOutputVersion(t *testing.T) {
	tests := []struct {
		version gedcom.Version
		want    string
	}{
		{gedcom.Version551, "1 OBJE\n2 FILE C:\\photos\\a.jpg\n3 FORM jpg\n4 MEDI photo\n2 TITL Pic\n"},
		{gedcom.Version55, "1 OBJE\n2 FORM jpg\n2 FILE C:\\photos\\a.jpg\n2 TITL Pic\n"},
		{"", "1 OBJE\n2 FILE C:\\photos\\a.jpg\n3 FORM jpg\n4 MEDI photo\n2 TITL Pic\n"},
		{gedcom.Version70, "1 OBJE @INLINE_OBJE1@\n2 TITL Pic\n0 @INLINE_OBJE1@ OBJE\n" +
			"1 FILE C:\\photos\\a.jpg\n2 FORM jpg\n3 MEDI photo\n0 TRLR\n"},
	}
	for _, tt := range tests {
		t.Run(string(tt.version), func(t *testing.T) {
			out := encodeIndividual(t, tt.version, &gedcom.Individual{Media: []*gedcom.MediaLink{inlinePhotoLink()}})
			if !strings.Contains(out, tt.want) {
				t.Errorf("want\n%s\ngot:\n%s", tt.want, out)
			}
		})
	}
}

// TestInlineMediaLinkPromotion70 checks the promoted record identifiers: one
// record per link, skipping identifiers the document uses, and the same
// record for a link written twice. The streaming encoder writes the same.
func TestInlineMediaLinkPromotion70(t *testing.T) {
	shared := inlinePhotoLink()
	other := &gedcom.MediaLink{Files: []*gedcom.MediaFile{{FileRef: "b.png", Form: "image/png"}}}
	doc := &gedcom.Document{
		Header: &gedcom.Header{Version: gedcom.Version70},
		Records: []*gedcom.Record{
			{Type: gedcom.RecordTypeIndividual, XRef: "@I1@", Entity: &gedcom.Individual{
				XRef:  "@I1@",
				Media: []*gedcom.MediaLink{shared, other},
				Events: []*gedcom.Event{{
					Type:  gedcom.EventBirth,
					Media: []*gedcom.MediaLink{shared},
				}},
			}},
			{Type: gedcom.RecordTypeMedia, XRef: "@INLINE_OBJE1@", Entity: &gedcom.MediaObject{
				Files: []*gedcom.MediaFile{{FileRef: "taken.jpg", Form: "image/jpeg"}},
			}},
		},
	}
	out := encodeToString(t, doc)
	for _, want := range []string{
		"1 OBJE @INLINE_OBJE2@\n2 TITL Pic\n1 OBJE @INLINE_OBJE3@\n",
		"2 OBJE @INLINE_OBJE2@\n",
		"0 @INLINE_OBJE1@ OBJE\n1 FILE taken.jpg\n",
		"0 @INLINE_OBJE2@ OBJE\n1 FILE C:\\photos\\a.jpg\n2 FORM jpg\n3 MEDI photo\n" +
			"0 @INLINE_OBJE3@ OBJE\n1 FILE b.png\n2 FORM image/png\n0 TRLR\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("want\n%s\ngot:\n%s", want, out)
		}
	}
	if n := strings.Count(out, "0 @INLINE_OBJE2@ OBJE"); n != 1 {
		t.Errorf("@INLINE_OBJE2@ written %d times, want 1", n)
	}

	var streamed bytes.Buffer
	if err := EncodeStreaming(&streamed, doc); err != nil {
		t.Fatalf("EncodeStreaming() error = %v", err)
	}
	if streamed.String() != out {
		t.Errorf("EncodeStreaming wrote\n%s\nEncode wrote\n%s", streamed.String(), out)
	}

	// Encoding writes the promotion; it never edits the caller's links.
	if shared.MediaXRef != "" || other.MediaXRef != "" {
		t.Errorf("encoding set MediaXRef: %q, %q", shared.MediaXRef, other.MediaXRef)
	}
}

// TestInlineMediaLinkBareStreamEncoder70: a bare StreamEncoder has no document
// to promote against, so it writes the link inline, losing nothing.
func TestInlineMediaLinkBareStreamEncoder70(t *testing.T) {
	var buf bytes.Buffer
	enc := NewStreamEncoder(&buf)
	if err := enc.WriteHeader(&gedcom.Header{Version: gedcom.Version70}); err != nil {
		t.Fatal(err)
	}
	if err := enc.WriteRecord(&gedcom.Record{Type: gedcom.RecordTypeIndividual, XRef: "@I1@",
		Entity: &gedcom.Individual{XRef: "@I1@", Media: []*gedcom.MediaLink{inlinePhotoLink()}}}); err != nil {
		t.Fatal(err)
	}
	if err := enc.WriteTrailer(); err != nil {
		t.Fatal(err)
	}
	if err := enc.Close(); err != nil {
		t.Fatal(err)
	}
	want := "1 OBJE\n2 FILE C:\\photos\\a.jpg\n3 FORM jpg\n4 MEDI photo\n2 TITL Pic\n0 TRLR\n"
	if !strings.Contains(buf.String(), want) {
		t.Errorf("want\n%s\ngot:\n%s", want, buf.String())
	}
}

// TestPointerMediaLinkIgnoresFiles: a link with a pointer is written as that
// pointer; its Files are not written beside it.
func TestPointerMediaLinkIgnoresFiles(t *testing.T) {
	link := inlinePhotoLink()
	link.MediaXRef = "@O1@"
	for _, v := range []gedcom.Version{gedcom.Version55, gedcom.Version551, gedcom.Version70} {
		out := encodeIndividual(t, v, &gedcom.Individual{Media: []*gedcom.MediaLink{link}})
		if !strings.Contains(out, "1 OBJE @O1@\n2 TITL Pic\n0 TRLR\n") {
			t.Errorf("%s: got\n%s", v, out)
		}
	}
}

// TestInlineMediaLinkRoundTrip: decode -> clear Tags -> encode -> decode keeps
// an inline link's files, in both 5.x layouts.
func TestInlineMediaLinkRoundTrip(t *testing.T) {
	for _, vers := range []string{"5.5", "5.5.1"} {
		t.Run(vers, func(t *testing.T) {
			input := "0 HEAD\n1 GEDC\n2 VERS " + vers + "\n0 @I1@ INDI\n" +
				"1 OBJE\n2 FORM jpg\n2 TITL Pic\n2 FILE a.jpg\n" +
				"1 OBJE\n2 FILE b.png\n3 FORM png\n2 TITL Other\n0 TRLR\n"
			orig := decodeGEDCOM(t, input)
			doc := decodeGEDCOM(t, input)
			clearRecordTags(doc)
			redoc := decodeGEDCOM(t, encodeToString(t, doc))

			want := orig.GetIndividual("@I1@").Media
			got := redoc.GetIndividual("@I1@").Media
			if len(want) != 2 || len(want[0].Files) != 1 || len(want[1].Files) != 1 {
				t.Fatalf("decoded links = %+v, want two with one file each", want)
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("round trip changed the links:\n got %+v\nwant %+v", got, want)
			}
		})
	}
}
