package main

import (
	"bufio"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

const fixHint = `Each incompatible change must either:
  - remove a symbol that carried a "// Deprecated:" paragraph in the baseline
    release (add the marker in a released minor before removing it), or
  - be listed in the allowlist with a migration-guide heading that explains
    the upgrade path (add the guide section, then the allowlist entry).`

// change is one incompatible change reported by `apidiff -m`, split at the
// first ": " — e.g. key "./gedcom.Address.Phone", message "removed".
type change struct {
	key     string
	message string
}

// allowEntry is one allowlist line: an apidiff key and a guide anchor.
type allowEntry struct {
	key    string
	target string
	line   int
}

// result holds what the gate found; any failure fails the gate.
type result struct {
	failures []string
	warnings []string
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("checkdeprecated", flag.ContinueOnError)
	fs.SetOutput(stderr)
	baselineDir := fs.String("baseline", "", "directory holding the baseline release's source tree")
	module := fs.String("module", "", "module path of the baseline release")
	allowlist := fs.String("allowlist", "", "allowlist file (see scripts/api-compat-allowlist.txt)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *baselineDir == "" || *module == "" || *allowlist == "" {
		fmt.Fprintln(stderr, "usage: checkdeprecated -baseline DIR -module PATH -allowlist FILE < apidiff-output")
		return 2
	}

	changes, err := parseChanges(stdin)
	if err != nil {
		fmt.Fprintf(stderr, "✗ %v\n", err)
		return 1
	}
	entries, err := readAllowlist(*allowlist)
	if err != nil {
		fmt.Fprintf(stderr, "✗ %v\n", err)
		return 1
	}
	res, err := check(changes, entries, *allowlist, &baseline{dir: *baselineDir, module: *module})
	if err != nil {
		fmt.Fprintf(stderr, "✗ %v\n", err)
		return 1
	}

	for _, w := range res.warnings {
		fmt.Fprintf(stdout, "⚠ %s\n", w)
	}
	if len(res.failures) > 0 {
		fmt.Fprintln(stderr, "✗ Deprecation gate failed:")
		for _, f := range res.failures {
			fmt.Fprintf(stderr, "  - %s\n", f)
		}
		fmt.Fprintf(stderr, "\n%s\n", fixHint)
		return 1
	}
	fmt.Fprintln(stdout, "✓ Every incompatible change was deprecated in the baseline or is allowlisted")
	return 0
}

// check applies the gate to changes. Allowlist entries whose anchor does not
// resolve are failures; entries matching no change are warnings.
func check(changes []change, entries []allowEntry, allowlistPath string, b *baseline) (result, error) {
	var res result
	listed := make(map[string]bool, len(entries))
	for _, e := range entries {
		listed[e.key] = true
		if err := resolveAnchor(e.target); err != nil {
			res.failures = append(res.failures,
				fmt.Sprintf("%s:%d: %s: %v", allowlistPath, e.line, e.key, err))
		}
	}

	used := make(map[string]bool)
	for _, c := range changes {
		if listed[c.key] {
			used[c.key] = true
			continue
		}
		if c.message == "removed" {
			dep, err := b.deprecated(c.key)
			if err != nil {
				return res, err
			}
			if dep {
				continue
			}
			res.failures = append(res.failures,
				fmt.Sprintf("%s: removed without a Deprecated: marker in the baseline, and not allowlisted", c.key))
			continue
		}
		res.failures = append(res.failures, fmt.Sprintf("%s: %s — not allowlisted", c.key, c.message))
	}

	for _, e := range entries {
		if !used[e.key] {
			res.warnings = append(res.warnings, fmt.Sprintf(
				"%s:%d: allowlist entry %s matches no incompatible change", allowlistPath, e.line, e.key))
		}
	}
	return res, nil
}

// parseChanges reads the "Incompatible changes:" section of `apidiff -m`
// output. Any other line inside that section is an error, so a format change
// in apidiff fails loudly instead of letting changes through.
func parseChanges(r io.Reader) ([]change, error) {
	var changes []change
	inIncompatible := false
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := sc.Text()
		switch line {
		case "Incompatible changes:":
			inIncompatible = true
			continue
		case "Compatible changes:":
			inIncompatible = false
			continue
		}
		if !inIncompatible {
			continue
		}
		key, message, ok := strings.Cut(strings.TrimPrefix(line, "- "), ": ")
		if !strings.HasPrefix(line, "- ") || !ok {
			return nil, fmt.Errorf("unrecognised apidiff line: %q", line)
		}
		changes = append(changes, change{key: key, message: message})
	}
	return changes, sc.Err()
}

