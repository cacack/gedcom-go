package validator

import (
	"testing"

	"github.com/cacack/gedcom-go/gedcom"
)

// BenchmarkValidateMinimal benchmarks validating a minimal document
func BenchmarkValidateMinimal(b *testing.B) {
	doc := &gedcom.Document{
		Header: &gedcom.Header{Version: "5.5"},
		Records: []*gedcom.Record{
			{
				XRef: "@I1@",
				Type: gedcom.RecordTypeIndividual,
				Tags: []*gedcom.Tag{
					{Level: 1, Tag: "NAME", Value: "John /Doe/"},
				},
			},
		},
		XRefMap: map[string]*gedcom.Record{
			"@I1@": {XRef: "@I1@", Type: gedcom.RecordTypeIndividual},
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		v := New()
		_ = v.Validate(doc)
	}
}

// BenchmarkValidateSmall benchmarks validating a small document (10 individuals)
func BenchmarkValidateSmall(b *testing.B) {
	doc := generateValidDocument(10)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		v := New()
		_ = v.Validate(doc)
	}
}

// BenchmarkValidateMedium benchmarks validating a medium document (100 individuals)
func BenchmarkValidateMedium(b *testing.B) {
	doc := generateValidDocument(100)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		v := New()
		_ = v.Validate(doc)
	}
}

// BenchmarkValidateLarge benchmarks validating a large document (1000 individuals)
func BenchmarkValidateLarge(b *testing.B) {
	doc := generateValidDocument(1000)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		v := New()
		_ = v.Validate(doc)
	}
}

// BenchmarkValidateWithErrors benchmarks validating a document with broken references
func BenchmarkValidateWithErrors(b *testing.B) {
	doc := generateInvalidDocument(100)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		v := New()
		_ = v.Validate(doc)
	}
}

// Helper to generate a valid document with N individuals
func generateValidDocument(numIndividuals int) *gedcom.Document {
	records := make([]*gedcom.Record, 0, numIndividuals)
	xrefMap := make(map[string]*gedcom.Record)

	for i := 0; i < numIndividuals; i++ {
		xref := "@I" + string(rune('0'+i%10)) + "@"
		rec := &gedcom.Record{
			XRef: xref,
			Type: gedcom.RecordTypeIndividual,
			Tags: []*gedcom.Tag{
				{Level: 1, Tag: "NAME", Value: "Person " + string(rune('0'+i%10)) + " /Surname/"},
				{Level: 1, Tag: "SEX", Value: "M"},
			},
			Entity: &gedcom.Individual{
				XRef: xref,
				Names: []*gedcom.PersonalName{
					{Full: "Person " + string(rune('0'+i%10)) + " /Surname/"},
				},
				Sex: "M",
			},
		}
		records = append(records, rec)
		xrefMap[xref] = rec
	}

	return &gedcom.Document{
		Header:  &gedcom.Header{Version: "5.5"},
		Records: records,
		XRefMap: xrefMap,
	}
}

// Helper to generate an invalid document with broken references
func generateInvalidDocument(numIndividuals int) *gedcom.Document {
	records := make([]*gedcom.Record, 0, numIndividuals+numIndividuals/10)
	xrefMap := make(map[string]*gedcom.Record)

	// Add valid individuals
	for i := 0; i < numIndividuals; i++ {
		xref := "@I" + string(rune('0'+i%10)) + "@"
		rec := &gedcom.Record{
			XRef: xref,
			Type: gedcom.RecordTypeIndividual,
			Tags: []*gedcom.Tag{
				{Level: 1, Tag: "NAME", Value: "Person " + string(rune('0'+i%10)) + " /Surname/"},
				{Level: 1, Tag: "SEX", Value: "M"},
			},
			Entity: &gedcom.Individual{
				XRef: xref,
				Names: []*gedcom.PersonalName{
					{Full: "Person " + string(rune('0'+i%10)) + " /Surname/"},
				},
				Sex: "M",
			},
		}
		records = append(records, rec)
		xrefMap[xref] = rec
	}

	// Add families with broken references (10%)
	for i := 0; i < numIndividuals/10; i++ {
		xref := "@F" + string(rune('0'+i%10)) + "@"
		rec := &gedcom.Record{
			XRef: xref,
			Type: gedcom.RecordTypeFamily,
			Tags: []*gedcom.Tag{
				{Level: 1, Tag: "HUSB", Value: "@NONEXISTENT@"}, // Broken reference
				{Level: 1, Tag: "WIFE", Value: "@ALSO_MISSING@"}, // Broken reference
			},
			Entity: &gedcom.Family{
				XRef:    xref,
				Husband: "@NONEXISTENT@",
				Wife:    "@ALSO_MISSING@",
			},
		}
		records = append(records, rec)
		xrefMap[xref] = rec
	}

	return &gedcom.Document{
		Header:  &gedcom.Header{Version: "5.5"},
		Records: records,
		XRefMap: xrefMap,
	}
}
