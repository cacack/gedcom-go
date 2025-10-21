// Example: Query and navigate GEDCOM data
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/cacack/gedcom-go/decoder"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <gedcom_file>")
		fmt.Println("Example: go run main.go ../../testdata/gedcom-5.5/royal92.ged")
		os.Exit(1)
	}

	filename := os.Args[1]

	// Open and parse GEDCOM file
	f, err := os.Open(filename)
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
	}
	defer f.Close()

	doc, err := decoder.Decode(f)
	if err != nil {
		log.Fatalf("Failed to decode GEDCOM: %v", err)
	}

	fmt.Printf("Querying GEDCOM File: %s\n", filename)
	fmt.Printf("Version: %s\n\n", doc.Header.Version)

	// Example 1: List all individuals with their names
	fmt.Println("=== All Individuals ===")
	individuals := doc.Individuals()
	fmt.Printf("Found %d individuals:\n", len(individuals))
	for i, ind := range individuals {
		if i < 10 { // Show first 10
			name := ""
			if len(ind.Names) > 0 {
				name = ind.Names[0].Full
			}
			fmt.Printf("%s: %s", ind.XRef, name)
			if ind.Sex != "" {
				fmt.Printf(" (%s)", ind.Sex)
			}
			// Show birth year if available
			for _, event := range ind.Events {
				if event.Type == "BIRT" && event.Date != "" {
					fmt.Printf(" - Born: %s", event.Date)
					break
				}
			}
			fmt.Println()
		}
	}
	if len(individuals) > 10 {
		fmt.Printf("... and %d more\n", len(individuals)-10)
	}

	// Example 2: Look up specific individual by XRef
	fmt.Println("\n=== Lookup by Cross-Reference ===")
	if len(individuals) > 0 {
		targetXRef := individuals[0].XRef
		person := doc.GetIndividual(targetXRef)
		if person != nil {
			fmt.Printf("Found individual %s:\n", targetXRef)
			if len(person.Names) > 0 {
				fmt.Printf("  Name: %s\n", person.Names[0].Full)
			}
			fmt.Printf("  Sex: %s\n", person.Sex)

			// Show events
			if len(person.Events) > 0 {
				fmt.Println("  Events:")
				for _, event := range person.Events {
					fmt.Printf("    %s", event.Type)
					if event.Date != "" {
						fmt.Printf(": %s", event.Date)
					}
					if event.Place != "" {
						fmt.Printf(" at %s", event.Place)
					}
					fmt.Println()
				}
			}

			// Show family relationships
			if len(person.ChildInFamilies) > 0 {
				fmt.Printf("  Child in families: %v\n", person.ChildInFamilies)
			}
			if len(person.SpouseInFamilies) > 0 {
				fmt.Printf("  Spouse in families: %v\n", person.SpouseInFamilies)
			}
		}
	}

	// Example 3: List all families
	fmt.Println("\n=== All Families ===")
	families := doc.Families()
	fmt.Printf("Found %d families:\n", len(families))
	for i, fam := range families {
		if i < 5 { // Show first 5
			fmt.Printf("%s:", fam.XRef)

			// Get husband and wife names
			if fam.Husband != "" {
				husband := doc.GetIndividual(fam.Husband)
				if husband != nil && len(husband.Names) > 0 {
					fmt.Printf(" %s", husband.Names[0].Full)
				}
			}

			if fam.Wife != "" {
				wife := doc.GetIndividual(fam.Wife)
				if wife != nil && len(wife.Names) > 0 {
					if fam.Husband != "" {
						fmt.Printf(" & %s", wife.Names[0].Full)
					} else {
						fmt.Printf(" %s", wife.Names[0].Full)
					}
				}
			}

			if len(fam.Children) > 0 {
				fmt.Printf(" (%d children)", len(fam.Children))
			}
			fmt.Println()
		}
	}
	if len(families) > 5 {
		fmt.Printf("... and %d more\n", len(families)-5)
	}

	// Example 4: List all sources
	fmt.Println("\n=== All Sources ===")
	sources := doc.Sources()
	fmt.Printf("Found %d sources:\n", len(sources))
	for i, src := range sources {
		if i < 5 { // Show first 5
			fmt.Printf("%s: %s\n", src.XRef, src.Title)
		}
	}
	if len(sources) > 5 {
		fmt.Printf("... and %d more\n", len(sources)-5)
	}

	// Example 5: Direct record access by XRef
	fmt.Println("\n=== Direct Record Access ===")
	if len(doc.Records) > 0 {
		firstRecord := doc.Records[0]
		if firstRecord.XRef != "" {
			record := doc.GetRecord(firstRecord.XRef)
			if record != nil {
				fmt.Printf("Record %s (Type: %s):\n", record.XRef, record.Type)
				fmt.Printf("  Has %d tags\n", len(record.Tags))
			}
		}
	}
}
