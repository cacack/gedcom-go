package encoder

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/decoder"
	"github.com/cacack/gedcom-go/gedcom"
)

func TestEncodeRoundtrip(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Smith/
0 TRLR
`

	// Decode
	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	// Encode
	var buf bytes.Buffer
	if err := Encode(&buf, doc); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	output := buf.String()
	t.Logf("Encoded output:\n%s", output)

	// Verify it contains key elements
	if !strings.Contains(output, "0 HEAD") {
		t.Error("Output should contain HEAD")
	}
	if !strings.Contains(output, "0 @I1@ INDI") {
		t.Error("Output should contain INDI record")
	}
	if !strings.Contains(output, "1 NAME John /Smith/") {
		t.Error("Output should contain NAME tag")
	}
	if !strings.Contains(output, "0 TRLR") {
		t.Error("Output should contain TRLR")
	}

	// Decode the output to verify it's valid GEDCOM
	doc2, err := decoder.Decode(strings.NewReader(output))
	if err != nil {
		t.Fatalf("Failed to decode encoded output: %v", err)
	}

	if len(doc2.Records) != len(doc.Records) {
		t.Errorf("Record count mismatch: got %d, want %d", len(doc2.Records), len(doc.Records))
	}
}

func TestEncodeCRLF(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5
0 TRLR
`

	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	opts := &EncodeOptions{
		LineEnding: "\r\n",
	}

	var buf bytes.Buffer
	if err := EncodeWithOptions(&buf, doc, opts); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "\r\n") {
		t.Error("Output should contain CRLF line endings")
	}
}

func TestEncodeWithOptionsNil(t *testing.T) {
	doc := &gedcom.Document{
		Header: &gedcom.Header{
			Version:  "5.5",
			Encoding: "UTF-8",
		},
		Records: []*gedcom.Record{},
	}

	var buf bytes.Buffer
	// Pass nil options - should use defaults
	if err := EncodeWithOptions(&buf, doc, nil); err != nil {
		t.Fatalf("EncodeWithOptions() with nil options error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "0 HEAD") {
		t.Error("Output should contain HEAD")
	}
	// Default line ending is \n
	if !strings.Contains(output, "\n") {
		t.Error("Output should contain LF line endings (default)")
	}
}

func TestEncodeHeaderFields(t *testing.T) {
	tests := []struct {
		name   string
		header *gedcom.Header
		want   []string
	}{
		{
			name: "full header",
			header: &gedcom.Header{
				Version:      "5.5.1",
				Encoding:     "UTF-8",
				SourceSystem: "MyGedcomApp",
				Language:     "English",
			},
			want: []string{
				"0 HEAD",
				"1 GEDC",
				"2 VERS 5.5.1",
				"1 CHAR UTF-8",
				"1 SOUR MyGedcomApp",
				"1 LANG English",
			},
		},
		{
			name: "minimal header",
			header: &gedcom.Header{
				Version:  "",
				Encoding: "",
			},
			want: []string{
				"0 HEAD",
			},
		},
		{
			name: "header with version only",
			header: &gedcom.Header{
				Version: "7.0",
			},
			want: []string{
				"0 HEAD",
				"1 GEDC",
				"2 VERS 7.0",
			},
		},
		{
			name: "header with encoding only",
			header: &gedcom.Header{
				Encoding: "ANSEL",
			},
			want: []string{
				"0 HEAD",
				"1 CHAR ANSEL",
			},
		},
		{
			name: "header with source only",
			header: &gedcom.Header{
				SourceSystem: "TestApp 1.0",
			},
			want: []string{
				"0 HEAD",
				"1 SOUR TestApp 1.0",
			},
		},
		{
			name: "header with language only",
			header: &gedcom.Header{
				Language: "French",
			},
			want: []string{
				"0 HEAD",
				"1 LANG French",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &gedcom.Document{
				Header:  tt.header,
				Records: []*gedcom.Record{},
			}

			var buf bytes.Buffer
			if err := Encode(&buf, doc); err != nil {
				t.Fatalf("Encode() error = %v", err)
			}

			output := buf.String()
			for _, want := range tt.want {
				if !strings.Contains(output, want) {
					t.Errorf("Output missing expected line: %q\nGot:\n%s", want, output)
				}
			}
		})
	}
}

func TestEncodeRecords(t *testing.T) {
	tests := []struct {
		name    string
		records []*gedcom.Record
		want    []string
	}{
		{
			name: "record with xref",
			records: []*gedcom.Record{
				{
					XRef: "@I1@",
					Type: gedcom.RecordTypeIndividual,
					Tags: []*gedcom.Tag{
						{Level: 1, Tag: "NAME", Value: "John /Doe/"},
					},
				},
			},
			want: []string{
				"0 @I1@ INDI",
				"1 NAME John /Doe/",
			},
		},
		{
			name: "record without xref",
			records: []*gedcom.Record{
				{
					Type: gedcom.RecordTypeNote,
					Tags: []*gedcom.Tag{
						{Level: 1, Tag: "CONT", Value: "This is a note"},
					},
				},
			},
			want: []string{
				"0 NOTE",
				"1 CONT This is a note",
			},
		},
		{
			name: "multiple records",
			records: []*gedcom.Record{
				{
					XRef: "@I1@",
					Type: gedcom.RecordTypeIndividual,
					Tags: []*gedcom.Tag{
						{Level: 1, Tag: "NAME", Value: "Jane /Smith/"},
					},
				},
				{
					XRef: "@F1@",
					Type: gedcom.RecordTypeFamily,
					Tags: []*gedcom.Tag{
						{Level: 1, Tag: "HUSB", Value: "@I1@"},
					},
				},
			},
			want: []string{
				"0 @I1@ INDI",
				"1 NAME Jane /Smith/",
				"0 @F1@ FAM",
				"1 HUSB @I1@",
			},
		},
		{
			name: "record with no tags",
			records: []*gedcom.Record{
				{
					XRef: "@S1@",
					Type: gedcom.RecordTypeSource,
					Tags: []*gedcom.Tag{},
				},
			},
			want: []string{
				"0 @S1@ SOUR",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &gedcom.Document{
				Header: &gedcom.Header{
					Version: "5.5",
				},
				Records: tt.records,
			}

			var buf bytes.Buffer
			if err := Encode(&buf, doc); err != nil {
				t.Fatalf("Encode() error = %v", err)
			}

			output := buf.String()
			for _, want := range tt.want {
				if !strings.Contains(output, want) {
					t.Errorf("Output missing expected line: %q\nGot:\n%s", want, output)
				}
			}
		})
	}
}

func TestEncodeTags(t *testing.T) {
	tests := []struct {
		name string
		tags []*gedcom.Tag
		want []string
	}{
		{
			name: "tag with value",
			tags: []*gedcom.Tag{
				{Level: 1, Tag: "NAME", Value: "Test Value"},
			},
			want: []string{
				"1 NAME Test Value",
			},
		},
		{
			name: "tag without value",
			tags: []*gedcom.Tag{
				{Level: 1, Tag: "BIRT"},
			},
			want: []string{
				"1 BIRT",
			},
		},
		{
			name: "nested tags",
			tags: []*gedcom.Tag{
				{Level: 1, Tag: "BIRT"},
				{Level: 2, Tag: "DATE", Value: "1 JAN 1900"},
				{Level: 2, Tag: "PLAC", Value: "London, England"},
			},
			want: []string{
				"1 BIRT",
				"2 DATE 1 JAN 1900",
				"2 PLAC London, England",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &gedcom.Document{
				Header: &gedcom.Header{},
				Records: []*gedcom.Record{
					{
						XRef: "@I1@",
						Type: gedcom.RecordTypeIndividual,
						Tags: tt.tags,
					},
				},
			}

			var buf bytes.Buffer
			if err := Encode(&buf, doc); err != nil {
				t.Fatalf("Encode() error = %v", err)
			}

			output := buf.String()
			for _, want := range tt.want {
				if !strings.Contains(output, want) {
					t.Errorf("Output missing expected line: %q\nGot:\n%s", want, output)
				}
			}
		})
	}
}

func TestEncodeTrailer(t *testing.T) {
	doc := &gedcom.Document{
		Header: &gedcom.Header{
			Version: "5.5",
		},
		Records: []*gedcom.Record{},
	}

	var buf bytes.Buffer
	if err := Encode(&buf, doc); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	output := buf.String()
	if !strings.HasSuffix(strings.TrimSpace(output), "0 TRLR") {
		t.Error("Output should end with TRLR")
	}
}

// failWriter is a writer that always returns an error
type failWriter struct {
	failAfter int
	count     int
}

func (w *failWriter) Write(p []byte) (n int, err error) {
	if w.count >= w.failAfter {
		return 0, errors.New("write error")
	}
	w.count++
	return len(p), nil
}

func TestEncodeWriteErrors(t *testing.T) {
	doc := &gedcom.Document{
		Header: &gedcom.Header{
			Version:      "5.5",
			Encoding:     "UTF-8",
			SourceSystem: "Test",
			Language:     "English",
		},
		Records: []*gedcom.Record{
			{
				XRef: "@I1@",
				Type: gedcom.RecordTypeIndividual,
				Tags: []*gedcom.Tag{
					{Level: 1, Tag: "NAME", Value: "Test"},
				},
			},
		},
	}

	tests := []struct {
		name      string
		failAfter int
	}{
		{"fail on header", 0},
		{"fail on version", 1},
		{"fail on version value", 2},
		{"fail on encoding", 3},
		{"fail on source", 4},
		{"fail on language", 5},
		{"fail on record", 6},
		{"fail on tag", 7},
		{"fail on trailer", 8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &failWriter{failAfter: tt.failAfter}
			err := Encode(w, doc)
			if err == nil {
				t.Error("Expected error from Encode(), got nil")
			}
		})
	}
}

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()
	if opts == nil {
		t.Fatal("DefaultOptions() returned nil")
		return // unreachable, but satisfies staticcheck
	}
	if opts.LineEnding != "\n" {
		t.Errorf("DefaultOptions().LineEnding = %q, want %q", opts.LineEnding, "\n")
	}
}

