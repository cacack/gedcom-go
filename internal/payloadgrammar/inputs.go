package payloadgrammar

import (
	"bufio"
	"bytes"
	"hash/fnv"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/cacack/gedcom-go/v3/internal/abnf"
	"github.com/cacack/gedcom-go/v3/internal/corpora"
	"github.com/cacack/gedcom-go/v3/internal/specgraph"
)

// CorpusDirs are the directories, relative to the repository root, whose
// GEDCOM 7.0 files supply corpus values. The fetched gedcom.io files
// (corpora.GEDCOMIODir) are added when present.
var CorpusDirs = []string{"testdata/gedcom-7.0", "testdata/gedcom7code", "testdata/oxidged"}

// payloadTypes maps a payloads.tsv datatype URI to the Type it holds.
var payloadTypes = map[string]Type{
	"type-Date":       DateValue,
	"type-Date#exact": DateExact,
	"type-Time":       Time,
	"type-Age":        Age,
	"type-Name":       PersonalName,
	"type-Latitude":   Latitude,
	"type-Longitude":  Longitude,
}

// Generated is how many strings [Inputs] generates from the grammar for
// each Type, and each generated or corpus value also yields one near miss.
const Generated = 3000

// edgeCases are inputs no corpus or walk is sure to produce.
var edgeCases = map[Type][]string{
	DateValue: {"", " ", "1 JAN 2000 ", "1  JAN 2000", "abt 2000", "FROM", "BET 2000 AND", "TO 1 JAN 2000",
		"FROM JULIAN 1 JAN 1700 TO 1 JAN 1800", "BET HEBREW 1 TSH 5000 AND 1 JAN 2000", "HEBREW 5000 BCE",
		"31 FEB 2000", "30 COMP 5", "FRENCH_R 6 COMP 5", "_MAYAN 5 _KIN 2000", "_X 2000", "2000 _EP", "0 JAN 2000", "JAN 0",
		"INT 2000 (x)", "(x)", "1750/51", "44 BC", "44 B.C.", "@#DJULIAN@ 2000"},
	DateExact: {"", "1 JAN 2000", "01 JAN 2000", "31 FEB 2000", "29 FEB 1900", "32 JAN 2000", "0 JAN 2000", "1 JAN 0",
		"1 jan 2000", "1 JAN 1699/00", "1 VEND 2000"},
	Time:         {"", "0:00", "24:00", "23:59:60", "1:2", "12:00:00.Z", "12:00z"},
	Age:          {"", "0y", "> 1y", "<1y", "1y2m", "1Y", "CHILD", "1y 1y"},
	PersonalName: {"", "/", "//", "///", "John", "John /Doe", "/Doe/", "John\t/Doe/", "John /Doe/ Jr./Sr"},
	Latitude:     {"N", "N.5", "N5.", "N090", "N90.0", "N90.5", "n42", " N42", "N-1", "N1e2"},
	Longitude:    {"E", "E.5", "E5.", "E0180", "E180.5", "E018", "w71", "E-1"},
	Xref:         {"@@", "@VOID@", "@void@", "@I1", "I1@", "@I 1@", "@I@1@", "@i1@", "@_@", "@é@"},
}

// overrides steer the generator to the values a datatype's prose allows,
// with near misses among them.
var overrides = map[Type]map[string][]string{
	DateValue: {
		"calendar": {"GREGORIAN", "JULIAN", "FRENCH_R", "HEBREW", "_CAL"},
		"month": {"JAN", "FEB", "MAR", "APR", "MAY", "JUN", "JUL", "AUG", "SEP", "OCT", "NOV", "DEC",
			"VEND", "BRUM", "FRIM", "NIVO", "PLUV", "VENT", "GERM", "FLOR", "PRAI", "MESS", "THER", "FRUC", "COMP", //nolint:misspell // THER is Thermidor
			"TSH", "CSH", "KSL", "TVT", "SHV", "ADR", "ADS", "NSN", "IYR", "SVN", "TMZ", "AAV", "ELL", "_MON", "FOO"},
		"epoch": {"BCE", "_EP"},
		"day":   {"1", "9", "28", "29", "30", "31", "32", "0", "01"},
		"year":  {"1", "44", "1066", "1900", "2000", "5785", "0", "02000"},
	},
	DateExact: {
		"month": {"JAN", "FEB", "APR", "DEC", "VEND", "TSH", "_MON"},
		"day":   {"1", "9", "28", "29", "30", "31", "32", "0", "01"},
		"year":  {"1", "1900", "2000", "0", "02000"},
	},
}

var (
	corpusOnce   sync.Once
	corpusValues map[Type][]string
	errCorpus    error
)

// Corpus returns the distinct payload values of each Type in the GEDCOM 7.0
// files under [CorpusDirs], in a stable order. Xref values are every
// cross-reference identifier and every pointer payload.
func Corpus(tb testing.TB) map[Type][]string {
	tb.Helper()
	corpusOnce.Do(func() { corpusValues, errCorpus = readCorpus() })
	if errCorpus != nil {
		tb.Fatal(errCorpus)
	}
	return corpusValues
}

