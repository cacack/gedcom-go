package gedcomgo

// README.md's Go blocks and the code in each package overview (doc.go, and
// gedcom_api.go for this package) are the first code a pkg.go.dev visitor
// reads, and they drifted from the API unnoticed (#713). This type-checks every
// one of them against the module source.
//
// A block is checked as written when it is a whole file (it has a package
// clause) or parses as top-level declarations (imports, funcs). Anything else
// is a fragment: it is placed in a function body, with the module's packages
// and a few standard ones imported, and the free variables fragments assume
// (doc, reader, f, ...) declared by snippetFixtures. Each block is checked on
// its own, so blocks never see each other's variables.
//
// Unused variables and imports are not reported: a fragment that ends on
// `converted, report, err := ...` is showing a call, not finishing a program.
// Everything else the type checker rejects fails the test.

import (
	"fmt"
	"go/ast"
	"go/doc/comment"
	"go/importer"
	goparser "go/parser"
	"go/token"
	"go/types"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// snippetImports are the packages a fragment may use without importing them.
const snippetImports = `import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"testing"
	"time"

	gedcomgo "github.com/cacack/gedcom-go/v3"
	"github.com/cacack/gedcom-go/v3/charset"
	"github.com/cacack/gedcom-go/v3/converter"
	"github.com/cacack/gedcom-go/v3/decoder"
	"github.com/cacack/gedcom-go/v3/encoder"
	"github.com/cacack/gedcom-go/v3/gedcom"
	gedcomtesting "github.com/cacack/gedcom-go/v3/gedcom/testing"
	"github.com/cacack/gedcom-go/v3/merge"
	"github.com/cacack/gedcom-go/v3/parser"
	"github.com/cacack/gedcom-go/v3/validator"
	"github.com/cacack/gedcom-go/v3/version"
)
`

// snippetFixtures declares the variables fragments use without declaring. A
// fragment that uses one with a different type fails, as it should.
const snippetFixtures = `import (
	"io"
	"os"

	gedcomgo "github.com/cacack/gedcom-go/v3"
	"github.com/cacack/gedcom-go/v3/gedcom"
)

var (
	doc          *gedcom.Document
	records      []*gedcom.Record
	errs         []error
	reader, r    io.Reader
	w, writer    io.Writer
	f, file      *os.File
	fileInfo     os.FileInfo
	decodeOpts   *gedcomgo.DecodeOptions
	encodeOpts   *gedcomgo.EncodeOptions
	validateOpts *gedcomgo.ValidateOptions
)
`

// snippet is one code block and where it came from.
type snippet struct {
	where string
	src   string
}

func TestDocSnippetsCompile(t *testing.T) {
	checker := newSnippetChecker(t)
	snippets := append(readmeSnippets(t), packageOverviewSnippets(t)...)
	if len(snippets) < 20 {
		t.Fatalf("found only %d snippets; has the README fence or doc.go format changed?", len(snippets))
	}
	for _, s := range snippets {
		for _, problem := range checker.check(s.src) {
			t.Errorf("%s: %s\n%s", s.where, problem, s.src)
		}
	}
}

// TestReadmeStabilityTableNamesExist checks the README "Stable Packages" table:
// `Name()` must be a package function, `Type.Method()` a method, and a bare
// `Name` a type.
func TestReadmeStabilityTableNamesExist(t *testing.T) {
	readme := readFile(t, "README.md")
	_, table, ok := strings.Cut(readme, "### Stable Packages\n")
	if !ok {
		t.Fatal(`README.md has no "### Stable Packages" section; update this test if it moved`)
	}
	table, _, _ = strings.Cut(table, "\n###")
	rows := regexp.MustCompile("(?m)^\\| `(\\w+)` \\| (.+) \\|$").FindAllStringSubmatch(table, -1)
	if len(rows) == 0 {
		t.Fatal("found no rows in the README stability table")
	}

	checker := newSnippetChecker(t)
	for _, row := range rows {
		pkg := row[1]
		for _, item := range strings.Split(row[2], ", ") {
			var decl string
			name, isFunc := strings.CutSuffix(item, "()")
			switch typ, method, isMethod := strings.Cut(name, "."); {
			case isMethod && isFunc:
				decl = "var _ = (*" + pkg + "." + typ + ")." + method
			case isFunc:
				decl = "var _ = " + pkg + "." + name
			default:
				decl = "var _ *" + pkg + "." + name
			}
			for _, problem := range checker.check(decl) {
				t.Errorf("README stability table, %s %s: %s", pkg, item, problem)
			}
		}
	}
}

// readmeSnippets returns README.md's ```go blocks.
func readmeSnippets(t *testing.T) []snippet {
	t.Helper()
	readme := readFile(t, "README.md")
	var out []snippet
	for _, loc := range regexp.MustCompile("(?s)```go\r?\n(.*?)```").FindAllStringSubmatchIndex(readme, -1) {
		line := strings.Count(readme[:loc[0]], "\n") + 1
		out = append(out, snippet{
			where: "README.md:" + strconv.Itoa(line),
			src:   readme[loc[2]:loc[3]],
		})
	}
	return out
}

// packageOverviewSnippets returns the code blocks of every package overview:
// each doc.go, and gedcom_api.go for the root package.
func packageOverviewSnippets(t *testing.T) []snippet {
	t.Helper()
	var out []snippet
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != "." && (strings.HasPrefix(d.Name(), ".") || d.Name() == "testdata") {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() != "doc.go" && path != "gedcom_api.go" {
			return nil
		}
		file, err := goparser.ParseFile(token.NewFileSet(), path, nil, goparser.ParseComments|goparser.PackageClauseOnly)
		if err != nil {
			return err
		}
		if file.Doc == nil {
			return nil
		}
		for i, block := range new(comment.Parser).Parse(file.Doc.Text()).Content {
			if code, ok := block.(*comment.Code); ok {
				out = append(out, snippet{where: path + " overview block " + strconv.Itoa(i), src: code.Text})
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// snippetExports maps each package a snippet may import to its compiled
// export data, which `go list -export` builds (or finds in the build cache).
// Type-checking against export data instead of source keeps the test fast.
var snippetExports = sync.OnceValues(func() (map[string]string, error) {
	args := []string{"list", "-export", "-deps", "-f", "{{if .Export}}{{.ImportPath}}={{.Export}}{{end}}", "./..."}
	args = append(args, "context", "errors", "fmt", "io", "log", "os", "testing", "time")
	out, err := exec.Command("go", args...).Output()
	if err != nil {
		return nil, fmt.Errorf("go list -export: %w", err)
	}
	exports := make(map[string]string)
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if path, file, ok := strings.Cut(line, "="); ok {
			exports[path] = file
		}
	}
	return exports, nil
})

// snippetChecker type-checks snippets against the module.
type snippetChecker struct {
	t    *testing.T
	fset *token.FileSet
	imp  types.Importer
	dir  string
}

func newSnippetChecker(t *testing.T) *snippetChecker {
	t.Helper()
	dir, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	exports, err := snippetExports()
	if err != nil {
		t.Fatal(err)
	}
	lookup := func(path string) (io.ReadCloser, error) {
		file, ok := exports[path]
		if !ok {
			return nil, fmt.Errorf("no export data for %q; add it to snippetExports", path)
		}
		return os.Open(file)
	}
	fset := token.NewFileSet()
	return &snippetChecker{t: t, fset: fset, imp: importer.ForCompiler(fset, "gc", lookup), dir: dir}
}

// check returns the type errors in src, other than unused variables and
// imports.
func (c *snippetChecker) check(src string) []string {
	c.t.Helper()
	file, err := c.parseSnippet(src)
	if err != nil {
		return []string{"does not parse: " + err.Error()}
	}
	fixtures, err := goparser.ParseFile(c.fset, filepath.Join(c.dir, "snippet_fixtures.go"),
		"package "+file.Name.Name+"\n"+snippetFixtures, 0)
	if err != nil {
		c.t.Fatalf("snippetFixtures does not parse: %v", err)
	}

	var problems []string
	conf := types.Config{
		Importer: c.imp,
		Error: func(err error) {
			if te, ok := err.(types.Error); ok && te.Soft && strings.Contains(te.Msg, "not used") {
				return
			}
			problems = append(problems, err.Error())
		},
	}
	_, _ = conf.Check(file.Name.Name, c.fset, []*ast.File{file, fixtures}, nil)
	return problems
}

// parseSnippet parses src as a file if it is one, as top-level declarations if
// it parses as those (adding snippetImports when it imports nothing), and
// otherwise as a function body.
func (c *snippetChecker) parseSnippet(src string) (*ast.File, error) {
	name := filepath.Join(c.dir, "snippet.go")
	if strings.HasPrefix(strings.TrimSpace(src), "package ") {
		return goparser.ParseFile(c.fset, name, src, 0)
	}
	if decls, err := goparser.ParseFile(token.NewFileSet(), name, "package snippet\n"+src, 0); err == nil {
		if len(decls.Imports) == 0 {
			src = snippetImports + src
		}
		return goparser.ParseFile(c.fset, name, "package snippet\n"+src, 0)
	}
	return goparser.ParseFile(c.fset, name, "package snippet\n"+snippetImports+"func _() {\n"+src+"\n}\n", 0)
}
