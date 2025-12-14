# GEDCOM Feature Gaps Analysis

**Date**: 2025-12-13
**Current Version**: go-gedcom v0.1.0
**Analysis Scope**: GEDCOM 5.5, 5.5.1, and 7.0 specifications

## Executive Summary

This document provides a comprehensive analysis of missing or incomplete features in the go-gedcom library compared to the GEDCOM specifications (5.5, 5.5.1, and 7.0). The analysis identified **23 significant gaps** across individual events, attributes, family events, LDS ordinances, source citations, and record structures.

### Key Findings

- **Individual Events**: 14 event types not parsed (BARM, BASM, BLES, CHRA, CONF, FCOM, GRAD, NATU, ORDN, PROB, WILL, RETI, CREM)
- **Individual Attributes**: 10 attribute types not parsed (CAST, DSCR, EDUC, IDNO, NATI, NCHI, NMR, PROP, RELI, SSN)
- **Event Structures**: Missing subordinate tags for events (AGE, ADDR, AGNC, CAUS, TYPE, UID, etc.)
- **Family Events**: Missing several family event types (MARB, MARC, MARL, MARS, DIVF, ENGA, etc.)
- **LDS Ordinances**: Complete absence of LDS ordinance support (BAPL, CONL, ENDL, SLGC, SLGS)
- **Source Citations**: Missing source citation structure with PAGE, QUAY, DATA subordinates
- **Name Variants**: Missing NICK, SPFX, and TRAN subordinate tags for names
- **Associations**: ASSO tag not parsed for individual associations
- **Record Types**: Missing Submitter (SUBM) entity parsing

### Impact Assessment

**Critical Data Loss**: Real-world GEDCOM files using these features will have data stored only in raw `Tags` arrays, making it inaccessible to downstream applications without custom parsing.

**Current Support Matrix**:
| Feature Category | Supported | Partial | Missing |
|-----------------|-----------|---------|---------|
| Individual Events | 9/23 (39%) | 0 | 14 (61%) |
| Individual Attributes | 1/11 (9%) | 0 | 10 (91%) |
| Family Events | 4/9 (44%) | 0 | 5 (56%) |
| Event Details | 3/15 (20%) | 0 | 12 (80%) |
| LDS Ordinances | 0/5 (0%) | 0 | 5 (100%) |
| Name Components | 5/8 (63%) | 0 | 3 (37%) |

## Detailed Gap Analysis

### 1. Individual Events (Priority: P1-P2)

#### 1.1 Religious Events (P2 - High Impact)

**Missing Events**:
- `BARM` - Bar Mitzvah (Jewish ceremony at age 13)
- `BASM` - Bas Mitzvah (Jewish ceremony at age 12)
- `BLES` - Blessing (religious blessing, often with naming)
- `CHRA` - Adult Christening
- `CONF` - Confirmation (religious)
- `FCOM` - First Communion

**Specification Reference**: GEDCOM 5.5.1 Section 2.6 (Individual Event Structure)
**Current Behavior**: Tags are parsed but stored in raw `Tags` array only
**Impact**: Religious genealogy applications lose critical lifecycle events
**Complexity**: Low - Same structure as existing events (DATE, PLAC subordinates)
**Priority**: P2 (Important - common in religious genealogy)

**Real-world Usage**: Found in torture test files (TGC551LF.ged lines 542, 555, 568, 596, 609, 621)

#### 1.2 Life Status Events (P2 - High Impact)

**Missing Events**:
- `GRAD` - Graduation (educational milestone)
- `RETI` - Retirement
- `ORDN` - Ordination (religious)

**Specification Reference**: GEDCOM 5.5.1 Section 2.6
**Current Behavior**: Not parsed
**Impact**: Educational and occupational history lost
**Complexity**: Low
**Priority**: P2

#### 1.3 Citizenship Events (P2 - Medium Impact)

**Missing Events**:
- `NATU` - Naturalization (citizenship)

