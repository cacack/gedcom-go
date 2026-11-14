// Package main demonstrates GEDZIP archives: packaging a GEDCOM 7 document with the media it references, and reading one back.
package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedzip"
)

// sampleGED references one local file, whose name needs percent-escaping in
// a FILE payload, and one web file, which has no entry in an archive.
const sampleGED = `0 HEAD
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

func main() {
	if len(os.Args) > 1 {
		inspect(filepath.Clean(os.Args[1]))
		return
	}

	// With no argument, package a sample document first. Everything lives in
	// a temporary directory that is removed afterwards.
	dir, err := os.MkdirTemp("", "gedzip-example")
	if err != nil {
		log.Fatalf("Failed to create a temporary directory: %v", err)
	}
	defer os.RemoveAll(dir)

	archive := filepath.Join(dir, "sample.gdz")
	if err := pack(archive, filepath.Join(dir, "files")); err != nil {
		log.Printf("Failed to write the archive: %v", err)
		return
	}
	fmt.Println("Wrote a sample archive: sample.gdz")
	fmt.Println()
	inspect(archive)
}

// pack writes sampleGED and the file it references to a GEDZIP archive.
func pack(archive, mediaDir string) error {
	// The media source is a directory laid out as the archive will be: each
	// file at the path its FILE payload names, percent-escapes decoded.
	if err := os.MkdirAll(filepath.Join(mediaDir, "media"), 0o750); err != nil {
		return err
	}
	portrait := filepath.Join(mediaDir, "media", "family portrait.txt")
	if err := os.WriteFile(portrait, []byte("A portrait of the family.\n"), 0o600); err != nil {
		return err
	}

	doc, err := decoder.Decode(strings.NewReader(sampleGED))
	if err != nil {
		return err
	}

	// #nosec G304 -- the path is under a directory this program just created.
	out, err := os.Create(archive)
	if err != nil {
		return err
	}
	// Write refuses a document that is not GEDCOM 7, and one referencing a
	// local file the media source lacks, before it writes anything. Any
	// fs.FS is a media source; os.DirFS serves a directory.
	if err := gedzip.Write(out, doc, os.DirFS(mediaDir), nil); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

// inspect reads a GEDZIP archive: its entries, its dataset, and whether the
// archive holds every file the dataset references.
func inspect(filename string) {
	// A nil *ReadOptions applies the default limits (entry count, sizes and
	// compression ratio), which suit an archive from an untrusted source.
	// #nosec G703 -- the path is this CLI's own argv, not attacker-controlled input.
	r, err := gedzip.OpenReader(filename, nil)
	if err != nil {
		log.Fatalf("Failed to open archive: %v", err)
	}
	defer r.Close()

	fmt.Printf("GEDZIP archive: %s\n", filepath.Base(filename))
	fmt.Println("Entries:")
	for _, entry := range r.Entries() {
		if entry.Unusable != nil {
			// An unusable entry (an unsafe name, encryption, a duplicate)
			// cannot be opened, but does not stop the rest being read.
			fmt.Printf("  %q: unusable: %v\n", entry.Name, entry.Unusable)
			continue
		}
		fmt.Printf("  %s (%d bytes)\n", entry.Name, entry.Size)
	}

	// The dataset goes through the ordinary decoder; nil means its defaults.
	doc, err := r.Decode(nil)
	if err != nil {
		fmt.Printf("Failed to decode gedcom.ged: %v\n", err)
		return
	}
	fmt.Printf("\nVersion: %s\n", doc.Header.Version)
	fmt.Printf("Records: %d (%d individuals, %d media objects)\n",
		len(doc.Records), len(doc.Individuals()), len(doc.MediaObjects()))

	fmt.Println("\nFile references:")
	for _, media := range doc.MediaObjects() {
		for _, file := range media.Files {
			describe(&r.Reader, media.XRef, file.FileRef)
			for _, tran := range file.Translations {
				describe(&r.Reader, media.XRef, tran.FileRef)
			}
		}
	}

	// CheckRefs lists the references the archive cannot satisfy, without
	// reading any entry.
	problems := r.CheckRefs(doc)
	if len(problems) == 0 {
		fmt.Println("\nEvery local reference has an entry.")
		return
	}
	fmt.Printf("\n%d reference(s) cannot be resolved:\n", len(problems))
	for _, p := range problems {
		fmt.Printf("  %q: %v\n", p.XRef, p.Err)
	}
}

// describe prints what one FILE or FILE.TRAN payload refers to, reading the
// entry when it is a local file.
func describe(r *gedzip.Reader, xref, ref string) {
	kind, name := gedzip.ClassifyFileRef(ref)
	if kind != gedzip.RefLocal {
		fmt.Printf("  %q %q: %s, no entry\n", xref, ref, kind)
		return
	}
	// Media is opened by payload and inflated only now, under the limits.
	rc, err := r.Open(ref)
	if err != nil {
		fmt.Printf("  %q %q: %v\n", xref, ref, err)
		return
	}
	defer rc.Close()
	n, err := io.Copy(io.Discard, rc)
	if err != nil {
		fmt.Printf("  %q %q: %v\n", xref, ref, err)
		return
	}
	fmt.Printf("  %q %q: local, entry %q, read %d bytes\n", xref, ref, name, n)
}
