// name_style.go provides the opt-in personal-name style checks
// (ValidatorConfig.StyleChecks).

package validator

import (
	"maps"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// Codes for the personal-name style checks. They report habits in an
// individual's NAME that are valid GEDCOM but usually worth tidying; choosing
// the spelling is left to the user. The checks are opt-in, one code at a time,
// through ValidatorConfig.StyleChecks, and [AllStyleChecks] lists them.
//
// Each check reads the raw level-1 NAME lines of INDI records, splitting the
// value with gedcom.SplitPersonalName, and is reported at most once per NAME
// line, with the individual as RecordXRef, the NAME line as LineNumber and
// the NAME value under Details["value"]. Every check is SeverityWarning, so
// a check turned on reports at the default StrictnessNormal. A record with no
// raw Tags (one built in code) is not checked.
const (
	// CodeNameAllCaps indicates a given name or surname with a word of two
	// or more letters all in capitals, such as "John /SMITH/". A single
	// initial, "J. R. Smith", is not reported.
	CodeNameAllCaps = "NAME_ALL_CAPS"

	// CodeNameTextAfterSurname indicates text after the closing slash of
	// the surname, such as "John /Smith/ Jr.", which belongs in NSFX.
	CodeNameTextAfterSurname = "NAME_TEXT_AFTER_SURNAME"

	// CodeNameSpacing indicates a NAME value with a leading, trailing or
	// doubled space, or a surname with a space just inside its slashes.
	CodeNameSpacing = "NAME_SPACING"

	// CodeNameQuotesOrParens indicates a NAME value holding a double quote
	// or a parenthesis, usually a nickname or an alternative written into
	// the name, such as `John "Jack" /Smith/`.
	CodeNameQuotesOrParens = "NAME_QUOTES_OR_PARENS"

	// CodeNameUncertainMark indicates a NAME value holding a mark of doubt
	// or an alternative: "?", "*", a digit, or the word "or" between two
	// words, such as "John /Smith or Smyth/".
	CodeNameUncertainMark = "NAME_UNCERTAIN_MARK"

	// CodeNamePartPunctuation indicates a raw NSFX or NICK line directly
	// under the NAME whose value starts or ends with punctuation or a space,
	// such as ", Jr" or "Jr.". Details["tags"] lists the tags.
	CodeNamePartPunctuation = "NAME_PART_PUNCTUATION"

	// CodeNameVendorTag indicates an extension tag ("_X") directly under the
	// NAME. Details["tags"] lists the tags.
	CodeNameVendorTag = "NAME_VENDOR_TAG"

	// CodeNamePartMismatch indicates a GIVN or SURN directly under the NAME
	// that is not the given name or surname the NAME value holds, such as
	// "John /Smith/" with GIVN Jon. Details["tags"] lists the disagreeing
	// tags; Details["GIVN"] and Details["SURN"] hold each one's text, its
	// values joined by ", ", and Details["given"] and Details["surname"] the
	// value's text, from gedcom.SplitPersonalName.
	//
	// A piece is compared only when the NAME has at least one line of it.
	// Several lines of one piece (7.0 repeats GIVN and SURN) are read in
	// order as one. The texts are compared word by word, a word being a run
	// of characters other than spaces and commas (5.5.1 separates the names
	// in one piece with commas), with case folded: spacing, commas and case
	// are not a disagreement, and NAME_SPACING and NAME_ALL_CAPS report them.
	// The given name may begin with the NPFX values and end with the NSFX
	// values, and the surname begin with the SPFX values, and a GIVN or SURN
	// may leave those words out, so "Dr. John /Smith/" with NPFX Dr. and GIVN
	// John, and "John /van der Berg/" with SPFX "van der" and SURN Berg, are
	// not reported; a GIVN or SURN holding words the value lacks is. Text in
	// double quotes or parentheses, usually a nickname written into the name
	// such as `John "Jack" /Smith/`, is not compared, and
	// NAME_QUOTES_OR_PARENS reports it. A value with no slashes does not mark
	// its surname, so a GIVN or SURN agrees with it when its words are a run
	// of the value's words, as "John Smith" with GIVN John and SURN Smith
	// (spec-interpretation policy SI-022).
	CodeNamePartMismatch = "NAME_PART_MISMATCH"
)

// StyleCheck describes one style check that ValidatorConfig.StyleChecks can
// turn on.
type StyleCheck struct {
	// Code is the check's issue code, the value to list in StyleChecks.
	Code string

	// Severity is the severity of the issues the check reports.
	Severity Severity

	// Description says what the check reports.
	Description string
}

// styleCheck is one entry of styleChecks: the check as AllStyleChecks
// describes it, and find, which reports whether it applies to a NAME line
// and the details it adds to the issue, such as the offending tags,
// comma-separated, under "tags".
type styleCheck struct {
	StyleCheck
	find func(l *nameLine) (details map[string]string, found bool)
}

// styleChecks is the single list of the style checks, in the order they are
// reported for a NAME line. A check is added here and nowhere else, besides
// its code constant.
var styleChecks = []styleCheck{
	{StyleCheck{CodeNameAllCaps, SeverityWarning, "a given name or surname with a word of two or more letters all in capitals"},
		func(l *nameLine) (map[string]string, bool) {
			return nil, hasCapitalWord(l.given) || hasCapitalWord(l.surname)
		}},
	{StyleCheck{CodeNameTextAfterSurname, SeverityWarning, "text after the closing slash of the surname"},
		func(l *nameLine) (map[string]string, bool) { return nil, l.suffix != "" }},
	{StyleCheck{CodeNameSpacing, SeverityWarning, "a leading, trailing or doubled space, or a space just inside the surname slashes"},
		func(l *nameLine) (map[string]string, bool) {
			return nil, strings.TrimSpace(l.value) != l.value || strings.Contains(l.value, "  ") ||
				(strings.Count(l.value, "/") >= 2 && strings.TrimSpace(l.surname) != l.surname)
		}},
	{StyleCheck{CodeNameQuotesOrParens, SeverityWarning, "a double quote or parenthesis in the NAME value"},
		func(l *nameLine) (map[string]string, bool) { return nil, strings.ContainsAny(l.value, "\"“”()") }},
	{StyleCheck{CodeNameUncertainMark, SeverityWarning, `"?", "*", a digit, or "or" between two alternatives in the NAME value`},
		func(l *nameLine) (map[string]string, bool) {
			return nil, strings.ContainsAny(l.value, "?*0123456789") || hasAlternative(l.value)
		}},
	{StyleCheck{CodeNamePartPunctuation, SeverityWarning, "an NSFX or NICK value starting or ending with punctuation or a space"},
		func(l *nameLine) (map[string]string, bool) {
			return l.partTags(func(t *gedcom.Tag) bool {
				return (t.Tag == "NSFX" || t.Tag == "NICK") && hasEdgePunctuation(t.Value)
			})
		}},
	{StyleCheck{CodeNameVendorTag, SeverityWarning, `an extension ("_") tag directly under NAME`},
		func(l *nameLine) (map[string]string, bool) {
			return l.partTags(func(t *gedcom.Tag) bool { return strings.HasPrefix(t.Tag, "_") })
		}},
	{StyleCheck{CodeNamePartMismatch, SeverityWarning, "a GIVN or SURN that is not the given name or surname in the NAME value"},
		(*nameLine).partMismatch},
}

// AllStyleChecks returns every style check ValidatorConfig.StyleChecks
// accepts, in the order they are reported. The slice is a copy.
func AllStyleChecks() []StyleCheck {
	checks := make([]StyleCheck, len(styleChecks))
	for i, check := range styleChecks {
		checks[i] = check.StyleCheck
	}
	return checks
}

// nameStyleIssues runs the style checks the configuration's StyleChecks
// lists, if any.
func (v *Validator) nameStyleIssues(doc *gedcom.Document) []Issue {
	if v.config == nil || len(v.config.StyleChecks) == 0 {
		return nil
	}
	return validateNameStyle(doc, v.config.StyleChecks)
}

// validateNameStyle runs the style checks whose codes enabled lists over the
// level-1 NAME lines of every individual. Codes it does not know are ignored.
func validateNameStyle(doc *gedcom.Document, enabled []string) []Issue {
	on := make(map[string]bool, len(enabled))
	for _, code := range enabled {
		on[code] = true
	}
	var issues []Issue
	for _, record := range doc.Records {
		if record == nil || record.Type != gedcom.RecordTypeIndividual {
			continue
		}
		for i, tag := range record.Tags {
			if tag == nil || tag.Level != 1 || tag.Tag != "NAME" {
				continue
			}
			line := newNameLine(tag, record.Tags[i+1:])
			for _, check := range styleChecks {
				if !on[check.Code] {
					continue
				}
				if details, found := check.find(&line); found {
					issue := NewIssue(check.Severity, check.Code, "Name style: "+check.Description, record.XRef).
						WithLineNumber(tag.LineNumber).
						WithDetail("value", tag.Value)
					for _, key := range slices.Sorted(maps.Keys(details)) {
						issue = issue.WithDetail(key, details[key])
					}
					issues = append(issues, issue)
				}
			}
		}
	}
	return issues
}

// nameLine is a raw NAME line split for the style checks: its value, the
// value's pieces from gedcom.SplitPersonalName, and the tags directly under it.
type nameLine struct {
	value, given, surname, suffix string
	parts                         []*gedcom.Tag
}

// newNameLine splits the level-1 NAME line name, whose following lines in the
// record are rest (spec-interpretation policy SI-009).
func newNameLine(name *gedcom.Tag, rest []*gedcom.Tag) nameLine {
	line := nameLine{value: name.Value}
	line.given, line.surname, line.suffix = gedcom.SplitPersonalName(name.Value)
	for _, tag := range rest {
		if tag == nil {
			continue
		}
		if tag.Level <= 1 {
			break
		}
		if tag.Level == 2 {
			line.parts = append(line.parts, tag)
		}
	}
	return line
}

// partTags returns the tags of the line's parts that match accepts,
// comma-separated under "tags", and whether there were any.
func (l *nameLine) partTags(match func(*gedcom.Tag) bool) (map[string]string, bool) {
	var tags []string
	for _, part := range l.parts {
		if match(part) {
			tags = append(tags, part.Tag)
		}
	}
	if len(tags) == 0 {
		return nil, false
	}
	return map[string]string{"tags": strings.Join(tags, ",")}, true
}

// partValues returns the values of the line's parts tagged tag, in order.
func (l *nameLine) partValues(tag string) []string {
	var values []string
	for _, part := range l.parts {
		if part.Tag == tag {
			values = append(values, part.Value)
		}
	}
	return values
}

// partMismatch reports whether the line's GIVN or SURN disagrees with the
// given name or surname of its value, as CodeNamePartMismatch describes, with
// the disagreeing tags under "tags" and both texts of each.
func (l *nameLine) partMismatch() (map[string]string, bool) {
	var tags []string
	details := map[string]string{}
	// before and after are the pieces that may begin and end the text.
	pieces := []struct {
		tag, key, text string
		before, after  []string
	}{
		{"GIVN", "given", l.given, l.partValues("NPFX"), l.partValues("NSFX")},
		{"SURN", "surname", l.surname, l.partValues("SPFX"), nil},
	}
	noSlash := !strings.Contains(l.value, "/")
	for _, p := range pieces {
		values := l.partValues(p.tag)
		if len(values) == 0 {
			continue
		}
		piece := nameWords(values...)
		if noSlash && hasRun(nameWords(l.given), piece) ||
			!noSlash && pieceMatches(piece, nameWords(p.text), nameWords(p.before...), nameWords(p.after...)) {
			continue
		}
		tags = append(tags, p.tag)
		details[p.tag] = strings.Join(values, ", ")
		details[p.key] = p.text
	}
	if len(tags) == 0 {
		return nil, false
	}
	details["tags"] = strings.Join(tags, ",")
	return details, true
}

// pieceMatches reports whether the words of a name piece are the words of the
// value's text, with or without the leading prefix words and trailing suffix
// words the text holds, case folded.
func pieceMatches(piece, text, prefix, suffix []string) bool {
	if wordsEqual(piece, text) {
		return true
	}
	if len(prefix) <= len(text) && wordsEqual(prefix, text[:len(prefix)]) {
		text = text[len(prefix):]
	}
	if len(suffix) <= len(text) && wordsEqual(suffix, text[len(text)-len(suffix):]) {
		text = text[:len(text)-len(suffix)]
	}
	return wordsEqual(piece, text)
}

// hasRun reports whether run is a nonempty run of consecutive words of
// text, case folded.
func hasRun(text, run []string) bool {
	if len(run) == 0 {
		return false
	}
	for i := 0; i+len(run) <= len(text); i++ {
		if wordsEqual(run, text[i:i+len(run)]) {
			return true
		}
	}
	return false
}

// nameWords returns the words of texts, in order, split at spaces and
// commas, leaving out text in double quotes or parentheses.
func nameWords(texts ...string) []string {
	var words []string
	for _, text := range texts {
		words = append(words, strings.FieldsFunc(dropAsides(text), func(r rune) bool { return r == ',' || unicode.IsSpace(r) })...)
	}
	return words
}

// dropAsides returns s with each span in double quotes or parentheses, its
// marks included, replaced by a space. A mark with no closing mark is kept.
func dropAsides(s string) string {
	for _, marks := range [][2]string{{`"`, `"`}, {"“", "”"}, {"(", ")"}} {
		for {
			open := strings.Index(s, marks[0])
			if open < 0 {
				break
			}
			rest := open + len(marks[0])
			end := strings.Index(s[rest:], marks[1])
			if end < 0 {
				break
			}
			s = s[:open] + " " + s[rest+end+len(marks[1]):]
		}
	}
	return s
}

// wordsEqual reports whether a and b hold the same words, case folded.
func wordsEqual(a, b []string) bool {
	return slices.EqualFunc(a, b, strings.EqualFold)
}

// hasCapitalWord reports whether s has a word, a run of letters, of two or
// more letters that are all upper case.
func hasCapitalWord(s string) bool {
	notLetter := func(r rune) bool { return !unicode.IsLetter(r) }
	for _, word := range strings.FieldsFunc(s, notLetter) {
		if utf8.RuneCountInString(word) >= 2 && strings.IndexFunc(word, func(r rune) bool { return !unicode.IsUpper(r) }) < 0 {
			return true
		}
	}
	return false
}

// hasAlternative reports whether the word "or" stands between two words of
// value, slashes counting as spaces.
func hasAlternative(value string) bool {
	words := strings.Fields(strings.ReplaceAll(value, "/", " "))
	for i := 1; i < len(words)-1; i++ {
		if strings.EqualFold(words[i], "or") {
			return true
		}
	}
	return false
}

// hasEdgePunctuation reports whether s starts or ends with punctuation or a
// space.
func hasEdgePunctuation(s string) bool {
	if s == "" {
		return false
	}
	edge := func(r rune) bool { return unicode.IsPunct(r) || unicode.IsSpace(r) }
	runes := []rune(s)
	return edge(runes[0]) || edge(runes[len(runes)-1])
}
