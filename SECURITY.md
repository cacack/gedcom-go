# Security Policy

## Supported Versions

`gedcom-go` is pre-1.0. Security fixes are applied to the latest published minor release. Once a 1.0 release is cut, this policy will be updated to cover multiple supported versions.

| Version | Supported          |
| ------- | ------------------ |
| Latest  | :white_check_mark: |
| Older   | :x:                |

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
- Validation logic in `validator/`
- Any helper scripts in `scripts/`

Out of scope:

- Vulnerabilities in consuming applications that are not caused by this library.
- Issues in third-party genealogy software that merely happen to produce GEDCOM files we parse.

## Security Tooling

This project runs automated security scans on every pull request and nightly:

- [CodeQL](https://github.com/cacack/gedcom-go/actions/workflows/codeql.yml) — static analysis
- [govulncheck](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck) — Go vulnerability database
- [gosec](https://github.com/securego/gosec) — Go security linter
- [Trivy](https://trivy.dev/) — filesystem vulnerability scanning
- [Semgrep](https://semgrep.dev/) — rule-based SAST
- [gitleaks](https://github.com/gitleaks/gitleaks) — secret detection
- [OSSF Scorecard](https://github.com/cacack/gedcom-go/actions/workflows/scorecard.yml) — supply-chain posture

Results are uploaded to the repository's Security tab.
