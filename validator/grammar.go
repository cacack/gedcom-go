// grammar.go provides the grammar conformance check, the opt-in check
// ([ValidatorConfig.CheckGrammar]) that holds each structure in a document
// against the structure tables of the GEDCOM version it declares: which
// substructures are permitted where and how many times, which are required,
// which enumeration values and payload datatypes each accepts, and which tags
// belong to another version.
//
// The tables live in grammar_tables.go, generated from the specification
// inventories under testdata/spec by internal/cmd/gengrammar. Regenerate them
// with `go generate ./validator` after updating an inventory; a test fails
// while they are out of date.

//go:generate go run ../internal/cmd/gengrammar -spec ../testdata/spec -out grammar_tables.go

package validator

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// payloadKind is the datatype a structure's payload must have.
type payloadKind int

const (
	// payloadText is a payload the check does not parse: free text, and the
	// datatypes other checks own (dates are CodeInvalidDate's).
	payloadText payloadKind = iota
	// payloadNone is a structure that defines no payload.
	payloadNone
	// payloadYNull is "Y" or empty.
	payloadYNull
	// payloadPointer is a pointer to a record.
	payloadPointer
	// payloadPointerOrNull is a pointer to a record, or empty.
	payloadPointerOrNull
	// payloadInteger is a non-negative integer.
	payloadInteger
	// payloadLanguage is a BCP 47 language tag.
	payloadLanguage
	// payloadEnum is one value of the structure's enumeration.
	payloadEnum
	// payloadEnumList is a comma-separated list of enumeration values.
	payloadEnumList
	// payloadDateExact is a day, month and year, as CHAN.DATE carries.
	payloadDateExact
	// payloadTime is a time of day.
	payloadTime
	// payloadAge is an age at an event.
	payloadAge
)

// enumOpenness says which values beyond its list an enumeration accepts.
type enumOpenness int

const (
	// The zero value accepts only the listed values.
	_ enumOpenness = iota
	// enumOpen accepts any value ("<user defined>" in 5.5.x).
	enumOpen
	// enumOpenParenthesized also accepts a parenthesized descriptor, as
	// 5.5.x ROLE does: "(<ROLE_DESCRIPTOR>)".
	enumOpenParenthesized
)

// childDef is one substructure a structure permits, with its cardinality.
type childDef struct {
	id       string
	min, max int // max -1 is unbounded
}

// structureDef is one structure type of a version's grammar.
type structureDef struct {
	payload  payloadKind
	enum     []string
	open     enumOpenness
	children map[string][]childDef // by tag; more than one is an alternative
}

// grammarTable is one GEDCOM version's grammar.
type grammarTable struct {
	version string
	// foldCase compares enumeration values, months and age units without
	// regard to case, as 5.5.x readers conventionally do. 7.0 enumeration
	// values are tags, which are case-sensitive.
	foldCase   bool
	records    map[string]string // level-0 tag -> structure id
	structures map[string]*structureDef
}

// grammarFor returns the grammar for a version, or nil for one the tables do
// not cover.
func grammarFor(v gedcom.Version) *grammarTable {
	switch v {
	case gedcom.Version55:
		return grammar55
	case gedcom.Version551:
		return grammar551
	case gedcom.Version70:
		return grammar70
	}
	return nil
}

// voidXRef is the GEDCOM 7.0 null pointer.
const voidXRef = "@VOID@"

var (
	grammarTagsOnce sync.Once
	// grammarTags maps each grammar to the set of tags it defines anywhere.
	grammarTags map[*grammarTable]map[string]bool
)

// definedTags returns the tags g defines anywhere, at any level.
func definedTags(g *grammarTable) map[string]bool {
	grammarTagsOnce.Do(func() {
		grammarTags = map[*grammarTable]map[string]bool{}
		for _, t := range []*grammarTable{grammar55, grammar551, grammar70} {
			tags := map[string]bool{}
			for tag := range t.records {
				tags[tag] = true
			}
			for _, s := range t.structures {
				for tag := range s.children {
					tags[tag] = true
				}
			}
			grammarTags[t] = tags
		}
	})
	return grammarTags[g]
}

// deprecated70 lists the 7.0 structure types the specification deprecates,
// with what it says to use instead.
var deprecated70 = map[string]string{
	"ADR1":           "ADR1 is deprecated; the ADDR payload carries the full address",
	"ADR2":           "ADR2 is deprecated; the ADDR payload carries the full address",
	"ADR3":           "ADR3 is deprecated; the ADDR payload carries the full address",
	"HEAD-SOUR-DATA": "HEAD.SOUR.DATA is deprecated; use HEAD.SOUR.NAME",
}

