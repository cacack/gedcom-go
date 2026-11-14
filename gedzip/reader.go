package gedzip

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"math/bits"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// datasetName is the entry holding the GEDCOM data stream. The name is fixed
// by the specification and matched byte-exact.
const datasetName = "gedcom.ged"

// flagEncrypted is bit 0 of a zip entry's general purpose flags, set when the
// entry's data is encrypted. archive/zip does not look at it.
const flagEncrypted = 0x1

// Default limits of a [Reader], used where a [ReadOptions] field is zero
// (#127). Nothing in the specification dictates them; they are sized so that
// a legitimate archive passes and a hostile one is refused before it is
// inflated.
const (
	// DefaultMaxEntries is the default [ReadOptions.MaxEntries].
	DefaultMaxEntries = 100_000

	// DefaultMaxDocumentBytes is the default [ReadOptions.MaxDocumentBytes],
	// 256 MiB. It is the tight one because decoding a dataset takes many
	// times its size in memory.
	DefaultMaxDocumentBytes int64 = 256 << 20

	// DefaultMaxEntryBytes is the default [ReadOptions.MaxEntryBytes], 1 GiB.
	DefaultMaxEntryBytes int64 = 1 << 30

	// DefaultMaxTotalBytes is the default [ReadOptions.MaxTotalBytes], 4 GiB.
	DefaultMaxTotalBytes int64 = 4 << 30

	// DefaultMaxCompressionRatio is the default
	// [ReadOptions.MaxCompressionRatio].
	DefaultMaxCompressionRatio = 200

	// CompressionRatioMinBytes is the declared uncompressed size an entry has
	// to exceed before [ReadOptions.MaxCompressionRatio] applies to it, 1 MiB.
	// A small entry of repetitive text compresses very well and costs little.
	CompressionRatioMinBytes int64 = 1 << 20
)

// ReadOptions bounds what a [Reader] accepts from an archive (#127). In every
// field zero means the default and a negative value means unlimited, the
// convention of decoder.DecodeOptions.MaxDiagnostics. A nil *ReadOptions means
// all defaults.
//
// The limits are checked against the sizes the archive's central directory
// declares when the archive is opened, and an archive over any of them is
// refused whole with [ErrLimitExceeded]. The per-entry size limits are
// enforced again on the bytes actually inflated, so an entry that understates
// its size is cut off with an error, never returned short.
type ReadOptions struct {
	// MaxEntries caps the number of entries in the central directory,
	// directories included. Default [DefaultMaxEntries].
	MaxEntries int

	// MaxDocumentBytes caps the uncompressed size of gedcom.ged. Default
	// [DefaultMaxDocumentBytes].
	MaxDocumentBytes int64

	// MaxEntryBytes caps the uncompressed size of each entry other than
	// gedcom.ged. Default [DefaultMaxEntryBytes].
	MaxEntryBytes int64

	// MaxTotalBytes caps the sum of the declared uncompressed sizes of all
	// entries. Default [DefaultMaxTotalBytes].
	MaxTotalBytes int64

	// MaxCompressionRatio caps uncompressed size divided by compressed size
	// for each entry declaring more than [CompressionRatioMinBytes]. Default
	// [DefaultMaxCompressionRatio].
	MaxCompressionRatio int
}

// limits is a ReadOptions with its defaults applied: no field is zero, and a
// negative one is unlimited.
type limits struct {
	entries  int
	document int64
	entry    int64
	total    int64
	ratio    int
}

// resolve applies the defaults. A nil receiver is all defaults.
func (o *ReadOptions) resolve() limits {
	if o == nil {
		o = &ReadOptions{}
	}
	return limits{
		entries:  orDefault(o.MaxEntries, DefaultMaxEntries),
		document: orDefault(o.MaxDocumentBytes, DefaultMaxDocumentBytes),
		entry:    orDefault(o.MaxEntryBytes, DefaultMaxEntryBytes),
		total:    orDefault(o.MaxTotalBytes, DefaultMaxTotalBytes),
		ratio:    orDefault(o.MaxCompressionRatio, DefaultMaxCompressionRatio),
	}
}

