<!-- PR titles must NOT use conventional commit format (release-please reads both
     titles and commits). Good: "Add GEDCOM 7.0 header parsing". -->

## Description

<!-- What changed and why. -->

## Related Issue

<!-- Closes #N if this delivers the issue's acceptance criteria; Refs #N if partial. -->

## Breaking Changes

<!-- None, or what callers must change. See docs/governance/policies/api-stability.md. -->

## Checklist

<!-- Test items: authoritative wording lives in docs/testing.md, "Checklist for New Features". -->

- [ ] `make preflight` passes
- [ ] Each test for a bug fix or behaviour change was shown to fail without that change (N/A for docs and refactors)
- [ ] Assertions check content, not just counts
- [ ] Coverage ≥85% per package (a floor, not evidence of quality; see [docs/testing.md](https://github.com/cacack/gedcom-go/blob/main/docs/testing.md#what-coverage-measures))
- [ ] Documentation updated
