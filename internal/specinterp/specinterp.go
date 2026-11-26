// Package specinterp reads the specification interpretation register,
// docs/governance/policies/spec-interpretation.md: the policies the library
// picked where a GEDCOM specification offers a choice, is lenient or is
// silent, and the forms it allows that the library does not support, each
// under a stable ID such as SI-001.
//
// It is for tests only. Its own test holds every row's godoc symbols and
// tests to the source tree, and a test that finds the library departing from
// the specification looks the departure up with [Require], so an undocumented
// departure fails. No library package may import it.
package specinterp

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/cacack/gedcom-go/v3/internal/corpora"
)

// Path is the register's location, relative to the repository root.
const Path = "docs/governance/policies/spec-interpretation.md"

// Kinds of departure a row records.
const (
	KindChoice      = "choice"      // the specification allows more than one reading
	KindLenient     = "lenient"     // the library accepts what the specification forbids
	KindSilent      = "silent"      // the specification says nothing
	KindUnsupported = "unsupported" // the library rejects or misreads what the specification allows
)

// Row is one policy in the register.
type Row struct {
	ID            string
	Specification string
	Kind          string
	Policy        string
	// StatedIn names the exported symbol whose godoc states the policy, then
	// the symbol in each other package that applies it, as
	// "<package directory>.<Name>" or "<package directory>.<Type>.<Member>".
	StatedIn []string
	// EnforcedBy names the tests that hold every package to the policy, as
	// "<package directory>.<Test or Fuzz name>".
	EnforcedBy []string
}

// Register is the parsed register, in table order.
type Register struct {
	Rows []Row
	byID map[string]int
}

// Row returns the row with the given ID.
func (r *Register) Row(id string) (Row, bool) {
	i, ok := r.byID[id]
	if !ok {
		return Row{}, false
	}
	return r.Rows[i], true
}

var (
	idPattern = regexp.MustCompile(`^SI-\d{3}$`)
	// specLocation is what a Specification cell must cite: a 7.0 section on
	// gedcom.io or a 5.5.x page.
	specLocation = regexp.MustCompile(`FamilySearchGEDCOMv7\.html#|\bp\.\d+`)
	codeSpan     = regexp.MustCompile("`([^`]+)`")
)

// header is the register table's header row, cell by cell.
var header = []string{"ID", "Specification", "Kind", "Policy", "Stated in", "Enforced by"}

// Parse reads the register from r: the one Markdown table whose header row is
// [header]. Every row must have an ID unique in the table, a Specification
// citing a location, a known Kind, a Policy, and at least one symbol and one
// test, each written as a code span.
func Parse(r io.Reader) (*Register, error) {
	reg := &Register{byID: map[string]int{}}
	sc := bufio.NewScanner(r)
	inTable, n := false, 0
	for sc.Scan() {
		n++
		line := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(line, "|") {
			if inTable {
				break
			}
			continue
		}
		cells := splitRow(line)
		if !inTable {
			if slices.Equal(cells, header) {
				inTable = true
				if !sc.Scan() {
					return nil, errors.New("register table has no delimiter row")
				}
				n++
			}
			continue
		}
		row, err := parseRow(cells)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", n, err)
		}
		if _, dup := reg.byID[row.ID]; dup {
			return nil, fmt.Errorf("line %d: duplicate ID %s", n, row.ID)
		}
		reg.byID[row.ID] = len(reg.Rows)
		reg.Rows = append(reg.Rows, row)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if !inTable {
		return nil, fmt.Errorf("no table with header %q", strings.Join(header, " | "))
	}
	if len(reg.Rows) == 0 {
		return nil, errors.New("register table has no rows")
	}
	return reg, nil
}

func parseRow(cells []string) (Row, error) {
	if len(cells) != len(header) {
		return Row{}, fmt.Errorf("%d cells, want %d", len(cells), len(header))
	}
	row := Row{
		ID:            cells[0],
		Specification: cells[1],
		Kind:          cells[2],
		Policy:        cells[3],
		StatedIn:      codeSpans(cells[4]),
		EnforcedBy:    codeSpans(cells[5]),
	}
	switch {
	case !idPattern.MatchString(row.ID):
		return Row{}, fmt.Errorf("ID %q is not SI-NNN", row.ID)
	case !specLocation.MatchString(row.Specification):
		return Row{}, fmt.Errorf("%s: Specification cites no 7.0 section or 5.5.x page", row.ID)
	case !slices.Contains([]string{KindChoice, KindLenient, KindSilent, KindUnsupported}, row.Kind):
		return Row{}, fmt.Errorf("%s: Kind %q is not choice, lenient, silent or unsupported", row.ID, row.Kind)
	case row.Policy == "":
		return Row{}, fmt.Errorf("%s: no Policy", row.ID)
	case len(row.StatedIn) == 0:
		return Row{}, fmt.Errorf("%s: Stated in names no symbol", row.ID)
	case len(row.EnforcedBy) == 0:
		return Row{}, fmt.Errorf("%s: Enforced by names no test", row.ID)
	}
	return row, nil
}

// splitRow returns the trimmed cells of a Markdown table row. A "\|" is a
// literal "|" inside a cell, not a cell boundary.
func splitRow(line string) []string {
	line = strings.TrimSuffix(strings.TrimPrefix(line, "|"), "|")
	var cells []string
	var cell strings.Builder
	for i := 0; i < len(line); i++ {
		switch {
		case line[i] == '\\' && i+1 < len(line) && line[i+1] == '|':
			cell.WriteByte('|')
			i++
		case line[i] == '|':
			cells = append(cells, strings.TrimSpace(cell.String()))
			cell.Reset()
		default:
			cell.WriteByte(line[i])
		}
	}
	return append(cells, strings.TrimSpace(cell.String()))
}

func codeSpans(cell string) []string {
	var spans []string
	for _, m := range codeSpan.FindAllStringSubmatch(cell, -1) {
		spans = append(spans, m[1])
	}
	return spans
}

// Load parses the register from the repository the working directory is in.
func Load() (*Register, error) {
	root, err := corpora.RepoRoot()
	if err != nil {
		return nil, err
	}
	f, err := os.Open(filepath.Join(root, filepath.FromSlash(Path))) // #nosec G304 -- the register under the repository root
	if err != nil {
		return nil, err
	}
	defer f.Close()
	reg, err := Parse(f)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", Path, err)
	}
	return reg, nil
}

// Require returns the row with the given ID, failing tb if the register does
// not load or holds no such row. A test that allows a departure from the
// specification calls it with the row that documents the departure.
func Require(tb testing.TB, id string) Row {
	tb.Helper()
	reg, err := Load()
	if err != nil {
		tb.Fatal(err)
	}
	row, ok := reg.Row(id)
	if !ok {
		tb.Fatalf("%s has no row %s", Path, id)
	}
	return row
}