// readAllowlist parses the allowlist: one "<apidiff key> <guide.md#anchor>"
// pair per line; blank lines and lines starting with # are ignored.
func readAllowlist(path string) ([]allowEntry, error) {
	f, err := os.Open(path) // #nosec G304 -- path is the repo's own allowlist, passed by scripts/check-api-compat.sh
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []allowEntry
	sc := bufio.NewScanner(f)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 {
			return nil, fmt.Errorf("%s:%d: want \"<symbol> <guide.md#anchor>\", got %q", path, n, line)
		}
		entries = append(entries, allowEntry{key: fields[0], target: fields[1], line: n})
	}
	return entries, sc.Err()
}

// resolveAnchor checks that target ("path/to/guide.md#anchor", relative to
// the working directory) names an existing file with a matching heading.
func resolveAnchor(target string) error {
	file, anchor, ok := strings.Cut(target, "#")
	if !ok || file == "" || anchor == "" {
		return fmt.Errorf("target %q is not of the form <guide.md#anchor>", target)
	}
	f, err := os.Open(file) // #nosec G304 -- path comes from the repo's own allowlist
	if err != nil {
		return fmt.Errorf("migration guide: %w", err)
	}
	defer f.Close()

	slugs, err := headingSlugs(f)
	if err != nil {
		return err
	}
	if !slugs[anchor] {
		return fmt.Errorf("no heading in %s has anchor #%s", file, anchor)
	}
	return nil
}

var headingRE = regexp.MustCompile(`^#{1,6}\s+(.*)$`)

// headingSlugs returns the GitHub-style anchors of the Markdown headings in
// r, skipping fenced code blocks.
func headingSlugs(r io.Reader) (map[string]bool, error) {
	slugs := make(map[string]bool)
	inFence := false
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := sc.Text()
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		if m := headingRE.FindStringSubmatch(line); m != nil {
			slugs[slugify(m[1])] = true
		}
	}
	return slugs, sc.Err()
}

// slugify turns heading text into a GitHub-style anchor: lowercase, keep
// letters, digits, hyphens and underscores, turn spaces into hyphens, and
// drop everything else.
func slugify(heading string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(heading)) {
		switch {
		case r == ' ':
			b.WriteRune('-')
		case unicode.IsLetter(r), unicode.IsDigit(r), r == '-', r == '_':
			b.WriteRune(r)
		}
	}
	return b.String()
}

// baseline answers whether a symbol in the baseline source tree was
// deprecated. Parsed packages are cached by directory.
type baseline struct {
	dir    string
	module string
	pkgs   map[string][]*ast.File
}

// deprecated reports whether the symbol named by an apidiff key carried a
// deprecation paragraph. Keys take the forms apidiff prints:
//
//	Sym, T.Member         symbol in the module's root package
//	./pkg.Sym             package-level type, func, const or var
//	./pkg.T.Member        field, interface method or value-receiver method
//	./pkg.(*T).Method     pointer-receiver method
//	package <import path> a whole package
func (b *baseline) deprecated(key string) (bool, error) {
	if importPath, ok := strings.CutPrefix(key, "package "); ok {
		rel, ok := strings.CutPrefix(importPath, b.module)
		if !ok || (rel != "" && !strings.HasPrefix(rel, "/")) {
			return false, nil
		}
		files, err := b.files(filepath.Join(b.dir, filepath.FromSlash(rel)))
		if err != nil {
			return false, err
		}
		for _, f := range files {
			if isDeprecated(f.Doc) {
				return true, nil
			}
		}
		return false, nil
	}

	dir, sym := b.dir, key
	if strings.HasPrefix(key, "./") {
		slash := strings.LastIndex(key, "/")
		dot := strings.Index(key[slash:], ".")
		if dot < 0 {
			return false, fmt.Errorf("unrecognised apidiff symbol %q", key)
		}
		dir = filepath.Join(b.dir, filepath.FromSlash(key[:slash+dot]))
		sym = key[slash+dot+1:]
	}
	files, err := b.files(dir)
	if err != nil {
		return false, err
	}

	typ, member := splitSymbol(sym)
	if member == "" {
		return topLevelDeprecated(files, typ), nil
	}
	return memberDeprecated(files, typ, member), nil
}