**Specification Reference**: GEDCOM 5.5.1 Section 2.6
**Current Behavior**: Not parsed
**Impact**: Immigration research significantly impaired
**Complexity**: Low
**Priority**: P2
**Note**: `IMMI` and `EMIG` are already supported, but naturalization completes the immigration lifecycle

#### 1.4 Legal/Estate Events (P2 - Medium Impact)

**Missing Events**:
- `PROB` - Probate (judicial determination of will validity)
- `WILL` - Will (legal document, date is signing date)

**Specification Reference**: GEDCOM 5.5.1 Section 2.6
**Current Behavior**: Not parsed
**Impact**: Estate and inheritance research incomplete
**Complexity**: Low
**Priority**: P2

**Real-world Usage**: Found in torture test files and pres2020.ged (line 32467)

#### 1.5 Death-related Events (P3 - Low Impact)

**Missing Events**:
- `CREM` - Cremation (alternative to burial)

**Specification Reference**: GEDCOM 5.5.1 Section 2.6
**Current Behavior**: Not parsed (BURI is supported)
**Impact**: Low - burial already captured, cremation is alternative
**Complexity**: Low
**Priority**: P3

**Real-world Usage**: Found in GEDCOM 7.0 maximal70.ged (line 316)

### 2. Individual Attributes (Priority: P2-P3)

**Specification Note**: Attributes differ from events - they represent ongoing states or characteristics rather than point-in-time occurrences.

#### 2.1 Identity Attributes (P2 - High Impact)

**Missing Attributes**:
- `CAST` - Caste name (social status)
- `DSCR` - Physical description
- `IDNO` - National identification number
- `SSN` - Social Security Number (U.S. specific)
- `NATI` - Nationality or tribal origin
- `TITL` - Nobility or official title

**Specification Reference**: GEDCOM 5.5.1 Section 2.7 (Individual Attribute Structure)
**Current Behavior**: Not parsed at all
**Impact**: Identity and social context lost
**Complexity**: Low - Similar to existing Attribute structure
**Priority**: P2

**Real-world Usage**:
- CAST, DSCR, IDNO, NATI, TITL found in maximal70.ged
- SSN found extensively in pres2020.ged (U.S. presidents data)
- TITL found in royal92.ged (royalty titles)

**Note**: Current `Attribute` struct exists but only `OCCU` is parsed in decoder/entity.go

#### 2.2 Educational Attributes (P2 - Medium Impact)

**Missing Attributes**:
- `EDUC` - Education (degree/level achieved)

**Current Behavior**: Not parsed
**Impact**: Educational history lost
**Complexity**: Low
**Priority**: P2

**Real-world Usage**: Found in comprehensive.ged and torture test files

#### 2.3 Religious Attributes (P3 - Low Impact)

**Missing Attributes**:
- `RELI` - Religious affiliation

**Specification Reference**: GEDCOM 5.5.1 Section 2.7
**Current Behavior**: Not parsed
**Impact**: Medium - religious context useful but not critical
**Complexity**: Low
**Priority**: P3

**Note**: Currently appears as event subordinate (DEAT/RELI in maximal70.ged line 332)

#### 2.4 Family Statistics Attributes (P3 - Low Impact)

**Missing Attributes**:
- `NCHI` - Number of children
- `NMR` - Number of marriages
- `PROP` - Property/possessions

**Specification Reference**: GEDCOM 5.5.1 Section 2.7
**Current Behavior**: Not parsed
**Impact**: Low - statistical metadata, not genealogical facts
**Complexity**: Low
**Priority**: P3

**Real-world Usage**: NCHI found in maximal70.ged (both individual and family contexts)

### 3. Event Subordinate Tags (Priority: P1-P2)

**Critical Issue**: Current `Event` struct only captures DATE, PLAC, Description. Many subordinate tags are missing.

#### 3.1 Event Metadata (P1 - High Impact)

