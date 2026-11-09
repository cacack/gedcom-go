package converter

import (
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// uidEventTags are the GEDCOM 7.0 individual and family events and
// attributes whose event detail defines UID, read from the 7.0
// substructures table (testdata/spec/gedcom-7.0/substructures.tsv): every
// superstructure of UID other than the seven record types.
var uidEventTags = map[string]bool{
	"ADOP": true, "ANUL": true, "BAPM": true, "BARM": true, "BASM": true, "BIRT": true,
	"BLES": true, "BURI": true, "CAST": true, "CENS": true, "CHR": true, "CHRA": true,
	"CONF": true, "CREM": true, "DEAT": true, "DIV": true, "DIVF": true, "DSCR": true,
	"EDUC": true, "EMIG": true, "ENGA": true, "EVEN": true, "FACT": true, "FCOM": true,
	"GRAD": true, "IDNO": true, "IMMI": true, "MARB": true, "MARC": true, "MARL": true,
	"MARR": true, "MARS": true, "NATI": true, "NATU": true, "NCHI": true, "NMR": true,
	"OCCU": true, "ORDN": true, "PROB": true, "PROP": true, "RELI": true, "RESI": true,
	"RETI": true, "SSN": true, "TITL": true, "WILL": true,
}

// uidRecordTypes are the record types whose level 1 defines UID in GEDCOM
// 7.0, from the same table. A 5.x NOTE record is an SNOTE by the time
// transformUIDTags runs (upgradeNoteRecordsForV7 precedes it).
var uidRecordTypes = map[gedcom.RecordType]bool{
	gedcom.RecordTypeIndividual: true, gedcom.RecordTypeFamily: true,
	gedcom.RecordTypeSource: true, gedcom.RecordTypeRepository: true,
	gedcom.RecordTypeMedia: true, gedcom.RecordTypeSharedNote: true,
	gedcom.RecordTypeSubmitter: true,
}

// transformUIDTags rewrites unique identifiers between their GEDCOM 7.0
// spelling, UID, and the vendor spelling GEDCOM 5.5/5.5.1 producers use for
// the same identifier, _UID (#715). _UID is the most common extension tag in
// real 5.5.x exports (MyHeritage, PAF, RootsMagic, Legacy, Family Historian
// all write it) and is exactly what 7.0 standardised as UID, so the
// identifier survives conversion in both directions instead of being dropped
// on a downgrade or left as an extension on an upgrade.
//
// On a downgrade every UID becomes _UID, at any level: an extension tag is
// legal anywhere. On an upgrade a _UID becomes UID only where 7.0 defines
// UID, at level 1 of the seven record types in uidRecordTypes and at level 2
// under an individual or family event or attribute (uidEventTags); a _UID
// anywhere else, such as under a NAME or on a SUBM's address, stays the
// extension it was. The value is carried unchanged: 7.0 says an existing
// globally unique identifier "should be used without modification", so a 5.x
// identifier that is not a UUID is still the right UID payload (whose type
// is a plain string).
//
// The typed UIDs fields are left as they are, since the decoder fills them
// from either spelling and the encoder writes them under the target's
// spelling. Each rename is recorded as a normalized note whose path names the
// enclosing event for a nested one, plus one aggregate transformation entry.
func transformUIDTags(doc *gedcom.Document, report *gedcom.ConversionReport, targetVersion gedcom.Version) {
	upgrade := targetVersion == gedcom.Version70
	from, to, typ := "UID", "_UID", "UID_TO_VENDOR_TAG"
	if upgrade {
		from, to, typ = "_UID", "UID", "UID_FROM_VENDOR_TAG"
	}
	total := 0
	for _, record := range doc.Records {
		if record == nil {
			continue
		}
		var parent string // the level-1 tag enclosing the current tag
		for _, tag := range record.Tags {
			if tag == nil {
				continue
			}
			if tag.Level == 1 {
				parent = tag.Tag
			}
			if tag.Tag != from || (upgrade && !uidDefinedAt(record.Type, tag.Level, parent)) {
				continue
			}
			tag.Tag = to
			path := []string{to}
			if tag.Level > 1 {
				path = []string{parent, to}
			}
			report.AddNormalized(gedcom.ConversionNote{
				Path:     BuildNestedPath(string(record.Type), record.XRef, path...),
				Original: from,
				Result:   to,
				Reason:   "Unique identifier written as " + to + " for GEDCOM " + targetVersion.String(),
			})
			total++
		}
	}
	if total > 0 {
		report.AddTransformation(gedcom.Transformation{
			Type:        typ,
			Description: "Rewrote " + from + " as " + to + " for GEDCOM " + targetVersion.String(),
			Count:       total,
		})
	}
}

// uidDefinedAt reports whether GEDCOM 7.0 defines UID at the given level of a
// record of type rt, where parent is the level-1 tag enclosing that level.
func uidDefinedAt(rt gedcom.RecordType, level int, parent string) bool {
	switch level {
	case 1:
		return uidRecordTypes[rt]
	case 2:
		return (rt == gedcom.RecordTypeIndividual || rt == gedcom.RecordTypeFamily) && uidEventTags[parent]
	}
	return false
}