// files parses the non-test Go files in dir, with comments.
func (b *baseline) files(dir string) ([]*ast.File, error) {
	if files, ok := b.pkgs[dir]; ok {
		return files, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("baseline package: %w", err)
	}
	fset := token.NewFileSet()
	var files []*ast.File
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, parser.ParseComments)
		if err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	if b.pkgs == nil {
		b.pkgs = make(map[string][]*ast.File)
	}
	b.pkgs[dir] = files
	return files, nil
}

// splitSymbol splits "T", "T.M" or "(*T).M" into type (or top-level name)
// and member.
func splitSymbol(sym string) (typ, member string) {
	if rest, ok := strings.CutPrefix(sym, "(*"); ok {
		typ, member, _ = strings.Cut(rest, ").")
		return typ, member
	}
	typ, member, _ = strings.Cut(sym, ".")
	return typ, member
}

// topLevelDeprecated reports whether the package-level name is deprecated,
// counting the doc of a grouped declaration as applying to every spec in it.
func topLevelDeprecated(files []*ast.File, name string) bool {
	for _, f := range files {
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if d.Recv == nil && d.Name.Name == name {
					return isDeprecated(d.Doc)
				}
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						if s.Name.Name == name {
							return isDeprecated(d.Doc, s.Doc, s.Comment)
						}
					case *ast.ValueSpec:
						for _, n := range s.Names {
							if n.Name == name {
								return isDeprecated(d.Doc, s.Doc, s.Comment)
							}
						}
					}
				}
			}
		}
	}
	return false
}

// memberDeprecated reports whether member of type typ — a method, struct
// field or interface method — is deprecated. A deprecated type covers its
// members. A member not declared on the type itself (promoted through
// embedding) counts as deprecated when the type has a deprecated embedded
// field, since apidiff reports promoted members alongside the embed.
func memberDeprecated(files []*ast.File, typ, member string) bool {
	if topLevelDeprecated(files, typ) {
		return true
	}
	if doc, ok := findMethod(files, typ, member); ok {
		return isDeprecated(doc)
	}
	embedDeprecated := false
	for _, field := range typeFields(files, typ) {
		if len(field.Names) == 0 {
			if typeName(field.Type) == member {
				return isDeprecated(field.Doc, field.Comment)
			}
			embedDeprecated = embedDeprecated || isDeprecated(field.Doc, field.Comment)
		}
		for _, n := range field.Names {
			if n.Name == member {
				return isDeprecated(field.Doc, field.Comment)
			}
		}
	}
	return embedDeprecated
}

// findMethod returns the doc of the method named member declared with
// receiver type typ (or *typ).
func findMethod(files []*ast.File, typ, member string) (*ast.CommentGroup, bool) {
	for _, f := range files {
		for _, decl := range f.Decls {
			d, ok := decl.(*ast.FuncDecl)
			if ok && d.Recv != nil && d.Name.Name == member && len(d.Recv.List) == 1 &&
				typeName(d.Recv.List[0].Type) == typ {
				return d.Doc, true
			}
		}
	}
	return nil, false
}

// typeFields returns the struct fields or interface methods of type typ.
func typeFields(files []*ast.File, typ string) []*ast.Field {
	for _, f := range files {
		for _, decl := range f.Decls {
			d, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range d.Specs {
				if ts, ok := spec.(*ast.TypeSpec); ok && ts.Name.Name == typ {
					return fieldList(ts.Type)
				}
			}
		}
	}
	return nil
}

// fieldList returns the fields of a struct or the methods of an interface.
func fieldList(expr ast.Expr) []*ast.Field {
	switch t := expr.(type) {
	case *ast.StructType:
		return t.Fields.List
	case *ast.InterfaceType:
		return t.Methods.List
	}
	return nil
}

// typeName returns the bare name of a receiver or embedded type expression:
// T, *T, pkg.T, T[P] all yield T.
func typeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return typeName(t.X)
	case *ast.SelectorExpr:
		return t.Sel.Name
	case *ast.IndexExpr:
		return typeName(t.X)
	case *ast.IndexListExpr:
		return typeName(t.X)
	}
	return ""
}

// isDeprecated reports whether any of the doc comments has a line starting
// with "Deprecated:".
func isDeprecated(groups ...*ast.CommentGroup) bool {
	for _, g := range groups {
		if g == nil {
			continue
		}
		for _, line := range strings.Split(g.Text(), "\n") {
			if strings.HasPrefix(line, "Deprecated:") {
				return true
			}
		}
	}
	return false
}