// orDefault returns def for a zero v, and v otherwise.
func orDefault[T int | int64](v, def T) T {
	if v == 0 {
		return def
	}
	return v
}

// over reports whether n exceeds limit. A negative limit is unlimited.
func over(n uint64, limit int64) bool {
	return limit >= 0 && n > uint64(limit)
}

// Entry describes one file entry of an archive, as listed by [Reader.Entries].
type Entry struct {
	// Name is the entry's name exactly as stored. For an unusable entry it
	// may be anything, including invalid UTF-8 or an unsafe path: never use
	// it as a filesystem path.
	Name string

	// Size is the uncompressed size the archive declares, in bytes.
	Size int64

	// Unusable is nil for an entry that can be read. Otherwise it says why the
	// entry cannot be, and wraps [ErrUnusableEntry].
	Unusable error
}

// entry is an Entry with the zip file behind it.
type entry struct {
	Entry
	file *zip.File
}

// Reader reads a GEDZIP archive (#127): it decodes the dataset in gedcom.ged
// and opens the media entries that FILE and FILE.TRAN payloads name. Media is
// read lazily, one entry per call; nothing is extracted to the filesystem.
//
// Create one with [NewReader] or [OpenReader]. A Reader is safe for
// concurrent use if the io.ReaderAt it reads from is.
type Reader struct {
	entries []entry
	byName  map[string]int // entry name -> index of its first entry
	dataset int            // index of gedcom.ged
	limits  limits
}

// ReadCloser is a [Reader] over a file opened by [OpenReader]. Close it when
// done.
type ReadCloser struct {
	Reader
	f *os.File
}

// Close closes the archive file. Readers obtained from the archive must not be
// used afterwards.
func (rc *ReadCloser) Close() error {
	return rc.f.Close()
}

// OpenReader opens the GEDZIP archive at name, a file path the caller chooses,
// and behaves as [NewReader] does. The file stays open until the returned
// ReadCloser is closed.
func OpenReader(name string, opts *ReadOptions) (*ReadCloser, error) {
	f, err := os.Open(name) // #nosec G304 -- the caller's path; nothing from an archive or payload reaches here
	if err != nil {
		return nil, err
	}
	fi, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, err
	}
	rc := &ReadCloser{f: f}
	if err := rc.init(f, fi.Size(), opts); err != nil {
		_ = f.Close()
		return nil, err
	}
	return rc, nil
}

// NewReader reads the central directory of the GEDZIP archive in r, which is
// size bytes long, and checks it against opts (#127). It inflates nothing.
//
// It fails if r is not a zip archive, if the archive is over a limit
// ([ErrLimitExceeded]), if it has no entry named exactly "gedcom.ged"
// ([ErrNoDataset]; the name is case-sensitive and there is no fallback, but
// the message names any other .ged entries at the root), or if that entry is
// unusable or duplicated ([ErrUnusableEntry], with [ErrEncrypted] when
// encryption is the reason).
//
// Every other entry is judged on its own. An entry is usable only if it is
// not encrypted, is a regular file (not a symbolic link or other special
// file), is stored or deflated, has a name that is valid UTF-8 and a clean
// relative slash-separated path (no leading "/", no backslash, no empty, "."
// or ".." segment, no control character), and is the only entry of that name.
// An unusable entry is listed by [Reader.Entries] with the reason and can
// never be opened; it does not prevent reading gedcom.ged or other media.
// Directory entries (names ending in "/") are ignored. Entries the dataset
// does not reference are allowed.
//
// The zip UTF-8 name flag is not consulted: GEDZIP names are UTF-8 by
// definition, so a name is taken as UTF-8 whenever it is valid UTF-8.
func NewReader(r io.ReaderAt, size int64, opts *ReadOptions) (*Reader, error) {
	rd := &Reader{}
	if err := rd.init(r, size, opts); err != nil {
		return nil, err
	}
	return rd, nil
}

