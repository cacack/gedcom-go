package decoder

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/charset"
	"github.com/cacack/gedcom-go/v3/gedcom"
	"github.com/cacack/gedcom-go/v3/parser"
	"github.com/cacack/gedcom-go/v3/version"
)

// Issue #712: DecodeRecord gives a streaming caller the typed record Decode
// would give it.

// streamDecode decodes data record by record, the way a streaming caller
// would: the declared version is read from the HEAD record, yielded first.
// ok is false when the streaming parser cannot read data, which it rejects on
// the first malformed line where Decode recovers.
func streamDecode(t *testing.T, data []byte) (records []*gedcom.Record, ok bool) {
	t.Helper()
	var ver gedcom.Version
	for raw, err := range parser.Records(charset.NewReader(strings.NewReader(string(data)))) {
		if err != nil {
			return nil, false
		}
		if raw.Type == "HEAD" && raw.XRef == "" {
			ver = version.DeclaredVersion(raw.Lines)
		}
		rec, err := DecodeRecord(raw, ver, nil)
		if err != nil {
			return nil, false
		}
		if (rec.Type == "HEAD" || rec.Type == "TRLR") && rec.XRef == "" {
			continue // Decode keeps these in Header and Trailer
		}
		records = append(records, rec)
	}
	return records, true
}

// maxComparedFileBytes leaves the multi-megabyte scale fixtures out of the
// record-by-record comparison, which deep-compares every record.
const maxComparedFileBytes = 4 << 20

func TestDecodeRecord_MatchesDecodeOnTestdata(t *testing.T) {
	paths, err := filepath.Glob("../testdata/*/*.ged")
	if err != nil {
		t.Fatal(err)
	}
	compared := 0
	for _, path := range paths {
		data, err := os.ReadFile(path) // #nosec G304 -- paths come from the repo's own testdata
		if err != nil {
			t.Fatal(err)
		}
		if len(data) > maxComparedFileBytes {
			continue
		}
		doc, err := Decode(strings.NewReader(string(data)))
		if err != nil {
			continue
		}
		if doc.Header.DeclaredVersion == "" && doc.Header.Version != gedcom.Version55 {
			// Decode guessed a version from tags across the file; a
			// record-at-a-time decode cannot, and is documented to read
			// such a file as 5.5.
			continue
		}
		got, ok := streamDecode(t, data)
		if !ok {
			continue
		}
		t.Run(filepath.Base(path), func(t *testing.T) {
			if len(got) != len(doc.Records) {
				t.Fatalf("records = %d, Decode gives %d", len(got), len(doc.Records))
			}
			for i, want := range doc.Records {
				if !reflect.DeepEqual(got[i], want) {
					t.Errorf("record %d (%s %s, line %d) differs from Decode's:\n got %+v\nwant %+v",
						i, want.XRef, want.Type, want.LineNumber, got[i].Entity, want.Entity)
				}
			}
		})
		compared++
	}
	t.Logf("compared %d of %d testdata files record by record", compared, len(paths))
	if compared < 20 {
		t.Fatalf("compared only %d testdata files; the corpus comparison is not exercising DecodeRecord", compared)
	}
}

