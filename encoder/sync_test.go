package encoder

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// Issue #725: SyncTags brings Record.Tags and Header.Tags in step with typed
// edits, keeping every raw line the edit did not touch.

func decodeFile(t *testing.T, path string) *gedcom.Document {
	t.Helper()
	f, err := os.Open(path) // #nosec G304 -- path is a constant in the repo's own testdata
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	doc, err := decoder.Decode(f)
	if err != nil {
		t.Fatalf("Decode %s: %v", path, err)
	}
	return doc
}

func syncTags(t *testing.T, doc *gedcom.Document) {
	t.Helper()
	if err := SyncTags(doc); err != nil {
		t.Fatalf("SyncTags: %v", err)
	}
}

// TestSyncTagsUnmodifiedIsNoOp pins the acceptance criterion: SyncTags on a
// document nobody edited changes no line of the output, across every
// fixture in testdata, pres2020.ged among them.
func TestSyncTagsUnmodifiedIsNoOp(t *testing.T) {
	paths, err := filepath.Glob("../testdata/*/*.ged")
	if err != nil {
		t.Fatal(err)
	}
	more, _ := filepath.Glob("../testdata/*/*/*.ged")
	paths = append(paths, more...)
	if len(paths) < 10 {
		t.Fatalf("found %d fixtures; the glob no longer reaches testdata", len(paths))
	}
	for _, path := range paths {
		if strings.Contains(path, "malformed") {
			continue
		}
		t.Run(path, func(t *testing.T) {
			f, err := os.Open(path) // #nosec G304 -- path comes from the repo's own testdata
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			doc, err := decoder.Decode(f)
			if err != nil {
				t.Skipf("fixture does not decode: %v", err)
			}
			before := encodeToString(t, doc)
			if err := SyncTags(doc); err != nil {
				if errors.Is(err, ErrSyncBaseline) {
					t.Skipf("fixture does not read back: %v", err)
				}
				t.Fatalf("SyncTags: %v", err)
			}
			if after := encodeToString(t, doc); after != before {
				t.Errorf("SyncTags on an unmodified document changed the output:\n%s", firstDiff(before, after))
			}
		})
	}
}

// firstDiff returns the first line where two encodings differ, with context.
func firstDiff(a, b string) string {
	al, bl := strings.Split(a, "\n"), strings.Split(b, "\n")
	for i := 0; i < len(al) || i < len(bl); i++ {
		var x, y string
		if i < len(al) {
			x = al[i]
		}
		if i < len(bl) {
			y = bl[i]
		}
		if x != y {
			return "line " + strconv.Itoa(i+1) + ":\n  before: " + x + "\n  after:  " + y
		}
	}
	return ""
}

