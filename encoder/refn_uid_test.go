package encoder

import (
	"bytes"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v2/decoder"
	"github.com/cacack/gedcom-go/v2/gedcom"
)

// Issue #554: REFN (with its TYPE) and UID repeat. The entity write path must
// emit every occurrence, in order, so decode -> encode -> decode returns the
// same slices.

// refnUIDSnapshot gathers every typed REFN/UID carrier in doc, keyed by a
// readable path, so two decodes can be compared field by field.
func refnUIDSnapshot(doc *gedcom.Document) map[string]any {
	snap := map[string]any{}
	eventUIDs := func(prefix string, events []*gedcom.Event) {
		for i, e := range events {
			if len(e.UIDs) > 0 {
				snap[prefix+".Events["+strconv.Itoa(i)+"]."+string(e.Type)+".UIDs"] = e.UIDs
			}
		}
	}
	for _, rec := range doc.Records {
		if v, ok := rec.GetIndividual(); ok {
			snap[v.XRef+".RefNumbers"] = v.RefNumbers
			snap[v.XRef+".UIDs"] = v.UIDs
			eventUIDs(v.XRef, v.Events)
			for i, a := range v.Attributes {
				if len(a.UIDs) > 0 {
					snap[v.XRef+".Attributes["+strconv.Itoa(i)+"].UIDs"] = a.UIDs
				}
			}
		} else if v, ok := rec.GetFamily(); ok {
			snap[v.XRef+".RefNumbers"] = v.RefNumbers
			snap[v.XRef+".UIDs"] = v.UIDs
			eventUIDs(v.XRef, v.Events)
		} else if v, ok := rec.GetSource(); ok {
			snap[v.XRef+".RefNumbers"] = v.RefNumbers
			snap[v.XRef+".UIDs"] = v.UIDs
		} else if v, ok := rec.GetMediaObject(); ok {
			snap[v.XRef+".RefNumbers"] = v.RefNumbers
			snap[v.XRef+".UIDs"] = v.UIDs
		}
	}
	return snap
}

// roundTripEntityPath decodes path, clears Record.Tags to force the typed
// rebuild, encodes, and decodes the result.
func roundTripEntityPath(t *testing.T, path string) (before, after *gedcom.Document) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	before, err = decoder.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Decode(%s): %v", path, err)
	}
	// Decode a second copy to mutate, so before keeps its tags untouched.
	work, err := decoder.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Decode(%s): %v", path, err)
	}
	for _, rec := range work.Records {
		rec.Tags = nil // force the entity path
	}
	var buf bytes.Buffer
	if err := Encode(&buf, work); err != nil {
		t.Fatalf("Encode(%s): %v", path, err)
	}
	after, err = decoder.Decode(&buf)
	if err != nil {
		t.Fatalf("re-Decode(%s): %v", path, err)
	}
	return before, after
}

func TestRefnUIDEntityRoundTrip(t *testing.T) {
	for _, path := range []string{
		"../testdata/gedcom-7.0/maximal70.ged",
		"../testdata/gedcom-5.5/torture-test/TGC551LF.ged",
		"../testdata/gedcom-5.5/pres2020.ged",
		"../testdata/edge-cases/vendor-customtags-torture.ged",
	} {
		t.Run(path, func(t *testing.T) {
			before, after := roundTripEntityPath(t, path)
			want := refnUIDSnapshot(before)
			got := refnUIDSnapshot(after)
			if !reflect.DeepEqual(got, want) {
				for k, w := range want {
					if g := got[k]; !reflect.DeepEqual(g, w) {
						t.Errorf("%s: got %+v, want %+v", k, g, w)
					}
				}
				for k, g := range got {
					if _, ok := want[k]; !ok {
						t.Errorf("%s: unexpected %+v after round trip", k, g)
					}
				}
			}
		})
	}
}

// TestRefnUIDMaximal70MultipleSurvive guards the specific v2 loss: two REFN
// and two UID per record and two UID per event must all come back.
func TestRefnUIDMaximal70MultipleSurvive(t *testing.T) {
	_, after := roundTripEntityPath(t, "../testdata/gedcom-7.0/maximal70.ged")

	wantRefs := []gedcom.RefNumber{
		{Value: "1", Type: "User-generated identifier"},
		{Value: "10", Type: "User-generated identifier"},
	}
	indi := after.GetIndividual("@I1@")
	fam := after.GetFamily("@F1@")
	src := after.GetSource("@S1@")
	media := after.GetMediaObject("@O1@")
	for name, refs := range map[string][]gedcom.RefNumber{
		"@I1@": indi.RefNumbers, "@F1@": fam.RefNumbers, "@S1@": src.RefNumbers, "@O1@": media.RefNumbers,
	} {
		if !reflect.DeepEqual(refs, wantRefs) {
			t.Errorf("%s RefNumbers = %+v, want %+v", name, refs, wantRefs)
		}
	}
	for name, uids := range map[string][]string{
		"@I1@": indi.UIDs, "@F1@": fam.UIDs, "@S1@": src.UIDs, "@O1@": media.UIDs,
	} {
		if len(uids) != 2 {
			t.Errorf("%s UIDs = %v, want 2 values", name, uids)
		}
	}
	for _, e := range fam.Events {
		if e.Type == gedcom.EventMarriage && len(e.UIDs) != 2 {
			t.Errorf("@F1@ MARR UIDs = %v, want 2 values", e.UIDs)
		}
	}
	for _, e := range indi.Events {
		if e.Type == gedcom.EventDeath && len(e.UIDs) != 2 {
			t.Errorf("@I1@ DEAT UIDs = %v, want 2 values", e.UIDs)
		}
	}
}

