# GEDCOM Date Formats: Comprehensive Research Report

**Project**: go-gedcom
**Date**: 2025-10-16
**Purpose**: Complete specification of date formats for GEDCOM parser implementation

---

## Executive Summary

GEDCOM genealogical data files support complex date representations across multiple calendar systems, with various modifiers for uncertainty, ranges, and periods. This document provides detailed format specifications, conversion formulas, parsing strategies, and real-world edge cases necessary for implementing a robust GEDCOM date parser.

---

## 1. Gregorian Calendar Date Formats

### 1.1 Standard Format Specification

**GEDCOM 5.5/5.5.1 Format**: `DD MMM YYYY`
- DD = day (1-2 digits, range 1-31)
- MMM = three-letter month abbreviation (uppercase)
- YYYY = four-digit year

**GEDCOM 7.0 Format**: Same as 5.5, but with stricter grammar enforcement

**Month Abbreviations** (case-insensitive in practice):
```
JAN, FEB, MAR, APR, MAY, JUN, JUL, AUG, SEP, OCT, NOV, DEC
```

**Examples**:
```
25 DEC 2020
1 JAN 1900
14 FEB 1890
```

### 1.2 Date Modifiers

Date modifiers express uncertainty or approximation:

| Modifier | Meaning | Example | Use Case |
|----------|---------|---------|----------|
| **ABT** | About/Approximate | `ABT 1850` | Exact date unknown but near specified date |
| **CAL** | Calculated | `CAL 1875` | Mathematically derived from other events |
| **EST** | Estimated | `EST 1820` | Estimated based on algorithm using other data |
| **BEF** | Before | `BEF 12 JUN 1900` | Event occurred before this date |
| **AFT** | After | `AFT 1 JAN 1850` | Event occurred after this date |

**Semantic Differences by Version**:

**GEDCOM 5.5/5.5.1**:
- ABT = "About, meaning the date is not exact"
- CAL = "Calculated mathematically"
- EST = "Estimated based on an algorithm"

**GEDCOM 7.0**:
- BEF = "Exact date unknown, but no later than specified date"
- AFT = "Exact date unknown, but no earlier than specified date"
- ABT/CAL/EST have same general meanings

### 1.3 Date Ranges

**BET...AND Syntax**: Indicates event occurred once within a bounded timeframe

**Format**: `BET <date1> AND <date2>`

**Examples**:
```
BET 1 JAN 1874 AND 16 JAN 1874
BET 3 MAR 1833 AND 3 MAR 1843
BET 1850 AND 1860
```

**Version Differences**:
- **GEDCOM 5.5.1**: "Event happened some time between date 1 AND date 2"
- **GEDCOM 7.0**: More explicit - BET means "no earlier than" and AND means "no later than"

**Semantic Equivalences** (short form expansion):
```
1852           ≡  BET 1 JAN 1852 AND 31 DEC 1852
JAN 1920       ≡  BET 1 JAN 1920 AND 31 JAN 1920
```

### 1.4 Date Periods

**FROM...TO Syntax**: Indicates event lasting over a time interval (not a single point)

**Format Options**:
- `FROM <date>` - open-ended start
- `TO <date>` - open-ended end
- `FROM <date> TO <date>` - bounded period

**Examples**:
```
FROM 1851                    (started 1851, ongoing/unknown end)
FROM FEB 1852                (partial date start)
FROM 3 MAR 1853 TO 10 APR 1855
TO 1872                      (ended 1872, unknown start)
```

**Use Cases**:
- Occupation periods: `FROM 1880 TO 1920`
- Residence periods: `FROM JAN 1900 TO DEC 1905`
- Military service: `FROM 1914 TO 1918`

### 1.5 Partial Dates

GEDCOM supports dates with missing components:

**Year Only**:
```
1850
1920
2000
```

**Month and Year** (no day):
```
JAN 1900
FEB 1802
DEC 2020
```

**Day and Month** (no year - not standard, becomes phrase):
```
25 DEC         → Must be stored as date phrase in GEDCOM 7.0
```

**Critical Rule** (GEDCOM 7.0): Every date MUST have a year. If no year is known, omit the entire date and use PHRASE substructure instead.

**Sorting Strategy**: When sorting partial dates, use zero for missing components:
- `JAN 1920` → treat as `1 JAN 1920` (day = 1)
- `1850` → treat as `1 JAN 1850` (day = 1, month = 1)

---

## 2. Hebrew Calendar Support

### 2.1 Calendar Escape Sequence

**Format**: `@#DHEBREW@ <day> <month> <year>`

**Example**:
```
@#DHEBREW@ 13 CSH 5760
```

### 2.2 Hebrew Month Names and GEDCOM Codes

Hebrew calendar months in order with GEDCOM three-letter codes:

| Month # | Hebrew Name | GEDCOM Code | Approximate Gregorian |
|---------|-------------|-------------|----------------------|
| 7 | Tishrei | **TSH** | Sep-Oct |
| 8 | Cheshvan/Marcheshvan | **CSH** | Oct-Nov |
| 9 | Kislev | **KSL** | Nov-Dec |
| 10 | Tevet | **TVT** | Dec-Jan |
| 11 | Shevat | **SHV** | Jan-Feb |
| 12 | Adar | **ADR** | Feb-Mar |
| 13 | Adar II (leap years) | **ADS** | Mar-Apr |
| 1 | Nisan | **NSN** | Mar-Apr |
| 2 | Iyar | **IYR** | Apr-May |
| 3 | Sivan | **SVN** | May-Jun |
| 4 | Tammuz | **TMZ** | Jun-Jul |
| 5 | Av | **AAV** | Jul-Aug |
| 6 | Elul | **ELL** | Aug-Sep |

