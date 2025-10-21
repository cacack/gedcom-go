// Example: Create and encode a GEDCOM document
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/cacack/gedcom-go/encoder"
	"github.com/cacack/gedcom-go/gedcom"
)

func main() {
	// Create a simple GEDCOM document
	doc := &gedcom.Document{
		Header: &gedcom.Header{
			Version:      "5.5",
			Encoding:     "UTF-8",
			SourceSystem: "go-gedcom example",
			Language:     "English",
		},
		Records: []*gedcom.Record{
			// Individual record
			{
				XRef: "@I1@",
				Type: gedcom.RecordTypeIndividual,
				Tags: []*gedcom.Tag{
					{Level: 1, Tag: "NAME", Value: "John /Doe/"},
					{Level: 2, Tag: "GIVN", Value: "John"},
					{Level: 2, Tag: "SURN", Value: "Doe"},
					{Level: 1, Tag: "SEX", Value: "M"},
					{Level: 1, Tag: "BIRT"},
					{Level: 2, Tag: "DATE", Value: "1 JAN 1900"},
					{Level: 2, Tag: "PLAC", Value: "New York, USA"},
				},
				Entity: &gedcom.Individual{
					XRef: "@I1@",
					Names: []*gedcom.PersonalName{
						{Full: "John /Doe/", Given: "John", Surname: "Doe"},
					},
					Sex: "M",
				},
			},
			// Another individual
			{
				XRef: "@I2@",
				Type: gedcom.RecordTypeIndividual,
				Tags: []*gedcom.Tag{
					{Level: 1, Tag: "NAME", Value: "Jane /Smith/"},
					{Level: 2, Tag: "GIVN", Value: "Jane"},
					{Level: 2, Tag: "SURN", Value: "Smith"},
					{Level: 1, Tag: "SEX", Value: "F"},
				},
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
				Tags: []*gedcom.Tag{
					{Level: 1, Tag: "HUSB", Value: "@I1@"},
					{Level: 1, Tag: "WIFE", Value: "@I2@"},
					{Level: 1, Tag: "MARR"},
					{Level: 2, Tag: "DATE", Value: "15 JUN 1925"},
					{Level: 2, Tag: "PLAC", Value: "Boston, Massachusetts, USA"},
				},
				Entity: &gedcom.Family{
					XRef:    "@F1@",
					Husband: "@I1@",
					Wife:    "@I2@",
				},
			},
		},
		XRefMap: make(map[string]*gedcom.Record),
	}

	// Build the XRefMap
	for _, record := range doc.Records {
		if record.XRef != "" {
			doc.XRefMap[record.XRef] = record
		}
	}

	// Encode to file or stdout
	var out *os.File
	var err error

	if len(os.Args) > 1 {
		filename := os.Args[1]
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