// TestSyncTagsCarriesTypedEdits is the issue's scenario on the two fixtures it
// was reproduced on: a name and a header SOUR edited in the typed model reach
// the output after SyncTags, and nothing else changes. Editing Full alone
// leaves Given and Surname holding the old name (the decoder filled them from
// the NAME value), so those are written as GIVN and SURN where the source had
// none; editing all three changes only the NAME structure.
func TestSyncTagsCarriesTypedEdits(t *testing.T) {
	for _, path := range []string{"../testdata/gedcom-5.5/royal92.ged", "../testdata/gedcom-5.5/pres2020.ged"} {
		for _, together := range []bool{false, true} {
			t.Run(path+"/together="+strconv.FormatBool(together), func(t *testing.T) {
				doc := decodeFile(t, path)
				before := encodeToString(t, doc)

				rec := doc.Records[0]
				for _, r := range doc.Records {
					if r.Type == gedcom.RecordTypeIndividual {
						rec = r
						break
					}
				}
				rawBefore := encodeToString(t, &gedcom.Document{Records: []*gedcom.Record{rec}})
				name := rec.Entity.(*gedcom.Individual).Names[0]
				oldName, oldGiven, oldSurname := name.Full, name.Given, name.Surname
				oldSour := doc.Header.SourceSystem
				name.Full = "Zzz /Edited/"
				if together {
					name.Given, name.Surname = "Zzz", "Edited"
				}
				doc.Header.SourceSystem = "EDITED-APP"

				if out := encodeToString(t, doc); out != before {
					t.Fatal("typed edits reached the output without SyncTags; the authority rule this test relies on changed")
				}

				syncTags(t, doc)
				after := encodeToString(t, doc)

				hasGIVN := strings.Contains(rawBefore, "\n2 GIVN ")
				hasSURN := strings.Contains(rawBefore, "\n2 SURN ")
				removed, added := lineDiff(before, after)
				wantRemoved := []string{"1 NAME " + oldName, "1 SOUR " + oldSour}
				wantAdded := []string{"1 NAME Zzz /Edited/", "1 SOUR EDITED-APP"}
				switch {
				case together && hasGIVN:
					wantRemoved = append(wantRemoved, "2 GIVN "+oldGiven)
					wantAdded = append(wantAdded, "2 GIVN Zzz")
				case together:
					wantAdded = append(wantAdded, "2 GIVN Zzz")
				case !hasGIVN:
					wantAdded = append(wantAdded, "2 GIVN "+oldGiven)
				}
				switch {
				case together && hasSURN:
					wantRemoved = append(wantRemoved, "2 SURN "+oldSurname)
					wantAdded = append(wantAdded, "2 SURN Edited")
				case together:
					wantAdded = append(wantAdded, "2 SURN Edited")
				case !hasSURN:
					wantAdded = append(wantAdded, "2 SURN "+oldSurname)
				}
				if !sameLineSet(removed, wantRemoved) || !sameLineSet(added, wantAdded) {
					t.Errorf("SyncTags removed %q and added %q, want removed %q and added %q",
						removed, added, wantRemoved, wantAdded)
				}

				// The result reads back as the typed fields.
				redoc := decodeGEDCOM(t, after)
				got := redoc.GetIndividual(rec.XRef).Names[0]
				if got.Full != name.Full || got.Given != name.Given || got.Surname != name.Surname {
					t.Errorf("name read back as %q/%q/%q, want %q/%q/%q",
						got.Full, got.Given, got.Surname, name.Full, name.Given, name.Surname)
				}
				if redoc.Header.SourceSystem != "EDITED-APP" {
					t.Errorf("header SOUR read back as %q", redoc.Header.SourceSystem)
				}
			})
		}
	}
}

// lineDiff returns the lines of a missing from b and of b missing from a,
// counted as multisets.
func lineDiff(a, b string) (removed, added []string) {
	count := make(map[string]int)
	for _, line := range strings.Split(a, "\n") {
		count[line]++
	}
	for _, line := range strings.Split(b, "\n") {
		count[line]--
	}
	for line, n := range count {
		for ; n > 0; n-- {
			removed = append(removed, line)
		}
		for ; n < 0; n++ {
			added = append(added, line)
		}
	}
	return removed, added
}

func sameLineSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	count := make(map[string]int)
	for _, line := range a {
		count[line]++
	}
	for _, line := range b {
		count[line]--
		if count[line] < 0 {
			return false
		}
	}
	return true
}

const syncHead = "0 HEAD\n1 SOUR APP\n2 VERS 1.0\n2 _BUILD 42\n1 GEDC\n2 VERS 5.5.1\n2 FORM LINEAGE-LINKED\n1 CHAR UTF-8\n"

// syncCase decodes in, edits it, runs SyncTags and compares the record lines
// of the output with want.
func syncCase(t *testing.T, in string, edit func(doc *gedcom.Document), want string) {
	t.Helper()
	doc := decodeGEDCOM(t, syncHead+in+"0 TRLR\n")
	edit(doc)
	syncTags(t, doc)
	out := encodeToString(t, doc)
	if got := strings.TrimSuffix(strings.TrimPrefix(out, syncHead), "0 TRLR\n"); got != want {
		t.Errorf("after SyncTags:\n%s\nwant:\n%s", got, want)
	}

	// A second SyncTags has nothing left to apply.
	syncTags(t, doc)
	if again := encodeToString(t, doc); again != out {
		t.Errorf("second SyncTags changed the output:\n%s", firstDiff(out, again))
	}
}

