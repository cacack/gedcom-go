package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The fixtures in testdata are:
//   - baseline/: a fake baseline release tree (module example.com/probe/v3)
//   - apidiff.txt: real `apidiff -m` output for baseline/ against a copy with
//     the deprecated and undeprecated symbols removed or changed
//   - docs/guides/migration-test.md: a fake migration guide
//
// Tests chdir into testdata so guide paths resolve as they do from the repo root.

const fixtureModule = "example.com/probe/v3"

// guide is the anchor prefix of the fake migration guide.
const guide = "docs/guides/migration-test.md#"

// runGate runs the gate from testdata with the given allowlist and stdin.
func runGate(t *testing.T, allowlist, stdin string, extraArgs ...string) (code int, stdout, stderr string) {
	t.Helper()
	t.Chdir("testdata")
	path := filepath.Join(t.TempDir(), "allowlist.txt")
	if err := os.WriteFile(path, []byte(allowlist), 0o600); err != nil {
		t.Fatal(err)
	}
	args := append([]string{"-baseline", "baseline", "-module", fixtureModule, "-allowlist", path}, extraArgs...)
	var out, errOut bytes.Buffer
	code = run(args, strings.NewReader(stdin), &out, &errOut)
	return code, out.String(), errOut.String()
}

func incompatible(lines ...string) string {
	return "Incompatible changes:\n- " + strings.Join(lines, "\n- ") + "\n"
}