**Critical Notes**:
1. Hebrew calendar is lunisolar (months follow moon, years adjusted to solar)
2. Year numbering: 5786 = 2025/2026 CE (approximately)
3. Hebrew dates are right-to-left: day, month, year
4. Days begin at sunset, not midnight
5. Leap years add an extra month (Adar II / ADS)

### 2.3 Hebrew-Gregorian Conversion

**Complexity**: Hebrew-Gregorian conversion requires complex astronomical calculations due to:
- Variable month lengths (29 or 30 days)
- Leap year cycle (7 leap years in every 19-year cycle)
- Year length varies (353, 354, 355, 383, 384, or 385 days)

**Conversion Strategy for Implementation**:
1. **Use lookup tables** for common date ranges (1700-2100 CE)
2. **Julian Day Number (JDN)** as intermediate format
3. **External libraries** recommended for precise conversion (e.g., algorithm from "Calendrical Calculations" by Dershowitz & Reingold)

**Practical Example**:
```
Gregorian: 16 OCT 2025  →  Hebrew: 24 TSH 5786
```

**Implementation Warning**: Since Hebrew days begin at sunset, events from sunset to midnight will be one day earlier in Hebrew calendar. GEDCOM implementations should:
- Parse Hebrew dates as-written in source documents
- Warn users about potential sunset boundary issues
- Preserve original date string for round-trip fidelity

---

## 3. French Republican Calendar Support

### 3.1 Calendar Escape Sequence

**Format**: `@#DFRENCH R@ <day> <month> <year>`

**Historical Context**:
- Used: 22 September 1792 - 31 December 1805 (12 years)
- Geographic scope: France, Belgium, Luxembourg, parts of Netherlands, Germany, Switzerland, Italy
- Critical for genealogy: Civil registration records from this period use only Republican dates

### 3.2 French Republican Month Names and GEDCOM Codes

The calendar had 12 months of 30 days each, plus 5-6 complementary days:

| Month # | French Name | GEDCOM Code | Meaning | Season | Approx. Gregorian |
|---------|-------------|-------------|---------|--------|------------------|
| 1 | Vendémiaire | **VEND** | Grape harvest | Autumn | Sep 22-Oct 21 |
| 2 | Brumaire | **BRUM** | Fog | Autumn | Oct 22-Nov 20 |
| 3 | Frimaire | **FRIM** | Frost | Autumn | Nov 21-Dec 20 |
| 4 | Nivôse | **NIVO** | Snow | Winter | Dec 21-Jan 19 |
| 5 | Pluviôse | **PLUV** | Rain | Winter | Jan 20-Feb 18 |
| 6 | Ventôse | **VENT** | Wind | Winter | Feb 19-Mar 20 |
| 7 | Germinal | **GERM** | Germination | Spring | Mar 21-Apr 19 |
| 8 | Floréal | **FLOR** | Flowers | Spring | Apr 20-May 19 |
| 9 | Prairial | **PRAI** | Meadows | Spring | May 20-Jun 18 |
| 10 | Messidor | **MESS** | Harvest | Summer | Jun 19-Jul 18 |
| 11 | Thermidor | **THER** | Heat | Summer | Jul 19-Aug 17 |
| 12 | Fructidor | **FRUC** | Fruits | Summer | Aug 18-Sep 16 |
| - | Sans-culottides | **COMP** | Complementary days | - | Sep 17-21/22 |

**Day Structure**:
- 30 days per month (numbered 1-30)
- Each 10-day period = décade (replacing weeks)
- 5 complementary days (6 in leap years) after Fructidor

### 3.3 French Republican-Gregorian Conversion

**Official Start Date**: 22 September 1792 = 1 Vendémiaire An I (Year 1)

**Year Numbering**: Years counted from proclamation of First Republic
- An I = 1792/1793
- An XIV = 1805/1806 (last year used)

**Conversion Formulas**:

**Republican → Gregorian** (simplified):
```
Base: September 22, 1792 (Gregorian) = 1 VEND 1 (Republican)

Gregorian_Date = Base_Date + (year - 1) × 365 + leap_days + month_offset + (day - 1)

Where:
  month_offset = (month - 1) × 30
  leap_days = number of leap years since An I
```

**Leap Years in Republican Calendar**:
- Original rule: Years divisible by 4 (like Gregorian)
- Complex later modifications never fully standardized
- Most tools use Romme's rule: years divisible by 4, except century years not divisible by 400

**Conversion Tools**:
- GEDCOM Date Calculator supports all four calendar systems
- Online converters available (napoleon-empire.org, geditcom.com)
- Complexity: requires accurate leap year determination

**Example Conversions**:
```
@#DFRENCH R@ 15 VEND 3     →  Gregorian: 6 OCT 1794
@#DFRENCH R@ 1 PLUV 8      →  Gregorian: 20 JAN 1800
@#DFRENCH R@ 30 FRUC 13    →  Gregorian: 16 SEP 1805
```

---

## 4. Julian Calendar Support

### 4.1 Calendar Escape Sequence

**Format**: `@#DJULIAN@ <day> <month> <year>`

**Example**:
```
@#DJULIAN@ 25 DEC 1700
```

### 4.2 Historical Context and Gregorian Adoption

**Julian Calendar**: Introduced by Julius Caesar in 45 BCE, used throughout Europe until Gregorian reform

**Gregorian Adoption Timeline** (critical for genealogy):

| Region | Adoption Date | Days Dropped |
|--------|--------------|--------------|
| Catholic Europe (Italy, Spain, Portugal) | October 1582 | 10 days |
| Germany (Catholic states) | 1583-1585 | 10 days |
| Protestant Germany | 1700 | 10 days |
| Great Britain & Colonies (incl. America) | September 1752 | 11 days |
| Sweden | 1753 | 11 days |
| Russia | 1918 | 13 days |
| Greece | 1923 | 13 days |