// init fills r from the archive. It is shared by NewReader and OpenReader.
func (r *Reader) init(ra io.ReaderAt, size int64, opts *ReadOptions) error {
	if ra == nil {
		return errors.New("gedzip: nil io.ReaderAt")
	}
	// zip.NewReader reports zip.ErrInsecurePath, alongside a usable reader,
	// only when GODEBUG=zipinsecurepath=0 is set. The names it objects to are
	// ones the usability rules below already make unreachable.
	zr, err := zip.NewReader(ra, size)
	if err != nil && !errors.Is(err, zip.ErrInsecurePath) {
		return fmt.Errorf("gedzip: %w", err)
	}
	r.limits = opts.resolve()
	if r.limits.entries >= 0 && len(zr.File) > r.limits.entries {
		return fmt.Errorf("%w: archive has %d entries, over MaxEntries (%d)",
			ErrLimitExceeded, len(zr.File), r.limits.entries)
	}
	if err := r.limits.check(zr.File, size); err != nil {
		return err
	}
	r.index(zr.File)

	i, ok := r.byName[datasetName]
	if !ok {
		return noDatasetError(r.entries)
	}
	if err := r.entries[i].Unusable; err != nil {
		return err
	}
	r.dataset = i
	return nil
}

// check applies the size, total and ratio limits to the sizes the central
// directory declares. archiveSize is the length of the whole archive.
func (l limits) check(files []*zip.File, archiveSize int64) error {
	var total uint64
	for _, f := range files {
		size := f.UncompressedSize64
		limit, field := l.entry, "MaxEntryBytes"
		if f.Name == datasetName {
			limit, field = l.document, "MaxDocumentBytes"
		}
		if over(size, limit) {
			return fmt.Errorf("%w: entry %q declares %d bytes, over %s (%d)",
				ErrLimitExceeded, f.Name, size, field, limit)
		}
		if l.overRatio(size, f.CompressedSize64, archiveSize) {
			return fmt.Errorf("%w: entry %q declares %d bytes from %d compressed, over MaxCompressionRatio (%d)",
				ErrLimitExceeded, f.Name, size, f.CompressedSize64, l.ratio)
		}
		var carry uint64
		if total, carry = bits.Add64(total, size, 0); carry != 0 {
			total = math.MaxUint64
		}
		if over(total, l.total) {
			return fmt.Errorf("%w: entries declare more than MaxTotalBytes (%d) in total",
				ErrLimitExceeded, l.total)
		}
	}
	return nil
}

// overRatio reports whether an entry declaring size bytes from compressed
// bytes is over the compression ratio limit. An entry cannot hold more
// compressed bytes than the archive is long, so a larger claim, which would
// understate the ratio, is cut down to that.
func (l limits) overRatio(size, compressed uint64, archiveSize int64) bool {
	if l.ratio < 0 || size <= uint64(CompressionRatioMinBytes) {
		return false
	}
	compressed = min(compressed, uint64(archiveSize)) // #nosec G115 -- zip.NewReader rejected a negative size
	hi, lo := bits.Mul64(compressed, uint64(l.ratio)) // #nosec G115 -- ratio is positive here
	return hi == 0 && size > lo
}

// index lists the file entries, decides which are usable and maps names to
// them.
func (r *Reader) index(files []*zip.File) {
	r.entries = make([]entry, 0, len(files))
	r.byName = make(map[string]int, len(files))
	count := make(map[string]int, len(files))
	for _, f := range files {
		if strings.HasSuffix(f.Name, "/") {
			continue
		}
		if _, ok := r.byName[f.Name]; !ok {
			r.byName[f.Name] = len(r.entries)
		}
		count[f.Name]++
		r.entries = append(r.entries, entry{
			Entry: Entry{
				Name:     f.Name,
				Size:     int64(min(f.UncompressedSize64, math.MaxInt64)), // #nosec G115 -- clamped
				Unusable: unusable(f),
			},
			file: f,
		})
	}
	for i := range r.entries {
		e := &r.entries[i]
		if e.Unusable == nil && count[e.Name] > 1 {
			e.Unusable = fmt.Errorf("%w: %d entries are named %q", ErrUnusableEntry, count[e.Name], e.Name)
		}
	}
}

