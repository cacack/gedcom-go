// Package main demonstrates creating GEDCOM documents programmatically and encoding them to standard GEDCOM format.
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/cacack/gedcom-go/v3/encoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

func main() {
	// Create a simple GEDCOM document
	doc := gedcom.NewDocument(gedcom.Version55)
	doc.Header.Encoding = "UTF-8"
	doc.Header.SourceSystem = "go-gedcom example"
	doc.Header.Language = "English"

	// Records carry only typed Entities; the encoder builds their tags from
	// the typed fields (set Record.Tags only for structures the model lacks).
	records := []*gedcom.Record{
		// Individual record
		{
			XRef: "@I1@",
			Type: gedcom.RecordTypeIndividual,
			Entity: &gedcom.Individual{
				XRef: "@I1@",
				Names: []*gedcom.PersonalName{
					{Full: "John /Doe/", Given: "John", Surname: "Doe"},
				},
				Sex: "M",
				Events: []*gedcom.Event{
					{Type: gedcom.EventBirth, Date: "1 JAN 1900", PlaceDetail: &gedcom.PlaceDetail{Name: "New York, USA"}},
				},
			},
		},
		// Another individual
		{
			XRef: "@I2@",
			Type: gedcom.RecordTypeIndividual,
			Entity: &gedcom.Individual{
				XRef: "@I2@",
				Names: []*gedcom.PersonalName{
					{Full: "Jane /Smith/", Given: "Jane", Surname: "Smith"},
				},
				Sex: "F",
			},
		},
		// Family record
		{
			XRef: "@F1@",
			Type: gedcom.RecordTypeFamily,
			Entity: &gedcom.Family{
				XRef:    "@F1@",
				Husband: "@I1@",
				Wife:    "@I2@",
				Events: []*gedcom.Event{
					{Type: gedcom.EventMarriage, Date: "15 JUN 1925", PlaceDetail: &gedcom.PlaceDetail{Name: "Boston, Massachusetts, USA"}},
				},
			},
		},
	}

	// AddRecord appends each record and indexes it for XRef lookup
	for _, record := range records {
		if err := doc.AddRecord(record); err != nil {
			log.Fatalf("Failed to add record: %v", err)
		}
	}

	// Encode to file or stdout
	var out *os.File
	var err error

	if len(os.Args) > 1 {
		filename := filepath.Clean(os.Args[1])
		// #nosec G703 -- the path is this CLI's own argv, not attacker-controlled input; see the package comment on validating against an allowed root in production.
		out, err = os.Create(filename)
		if err != nil {
			log.Fatalf("Failed to create file: %v", err)
		}
		defer out.Close()
		fmt.Printf("Writing GEDCOM to: %s\n", filename)
	} else {
		out = os.Stdout
		fmt.Println("GEDCOM output:")
		fmt.Println("===============")
	}

	// Encode with CRLF line endings (standard for GEDCOM)
	opts := &encoder.EncodeOptions{
		LineEnding: "\r\n",
	}

	if err := encoder.EncodeWithOptions(out, doc, opts); err != nil {
		log.Fatalf("Failed to encode GEDCOM: %v", err)
	}

	if len(os.Args) > 1 {
		fmt.Println("Successfully wrote GEDCOM file!")
	}
}