**Missing Subordinate Tags**:
- `TYPE` - Event type classification (subordinate to event tag)
- `AGE` - Age at event occurrence
- `CAUS` - Cause (especially for death)
- `AGNC` - Responsible agency

**Specification Reference**: GEDCOM 5.5.1 Section 2.5 (Event Detail)
**Current Behavior**: Not parsed from events
**Impact**: Critical detail lost (e.g., "cause of death" is very common)
**Complexity**: Medium - requires extending Event struct and parser
**Priority**: P1 (Critical)

**Real-world Usage**:
- TYPE found extensively in maximal70.ged for all event types
- AGE found in GEDCOM 7.0 (maximal70.ged line 387: "AGE 8d")
- CAUS found in maximal70.ged line 333 (death cause)
- AGNC found in comprehensive.ged and maximal70.ged

#### 3.2 Event Location Details (P2 - Medium Impact)

**Missing Subordinate Tags**:
- `ADDR` - Address structure (subordinate to PLAC)
- `PHON`, `EMAIL`, `FAX`, `WWW` - Contact information

**Specification Reference**: GEDCOM 5.5.1 Section ADDRESS_STRUCTURE
**Current Behavior**: Not parsed
**Impact**: Medium - detailed location info lost
**Complexity**: Medium - requires Address struct and parser
**Priority**: P2

**Real-world Usage**: Found in comprehensive.ged (RESI with full address) and maximal70.ged

#### 3.3 Event Administrative Tags (P3 - Low Impact)

**Missing Subordinate Tags**:
- `RESN` - Restriction notice (privacy)
- `UID` - Unique identifier
- `SDATE` - Sort date (GEDCOM 7.0)

**Specification Reference**: GEDCOM 7.0 Section
**Current Behavior**: Not parsed
**Impact**: Low - administrative metadata
**Complexity**: Low
**Priority**: P3

### 4. Family Events (Priority: P2)

#### 4.1 Marriage-related Events (P2 - Medium Impact)

**Missing Events**:
- `MARB` - Marriage Bann (announcement of intent)
- `MARC` - Marriage Contract
- `MARL` - Marriage License
- `MARS` - Marriage Settlement
- `ENGA` - Engagement (betrothal)

**Specification Reference**: GEDCOM 5.5.1 Section 2.4 (Family Event Structure)
**Current Behavior**: MARR, DIV, ANUL supported; others not parsed
**Impact**: Medium - detailed marriage history lost
**Complexity**: Low - same as existing family events
**Priority**: P2

**Real-world Usage**: ENGA, MARB, MARC, MARL, MARS found in maximal70.ged

**Currently Supported**: MARR, DIV, ENGA, ANUL (see decoder/entity.go line 187)

#### 4.2 Divorce-related Events (P3 - Low Impact)

**Missing Events**:
- `DIVF` - Divorce Filing

**Specification Reference**: GEDCOM 5.5.1 Section 2.4
**Current Behavior**: Not parsed
**Impact**: Low - DIV already captured
**Complexity**: Low
**Priority**: P3

### 5. LDS Ordinances (Priority: P2-P3)

**Critical Gap**: Completely missing LDS (Latter-Day Saints) ordinance support.

#### 5.1 Individual LDS Ordinances (P2 - High Impact for LDS users)

**Missing Ordinances**:
- `BAPL` - LDS Baptism
- `CONL` - LDS Confirmation
- `ENDL` - LDS Endowment
- `SLGC` - LDS Sealing Child to Parents

**Specification Reference**: GEDCOM 5.5.1 Section 2.8 (LDS Individual Ordinance)
**Current Behavior**: Not parsed at all
**Impact**: Critical for LDS genealogy (FamilySearch is major GEDCOM producer)
**Complexity**: Medium - requires new struct with STAT, TEMP, PLAC subordinates
**Priority**: P2 (Important for large user base)

