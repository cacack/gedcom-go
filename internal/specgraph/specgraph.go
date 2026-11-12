// Package specgraph reads a GEDCOM version's structure graph from the
// substructures.tsv vendored under testdata/spec, derives from two such
// graphs the structures one version defines that the other does not, and
// generates the converter's table of them (issue #705).
//
// It runs only at development time: internal/cmd/genforeign (go generate in
// the converter package) writes the table, and a converter test regenerates
// it to keep it in step with the specification files. Library code never
// reads testdata.
package specgraph

import (
	"bufio"
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// V7Prefix is the namespace of every standard GEDCOM 7.0 structure URI. It is
// trimmed from structure names so the generated table stays readable.
const V7Prefix = "https://gedcom.io/terms/v7/"

// Graph is one version's structure graph: the substructures of each
// structure, by tag. The empty superstructure holds the top-level structures
// (the records, HEAD and TRLR). A tag can name more than one structure under
// one superstructure -- 5.5.1 has a pointer and an inline
// NOTE_STRUCTURE.NOTE -- so each tag maps to a sorted list.
type Graph map[string]map[string][]string

// Pair is a tag in the context of the structure it appears under. An empty
// Super is the top level.
type Pair struct {
	Super string
	Tag   string
}

// Load reads a substructures.tsv: a header row "superstructure tag
// structure", then one tab-separated row per pair. prefix is trimmed from
// every structure name.
func Load(path, prefix string) (Graph, error) {
	f, err := os.Open(path) // #nosec G304 -- development-time tool reading the vendored spec data
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	g := Graph{}
	scanner := bufio.NewScanner(f)
	for line := 1; scanner.Scan(); line++ {
		fields := strings.Split(scanner.Text(), "\t")
		if len(fields) != 3 {
			return nil, fmt.Errorf("%s:%d: want 3 fields, got %d", path, line, len(fields))
		}
		if line == 1 {
			if fields[0] != "superstructure" || fields[1] != "tag" || fields[2] != "structure" {
				return nil, fmt.Errorf("%s: missing header row", path)
			}
			continue
		}
		super := strings.TrimPrefix(fields[0], prefix)
		structure := strings.TrimPrefix(fields[2], prefix)
		tags := g[super]
		if tags == nil {
			tags = map[string][]string{}
			g[super] = tags
		}
		if !slices.Contains(tags[fields[1]], structure) {
			tags[fields[1]] = append(tags[fields[1]], structure)
			slices.Sort(tags[fields[1]])
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return g, nil
}

// Triple is a tag in the context of its superstructure and that
// superstructure's own parent (Grand), for a pair whose foreignness the
// superstructure alone does not decide.
type Triple struct {
	Grand string
	Super string
	Tag   string
}

// Foreign returns what src defines that tgt has no structure for.
//
// Contexts correspond by tag path: a src structure reached by INDI.BIRT
// corresponds to whatever INDI.BIRT reaches in tgt. A pair is foreign when,
// in every context its superstructure is reached in, the corresponding tgt
// structures have no substructure with its tag. Some pairs are foreign only
// in some contexts: 7.0 has one DATE-exact, with a TIME, where 5.5.1 allows
// TIME only on a CHAN's DATE. For those the superstructure's parent is
// considered too, and a triple is foreign when tgt lacks it in every context
// that triple is reached in. What even that does not decide is returned as
// mixed and left alone: dropping a structure the target can carry would lose
// data for nothing.
//
// Nothing beneath a foreign structure is visited: it goes with its parent.
// All three slices are sorted.
func Foreign(src, tgt Graph) (pairs []Pair, triples, mixed []Triple) {
	lacks, has := map[Pair]bool{}, map[Pair]bool{}
	lacks3, has3 := map[Triple]bool{}, map[Triple]bool{}
	seen := map[string]bool{}

	var visit func(grand, super string, tgtSupers []string)
	visit = func(grand, super string, tgtSupers []string) {
		key := grand + "\x01" + super + "\x01" + strings.Join(tgtSupers, "\x00")
		if seen[key] {
			return
		}
		seen[key] = true
		for _, tag := range sortedKeys(src[super]) {
			var next []string
			for _, t := range tgtSupers {
				for _, s := range tgt[t][tag] {
					if !slices.Contains(next, s) {
						next = append(next, s)
					}
				}
			}
			slices.Sort(next)
			pair, triple := Pair{super, tag}, Triple{grand, super, tag}
			if len(next) == 0 {
				lacks[pair], lacks3[triple] = true, true
				continue
			}
			has[pair], has3[triple] = true, true
			for _, s := range src[super][tag] {
				visit(super, s, next)
			}
		}
	}
	visit("", "", []string{""})

	for pair := range lacks {
		if !has[pair] {
			pairs = append(pairs, pair)
		}
	}
	for triple := range lacks3 {
		if !has[Pair{triple.Super, triple.Tag}] {
			continue // the pair is foreign everywhere
		}
		if has3[triple] {
			mixed = append(mixed, triple)
		} else {
			triples = append(triples, triple)
		}
	}
	slices.SortFunc(pairs, comparePairs)
	slices.SortFunc(triples, compareTriples)
	slices.SortFunc(mixed, compareTriples)
	return pairs, triples, mixed
}

func compareTriples(a, b Triple) int {
	if c := strings.Compare(a.Grand, b.Grand); c != 0 {
		return c
	}
	return comparePairs(Pair{a.Super, a.Tag}, Pair{b.Super, b.Tag})
}

func comparePairs(a, b Pair) int {
	if c := strings.Compare(a.Super, b.Super); c != 0 {
		return c
	}
	return strings.Compare(a.Tag, b.Tag)
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// Generate returns the Go source of the converter's structure table, read
// from the spec directory (testdata/spec): the 7.0, 5.5.1 and 5.5 structure
// graphs and, between 7.0 and each 5.x version, the pairs each lacks of the
// other. The output is gofmt-formatted and
// depends only on the files, so a regeneration that changes nothing is
// byte-identical.
func Generate(specDir string) ([]byte, error) {
	g70, err := Load(filepath.Join(specDir, "gedcom-7.0", "substructures.tsv"), V7Prefix)
	if err != nil {
		return nil, err
	}
	g551, err := Load(filepath.Join(specDir, "gedcom-5.5.1", "substructures.tsv"), "")
	if err != nil {
		return nil, err
	}
	g55, err := Load(filepath.Join(specDir, "gedcom-5.5", "substructures.tsv"), "")
	if err != nil {
		return nil, err
	}
	foreign70, foreign70In, _ := Foreign(g70, g551)
	foreign551, foreign551In, _ := Foreign(g551, g70)
	foreign70To55, foreign70To55In, _ := Foreign(g70, g55)
	foreign55, foreign55In, _ := Foreign(g55, g70)

	var b bytes.Buffer
	b.WriteString(`// Code generated by internal/cmd/genforeign from testdata/spec/*/substructures.tsv; DO NOT EDIT.

package converter

`)
	writeGraph(&b, "specStructures70", "is the GEDCOM 7.0 structure graph, with structure URIs shortened\n// to their term names (record-INDI, INDI-NAME, ...).", g70)
	writeGraph(&b, "specStructures551", "is the GEDCOM 5.5.1 structure graph, structures named by their\n// grammar productions (INDIVIDUAL_RECORD.INDI, ...).", g551)
	writeGraph(&b, "specStructures55", "is the GEDCOM 5.5 structure graph, structures named by their\n// grammar productions (INDIVIDUAL_RECORD.INDI, ...).", g55)
	writePairs(&b, "foreignStructures70To551", "are the 7.0 structures with no GEDCOM 5.5.1 form, by the 7.0\n// superstructure they appear under.", foreign70)
	writePairs(&b, "foreignStructures551To70", "are the 5.5.1 structures with no GEDCOM 7.0 form, by the 5.5.1\n// superstructure they appear under.", foreign551)
	writeTriples(&b, "foreignStructuresIn70To551", "are the 7.0 structures with no GEDCOM 5.5.1 form under\n// some parents of their superstructure only, by superstructure and parent.", foreign70In)
	writeTriples(&b, "foreignStructuresIn551To70", "are the 5.5.1 structures with no GEDCOM 7.0 form under\n// some parents of their superstructure only, by superstructure and parent.", foreign551In)
	writePairs(&b, "foreignStructures70To55", "are the 7.0 structures with no GEDCOM 5.5 form, by the 7.0\n// superstructure they appear under.", foreign70To55)
	writePairs(&b, "foreignStructures55To70", "are the 5.5 structures with no GEDCOM 7.0 form, by the 5.5\n// superstructure they appear under.", foreign55)
	writeTriples(&b, "foreignStructuresIn70To55", "are the 7.0 structures with no GEDCOM 5.5 form under\n// some parents of their superstructure only, by superstructure and parent.", foreign70To55In)
	writeTriples(&b, "foreignStructuresIn55To70", "are the 5.5 structures with no GEDCOM 7.0 form under\n// some parents of their superstructure only, by superstructure and parent.", foreign55In)

	return format.Source(b.Bytes())
}

func writeGraph(b *bytes.Buffer, name, doc string, g Graph) {
	fmt.Fprintf(b, "// %s %s\nvar %s = specGraph{\n", name, doc, name)
	for _, super := range sortedKeys(g) {
		fmt.Fprintf(b, "\t%q: {", super)
		for i, tag := range sortedKeys(g[super]) {
			if i > 0 {
				b.WriteString(", ")
			}
			fmt.Fprintf(b, "%q: {", tag)
			for j, s := range g[super][tag] {
				if j > 0 {
					b.WriteString(", ")
				}
				fmt.Fprintf(b, "%q", s)
			}
			b.WriteString("}")
		}
		b.WriteString("},\n")
	}
	b.WriteString("}\n\n")
}

func writePairs(b *bytes.Buffer, name, doc string, pairs []Pair) {
	fmt.Fprintf(b, "// %s %s\nvar %s = map[structurePair]bool{\n", name, doc, name)
	for _, p := range pairs {
		fmt.Fprintf(b, "\t{%q, %q}: true,\n", p.Super, p.Tag)
	}
	b.WriteString("}\n\n")
}

func writeTriples(b *bytes.Buffer, name, doc string, triples []Triple) {
	fmt.Fprintf(b, "// %s %s\nvar %s = map[structureTriple]bool{\n", name, doc, name)
	for _, t := range triples {
		fmt.Fprintf(b, "\t{%q, %q, %q}: true,\n", t.Grand, t.Super, t.Tag)
	}
	b.WriteString("}\n\n")
}