// unusable returns why f cannot be read safely, or nil if it can. Uniqueness
// of the name is checked by the caller. archive/zip checks none of these: it
// ignores the encryption flag, opens any mode, and accepts any name.
func unusable(f *zip.File) error {
	switch {
	case f.Flags&flagEncrypted != 0:
		return fmt.Errorf("%w: %q: %w", ErrUnusableEntry, f.Name, ErrEncrypted)
	case f.Mode()&fs.ModeSymlink != 0:
		return fmt.Errorf("%w: %q is a symbolic link", ErrUnusableEntry, f.Name)
	case !f.Mode().IsRegular():
		return fmt.Errorf("%w: %q is not a regular file", ErrUnusableEntry, f.Name)
	case f.Method != zip.Store && f.Method != zip.Deflate:
		return fmt.Errorf("%w: %q uses compression method %d, not Store or Deflate",
			ErrUnusableEntry, f.Name, f.Method)
	case !isCleanRelative(f.Name):
		return fmt.Errorf("%w: name %q is not a clean relative UTF-8 path", ErrUnusableEntry, f.Name)
	}
	return nil
}

// maxNamedDatasets caps the entries a missing-dataset error names.
const maxNamedDatasets = 5

// noDatasetError is the error for an archive with no gedcom.ged. It names the
// root-level entries ending in .ged, in any case, since a dataset stored under
// another name is the likely mistake; they are reported, never read.
func noDatasetError(entries []entry) error {
	var found []string
	for i := range entries {
		name := entries[i].Name
		if !strings.Contains(name, "/") && strings.EqualFold(path.Ext(name), ".ged") {
			found = append(found, fmt.Sprintf("%q", name))
		}
	}
	if len(found) == 0 {
		return ErrNoDataset
	}
	more := ""
	if len(found) > maxNamedDatasets {
		more = fmt.Sprintf(" and %d more", len(found)-maxNamedDatasets)
		found = found[:maxNamedDatasets]
	}
	return fmt.Errorf("%w (found %s%s; the dataset must be named exactly %q)",
		ErrNoDataset, strings.Join(found, ", "), more, datasetName)
}

// Entries lists the archive's file entries in archive order, gedcom.ged
// included and directory entries left out. The slice is the caller's.
func (r *Reader) Entries() []Entry {
	out := make([]Entry, len(r.entries))
	for i := range r.entries {
		out[i] = r.entries[i].Entry
	}
	return out
}

// Decode decodes the archive's gedcom.ged. The entry is streamed straight
// into decoder.DecodeWithOptions, so line length, character set and BOM
// handling are exactly those of decoding the file on its own, and the result
// follows that function's contract. A nil opts means the decoder's defaults.
//
// The error wraps [ErrLimitExceeded] if the entry inflates past
// [ReadOptions.MaxDocumentBytes], and archive/zip's error if its size or
// checksum does not match what the archive declares.
func (r *Reader) Decode(opts *decoder.DecodeOptions) (*gedcom.Document, error) {
	rc, err := r.openDataset()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rc.Close() }()
	return decoder.DecodeWithOptions(rc, opts)
}

// DecodeWithDiagnostics is [Reader.Decode] through
// decoder.DecodeWithDiagnostics: it returns the diagnostics as well, under
// that function's contract.
func (r *Reader) DecodeWithDiagnostics(opts *decoder.DecodeOptions) (*decoder.DecodeResult, error) {
	rc, err := r.openDataset()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rc.Close() }()
	return decoder.DecodeWithDiagnostics(rc, opts)
}

// openDataset opens gedcom.ged. A Reader not made by NewReader or OpenReader
// has none.
func (r *Reader) openDataset() (io.ReadCloser, error) {
	if r.dataset >= len(r.entries) {
		return nil, ErrNoDataset
	}
	return r.open(r.dataset)
}