// deprecatedEnum70 lists, per 7.0 structure type, the enumeration values the
// specification marks deprecated.
var deprecatedEnum70 = map[string]map[string]string{
	"ord-STAT": {
		"COMPLETED": "use DATE BEF date",
		"EXCLUDED":  "it was defined for TempleReady, which is no longer in use",
		"INFANT":    "use CHILD",
		"PRE_1970":  "use DATE BEF 1970",
		"SUBMITTED": "it was defined for TempleReady, which is no longer in use",
		"UNCLEARED": "it was defined for TempleReady, which is no longer in use",
	},
}

var (
	// xref70 is a 7.0 cross-reference identifier: uppercase letters,
	// digits and underscores.
	xref70 = regexp.MustCompile(`^@[A-Z0-9_]+@$`)
	// language70 is the shape of a BCP 47 language tag: a 2-8 letter
	// primary subtag and 1-8 character subtags.
	language70 = regexp.MustCompile(`^[A-Za-z]{2,8}(-[A-Za-z0-9]{1,8})*$`)
	integer    = regexp.MustCompile(`^\d+$`)
	// dateExact70 is DateExact: day, Gregorian month and year.
	dateExact70 = regexp.MustCompile(`^(\d+) (JAN|FEB|MAR|APR|MAY|JUN|JUL|AUG|SEP|OCT|NOV|DEC) \d+$`)
	// dateExact55 is DATE_EXACT, whose year may be a dual year (1699/00).
	dateExact55 = regexp.MustCompile(`(?i)^(\d{1,2}) (JAN|FEB|MAR|APR|MAY|JUN|JUL|AUG|SEP|OCT|NOV|DEC) \d{3,4}(/\d{2})?$`)
	time70      = regexp.MustCompile(`^([01]?\d|2[0-3]):[0-5]\d(:[0-5]\d(\.\d+)?)?Z?$`)
	time55      = regexp.MustCompile(`^([01]?\d|2[0-3]):[0-5]\d(:[0-5]\d(\.\d+)?)?$`)
	// age70 is Age: an optional bound and space, then years, months, weeks
	// and days in that order, each optional but at least one present.
	age70 = regexp.MustCompile(`^([<>] )?(\d+y( \d+m)?( \d+w)?( \d+d)?|\d+m( \d+w)?( \d+d)?|\d+w( \d+d)?|\d+d)$`)
	// age55 is AGE_AT_EVENT: a bound written against the number, then
	// years, months and days in that order, or a keyword.
	age55 = regexp.MustCompile(`(?i)^([<>]?(\d+y( \d+m)?( \d+d)?|\d+m( \d+d)?|\d+d)|CHILD|INFANT|STILLBORN)$`)
)

// grammarChecker walks one document against one grammar.
type grammarChecker struct {
	g      *grammarTable
	v7     bool
	issues []Issue
}

// validateGrammar runs the grammar conformance check: every structure in the
// header and records, read from their raw Tags, against the tables for the
// version the header declares (see grammarVersion). A document whose version
// the tables do not cover, such as 7.1, is not checked. Nor are the substructures of a header
// with no Tags, or of a record with no Tags and no LineNumber: both were
// built in memory, and the encoder writes them from their typed fields.
//
// Extension structures (tags starting with "_") and their substructures are
// skipped. CONT is permitted under any structure. In 5.5 and 5.5.1 so is
// CONC, which the printed grammar lists only where it expects long values but
// which writers split any line with; in 7.0, which removed CONC, it is
// reported as a version-foreign structure.
func validateGrammar(doc *gedcom.Document) []Issue {
	if doc == nil || doc.Header == nil {
		return nil
	}
	g := grammarFor(grammarVersion(doc.Header))
	if g == nil {
		return nil
	}
	c := &grammarChecker{g: g, v7: g == grammar70}
	if len(doc.Header.Tags) > 0 {
		c.checkStructure("", 0, g.records["HEAD"], "HEAD", buildGrammarTree(doc.Header.Tags))
	}
	for _, record := range doc.Records {
		if record != nil {
			c.checkRecord(record)
		}
	}
	return c.issues
}

