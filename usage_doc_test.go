package gedcomgo

// USAGE.md's examples are fragments, not programs, so they cannot be compiled
// as they stand. What drifts in them is the API they name: a renamed function,
// or a struct literal setting a field that never existed (#405). This checks
// every pkg.Name reference to one of this module's packages, and the top-level
// field keys of each pkg.Type{...} literal whose type is a struct declared in
// that package (directly or through a same-package alias), against the package
// source.
//
// Limits, since there is no type information:
//   - selectors on values (x.Field, x.Method()) are not checked;
//   - keys of nested literals, and literals of non-struct types, are skipped;
//   - a local identifier named like a package (version := ...; version.Foo) is
//     taken for the package and flagged: rename the variable in USAGE.md;
//   - only blocks fenced exactly as ```go are scanned.

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// usageDocPackages are the module packages USAGE.md's examples may name.
var usageDocPackages = []string{
	"charset", "converter", "decoder", "encoder", "gedcom", "merge", "parser", "validator", "version",
}

// packageAPI is a package's exported top-level names, and the field names of
// each of its struct types.
type packageAPI struct {
	names  map[string]bool
	fields map[string]map[string]bool
}

func TestUsageDocNamesExist(t *testing.T) {
	apis := loadUsageDocAPIs(t)
	blocks := regexp.MustCompile("(?s)```go\n(.*?)```").FindAllStringSubmatch(readFile(t, "USAGE.md"), -1)
	if len(blocks) == 0 {
		t.Fatal("USAGE.md has no ```go blocks; update this test if the format changed")
	}
	for _, block := range blocks {
		for _, problem := range checkUsageSource(block[1], apis) {
			t.Error("USAGE.md " + problem)
		}
	}
}

// TestCheckUsageSource shows the check catches each kind of drift, and does
// not flag what it should not.
func TestCheckUsageSource(t *testing.T) {
	apis := loadUsageDocAPIs(t)
	tests := []struct {
		name string
		src  string
		want []string
	}{
		{"valid", "opts := &decoder.DecodeOptions{Context: ctx, StrictMode: true}\ndoc, err := decoder.DecodeWithOptions(f, opts)", nil},
		{"missing name", "decoder.DecodeFast(f)", []string{"references decoder.DecodeFast, which does not exist"}},
		{"bad field", "decoder.DecodeOptions{MaxLineLength: 10}", []string{"sets decoder.DecodeOptions{MaxLineLength: ...}, which is not a field"}},
		{"bad field after newline", "decoder.DecodeOptions{\n\tContext: ctx,\n\tBogus: 1,\n}", []string{"sets decoder.DecodeOptions{Bogus: ...}, which is not a field"}},
		{"nested literal keys skipped", "decoder.DecodeOptions{Context: f(struct{ Bogus int }{Bogus: 1})}", nil},
		{"alias resolved", "validator.ValidateOptions{Bogus: 1}", []string{"sets validator.ValidateOptions{Bogus: ...}, which is not a field"}},
		{"chained selector skipped", "x.decoder.Bogus", nil},
		{"other package skipped", "fmt.Bogus()", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkUsageSource(tt.src, apis)
			if strings.Join(got, "\n") != strings.Join(tt.want, "\n") {
				t.Errorf("checkUsageSource(%q) = %q, want %q", tt.src, got, tt.want)
			}
		})
	}
}

func loadUsageDocAPIs(t *testing.T) map[string]packageAPI {
	t.Helper()
	apis := make(map[string]packageAPI, len(usageDocPackages))
	for _, pkg := range usageDocPackages {
		apis[pkg] = loadPackageAPI(t, pkg)
	}
	return apis
}

