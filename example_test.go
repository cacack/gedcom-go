package gedcomgo_test

import (
	"errors"
	"fmt"
	"strings"

	gedcomgo "github.com/cacack/gedcom-go/v3"
	"github.com/cacack/gedcom-go/v3/validator"
)

// ExampleDecode shows decoding a GEDCOM file and walking its individuals and
// families.
func ExampleDecode() {
	gedcomData := `0 HEAD
1 GEDC
2 VERS 5.5.1
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Smith/
1 BIRT
2 DATE 1 JAN 1900
0 @I2@ INDI
1 NAME Mary /Jones/
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
0 TRLR`

	doc, err := gedcomgo.Decode(strings.NewReader(gedcomData))
	if err != nil {
		fmt.Printf("Decode: %v\n", err)
		return
	}

	fmt.Printf("Version: %s\n", doc.Header.Version)
	for _, ind := range doc.Individuals() {
		fmt.Printf("%s %s\n", ind.XRef, ind.FullName())
	}
	for _, fam := range doc.Families() {
		husband := doc.GetIndividual(fam.Husband)
		wife := doc.GetIndividual(fam.Wife)
		fmt.Printf("%s: %s & %s\n", fam.XRef, husband.FullName(), wife.FullName())
	}

	// Output:
	// Version: 5.5.1
	// @I1@ John /Smith/
	// @I2@ Mary /Jones/
	// @F1@: John /Smith/ & Mary /Jones/
}

// ExampleValidate shows validating a document and recovering each finding's
// code and severity with errors.As.
func ExampleValidate() {
	gedcomData := `0 HEAD
1 GEDC
2 VERS 7.0
0 @I1@ INDI
1 NAME John /Smith/
1 FAMC @F9@
0 TRLR`

	doc, err := gedcomgo.Decode(strings.NewReader(gedcomData))
	if err != nil {
		fmt.Printf("Decode: %v\n", err)
		return
	}

	for _, err := range gedcomgo.Validate(doc) {
		var issue *validator.Issue
		if errors.As(err, &issue) {
			fmt.Printf("%s %s: %s\n", issue.Severity, issue.Code, issue.Message)
		}
	}

	// Output:
	// ERROR ORPHANED_FAMC: FAMC reference to non-existent family @F9@
}

// ExampleConvert shows converting a document to GEDCOM 7.0 and reading the
// conversion report.
func ExampleConvert() {
	gedcomData := `0 HEAD
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Smith/
1 NOTE @N1@
0 @N1@ NOTE A shared note
0 TRLR`

	doc, err := gedcomgo.Decode(strings.NewReader(gedcomData))
	if err != nil {
		fmt.Printf("Decode: %v\n", err)
		return
	}

	converted, report, err := gedcomgo.Convert(doc, gedcomgo.Version70)
	if err != nil {
		fmt.Printf("Convert: %v\n", err)
		return
	}

	fmt.Printf("%s -> %s, success: %t\n", report.SourceVersion, report.TargetVersion, report.Success)
	fmt.Printf("Converted version: %s\n", converted.Header.Version)
	fmt.Printf("Original version: %s\n", doc.Header.Version)
	for _, n := range report.Normalized {
		fmt.Printf("Normalized %s: %s -> %s\n", n.Path, n.Original, n.Result)
	}

	// Output:
	// 5.5.1 -> 7.0, success: true
	// Converted version: 7.0
	// Original version: 5.5.1
	// Normalized Note @N1@: NOTE -> SNOTE
}
