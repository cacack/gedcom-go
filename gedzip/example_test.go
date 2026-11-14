package gedzip_test

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing/fstest"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedzip"
)

// exampleGED is a GEDCOM 7.0 dataset with one local file, whose name needs
// percent-escaping, and one web file.
const exampleGED = `0 HEAD
1 GEDC
2 VERS 7.0
0 @O1@ OBJE
1 FILE media/family%20portrait.txt
2 FORM text/plain
1 FILE https://example.com/portrait.jpg
2 FORM image/jpeg
0 @I1@ INDI
1 NAME Ada /Lovelace/
1 OBJE @O1@
0 TRLR
`

// exampleArchive builds a small GEDZIP archive in memory: gedcom.ged and the
// one local file it references, stored under its unescaped name.
func exampleArchive() []byte {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, f := range []struct{ name, data string }{
		{"gedcom.ged", exampleGED},
		{"media/family portrait.txt", "A portrait of the family."},
	} {
		w, err := zw.Create(f.name)
		if err != nil {
			panic(err)
		}
		if _, err := io.WriteString(w, f.data); err != nil {
			panic(err)
		}
	}
	if err := zw.Close(); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

// ExampleNewReader reads an archive held in memory: it decodes the dataset,
// checks that the archive holds every local file the dataset references, and
// opens one by its FILE payload. Use OpenReader for an archive on disk.
func ExampleNewReader() {
	archive := exampleArchive()

	// A nil *ReadOptions applies the default limits.
	r, err := gedzip.NewReader(bytes.NewReader(archive), int64(len(archive)), nil)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	doc, err := r.Decode(nil)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Individuals: %d\n", len(doc.Individuals()))
	fmt.Printf("Unresolved references: %d\n", len(r.CheckRefs(doc)))

	for _, media := range doc.MediaObjects() {
		for _, file := range media.Files {
			kind, _ := gedzip.ClassifyFileRef(file.FileRef)
			if kind != gedzip.RefLocal {
				fmt.Printf("%s: %s, not in the archive\n", file.FileRef, kind)
				continue
			}
			// Media is opened by FILE payload and read only on request.
			rc, err := r.Open(file.FileRef)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				return
			}
			content, err := io.ReadAll(rc)
			_ = rc.Close()
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				return
			}
			fmt.Printf("%s: %q\n", file.FileRef, content)
		}
	}

	// Output:
	// Individuals: 1
	// Unresolved references: 0
	// media/family%20portrait.txt: "A portrait of the family."
	// https://example.com/portrait.jpg: web, not in the archive
}

// ExampleWrite packages a GEDCOM 7.0 document and the local files it
// references. The media source is any fs.FS: a fstest.MapFS here, os.DirFS
// for a directory, or the FS of another archive's Reader.
func ExampleWrite() {
	doc, err := decoder.Decode(strings.NewReader(exampleGED))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Files are named as the archive names them: the FILE payload with its
	// percent-escapes decoded.
	media := fstest.MapFS{
		"media/family portrait.txt": {Data: []byte("A portrait of the family.")},
		"media/unreferenced.txt":    {Data: []byte("Nothing references this, so it is left out.")},
	}

	var buf bytes.Buffer
	if err := gedzip.Write(&buf, doc, media, nil); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Read the archive back to see what it holds.
	r, err := gedzip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()), nil)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	for _, entry := range r.Entries() {
		fmt.Println(entry.Name)
	}

	// Output:
	// gedcom.ged
	// media/family portrait.txt
}
