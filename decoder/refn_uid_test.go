package decoder

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// Issue #554: REFN and UID repeat, and the decoder used to assign rather than
// append, so only the last value survived. These tests pin that every
// occurrence is kept, in file order, with REFN.TYPE typed alongside its value.

// firstRefNumber returns the value of the first reference number, or "" when
// there is none.
func firstRefNumber(refs []gedcom.RefNumber) string {
	if len(refs) == 0 {
		return ""
	}
	return refs[0].Value
}

// firstString returns the first element of s, or "" when s is empty.
func firstString(s []string) string {
	if len(s) == 0 {
		return ""
	}
	return s[0]
}

func decodeMaximal70(t *testing.T) *gedcom.Document {
	t.Helper()
	f, err := os.Open("../testdata/gedcom-7.0/maximal70.ged")
	if err != nil {
		t.Fatalf("open maximal70.ged: %v", err)
	}
	defer f.Close()
	doc, err := Decode(f)
	if err != nil {
		t.Fatalf("Decode(maximal70.ged): %v", err)
	}
	return doc
}

func findEvent(events []*gedcom.Event, typ gedcom.EventType) *gedcom.Event {
	for _, e := range events {
		if e.Type == typ {
			return e
		}
	}
	return nil
}

func TestMaximal70RepeatedRefnAndUID(t *testing.T) {
	doc := decodeMaximal70(t)

	wantRefs := []gedcom.RefNumber{
		{Value: "1", Type: "User-generated identifier"},
		{Value: "10", Type: "User-generated identifier"},
	}

	indi := doc.GetIndividual("@I1@")
	fam := doc.GetFamily("@F1@")
	src := doc.GetSource("@S1@")
	media := doc.GetMediaObject("@O1@")
	if indi == nil || fam == nil || src == nil || media == nil {
		t.Fatalf("missing record: indi=%v fam=%v src=%v media=%v", indi != nil, fam != nil, src != nil, media != nil)
	}

	tests := []struct {
		name     string
		refs     []gedcom.RefNumber
		uids     []string
		wantUIDs []string
	}{
		{"@I1@", indi.RefNumbers, indi.UIDs, []string{
			"3d75b5eb-36e9-40b3-b79f-f088b5c18595", "cb49c361-7124-447e-b587-4c6d36e51825"}},
		{"@F1@", fam.RefNumbers, fam.UIDs, []string{
			"f096b664-5e40-40e2-bb72-c1664a46fe45", "1f76f868-8a36-449c-af0d-a29247b3ab50"}},
		{"@S1@", src.RefNumbers, src.UIDs, []string{
			"f065a3e8-5c03-4b4a-a89d-6c5e71430a8d", "9441c3f3-74df-42b4-bbc1-fed42fd7f536"}},
		{"@O1@", media.RefNumbers, media.UIDs, []string{
			"69ebdd0e-c78c-4b81-873f-dc8ac30a48b9", "79cae8c4-e673-4e4f-bc5d-13b02d931302"}},
	}
	for _, tt := range tests {
		if !reflect.DeepEqual(tt.refs, wantRefs) {
			t.Errorf("%s RefNumbers = %+v, want %+v", tt.name, tt.refs, wantRefs)
		}
		if !reflect.DeepEqual(tt.uids, tt.wantUIDs) {
			t.Errorf("%s UIDs = %v, want %v", tt.name, tt.uids, tt.wantUIDs)
		}
	}

	// Event-level pairs: FAM.MARR and INDI.DEAT each carry two UIDs.
	if marr := findEvent(fam.Events, gedcom.EventMarriage); marr == nil {
		t.Error("@F1@ has no MARR event")
	} else if want := []string{"bbcc0025-34cb-4542-8cfb-45ba201c9c2c", "9ead4205-5bad-4c05-91c1-0aecd3f5127d"}; !reflect.DeepEqual(marr.UIDs, want) {
		t.Errorf("@F1@ MARR UIDs = %v, want %v", marr.UIDs, want)
	}
	if deat := findEvent(indi.Events, gedcom.EventDeath); deat == nil {
		t.Error("@I1@ has no DEAT event")
	} else if want := []string{"82092878-6f4f-4bca-ad59-d1ae87c5e521", "daf4b8c0-4141-42c4-bec8-01d1d818dfaf"}; !reflect.DeepEqual(deat.UIDs, want) {
		t.Errorf("@I1@ DEAT UIDs = %v, want %v", deat.UIDs, want)
	}
}