func TestRun(t *testing.T) {
	tests := []struct {
		name       string
		allowlist  string
		stdin      string
		wantCode   int
		wantStdout []string
		wantStderr []string
	}{
		{
			name:       "removed and deprecated passes",
			stdin:      incompatible("./gedcom.ProbeFunc: removed"),
			wantStdout: []string{"✓ Every incompatible change"},
		},
		{
			name:       "removed and undeprecated fails",
			stdin:      incompatible("./gedcom.PlainFunc: removed"),
			wantCode:   1,
			wantStderr: []string{"./gedcom.PlainFunc: removed without a Deprecated: marker", "// Deprecated:", "allowlist"},
		},
		{
			name: "deprecated field and methods pass",
			stdin: incompatible(
				"./gedcom.ProbeType.Gone: removed",
				"./gedcom.ProbeType.Legacy: removed",
				"./gedcom.(*ProbeType).ProbeMethod: removed",
				"./gedcom.ProbeIface.Gone: removed",
			),
		},
		{
			name:      "retype allowlisted passes",
			allowlist: "./gedcom.ProbeType.Retyped " + guide + "probetyperetyped-is-now-int\n",
			stdin:     incompatible("./gedcom.ProbeType.Retyped: changed from string to int"),
		},
		{
			name:       "retype not allowlisted fails",
			stdin:      incompatible("./gedcom.ProbeType.Retyped: changed from string to int"),
			wantCode:   1,
			wantStderr: []string{"./gedcom.ProbeType.Retyped: changed from string to int — not allowlisted"},
		},
		{
			name:      "undeprecated removal allowlisted passes",
			allowlist: "./gedcom.PlainFunc " + guide + "addressphone-email-and-website-in-detail\n",
			stdin:     incompatible("./gedcom.PlainFunc: removed"),
		},
		{
			name:       "bad anchor fails",
			allowlist:  "./gedcom.ProbeType.Retyped " + guide + "no-such-heading\n",
			stdin:      incompatible("./gedcom.ProbeType.Retyped: changed from string to int"),
			wantCode:   1,
			wantStderr: []string{"allowlist.txt:1: ./gedcom.ProbeType.Retyped: no heading in docs/guides/migration-test.md has anchor #no-such-heading"},
		},
		{
			name:       "heading inside a code fence does not count",
			allowlist:  "./gedcom.ProbeType.Retyped " + guide + "not-a-heading\n",
			stdin:      incompatible("./gedcom.ProbeType.Retyped: changed from string to int"),
			wantCode:   1,
			wantStderr: []string{"has anchor #not-a-heading"},
		},
		{
			name:       "missing guide file fails",
			allowlist:  "./gedcom.ProbeType.Retyped docs/guides/missing.md#retypes\n",
			stdin:      incompatible("./gedcom.ProbeType.Retyped: changed from string to int"),
			wantCode:   1,
			wantStderr: []string{"migration guide: open docs/guides/missing.md"},
		},
		{
			name:       "target without anchor fails",
			allowlist:  "./gedcom.ProbeType.Retyped docs/guides/migration-test.md\n",
			stdin:      incompatible("./gedcom.ProbeType.Retyped: changed from string to int"),
			wantCode:   1,
			wantStderr: []string{"is not of the form <guide.md#anchor>"},
		},
		{
			name:       "unused entry warns",
			allowlist:  "./gedcom.Unused " + guide + "retypes\n",
			stdin:      incompatible("./gedcom.ProbeFunc: removed"),
			wantStdout: []string{"⚠ ", "allowlist.txt:1: allowlist entry ./gedcom.Unused matches no incompatible change", "✓"},
		},
		{
			name: "comments and blank lines are ignored",
			allowlist: "# header comment\n\n   # indented comment\n" +
				"./gedcom.ProbeType.Retyped   " + guide + "retypes\n\n",
			stdin: incompatible("./gedcom.ProbeType.Retyped: changed from string to int"),
		},
		{
			name:       "malformed allowlist line fails",
			allowlist:  "# ok\n./gedcom.ProbeType.Retyped\n",
			stdin:      incompatible("./gedcom.ProbeType.Retyped: changed from string to int"),
			wantCode:   1,
			wantStderr: []string{`allowlist.txt:2: want "<symbol> <guide.md#anchor>"`},
		},
		{
			name:       "unrecognised apidiff line fails",
			stdin:      "Incompatible changes:\nsomething new\n",
			wantCode:   1,
			wantStderr: []string{`unrecognised apidiff line: "something new"`},
		},
		{
			name:  "compatible changes are ignored",
			stdin: incompatible("./gedcom.ProbeFunc: removed") + "Compatible changes:\n- ./gedcom.Added: added\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, stdout, stderr := runGate(t, tt.allowlist, tt.stdin)
			if code != tt.wantCode {
				t.Errorf("exit code = %d, want %d\nstdout: %s\nstderr: %s", code, tt.wantCode, stdout, stderr)
			}
			for _, want := range tt.wantStdout {
				if !strings.Contains(stdout, want) {
					t.Errorf("stdout missing %q:\n%s", want, stdout)
				}
			}
			for _, want := range tt.wantStderr {
				if !strings.Contains(stderr, want) {
					t.Errorf("stderr missing %q:\n%s", want, stderr)
				}
			}
			if len(tt.wantStdout) == 0 && strings.Contains(stdout, "⚠") {
				t.Errorf("unexpected warning:\n%s", stdout)
			}
		})
	}
}

