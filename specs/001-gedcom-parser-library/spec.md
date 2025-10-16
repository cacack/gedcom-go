# Feature Specification: GEDCOM Parser Library

**Feature Branch**: `001-gedcom-parser-library`
**Created**: 2025-10-16
**Status**: Draft
**Input**: User description: "Build a pure Go library for parsing GEDCOM files. All official versions of the GEDCOM standard MUST be supported."

## Clarifications

### Session 2025-10-16

- Q: When a GEDCOM file is empty or contains only a header (no records), what should the parser return? → A: Return success with an empty record collection
- Q: How should the library handle circular family relationships (e.g., a person referenced as their own ancestor)? → A: Parse successfully but report circular references as validation warnings
- Q: When a GEDCOM file exceeds available memory in non-streaming mode, how should the parser behave? → A: Return an error when memory threshold exceeded, suggest streaming mode
- Q: How should the library handle GEDCOM files with non-standard character encodings not declared in the header? → A: Attempt UTF-8 first, fall back to Latin-1, return error if both fail
- Q: How should the library handle cross-references that use non-standard formats (e.g., `@I-001@` instead of `@I1@`)? → A: Accept any alphanumeric cross-reference format with delimiters (-, _) as long as properly delimited with @ and unique, but issue validation warnings about non-standard format
- Q: Should the library provide progress callbacks or hooks for long-running parsing operations? → A: Optional progress callbacks via configuration (callback function receives % complete, record count)
- Q: Should the library enforce limits to protect against resource exhaustion attacks? → A: Configurable limits with safe defaults (max depth: 100 levels, max entities: 1M, timeout: 5 min)

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Parse Valid GEDCOM File (Priority: P1)

A developer integrates the library into their genealogy application to load and process GEDCOM files exported from various genealogy software programs. They need to read individual records, family relationships, and associated metadata from files that conform to any official GEDCOM standard version.

**Why this priority**: This is the core capability - without being able to parse valid GEDCOM files, the library has no value. This represents the minimum viable functionality.

**Independent Test**: Can be fully tested by providing a valid GEDCOM file and verifying that all records, relationships, and metadata are correctly extracted and accessible through the library API.

**Acceptance Scenarios**:

1. **Given** a valid GEDCOM 5.5 file with individuals and families, **When** the developer parses the file, **Then** all individual records are accessible with their names, dates, and places
2. **Given** a valid GEDCOM 5.5.1 file with source citations, **When** the developer parses the file, **Then** source records and citations are correctly linked to individuals and events
3. **Given** a valid GEDCOM 7.0 file with extended attributes, **When** the developer parses the file, **Then** all version-specific features are correctly parsed and accessible
4. **Given** a GEDCOM file without explicit version header, **When** the developer parses the file, **Then** the library auto-detects the version based on content structure and tags
5. **Given** a large GEDCOM file (10MB+) with thousands of records, **When** the developer parses the file, **Then** parsing completes without excessive memory consumption

---

### User Story 2 - Handle Malformed GEDCOM Files (Priority: P2)

A developer processes GEDCOM files from diverse sources, including legacy software and user-generated exports, which may contain formatting errors, invalid tags, or structural inconsistencies. They need clear, actionable error messages that identify exactly what is wrong and where.

**Why this priority**: Real-world GEDCOM files frequently contain errors. Without graceful error handling, the library will be impractical for production use.

**Independent Test**: Can be tested by providing intentionally malformed GEDCOM files with specific errors and verifying that appropriate error messages are returned with line numbers and context.

**Acceptance Scenarios**:

1. **Given** a GEDCOM file with an invalid tag on line 45, **When** the developer parses the file, **Then** an error message reports "Invalid tag 'XYZ' at line 45: expected INDI, FAM, or SOUR"
2. **Given** a GEDCOM file with mismatched hierarchy levels, **When** the developer parses the file, **Then** an error identifies the specific line where level inconsistency occurs
3. **Given** a GEDCOM file with a missing cross-reference target, **When** the developer parses the file, **Then** an error identifies which reference is broken and on which line
4. **Given** a GEDCOM file with encoding issues (mixed UTF-8 and Latin-1), **When** the developer parses the file, **Then** an error reports the character encoding problem with the affected line
5. **Given** a partially corrupted GEDCOM file, **When** the developer parses with error recovery enabled, **Then** valid records are extracted and errors are reported without crashing

---

### User Story 3 - Validate GEDCOM Against Specification (Priority: P3)

