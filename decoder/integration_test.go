package decoder

import (
	"os"
	"path/filepath"
	"testing"
)

// T058: Parse all sample files in testdata/ and verify success
func TestParseRealGEDCOMFiles(t *testing.T) {
	testFiles := []string{
		"../testdata/gedcom-5.5/minimal.ged",
		"../testdata/gedcom-5.5.1/minimal.ged",
		"../testdata/gedcom-7.0/minimal.ged",
	}

	for _, testFile := range testFiles {
		t.Run(filepath.Base(testFile), func(t *testing.T) {
			f, err := os.Open(testFile)
			if err != nil {
				t.Skipf("Test file not found: %s", testFile)
				return
			}
			defer f.Close()

			doc, err := Decode(f)
			if err != nil {
				t.Fatalf("Decode() error = %v for file %s", err, testFile)
			}

			if doc == nil {
				t.Fatal("Decode() returned nil document")
			}

			t.Logf("Successfully parsed %s: %d records", testFile, len(doc.Records))
		})
	}
}