**Key Date Example** (Britain/American colonies):
```
September 2, 1752 (Julian) → September 14, 1752 (Gregorian)
(September 3-13, 1752 never existed in British calendar)
```

### 4.3 Julian-Gregorian Conversion Formula

**Difference Between Calendars**:

The Julian calendar gains approximately 1 day per 128 years compared to solar year.

**Conversion offset by century**:
- 1500s: +10 days (Julian → Gregorian)
- 1600s: +10 days
- 1700-1799: +11 days
- 1800-1899: +12 days
- 1900-2099: +13 days

**Formula**:
```
Gregorian_Date = Julian_Date + offset_days

Where offset_days = 13 + (Julian_Year / 100) - (Julian_Year / 400) - 2
```

**Julian Day Number (JDN) Method** (recommended):
1. Convert Julian date → JDN
2. Convert JDN → Gregorian date

**Example Conversions**:
```
@#DJULIAN@ 25 DEC 1700  →  @#DGREGORIAN@ 5 JAN 1701  (11 days)
@#DJULIAN@ 1 JAN 1900   →  @#DGREGORIAN@ 14 JAN 1900 (13 days)
```

### 4.4 Dual Dating and New Year Issues

**Historical Issue**: Until 1752, England and colonies considered March 25 as New Year's Day (not January 1)

**Dual Dating Format**: `DD MMM YYYY/YY`

**Examples**:
```
21 FEB 1750/51     Meaning: Feb 21, 1750 (Old Style) = Feb 21, 1751 (New Style)
10 JAN 1680/81     Meaning: Jan 10, 1680 (Old Style) = Jan 10, 1681 (New Style)
```

**GEDCOM Dual Dating Support**:
- **GEDCOM 5.5/5.5.1**: Dual dating allowed in Gregorian calendar (slash format in YYYY/YY)
- **GEDCOM 7.0**: Slash format deprecated; use DATE with PHRASE substructure instead

**Critical Parsing Issue**: The dual date `21 FEB 1750/51` in GEDCOM is technically marked as Gregorian (default), but historically it's a Julian date. GEDCOM specifications have this backwards. Best practice:
- Parse slash dates as written
- Assume Julian calendar context for pre-1752 dates
- Preserve original string for display
- Issue validation warning about ambiguity

**Genealogist Best Practice**: "Keep the date found in the actual record (e.g., 1 March), but clarify the year with double dating like 1615/16."

**Dual-Calendar vs. Dual-Style** (important distinction):
- **Dual-style date**: Same date in Old Style vs. New Style year numbering (21 FEB 1750/51)
- **Dual-calendar date**: Same day in Julian vs. Gregorian calendar (2/12 MAR 1608 = 2 Mar Julian, 12 Mar Gregorian)
- GEDCOM supports dual-style, NOT dual-calendar

---

## 5. Date Ranges and Periods Specification

### 5.1 Complete Grammar

**GEDCOM 5.5/5.5.1 Grammar**:
```
DATE_VALUE =
    [ <date> ] |
    [ <date_phrase> ] |
    [ <date_range> ] |
    [ <date_period> ] |
    [ <date_approx> <date> ] |
    INT <date> (<date_phrase>)

date_range = BEF <date> | AFT <date> | BET <date> AND <date>
date_period = FROM <date> | TO <date> | FROM <date> TO <date>
date_approx = ABT | CAL | EST
date_phrase = (<text>)
```

**GEDCOM 7.0 Grammar**:
```
DateValue = [date | DatePeriod | dateRange | dateApprox]

date = [calendar D] [[day D] month D] year [D epoch]
DatePeriod = ["TO" D date] | "FROM" D date [D "TO" D date]
dateRange = "BET" D date D "AND" D date | "BEF" D date | "AFT" D date
dateApprox = ("ABT" | "CAL" | "EST") D date
```

### 5.2 Complete Examples by Category

**Single Dates**:
```
25 DEC 2020
JAN 1900
1850
```

**Approximate Dates**:
```
ABT 1850
CAL 12 MAY 1875
EST 1820
```

**Before/After**:
```
BEF 1900
AFT 15 JUN 1850
BEF JAN 1970
```

**Ranges** (single event within timeframe):
```
BET 1850 AND 1860
BET 1 JAN 1900 AND 31 DEC 1900
BET FEB 1920 AND MAR 1920
```

**Periods** (duration/interval):
```
FROM 1880
TO 1920
FROM 1880 TO 1920
FROM JAN 1900 TO DEC 1905
```

**Calendar-specific**:
```
@#DJULIAN@ 25 DEC 1700
@#DHEBREW@ 13 CSH 5760
@#DFRENCH R@ 15 VEND 3
@#DGREGORIAN@ 1 JAN 2000     (explicit, though default)
```

**Interpreted Dates** (GEDCOM 5.5 only, removed in 7.0):
```
INT 1900 (probably around 1900)
```

**Date Phrases** (GEDCOM 5.5):
```
(unknown)
(sometime in the 1800s)
(before the war)
```

**Date Phrases** (GEDCOM 7.0 - uses substructure):
```
2 DATE
3 PHRASE 5 January (year unknown)
```

### 5.3 Edge Cases and Ambiguities

**Case Sensitivity**:
- Standard: Month abbreviations should be uppercase
- Reality: Most software accepts case-insensitive (Jan, jan, JAN)
- Recommendation: Parse case-insensitively, generate uppercase

**Whitespace Handling**:
- GEDCOM 7.0 explicitly defines whitespace rules
- Single space separates components
- Leading/trailing whitespace should be trimmed
- Multiple spaces should be normalized to single space

**Invalid Day Numbers**:
```
32 JAN 2020        Invalid: day 32 doesn't exist
30 FEB 2020        Invalid: February has 28/29 days
31 APR 2020        Invalid: April has 30 days
```