// grammarVersion returns the version whose grammar applies: the one the
// header declares (DeclaredVersion). A VERS the decoder does not recognize,
// such as 7.1, selects none, so a 7.1 file is not held to 7.0's tables. A
// header with no GEDC.VERS at all -- one built in memory, or a file missing
// the required line -- has no declaration to contradict, so the decoder's
// Version is used.
func grammarVersion(h *gedcom.Header) gedcom.Version {
	if h.DeclaredVersion != "" {
		return h.DeclaredVersion
	}
	inGEDC := false
	for _, tag := range h.Tags {
		if tag == nil {
			continue
		}
		switch {
		case tag.Level == 1:
			inGEDC = tag.Tag == "GEDC"
		case inGEDC && tag.Level == 2 && tag.Tag == "VERS" && tag.Value != "":
			return ""
		}
	}
	return h.Version
}

// grammarNode is one line of a record or header with its substructures.
type grammarNode struct {
	tag  *gedcom.Tag
	kids []*grammarNode
}

// buildGrammarTree nests a flat, level-ordered tag list. A tag's parent is
// the nearest earlier tag with a lower level.
func buildGrammarTree(tags []*gedcom.Tag) []*grammarNode {
	var roots, stack []*grammarNode
	for _, tag := range tags {
		if tag == nil {
			continue
		}
		n := &grammarNode{tag: tag}
		for len(stack) > 0 && stack[len(stack)-1].tag.Level >= tag.Level {
			stack = stack[:len(stack)-1]
		}
		if len(stack) == 0 {
			roots = append(roots, n)
		} else {
			parent := stack[len(stack)-1]
			parent.kids = append(parent.kids, n)
		}
		stack = append(stack, n)
	}
	return roots
}

func isExtensionTag(tag string) bool {
	return strings.HasPrefix(tag, "_")
}

func (c *grammarChecker) checkRecord(record *gedcom.Record) {
	tag := string(record.Type)
	if tag == "" || isExtensionTag(tag) || tag == "HEAD" || tag == "TRLR" {
		return
	}
	id, ok := c.g.records[tag]
	if !ok {
		c.unexpected(record.XRef, record.LineNumber, tag, "")
		return
	}
	c.checkRecordXRef(record)
	if len(record.Tags) > 0 || record.LineNumber > 0 {
		c.checkStructure(record.XRef, record.LineNumber, id, tag, buildGrammarTree(record.Tags))
	}
}

// checkRecordXRef reports, in 5.5.x, a record with no cross-reference
// identifier, which their grammar requires, and in 7.0 one outside the
// identifier alphabet or that is the reserved null pointer @VOID@. A 7.0
// record may omit its identifier (gedcom.io's xref.ged has one that does).
func (c *grammarChecker) checkRecordXRef(record *gedcom.Record) {
	var msg string
	switch {
	case record.XRef == "" && c.v7:
		return
	case record.XRef == "":
		msg = fmt.Sprintf("%s record has no cross-reference identifier", record.Type)
	case c.v7 && record.XRef == voidXRef:
		msg = fmt.Sprintf("%s record uses the reserved null pointer @VOID@ as its identifier", record.Type)
	case c.v7 && !xref70.MatchString(record.XRef):
		msg = fmt.Sprintf("%s record identifier %s uses characters outside A-Z, 0-9 and _", record.Type, record.XRef)
	default:
		return
	}
	c.issues = append(c.issues, NewIssue(SeverityError, CodeInvalidRecordXRef, msg, record.XRef).
		WithLineNumber(record.LineNumber).
		WithDetail("tag", string(record.Type)).
		WithDetail("version", c.g.version))
}

// checkStructure checks the substructures of one structure of type id.
func (c *grammarChecker) checkStructure(xref string, line int, id, tag string, kids []*grammarNode) {
	def := c.g.structures[id]
	if def == nil {
		return
	}
	perID := map[string]int{}
	perTag := map[string]int{}
	for _, kid := range kids {
		t := kid.tag
		if isExtensionTag(t.Tag) {
			continue
		}
		alts := def.children[t.Tag]
		if len(alts) == 0 {
			if !c.continuation(t.Tag) {
				c.unexpected(xref, t.LineNumber, t.Tag, tag)
			}
			continue
		}
		child := c.pick(alts, t.Value)
		perID[child.id]++
		perTag[t.Tag]++
		if child.max >= 0 && perID[child.id] > child.max {
			c.issues = append(c.issues, NewIssue(SeverityError, CodeCardinalityExceeded,
				fmt.Sprintf("%s appears more than %d time(s) under %s", t.Tag, child.max, tag), xref).
				WithLineNumber(t.LineNumber).
				WithDetail("tag", t.Tag).
				WithDetail("superstructure", tag).
				WithDetail("cardinality", cardinality(child)).
				WithDetail("version", c.g.version))
		}
		c.checkPayload(xref, t, child.id)
		c.checkDeprecated(xref, t, child.id, kid.kids)
		c.checkStructure(xref, t.LineNumber, child.id, t.Tag, kid.kids)
	}
	c.checkRequired(xref, line, id, tag, def, perTag)
}

