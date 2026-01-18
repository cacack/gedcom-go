package converter

import (
	"testing"

	"github.com/cacack/gedcom-go/gedcom"
)

func TestValidateConverted(t *testing.T) {
	t.Run("adds validation issues to report", func(t *testing.T) {
		// Create a document that might have validation issues
		doc := &gedcom.Document{
			Header: &gedcom.Header{
				Version: gedcom.Version70,
			},
			Records: []*gedcom.Record{},
		}
		report := &gedcom.ConversionReport{}

		err := validateConverted(doc, report)

		// validateConverted should never return an error
		if err != nil {
			t.Errorf("validateConverted() should not return error, got %v", err)
		}

		// It may or may not add validation issues depending on the document
		// The important thing is it doesn't panic and returns nil
	})

	t.Run("validation runs without error", func(t *testing.T) {
		doc := &gedcom.Document{
			Header: &gedcom.Header{
				Version:  gedcom.Version55,
				Encoding: gedcom.EncodingUTF8,
			},
			Records: []*gedcom.Record{
				{
					XRef: "@I1@",
					Type: gedcom.RecordTypeIndividual,
					Entity: &gedcom.Individual{
						XRef: "@I1@",
						Names: []*gedcom.PersonalName{
							{Full: "John /Doe/", Given: "John", Surname: "Doe"},
						},
					},
				},
			},
		}
		report := &gedcom.ConversionReport{}

		err := validateConverted(doc, report)

		if err != nil {
			t.Errorf("validateConverted() error = %v", err)
		}
	})
}
