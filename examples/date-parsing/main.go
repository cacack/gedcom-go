package main

import (
	"fmt"
	"log"

	"github.com/cacack/gedcom-go/gedcom"
)

func main() {
	// Example date strings in GEDCOM format
	examples := []string{
		"25 DEC 2020",
		"JAN 1900",
		"1850",
		"ABT 1850",
		"BEF 12 JUN 1900",
		"AFT 1 JAN 1850",
		"BET 1850 AND 1860",
		"FROM 1880 TO 1920",
		"FROM 1914",
		"TO 1918",
	}

	fmt.Println("GEDCOM Date Parsing Examples (Phase 1: Gregorian Calendar)")
	fmt.Println("==========================================================")
	fmt.Println()

	for _, dateStr := range examples {
		date, err := gedcom.ParseDate(dateStr)
		if err != nil {
			log.Printf("Error parsing '%s': %v\n", dateStr, err)
			continue
		}

		fmt.Printf("Input: %s\n", dateStr)
		fmt.Printf("  Parsed: Year=%d, Month=%d, Day=%d\n", date.Year, date.Month, date.Day)
		fmt.Printf("  Modifier: %s\n", date.Modifier)
		fmt.Printf("  Calendar: %s\n", date.Calendar)

		if date.EndDate != nil {
			fmt.Printf("  End Date: Year=%d, Month=%d, Day=%d\n",
				date.EndDate.Year, date.EndDate.Month, date.EndDate.Day)
		}

		// Try to convert to time.Time (only works for complete dates)
		if t, err := date.ToTime(); err == nil {
			fmt.Printf("  As time.Time: %s\n", t.Format("2006-01-02"))
		}

		fmt.Println()
	}

	// Demonstrate date comparison
	fmt.Println("Date Comparison Examples")
	fmt.Println("========================")
	fmt.Println()

	comparisons := []struct {
		date1, date2 string
	}{
		{"1850", "1860"},
		{"JAN 1900", "FEB 1900"},
		{"25 DEC 2020", "26 DEC 2020"},
		{"1920", "1 JAN 1920"},
	}

	for _, cmp := range comparisons {
		d1, _ := gedcom.ParseDate(cmp.date1)
		d2, _ := gedcom.ParseDate(cmp.date2)

		result := d1.Compare(d2)
		var comparison string
		switch result {
		case -1:
			comparison = "<"
		case 0:
			comparison = "="
		case 1:
			comparison = ">"
		}

		fmt.Printf("%s %s %s\n", cmp.date1, comparison, cmp.date2)
	}
}
