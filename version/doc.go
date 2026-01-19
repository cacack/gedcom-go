// Package version provides GEDCOM version detection and validation.
//
// This package helps identify which GEDCOM specification version (5.5, 5.5.1, or 7.0)
// a file conforms to. It can detect the version from the header or use tag-based
// heuristics to make an educated guess.
//
// Example usage:
//
//	lines, _ := parser.Parse(reader)
//	version, err := version.DetectVersion(lines)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Detected GEDCOM version: %s\n", version)
package version
