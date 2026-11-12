package converter

import (
	"slices"
	"strings"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// restrictionPrecedence lists the restriction notice values from most to
// least protective. GEDCOM 5.5/5.5.1 write one lower-case value; 7.0 writes
// upper case and allows a comma-separated list of them.
var restrictionPrecedence = []string{"privacy", "confidential", "locked"}

// transformRestrictions rewrites every RESN payload, at any level of any
// record, into the target version's spelling (#739): upper case for 7.0
// ("confidential" -> "CONFIDENTIAL"), lower case for 5.5/5.5.1. A value that
// is not one of the three restriction keywords is left as written.
//
// A 7.0 list with more than one keyword ("CONFIDENTIAL, LOCKED") has no 5.x
// form, since 5.x takes a single value, so a downgrade keeps the most
// protective keyword (privacy, then confidential, then locked) rather than
// risk publishing a restricted record, and reports the rest as data loss.
//
// It reads the RESN tags, so a record with no Tags is left as it is; the
// typed Restriction fields follow the tags when the converter re-derives
// each entity. Each rewrite is a Normalized note under one
// RESTRICTION_VALUES_MAPPED transformation.
func transformRestrictions(doc *gedcom.Document, report *gedcom.ConversionReport, targetVersion gedcom.Version) {
	upgrade := targetVersion == gedcom.Version70
	total := 0
	var lossRecords []string
	for _, record := range doc.Records {
		if record == nil {
			continue
		}
		var path []string // tags enclosing the current one, by level
		lost := false
		for _, tag := range record.Tags {
			if tag == nil || tag.Level < 1 {
				continue
			}
			if tag.Level-1 < len(path) {
				path = path[:tag.Level-1]
			}
			path = append(path, tag.Tag)
			if tag.Tag != "RESN" {
				continue
			}
			value, dropped := restrictionFor(tag.Value, upgrade)
			if value == tag.Value {
				continue
			}
			original := tag.Value
			tag.Value = value
			notePath := BuildNestedPath(string(record.Type), record.XRef, path...)
			reason := "Restriction notice written in GEDCOM " + targetVersion.String() + " spelling"
			if dropped != "" {
				reason = "GEDCOM " + targetVersion.String() + " takes one restriction value; kept the most protective, dropped " + dropped
				lost = true
			}
			report.AddNormalized(gedcom.ConversionNote{
				Path:     notePath,
				Original: "RESN " + original,
				Result:   "RESN " + value,
				Reason:   reason,
			})
			total++
		}
		if lost {
			lossRecords = append(lossRecords, record.XRef)
		}
	}
	if len(lossRecords) > 0 {
		report.AddDataLoss(gedcom.DataLossItem{
			Feature:         "RESN restriction lists",
			Reason:          "GEDCOM " + targetVersion.String() + " takes one restriction value; the less protective values were dropped",
			AffectedRecords: lossRecords,
		})
	}
	if total > 0 {
		report.AddTransformation(gedcom.Transformation{
			Type:        "RESTRICTION_VALUES_MAPPED",
			Description: "Rewrote RESN values for GEDCOM " + targetVersion.String(),
			Count:       total,
		})
	}
}

// restrictionFor returns the RESN payload value written for the target
// (upgrade: GEDCOM 7.0, otherwise 5.5/5.5.1) and, on a downgrade of a list,
// the keywords dropped from it joined by ", ". A value holding anything but
// restriction keywords is returned unchanged.
func restrictionFor(value string, upgrade bool) (result, dropped string) {
	var keywords []string
	for _, part := range strings.Split(value, ",") {
		keyword := strings.ToLower(strings.TrimSpace(part))
		if !slices.Contains(restrictionPrecedence, keyword) {
			return value, ""
		}
		keywords = append(keywords, keyword)
	}
	if upgrade {
		return strings.ToUpper(strings.Join(keywords, ", ")), ""
	}
	if len(keywords) == 1 {
		return keywords[0], ""
	}
	var kept string
	for _, candidate := range restrictionPrecedence {
		if slices.Contains(keywords, candidate) {
			kept = candidate
			break
		}
	}
	var rest []string
	for _, keyword := range keywords {
		if keyword != kept && !slices.Contains(rest, keyword) {
			rest = append(rest, keyword)
		}
	}
	return kept, strings.Join(rest, ", ")
}