func TestEncodeComplexDocument(t *testing.T) {
	// Test a more complex document with multiple record types
	input := `0 HEAD
1 GEDC
2 VERS 5.5.1
1 CHAR UTF-8
1 SOUR FamilyTree
1 LANG English
0 @I1@ INDI
1 NAME John /Doe/
2 GIVN John
2 SURN Doe
1 SEX M
1 BIRT
2 DATE 1 JAN 1900
2 PLAC London, England
0 @I2@ INDI
1 NAME Jane /Smith/
1 SEX F
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 MARR
2 DATE 1 JUN 1920
0 @S1@ SOUR
1 TITL Birth Records
1 ABBR BR
0 TRLR
`

	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	var buf bytes.Buffer
	if err := Encode(&buf, doc); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	output := buf.String()

	// Verify all major components are present
	expectedLines := []string{
		"0 HEAD",
		"1 GEDC",
		"2 VERS 5.5.1",
		"1 CHAR UTF-8",
		"0 @I1@ INDI",
		"1 NAME John /Doe/",
		"2 GIVN John",
		"2 SURN Doe",
		"1 SEX M",
		"1 BIRT",
		"2 DATE 1 JAN 1900",
		"0 @I2@ INDI",
		"0 @F1@ FAM",
		"1 HUSB @I1@",
		"1 WIFE @I2@",
		"0 @S1@ SOUR",
		"0 TRLR",
	}

	for _, line := range expectedLines {
		if !strings.Contains(output, line) {
			t.Errorf("Output missing expected line: %q", line)
		}
	}

	// Verify roundtrip
	doc2, err := decoder.Decode(strings.NewReader(output))
	if err != nil {
		t.Fatalf("Failed to decode encoded output: %v", err)
	}

	if len(doc2.Records) != len(doc.Records) {
		t.Errorf("Record count mismatch after roundtrip: got %d, want %d", len(doc2.Records), len(doc.Records))
	}
}

