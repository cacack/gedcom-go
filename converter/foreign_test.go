package converter

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/encoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
	"github.com/cacack/gedcom-go/v3/internal/specgraph"
)

// TestStructuresGenUpToDate keeps structures_gen.go in step with the vendored
// specification files: regenerate it with go generate ./converter.
func TestStructuresGenUpToDate(t *testing.T) {
	want, err := specgraph.Generate("../testdata/spec")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	got, err := os.ReadFile("structures_gen.go")
	if err != nil {
		t.Fatalf("read structures_gen.go: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Error("structures_gen.go is stale; run go generate ./converter")
	}
}

// undefinedStructures returns the path of every tag under root that graph
// does not define in its context. Extension tags, CONT/CONC and what lies
// under an undefined tag are not reported; skip names foreign tags another
// conversion maps (#704).
func undefinedStructures(graph specGraph, root string, tags []*gedcom.Tag, skip map[string]bool) []string {
	type frame struct {
		level      int
		path       string
		structures []string
	}
	stack := []frame{{path: root, structures: graph[""][root]}}
	var undefined []string
	for _, tag := range tags {
		for len(stack) > 1 && stack[len(stack)-1].level >= tag.Level {
			stack = stack[:len(stack)-1]
		}
		parent := stack[len(stack)-1]
		path := parent.path + "." + tag.Tag
		var structures []string
		for _, s := range parent.structures {
			structures = append(structures, graph[s][tag.Tag]...)
		}
		if len(structures) == 0 && len(parent.structures) > 0 &&
			!strings.HasPrefix(tag.Tag, "_") && tag.Tag != "CONT" && tag.Tag != "CONC" && !skip[tag.Tag] {
			undefined = append(undefined, path)
		}
		stack = append(stack, frame{level: tag.Level, path: path, structures: structures})
	}
	return undefined
}

func convertFile(t *testing.T, path string, target gedcom.Version) (*gedcom.Document, *gedcom.ConversionReport, string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return convertText(t, string(data), target)
}

func convertText(t *testing.T, text string, target gedcom.Version) (*gedcom.Document, *gedcom.ConversionReport, string) {
	t.Helper()
	doc, err := decoder.Decode(strings.NewReader(text))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	converted, report, err := Convert(doc, target)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	var buf bytes.Buffer
	if err := encoder.Encode(&buf, converted); err != nil {
		t.Fatalf("encode: %v", err)
	}
	return converted, report, buf.String()
}

// TestMaximal70To551HasNoForeignStructures is issue #705's downgrade
// criterion: maximal70 converted to 5.5.1 and read back holds only
// structures 5.5.1 defines, carries HEAD.SOUR and HEAD.SUBM, and the report
// names what was dropped.
func TestMaximal70To551HasNoForeignStructures(t *testing.T) {
	_, report, out := convertFile(t, "../testdata/gedcom-7.0/maximal70.ged", gedcom.Version551)
	back, err := decoder.Decode(strings.NewReader(out))
	if err != nil {
		t.Fatalf("decode output: %v", err)
	}

	// ROLE and FORM.MEDI are the enumeration translation's (#704).
	skip := map[string]bool{"ROLE": true, "MEDI": true}
	undefined := undefinedStructures(specStructures551, "HEAD", back.Header.Tags, skip)
	for _, record := range back.Records {
		undefined = append(undefined, undefinedStructures(specStructures551, string(record.Type), record.Tags, skip)...)
	}
	if len(undefined) > 0 {
		t.Errorf("output keeps structures 5.5.1 does not define: %v", undefined)
	}

	if v := headerValue(back.Header, "SOUR", ""); v == "" {
		t.Error("output has no HEAD.SOUR")
	}
	if sub := headerValue(back.Header, "SUBM", ""); back.GetSubmitter(sub) == nil {
		t.Errorf("HEAD.SUBM = %q, want a pointer to a SUBM record", sub)
	}

	// Some of the drops, each named in the report.
	for _, path := range []string{
		"Individual @I1@ > INIL",
		"Family @F1@ > FACT",
		"Family @F1@ > MARR > SDATE",
		"Individual @I1@ > DEAT > ASSO",
		"Header > NOTE > MIME",
		"Header > NOTE > TRAN",
		"Submitter @U1@ > REFN",
	} {
		if !hasDropped(report, path) {
			t.Errorf("report has no Dropped note for %q", path)
		}
	}
	if !hasDataLoss(report, "INDI.INIL structures", "@I1@") {
		t.Errorf("no DataLoss for INDI.INIL naming @I1@: %+v", report.DataLoss)
	}
}

// TestSharedNoteForeignStructuresDropped70To551: a shared note is renamed
// NOTE before the drop pass, and its 7.0-only substructures are still
// dropped and reported, not written into the 5.5.1 output.
func TestSharedNoteForeignStructuresDropped70To551(t *testing.T) {
	const in = `0 HEAD
1 GEDC
2 VERS 7.0
0 @S1@ SOUR
1 TITL Source
0 @N1@ SNOTE Shared text
1 SOUR @S1@
2 DATA
3 DATE 1 JAN 2000
4 TIME 10:00
2 NOTE Inner
3 LANG en
0 TRLR
`
	converted, report, out := convertText(t, in, gedcom.Version551)
	note := converted.GetRecord("@N1@")
	if note == nil || note.Type != gedcom.RecordTypeNote {
		t.Fatalf("@N1@ = %+v, want a NOTE record", note)
	}
	if undefined := undefinedStructures(specStructures551, "NOTE", note.Tags, nil); len(undefined) > 0 {
		t.Errorf("NOTE record keeps structures 5.5.1 does not define: %v", undefined)
	}
	for _, line := range []string{"4 TIME 10:00", "3 LANG en"} {
		if strings.Contains(out, line) {
			t.Errorf("output keeps %q:\n%s", line, out)
		}
	}
	for _, path := range []string{
		"Note @N1@ > SOUR > DATA > DATE > TIME",
		"Note @N1@ > SOUR > NOTE > LANG",
	} {
		if !hasDropped(report, path) {
			t.Errorf("report has no Dropped note for %q: %+v", path, report.Dropped)
		}
	}
	if !strings.Contains(out, "3 DATE 1 JAN 2000") || !strings.Contains(out, "2 NOTE Inner") {
		t.Errorf("output lost structures 5.5.1 defines:\n%s", out)
	}
}

func hasDropped(report *gedcom.ConversionReport, path string) bool {
	for _, n := range report.Dropped {
		if n.Path == path {
			return true
		}
	}
	return false
}

func hasDataLoss(report *gedcom.ConversionReport, feature, xref string) bool {
	for _, d := range report.DataLoss {
		if d.Feature != feature {
			continue
		}
		for _, x := range d.AffectedRecords {
			if x == xref {
				return true
			}
		}
	}
	return false
}

// TestMaximal70To55HasNoForeignStructures is issue #794's downgrade
// criterion: maximal70 converted to 5.5 and read back holds only structures
// 5.5 defines, each drop named in the report, and its OBJE records' files
// survive as inline links.
func TestMaximal70To55HasNoForeignStructures(t *testing.T) {
	_, report, out := convertFile(t, "../testdata/gedcom-7.0/maximal70.ged", gedcom.Version55)
	back, err := decoder.Decode(strings.NewReader(out))
	if err != nil {
		t.Fatalf("decode output: %v", err)
	}

	// ROLE is the enumeration translation's (#704).
	skip := map[string]bool{"ROLE": true}
	undefined := undefinedStructures(specStructures55, "HEAD", back.Header.Tags, skip)
	for _, record := range back.Records {
		undefined = append(undefined, undefinedStructures(specStructures55, string(record.Type), record.Tags, skip)...)
	}
	if len(undefined) > 0 {
		t.Errorf("output keeps structures 5.5 does not define: %v", undefined)
	}

	for _, path := range []string{
		"Individual @I1@ > INIL",
		"Family @F1@ > FACT",
		"Family @F1@ > MARR > SDATE",
		"Individual @I1@ > DEAT > ASSO",
		"Header > NOTE > MIME",
		"Submitter @U1@ > REFN",
		"Submitter @U1@ > OBJE > CROP",
	} {
		if !hasDropped(report, path) {
			t.Errorf("report has no Dropped note for %q", path)
		}
	}

	// @O1@'s three files and @O2@'s one survive, in inline links.
	for _, want := range []string{
		"1 OBJE\n2 FORM image/jpeg\n2 TITL Title\n2 FILE media/CharlotteBront%C3%AB.jpg\n",
		"2 FILE file:///path/to/file1\n",
		"2 FILE media/original.mp3\n",
		"2 FILE http://host.example.com/path/to/file2\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q", want)
		}
	}
	if strings.Contains(out, "0 @O1@ OBJE") || strings.Contains(out, "0 @O2@ OBJE") {
		t.Error("output keeps an OBJE record")
	}
	if !hasDropped(report, "MediaObject @O1@ > FILE > FORM > MEDI") || !hasDataLoss(report, "OBJE record structures", "@O1@") {
		t.Errorf("what the inline links cannot hold is not reported: %+v", report.DataLoss)
	}
}