func TestSyncTagsStructures(t *testing.T) {
	const indi = "0 @I1@ INDI\n" +
		"1 NAME John /Smith/\n2 GIVN John\n2 _AKA Jack\n" +
		"1 SEX M\n" +
		"1 BIRT\n2 DATE 1 JAN 1900\n2 _FOO bar\n2 PLAC Here\n" +
		"1 _MYAPP 7\n"
	tests := []struct {
		name string
		in   string
		edit func(doc *gedcom.Document)
		want string
	}{
		{
			name: "edited value keeps the untyped tags beside it",
			in:   indi,
			edit: func(doc *gedcom.Document) {
				doc.GetIndividual("@I1@").Events[0].Date = "2 FEB 1901"
			},
			want: "0 @I1@ INDI\n1 NAME John /Smith/\n2 GIVN John\n2 _AKA Jack\n1 SEX M\n" +
				"1 BIRT\n2 DATE 2 FEB 1901\n2 _FOO bar\n2 PLAC Here\n1 _MYAPP 7\n",
		},
		{
			name: "added structure goes after its predecessor",
			in:   indi,
			edit: func(doc *gedcom.Document) {
				ind := doc.GetIndividual("@I1@")
				ind.Events = append(ind.Events, &gedcom.Event{Type: gedcom.EventDeath, Date: "1980"})
			},
			want: "0 @I1@ INDI\n1 NAME John /Smith/\n2 GIVN John\n2 _AKA Jack\n1 SEX M\n" +
				"1 BIRT\n2 DATE 1 JAN 1900\n2 _FOO bar\n2 PLAC Here\n1 DEAT\n2 DATE 1980\n1 _MYAPP 7\n",
		},
		{
			name: "removed structure stays removed with its untyped tags",
			in:   indi,
			edit: func(doc *gedcom.Document) {
				doc.GetIndividual("@I1@").Events = nil
			},
			want: "0 @I1@ INDI\n1 NAME John /Smith/\n2 GIVN John\n2 _AKA Jack\n1 SEX M\n1 _MYAPP 7\n",
		},
		{
			name: "removing one of two keeps the other's untyped tags",
			in:   "0 @I1@ INDI\n1 NAME A /One/\n2 _X 1\n1 NAME B /Two/\n2 _X 2\n",
			edit: func(doc *gedcom.Document) {
				ind := doc.GetIndividual("@I1@")
				ind.Names = ind.Names[1:]
			},
			want: "0 @I1@ INDI\n1 NAME B /Two/\n2 _X 2\n",
		},
		{
			name: "removed pointer among several",
			in:   "0 @I1@ INDI\n1 FAMS @F1@\n1 FAMS @F2@\n2 _PRIMARY Y\n1 FAMS @F3@\n",
			edit: func(doc *gedcom.Document) {
				ind := doc.GetIndividual("@I1@")
				ind.SpouseInFamilies = append(ind.SpouseInFamilies[:0:0], ind.SpouseInFamilies[1:]...)
			},
			want: "0 @I1@ INDI\n1 FAMS @F2@\n2 _PRIMARY Y\n1 FAMS @F3@\n",
		},
		{
			name: "subordinate typed field",
			in:   indi,
			edit: func(doc *gedcom.Document) {
				doc.GetIndividual("@I1@").Names[0].Given = "Jon"
			},
			want: "0 @I1@ INDI\n1 NAME John /Smith/\n2 GIVN Jon\n2 _AKA Jack\n1 SEX M\n" +
				"1 BIRT\n2 DATE 1 JAN 1900\n2 _FOO bar\n2 PLAC Here\n1 _MYAPP 7\n",
		},
		{
			name: "edited inline note beside a pointer the writer orders first",
			in:   "0 @I1@ INDI\n1 NOTE inline text\n1 NOTE @N1@\n0 @N1@ NOTE shared\n",
			edit: func(doc *gedcom.Document) {
				doc.GetIndividual("@I1@").InlineNotes[0] = "inline, edited"
			},
			want: "0 @I1@ INDI\n1 NOTE inline, edited\n1 NOTE @N1@\n0 @N1@ NOTE shared\n",
		},
		{
			name: "first structure of a record goes to the front",
			in:   "0 @I1@ INDI\n1 SEX F\n",
			edit: func(doc *gedcom.Document) {
				ind := doc.GetIndividual("@I1@")
				ind.Names = append(ind.Names, &gedcom.PersonalName{Full: "Ann /New/"})
			},
			want: "0 @I1@ INDI\n1 NAME Ann /New/\n1 SEX F\n",
		},
		{
			name: "note record text rewrites its continuation lines",
			in:   "0 @N1@ NOTE first\n1 CONC  part\n1 CONT second\n1 _KEEP x\n",
			edit: func(doc *gedcom.Document) {
				note, _ := doc.Records[0].GetNote()
				note.Text = "first part\nsecond, edited\nthird"
			},
			want: "0 @N1@ NOTE first part\n1 CONT second, edited\n1 CONT third\n1 _KEEP x\n",
		},
		{
			name: "unchanged note keeps its original split",
			in:   "0 @N1@ NOTE first\n1 CONC  part\n1 CONT second\n",
			edit: func(doc *gedcom.Document) {},
			want: "0 @N1@ NOTE first\n1 CONC  part\n1 CONT second\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			syncCase(t, tt.in, tt.edit, tt.want)
		})
	}
}

