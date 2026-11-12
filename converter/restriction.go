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
// It reads the RESN tags when the record has them, and otherwise the typed
// Restriction fields of the entity (record, event, attribute and media), the
// representation the encoder writes for a record with no Tags; the typed
// fields of a record with Tags follow the tags when the converter re-derives
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
		lost := false
		// rewrite converts one RESN payload in place and reports it under the
		// structure path leading to it.
		rewrite := func(payload *string, path ...string) {
			value, dropped := restrictionFor(*payload, upgrade)
			if value == *payload {
				return
			}
			original := *payload
			*payload = value
			reason := "Restriction notice written in GEDCOM " + targetVersion.String() + " spelling"
			if dropped != "" {
				reason = "GEDCOM " + targetVersion.String() + " takes one restriction value; kept the most protective, dropped " + dropped
				lost = true
			}
			report.AddNormalized(gedcom.ConversionNote{
				Path:     BuildNestedPath(string(record.Type), record.XRef, path...),
				Original: "RESN " + original,
				Result:   "RESN " + value,
				Reason:   reason,
			})
			total++
		}
		if len(record.Tags) == 0 {
			rewriteEntityRestrictions(record, rewrite)
		}
		var path []string // tags enclosing the current one, by level
		for _, tag := range record.Tags {
			if tag == nil || tag.Level < 1 {
				continue
			}
			if tag.Level-1 < len(path) {
				path = path[:tag.Level-1]
			}
			path = append(path, tag.Tag)
			if tag.Tag == "RESN" {
				rewrite(&tag.Value, path...)
			}
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

// rewriteEntityRestrictions passes each typed RESN payload of a record's
// entity to rewrite, with the structure path leading to it.
func rewriteEntityRestrictions(record *gedcom.Record, rewrite func(*string, ...string)) {
	field := func(payload *string, path ...string) {
		if *payload != "" {
			rewrite(payload, append(path, "RESN")...)
		}
	}
	facts := func(events []*gedcom.Event, attributes []*gedcom.Attribute) {
		for _, e := range events {
			if e != nil {
				field(&e.Restriction, string(e.Type))
			}
		}
		for _, a := range attributes {
			if a != nil {
				field(&a.Restriction, string(a.Type))
			}
		}
	}
	if ind, ok := record.GetIndividual(); ok && ind != nil {
		field(&ind.Restriction)
		facts(ind.Events, ind.Attributes)
	} else if fam, ok := record.GetFamily(); ok && fam != nil {
		field(&fam.Restriction)
		facts(fam.Events, fam.Attributes)
	} else if media, ok := record.GetMediaObject(); ok && media != nil {
		field(&media.Restriction)
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
