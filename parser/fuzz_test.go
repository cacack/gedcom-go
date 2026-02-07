package parser

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// FuzzParse fuzzes the Parse method with arbitrary byte input.
// Seeds are loaded from testdata GEDCOM files to give the fuzzer
// realistic starting points.
func FuzzParse(f *testing.F) {
	// Seed from testdata files
	seedDirs := []string{
		"../testdata/gedcom-5.5",
		"../testdata/gedcom-5.5.1",
		"../testdata/gedcom-7.0",
		"../testdata/malformed",
		"../testdata/edge-cases",
	}

	for _, dir := range seedDirs {
		files, err := filepath.Glob(filepath.Join(dir, "*.ged"))
		if err != nil {
			continue
		}
		for _, file := range files {
			data, err := os.ReadFile(file)
			if err != nil {
				continue
			}
			f.Add(data)
		}
	}

	// Also seed some synthetic edge cases
	f.Add([]byte("0 HEAD\n0 TRLR\n"))
	f.Add([]byte("0 HEAD\r\n1 GEDC\r\n2 VERS 5.5\r\n0 TRLR\r\n"))
	f.Add([]byte("0 HEAD\r1 SOUR Test\r0 TRLR\r"))
	f.Add([]byte(""))
	f.Add([]byte("\n\n\n"))
	f.Add([]byte("0 @I1@ INDI\n1 NAME John /Smith/\n"))

	f.Fuzz(func(t *testing.T, data []byte) {
		p := NewParser()
		// Errors are expected; panics are not.
		_, _ = p.Parse(bytes.NewReader(data))
	})
}

// FuzzParseLine fuzzes the ParseLine method with arbitrary string input.
func FuzzParseLine(f *testing.F) {
	// Seed with known valid and invalid lines
	seeds := []string{
		"0 HEAD",
		"0 @I1@ INDI",
		"1 NAME John /Smith/",
		"2 GIVN John",
		"1 NOTE This is a note with spaces",
		"1 HUSB @I1@",
		"1 SEX",
		"0 TRLR",
		"0 HEAD\r\n",
		"0 HEAD\n",
		"0 HEAD\r",
		"",
		"   ",
		"0",
		"-1 HEAD",
		"X HEAD",
		"0 @I1@",
		"99999999999999 TAG",
		"0 @@ TAG",
		"0 @ TAG",
		"1 CONT This is a continuation",
		"1 CONC This is concatenated",
		"0 @F1@ FAM",
		"1 NAME    John",
		"1 NAME John  Smith",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		p := NewParser()
		// Errors are expected; panics are not.
		_, _ = p.ParseLine(input)
	})
}
