package gedzip

import "errors"

// Sentinel errors of the package (#127). Each is returned wrapped, with the
// entry or reference it concerns in the message; test for one with errors.Is.
var (
	// ErrNoDataset means the archive has no entry named exactly "gedcom.ged".
	ErrNoDataset = errors.New("gedzip: archive has no gedcom.ged entry")

	// ErrEncrypted means an entry is encrypted. Encrypted entries are neither
	// read nor written. For a media entry it accompanies [ErrUnusableEntry].
	ErrEncrypted = errors.New("gedzip: entry is encrypted")

	// ErrLimitExceeded means the archive, or what an entry inflated to, is
	// over one of the [ReadOptions] limits.
	ErrLimitExceeded = errors.New("gedzip: limit exceeded")

	// ErrEntryNotFound means a local reference names no entry of the archive.
	ErrEntryNotFound = errors.New("gedzip: no such entry")

	// ErrNotLocal means a reference is not a local file: it is a web file, a
	// file: URL or invalid, as [ClassifyFileRef] reports, and has no entry.
	ErrNotLocal = errors.New("gedzip: reference is not a local file")

	// ErrUnusableEntry means an entry exists but cannot be read safely: it is
	// encrypted, not a regular file, compressed with a method other than Store
	// or Deflate, named with something other than a clean relative UTF-8 path,
	// or shares its name with another entry.
	ErrUnusableEntry = errors.New("gedzip: entry is unusable")
)