**Real-world Usage**: Found extensively in maximal70.ged and maximal70-lds.ged:
- BAPL with STAT (STILLBORN, SUBMITTED, etc.) lines 401-408
- CONL with STAT (INFANT) lines 409-413
- ENDL with STAT (CHILD) lines 414-418
- SLGC with TEMP, FAMC subordinates lines 424-443

#### 5.2 Family LDS Ordinances (P2)

**Missing Ordinances**:
- `SLGS` - LDS Sealing Spouse to Spouse

**Specification Reference**: GEDCOM 5.5.1 Section 2.9 (LDS Spouse Sealing)
**Current Behavior**: Not parsed
**Impact**: Critical for LDS family structure
**Complexity**: Medium
**Priority**: P2

**Real-world Usage**: Found in maximal70.ged lines 161-199 with multiple STAT values

### 6. Source Citations (Priority: P1)

**Critical Gap**: Source citations are currently only captured as XRef strings.

#### 6.1 Source Citation Structure (P1 - Critical)

**Missing Structure**:
```gedcom
1 SOUR @S1@
  2 PAGE 42       # Page/location within source
  2 QUAY 1        # Quality of evidence (0-3)
  2 DATA          # Data from source
    3 DATE ...
    3 TEXT ...
  2 OBJE @O1@     # Media links
  2 NOTE ...      # Citation notes
```

**Specification Reference**: GEDCOM 5.5.1 Section SOURCE_CITATION
**Current Behavior**: Only XRef captured, no subordinate tags
**Impact**: Critical - source quality and page references lost
**Complexity**: High - requires SourceCitation struct, affects Individual, Family, Event
**Priority**: P1 (Critical for serious genealogy)

**Real-world Usage**: Found extensively in:
- maximal70.ged lines 118-121 (source with PAGE and QUAY)
- comprehensive.ged line 59 (PAGE reference)

**Affected Structures**:
- Individual.Sources (currently []string)
- Family.Sources (currently []string)
- Event.Sources (currently []string)
- Source.RepositoryRef (currently string, should have CALN subordinate)

### 7. Personal Name Extensions (Priority: P2)

#### 7.1 Name Components (P2 - Medium Impact)

**Missing Subordinate Tags**:
- `NICK` - Nickname
- `SPFX` - Surname prefix (de, van, von, etc.)
- `TRAN` - Transliteration (for non-Latin scripts)

**Specification Reference**: GEDCOM 5.5.1 PERSONAL_NAME_STRUCTURE
**Current Behavior**: PREFIX (NPFX) and SUFFIX (NSFX) supported; NICK, SPFX, TRAN not parsed
**Impact**: Medium - international names and nicknames lost
**Complexity**: Low - extend PersonalName struct
**Priority**: P2

**Real-world Usage**: Found in maximal70.ged:
- NICK (line 241)
- SPFX (line 242)
- TRAN (lines 245-254 with LANG subordinate)

**Currently Supported**: Full, Given, Surname, Prefix (NPFX), Suffix (NSFX), Type

### 8. Individual Associations (Priority: P2)

#### 8.1 ASSO Tag (P2 - Medium Impact)

**Missing Feature**:
```gedcom
1 ASSO @I2@
  2 ROLE GODP    # Godparent, witness, friend, etc.
  2 NOTE ...
```

**Specification Reference**: GEDCOM 5.5.1 ASSOCIATION_STRUCTURE
**Current Behavior**: Not parsed
**Impact**: Medium - relationship context lost (godparents, witnesses, friends)
**Complexity**: Medium - requires Association struct
**Priority**: P2

**Real-world Usage**: Found extensively in maximal70.ged (lines 465-483) with roles:
- FRIEND, NGHBR (neighbor), FATH, MOTH, GODP, HUSB, WIFE, SPOU, MULTIPLE

**GEDCOM 7.0 Enhancement**: ASSO can reference events with shared events structure

### 9. Record Types (Priority: P2-P3)

#### 9.1 Submitter Record (P3 - Low Impact)

**Missing Entity Parsing**:
- `SUBM` - Submitter record (person who submitted the data)

