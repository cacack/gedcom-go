package encoder

import (
	"bytes"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v2/decoder"
	"github.com/cacack/gedcom-go/v2/gedcom"
)

// TestRecordTagsAreAuthoritativeOnEncode is the executable form of the rule
// documented on gedcom.Record.Tags and in this package's doc.go, so the
// documentation cannot drift from the code without a test failing (issue #510).
//
// The rule: Record.Tags is written verbatim when non-empty, and the typed
// Entity is consulted only for a record that has none. On a decoded document
// that makes typed-field edits invisible to the encoder -- a write that
// succeeds in memory and silently never reaches the file.
func TestRecordTagsAreAuthoritativeOnEncode(t *testing.T) {
	const input = "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n1 CHAR UTF-8\n" +
		"0 @I1@ INDI\n1 NAME John /Doe/\n1 SEX M\n0 TRLR\n"

	decode := func(t *testing.T) (*gedcom.Document, *gedcom.Individual) {
		t.Helper()
		doc, err := decoder.Decode(strings.NewReader(input))
		if err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		indi := doc.GetIndividual("@I1@")
		if indi == nil {
			t.Fatal("GetIndividual(@I1@) returned nil")
		}
		if len(indi.Names) == 0 {
			t.Fatal("decoded individual has no Names; the fixture would prove nothing")
		}
		return doc, indi
	}

	encode := func(t *testing.T, doc *gedcom.Document) string {
		t.Helper()
		var buf bytes.Buffer
		if err := Encode(&buf, doc); err != nil {
			t.Fatalf("Encode() error = %v", err)
		}
		return buf.String()
	}

	t.Run("editing a typed field does not change the output", func(t *testing.T) {
		doc, indi := decode(t)
		indi.Names[0].Full = "Jane /Roe/"

		out := encode(t, doc)
		if strings.Contains(out, "Jane /Roe/") {
			t.Errorf("typed edit reached the output; Record.Tags is meant to win:\n%s", out)
		}
		if !strings.Contains(out, "1 NAME John /Doe/") {
			t.Errorf("original raw tag missing from output:\n%s", out)
		}
	})

	t.Run("editing the raw tag does change the output", func(t *testing.T) {
		doc, _ := decode(t)
		rec := doc.GetRecord("@I1@")
		if rec == nil {
			t.Fatal("GetRecord(@I1@) returned nil")
		}
		for _, tag := range rec.Tags {
			if tag.Tag == "NAME" {
				tag.Value = "Jane /Roe/"
			}
		}

		if out := encode(t, doc); !strings.Contains(out, "1 NAME Jane /Roe/") {
			t.Errorf("raw-tag edit did not reach the output:\n%s", out)
		}
	})

	t.Run("clearing Tags rebuilds the record from Entity", func(t *testing.T) {
		doc, indi := decode(t)
		indi.Names[0].Full = "Jane /Roe/"
		rec := doc.GetRecord("@I1@")
		if rec == nil {
			t.Fatal("GetRecord(@I1@) returned nil")
		}
		rec.Tags = nil

		if out := encode(t, doc); !strings.Contains(out, "Jane /Roe/") {
			t.Errorf("clearing Tags did not hand authority back to Entity:\n%s", out)
		}
	})
}
