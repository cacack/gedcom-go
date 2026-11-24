package gedcom

import (
	"fmt"
	"strings"

	"github.com/cacack/gedcom-go/v3/internal/mediatype"
)

// AttachStatus is what Document.AttachMedia did.
type AttachStatus int

const (
	// AttachCreated means an OBJE record was added for the file and a link
	// to it added to the record.
	AttachCreated AttachStatus = iota + 1

	// AttachLinked means an existing OBJE record names the file and a link
	// to it was added to the record.
	AttachLinked

	// AttachAlreadyLinked means the record already links an OBJE record that
	// names the file; nothing changed.
	AttachAlreadyLinked
)

// String returns the status as a lowercase phrase: "created", "linked" or
// "already linked".
func (s AttachStatus) String() string {
	switch s {
	case AttachCreated:
		return "created"
	case AttachLinked:
		return "linked"
	case AttachAlreadyLinked:
		return "already linked"
	default:
		return "unknown"
	}
}

// AttachMediaResult is the outcome of Document.AttachMedia.
type AttachMediaResult struct {
	// MediaXRef is the XRef of the OBJE record the record now links.
	MediaXRef string

	// Status says whether the OBJE record and the link were added.
	Status AttachStatus
}

// unknownMediaType is the GEDCOM 7.0 FORM AttachMedia gives a file whose
// extension it does not know: any binary data (RFC 2046).
const unknownMediaType = "application/octet-stream"

// AttachMedia links the media file at path from the record d indexes under
// xref, finding or creating the OBJE record that names the file:
//
//   - An OBJE record whose FILE is path is reused. Paths are compared as
//     written except that "\" reads as "/", so `photos\ada.jpg` matches
//     "photos/ada.jpg". When several OBJE records name the file, one the
//     record already links is preferred, then the first in document order.
//   - Otherwise an OBJE record is added under the XRef [Document.FreeXRef]
//     returns for prefix "O", with one FILE holding path as given and a FORM
//     for its extension (see below).
//   - A pointer to that record is appended to the record's media links
//     ([Record.AddMediaLink]) unless one is there already, so calling
//     AttachMedia twice links the file once. Inline links are not looked at.
//
// The result's Status says which happened, and MediaXRef names the OBJE
// record.
//
// The new record's FORM follows Header.DeclaredVersion:
//
//   - GEDCOM 5.5: the 5.5 format (jpg and jpeg give "jpeg", tif and tiff
//     "tiff"), written beside FILE in the 5.5 record layout (MediaObject.Form).
//   - GEDCOM 5.5.1: the 5.5.1 format (jpg and jpeg give "jpg", tif and tiff
//     "tif") under FILE.
//   - GEDCOM 7.0 or no recognized version: the IANA media type (jpg gives
//     "image/jpeg") under FILE, as the converter maps 5.x formats to 7.0.
//
// For an extension it does not know, the 5.x FORM is the extension in
// lowercase ("png", "pdf"), as most 5.x writers do, and the 7.0 FORM is
// "application/octet-stream". A path with no extension gets that media type
// for 7.0 and no FORM for 5.x. Edit the record afterwards (GetMediaObject)
// for another FORM.
//
// Only the typed entities change. Record.Tags is authoritative when encoding,
// so on a record that has raw Tags (any decoded record) call encoder.SyncTags
// on the document afterwards for the encoder to write the new link. The new
// OBJE record has no Tags and is written from its entity.
//
// It returns an error wrapping ErrUnknownXRef when d indexes no record under
// xref, one wrapping ErrNoMediaLinks when that record cannot carry media links
// (see [Record.MediaLinks]), ErrEmptyMediaPath for an empty path and
// ErrNilDocument for a nil d. d is unchanged in each case.
func (d *Document) AttachMedia(xref, path string) (AttachMediaResult, error) {
	if d == nil {
		return AttachMediaResult{}, ErrNilDocument
	}
	if path == "" {
		return AttachMediaResult{}, ErrEmptyMediaPath
	}
	if d.XRefMap == nil {
		d.ReindexXRefs()
	}
	record := d.GetRecord(xref)
	if record == nil {
		return AttachMediaResult{}, fmt.Errorf("attach media to %q: %w", xref, ErrUnknownXRef)
	}
	links, ok := record.MediaLinks()
	if !ok {
		return AttachMediaResult{}, fmt.Errorf("attach media to %s record %q: %w", record.Type, xref, ErrNoMediaLinks)
	}

	naming := d.mediaRecordsNaming(path)
	for _, link := range links {
		if link != nil && naming[link.MediaXRef] {
			return AttachMediaResult{MediaXRef: link.MediaXRef, Status: AttachAlreadyLinked}, nil
		}
	}

	result := AttachMediaResult{Status: AttachLinked}
	for _, r := range d.Records {
		if r != nil && naming[r.XRef] {
			result.MediaXRef = r.XRef
			break
		}
	}
	if result.MediaXRef == "" {
		result = AttachMediaResult{MediaXRef: d.FreeXRef("O"), Status: AttachCreated}
		if err := d.AddRecord(newMediaRecord(result.MediaXRef, path, d.declaredVersion())); err != nil {
			return AttachMediaResult{}, fmt.Errorf("attach media to %q: %w", xref, err)
		}
	}
	// The record was checked above to carry media links, so this cannot fail.
	_ = record.AddMediaLink(&MediaLink{MediaXRef: result.MediaXRef})
	return result, nil
}

