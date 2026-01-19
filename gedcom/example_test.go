package gedcom_test

import (
	"fmt"
	"strings"

	"github.com/cacack/gedcom-go/decoder"
	"github.com/cacack/gedcom-go/gedcom"
)

// ExampleDocument demonstrates accessing records in a GEDCOM document.
func ExampleDocument() {
	gedcomData := `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME John /Smith/
1 SEX M
0 @I2@ INDI
1 NAME Jane /Doe/
1 SEX F
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
0 TRLR`

	doc, _ := decoder.Decode(strings.NewReader(gedcomData))

	// Access all individuals
	fmt.Printf("Individuals: %d\n", len(doc.Individuals()))

	// Access all families
	fmt.Printf("Families: %d\n", len(doc.Families()))

	// Output:
	// Individuals: 2
	// Families: 1
}

// ExampleDocument_GetIndividual shows XRef-based individual lookup.
func ExampleDocument_GetIndividual() {
	gedcomData := `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME Alice /Johnson/
1 SEX F
0 @I2@ INDI
1 NAME Bob /Johnson/
1 SEX M
0 TRLR`

	doc, _ := decoder.Decode(strings.NewReader(gedcomData))

	// Lookup by XRef
	alice := doc.GetIndividual("@I1@")
	if alice != nil && len(alice.Names) > 0 {
		fmt.Printf("Found: %s\n", alice.Names[0].Full)
	}

	// Non-existent XRef returns nil
	unknown := doc.GetIndividual("@I999@")
	fmt.Printf("Unknown is nil: %v\n", unknown == nil)

	// Output:
	// Found: Alice /Johnson/
	// Unknown is nil: true
}

// ExampleDocument_GetFamily shows XRef-based family lookup.
func ExampleDocument_GetFamily() {
	gedcomData := `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME John /Smith/
0 @I2@ INDI
1 NAME Jane /Smith/
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
0 TRLR`

	doc, _ := decoder.Decode(strings.NewReader(gedcomData))

	// Lookup family
	family := doc.GetFamily("@F1@")
	if family != nil {
		fmt.Printf("Husband XRef: %s\n", family.Husband)
		fmt.Printf("Wife XRef: %s\n", family.Wife)
	}

	// Output:
	// Husband XRef: @I1@
	// Wife XRef: @I2@
}

// ExampleDocument_traversal shows how to traverse family relationships.
func ExampleDocument_traversal() {
	gedcomData := `0 HEAD
1 GEDC
2 VERS 5.5
0 @I1@ INDI
1 NAME John /Smith/
0 @I2@ INDI
1 NAME Jane /Smith/
0 @I3@ INDI
1 NAME Jimmy /Smith/
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 CHIL @I3@
0 TRLR`

	doc, _ := decoder.Decode(strings.NewReader(gedcomData))

	// Get a family and resolve its members
	family := doc.GetFamily("@F1@")
	if family != nil {
		husband := doc.GetIndividual(family.Husband)
		wife := doc.GetIndividual(family.Wife)

		if husband != nil && len(husband.Names) > 0 {
			fmt.Printf("Husband: %s\n", husband.Names[0].Full)
		}
		if wife != nil && len(wife.Names) > 0 {
			fmt.Printf("Wife: %s\n", wife.Names[0].Full)
		}

		// List children
		for _, childXRef := range family.Children {
			child := doc.GetIndividual(childXRef)
			if child != nil && len(child.Names) > 0 {
				fmt.Printf("Child: %s\n", child.Names[0].Full)
			}
		}
	}

	// Output:
	// Husband: John /Smith/
	// Wife: Jane /Smith/
	// Child: Jimmy /Smith/
}

// ExampleParseDate demonstrates parsing various date formats.
func ExampleParseDate() {
	// Exact date
	date1, _ := gedcom.ParseDate("25 DEC 2020")
	fmt.Printf("Day: %d, Month: %d, Year: %d\n", date1.Day, date1.Month, date1.Year)

	// Partial date (year only)
	date2, _ := gedcom.ParseDate("1850")
	fmt.Printf("Year only: %d\n", date2.Year)

	// Approximate date
	date3, _ := gedcom.ParseDate("ABT 1920")
	fmt.Printf("Approximate year: %d, Modifier: %s\n", date3.Year, date3.Modifier)

	// Output:
	// Day: 25, Month: 12, Year: 2020
	// Year only: 1850
	// Approximate year: 1920, Modifier: ABT
}

// ExampleParseDate_ranges shows parsing date ranges and periods.
func ExampleParseDate_ranges() {
	// Date range (BET...AND)
	rangeDate, _ := gedcom.ParseDate("BET 1850 AND 1860")
	fmt.Printf("Between: %d and %d\n", rangeDate.Year, rangeDate.EndDate.Year)

	// Date period (FROM...TO)
	periodDate, _ := gedcom.ParseDate("FROM 1880 TO 1920")
	fmt.Printf("From: %d to %d\n", periodDate.Year, periodDate.EndDate.Year)

	// Before date
	beforeDate, _ := gedcom.ParseDate("BEF 1900")
	fmt.Printf("Before: %d, Modifier: %s\n", beforeDate.Year, beforeDate.Modifier)

	// Output:
	// Between: 1850 and 1860
	// From: 1880 to 1920
	// Before: 1900, Modifier: BEF
}

// ExampleDate_Compare demonstrates date comparison.
func ExampleDate_Compare() {
	earlier, _ := gedcom.ParseDate("15 MAR 1920")
	later, _ := gedcom.ParseDate("20 JUN 1985")

	result := earlier.Compare(later)
	if result < 0 {
		fmt.Println("1920 is before 1985")
	}

	// Use convenience methods
	fmt.Printf("IsBefore: %v\n", earlier.IsBefore(later))
	fmt.Printf("IsAfter: %v\n", earlier.IsAfter(later))

	// Output:
	// 1920 is before 1985
	// IsBefore: true
	// IsAfter: false
}

// ExampleDate_ToTime shows converting a date to time.Time.
func ExampleDate_ToTime() {
	date, _ := gedcom.ParseDate("4 JUL 1776")

	t, err := date.ToTime()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Year: %d, Month: %s, Day: %d\n", t.Year(), t.Month(), t.Day())

	// Output:
	// Year: 1776, Month: July, Day: 4
}

// ExampleYearsBetween demonstrates calculating age from dates.
func ExampleYearsBetween() {
	birth, _ := gedcom.ParseDate("15 MAR 1920")
	death, _ := gedcom.ParseDate("20 JUN 1985")

	years, exact, err := gedcom.YearsBetween(birth, death)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Years: %d, Exact calculation: %v\n", years, exact)

	// Output:
	// Years: 65, Exact calculation: true
}

// ExampleDate_Validate shows date validation.
func ExampleDate_Validate() {
	// Valid date
	validDate, _ := gedcom.ParseDate("28 FEB 2020")
	if err := validDate.Validate(); err == nil {
		fmt.Println("Feb 28, 2020 is valid")
	}

	// Invalid date (Feb 30 doesn't exist)
	invalidDate := &gedcom.Date{Day: 30, Month: 2, Year: 2020}
	if err := invalidDate.Validate(); err != nil {
		fmt.Println("Feb 30 is invalid")
	}

	// Output:
	// Feb 28, 2020 is valid
	// Feb 30 is invalid
}
