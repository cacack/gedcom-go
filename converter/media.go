package converter

import (
	"strings"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// legacyToIANA maps legacy GEDCOM 5.5/5.5.1 media types to IANA media types.
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

// ianaToLegacy maps IANA media types to legacy GEDCOM 5.5/5.5.1 media types.
var ianaToLegacy = map[string]string{
	"image/jpeg":      "JPG",
	"image/png":       "PNG",
	"image/gif":       "GIF",
	"image/tiff":      "TIFF",
	"image/bmp":       "BMP",
	"audio/mpeg":      "MP3",
	"audio/wav":       "WAV",
	"video/mp4":       "MP4",
	"video/mpeg":      "MPEG",
	"video/x-msvideo": "AVI",
	"application/pdf": "PDF",
	"text/plain":      "TXT",
}

// transformMediaTypes updates media type formats in OBJE records based on target version.
// GEDCOM 5.5/5.5.1 uses short formats (JPG, PNG), while GEDCOM 7.0 uses IANA media types.
//
// A record's Tags are rewritten (FILE.FORM and FILE.TRAN.FORM), since they are
// what the encoder writes and what the record's entity is re-derived from
// (#777); a record with no Tags has its MediaObject entity rewritten instead.
// FILE.TRAN is 7.0-only and a downgrade drops it from Tags (dropV7OnlyTags),
// so its FORM is mapped there only on a conversion to 7.0.
// Each value actually changed is an Approximated note and counts toward one
// MEDIA_TYPE_MAPPED transformation.
func transformMediaTypes(doc *gedcom.Document, targetVersion gedcom.Version, report *gedcom.ConversionReport) {
	m := &mediaTypeMapper{target: targetVersion, report: report}
	for _, record := range doc.Records {
		if record == nil || record.Type != gedcom.RecordTypeMedia {
			continue
		}
		if len(record.Tags) > 0 {
			m.mapTags(record)
		} else {
			m.mapEntity(record)
		}
	}

	if m.count > 0 {
		report.AddTransformation(gedcom.Transformation{
			Type:        "MEDIA_TYPE_MAPPED",
			Description: "Converted media type formats for target GEDCOM version",
			Count:       m.count,
			Details:     m.details,
		})
	}
}

// mediaTypeMapper carries one transformMediaTypes pass.
type mediaTypeMapper struct {
	target  gedcom.Version
	report  *gedcom.ConversionReport
	count   int
	details []string
}

// mapTags rewrites the FILE.FORM and FILE.TRAN.FORM values in record.Tags.
func (m *mediaTypeMapper) mapTags(record *gedcom.Record) {
	fileIdx, transIdx := -1, -1
	var path []string // tag names by level, starting at level 1
	for _, tag := range record.Tags {
		if tag == nil || tag.Level < 1 {
			continue
		}
		path = append(path[:min(tag.Level-1, len(path))], tag.Tag)
		switch strings.Join(path, ".") {
		case "FILE":
			fileIdx++
			transIdx = -1
		case "FILE.TRAN":
			transIdx++
		case "FILE.FORM":
			tag.Value = m.mapForm(record.XRef, tag.Value, formatFileIndex(fileIdx), "FORM")
		case "FILE.TRAN.FORM":
			if m.target == gedcom.Version70 {
				tag.Value = m.mapForm(record.XRef, tag.Value, formatFileIndex(fileIdx), formatTransIndex(transIdx), "FORM")
			}
		}
	}
}

// mapEntity rewrites the Form of each file and file translation in the
// record's MediaObject entity.
func (m *mediaTypeMapper) mapEntity(record *gedcom.Record) {
	media, ok := record.GetMediaObject()
	if !ok || media == nil {
		return
	}
	for fileIdx, file := range media.Files {
		if file == nil {
			continue
		}
		file.Form = m.mapForm(record.XRef, file.Form, formatFileIndex(fileIdx), "FORM")
		for transIdx, trans := range file.Translations {
			if trans == nil {
				continue
			}
			trans.Form = m.mapForm(record.XRef, trans.Form, formatFileIndex(fileIdx), formatTransIndex(transIdx), "FORM")
		}
	}
}

// mapForm returns form in the target version's media type form, reporting
// the change at path (below the OBJE record xref) when there is one.
func (m *mediaTypeMapper) mapForm(xref, form string, path ...string) string {
	if form == "" {
		return form
	}
	var mapped, reason string
	if m.target == gedcom.Version70 {
		mapped = toIANAMediaType(form)
		reason = "GEDCOM 7.0 uses IANA media types instead of legacy format identifiers"
	} else {
		mapped = toLegacyMediaType(form)
		reason = "GEDCOM 5.x uses legacy format identifiers instead of IANA media types"
	}
	if mapped == "" || mapped == form {
		return form
	}
	m.count++
	m.details = append(m.details, form+" -> "+mapped)
	m.report.AddApproximated(gedcom.ConversionNote{
		Path:     BuildNestedPath("OBJE", xref, path...),
		Original: form,
		Result:   mapped,
		Reason:   reason,
	})
	return mapped
}

// formatFileIndex formats a file index for path display.
func formatFileIndex(idx int) string {
	return "FILE[" + itoa(idx) + "]"
}

// formatTransIndex formats a translation index for path display.
func formatTransIndex(idx int) string {
	return "TRAN[" + itoa(idx) + "]"
}

// itoa converts an int to a string without importing strconv.
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var digits []byte
	for i > 0 {
		digits = append([]byte{byte('0' + i%10)}, digits...)
		i /= 10
	}
	return string(digits)
}

// toIANAMediaType converts a legacy media type to IANA format.
// Returns the IANA media type if a mapping exists, or empty string if not found.
// If the input already contains a slash (indicating IANA format), it is returned as-is.
func toIANAMediaType(legacy string) string {
	upper := strings.ToUpper(strings.TrimSpace(legacy))
	if iana, ok := legacyToIANA[upper]; ok {
		return iana
	}
	// Already in IANA format
	if strings.Contains(legacy, "/") {
		return legacy
	}
	return ""
}

// toLegacyMediaType converts an IANA media type to legacy format.
// Returns the legacy media type if a mapping exists, or empty string if not found.
// If the input does not contain a slash (indicating legacy format), it is returned as-is.
func toLegacyMediaType(iana string) string {
	lower := strings.ToLower(strings.TrimSpace(iana))
	if legacy, ok := ianaToLegacy[lower]; ok {
		return legacy
	}
	// Already in legacy format
	if !strings.Contains(iana, "/") {
		return iana
	}
	return ""
}
