# GEDCOM Feature Implementation Plan

**Date**: 2025-12-13
**Based on**: docs/FEATURE-GAPS.md
**Target**: Comprehensive GEDCOM 5.5, 5.5.1, and 7.0 support

## Overview

This document provides a detailed, actionable implementation plan for closing the 23 feature gaps identified in FEATURE-GAPS.md. The plan is organized into 5 phases with estimated complexity and dependencies clearly marked.

## Implementation Principles

1. **Incremental**: Each feature should be a complete, self-contained commit
2. **Test-driven**: Uncomment corresponding tests from `decoder/entity_test.go` as features are implemented
3. **Backward compatible**: Preserve existing API where possible
4. **Consumer-driven**: Prioritize features needed by `my-family` downstream application
5. **Documentation**: Update godoc comments for all public APIs

## Phase 1: Critical Event Details (Priority: P1)

**Timeline**: 2-3 weeks
**Goal**: Capture critical event metadata currently lost

### Feature 1.1: Source Citations with PAGE, QUAY, DATA

**Priority**: P1 (Critical)
**Complexity**: High
**Estimated Time**: 1 week
**Depends On**: None
**Test**: `TestSourceCitationStructure_NotImplemented` in `decoder/entity_test.go`

**Changes Required**:

#### 1. Create new source citation struct (`gedcom/source.go`)

```go
// SourceCitation represents a citation of a source with additional details.
type SourceCitation struct {
	// SourceXRef is the cross-reference to the source record
	SourceXRef string

	// Page is where within the source this data was found (e.g., "Page 42, Entry 103")
	Page string

	// Quality is the quality assessment of the evidence (0-3)
	// 0 = Unreliable/estimated
	// 1 = Questionable reliability
	// 2 = Secondary evidence
	// 3 = Direct and primary evidence
	Quality int

	// Data contains extracted information from the source
	Data *SourceCitationData

	// Notes are references to note records
	Notes []string

	// MediaRefs are references to media objects
	MediaRefs []string

	// Tags contains all raw tags for unknown/custom fields
	Tags []*Tag
}

// SourceCitationData represents data extracted from a source.
type SourceCitationData struct {
	// Date when the data was recorded in the source
	Date string

	// Text is the actual text from the source
	Text string
}
```

#### 2. Update Event struct (`gedcom/event.go`)

```go
// Event struct - change Sources field
type Event struct {
	// ... existing fields ...

	// Sources are source citations (replaces []string)
	SourceCitations []*SourceCitation

	// Notes are references to note records
	Notes []string

	// ... rest of fields ...
}
```

#### 3. Update Individual, Family, Attribute structs

Replace `Sources []string` with `SourceCitations []*SourceCitation` in:
- `Individual` (`gedcom/individual.go`)
- `Family` (`gedcom/family.go`)
- `Attribute` (`gedcom/individual.go`)

#### 4. Update decoder (`decoder/entity.go`)

Add new function to parse source citations:

```go
// parseSourceCitation extracts source citation from tags starting at sourIdx.
func parseSourceCitation(tags []*Tag, sourIdx int) *SourceCitation {
	cite := &SourceCitation{
		SourceXRef: tags[sourIdx].Value,
	}

	// Look for subordinate tags (level +1)
	baseLevel := tags[sourIdx].Level
	for i := sourIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= baseLevel {
			break
		}
		if tag.Level == baseLevel+1 {
			switch tag.Tag {
			case "PAGE":
				cite.Page = tag.Value
			case "QUAY":
				// Convert string to int (0-3)
				if q, err := strconv.Atoi(tag.Value); err == nil {
					cite.Quality = q
				}
			case "DATA":
				cite.Data = parseSourceCitationData(tags, i)
			case "NOTE":
				cite.Notes = append(cite.Notes, tag.Value)
			case "OBJE":
				cite.MediaRefs = append(cite.MediaRefs, tag.Value)
			}
		}
	}

	return cite
}

func parseSourceCitationData(tags []*Tag, dataIdx int) *SourceCitationData {
	data := &SourceCitationData{}
	baseLevel := tags[dataIdx].Level

	for i := dataIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= baseLevel {
			break
		}
		if tag.Level == baseLevel+1 {
			switch tag.Tag {
			case "DATE":
				data.Date = tag.Value
			case "TEXT":
				data.Text = tag.Value
			}
		}
	}

	return data
}
```

Update `parseEvent`, `parseIndividual`, `parseFamily` to use `parseSourceCitation`.