// mediaRecordsNaming returns the XRefs of d's OBJE records with a FILE that
// names path, comparing with "\" read as "/". Records with no XRef are left
// out: nothing can point to them.
func (d *Document) mediaRecordsNaming(path string) map[string]bool {
	want := slashPath(path)
	naming := map[string]bool{}
	for _, r := range d.Records {
		media, ok := r.GetMediaObject()
		if !ok || r.XRef == "" {
			continue
		}
		for _, file := range media.Files {
			if file != nil && slashPath(file.FileRef) == want {
				naming[r.XRef] = true
				break
			}
		}
	}
	return naming
}

// slashPath returns p with each "\" replaced by "/".
func slashPath(p string) string {
	return strings.ReplaceAll(p, `\`, "/")
}

// newMediaRecord returns an OBJE record under xref holding the one file path,
// with the FORM AttachMedia documents for version v.
func newMediaRecord(xref, path string, v Version) *Record {
	ext := fileExtension(path)
	media := &MediaObject{XRef: xref, Files: []*MediaFile{{FileRef: path}}}
	switch v {
	case Version55:
		media.Form = legacyMediaForm(ext, legacyForms55)
	case Version551:
		media.Files[0].Form = legacyMediaForm(ext, legacyForms551)
	default:
		form, ok := mediatype.IANA(ext)
		if !ok {
			form = unknownMediaType
		}
		media.Files[0].Form = form
	}
	return &Record{XRef: xref, Type: RecordTypeMedia, Entity: media}
}

// legacyForms55 and legacyForms551 map an uppercase file extension to the
// spec's MULTIMEDIA_FORMAT value for GEDCOM 5.5 and 5.5.1, which spell JPEG
// and TIFF differently.
var (
	legacyForms55 = map[string]string{
		"BMP": "bmp", "GIF": "gif", "JPG": "jpeg", "JPEG": "jpeg", "OLE": "ole",
		"PCX": "pcx", "TIF": "tiff", "TIFF": "tiff", "WAV": "wav",
	}
	legacyForms551 = map[string]string{
		"BMP": "bmp", "GIF": "gif", "JPG": "jpg", "JPEG": "jpg", "OLE": "ole",
		"PCX": "pcx", "TIF": "tif", "TIFF": "tif", "WAV": "wav",
	}
)

// legacyMediaForm returns the 5.x FORM for ext from forms, or ext in
// lowercase when forms has no entry for it.
func legacyMediaForm(ext string, forms map[string]string) string {
	if form, ok := forms[strings.ToUpper(ext)]; ok {
		return form
	}
	return strings.ToLower(ext)
}

// fileExtension returns the extension of the last element of path, without
// its dot, or "" when it has none. "\" separates elements as "/" does.
func fileExtension(path string) string {
	base := slashPath(path)
	base = base[strings.LastIndex(base, "/")+1:]
	dot := strings.LastIndex(base, ".")
	if dot < 0 {
		return ""
	}
	return base[dot+1:]
}
