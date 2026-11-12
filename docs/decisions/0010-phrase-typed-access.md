# ADR-010: GEDCOM 7.0 PHRASE as a Sibling Field

**Status**: Accepted
**Date**: 2026-11-05
**Context**: Typed access for `PHRASE` across the 17 places GEDCOM 7.0 defines it ([#450](https://github.com/cacack/gedcom-go/issues/450))

## Decision

A `PHRASE` is typed as a `string` field named `<Value>Phrase`, placed beside the
typed value it qualifies: `PersonalName.TypePhrase`, `FamilyLink.PedigreePhrase`,
`Association.RolePhrase`, `Event.AgePhrase`, `Event.SortDatePhrase`,
`Attribute.SortDatePhrase`, `Family.HusbandPhrase`, `Family.WifePhrase` and
`MediaFile.MediaPhrase`. This is the shape `Association.Phrase` and
`CallNumber.MediaPhrase` already had.

Two cases do not follow the naming, because the value already owns a phrase:

- `DATE`'s `PHRASE` goes to `Date.Phrase` on the parsed date. An empty
  `DateValue` with a `PHRASE` — which 7.0 permits — decodes to a phrase-only
  `Date` (`IsPhrase` set, no components) instead of an `INVALID_VALUE`.
- `ASSO`'s own `PHRASE` stays `Association.Phrase`; `RolePhrase` is the
  `PHRASE` under its `ROLE`.

**A `PHRASE` is typed only where the value it qualifies is typed.** The encoder
writes a phrase only beneath its value, never on its own. `Date.Phrase` is the
exception: it is decode-only, because the encoder writes `DATE` from the raw
string and has never written its `PHRASE` from the entity. A decoded record
keeps the `PHRASE` in `Record.Tags`, which the encoder writes; a record encoded
from its entity alone (built in code, or with `Tags` cleared) loses it.

## Context

`PHRASE` is never an independent structure; it is free text qualifying its
superstructure's payload, usually an enumeration set to `OTHER`. The question
was whether to model it once, generically, or per site.

## Alternatives considered

- **A generic `Phrases map[string]string` keyed by path.** One field for every
  site, but stringly-typed, unchecked by the compiler, and a second way to reach
  data that every other typed field exposes directly.
- **Wrapping each value in a `{Value, Phrase}` struct.** Exact, but changes the
  type of fields such as `Family.Husband` and `PersonalName.Type`: a major break
  for every caller of the value to model a qualifier few files carry.

The sibling field is additive, reads like its neighbours, and costs one line per
site in the decoder, encoder, clone and `RequiresGEDCOM7`.

## Deferred sites

Where the qualified value has no typed field, its `PHRASE` stays in
`Record.Tags` (ADR 0003) and is typed when the value is:

| Site | Why deferred |
|------|--------------|
| `FAM.CHIL` | `Family.Children` is a `[]string`; a per-child phrase needs a new element type (a major) or a parallel slice that can fall out of step with it. |
| `FAM.ASSO`, `FAM.ASSO.ROLE` | `Family` has no typed associations. `RolePhrase` already covers `ROLE` under an individual's or event's `ASSO`; the coverage report measures `ASSO` under `FAM` only. |
| `INDI.ALIA` | `ALIA` has no typed field ([#375](https://github.com/cacack/gedcom-go/issues/375)). |
| `INDI.FAMC.STAT` | `STAT` under `FAMC` has no typed field. |
| `FAM.SOUR.EVEN` | A citation's `EVEN` has no typed field. |
| `SOUR.DATA.EVEN.DATE` | A source's `DATA.EVEN` has no typed field. |

## Consequences

- `make spec-coverage` reclassifies seven `PHRASE` rows as `typed`; the rows in
  the table above stay `raw (undiagnosed)` until their value is typed.
- Typing one of the deferred values brings its `PHRASE` with it, under the same
  `<Value>Phrase` rule. `INDI.ADOP.FAMC.ADOP` was deferred until
  [#740](https://github.com/cacack/gedcom-go/issues/740) typed the adoption
  `FAMC`; its `PHRASE` is now `Event.AdoptedByPhrase`.
  [#736](https://github.com/cacack/gedcom-go/issues/736) typed `AGE` on an
  attribute and the spouse `HUSB.AGE`/`WIFE.AGE` on a family's events and
  attributes; their `PHRASE`s are `Attribute.AgePhrase` and
  `HusbandAgePhrase`/`WifeAgePhrase` on `Event` and `Attribute`.