// continuation reports whether tag is a continuation line accepted under any
// structure: CONT, and in 5.5.x CONC. Where the grammar lists one (5.5 BLOB
// requires CONT) it is counted like any other substructure.
func (c *grammarChecker) continuation(tag string) bool {
	return tag == "CONT" || (tag == "CONC" && !c.v7)
}

// checkRequired reports each required substructure def is missing. A tag
// with alternatives (5.5.x NOTE as a pointer or as text) is satisfied by any
// of them. HEAD.SUBM is left to CodeMissingSUBM, which already reports it.
func (c *grammarChecker) checkRequired(xref string, line int, id, tag string, def *structureDef, perTag map[string]int) {
	var missing []string
	for childTag, alts := range def.children {
		need := 0
		for _, a := range alts {
			need = max(need, a.min)
		}
		if need > perTag[childTag] && (id != c.g.records["HEAD"] || childTag != "SUBM") {
			missing = append(missing, childTag)
		}
	}
	sort.Strings(missing)
	for _, childTag := range missing {
		c.issues = append(c.issues, NewIssue(SeverityError, CodeMissingRequiredSubstructure,
			fmt.Sprintf("%s requires a %s substructure", tag, childTag), xref).
			WithLineNumber(line).
			WithDetail("tag", childTag).
			WithDetail("superstructure", tag).
			WithDetail("version", c.g.version))
	}
}

func cardinality(child childDef) string {
	if child.max < 0 {
		return fmt.Sprintf("{%d:M}", child.min)
	}
	return fmt.Sprintf("{%d:%d}", child.min, child.max)
}

// pick returns the alternative a line is: the pointer one for a
// pointer-shaped value, otherwise the first that is not a pointer.
func (c *grammarChecker) pick(alts []childDef, value string) childDef {
	if len(alts) == 1 {
		return alts[0]
	}
	pointer := c.isPointer(value)
	for _, a := range alts {
		if def := c.g.structures[a.id]; def != nil {
			isPtr := def.payload == payloadPointer || def.payload == payloadPointerOrNull
			if isPtr == pointer {
				return a
			}
		}
	}
	return alts[0]
}

// unexpected reports a standard tag the grammar does not permit where it
// stands: as version-foreign when only another version defines it, and as
// unexpected otherwise. superTag is empty for a level-0 record.
func (c *grammarChecker) unexpected(xref string, line int, tag, superTag string) {
	if !definedTags(c.g)[tag] {
		for _, other := range []*grammarTable{grammar55, grammar551, grammar70} {
			if other != c.g && definedTags(other)[tag] {
				c.issues = append(c.issues, NewIssue(SeverityError, CodeVersionForeignStructure,
					fmt.Sprintf("%s is a GEDCOM %s structure, not defined by GEDCOM %s", tag, other.version, c.g.version), xref).
					WithLineNumber(line).
					WithDetail("tag", tag).
					WithDetail("defined_in", other.version).
					WithDetail("version", c.g.version))
				return
			}
		}
	}
	where := "at level 0"
	if superTag != "" {
		where = "under " + superTag
	}
	issue := NewIssue(SeverityError, CodeUnexpectedSubstructure,
		fmt.Sprintf("%s is not permitted %s in GEDCOM %s", tag, where, c.g.version), xref).
		WithLineNumber(line).
		WithDetail("tag", tag).
		WithDetail("version", c.g.version)
	if superTag != "" {
		issue = issue.WithDetail("superstructure", superTag)
	}
	c.issues = append(c.issues, issue)
}

// isPointer reports whether value has the shape of a record pointer in the
// grammar's version. A 5.5.x "@#D...@" calendar escape is not one.
func (c *grammarChecker) isPointer(value string) bool {
	if c.v7 {
		return value == voidXRef || xref70.MatchString(value)
	}
	return len(value) > 2 && value[0] == '@' && value[len(value)-1] == '@' && value[1] != '#' &&
		!strings.Contains(value[1:len(value)-1], "@")
}

