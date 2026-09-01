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
// because MAP hangs off PLAC, the coordinates with it. The gate is now the
// presence of either carrier.
//
// The shapes below are the ones the gate has to get right: scalar only (the
// common hand-built shape, which must keep working), PlaceDetail only, a
// PlaceDetail with no Name at all, and both carriers set (precedence). The two
// decode tests separate the raw-tag path from the entity-rebuild path, because
// only the latter reaches the gate.

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

// TestEncodeScalarOnlyStillEmitsPLAC guards against inverting the gate rather
// than widening it: a caller who sets only the legacy scalar must keep getting
// a PLAC line.
func TestEncodeScalarOnlyStillEmitsPLAC(t *testing.T) {
	doc := placeDoc(&gedcom.Event{Type: gedcom.EventBirth, Place: "Boston, MA"})

	if got := mustEncode(t, doc); !strings.Contains(got, "2 PLAC Boston, MA") {
		t.Errorf("encoded output missing the scalar's PLAC line; got:\n%s", got)
	}
}

// TestEncodeAfterSetPlaceName is the end-to-end check for a write migrated
// ahead of v3: a decoded event carries both carriers, SetPlaceName replaces the
// name, and the encoder must write the new one. A setter that touched only
// PlaceDetail would leave the scalar behind for the encoder to prefer, so the
// same call site would encode the old name under v2 and the new one under v3.
func TestEncodeAfterSetPlaceName(t *testing.T) {
	const input = "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n1 CHAR UTF-8\n" +
		"0 @I1@ INDI\n1 BIRT\n2 PLAC Boston, MA\n0 TRLR\n"

	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	doc.GetIndividual("@I1@").Events[0].SetPlaceName("Cambridge, MA")
	for _, rec := range doc.Records {
		rec.Tags = nil // force the entity path
	}

	got := mustEncode(t, doc)

	if !strings.Contains(got, "2 PLAC Cambridge, MA") {
		t.Errorf("encoded output missing the name just set; got:\n%s", got)
	}
	if strings.Contains(got, "Boston, MA") {
		t.Errorf("encoded output still carries the replaced name; got:\n%s", got)
	}
}

// TestEncodePlacePrecedence pins the documented precedence for a hand-built
// event whose two carriers disagree: the encoder writes the scalar. Decode sets
// both from the same line, so this only arises when a caller sets them apart;
// byte-fidelity for a decoded document comes from Record.Tags, not from this
// precedence.
//
// This is deliberately the opposite of the read path: Event.PlaceName prefers
// PlaceDetail.Name so that a call site stays correct once the scalar is gone.
func TestEncodePlacePrecedence(t *testing.T) {
	ev := &gedcom.Event{
		Type:        gedcom.EventBirth,
		Place:       "Scalar Place",
		PlaceDetail: &gedcom.PlaceDetail{Name: "Detail Place"},
	}

	got := mustEncode(t, placeDoc(ev))

	if !strings.Contains(got, "2 PLAC Scalar Place") {
		t.Errorf("encoder should prefer the scalar when both carriers are set; got:\n%s", got)
	}
	if strings.Contains(got, "Detail Place") {
		t.Errorf("encoder wrote both carriers; got:\n%s", got)
	}
	if n := strings.Count(got, "PLAC"); n != 1 {
		t.Errorf("PLAC appears %d times, want 1; got:\n%s", n, got)
	}

	// The read path resolves the same event the other way.
	if want := "Detail Place"; ev.PlaceName() != want {
		t.Errorf("PlaceName() = %q, want %q", ev.PlaceName(), want)
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
	if err := EncodeWithOptions(&buf, doc, &EncodeOptions{LineEnding: "\n"}); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	if got := buf.String(); got != input {
		t.Errorf("round trip not byte-identical:\ngot:\n%s\nwant:\n%s", got, input)
	}
}

// TestDecodedPlaceRebuildsFromEntity clears Record.Tags to force the typed
// rebuild, which is the only way a decoded document reaches the PLAC gate.
// Without it the place block is reconstructed from Event.Place and
// Event.PlaceDetail rather than replayed from raw tags.
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
	if err := EncodeWithOptions(&buf, doc, &EncodeOptions{LineEnding: "\n"}); err != nil {
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
	if err := EncodeWithOptions(&buf, doc, &EncodeOptions{LineEnding: "\n"}); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	for _, want := range []string{"2 PLAC", "4 LATI N42.3601", "4 LONG W71.0589"} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("rebuilt output missing %q; got:\n%s", want, buf.String())
		}
	}
}