#### 5. Update encoder (`encoder/encoder.go`)

Add encoding for SourceCitation structures.

#### 6. Migration notes

For downstream consumers:
- Old: `event.Sources []string` → New: `event.SourceCitations []*SourceCitation`
- Migration: `cite.SourceXRef` provides the XRef that was in the old `[]string`
- Breaking change: Consider deprecation period or keeping both fields temporarily

**Suggested Issue Title**: `feat: add source citation structure with PAGE, QUAY, and DATA subordinates`

---

### Feature 1.2: Event Subordinate Tags (TYPE, CAUS, AGE, AGNC)

**Priority**: P1 (Critical)
**Complexity**: Medium
**Estimated Time**: 1-2 weeks
**Depends On**: None
**Test**: `TestEventSubordinateTags_NotImplemented` in `decoder/entity_test.go`

**Changes Required**:

#### 1. Update Event struct (`gedcom/event.go`)

```go
type Event struct {
	// Type is the event type tag (birth, death, marriage, etc.)
	Type EventType

	// Date is when the event occurred (in GEDCOM date format)
	Date string

	// Place is where the event occurred
	Place string

	// Description provides additional details
	Description string

	// NEW FIELDS:

	// EventDetail is the event type classification (subordinate TYPE tag)
	// Example: "Natural death" for a DEAT event
	EventDetail string

	// Cause is the cause of the event (especially for death)
	Cause string

	// Age is the age at the time of the event
	Age string

	// Agency is the responsible agency
	Agency string

	// SourceCitations are source citations (from Phase 1.1)
	SourceCitations []*SourceCitation

	// Notes are references to note records
	Notes []string

	// MediaRefs are references to media objects
	MediaRefs []string

	// Tags contains all raw tags for this event (for unknown/custom fields)
	Tags []*Tag
}
```

**Note**: `EventDetail` instead of `Detail` or `TypeDetail` to avoid confusion with `Type` field.

#### 2. Update parseEvent function (`decoder/entity.go`)

```go
func parseEvent(tags []*Tag, eventIdx int, eventTag string) *Event {
	event := &Event{
		Type: EventType(eventTag),
	}

	// Look for subordinate tags (level 2)
	for i := eventIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= 1 {
			break
		}
		if tag.Level == 2 {
			switch tag.Tag {
			case "DATE":
				event.Date = tag.Value
			case "PLAC":
				event.Place = tag.Value
			case "TYPE":
				event.EventDetail = tag.Value
			case "CAUS":
				event.Cause = tag.Value
			case "AGE":
				event.Age = tag.Value
			case "AGNC":
				event.Agency = tag.Value
			case "NOTE":
				event.Notes = append(event.Notes, tag.Value)
			case "SOUR":
				cite := parseSourceCitation(tags, i)
				event.SourceCitations = append(event.SourceCitations, cite)
			case "OBJE":
				event.MediaRefs = append(event.MediaRefs, tag.Value)
			}
		}
	}

	return event
}
```

#### 3. Update encoder

Add encoding for new event fields.

#### 4. Real-world testing

Test with:
- `testdata/gedcom-7.0/maximal70.ged` (has TYPE, CAUS, AGE, AGNC)
- `testdata/gedcom-5.5.1/comprehensive.ged`

**Suggested Issue Title**: `feat: add event subordinate tags (TYPE, CAUS, AGE, AGNC)`

---

## Phase 2: Core Events & Attributes (Priority: P2)

**Timeline**: 2-3 weeks
**Goal**: Expand parsed event types and attributes

### Feature 2.1: Individual Attributes (CAST, DSCR, EDUC, IDNO, NATI, SSN, TITL, RELI)

**Priority**: P2 (Important)
**Complexity**: Low
**Estimated Time**: 3-4 days
**Depends On**: None
**Test**: `TestIndividualAttributes_NotImplemented` in `decoder/entity_test.go`

**Changes Required**:

#### 1. Update Attribute struct (`gedcom/individual.go`)

Current structure is already suitable, just need to parse more types:

```go
type Attribute struct {
	Type    string   // OCCU, CAST, DSCR, EDUC, IDNO, NATI, SSN, TITL, RELI
	Value   string
	Date    string   // Optional
	Place   string   // Optional
	Sources []string
}
```

**Note**: In Phase 1.1, we'll change `Sources` to `SourceCitations`.

#### 2. Update decoder (`decoder/entity.go`)

In `parseIndividual`, add case for attribute tags:

