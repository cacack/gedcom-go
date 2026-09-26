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

// sourceRepositoryLinks returns each source's RepositoryLinks keyed by XRef.
func sourceRepositoryLinks(doc *gedcom.Document) map[string][]*gedcom.SourceRepositoryLink {
	out := map[string][]*gedcom.SourceRepositoryLink{}
	for _, src := range doc.Sources() {
		out[src.XRef] = src.RepositoryLinks
	}
	return out
}

// TestSourceRepositoryLinksRoundTrip is the #553 round trip over the two
// corpus files that exposed the loss, plus pre-7.0 files with CALN: decode, encode, decode again, and
// require every source's repository links to come back unchanged. It runs
// both encode paths -- the decoded Record.Tags, and the typed entities with
// the raw tags cleared -- since only the second exercises the entity writer.
func TestSourceRepositoryLinksRoundTrip(t *testing.T) {
	files := []struct {
		path      string
		source    string
		wantLinks int
		wantCALNs int
	}{
		{"../testdata/gedcom-7.0/maximal70.ged", "@S1@", 2, 11},
		{"../testdata/edge-cases/vendor-ancestris11-export.ged", "@S1@", 2, 0},
		// 5.5 / 5.5.1 files with CALN (and, in TGC551, MEDI) under REPO,
		// so the pre-7.0 files reach the entity writer too.
		{"../testdata/gedcom-5.5/555SAMPLE.GED", "@S1@", 1, 1},
		{"../testdata/gedcom-5.5/torture-test/TGC551LF.ged", "@SOURCE1@", 1, 1},
		{"../testdata/encoding/ansi-cp1252-ftm17.ged", "@S00002@", 1, 1},
	}

	for _, f := range files {
		for _, clearTags := range []bool{false, true} {
			name := f.path
			if clearTags {
				name += "/entities"
			} else {
				name += "/tags"
			}
			t.Run(name, func(t *testing.T) {
				data, err := os.ReadFile(f.path)
				if err != nil {
					t.Fatal(err)
				}
				doc, err := decoder.Decode(bytes.NewReader(data))
				if err != nil {
					t.Fatalf("Decode() error = %v", err)
				}
				want := sourceRepositoryLinks(doc)

				links := want[f.source]
				if len(links) != f.wantLinks {
					t.Fatalf("%s: len(RepositoryLinks) = %d, want %d", f.source, len(links), f.wantLinks)
				}
				calns := 0
				for _, l := range links {
					calns += len(l.CallNumbers)
				}
				if calns != f.wantCALNs {
					t.Fatalf("%s: %d call numbers across links, want %d", f.source, calns, f.wantCALNs)
				}

				if clearTags {
					for _, rec := range doc.Records {
						rec.Tags = nil
					}
				}
				var buf bytes.Buffer
				if err := Encode(&buf, doc); err != nil {
					t.Fatalf("Encode() error = %v", err)
				}
				redoc, err := decoder.Decode(strings.NewReader(buf.String()))
				if err != nil {
					t.Fatalf("re-Decode() error = %v", err)
				}
				if got := sourceRepositoryLinks(redoc); !reflect.DeepEqual(got, want) {
					t.Errorf("repository links changed across the round trip\n got: %s\nwant: %s",
						formatLinks(got[f.source]), formatLinks(want[f.source]))
				}
			})
		}
	}
}

func formatLinks(links []*gedcom.SourceRepositoryLink) string {
	var b strings.Builder
	for _, l := range links {
		b.WriteString("{" + l.XRef)
		for _, c := range l.CallNumbers {
			b.WriteString(" [" + c.Value + "|" + c.MediaType + "|" + c.MediaPhrase + "]")
		}
		b.WriteString("}")
	}
	return b.String()
}

// TestSourceToTagsWritesEveryRepositoryLink checks the entity writer's output
// for a hand-built source with two links, duplicate CALN text with distinct
// MEDI values, and a MEDI PHRASE -- including a phrase with no media type,
// which still needs a MEDI line to hang from.
func TestSourceToTagsWritesEveryRepositoryLink(t *testing.T) {
	src := &gedcom.Source{
		XRef: "@S1@",
		RepositoryLinks: []*gedcom.SourceRepositoryLink{
			{
				XRef: "@R1@",
				CallNumbers: []*gedcom.CallNumber{
					{Value: "Box 1", MediaType: "BOOK"},
					{Value: "Box 1", MediaType: "OTHER", MediaPhrase: "Pamphlet"},
					nil,
				},
				InlineNotes: []string{"First link"},
			},
			nil,
			{
				Inline:      &gedcom.InlineRepository{Name: "Parish chest"},
				CallNumbers: []*gedcom.CallNumber{{Value: "", MediaPhrase: "Loose leaves"}, {Value: "C2"}},
			},
		},
	}

	var got []string
	for _, tag := range sourceToTags(src, nil) {
		got = append(got, strings.TrimSpace(strings.Join([]string{strconv.Itoa(tag.Level), tag.Tag, tag.Value}, " ")))
	}
	want := []string{
		"1 REPO @R1@",
		"2 CALN Box 1",
		"3 MEDI BOOK",
		"2 CALN Box 1",
		"3 MEDI OTHER",
		"4 PHRASE Pamphlet",
		"2 NOTE First link",
		"1 REPO",
		"2 NAME Parish chest",
		"2 CALN",
		"3 MEDI",
		"4 PHRASE Loose leaves",
		"2 CALN C2",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("sourceToTags() =\n  %s\nwant\n  %s", strings.Join(got, "\n  "), strings.Join(want, "\n  "))
	}
}
