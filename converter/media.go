package converter

import (
	"strings"

	"github.com/cacack/gedcom-go/gedcom"
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
//nolint:gocyclo // Processing media files and translations requires nested iteration
func transformMediaTypes(doc *gedcom.Document, targetVersion gedcom.Version, report *gedcom.ConversionReport) {
	var transformCount int
	var details []string

	for _, record := range doc.Records {
		if record.Type != gedcom.RecordTypeMedia {
			continue
		}

		media, ok := record.GetMediaObject()
		if !ok || media == nil {
			continue
		}

		// Transform Form field in each MediaFile
		for fileIdx, file := range media.Files {
			if file == nil || file.Form == "" {
				continue
			}

			oldValue := file.Form
			var newValue string

			switch targetVersion {
			case gedcom.Version70:
				newValue = toIANAMediaType(oldValue)
			case gedcom.Version55, gedcom.Version551:
				newValue = toLegacyMediaType(oldValue)
			}

			if newValue != "" && newValue != oldValue {
				file.Form = newValue
				transformCount++
				details = append(details, oldValue+" -> "+newValue)

				// Add per-item approximated note
				path := BuildNestedPath("OBJE", record.XRef, formatFileIndex(fileIdx), "FORM")
				var reason string
				if targetVersion == gedcom.Version70 {
					reason = "GEDCOM 7.0 uses IANA media types instead of legacy format identifiers"
				} else {
					reason = "GEDCOM 5.x uses legacy format identifiers instead of IANA media types"
				}
				report.AddApproximated(gedcom.ConversionNote{
					Path:     path,
					Original: oldValue,
					Result:   newValue,
					Reason:   reason,
				})
			}
		}

		// Also transform Form in MediaFile Translations
		for fileIdx, file := range media.Files {
			if file == nil {
				continue
			}
			for transIdx, trans := range file.Translations {
				if trans == nil || trans.Form == "" {
					continue
				}

				oldValue := trans.Form
				var newValue string

				switch targetVersion {
				case gedcom.Version70:
					newValue = toIANAMediaType(oldValue)
				case gedcom.Version55, gedcom.Version551:
					newValue = toLegacyMediaType(oldValue)
				}

				if newValue != "" && newValue != oldValue {
					trans.Form = newValue
					transformCount++
					details = append(details, oldValue+" -> "+newValue)

					// Add per-item approximated note
					path := BuildNestedPath("OBJE", record.XRef, formatFileIndex(fileIdx), formatTransIndex(transIdx), "FORM")
					var reason string
					if targetVersion == gedcom.Version70 {
						reason = "GEDCOM 7.0 uses IANA media types instead of legacy format identifiers"
					} else {
						reason = "GEDCOM 5.x uses legacy format identifiers instead of IANA media types"
					}
					report.AddApproximated(gedcom.ConversionNote{
						Path:     path,
						Original: oldValue,
						Result:   newValue,
						Reason:   reason,
					})
				}
			}
		}
	}

	if transformCount > 0 {
		report.AddTransformation(gedcom.Transformation{
			Type:        "MEDIA_TYPE_MAPPED",
			Description: "Converted media type formats for target GEDCOM version",
			Count:       transformCount,
			Details:     details,
		})
	}
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
