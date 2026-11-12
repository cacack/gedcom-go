package converter

import (
	"strconv"
	"strings"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// enumKind names an enumerated payload whose vocabulary differs between
// GEDCOM 5.5/5.5.1 and 7.0 (issue #704). Each is identified by its tag and
// where it sits; see enumKindAt.
type enumKind int

const (
	enumNone            enumKind = iota
	enumSex                      // INDI.SEX
	enumNameType                 // INDI.NAME.TYPE
	enumPedigree                 // FAMC.PEDI
	enumChildStatus              // FAMC.STAT
	enumOrdinanceStatus          // BAPL/CONL/ENDL/INIL/SLGC/SLGS.STAT
	enumMedia                    // MEDI
	enumRecordMedia              // OBJE.FILE.FORM.TYPE (5.5.1) / OBJE.FILE.FORM.MEDI (7.0)
	enumAdoptedBy                // FAMC.ADOP
	enumAssociation              // ASSO.RELA (5.x) / ASSO.ROLE (7.0)
	enumEventRole                // SOUR.EVEN.ROLE
	enumAge                      // AGE
)

// ordinanceTags are the LDS ordinances whose STAT draws from the ordinance
// status set.
var ordinanceTags = map[string]bool{
	"BAPL": true, "CONL": true, "ENDL": true, "INIL": true, "SLGC": true, "SLGS": true,
}

// v7Enums lists, for each enumerated payload, its GEDCOM 7.0 values
// (enumerationsets.tsv), each keyed by the uppercase GEDCOM 5.5/5.5.1 spelling
// that maps to it. Most spellings differ only in case; the ordinance status
// set also renames PRE-1970 and DNS/CAN. A set with OTHER is open: a 5.x value
// it does not list becomes OTHER with the value as its PHRASE.
var v7Enums = map[enumKind]map[string]string{
	enumSex:      {"M": "M", "F": "F", "U": "U", "X": "X"},
	enumNameType: {"AKA": "AKA", "BIRTH": "BIRTH", "IMMIGRANT": "IMMIGRANT", "MAIDEN": "MAIDEN", "MARRIED": "MARRIED", "PROFESSIONAL": "PROFESSIONAL", "OTHER": "OTHER"},
	enumPedigree: {"ADOPTED": "ADOPTED", "BIRTH": "BIRTH", "FOSTER": "FOSTER", "SEALING": "SEALING", "OTHER": "OTHER"},
	enumChildStatus: {
		"CHALLENGED": "CHALLENGED", "DISPROVEN": "DISPROVEN", "PROVEN": "PROVEN",
	},
	enumOrdinanceStatus: {
		"BIC": "BIC", "CANCELED": "CANCELED", "CHILD": "CHILD", "COMPLETED": "COMPLETED",
		"DNS": "DNS", "DNS/CAN": "DNS_CAN", "DNS_CAN": "DNS_CAN", "EXCLUDED": "EXCLUDED",
		"INFANT": "INFANT", "PRE-1970": "PRE_1970", "PRE_1970": "PRE_1970",
		"STILLBORN": "STILLBORN", "SUBMITTED": "SUBMITTED", "UNCLEARED": "UNCLEARED",
	},
	enumMedia: {
		"AUDIO": "AUDIO", "BOOK": "BOOK", "CARD": "CARD", "ELECTRONIC": "ELECTRONIC",
		"FICHE": "FICHE", "FILM": "FILM", "MAGAZINE": "MAGAZINE", "MANUSCRIPT": "MANUSCRIPT",
		"MAP": "MAP", "NEWSPAPER": "NEWSPAPER", "OTHER": "OTHER", "PHOTO": "PHOTO",
		"TOMBSTONE": "TOMBSTONE", "VIDEO": "VIDEO",
	},
	enumAdoptedBy: {"HUSB": "HUSB", "WIFE": "WIFE", "BOTH": "BOTH"},
}

// v5Enums lists, for each enumerated payload, the GEDCOM 5.5/5.5.1 spelling of
// each 7.0 value that grammar defines. A 7.0 value missing here has no 5.x
// equivalent: in a payload that admits user-defined text (v5OpenEnums) it is
// kept as that text, and otherwise it is kept and reported as data loss.
var v5Enums = map[enumKind]map[string]string{
	enumSex:      {"M": "M", "F": "F", "U": "U"},
	enumNameType: {"AKA": "aka", "BIRTH": "birth", "IMMIGRANT": "immigrant", "MAIDEN": "maiden", "MARRIED": "married"},
	enumPedigree: {"ADOPTED": "adopted", "BIRTH": "birth", "FOSTER": "foster", "SEALING": "sealing"},
	enumChildStatus: {
		"CHALLENGED": "challenged", "DISPROVEN": "disproven", "PROVEN": "proven",
	},
	enumOrdinanceStatus: {
		"BIC": "BIC", "CANCELED": "CANCELED", "CHILD": "CHILD", "COMPLETED": "COMPLETED",
		"DNS": "DNS", "DNS_CAN": "DNS/CAN", "EXCLUDED": "EXCLUDED", "INFANT": "INFANT",
		"PRE_1970": "PRE-1970", "STILLBORN": "STILLBORN", "SUBMITTED": "SUBMITTED",
		"UNCLEARED": "UNCLEARED",
	},
	enumMedia: {
		"AUDIO": "audio", "BOOK": "book", "CARD": "card", "ELECTRONIC": "electronic",
		"FICHE": "fiche", "FILM": "film", "MAGAZINE": "magazine", "MANUSCRIPT": "manuscript",
		"MAP": "map", "NEWSPAPER": "newspaper", "PHOTO": "photo", "TOMBSTONE": "tombstone",
		"VIDEO": "video",
	},
	enumAdoptedBy: {"HUSB": "HUSB", "WIFE": "WIFE", "BOTH": "BOTH"},
}

// v7Values is, for each enumerated payload in v7Enums, the set of values
// GEDCOM 7.0 defines for it.
var v7Values = func() map[enumKind]map[string]bool {
	m := make(map[enumKind]map[string]bool, len(v7Enums))
	for kind, values := range v7Enums {
		m[kind] = make(map[string]bool, len(values))
		for _, v := range values {
			m[kind][v] = true
		}
	}
	return m
}()

// v5OpenEnums are the 5.x payloads that admit user-defined text, so a 7.0
// value they do not define (NAME.TYPE PROFESSIONAL, MEDI OTHER) is kept as
// that text, and an OTHER with a PHRASE becomes the phrase.
var v5OpenEnums = map[enumKind]bool{enumNameType: true, enumMedia: true}

// v7Roles maps each GEDCOM 7.0 ROLE value to the word for it, which a 5.x
// RELA or role descriptor spells out.
var v7Roles = map[string]string{
	"CHIL": "child", "CLERGY": "clergy", "FATH": "father", "FRIEND": "friend",
	"GODP": "godparent", "HUSB": "husband", "MOTH": "mother", "MULTIPLE": "multiple",
	"NGHBR": "neighbor", "OFFICIATOR": "officiator", "PARENT": "parent",
	"SPOU": "spouse", "WIFE": "wife", "WITN": "witness", "OTHER": "other",
}

// v7RoleByWord is v7Roles inverted, keyed by the uppercase word.
var v7RoleByWord = func() map[string]string {
	m := make(map[string]string, len(v7Roles))
	for role, word := range v7Roles {
		m[strings.ToUpper(word)] = role
	}
	return m
}()

// v5EventRoles are the ROLE_IN_EVENT values 5.x defines besides a
// parenthesised role descriptor.
var v5EventRoles = map[string]bool{"CHIL": true, "HUSB": true, "WIFE": true, "MOTH": true, "FATH": true, "SPOU": true}

// v5AgeKeywords maps each 5.x AGE keyword to the 7.0 age the specification's
// migration notes give it; the keyword itself moves into PHRASE.
var v5AgeKeywords = map[string]string{"CHILD": "< 8y", "INFANT": "< 1y", "STILLBORN": "0y"}

// transformEnumerations rewrites every enumerated payload in record.Tags into
// the target version's vocabulary (issue #704). To 7.0 the values are
// uppercased (NAME.TYPE aka -> AKA, PEDI birth -> BIRTH, STAT challenged ->
// CHALLENGED, MEDI photo -> PHOTO); a value an open 7.0 set does not list
// becomes OTHER with the value as its PHRASE; ASSO.RELA
// becomes ROLE (an exact role name as is, a role's word, such as neighbor, as
// that role with the word as PHRASE, anything else as OTHER + PHRASE); a 5.x
// AGE keyword (CHILD, INFANT, STILLBORN) becomes the age the 7.0 migration
// notes give it with the keyword as PHRASE; and a 5.x age is written in 7.0
// age syntax (<8 -> "< 8y", 11m99y -> "99y 11m").
//
// To 5.5/5.5.1 the inverse applies: values are written in the 5.x spelling,
// OTHER + PHRASE in a payload that admits free text becomes the phrase,
// ASSO.ROLE becomes RELA (its PHRASE, or the role's word), a SOUR.EVEN.ROLE
// 5.x does not define becomes a parenthesised descriptor, an AGE whose PHRASE
// is a 5.x keyword becomes that keyword, and a 7.0 age loses the space after
// its bound and has its weeks counted as days. A PHRASE consumed this way is
// removed; one that is not is left for dropV7OnlyTags to report.
//
// Each rewrite is a Normalized note. A value with no equivalent in the target
// (SEX X, PEDI OTHER) is left unchanged, reported as a Preserved note and,
// once per conversion, as DataLoss naming the affected records, so
// StrictDataLoss rejects it. A value its source version does not define
// either (an AGE that does not parse, an unknown value in a closed set) is
// left unchanged and reported as a Preserved note only: the conversion loses
// nothing that was valid. RESN is left to transformRestrictions.
//
// Only record.Tags is rewritten; a record without Tags is written from its
// entity and is left alone. Nor is anything beneath an extension tag, whose
// substructures the specification does not define.
func transformEnumerations(doc *gedcom.Document, target gedcom.Version, report *gedcom.ConversionReport) {
	t := &enumTranslator{target: target, toV7: target == gedcom.Version70, report: report}
	for _, record := range doc.Records {
		if record == nil || len(record.Tags) == 0 {
			continue
		}
		record.Tags = t.translateTags(record.Tags, string(record.Type), record.XRef)
	}
	if t.count > 0 {
		report.AddTransformation(gedcom.Transformation{
			Type:        "ENUMERATION_CONVERTED",
			Description: "Converted enumerated values to the target GEDCOM version's vocabulary",
			Count:       t.count,
		})
	}
	if len(t.unmapped) > 0 {
		report.AddDataLoss(gedcom.DataLossItem{
			Feature:         "Enumerated values",
			Reason:          "No GEDCOM " + target.String() + " equivalent; value left unchanged and not valid " + target.String(),
			AffectedRecords: t.unmapped,
		})
	}
}

// enumTranslator carries one transformEnumerations pass.
type enumTranslator struct {
	target gedcom.Version
	toV7   bool
	report *gedcom.ConversionReport
	count  int

	// unmapped lists, once each in visit order, the records holding a value
	// with no target equivalent.
	unmapped     []string
	unmappedSeen map[string]bool
}

// enumResult is the outcome of translating one enumerated payload.
type enumResult struct {
	tag, value string
	// phrase is the PHRASE to add beneath the tag (to 7.0); usePhrase says the
	// existing PHRASE was folded into value and is removed (to 5.x).
	phrase    string
	usePhrase bool
	// unmapped says the value has no target equivalent and is left as is;
	// invalid says it is not a valid source-version value either.
	unmapped, invalid bool
}

// translateTags returns tags with every enumerated payload translated.
func (t *enumTranslator) translateTags(tags []*gedcom.Tag, recordType, xref string) []*gedcom.Tag {
	out := make([]*gedcom.Tag, 0, len(tags))
	var path []string // tag names by level, starting at level 1
	skipBelow := -1   // level of a removed PHRASE whose block is being skipped
	removed := map[*gedcom.Tag]bool{}
	for i, tag := range tags {
		if tag == nil {
			out = append(out, tag)
			continue
		}
		if skipBelow >= 0 && tag.Level > skipBelow {
			continue
		}
		skipBelow = -1
		if removed[tag] {
			skipBelow = tag.Level
			continue
		}
		out = append(out, tag)
		if tag.Level < 1 {
			continue
		}
		path = append(path[:min(tag.Level-1, len(path))], tag.Tag)
		kind := enumKindAt(recordType, path, t.toV7)
		if kind == enumNone || tag.Value == "" {
			continue
		}

		added, consumed := t.translate(tags, i, kind, path, recordType, xref)
		if added != nil {
			out = append(out, added)
		}
		if consumed != nil {
			removed[consumed] = true
		}
	}
	return out
}

// translate translates the enumerated payload of kind at tags[i], whose path
// is path, renaming the tag in place and recording the rewrite. It returns the
// PHRASE to insert after the tag (to 7.0) and the existing PHRASE it folded
// into the value, which the caller removes (to 5.x).
func (t *enumTranslator) translate(tags []*gedcom.Tag, i int, kind enumKind, path []string, recordType, xref string) (added, consumed *gedcom.Tag) {
	tag := tags[i]
	phraseTag := subordinatePhrase(tags, i)
	var phrase string
	if phraseTag != nil {
		phrase = phraseTag.Value
	}
	var r enumResult
	if t.toV7 {
		r = upgradeEnum(kind, tag.Tag, tag.Value)
	} else {
		r = downgradeEnum(kind, tag.Tag, tag.Value, phrase)
	}

	if r.unmapped {
		t.reportUnmapped(BuildNestedPath(recordType, xref, path...), tag.Tag, tag.Value, xref, r.invalid)
		return nil, nil
	}
	if r.tag == tag.Tag && r.value == tag.Value && (r.phrase == "" || phraseTag != nil) && !r.usePhrase {
		return nil, nil
	}
	original := tag.Value
	if r.usePhrase {
		original += " + PHRASE " + phrase
		consumed = phraseTag
	}
	addPhrase := r.phrase != "" && phraseTag == nil
	result := r.value
	if addPhrase {
		result += " + PHRASE " + r.phrase
	}
	if r.tag != tag.Tag {
		original = tag.Tag + " " + original
		result = r.tag + " " + result
	}
	tag.Tag, tag.Value = r.tag, r.value
	path[len(path)-1] = r.tag
	if addPhrase {
		added = &gedcom.Tag{Level: tag.Level + 1, Tag: "PHRASE", Value: r.phrase}
	}
	t.count++
	t.report.AddNormalized(gedcom.ConversionNote{
		Path:     BuildNestedPath(recordType, xref, path...),
		Original: original,
		Result:   result,
		Reason:   "Translated to the GEDCOM " + t.target.String() + " " + r.tag + " vocabulary",
	})
	return added, consumed
}

// reportUnmapped reports a value with no equivalent in the target version: as
// data loss too, unless it was not a valid source-version value either.
func (t *enumTranslator) reportUnmapped(where, tag, value, xref string, invalid bool) {
	reason := "No GEDCOM " + t.target.String() + " equivalent for this " + tag + " value; left unchanged"
	if invalid {
		reason = "Not a valid " + tag + " value in the source GEDCOM version either; left unchanged"
	}
	t.report.AddPreserved(gedcom.ConversionNote{
		Path:     where,
		Original: value,
		Result:   value,
		Reason:   reason,
	})
	if invalid {
		return
	}
	if t.unmappedSeen == nil {
		t.unmappedSeen = make(map[string]bool)
	}
	if !t.unmappedSeen[xref] {
		t.unmappedSeen[xref] = true
		t.unmapped = append(t.unmapped, xref)
	}
}

// subordinatePhrase returns the PHRASE directly beneath tags[i], or nil.
func subordinatePhrase(tags []*gedcom.Tag, i int) *gedcom.Tag {
	level := tags[i].Level
	for _, sub := range tags[i+1:] {
		if sub == nil {
			continue
		}
		if sub.Level <= level {
			break
		}
		if sub.Level == level+1 && sub.Tag == "PHRASE" {
			return sub
		}
	}
	return nil
}

// enumKindAt names the enumerated payload at path (tag names by level, the
// last being the tag itself) in a record of recordType, for a conversion to
// 7.0 when toV7 is set and to 5.x otherwise. Only the association tag differs
// by direction: 5.x RELA, 7.0 ROLE. Nothing in an extension record or beneath
// an extension tag is one: the specification does not define their
// substructures.
//
//nolint:gocyclo // One case per enumerated structure; splitting it would scatter the table
func enumKindAt(recordType string, path []string, toV7 bool) enumKind {
	if strings.HasPrefix(recordType, "_") {
		return enumNone
	}
	for _, p := range path {
		if strings.HasPrefix(p, "_") {
			return enumNone
		}
	}
	tag := path[len(path)-1]
	parent, grandparent := recordType, ""
	if len(path) >= 2 {
		parent = path[len(path)-2]
		grandparent = recordType
	}
	if len(path) >= 3 {
		grandparent = path[len(path)-3]
	}
	indi := recordType == string(gedcom.RecordTypeIndividual)
	switch tag {
	case "SEX":
		if indi && len(path) == 1 {
			return enumSex
		}
	case "TYPE":
		if indi && len(path) == 2 && parent == "NAME" {
			return enumNameType
		}
		if toV7 && isRecordMediaForm(recordType, path) {
			return enumRecordMedia
		}
	case "PEDI":
		if parent == "FAMC" {
			return enumPedigree
		}
	case "STAT":
		if parent == "FAMC" {
			return enumChildStatus
		}
		if ordinanceTags[parent] {
			return enumOrdinanceStatus
		}
	// MEDI and AGE are legal under many parents, so outside extensions they
	// are matched by tag alone.
	case "MEDI":
		if !toV7 && isRecordMediaForm(recordType, path) {
			return enumRecordMedia
		}
		return enumMedia
	case "ADOP":
		if parent == "FAMC" {
			return enumAdoptedBy
		}
	case "RELA":
		if toV7 && parent == "ASSO" {
			return enumAssociation
		}
	case "ROLE":
		if !toV7 && parent == "ASSO" {
			return enumAssociation
		}
		if parent == "EVEN" && grandparent == "SOUR" {
			return enumEventRole
		}
	case "AGE":
		return enumAge
	}
	return enumNone
}

// isRecordMediaForm reports whether path is the media type of a multimedia
// record's file: FILE.FORM.TYPE in 5.5.1, FILE.FORM.MEDI in 7.0. (A 5.5.1
// multimedia link writes it MEDI, as 7.0 does.)
func isRecordMediaForm(recordType string, path []string) bool {
	return recordType == string(gedcom.RecordTypeMedia) && len(path) == 3 &&
		path[0] == "FILE" && path[1] == "FORM"
}

// upgradeEnum translates a 5.x payload of kind into 7.0.
func upgradeEnum(kind enumKind, tag, value string) enumResult {
	if kind == enumRecordMedia {
		r := upgradeEnum(enumMedia, tag, value)
		r.tag = "MEDI"
		return r
	}
	r := enumResult{tag: tag, value: value}
	trimmed := strings.TrimSpace(value)
	switch kind {
	case enumAssociation:
		r.tag = "ROLE"
		r.value, r.phrase = upgradeRole(trimmed)
	case enumEventRole:
		inner := trimmed
		if strings.HasPrefix(inner, "(") && strings.HasSuffix(inner, ")") {
			inner = strings.TrimSpace(inner[1 : len(inner)-1])
		}
		if inner == "" {
			r.unmapped = true
			break
		}
		r.value, r.phrase = upgradeRole(inner)
	case enumAge:
		r.value, r.phrase, r.unmapped = upgradeAge(trimmed)
	default:
		if v, ok := v7Enums[kind][strings.ToUpper(trimmed)]; ok {
			r.value = v
		} else if _, open := v7Enums[kind]["OTHER"]; open {
			r.value, r.phrase = "OTHER", trimmed
		} else {
			r.unmapped = true
		}
	}
	if r.unmapped {
		// Every 5.x value 5.x defines has a 7.0 equivalent, so one without is
		// not a valid 5.x value either.
		return enumResult{tag: tag, value: value, unmapped: true, invalid: true}
	}
	return r
}

// upgradeRole maps a 5.x role description to a 7.0 ROLE and PHRASE: a role
// name as is, a role's word as that role with the word kept as PHRASE (the
// gedcom7code rela_1 conversion), and anything else as OTHER + PHRASE.
func upgradeRole(text string) (role, phrase string) {
	upper := strings.ToUpper(text)
	if _, ok := v7Roles[upper]; ok && upper != "OTHER" {
		return upper, ""
	}
	if role, ok := v7RoleByWord[upper]; ok && role != "OTHER" {
		return role, text
	}
	return "OTHER", text
}

// downgradeEnum translates a 7.0 payload of kind, with the PHRASE beneath it
// (empty when there is none), into 5.x.
func downgradeEnum(kind enumKind, tag, value, phrase string) enumResult {
	if kind == enumRecordMedia {
		r := downgradeEnum(enumMedia, tag, value, phrase)
		r.tag = "TYPE"
		return r
	}
	r := enumResult{tag: tag, value: value}
	upper := strings.ToUpper(strings.TrimSpace(value))
	switch kind {
	case enumAssociation:
		r.tag = "RELA"
		r.value, r.usePhrase = downgradeRole(upper, value, phrase)
	case enumEventRole:
		if phrase == "" && v5EventRoles[upper] {
			r.value = upper
			break
		}
		var desc string
		desc, r.usePhrase = downgradeRole(upper, value, phrase)
		r.value = "(" + desc + ")"
	case enumAge:
		if _, ok := v5AgeKeywords[strings.ToUpper(strings.TrimSpace(phrase))]; ok {
			r.value, r.usePhrase = strings.TrimSpace(phrase), true
			break
		}
		r.value, r.unmapped = downgradeAge(value)
		r.invalid = r.unmapped
	default:
		if v, ok := v5Enums[kind][upper]; ok {
			r.value = v
		} else if !v5OpenEnums[kind] {
			r.unmapped, r.invalid = true, !v7Values[kind][upper]
		} else if upper == "OTHER" && phrase != "" {
			r.value, r.usePhrase = phrase, true
		}
	}
	if r.unmapped {
		return enumResult{tag: tag, value: value, unmapped: true, invalid: r.invalid}
	}
	return r
}

// downgradeRole describes a 7.0 ROLE for 5.x: its PHRASE when it has one,
// otherwise the role's word, or the value itself for a role 7.0 does not
// define (an extension). usePhrase reports that the PHRASE was used.
func downgradeRole(upper, value, phrase string) (desc string, usePhrase bool) {
	if phrase != "" {
		return phrase, true
	}
	if word, ok := v7Roles[upper]; ok {
		return word, false
	}
	return value, false
}

// parseAge reads an age leniently: an optional < or > bound, then terms of
// a count and a unit (y, m, w, d, in either case) in any order, with any
// spacing, or a bare count read as years. It reports !ok for anything else,
// or a unit given twice.
func parseAge(s string) (bound byte, terms map[byte]int, ok bool) {
	s = strings.Join(strings.Fields(s), "")
	if s != "" && (s[0] == '<' || s[0] == '>') {
		bound, s = s[0], s[1:]
	}
	if s == "" {
		return 0, nil, false
	}
	terms = map[byte]int{}
	for s != "" {
		j := 0
		for j < len(s) && s[j] >= '0' && s[j] <= '9' {
			j++
		}
		if j == 0 {
			return 0, nil, false
		}
		n, err := strconv.Atoi(s[:j])
		if err != nil {
			return 0, nil, false
		}
		if j == len(s) {
			if len(terms) > 0 {
				return 0, nil, false
			}
			terms['y'] = n
			break
		}
		unit := s[j] | 0x20 // ASCII lowercase
		if !strings.ContainsRune("ymwd", rune(unit)) {
			return 0, nil, false
		}
		if _, dup := terms[unit]; dup {
			return 0, nil, false
		}
		terms[unit] = n
		s = s[j+1:]
	}
	return bound, terms, true
}

// formatAge writes terms in y, m, w, d order, after bound and boundSep.
func formatAge(bound byte, boundSep string, terms map[byte]int) string {
	var parts []string
	for _, unit := range []byte("ymwd") {
		if n, ok := terms[unit]; ok {
			parts = append(parts, strconv.Itoa(n)+string(unit))
		}
	}
	age := strings.Join(parts, " ")
	if bound != 0 {
		age = string(bound) + boundSep + age
	}
	return age
}

// upgradeAge translates a 5.x AGE payload into 7.0 age syntax, moving a
// keyword into PHRASE. unmapped reports a payload that does not parse.
func upgradeAge(value string) (age, phrase string, unmapped bool) {
	if v7, ok := v5AgeKeywords[strings.ToUpper(value)]; ok {
		return v7, value, false
	}
	bound, terms, ok := parseAge(value)
	if !ok {
		return "", "", true
	}
	return formatAge(bound, " ", terms), "", false
}

// downgradeAge translates a 7.0 AGE payload into 5.x syntax: no space after
// the bound, and weeks, which 5.x lacks, counted as days. unmapped reports a
// payload that does not parse.
func downgradeAge(value string) (age string, unmapped bool) {
	bound, terms, ok := parseAge(value)
	if !ok {
		return "", true
	}
	if w, ok := terms['w']; ok {
		terms['d'] += 7 * w
		delete(terms, 'w')
	}
	return formatAge(bound, "", terms), false
}
