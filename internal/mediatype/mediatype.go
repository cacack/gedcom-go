// Package mediatype maps a legacy GEDCOM 5.5/5.5.1 media format, or a file
// extension, to the IANA media type GEDCOM 7.0 writes as FORM.
//
// The converter maps a 5.x FORM to 7.0 with it, and gedcom.Document.AttachMedia
// picks a new OBJE record's 7.0 FORM from a file's extension with it. Both
// must agree on what JPG or TIF means, so the table lives here once.
package mediatype

import "strings"

// legacyToIANA maps legacy GEDCOM 5.5/5.5.1 media formats, which are also
// the file extensions they name, to IANA media types. Keys are uppercase.
var legacyToIANA = map[string]string{
	"JPG":  "image/jpeg",
	"JPEG": "image/jpeg",
	"PNG":  "image/png",
	"GIF":  "image/gif",
	"TIFF": "image/tiff",
	"TIF":  "image/tiff",
	"BMP":  "image/bmp",
	"MP3":  "audio/mpeg",
	"WAV":  "audio/wav",
	"MP4":  "video/mp4",
	"MPEG": "video/mpeg",
	"MPG":  "video/mpeg",
	"AVI":  "video/x-msvideo",
	"PDF":  "application/pdf",
	"TXT":  "text/plain",
	"TEXT": "text/plain",
}

// IANA returns the IANA media type for legacy, a GEDCOM 5.5/5.5.1 format or a
// file extension without its dot, compared ignoring case and surrounding
// space. ok is false when the table has no entry for it.
func IANA(legacy string) (iana string, ok bool) {
	iana, ok = legacyToIANA[strings.ToUpper(strings.TrimSpace(legacy))]
	return iana, ok
}