A developer creating or modifying GEDCOM data needs to verify that their output conforms to a specific GEDCOM standard version. They need validation that checks not just syntax, but also semantic rules like required fields, value formats, and relationship integrity.

**Why this priority**: Validation ensures data quality and interoperability but is not required for basic parsing. Users can parse without validating if they trust their data source.

**Independent Test**: Can be tested by providing GEDCOM data structures and verifying that validation correctly identifies missing required fields, invalid date formats, broken relationships, and other spec violations.

**Acceptance Scenarios**:

1. **Given** an Individual record missing a required NAME tag (per GEDCOM 5.5 spec), **When** the developer validates the record, **Then** a validation error reports "Individual @I1@ missing required NAME tag"
2. **Given** a date value "32 JAN 2020" (invalid day), **When** the developer validates the record, **Then** a validation error reports the invalid date format
3. **Given** a family record with a child reference that doesn't exist, **When** the developer validates the data, **Then** a validation error reports the broken cross-reference
4. **Given** a GEDCOM 7.0 record using deprecated GEDCOM 5.5 tags, **When** the developer validates against the 7.0 spec, **Then** validation warnings identify deprecated tag usage
5. **Given** a complete and valid GEDCOM structure, **When** the developer validates the data, **Then** validation passes with no errors or warnings

---

### User Story 4 - Convert Between GEDCOM Versions (Priority: P4)

A developer needs to migrate genealogy data between software systems that support different GEDCOM versions. They need to convert GEDCOM 5.5 data to 7.0 format, or downgrade 7.0 data to 5.5.1 for compatibility with legacy systems.

**Why this priority**: Version conversion enables interoperability across the ecosystem but depends on parsing and validation working correctly first. This is an enhancement that adds significant value once core parsing is stable.

**Independent Test**: Can be tested by parsing a file in one version, converting to another version, and verifying that data is preserved and transformed according to version-specific rules.

**Acceptance Scenarios**:

1. **Given** a GEDCOM 5.5 file, **When** the developer converts it to GEDCOM 7.0, **Then** all data is preserved and tags are updated to 7.0 equivalents
2. **Given** a GEDCOM 7.0 file with version-specific features, **When** the developer converts it to GEDCOM 5.5.1, **Then** features without 5.5.1 equivalents are handled gracefully (documented as notes or custom tags)
3. **Given** conversion from 5.5 to 7.0, **When** the developer writes the output, **Then** the resulting file validates against the GEDCOM 7.0 specification
4. **Given** a conversion that requires data transformation, **When** the developer converts the file, **Then** a conversion report lists all transformations and potential data loss

---

### Edge Cases

- Empty files or header-only files return success with an empty record collection
- Circular family relationships (person as their own ancestor) are parsed successfully but reported as validation warnings
- Files exceeding available memory in non-streaming mode return an error indicating memory threshold exceeded and suggest using streaming mode
- Files with undeclared character encoding are processed with UTF-8 fallback to Latin-1, returning error if both fail
- Non-standard cross-reference formats (e.g., `@I-001@`) are accepted if properly delimited and unique, but validation issues warnings about non-standard format
- All line ending formats (DOS CRLF, Unix LF, Mac CR) are handled transparently (see FR-018)
- Very long lines (>1000 characters) are handled via streaming mode and resource limits (see FR-016, FR-024)
- Date formats from different locales and calendars (Hebrew, French Republican, Julian) are parsed according to GEDCOM specification rules (see FR-001)

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Library MUST parse GEDCOM 5.5, 5.5.1, and 7.0 files
- **FR-002**: Library MUST auto-detect GEDCOM version from file header or content structure
- **FR-003**: Library MUST support reading GEDCOM data from any readable stream (not just files)
- **FR-004**: Library MUST extract all standard record types (INDI, FAM, SOUR, REPO, NOTE, OBJE, SUBM, SUBN)
- **FR-005**: Library MUST preserve cross-reference relationships between records
- **FR-022**: Library MUST accept cross-references in any alphanumeric format with delimiters (-, _) as long as properly delimited with @ and unique within the file, but validation MUST issue warnings for non-standard formats
- **FR-006**: Library MUST report parsing errors with line number and content context
- **FR-007**: Library MUST handle character encodings declared in GEDCOM header (ASCII, UTF-8, UNICODE, ANSEL)
- **FR-021**: Library MUST attempt UTF-8 decoding first for files without declared encoding, falling back to Latin-1 (ISO-8859-1) if UTF-8 fails, and return error if both fail
- **FR-008**: Library MUST provide access to all tags and values for each record, including custom/extension tags
- **FR-009**: Library MUST validate GEDCOM data structures against version-specific specifications
- **FR-010**: Library MUST identify missing required fields, invalid values, broken cross-references, and circular relationship chains during validation
- **FR-011**: Library MUST support conversion between GEDCOM versions (5.5 ↔ 5.5.1 ↔ 7.0)
- **FR-012**: Library MUST document data transformations and potential loss during version conversion
- **FR-013**: Library MUST support writing GEDCOM data back to any writable stream
- **FR-014**: Library MUST generate valid GEDCOM output that passes validation for the target version
- **FR-015**: Library MUST handle malformed GEDCOM gracefully without panicking
- **FR-019**: Library MUST return success with empty record collection when parsing files with only header/trailer (no data records)
- **FR-016**: Library MUST support streaming mode for large files to minimize memory usage
- **FR-020**: Library MUST detect when memory threshold is exceeded in non-streaming mode and return actionable error suggesting streaming mode
- **FR-017**: Library MUST preserve original line information for error reporting
- **FR-018**: Library MUST handle all line ending formats (CRLF, LF, CR)
- **FR-023**: Library MUST support optional progress callbacks that report parsing progress (percentage complete and record count processed)
- **FR-024**: Library MUST enforce configurable resource limits with safe defaults: maximum nesting depth (default: 100 levels), maximum entity count (default: 1 million records), and maximum processing time (default: 5 minutes)
- **FR-025**: Library MUST return clear error messages when resource limits are exceeded, indicating which limit was reached and suggesting configuration adjustments

