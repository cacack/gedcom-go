# v3.0.0 Exported-API Breaking-Change Audit

**Date:** 2026-08-24
**Prepared for:** v3.0.0
**Issue:** #479 — audit the exported API for breaking changes worth making while the v3 window is open
**Method:** six independent lens passes over the full exported surface, consolidated here
**Audited tree:** `1ad200eb20b6e410f05f28be2f9c7c97673f83a3` (`main`), compared against tag `v2.4.0`
**Downstream checked:** `github.com/cacack/my-family` @ `8865119`, then pinning `gedcom-go/v2 v2.3.1`

> **This is a point-in-time snapshot.** Every `file:line` citation below was read against the
> commit named above. `docs/archive/` is explicitly *not maintained* (CLAUDE.md), so these
> citations will drift as the code moves and nothing obliges anyone to update them. Treat a
> citation that does not match current code as **stale, not necessarily wrong** — re-derive
> before concluding a finding was fixed. The Part B decline records are the part meant to survive
> longest; their *reasoning* is durable even where their line numbers are not.

## Summary

| Metric | Count |
|---|---:|
| Exported symbols in the module | **1039** |
| Symbols walked by at least one lens pass | **1039** (arithmetic below) |
| Candidates raised by the six lens passes, after dedupe | **35** |
| **Part A — breaking and worth it** | **22** (21 code changes + 1 scheduling decision) |
| **Part B — breaking and not worth it** (declined, with reasons) | **24** decline records |
| **Part C — real but not breaking**, routed out of the v3 window | **36** entries covering **66** items |
| Already filed and not re-proposed | #472, #473, #476, #477 |
| Issues whose scope this audit changes | **#472** and **#473** |
| Issues that gate the v3 tag | **none** — #476 is not v3-gated |

**How the three lists relate to the 35 raised candidates.** The lists are dispositions, not a
partition of the raised set, and they do not sum to 35 for two deliberate reasons. First, **a
single raised candidate can split across lists**: BW-3 (`Event.Place`) is one raised candidate that
produces one Part A entry and two Part C entries, because two of its three steps are not breaking.
Second, **Part B is larger than the raised set it draws from**, because it also records candidates
the lens passes evaluated and rejected *in-pass* — those never reached this document as proposals,
but recording them is the point of the section: it is the durable answer to "was this ever
considered?" Four raised candidates were deduplicated before disposition (`WithHeaderTagComparison`
by passes 003 and 007; the `[]error` surface by 005 and 007; `Family.NumberOfChildren` by 003 and
004; `Association.Notes`/`SourceRepositoryLink.Notes` by 002 and 003, which is a scope amendment to
#473 rather than a new candidate).

### The four things a reader should take away

1. **The most urgent finding in the audit is not a breaking change.** `parser.RecordIterator`
   ships byte offsets that are wrong by the cumulative length of every level-0 line seen; seven
   public entry points reach the broken producer. It should be filed and fixed as a patch today
   and must not wait for v3. Second and third on that list — the `Header.Submitter` validator
   false positive that fires on *every* decoded 5.5/5.5.1 document, and `ParseCoordinate`
   accepting `NaN` — are also non-breaking.
2. **Two candidates protect the NON-NEGOTIABLE Lossless Representation principle.**
   `Event.Place` (the encoder gates `PLAC` on the legacy scalar, so a caller who sets only
   `PlaceDetail` gets no `PLAC` tag at all) and `encoder.EncodeOptions.PreserveUnknownTags`
   (whose zero value silently drops every custom tag with its subtree).
3. **The window's cheapest and most defensible work is honouring promises already published.**
   `Source.RepositoryRef`/`Source.Repository` say "until the next major release";
   `decoder.DecodeOptions.MaxNestingDepth` and `gedcom/testing.WithHeaderTagComparison` say
   "Will be removed in v3"; `validator.ValidatorConfig` says it will *not* be removed. v3 should
   honour all five as written, or `Deprecated:` stops meaning anything in this module.
4. **The declined list is the durable half of this document.** 24 candidates were considered
   against the CONSTITUTION evidence bar and rejected. An open major-version window is
   explicitly not evidence of need, and the report says so in each case so that the next person
   does not re-derive them.

---

## Coverage attestation

`inventory.md` counts **1039** exported symbols module-wide: exported top-level types, exported
fields of exported structs, exported methods on exported receivers, exported top-level funcs,
consts and vars. `*_test.go` and `examples/` (all `package main`) are excluded by design.

| Scope | Symbols | Pass(es) | Pass's own reconciliation |
|---|---:|---|---|
| `gedcom/` | 574 | 002, 003, 004 | 002: 50 types + 313 fields + 105 methods + 29 funcs + 76 consts + 1 var = 574 ✓ · 003: 11 raised + 30 declined + 13 already-claimed + 520 cleared = 574 ✓ · 004: 3 v3-gated + 11 behaviour-bug + 47 doc-only + 73 cleared = 134 (the methods+funcs subset) ✓ |
| `validator/` 200 + `charset/` 26 + `version/` 2 | 228 | 005 | 172 cleared + 56 named in a candidate = 228 ✓ |
| `decoder/` 43 + `encoder/` 28 + `parser/` 81 | 152 | 006 | 145 cleared + 7 raised = 152 ✓ |
| facade 27 + `merge/` 32 + `converter/` 13 + `gedcom/testing/` 13 | 85 | 007 | 79 cleared + 6 raised = 85 ✓ |
| **Module total** | **1039** | | **574 + 228 + 152 + 85 = 1039** ✓ |

**The arithmetic reconciles with no gap.** Three notes on how, stated so the claim is checkable
rather than asserted:

- **`gedcom/`'s 574 is covered twice over, not divided.** Passes 002 and 003 each walked all
  574 independently and each reconciled to 574 on its own. Pass 004 walked the 134-symbol
  methods-and-funcs subset in depth (plus the fields those methods read and write), confirming
  behaviour by running probe binaries against the working tree rather than reading doc comments.
  The three passes are redundant by design; the redundancy is what produced the disagreements
  resolved below, and those disagreements are signal about how marginal the affected candidates
  are.
- **`gedcom/testing/`'s 13 symbols are counted once**, in pass 007's 85. Pass 003 also reached
  `WithHeaderTagComparison` (its C9) because the issue text named it, and said explicitly that
  it was doing so from outside its own 574. That is one symbol reviewed by two passes, not one
  symbol counted twice — 003's reconciliation excludes it and 007's includes it.
- **No symbol is unaccounted for.** Every pass enumerated its cleared set by name or by
  per-type table, and the four scope groups are disjoint and exhaustive against
  `inventory.md`'s per-package counts.

**One honest limit on the evidence, not on the coverage.** Downstream call-site claims are
strong at *type* level (qualified `gedcom.X` references disambiguated by import block) and were
initially weak at *field* level, because a bare `.Field` selector carries no package qualifier.
Stage 001 recorded `Note.Continuation` and `Event.Place` as having zero downstream call sites;
**both were wrong**, caught by pass 003 and confirmed by re-reading my-family. Every field-level
downstream claim below has since been re-verified by tracing the receiver's declaration.
`downstream-usage.md` now carries the correction and the caveat.

---

## Cross-pass conflicts, resolved

Five candidates were reached by more than one pass with differing recommendations. Where two
passes contradict, the one citing code it read wins over the one reasoning from a doc comment.

### 1. #476 — two remedies, one answer

- **Pass 003:** removing `Source.RepositoryRef` *dissolves* #476. The defect is one assignment,
  `gedcom/xrefwalk.go:389`, and it is the only write inside the shared `walk*` family that
  `Visit` traverses (every other assignment in that file — lines 103, 120, 163, 232–260 — sits
  inside `Apply`/`applyToRecords`/`remapXRefMap`, which are contractually allowed to mutate).
  Sequence the removal ahead and close #476 as resolved-by.
- **Pass 004:** #476 is **not breaking**. Moving the re-sync out of `walkSource` and into
  `Apply`'s own record loop (`applyToRecords`, `xrefwalk.go:127-138`) removes the write from the
  `Visit` path with no signature change. Ship as a patch. #476 must not be listed as a v3
  blocker.

Both read the same code; pass 004 additionally confirmed the data loss with a probe
(`doc.Subset()` — a documented read — overwrites a caller-set `RepositoryRef`). The two are
compatible, and both agree #476 does not gate the tag.

**Decision.** **#476 is not v3-gated.** Take pass 003's route: land **BW-1** (remove
`Source.RepositoryRef` and `Source.Repository`) and close #476 as resolved-by that change,
carrying its AC1 ("`Visit` does not modify the document on any path including `Subset`") and AC5
(the `exemptFromFixture` entry) across as verification steps. Do **not** build the `Apply`
post-pass first: it is machinery whose only purpose is to service a field being deleted in the
same release. Pass 004's patch is the **fallback**, to be taken only if BW-1 is declined, or if
v3 slips far enough that #476 needs an answer sooner.

### 2. The `[]error` validation surface — 005 C3 and 007 MCF-6

Pass 005 approached it from `validator/`: `Validator.Validate` (`validator/validator.go:233-246`)
reads **no** field of its own config, so `gedcomgo.ValidateWithOptions` honours none of six
options; its three rules (`BROKEN_XREF`, `MISSING_REQUIRED_FIELD`, `EMPTY_FAMILY`) are disjoint
from `ValidateAll`'s 23 and use bare string literals rather than exported `Code*` constants; and
it breaks ADR 0007 twice. Pass 007 approached it from the facade: `gedcom_api.go:191` documents
the ignoring rather than fixing it, and asked whether the project intends to retire the surface.

**Decision, made once.** **The project does not retire the `[]error` surface — it retires the
divergence.** `Validate` is the showcased quick start (`gedcom_api.go:26-30`); removing it is a
worse API, not a cleaner one. Instead: reimplement `Validate` over `ValidateAll` (`Issue` already
implements `error` at `validator/issue.go:172`), port the three orphan rules in with real `Code*`
constants, remove `ValidationError` (the breaking half), make `ValidateWithOptions` honour
`MaxErrors` and `SkipRules`, and delete the "legacy" wording from `validator/doc.go:9` and
`gedcom_api.go:190`. Filed as **BW-20**, owned by `validator/`, with MCF-6 folded in as its
facade half. Pass 005's preferred option; pass 007's open question answered.

### 3. `decoder.Severity` vs `validator.Severity` — routed by both, owned by neither

`decoder.Severity` (`decoder/diagnostics.go:18-45`) is a hand-maintained value-for-value
duplicate of `validator.Severity` (`validator/issue.go:15-42`) — same constants, same `String()`
— kept apart deliberately to dodge an import cycle, with the reason recorded at
`decoder/diagnostics.go:16-17`.

**Decision: declined as a v3 candidate** (see DEC-19). The duplication is deliberate and
documented; the cost is one mirror of three constants; unifying it needs a new shared package,
which is a larger structural change with no evidence of need. No downstream site is confused —
my-family uses `decoder.SeverityError` (`internal/gedcom/importer.go:363`) and
`validator.Severity*` (`internal/query/validation_service.go:200`), each correctly and
explicitly. **Route:** a one-line cross-reference comment on each enum pointing at its twin, and
a note that the two must be renumbered together if either is ever renumbered. Doc-only, patch.
This also locks pass 005's D4: `validator.Severity`'s zero value stays `SeverityError`.

### 4. `encoder.EncodeOptions.PreserveUnknownTags` — 006 E-1 and 007 MCF-3

No real conflict: pass 007 explicitly says the encoder's instance is "strictly worse" than the
converter's — the encoder field *does* destroy data, the converter's only affects a report — and
routes it to pass 006. **Pass 006's E-1 owns it.** The two halves split cleanly:

- **E-1a is non-breaking** (`LineEnding == "" → "\n"`, mirroring the guard `MaxLineLength`
  already has at `encoder/options.go:52`) and must not wait for v3 → routed to Part C.
