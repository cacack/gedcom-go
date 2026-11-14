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
// # Untrusted input
//
// Reads are bounded: entry count, per-entry size, total size and compression
// ratio are capped, and an entry whose name is not a clean relative path can
// never be reached. The package never touches the filesystem with a path
// derived from an archive or from a payload.
package gedzip