// TestEncodeRoundtripNewFeatures tests round-trip encoding/decoding of new features
// added in issues #2-11: source citations, event subordinates, LDS ordinances, associations, etc.
func TestEncodeRoundtripNewFeatures(t *testing.T) {
	input := `0 HEAD
1 GEDC
2 VERS 5.5.1
1 CHAR UTF-8
0 @I1@ INDI
1 NAME Johannes Ludwig /von Beethoven/
2 GIVN Johannes Ludwig
2 SURN Beethoven
2 SPFX von
2 NICK Ludwig
2 NPFX Dr.
2 NSFX III
2 TYPE BIRTH
1 SEX M
1 BIRT
2 DATE 15 JAN 1850
2 PLAC Boston, Suffolk, MA, USA
3 FORM City, County, State, Country
3 MAP
4 LATI N42.3601
4 LONG W71.0589
2 TYPE Hospital birth
2 CAUS Natural
2 AGE 0y
2 AGNC General Hospital
2 NOTE Birth note
2 SOUR @S1@
3 PAGE p. 42
3 QUAY 3
3 DATA
4 DATE 15 JAN 1850
4 TEXT Original birth record entry
1 DEAT
2 DATE 20 MAR 1920
2 PLAC Springfield, IL
2 CAUS Heart failure
2 AGE 70y
1 BARM
2 DATE 15 MAR 1863
2 PLAC Temple Beth Israel
1 GRAD
2 DATE 1972
2 PLAC MIT, Cambridge, MA
1 NATU
2 DATE 4 JUL 1880
1 RETI
2 DATE 1 JAN 1915
1 PROB
2 DATE 1 APR 1920
1 CREM
2 DATE 25 MAR 1920
1 OCCU Software Engineer
2 DATE 2000
2 PLAC Silicon Valley, CA
2 SOUR @S1@
3 PAGE Employment records
1 CAST Brahmin
1 EDUC PhD Computer Science
1 RELI Methodist
1 ASSO @I2@
2 ROLE GODP
2 NOTE Godparent note
1 ASSO @I3@
2 ROLE WITN
1 BAPL
2 DATE 1 JAN 1860
2 TEMP SLAKE
2 STAT COMPLETED
1 CONL
2 DATE 1 FEB 1860
2 TEMP SLAKE
1 ENDL
2 DATE 1 MAR 1880
2 TEMP LOGAN
1 SLGC
2 DATE 15 APR 1861
2 TEMP SLAKE
2 FAMC @F1@
1 FAMC @F1@
2 PEDI BIRTH
1 FAMS @F2@
1 SOUR @S1@
2 PAGE Entire individual
2 QUAY 2
0 @I2@ INDI
1 NAME Jane /Smith/
1 SEX F
0 @I3@ INDI
1 NAME Bob /Witness/
1 SEX M
0 @F1@ FAM
1 HUSB @I4@
1 WIFE @I5@
1 CHIL @I1@
0 @F2@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 MARB
2 DATE 1 JAN 1875
2 PLAC Boston, MA
1 MARC
2 DATE 5 JAN 1875
1 MARL
2 DATE 8 JAN 1875
1 MARS
2 DATE 10 JAN 1875
1 MARR
2 DATE 15 JAN 1875
2 PLAC Boston, MA
2 AGNC City Hall
2 CAUS Love
2 SOUR @S1@
3 PAGE Marriage cert
3 QUAY 3
1 DIVF
2 DATE 1 JUN 1900
1 DIV
2 DATE 1 JUL 1900
1 SLGS
2 DATE 10 JUN 1875
2 TEMP SLAKE
2 STAT COMPLETED
1 SOUR @S1@
2 PAGE Family records
2 QUAY 1
0 @I4@ INDI
1 NAME Father /Beethoven/
0 @I5@ INDI
1 NAME Mother /Beethoven/
0 @S1@ SOUR
1 TITL County Records
1 AUTH John Archivist
1 PUBL County Publishing, 2000
1 TEXT Original source text
0 TRLR
`

	// Decode
	doc, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Initial Decode() error = %v", err)
	}

	// Encode
	var buf bytes.Buffer
	if err := Encode(&buf, doc); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	// Decode again
	doc2, err := decoder.Decode(strings.NewReader(buf.String()))
	if err != nil {
		t.Fatalf("Second Decode() error = %v", err)
	}

	// Verify record counts
	if len(doc2.Records) != len(doc.Records) {
		t.Errorf("Record count mismatch: got %d, want %d", len(doc2.Records), len(doc.Records))
	}

	// Verify individual @I1@
	indi := doc2.GetIndividual("@I1@")
	if indi == nil {
		t.Fatal("Individual @I1@ not found after round-trip")
	}

	// Name extensions
	if len(indi.Names) < 1 {
		t.Fatal("No names found")
	}
	name := indi.Names[0]
	if name.Nickname != "Ludwig" {
		t.Errorf("Name.Nickname = %s, want Ludwig", name.Nickname)
	}
	if name.SurnamePrefix != "von" {
		t.Errorf("Name.SurnamePrefix = %s, want von", name.SurnamePrefix)
	}

	// Events with subordinates
	var birthEvent, deathEvent *struct{ Type, Date, Cause, Age, Agency string }
	for _, ev := range indi.Events {
		switch ev.Type {
		case "BIRT":
			birthEvent = &struct{ Type, Date, Cause, Age, Agency string }{
				string(ev.Type), ev.Date, ev.Cause, ev.Age, ev.Agency,
			}
		case "DEAT":
			deathEvent = &struct{ Type, Date, Cause, Age, Agency string }{
				string(ev.Type), ev.Date, ev.Cause, ev.Age, ev.Agency,
			}
		}
	}

	if birthEvent == nil {
		t.Fatal("Birth event not found")
	}
	if birthEvent.Date != "15 JAN 1850" {
		t.Errorf("Birth.Date = %s, want '15 JAN 1850'", birthEvent.Date)
	}
	if birthEvent.Age != "0y" {
		t.Errorf("Birth.Age = %s, want '0y'", birthEvent.Age)
	}
	if birthEvent.Agency != "General Hospital" {
		t.Errorf("Birth.Agency = %s, want 'General Hospital'", birthEvent.Agency)
	}

	if deathEvent == nil {
		t.Fatal("Death event not found")
	}
	if deathEvent.Cause != "Heart failure" {
		t.Errorf("Death.Cause = %s, want 'Heart failure'", deathEvent.Cause)
	}

	// Place coordinates
	for _, ev := range indi.Events {
		if ev.Type == "BIRT" && ev.PlaceDetail != nil && ev.PlaceDetail.Coordinates != nil {
			if ev.PlaceDetail.Coordinates.Latitude != "N42.3601" {
				t.Errorf("Birth coords Lat = %s, want N42.3601", ev.PlaceDetail.Coordinates.Latitude)
			}
			if ev.PlaceDetail.Coordinates.Longitude != "W71.0589" {
				t.Errorf("Birth coords Long = %s, want W71.0589", ev.PlaceDetail.Coordinates.Longitude)
			}
		}
	}

	// Source citations on events
	for _, ev := range indi.Events {
		if ev.Type != "BIRT" || len(ev.SourceCitations) == 0 {
			continue
		}
		cite := ev.SourceCitations[0]
		if cite.Page != "p. 42" {
			t.Errorf("Birth citation Page = %s, want 'p. 42'", cite.Page)
		}
		if cite.Quality != 3 {
			t.Errorf("Birth citation Quality = %d, want 3", cite.Quality)
		}
		if cite.Data == nil {
			t.Error("Birth citation Data is nil")
		} else if cite.Data.Text != "Original birth record entry" {
			t.Errorf("Birth citation Data.Text = %s, want 'Original birth record entry'", cite.Data.Text)
		}
		break
	}

	// Attributes
	attrTypes := make(map[string]bool)
	for _, attr := range indi.Attributes {
		attrTypes[attr.Type] = true
	}
	for _, exp := range []string{"OCCU", "CAST", "EDUC", "RELI"} {
		if !attrTypes[exp] {
			t.Errorf("Attribute %s not found", exp)
		}
	}

	// Associations
	if len(indi.Associations) < 2 {
		t.Errorf("len(Associations) = %d, want at least 2", len(indi.Associations))
	} else if indi.Associations[0].Role != "GODP" {
		t.Errorf("Association[0].Role = %s, want GODP", indi.Associations[0].Role)
	}

	// LDS ordinances
	if len(indi.LDSOrdinances) < 4 {
		t.Errorf("len(LDSOrdinances) = %d, want at least 4", len(indi.LDSOrdinances))
	} else {
		ordTypes := make(map[string]bool)
		for _, ord := range indi.LDSOrdinances {
			ordTypes[string(ord.Type)] = true
		}
		for _, exp := range []string{"BAPL", "CONL", "ENDL", "SLGC"} {
			if !ordTypes[exp] {
				t.Errorf("LDS ordinance %s not found", exp)
			}
		}
	}

	// Family @F2@
	fam := doc2.GetFamily("@F2@")
	if fam == nil {
		t.Fatal("Family @F2@ not found")
	}

	// Family events
	famEventTypes := make(map[string]bool)
	for _, ev := range fam.Events {
		famEventTypes[string(ev.Type)] = true
	}
	for _, exp := range []string{"MARB", "MARC", "MARL", "MARS", "MARR", "DIVF", "DIV"} {
		if !famEventTypes[exp] {
			t.Errorf("Family event %s not found", exp)
		}
	}

	// Family LDS ordinance
	if len(fam.LDSOrdinances) < 1 {
		t.Error("Family has no LDS ordinances")
	} else if fam.LDSOrdinances[0].Type != "SLGS" {
		t.Errorf("Family LDS ord type = %s, want SLGS", fam.LDSOrdinances[0].Type)
	}

	// Source record
	src := doc2.GetSource("@S1@")
	if src == nil {
		t.Fatal("Source @S1@ not found")
	}
	if src.Title != "County Records" {
		t.Errorf("Source.Title = %s, want 'County Records'", src.Title)
	}
	if src.Author != "John Archivist" {
		t.Errorf("Source.Author = %s, want 'John Archivist'", src.Author)
	}
}

