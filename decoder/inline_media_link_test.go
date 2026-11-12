package decoder

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// decodeInlineMediaDoc decodes body under a header declaring vers.
func decodeInlineMediaDoc(t *testing.T, vers, body string) *DecodeResult {
	t.Helper()
	input := "0 HEAD\n1 GEDC\n2 VERS " + vers + "\n" + body + "\n0 TRLR"
	result, err := DecodeWithDiagnostics(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("DecodeWithDiagnostics() error = %v", err)
	}
	return result
}

// TestInlineMediaLinkFiles pins issue #733: an inline OBJE link (no pointer)
// keeps its FILE, FORM and MEDI in MediaLink.Files, in both the 5.5.1 layout
// (FORM under FILE) and the 5.5 one (FORM beside FILE).
func TestInlineMediaLinkFiles(t *testing.T) {
	tests := []struct {
		name        string
		vers        string
		body        string
		wantFiles   []gedcom.MediaFile
		wantTitle   string
		wantUnknown []string
	}{
		{
			name: "5.5.1 layout",
			vers: "5.5.1",
			body: `0 @I1@ INDI
1 OBJE
2 FILE C:\photos\a.jpg
3 FORM jpg
4 MEDI photo
2 FILE b.png
3 FORM png
2 TITL Pic`,
			wantFiles: []gedcom.MediaFile{
				{FileRef: `C:\photos\a.jpg`, Form: "jpg", MediaType: "photo"},
				{FileRef: "b.png", Form: "png"},
			},
			wantTitle: "Pic",
		},
		{
			name: "5.5 layout in a 5.5 document",
			vers: "5.5",
			body: `0 @I1@ INDI
1 OBJE
2 FORM jpg
2 TITL Pic
2 FILE C:\photos\a.jpg`,
			wantFiles: []gedcom.MediaFile{{FileRef: `C:\photos\a.jpg`, Form: "jpg"}},
			wantTitle: "Pic",
		},
		{
			// The FORM is typed whatever the version, and still reported in a
			// document declaring another, as MyHeritage FTB 8 writes it.
			name: "5.5 layout in a 5.5.1 document",
			vers: "5.5.1",
			body: `0 @I1@ INDI
1 OBJE
2 FORM jpg
2 TITL Pic
2 FILE a.jpg`,
			wantFiles:   []gedcom.MediaFile{{FileRef: "a.jpg", Form: "jpg"}},
			wantTitle:   "Pic",
			wantUnknown: []string{"line 6: unknown tag: FORM"},
		},
		{
			name: "5.5 FORM with no FILE",
			vers: "5.5",
			body: `0 @I1@ INDI
1 OBJE
2 FORM jpg
2 TITL Pic`,
			wantFiles: []gedcom.MediaFile{{Form: "jpg"}},
			wantTitle: "Pic",
		},
		{
			name: "pointer link has no files",
			vers: "5.5.1",
			body: `0 @I1@ INDI
1 OBJE @O1@
0 @O1@ OBJE
1 FILE a.jpg
2 FORM jpg`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := decodeInlineMediaDoc(t, tt.vers, tt.body)
			if got := unknownTagMessages(result.Diagnostics); !reflect.DeepEqual(got, tt.wantUnknown) {
				t.Errorf("UNKNOWN_TAG = %q, want %q", got, tt.wantUnknown)
			}
			indi := result.Document.GetIndividual("@I1@")
			if indi == nil || len(indi.Media) != 1 {
				t.Fatalf("want one media link on @I1@, got %+v", indi)
			}
			link := indi.Media[0]
			if link.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", link.Title, tt.wantTitle)
			}
			var got []gedcom.MediaFile
			for _, f := range link.Files {
				got = append(got, *f)
			}
			if !reflect.DeepEqual(got, tt.wantFiles) {
				t.Errorf("Files = %+v, want %+v", got, tt.wantFiles)
			}
		})
	}
}

// TestMediaRecordFormType pins the 5.5.1 OBJE record's FORM.TYPE, which that
// version writes where the link has MEDI, as the file's MediaType (#733).
func TestMediaRecordFormType(t *testing.T) {
	result := decodeInlineMediaDoc(t, "5.5.1", `0 @O1@ OBJE
1 FILE a.jpg
2 FORM jpg
3 TYPE photo`)
	media := result.Document.GetMediaObject("@O1@")
	if media == nil || len(media.Files) != 1 {
		t.Fatalf("want one file on @O1@, got %+v", media)
	}
	if got := media.Files[0].MediaType; got != "photo" {
		t.Errorf("MediaType = %q, want %q", got, "photo")
	}
}

// TestInlineMediaLinkCorpus checks the three corpus files with inline links
// named in #733: every inline link's file path reaches MediaLink.Files.
func TestInlineMediaLinkCorpus(t *testing.T) {
	tests := []struct {
		path      string
		links     int // inline links with a FILE in the file
		xref      string
		index     int
		wantFile  gedcom.MediaFile
		wantTitle string
	}{
		{
			path:      "../testdata/edge-cases/ftm-photo-test.ged",
			links:     3,
			xref:      "@I3@",
			index:     2,
			wantFile:  gedcom.MediaFile{FileRef: "O5.jpg", Form: "jpeg"},
			wantTitle: "Edwin & Janice Smith",
		},
		{
			path:      "../testdata/edge-cases/vendor-legacy.ged",
			links:     1,
			xref:      "@I2@",
			wantFile:  gedcom.MediaFile{FileRef: "browndavid.jpg", Form: "jpg"},
			wantTitle: "Somebody else",
		},
		{
			path:      "../testdata/edge-cases/mhftb8-export.ged",
			links:     10,
			xref:      "@I480@",
			wantFile:  gedcom.MediaFile{FileRef: `c:\users\merri\documents\myheritage\2017 corew\Photos\`, Form: "jpg"},
			wantTitle: "Portrait Roi de france Clodion",
		},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			f, err := os.Open(tt.path)
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			defer f.Close()
			doc, err := Decode(f)
			if err != nil {
				t.Fatalf("Decode() error = %v", err)
			}

			inline := 0
			for _, indi := range doc.Individuals() {
				for _, link := range indi.Media {
					if link.MediaXRef != "" {
						continue
					}
					if len(link.Files) == 0 || link.Files[0].FileRef == "" {
						t.Errorf("%s: inline link %q has no file path", indi.XRef, link.Title)
						continue
					}
					inline++
				}
			}
			if inline != tt.links {
				t.Errorf("inline links with a file = %d, want %d", inline, tt.links)
			}

			indi := doc.GetIndividual(tt.xref)
			if indi == nil || len(indi.Media) <= tt.index {
				t.Fatalf("%s: want media link %d, got %+v", tt.xref, tt.index, indi)
			}
			link := indi.Media[tt.index]
			if link.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", link.Title, tt.wantTitle)
			}
			if len(link.Files) != 1 || !reflect.DeepEqual(*link.Files[0], tt.wantFile) {
				t.Errorf("Files = %+v, want [%+v]", link.Files, tt.wantFile)
			}
		})
	}
}
