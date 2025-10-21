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