```go
func parseIndividual(record *Record) *Individual {
	indi := &Individual{
		XRef: record.XRef,
		Tags: record.Tags,
	}

	for i := 0; i < len(record.Tags); i++ {
		tag := record.Tags[i]
		if tag.Level != 1 {
			continue
		}

		switch tag.Tag {
		// ... existing cases ...

		case "CAST", "DSCR", "EDUC", "IDNO", "NATI", "SSN", "TITL", "RELI":
			attr := parseAttribute(record.Tags, i, tag.Tag)
			indi.Attributes = append(indi.Attributes, attr)

		// ... rest of cases ...
		}
	}

	return indi
}

// parseAttribute extracts an attribute from tags starting at attrIdx.
func parseAttribute(tags []*Tag, attrIdx int, attrType string) *Attribute {
	attr := &Attribute{
		Type:  attrType,
		Value: tags[attrIdx].Value,
	}

	// Look for subordinate tags (level 2)
	for i := attrIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= 1 {
			break
		}
		if tag.Level == 2 {
			switch tag.Tag {
			case "DATE":
				attr.Date = tag.Value
			case "PLAC":
				attr.Place = tag.Value
			case "SOUR":
				attr.Sources = append(attr.Sources, tag.Value)
			}
		}
	}

	return attr
}
```

#### 3. Update constants/documentation

Consider adding attribute type constants in `gedcom/individual.go`:

```go
// Attribute types
const (
	AttrOccupation    = "OCCU"
	AttrCaste         = "CAST"
	AttrDescription   = "DSCR"
	AttrEducation     = "EDUC"
	AttrIDNumber      = "IDNO"
	AttrNationality   = "NATI"
	AttrSSN           = "SSN"
	AttrTitle         = "TITL"
	AttrReligion      = "RELI"
	AttrProperty      = "PROP"
	AttrNumChildren   = "NCHI"
	AttrNumMarriages  = "NMR"
)
```

#### 4. Testing

Use `testdata/gedcom-7.0/maximal70.ged` which has all these attributes.

**Suggested Issue Title**: `feat: add individual attribute parsing (CAST, DSCR, EDUC, IDNO, NATI, SSN, TITL, RELI)`

---

### Feature 2.2: Religious Events (BARM, BASM, BLES, CONF, FCOM, CHRA)

**Priority**: P2 (Important)
**Complexity**: Low
**Estimated Time**: 2-3 days
**Depends On**: Feature 1.2 (event subordinates) for consistency
**Test**: `TestReligiousEvents_NotImplemented` in `decoder/entity_test.go`

**Changes Required**:

#### 1. Add event type constants (`gedcom/event.go`)

```go
const (
	// ... existing constants ...

	// Religious events
	EventBarMitzvah      EventType = "BARM" // Bar Mitzvah
	EventBasMitzvah      EventType = "BASM" // Bas Mitzvah (Bat Mitzvah)
	EventBlessing        EventType = "BLES" // Blessing
	EventAdultChristening EventType = "CHRA" // Adult Christening
	EventConfirmation    EventType = "CONF" // Confirmation
	EventFirstCommunion  EventType = "FCOM" // First Communion
)
```

#### 2. Update decoder (`decoder/entity.go`)

In `parseIndividual`, add to event case:

```go
case "BIRT", "DEAT", "BAPM", "BURI", "CENS", "CHR", "ADOP", "OCCU", "RESI", "IMMI", "EMIG",
     "BARM", "BASM", "BLES", "CHRA", "CONF", "FCOM":
	event := parseEvent(record.Tags, i, tag.Tag)
	indi.Events = append(indi.Events, event)
```

#### 3. Testing

Test with `testdata/gedcom-5.5/torture-test/TGC551LF.ged` which has comprehensive religious events.

**Suggested Issue Title**: `feat: add religious event types (BARM, BASM, BLES, CONF, FCOM, CHRA)`

---

### Feature 2.3: Life Events (GRAD, RETI, NATU, ORDN, PROB, WILL, CREM)

**Priority**: P2 (Important)
**Complexity**: Low
**Estimated Time**: 2-3 days
**Depends On**: Feature 1.2 (event subordinates)
**Test**: `TestLifeEvents_NotImplemented` in `decoder/entity_test.go`

**Changes Required**:

#### 1. Add event type constants (`gedcom/event.go`)

