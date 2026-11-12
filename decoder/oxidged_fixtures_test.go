package decoder

// oxidged_fixtures_test.go checks the decoder against oxidged's hand-written
// "every structure" files (issue #763), vendored in testdata/oxidged. They were
// written by another project for its own 5.5.1 and 7.0 conformance reviews, so
// a structure this library mishandles is not hidden by the blind spot of the
// author who wrote our own fixtures.
//
// TestConformanceSuitesDecode covers strict and lenient decoding of them; the
// encoder package's TestOxidgedEntityRoundTrip covers the write path. The two
// checks here are what neither of those can see:
//
//   - every standard (non-underscore) tag is recognized: no UNKNOWN_TAG;
//   - every line the coverage reports call typed does reach the typed model
//     here too. Those reports probe each structure in a minimal document built
//     from the specification tables; these files are the independent
//     cross-check that the same structure is typed in a realistic one.
//
// Both lists are self-cleaning, in the style of byte_roundtrip_test.go.

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// oxidgedDir holds the vendored files; see testdata/oxidged/SOURCE.
const oxidgedDir = "../testdata/oxidged"

// oxidgedUnknownTagGaps lists standard tags the decoder reports as
// UNKNOWN_TAG, keyed "file:line TAG", with the issue responsible.
var oxidgedUnknownTagGaps = map[string]string{
	"full551.ged:166 CONC":    "#734: DSCR payload continuation",
	"full551.ged:167 CONT":    "#734: DSCR payload continuation",
	"full551.ged:247 CONT":    "#738: inline-text citation continuation",
	"full551.ged:307 RIN":     "#730: RIN on FAM",
	"full551.ged:329 RIN":     "#730: RIN on OBJE",
	"full551.ged:342 RIN":     "#730: RIN on NOTE",
	"full551.ged:356 RIN":     "#730: RIN on REPO",
	"full551.ged:386 RIN":     "#730: RIN on SOUR",
	"full551.ged:60 RFN":      "#730: RFN on SUBM",
	"maximal551.ged:167 CONC": "#734: DSCR payload continuation",
	"maximal551.ged:168 CONT": "#734: DSCR payload continuation",
	"maximal551.ged:248 CONT": "#738: inline-text citation continuation",
	"maximal551.ged:308 RIN":  "#730: RIN on FAM",
	"maximal551.ged:330 RIN":  "#730: RIN on OBJE",
	"maximal551.ged:343 RIN":  "#730: RIN on NOTE",
	"maximal551.ged:357 RIN":  "#730: RIN on REPO",
	"maximal551.ged:387 RIN":  "#730: RIN on SOUR",
	"maximal551.ged:61 RFN":   "#730: RFN on SUBM",
	"full70.ged:43 REFN":      "#730: REFN on SUBM",
	"full70.ged:53 CREA":      "#730: CREA on SUBM",
	"full70.ged:361 CREA":     "#730: CREA on REPO",
	"full70.ged:372 REFN":     "#730: REFN on SNOTE",
	"full70.ged:374 UID":      "#730: UID on SNOTE",
	"full70.ged:379 CREA":     "#730: CREA on SNOTE",
}

