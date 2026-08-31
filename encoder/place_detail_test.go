package encoder

import (
	"bytes"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v2/decoder"
	"github.com/cacack/gedcom-go/v2/gedcom"
)

// Issue #505: the PLAC line used to be gated on the legacy Place scalar alone,
// so an event carrying only PlaceDetail encoded to nothing -- the name and,
// because MAP hangs off PLAC, the coordinates with it. Issue #483 then removed
// the scalar outright, leaving PlaceDetail as the sole carrier and a non-nil
// PlaceDetail as the whole gate.
//
// The shapes below are the ones the gate has to get right: a PlaceDetail with
// a name, one with no Name but subordinates, and an entirely empty one, which
// must still write a bare PLAC line. The two decode tests separate the raw-tag
// path from the entity-rebuild path, because only the latter reaches the gate.

// placeDoc wraps an individual carrying one birth event in a minimal document.
func placeDoc(ev *gedcom.Event) *gedcom.Document {
	return &gedcom.Document{
		Header: &gedcom.Header{Version: gedcom.Version551},
		Records: []*gedcom.Record{
			{
				XRef: "@I1@",
				Type: gedcom.RecordTypeIndividual,
				Entity: &gedcom.Individual{
					XRef:   "@I1@",
					Events: []*gedcom.Event{ev},
				},
			},
		},
	}
}

func TestEncodePlaceDetailOnlyEmitsPLAC(t *testing.T) {
	doc := placeDoc(&gedcom.Event{
		Type: gedcom.EventBirth,
		PlaceDetail: &gedcom.PlaceDetail{
			Name:        "Boston, Suffolk, Massachusetts, USA",
			Form:        "City, County, State, Country",
			Coordinates: &gedcom.Coordinates{Latitude: "N42.3601", Longitude: "W71.0589"},
		},
	})

	got := mustEncode(t, doc)

	for _, want := range []string{
		"2 PLAC Boston, Suffolk, Massachusetts, USA",
		"3 FORM City, County, State, Country",
		"3 MAP",
		"4 LATI N42.3601",
		"4 LONG W71.0589",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("encoded output missing %q; got:\n%s", want, got)
		}
	}
}

func TestEncodeAttributePlaceDetailOnlyEmitsPLAC(t *testing.T) {
	doc := &gedcom.Document{
		Header: &gedcom.Header{Version: gedcom.Version551},
		Records: []*gedcom.Record{
			{
				XRef: "@I1@",
				Type: gedcom.RecordTypeIndividual,
				Entity: &gedcom.Individual{
					XRef: "@I1@",
					Attributes: []*gedcom.Attribute{
						{
							Type:  "OCCU",
							Value: "Blacksmith",
							PlaceDetail: &gedcom.PlaceDetail{
								Name:        "Salem, Essex, Massachusetts, USA",
								Coordinates: &gedcom.Coordinates{Latitude: "N42.5195", Longitude: "W70.8967"},
							},
						},
					},
				},
			},
		},
	}

	got := mustEncode(t, doc)

	for _, want := range []string{
		"2 PLAC Salem, Essex, Massachusetts, USA",
		"3 MAP",
		"4 LATI N42.5195",
		"4 LONG W70.8967",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("encoded output missing %q; got:\n%s", want, got)
		}
	}
}

// TestDecodedPlaceRoundTripsViaRawTags pins the raw-tag path, which is what
// actually preserves a decoded document: decode populates Record.Tags, and
// writeRecord writes those verbatim. Note this does NOT exercise the PLAC gate
// -- entityToTags is unreachable while Tags is non-empty, so this test passes
// with the gate reverted. TestDecodedPlaceRebuildsFromEntity is the one that
// covers the gate.
func TestDecodedPlaceRoundTripsViaRawTags(t *testing.T) {
	const input = "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n1 CHAR UTF-8\n" +
		"0 @I1@ INDI\n1 BIRT\n2 PLAC Boston, Suffolk, Massachusetts, USA\n" +
		"3 FORM City, County, State, Country\n3 MAP\n4 LATI N42.3601\n4 LONG W71.0589\n" +
		"0 TRLR\n"

	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	var buf bytes.Buffer
	if err := EncodeWithOptions(&buf, doc, &EncodeOptions{}); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	if got := buf.String(); got != input {
		t.Errorf("round trip not byte-identical:\ngot:\n%s\nwant:\n%s", got, input)
	}
}