// TestEncodeEdgeCases tests edge cases in encoding
func TestEncodeEdgeCases(t *testing.T) {
	t.Run("record with nil tags slice", func(t *testing.T) {
		doc := &gedcom.Document{
			Header: &gedcom.Header{Version: "5.5"},
			Records: []*gedcom.Record{
				{
					XRef: "@I1@",
					Type: gedcom.RecordTypeIndividual,
					Tags: nil, // nil slice
				},
			},
		}

		var buf bytes.Buffer
		if err := Encode(&buf, doc); err != nil {
			t.Fatalf("Encode() error = %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "0 @I1@ INDI") {
			t.Error("Output should contain INDI record")
		}
	})

	t.Run("tag with empty string value", func(t *testing.T) {
		doc := &gedcom.Document{
			Header: &gedcom.Header{Version: "5.5"},
			Records: []*gedcom.Record{
				{
					Type: gedcom.RecordTypeNote,
					Tags: []*gedcom.Tag{
						{Level: 1, Tag: "NOTE", Value: ""}, // Empty value
					},
				},
			},
		}

		var buf bytes.Buffer
		if err := Encode(&buf, doc); err != nil {
			t.Fatalf("Encode() error = %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "1 NOTE\n") {
			t.Error("Output should contain NOTE tag without value")
		}
	})

	t.Run("mixed tags with and without values", func(t *testing.T) {
		doc := &gedcom.Document{
			Header: &gedcom.Header{Version: "5.5"},
			Records: []*gedcom.Record{
				{
					XRef: "@I1@",
					Type: gedcom.RecordTypeIndividual,
					Tags: []*gedcom.Tag{
						{Level: 1, Tag: "NAME", Value: "Test Name"},
						{Level: 1, Tag: "BIRT", Value: ""},
						{Level: 2, Tag: "DATE", Value: "1 JAN 2000"},
						{Level: 2, Tag: "PLAC", Value: ""},
						{Level: 1, Tag: "SEX", Value: "M"},
					},
				},
			},
		}

		var buf bytes.Buffer
		if err := Encode(&buf, doc); err != nil {
			t.Fatalf("Encode() error = %v", err)
		}

		output := buf.String()
		// Verify tags with values
		if !strings.Contains(output, "1 NAME Test Name") {
			t.Error("Output should contain NAME with value")
		}
		if !strings.Contains(output, "2 DATE 1 JAN 2000") {
			t.Error("Output should contain DATE with value")
		}
		if !strings.Contains(output, "1 SEX M") {
			t.Error("Output should contain SEX with value")
		}
		// Verify tags without values
		if !strings.Contains(output, "1 BIRT\n") {
			t.Error("Output should contain BIRT without value")
		}
		if !strings.Contains(output, "2 PLAC\n") {
			t.Error("Output should contain PLAC without value")
		}
	})
}

// TestRoundtripMinimalFile tests round-trip encoding with the minimal.ged test file
func TestRoundtripMinimalFile(t *testing.T) {
	// Read original file
	f, err := os.Open("../testdata/gedcom-5.5.1/minimal.ged")
	if err != nil {
		t.Fatalf("Failed to open test file: %v", err)
	}
	defer f.Close()

	// Decode original
	doc1, err := decoder.Decode(f)
	if err != nil {
		t.Fatalf("Failed to decode original file: %v", err)
	}

	// Encode to buffer
	var buf bytes.Buffer
	if err := Encode(&buf, doc1); err != nil {
		t.Fatalf("Failed to encode: %v", err)
	}

	// Decode again
	doc2, err := decoder.Decode(strings.NewReader(buf.String()))
	if err != nil {
		t.Fatalf("Failed to decode encoded output: %v", err)
	}

	// Compare record counts
	if len(doc1.Records) != len(doc2.Records) {
		t.Errorf("Record count mismatch: got %d, want %d", len(doc2.Records), len(doc1.Records))
	}

	// Compare header
	if doc1.Header.Version != doc2.Header.Version {
		t.Errorf("Header version mismatch: got %q, want %q", doc2.Header.Version, doc1.Header.Version)
	}

	// Compare individual @I1@
	indi1 := doc1.GetIndividual("@I1@")
	indi2 := doc2.GetIndividual("@I1@")

	if indi1 == nil || indi2 == nil {
		t.Fatal("Individual @I1@ not found in one or both documents")
	}

	// Compare key fields
	if len(indi1.Names) != len(indi2.Names) {
		t.Errorf("Name count mismatch: got %d, want %d", len(indi2.Names), len(indi1.Names))
	}
	if indi1.Sex != indi2.Sex {
		t.Errorf("Sex mismatch: got %q, want %q", indi2.Sex, indi1.Sex)
	}
	if len(indi1.Events) != len(indi2.Events) {
		t.Errorf("Event count mismatch: got %d, want %d", len(indi2.Events), len(indi1.Events))
	}
}

// TestRoundtripEntityEncoding tests that entities without tags are properly encoded
func TestRoundtripEntityEncoding(t *testing.T) {
	// Create a document with entities but no tags
	doc := &gedcom.Document{
		Header: &gedcom.Header{
			Version:  "5.5.1",
			Encoding: "UTF-8",
		},
		Records: []*gedcom.Record{
			{
				XRef: "@I1@",
				Type: gedcom.RecordTypeIndividual,
				Tags: nil, // No tags - should use entity
				Entity: &gedcom.Individual{
					XRef: "@I1@",
					Names: []*gedcom.PersonalName{
						{Full: "John /Smith/", Given: "John", Surname: "Smith"},
					},
					Sex: "M",
					Events: []*gedcom.Event{
						{
							Type:  gedcom.EventBirth,
							Date:  "1 JAN 1950",
							Place: "Boston, MA",
						},
					},
				},
			},
			{
				XRef: "@F1@",
				Type: gedcom.RecordTypeFamily,
				Tags: nil,
				Entity: &gedcom.Family{
					Husband:  "@I1@",
					Wife:     "@I2@",
					Children: []string{"@I3@"},
					Events: []*gedcom.Event{
						{Type: gedcom.EventMarriage, Date: "15 JUN 1975"},
					},
				},
			},
			{
				XRef: "@S1@",
				Type: gedcom.RecordTypeSource,
				Tags: nil,
				Entity: &gedcom.Source{
					Title:  "Birth Records",
					Author: "County Archives",
				},
			},
			{
				XRef: "@SUBM1@",
				Type: gedcom.RecordTypeSubmitter,
				Tags: nil,
				Entity: &gedcom.Submitter{
					Name:     "Test User",
					Email:    []string{"test@example.com"},
					Language: []string{"English"},
				},
			},
			{
				XRef: "@R1@",
				Type: gedcom.RecordTypeRepository,
				Tags: nil,
				Entity: &gedcom.Repository{
					Name: "City Archives",
					Address: &gedcom.Address{
						City:    "Boston",
						State:   "MA",
						Country: "USA",
					},
				},
			},
			{
				XRef: "@N1@",
				Type: gedcom.RecordTypeNote,
				Tags: nil,
				Entity: &gedcom.Note{
					Text:         "This is a note",
					Continuation: []string{"with continuation"},
				},
			},
			{
				XRef: "@O1@",
				Type: gedcom.RecordTypeMedia,
				Tags: nil,
				Entity: &gedcom.MediaObject{
					Files: []*gedcom.MediaFile{
						{FileRef: "photo.jpg", Form: "image/jpeg", Title: "Family Photo"},
					},
				},
			},
		},
		XRefMap: make(map[string]*gedcom.Record),
	}

	// Build XRefMap
	for _, r := range doc.Records {
		doc.XRefMap[r.XRef] = r
	}

	// Encode
	var buf bytes.Buffer
	if err := Encode(&buf, doc); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	output := buf.String()

	// Verify entity data was encoded
	expectedPatterns := []string{
		"0 @I1@ INDI",
		"1 NAME John /Smith/",
		"2 GIVN John",
		"2 SURN Smith",
		"1 SEX M",
		"1 BIRT",
		"2 DATE 1 JAN 1950",
		"2 PLAC Boston, MA",
		"0 @F1@ FAM",
		"1 HUSB @I1@",
		"1 WIFE @I2@",
		"1 CHIL @I3@",
		"1 MARR",
		"0 @S1@ SOUR",
		"1 TITL Birth Records",
		"1 AUTH County Archives",
		"0 @SUBM1@ SUBM",
		"1 NAME Test User",
		"1 EMAIL test@example.com",
		"1 LANG English",
		"0 @R1@ REPO",
		"1 NAME City Archives",
		"1 ADDR",
		"0 @N1@ NOTE",
		"1 CONT with continuation",
		"0 @O1@ OBJE",
		"1 FILE photo.jpg",
		"2 FORM image/jpeg",
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(output, pattern) {
			t.Errorf("Output missing expected pattern: %q\nGot:\n%s", pattern, output)
		}
	}

	// Verify round-trip
	doc2, err := decoder.Decode(strings.NewReader(output))
	if err != nil {
		t.Fatalf("Failed to decode encoded output: %v", err)
	}

	if len(doc2.Records) != len(doc.Records) {
		t.Errorf("Record count mismatch: got %d, want %d", len(doc2.Records), len(doc.Records))
	}

	// Verify individual was decoded correctly
	indi := doc2.GetIndividual("@I1@")
	if indi == nil {
		t.Fatal("Individual @I1@ not found after round-trip")
	}
	if indi.Sex != "M" {
		t.Errorf("Individual sex = %q, want %q", indi.Sex, "M")
	}
	if len(indi.Names) != 1 || indi.Names[0].Given != "John" {
		t.Error("Individual name not preserved after round-trip")
	}

	// Verify family was decoded correctly
	fam := doc2.GetFamily("@F1@")
	if fam == nil {
		t.Fatal("Family @F1@ not found after round-trip")
	}
	if fam.Husband != "@I1@" {
		t.Errorf("Family husband = %q, want %q", fam.Husband, "@I1@")
	}

	// Verify source was decoded correctly
	src := doc2.GetSource("@S1@")
	if src == nil {
		t.Fatal("Source @S1@ not found after round-trip")
	}
	if src.Title != "Birth Records" {
		t.Errorf("Source title = %q, want %q", src.Title, "Birth Records")
	}

	// Verify submitter was decoded correctly
	subm := doc2.GetSubmitter("@SUBM1@")
	if subm == nil {
		t.Fatal("Submitter @SUBM1@ not found after round-trip")
	}
	if subm.Name != "Test User" {
		t.Errorf("Submitter name = %q, want %q", subm.Name, "Test User")
	}

	// Verify repository was decoded correctly
	repo := doc2.GetRepository("@R1@")
	if repo == nil {
		t.Fatal("Repository @R1@ not found after round-trip")
	}
	if repo.Name != "City Archives" {
		t.Errorf("Repository name = %q, want %q", repo.Name, "City Archives")
	}
}

// TestRoundtripTagsPreserved tests that when Tags are present, they are used instead of Entity
func TestRoundtripTagsPreserved(t *testing.T) {
	// Create a document with both tags and entity - tags should be used
	doc := &gedcom.Document{
		Header: &gedcom.Header{Version: "5.5"},
		Records: []*gedcom.Record{
			{
				XRef: "@I1@",
				Type: gedcom.RecordTypeIndividual,
				Tags: []*gedcom.Tag{
					{Level: 1, Tag: "NAME", Value: "Tags /Used/"},
				},
				Entity: &gedcom.Individual{
					Names: []*gedcom.PersonalName{
						{Full: "Entity /Ignored/"},
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	if err := Encode(&buf, doc); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	output := buf.String()

	// Tags should be used
	if !strings.Contains(output, "Tags /Used/") {
		t.Error("Tags should be used when present")
	}
	if strings.Contains(output, "Entity /Ignored/") {
		t.Error("Entity should be ignored when tags are present")
	}
}

// TestRoundtripComplexIndividual tests round-trip of an individual with all fields
func TestRoundtripComplexIndividual(t *testing.T) {
	doc := &gedcom.Document{
		Header: &gedcom.Header{
			Version:  "5.5.1",
			Encoding: "UTF-8",
		},
		Records: []*gedcom.Record{
			{
				XRef: "@I1@",
				Type: gedcom.RecordTypeIndividual,
				Tags: nil,
				Entity: &gedcom.Individual{
					XRef: "@I1@",
					Names: []*gedcom.PersonalName{
						{
							Full:          "Dr. Johann Ludwig /von Beethoven/ III",
							Given:         "Johann Ludwig",
							Surname:       "Beethoven",
							Prefix:        "Dr.",
							Suffix:        "III",
							Nickname:      "Ludwig",
							SurnamePrefix: "von",
							Type:          "birth",
						},
					},
					Sex: "M",
					Events: []*gedcom.Event{
						{
							Type:  gedcom.EventBirth,
							Date:  "17 DEC 1770",
							Place: "Bonn, Germany",
							PlaceDetail: &gedcom.PlaceDetail{
								Form: "City, Country",
								Coordinates: &gedcom.Coordinates{
									Latitude:  "N50.7339",
									Longitude: "E7.0998",
								},
							},
							EventTypeDetail: "Home birth",
							Cause:           "Natural",
							Age:             "0y",
							Agency:          "Family records",
							Notes:           []string{"Birth note"},
							SourceCitations: []*gedcom.SourceCitation{
								{
									SourceXRef: "@S1@",
									Page:       "p. 42",
									Quality:    3,
									Data: &gedcom.SourceCitationData{
										Date: "17 DEC 1770",
										Text: "Birth entry",
									},
								},
							},
						},
						{Type: gedcom.EventDeath, Date: "26 MAR 1827", Cause: "Liver disease"},
					},
					Attributes: []*gedcom.Attribute{
						{Type: "OCCU", Value: "Composer", Date: "1792", Place: "Vienna"},
					},
					ChildInFamilies:  []gedcom.FamilyLink{{FamilyXRef: "@F1@", Pedigree: "birth"}},
					SpouseInFamilies: []string{"@F2@"},
					Associations: []*gedcom.Association{
						{IndividualXRef: "@I2@", Role: "GODP", Notes: []string{"Godfather"}},
					},
					LDSOrdinances: []*gedcom.LDSOrdinance{
						{Type: "BAPL", Date: "1 JAN 1900", Temple: "SLAKE", Status: "COMPLETED"},
					},
					SourceCitations: []*gedcom.SourceCitation{
						{SourceXRef: "@S1@", Page: "Entire file"},
					},
					Notes:        []string{"Famous composer"},
					Media:        []*gedcom.MediaLink{{MediaXRef: "@O1@", Title: "Portrait"}},
					ChangeDate:   &gedcom.ChangeDate{Date: "1 JAN 2024", Time: "12:00:00"},
					CreationDate: &gedcom.ChangeDate{Date: "1 JAN 2020"},
					RefNumber:    "REF001",
					UID:          "UID-BEETHOVEN",
				},
			},
		},
	}

	// Encode
	var buf bytes.Buffer
	if err := Encode(&buf, doc); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	output := buf.String()

	// Verify key elements are present
	expectedPatterns := []string{
		"1 NAME Dr. Johann Ludwig /von Beethoven/ III",
		"2 GIVN Johann Ludwig",
		"2 SURN Beethoven",
		"2 NPFX Dr.",
		"2 NSFX III",
		"2 NICK Ludwig",
		"2 SPFX von",
		"2 TYPE birth",
		"1 SEX M",
		"1 BIRT",
		"2 DATE 17 DEC 1770",
		"2 PLAC Bonn, Germany",
		"3 FORM City, Country",
		"3 MAP",
		"4 LATI N50.7339",
		"4 LONG E7.0998",
		"2 TYPE Home birth",
		"2 CAUS Natural",
		"2 AGE 0y",
		"2 AGNC Family records",
		"2 NOTE Birth note",
		"2 SOUR @S1@",
		"3 PAGE p. 42",
		"3 QUAY 3",
		"3 DATA",
		"4 DATE 17 DEC 1770",
		"4 TEXT Birth entry",
		"1 DEAT",
		"2 CAUS Liver disease",
		"1 OCCU Composer",
		"2 DATE 1792",
		"2 PLAC Vienna",
		"1 FAMC @F1@",
		"2 PEDI birth",
		"1 FAMS @F2@",
		"1 ASSO @I2@",
		"2 ROLE GODP",
		"2 NOTE Godfather",
		"1 BAPL",
		"2 TEMP SLAKE",
		"2 STAT COMPLETED",
		"1 SOUR @S1@",
		"2 PAGE Entire file",
		"1 NOTE Famous composer",
		"1 OBJE @O1@",
		"2 TITL Portrait",
		"1 CHAN",
		"2 DATE 1 JAN 2024",
		"3 TIME 12:00:00",
		"1 CREA",
		"1 REFN REF001",
		"1 UID UID-BEETHOVEN",
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(output, pattern) {
			t.Errorf("Output missing expected pattern: %q", pattern)
		}
	}

	// Round-trip decode
	doc2, err := decoder.Decode(strings.NewReader(output))
	if err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	indi := doc2.GetIndividual("@I1@")
	if indi == nil {
		t.Fatal("Individual not found after round-trip")
	}

	// Verify name components
	if len(indi.Names) < 1 {
		t.Fatal("Names not preserved")
	}
	name := indi.Names[0]
	if name.Nickname != "Ludwig" {
		t.Errorf("Nickname = %q, want %q", name.Nickname, "Ludwig")
	}
	if name.SurnamePrefix != "von" {
		t.Errorf("SurnamePrefix = %q, want %q", name.SurnamePrefix, "von")
	}

	// Verify associations
	if len(indi.Associations) < 1 {
		t.Fatal("Associations not preserved")
	}
	if indi.Associations[0].Role != "GODP" {
		t.Errorf("Association role = %q, want %q", indi.Associations[0].Role, "GODP")
	}

	// Verify LDS ordinances
	if len(indi.LDSOrdinances) < 1 {
		t.Fatal("LDS ordinances not preserved")
	}
	if indi.LDSOrdinances[0].Temple != "SLAKE" {
		t.Errorf("LDS temple = %q, want %q", indi.LDSOrdinances[0].Temple, "SLAKE")
	}
}

// TestRoundtripFamilySearchID tests round-trip encoding of _FSFTID tag.
// Ref: Issue #80
func TestRoundtripFamilySearchID(t *testing.T) {
	input := `0 HEAD
1 SOUR FamilySearch
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME John /Doe/
1 _FSFTID KWCJ-QN7
0 @I2@ INDI
1 NAME Jane /Smith/
0 TRLR
`

	// Decode
	doc1, err := decoder.Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Initial Decode() error = %v", err)
	}

	// Verify initial decode
	indi1 := doc1.GetIndividual("@I1@")
	if indi1 == nil {
		t.Fatal("Individual @I1@ not found")
	}
	if indi1.FamilySearchID != "KWCJ-QN7" {
		t.Errorf("Initial FamilySearchID = %q, want %q", indi1.FamilySearchID, "KWCJ-QN7")
	}

	// Encode
	var buf bytes.Buffer
	if err := Encode(&buf, doc1); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	output := buf.String()

	// Verify _FSFTID tag is in output
	if !strings.Contains(output, "1 _FSFTID KWCJ-QN7") {
		t.Errorf("Output missing _FSFTID tag. Got:\n%s", output)
	}

	// Decode again to verify round-trip
	doc2, err := decoder.Decode(strings.NewReader(output))
	if err != nil {
		t.Fatalf("Second Decode() error = %v", err)
	}

	// Verify FamilySearchID preserved after round-trip
	indi2 := doc2.GetIndividual("@I1@")
	if indi2 == nil {
		t.Fatal("Individual @I1@ not found after round-trip")
	}
	if indi2.FamilySearchID != "KWCJ-QN7" {
		t.Errorf("Round-trip FamilySearchID = %q, want %q", indi2.FamilySearchID, "KWCJ-QN7")
	}

	// Verify FamilySearchURL works
	expectedURL := "https://www.familysearch.org/tree/person/details/KWCJ-QN7"
	if indi2.FamilySearchURL() != expectedURL {
		t.Errorf("FamilySearchURL() = %q, want %q", indi2.FamilySearchURL(), expectedURL)
	}

	// Verify individual without FamilySearchID still has empty value
	indi3 := doc2.GetIndividual("@I2@")
	if indi3 == nil {
		t.Fatal("Individual @I2@ not found")
	}
	if indi3.FamilySearchID != "" {
		t.Errorf("@I2@ FamilySearchID = %q, want empty", indi3.FamilySearchID)
	}
	if indi3.FamilySearchURL() != "" {
		t.Errorf("@I2@ FamilySearchURL() = %q, want empty", indi3.FamilySearchURL())
	}
}

// TestEncodeTargetVersion tests that TargetVersion option overrides header version.
func TestEncodeTargetVersion(t *testing.T) {
	doc := &gedcom.Document{
		Header: &gedcom.Header{
			Version:  gedcom.Version55, // Original version
			Encoding: "UTF-8",
		},
		Records: []*gedcom.Record{},
	}

	t.Run("target version overrides header", func(t *testing.T) {
		opts := &EncodeOptions{
			LineEnding:          "\n",
			TargetVersion:       gedcom.Version70,
			PreserveUnknownTags: true,
		}

		var buf bytes.Buffer
		if err := EncodeWithOptions(&buf, doc, opts); err != nil {
			t.Fatalf("Encode() error = %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "2 VERS 7.0") {
			t.Errorf("Output should contain target version 7.0, got:\n%s", output)
		}
		if strings.Contains(output, "2 VERS 5.5\n") {
			t.Error("Output should not contain original version 5.5")
		}
	})

	t.Run("empty target version uses header", func(t *testing.T) {
		opts := &EncodeOptions{
			LineEnding:          "\n",
			TargetVersion:       "", // Empty - should use header
			PreserveUnknownTags: true,
		}

		var buf bytes.Buffer
		if err := EncodeWithOptions(&buf, doc, opts); err != nil {
			t.Fatalf("Encode() error = %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "2 VERS 5.5") {
			t.Errorf("Output should contain header version 5.5, got:\n%s", output)
		}
	})

	t.Run("target version with empty header", func(t *testing.T) {
		docNoVersion := &gedcom.Document{
			Header:  &gedcom.Header{Encoding: "UTF-8"},
			Records: []*gedcom.Record{},
		}

		opts := &EncodeOptions{
			LineEnding:          "\n",
			TargetVersion:       gedcom.Version551,
			PreserveUnknownTags: true,
		}

		var buf bytes.Buffer
		if err := EncodeWithOptions(&buf, docNoVersion, opts); err != nil {
			t.Fatalf("Encode() error = %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "2 VERS 5.5.1") {
			t.Errorf("Output should contain target version 5.5.1, got:\n%s", output)
		}
	})
}

// TestEncodePreserveUnknownTags tests the PreserveUnknownTags option.
func TestEncodePreserveUnknownTags(t *testing.T) {
	doc := &gedcom.Document{
		Header: &gedcom.Header{
			Version:  gedcom.Version551,
			Encoding: "UTF-8",
		},
		Records: []*gedcom.Record{
			{
				XRef: "@I1@",
				Type: gedcom.RecordTypeIndividual,
				Tags: []*gedcom.Tag{
					{Level: 1, Tag: "NAME", Value: "John /Doe/"},
					{Level: 1, Tag: "_CUSTOM", Value: "custom value"},
					{Level: 2, Tag: "_NESTED", Value: "nested under custom"},
					{Level: 1, Tag: "SEX", Value: "M"},
					{Level: 1, Tag: "_ANOTHER", Value: "another custom"},
					{Level: 1, Tag: "BIRT"},
					{Level: 2, Tag: "DATE", Value: "1 JAN 1900"},
				},
			},
		},
	}

	t.Run("preserve unknown tags (default)", func(t *testing.T) {
		opts := DefaultOptions()

		var buf bytes.Buffer
		if err := EncodeWithOptions(&buf, doc, opts); err != nil {
			t.Fatalf("Encode() error = %v", err)
		}

		output := buf.String()
		// All tags should be present
		if !strings.Contains(output, "1 _CUSTOM custom value") {
			t.Error("Output should contain _CUSTOM tag when PreserveUnknownTags is true")
		}
		if !strings.Contains(output, "2 _NESTED nested under custom") {
			t.Error("Output should contain _NESTED tag when PreserveUnknownTags is true")
		}
		if !strings.Contains(output, "1 _ANOTHER another custom") {
			t.Error("Output should contain _ANOTHER tag when PreserveUnknownTags is true")
		}
	})

	t.Run("filter unknown tags", func(t *testing.T) {
		opts := &EncodeOptions{
			LineEnding:          "\n",
			PreserveUnknownTags: false,
		}

		var buf bytes.Buffer
		if err := EncodeWithOptions(&buf, doc, opts); err != nil {
			t.Fatalf("Encode() error = %v", err)
		}

		output := buf.String()

		// Custom tags should be filtered
		if strings.Contains(output, "_CUSTOM") {
			t.Error("Output should not contain _CUSTOM tag when PreserveUnknownTags is false")
		}
		if strings.Contains(output, "_NESTED") {
			t.Error("Output should not contain _NESTED tag (child of custom) when PreserveUnknownTags is false")
		}
		if strings.Contains(output, "_ANOTHER") {
			t.Error("Output should not contain _ANOTHER tag when PreserveUnknownTags is false")
		}

		// Standard tags should still be present
		if !strings.Contains(output, "1 NAME John /Doe/") {
			t.Error("Output should contain NAME tag")
		}
		if !strings.Contains(output, "1 SEX M") {
			t.Error("Output should contain SEX tag")
		}
		if !strings.Contains(output, "1 BIRT") {
			t.Error("Output should contain BIRT tag")
		}
		if !strings.Contains(output, "2 DATE 1 JAN 1900") {
			t.Error("Output should contain DATE tag")
		}
	})

	t.Run("filter preserves nested standard tags", func(t *testing.T) {
		docWithNesting := &gedcom.Document{
			Header: &gedcom.Header{Version: gedcom.Version55},
			Records: []*gedcom.Record{
				{
					XRef: "@I1@",
					Type: gedcom.RecordTypeIndividual,
					Tags: []*gedcom.Tag{
						{Level: 1, Tag: "NAME", Value: "Jane /Smith/"},
						{Level: 2, Tag: "GIVN", Value: "Jane"},
						{Level: 2, Tag: "_CUSTOM_NAME", Value: "custom name data"},
						{Level: 3, Tag: "_DEEP", Value: "deep custom"},
						{Level: 2, Tag: "SURN", Value: "Smith"},
					},
				},
			},
		}

		opts := &EncodeOptions{
			LineEnding:          "\n",
			PreserveUnknownTags: false,
		}

		var buf bytes.Buffer
		if err := EncodeWithOptions(&buf, docWithNesting, opts); err != nil {
			t.Fatalf("Encode() error = %v", err)
		}

		output := buf.String()

		// Standard nested tags should be preserved
		if !strings.Contains(output, "2 GIVN Jane") {
			t.Error("Output should contain GIVN tag")
		}
		if !strings.Contains(output, "2 SURN Smith") {
			t.Error("Output should contain SURN tag")
		}

		// Custom tags and their children should be filtered
		if strings.Contains(output, "_CUSTOM_NAME") {
			t.Error("Output should not contain _CUSTOM_NAME")
		}
		if strings.Contains(output, "_DEEP") {
			t.Error("Output should not contain _DEEP (child of custom)")
		}
	})
}

// TestDefaultOptionsPreserveUnknownTags verifies default value for PreserveUnknownTags.
func TestDefaultOptionsPreserveUnknownTags(t *testing.T) {
	opts := DefaultOptions()
	if !opts.PreserveUnknownTags {
		t.Error("DefaultOptions().PreserveUnknownTags should be true")
	}
}

// TestFilterTagsHelper tests the filterTags helper function directly.
func TestFilterTagsHelper(t *testing.T) {
	tags := []*gedcom.Tag{
		{Level: 1, Tag: "NAME", Value: "Test"},
		{Level: 1, Tag: "_CUSTOM", Value: "custom"},
		{Level: 2, Tag: "CHILD", Value: "child of custom"},
		{Level: 3, Tag: "_DEEP", Value: "deep custom"},
		{Level: 4, Tag: "DEEPER", Value: "even deeper"},
		{Level: 1, Tag: "NEXT", Value: "back to level 1"},
		{Level: 2, Tag: "SUB", Value: "sub of next"},
	}

	t.Run("preserve all", func(t *testing.T) {
		result := filterTags(tags, true)
		if len(result) != len(tags) {
			t.Errorf("filterTags(true) = %d tags, want %d", len(result), len(tags))
		}
	})

	t.Run("filter custom", func(t *testing.T) {
		result := filterTags(tags, false)

		// Should have: NAME, NEXT, SUB (filtered: _CUSTOM and all children until NEXT)
		// _CUSTOM at level 1 -> skip until we see level 1 or lower
		// CHILD at level 2 -> skipped (higher than 1)
		// _DEEP at level 3 -> skipped (higher than 1)
		// DEEPER at level 4 -> skipped (higher than 1)
		// NEXT at level 1 -> not skipped (back to level 1)
		// SUB at level 2 -> not skipped (no active skip)
		expected := []string{"NAME", "NEXT", "SUB"}
		if len(result) != len(expected) {
			var got []string
			for _, t := range result {
				got = append(got, t.Tag)
			}
			t.Errorf("filterTags(false) = %v, want %v", got, expected)
		}
	})
}

// TestIsCustomTag tests the isCustomTag helper function.
func TestIsCustomTag(t *testing.T) {
	tests := []struct {
		tag    string
		custom bool
	}{
		{"NAME", false},
		{"_CUSTOM", true},
		{"_", true},
		{"BIRT", false},
		{"_FSFTID", true},
		{"__DOUBLE", true},
		{"INDI", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			if got := isCustomTag(tt.tag); got != tt.custom {
				t.Errorf("isCustomTag(%q) = %v, want %v", tt.tag, got, tt.custom)
			}
		})
	}
}