// oxidgedTypedGaps lists lines whose structure the coverage reports mark
// typed but which do not reach the typed model in these files, keyed
// "file:line CONTEXT.TAG", with the issue responsible.
var oxidgedTypedGaps = map[string]string{
	"full551.ged:17 HEAD.SOUR.DATA.COPR":             "#785: read into Header.Copyright, then replaced by HEAD.COPR",
	"full551.ged:344 NOTE.SOUR.PAGE":                 "#786: NOTE record citations untyped",
	"full551.ged:346 NOTE.CHAN.DATE":                 "#786: NOTE record CHAN untyped",
	"full551.ged:358 REPO.CHAN.DATE":                 "#786: REPO CHAN untyped",
	"full551.ged:64 SUBM.CHAN.DATE":                  "#786: SUBM CHAN untyped",
	"full551.ged:65 SUBM.CHAN.NOTE":                  "#786: SUBM CHAN untyped",
	"maximal551.ged:17 HEAD.SOUR.DATA.COPR":          "#785: read into Header.Copyright, then replaced by HEAD.COPR",
	"maximal551.ged:345 NOTE.SOUR.PAGE":              "#786: NOTE record citations untyped",
	"maximal551.ged:347 NOTE.CHAN.DATE":              "#786: NOTE record CHAN untyped",
	"maximal551.ged:359 REPO.CHAN.DATE":              "#786: REPO CHAN untyped",
	"maximal551.ged:65 SUBM.CHAN.DATE":               "#786: SUBM CHAN untyped",
	"maximal551.ged:66 SUBM.CHAN.NOTE":               "#786: SUBM CHAN untyped",
	"full70.ged:20 HEAD.SOUR.DATA.DATE.TIME":         "#785: HEAD.SOUR.DATA untyped",
	"full70.ged:21 HEAD.SOUR.DATA.COPR":              "#785: read into Header.Copyright, then replaced by HEAD.COPR",
	"full70.ged:27 HEAD.LANG":                        "#785: Header.Language overwritten by HEAD.NOTE.LANG",
	"full70.ged:44 SUBM.REFN.TYPE":                   "#786: SUBM REFN untyped",
	"full70.ged:50 SUBM.CHAN.DATE":                   "#786: SUBM CHAN untyped",
	"full70.ged:51 SUBM.CHAN.DATE.TIME":              "#786: SUBM CHAN untyped",
	"full70.ged:52 SUBM.CHAN.NOTE":                   "#786: SUBM CHAN untyped",
	"full70.ged:54 SUBM.CREA.DATE":                   "#786: SUBM CREA untyped",
	"full70.ged:55 SUBM.CREA.DATE.TIME":              "#786: SUBM CREA untyped",
	"full70.ged:62 INDI.NAME.GIVN":                   "#787: a repeated GIVN keeps only the last",
	"full70.ged:92 INDI.BIRT.PLAC.EXID.TYPE":         "#787: PLAC.EXID untyped",
	"full70.ged:123 INDI.BIRT.SOUR.DATA.DATE.PHRASE": "#787: citation DATA.DATE PHRASE untyped",
	"full70.ged:134 INDI.BIRT.SOUR.OBJE.CROP":        "#737: citation OBJE untyped",
	"full70.ged:135 INDI.BIRT.SOUR.OBJE.CROP.TOP":    "#737: citation OBJE untyped",
	"full70.ged:136 INDI.BIRT.SOUR.OBJE.CROP.LEFT":   "#737: citation OBJE untyped",
	"full70.ged:137 INDI.BIRT.SOUR.OBJE.CROP.HEIGHT": "#737: citation OBJE untyped",
	"full70.ged:138 INDI.BIRT.SOUR.OBJE.CROP.WIDTH":  "#737: citation OBJE untyped",
	"full70.ged:139 INDI.BIRT.SOUR.OBJE.TITL":        "#737: citation OBJE untyped",
	"full70.ged:196 INDI.INIL.STAT.DATE":             "#386: INIL left raw by decision",
	"full70.ged:240 INDI.NOTE.TRAN.LANG":             "#787: inline NOTE TRAN untyped",
	"full70.ged:284 FAM.MARR.HUSB.AGE.PHRASE":        "#736: family event HUSB/WIFE AGE untyped",
	"full70.ged:360 REPO.CHAN.DATE":                  "#786: REPO CHAN untyped",
	"full70.ged:362 REPO.CREA.DATE":                  "#786: REPO CREA untyped",
	"full70.ged:373 SNOTE.REFN.TYPE":                 "#786: SNOTE REFN untyped",
	"full70.ged:380 SNOTE.CREA.DATE":                 "#786: SNOTE CREA untyped",
	"full70.ged:387 SOUR.DATA.EVEN.PLAC.FORM":        "#737: SOUR.DATA untyped",
}

// oxidgedFiles returns the vendored .ged files, failing if there are none.
func oxidgedFiles(t *testing.T) []string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(oxidgedDir, "*.ged"))
	if err != nil {
		t.Fatalf("glob %s: %v", oxidgedDir, err)
	}
	if len(files) != 4 {
		t.Fatalf("found %d files in %s, want the 4 vendored ones", len(files), oxidgedDir)
	}
	sort.Strings(files)
	return files
}

// checkGaps reports each key of got that known does not list, and each entry
// of known that no longer occurs.
func checkGaps(t *testing.T, got, known map[string]string, listName string) {
	t.Helper()
	var unexpected, fixed []string
	for key, detail := range got {
		if _, ok := known[key]; !ok {
			unexpected = append(unexpected, key+": "+detail)
		}
	}
	for key, reason := range known {
		if _, ok := got[key]; !ok {
			fixed = append(fixed, fmt.Sprintf("%s (was: %s)", key, reason))
		}
	}
	sort.Strings(unexpected)
	sort.Strings(fixed)
	if len(unexpected) > 0 {
		t.Errorf("not listed in %s:\n  %s", listName, strings.Join(unexpected, "\n  "))
	}
	if len(fixed) > 0 {
		t.Errorf("no longer occur; remove them from %s:\n  %s", listName, strings.Join(fixed, "\n  "))
	}
}

