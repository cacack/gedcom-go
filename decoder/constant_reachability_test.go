package decoder

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v2/gedcom"
)

// reachabilityGEDCOM carries, as positive level-1 lines, every tag the decoder
// maps to an EventType or AttributeType: the INDI event and attribute case
// lists and the FAM event and attribute case lists in entity.go.
const reachabilityGEDCOM = `0 HEAD
1 GEDC
2 VERS 5.5.1
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Doe/
1 BIRT
1 DEAT
1 BAPM
1 BURI
1 CENS
1 CHR
1 ADOP
1 RESI
1 IMMI
1 EMIG
1 BARM
1 BASM
1 BLES
1 CHRA
1 CONF
1 FCOM
1 GRAD
1 RETI
1 NATU
1 ORDN
1 PROB
1 WILL
1 CREM
1 EVEN
1 OCCU Farmer
1 CAST Caste
1 DSCR Tall
1 EDUC College
1 IDNO 123
1 NATI American
1 SSN 000-00-0000
1 TITL Sir
1 RELI Methodist
1 NCHI 2
1 NMR 1
1 PROP House
1 FACT Fact
0 @F1@ FAM
1 HUSB @I1@
1 MARR
1 DIV
1 ENGA
1 ANUL
1 MARB
1 MARC
1 MARL
1 MARS
1 DIVF
1 CENS
1 RESI
1 EVEN
1 NCHI 2
1 FACT Fact
0 TRLR
`

// TestConstantReachability guards issue #484: every exported EventType and
// AttributeType constant must be produced by decoding an ordinary document,
// so a constant the decoder never emits (like the removed EventOccupation)
// cannot reappear. Constants are read from the gedcom package source, not
// listed by hand, and negative (NO) assertions do not count.
func TestConstantReachability(t *testing.T) {
	doc, err := Decode(strings.NewReader(reachabilityGEDCOM))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	produced := map[string]bool{}
	for _, indi := range doc.Individuals() {
		collectProduced(produced, indi.Events, indi.Attributes)
	}
	for _, fam := range doc.Families() {
		collectProduced(produced, fam.Events, fam.Attributes)
	}

	consts := typedConstants(t, "../gedcom")
	for _, typeName := range []string{"EventType", "AttributeType"} {
		t.Run(typeName, func(t *testing.T) {
			byName := consts[typeName]
			if len(byName) == 0 {
				t.Fatalf("found no %s constants in ../gedcom; source scan is broken", typeName)
			}
			names := make([]string, 0, len(byName))
			for name := range byName {
				names = append(names, name)
			}
			sort.Strings(names)
			for _, name := range names {
				if value := byName[name]; !produced[typeName+"="+value] {
					t.Errorf("gedcom.%s (%q) is never produced by decoding", name, value)
				}
			}
		})
	}

	// The reverse holds for attributes only: every attribute tag the decoder
	// routes has a constant, as AttributeType's doc promises. Events have no
	// such promise (EVEN decodes with no constant), so they are not checked.
	attrValues := map[string]bool{}
	for _, value := range consts["AttributeType"] {
		attrValues[value] = true
	}
	for key := range produced {
		if value, ok := strings.CutPrefix(key, "AttributeType="); ok && !attrValues[value] {
			t.Errorf("decoder produces attribute type %q with no AttributeType constant", value)
		}
	}
}

// collectProduced records each positive event type and attribute type, keyed
// by Go type name so an EventType and AttributeType sharing a tag stay distinct.
func collectProduced(produced map[string]bool, events []*gedcom.Event, attrs []*gedcom.Attribute) {
	for _, event := range events {
		if !event.IsNegative {
			produced["EventType="+string(event.Type)] = true
		}
	}
	for _, attr := range attrs {
		produced["AttributeType="+string(attr.Type)] = true
	}
}

// typedConstants parses the non-test Go files in dir and returns, per declared
// type name, a map of constant name to string value for every const spec with
// an explicit type and string literal value.
func typedConstants(t *testing.T, dir string) map[string]map[string]string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatalf("glob %s: %v", dir, err)
	}

	consts := map[string]map[string]string{}
	fset := token.NewFileSet()
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.CONST {
				continue
			}
			for _, spec := range gen.Specs {
				addTypedConstant(t, consts, spec.(*ast.ValueSpec))
			}
		}
	}
	return consts
}

// addTypedConstant records the names and string values of one const spec
// whose type is a plain identifier.
func addTypedConstant(t *testing.T, consts map[string]map[string]string, spec *ast.ValueSpec) {
	t.Helper()
	typeIdent, ok := spec.Type.(*ast.Ident)
	if !ok {
		return
	}
	for i, name := range spec.Names {
		if i >= len(spec.Values) {
			continue
		}
		lit, ok := spec.Values[i].(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			continue
		}
		value, err := strconv.Unquote(lit.Value)
		if err != nil {
			t.Fatalf("unquote %s: %v", name.Name, err)
		}
		if consts[typeIdent.Name] == nil {
			consts[typeIdent.Name] = map[string]string{}
		}
		consts[typeIdent.Name][name.Name] = value
	}
}
