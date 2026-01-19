// Package parser provides low-level GEDCOM line parsing functionality.
//
// This package handles the tokenization and parsing of individual GEDCOM lines,
// converting them into Line structures with level, tag, value, and cross-reference
// information. It supports all standard GEDCOM formats and provides detailed error
// reporting with line numbers.
//
// Example usage:
//
//	p := parser.NewParser(reader)
//	for {
//	    line, err := p.ParseLine()
//	    if err == io.EOF {
//	        break
//	    }
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//	    fmt.Printf("Level %d: %s = %s\n", line.Level, line.Tag, line.Value)
//	}
package parser