func TestOxidgedNoUnknownStandardTags(t *testing.T) {
	got := map[string]string{}
	for _, path := range oxidgedFiles(t) {
		data, err := os.ReadFile(path) // #nosec G304 -- path comes from globbing the repo's own testdata
		if err != nil {
			t.Fatal(err)
		}
		res, err := DecodeWithDiagnostics(bytes.NewReader(data), nil)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		lines := strings.Split(string(data), "\n")
		for _, d := range res.Diagnostics {
			if d.Code != CodeUnknownTag || d.Line < 1 || d.Line > len(lines) {
				continue
			}
			line, ok := parseOxidgedLine(lines[d.Line-1])
			if !ok || strings.HasPrefix(line.tag, "_") {
				continue // an extension tag: ADR 0003 keeps it raw by design
			}
			got[fmt.Sprintf("%s:%d %s", filepath.Base(path), d.Line, line.tag)] = d.Message
		}
	}
	checkGaps(t, got, oxidgedUnknownTagGaps, "oxidgedUnknownTagGaps")
}

// oxidgedLine is one line of a fixture, split into the parts the cross-check
// needs.
type oxidgedLine struct {
	level   int
	tag     string
	payload string
}

// parseOxidgedLine splits a GEDCOM line. These files are well-formed, so this
// does not attempt the parser's error handling.
func parseOxidgedLine(raw string) (oxidgedLine, bool) {
	fields := strings.SplitN(strings.TrimRight(raw, "\r"), " ", 3)
	if len(fields) < 2 {
		return oxidgedLine{}, false
	}
	var l oxidgedLine
	if _, err := fmt.Sscanf(fields[0], "%d", &l.level); err != nil {
		return oxidgedLine{}, false
	}
	rest := fields[1:]
	if strings.HasPrefix(rest[0], "@") && len(rest) > 1 {
		rest = strings.SplitN(rest[1], " ", 2) // skip the record's own XRef
	}
	l.tag = rest[0]
	if len(rest) > 1 {
		l.payload = rest[1]
	}
	return l, true
}

// oxidgedSpec is what the cross-check needs of one version's specification.
type oxidgedSpec struct {
	graph *specGraph
	form  *specForm
}

func oxidgedSpecFor(t *testing.T, version string) oxidgedSpec {
	t.Helper()
	switch version {
	case "5.5.1":
		for _, edition := range spec55Editions {
			if edition.version == version {
				s := loadSpec55(t, edition)
				return oxidgedSpec{graph: s.specGraph, form: s.specForm}
			}
		}
	case "7.0":
		s := loadSpec7(t)
		return oxidgedSpec{graph: s.specGraph, form: s.specForm}
	}
	t.Fatalf("no specification for version %q", version)
	return oxidgedSpec{}
}

// resolve returns the pair a line with this tag and payload denotes under
// superstructure, and false when the tag is not defined there or when it names
// more than one structure that the payload's shape cannot tell apart.
func (s oxidgedSpec) resolve(byParent map[string][]specPair, superstructure, tag, payload string) (specPair, bool) {
	isPointer := strings.HasPrefix(payload, "@") && strings.HasSuffix(payload, "@") &&
		!strings.HasPrefix(payload, "@@") && !strings.HasPrefix(payload, "@#")
	var match []specPair
	for _, p := range byParent[superstructure] {
		if p.tag != tag {
			continue
		}
		if _, ptr := s.form.pointer(p.structure); ptr != isPointer {
			continue
		}
		match = append(match, p)
	}
	if len(match) != 1 {
		return specPair{}, false
	}
	return match[0], true
}

