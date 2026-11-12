package converter

import (
	"net/url"
	"slices"
	"strings"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// The EXID.TYPE base URIs GEDCOM 7.0 registers for the 5.x identifiers it
// replaced (https://gedcom.io/migrate, "AFN, RFN, RIN").
const (
	exidTypeAFN = "https://gedcom.io/terms/v7/AFN"
	exidTypeRFN = "https://gedcom.io/terms/v7/RFN"
	exidTypeRIN = "https://gedcom.io/terms/v7/RIN"
)

// exidRecordTypes are the records whose AFN, RFN and RIN have an EXID to
// become: every 7.0 record that takes EXID, NOTE included, since it is an
// SNOTE by the time this runs or soon after. An extension record is not
// governed by either grammar and is left as it is.
var exidRecordTypes = map[gedcom.RecordType]bool{
	gedcom.RecordTypeIndividual: true,
	gedcom.RecordTypeFamily:     true,
	gedcom.RecordTypeSource:     true,
	gedcom.RecordTypeRepository: true,
	gedcom.RecordTypeMedia:      true,
	gedcom.RecordTypeSubmitter:  true,
	gedcom.RecordTypeNote:       true,
	gedcom.RecordTypeSharedNote: true,
}

// transformIdentifiersToEXID rewrites each record's level-1 AFN, RFN and RIN,
// which GEDCOM 7.0 removed, as the EXID the 7.0 migration guide gives for
// each, so the identifier survives the upgrade:
//
//	1 AFN 123456789     ->  1 EXID 123456789
//	                          2 TYPE https://gedcom.io/terms/v7/AFN
//	1 RIN 9876          ->  1 EXID 9876
//	                          2 TYPE https://gedcom.io/terms/v7/RIN#MySystem
//	1 RFN Resource:5431 ->  1 EXID 5431
//	                          2 TYPE https://gedcom.io/terms/v7/RFN#Resource
//
// A RIN is unique only within the system that assigned it, named by the
// fragment: the header's SOUR (system ID), or no fragment when the header
// names none. An RFN's fragment is the registered resource before its first
// colon; one with no colon keeps its whole value and takes the bare base URI.
// Each rewrite is a Normalized note, with one aggregate transformation.
func transformIdentifiersToEXID(doc *gedcom.Document, report *gedcom.ConversionReport) {
	system := ""
	if doc.Header != nil {
		system = headerSourceSystem(doc.Header)
	}
	count := 0
	for _, record := range doc.Records {
		if record == nil || !exidRecordTypes[record.Type] {
			continue
		}
		for i := 0; i < len(record.Tags); i++ {
			tag := record.Tags[i]
			if tag == nil || tag.Level != 1 {
				continue
			}
			value, typ, ok := exidFor(tag.Tag, tag.Value, system)
			if !ok {
				continue
			}
			name, original := tag.Tag, tag.Tag+" "+tag.Value
			tag.Tag, tag.Value = "EXID", value
			record.Tags = slices.Insert(record.Tags, i+1, &gedcom.Tag{Level: 2, Tag: "TYPE", Value: typ})
			i++
			count++
			report.AddNormalized(gedcom.ConversionNote{
				Path:     BuildNestedPath(string(record.Type), record.XRef, name),
				Original: original,
				Result:   "EXID " + value + " (TYPE " + typ + ")",
				Reason:   "GEDCOM 7.0 replaced AFN, RFN and RIN with EXID",
			})
		}
	}
	if count > 0 {
		report.AddTransformation(gedcom.Transformation{
			Type:        "IDENTIFIER_TO_EXID",
			Description: "Converted AFN, RFN and RIN identifiers to EXID",
			Count:       count,
		})
	}
}

// exidFor returns the EXID value and TYPE for a 5.x identifier tag, and false
// for any other tag or an empty value.
func exidFor(tag, value, system string) (exid, typ string, ok bool) {
	if value == "" {
		return "", "", false
	}
	switch tag {
	case "AFN":
		return value, exidTypeAFN, true
	case "RIN":
		return value, withFragment(exidTypeRIN, system), true
	case "RFN":
		resource, id, found := strings.Cut(value, ":")
		if !found || resource == "" || id == "" {
			return value, exidTypeRFN, true
		}
		return id, withFragment(exidTypeRFN, resource), true
	}
	return "", "", false
}

// withFragment appends fragment to base as a URI fragment, escaped, or
// returns base when fragment is empty.
func withFragment(base, fragment string) string {
	fragment = strings.TrimSpace(fragment)
	if fragment == "" {
		return base
	}
	return base + "#" + (&url.URL{Fragment: fragment}).EscapedFragment()
}

// headerSourceSystem returns the header's SOUR value: its level-1 SOUR tag
// when the header has raw tags, its typed SourceSystem otherwise.
func headerSourceSystem(header *gedcom.Header) string {
	if len(header.Tags) == 0 {
		return header.SourceSystem
	}
	for _, tag := range header.Tags {
		if tag != nil && tag.Level == 1 && tag.Tag == "SOUR" {
			return tag.Value
		}
	}
	return ""
}
