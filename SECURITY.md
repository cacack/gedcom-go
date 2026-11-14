# Security Policy

## Supported Versions

Security fixes are released on the latest minor of the current major, and
backported to the last minor of the previous major from its maintenance
branch (`release-2.x`; see
[api-stability.md](docs/governance/policies/api-stability.md#cutting-the-last-minor-of-a-major-line)).

| Version | Supported          |
| ------- | ------------------ |
| 3.x (latest minor) | :white_check_mark: |
| 2.5.x   | :white_check_mark: security fixes only |
| < 2.5   | :x:                |

## Decoding Untrusted Input

What the library bounds on hostile input, what it does not, and how to size an
input limit for each API is in
[docs/guides/untrusted-input.md](docs/guides/untrusted-input.md). Memory that
grows with input within those documented multipliers is expected behaviour,
not a vulnerability; a panic, or memory beyond them, is.

## Reporting a Vulnerability

**Please do not open a public GitHub issue for security vulnerabilities.**

Report suspected vulnerabilities privately via GitHub's security advisory workflow:

- **Preferred:** [Open a private vulnerability report](https://github.com/cacack/gedcom-go/security/advisories/new)
- This uses GitHub's private vulnerability reporting feature and is the fastest way to reach a maintainer.

When reporting, please include:

1. A description of the vulnerability and its potential impact.
2. Steps to reproduce (ideally a minimal GEDCOM sample or code snippet).
3. The affected version or commit SHA.
4. Any proposed mitigation, if known.

## What to Expect

- **Acknowledgement** within 7 days of your report.
- **Triage and severity assessment** within 14 days.
- **Fix or mitigation plan** communicated back to you as soon as it is ready.
- **Coordinated disclosure** once a fix is available. We will credit reporters who wish to be named.

If you do not receive a response within 7 days, please follow up on the advisory thread.

## Scope

Security reports are welcomed for any code in this repository, including:

- Parsing logic in `parser/`, `decoder/`, `charset/`, `version/`
- Encoding logic in `encoder/`
- Archive handling in `gedzip/`
- Validation logic in `validator/`
- Any helper scripts in `scripts/`

Out of scope:

- Vulnerabilities in consuming applications that are not caused by this library.
- Issues in third-party genealogy software that merely happen to produce GEDCOM files we parse.

## Security Tooling

These scanners run automatically. Results, except gitleaks' and dependency
review's, are uploaded to the repository's Security tab.

| Scanner | Pull requests | Nightly | Weekly |
| ------- | ------------- | ------- | ------ |
| [CodeQL](https://github.com/cacack/gedcom-go/actions/workflows/codeql.yml) (static analysis) | yes | | yes |
| [govulncheck](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck) (Go vulnerability database) | yes | yes | |
| [gosec](https://github.com/securego/gosec) (Go security linter) | yes | yes | |
| [Trivy](https://trivy.dev/) (filesystem vulnerability scan) | yes | yes | |
| [osv-scanner](https://google.github.io/osv-scanner/) (OSV vulnerability scan) | yes | | |
| [Semgrep](https://semgrep.dev/) (rule-based SAST) | yes | | |
| [gitleaks](https://github.com/gitleaks/gitleaks) (secret detection) | yes | | |
| [Dependency Review](https://github.com/actions/dependency-review-action) (new vulnerable or disallowed dependencies) | yes | | |
| [OSSF Scorecard](https://github.com/cacack/gedcom-go/actions/workflows/scorecard.yml) (supply-chain posture) | | | yes |

Every pull-request scanner but Dependency Review also runs on pushes to
`main`, and all but CodeQL run on the `release-2.x` maintenance branch.
Scorecard also runs on pushes to `main`. The parser, charset readers, date
parser, decoder, encoder, converter and GEDZIP reader are fuzzed nightly.