**Specification Reference**: GEDCOM 5.5.1 SUBMITTER_RECORD
**Current Behavior**: Record type defined, but entity not parsed in decoder/entity.go
**Impact**: Low - metadata about file submission
**Complexity**: Low
**Priority**: P3

**Structure**:
```gedcom
0 @U1@ SUBM
1 NAME John Q. Genealogist
1 ADDR ...
1 PHON ...
1 EMAIL ...
```

**Real-world Usage**: Found in comprehensive.ged (lines 27-44)

#### 9.2 Repository and Note Records (P3 - Low Impact)

**Missing Entity Parsing**:
- Repository records defined but not parsed in decoder
- Note records defined but not parsed in decoder

**Impact**: Low - already have struct definitions
**Complexity**: Low
**Priority**: P3

### 10. Place Structure (Priority: P2)

#### 10.1 Place Hierarchy (P2 - Medium Impact)

**Missing Feature**:
```gedcom
2 PLAC City, County, State, Country
  3 FORM City, County, State, Country  # Place format
  3 MAP                                 # Coordinates
    4 LATI N42.3601
    4 LONG W71.0589
```

**Specification Reference**: GEDCOM 5.5.1 PLACE_STRUCTURE
**Current Behavior**: Only text string captured
**Impact**: Medium - geolocation and place hierarchy lost
**Complexity**: Medium - requires Place struct
**Priority**: P2

**GEDCOM 7.0 Enhancement**: Enhanced place structure with additional fields

### 11. Multimedia (Priority: P3)

#### 11.1 Multimedia Links (P3 - Low Impact)

**Current Limitation**: MediaObject struct exists but lacks:
- Multiple file references (different resolutions)
- Crop/region information
- GEDCOM 7.0 MIME type support

**Impact**: Low - basic media reference works
**Complexity**: Medium
**Priority**: P3

### 12. Change/Creation Metadata (Priority: P3)

#### 12.1 CHAN and CREA Tags (P3 - Low Impact)

**Missing Feature**:
```gedcom
1 CHAN
  2 DATE 27 MAR 2022
    3 TIME 08:56
1 CREA
  2 DATE 27 MAR 2022
    3 TIME 08:55
```

**Specification Reference**: GEDCOM 5.5.1 CHANGE_DATE
**Current Behavior**: Not parsed
**Impact**: Low - file metadata, not genealogical data
**Complexity**: Low
**Priority**: P3

## Priority Matrix

### P1 - Critical (Implement First)

| Gap | Impact | Frequency | Complexity |
|-----|--------|-----------|------------|
| Source Citations with PAGE/QUAY | Critical | Very High | High |
| Event subordinates (TYPE, CAUS, AGE, AGNC) | Critical | High | Medium |

### P2 - Important (Implement Second)

| Gap | Impact | Frequency | Complexity |
|-----|--------|-----------|------------|
| Religious events (BARM, BASM, BLES, CONF, FCOM, CHRA) | High | Medium | Low |
| Individual attributes (CAST, DSCR, EDUC, IDNO, NATI, SSN, TITL) | High | High | Low |
| Life events (GRAD, RETI, NATU, ORDN, PROB, WILL) | High | Medium | Low |
| LDS ordinances (BAPL, CONL, ENDL, SLGC, SLGS) | Critical* | High* | Medium |
| Name extensions (NICK, SPFX, TRAN) | Medium | Medium | Low |
| Individual associations (ASSO) | Medium | Medium | Medium |
| Place structure with MAP | Medium | Low | Medium |
| Family events (MARB, MARC, MARL, MARS) | Medium | Low | Low |
| Event location details (ADDR subordinate) | Medium | Medium | Medium |

\* Critical for LDS users, low priority for others

### P3 - Nice-to-Have (Implement Later)