// checkUsageSource returns a description of each reference in a code fragment
// to API that apis does not have.
func checkUsageSource(src string, apis map[string]packageAPI) []string {
	var problems []string
	toks := scanGo(src)
	for i := 0; i+2 < len(toks); i++ {
		if toks[i].tok != token.IDENT || toks[i+1].tok != token.PERIOD || toks[i+2].tok != token.IDENT {
			continue
		}
		pkg := toks[i].lit
		api, ok := apis[pkg]
		if !ok || (i > 0 && toks[i-1].tok == token.PERIOD) {
			continue
		}
		name := toks[i+2].lit
		if !api.names[name] {
			problems = append(problems, fmt.Sprintf("references %s.%s, which does not exist", pkg, name))
			continue
		}
		fields, isStruct := api.fields[name]
		if !isStruct || i+3 >= len(toks) || toks[i+3].tok != token.LBRACE {
			continue
		}
		for _, key := range literalKeys(toks[i+3:]) {
			if !fields[key] {
				problems = append(problems, fmt.Sprintf("sets %s.%s{%s: ...}, which is not a field", pkg, name, key))
			}
		}
	}
	return problems
}

type goToken struct {
	tok token.Token
	lit string
}

// scanGo tokenizes a code fragment, ignoring errors: fragments need not parse.
func scanGo(src string) []goToken {
	var s scanner.Scanner
	fset := token.NewFileSet()
	s.Init(fset.AddFile("", -1, len(src)), []byte(src), func(token.Position, string) {}, 0)
	var toks []goToken
	for {
		_, tok, lit := s.Scan()
		if tok == token.EOF {
			return toks
		}
		toks = append(toks, goToken{tok, lit})
	}
}

// literalKeys returns the keys of a composite literal's top-level Key: value
// elements; toks starts at its opening brace.
func literalKeys(toks []goToken) []string {
	var keys []string
	depth := 0
	for i, tk := range toks {
		switch tk.tok {
		case token.LBRACE, token.LPAREN, token.LBRACK:
			depth++
		case token.RBRACE, token.RPAREN, token.RBRACK:
			depth--
			if depth == 0 {
				return keys
			}
		case token.IDENT:
			if depth == 1 && i+1 < len(toks) && toks[i+1].tok == token.COLON &&
				(toks[i-1].tok == token.LBRACE || toks[i-1].tok == token.COMMA || toks[i-1].tok == token.SEMICOLON) {
				keys = append(keys, tk.lit)
			}
		}
	}
	return keys
}

// loadPackageAPI parses a package directory's non-test sources.
func loadPackageAPI(t *testing.T, dir string) packageAPI {
	t.Helper()

	api := packageAPI{names: map[string]bool{}, fields: map[string]map[string]bool{}}
	paths, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil || len(paths) == 0 {
		t.Fatalf("no Go sources in %s: %v", dir, err)
	}
	aliases := map[string]string{}
	fset := token.NewFileSet()
	for _, path := range paths {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		file, err := parser.ParseFile(fset, path, src, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if d.Recv == nil {
					api.names[d.Name.Name] = true
				}
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						api.names[s.Name.Name] = true
						switch typ := s.Type.(type) {
						case *ast.StructType:
							api.fields[s.Name.Name] = structFields(typ)
						case *ast.Ident:
							if s.Assign.IsValid() {
								aliases[s.Name.Name] = typ.Name
							}
						}
					case *ast.ValueSpec:
						for _, n := range s.Names {
							api.names[n.Name] = true
						}
					}
				}
			}
		}
	}
	for alias, target := range aliases {
		if fields, ok := api.fields[target]; ok {
			api.fields[alias] = fields
		}
	}
	return api
}

func structFields(st *ast.StructType) map[string]bool {
	fields := map[string]bool{}
	for _, f := range st.Fields.List {
		for _, n := range f.Names {
			fields[n.Name] = true
		}
		if len(f.Names) == 0 { // embedded: the field is named after its type
			typ := f.Type
			if star, ok := typ.(*ast.StarExpr); ok {
				typ = star.X
			}
			switch e := typ.(type) {
			case *ast.Ident:
				fields[e.Name] = true
			case *ast.SelectorExpr:
				fields[e.Sel.Name] = true
			}
		}
	}
	return fields
}