func TestSyncTagsHeader(t *testing.T) {
	doc := decodeGEDCOM(t, syncHead+"0 TRLR\n")
	doc.Header.SourceSystem = "NEWAPP"
	doc.Header.Language = "English"
	syncTags(t, doc)
	want := "0 HEAD\n1 SOUR NEWAPP\n2 VERS 1.0\n2 _BUILD 42\n1 GEDC\n2 VERS 5.5.1\n2 FORM LINEAGE-LINKED\n1 CHAR UTF-8\n1 LANG English\n0 TRLR\n"
	if got := encodeToString(t, doc); got != want {
		t.Errorf("header after SyncTags:\n%s\nwant:\n%s", got, want)
	}
}

// A 7.0 header has no CHAR, and converter.Convert sets Encoding to UTF-8 on
// one anyway; SyncTags must not add a CHAR line for it.
func TestSyncTagsHeader70IgnoresEncoding(t *testing.T) {
	in := "0 HEAD\n1 GEDC\n2 VERS 7.0\n1 SOUR APP\n0 TRLR\n"
	doc := decodeGEDCOM(t, in)
	doc.Header.Encoding = gedcom.EncodingUTF8
	syncTags(t, doc)
	if got := encodeToString(t, doc); got != in {
		t.Errorf("7.0 header after SyncTags:\n%s\nwant:\n%s", got, in)
	}
}

func TestSyncTagsLeavesUnsyncableRecords(t *testing.T) {
	built := &gedcom.Record{XRef: "@I9@", Type: gedcom.RecordTypeIndividual,
		Entity: &gedcom.Individual{XRef: "@I9@", Names: []*gedcom.PersonalName{{Full: "Built /Here/"}}}}
	rawOnly := &gedcom.Record{XRef: "@X1@", Type: "_CUSTOM", Tags: []*gedcom.Tag{{Level: 1, Tag: "_A", Value: "1"}}}
	typedNil := &gedcom.Record{XRef: "@I8@", Type: gedcom.RecordTypeIndividual, Entity: (*gedcom.Individual)(nil),
		Tags: []*gedcom.Tag{{Level: 1, Tag: "NAME", Value: "Kept /Raw/"}}}
	doc := &gedcom.Document{Records: []*gedcom.Record{nil, built, rawOnly, typedNil}}
	syncTags(t, doc)
	if len(typedNil.Tags) != 1 {
		t.Errorf("a record whose entity is a typed nil lost its tags: %+v", typedNil.Tags)
	}
	if built.Tags != nil {
		t.Errorf("a record with no Tags got %d tags; it is written from its entity already", len(built.Tags))
	}
	if len(rawOnly.Tags) != 1 || rawOnly.Tags[0].Value != "1" {
		t.Errorf("a record with no entity was changed: %+v", rawOnly.Tags)
	}
}

