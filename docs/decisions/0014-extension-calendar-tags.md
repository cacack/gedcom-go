# ADR-014: Extension Calendars, Months and Epochs Kept as Tags

**Status**: Accepted
**Date**: 2026-11-26 ([#969](https://github.com/cacack/gedcom-go/issues/969))
**Context**: Extends [ADR-001](0001-custom-date-struct.md); narrows the gap [ADR-013](0013-payload-grammar-oracle.md) recorded as SI-018

## Decision

`gedcom.ParseDate` reads a GEDCOM 7.0 date whose calendar, month or epoch is an
extension tag, and keeps each tag as written in a new string field of `Date`. It
does not consult a `SCHMA`, and it gives such a date no place in time:

| Written | `Calendar` | Kept in |
|---------|------------|---------|
| `_MAYAN 5 _KIN 2000` (extension calendar) | `CalendarExtension` | `CalendarTag` `_MAYAN`; `MonthTag` `_KIN` |
| `_CALENDRIER 4 COMP 8` (extension calendar, standard-looking month) | `CalendarExtension` | `CalendarTag` `_CALENDRIER`; `MonthTag` `COMP` |
| `FRENCH_R 2 _JOUR 8` (extension month) | `CalendarFrenchRepublican` | `MonthTag` `_JOUR`, `Month` 0 |
| `1 JAN 2000 _AD` (extension epoch) | `CalendarGregorian` | `EpochTag` `_AD` |
| `_CALENDAR 8 _MONTH 190 _EPOCH` | `CalendarExtension` | all three |

- **An extension calendar's months and epochs are all tags.** The library does
  not know what `COMP` or `BCE` means on `_CALENDRIER`, so under an extension
  calendar every month goes to `MonthTag` and every epoch (`BCE` included) to
  `EpochTag`; `Month` stays 0 and `IsBC` false. Under a standard calendar a
  standard month sets `Month` and an extension month sets `MonthTag`.
- **A leading extension tag followed by more of the date is a calendar.** The
  grammar reads `_X 2000` either as calendar `_X` with year 2000 or as a Gregorian
  extension month; ParseDate takes the calendar, which accepts every string the
  month reading does, since the tail is the same. After a named calendar,
  including one written ahead of the modifier (`JULIAN _X 1700`), it is a month.
- **Tags are read as the grammar spells them.** `_jour` is not an `extTag`, so it
  is an error rather than a month; standard months keep their case-folding
  (SI-003).
- **Day and year are still integers, and 0 is still an error.** 7.0 leaves an
  extension month's day range to the extension, but 0 is `Date`'s "absent"
  value, so `FRENCH_R 0 _JOUR 8` stays an error (SI-018, narrowed).
- **`SCHMA` is not consulted.** `ParseDate` has no document, and the decoder does
  not resolve `_CALENDRIER` to `FRENCH_R` through `HEAD.SCHMA` either: the typed
  fields say what was written, never a reading of it.

What the arithmetic returns, for a date that `Date.HasExtension` reports:

| API | Result |
|-----|--------|
| `Bounds`, `AgeAt` | `ok` false, as for a date with no year; a range with such an endpoint too |
| `Compare` | sorts after every date without one; two such dates in byte order of `Original`, equal when it is equal. A stable order, not a chronological one |
| `IsBefore`, `IsAfter` | false when either date has one |
| `ToGregorian`, `ToTime`, `YearsBetween` | an error naming the extension |
| `Validate` | checks the year range and dual year, not the day: a month's length is unknown |

The converter keeps such a date unchanged and reports it as Preserved, as it
already did; the encoder writes `Event.Date` as before. Duplicate detection gives
two dates no year credit when either has an extension.

## Context

GEDCOM 7.0's `date = [calendar D] [[day D] month D] year [D epoch]` allows an
`extTag` in each of the three places, and a `SCHMA` may map one to a standard
URI (`_JOUR` is `month-COMP` in gedcom.io's `extensions.ged`). Until now
`ParseDate` rejected all of them, the largest group of disagreements the ABNF
oracle found, so a decoded event's `ParsedDate` was nil and the validator reported
`INVALID_DATE` for a valid 7.0 date.

## Considered Options

### Option A: Keep rejecting (status quo)

- **Pros**: No new fields.
- **Cons**: Valid 7.0 input has no typed date, and the validator calls it invalid.
- **Verdict**: Rejected.

### Option B: Resolve tags through the document's `SCHMA`

- **Pros**: `_CALENDRIER 4 COMP 8` would convert and compare as French
  Republican.
- **Cons**: `ParseDate` has no document, so the same value would parse two ways
  depending on the caller; a tag documented with a non-standard URI is no better
  off; no corpus outside gedcom.io's test file aliases a standard calendar.
- **Verdict**: Rejected for now. The kept tags leave room to add it on evidence.

### Option C: Keep the tags, leave arithmetic undefined (Selected)

- **Pros**: Lossless; every valid extension date parses; no result claims a
  calendar the library does not know.
- **Cons**: Five additive API members (`CalendarExtension`, three tag fields,
  `HasExtension`), and callers that switch on `Calendar`
  meet a new value.
- **Verdict**: Accepted.

## Consequences

### Positive

- The two `extensions.ged` dates parse, and their tags round-trip.
- SI-018 shrinks to day and year 0, and to the grammar check's `DateExact`.

### Negative

- A date that a `SCHMA` maps to a standard calendar still cannot be compared or
  converted.
- Decoded events that had a nil `ParsedDate` now have one, and the validator no
  longer reports them; a behavior change listed in api-stability.md.
- `ParseDate` reads every version alike (SI-002), so a 5.5.x placeholder that is
  an extension tag reads as one: Brother's Keeper's `__ JAN 1717` is calendar
  `__`, no longer reported as an invalid date, though it still has no place in
  time.

## References

- `gedcom.ParseDate`, `gedcom.Date.HasExtension`
- [Spec interpretation register](../governance/policies/spec-interpretation.md) SI-018
- [GEDCOM 7.0 Calendars](https://gedcom.io/specifications/FamilySearchGEDCOMv7.html#A-calendars)
