// structure.go provides the record-structure checks run by ValidateAll:
// pointers to records that do not exist, individuals with no NAME, and
// families with no members.
//
// These three rules predate the Issue model. Until v3 they were reachable only
// through the []error Validate path, as bare string codes with no line number.
// They now run as part of ValidateAll, report exported Code* constants, and
// carry the source line of the offending tag or record.

package validator

import (
	"fmt"
	"strings"

	"github.com/cacack/gedcom-go/v2/gedcom"
)

// validateBrokenXRefs reports every pointer-shaped raw tag value that names
// no record in doc.XRefMap, as CodeBrokenXRef with the tag's line number.
//
// The pointers ReferenceValidator already checks through the typed entity --
// level-1 FAMC, FAMS and SOUR on an individual, level-1 HUSB, WIFE and CHIL on
// a family -- are skipped, so a broken FAMC is reported once, as
// CodeOrphanedFAMC (with its line), rather than twice. A raw pointer is
// skipped only when its value is one the typed entity holds: a malformed
// family with two level-1 HUSB lines keeps only the last in Family.Husband,
// and the other is still reported here. Records without a typed entity have
// no ReferenceValidator coverage and are checked in full.
//
// CONT and CONC lines are text continuations, never pointers, and are skipped.
// The GEDCOM 7.0 "@VOID@" sentinel is a deliberate null pointer, not a broken
// one; see [gedcom.IsPointerXRef].
func validateBrokenXRefs(doc *gedcom.Document) []Issue {
	var issues []Issue
	for _, record := range doc.Records {
		if record == nil {
			continue
		}
		// Built on the record's first dangling level-1 pointer: most records
		// have none, and need no map. Checking existence first does not
		// change which pointers are skipped, because take only consumes the
		// counts of a (tag, xref) pair whose xref names no record.
		var covered pointerCounts

		for _, tag := range record.Tags {
			if tag == nil || tag.Tag == "CONT" || tag.Tag == "CONC" {
				continue
			}
			xref := strings.TrimSpace(tag.Value)
			if !gedcom.IsPointerXRef(xref) {
				continue
			}
			if doc.XRefMap[xref] != nil {
				continue
			}
			if tag.Level == 1 {
				if covered == nil {
					covered = typedPointers(record)
				}
				if covered.take(tag.Tag, xref) {
					continue
				}
			}
			issues = append(issues, NewIssue(
				SeverityError,
				CodeBrokenXRef,
				fmt.Sprintf("%s reference to non-existent record %s", tag.Tag, xref),
				record.XRef,
			).
				WithRelatedXRef(xref).
				WithLineNumber(tag.LineNumber).
				WithDetail("tag", tag.Tag))
		}
	}
	return issues
}

// pointerCounts counts, per tag and XRef, the level-1 pointers
// ReferenceValidator checks on a record's typed entity.
type pointerCounts map[string]int

// typedPointers returns the pointers ReferenceValidator checks on record's
// typed entity: FAMC, FAMS and SOUR on an individual; HUSB, WIFE and CHIL on a
// family. It is empty for a record with neither entity.
//
// Keep it in step with ReferenceValidator: a pointer ReferenceValidator starts
// checking through the typed entity must be added here, or it is reported
// twice (once by each check).
func typedPointers(record *gedcom.Record) pointerCounts {
	counts := pointerCounts{}
	if ind, ok := record.GetIndividual(); ok {
		for _, link := range ind.ChildInFamilies {
			counts.add("FAMC", link.FamilyXRef)
		}
		for _, link := range ind.SpouseInFamilies {
			counts.add("FAMS", link.FamilyXRef)
		}
		for _, cite := range ind.SourceCitations {
			if cite != nil {
				counts.add("SOUR", cite.SourceXRef)
			}
		}
	}
	if fam, ok := record.GetFamily(); ok {
		counts.add("HUSB", fam.Husband)
		counts.add("WIFE", fam.Wife)
		for _, child := range fam.Children {
			counts.add("CHIL", child)
		}
	}
	return counts
}

func (c pointerCounts) add(tag, xref string) {
	if xref = strings.TrimSpace(xref); xref != "" {
		c[tag+"\x00"+xref]++
	}
}

// take reports whether a level-1 tag pointing at xref is one the typed entity
// holds, consuming one occurrence so that a raw pointer repeated more often
// than the entity holds it is still reported.
func (c pointerCounts) take(tag, xref string) bool {
	key := tag + "\x00" + xref
	if c[key] == 0 {
		return false
	}
	c[key]--
	return true
}

// validateRecordStructure reports individuals with no name
// (CodeMissingRequiredField) and families with no members (CodeEmptyFamily),
// each carrying the line number of the record's level-0 line.
//
// Both the raw tags and the typed entity are consulted, so a record built in
// code with a typed entity and no Tags -- which the encoder writes from the
// entity -- is not reported for lacking tags it will be written with.
func validateRecordStructure(doc *gedcom.Document) []Issue {
	var issues []Issue
	for _, record := range doc.Records {
		if record == nil {
			continue
		}
		switch record.Type {
		case gedcom.RecordTypeIndividual:
			if !individualHasName(record) {
				issues = append(issues, NewIssue(
					SeverityWarning,
					CodeMissingRequiredField,
					"Individual record has no NAME",
					record.XRef,
				).
					WithLineNumber(record.LineNumber).
					WithDetail("field", "NAME"))
			}
		case gedcom.RecordTypeFamily:
			if !familyHasMembers(record) {
				issues = append(issues, NewIssue(
					SeverityWarning,
					CodeEmptyFamily,
					"Family record has no members (no HUSB, WIFE, or CHIL)",
					record.XRef,
				).
					WithLineNumber(record.LineNumber))
			}
		}
	}
	return issues
}

// individualHasName reports whether an individual record has a level-1 NAME
// tag or a typed name.
func individualHasName(record *gedcom.Record) bool {
	if ind, ok := record.GetIndividual(); ok && len(ind.Names) > 0 {
		return true
	}
	return hasLevelOneTag(record, "NAME")
}

// familyHasMembers reports whether a family record has a level-1 HUSB, WIFE
// or CHIL tag, or a typed husband, wife or child.
func familyHasMembers(record *gedcom.Record) bool {
	if fam, ok := record.GetFamily(); ok &&
		(fam.Husband != "" || fam.Wife != "" || len(fam.Children) > 0) {
		return true
	}
	return hasLevelOneTag(record, "HUSB", "WIFE", "CHIL")
}

// hasLevelOneTag reports whether record has a level-1 tag named any of names.
// Only level 1 counts: "2 HUSB" under a family event is the husband's age
// structure, not a member.
func hasLevelOneTag(record *gedcom.Record, names ...string) bool {
	for _, tag := range record.Tags {
		if tag == nil || tag.Level != 1 {
			continue
		}
		for _, name := range names {
			if tag.Tag == name {
				return true
			}
		}
	}
	return false
}
