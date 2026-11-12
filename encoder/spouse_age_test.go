package encoder

import (
	"bytes"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// TestSpouseAgesEntityRoundTrip pins issue #736: decode -> entity-encode ->
// decode keeps a family event's and family attribute's HUSB/WIFE AGE (with
// the 7.0 PHRASE) and an attribute's AGE, in 5.5.1 and 7.0.
func TestSpouseAgesEntityRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name: "5.5.1",
			input: "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n2 FORM LINEAGE-LINKED\n1 CHAR UTF-8\n" +
				"0 @I1@ INDI\n1 OCCU Smith\n2 AGE 30y\n" +
				"0 @F1@ FAM\n1 MARR\n2 DATE 1 JUN 1900\n2 HUSB\n3 AGE 25y\n2 WIFE\n3 AGE 22y\n" +
				"1 NCHI 3\n2 HUSB\n3 AGE 40y\n" +
				"0 TRLR\n",
			want: []string{
				"1 OCCU Smith\n2 AGE 30y\n",
				"1 MARR\n2 HUSB\n3 AGE 25y\n2 WIFE\n3 AGE 22y\n2 DATE 1 JUN 1900\n",
				"1 NCHI 3\n2 HUSB\n3 AGE 40y\n",
			},
		},
		{
			name: "7.0",
			input: "0 HEAD\n1 GEDC\n2 VERS 7.0\n" +
				"0 @I1@ INDI\n1 OCCU Smith\n2 AGE 30y\n3 PHRASE About thirty\n" +
				"0 @F1@ FAM\n1 MARR\n2 HUSB\n3 AGE 25y\n4 PHRASE Twenty-five\n2 WIFE\n3 AGE 22y\n" +
				"1 FACT Farmers\n2 TYPE Livelihood\n2 WIFE\n3 AGE 38y\n4 PHRASE Late thirties\n" +
				"0 TRLR\n",
			want: []string{
				"1 OCCU Smith\n2 AGE 30y\n3 PHRASE About thirty\n",
				"1 MARR\n2 HUSB\n3 AGE 25y\n4 PHRASE Twenty-five\n2 WIFE\n3 AGE 22y\n",
				"1 FACT Farmers\n2 WIFE\n3 AGE 38y\n4 PHRASE Late thirties\n2 TYPE Livelihood\n",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := decoder.Decode(strings.NewReader(tt.input))
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
			for _, want := range tt.want {
				if !strings.Contains(out, want) {
					t.Errorf("output missing %q:\n%s", want, out)
				}
			}

			again, err := decoder.Decode(strings.NewReader(out))
			if err != nil {
				t.Fatalf("re-Decode() error = %v", err)
			}
			assertAges(t, doc, again)
		})
	}
}

// assertAges compares every age field of every event and attribute.
func assertAges(t *testing.T, want, got *gedcom.Document) {
	t.Helper()
	ages := func(doc *gedcom.Document) []string {
		var out []string
		add := func(xs ...string) { out = append(out, xs...) }
		for _, rec := range doc.Records {
			var events []*gedcom.Event
			var attrs []*gedcom.Attribute
			switch e := rec.Entity.(type) {
			case *gedcom.Individual:
				events, attrs = e.Events, e.Attributes
			case *gedcom.Family:
				events, attrs = e.Events, e.Attributes
			}
			for _, ev := range events {
				add(ev.Age, ev.AgePhrase, ev.HusbandAge, ev.HusbandAgePhrase, ev.WifeAge, ev.WifeAgePhrase)
			}
			for _, at := range attrs {
				add(at.Age, at.AgePhrase, at.HusbandAge, at.HusbandAgePhrase, at.WifeAge, at.WifeAgePhrase)
			}
		}
		return out
	}
	w, g := ages(want), ages(got)
	if strings.Join(w, "|") != strings.Join(g, "|") {
		t.Errorf("ages after round trip = %q, want %q", g, w)
	}
}

// A spouse age set on an individual's event has no position in any version
// and is not written; on a family's event it is.
func TestSpouseAgesOnlyUnderFamily(t *testing.T) {
	ev := func() *gedcom.Event {
		return &gedcom.Event{Type: gedcom.EventResidence, HusbandAge: "25y", WifeAge: "22y"}
	}
	indi := individualToTags(&gedcom.Individual{XRef: "@I1@", Events: []*gedcom.Event{ev()}}, nil)
	for _, tag := range indi {
		if tag.Tag == "HUSB" || tag.Tag == "WIFE" {
			t.Errorf("spouse age written under an individual's event: %+v", tag)
		}
	}
	fam := familyToTags(&gedcom.Family{XRef: "@F1@", Events: []*gedcom.Event{ev(), nil}}, nil)
	got := formatTags(fam)
	want := []string{"1 RESI", "2 HUSB", "3 AGE 25y", "2 WIFE", "3 AGE 22y"}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("familyToTags() = %q, want %q", got, want)
	}
}