// Inputs returns the strings to compare for t: the corpus values, the edge
// cases, [Generated] strings walked from t's rule, and a near miss of each,
// deduplicated. The same grammar always gives the same inputs.
func Inputs(tb testing.TB, t Type) []string {
	tb.Helper()
	o := Load(tb)
	h := fnv.New64a()
	_, _ = h.Write([]byte(t))
	gen := abnf.NewGenerator(o.g, h.Sum64())
	gen.Override = overrides[t]
	gen.Exclude = func(r rune) bool { return o.Match("banned", string(r)) }

	inputs := slices.Clone(Corpus(tb)[t])
	inputs = append(inputs, edgeCases[t]...)
	for range Generated {
		s, err := gen.Generate(string(t))
		if err != nil {
			tb.Fatalf("generate %s: %v", t, err)
		}
		inputs = append(inputs, s)
	}
	for _, s := range slices.Clone(inputs) {
		inputs = append(inputs, gen.Mutate(s))
	}
	seen := map[string]bool{}
	return slices.DeleteFunc(inputs, func(s string) bool {
		dup := seen[s]
		seen[s] = true
		return dup
	})
}

func readCorpus() (map[Type][]string, error) {
	root, err := corpora.RepoRoot()
	if err != nil {
		return nil, err
	}
	spec := filepath.Join(root, "testdata", "spec", "gedcom-7.0")
	graph, err := specgraph.Load(filepath.Join(spec, "substructures.tsv"), specgraph.V7Prefix)
	if err != nil {
		return nil, err
	}
	payloads, err := readPayloads(filepath.Join(spec, "payloads.tsv"))
	if err != nil {
		return nil, err
	}
	dirs := slices.Clone(CorpusDirs)
	if info, err := os.Stat(filepath.Join(root, filepath.FromSlash(corpora.GEDCOMIODir))); err == nil && info.IsDir() {
		dirs = append(dirs, corpora.GEDCOMIODir)
	}
	c := &collector{graph: graph, payloads: payloads, values: map[Type][]string{}, seen: map[Type]map[string]bool{}}
	for _, dir := range dirs {
		err := filepath.WalkDir(filepath.Join(root, filepath.FromSlash(dir)), func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.EqualFold(filepath.Ext(path), ".ged") {
				return err
			}
			data, err := os.ReadFile(path) // #nosec G122 G304 -- test corpus under the repository root
			if err != nil {
				return err
			}
			c.file(data)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return c.values, nil
}

// readPayloads reads payloads.tsv: structure name (prefix trimmed) to its
// datatype, also prefix trimmed.
func readPayloads(path string) (map[string]string, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- vendored spec data
	if err != nil {
		return nil, err
	}
	payloads := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n")[1:] {
		structure, payload, _ := strings.Cut(line, "\t")
		payloads[strings.TrimPrefix(structure, specgraph.V7Prefix)] = strings.TrimPrefix(payload, specgraph.V7Prefix)
	}
	return payloads, nil
}

type collector struct {
	graph    specgraph.Graph
	payloads map[string]string
	values   map[Type][]string
	seen     map[Type]map[string]bool
}

func (c *collector) add(t Type, v string) {
	if c.seen[t] == nil {
		c.seen[t] = map[string]bool{}
	}
	if !c.seen[t][v] {
		c.seen[t][v] = true
		c.values[t] = append(c.values[t], v)
	}
}

// file collects the values of one file, if its header declares 7.x. It
// tracks the structure of each level through substructures.tsv; below an
// extension or unknown tag nothing is typed.
func (c *collector) file(data []byte) {
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF}) // a UTF-8 byte order mark
	lines := splitLines(data)
	if !declares7(lines) {
		return
	}
	var stack []string // structure at each level; "" when unknown
	for _, line := range lines {
		level, xref, tag, value, ok := splitLine(line)
		if !ok || level > len(stack) {
			continue
		}
		stack = stack[:level]
		super := ""
		if level > 0 {
			super = stack[level-1]
		}
		structure := ""
		if level == 0 || super != "" {
			if candidates := c.graph[super][tag]; len(candidates) == 1 {
				structure = candidates[0]
			}
		}
		stack = append(stack, structure)
		if xref != "" {
			c.add(Xref, xref)
		}
		payload := c.payloads[structure]
		switch t, typed := payloadTypes[payload]; {
		case typed:
			c.add(t, value)
		case strings.HasPrefix(payload, "@<") && value != "@VOID@" && value != "":
			c.add(Xref, value)
		}
	}
}

func splitLines(data []byte) []string {
	var lines []string
	sc := bufio.NewScanner(bytes.NewReader(bytes.ReplaceAll(bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n")), []byte("\r"), []byte("\n"))))
	sc.Buffer(nil, 1<<20)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	return lines
}

// declares7 reports whether the header's GEDC.VERS is 7.x.
func declares7(lines []string) bool {
	inGEDC := false
	for _, line := range lines {
		level, _, tag, value, ok := splitLine(line)
		switch {
		case !ok:
		case level == 0 && tag != "HEAD":
			return false
		case level == 1:
			inGEDC = tag == "GEDC"
		case level == 2 && inGEDC && tag == "VERS":
			return strings.HasPrefix(value, "7.")
		}
	}
	return false
}

// splitLine splits "level [xref] tag [value]" as the 7.0 Line rule does,
// one space between fields.
func splitLine(line string) (level int, xref, tag, value string, ok bool) {
	levelText, rest, _ := strings.Cut(line, " ")
	if levelText == "" || strings.Trim(levelText, "0123456789") != "" || len(levelText) > 2 {
		return 0, "", "", "", false
	}
	level = atoi(levelText)
	if strings.HasPrefix(rest, "@") {
		xref, rest, _ = strings.Cut(rest, " ")
	}
	tag, value, _ = strings.Cut(rest, " ")
	return level, xref, tag, value, tag != ""
}
