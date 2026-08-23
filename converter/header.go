package converter

import "github.com/cacack/gedcom-go/v2/gedcom"

// transformHeader updates the header for the target version.
func transformHeader(doc *gedcom.Document, targetVersion gedcom.Version, report *gedcom.ConversionReport) {
	if doc.Header == nil {
		doc.Header = &gedcom.Header{}
	}

	switch targetVersion {
	case gedcom.Version70:
		upgradeHeaderTo70(doc.Header, report)
	case gedcom.Version55, gedcom.Version551:
		downgradeHeaderFrom70(doc.Header, targetVersion, report)
	}

	updateEncoding(doc.Header, targetVersion, report)
	updateVersionTag(doc.Header, targetVersion)
}

// updateVersionTag rewrites GEDC.VERS in the raw header tags to the target
// version.
//
// The typed Header.Version is set by the caller, but the encoder writes a
// decoded document's header from Header.Tags -- so leaving the tag alone would
// emit a converted document still declaring its source version. Conversion owns
// the version change, so it owns the tag that states it.
//
// VERS is matched under GEDC only: it also appears under SOUR as the source
// system's own version, which a conversion must not touch.
func updateVersionTag(header *gedcom.Header, targetVersion gedcom.Version) {
	// Nothing to keep in sync: a header with no raw tags is written from the
	// typed fields, which the caller already set.
	if len(header.Tags) == 0 {
		return
	}

	inGEDC := false
	updated := false

	for _, tag := range header.Tags {
		// A nil tag has no level to read; it also cannot open or close a GEDC
		// structure, so the surrounding context carries across it unchanged.
		if tag == nil {
			continue
		}

		if tag.Level <= 1 {
			inGEDC = tag.Level == 1 && tag.Tag == "GEDC"
		}

		if inGEDC && tag.Level == 2 && tag.Tag == "VERS" {
			tag.Value = targetVersion.String()
			updated = true
		}
	}

	if updated {
		return
	}

	// Some 5.5 files carry no GEDC block at all (royal92.ged is one). A
	// conversion has to leave the target version stated somewhere the encoder
	// will write it, so the structure is added rather than assumed.
	header.Tags = append(header.Tags,
		&gedcom.Tag{Level: 1, Tag: "GEDC"},
		&gedcom.Tag{Level: 2, Tag: "VERS", Value: targetVersion.String()},
	)
}

// upgradeHeaderTo70 prepares the header for GEDCOM 7.0.
func upgradeHeaderTo70(header *gedcom.Header, report *gedcom.ConversionReport) {
	// GEDCOM 7.0 requires UTF-8 encoding
	if header.Encoding != gedcom.EncodingUTF8 {
		oldEncoding := header.Encoding
		header.Encoding = gedcom.EncodingUTF8
		report.AddTransformation(gedcom.Transformation{
			Type:        "ENCODING_UPDATED",
			Description: "Updated encoding to UTF-8 (required for GEDCOM 7.0)",
			Count:       1,
			Details:     []string{"From: " + string(oldEncoding), "To: UTF-8"},
		})
	}

	// Check if SCHMA tag exists in header tags
	hasSCHMA := false
	for _, tag := range header.Tags {
		if tag != nil && tag.Tag == "SCHMA" {
			hasSCHMA = true
			break
		}
	}

	// Note: SCHMA is only required if there are extension tags.
	// Future enhancement could analyze for custom tags and add SCHMA as needed.
	_ = hasSCHMA // Used in potential future enhancements
}

// downgradeHeaderFrom70 prepares the header for GEDCOM 5.x from 7.0.
func downgradeHeaderFrom70(header *gedcom.Header, targetVersion gedcom.Version, report *gedcom.ConversionReport) {
	// Remove SCHMA tag (not supported in 5.x)
	var newTags []*gedcom.Tag
	schmaRemoved := false
	for _, tag := range header.Tags {
		// A nil tag is not a SCHMA; it lands in newTags unchanged so the
		// rebuilt slice keeps every element the caller put in it.
		if tag != nil && tag.Tag == "SCHMA" {
			schmaRemoved = true
			continue
		}
		newTags = append(newTags, tag)
	}

	if schmaRemoved {
		header.Tags = newTags
		report.AddTransformation(gedcom.Transformation{
			Type:        "SCHMA_REMOVED",
			Description: "Removed SCHMA tag (not supported in GEDCOM 5.x)",
			Count:       1,
		})
		report.AddDataLoss(gedcom.DataLossItem{
			Feature: "SCHMA schema definitions",
			Reason:  "SCHMA tag not supported in GEDCOM " + targetVersion.String(),
		})
	}
}

// updateEncoding sets the appropriate encoding for the target version.
func updateEncoding(header *gedcom.Header, targetVersion gedcom.Version, _ *gedcom.ConversionReport) {
	switch targetVersion {
	case gedcom.Version70:
		// GEDCOM 7.0 is always UTF-8 (already handled in upgradeHeaderTo70)
		header.Encoding = gedcom.EncodingUTF8
	case gedcom.Version551:
		// GEDCOM 5.5.1 supports UTF-8 (preferred) or ANSEL
		if header.Encoding == "" {
			header.Encoding = gedcom.EncodingUTF8
		}
	case gedcom.Version55:
		// GEDCOM 5.5 typically uses ANSEL, but UTF-8 via UNICODE is possible
		if header.Encoding == "" {
			header.Encoding = gedcom.EncodingANSEL
		}
	}
}