// Open opens the entry that fileRef, the payload of a FILE or FILE.TRAN line,
// names. The payload is classified by [ClassifyFileRef] and its entry name,
// the payload percent-decoded, is matched byte-exact: entry names themselves
// are never percent-decoded, case folded or normalised.
//
// The error wraps [ErrNotLocal] if the payload is a web file, a file: URL or
// invalid, [ErrEntryNotFound] if the archive has no such entry, and
// [ErrUnusableEntry] if it has one that cannot be read safely. Reading returns
// an error wrapping [ErrLimitExceeded] if the entry inflates past
// [ReadOptions.MaxEntryBytes], and archive/zip's error if its size or
// checksum does not match what the archive declares.
//
// "gedcom.ged" opens the dataset's raw bytes, under
// [ReadOptions.MaxDocumentBytes]. That is access to the dataset, not to a
// media file: a FILE payload that names it is one [Reader.CheckRefs] reports,
// so check a document's references before serving them through Open.
func (r *Reader) Open(fileRef string) (io.ReadCloser, error) {
	kind, name := ClassifyFileRef(fileRef)
	if kind != RefLocal {
		return nil, notLocalError(fileRef, kind)
	}
	i, ok := r.byName[name]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrEntryNotFound, name)
	}
	return r.open(i)
}

// open opens entry i under the size limit that applies to it.
func (r *Reader) open(i int) (io.ReadCloser, error) {
	e := &r.entries[i]
	if e.Unusable != nil {
		return nil, e.Unusable
	}
	rc, err := e.file.Open()
	if err != nil {
		return nil, fmt.Errorf("gedzip: open %q: %w", e.Name, err)
	}
	limit, field := r.limits.entry, "MaxEntryBytes"
	if i == r.dataset {
		limit, field = r.limits.document, "MaxDocumentBytes"
	}
	if limit < 0 {
		return rc, nil
	}
	return &limitedReadCloser{rc: rc, remaining: limit, name: e.Name, field: field}, nil
}

// limitedReadCloser returns ErrLimitExceeded once more than the limit has been
// read from rc. It reads one byte past the limit to tell an entry of exactly
// the limit from a longer one, so the caller gets an error and never a
// silently short entry.
//
// archive/zip already fails a read that passes the entry's declared size,
// which NewReader has checked against the same limit; this holds the limit
// without relying on that.
type limitedReadCloser struct {
	rc        io.ReadCloser
	remaining int64 // bytes still allowed
	name      string
	field     string
	err       error
}

// Read implements io.Reader.
func (l *limitedReadCloser) Read(p []byte) (int, error) {
	if l.err != nil {
		return 0, l.err
	}
	// At a limit of math.MaxInt64 the sum overflows and p is left alone,
	// which is right: no read can pass that limit.
	if most := l.remaining + 1; most > 0 && int64(len(p)) > most {
		p = p[:most]
	}
	n, err := l.rc.Read(p)
	if int64(n) > l.remaining {
		n = int(l.remaining) // #nosec G115 -- remaining < n, an int
		l.remaining = 0
		l.err = fmt.Errorf("%w: entry %q inflates past %s", ErrLimitExceeded, l.name, l.field)
		return n, l.err
	}
	l.remaining -= int64(n)
	return n, err
}

// Close implements io.Closer.
func (l *limitedReadCloser) Close() error {
	return l.rc.Close()
}

// FS returns the usable media entries as a read-only file system, so that the
// archive's media can be walked, or handed to anything that takes an fs.FS.
// gedcom.ged and unusable entries are not in it. Files are named by entry
// name, not by FILE payload, and are read under the same limits as
// [Reader.Open]; directories are synthesised from the entry names.
//
// Should one entry be named as another's directory ("a" beside "a/b"), both
// open by name and "a" is listed as a file.
func (r *Reader) FS() fs.FS {
	fsys := &mediaFS{r: r, dirs: map[string][]fs.DirEntry{".": nil}}
	for i := range r.entries {
		if !r.isMedia(i) {
			continue
		}
		e := &r.entries[i]
		name := e.Name
		var child fs.DirEntry = fileInfo{name: path.Base(name), size: e.Size, modTime: e.file.Modified}
		for {
			dir := path.Dir(name)
			_, known := fsys.dirs[dir]
			fsys.dirs[dir] = append(fsys.dirs[dir], child)
			// Stop at a directory already linked to its parent, and at a
			// name that is also a media file ("a" beside "a/b"): that name
			// is listed as the file, not as a directory
			// (TestReaderFSFileAndDirectoryOfOneName).
			if known || r.isMediaName(dir) {
				break
			}
			name, child = dir, fileInfo{name: path.Base(dir), dir: true}
		}
	}
	for _, list := range fsys.dirs {
		sort.Slice(list, func(a, b int) bool { return list[a].Name() < list[b].Name() })
	}
	return fsys
}

