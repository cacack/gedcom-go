// Package main demonstrates querying GEDCOM data including individual lookups, family traversal, and relationship navigation.
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <gedcom_file>")
		fmt.Println("Example: go run main.go ../../testdata/gedcom-5.5/royal92.ged")
		os.Exit(1)
	}

	filename := filepath.Clean(os.Args[1])

	// Open and parse GEDCOM file
	// #nosec G703 -- the path is this CLI's own argv, not attacker-controlled input; see the package comment on validating against an allowed root in production.
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
			fmt.Printf("%s: %s", ind.XRef, ind.FullName())
			if ind.Sex != "" {
				fmt.Printf(" (%s)", ind.Sex)
			}
			// Show birth date if available. OccurredEvents skips GEDCOM 7.0
			// negative assertions (NO BIRT), which record a birth that did
			// not happen.
			for _, event := range ind.OccurredEvents() {
				if event.Type == gedcom.EventBirth && event.Date != "" {
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
			if name := person.FullName(); name != "" {
				fmt.Printf("  Name: %s\n", name)
			}
			fmt.Printf("  Sex: %s\n", person.Sex)

			// Show events. Events also holds GEDCOM 7.0 negative assertions
			// (NO <EVENT>, IsNegative), which record that an event did NOT
			// happen; OccurredEvents leaves them out, so they are listed
			// separately below instead of being printed as facts.
			if events := person.OccurredEvents(); len(events) > 0 {
				fmt.Println("  Events:")
				for _, event := range events {
					fmt.Printf("    %s", event.Type)
					if event.Date != "" {
						fmt.Printf(": %s", event.Date)
					}
					if place := event.PlaceName(); place != "" {
						fmt.Printf(" at %s", place)
					}
					fmt.Println()
				}
			}
			if negated := person.NegativeAssertions(); len(negated) > 0 {
				fmt.Println("  Did not happen (negative assertions):")
				for _, event := range negated {
					fmt.Printf("    NO %s", event.Type)
					if event.Date != "" {
						fmt.Printf(": %s", event.Date)
					}
					fmt.Println()
				}
			}

			// Show family relationships
			if len(person.ChildInFamilies) > 0 {
				childFams := make([]string, len(person.ChildInFamilies))
				for i, link := range person.ChildInFamilies {
					childFams[i] = link.FamilyXRef
				}
				fmt.Printf("  Child in families: %v\n", childFams)
			}
			if len(person.SpouseInFamilies) > 0 {
				spouseFams := make([]string, len(person.SpouseInFamilies))
				for i, link := range person.SpouseInFamilies {
					spouseFams[i] = link.FamilyXRef
				}
				fmt.Printf("  Spouse in families: %v\n", spouseFams)
			}
		}
	}

	// Example 3: List all families
	// HusbandIndividual/WifeIndividual resolve the spouse XRefs to typed
	// *Individual values, so no manual doc.GetIndividual lookup is needed.
	fmt.Println("\n=== All Families ===")
	families := doc.Families()
	fmt.Printf("Found %d families:\n", len(families))
	for i, fam := range families {
		if i < 5 { // Show first 5
			fmt.Printf("%s:", fam.XRef)

			// FullName is nil-safe, so an unresolved or nameless spouse yields "".
			husband := fam.HusbandIndividual(doc).FullName()
			if husband != "" {
				fmt.Printf(" %s", husband)
			}

			if wife := fam.WifeIndividual(doc).FullName(); wife != "" {
				if husband != "" {
					fmt.Printf(" & %s", wife)
				} else {
					fmt.Printf(" %s", wife)
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

	// Example 6: Relationship Traversal API
	// Parents, Spouses, Children and AllMembers follow the XRef links for you
	// and return typed *Individual values.
	fmt.Println("\n=== Relationship Traversal ===")

	// Find an individual with family relationships to demonstrate traversal
	var targetPerson *gedcom.Individual
	for _, ind := range individuals {
		// Find someone who is both a child and a spouse (has rich relationships)
		if len(ind.ChildInFamilies) > 0 && len(ind.SpouseInFamilies) > 0 {
			targetPerson = ind
			break
		}
	}

	if targetPerson != nil {
		fmt.Printf("Traversing relationships for: %s (%s)\n", targetPerson.FullName(), targetPerson.XRef)

		// Parents across every family the person is a child in
		parents := targetPerson.Parents(doc)
		fmt.Printf("\n  Parents (%d):\n", len(parents))
		for _, parent := range parents {
			fmt.Printf("    - %s (%s)\n", parent.FullName(), parent.XRef)
		}

		// Spouses across every family the person is a spouse in (handles remarriage)
		spouses := targetPerson.Spouses(doc)
		fmt.Printf("\n  Spouses (%d):\n", len(spouses))
		for _, spouse := range spouses {
			fmt.Printf("    - %s (%s)\n", spouse.FullName(), spouse.XRef)
		}

		// Children across every family the person is a spouse in
		children := targetPerson.Children(doc)
		fmt.Printf("\n  Children (%d):\n", len(children))
		for _, child := range children {
			fmt.Printf("    - %s (%s)\n", child.FullName(), child.XRef)
		}

		// Demonstrate Family traversal methods
		if len(families) > 0 {
			// Find a family with children for a good demo
			var demoFamily *gedcom.Family
			for _, fam := range families {
				if len(fam.Children) > 0 && fam.Husband != "" && fam.Wife != "" {
					demoFamily = fam
					break
				}
			}

			if demoFamily != nil {
				fmt.Printf("\n  Family %s members using AllMembers():\n", demoFamily.XRef)
				for _, member := range demoFamily.AllMembers(doc) {
					fmt.Printf("    - %s (%s)\n", member.FullName(), member.XRef)
				}
			}
		}
	} else {
		fmt.Println("No individual with both parent and spouse relationships found.")
	}
}