```go
const (
	// ... existing constants ...

	// Life status events
	EventGraduation    EventType = "GRAD" // Graduation
	EventRetirement    EventType = "RETI" // Retirement
	EventNaturalization EventType = "NATU" // Naturalization
	EventOrdination    EventType = "ORDN" // Ordination

	// Legal/estate events
	EventProbate       EventType = "PROB" // Probate
	EventWill          EventType = "WILL" // Will

	// Death-related
	EventCremation     EventType = "CREM" // Cremation
)
```

#### 2. Update decoder (`decoder/entity.go`)

Add to event case in `parseIndividual`:

```go
case "BIRT", "DEAT", "BAPM", "BURI", "CENS", "CHR", "ADOP", "OCCU", "RESI", "IMMI", "EMIG",
     "BARM", "BASM", "BLES", "CHRA", "CONF", "FCOM",
     "GRAD", "RETI", "NATU", "ORDN", "PROB", "WILL", "CREM":
	event := parseEvent(record.Tags, i, tag.Tag)
	indi.Events = append(indi.Events, event)
```

#### 3. Testing

Test with torture test files and `pres2020.ged` which has PROB.

**Suggested Issue Title**: `feat: add life event types (GRAD, RETI, NATU, ORDN, PROB, WILL, CREM)`

---

## Phase 3: LDS Support (Priority: P2)

**Timeline**: 2 weeks
**Goal**: Support LDS ordinances for FamilySearch compatibility

### Feature 3.1: LDS Ordinances (BAPL, CONL, ENDL, SLGC, SLGS)

**Priority**: P2 (Critical for LDS users)
**Complexity**: Medium
**Estimated Time**: 2 weeks
**Depends On**: None (separate from Events)
**Test**: `TestLDSOrdinances_NotImplemented` in `decoder/entity_test.go`

**Changes Required**:

#### 1. Create LDS ordinance structs (`gedcom/lds.go` - new file)

```go
package gedcom

// LDSOrdinanceType represents the type of LDS ordinance.
type LDSOrdinanceType string

const (
	LDSBaptism      LDSOrdinanceType = "BAPL" // LDS Baptism
	LDSConfirmation LDSOrdinanceType = "CONL" // LDS Confirmation
	LDSEndowment    LDSOrdinanceType = "ENDL" // LDS Endowment
	LDSSealingChild LDSOrdinanceType = "SLGC" // LDS Sealing Child to Parents
	LDSSealingSpouse LDSOrdinanceType = "SLGS" // LDS Sealing Spouse to Spouse
)

// LDSOrdinanceStatus represents the status of an LDS ordinance.
type LDSOrdinanceStatus string

const (
	LDSStatusCompleted  LDSOrdinanceStatus = "COMPLETED"
	LDSStatusStillborn  LDSOrdinanceStatus = "STILLBORN"
	LDSStatusSubmitted  LDSOrdinanceStatus = "SUBMITTED"
	LDSStatusInfant     LDSOrdinanceStatus = "INFANT"
	LDSStatusChild      LDSOrdinanceStatus = "CHILD"
	LDSStatusExcluded   LDSOrdinanceStatus = "EXCLUDED"
	LDSStatusBIC        LDSOrdinanceStatus = "BIC"
	LDSStatusCanceled   LDSOrdinanceStatus = "CANCELED"
	LDSStatusDNS        LDSOrdinanceStatus = "DNS"
	LDSStatusDNSCAN     LDSOrdinanceStatus = "DNS_CAN"
	LDSStatusPre1970    LDSOrdinanceStatus = "PRE_1970"
	LDSStatusUncleared  LDSOrdinanceStatus = "UNCLEARED"
)

// LDSOrdinance represents an LDS ordinance.
type LDSOrdinance struct {
	// Type is the ordinance type
	Type LDSOrdinanceType

	// Date when the ordinance was performed
	Date string

	// Temple code where ordinance was performed
	Temple string

	// Place where ordinance was performed
	Place string

	// Status is the ordinance status
	Status LDSOrdinanceStatus

	// StatusDate is when the status was set
	StatusDate string

	// FamilyXRef for SLGC (sealing to parents)
	FamilyXRef string

	// Notes are references to note records
	Notes []string

	// SourceCitations are source citations
	SourceCitations []*SourceCitation

	// Tags contains all raw tags
	Tags []*Tag
}
```

#### 2. Update Individual struct (`gedcom/individual.go`)

```go
type Individual struct {
	// ... existing fields ...

	// LDSOrdinances are LDS ordinances for this individual
	LDSOrdinances []*LDSOrdinance

	// ... rest of fields ...
}
```

#### 3. Update Family struct (`gedcom/family.go`)