func TestSyncTagsErrors(t *testing.T) {
	if err := SyncTags(nil); !errors.Is(err, ErrNilDocument) {
		t.Errorf("SyncTags(nil) = %v, want ErrNilDocument", err)
	}

	t.Run("line break leaves the document unchanged", func(t *testing.T) {
		doc := decodeGEDCOM(t, syncHead+"0 @I1@ INDI\n1 NAME A /B/\n0 TRLR\n")
		doc.Individuals()[0].Names[0].Full = "C /D/"
		doc.Header.Submitter = "@S1@\n0 @X@ INDI"
		before := encodeToString(t, &gedcom.Document{Records: doc.Records})
		if err := SyncTags(doc); !errors.Is(err, ErrLineBreak) {
			t.Fatalf("SyncTags = %v, want ErrLineBreak", err)
		}
		if after := encodeToString(t, &gedcom.Document{Records: doc.Records}); after != before {
			t.Errorf("a failed SyncTags changed a record:\n%s", firstDiff(before, after))
		}
	})

	t.Run("raw tags that do not read back", func(t *testing.T) {
		// A level-0 line inside Tags reads back as a second record.
		doc := &gedcom.Document{Records: []*gedcom.Record{{
			XRef: "@I1@", Type: gedcom.RecordTypeIndividual, Entity: &gedcom.Individual{},
			Tags: []*gedcom.Tag{{Level: 1, Tag: "SEX", Value: "F"}, {Level: 0, XRef: "@I2@", Tag: "INDI"}},
		}}}
		if err := SyncTags(doc); !errors.Is(err, ErrSyncBaseline) {
			t.Errorf("SyncTags = %v, want ErrSyncBaseline", err)
		}
	})
}

// TestSyncTagsReachesFixpoint edits every fixture broadly -- each person's
// first name, last event and a new attribute, each family's last child -- and
// checks that one SyncTags leaves nothing for a second to do, so the tags
// read back as the typed fields.
func TestSyncTagsReachesFixpoint(t *testing.T) {
	paths, _ := filepath.Glob("../testdata/*/*.ged")
	more, _ := filepath.Glob("../testdata/*/*/*.ged")
	for _, path := range append(paths, more...) {
		if strings.Contains(path, "malformed") {
			continue
		}
		t.Run(path, func(t *testing.T) {
			f, err := os.Open(path) // #nosec G304 -- path comes from the repo's own testdata
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			doc, err := decoder.Decode(f)
			if err != nil {
				t.Skipf("fixture does not decode: %v", err)
			}
			for _, indi := range doc.Individuals() {
				if len(indi.Names) > 0 {
					indi.Names[0].Full = "Edited /Name/"
					indi.Names[0].Given, indi.Names[0].Surname = "Edited", "Name"
				}
				if n := len(indi.Events); n > 0 {
					indi.Events = indi.Events[:n-1]
				}
				indi.Attributes = append(indi.Attributes, &gedcom.Attribute{Type: "OCCU", Value: "Tester"})
			}
			for _, fam := range doc.Families() {
				if n := len(fam.Children); n > 0 {
					fam.Children = fam.Children[:n-1]
				}
			}
			if err := SyncTags(doc); err != nil {
				if errors.Is(err, ErrSyncBaseline) || errors.Is(err, ErrLineBreak) {
					t.Skipf("fixture cannot be synced: %v", err)
				}
				t.Fatalf("SyncTags: %v", err)
			}
			once := encodeToString(t, doc)
			syncTags(t, doc)
			if twice := encodeToString(t, doc); twice != once {
				t.Errorf("second SyncTags changed the output:\n%s", firstDiff(once, twice))
			}
		})
	}
}
