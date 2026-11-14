package gedzip

import (
	"archive/zip"
	"bytes"
	"io"
	"io/fs"
	"testing"
)

// FuzzGEDZipReader holds the reader to its bounds on arbitrary bytes (#127):
// whatever the archive, opening it, decoding the dataset, checking references
// and reading every entry neither panics nor returns more than the limits
// allow.
func FuzzGEDZipReader(f *testing.F) {
	const maxBytes = 1 << 16
	opts := &ReadOptions{
		MaxEntries:          64,
		MaxDocumentBytes:    maxBytes,
		MaxEntryBytes:       maxBytes,
		MaxTotalBytes:       4 * maxBytes,
		MaxCompressionRatio: 50,
	}

	f.Add(buildZip(f, file("gedcom.ged", minimalGED)))
	f.Add(buildZip(f,
		file("gedcom.ged", refsGED),
		file("media/", ""),
		file("media/a b.jpg", "space"),
		zipEntry{name: "media/thumb.jpg", data: "stored", raw: true},
	))
	f.Add(buildZip(f,
		file("gedcom.ged", minimalGED),
		file("../evil", "x"),
		file("/abs", "x"),
		file(`back\slash`, "x"),
		file("bad\xff.jpg", "x"),
		file("dup", "1"),
		file("dup", "2"),
		zipEntry{name: "link", data: "gedcom.ged", mod: func(h *zip.FileHeader) { h.SetMode(fs.ModeSymlink | 0o777) }},
		zipEntry{name: "secret", data: "x", mod: func(h *zip.FileHeader) { h.Flags |= flagEncrypted }},
		zipEntry{name: "bzip2", data: "x", raw: true, mod: func(h *zip.FileHeader) { h.Method = 12 }},
	))
	f.Add(buildZip(f,
		deflated(f, "gedcom.ged", minimalGED, declare(3)),
		deflated(f, "short.jpg", "0123456789", declare(4)),
		zipEntry{name: "long.jpg", data: "0123456789", raw: true, mod: declare(64)},
	))
	f.Add(buildZip(f, file("gedcom.ged", minimalGED), file("zeros.bin", string(make([]byte, 2<<20)))))
	f.Add(buildZip(f, file("Family.GED", minimalGED)))
	f.Add([]byte(minimalGED))
	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, data []byte) {
		r, err := NewReader(bytes.NewReader(data), int64(len(data)), opts)
		if err != nil {
			if r != nil {
				t.Fatal("NewReader() returned a Reader with an error")
			}
			return
		}

		doc, _ := r.Decode(nil)
		if res, _ := r.DecodeWithDiagnostics(nil); res != nil && doc == nil {
			doc = res.Document
		}
		r.CheckRefs(doc)

		entries := r.Entries()
		if len(entries) > opts.MaxEntries {
			t.Fatalf("%d entries, over MaxEntries", len(entries))
		}
		for i, e := range entries {
			rc, err := r.open(i)
			if err == nil && e.Unusable != nil {
				t.Fatalf("unusable entry %q opened", e.Name)
			}
			if err != nil {
				continue
			}
			n, _ := io.Copy(io.Discard, rc)
			_ = rc.Close()
			if n > maxBytes {
				t.Fatalf("entry %q gave %d bytes, over the limit", e.Name, n)
			}
			if n > e.Size {
				t.Fatalf("entry %q gave %d bytes, more than the %d declared", e.Name, n, e.Size)
			}
		}

		// The file system holds usable media only, and walking and reading it
		// stays within the same bounds.
		_ = fs.WalkDir(r.FS(), ".", func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if p == datasetName {
				t.Fatal("FS holds the dataset")
			}
			if kind, name := ClassifyFileRef(p); kind == RefLocal && name == p {
				// Reachable by reference as well as by name.
				if rc, err := r.Open(p); err == nil {
					_ = rc.Close()
				}
			}
			file, err := r.FS().Open(p)
			if err != nil {
				return nil
			}
			n, _ := io.Copy(io.Discard, file)
			_ = file.Close()
			if n > maxBytes {
				t.Fatalf("FS file %q gave %d bytes, over the limit", p, n)
			}
			return nil
		})
	})
}