Strategy: Validation should catch these, but parsing can still extract components for error reporting.

**Missing Components in Ranges**:
```
BET 1850 AND FEB 1860        Valid: partial dates allowed
BET MAR AND JUN 1920         Invalid in GEDCOM 7.0: year required
```

**Ambiguous Formats** (not in spec, but found in wild):
```
1/5/2020           Ambiguous: Jan 5 or May 1? → Reject, require GEDCOM format
2020-01-05         ISO format → Reject, require GEDCOM format
```

---

## 6. Partial Dates and Missing Data

### 6.1 Supported Partial Date Formats

**Year Only** (most common partial date):
```
1850
2000
```
Use: When only year is known (e.g., "born in 1850")

**Month and Year**:
```
JAN 1900
DEC 2020
FEB 1802
```
Use: When day is unknown but month/year known

**Day and Month** (NO YEAR):
- GEDCOM 5.5: Can use date phrase: `(14 Nov)`
- GEDCOM 7.0: Must use PHRASE substructure, not DATE payload

### 6.2 Sorting and Comparison Strategy

**Sorting Rule**: When comparing dates with missing components, use earliest possible date:

| Input | Assumed for Sorting | Rationale |
|-------|---------------------|-----------|
| `1850` | `1 JAN 1850` | First day of year |
| `MAR 1920` | `1 MAR 1920` | First day of month |
| `BET 1850 AND 1860` | `1 JAN 1850` | Lower bound |
| `AFT 1900` | `1 JAN 1901` | Day after |
| `BEF 1900` | `31 DEC 1899` | Day before |
| `ABT 1850` | `1 JAN 1850` | Best estimate |
| `FROM 1880` | `1 JAN 1880` | Start date |

### 6.3 Display and Formatting

**Display Strategies**:
1. **Verbatim**: Show exactly as written: `JAN 1920`
2. **Expanded**: Show with placeholders: `? JAN 1920`
3. **Range**: Show as range: `1-31 JAN 1920`

**Recommendation for Library**: Preserve original string and provide helper methods for different display formats.

---

## 7. Invalid Date Handling Strategies

### 7.1 Categories of Invalid Dates

**Syntactically Invalid**:
- Wrong month abbreviation: `32 XYZ 2020`
- Invalid format: `2020-01-05` (ISO format)
- Missing required components (GEDCOM 7.0): `14 NOV` (no year)

**Semantically Invalid**:
- Impossible day: `32 JAN 2020`
- Wrong month length: `30 FEB 2020`
- Date doesn't exist: `10 SEP 1752` (skipped in British calendar)

**Historically Questionable**:
- Future dates in historical records
- Dates before birth/after death
- Child born before parents

### 7.2 Handling Strategies

**Strict Validation Mode**:
- Reject all invalid dates
- Return detailed error messages
- Fail parsing if dates don't conform

**Lenient Parsing Mode**:
- Parse what's possible, store as phrase if invalid
- Preserve original string
- Tag with "unparseable" flag
- Continue processing file

**GEDCOM Strategy** (from spec):
> "When Gedcom Publisher encounters an invalid date value in a GEDCOM file, it treats the date value as a date phrase."

**Recommended Multi-Tier Approach**:
1. **Parse**: Accept liberally, extract components
2. **Validate**: Check semantic correctness
3. **Store**: Preserve original + structured components + validity flag
4. **Report**: Generate warnings for invalid dates with line numbers

### 7.3 Error Messages

**Good Error Messages** include:
- Line number
- Original date string
- Specific problem
- Suggested correction

**Examples**:
```
Line 45: Invalid date "32 JAN 2020" - day must be 1-31
Line 67: Invalid month "XYZ" in date "10 XYZ 1900" - expected JAN, FEB, MAR, ...
Line 89: Missing year in date "14 NOV" (required in GEDCOM 7.0)
Line 102: Date "30 FEB 2020" - February has only 28/29 days
```

---

## 8. Date Parsing Edge Cases

### 8.1 Leap Year Handling

**Leap Year Rules** (Gregorian):
- Divisible by 4: leap year
- EXCEPT divisible by 100: not leap year
- EXCEPT divisible by 400: leap year

**Edge Dates**:
```
29 FEB 2000        Valid (divisible by 400)
29 FEB 1900        Invalid (divisible by 100, not 400)
29 FEB 2024        Valid (divisible by 4)
29 FEB 2023        Invalid (not divisible by 4)
```

**Julian Calendar**: Only "divisible by 4" rule (simpler)

**Hebrew Calendar**: Variable leap years (19-year cycle, years 3,6,8,11,14,17,19 are leap)

**French Republican**: Originally divisible by 4; later Romme's rule (like Gregorian)

### 8.2 Calendar Transition Dates

**British Calendar 1752**:
```
2 SEP 1752 → 14 SEP 1752  (September 3-13 never existed)
```

Any dates in the gap should be flagged as impossible.

**Other transitions**: Russia 1918, Greece 1923, etc. (see section 4.2)

### 8.3 B.C./B.C.E. Dates

**GEDCOM Format**: Year can be negative or use "B.C." suffix

**Examples**:
```
500 B.C.
-500
100 BCE          (not standard, treat as phrase)
```

**Sorting Issues**:
- 100 B.C. comes AFTER 200 B.C.
- Use negation: -100 > -200

**Conversion Strategy**:
```
If year contains "B.C." suffix:
    year_number = -(parsed_number)
```

**Year Zero Issue**: No year zero in historical calendars
- 1 B.C. → 1 C.E. (no year zero)
- Astronomical year numbering uses year zero
- Be explicit in documentation

### 8.4 Very Large Year Numbers

**GEDCOM allows** up to 4-digit years, but software may encounter:
```
10000           (5-digit year)
1000000         (7-digit year)
```

