package gedzip

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/encoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
	"github.com/cacack/gedcom-go/v3/internal/gedcvers"
)

// WriteOptions configures [Write] (#127). The zero value, like a nil
// *WriteOptions, writes the dataset with the encoder's defaults.
type WriteOptions struct {
	// Encode is how the dataset is encoded into gedcom.ged; nil means the
	// encoder's defaults. Its TargetVersion, when set, decides the version
	// the dataset is written as, and so whether Write accepts the document.
	Encode *encoder.EncodeOptions
}

// entryTime is the modification time every entry is written with, the
// earliest a zip entry can carry, so that an archive depends on nothing but
// the document and the media.
var entryTime = time.Date(1980, time.January, 1, 0, 0, 0, 0, time.UTC)

// deflatedExtensions are the extensions, lower case, of the media Write
// deflates: text, which compresses well. Everything else is taken to be
// compressed already (images, audio, video, PDF) and is stored.
var deflatedExtensions = map[string]bool{
	".csv":  true,
	".ged":  true,
	".htm":  true,
	".html": true,
	".json": true,
	".md":   true,
	".svg":  true,
	".txt":  true,
	".vtt":  true,
	".xml":  true,
}

// Write writes doc and the local files it references to w as a GEDZIP
// archive (#127).
//
// The archive holds the dataset first, as an entry named gedcom.ged, then one
// entry for each distinct local file a FILE or FILE.TRAN payload of the
// written dataset names, read from media and sorted by entry name. An entry's
// name is the payload percent-decoded, as [ClassifyFileRef] returns it, in
// UTF-8. A file referenced more than once is written once; a file in media
// that nothing references is not written. Web references (ftp, http, https)
// name no entry and pass through untouched. The dataset and media with a
// text extension (csv, ged, htm, html, json, md, svg, txt, vtt, xml) are
// deflated, other media is stored, and there are no directory entries.
// Entries carry a fixed timestamp and mode (0644), whatever the media source
// reports, so the same document and media give the same bytes.
//
// media may be any fs.FS: os.DirFS for a directory, a testing/fstest.MapFS,
// or the [Reader.FS] of another archive. Files are streamed from it one at a
// time. It may be nil when the dataset has no local reference. doc is not
// modified, and payloads are written as they are.
//
// The references, and the version, are taken from the dataset as encoded,
// read back, rather than from doc's typed fields, because the two can differ:
// the encoder writes a header or record from its Tags when it has them, and
// builds one from the typed fields only when it has none, moving inline media
// links into new OBJE records for GEDCOM 7. An inline OBJE structure the
// encoder leaves in place, as it does in a decoded 5.5.1 record written with a
// GEDCOM 7 TargetVersion, is a reference like any other: its FILE payloads
// are checked and their files archived. The archive therefore always matches
// the gedcom.ged it holds. The cost is one decode of the encoded dataset, so
// Write holds doc, the encoded bytes and a second document in memory at once.
//
// Write refuses, before writing anything to w:
//   - a nil doc ([encoder.ErrNilDocument]) or one the encoder rejects;
//   - a dataset that is not written as GEDCOM 7.x ([ErrNotGEDCOM7]): GEDZIP
//     is a GEDCOM 7 format, and Write converts nothing, so convert the
//     document with the converter package first. Both the version the
//     encoder writes by (TargetVersion, else Header.Version) and the one the
//     written header declares must be 7.x; setting Header.Version on a
//     decoded document changes the first and not the second;
//   - a file: URL, which must not be used in a GEDZIP, and a payload that is
//     not a valid File Path (both [ErrNotLocal]);
//   - a local reference to gedcom.ged, the dataset's own name, which the file
//     must be renamed to avoid ([ErrReservedName]);
//   - a local reference with no regular file in media ([ErrMissingMedia]).
//
// Every refused reference is reported, joined into one error; test it with
// errors.Is. References to MANIFEST.MF and under META-INF/ are discouraged by
// the specification but not forbidden, and are written like any other.
//
// An error after validation (w failing, or a media file that cannot be read
// in full) leaves w holding an incomplete archive, so write to a temporary
// file or a buffer and keep the result only when Write returns nil.
//
// Write copies whatever regular file a reference names, so when doc comes
// from an untrusted source its author chooses which files of media are
// published: give Write a media source that holds nothing else. os.DirFS
// follows symbolic links, including ones that lead outside its directory.
//
// Write applies no size limit. An archive with more or larger media than the
// [ReadOptions] defaults allow is written as asked, and [NewReader] then
// refuses it until those limits are raised.
func Write(w io.Writer, doc *gedcom.Document, media fs.FS, opts *WriteOptions) error {
	var enc *encoder.EncodeOptions
	if opts != nil {
		enc = opts.Encode
	}
	dataset, err := encodeDataset(doc, enc)
	if err != nil {
		return err
	}
	written, err := readBack(dataset)
	if err != nil {
		return err
	}
	names, err := mediaEntries(written, media)
	if err != nil {
		return err
	}

	zw := zip.NewWriter(w)
	if err := writeEntry(zw, datasetName, zip.Deflate, bytes.NewReader(dataset)); err != nil {
		return err
	}
	for _, name := range names {
		if err := writeMedia(zw, media, name); err != nil {
			return err
		}
	}
	if err := zw.Close(); err != nil {
		return fmt.Errorf("gedzip: %w", err)
	}
	return nil
}