| Gap | Impact | Frequency | Complexity |
|-----|--------|-----------|------------|
| CREM event | Low | Low | Low |
| Family statistics (NCHI, NMR, PROP) | Low | Low | Low |
| DIVF event | Low | Low | Low |
| Submitter entity parsing | Low | Low | Low |
| Repository/Note entity parsing | Low | Low | Low |
| Change/creation metadata (CHAN, CREA) | Low | Medium | Low |
| Event administrative tags (RESN, UID, SDATE) | Low | Low | Low |
| Enhanced multimedia | Low | Low | Medium |

## Recommended Implementation Order

Based on impact, frequency in real-world files, and implementation complexity:

### Phase 1: Critical Event Details (2-3 weeks)
1. **Source Citations** - Add SourceCitation struct with PAGE, QUAY, DATA
2. **Event Extensions** - Add TYPE, CAUS, AGE, AGNC to Event struct

### Phase 2: Core Events & Attributes (2-3 weeks)
3. **Individual Attributes** - Parse CAST, DSCR, EDUC, IDNO, NATI, SSN, TITL, RELI
4. **Life Events** - Add GRAD, RETI, NATU, ORDN, PROB, WILL, CREM
5. **Religious Events** - Add BARM, BASM, BLES, CONF, FCOM, CHRA

### Phase 3: LDS Support (2 weeks)
6. **LDS Ordinances** - Add BAPL, CONL, ENDL, SLGC, SLGS with STAT, TEMP subordinates

### Phase 4: Relationships & Structure (1-2 weeks)
7. **Name Extensions** - Add NICK, SPFX, TRAN to PersonalName
8. **Associations** - Add ASSO support with ROLE subordinate
9. **Family Events** - Add MARB, MARC, MARL, MARS, DIVF

### Phase 5: Polish (1 week)
10. **Place Structure** - Add MAP/LATI/LONG support
11. **Event Addresses** - Add ADDR subordinate parsing
12. **Metadata** - Add CHAN, CREA, RESN, UID support

## Testing Recommendations

For each gap, create tests using:
1. **Torture test files** - TGC551.ged, TGC551LF.ged (comprehensive GEDCOM 5.5.1)
2. **GEDCOM 7.0 maximal** - maximal70.ged (comprehensive GEDCOM 7.0)
3. **Real-world files** - royal92.ged, pres2020.ged, comprehensive.ged
4. **Edge cases** - Empty values, multiple instances, missing required subordinates

## Version-Specific Considerations

### GEDCOM 5.5.1 vs 7.0 Differences

1. **PEDI values**: 5.5.1 uses lowercase (birth, adopted), 7.0 uses uppercase (BIRTH, ADOPTED)
   - **Current handling**: Preserves original casing ✓

2. **Event structure**: 7.0 adds SDATE, PHRASE, enhanced ASSO
   - **Current handling**: Not implemented

3. **Place structure**: 7.0 simplifies and enhances
   - **Current handling**: Basic text only

4. **Source structure**: 7.0 restructures significantly
   - **Current handling**: Basic 5.5.1 structure

## References

- **GEDCOM 5.5.1 Specification**: https://gedcom.io/specifications/ged551.pdf
- **GEDCOM 7.0 Specification**: https://gedcom.io/specifications/FamilySearchGEDCOMv7.pdf
- **Test Files**: testdata/gedcom-5.5/, testdata/gedcom-5.5.1/, testdata/gedcom-7.0/

## Downstream Impact Assessment

For the `my-family` consumer application:

1. **Currently Blocked**: Source page references, event details (cause of death)
2. **Future Needs**: LDS ordinances (if expanding user base), educational history
3. **Nice-to-Have**: Place coordinates, multimedia enhancements

## Conclusion

The go-gedcom library has a solid foundation with 39-44% coverage of major event types, but significant gaps exist in:
- Event detail subordinates (critical)
- Individual attributes (high impact)
- LDS ordinances (critical for major user segment)
- Source citation details (critical for serious genealogy)

Implementing the Phase 1 and Phase 2 recommendations would bring coverage to approximately 80% of common real-world GEDCOM usage.
