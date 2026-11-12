package converter

import (
	"strings"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// pedigreeForRelation maps a _FREL/_MREL value to the PEDI value (lower
// case, as GEDCOM 5.5/5.5.1 spell it) that says the same thing, or "" for a
// relationship PEDI has no value for (Step, Guardian, Private, ...).
func pedigreeForRelation(relation string) string {
	switch strings.ToLower(strings.TrimSpace(relation)) {
	case "natural", "birth":
		return "birth"
	case "adopted":
		return "adopted"
	case "foster":
		return "foster"
	case "sealing":
		return "sealing"
	}
	return ""
}

// synthesizePedigrees adds a PEDI to a child's FAMC from the _FREL/_MREL
// beneath the family's CHIL line, on an upgrade (#732). Family Tree Maker,
// Ancestry and Legacy record adoption this way and rarely write PEDI, which
// is the only place GEDCOM itself records it.
//
// PEDI is one value for the child's link to the whole family, so it is added
// only when the mapping is unambiguous: each parent the family names (HUSB
// for _FREL, WIFE for _MREL; either one when it names neither) has a
// relationship, every relationship maps to the same PEDI value, and that
// value is not birth (the GEDCOM default, which adding would only restate).
// A FAMC that already has a PEDI, or a child with no FAMC to the family, is
// left alone. It reads the CHIL and FAMC tags, not Family.ChildLinks, so a
// record with no Tags gets no PEDI. The vendor tags stay where they are.
// Each PEDI added is a
// Normalized note under one PEDIGREE_FROM_VENDOR_TAG transformation.
func synthesizePedigrees(doc *gedcom.Document, report *gedcom.ConversionReport, targetVersion gedcom.Version) {
	pedigrees := familyPedigrees(doc)
	if len(pedigrees) == 0 {
		return
	}
	total := 0
	for _, record := range doc.Records {
		if record == nil || record.Type != gedcom.RecordTypeIndividual {
			continue
		}
		var out []*gedcom.Tag
		for i, tag := range record.Tags {
			out = append(out, tag)
			if tag == nil || tag.Level != 1 || tag.Tag != "FAMC" || hasSubordinate(record.Tags, i, "PEDI") {
				continue
			}
			pedi := pedigrees[pedigreeKey{family: strings.TrimSpace(tag.Value), child: record.XRef}]
			if pedi.value == "" {
				continue
			}
			value := pedi.value
			if targetVersion == gedcom.Version70 {
				value = strings.ToUpper(value)
			}
			// PEDI is the first substructure of a FAMC, before any NOTE.
			out = append(out, &gedcom.Tag{Level: 2, Tag: "PEDI", Value: value})
			report.AddNormalized(gedcom.ConversionNote{
				Path:     BuildNestedPath(string(record.Type), record.XRef, "FAMC", "PEDI"),
				Original: pedi.source,
				Result:   "PEDI " + value,
				Reason:   "Pedigree written from the vendor relationship tags under " + strings.TrimSpace(tag.Value) + " CHIL",
			})
			total++
		}
		if len(out) != len(record.Tags) {
			record.Tags = out
		}
	}
	if total > 0 {
		report.AddTransformation(gedcom.Transformation{
			Type:        "PEDIGREE_FROM_VENDOR_TAG",
			Description: "Added FAMC.PEDI from _FREL/_MREL for GEDCOM " + targetVersion.String(),
			Count:       total,
		})
	}
}

// pedigreeKey names one child's link to one family.
type pedigreeKey struct{ family, child string }

// pedigree is a PEDI value derived from a CHIL's vendor tags, with those
// tags as written for the report.
type pedigree struct{ value, source string }

// familyPedigrees returns the unambiguous non-birth PEDI value of every
// family's CHIL lines, keyed by family and child, as described on
// synthesizePedigrees.
func familyPedigrees(doc *gedcom.Document) map[pedigreeKey]pedigree {
	out := make(map[pedigreeKey]pedigree)
	for _, record := range doc.Records {
		if record == nil || record.Type != gedcom.RecordTypeFamily {
			continue
		}
		husband, wife := hasLevel1Tag(record.Tags, "HUSB"), hasLevel1Tag(record.Tags, "WIFE")
		for i, tag := range record.Tags {
			if tag == nil || tag.Level != 1 || tag.Tag != "CHIL" {
				continue
			}
			father, mother := subordinateValue(record.Tags, i, "_FREL"), subordinateValue(record.Tags, i, "_MREL")
			value := agreedPedigree(father, mother, husband, wife)
			if value == "" || value == "birth" {
				continue
			}
			var source []string
			if father != "" {
				source = append(source, "_FREL "+father)
			}
			if mother != "" {
				source = append(source, "_MREL "+mother)
			}
			out[pedigreeKey{family: record.XRef, child: strings.TrimSpace(tag.Value)}] = pedigree{
				value:  value,
				source: strings.Join(source, ", "),
			}
		}
	}
	return out
}

// agreedPedigree returns the PEDI value both relationships agree on, or ""
// when a required one is missing, one has no PEDI value, or they disagree.
// A relationship is required for each parent the family names; when it names
// neither, any one present is enough.
func agreedPedigree(father, mother string, husband, wife bool) string {
	if !husband && !wife {
		husband, wife = father != "", mother != ""
	}
	var values []string
	if husband {
		values = append(values, pedigreeForRelation(father))
	}
	if wife {
		values = append(values, pedigreeForRelation(mother))
	}
	if len(values) == 0 {
		return ""
	}
	for _, v := range values[1:] {
		if v != values[0] {
			return ""
		}
	}
	return values[0]
}

// marriedNamesToV7 adds a NAME with TYPE MARRIED for each _MARNM beneath an
// individual's NAME, on an upgrade to GEDCOM 7.0 (#732). MyHeritage writes
// the married surname alone as _MARNM; 7.0 records a married name as its own
// NAME structure. The new NAME follows the one carrying _MARNM, with the
// _MARNM value as its surname in place of that name's surname (or the whole
// value, when _MARNM has a slash and so is a full name already). A matching
// TYPE MARRIED name the individual already has, or that an earlier _MARNM
// on the same individual added, is not added again. _MARNM
// stays where it is. It reads the NAME tags, not PersonalName.MarriedName, so
// a record with no Tags gets no new NAME. Each NAME added is a Normalized
// note under one
// MARRIED_NAME_FROM_VENDOR_TAG transformation.
func marriedNamesToV7(doc *gedcom.Document, report *gedcom.ConversionReport) {
	total := 0
	for _, record := range doc.Records {
		if record == nil || record.Type != gedcom.RecordTypeIndividual {
			continue
		}
		var out []*gedcom.Tag
		var pending []*gedcom.Tag // the NAME to add after the current NAME block
		for i, tag := range record.Tags {
			if tag != nil && tag.Level <= 1 {
				out, pending = append(out, pending...), nil
			}
			out = append(out, tag)
			if tag == nil || tag.Level != 1 || tag.Tag != "NAME" {
				continue
			}
			married := subordinateValue(record.Tags, i, "_MARNM")
			if married == "" {
				continue
			}
			value := marriedNameValue(tag.Value, married)
			if hasMarriedName(record.Tags, value) || hasMarriedName(out, value) {
				continue
			}
			pending = []*gedcom.Tag{
				{Level: 1, Tag: "NAME", Value: value},
				{Level: 2, Tag: "TYPE", Value: "MARRIED"},
			}
			report.AddNormalized(gedcom.ConversionNote{
				Path:     BuildNestedPath(string(record.Type), record.XRef, "NAME", "_MARNM"),
				Original: "_MARNM " + married,
				Result:   "NAME " + value + " (TYPE MARRIED)",
				Reason:   "Married name written as a TYPE MARRIED name for GEDCOM 7.0",
			})
			total++
		}
		out = append(out, pending...)
		if len(out) != len(record.Tags) {
			record.Tags = out
		}
	}
	if total > 0 {
		report.AddTransformation(gedcom.Transformation{
			Type:        "MARRIED_NAME_FROM_VENDOR_TAG",
			Description: "Added TYPE MARRIED names from _MARNM for GEDCOM 7.0",
			Count:       total,
		})
	}
}

// marriedNameValue returns the NAME value of the married name: married itself
// when it holds a slash (a full name), otherwise name with its /surname/
// replaced by /married/, or /married/ appended when name has no surname. As
// in the decoder, the last slash pair delimits the surname, so the married
// name reads back with married as its Surname.
func marriedNameValue(name, married string) string {
	if strings.Contains(married, "/") {
		return married
	}
	name = strings.TrimSpace(name)
	surname := "/" + married + "/"
	last := strings.LastIndex(name, "/")
	if last < 0 {
		return strings.TrimSpace(name + " " + surname)
	}
	open := strings.LastIndex(name[:last], "/")
	if open < 0 {
		return name[:last] + surname
	}
	return name[:open] + surname + name[last+1:]
}

// hasMarriedName reports whether tags hold a level-1 NAME of value whose TYPE
// is MARRIED (in any case).
func hasMarriedName(tags []*gedcom.Tag, value string) bool {
	for i, tag := range tags {
		if tag != nil && tag.Level == 1 && tag.Tag == "NAME" && strings.TrimSpace(tag.Value) == value &&
			strings.EqualFold(subordinateValue(tags, i, "TYPE"), "MARRIED") {
			return true
		}
	}
	return false
}

// hasSubordinate reports whether the tag at tags[parent] has a direct
// subordinate named name.
func hasSubordinate(tags []*gedcom.Tag, parent int, name string) bool {
	return subordinateIndex(tags, parent, name) >= 0
}

// subordinateValue returns the trimmed value of the first direct subordinate
// of tags[parent] named name, or "".
func subordinateValue(tags []*gedcom.Tag, parent int, name string) string {
	if i := subordinateIndex(tags, parent, name); i >= 0 {
		return strings.TrimSpace(tags[i].Value)
	}
	return ""
}

// subordinateIndex returns the index of the first direct subordinate of
// tags[parent] named name, or -1.
func subordinateIndex(tags []*gedcom.Tag, parent int, name string) int {
	level := tags[parent].Level
	for i := parent + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag == nil {
			continue
		}
		if tag.Level <= level {
			break
		}
		if tag.Level == level+1 && tag.Tag == name {
			return i
		}
	}
	return -1
}