// encodeDataset returns doc encoded as the bytes of gedcom.ged, or
// ErrNotGEDCOM7 when the encoder would not write it by the GEDCOM 7 rules.
// readBack then checks the version the bytes declare; both must be 7.x.
func encodeDataset(doc *gedcom.Document, enc *encoder.EncodeOptions) ([]byte, error) {
	if doc == nil {
		return nil, fmt.Errorf("gedzip: %w", encoder.ErrNilDocument)
	}
	if v := encoderVersion(doc, enc); strings.TrimSpace(string(v)) != string(v) || !isGEDCOM7(v) {
		return nil, notGEDCOM7Error("would be written by the rules of", v)
	}
	var buf bytes.Buffer
	var err error
	if enc == nil {
		err = encoder.Encode(&buf, doc)
	} else {
		err = encoder.EncodeWithOptions(&buf, doc, enc)
	}
	if err != nil {
		return nil, fmt.Errorf("gedzip: encode %s: %w", datasetName, err)
	}
	return buf.Bytes(), nil
}

// encoderVersion is the version whose rules the encoder writes doc by: the
// options' TargetVersion, or Header.Version when no target is set. A value
// with surrounding whitespace is no version to the encoder.
func encoderVersion(doc *gedcom.Document, enc *encoder.EncodeOptions) gedcom.Version {
	if enc != nil && enc.TargetVersion != "" {
		return enc.TargetVersion
	}
	if doc.Header != nil {
		return doc.Header.Version
	}
	return ""
}

// notGEDCOM7Error is the refusal of a dataset whose version v is not 7.x.
func notGEDCOM7Error(how string, v gedcom.Version) error {
	return fmt.Errorf("%w: the dataset %s version %q; convert it with the converter package first",
		ErrNotGEDCOM7, how, string(v))
}

// readBack decodes dataset, an encoded gedcom.ged, or returns ErrNotGEDCOM7
// when it does not declare GEDCOM 7.x. The version is read from the bytes as
// well as predicted (see encodeDataset) because the two can differ: without a
// TargetVersion the encoder writes a header's GEDC.VERS from its Tags when it
// has them, whatever Header.Version says.
func readBack(dataset []byte) (*gedcom.Document, error) {
	written, err := decoder.Decode(bytes.NewReader(dataset))
	if err != nil {
		return nil, fmt.Errorf("gedzip: read back the encoded %s: %w", datasetName, err)
	}
	var v gedcom.Version
	if written.Header != nil {
		v = written.Header.Version
	}
	if !isGEDCOM7(v) {
		return nil, notGEDCOM7Error("declares", v)
	}
	return written, nil
}