func TestAttributeRepeatedUID(t *testing.T) {
	input := "0 HEAD\n1 GEDC\n2 VERS 7.0\n" +
		"0 @I1@ INDI\n1 OCCU Smith\n2 UID a\n2 UID b\n" +
		"1 REFN only\n" +
		"0 TRLR\n"
	doc, err := Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	indi := doc.GetIndividual("@I1@")
	if len(indi.Attributes) != 1 {
		t.Fatalf("len(Attributes) = %d, want 1", len(indi.Attributes))
	}
	if want := []string{"a", "b"}; !reflect.DeepEqual(indi.Attributes[0].UIDs, want) {
		t.Errorf("OCCU UIDs = %v, want %v", indi.Attributes[0].UIDs, want)
	}
	// A REFN without TYPE leaves Type empty.
	if want := []gedcom.RefNumber{{Value: "only"}}; !reflect.DeepEqual(indi.RefNumbers, want) {
		t.Errorf("RefNumbers = %+v, want %+v", indi.RefNumbers, want)
	}
	if indi.UIDs != nil {
		t.Errorf("UIDs = %v, want nil", indi.UIDs)
	}
}

// TestTortureRefnType covers REFN with a TYPE subordinate on all four typed
// record kinds in the TGC 5.5 torture file (TGC55CLF.ged is the same content
// with CRLF line endings). TestVendorCustomTagsRefnType covers a file that
// declares 5.5.1.
func TestTortureRefnType(t *testing.T) {
	f, err := os.Open("../testdata/gedcom-5.5/torture-test/TGC551LF.ged")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()
	doc, err := Decode(f)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	cases := []struct {
		name string
		got  []gedcom.RefNumber
		want gedcom.RefNumber
	}{
		{"@PERSON1@", doc.GetIndividual("@PERSON1@").RefNumbers, gedcom.RefNumber{Value: "User reference number", Type: "Type of user number"}},
		{"@FAMILY1@", doc.GetFamily("@FAMILY1@").RefNumbers, gedcom.RefNumber{Value: "User Reference Number", Type: "Type of user number"}},
		{"@SOURCE1@", doc.GetSource("@SOURCE1@").RefNumbers, gedcom.RefNumber{Value: "User Reference Number", Type: "User Reference Type"}},
		{"@M1@", doc.GetMediaObject("@M1@").RefNumbers, gedcom.RefNumber{Value: "User Reference Number", Type: "User Reference Type"}},
	}
	for _, c := range cases {
		if want := []gedcom.RefNumber{c.want}; !reflect.DeepEqual(c.got, want) {
			t.Errorf("%s RefNumbers = %+v, want %+v", c.name, c.got, want)
		}
	}
}

// TestVendorCustomTagsRefnType covers REFN+TYPE in a file that declares
// 5.5.1, with vendor custom tags under both REFN and REFN.TYPE. The custom
// tags must not disturb the typed value or type.
func TestVendorCustomTagsRefnType(t *testing.T) {
	f, err := os.Open("../testdata/edge-cases/vendor-customtags-torture.ged")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()
	doc, err := Decode(f)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if doc.Header.Version != gedcom.Version551 {
		t.Fatalf("Header.Version = %q, want %q", doc.Header.Version, gedcom.Version551)
	}

	fam := doc.GetFamily("@F7@")
	media := doc.GetMediaObject("@M1@")
	src := doc.GetSource("@S0015@")
	if fam == nil || media == nil || src == nil {
		t.Fatalf("missing record: @F7@ %v, @M1@ %v, @S0015@ %v", fam != nil, media != nil, src != nil)
	}
	cases := []struct {
		name string
		got  []gedcom.RefNumber
		want gedcom.RefNumber
	}{
		{"@F7@", fam.RefNumbers, gedcom.RefNumber{Value: "123", Type: "Some sort of number"}},
		{"@M1@", media.RefNumbers, gedcom.RefNumber{Value: "444", Type: "A three digit number"}},
		{"@S0015@", src.RefNumbers, gedcom.RefNumber{Value: "999", Type: "Some type"}},
	}
	for _, c := range cases {
		if want := []gedcom.RefNumber{c.want}; !reflect.DeepEqual(c.got, want) {
			t.Errorf("%s RefNumbers = %+v, want %+v", c.name, c.got, want)
		}
	}
}