```go
type Family struct {
	// ... existing fields ...

	// LDSOrdinances are LDS ordinances for this family (SLGS)
	LDSOrdinances []*LDSOrdinance

	// ... rest of fields ...
}
```

#### 4. Create parser (`decoder/lds.go` - new file)

```go
package decoder

import "github.com/cacack/gedcom-go/gedcom"

// parseLDSOrdinance extracts an LDS ordinance from tags.
func parseLDSOrdinance(tags []*gedcom.Tag, ordIdx int, ordType string) *gedcom.LDSOrdinance {
	ord := &gedcom.LDSOrdinance{
		Type: gedcom.LDSOrdinanceType(ordType),
	}

	baseLevel := tags[ordIdx].Level

	for i := ordIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= baseLevel {
			break
		}
		if tag.Level == baseLevel+1 {
			switch tag.Tag {
			case "DATE":
				ord.Date = tag.Value
			case "TEMP":
				ord.Temple = tag.Value
			case "PLAC":
				ord.Place = tag.Value
			case "STAT":
				ord.Status = gedcom.LDSOrdinanceStatus(tag.Value)
				// Look for STAT subordinate DATE
				if i+1 < len(tags) && tags[i+1].Level == baseLevel+2 && tags[i+1].Tag == "DATE" {
					ord.StatusDate = tags[i+1].Value
				}
			case "FAMC":
				ord.FamilyXRef = tag.Value
			case "NOTE":
				ord.Notes = append(ord.Notes, tag.Value)
			case "SOUR":
				cite := parseSourceCitation(tags, i)
				ord.SourceCitations = append(ord.SourceCitations, cite)
			}
		}
	}

	return ord
}
```

#### 5. Update entity parsers (`decoder/entity.go`)

In `parseIndividual`:

```go
case "BAPL", "CONL", "ENDL", "SLGC":
	ord := parseLDSOrdinance(record.Tags, i, tag.Tag)
	indi.LDSOrdinances = append(indi.LDSOrdinances, ord)
```

In `parseFamily`:

```go
case "SLGS":
	ord := parseLDSOrdinance(record.Tags, i, tag.Tag)
	fam.LDSOrdinances = append(fam.LDSOrdinances, ord)
```

#### 6. Testing

Test extensively with:
- `testdata/gedcom-7.0/maximal70.ged` (lines 401-443 for individual ordinances)
- `testdata/gedcom-7.0/maximal70-lds.ged` (dedicated LDS examples)

**Suggested Issue Title**: `feat: add LDS ordinance support (BAPL, CONL, ENDL, SLGC, SLGS)`

---

## Phase 4: Relationships & Structure (Priority: P2)

**Timeline**: 1-2 weeks
**Goal**: Enhance name parsing and relationship tracking

### Feature 4.1: Name Extensions (NICK, SPFX)

**Priority**: P2 (Important for international genealogy)
**Complexity**: Low
**Estimated Time**: 2-3 days
**Depends On**: None
**Test**: `TestNameExtensions_NotImplemented` in `decoder/entity_test.go`

**Changes Required**:

#### 1. Update PersonalName struct (`gedcom/individual.go`)

```go
type PersonalName struct {
	Full   string // Full name (e.g., "John /Doe/")
	Given  string // Given (first) name
	Surname string // Family name
	Prefix string // Name prefix (e.g., "Dr.", "Sir") - NPFX
	Suffix string // Name suffix (e.g., "Jr.", "III") - NSFX
	Type   string // Name type (e.g., "birth", "married", "aka")

	// NEW FIELDS:
	Nickname      string // Nickname (NICK)
	SurnamePrefix string // Surname prefix (SPFX) - e.g., "de", "van", "von"
}
```

**Note**: TRAN (transliteration) is more complex and lower priority. Skip for now.

#### 2. Update parsePersonalName (`decoder/entity.go`)

```go
func parsePersonalName(tags []*gedcom.Tag, nameIdx int) *gedcom.PersonalName {
	name := &gedcom.PersonalName{
		Full: tags[nameIdx].Value,
	}

	// ... existing full name parsing ...

	// Look for subordinate tags (level 2)
	for i := nameIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= 1 {
			break
		}
		if tag.Level == 2 {
			switch tag.Tag {
			case "GIVN":
				name.Given = tag.Value
			case "SURN":
				name.Surname = tag.Value
			case "NPFX":
				name.Prefix = tag.Value
			case "NSFX":
				name.Suffix = tag.Value
			case "TYPE":
				name.Type = tag.Value
			case "NICK":
				name.Nickname = tag.Value
			case "SPFX":
				name.SurnamePrefix = tag.Value
			}
		}
	}

	return name
}
```