**Strategy**:
- Set reasonable maximum (e.g., year 9999 for storage)
- Treat out-of-range years as date phrases
- Document limitations

### 8.5 Unicode and Encoding Issues

**Month Names**: Should be ASCII-only (JAN, FEB, etc.)

**Found in Wild**:
- UTF-8 encoded text: `25 décembre 2020` (French)
- Accented characters in phrases: `(né environ 1850)`
- Non-Latin scripts: Cyrillic, Hebrew characters

**Strategy**:
- Standardize to ASCII month abbreviations
- Preserve original in phrase if non-standard
- Warn about non-ASCII in DATE field

### 8.6 Case Sensitivity in Month Names

**Standard**: Uppercase (`JAN`, `FEB`, `MAR`)

**Found in Wild**:
- Title case: `Jan`, `Feb`, `Mar`
- Lowercase: `jan`, `feb`, `mar`
- Mixed case: `jAn`, `FEb`

**Best Practice** (from research):
> "It would be unwise for a genealogy application developer to create a GEDCOM reader that doesn't accept GEDCOM files from popular applications merely because it doesn't like the casing style of the month abbreviations."

**Recommendation**: Parse case-insensitively, generate uppercase on output.

### 8.7 Whitespace Variations

**Standard**: Single space between components: `25 DEC 2020`

**Found in Wild**:
- Multiple spaces: `25  DEC  2020`
- Tabs: `25→DEC→2020`
- Leading/trailing spaces: ` 25 DEC 2020 `

**Strategy**:
- Trim leading/trailing whitespace
- Normalize internal whitespace to single space
- Accept tabs as whitespace

### 8.8 Date Phrase Handling

**GEDCOM 5.5 Format**: Phrases in parentheses within DATE
```
(unknown)
(sometime in 1800s)
(before the war)
(circa 1850)
```

**GEDCOM 7.0 Format**: Separate PHRASE substructure
```
2 DATE
3 PHRASE sometime in the 1800s
```

**Mixed Format** (common issue):
```
ABT 1850 (probably)       Valid in 5.5, invalid in 7.0
```

**Strategy**:
- Parse according to detected version
- Extract phrase text if present
- Store phrase separately from structured date

---

## 9. Real-World GEDCOM Date Examples

### 9.1 Birth Date Examples

```
1 BIRT
2 DATE 25 DEC 1850

1 BIRT
2 DATE ABT 1850
2 PLAC England

1 BIRT
2 DATE BET 1848 AND 1852
2 NOTE Estimated from census records

1 BIRT
2 DATE @#DHEBREW@ 15 NSN 5623

1 BIRT
2 DATE (unknown)
```

### 9.2 Death Date Examples

```
1 DEAT
2 DATE 15 MAR 1920

1 DEAT
2 DATE BEF 1925
2 NOTE Not found in 1925 census

1 DEAT
2 DATE AFT 1 JAN 1930
2 PLAC Somewhere in California

1 DEAT Y
```
(Note: `1 DEAT Y` means "died, date unknown")

### 9.3 Marriage Date Examples

```
1 MARR
2 DATE 14 FEB 1890
2 PLAC St. Mary's Church, London

1 MARR
2 DATE ABT 1875

1 MARR
2 DATE @#DJULIAN@ 3 MAY 1725
2 PLAC England
```

### 9.4 Occupation Period Examples

```
1 OCCU Farmer
2 DATE FROM 1880 TO 1920
2 PLAC Iowa

1 OCCU Blacksmith
2 DATE FROM 1850

1 OCCU Teacher
2 DATE FROM 1900 TO 1905
```

### 9.5 Residence Period Examples

```
1 RESI
2 DATE FROM 1 JAN 1900 TO 31 DEC 1910
2 ADDR 123 Main Street
3 CITY New York

1 RESI
2 DATE FROM 1920
2 PLAC Chicago, Illinois
```

### 9.6 Calendar-Specific Examples

**Jewish Records**:
```
1 BIRT
2 DATE @#DHEBREW@ 3 TVT 5650
2 PLAC Warsaw

1 DEAT
2 DATE @#DHEBREW@ 15 ELL 5710
```

**French Revolutionary Records**:
```
1 BIRT
2 DATE @#DFRENCH R@ 12 VEND 3
2 PLAC Paris

1 MARR
2 DATE @#DFRENCH R@ 25 PRAI 8
2 PLAC Lyon
```

**Julian Calendar (pre-1752 England)**:
```
1 BIRT
2 DATE @#DJULIAN@ 25 FEB 1720/21
2 PLAC London, England

1 DEAT
2 DATE @#DJULIAN@ 10 MAR 1724
```

---

## 10. Best Practices for Date Validation and Parsing

### 10.1 Parsing Strategy

**Recommended Multi-Stage Parser**:

**Stage 1: Tokenization**
- Split by whitespace
- Identify calendar escape if present
- Extract modifier keywords (ABT, BEF, AFT, BET, AND, FROM, TO)
- Identify components: day, month, year

**Stage 2: Component Validation**
- Validate month abbreviation
- Validate day range (1-31, calendar-specific)
- Validate year (reasonable range)
- Check leap year for Feb 29

**Stage 3: Semantic Validation**
- Check range ordering (BET date1 AND date2: date1 < date2)
- Check period ordering (FROM date1 TO date2: date1 < date2)
- Verify calendar-specific rules

**Stage 4: Structure Creation**
- Build typed date object
- Preserve original string
- Store validity flags
- Generate warnings list

### 10.2 Data Structure Recommendations

**Go Structure** (from data-model.md):
```go
type Date struct {
    Raw      string     // Original date string from GEDCOM
    Parsed   time.Time  // Parsed Go time (may be zero if unparseable)
    Calendar string     // Gregorian, Hebrew, French Republican, Julian
    Modifier string     // ABT (about), BEF (before), AFT (after), etc.
    Valid    bool       // Whether date passed validation
    Warnings []string   // Any validation warnings
}
```

