package specinterp

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/internal/corpora"
)

const tableHead = "| ID | Specification | Kind | Policy | Stated in | Enforced by |\n|---|---|---|---|---|---|\n"

func TestParse(t *testing.T) {
	text := "# Register\n\nprose `x`\n\n" + tableHead +
		"| SI-001 | 5.5.1 p.12 `n DEAT [Y\\|<NULL>]` | lenient | Policy. | `gedcom.A`, `gedcom.B.C` | `gedcom.TestA` |\n" +
		"| SI-002 | [7.0](https://gedcom.io/specifications/FamilySearchGEDCOMv7.html#lines) | choice | P | `parser.X` | `parser.TestX`, `parser.FuzzX` |\n" +
		"\nafter\n| not | the | table |\n"
	reg, err := Parse(strings.NewReader(text))
	if err != nil {
		t.Fatal(err)
	}
	if len(reg.Rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(reg.Rows))
	}
	row, ok := reg.Row("SI-001")
	if !ok {
		t.Fatal("SI-001 not found")
	}
	if row.Specification != "5.5.1 p.12 `n DEAT [Y|<NULL>]`" {
		t.Errorf("Specification = %q: an escaped pipe must stay in its cell", row.Specification)
	}
	if strings.Join(row.StatedIn, ",") != "gedcom.A,gedcom.B.C" || strings.Join(row.EnforcedBy, ",") != "gedcom.TestA" {
		t.Errorf("StatedIn = %q, EnforcedBy = %q", row.StatedIn, row.EnforcedBy)
	}
	if row, _ := reg.Row("SI-002"); row.Kind != KindChoice || len(row.EnforcedBy) != 2 {
		t.Errorf("SI-002 = %+v", row)
	}
	if _, ok := reg.Row("SI-003"); ok {
		t.Error("SI-003 found in a register without it")
	}
}

func TestParseErrors(t *testing.T) {
	good := "| SI-001 | p.1 | lenient | P | `a.B` | `a.TestB` |\n"
	tests := map[string]string{
		"no table":        "# Register\n",
		"no delimiter":    "| ID | Specification | Kind | Policy | Stated in | Enforced by |",
		"no rows":         tableHead,
		"cell count":      tableHead + "| SI-001 | p.1 | lenient |\n",
		"bad ID":          tableHead + "| SI-1 | p.1 | lenient | P | `a.B` | `a.TestB` |\n",
		"duplicate ID":    tableHead + good + good,
		"no location":     tableHead + "| SI-001 | somewhere | lenient | P | `a.B` | `a.TestB` |\n",
		"bad kind":        tableHead + "| SI-001 | p.1 | odd | P | `a.B` | `a.TestB` |\n",
		"no policy":       tableHead + "| SI-001 | p.1 | silent |  | `a.B` | `a.TestB` |\n",
		"no symbol":       tableHead + "| SI-001 | p.1 | silent | P | a.B | `a.TestB` |\n",
		"no test":         tableHead + "| SI-001 | p.1 | silent | P | `a.B` | none |\n",
		"scanner failure": tableHead + "| " + strings.Repeat("x", 1<<17) + " |\n",
	}
	for name, text := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := Parse(strings.NewReader(text)); err == nil {
				t.Error("Parse succeeded, want an error")
			}
		})
	}
}

func TestRequire(t *testing.T) {
	if row := Require(t, "SI-001"); row.ID != "SI-001" {
		t.Errorf("Require(SI-001).ID = %q", row.ID)
	}
}

// TestRegisterMatchesSource holds the register to the source tree: each
// row's first symbol is exported, every symbol exists and its doc comment
// cites the row, and every test exists.
func TestRegisterMatchesSource(t *testing.T) {
	reg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	root := repoRoot(t)
	pkgs := map[string]*sourcePackage{}
	load := func(dir string) *sourcePackage {
		if p, ok := pkgs[dir]; ok {
			return p
		}
		p, err := parsePackage(filepath.Join(root, filepath.FromSlash(dir)))
		if err != nil {
			t.Errorf("package %s: %v", dir, err)
		}
		pkgs[dir] = p
		return p
	}
	for _, row := range reg.Rows {
		for i, sym := range row.StatedIn {
			dir, name, ok := splitSymbol(sym)
			if !ok {
				t.Errorf("%s: %q is not <package directory>.<symbol>", row.ID, sym)
				continue
			}
			p := load(dir)
			if p == nil {
				continue
			}
			doc, found := p.docs[name]
			switch {
			case !found:
				t.Errorf("%s: symbol %s does not exist", row.ID, sym)
			case !strings.Contains(doc, row.ID):
				t.Errorf("%s: the doc comment of %s does not cite %s", row.ID, sym, row.ID)
			case i == 0 && !ast.IsExported(lastPart(name)):
				t.Errorf("%s: %s, the symbol that states the policy, is not exported", row.ID, sym)
			}
		}
		for _, test := range row.EnforcedBy {
			dir, name, ok := splitSymbol(test)
			if !ok || strings.Contains(name, ".") ||
				!strings.HasPrefix(name, "Test") && !strings.HasPrefix(name, "Fuzz") {
				t.Errorf("%s: %q is not <package directory>.<Test or Fuzz name>", row.ID, test)
				continue
			}
			if p := load(dir); p != nil && !p.tests[name] {
				t.Errorf("%s: test %s does not exist", row.ID, test)
			}
		}
	}
}