#### 3. Testing

Test with `testdata/gedcom-7.0/maximal70.ged` which has NICK and SPFX.

**Suggested Issue Title**: `feat: add name component extensions (NICK, SPFX)`

---

### Feature 4.2: Individual Associations (ASSO)

**Priority**: P2 (Important for relationship context)
**Complexity**: Medium
**Estimated Time**: 3-4 days
**Depends On**: None
**Test**: `TestIndividualAssociations_NotImplemented` in `decoder/entity_test.go`

**Changes Required**:

#### 1. Create Association struct (`gedcom/individual.go`)

```go
// Association represents a relationship to another individual.
type Association struct {
	// AssociateXRef is the cross-reference to the associated individual
	AssociateXRef string

	// Role is the role of the associate (e.g., "GODP", "WITN", "FRIEND")
	Role string

	// Notes are references to note records
	Notes []string

	// SourceCitations are source citations
	SourceCitations []*SourceCitation
}
```

#### 2. Update Individual struct (`gedcom/individual.go`)

```go
type Individual struct {
	// ... existing fields ...

	// Associations are relationships to other individuals
	Associations []*Association

	// ... rest of fields ...
}
```

#### 3. Update decoder (`decoder/entity.go`)

Add parser function:

```go
// parseAssociation extracts an association from tags.
func parseAssociation(tags []*gedcom.Tag, assoIdx int) *gedcom.Association {
	assoc := &gedcom.Association{
		AssociateXRef: tags[assoIdx].Value,
	}

	baseLevel := tags[assoIdx].Level

	for i := assoIdx + 1; i < len(tags); i++ {
		tag := tags[i]
		if tag.Level <= baseLevel {
			break
		}
		if tag.Level == baseLevel+1 {
			switch tag.Tag {
			case "ROLE":
				assoc.Role = tag.Value
			case "NOTE":
				assoc.Notes = append(assoc.Notes, tag.Value)
			case "SOUR":
				cite := parseSourceCitation(tags, i)
				assoc.SourceCitations = append(assoc.SourceCitations, cite)
			}
		}
	}

	return assoc
}
```

In `parseIndividual`:

```go
case "ASSO":
	assoc := parseAssociation(record.Tags, i)
	indi.Associations = append(indi.Associations, assoc)
```

#### 4. Document common roles

Add constants or documentation:

```go
// Common association roles (not exhaustive)
const (
	AssocRoleGodparent   = "GODP"
	AssocRoleWitness     = "WITN"
	AssocRoleFriend      = "FRIEND"
	AssocRoleNeighbor    = "NGHBR"
	AssocRoleClergy      = "CLERGY"
	AssocRoleOfficiator  = "OFFICIATOR"
)
```

#### 5. Testing

Test with `testdata/gedcom-7.0/maximal70.ged` (lines 465-483).

**Suggested Issue Title**: `feat: add individual association support (ASSO with ROLE)`

---

### Feature 4.3: Family Events (MARB, MARC, MARL, MARS)

**Priority**: P2 (Medium)
**Complexity**: Low
**Estimated Time**: 2 days
**Depends On**: Feature 1.2 (event subordinates)
**Test**: Add to existing family event tests

**Changes Required**:

#### 1. Add event type constants (`gedcom/event.go`)

```go
const (
	// ... existing constants ...

	// Family events (additional)
	EventMarriageBann       EventType = "MARB" // Marriage Bann
	EventMarriageContract   EventType = "MARC" // Marriage Contract
	EventMarriageLicense    EventType = "MARL" // Marriage License
	EventMarriageSettlement EventType = "MARS" // Marriage Settlement
	EventDivorceFiling      EventType = "DIVF" // Divorce Filing
)
```

#### 2. Update decoder (`decoder/entity.go`)

In `parseFamily`:

```go
case "MARR", "DIV", "ENGA", "ANUL",
     "MARB", "MARC", "MARL", "MARS", "DIVF":
	event := parseEvent(record.Tags, i, tag.Tag)
	fam.Events = append(fam.Events, event)
```

#### 3. Testing

Test with `testdata/gedcom-7.0/maximal70.ged`.

**Suggested Issue Title**: `feat: add family event types (MARB, MARC, MARL, MARS, DIVF)`

---

## Phase 5: Polish & Metadata (Priority: P3)