**Enhanced Structure** (recommended):
```go
type Date struct {
    Raw      string          // "25 DEC 1850", "@#DHEBREW@ 13 CSH 5760"
    Calendar Calendar        // Enum: Gregorian, Julian, Hebrew, FrenchR
    Modifier DateModifier    // Enum: Exact, About, Before, After, etc.

    // Parsed components (nil if not present)
    Day      *int            // 1-31
    Month    *int            // 1-12 (or 1-13 for Hebrew leap years)
    Year     int             // Required in GEDCOM 7.0

    // For ranges/periods
    IsRange  bool            // BET...AND
    IsPeriod bool            // FROM...TO
    EndDate  *Date           // Second date in range/period

    // Validation
    Valid    bool
    Warnings []DateWarning

    // Conversion
    GregorianDate time.Time  // Converted to Gregorian (if possible)
    JDN          int         // Julian Day Number (universal)
}

type Calendar int
const (
    Gregorian Calendar = iota
    Julian
    Hebrew
    FrenchRepublican
    Unknown
)

type DateModifier int
const (
    Exact DateModifier = iota  // No modifier
    About                      // ABT
    Calculated                 // CAL
    Estimated                  // EST
    Before                     // BEF
    After                      // AFT
    Between                    // BET...AND
    From                       // FROM
    To                         // TO
    FromTo                     // FROM...TO
)

type DateWarning struct {
    Code    WarningCode
    Message string
}
```

### 10.3 Validation Rules Summary

**Level 1 - Syntax Validation**:
- [ ] Calendar escape valid (if present)
- [ ] Modifier keyword valid
- [ ] Month abbreviation recognized
- [ ] Day is numeric (if present)
- [ ] Year is numeric
- [ ] Whitespace properly formatted

**Level 2 - Semantic Validation**:
- [ ] Day in valid range for month (1-28/29/30/31)
- [ ] Leap year rules applied correctly
- [ ] Year in reasonable range (-10000 to 10000)
- [ ] Range dates ordered correctly (date1 < date2)
- [ ] Calendar-specific rules (Hebrew leap months, French complementary days)

**Level 3 - Logical Validation**:
- [ ] Date not in calendar transition gap (e.g., Sep 3-13, 1752 for British)
- [ ] Date plausible for event type (birth before death, etc.)
- [ ] Cross-reference validation (child born after parents, etc.)

**Level 4 - Version-Specific Validation**:
- [ ] GEDCOM 7.0: Year required for all dates
- [ ] GEDCOM 7.0: Phrases in PHRASE substructure, not DATE payload
- [ ] GEDCOM 5.5: Date phrases allowed in parentheses
- [ ] Dual dating format matches version rules

### 10.4 Error Recovery Strategies

**Strategy 1: Graceful Degradation**
- Parse what's possible
- Store unparseable portions as phrase
- Continue processing file

**Strategy 2: Best-Effort Parsing**
- Attempt multiple parse strategies
- Fuzzy matching for month names (JNA → JAN?)
- Infer missing components from context

**Strategy 3: Strict Mode**
- Fail on first invalid date
- Require perfect conformance
- Return detailed error location

**Recommendation**: Provide mode configuration
```go
type ParseOptions struct {
    Strict          bool  // Fail on errors vs. warnings
    AllowCaseInsensitive bool
    NormalizeWhitespace  bool
    InferMissingComponents bool
    MaxYear          int   // Default 9999
}
```

### 10.5 Testing Strategies

**Unit Tests - Date Parser**:
- Test each calendar system independently
- Test all modifiers (ABT, BEF, AFT, BET/AND, FROM, TO)
- Test partial dates (year only, month-year)
- Test invalid dates
- Test edge cases (leap years, calendar transitions)

**Integration Tests**:
- Parse real GEDCOM files from multiple sources
- Compare against known-good parsers
- Test round-trip: parse → format → parse

**Property-Based Tests**:
- Generate random valid dates, ensure parsing succeeds
- Generate random invalid dates, ensure proper errors
- Test date ordering invariants

**Benchmark Tests**:
- Parse 10,000 dates, measure time
- Test memory allocation patterns
- Optimize hot paths

---

## 11. Conversion Formulas and Algorithms

### 11.1 Julian Day Number (JDN)

**Purpose**: Universal intermediate format for calendar conversion

**Definition**: Number of days since January 1, 4713 BCE (Julian proleptic calendar)

**Gregorian → JDN** (algorithm):
```
a = (14 - month) / 12
y = year + 4800 - a
m = month + 12*a - 3

JDN = day + (153*m + 2)/5 + 365*y + y/4 - y/100 + y/400 - 32045
```

**JDN → Gregorian** (algorithm):
```
a = JDN + 32044
b = (4*a + 3) / 146097
c = a - (146097*b) / 4
d = (4*c + 3) / 1461
e = c - (1461*d) / 4
m = (5*e + 2) / 153

day = e - (153*m + 2)/5 + 1
month = m + 3 - 12*(m/10)
year = 100*b + d - 4800 + (m/10)
```

**Julian → JDN** (simpler, no 100/400 year corrections):
```
a = (14 - month) / 12
y = year + 4800 - a
m = month + 12*a - 3

JDN = day + (153*m + 2)/5 + 365*y + y/4 - 32083
```

### 11.2 Hebrew Calendar Conversion

**Complexity**: Hebrew calendar is extremely complex:
- 19-year Metonic cycle for leap years
- Variable year lengths (353-385 days)
- Complex rules for year structure

**Recommendation**: Use established library or lookup tables
- Reference: "Calendrical Calculations" by Dershowitz & Reingold
- Algorithm too complex for inline implementation (~200+ lines)