### Key Entities

- **GEDCOM Document**: Represents a complete GEDCOM file with header, records, and trailer. Contains version information, character encoding, and source system metadata.

- **Record**: Base entity representing any GEDCOM record (Individual, Family, Source, etc.). Has a unique identifier (cross-reference ID), record type, and hierarchical tag-value pairs.

- **Individual (INDI)**: Represents a person with biographical information including names, events (birth, death, marriage, etc.), attributes (occupation, education, etc.), and relationships to families.

- **Family (FAM)**: Represents a family unit linking spouses and children. Contains marriage/relationship events and references to Individual records.

- **Source (SOUR)**: Represents a source of genealogical information (book, document, website, etc.). Includes publication details, repository information, and quality assessment.

- **Repository (REPO)**: Represents a physical or digital location where sources are stored (library, archive, website).

- **Event**: Represents a life event (birth, death, marriage, immigration, etc.) with date, place, and associated sources.

- **Note**: Represents extended textual information attached to records or events.

- **Media Object (OBJE)**: Represents multimedia files (photos, documents, audio, video) linked to individuals or events.

- **Cross-Reference**: Represents a link between records using unique identifiers (e.g., Individual reference in Family record).

- **Tag-Value Pair**: Fundamental GEDCOM structure representing a hierarchical data element with level, tag name, optional value, and optional cross-reference.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Developers can parse any valid GEDCOM file (5.5, 5.5.1, or 7.0) and access all records in under 5 lines of code
- **SC-002**: Library processes GEDCOM files at a rate of at least 10,000 records per second on standard hardware
- **SC-003**: Library handles GEDCOM files up to 100MB in size with less than 200MB peak memory usage
- **SC-004**: Parser correctly identifies and reports errors for 95% of common GEDCOM malformations
- **SC-005**: Validation catches 100% of required field violations and invalid cross-references
- **SC-006**: Round-trip conversion (parse → convert → write → parse) preserves 100% of data for version-compatible fields
- **SC-007**: Error messages include line numbers and context for 100% of parsing and validation errors
- **SC-008**: Library completes parsing of a 10MB GEDCOM file in under 2 seconds
- **SC-009**: Documentation and examples enable developers to integrate the library in under 30 minutes
- **SC-011**: Progress callbacks (when enabled) report status at least every 1% of file processed or every 100 records, whichever comes first
- **SC-012**: Resource limit checks add less than 5% performance overhead to parsing operations
- **SC-010**: Library handles all official GEDCOM test files from the specification without errors

### Assumptions

- GEDCOM files follow the line-based hierarchical format defined in the specifications
- Character encoding is either declared in the GEDCOM header or can be reliably detected
- Users integrating the library have basic understanding of GEDCOM format concepts
- The library will be used in contexts where Go standard library is available (not embedded/restricted environments)
- Performance measurements assume modern hardware (multi-core CPU, 8GB+ RAM, SSD storage)
- Developers using the library have access to GEDCOM specification documents for reference
- Most GEDCOM files will be under 50MB; files larger than 100MB are uncommon