- **E-1b is v3** (invert `PreserveUnknownTags` → `DropUnknownTags` so the zero value means "lose
  nothing") → **BW-6**.

The converter's same-named bool is a separate candidate (**BW-11**). Taking both leaves the
module with zero same-named/opposite-meaning option pairs, which is the point.

### 5. `Family.NumberOfChildren` — three views, one of them misread as opposition

- **Pass 004 (V3-1):** remove the field, replace with a read-only
  `func (f *Family) NumberOfChildren() string` reading the NCHI entry in `Family.Attributes`.
- **Pass 003 (C5):** remove the field; an accessor is a separate additive decision that "should
  not gate removal".
- **Pass 002:** the field is `string`, so `""` is a correct absent marker.

Pass 002 is not defending the field. Its lens was zero-value semantics, and on that lens the
field is clean — `""` genuinely means absent. The defect lies on a different lens: the field is
the *second* store for a fact whose first store is richer, and pass 004 proved with a probe that
the doc comment's own stated remedy ("update the Attributes entry") **does not work on a decoded
family**, because `encoder/encoder.go:284` writes `record.Tags` verbatim and `familyToTags` is
never reached.

**Decision: pass 004's shape.** Remove the field, add the read-only accessor, and ship
pass 004's `Record.Tags` doc fix (DOC-6) in the same release — without it a caller who dutifully
edits `Attributes` still loses the write and the fix is dishonest. Recorded as **BW-5**. Pass
002's observation stands and is simply about a different property.

### Sixth, resolved without conflict: `Event.Tags` vs the general entity-`Tags` removal

**BW-12** removes `Event.Tags` while **DEC-10** declines removing entity-level `Tags` from the
other nine structs. That is not inconsistent. The decoder *aliases* each record entity's `Tags`
to `Record.Tags` (`decoder/entity.go:102, 890, 986, 1161, 1209, 1319, 1366, 1461`), so
`indi.Tags` is a live shorthand for real data. It never assigns `Event.Tags`
(`decoder/entity.go:640-676`) and the encoder never reads it
(`encoder/entity_writer.go:910-950`) — `Event.Tags` is dead in a way the others are not.

---

# Part A — Breaking and worth it

Ordered by value against effort, in three tiers so the line can be cut anywhere. Each entry
carries: caller cost today (`file:line`), what breaks if deferred, downstream my-family impact
(real `file:line` or explicit "no call sites"), recommendation, effort.

**Reading the downstream column.** my-family pins `v2.3.1` (`go.mod:8`) while the latest release
is `v2.4.0`. For the five symbols added in v2.4.0, absence of a call site means "not yet
adopted". **No candidate below is affected**: every symbol here predates v2.3.1, so "no call
sites" is genuine evidence of non-use. CLAUDE.md's rule still applies — my-family's silence is
never evidence of *sufficiency*, because `Record.Tags` always lets it work around a gap.

## Tier 1 — the window exists for these

### BW-1 — Remove `Source.RepositoryRef` and `Source.Repository`

`gedcom/source.go:29`, `gedcom/source.go:37` · pass 003 C1+C2 · **new**

- **Caller cost today.** Both are documented as superseded by `SourceRepositoryLink` and
  scheduled for removal "until the next major release", but carry **no `Deprecated:` marker**, so
  godoc renders them as ordinary fields and no `staticcheck` SA1019 fires for anyone using them.
  Worse, `RepositoryRef` is a *mutable* alias: `gedcom/xrefwalk.go:389` overwrites it from
  `RepositoryLink.XRef` on every walk, so a caller-constructed value is blanked by a read-only
  `Visit` or `Document.Subset` — that is #476 in full. `Repository` creates a silent three-way
  precedence in the encoder (`encoder/entity_writer.go:459-470`): set `RepositoryLink` *and*
  `Repository` and the latter is ignored with nothing reporting the conflict.
- **Defer cost.** The field survives, and with it the re-sync line, the legacy-only fallback
  branch, `"RepositoryRef"` in `carrierFieldNames()` (`gedcom/xrefwalk_fixture_test.go:48`), the
  `exemptFromFixture` hole in the reachability guarantee (`:57-60`), the encoder's
  `case src.RepositoryRef != "":` (`encoder/entity_writer.go:462`), and the "Populate deprecated
  fields for backward compatibility" step (`decoder/entity.go:1006`). #476 then has to be
  implemented for real.
- **Downstream.** **No call sites.** my-family's only mention is a *comment* at
  `internal/gedcom/importer.go:810` explaining that it uses `RepositoryLink`, "which supersedes
  gedcom-go's flat RepositoryRef/Repository fields". The consumer migrated and said so in prose.
- **Recommendation.** **Remove both in v3, clean break, no deprecated alias.** The doc comments
  already promise removal at exactly this boundary; the removal deletes code in four packages
  rather than just two fields; and it dissolves #476. Reinstating either as a deprecated alias
  would recreate the precise legacy surface this sweep exists to remove. `InlineRepository`
  survives as the type of `SourceRepositoryLink.Inline`.
- **Effort.** `effort:low`. One change covering both fields — they were introduced together,
  deprecated together, and the encoder switch servicing them is one statement.
- **Overlap.** Dissolves **#476**. Not #472/#473/#477.

### BW-2 — Honour the two published "will be removed in v3" markers

`decoder/options.go:22-25` and `gedcom/testing/options.go:35` · passes 006 D-1, 003 C9, 007 MCF-4
· **new**

- **Caller cost today.** `decoder.DecodeOptions.MaxNestingDepth` is never read.
  `DefaultOptions()` (`decoder/options.go:72`) populates it with 100, making it look live; the
  real ceiling is enforced at `parser/parser.go:93`. Observed: `MaxNestingDepth: 2` decoding a
  document with a level-3 line succeeds, `err == nil`, record intact — a silent no-op.
  `gedcom/testing.WithHeaderTagComparison` is a literal no-op — its body is
  `func(*roundTripConfig) {}` and `roundTripConfig` is an empty struct
  (`gedcom/testing/options.go:12`) — so a caller who passes it believes they are enabling
  header-tag comparison that has been unconditional since #429.
- **Defer cost.** v3 ships contradicting its own published godoc, against
  `docs/governance/policies/api-stability.md`. Not honouring an explicit removal promise devalues
  every other `Deprecated:` marker in the module — which is exactly what makes the *next* audit
  necessary.
- **Downstream.** **No call sites** for either. `internal/gedcom/importer.go:337` takes
  `DefaultOptions()` and sets only `OnProgress` at `:345`; my-family imports `gedcom/testing` in
  zero files.
- **Recommendation.** **Remove both in v3, as one decision.** Present them together, per pass
  007's route, so v3 honours both promises or neither. Keep `testing.Option`, `applyOptions` and
  `roundTripConfig` — the variadic `opts ...Option` parameter is part of the
  `AssertRoundTrip`/`CheckRoundTrip` signatures and is a deliberate reserved extension point
  (`gedcom/testing/roundtrip.go:55-58`). And in the same decision, **honour the third marker as
  written**: `validator.ValidatorConfig` (`validator/validator.go:41`) promises it will *not* be
  removed, and v3 keeps it (see DEC-18).
- **Effort.** `effort:low`. The `gedcom/testing` half is the cheapest v3 change in the module —
  a compile error in test files only, surfacing on `go test`, fixed by deleting one argument.

### BW-3 — Remove `Event.Place` and `Attribute.Place`

`gedcom/event.go:112`, `gedcom/individual.go:198` · pass 003 C4 · **new**

- **Caller cost today — a live Lossless Representation violation.** The encoder gates the `PLAC`
  line on the *legacy scalar*, not the structured field: `encoder/entity_writer.go:829-830` tests
  `if d.place != ""`, and `placeToTags` (`:1088-1092`) writes the scalar onto the `PLAC` line,
  consulting `detail` only for `FORM` and `MAP`. A caller who follows the structured advice and
  sets **only** `Event.PlaceDetail = &PlaceDetail{Name: "Boston", Coordinates: …}` gets **no
  `PLAC` tag emitted at all** — name *and* coordinates dropped, silently, with no diagnostic. The
  legacy field is not merely redundant; it is load-bearing, and the modern field alone does not
  work. On decode the two are always identical: `decoder/entity.go:591-593` sets both from the
  same `tag.Value`, and `parsePlaceDetail` (`:730-732`) uses the same string.
- **Defer cost.** The silent-drop bug persists for a whole major cycle, and it is a violation of
  one of the two NON-NEGOTIABLE principles in CONSTITUTION.md. The mandatory dual write stays
  mandatory for every writer.
- **Downstream — the heaviest migration in the sweep.** **~13 hot-path sites on `Event.Place`**:
  reads at `internal/gedcom/importer.go:592, 613, 701, 1148, 1202, 1266, 1278, 1405, 1455`;
  writes at `internal/gedcom/exporter.go:660, 684, 874, 977`. Plus `Attribute.Place` at
  `exporter.go:945` and `LDSOrdinance.Place` at `exporter.go:1272`, `importer.go:1528, 1550`
  (the LDS field is untouched by this candidate — it has no `PlaceDetail` twin). **The dual write
  is visible in the consumer's source**: at `exporter.go:874-878` it assigns the same place-name
  value twice — once to the legacy `Event.Place` scalar and once to `PlaceDetail.Name` on a
  freshly constructed `PlaceDetail` — because the encoder gates on the scalar. (Pass 003 counted 7 sites; the fuller re-check in
  `downstream-usage.md` found 13. Both are corrections to stage 001, which recorded zero.)
- **Recommendation.** **Remove both in v3 — but only as step 3 of three, and the first two steps
  are not breaking and should ship first:**
  1. **(patch, urgent)** Fix `eventDetailToTags`/`placeToTags` so `PLAC` is emitted when *either*
     source carries a name. The gate at `encoder/entity_writer.go:828` is currently
     `if d.place != ""`; it must become the **OR of both sources**, not a replacement:

     ```go
     if d.place != "" || (d.placeDetail != nil && d.placeDetail.Name != "") {
     ```

     **Widen the gate; do not invert it.** Replacing the condition with a `placeDetail`-only test
     would silently stop emitting `PLAC` for every caller who sets only the legacy scalar — which
     is the more common shape on hand-built events today — reintroducing the exact class of
     silent encode loss this fix exists to remove, and doing it in a *patch* release rather than
     a version-gated one. With the OR form the change is genuinely *additive*: it emits a tag
     where none was emitted and changes no existing output. Routed to Part C as **RT-4**.
  2. **(v2.x, additive)** Add nil-safe `func (e *Event) PlaceName() string` and
     `func (a *Attribute) PlaceName() string`, so my-family can migrate its nine read sites
     *before* the break lands. Routed to Part C as **RT-13**.
  3. **(v3)** Remove `Event.Place` and `Attribute.Place`.

  Without step 1 the removal turns a silent bug into a total failure. Without step 2 the common
  read case gets worse than it is today, because a scalar with a safe zero value becomes a
  pointer deref. Do **not** bridge with a deprecated alias — that is what the field already is.
- **Effort.** `effort:medium` for the removal; `effort:low` for each of steps 1 and 2.
- **Overlap.** None of #472/#473/#476/#477. `PlaceDetail` becomes the sole place authority, which
  interacts with #472's note-split on that type — sequence after #472.

### BW-4 — Remove `EventOccupation`; add an `AttributeType` constant set

`gedcom/event.go:22` · pass 003 C6 · **new**

- **Caller cost today — proven, not hypothetical.** `EventOccupation EventType = "OCCU"` is
  documented as "represents an occupation event", but the decoder never produces it as an event:
  `decoder/entity.go:119-121` routes 24 tags into `indi.Events` and `OCCU` is not among them;
  `:140` routes `OCCU` (with `CAST`, `DSCR`, `EDUC`, `IDNO`, `NATI`, `SSN`, `TITL`, `RELI`,
  `NCHI`, `NMR`, `PROP`, `FACT`) into `indi.Attributes`. **No *positive* event decode ever
  produces `Event.Type == EventOccupation`** — `OCCU` is the only tag in the `EventType` constant
  set that the positive decode path never routes to `Events`.

  > **Corrected after panel review.** An earlier draft of this entry claimed *no* decoded
  > `Event.Type` can equal `EventOccupation`, and called that "proven, not hypothetical". That is
  > false. The GEDCOM 7 negative-assertion path at `decoder/entity.go:132` (and `:930` for
  > families) passes the `NO` tag's **value** into `parseEvent`, which sets
  > `Type: gedcom.EventType(eventTag)` at `:642` with no validation against the event-tag set. So
  > `1 NO OCCU` decodes to `&Event{Type: "OCCU", IsNegative: true}`, which does equal
  > `EventOccupation`. `parseEvent`'s own comment at `:647-649` acknowledges that `eventTag` comes
  > from the `NO` value. The reachable domain of `Event.Type` is therefore *wider* than the case
  > list, not narrower. The recommendation below is unchanged; the justification is narrower, and
  > the downstream claim below is corrected.

  (`EventResidence` is *not* affected; `RESI` is in the event list.) The root cause is an
  asymmetry: `Attribute.Type` is a bare `string` (`gedcom/individual.go:189`) with **zero**
  exported constants for its 13 tags, while `Event.Type` is a typed `EventType` — so a caller
  looking for "the constant for occupation" finds exactly one candidate and it is wrong.
- **Defer cost.** The dead-branch trap stays reachable by every new caller for a whole major
  cycle. Documenting it does not help: the existing doc comment is what caused the error.
- **Downstream — a live consumer bug.** my-family switches on it over `indi.Events` at
  `internal/gedcom/importer.go:989` and `:1260-1265`. Both branches are unreachable for an
  ordinary `1 OCCU …` line, which is the case that matters: the consumer's occupation import from
  GEDCOM is silently broken, and the misleading constant is why. The second site sits inside
  `extractAttributesFromIndividual`, whose entire purpose is to pull attributes out of an
  individual and which iterates the wrong collection.

  **Not dead code, though — corrected.** Both branches *do* fire for a GEDCOM 7 negative
  assertion (`1 NO OCCU`), via the `decoder/entity.go:132` path described above. So removing
  `EventOccupation` without a migration note would silently drop the consumer's handling of a
  state that is genuinely reachable. The v3 migration note must cover `IsNegative` events, not
  just redirect callers to the new attribute constant.
- **Recommendation.** **Remove `EventOccupation` in v3; add an `AttributeType` typed constant set**
  (`AttributeOccupation = "OCCU"` plus the other 12 tags from `decoder/entity.go:140`) and retype
  `Attribute.Type` to `AttributeType`. The additive half ships in v2.x (Part C, **RT-14**); only
  the removal needs the boundary. Removing the constant turns a silent behavioural bug into a
  compile error pointing straight at the broken switch — exactly what a major release is for.
  Do **not** keep `EventOccupation` as an alias: an alias of the wrong type *is* the bug.
  Note on blast radius, which is narrower than it looks: `AttributeType` as a *defined string
  type* means `attr.Type == "OCCU"` keeps compiling (untyped constants convert); only assignment
  from a `string` variable breaks. If the retype is still judged too wide, the minimum viable fix
  stands on its own — remove `EventOccupation`, add `AttributeOccupation` as an untyped string
  constant.
- **Effort.** `effort:medium` with the retype; `effort:low` without.

### BW-5 — `Family.NumberOfChildren`: field → read-only method

`gedcom/family.go:25` · passes 004 V3-1, 003 C5 · **covered in spirit by #477**, which introduced
the dual storage and documented rather than resolved it

- **Caller cost today.** A caller who does the obvious thing — `fam.NumberOfChildren = "4"` —
  gets a write that appears to succeed, survives in memory, and vanishes on encode with no error
  and no diagnostic (`encoder/entity_writer.go:369` suppresses the scalar whenever an `NCHI`
  entry exists in `Attributes`). The field's own doc comment tells the caller to "Update the
  Attributes entry, or clear it" instead — and **that remedy is false for the exact case the
  paragraph describes**. Pass 004 proved it with a probe: on a decoded family, editing
  `fam.Attributes[0].Value` still emits the original value, because `encoder/encoder.go:284`
  writes `record.Tags` verbatim and `familyToTags` is never reached. On a decoded family
  *neither* carrier is authoritative — the raw tag stream is.
- **Defer cost.** Rises monotonically. `NumberOfChildren` predates v2.3.1; `Family.Attributes`
  was added days ago (`fd7a09c`). Every quarter of delay is another quarter of new callers
  writing to the *older, more discoverable* of the two carriers and losing the write. There is no
  natural later moment.
- **Downstream.** **No call sites.** my-family reads family attributes through `fam.Attributes`
  generally. Predates v2.3.1, so the absence is genuine.
- **Recommendation.** **Remove the string field; add `func (f *Family) NumberOfChildren() string`**
  returning the `Value` of the `NCHI` entry in `Family.Attributes` (empty string when absent).
  This makes exactly one store authoritative, and it is the strictly richer one — it holds the
  value *and* the `NCHI` line's subordinates, so Lossless Representation is satisfied by
  construction rather than by convention. It deletes the only write path in the type that
  silently drops caller data, and converts that loss into a compile error, which no patch release
  can do. Three alternatives were weighed and rejected: documenting harder (the status quo, whose
  documentation is already wrong), making the scalar authoritative (a Lossless Representation
  violation — the scalar cannot carry the line's subordinates), and plain removal with no
  replacement (acceptable but worse: a legitimate NCHI *read* loses its one-liner).
- **Required companion, or the fix is dishonest.** Ship the `Record.Tags` doc fix (DOC-6, Part C
  **RT-24**) in the same release. The accessor fixes the *model*; the doc fix tells the truth
  about the *encoder*. Neither alone is sufficient.
- **The write side needs an answer too — added after panel review.** A read-only accessor removes
  the only *typed* way to set a child count on a hand-constructed `Family`. A caller authoring
  documents programmatically (rather than decoding them) currently writes
  `fam.NumberOfChildren = "4"`; after this change the only path is appending
  `&Attribute{Type: "NCHI", Value: "4"}` to `Family.Attributes` by hand, which requires knowing
  the raw GEDCOM tag name — strictly less discoverable than the field it replaces. That trade is
  defensible (the field it replaces was *broken*, not merely undiscoverable), but it must not go
  unstated. **The migration note must show the `Attributes` write explicitly**, and a
  `SetNumberOfChildren(string)` helper alongside the accessor is worth weighing during
  implementation — it would keep the write path typed and discoverable while still leaving
  `Attributes` the single authoritative store.
- **Effort.** `effort:low`.

### BW-6 — `encoder.EncodeOptions`: invert `PreserveUnknownTags` → `DropUnknownTags`

`encoder/options.go:38` · pass 006 E-1b, cross-referenced by 007 · **new**

- **Caller cost today.** The field's zero value is `false` while its own doc
  (`encoder/options.go:38-40`) says "Default: true" and `DefaultOptions()` sets true.
  `encoder/encoder.go:265` and `filterTags` read `false` as "drop each custom tag with its whole
  subtree, and drop a record whose type is a custom tag entirely." So `&encoder.EncodeOptions{}`
  silently discards vendor data. **On the hand-constructed-options path this violates Lossless
  Representation by default** — a contract violation, not a bug report. The struct is also
  inconsistent about what zero means: `MaxLineLength` guards its own zero (`options.go:52`), the
  other two do not, so a caller cannot form a general rule.
- **Defer cost.** Nothing in-tree breaks — every in-repo caller sets the field explicitly. The
  cost lands entirely on new callers, silently, producing plausible-looking non-empty output.
- **Downstream.** **3 hot-path sites**, all of which already work around it:
  `internal/gedcom/exporter.go:462`, `:486`, `:501` each spell out `LineEnding` and
  `PreserveUnknownTags` by hand while omitting `MaxLineLength`. The consumer had to discover
  which fields are unsafe to omit and encode that at three sites; my-family has already filed and
  fixed a live bug from this exact pattern, with a comment at `exporter.go:496-499` warning that
  a bare options literal would silently strip the tags. Under this recommendation those
  three literals just lose one term — a delete-only migration.
- **Recommendation.** **Invert to `DropUnknownTags` in v3**, so the zero value means "lose
  nothing" — the only default consistent with Lossless Representation, and the name stops
  disagreeing with its own zero value. `apidiff` flags it loudly; migration is mechanical.
  The sibling half, `LineEnding "" → "\n"`, is non-breaking and must ship sooner (Part C
  **RT-5**). Replacing `DefaultOptions()` with functional options was considered and declined
  (DEC-24).
- **Effort.** `effort:low`.

### BW-7 — Remove `Note.Continuation` and `Note.FullText()`

`gedcom/note.go:21`, `gedcom/note.go:37` · pass 003 C3 · **new**

- **Caller cost today — two distinct failures.** *Read side:* the field is **never populated on
  decode**. `decoder/entity.go:1334-1341` folds `CONT`/`CONC` into the `Text` builder, and there
  is no assignment to `note.Continuation` anywhere in `decoder/`, so a caller who reads it on a
  decoded document gets nil and concludes the note is single-line. *Write side:* it is a live
  second way to say the same thing, and the two conflict — `encoder/entity_writer.go:1370-1376`
  documents the hazard verbatim: a hand-built note with a multi-line `Text` **and** a non-empty
  `Continuation` "writes the body twice". There is no guard; the encoder emits both.
  `FullText()` exists only to bridge this and becomes exactly `return n.Text` once the field is
  gone.
- **Defer cost.** The double-emit trap stays armed for another major cycle, and the library keeps
  a `//nolint:staticcheck // SA1019` suppression (`encoder/entity_writer.go:645`) to silence its
  own linter about its own deprecated field. `Note` also stays structurally different from
  `SharedNote`, which has only `Text` — the asymmetry #439 set out to end.
- **Downstream.** **2 hot-path sites** — a real compile break, not a free deletion. Write:
  `internal/gedcom/exporter.go:1171` constructs a `gedcom.Note` populating both `Text` and
  `Continuation` from a hand-split body.
  Read: `internal/gedcom/importer.go:907` calls `note.FullText()`. (Stage 001 recorded zero call
  sites for `Continuation`; pass 003 corrected it.) **The migration improves the caller**:
  `toGedcomNoteRecord` (`exporter.go:1163`) already splits `n.Text` by hand, and after removal it
  sets `Entity: &gedcom.Note{Text: n.Text}` and lets the encoder's `entityRecordText`
  (`encoder/entity_writer.go:1377`) do the split — which additionally enforces `MaxLineLength` and
  guards against an embedded newline forging a GEDCOM record, neither of which my-family's hand
  split does. The read becomes `note.Text`.
- **Recommendation.** **Remove both in v3, as one change.** Removing `Continuation` without
  removing `FullText()` leaves a method that only restates a field. **Flag the two my-family
  sites explicitly in the v3 migration notes** with the replacement shown — this is the one
  candidate in the sweep with a non-trivial downstream migration, and the consumer should not
  discover it at `go get` time.
- **Effort.** `effort:low`, plus a release-note obligation.
- **Overlap.** None. #473 is the `Notes []string` set; `Continuation` is a different field on a
  different type with a different failure mode.

## Tier 2 — clear wins

### BW-8 — `version/`: remove `IsValidVersion`, drop `DetectVersion`'s always-nil error

`version/detect.go:131-138`, `version/detect.go:14-25` · pass 005 C5+C6 · **new**

- **Caller cost today.** `version.IsValidVersion` is the same switch over the same three
  constants as `gedcom.Version.IsValid` (`gedcom/version.go:22-29`) — a package-level copy of a
  method on the type it takes as its parameter, and its doc says "checks if a version *string*"
  while the parameter is `gedcom.Version`, a tell that it predates the method.
  `version.DetectVersion` has exactly two returns, both `return version, nil`; no error path
  exists or can, because the doc says failure returns `Version55` by design (ADR 0005). Every
  caller writes dead handling (`decoder/decoder.go:79`, `:213`), and the package doc example at
  `version/doc.go:10-13` shows `log.Fatal(err)` on an error that cannot fire, teaching callers a
  branch that is never taken.
- **Defer cost.** Nothing breaks; the dead `err` keeps propagating and the duplicate keeps
  drifting from its original.
- **Downstream.** **No call sites** — none of my-family's nine importing files imports our
  `version` package at all. In-module callers of `IsValidVersion`: zero. Both predate v2.3.1.
- **Recommendation.** **Both in v3.** Deletion and an arity change are each major-only. Together
  they halve a two-symbol package's surface at zero caller cost. Counter-argument considered and
  rejected: ADR 0005's "detect version mismatches" driver would want a signal, but a mismatch is
  not a *detection failure* — it is a `decoder.Diagnostic`, and `decoder/diagnostics.go` already
  carries that shape. Removing the error does not foreclose it.
- **Effort.** `effort:low`.

### BW-9 — `validator.Strictness`: renumber so the zero value is `Normal`

`validator/validator.go:32` · pass 005 C1 · **new**

- **Caller cost today.** `StrictnessRelaxed Strictness = iota` makes `Relaxed` the zero value,
  while three separate docs promise `Normal` (`validator/validator.go:56-57`,
  `validator/streaming.go:39-40`, and `:78` — "If opts is the zero value, default options are
  used", though `NewStreamingValidator` at `:79-88` stores `opts` verbatim and does no
  defaulting). So `validator.NewWithOptions(&validator.ValidateOptions{MaxErrors: 100})` silently
  discards **every warning and info**: `filterByStrictness` (`:463-475`) hits
  `case StrictnessRelaxed` and `CodeMissingSUBM` (`validator/header.go:38`), `CodeXRefTooLong`
  (`validator/xref.go:49`) and `CodeUnknownCustomTag` (`validator/tag_validator.go:130`) all
  vanish. `NewStreamingValidator(StreamingOptions{})` — the package's own doc example at
  `streaming.go:14` — has the same result.
- **Defer cost.** Nothing breaks; callers who do not set `Strictness` explicitly keep silently
  under-reporting.
- **Downstream.** **Insulated.** my-family sets `StrictnessStrict` explicitly at
  `internal/query/validation_service.go:77` and `:188`.
- **Recommendation.** **Renumber in v3.** Major-only *for a reason `apidiff` cannot see*:
  changing an untyped constant's value is not a signature change, so `make api-check` reports
  clean while any caller who serialized a `Strictness` int silently changes meaning. That
  invisibility is exactly why it cannot ship in a minor. The separable patch-only alternative —
  fix the three doc comments to admit the zero value is `Relaxed` — is worse, because the docs
  currently promise the *safer* behaviour and the code should meet them, not retreat.
- **Effort.** `effort:low`, plus a mandatory migration-note entry (see BW-10's process note).

### BW-10 — `decoder.DecodeOptions.StrictMode` must govern every decode entry point

`decoder/options.go:60` · pass 006 D-2 · **new**

- **Caller cost today.** `DecodeWithOptions` never reads the field: `decoder/decoder.go:63` calls
  `p.Parse()` unconditionally — the fail-on-first-error path, with no lenient branch. Only
  `DecodeWithDiagnostics` (`:113`) consults it, at **three** sites: `:150`, `:221` and `:229`.
  The third is easy to miss and is a **distinct behaviour**, not a duplicate of the second:
  `:229` gates `normalizeLevelJumps`, which clamps over-jumped levels and emits
  `CodeBadLevelJump`. Any fix that makes `DecodeWithOptions` honour `StrictMode` must cover all
  three, or it ships an entry point that is lenient about parse errors but still strict about
  indentation. Observed with a bad line 3:
  `DecodeWithOptions(StrictMode=false)` returns `doc=nil, err="line 3: invalid level number"`,
  while `DecodeWithDiagnostics(false)` returns a document and one diagnostic. **Four docs say the
  opposite:** the field's own comment (`options.go:33-38`, unscoped to any entry point),
  `options.go:30` naming "Decode/DecodeWithOptions", `docs/guides/decoding.md:208` putting
  `DecodeWithOptions` under a **Lenient Mode** column, and `FEATURES.md:44` listing `StrictMode`
  as a knob of `gedcomgo.DecodeWithOptions`. So one field means two things depending on which of
  three entry points you picked.
- **Defer cost.** A caller who reads the godoc, sets `StrictMode: false` for vendor tolerance and
  calls `DecodeWithOptions` gets a hard nil-document failure on the first malformed line — the
  exact scenario the option exists to prevent.
- **Downstream.** **No behaviour change.** my-family is already on the correct paths:
  `internal/gedcom/importer.go:348` uses `DecodeWithDiagnostics`, `internal/gedcom/exporter.go:489`
  uses `Decode` for a round-trip re-decode where strict is wanted.
- **Recommendation.** **Make `DecodeWithOptions` honour it in v3**, so one field means one thing;
  decide it together with BW-2's `MaxNestingDepth` removal, which gives `DecodeOptions` a coherent
  story for the first time. The doc half (correcting `decoding.md:208`, `FEATURES.md:44` and the
  field comment) is non-breaking and should ship now regardless (Part C **RT-25**).
  **Process requirement, and the answer to pass 006's open question:** this is a *semantic* break
  invisible to `make api-check`, as is BW-9. `docs/governance/policies/api-stability.md` needs a
  **"semantic breaks"** section as a home for changes `apidiff` cannot see, and both must be
  listed in the v3 migration notes. If that is judged too quiet, the loud alternative is deleting
  the field and splitting the entry point (`DecodeStrict`/`DecodeLenient`), which `apidiff` does
  flag — recorded as the fallback, not the recommendation.
- **Effort.** `effort:medium`.

### BW-11 — `converter.ConvertOptions`: split `PreserveUnknownTags`, fix the zero value

`converter/options.go:15` · pass 007 MCF-2 + MCF-3 · **new**

- **Caller cost today.** All eight uses of `ConvertOptions.PreserveUnknownTags` gate only
  `recordPreservedUnknownTags` (report-only, `converter/converter.go:367-402`) and the EXID
  mapping — it does **not** control preservation. Meanwhile
  `encoder.EncodeOptions.PreserveUnknownTags` (`encoder/options.go:38`) with the identical name
  really does drop tags and subtrees. Two same-named bools with opposite semantics in one module.
  Separately, `ConvertOptions{}` gives `Validate=false` and `PreserveUnknownTags=false` against
  doc comments saying "Default: true", because `converter/converter.go:27-29` only substitutes
  defaults for a **nil** pointer — a trap the project's own guide publishes at
  `docs/guides/converter.md:199-203`.
- **Defer cost.** The name collision stays; every reader of either field has to check which one
  they have. Scoped honestly: given the first defect this is *report and mapping fidelity*, not
  silent data loss, so it is not the same severity as BW-6.
- **Downstream.** **No exposure.** `internal/gedcom/exporter.go:493` passes `DefaultOptions()`.
- **Recommendation.** **Split into `ReportPreservedTags` and `MapEXIDToVendorTags` in v3**, and
  give the zero value the documented meaning. Fold MCF-3 into the same change. Sequence with
  BW-6: once the encoder's field is `DropUnknownTags` and the converter's is split, the module has
  zero same-named/opposite-meaning option pairs. Making the converter's flag actually drop tags
  was considered and declined (DEC-22) — it is forbidden by Lossless Representation.
- **Effort.** `effort:low`.

### BW-12 — Remove `Event.Tags`

`gedcom/event.go:194-195` · pass 004 V3-2 · **new**

- **Caller cost today.** The field is a dead end in both directions. Decode: `parseEvent`
  (`decoder/entity.go:640-676`) constructs `&gedcom.Event{Type: …}` and **never assigns `.Tags`**;
  unrecognised subtags go to `collector.addUnknownTag` (`:630`) and the raw lines stay on
  `Record.Tags` only. Encode: `eventToTags` (`encoder/entity_writer.go:910-950`) builds output
  purely from typed fields and **never reads `event.Tags`**. Yet `gedcom/clone.go:619` faithfully
  copies it and `gedcom/xrefwalk.go:471-473` walks it, so a pointer there *is* followed into a
  `Subset` closure and *is* rewritten by `Apply`. Probe result: an `Event` carrying
  `{Level: 2, Tag: "_EVCUST", Value: "e"}` encodes to `1 BIRT` with no subordinate — accepted,
  cloned, walked, and dropped.
- **Defer cost.** Nothing compiles differently. The field keeps looking like the place to put a
  custom event subtag and keeps not being one.
- **Downstream.** **No call sites.** `downstream-usage.md` records `Event` at 11 references across
  `internal/gedcom/{importer,exporter}.go` but nothing reaching `.Tags`; a targeted grep for an
  `Event.Tags` reference finds none.
- **Recommendation.** **Remove in v3.** Preferable to teaching `eventToTags` to emit it, which
  would change encoder output for anyone who populated the field via `Clone` — a behaviour change
  with no compile-time signal. Removal resolves the `Event`/`Attribute` asymmetry in the direction
  that leaves **one** place for raw tags (`Record.Tags`) rather than adding a second dead one
  (`Attribute.Tags`), which would double the wart. If event-level raw tags are ever wanted as a
  real feature, that is a separate additive change.
- **Effort.** `effort:low`.

### BW-13 — `SourceCitation.Quality int` → `*int`

`gedcom/source.go:108` · pass 002 C5 · **new**

- **Caller cost today.** `QUAY` is an enumeration whose **`0` is a meaningful value** — the
  field's own doc at `gedcom/source.go:103` says "0 = unreliable evidence or estimated data".
  Go's zero value for `int` is also `0`, and the encoder resolves the ambiguity in favour of
  "absent": `encoder/entity_writer.go:1004` emits the tag only `if cite.Quality > 0`. The decoder
  *does* parse it (`decoder/entity.go:422-427`), so `1 QUAY 0` decodes to `Quality: 0` and then
  silently disappears on re-encode through the entity path. A caller who explicitly asserts
  "unreliable evidence" cannot express it. Byte round-trip is unaffected (`Record.Tags` is
  authoritative when non-empty), so the loss shows on hand-built documents, the converter, and
  anything that clears `Tags`.
- **Defer cost.** Nothing breaks; the silent drop continues. The fix is a signature change, so it
  cannot land non-breakingly. Timing argument: `SourceCitation` **already** lost comparability in
  v2.4.0 (#477 — one of `apidiff`'s three incompatible lines), so retyping `Quality` costs nothing
  on the comparability axis that has not already been spent. The same change after v3 is a fresh
  incompatible line.
- **Downstream.** `SourceCitation` has 4 hot-path references; **no `.Quality` call site**. Predates
  v2.3.1, so the absence is genuine.
- **Recommendation.** **Retype to `*int` in v3.** Prefer the pointer over a companion
  `HasQuality bool`: the package has no `HasX` precedent, and the pointer matches how every other
  optional substructure on this type is modelled (`Data *SourceCitationData`,
  `AncestryAPID *AncestryAPID`). Note for the record: `Quality` is the **only** exported `int` in
  `gedcom/` whose zero value collides with a meaningful domain value — `CropRegion`'s four ints
  look similar and do not (DEC-2), and `Tag.Level`/`Tag.LineNumber`/`Trailer.LineNumber` are
  positional metadata where `0` is a real position.
- **Allocation cost — measure before implementing (added after panel review).** This audit scored
  every candidate on API shape and caller cost and **never on allocation cost**, which is a real
  gap for this entry and for BW-17. `SourceCitation` is embedded at ~35 sites across `gedcom/`,
  so its cardinality scales with individual and event count, not source count.
  `docs/guides/performance.md` documents decode as already allocation-dominated at scale — 18.1M
  allocations for a 46 MB / 203K-individual file, with GC around 30% of decode CPU. Retyping
  `Quality` to `*int` turns a zero-allocation inline field into one heap allocation per citation
  that carries `QUAY`. Run `go test -bench . -benchmem ./decoder/` against a prototype and record
  the `allocs/op` delta before committing. This does not change the recommendation — the pointer
  is still the right shape, and it matches the type's existing pointer fields — but the cost
  should be measured rather than assumed negligible.
- **Effort.** `effort:low`.

### BW-14 — Move `Address.Phone/.Email/.Website` to `Repository` as slices

`gedcom/repository.go:114,117,120` (fields), `gedcom/repository.go:92` (`Address`) · pass 002 C1
· **new**

- **Caller cost today.** `PHON`, `EMAIL`, `FAX` and `WWW` are siblings of `ADDR` inside
  `ADDRESS_STRUCTURE`, and every one of them is `{0:3}`
  (`docs/reference/gedcom-5.5-coverage.md:2288-2294`, `:2509-2522`). The library models them three
  mutually inconsistent ways: `Event`/`Attribute` correctly as `[]string`
  (`gedcom/event.go:127-136`); `Submitter` partially (`Phone`/`Email` slices, no `Fax`, no
  `Website`); `Repository` not at all — pushed into `Address` as **scalars**, assigned
  last-writer-wins at `decoder/entity.go:1229, 1235, 1241`, with `FAX` not handled at all
  (`:1249` lists it among "known tags not yet parsed"). Second, subtler cost: `Address.Phone` is
  populated **only** on the `Repository` path — `parseEventAddress` (`decoder/entity.go:679`)
  never touches it — so on an event or attribute address the same field is permanently `""` while
  the real values sit in `Event.Phone`. One field, two meanings, one of them "always empty".
- **Defer cost.** Nothing breaks; the loss continues, but **the cheap fix expires.** Two routes
  exist today: (a) retype the three `Address` fields to `[]string`, which fixes the loss and
  **breaks `Address` comparability permanently** — for a type whose own structure guarantees it
  never needed to lose it, since all seven `ADDR` substructures are `{0:1}`
  (`gedcom-5.5-coverage.md:270-279`); or (b) remove the three fields from `Address` and add
  `Repository.Phone/Email/Fax/Website []string`. Deferring means either living with three
  permanently-misleading fields or someone reaching for (a) in v4 because it is the smaller diff —
  which is how a comparable type loses `==` for no structural reason.
- **Downstream.** **6 hot-path sites**, both directions:
  `internal/gedcom/importer.go:1242-1244` (`convertGedcomAddress` copies all three into scalar
  `domain.Address` fields) and `internal/gedcom/exporter.go:922-924` (writes them back). Both
  break under either route. Note what the sites reveal: my-family's own `domain.Address` is scalar
  too, so route (a) would push a slice into its domain model.

  > **Corrected after panel review — route (b) is not a delete-only edit to two converters.**
  > An earlier draft framed the migration as confined to the two shared helpers. It is not.
  > `convertDomainAddressToGedcom` is also called for `Repository.Address` at
  > `internal/gedcom/exporter.go:1245`, under a comment that reads "Convert address if present
  > (includes phone/email/website)" — and `Repository` is the very type gaining the new slice
  > fields. Under route (b) that call stops carrying phone/email/website at all, so
  > `toGedcomRepository` (`exporter.go:1237-1245`) needs **new bespoke logic** to peel the three
  > values off the scalar `domain.Address` and populate the new `Repository.*` slices. The shared
  > converters are also reached for `Submitter.Address` (`importer.go:950`, `exporter.go:1225`).
  > The migration is real work on the write path, not a deletion — weigh that against the
  > "cheapest mechanical migration" framing when scheduling this.

  Note also that my-family calls
  `convertGedcomAddress` on **event** addresses, where these three fields are structurally always
  empty, so its event phone/email import is already a no-op and this change is a net gain there.
- **Recommendation.** **Route (b), in v3.** This is the only candidate in the sweep where the
  break *protects* comparability rather than spending it: `Address` ends up permanently `==`-safe
  and exactly matching the structure it is named for, the `{0:3}` loss is fixed, `REPO.FAX` gains
  typed access, and the library stops having three answers to one question. The additive half
  (`Repository.Phone/Email/Fax/Website`) is non-breaking and should ship in v2.x ahead of the
  removal (Part C **RT-15**). If only half can land, land the additive half and mark the three
  `Address` fields `Deprecated:` — but then the removal is a v4 change and `Address` carries dead
  fields through all of v3.
- **Effort.** `effort:medium`.

### BW-15 — `Header.Date time.Time` → `string`, populated and encoded

`gedcom/header.go:17` · pass 002 C4 · **new**

- **Caller cost today.** The field is never read and never written by this library. Never
  populated on decode: `buildHeader`'s tag switch (`decoder/decoder.go:373-388`) handles `CHAR`,
  `LANG`, `COPR` and `_TREE` only, and the generated report confirms `HEADER.HEAD.DATE` is
  `raw (undiagnosed)` in both 5.5 and 5.5.1 (`docs/reference/gedcom-5.5-coverage.md:769`). Never
  written on encode: `writeHeaderFields` (`encoder/encoder.go:217-253`) has no `DATE` branch. The
  only code that touches it is `gedcom/subset.go:170`, copying a value that is always the zero
  `time.Time` — while `subset.go`'s own doc comment (`:47-49`) promises callers that `Date` is
  carried over. So the field, its doc comment and the `Subset` contract all describe behaviour
  that does not exist, and its zero value is `0001-01-01T00:00:00Z`, a valid-*looking* timestamp
  a caller can format. Separately the type is wrong for the job: `HEAD.DATE` is `DATE_EXACT`, and
  the library's answer to GEDCOM dates everywhere else is `string` + `*gedcom.Date` per ADR 0001
  (Lossless Representation, NON-NEGOTIABLE). This is the one place the API converts a GEDCOM date
  into a Go type that cannot round-trip it.
- **Defer cost.** Nothing breaks; the field stays inert. But it cannot be *fixed* non-breakingly:
  populating it means choosing a lossy `time.Time` conversion, contradicting ADR 0001, and
  retyping it is a signature change. "Leave it and populate it later" is not available. It is also
  the only `time.Time` in the exported `gedcom/` surface, and `time.Time` carries its own `==`
  trap (wall clock, monotonic reading and `*Location` pointer all participate) — inert here only
  because `Header` is non-comparable via `Tags`. Removing it removes that trap from the module.
- **Downstream.** **No call sites.** `Header` appears once in my-family; no `Header.Date`
  reference exists. Predates v2.3.1.
- **Recommendation.** **Retype to `Date string` in v3**, matching `Event.Date`, `Attribute.Date`,
  `ChangeDate.Date` and `LDSOrdinance.Date`; populate it in `buildHeader` and write it back in
  `writeHeaderFields`. One break that also closes a coverage gap and removes an ADR 0001
  violation. Removing the field outright is smaller but leaves `HEAD.DATE` unmodelled and makes
  the `Subset` doc wrong in the other direction. Sequence with #449 (`buildHeader` takes no
  diagnostic collector) and #465 (header structures appended out of grammar order) — all three
  touch `buildHeader`.
- **Effort.** `effort:medium`.

## Tier 3 — take if the window has room

### BW-16 — Rename `Individual.ParentalFamilies` → `FamiliesAsChild`, `SpouseFamilies` → `FamiliesAsSpouse`

`gedcom/individual.go:426`, `:443` · pass 003 C8 · **new** · the only rename this audit recommends

- **Caller cost today.** The two methods sit adjacent and use **opposite** naming conventions for
  the same kind of relation. `SpouseFamilies` returns families where the individual is a spouse —
  correct under the rule its own name establishes (`<Role>Families` = families where I hold
  `<Role>`). `ParentalFamilies` returns families where the individual is a **child**, which under
  that same rule reads as the exact opposite, and "families where I am a parent" is what
  `SpouseFamilies` returns. The two names are not merely inconsistent, they are swappable in a
  caller's head — and both return `[]*Family`, both are non-nil, neither errors, so transposing
  them yields a wrong-but-plausible family tree with no type error, no panic and no diagnostic.
  A name that has to be contradicted by its own first sentence is the definition of a misleading
  name. The *fields* get it right: `ChildInFamilies` and `SpouseInFamilies` both state the role.
- **Defer cost.** A rename is a pure breaking change with no runtime benefit, so it is free now
  and impossible until the next major. The cost of deferring is that every caller written against
  v3 inherits the trap.
- **Downstream.** **No call sites.** Neither method appears in my-family's non-test sources; both
  predate v2.3.1. The cheapest possible rename.
- **Recommendation.** **Rename in v3, clean break, no deprecated alias** — the whole point is that
  the old name misleads, and keeping it reachable keeps the trap reachable. Note the obvious
  alternative, renaming `ParentalFamilies` to `ChildInFamilies`, is **illegal**: Go forbids a
  method and a field of the same name on one type, and `Individual.ChildInFamilies` is a field
  (`gedcom/individual.go:23`). `FamiliesAs<Role>` avoids the collision for both, states the role
  explicitly, and cannot be transposed.
- **Effort.** `effort:low`.

### BW-17 — `SourceCitationData.Text string` → `[]*SourceText`

`gedcom/source.go:92` · pass 002 C2 · **new**

- **Caller cost today.** `SOURCE_CITATION.SOUR.DATA.TEXT` is **`{0:M}`** in both 5.5 and 5.5.1
  (`docs/reference/gedcom-5.5-coverage.md:2355`), and `decoder/entity.go:464` is a bare
  `data.Text = tag.Value` — last-writer-wins across repeats. Separately there is no
  `foldContinuation`, so a multi-line `TEXT` reads back as its first line; that half is **already
  filed as #442**, which measured it (6 of 13 `TEXT` tags in `testdata/` carry subordinate
  `CONT`/`CONC`). The repeat half is not in #442's scope and is the part that decides the type's
  shape.
- **Defer cost.** `SourceCitationData` is one of only two remaining comparable types with a
  documented, evidence-backed reason to grow (`Transliteration` is the other — BW-22). Fixing it
  later is exactly the "old is comparable, new is not" line `apidiff` produced three times for
  #477: a v4 break for a defect known in v3. Deferring also means editing the same line twice
  across a major boundary, since #442's natural fix touches this field.
- **Downstream.** `SourceCitationData` appears once in the hot path; **no `.Data.Text` call site**.
  Predates v2.3.1, so genuine.
- **Recommendation.** **Retype in v3.** `Text []string` at minimum; prefer `[]*SourceText` with
  `Value`/`MIME`/`Language`, mirroring `SharedNoteTranslation`, because the element type can only
  be enriched additively later if it starts as a struct slice — and it closes the two
  `raw (undiagnosed)` entries for `TEXT.LANG`/`TEXT.MIME`. **Sequence #442 after this**, so the
  folding fix is written once against the final shape.
- **Allocation cost — measure before implementing (added after panel review).** Same gap as
  BW-13, and larger here: `[]*SourceText` costs a slice header plus one heap-allocated struct per
  element, where today there is one inline string. `SourceCitation` is embedded ~35 times across
  `gedcom/` and its cardinality scales with individual and event count;
  `docs/guides/performance.md` records decode at 18.1M allocations for a 46 MB corpus with GC
  near 30% of decode CPU. If `[]*SourceText` measures badly, `[]string` is the fallback that
  still fixes the `{0:M}` loss — at the cost of foreclosing the additive `MIME`/`Language`
  enrichment, which is precisely the trade to make with a benchmark in hand rather than without.
- **Effort.** `effort:medium`.

### BW-18 — `validator.Issue`: remove the `Details["line_number"]` and `["position"]` keys

`validator/tag_validator.go:124`, `:135`, `validator/encoding.go:169` · pass 005 C4 · **new**

- **Caller cost today.** ADR 0008's "Issue Structure" declares `LineNumber int // Source
  location`; the shipped `Issue` (`validator/issue.go:145-167`) has the ADR's list *minus* that
  field. The line number **is** known and is stuffed into the `Details` map as a string, present
  for 2 of 23 codes, so a caller must `strconv.Atoi` a map lookup that usually is not there.
- **Defer cost.** Honestly, only half a v3 item. **Adding `Issue.LineNumber int` is additive** —
  `Issue` is already non-comparable via its `Details` map — so it can ship in a minor and should
  (Part C **RT-16**). What needs v3 is *removing the map keys*: a map key appears in no signature,
  so `apidiff` cannot see it disappear and a reader breaks silently in a minor.
- **Downstream.** **No call sites for the key.** my-family reads
  `Code`/`Severity`/`Message`/`RecordXRef`/`RelatedXRef` (`internal/query/validation_service.go:55-59`,
  `:193-200`) and never touches `Details`.
- **Recommendation.** **Add the field in a minor now; remove the keys in v3.** Bundle the ADR 0008
  name-drift doc corrections (pass 005 D7) with the field addition.
- **Effort.** `effort:low`.

### BW-19 — `MediaObject.SharedNoteXRefs`: make it partition with `NoteXRefs`

`gedcom/media.go:84` · pass 003 C7 · **new** · pass 003 flagged this as genuinely two-sided

- **Caller cost today.** The doc comment is **false**. `gedcom/media.go:63-66` says "SNOTE
  pointers are tracked separately in `SharedNoteXRefs`" — but `decoder/entity.go:1476-1482`
  appends every SNOTE pointer to **both** slices, and `gedcom/xrefwalk.go:429` states the truth:
  "SharedNoteXRefs holds the same SNOTE pointers as NoteXRefs." Four costs follow: a caller who
  believes `media.go:65` and concatenates the two gets every shared note twice;
  `MediaObject.AllNotes` (`gedcom/media.go:111-118`) runs an O(n²) dedup purely to undo the
  duplication; `walkMediaObject` documents a remap hazard the duplication creates
  (`gedcom/xrefwalk.go:429-432`); and `MediaObject` is the *only* note-bearing type with this
  shape.
- **Defer cost.** The false doc comment, the dedup and the lockstep-remap hazard all stay, and
  every future note-related change to `MediaObject` has to special-case it.
- **Downstream.** **No call sites.** `SharedNoteXRefs` does not appear in my-family's non-test
  sources; predates v2.3.1.
- **Recommendation.** **Partition, in v3 — not remove.** Make `NoteXRefs` hold NOTE pointers only
  and `SharedNoteXRefs` hold SNOTE pointers only, so the code matches `media.go:65`. This is a
  behavioural break (the observable contents of `NoteXRefs` change) and so is v3-or-never; it
  deletes the dedup and the hazard. **The deciding factor between the two readings, which pass 003
  left open and no other pass claimed:** straight removal costs a real capability —
  `gedcom/minversion.go:207` uses `len(m.SharedNoteXRefs) > 0` as one of `RequiresGEDCOM7`'s
  signals, and with the slices merged there is no way to tell a NOTE pointer from an SNOTE
  pointer. Nothing in passes 004, 005 or 006 offers a substitute signal, and pass 004's B-2 is
  already *widening* what `RequiresGEDCOM7` detects. Partitioning keeps the signal and is the
  smaller conceptual change. **Sequence after #472 and #473**, so `MediaObject`'s note fields are
  settled first.
- **Effort.** `effort:medium`.

### BW-20 — Retire the divergence in the `[]error` validation surface

`validator/validator.go:233-246`, `validator/validator.go:12-25`, `gedcom_api.go:186-192` ·
passes 005 C3 + 007 MCF-6 · **new** · see "Cross-pass conflicts" §2

- **Caller cost today, four ways.** (i) **It ignores its own options.**
  `gedcomgo.ValidateWithOptions` takes `*ValidateOptions` and calls
  `NewWithOptions(opts).Validate(doc)`, and `Validator.Validate` reads **no** field of
  `v.config` — not `Strictness`, `MaxErrors`, `SkipRules`, `TagRegistry`, `ValidateCustomTags` or
  `SkipEncodingValidation`. A function named *WithOptions* honours none of six; its doc
  (`gedcom_api.go:189-190`) admits two and is silent on four. (ii) **Its rules are disjoint from
  `ValidateAll`'s.** It reports exactly `BROKEN_XREF`, `MISSING_REQUIRED_FIELD` and `EMPTY_FAMILY`;
  none of `ValidateAll`'s 23 codes is reachable here and none of these three exists there. They
  are not "basic" and "enhanced" as `validator/doc.go:7-10` claims — they are two validators
  overlapping on nothing. (iii) **Its codes are unreachable**: all 23 `Issue` codes are exported
  constants, all three `ValidationError` codes are bare literals. (iv) **It breaks ADR 0007
  twice**: `validateIndividual`/`validateFamily` leave `Line` at 0 although
  `record.Tags[i].LineNumber` is in hand, and `ValidationError.Error()` tests `XRef` **before**
  `Line`, so an error carrying both never prints the line.
- **Defer cost.** It keeps shipping as the *first* API `validator/doc.go` shows a new caller, and
  as the showcased quick start at `gedcom_api.go:26-30`.
- **Downstream.** **No call sites.** my-family calls only `NewWithOptions`
  (`internal/query/validation_service.go:76`, `:187`), `QualityReport` (`:81`) and `ValidateAll`
  (`:193`) — never `Validate`, never `ValidationError`. Both predate v2.3.1.
- **Recommendation.** **Reimplement, do not remove.** Reimplement `Validate` over `ValidateAll`
  (`Issue` already implements `error` at `validator/issue.go:172`), port the three orphan rules in
  with real `Code*` constants, remove `ValidationError` (the breaking half), make
  `ValidateWithOptions` honour `MaxErrors` and `SkipRules`, and delete the "legacy" wording from
  `validator/doc.go:9` and `gedcom_api.go:190`. This makes `ValidateWithOptions` honest and
  *recovers three rules `ValidateAll` currently cannot report*. Removing the surface outright was
  the alternative and is worse: it is the quick start, and pass 007 declined removing it from the
  facade for the same reason.
- **Effort.** `effort:high`. The largest single piece of work in Part A.

### BW-21 — #472 scope: retype `FamilyLink.Pedigree` to `[]string`, add `FamilyLink.Status`

`gedcom/individual.go:150` · pass 002, Part 4 · **covered by #472 — a scope amendment, not a new
issue**

- **Caller cost today.** `INDI.FAMC.PEDI` is `{0:M}` in 5.5 (`docs/reference/gedcom-5.5-coverage.md:326`)
  against a scalar `Pedigree`, so a link with two pedigree assertions reaches the typed model with
  one. `INDI.FAMC.STAT` is `raw (accepted)` in both 7.0 (`docs/reference/gedcom-7-coverage.md:1588`)
  and 5.5.1 (`gedcom-5.5-coverage.md:327`) — unmodelled entirely.
- **Defer cost.** The `Status` addition is free forever (additive). The `Pedigree` **retype is
  v3-or-never**: it is a field signature change, breaking regardless of comparability, and #472
  is already opening this type.
- **Downstream.** **No field-level call sites.** `FamilyLink` does not appear in
  `downstream-usage.md` at field level; predates v2.3.1.
- **Recommendation.** **Fold into #472's acceptance criteria** rather than filing separately —
  #472 is already breaking `FamilyLink`'s comparability by adding note fields, so the retype
  arrives with a break already being paid for. Do it while the type is open.
- **Effort.** `effort:low` as an amendment to work already scoped.

### BW-22 — Schedule #453 into v3.0.0, or record the accepted v4 break

`gedcom/individual.go:121` (`Transliteration`) · pass 002 C3 · **a scheduling decision, not an
implementation**

- **Caller cost today.** None. `Transliteration` is comparable and, on its own evidence, has no
  reason not to stay that way: all seven `NAME-TRAN` substructures are `typed`
  (`docs/reference/gedcom-7-coverage.md:1761-1767`) and the 5.5.1 name pieces it mirrors are
  `{0:1}`. This is purely a sequencing question, which is why it is recorded as a decision.
- **Defer cost — the sharpest instance of what #479 exists to prevent.** #453 is open and scoped:
  5.5.1's `FONE` and `ROMN` are accepted and dropped, and its stated plan maps both onto
  `PersonalName.Transliterations` — i.e. onto **this type**. `FONE`'s substructures include
  `NOTE {0:M}` and `SOUR {0:M}` (`docs/reference/gedcom-5.5-coverage.md:2215-2220`) plus a required
  `TYPE {1:1}` naming the scheme (`:2223`). So #453 as designed gives `Transliteration` at least
  `SourceCitations []*SourceCitation` plus the `NoteXRefs`/`InlineNotes` pair, and the type stops
  being comparable the day it lands. If #453 lands **in v3**, that is one `apidiff` incompatible
  line absorbed by a release that is already breaking. If it lands **after**, #453 — a 5.5.1
  fidelity fix, for the most common format in the wild, in its own words — becomes a v4-gated
  feature for a purely mechanical reason.
- **Downstream.** **No call sites.** `Transliteration` appears nowhere in `downstream-usage.md`;
  predates v2.3.1.
- **Recommendation.** **Schedule #453 into the v3.0.0 milestone**, so the break arrives with the
  feature that justifies it and the fields are populated when they appear. Two fallbacks, in
  order: (i) if #453 cannot land in v3, **record the v4 break as accepted** in
  `docs/governance/policies/api-stability.md`, so #453 is not later abandoned as "too breaking";
  (ii) adding `NoteXRefs`/`InlineNotes` to `Transliteration` inside #472's sweep half-solves and
  leaves `SourceCitations` and the scheme field as a *second* break for v4 — pass 002 judged it
  "genuinely worse" and this audit upholds that. **Explicitly do not** add empty slice fields with
  no decoder support just to spend the window; that is dead surface and CONSTITUTION.md rules it
  out.
- **Effort.** `effort:high` (this is #453's own effort, not new work) — or `effort:low` for the
  documentation fallback.

## Already filed — not re-proposed, but scope changes

| Issue | Status against this audit |
|---|---|
| **#472** — complete the note split on `MediaLink`, `FamilyLink`, `PlaceDetail`, `Association`, `SourceRepositoryLink` | **Scope changes — see the sequencing answer below.** Must land before #473 (hard constraint). Should absorb **BW-21**. Its `PlaceDetail` widening is *not* v3-gated and should be a follow-up, not a scope increase. |
| **#473** — remove the deprecated `Notes []string` fields | **Scope changes: the list goes from 11 to 13.** Add `Association.Notes` (`gedcom/individual.go:180`) and `SourceRepositoryLink.Notes` (`gedcom/repository.go:89`). |
| **#476** — `Visit` mutates the document its doc says it only reads | **Not v3-gated.** Closes as resolved-by **BW-1**. Full reasoning in "Cross-pass conflicts" §1. |
| **#477** — merged; introduced the `Family.NumberOfChildren`/`NCHI` dual storage | **BW-5 is the resolution #477 deferred.** #477's three comparability losses (`ChangeDate`, `LDSOrdinance`, `SourceCitation`) are already spent and are not recoverable before v3 — `SourceCitation` is where BW-13 lands, at no extra comparability cost. |

---

# Part B — Breaking and not worth it

**This section is the point of the audit.** Each entry records what was considered, and why it was
rejected, for a reader a year from now with none of this context who is wondering whether it was
ever looked at. CONSTITUTION.md is explicit that an open major-version window is not itself
evidence of need; that bar was applied to every entry here. No entry is softened into a maybe.

### DEC-1 — Make `gedcom.Date` non-comparable to remove the `==` trap

`gedcom/date.go:101` · pass 002 D1. `Date` is comparable and contains `EndDate *Date`
(`gedcom/date.go:121`), so `==` compares ranges and periods (`BET…AND`, `FROM…TO`) by **pointer
identity**: two structurally identical `BET 1900 AND 1910` values decoded from two documents
compare unequal. Making the type deliberately non-comparable is the only *structural* fix.
**Declined** — it would break map keys, `==` in tests, and every caller who compares two exact
dates correctly today, for a hazard that has produced no reported bug and already has a working,
documented alternative in the same file (`(*Date).IsEqual`, `gedcom/date.go:967`, delegating to
`Compare` at `:728`). my-family has 3 hot-path references plus one in `internal/domain/calendar.go`.
**The recommendation is the opposite of a break: `Date` should be recorded as a type that keeps
`==` past v3, deliberately.** The residual hazard is fixable non-breakingly with a doc comment on
`EndDate` and on the type (Part C **RT-32**).

### DEC-2 — Retype `CropRegion`'s four `int` fields to `*int`

`gedcom/media.go:5` · pass 002 D2. All four are `int` and `cropRegionToTags` guards each with
`!= 0` (`encoder/entity_writer.go:1290-1302`), so the same "0 vs absent" collision BW-13 flags on
`Quality` exists in *form*. **Declined — it does not exist in substance.** `CROP`'s own doc
comments record the defaults (`gedcom/media.go:8,11,14`): `Left` and `Top` default to 0, so
writing `TOP 0` and omitting it produce **the same rendered region**; `HEIGHT 0`/`WIDTH 0`
describe an empty region no producer emits. All four substructures are `typed`
(`docs/reference/gedcom-7-coverage.md:735-738`), `CROP` is 7.0-only with no 5.5 analogue so there
is no cardinality evidence, and there are **no call sites**. `*int` would cost `CropRegion` its
comparability and every caller a dereference, to preserve a distinction with no observable effect.
One real, non-breaking defect found while looking is routed to Part C (**RT-9**).

### DEC-3 — Remove `gedcom.Trailer`

`gedcom/trailer.go:5` · pass 002 D3. `Document.Trailer` is decorative on the write path:
`encoder/encoder.go:437` writes `0 TRLR` unconditionally, so nil and non-nil produce identical
output. **Declined both ways.** `TRLR` has exactly one grammar position with no substructures, so
the type can never gain a field and its comparability is not at risk and needs no protection; and
removing it is a gratuitous break of a field that costs nothing and carries a genuinely useful
`LineNumber` for diagnostics. **No call sites.** Leave it exactly as it is.

### DEC-4 — Break comparability on `Coordinates`, `ExternalID`, `MediaTranslation`, `SharedNoteTranslation`

`gedcom/event.go:80`, `gedcom/external_id.go:15`, `gedcom/media.go:134`, `gedcom/shared_note.go:40`
· pass 002 D4+D5. Each models a structure whose full substructure set is already typed with no
repeatable member the type does not hold: `PLAC.MAP` is `LATI`+`LONG`, both typed, `MAP` itself
`{0:1}`; `EXID` has only `TYPE`; `FILE-TRAN` only `FORM`; `NOTE-TRAN` `LANG` and `MIME`, both
typed. All but `Coordinates` are 7.0-only structures with no 5.5 analogue, so unlike
`Transliteration` (BW-22) there is no `FONE`/`ROMN` back-pressure. **Declined — keep all four
comparable.** `Coordinates` in particular is a type callers *do* handle (4 hot-path references
plus 12 to `ParseCoordinate`, my-family's most-referenced symbol of ours) and `==` on it is
meaningful and safe; modelling latitude/longitude as `string` is the lossless choice per ADR 0001,
not an oversight. `SharedNoteTranslation` is the explicit counter-example to BW-22: it is
`Transliteration`'s sibling in shape and it is safe, because nothing in 5.5 maps onto it.

### DEC-5 — Break comparability on `Tag`, `AncestryAPID`, `ConversionNote`, `InlineRepository`, `UnknownXRefError`

pass 002 D6. **Declined, individually.** `Tag` (`gedcom/tag.go:6`) structurally cannot gain a
slice — children live in the flat `Record.Tags` list, not inside a `Tag` — and its `==` is
load-bearing in tests and in `gedcom/testing`'s round-trip harness (8 hot-path references
downstream); protect it. `AncestryAPID` (`gedcom/ancestry.go:15`) is a vendor extension with a
fixed three-part parse; the format is Ancestry's, not ours to grow. `ConversionNote`
(`gedcom/report.go:11`) is a library-internal record type with no GEDCOM structure behind it, and
`ConversionReport` (which aggregates them) is already non-comparable. `InlineRepository`
(`gedcom/repository.go:46`) has one `Name` field and its only 5.5 grammar position is the `<NULL>`
branch of `SOURCE_REPOSITORY_CITATION`, which carries a name and nothing else.
`UnknownXRefError` (`gedcom/subset.go:16`) has pointer-receiver `Error()` and `Is()`, so the
**value** type does not implement `error` and never reaches a caller — its comparability is
unobservable.

### DEC-6 — Change any of the eight defined scalar types

`Calendar`, `DateModifier`, `Encoding`, `EventType`, `LDSOrdinanceType`, `RecordType`, `Vendor`,
`Version` · pass 002 D7. **Not at risk.** Comparability follows from the underlying kind and
cannot be lost without changing the kind. All eight are used downstream. The 76 exported constants
in `gedcom/` are values of these types and inherit the verdict.

### DEC-7 — Rename `PersonalName.Full`

`gedcom/individual.go:87` · pass 003. It holds the raw GEDCOM `NAME` value *with surname slashes*
(`"John /Doe/"`), not a display name, while the adjacent `Transliteration.Value` (`:120`) calls the
same concept `Value` and says "in GEDCOM format" — so the package has two names for one thing and
only one signals the encoding. **Declined:** the doc comment gives the slashed example on its first
line, and the actual downstream caller reads it correctly — my-family builds it with slashes at
`internal/query/validation_service.go:370` and strips them for display at `:517`. A caller who got
it right is not evidence of a misleading name. Renaming costs 4 downstream edits and buys no
behaviour change. Churn.

### DEC-8 — Rename `RecordTypeMedia`

`gedcom/record.go:23` · pass 003. Every other `RecordType*` constant echoes its entity type; this
one is `RecordTypeMedia` for `MediaObject`, alongside `Record.GetMediaObject()` and
`Document.MediaObjects()`. **Declined:** there is exactly one media-ish record type, so no caller
can pick the wrong one. Cosmetic consistency, and my-family references it. Churn.

### DEC-9 — Rename `ConversionReport.AllNotes()`, `Document.Notes()`, `Document.RequiresGEDCOM7()`, `Version.Before()`, `Date.Calendar`, `CloneTags`

pass 003. **All declined, each for its own reason.**
`ConversionReport.AllNotes()` (`gedcom/report.go:139`) collides in name with the eight entity
`AllNotes(doc)` methods — but different receivers, different return types, and the entity form
takes a `*Document` argument while this one takes none, so a caller cannot reach one while meaning
the other and the compiler catches any confusion.
`Document.Notes()` (`gedcom/document.go:172`) returns `[]*Note` (NOTE *records*) beside eleven
`Notes []string` fields — but it is the exact parallel of `Individuals()`/`Families()`/`Sources()`,
breaking that pattern would be worse than the collision, the return type disambiguates
immediately, and it is a my-family hot-path call (`internal/gedcom/importer.go:448`). Declining
protects a working call site for zero benefit.
`Document.RequiresGEDCOM7()` (`gedcom/minversion.go:25`) asserts more than it delivers — it
inspects only typed fields — but the doc comment states this explicitly and lists which features
are detected and which are not, including the raw-tag caveat. Disclosed limits are not a misleading
name.
`Version.Before()` (`gedcom/version.go:34`) returns `false` for an unknown version on either side,
so `a.Before(b)` and `b.Before(a)` can both be false — documented in its first two lines. A
three-valued comparison would be a better API, but that is an additive redesign, not a hygiene fix.
`Date.Calendar` is a field named for its type — standard Go, unambiguous. `CloneTags`
(`gedcom/clone.go:477`) is the only exported clone helper that is not a method, but it is genuinely
useful standalone (raw-tag slices are not attached to any one entity) and the name is accurate.

### DEC-10 — Remove entity-level `Tags` from all nine structs (`Record.Tags` as the single raw-tag store)

pass 004 V3-3, structural option · **the largest breaking change any lens proposed this cycle, put
to synthesis as an explicit go/no-go.** **Declined.** Three reasons. (i) It removes the `indi.Tags`
shorthand that my-family's documented workaround pattern leans on — CLAUDE.md names `Record.Tags`
as the escape hatch that "always lets it work around a gap", and making that escape hatch harder to
reach is the wrong direction in the one release where downstream is already absorbing several
migrations. (ii) The actual live defect underneath it has a **non-breaking** fix: `Record.Clone`
(`gedcom/clone.go:64-76`) clones the record's `Tags` and the entity's `Tags` independently,
severing the alias the decoder establishes (`decoder/entity.go:102, 890, 986, 1161, 1209, 1319,
1366, 1461`), so `doc.GetIndividual("@I1@").Tags[0].Value = "X"` affects encoder output before a
`Clone` and silently does not after. Re-establishing the alias inside `Record.Clone` fixes it with
no signature change — routed to Part C (**RT-10**). (iii) It touches nine exported structs and
interacts with #472/#473's note-field removals on the same structs, so it would be the riskiest
change in the release for the least evidence. **Note:** removing `Event.Tags` (BW-12) is accepted
and is not inconsistent with this — see "Cross-pass conflicts", sixth item.

### DEC-11 — Remove the eight typed `Header` fields

`gedcom/header.go:8-31` · pass 003. All eight (`Version`, `Encoding`, `SourceSystem`, `Date`,
`Language`, `Copyright`, `Submitter`, `AncestryTreeID`) are *non-authoritative on encode* — the
`Header.Tags` doc (`:38-44`) states "Tags is authoritative when encoding … Editing a typed field on
a decoded document therefore does not change the encoded header." That is exactly the category
BW-5 removes `Family.NumberOfChildren` for. **Declined anyway:** unlike `NumberOfChildren`, these
are the *entire typed read surface of the header*. Removing them would force every caller to
hand-walk raw tags for `Header.Version`, one of the most-read fields in the package (my-family
reads `Version` 5 times). The right fix is behavioural — make the encoder reconcile typed fields
with `Tags`, or diagnose the conflict — which is not breaking and is routed to Part C (**RT-11**).
Two of the eight have their own separate dispositions: `Header.Date` is BW-15 (a type defect, not a
dual-storage one) and `Header.Submitter` is Part C **RT-2** (never populated at all).

### DEC-12 — `Record.Entity interface{}` → `any`; and `Record.Value`, `Record.Tags`, `Date.Original`

pass 003. `any` is a type *alias* for `interface{}`, so the change is not breaking, not API
surface, and not this sweep's business. The other three are ADR 0003 lossless dual storage —
deliberate, documented, and load-bearing for the Lossless Representation principle. **Not legacy.
Cleared.**

### DEC-13 — Remove `Individual.FamilySearchID`/`FamilySearchURL()`, `SourceCitation.AncestryAPID`, `Header.AncestryTreeID` as duplicates of `ExternalIDs`

pass 003. **Declined:** they are populated from different tags (`_FSFTID`, `_APID`, `_TREE`) than
`EXID`, so they carry data `ExternalIDs` does not. Not redundant.

### DEC-14 — Merge `Family.Attributes` into `Family.Events`; change `Document.MinimumVersion()`'s floor

pass 003. `Family.Attributes` vs `Family.Events` was considered as possible duplication given
CENS/RESI routing. **Declined:** `gedcom/family.go:32-36` documents the split precisely and
`decoder/entity.go:919` matches the documentation. Correct as designed.
`Document.MinimumVersion()` (`gedcom/minversion.go:83`) never returns `Version55` — **declined**,
documented with the reason (5.5.1 is a superset of 5.5 for writing purposes).

### DEC-15 — Make `QualityReport` honour `Strictness`

`validator/validator.go:450-465` · pass 005 C2. Three of eight `Validator` methods ignore
`Strictness`: `QualityReport`, `FindPotentialDuplicates` and `Validate`. There is real evidence a
caller was misled — my-family carries a comment at `internal/query/validation_service.go:75`
asserting that strict mode is being set in order to get all severity levels, immediately above a
`QualityReport` call that does not honour it. **Declined as a breaking fix:** `QualityReport` already partitions its output
into `Errors`/`Warnings`/`Info` (`validator/quality.go:36-39`), and filtering by severity before
partitioning by severity is self-defeating. **The defect is the doc**, routed to Part C
(**RT-27**).

### DEC-16 — Introduce a `Rule` interface in `validator/`, or harmonise the sub-validator entry-method names

`validator/validator.go:110-224` · pass 005 C7. The package ADR 0008 titles "pluggable,
composition-based" exports **no interface at all** (`inventory.md` reports `ifacemethod = 0` for
every package in the module), and its seven sub-validators do not even share a method name — four
use `Validate(*gedcom.Document) []Issue`, one `ValidateHeader`, one `ValidateXRefs`, one
`FindDuplicates`. `Validator` holds all seven in unexported fields behind unexported lazy getters,
so no caller can substitute or append one; `SkipRules` *subtracts* from output and nothing adds.
The module's only genuine caller extension point is `TagRegistry` — data, not code.
**Declined, and it is not an oversight:** ADR 0008 never promised caller-pluggability — its driver
reads "Easy to add **new validators**" and Option C's pro is "extensible", both in the maintainer's
voice, and the code matches that reading exactly. The word "pluggable" in the ADR *title*
over-promises to a reader expecting Go interfaces; the decision body does not. CONSTITUTION
requires evidence of need and there is none: no downstream request, no corpus signal, no
spec-coverage gap, and **no call sites** for any of the seven.
**The price of this decline, recorded honestly:** *declaring* a `Rule` interface later is additive
and can ship in any minor, but the three entry-method renames it needs are breaking, so deferral
does not lose the ability — it prices it at one whole major cycle whenever it is wanted. Doing the
renames alone in v3 would buy the option at v3 prices. Recommended only if v3 has slack, which
given Part A's size it does not.

### DEC-17 — Type `ReferenceReport.ByType`/`OrphanedByType` as `map[ReferenceType]int`

`validator/references.go:201-203` · pass 005 D1. A genuine typed/raw mismatch: the doc says "Keys
are ReferenceType values", `ReferenceType` exists with six constants, and they are used only as
`string(RefTypeCHIL)`. **Declined:** `ReferenceType` is a *defined string*, so `ByType["FAMC"]` and
`ByType[RefTypeFAMC]` read identically and no caller gets it wrong. Churn. **No call sites.**

### DEC-18 — Rename either `Encoding` type; swap `ValidatorConfig`/`ValidateOptions`; renumber `validator.Severity`

pass 005 D2, D4, and the resolved stage-001 lead.
`charset.Encoding` (int, 7 constants) and `gedcom.Encoding` (string, 5 constants) both export
`EncodingUTF8`/`ANSEL`/`ASCII`/`LATIN1`, and `validator/encoding.go:53-77` hand-rolls the mapping
between them. **Declined as a rename:** they are different types in different packages, so a mix-up
is a compile error and never a silent bug, and each name is right in its own package. Two real gaps
were found and both are **additive** — routed to Part C (**RT-19**).
`ValidatorConfig`/`ValidateOptions` (`validator/validator.go:41,46,91`): the deprecated name is the
concrete struct and the recommended name is a type alias pointing at it, which stage 001 flagged as
backwards. **Resolved as a non-candidate:** Go type aliases are fully symmetric — the two names are
the same type, so which is "real" is invisible to callers, godoc and `apidiff`, and swapping the
declaration is a zero-effect edit. **v3 honours the marker's promise that `ValidatorConfig` will
not be removed.** If the wording bothers anyone, reword the marker in a patch.
`validator.Severity`'s zero value is `SeverityError` (`validator/issue.go:20`). **Declined:** no
doc claims otherwise, defaulting to most-severe is the safe direction, and `decoder.Severity`
deliberately mirrors it value-for-value — renumbering one breaks the mirror. (Contrast BW-9, where
the docs promise the opposite of what the code does.)

### DEC-19 — Unify `decoder.Severity` and `validator.Severity`

`decoder/diagnostics.go:18-45`, `validator/issue.go:15-42` · passes 005 and 006, neither owning it.
Same constants, same `String()`, hand-maintained in two places. **Declined** — the duplication is
deliberate and its reason is recorded in-tree at `decoder/diagnostics.go:16-17` (avoiding an import
cycle); the cost is one mirror of three constants; unifying it requires a new shared package, which
is a larger structural change with no evidence of need. **No downstream confusion**: my-family uses
`decoder.SeverityError` at `internal/gedcom/importer.go:363` and `validator.Severity*` at
`internal/query/validation_service.go:200`, each correctly. Routed to Part C (**RT-31**) as a
cross-reference comment on each enum.

### DEC-20 — `*int` fields on `DateLogicConfig`; `[]Issue` from `TagRegistry.ValidateTag`; immutable `XRefPattern`/`YesNoPattern`

pass 005 D5, D6, and the cleared vars.
`NewDateLogicValidator` (`validator/date_logic.go:55-70`) writes defaults back **through the
caller's pointer**, and `DateLogicConfig`'s 0-means-default makes `MaxReasonableAge: 0`
unexpressible. **Declined as breaking:** copying before defaulting is a **non-breaking** fix
(Part C **RT-17**), and nobody wants a zero threshold — `SkipRules` disables checks properly.
`TagRegistry.ValidateTag` returns `*Issue` where every peer returns `[]Issue`
(`validator/registry.go:140`), because `tag_validator.go:120-125` mutates `Severity`/`RecordXRef`
after the fact. **Declined** — nil-means-no-issue is documented and unambiguous.
`XRefPattern`/`YesNoPattern` (`validator/registry.go:36-39`) are mutable exported `*regexp.Regexp`
values, the classic hazard. **Declined** — they are intended as ready-made `ValuePattern` values,
are safe for concurrent use, and have no reported cost.

### DEC-21 — Constrain `encoder.EncodeOptions.TargetVersion`; type `decoder.Code*`; type `StreamEncoder.State`; rename `decoder.NewParseError`; rename `parser.MaxNestingDepth`

pass 006.
`TargetVersion`'s zero value `""` correctly means "preserve the document's version"
(`encoder/encoder.go:206, 214, 229`), and `gedcom.Version` being an unvalidated string means
`"5.5.5"` passes through verbatim — but `encoder/encoder.go:203` documents verbatim pass-through as
**deliberate**, so `555SAMPLE.GED`'s `5.5.5` survives instead of being normalised. **Declined:**
constraining it trades a documented losslessness affordance for type tidiness.
The nine `decoder.Code*` untyped string constants: **declined**, no caller burned, and the
looseness actually lives on `Diagnostic.Code string`.
`StreamEncoder.State`: **declined** — stringly-typed but documented as a testing/debugging aid;
typing it enlarges the surface for no caller benefit.
`decoder.NewParseError` promises a `parser.ParseError` and returns a `decoder.Diagnostic`, and
`parser.ParseError` genuinely exists, so the collision is real — but its doc at
`decoder/diagnostics.go:149` says plainly what it returns, and the bar is name *and* doc. Churn.
`parser.MaxNestingDepth = 100` names a *ceiling count* (valid levels 0–99) rather than a depth, an
off-by-one waiting to happen, and BW-2's removal of `MaxNestingDepth` from `DecodeOptions` makes it
the sole survivor of that confusion — pass 006 declined it and explicitly invited synthesis to
overturn. **The decline is upheld:** the constant's own doc (`parser/parser.go:11-14`) spells the
`-1` relationship out in full, so a caller reading name *and* doc does not get it wrong, and
renaming a public constant to fix a readability wart with no reported caller error is exactly the
churn CONSTITUTION rules out.

### DEC-22 — `parser/`: collapse the two iterator types; drop the `Iterate*` trio; make converter's `PreserveUnknownTags` actually drop tags

pass 006 P-2c, P-3; pass 007.
**Collapsing `RecordIteratorWithOffset`/`RecordsWithOffset` into their plain siblings** (removing 2
types, 1 func, 3 methods and 1 constructor) is only defensible *after* the byte-offset bug fix
lands, because until then the two types differ in whether their numbers are usable. **Declined for
v3:** it would couple a cleanup to an unlanded bug fix, and there is no evidence anyone misuses the
split once the values are correct. **No call sites** — my-family does not import `parser`. Recorded
as an accepted v4-gated cleanup; the fix itself (Part C **RT-1**) is what matters and it is not
gated on anything.
**Dropping `LazyParser`'s `Iterate`/`IterateFrom`/`IterateAll` in favour of the range-over-func
`Records`/`RecordsFrom`/`AllRecords`** is a real "one obvious way" wart — six method docs to keep
in sync. **Declined:** the docs say outright "This is the range-over-func equivalent of [Iterate]",
so a caller is not misled, merely offered two spellings; pass 006 itself recorded it as raised
rather than recommended and named it "the one candidate that fails the CONSTITUTION evidence bar on
its own merits". **No call sites.**
**Making `converter.ConvertOptions.PreserveUnknownTags` actually drop tags** is **declined** —
forbidden by Lossless Representation. BW-11 renames the flag to describe what it does instead.

### DEC-23 — `merge.HeaderConflict`: add slice-shaped detail (`DroppedTags []*gedcom.Tag`)

`merge/combine.go:619-623`, `:170-174` · pass 007 MCF-5. `HeaderConflict` is comparable today, and
its `Doc1`/`Doc2` fields carry *prose sentinels* for two of three producers ("kept" / "dropped
(header structure may appear only once)" / "3 tag mapping(s)") where structured detail would be
better. Adding a `DroppedTags []*gedcom.Tag` field is **v3-or-never**, because it breaks
comparability the way `apidiff` flagged three `gedcom/` types. **Declined:** there is no evidence
of need — no downstream request (my-family imports `merge/` in **zero** files), no corpus signal,
no spec gap — and a pre-emptive comparability break to hold a field nobody has asked for is exactly
what CONSTITUTION rules out. **Consequence recorded so it is not a surprise later:** if structured
conflict detail is ever wanted, it is a v4-gated break. The `Kind` discriminator and the doc fix
that pass 007 also identified are *additive* and can ship in any minor — routed to Part C
(**RT-20**).

### DEC-24 — Facade coverage gaps; typing `CombineError.Kind`; removing `testing.Option`; functional options for the encoder; pre-emptive `ConvertOptions` comparability break

pass 007's declines, plus pass 006 E-1c.
**Facade coverage gaps** — `validator.Issue` is re-exported but `validator.Severity*` is not, and
`gedcom_api.go:196` names those severities without exporting them; likewise `decoder.Severity*`,
`converter.ConvertOptions`, `gedcom.Record`/`Source`/`Tag`. **Declined for v3:** every one is
*additive* per `docs/governance/policies/api-stability.md:36-38`, so a v3.1 minor does it at zero
cost and the major-version window should not be spent on additions.
**Typing `CombineError.Kind`** — no evidence of need.
**Removing `testing.Option`** — declined; it is a deliberate reserved extension point per
`gedcom/testing/roundtrip.go:55-58`, and the variadic parameter is part of the
`AssertRoundTrip`/`CheckRoundTrip` signatures.
**Replacing `encoder.DefaultOptions()` with functional options** — a larger break than BW-6 with no
evidence of need; declined.
**A pre-emptive `ConvertOptions` comparability break** — declined; no concrete pressure, unlike
BW-22's `Transliteration`, which has a filed, scoped issue behind it.

---

# Part C — Real, but not breaking: these must not consume the v3 window

**36 entries covering 66 items**, matching the summary table. (Nine entries — `RT-24`–`RT-32` —
carry an explicit count column summing to 39; the other 27 are one item each.) Several passes
flagged findings here as more urgent than anything in Part A. None of
them needs a major version, and holding a bug fix for a major version is worse than shipping it.
Each row says where it should go instead.

## C.1 — Live defects that ship wrong behaviour today (file as bugs, patch release)

| # | Finding | Site | Route |
|---|---|---|---|
| **RT-1** | **`parser.RecordIterator` produces byte offsets that are wrong**, by the cumulative length of every level-0 line seen. Slicing at them yields a window straddling two records and starting mid-line. `iterator.go:127` puts a buffered level-0 line's length into `pendingLen` and then `break`s, so the `byteOffset += lineLen` at `:132` never runs; `:100` then subtracts `pendingLen` from a `byteOffset` it was never added to. `:79`'s `lineEnding: 1` guess adds a second, independent error on CRLF. Seven public entry points reach the broken producer (`parser.Records`, `LazyParser.Iterate`/`IterateFrom`/`IterateAll`/`Records`/`RecordsFrom`/`AllRecords`); only `BuildIndex` and `RecordsWithOffset` use the correct one, which is why indexed lookup works and nobody noticed. Against ADR 0007 this is the failure mode the ADR itself names: wrong data, no error, no diagnostic. | `parser/iterator.go:41,44,79,100,127,132` | **Own issue, `area:parsing`, bug, patch. The single highest-value action from this audit.** |
| **RT-2** | **`Header.Submitter` is never populated, so the validator warns on every 5.5/5.5.1 document.** `HEAD.SUBM` reaches no typed field — absent from `buildHeader`'s switch, and `HEADER.HEAD.SUBM` is `{1:1} raw (undiagnosed)` in both 5.5 and 5.5.1 (`docs/reference/gedcom-5.5-coverage.md:777`). `validator/header.go:39` then tests exactly that field, so `CodeMissingSUBM` fires for **every** decoded 5.5/5.5.1 document including every one with a perfectly good `1 SUBM @U1@`. my-family calls the validator (`internal/query/validation_service.go:76,187`), so the false positive reaches a real consumer today. Fix: one `case "SUBM":` in `buildHeader`. | `gedcom/header.go:26`, `decoder/decoder.go:373-388`, `validator/header.go:39`, `validator/encoding.go:117` | **Own issue outside #479.** Largest zero-value defect found; needs no API change. |
| **RT-3** | **`ParseCoordinate`/`AsDecimal` accept `NaN` and `Inf`.** `strconv.ParseFloat` accepts `"NaN"`, `"Inf"`, `"Infinity"` and hex float literals; the range guards at `gedcom/coordinate.go:104,112` are NaN-blind (`NaN < -90` and `NaN > 90` are both false), so `AsDecimal` returns `(NaN, NaN, nil)` — a success return outside the documented range. `ParseCoordinate` alone returns `+Inf, nil` for `"Ninf"`. **`ParseCoordinate` is my-family's most-referenced symbol of ours — 12 references.** | `gedcom/coordinate.go:23,35,74,104,112` | **Own issue.** Highest-priority behaviour bug on downstream-exposure grounds. |
| **RT-4** | **The encoder gates `PLAC` on the legacy scalar**, so a caller who sets only `PlaceDetail` gets no `PLAC` tag at all — name and coordinates dropped silently. Fixing `eventDetailToTags`/`placeToTags` to drive `PLAC` off `PlaceDetail.Name` is *additive* behaviour and not breaking. | `encoder/entity_writer.go:829-830, 1088-1092` | **Own issue, urgent.** **Must land before BW-3.** |
| **RT-5** | **`encoder.EncodeOptions.LineEnding` zero is `""`**, interpolated raw at eleven write sites (`encoder.go:78,225,228,236,242,248,425,427,429,431,437`) with no default guard, producing **one unseparated line** that no GEDCOM reader — including this library's own parser — can read back, with `err == nil`. `StreamEncoder` has the same hole (`streaming.go:103` guards only `opts == nil`). `nil` is safer than `&EncodeOptions{}`; that inversion is the sharp edge. Fix: an `effectiveLineEnding()` mirroring `effectiveMaxLineLength()` (`options.go:52`). | `encoder/options.go:13` | **Own issue, patch.** Sequence with **#466** (encoder does not reject newlines in `Tag.Tag`/`XRef`/`Record.Type`) — both concern what separates one written line from the next. |
| **RT-6** | **`converter/` reports data loss that does not happen.** `record70DataLoss` reports `NO, TRAN, PHRASE, UID, CREA, SNOTE` as `DataLoss` + `Dropped` with `Result: ""`, but **nothing removes those six** — the only tag-slice writes in `converter/` are header-scoped (`header.go:71,132,160,221,232`) or the EXID rewrite (`exid.go:62`). **`EXID` is the exception and must not be swept up in the fix:** `transformEXIDToVendorTags` genuinely does remove converted EXID subtrees, and `exid.go:51-52` documents that it runs *before* `record70DataLoss` precisely so a converted EXID is not double-counted — so the surviving EXIDs it reports (non-ARK on `INDI`, and all of them on `FAM`/`SOUR`/`REPO`, which `exid.go:31-37` explicitly leaves "for the data-loss sweep") are reported correctly. Deleting the `EXID` entry from `tags70` would stop reporting real loss. A 7.0→5.5.1 downgrade emits a file declaring 5.5.1 that still contains the 7.0 structures while the report says they were dropped, and `StrictDataLoss` therefore fails conversions that lose nothing. Reaches users: my-family copies `report.DataLoss` into a JSON struct (`internal/gedcom/exporter.go:523-537`) and its `PreviewConversion` exists — per its own doc — "so callers can warn about data loss **before initiating a download**". | `converter/converter.go:99-102, 285-320` | **Own issue.** *Reclassified from pass 007, which proposed v3:* the fix changes report contents, not a signature, and results move in the honest direction — the same class as RT-7, which no pass proposed for v3. Consistency puts it here. Plus a doc-only patch to `docs/guides/converter.md:145-155`, which already tells the truth for the 5.5.1→5.5 set at `:163`. |
| **RT-7** | **`MinimumVersion()`/`RequiresGEDCOM7()` under-report at three levels**, against a doc that promises the result is "conservative". (i) A `Record.Entity` that is a `*SharedNote` with `Record.Type != RecordTypeSharedNote` — `minversion.go:48` tests `rec.Type` only and the type switch at `:51-70` has no `case *SharedNote:`. (ii) `Individual.Associations[i].SourceCitations` non-empty with an empty `Phrase` — `associationPhraseRequiresGEDCOM7` (`:191-198`) tests only `a.Phrase`. (iii) A `FACT` entry in `Family.Attributes` — 7.0-only *by position* (`docs/reference/gedcom-7-coverage.md:170,486` vs `gedcom-5.5-coverage.md:946,1907`), and `minversion.go:156-159` already applies exactly this positional reasoning to `Event.Associations`. Level 3 hits real decoded files. A caller doing "emit at the lowest compatible version" writes a 5.5.1 file that silently loses the structure. | `gedcom/minversion.go:48,51-70,124-143,171,191-198` | **Own issue.** Signature is already right; the fix moves results in the strictly-more-conservative direction the doc promises, so nobody's build breaks. Note the movement in release notes. |
| **RT-8** | **Both `gedcom/validation.go` age validators are direction-blind.** `ValidateParentChildDates` and `ValidateMarriageDates` both call `YearsBetween`, which returns an **absolute** value, so a child born *before* the parent passes (1900/1850 → 50 → `nil`) and marriage-before-birth is undetectable. Both are documented as *directional* checks. `(*Date).IsBefore`/`.Compare` already exist for the ordering test. | `gedcom/validation.go:27,32,51,58,67` | **Own issue.** No signature change; both already return `error`. |
| **RT-9** | Four smaller behaviour bugs, groupable into one issue or filed separately: **(a)** `(*Date).ToTime` ignores `IsBC`, so `"25 DEC 44 BC"` returns 25 Dec **44 AD** with a nil error, which propagates into `YearsBetween`'s "exact" path; `(*Date).Validate` ignores it too, so `29 FEB 5 BC` is rejected and `29 FEB 4 BC` accepted — both backwards, when `AstronomicalYear` (`gedcom/calendar.go:93`) exists for exactly this. **(b)** `HebrewDaysInYear` returns **356** for ~3.3% of years (100 of 3000–5999), a year length that does not exist, and `JDNToHebrew` inherits it (Elul day 31); root cause is `hebrewDelay`'s dehiyot rules 2/3 at `gedcom/calendar.go:437-443`. **(c)** `cropRegionToTags` appends the `CROP` tag *before* the four `!= 0` guards, so an all-zero `&CropRegion{}` encodes as a childless `n CROP`. **(d)** `Header.Copyright` and `Header.AncestryTreeID` are populated on decode and never written on encode. | `gedcom/date.go:790,805,654,666`; `gedcom/calendar.go:403-454,475,604`; `encoder/entity_writer.go:1287`; `encoder/encoder.go:217-253` | **Issues, patch.** All internal; no signature changes. |
| **RT-10** | **`Record.Clone` severs the `Record.Tags` ⇄ entity `.Tags` alias**, so `doc.GetIndividual("@I1@").Tags[0].Value = "X"` affects encoder output before a `Clone` and silently does not after — and `Subset` inherits it via `gedcom/subset.go:103`. `Record.Tags` is the escape hatch CLAUDE.md names, so its semantics being clone-dependent is a hazard on the documented workaround path. Fix: clone `r.Tags` once and assign that same slice into the cloned entity. | `gedcom/clone.go:64-76` | **Own issue, patch.** The non-breaking half of DEC-10. |
| **RT-11** | **The encoder ignores the typed `Header` fields whenever `Header.Tags` is non-empty**, so editing a typed field on a decoded document silently does nothing. Documented at `gedcom/header.go:33-42` and `encoder/doc.go:48-59` rather than fixed. Fix is behavioural: reconcile typed fields with `Tags`, or diagnose the conflict. | `encoder/encoder.go:83-88` | **Own issue.** The live half of DEC-11. Both passes 003 and 006 routed it here. |
| **RT-12** | **`version/detect.go`'s 7.0 marker set diverges from ADR 0005.** The ADR gives `EXID, SCHMA, PHRASE`; the code (`:104-106`) also treats `SNOTE, UID, CREA, MIME` as 7.0, and has `FACT` at `:110` where the ADR's 5.5.1 list has `FAX`. **`UID` is common in 5.5.1 vendor exports, so a 5.5.1 file can be reported as 7.0.** | `version/detect.go:104-110` | **Own non-#479 issue.** Behavioural, not API. |

## C.2 — Additive work that should ship in v2.x, ahead of the break it supports

| # | Addition | Supports |
|---|---|---|
| **RT-13** | `func (e *Event) PlaceName() string` and `func (a *Attribute) PlaceName() string`, nil-safe | **BW-3** — lets my-family migrate its nine read sites before the removal lands |
| **RT-14** | An `AttributeType` typed constant set — `AttributeOccupation = "OCCU"` plus the other 12 tags from `decoder/entity.go:140` | **BW-4** — the additive half; only the `EventOccupation` removal needs the boundary |
| **RT-15** | `Repository.Phone`, `.Email`, `.Fax`, `.Website` as `[]string` | **BW-14** — the additive half; also gives `REPO.FAX` typed access for the first time |
| **RT-16** | `validator.Issue.LineNumber int` — additive, since `Issue` is already non-comparable via its `Details` map | **BW-18** — the field lands in a minor, the map keys go in v3 |

## C.3 — Non-breaking hygiene and additions (normal issues)

| # | Item | Site |
|---|---|---|
| **RT-17** | `NewDateLogicValidator` writes defaults back **through the caller's pointer**; `Validator` stores the config by pointer rather than copying. Fix: copy before defaulting. | `validator/date_logic.go:55-70`, `validator/validator.go:130-135` |
| **RT-18** | `ParseWithOptions` mutates the caller's options (`if opts.MaxErrors < 0 { opts.MaxErrors = 0 }`), giving a data race to anyone sharing one `*ParseOptions` across goroutines. Fix: a local copy. Related: `Parser` is stateful and not safe for concurrent use, which no doc says. | `parser/parser.go:405-407` |
| **RT-19** | Add `charset.Encoding.String()` (today `fmt` prints a bare int, unlike `gedcom.Encoding.String()` and `validator.Severity.String()`), and a `charset`↔`gedcom` encoding converter so `validator/encoding.go:53-77` stops hand-rolling one. | `charset/charset.go:17` |
| **RT-20** | Add a `Kind` discriminator to `merge.HeaderConflict` and fix the doc, so callers stop parsing prose sentinels out of `Doc1`/`Doc2`. Additive — the slice-shaped detail is DEC-23. | `merge/combine.go:170-174, 619-623` |
| **RT-21** | Export the `interface{ ErrorLine() int }` extension point. It is pinned at `charset/charset.go:55` and `charset/ansel.go:37` ("The parser matches on this method, not on the concrete type, so pin it here") but declared unexported as `parser.lineLocatedError` (`parser/errors.go:46-48`), so a caller supplying their own reader to `parser.Parse` *can* satisfy it but cannot name it, assert against it, or find it in godoc. Give `*validator.ValidationError` the method too — it has `Line` and is the only line-carrying error type in the module that does not implement it. | `parser/errors.go:46-48` |
| **RT-22** | `RecordIterator` reports over-length lines through `bufio.ErrTooLong` while `RecordIteratorWithOffset` uses `ErrLineTooLong`, so a caller must check both. Fix: wrap it. | `parser/iterator.go:11-12` |
| **RT-23** | `applyOptions` panics on a nil `Option` — against ADR 0007. Add `Record.Is*` for `Repository`/`Note`/`MediaObject`/`Submitter` (`Is*` exists for Individual/Family/Source/SharedNote only). Fill the facade re-export gaps (DEC-24) in a v3.1 minor. | `gedcom/testing/options.go:21-24`; `gedcom/record.go`; `gedcom_api.go` |

## C.4 — Documentation fixes (patch)

| # | Item | Count |
|---|---|---|
| **RT-24** | **`Record.Tags` is authoritative on encode, and only `Header.Tags` says so.** `gedcom/record.go:43-44` says merely "Tags contains all the tags that make up this record"; the full rule is spelled out for `Header.Tags` at `gedcom/header.go:33-42` and in `encoder/doc.go` for the header only. Port that paragraph across. **The highest-value doc fix in the audit, and a required companion to BW-5.** | 1 |
| **RT-25** | Pass 004's contract-fidelity corrections DOC-1 … DOC-24, where the behaviour is right and the comment is wrong: `Descendants`/`Ancestors` ordering claims; `HasDataLoss` ignoring `Dropped`; five `Individual` methods promising "an empty slice" and returning nil; `Parents` promising "biological"; the `RequiresGEDCOM7` feature list omitting `UID` and OBJE-level `RESN`; `Clone`'s unknown-`Entity` pointer-sharing; all eight `AllNotes` methods on unresolvable XRefs; `DetectVendor`'s first-match-wins precedence; `GetRecord`'s `XRefMap`-only lookup; the `Document.*()` collections sharing element pointers; the nil-receiver boundary; `Get*`-vs-`Is*` authority; `FromAstronomicalYear`'s inverted bullet; `Compare`'s inverted BC example; `YearsBetween`'s both-vs-either; `ToGregorian`'s year-0 early return; `HebrewDaysInYear`'s wrong worked example; `HebrewDaysInMonth`'s undocumented `0` sentinel; `JDNToFrench`'s undocumented input domain; `ValidateParentChildDates`'s third silent-pass case; `(*AncestryAPID).URL`'s silent `""`. | 24 |
| **RT-26** | Pass 006's doc halves: correct `docs/guides/decoding.md:208`, `FEATURES.md:44` and `decoder/options.go:33-38` to say `StrictMode` governs `DecodeWithDiagnostics` only (do this **now**, regardless of BW-10); state the producer dependency **at** `RawRecord.ByteOffset`/`.ByteLength` (`parser/iterator.go:40,43`); note `RecordIndex.LookupByType`'s last-one-wins for XRef-less records. | 3 |
| **RT-27** | Pass 005's doc fixes: `QualityReport`/`FindPotentialDuplicates`/`Validate` do not honour `Strictness` (DEC-15); ADR 0008 name drift (`ValidateStructure`→`ValidateAll`, `ValidateReferences`→`FindOrphanedReferences`, `filterBySeverity`→`filterByStrictness`, `DateLogicConfig{MaxAge}`→`MaxReasonableAge`, `usedXRefs` shape); `charset.EncodingUnknown`'s doc says "no BOM was detected" but it is also `DetectEncodingFromHeader`'s no-CHAR result and a valid *input* to `NewReaderWithEncoding`. | 3 |
| **RT-28** | Pass 007's doc fixes: `gedcom/testing/doc.go:44` still says header structure is "not compared" (false since #429); `DecodeResult`'s doc is duplicated verbatim from `decoder/decoder.go:15-16` (DRY drift risk); three `docs/guides/converter.md` corrections. | 3 |
| **RT-29** | `gedcom/doc.go:18` and `:24` — the package-level example prints `individual.Name` and `person.Name`. There is no `Name` field on `Individual`; it is `Names []*PersonalName`. **The package doc's only code sample does not compile**, and it is the first thing a new caller reads. | 1 |
| **RT-30** | `(*Individual).Spouses` can return a non-spouse: `gedcom/individual.go:385` dispatches `if fam.Husband == i.XRef { …wife… } else { …husband… }`, so an asymmetric or dangling FAMS link — common in malformed exports — returns the family's **husband**, who is not this individual's spouse. Guarding is a *behaviour* change (narrowing); **the doc fix is recommended**, because malformed-export tolerance is worth more here and the library's stated posture (ADR 0007) is transparency over rejection. Recorded as the one Part-C item with a defensible behaviour-change reading. | 1 |
| **RT-31** | Add a cross-reference comment on `decoder.Severity` and `validator.Severity` pointing at each other, noting they must be renumbered together. Resolves the routing both passes left open (DEC-19). | 1 |
| **RT-32** | Doc `gedcom.Date`'s `==`-vs-`IsEqual` hazard on the type and on `EndDate` (DEC-1). Record in `parser/line.go` and in `api-stability.md` that **`parser.Line` stays Go-comparable indefinitely** — its five fields are pinned by the GEDCOM grammar itself, and the two futures that would cost `==` (continuation fragments, per-line diagnostics) are already designed out one layer up by ADR 0006 and ADR 0007. The rule to write down: *`Line` gains only scalar fields; plural data goes on `RawRecord` (already non-comparable) or a side table keyed by `LineNumber`.* This is precisely how `gedcom.ChangeDate`/`LDSOrdinance`/`SourceCitation` lost comparability in #477 — unnoticed. | 2 |

## C.5 — Policy and tooling

| # | Item |
|---|---|
| **RT-33** | **Add a "semantic breaks" section to `docs/governance/policies/api-stability.md`.** BW-9 and BW-10 are real breaking changes that `make api-check` reports clean, because neither changes a signature. The v3 process needs a home for changes `apidiff` cannot see, and both must appear in the v3 migration notes. This is the answer to pass 006's open question 1. |
| **RT-34** | Record accepted future breaks in the same policy file, so they are not later abandoned as "too breaking": **BW-22's fallback** (`Transliteration` loses `==` when #453 lands), **DEC-22** (collapsing the parser iterator types), **DEC-23** (`HeaderConflict` structured detail), **DEC-16** (the `Rule` interface's three renames). |
| **RT-35** | **ADR 0007 specifies a `ParseError.Column` field and prints "column 15" in its examples; `parser.ParseError` has never had one.** Either drop it from the ADR or add the field (additive, comparability-preserving). ADR/code divergence, not an API break. |
| **RT-36** | **Give `docs/reference/gedcom-7-coverage.md` a `Cardinality` column like its 5.5 sibling.** The 5.5 report is the *only* in-repo generated source of GEDCOM multiplicity, which is why no candidate in this audit rests on a 7.0 multiplicity claim — the 7.0 report explicitly disclaims measuring it (`:78-80`). With the column, audits of this kind become mechanical. `area:tooling`, independent of #479. |

---

# Sequencing answers

The release cannot be planned until these are settled, so they are answered here rather than
implied.

### Does this audit change #472's scope?

**Yes, three ways.**

1. **#472 must land before #473 — a hard constraint, not a preference.** `Association.Notes`
   (`gedcom/individual.go:180`) is the only bare `Notes []string` on one of our types that
   my-family actually calls, at `internal/gedcom/importer.go:1362-1363` (read) and
   `internal/gedcom/exporter.go:772` (write). `Association` has never been note-split, so there is
   no `InlineNotes` to migrate to until #472 adds one. Landing #473 first leaves the consumer with
   a removed field and no replacement.
2. **#472 should absorb BW-21**: retype `FamilyLink.Pedigree` to `[]string` (`{0:M}` in 5.5,
   `docs/reference/gedcom-5.5-coverage.md:326`) and add `FamilyLink.Status` for the `STAT`
   substructure that is `raw (accepted)` in both 7.0 and 5.5.1. The retype is v3-or-never and
   #472 is already breaking that type's comparability.
3. **#472 should *not* widen to cover `PlaceDetail`'s other unmodelled substructures.** `PLAC` also
   has `FONE {0:M}`, `ROMN {0:M}` and `SOUR {0:M}` unmodelled in 5.5
   (`gedcom-5.5-coverage.md:2249,2254,2255`) plus `TRAN`, `LANG` and `EXID` `raw (accepted)` in 7.0.
   Pass 002 suggested widening #472 "so they are not mistaken for a v4 break later" — but **the
   audit's answer is that they are not a v4 break at all**: once #472 removes `PlaceDetail`'s
   comparability, every one of those becomes an *additive* change, available in any minor forever
   after. File them as a normal follow-up issue and keep #472's scope tight. The same logic applies
   to `MediaLink`: #472's note fields are the only comparability-relevant change it needs.

### Does this audit change #473's scope?

**Yes, one way: the removal list goes from 11 to 13.** Add `Association.Notes`
(`gedcom/individual.go:180`) and `SourceRepositoryLink.Notes` (`gedcom/repository.go:89`). Neither
is in the deprecated set — they carry **no `Deprecated:` marker** and are the only two bare
`Notes []string` fields on the API that have never been note-split. The moment #472 lands they
become *undeprecated legacy*: a third representation of the same data, with no marker, no
supersession note and no removal plan — precisely the state the other 11 were in before #447
marked them. Removing all 13 together is **one** migration for my-family instead of two: once when
#472 gives `Association` its `InlineNotes`, and again in v4 when the field finally goes. The
fallback — mark both `Deprecated:` in v3 and remove in v4 — keeps the field my-family actively uses
alive for a whole major cycle, the worse outcome for the one consumer with a stake.

Pass 004 separately confirmed there is **no residual work** for #473 beyond the removals: the
interleaving that `recordNotesToEncode` (`encoder/entity_writer.go:76-87`) and `splitMatchesNotes`
(`:95-113`) exist to reconcile disappears with the fields, and both functions become trivial
(concatenate `NoteXRefs` then `InlineNotes`).

### Is #476 v3-gated?

**No. #476 does not block the v3 tag, and two independent passes agree on that.**

- The defect is a single assignment at `gedcom/xrefwalk.go:389`, and it is the only write inside
  the shared `walk*` family that `Visit` traverses.
- **BW-1 dissolves it.** Removing `Source.RepositoryRef` makes `walkSource` side-effect-free by
  construction: AC1 satisfied by construction, AC2 vacuous, AC3 unconstructible (the state it
  asserts on can no longer be expressed), AC4 still applies over one field instead of two, and AC5
  becomes *forced rather than optional* — `TestReachabilityFixtureIsComplete`'s self-cleaning loop
  (`gedcom/xrefwalk_fixture_test.go:145-149`) fails until the `exemptFromFixture` entry is deleted,
  and `"RepositoryRef"` must also come out of `carrierFieldNames()` at `:48`.
- **Action: close #476 as resolved-by BW-1**, carrying AC1 and AC5 across as verification steps. Do
  **not** implement #476's suggested `applyToRecords` post-pass first — it is machinery whose only
  purpose is to service a field being deleted in the same release.
- **Fallback, if and only if BW-1 is declined or v3 slips past the point where #476 needs an
  answer:** pass 004's non-breaking patch — move the re-sync out of `walkSource` into `Apply`'s own
  record loop (`applyToRecords`, `gedcom/xrefwalk.go:127-138`, which already handles definition
  sites separately "so `Visit` can ignore them"). No signature change, no documented-behaviour
  change; the code simply starts matching the doc comment already there.

### What gates the v3 tag?

**Nothing in Part C, and none of the four filed issues.** The v3 window is a scheduling constraint
on Part A only. #472 and #473 must land in v3 (they are removals), in that order. #476 closes
alongside BW-1. #477 is merged; its three comparability losses are already spent and are not
recoverable before v3.

**One caution on Part A's size.** Twenty-two entries is more than one release comfortably absorbs,
which is why they are tiered. Tier 1 is the defensible minimum: it honours every published
promise, closes two Lossless Representation violations, and fixes the one candidate with a proven
live consumer bug. Tier 3 is genuinely optional, and BW-20 (`effort:high`) is the single item most
likely to be worth deferring to v4 as an accepted break — its cost is high and none of its defects
reaches a caller today, because nobody calls it.

---

## Provenance

Six lens passes, each reconciling its own scope against a mechanical symbol inventory:

| Pass | Scope |
|---|---|
| 001 | inventory, comparability probe, `apidiff` vs `v2.4.0`, legacy-marker index, downstream call-site index |
| 002 | `gedcom/` type shape — comparability, zero/nil semantics |
| 003 | `gedcom/` surface hygiene — deprecated/legacy, naming |
| 004 | `gedcom/` contract fidelity and structural warts (probe-verified) |
| 005 | `validator/`, `charset/`, `version/` |
| 006 | `decoder/`, `encoder/`, `parser/` (zero values determined by execution, not by reading declarations) |
| 007 | root facade, `merge/`, `converter/`, `gedcom/testing/` |

> **The working artifacts these passes produced are not part of this repository, by design.**
> They lived under `.prompts/`, which `.gitignore` excludes, and they are gone. Two reasons they
> were not committed: they run to roughly 400 KB of intermediate notes, and they quote the private
> downstream consumer's source verbatim, which must not enter a public repo.
>
> **This report is therefore the sole surviving record**, and it is written to stand alone — the
> coverage attestation above restates every per-package count rather than referring out, and each
> finding carries its own `file:line` evidence. Where the text below names an artifact
> (`inventory.md`, `downstream-usage.md`, a `findings-*.md`), read it as *"the pass that produced
> this finding"*, not as a file you can open. Nothing in this report depends on retrieving one.

**Reproducing it.** The mechanical half is re-derivable at any commit: `make api-check` for the
`apidiff` delta, a `go/ast` walk for the symbol inventory (the counting rules are stated in the
coverage attestation), and `grep` for the deprecated/legacy index. The judgement half is not
mechanical and would need re-deriving from the code.

No `.go` file was modified in the production of this audit. `make preflight` was run against the
worktree after the report was written to confirm the audit changed no behaviour.

## Panel-review corrections

This report was reviewed by a six-persona panel after it was written and after its issues were
filed. Six corrections were applied and are marked inline where they occur:

| Correction | Entry |
|---|---|
| RT-1 cited `parser/iterator.go:196-208`, which is `RecordIteratorWithOffset` — the *correct* producer. Real defect is `:100`, `:127`, `:132`. | RT-1 |
| BW-4 claimed *no* decoded `Event.Type` can equal `EventOccupation`, and called it "proven". False — the GEDCOM 7 `NO` path synthesises one from an unvalidated tag value. | BW-4 |
| BW-14 framed the downstream migration as a delete-only edit to two shared converters. It is not: `Repository.Address` routes through the same helper. | BW-14 |
| BW-10 cited two `StrictMode` read sites; there are three, and the third gates a distinct behaviour. | BW-10 |
| RT-6 listed `EXID` among tags "nothing removes"; the EXID rewrite does remove them. | RT-6 |
| Part C's item count contradicted the summary table; `gedcom/header.go` citations were off by one. | Part C, BW-15, RT-2 |

Three further gaps were noted and addressed in place rather than as corrections: allocation cost
was never scored for the retype candidates (BW-13, BW-17), RT-4's fix instruction was ambiguous
enough to invert, and BW-5 left the write path unanswered.