// isMedia reports whether entry i is a usable entry other than the dataset.
func (r *Reader) isMedia(i int) bool {
	return i != r.dataset && r.entries[i].Unusable == nil
}

// isMediaName reports whether name is that of a usable media entry.
func (r *Reader) isMediaName(name string) bool {
	i, ok := r.byName[name]
	return ok && r.isMedia(i)
}

// mediaFS is the fs.FS of a Reader's usable media entries.
type mediaFS struct {
	r    *Reader
	dirs map[string][]fs.DirEntry // directory -> its entries, sorted by name
}

// Open implements fs.FS.
func (m *mediaFS) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}
	if m.r.isMediaName(name) {
		i := m.r.byName[name]
		rc, err := m.r.open(i)
		if err != nil {
			return nil, &fs.PathError{Op: "open", Path: name, Err: err}
		}
		e := &m.r.entries[i]
		info := fileInfo{name: path.Base(name), size: e.Size, modTime: e.file.Modified}
		return &mediaFile{ReadCloser: rc, info: info}, nil
	}
	if list, ok := m.dirs[name]; ok {
		return &mediaDir{info: fileInfo{name: path.Base(name), dir: true}, entries: list}, nil
	}
	return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
}

// mediaFile is an open media entry.
type mediaFile struct {
	io.ReadCloser
	info fileInfo
}

// Stat implements fs.File.
func (f *mediaFile) Stat() (fs.FileInfo, error) { return f.info, nil }

// mediaDir is an open synthesised directory.
type mediaDir struct {
	info    fileInfo
	entries []fs.DirEntry
	offset  int
}

// Stat implements fs.File.
func (d *mediaDir) Stat() (fs.FileInfo, error) { return d.info, nil }

// Read implements fs.File; a directory has no bytes.
func (d *mediaDir) Read([]byte) (int, error) {
	return 0, &fs.PathError{Op: "read", Path: d.info.name, Err: errors.New("is a directory")}
}

// Close implements fs.File.
func (d *mediaDir) Close() error { return nil }

// ReadDir implements fs.ReadDirFile.
func (d *mediaDir) ReadDir(n int) ([]fs.DirEntry, error) {
	rest := d.entries[d.offset:]
	if n > 0 {
		if len(rest) == 0 {
			return nil, io.EOF
		}
		rest = rest[:min(n, len(rest))]
	}
	d.offset += len(rest)
	return append([]fs.DirEntry(nil), rest...), nil
}

// fileInfo is the fs.FileInfo and fs.DirEntry of a media file or a
// synthesised directory.
type fileInfo struct {
	name    string
	size    int64
	modTime time.Time
	dir     bool
}

// Name implements fs.FileInfo and fs.DirEntry.
func (i fileInfo) Name() string { return i.name }

// Size implements fs.FileInfo.
func (i fileInfo) Size() int64 { return i.size }

// Mode implements fs.FileInfo: read-only, whatever the archive says.
func (i fileInfo) Mode() fs.FileMode {
	if i.dir {
		return fs.ModeDir | 0o555
	}
	return 0o444
}

// ModTime implements fs.FileInfo.
func (i fileInfo) ModTime() time.Time { return i.modTime }

// IsDir implements fs.FileInfo and fs.DirEntry.
func (i fileInfo) IsDir() bool { return i.dir }

// Sys implements fs.FileInfo.
func (i fileInfo) Sys() any { return nil }

// Type implements fs.DirEntry.
func (i fileInfo) Type() fs.FileMode { return i.Mode().Type() }

// Info implements fs.DirEntry.
func (i fileInfo) Info() (fs.FileInfo, error) { return i, nil }
