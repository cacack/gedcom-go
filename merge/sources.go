package merge

import (
	"github.com/cacack/gedcom-go/v3/gedcom"
)

// Conflict is one substructure of drop that Sources or Repositories did not
// carry to keep, because its line may occur only once in the record and keep
// holds a different one. Keep is keep's substructure, which the merged
// document has; Drop is drop's, repointed, which it does not, so a caller
// that wants its value (as a note, say) takes it from here.
type Conflict struct {
	Keep, Drop Substructure
}

// Sources is Individuals for two source records, with one difference: most
// level-1 lines of a source may occur only once, so drop's lines are not
// simply added beside keep's.
//
//   - A line that may occur only once (TITL, AUTH, PUBL, ABBR, TEXT, DATA,
//     CHAN; see below) is carried to keep when keep has none. When both have
//     one and they are identical, line for line, drop's is deduplicated.
//     When they differ, keep's stays and drop's is not carried: the pair is
//     listed in RecordMergeReport.Conflicts, and choosing between them is
//     the caller's job. Two records rarely share a CHAN, so expect one
//     there.
//   - Every other line (NOTE, OBJE, REFN, REPO, vendor tags) is added and
//     deduplicated as Individuals does it.
//   - Every pointer to drop anywhere in the document, at any level, in
//     typed fields and raw Tags alike (citations under records, events,
//     names and notes, vendor tags), is rewritten to name keep. No citation
//     is removed, even one that then repeats a sibling line for line, and
//     the lines under a citation (PAGE, DATA, QUAY, vendor tags) are not
//     touched: RecordMergeReport.Collapsed is always empty.
//   - drop is removed from Records and XRefMap.
//
// Which lines may occur only once is each version's own grammar: REPO does
// under a 5.5 source and not under a 5.5.1 or 7.0 one, RIN does in 5.5 and
// 5.5.1, CREA in 7.0. The version is the one the header declares, or else
// the one the decoder detected; for a document with neither, only the lines
// that may occur once in every version are held to it. A tag the version
// does not define, such as a vendor tag, may repeat.
//
// Errors, the treatment of doc and the rebuilding of keep's Entity are as
// for Individuals.
func Sources(doc *gedcom.Document, keep, drop string, opts RecordMergeOptions) (*gedcom.Document, *RecordMergeReport, error) {
	return mergeRecords(doc, gedcom.RecordTypeSource, keep, drop, opts)
}

// Repositories is Sources for two repository records. The lines that may
// occur only once are NAME, ADDR and CHAN, with RIN in 5.5 and 5.5.1 and
// CREA in 7.0; PHON, EMAIL, FAX, WWW, NOTE, REFN and vendor tags are added
// and deduplicated. Pointers to drop (the REPO lines of sources) are
// repointed to keep, and as in Individuals one that would then repeat a
// sibling line for line, such as the second REPO of a source that cited
// both repositories with the same call number, is removed and listed in
// RecordMergeReport.Collapsed.
func Repositories(doc *gedcom.Document, keep, drop string, opts RecordMergeOptions) (*gedcom.Document, *RecordMergeReport, error) {
	return mergeRecords(doc, gedcom.RecordTypeRepository, keep, drop, opts)
}

// onceTags is, per record type and GEDCOM version, the level-1 tags the
// version's grammar allows at most once. The library's grammar tables are
// generated into the validator, which merge does not import, so the tags are
// listed here; TestOnceTagsMatchSpec holds the list to the cardinalities
// vendored under testdata/spec.
var onceTags = map[gedcom.RecordType]map[gedcom.Version][]string{
	gedcom.RecordTypeSource: {
		gedcom.Version55:  {"ABBR", "AUTH", "CHAN", "DATA", "PUBL", "REPO", "RIN", "TEXT", "TITL"},
		gedcom.Version551: {"ABBR", "AUTH", "CHAN", "DATA", "PUBL", "RIN", "TEXT", "TITL"},
		gedcom.Version70:  {"ABBR", "AUTH", "CHAN", "CREA", "DATA", "PUBL", "TEXT", "TITL"},
	},
	gedcom.RecordTypeRepository: {
		gedcom.Version55:  {"ADDR", "CHAN", "NAME", "RIN"},
		gedcom.Version551: {"ADDR", "CHAN", "NAME", "RIN"},
		gedcom.Version70:  {"ADDR", "CHAN", "CREA", "NAME"},
	},
}

// onceOnly returns the level-1 tags that may occur only once in a record of
// type typ in doc: those of the version doc's header declares, else of the
// version detected, else those every version allows only once. It is nil
// for a record type merged without that rule (individuals and families).
func onceOnly(doc *gedcom.Document, typ gedcom.RecordType) map[string]bool {
	byVersion := onceTags[typ]
	if byVersion == nil {
		return nil
	}
	once := map[string]bool{}
	if doc.Header != nil {
		for _, v := range []gedcom.Version{doc.Header.DeclaredVersion, doc.Header.Version} {
			if tags, ok := byVersion[v]; ok {
				for _, tag := range tags {
					once[tag] = true
				}
				return once
			}
		}
	}
	count := map[string]int{}
	for _, tags := range byVersion {
		for _, tag := range tags {
			count[tag]++
		}
	}
	for tag, n := range count {
		once[tag] = n == len(byVersion)
	}
	return once
}

// collapsesRepeats reports whether merging two records of type typ removes a
// pointer line that, once repointed, would repeat a sibling. A source's
// pointers are citations, and a merge of sources leaves every citation in
// place.
func collapsesRepeats(typ gedcom.RecordType) bool {
	return typ != gedcom.RecordTypeSource
}
