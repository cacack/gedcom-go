package gedcomgo

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// Issue #706: the ADDR payload and ADR1 are separate fields. Line1 used to
// hold both -- seeded from the payload, overwritten by ADR1 -- and the entity
// writer wrote Line1 as ADDR and ADR1, so a multi-line address lost every line
// but ADR1's on a typed round trip.

// collectAddresses returns every typed address in doc, keyed by where it sits.
func collectAddresses(doc *gedcom.Document) map[string]gedcom.Address {
	out := map[string]gedcom.Address{}
	add := func(key string, a *gedcom.Address) {
		if a != nil {
			out[key] = *a
		}
	}
	if doc.Header != nil && doc.Header.SourceCorporation != nil {
		add("HEAD.SOUR.CORP", doc.Header.SourceCorporation.Address)
	}
	addEvents := func(xref string, events []*gedcom.Event, attrs []*gedcom.Attribute) {
		for i, e := range events {
			add(fmt.Sprintf("%s.Events[%d]", xref, i), e.Address)
		}
		for i, a := range attrs {
			add(fmt.Sprintf("%s.Attributes[%d]", xref, i), a.Address)
		}
	}
	for _, rec := range doc.Records {
		switch e := rec.Entity.(type) {
		case *gedcom.Individual:
			addEvents(rec.XRef, e.Events, e.Attributes)
		case *gedcom.Family:
			addEvents(rec.XRef, e.Events, e.Attributes)
		case *gedcom.Repository:
			add(rec.XRef, e.Address)
		case *gedcom.Submitter:
			add(rec.XRef, e.Address)
		}
	}
	return out
}

func decodeFile(t *testing.T, path string) *gedcom.Document {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	doc, err := Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Decode %s: %v", path, err)
	}
	return doc
}

func TestAddressPayloadSeparateFromADR1(t *testing.T) {
	doc := decodeFile(t, "testdata/edge-cases/rootsmagic-2026-export.ged")
	repo := doc.GetRepository("@R1@")
	if repo == nil || repo.Address == nil {
		t.Fatalf("@R1@ address missing: %+v", repo)
	}
	want := gedcom.Address{
		Full:       "Margaret Cross Norton Building\nCapitol Complex\nSpringfield, Illinois 62756\nUSA",
		Line1:      "Margaret Cross Norton Building",
		Line2:      "Capitol Complex",
		City:       "Springfield",
		State:      "Illinois",
		PostalCode: "62756",
		Country:    "USA",
	}
	if *repo.Address != want {
		t.Errorf("@R1@ Address =\n %+v\nwant\n %+v", *repo.Address, want)
	}

	doc = decodeFile(t, "testdata/gedcom-5.5/torture-test/TGC55C.ged")
	corp := doc.Header.SourceCorporation
	if corp == nil || corp.Address == nil {
		t.Fatalf("HEAD.SOUR.CORP address missing: %+v", corp)
	}
	if got, want := corp.Address.Full, "7108 South Pine Cone Street\nSalt Lake City, UT 84121\nUSA"; got != want {
		t.Errorf("CORP Address.Full = %q, want %q", got, want)
	}
	if got, want := corp.Address.Line1, "RSAC Software"; got != want {
		t.Errorf("CORP Address.Line1 = %q, want %q", got, want)
	}
}

// TestAddressEntityRoundTripCorpus clears every record's raw tags so the
// encoder writes each address from the typed model, then requires every
// address to decode back unchanged, and the output to carry no CONT or CONC
// under an ADR1-ADR3 line.
func TestAddressEntityRoundTripCorpus(t *testing.T) {
	for _, path := range []string{
		"testdata/edge-cases/rootsmagic-2026-export.ged",
		"testdata/gedcom-5.5/torture-test/TGC55C.ged",
	} {
		t.Run(path, func(t *testing.T) {
			doc := decodeFile(t, path)
			before := collectAddresses(doc)
			if len(before) == 0 {
				t.Fatal("no addresses decoded; the fixture no longer exercises this test")
			}
			for _, rec := range doc.Records {
				if rec.Entity != nil {
					rec.Tags = nil
				}
			}
			var buf bytes.Buffer
			if err := Encode(&buf, doc); err != nil {
				t.Fatalf("Encode: %v", err)
			}
			assertNoContinuationUnderADRn(t, buf.String())

			doc2, err := Decode(bytes.NewReader(buf.Bytes()))
			if err != nil {
				t.Fatalf("re-Decode: %v", err)
			}
			after := collectAddresses(doc2)
			keys := make([]string, 0, len(before))
			for k := range before {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				if got, ok := after[k]; !ok {
					t.Errorf("%s: address lost on round trip", k)
				} else if !reflect.DeepEqual(got, before[k]) {
					t.Errorf("%s: address changed on round trip\n got  %+v\n want %+v", k, got, before[k])
				}
			}
		})
	}
}

func assertNoContinuationUnderADRn(t *testing.T, out string) {
	t.Helper()
	lines := strings.Split(out, "\n")
	for i := 1; i < len(lines); i++ {
		prev := strings.Fields(lines[i-1])
		cur := strings.Fields(lines[i])
		if len(prev) < 2 || len(cur) < 2 {
			continue
		}
		switch prev[1] {
		case "ADR1", "ADR2", "ADR3":
			if cur[1] == "CONT" || cur[1] == "CONC" {
				t.Errorf("line %d: %s under %s", i+1, cur[1], prev[1])
			}
		}
	}
}