// TestOxidgedTypedStructuresDecode cross-checks the coverage reports: each
// line whose (superstructure, tag) pair the report's measurement calls typed is
// removed from the file, with its substructures, and the typed model must
// change -- or, failing that, must change when the line's payload does. A line
// that neither removal nor a new payload makes visible was not decoded into a
// typed field, whatever the minimal probe found.
//
// Lines the specification tables do not resolve to a single structure -- an
// extension tag, anything nested under one, CONT and CONC -- are skipped; the
// UNKNOWN_TAG test above is what covers recognition.
func TestOxidgedTypedStructuresDecode(t *testing.T) {
	specs := map[string]oxidgedSpec{}
	probers := map[string]*specProber{}
	statuses := map[string]map[specPair]specStatus{}

	got := map[string]string{}
	checked := 0
	for _, path := range oxidgedFiles(t) {
		data, err := os.ReadFile(path) // #nosec G304 -- path comes from globbing the repo's own testdata
		if err != nil {
			t.Fatal(err)
		}
		version := "7.0"
		if strings.Contains(filepath.Base(path), "551") {
			version = "5.5.1"
		}
		if _, ok := specs[version]; !ok {
			specs[version] = oxidgedSpecFor(t, version)
			probers[version] = newSpecProber(specs[version].graph, specs[version].form)
			statuses[version] = map[specPair]specStatus{}
		}
		spec, prober, status := specs[version], probers[version], statuses[version]
		byParent := map[string][]specPair{}
		for _, p := range spec.graph.pairs {
			byParent[p.superstructure] = append(byParent[p.superstructure], p)
		}

		lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
		full := oxidgedTypedModel(t, lines)

		// stack[i] is the structure of the enclosing line at level i, or ""
		// when that line did not resolve, which leaves its subtree unresolved.
		var stack []string
		var tagPath []string
		for i, raw := range lines {
			line, ok := parseOxidgedLine(raw)
			if !ok || line.level > len(stack) {
				t.Fatalf("%s:%d: unexpected line %q", path, i+1, raw)
			}
			stack, tagPath = stack[:line.level], tagPath[:line.level]
			superstructure := ""
			if line.level > 0 {
				superstructure = stack[line.level-1]
			}
			var pair specPair
			resolved := false
			if line.level == 0 || superstructure != "" {
				pair, resolved = spec.resolve(byParent, superstructure, line.tag, line.payload)
			}
			if line.tag == "CONT" || line.tag == "CONC" {
				resolved = false
			}
			structure := ""
			if resolved {
				structure = pair.structure
			}
			stack = append(stack, structure)
			tagPath = append(tagPath, line.tag)

			// Level-0 lines are records (always in the model) or HEAD/TRLR,
			// whose removal says nothing about one structure.
			if !resolved || line.level == 0 {
				continue
			}
			s, measured := status[pair]
			if !measured {
				_, s, ok = prober.measure(t, pair)
				if !ok {
					continue
				}
				status[pair] = s
			}
			if s != specTyped && s != specPartial {
				continue
			}
			checked++

			end := i + 1
			for end < len(lines) {
				next, ok := parseOxidgedLine(lines[end])
				if !ok || next.level <= line.level {
					break
				}
				end++
			}
			without := append(append([]string{}, lines[:i]...), lines[end:]...)
			if oxidgedTypedModel(t, without) == full && !oxidgedPayloadMatters(t, lines, i, line, full) {
				key := fmt.Sprintf("%s:%d %s", filepath.Base(path), i+1, strings.Join(tagPath, "."))
				got[key] = fmt.Sprintf("reported %s, but removing or changing %q leaves the typed model unchanged", s, raw)
			}
		}
	}
	if checked < 300 {
		t.Fatalf("only %d lines checked; the structure resolution no longer reaches the files", checked)
	}
	checkGaps(t, got, oxidgedTypedGaps, "oxidgedTypedGaps")
}

// oxidgedPayloadMatters reports whether changing line i's payload changes the
// typed model. It is the second chance for a line whose removal changed
// nothing because another line states the same value: SURN repeats the
// surname the NAME payload already gives, so removing it is invisible, but
// changing it is not.
func oxidgedPayloadMatters(t *testing.T, lines []string, i int, line oxidgedLine, full string) bool {
	t.Helper()
	if line.payload == "" || strings.HasPrefix(line.payload, "@") {
		return false // nothing to change, or a pointer whose target must exist
	}
	changed := append([]string{}, lines...)
	changed[i] = lines[i] + "X"
	return oxidgedTypedModel(t, changed) != full
}

// oxidgedTypedModel decodes lines leniently and renders the typed model. A
// document that no longer decodes renders as a marker, which differs from any
// model: the removed line evidently mattered.
func oxidgedTypedModel(t *testing.T, lines []string) string {
	t.Helper()
	res, err := DecodeWithDiagnostics(strings.NewReader(strings.Join(lines, "\n")+"\n"), nil)
	if err != nil || res == nil || res.Document == nil {
		return fmt.Sprintf("<decode failed: %v>", err)
	}
	return specTypedModel(res.Document)
}
