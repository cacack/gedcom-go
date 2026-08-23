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
//
// encoder.writeHeaderTags walks the same structure for its TargetVersion
// override and must agree with this on what "inside GEDC" means. The two are
// deliberately not shared -- see the note there -- so a change to either should
// be checked against the other.
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

	// GEDC.FORM ("2 FORM LINEAGE-LINKED") is a 5.5.x structure that 7.0
	// removed. The old encoder never wrote it, so an upgraded document lost it
	// silently; now the tag would be carried into 7.0 output that a strict 7.0
	// validator rejects. CHAR is the same story and is handled by updateCharTag.
	if removeGEDCForm(header) {
		report.AddTransformation(gedcom.Transformation{
			Type:        "GEDC_FORM_REMOVED",
			Description: "Removed GEDC.FORM (not a structure in GEDCOM 7.0)",
			Count:       1,
		})
	}

	// Note: SCHMA is only required if there are extension tags.
	// Future enhancement could analyze for custom tags and add SCHMA as needed.
	_ = hasHeaderTag(header.Tags, "SCHMA") // Used in potential future enhancements
}

// removeGEDCForm drops the FORM subtag of GEDC, reporting whether it was there.
// FORM is matched under GEDC specifically: PLAC.FORM is a different structure
// and remains valid in 7.0.
func removeGEDCForm(header *gedcom.Header) bool {
	inGEDC := false
	var kept []*gedcom.Tag
	removed := false

	for _, tag := range header.Tags {
		if tag == nil {
			kept = append(kept, tag)
			continue
		}
		if tag.Level <= 1 {
			inGEDC = tag.Level == 1 && tag.Tag == "GEDC"
		}
		if inGEDC && tag.Level == 2 && tag.Tag == "FORM" {
			removed = true
			continue
		}
		kept = append(kept, tag)
	}

	if removed {
		header.Tags = kept
	}
	return removed
}

// hasHeaderTag reports whether a level-1 header structure is present.
func hasHeaderTag(tags []*gedcom.Tag, name string) bool {
	for _, tag := range tags {
		if tag != nil && tag.Level <= 1 && tag.Tag == name {
			return true
		}
	}
	return false
}

// downgradeHeaderFrom70 prepares the header for GEDCOM 5.x from 7.0.
func downgradeHeaderFrom70(header *gedcom.Header, targetVersion gedcom.Version, report *gedcom.ConversionReport) {
	// Remove the SCHMA structure (not supported in 5.x).
	//
	// Header.Tags is a flat, level-encoded list, so dropping the SCHMA line
	// alone left its "2 TAG _SKYPEID ..." children behind. That was invisible
	// while the encoder discarded header tags; now they would be written out,
	// re-parenting under whichever level-1 structure preceded them -- maximal70
	// downgraded put its URI declarations under HEAD.NOTE, which is not a
	// structure in any version. removeHeaderTag takes the subtree.
	schmaRemoved := hasHeaderTag(header.Tags, "SCHMA")

	if schmaRemoved {
		header.Tags = removeHeaderTag(header.Tags, "SCHMA")
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
//
// The typed field and the raw CHAR tag are both updated, for the reason given
// on updateVersionTag: the encoder writes a decoded document's header from
// Header.Tags, so a typed field the tags contradict never reaches the output.
// CHAR differs from VERS in that its very presence is version-dependent -- 5.5
// and 5.5.1 require the line, 7.0 removed it -- so conversion has to add and
// remove it, not only rewrite its value.
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

	updateCharTag(header, targetVersion)
}

// updateCharTag brings the raw CHAR tag into line with the target version:
// present for 5.5 and 5.5.1, absent for 7.0.
//
// Without this, converting a 7.0 document down to 5.x produced a file with no
// CHAR line at all -- 7.0 headers carry no CHAR tag to rewrite, and the encoder
// preserves what the tags say rather than what the typed field says. The
// document looked correct in memory, so conversion-time validation saw nothing
// wrong; only the encoded file was invalid.
//
// The value written is the encoder's, not header.Encoding: Encode always emits
// UTF-8 and declares that, so a tag claiming ANSEL here would be overridden at
// write time anyway. Keeping them consistent avoids a header that says one
// thing in the tag and another in the file.
func updateCharTag(header *gedcom.Header, targetVersion gedcom.Version) {
	if len(header.Tags) == 0 {
		// Written from the typed fields, which are already set.
		return
	}

	if targetVersion == gedcom.Version70 {
		header.Tags = removeHeaderTag(header.Tags, "CHAR")
		return
	}

	for _, tag := range header.Tags {
		if tag != nil && tag.Level == 1 && tag.Tag == "CHAR" {
			tag.Value = string(gedcom.EncodingUTF8)
			return
		}
	}

	header.Tags = append(header.Tags,
		&gedcom.Tag{Level: 1, Tag: "CHAR", Value: string(gedcom.EncodingUTF8)},
	)
}

// removeHeaderTag drops a level-1 header structure and everything subordinate
// to it. Removing the parent alone would leave its children orphaned at a level
// the output no longer explains.
func removeHeaderTag(tags []*gedcom.Tag, name string) []*gedcom.Tag {
	var kept []*gedcom.Tag
	skipping := false

	for _, tag := range tags {
		if tag == nil {
			kept = append(kept, tag)
			continue
		}
		// Level <= 1 rather than == 1: a hand-built header may leave Level
		// unset on its top-level structures, and dropping SCHMA for such a
		// document worked before the removal became subtree-aware.
		if tag.Level <= 1 {
			skipping = tag.Tag == name
		}
		if skipping {
			continue
		}
		kept = append(kept, tag)
	}

	return kept
}
