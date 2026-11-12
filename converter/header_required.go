package converter

import (
	"strconv"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// convertedSourceSystem is the HEAD.SOUR system ID written for a document
// whose header names no source system.
const convertedSourceSystem = "gedcom-go"

// unknownSubmitterName is the NAME of the SUBM record added for a document
// that has none.
const unknownSubmitterName = "Unknown"

// addRequiredHeaderStructures adds the HEAD.SOUR and HEAD.SUBM that GEDCOM
// 5.5 and 5.5.1 require ({1:1} both) and 7.0 makes optional, on a downgrade
// of a header that has neither.
//
// A missing SOUR is written as convertedSourceSystem, the system writing the
// converted file. A missing SUBM points at the document's first SUBM record;
// when it has none, a SUBM record with NAME unknownSubmitterName (NAME is
// required too) is added at the first free @SUBM<n>@ and pointed at. Each
// addition is a transformation naming it. The typed fields follow the tags,
// and a header without tags gets the typed fields alone, which the encoder
// writes from; transformHeader orders the tags afterwards.
func addRequiredHeaderStructures(doc *gedcom.Document, targetVersion gedcom.Version, report *gedcom.ConversionReport) {
	if doc.Header == nil {
		doc.Header = &gedcom.Header{}
	}
	header := doc.Header
	required := " (required by GEDCOM " + targetVersion.String() + ")"

	if headerValue(header, "SOUR", header.SourceSystem) == "" {
		header.SourceSystem = convertedSourceSystem
		setHeaderValue(header, "SOUR", convertedSourceSystem)
		report.AddTransformation(gedcom.Transformation{
			Type:        "HEAD_SOUR_ADDED",
			Description: "Added HEAD.SOUR " + convertedSourceSystem + required,
			Count:       1,
		})
	}

	if headerValue(header, "SUBM", header.Submitter) != "" {
		return
	}
	submitter := ""
	for _, record := range doc.Records {
		if record != nil && record.Type == gedcom.RecordTypeSubmitter && record.XRef != "" {
			submitter = record.XRef
			break
		}
	}
	if submitter == "" {
		submitter = addUnknownSubmitter(doc)
		report.AddTransformation(gedcom.Transformation{
			Type:        "SUBM_RECORD_ADDED",
			Description: "Added SUBM record " + submitter + " (NAME " + unknownSubmitterName + ") for HEAD.SUBM to point at" + required,
			Count:       1,
		})
	}
	header.Submitter = submitter
	setHeaderValue(header, "SUBM", submitter)
	report.AddTransformation(gedcom.Transformation{
		Type:        "HEAD_SUBM_ADDED",
		Description: "Added HEAD.SUBM " + submitter + required,
		Count:       1,
	})
}

// headerValue returns the value of the header's level-1 tag name when the
// header has raw tags, and typed otherwise.
func headerValue(header *gedcom.Header, name, typed string) string {
	if len(header.Tags) == 0 {
		return typed
	}
	for _, tag := range header.Tags {
		if tag != nil && tag.Level == 1 && tag.Tag == name {
			return tag.Value
		}
	}
	return ""
}

// setHeaderValue sets the value of the header's level-1 tag name, appending
// the tag when there is none. A header without raw tags is left to its typed
// fields.
func setHeaderValue(header *gedcom.Header, name, value string) {
	if len(header.Tags) == 0 {
		return
	}
	for _, tag := range header.Tags {
		if tag != nil && tag.Level == 1 && tag.Tag == name {
			tag.Value = value
			return
		}
	}
	header.Tags = append(header.Tags, &gedcom.Tag{Level: 1, Tag: name, Value: value})
}

// addUnknownSubmitter appends a SUBM record named unknownSubmitterName at the
// first free @SUBM<n>@ and returns its XRef.
func addUnknownSubmitter(doc *gedcom.Document) string {
	used := xrefsInUse(doc)
	xref := ""
	for n := 1; ; n++ {
		xref = "@SUBM" + strconv.Itoa(n) + "@"
		if !used[xref] {
			break
		}
	}
	record := &gedcom.Record{
		XRef: xref,
		Type: gedcom.RecordTypeSubmitter,
		Tags: []*gedcom.Tag{{Level: 1, Tag: "NAME", Value: unknownSubmitterName}},
	}
	doc.Records = append(doc.Records, record)
	if doc.XRefMap == nil {
		doc.XRefMap = map[string]*gedcom.Record{}
	}
	doc.XRefMap[xref] = record
	return xref
}
