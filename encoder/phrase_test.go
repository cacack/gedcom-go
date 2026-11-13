package encoder

import (
	"bytes"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// TestPhraseEntityRoundTrip pins that each typed PHRASE field (issue #450) is
// written beneath its value when a record is encoded from its entity, and
// that a phrase with no value to qualify is not written at all.
func TestPhraseEntityRoundTrip(t *testing.T) {
	const input = "0 HEAD\n1 GEDC\n2 VERS 7.0\n" +
		"0 @I1@ INDI\n" +
		"1 NAME Alex /Example/\n2 TYPE OTHER\n3 PHRASE Religious name\n" +
		"1 FAMC @F1@\n2 PEDI OTHER\n3 PHRASE Raised by an aunt\n" +
		"1 ADOP\n2 AGE 3y\n3 PHRASE About three\n" +
		"2 SDATE 1900\n3 PHRASE Sorted by registration\n" +
		"1 ASSO @I2@\n2 ROLE OTHER\n3 PHRASE Village elder\n" +
		"1 OCCU Smith\n2 SDATE 1910\n3 PHRASE Sorted by census\n" +
		"0 @F1@ FAM\n" +
		"1 HUSB @I3@\n2 PHRASE Stepfather\n" +
		"1 WIFE @VOID@\n2 PHRASE Unknown mother\n" +
		"0 @O1@ OBJE\n1 FILE photo.jpg\n2 FORM image/jpeg\n3 MEDI OTHER\n4 PHRASE Tintype\n" +
		"0 TRLR\n"

	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	for _, rec := range doc.Records {
		rec.Tags = nil // force the entity path
	}
	var buf bytes.Buffer
	if err := Encode(&buf, doc); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		"2 TYPE OTHER\n3 PHRASE Religious name\n",
		"2 PEDI OTHER\n3 PHRASE Raised by an aunt\n",
		"2 AGE 3y\n3 PHRASE About three\n",
		"2 SDATE 1900\n3 PHRASE Sorted by registration\n",
		"2 ROLE OTHER\n3 PHRASE Village elder\n",
		"2 SDATE 1910\n3 PHRASE Sorted by census\n",
		"1 HUSB @I3@\n2 PHRASE Stepfather\n",
		"1 WIFE @VOID@\n2 PHRASE Unknown mother\n",
		"3 MEDI OTHER\n4 PHRASE Tintype\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

// TestPhraseWithoutValueNotWritten pins that a phrase whose value is required
// is not written without it. AgePhrase is the exception for 7.0 output (see
// TestAgePhraseWithoutValue); with no output version it is dropped too.
func TestPhraseWithoutValueNotWritten(t *testing.T) {
	fam := &gedcom.Family{XRef: "@F1@", HusbandPhrase: "orphan phrase"}
	ev := &gedcom.Event{Type: gedcom.EventBirth, AgePhrase: "orphan", SortDatePhrase: "orphan"}
	tags := append(familyToTags(fam, nil), eventToTags(ev, 1, nil)...)
	for _, tag := range tags {
		if tag.Tag == "PHRASE" || tag.Tag == "AGE" {
			t.Errorf("%s written with no value to qualify: %+v", tag.Tag, tag)
		}
	}
}

// TestAgePhraseWithoutValue pins issue #795: GEDCOM 7.0 allows an AGE with an
// empty payload qualified only by a PHRASE. Entity-encoded as 7.0 it keeps
// both lines, under an event and an attribute alike; as 5.5.1, where AGE
// requires a value, it writes neither. TestSpouseAgePhraseWithoutValue covers
// HUSB.AGE and WIFE.AGE.
func TestAgePhraseWithoutValue(t *testing.T) {
	const input = "0 HEAD\n1 GEDC\n2 VERS 7.0\n" +
		"0 @I1@ INDI\n1 CHR\n2 AGE\n3 PHRASE Zero\n1 BIRT\n2 DATE 1900\n" +
		"1 OCCU Farmer\n2 AGE\n3 PHRASE In his youth\n" +
		"0 TRLR\n"
	for _, tc := range []struct {
		target gedcom.Version
		want   []string
		absent []string
	}{
		{gedcom.Version70, []string{
			"1 CHR\n2 AGE\n3 PHRASE Zero\n1 BIRT\n",
			"1 OCCU Farmer\n2 AGE\n3 PHRASE In his youth\n",
		}, nil},
		{gedcom.Version551, []string{"1 CHR\n1 BIRT\n", "1 OCCU Farmer\n"}, []string{"AGE", "PHRASE"}},
	} {
		t.Run(string(tc.target), func(t *testing.T) {
			doc, err := decoder.Decode(strings.NewReader(input))
			if err != nil {
				t.Fatalf("Decode() error = %v", err)
			}
			for _, rec := range doc.Records {
				rec.Tags = nil // force the entity path
			}
			opts := DefaultOptions()
			opts.TargetVersion = tc.target
			var buf bytes.Buffer
			if err := EncodeWithOptions(&buf, doc, opts); err != nil {
				t.Fatalf("Encode() error = %v", err)
			}
			out := buf.String()
			for _, want := range tc.want {
				if !strings.Contains(out, want) {
					t.Errorf("output missing %q:\n%s", want, out)
				}
			}
			for _, tag := range tc.absent {
				if strings.Contains(out, " "+tag) {
					t.Errorf("output contains %s:\n%s", tag, out)
				}
			}
		})
	}
}