// citation is a register ID as source code cites it.
var citation = regexp.MustCompile(`\bSI-\d{3}\b`)

// TestSourceCitesOnlyRegisteredRows fails when a Go file outside this package
// cites an ID the register has no row for, and when a library package imports
// this one.
func TestSourceCitesOnlyRegisteredRows(t *testing.T) {
	reg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	root := repoRoot(t)
	importPath := strconv.Quote("github.com/cacack/gedcom-go/v3/internal/specinterp")
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != root && (strings.HasPrefix(d.Name(), ".") || d.Name() == "testdata" || d.Name() == "docs") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || filepath.Dir(path) == filepath.Join(root, "internal", "specinterp") {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		for _, id := range citation.FindAllString(string(src), -1) {
			if _, ok := reg.Row(id); !ok {
				t.Errorf("%s cites %s, which %s has no row for", rel, id, Path)
			}
		}
		if !strings.HasSuffix(path, "_test.go") && !strings.HasPrefix(filepath.ToSlash(rel), "internal/") &&
			strings.Contains(string(src), importPath) {
			t.Errorf("%s imports internal/specinterp, which only tests may", rel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := corpora.RepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// splitSymbol splits "internal/atsign.Unescape" into its package directory
// and the name within it.
func splitSymbol(s string) (dir, name string, ok bool) {
	slash := strings.LastIndex(s, "/")
	dot := strings.Index(s[slash+1:], ".")
	if dot < 0 {
		return "", "", false
	}
	dot += slash + 1
	dir, name = s[:dot], s[dot+1:]
	return dir, name, dir != "" && name != ""
}

func lastPart(name string) string {
	return name[strings.LastIndex(name, ".")+1:]
}

// sourcePackage is what the register check needs from one package
// directory: the doc comment of each symbol its non-test files declare, by
// "Name" or "Type.Member", and the names of the tests and fuzz targets its
// test files declare.
type sourcePackage struct {
	docs  map[string]string
	tests map[string]bool
}

func parsePackage(dir string) (*sourcePackage, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	p := &sourcePackage{docs: map[string]string{}, tests: map[string]bool{}}
	fset := token.NewFileSet()
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, e.Name()), nil, parser.ParseComments)
		if err != nil {
			return nil, err
		}
		if strings.HasSuffix(e.Name(), "_test.go") {
			for _, decl := range f.Decls {
				if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv == nil {
					p.tests[fn.Name.Name] = true
				}
			}
			continue
		}
		for _, decl := range f.Decls {
			p.addDecl(decl)
		}
	}
	return p, nil
}

func (p *sourcePackage) addDecl(decl ast.Decl) {
	switch d := decl.(type) {
	case *ast.FuncDecl:
		name := d.Name.Name
		if d.Recv != nil && len(d.Recv.List) == 1 {
			name = receiverType(d.Recv.List[0].Type) + "." + name
		}
		p.docs[name] = d.Doc.Text()
	case *ast.GenDecl:
		for _, spec := range d.Specs {
			switch s := spec.(type) {
			case *ast.TypeSpec:
				p.docs[s.Name.Name] = docOf(s.Doc, d)
				p.addMembers(s.Name.Name, s.Type)
			case *ast.ValueSpec:
				for _, n := range s.Names {
					p.docs[n.Name] = docOf(s.Doc, d)
				}
			}
		}
	}
}

// addMembers records the fields of a struct type and the methods of an
// interface type under "Type.Member".
func (p *sourcePackage) addMembers(typeName string, expr ast.Expr) {
	var fields *ast.FieldList
	switch t := expr.(type) {
	case *ast.StructType:
		fields = t.Fields
	case *ast.InterfaceType:
		fields = t.Methods
	default:
		return
	}
	for _, field := range fields.List {
		for _, n := range field.Names {
			p.docs[typeName+"."+n.Name] = field.Doc.Text()
		}
	}
}

// docOf returns a spec's own doc comment, or its declaration's when the
// declaration holds only that spec.
func docOf(doc *ast.CommentGroup, d *ast.GenDecl) string {
	if doc == nil && len(d.Specs) == 1 {
		doc = d.Doc
	}
	return doc.Text()
}

func receiverType(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return receiverType(t.X)
	case *ast.IndexExpr:
		return receiverType(t.X)
	case *ast.IndexListExpr:
		return receiverType(t.X)
	case *ast.Ident:
		return t.Name
	}
	return ""
}