// TestDecodedPlaceRebuildsFromEntity clears Record.Tags to force the typed
// rebuild, which is the only way a decoded document reaches the PLAC gate.
// Without it the place block is replayed from raw tags rather than
// reconstructed from Event.PlaceDetail.
func TestDecodedPlaceRebuildsFromEntity(t *testing.T) {
	const input = "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n1 CHAR UTF-8\n" +
		"0 @I1@ INDI\n1 BIRT\n2 PLAC Boston, Suffolk, Massachusetts, USA\n" +
		"3 FORM City, County, State, Country\n3 MAP\n4 LATI N42.3601\n4 LONG W71.0589\n" +
		"0 TRLR\n"

	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	for _, rec := range doc.Records {
		rec.Tags = nil // force the entity path
	}

	var buf bytes.Buffer
	if err := EncodeWithOptions(&buf, doc, &EncodeOptions{}); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	got := buf.String()
	for _, want := range []string{
		"2 PLAC Boston, Suffolk, Massachusetts, USA",
		"3 FORM City, County, State, Country",
		"3 MAP",
		"4 LATI N42.3601",
		"4 LONG W71.0589",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("rebuilt output missing %q; got:\n%s", want, got)
		}
	}
	if n := strings.Count(got, "PLAC"); n != 1 {
		t.Errorf("PLAC appears %d times, want 1; got:\n%s", n, got)
	}
}

// TestEncodeNamelessPlaceDetailKeepsCoordinates covers the shape the first cut
// of this fix still dropped: a PlaceDetail carrying FORM and MAP but no Name.
// Gating on Name would discard coordinates the library itself produces -- the
// decoder turns a bare "2 PLAC" with MAP children into exactly this value.
func TestEncodeNamelessPlaceDetailKeepsCoordinates(t *testing.T) {
	doc := placeDoc(&gedcom.Event{
		Type: gedcom.EventBirth,
		PlaceDetail: &gedcom.PlaceDetail{
			Form:        "City, County, State, Country",
			Coordinates: &gedcom.Coordinates{Latitude: "N42.3601", Longitude: "W71.0589"},
		},
	})

	got := mustEncode(t, doc)

	for _, want := range []string{"2 PLAC", "3 FORM", "3 MAP", "4 LATI N42.3601", "4 LONG W71.0589"} {
		if !strings.Contains(got, want) {
			t.Errorf("encoded output missing %q; got:\n%s", want, got)
		}
	}
}

// TestNamelessPlaceDetailSurvivesEntityRebuild is the round trip for the same
// shape: a valueless PLAC line with MAP children decodes, rebuilds from the
// typed model, and comes back intact.
func TestNamelessPlaceDetailSurvivesEntityRebuild(t *testing.T) {
	const input = "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n1 CHAR UTF-8\n" +
		"0 @I1@ INDI\n1 BIRT\n2 PLAC\n3 MAP\n4 LATI N42.3601\n4 LONG W71.0589\n0 TRLR\n"

	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	ev := doc.GetIndividual("@I1@").Events[0]
	if ev.PlaceDetail == nil || ev.PlaceDetail.Coordinates == nil {
		t.Fatalf("decode lost the coordinates: %+v", ev.PlaceDetail)
	}
	for _, rec := range doc.Records {
		rec.Tags = nil // force the entity path
	}

	var buf bytes.Buffer
	if err := EncodeWithOptions(&buf, doc, &EncodeOptions{}); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	for _, want := range []string{"2 PLAC", "4 LATI N42.3601", "4 LONG W71.0589"} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("rebuilt output missing %q; got:\n%s", want, buf.String())
		}
	}
}

// TestEncodeEmptyPlaceDetailEmitsBarePLAC locks the gate to the nil check. A
// PlaceDetail with nothing in it at all still has to write a bare PLAC line:
// that is what an input line "2 PLAC" with no value decodes to, and narrowing
// the gate to detail.Name != "" would silently stop it round-tripping.
func TestEncodeEmptyPlaceDetailEmitsBarePLAC(t *testing.T) {
	doc := placeDoc(&gedcom.Event{
		Type:        gedcom.EventBirth,
		PlaceDetail: &gedcom.PlaceDetail{},
	})

	got := mustEncode(t, doc)

	if !strings.Contains(got, "\n2 PLAC\n") {
		t.Errorf("encoded output missing the bare PLAC line; got:\n%s", got)
	}
	if n := strings.Count(got, "PLAC"); n != 1 {
		t.Errorf("PLAC appears %d times, want 1; got:\n%s", n, got)
	}
}

// TestBarePlaceLineRoundTripsThroughEntity is the round trip for the same
// shape with no subordinates to carry it: "2 PLAC" decodes to a non-nil,
// entirely empty PlaceDetail and comes back byte-identical off the typed model.
func TestBarePlaceLineRoundTripsThroughEntity(t *testing.T) {
	const input = "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n1 CHAR UTF-8\n" +
		"0 @I1@ INDI\n1 BIRT\n2 PLAC\n0 TRLR\n"

	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if ev := doc.GetIndividual("@I1@").Events[0]; ev.PlaceDetail == nil {
		t.Fatal("decode dropped the valueless PLAC line")
	}
	for _, rec := range doc.Records {
		rec.Tags = nil // force the entity path
	}

	var buf bytes.Buffer
	if err := EncodeWithOptions(&buf, doc, &EncodeOptions{}); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	if got := buf.String(); got != input {
		t.Errorf("round trip not byte-identical:\ngot:\n%s\nwant:\n%s", got, input)
	}
}