**Timeline**: 1 week
**Goal**: Complete remaining features and metadata support

### Feature 5.1: Event Address Details

**Priority**: P2 (Medium)
**Complexity**: Medium
**Estimated Time**: 3-4 days
**Depends On**: None
**Test**: Create new test

**Changes Required**:

#### 1. Enhance Address struct (`gedcom/repository.go`)

Current Address struct is already defined. Enhance if needed.

#### 2. Add Address to Event struct (`gedcom/event.go`)

```go
type Event struct {
	// ... existing fields ...

	// Address provides detailed location (subordinate to PLAC or standalone)
	Address *Address

	// ... rest of fields ...
}
```

#### 3. Update parseEvent (`decoder/entity.go`)

Add ADDR parsing:

```go
case "ADDR":
	event.Address = parseAddress(tags, i)
```

Create parseAddress function similar to how it's done for Repository.

**Suggested Issue Title**: `feat: add address subordinate to events and attributes`

---

### Feature 5.2: Place Structure with Coordinates

**Priority**: P2 (Medium)
**Complexity**: Medium
**Estimated Time**: 2-3 days
**Depends On**: None
**Test**: Create new test

**Changes Required**:

#### 1. Create Place struct (`gedcom/place.go` - new file)

```go
package gedcom

// Place represents a jurisdictional place with optional coordinates.
type Place struct {
	// Name is the place name (e.g., "Springfield, Hampden, MA, USA")
	Name string

	// Form is the hierarchical format (e.g., "City, County, State, Country")
	Form string

	// Coordinates are optional geographic coordinates
	Coordinates *Coordinates
}

// Coordinates represent geographic location.
type Coordinates struct {
	// Latitude (e.g., "N42.3601")
	Latitude string

	// Longitude (e.g., "W71.0589")
	Longitude string
}
```

#### 2. Update Event struct (`gedcom/event.go`)

Change `Place string` to `Place *Place` (breaking change - requires migration plan).

Alternatively, keep both:
```go
type Event struct {
	// ... fields ...

	// Place is the place name (deprecated, use PlaceDetail)
	Place string

	// PlaceDetail provides structured place information
	PlaceDetail *Place
}
```

#### 3. Update decoder

Parse MAP/LATI/LONG subordinates to PLAC.

**Suggested Issue Title**: `feat: add place structure with coordinates (MAP/LATI/LONG)`

---

### Feature 5.3: Metadata Tags (CHAN, CREA, REFN, UID)

**Priority**: P3 (Low - metadata)
**Complexity**: Low
**Estimated Time**: 2 days
**Depends On**: None
**Test**: Create new test

**Changes Required**:

#### 1. Create metadata structs (`gedcom/metadata.go` - new file)

```go
package gedcom

// ChangeDate represents when a record was last changed.
type ChangeDate struct {
	Date string
	Time string
	Notes []string
}

// CreationDate represents when a record was created.
type CreationDate struct {
	Date string
	Time string
}
```

#### 2. Add to Individual, Family, Source, etc.

```go
type Individual struct {
	// ... fields ...

	// ChangeDate is when this record was last modified
	ChangeDate *ChangeDate

	// CreationDate is when this record was created (GEDCOM 7.0)
	CreationDate *CreationDate

	// UserReferenceNumber (REFN) - user-defined cross-reference
	ReferenceNumbers []string

	// UniqueIdentifiers (UID) - persistent globally unique identifiers
	UniqueIdentifiers []string

	// ... rest ...
}
```

#### 3. Update parsers

Parse CHAN, CREA, REFN, UID tags.

**Suggested Issue Title**: `feat: add metadata support (CHAN, CREA, REFN, UID)`

---

### Feature 5.4: Submitter Entity Parsing

**Priority**: P3 (Low - metadata)
**Complexity**: Low
**Estimated Time**: 1 day
**Depends On**: None
**Test**: Create new test

**Changes Required**:

#### 1. Create Submitter struct (`gedcom/submitter.go` - new file)

```go
package gedcom

// Submitter represents the person who submitted the GEDCOM data.
type Submitter struct {
	XRef    string
	Name    string
	Address *Address
	Phone   []string
	Email   []string
	Fax     []string
	Website []string
	Language string
	Tags    []*Tag
}
```

#### 2. Update decoder (`decoder/entity.go`)

Add `parseSubmitter` function and call it in `populateEntities`.

#### 3. Update Document struct

Add method to retrieve submitters.

