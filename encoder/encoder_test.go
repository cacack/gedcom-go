package encoder

import (
	"bytes"
	"errors"
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