// isGEDCOM7 reports whether v is GEDCOM 7: any 7.x minor or patch version, by
// the rule the decoder reads GEDC.VERS with.
func isGEDCOM7(v gedcom.Version) bool {
	return gedcvers.Parse(string(v)) == gedcom.Version70
}

// mediaEntries returns the entry names, sorted and distinct, of the local
// files that written (the dataset, read back) references, having checked that
// each is a regular file in media. Every reference Write refuses is reported,
// once per distinct payload, in one joined error.
func mediaEntries(written *gedcom.Document, media fs.FS) ([]string, error) {
	var names []string
	var problems []error
	seen := make(map[string]bool)
	for _, ref := range collectFileRefs(written) {
		if seen[ref.ref] {
			continue
		}
		seen[ref.ref] = true
		name, err := localEntry(ref.ref, media)
		switch {
		case err != nil:
			problems = append(problems, fmt.Errorf("%s: %w", ref.xref, err))
		case name != "":
			names = append(names, name)
		}
	}
	if len(problems) > 0 {
		return nil, errors.Join(problems...)
	}

	// Two payloads can spell one name ("a%20b.jpg" and "a b.jpg").
	sort.Strings(names)
	return compact(names), nil
}

// localEntry returns the entry name for payload ref, or "" for a web file,
// which has no entry. It is an error for ref to be anything else that is not
// a local file, to name the dataset, or to name no regular file in media.
func localEntry(ref string, media fs.FS) (string, error) {
	kind, name := ClassifyFileRef(ref)
	switch kind {
	case RefWeb:
		return "", nil
	case RefLocal:
	default:
		return "", notLocalError(ref, kind)
	}
	if name == datasetName {
		return "", reservedNameError(ref)
	}
	if media == nil {
		return "", fmt.Errorf("%w: %q (no media source)", ErrMissingMedia, name)
	}
	info, err := fs.Stat(media, name)
	if err != nil {
		return "", fmt.Errorf("%w: %q: %w", ErrMissingMedia, name, err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("%w: %q is not a regular file", ErrMissingMedia, name)
	}
	return name, nil
}

// compact removes adjacent repeats from sorted names, in place.
func compact(names []string) []string {
	out := names[:0]
	for i, name := range names {
		if i == 0 || name != names[i-1] {
			out = append(out, name)
		}
	}
	return out
}

// writeMedia streams the file name of media into an entry of that name.
func writeMedia(zw *zip.Writer, media fs.FS, name string) (err error) {
	f, err := media.Open(name)
	if err != nil {
		return fmt.Errorf("gedzip: open media %q: %w", name, err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("gedzip: close media %q: %w", name, cerr)
		}
	}()
	method := zip.Store
	if deflatedExtensions[strings.ToLower(path.Ext(name))] {
		method = zip.Deflate
	}
	return writeEntry(zw, name, method, f)
}

// entryMode is the mode every entry is written with. Setting one marks the
// entry as made on Unix, which matters beyond permissions: Info-ZIP's unzip
// takes the name of an entry made on FAT (archive/zip's default) for a DOS
// code page and garbles a UTF-8 one, whatever the entry's UTF-8 flag says.
const entryMode fs.FileMode = 0o644

// writeEntry writes one entry, with the fixed timestamp and mode. archive/zip
// flags the name as UTF-8 when it is not ASCII, and switches to zip64 by
// itself.
func writeEntry(zw *zip.Writer, name string, method uint16, content io.Reader) error {
	hdr := &zip.FileHeader{Name: name, Method: method, Modified: entryTime}
	hdr.SetMode(entryMode)
	w, err := zw.CreateHeader(hdr)
	if err != nil {
		return fmt.Errorf("gedzip: write %q: %w", name, err)
	}
	if _, err := io.Copy(w, content); err != nil {
		return fmt.Errorf("gedzip: write %q: %w", name, err)
	}
	return nil
}