// TestRunRealApidiffOutput feeds the captured apidiff output through the gate
// and checks exactly which changes it rejects.
func TestRunRealApidiffOutput(t *testing.T) {
	stdin, err := os.ReadFile("testdata/apidiff.txt")
	if err != nil {
		t.Fatal(err)
	}
	code, _, stderr := runGate(t, "", string(stdin))
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	var got []string
	for _, line := range strings.Split(stderr, "\n") {
		if key, _, ok := strings.Cut(strings.TrimPrefix(line, "  - "), ": "); ok && strings.HasPrefix(line, "  - ") {
			got = append(got, key)
		}
	}
	want := []string{
		"RootFunc",
		"./gedcom.Mentions",
		"./gedcom.OtherVar",
		"./gedcom.PlainFunc",
		"./gedcom.ProbeCmp.A",
		"./gedcom.ProbeCmp",
		"./gedcom.ProbeConst",
		"./gedcom.ProbeType.ProbeVal",
		"./gedcom.ProbeType.Retyped",
		"./gedcom.ProbeType.Undeprecated",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("rejected changes:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestDeprecated(t *testing.T) {
	b := &baseline{dir: "testdata/baseline", module: fixtureModule}
	tests := []struct {
		key  string
		want bool
	}{
		{"RootFunc", false},
		{"./a/b.X", true},
		{"./gedcom.ProbeFunc", true},
		{"./gedcom.PlainFunc", false},
		{"./gedcom.GroupedVar", true},
		{"./gedcom.OtherVar", false},
		{"./gedcom.OldA", true},
		{"./gedcom.Mentions", false},
		{"./gedcom.(*ProbeType).ProbeMethod", true},
		{"./gedcom.ProbeType.ProbeVal", false},
		{"./gedcom.ProbeType.Gone", true},
		{"./gedcom.ProbeType.Legacy", true},
		{"./gedcom.ProbeType.Undeprecated", false},
		{"./gedcom.ProbeIface.Gone", true},
		{"./gedcom.ProbeIface.M", false},
		{"./gedcom.Embedder.ProbeCmp", true},
		{"./gedcom.Embedder.A", true},
		{"./gedcom.ProbeType.NoSuchMember", false},
		{"./gedcom.NoSuchSymbol", false},
		{"package example.com/probe/v3/gone", true},
		{"package example.com/probe/v3/a/b", false},
		{"package example.com/probe/v30/gone", false},
		{"package example.org/other", false},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got, err := b.deprecated(tt.key)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("deprecated(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

func TestDeprecatedErrors(t *testing.T) {
	b := &baseline{dir: "testdata/baseline", module: fixtureModule}
	for _, key := range []string{"./missing.X", "package example.com/probe/v3/missing", "./gedcom"} {
		if _, err := b.deprecated(key); err == nil {
			t.Errorf("deprecated(%q): want error", key)
		}
	}
}

func TestRunErrors(t *testing.T) {
	t.Run("usage", func(t *testing.T) {
		var out, errOut bytes.Buffer
		if code := run(nil, strings.NewReader(""), &out, &errOut); code != 2 {
			t.Errorf("exit code = %d, want 2", code)
		}
		if !strings.Contains(errOut.String(), "usage:") {
			t.Errorf("stderr = %q, want usage", errOut.String())
		}
	})
	t.Run("bad flag", func(t *testing.T) {
		var out, errOut bytes.Buffer
		if code := run([]string{"-nope"}, strings.NewReader(""), &out, &errOut); code != 2 {
			t.Errorf("exit code = %d, want 2", code)
		}
	})
	t.Run("missing allowlist", func(t *testing.T) {
		var out, errOut bytes.Buffer
		args := []string{"-baseline", "testdata/baseline", "-module", fixtureModule, "-allowlist", "testdata/missing.txt"}
		if code := run(args, strings.NewReader(""), &out, &errOut); code != 1 {
			t.Errorf("exit code = %d, want 1", code)
		}
	})
	t.Run("missing baseline package", func(t *testing.T) {
		code, _, stderr := runGate(t, "", incompatible("./missing.X: removed"))
		if code != 1 || !strings.Contains(stderr, "baseline package:") {
			t.Errorf("exit code = %d, stderr = %q", code, stderr)
		}
	})
}

func TestSlugify(t *testing.T) {
	tests := []struct{ heading, want string }{
		{"Retypes", "retypes"},
		{"`Address.Phone`, `.Email` and `.Website` in detail", "addressphone-email-and-website-in-detail"},
		{"`SourceCitation.Quality` is now `*int`", "sourcecitationquality-is-now-int"},
		{"snake_case and kebab-case", "snake_case-and-kebab-case"},
		{"Ünïcode letters 2", "ünïcode-letters-2"},
	}
	for _, tt := range tests {
		if got := slugify(tt.heading); got != tt.want {
			t.Errorf("slugify(%q) = %q, want %q", tt.heading, got, tt.want)
		}
	}
}