func rawRecord(t *testing.T, gedcomText string) *parser.RawRecord {
	t.Helper()
	for raw, err := range parser.Records(strings.NewReader(gedcomText)) {
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	t.Fatal("no record")
	return nil
}

func TestDecodeRecord_TypedIndividual(t *testing.T) {
	raw := rawRecord(t, "0 @I1@ INDI\n1 NAME John /Smith/\n1 BIRT\n2 DATE 1 JAN 1900\n1 FAMC @F1@\n")
	rec, err := DecodeRecord(raw, gedcom.Version551, nil)
	if err != nil {
		t.Fatal(err)
	}
	ind, ok := rec.GetIndividual()
	if !ok {
		t.Fatalf("Entity = %T, want *gedcom.Individual", rec.Entity)
	}
	if ind.XRef != "@I1@" || len(ind.Names) != 1 || ind.Names[0].Full != "John /Smith/" {
		t.Errorf("individual = %+v, want @I1@ named John /Smith/", ind)
	}
	if len(ind.ChildInFamilies) != 1 || ind.ChildInFamilies[0].FamilyXRef != "@F1@" {
		t.Errorf("ChildInFamilies = %+v, want one link to @F1@", ind.ChildInFamilies)
	}
	if b := ind.BirthEvent(); b == nil || b.Date != "1 JAN 1900" {
		t.Errorf("birth = %+v, want date 1 JAN 1900", b)
	}
}

func TestDecodeRecord_VersionDecidesTextUnescaping(t *testing.T) {
	raw := rawRecord(t, "0 @N1@ NOTE a @@ b\n")
	tests := []struct {
		ver  gedcom.Version
		want string
	}{
		{gedcom.Version551, "a @ b"},
		{"", "a @ b"}, // undeclared: read as 5.5
		{gedcom.Version70, "a @@ b"},
	}
	for _, tt := range tests {
		rec, err := DecodeRecord(raw, tt.ver, nil)
		if err != nil {
			t.Fatal(err)
		}
		note, ok := rec.GetNote()
		if !ok || note.Text != tt.want {
			t.Errorf("version %q: note = %+v, want text %q", tt.ver, note, tt.want)
		}
	}
}

func TestDecodeRecord_LevelJumpClampedOnACopy(t *testing.T) {
	text := "0 @I1@ INDI\n1 BIRT\n3 DATE 1 JAN 1900\n"
	raw := rawRecord(t, text)

	rec, err := DecodeRecord(raw, gedcom.Version551, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := rec.Tags[1].Level; got != 2 {
		t.Errorf("lenient: DATE level = %d, want 2 (clamped)", got)
	}
	ind, _ := rec.GetIndividual()
	if b := ind.BirthEvent(); b == nil || b.Date != "1 JAN 1900" {
		t.Errorf("lenient: birth = %+v, want the clamped DATE attached", b)
	}
	if got := raw.Lines[2].Level; got != 3 {
		t.Errorf("raw line level = %d after decode, want 3: DecodeRecord must not modify its input", got)
	}

	rec, err = DecodeRecord(raw, gedcom.Version551, &DecodeOptions{StrictMode: true})
	if err != nil {
		t.Fatal(err)
	}
	if got := rec.Tags[1].Level; got != 3 {
		t.Errorf("strict: DATE level = %d, want 3 (unclamped, as Decode in strict mode)", got)
	}
}

func TestDecodeRecord_HeadHasNoEntity(t *testing.T) {
	raw := rawRecord(t, "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n")
	rec, err := DecodeRecord(raw, gedcom.Version551, nil)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Type != "HEAD" || rec.Entity != nil || len(rec.Tags) != 2 || rec.Tags[1].Value != "5.5.1" {
		t.Errorf("HEAD record = %+v, want Type HEAD, its two Tags, no Entity", rec)
	}
}

func TestDecodeRecord_NoSubordinatesKeepsNilTags(t *testing.T) {
	rec, err := DecodeRecord(rawRecord(t, "0 @I1@ INDI\n"), "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Tags != nil {
		t.Errorf("Tags = %#v, want nil as Decode leaves it", rec.Tags)
	}
}

func TestDecodeRecord_Errors(t *testing.T) {
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	valid := &parser.RawRecord{Lines: []*parser.Line{{Level: 0, Tag: "INDI", XRef: "@I1@"}}}

	tests := []struct {
		name string
		raw  *parser.RawRecord
		opts *DecodeOptions
		want string
	}{
		{"nil record", nil, nil, "no lines"},
		{"no lines", &parser.RawRecord{}, nil, "no lines"},
		{"subordinate first line", &parser.RawRecord{Lines: []*parser.Line{{Level: 1, Tag: "NAME"}}}, nil, "not level 0"},
		{"canceled context", valid, &DecodeOptions{Context: canceled}, context.Canceled.Error()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec, err := DecodeRecord(tt.raw, gedcom.Version551, tt.opts)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("err = %v, want one containing %q", err, tt.want)
			}
			if rec != nil {
				t.Errorf("record = %+v, want nil on error", rec)
			}
		})
	}
}