**Simplified Approach for GEDCOM Library**:
1. Parse Hebrew date components
2. Store as-is (don't convert on parse)
3. Provide conversion method using external algorithm
4. Cache conversions in lookup table for performance

### 11.3 French Republican Calendar Conversion

**Simplified Algorithm** (Fr.Rep. → Gregorian):

```
// Republican date: day, month, year
// Base: 22 Sep 1792 = 1 VEND 1

days_since_epoch = (year - 1) * 365
                 + leap_days(year)
                 + (month - 1) * 30
                 + (day - 1)

// Add to Gregorian base date (22 Sep 1792)
gregorian_date = date(1792, 9, 22) + days_since_epoch

// Leap days calculation (using Romme's rule)
leap_days(year):
    if year <= 13:
        return year / 4  // Historical actual leap years
    else:
        // Hypothetical using Romme's rule
        return (year / 4) - (year / 100) + (year / 400)
```

**Gregorian → French Republican**:
```
days_diff = gregorian_date - date(1792, 9, 22)

year = days_diff / 365  // Approximate
// Refine using exact leap year calculations

day_of_year = days_diff % 365_or_366

if day_of_year <= 360:
    month = (day_of_year / 30) + 1
    day = (day_of_year % 30) + 1
else:
    month = 13  // COMP
    day = day_of_year - 360
```

**Note**: French Republican calendar was only used 1792-1805, so practical implementation can use lookup table for this limited range.

---

## 12. Implementation Recommendations for go-gedcom

### 12.1 Package Structure

```
gedcom/
  date/
    date.go              // Core Date type and parsing
    gregorian.go         // Gregorian calendar functions
    julian.go            // Julian calendar functions
    hebrew.go            // Hebrew calendar functions
    french.go            // French Republican calendar functions
    convert.go           // Conversion functions (JDN-based)
    validate.go          // Validation rules
    parse.go             // Parser implementation
    format.go            // Formatting back to GEDCOM
    date_test.go         // Comprehensive tests
```

### 12.2 API Design

**Parsing**:
```go
// Parse GEDCOM date string
func Parse(s string, version Version) (*Date, error)

// Parse with options
func ParseWithOptions(s string, opts ParseOptions) (*Date, error)

// Validate without full parsing
func Validate(s string, version Version) []DateWarning
```

**Conversion**:
```go
// Convert to Gregorian time.Time
func (d *Date) ToGregorian() (time.Time, error)

// Convert to Julian Day Number
func (d *Date) ToJDN() (int, error)

// Convert between calendars
func (d *Date) ConvertTo(calendar Calendar) (*Date, error)
```

**Formatting**:
```go
// Format back to GEDCOM string
func (d *Date) String() string

// Format for display
func (d *Date) Display() string

// Format with options
func (d *Date) Format(opts FormatOptions) string
```

**Comparison**:
```go
// Compare dates (returns -1, 0, 1)
func (d *Date) Compare(other *Date) int

// Check if date is before/after
func (d *Date) Before(other *Date) bool
func (d *Date) After(other *Date) bool

// Check if dates overlap (for ranges/periods)
func (d *Date) Overlaps(other *Date) bool
```

### 12.3 Testing Requirements

**Test Coverage Goals**:
- Unit tests: >90% code coverage
- Integration tests: All GEDCOM versions
- Edge case tests: All scenarios in this document
- Benchmark tests: Performance targets

**Test Data Sources**:
- GEDCOM official test files
- Real-world files from multiple genealogy programs
- Synthetic edge cases
- Invalid date corpus

**Specific Test Cases** (minimum):
1. All four calendars (Gregorian, Julian, Hebrew, French R.)
2. All modifiers (ABT, BEF, AFT, BET/AND, CAL, EST, FROM, TO)
3. Partial dates (year only, month-year)
4. Date ranges and periods
5. Invalid dates (syntax and semantic)
6. Calendar transitions (British 1752, etc.)
7. Leap years (1900, 2000, 2024)
8. Dual dating (slash format)
9. B.C./B.C.E. dates
10. Case sensitivity variations
11. Whitespace variations
12. Date phrases (5.5 vs 7.0 format)
13. Very large/small years
14. Round-trip: parse → format → parse
15. Sorting and comparison

### 12.4 Performance Considerations

**Optimization Targets**:
- Parse 10,000 dates per second
- Memory: <100 bytes per Date struct
- No allocations in hot path (reuse buffers)
- Cache calendar conversions

**Techniques**:
- Compile regex once (use sync.Once)
- Lookup tables for month names
- JDN cache for common dates
- Avoid string concatenation (use strings.Builder)
- Pool temporary buffers

### 12.5 Error Handling

**Error Types**:
```go
type DateParseError struct {
    Input    string
    Position int
    Reason   string
}

type DateValidationError struct {
    Date     *Date
    Rule     ValidationRule
    Severity ErrorSeverity  // Error, Warning, Info
}
```

**Severity Levels**:
- **Error**: Cannot parse or invalid
- **Warning**: Parseable but questionable (e.g., future date for historical record)
- **Info**: Non-standard but acceptable (e.g., lowercase month names)

---

## 13. Summary and Quick Reference

### 13.1 Date Format Quick Reference

| Format | Example | Calendar | GEDCOM Version |
|--------|---------|----------|----------------|
| Exact date | `25 DEC 2020` | Gregorian | All |
| Year only | `1850` | Gregorian | All |
| Month-Year | `JAN 1900` | Gregorian | All |
| Approximate | `ABT 1850` | Gregorian | All |
| Before | `BEF 1900` | Gregorian | All |
| After | `AFT 1850` | Gregorian | All |
| Range | `BET 1850 AND 1860` | Gregorian | All |
| Period | `FROM 1880 TO 1920` | Gregorian | All |
| Julian | `@#DJULIAN@ 25 DEC 1700` | Julian | All |
| Hebrew | `@#DHEBREW@ 13 CSH 5760` | Hebrew | All |
| French Rep. | `@#DFRENCH R@ 15 VEND 3` | French R. | All |
| Dual date | `21 FEB 1750/51` | Julian (implied) | 5.5 only |
| Date phrase | `(unknown)` | N/A | 5.5 only |
| Phrase (7.0) | `PHRASE ...` | N/A | 7.0 only |

### 13.2 Month Abbreviations

**Gregorian/Julian**: JAN, FEB, MAR, APR, MAY, JUN, JUL, AUG, SEP, OCT, NOV, DEC

**Hebrew**: TSH, CSH, KSL, TVT, SHV, ADR, ADS, NSN, IYR, SVN, TMZ, AAV, ELL

**French Republican**: VEND, BRUM, FRIM, NIVO, PLUV, VENT, GERM, FLOR, PRAI, MESS, THER, FRUC, COMP

### 13.3 Key Differences by GEDCOM Version

| Feature | GEDCOM 5.5/5.5.1 | GEDCOM 7.0 |
|---------|------------------|------------|
| Year required | No (can be omitted) | Yes (must be present) |
| Date phrases | In parentheses in DATE | PHRASE substructure |
| Dual dating | Slash format allowed | Use PHRASE instead |
| BET...AND | "Happened between" | "No earlier...no later" |
| INT dates | Supported | Removed |

### 13.4 Critical Implementation Notes

1. **Parse case-insensitively** for month names (accept Jan, JAN, jan)
2. **Generate uppercase** on output (JAN, not jan)
3. **Preserve original string** for round-trip fidelity
4. **Use JDN** as intermediate format for calendar conversion
5. **Validate semantically** but parse liberally
6. **Store validation warnings** separately from errors
7. **Handle calendar transitions** (British 1752 gap, etc.)
8. **Support partial dates** (year only, month-year)
9. **Implement sorting** with missing component rules
10. **Test thoroughly** with real-world files

---

## 14. References and Resources

### 14.1 Official Specifications

1. **GEDCOM 5.5** (1996)
   - URL: https://gedcom.io/specifications/ged55.pdf
   - Chapter 2: Grammar and data format

2. **GEDCOM 5.5.1** (1999)
   - URL: https://gedcom.io/specifications/ged551.pdf
   - Addendum to 5.5

3. **GEDCOM 7.0** (2021)
   - URL: https://gedcom.io/specifications/FamilySearchGEDCOMv7.html
   - Complete rewrite with stricter grammar

### 14.2 Conversion Tools

- GEDCOM Date Calculator: https://geditcom.com/DateCalculator/
- Hebcal Hebrew Date Converter: https://www.hebcal.com/converter/
- French Republican Calendar Converter: https://www.napoleon-empire.org/en/republican-calendar.php

### 14.3 Algorithms and Books

- **"Calendrical Calculations"** by Nachum Dershowitz and Edward M. Reingold
  - Definitive reference for calendar conversion algorithms
  - ISBN: 978-1107057623

### 14.4 Community Resources

- GEDCOM Analysis Tools: https://www.gedcompublisher.com/
- Louis Kessler's Behold Blog: https://www.beholdgenealogy.com/blog/
- GEDCOM Discussion Forums: https://www.tamurajones.net/

---

## Appendix A: Complete Grammar (GEDCOM 7.0)

```abnf
DateValue = [date | DatePeriod | dateRange | dateApprox]

date = [calendar D] [[day D] month D] year [D epoch]
calendar = "GREGORIAN" | "JULIAN" | "FRENCH_R" | "HEBREW" | stdTag
dateRange = "BET" D date D "AND" D date | "BEF" D date | "AFT" D date
dateApprox = ("ABT" | "CAL" | "EST") D date
DatePeriod = ["TO" D date] | "FROM" D date [D "TO" D date]

day = Integer
month = stdTag  ; e.g., JAN, FEB, MAR, ...
year = Integer
epoch = "BCE" | stdTag

D = %x20  ; single space character
```

---

## Appendix B: Test Case Examples

```go
var dateParseTests = []struct {
    input    string
    want     Date
    wantErr  bool
}{
    // Exact dates
    {"25 DEC 2020", Date{Day: 25, Month: 12, Year: 2020}, false},
    {"1 JAN 1900", Date{Day: 1, Month: 1, Year: 1900}, false},

    // Partial dates
    {"1850", Date{Year: 1850}, false},
    {"JAN 1900", Date{Month: 1, Year: 1900}, false},

    // Approximate
    {"ABT 1850", Date{Year: 1850, Modifier: About}, false},
    {"CAL 12 MAY 1875", Date{Day: 12, Month: 5, Year: 1875, Modifier: Calculated}, false},

    // Ranges
    {"BET 1850 AND 1860", Date{Year: 1850, IsRange: true, EndDate: &Date{Year: 1860}}, false},

    // Periods
    {"FROM 1880 TO 1920", Date{Year: 1880, IsPeriod: true, EndDate: &Date{Year: 1920}}, false},

    // Calendars
    {"@#DJULIAN@ 25 DEC 1700", Date{Calendar: Julian, Day: 25, Month: 12, Year: 1700}, false},
    {"@#DHEBREW@ 13 CSH 5760", Date{Calendar: Hebrew, Day: 13, Month: 2, Year: 5760}, false},

    // Invalid dates
    {"32 JAN 2020", Date{}, true},  // Invalid day
    {"30 FEB 2020", Date{}, true},  // Invalid day for month
    {"XYZ 2020", Date{}, true},     // Invalid month

    // Edge cases
    {"29 FEB 2000", Date{Day: 29, Month: 2, Year: 2000}, false},  // Valid leap year
    {"29 FEB 1900", Date{}, true},  // Invalid leap year
}
```

---

**End of Document**