**Suggested Issue Title**: `feat: add submitter record entity parsing`

---

## Dependencies & Order

Recommended implementation order:

1. **Phase 1.1** (Source Citations) - can run in parallel with 1.2
2. **Phase 1.2** (Event Subordinates) - can run in parallel with 1.1
3. **Phase 2.1** (Attributes) - depends on 1.1 for SourceCitations
4. **Phase 2.2** (Religious Events) - depends on 1.2 for consistency
5. **Phase 2.3** (Life Events) - depends on 1.2 for consistency
6. **Phase 3.1** (LDS Ordinances) - depends on 1.1 for SourceCitations
7. **Phase 4.1** (Name Extensions) - independent
8. **Phase 4.2** (Associations) - depends on 1.1 for SourceCitations
9. **Phase 4.3** (Family Events) - depends on 1.2
10. **Phase 5.1-5.4** (Polish) - can be done in any order

## Testing Strategy

For each feature:

1. **Uncomment corresponding test** in `decoder/entity_test.go`
2. **Run test to verify it fails** (red)
3. **Implement feature**
4. **Run test to verify it passes** (green)
5. **Test with real GEDCOM files**:
   - `testdata/gedcom-5.5/torture-test/TGC551LF.ged`
   - `testdata/gedcom-7.0/maximal70.ged`
   - `testdata/gedcom-5.5.1/comprehensive.ged`
6. **Run full test suite**: `go test ./... -v`
7. **Check coverage**: `go test -cover ./...` (maintain >90%)

## Documentation Updates

For each feature, update:

1. **Godoc comments** for all new types and fields
2. **README.md** - update features list
3. **CLAUDE.md** - update if development process changes
4. **CHANGELOG.md** (if exists) - document changes

## Version Strategy

Given the number of breaking changes (especially SourceCitations):

**Option A: Major version bump (v1.0.0)**
- Implement all changes
- Release as v1.0.0
- Provide migration guide

**Option B: Incremental with deprecation**
- Keep old `Sources []string` alongside new `SourceCitations`
- Mark old field as deprecated
- Release as v0.2.0, v0.3.0, etc.
- Remove deprecated fields in v1.0.0

**Recommendation**: Option B for downstream compatibility with `my-family`.

## Success Criteria

Implementation is complete when:

1. ✅ All 8 feature gap tests in `decoder/entity_test.go` pass
2. ✅ Full test suite passes with >90% coverage
3. ✅ All torture test files parse without errors
4. ✅ Encoder can write all new structures
5. ✅ Documentation is complete and accurate
6. ✅ `my-family` application can import and use new features
7. ✅ No regression in existing functionality

## Issue Tracking

Create GitHub issues for each feature with labels:

- `enhancement` - new feature
- `P1-critical`, `P2-important`, `P3-nice-to-have` - priority
- `phase-1`, `phase-2`, etc. - implementation phase
- `breaking-change` - if API changes

Example issue structure:
```markdown
## Feature: Source Citation Structure

**Reference**: docs/IMPLEMENTATION-PLAN.md Feature 1.1
**Priority**: P1 (Critical)
**Estimated Time**: 1 week

### Changes
- Add SourceCitation struct
- Update Event, Individual, Family structs
- Update decoder and encoder
- Uncomment TestSourceCitationStructure_NotImplemented

### Testing
- [ ] Test passes
- [ ] Coverage >90%
- [ ] Tested with maximal70.ged
- [ ] Tested with comprehensive.ged

### Documentation
- [ ] Godoc comments complete
- [ ] Migration guide written (if breaking change)
```

## Downstream Consumer Notes

For `my-family` application:

### Phase 1 Changes
- **Breaking**: `event.Sources` → `event.SourceCitations`
- **New**: Access to source page numbers and quality ratings
- **Migration**: Update code to use `cite.SourceXRef` instead of raw string

### Phase 2 Changes
- **Non-breaking**: New event types automatically available
- **New**: Access to educational history, religious events, etc.

### Phase 3 Changes
- **Non-breaking**: LDS ordinances in separate field
- **Benefit**: Can display LDS temple work for LDS users

## Conclusion

This implementation plan provides a clear, phased approach to closing all 23 feature gaps identified in FEATURE-GAPS.md. By following this plan, go-gedcom will achieve comprehensive GEDCOM 5.5, 5.5.1, and 7.0 support with minimal risk to existing functionality.

Estimated total time: **8-11 weeks** for all 5 phases.

Estimated time for critical features (Phase 1-2): **4-6 weeks**.
