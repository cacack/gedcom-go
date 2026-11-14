// Package gedzip reads and writes GEDZIP archives, the GEDCOM 7 packaging
// format (#127).
//
// A GEDZIP file (conventionally .gdz) is a zip archive holding the dataset in
// an entry named exactly "gedcom.ged", plus the local files that the dataset's
// FILE and FILE.TRAN payloads name. Entry names are case-sensitive UTF-8.
//
// # Archive model
//
// The archive is kept separate from the gedcom.Document: decoding the dataset
// yields an ordinary Document, and media stays in the archive until it is
// asked for. Media is therefore read lazily, one entry at a time, and never
// loaded as a whole.
//
// # Resolution
//
// References are resolved strictly as the GEDCOM 7 "File Path" datatype
// defines them. [ClassifyFileRef] decides whether a payload names a local
// archive entry, a web file, a file: URL or nothing valid; a local payload
// maps to its entry name by percent-decoding, and that name is matched
// byte-exact. There is no case folding, no Unicode normalisation, no backslash
// repair and no fallback to a differently named dataset.
//
// # Reading
//
// [NewReader] (or [OpenReader] for a file) reads the archive's directory and
// inflates nothing. [Reader.Decode] then decodes the dataset through the
// ordinary decoder, and [Reader.Open] opens a media file by the FILE or
// FILE.TRAN payload that names it, one entry at a time.
//
// [Reader.Entries] lists what the archive holds, [Reader.FS] exposes the media
// as an fs.FS, and [Reader.CheckRefs] reports the references the archive does
// not satisfy.
//
// # Untrusted input
//
// Reads are bounded by [ReadOptions], whose defaults are safe for an upload:
// entry count, dataset size, per-entry size, total size and compression ratio
// are capped from the archive's directory before anything is inflated, and the
// size caps are enforced again on the bytes inflated. An entry that is
// encrypted, a symbolic link or other special file, compressed with anything
// but Store or Deflate, duplicated, or named with something other than a clean
// relative path can never be reached, and does not stop the rest of the
// archive from being read. The package never writes to the filesystem and
// never touches it with a path derived from an archive or from a payload.
//
// Decoding a dataset takes many times its size in memory, so
// [ReadOptions.MaxDocumentBytes] is the limit to lower when memory is tight.
package gedzip