// TestRefnTypeVendor551Survives pins the REFN+TYPE values of a file that
// declares 5.5.1 after an entity-path round trip, so the generic snapshot
// comparison in TestRefnUIDEntityRoundTrip cannot pass on two empty results.
func TestRefnTypeVendor551Survives(t *testing.T) {
	_, after := roundTripEntityPath(t, "../testdata/edge-cases/vendor-customtags-torture.ged")

	fam := after.GetFamily("@F7@")
	media := after.GetMediaObject("@M1@")
	src := after.GetSource("@S0015@")
	if fam == nil || media == nil || src == nil {
		t.Fatalf("missing record: @F7@ %v, @M1@ %v, @S0015@ %v", fam != nil, media != nil, src != nil)
	}
	for name, c := range map[string]struct {
		got  []gedcom.RefNumber
		want gedcom.RefNumber
	}{
		"@F7@":    {fam.RefNumbers, gedcom.RefNumber{Value: "123", Type: "Some sort of number"}},
		"@M1@":    {media.RefNumbers, gedcom.RefNumber{Value: "444", Type: "A three digit number"}},
		"@S0015@": {src.RefNumbers, gedcom.RefNumber{Value: "999", Type: "Some type"}},
	} {
		if want := []gedcom.RefNumber{c.want}; !reflect.DeepEqual(c.got, want) {
			t.Errorf("%s RefNumbers = %+v, want %+v", name, c.got, want)
		}
	}
}

// TestEncodeRefnUIDHandBuilt pins the exact lines the entity path writes for a
// hand-built document: every REFN in order, TYPE only when set, every UID in
// order, at record, event and attribute level.
func TestEncodeRefnUIDHandBuilt(t *testing.T) {
	refs := []gedcom.RefNumber{{Value: "1", Type: "user"}, {Value: "10"}}
	uids := []string{"u1", "u2"}
	doc := &gedcom.Document{
		Header: &gedcom.Header{Version: gedcom.Version70},
		Records: []*gedcom.Record{
			{XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Entity: &gedcom.Individual{
				XRef:       "@I1@",
				RefNumbers: refs,
				UIDs:       uids,
				Events:     []*gedcom.Event{{Type: gedcom.EventBirth, UIDs: []string{"e1", "e2"}}},
				Attributes: []*gedcom.Attribute{{Type: "OCCU", Value: "Smith", UIDs: []string{"a1", "a2"}}},
			}},
			{XRef: "@F1@", Type: gedcom.RecordTypeFamily, Entity: &gedcom.Family{XRef: "@F1@", RefNumbers: refs, UIDs: uids}},
			{XRef: "@S1@", Type: gedcom.RecordTypeSource, Entity: &gedcom.Source{XRef: "@S1@", RefNumbers: refs, UIDs: uids}},
			{XRef: "@O1@", Type: gedcom.RecordTypeMedia, Entity: &gedcom.MediaObject{XRef: "@O1@", RefNumbers: refs, UIDs: uids}},
		},
	}

	var buf bytes.Buffer
	if err := Encode(&buf, doc); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	out := buf.String()

	recordBlock := "1 REFN 1\n2 TYPE user\n1 REFN 10\n1 UID u1\n1 UID u2\n"
	if n := strings.Count(out, recordBlock); n != 4 {
		t.Errorf("record REFN/UID block appears %d times, want 4; output:\n%s", n, out)
	}
	for _, want := range []string{
		"1 BIRT\n2 UID e1\n2 UID e2\n",
		"1 OCCU Smith\n2 UID a1\n2 UID a2\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q; got:\n%s", want, out)
		}
	}

	// And it decodes back to the same slices.
	back, err := decoder.Decode(strings.NewReader(out))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	indi := back.GetIndividual("@I1@")
	if !reflect.DeepEqual(indi.RefNumbers, refs) || !reflect.DeepEqual(indi.UIDs, uids) {
		t.Errorf("round trip: RefNumbers=%+v UIDs=%v", indi.RefNumbers, indi.UIDs)
	}
	if got := indi.Events[0].UIDs; !reflect.DeepEqual(got, []string{"e1", "e2"}) {
		t.Errorf("round trip: BIRT UIDs = %v", got)
	}
	if got := indi.Attributes[0].UIDs; !reflect.DeepEqual(got, []string{"a1", "a2"}) {
		t.Errorf("round trip: OCCU UIDs = %v", got)
	}
}