// checkPayload checks a line's payload against its structure's datatype.
func (c *grammarChecker) checkPayload(xref string, t *gedcom.Tag, id string) {
	def := c.g.structures[id]
	if def == nil {
		return
	}
	v := t.Value
	switch def.payload {
	case payloadEnum:
		if !c.enumMember(def, v) {
			c.invalidEnum(xref, t, def, v)
		}
		return
	case payloadEnumList:
		for _, item := range strings.Split(v, ",") {
			if item = strings.TrimSpace(item); !c.enumMember(def, item) {
				c.invalidEnum(xref, t, def, item)
			}
		}
		return
	}
	datatype, ok := c.payloadValid(def.payload, v)
	if ok {
		return
	}
	c.issues = append(c.issues, NewIssue(SeverityError, CodeInvalidPayload,
		fmt.Sprintf("%s payload %q is not a valid %s", t.Tag, v, datatype), xref).
		WithLineNumber(t.LineNumber).
		WithDetail("tag", t.Tag).
		WithDetail("value", v).
		WithDetail("datatype", datatype).
		WithDetail("version", c.g.version))
}

// payloadValid reports whether v is a payload of kind, naming the datatype
// for the message. A kind the check does not parse is always valid.
func (c *grammarChecker) payloadValid(kind payloadKind, v string) (datatype string, ok bool) {
	switch kind {
	case payloadYNull:
		return "Y or empty", v == "" || v == "Y" || (c.g.foldCase && v == "y")
	case payloadPointer:
		return "pointer", c.isPointer(v)
	case payloadPointerOrNull:
		return "pointer", v == "" || c.isPointer(v)
	case payloadInteger:
		return "integer", integer.MatchString(v)
	case payloadLanguage:
		return "language tag", language70.MatchString(v)
	case payloadDateExact:
		return "exact date", c.dateExact(v)
	case payloadTime:
		return "time", c.byVersion(time70, time55).MatchString(v)
	case payloadAge:
		return "age", v == "" || c.byVersion(age70, age55).MatchString(v)
	}
	return "", true
}

// byVersion returns the 7.0 or the 5.5.x form of a pattern.
func (c *grammarChecker) byVersion(v7, v55 *regexp.Regexp) *regexp.Regexp {
	if c.v7 {
		return v7
	}
	return v55
}

func (c *grammarChecker) dateExact(v string) bool {
	m := c.byVersion(dateExact70, dateExact55).FindStringSubmatch(v)
	if m == nil {
		return false
	}
	day, err := strconv.Atoi(m[1])
	return err == nil && day >= 1 && day <= 31
}

// enumMember reports whether v is a value def's enumeration accepts. In 7.0
// an extension value (one starting with "_") is always accepted.
func (c *grammarChecker) enumMember(def *structureDef, v string) bool {
	switch {
	case def.open == enumOpen:
		return true
	case def.open == enumOpenParenthesized && len(v) > 2 && v[0] == '(' && v[len(v)-1] == ')':
		return true
	case c.v7 && isExtensionTag(v):
		return true
	}
	for _, allowed := range def.enum {
		if v == allowed || (c.g.foldCase && strings.EqualFold(v, allowed)) {
			return true
		}
	}
	return false
}

func (c *grammarChecker) invalidEnum(xref string, t *gedcom.Tag, def *structureDef, v string) {
	c.issues = append(c.issues, NewIssue(SeverityError, CodeInvalidEnumValue,
		fmt.Sprintf("%s value %q is not one of %s", t.Tag, v, strings.Join(def.enum, ", ")), xref).
		WithLineNumber(t.LineNumber).
		WithDetail("tag", t.Tag).
		WithDetail("value", v).
		WithDetail("allowed", strings.Join(def.enum, ",")).
		WithDetail("version", c.g.version))
}

// checkDeprecated reports, in 7.0, a structure, an enumeration value or a
// missing substructure the specification deprecates.
func (c *grammarChecker) checkDeprecated(xref string, t *gedcom.Tag, id string, kids []*grammarNode) {
	if !c.v7 {
		return
	}
	var msg string
	if why, ok := deprecated70[id]; ok {
		msg = why
	} else if why, ok := deprecatedEnum70[id][t.Value]; ok {
		msg = fmt.Sprintf("%s value %s is deprecated; %s", t.Tag, t.Value, why)
	} else if id == "EXID" && !hasChild(kids, "TYPE") {
		msg = "EXID without an EXID.TYPE substructure is deprecated"
	} else {
		return
	}
	c.issues = append(c.issues, NewIssue(SeverityWarning, CodeDeprecatedStructure, msg, xref).
		WithLineNumber(t.LineNumber).
		WithDetail("tag", t.Tag).
		WithDetail("version", c.g.version))
}

func hasChild(kids []*grammarNode, tag string) bool {
	for _, k := range kids {
		if k.tag.Tag == tag {
			return true
		}
	}
	return false
}
