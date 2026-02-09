# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.2.0](https://github.com/cacack/gedcom-go/compare/v1.1.0...v1.2.0) (2026-02-09)


### Features

* **converter:** add granular ConversionNote tracking for version transformations ([7366ac6](https://github.com/cacack/gedcom-go/commit/7366ac672d582c5c5ded6c0a84301eb5b43eb3f6))
* **decoder:** add SNOTE handling to parseMediaObject ([91024d4](https://github.com/cacack/gedcom-go/commit/91024d453f688dcc3dbc11f92fdf59d89d3885da))
* **parser:** add Go 1.23 range-over-func iterators ([2e64ac6](https://github.com/cacack/gedcom-go/commit/2e64ac660afac2a2671631636ba256f6783c6fe1))

## [1.1.0](https://github.com/cacack/gedcom-go/compare/v1.0.0...v1.1.0) (2026-01-28)


### Features

* **api:** add top-level convenience API facade ([00029ed](https://github.com/cacack/gedcom-go/commit/00029ed16ae4855b80fff9474a17957bf2414654))
* **decoder:** add GEDCOM 7.0 EXID and SCHMA parsing ([5bdce94](https://github.com/cacack/gedcom-go/commit/5bdce945f422b3f44ccaddae7d78ab0a9018a3bc))
* **decoder:** add lenient parsing mode with diagnostic collection ([390e467](https://github.com/cacack/gedcom-go/commit/390e467a810dd282f4b1a4ba417c7c4649b237e0))
* **gedcom:** implement NO tag for negative assertions (GEDCOM 7.0) ([3081665](https://github.com/cacack/gedcom-go/commit/3081665ef028ccb815f2286c10c7c7de57fc26a5))
* **gedcom:** implement SNOTE (Shared Note) for GEDCOM 7.0 ([acbe0f8](https://github.com/cacack/gedcom-go/commit/acbe0f804739a4741d621187a65fd5bec1282b08))
* **parser,validator:** add INT date modifier and version-aware validation ([94955de](https://github.com/cacack/gedcom-go/commit/94955de7b7f9511335b124e9adc76aa38532e4d8))
* **testing:** add round-trip test helper package ([1784a38](https://github.com/cacack/gedcom-go/commit/1784a38327b3e0b0c09dd9aa7325ac0c9ad5b85e))
* **validator:** add GEDCOM 7.0 encoding validation ([29e4a30](https://github.com/cacack/gedcom-go/commit/29e4a30c7a84c6f1a1f058e9b4c81b66ae034f65))
* **validator:** add SkipEncodingValidation config option ([006069f](https://github.com/cacack/gedcom-go/commit/006069f3b304faeededef9024e70636d094cc48c))


### Bug Fixes

* address CodeRabbit review comments ([bab005c](https://github.com/cacack/gedcom-go/commit/bab005c454d29b6cfcef53e0ff34c2b926aa76b0))
* address CodeRabbit review feedback ([41c92f5](https://github.com/cacack/gedcom-go/commit/41c92f5d42abd2b0df96bfcab1b1947a2994972d))
* **api:** address lint issues in facade implementation ([f1a3f22](https://github.com/cacack/gedcom-go/commit/f1a3f229c6298ba4b1badbe7dfebd8ca7d6b52e4))
* **charset:** handle tiny output buffers in UTF-8 reader ([41f20ec](https://github.com/cacack/gedcom-go/commit/41f20ec5c55ce9e0d3c807e5c290a6a6bf0a71ad))
* **charset:** handle UTF-8 buffer boundary correctly ([0bf149d](https://github.com/cacack/gedcom-go/commit/0bf149d6f6b66db6b410668b6959daf85008c3ea))
* **charset:** return immediately for zero-length read buffer ([5794e37](https://github.com/cacack/gedcom-go/commit/5794e3769465e5f9993ab7bf01a5f059838723bc))
* **charset:** return valid bytes before error on EOF with incomplete UTF-8 ([f2bb667](https://github.com/cacack/gedcom-go/commit/f2bb667ea15c0e4e64a74c4f831004f257187f95))
* **ci:** capture apidiff exit status with set +e ([f46a873](https://github.com/cacack/gedcom-go/commit/f46a873bebdcd37043359c5e9cc20a9ed54174af))
* **ci:** correct apidiff command syntax for module comparison ([e9e26ae](https://github.com/cacack/gedcom-go/commit/e9e26ae3f4623c13bd5774afaf9e92b6012d149b))
* **ci:** download dependencies before running apidiff ([436c67b](https://github.com/cacack/gedcom-go/commit/436c67bdd17183703748e244d085c281055e12ca))
* **ci:** fail when apidiff itself errors ([b6bc0b2](https://github.com/cacack/gedcom-go/commit/b6bc0b27af4c1f374eee9ae677db73ff478e0882))
* **ci:** use clone and API export for apidiff comparison ([0de3072](https://github.com/cacack/gedcom-go/commit/0de3072a25caacee6a03c9816bc5703b9f225f88))
* **decoder:** address CodeRabbit review feedback ([6e7a3f6](https://github.com/cacack/gedcom-go/commit/6e7a3f6d44448508deeee204497998bc3a1f1a2e))
* **decoder:** address lint issues in SCHMA/EXID implementation ([5f27cb9](https://github.com/cacack/gedcom-go/commit/5f27cb9eacb1bb2ae4d5a6f927b46aa11c6b416b))
* **decoder:** validate NO tag has event type before parsing ([f6b3d45](https://github.com/cacack/gedcom-go/commit/f6b3d455529107d158620eed4478eb0797be928c))
* **docs:** address CodeRabbit review feedback ([cb51c53](https://github.com/cacack/gedcom-go/commit/cb51c53f1b3b807fd8094bb2b4e3eacd797a1f05))
* **docs:** clarify compatibility intro text ([e421f12](https://github.com/cacack/gedcom-go/commit/e421f12255c7a23f976d29dfdb7260ce14941e4b))
* **parser,decoder:** address CodeRabbit review feedback ([b6f46fa](https://github.com/cacack/gedcom-go/commit/b6f46faf31bb04b7b8717b8207e91ad394ca2e47))
* **testing:** wire up functional options in round-trip comparison ([0af8183](https://github.com/cacack/gedcom-go/commit/0af81831ce9e9e5b1ebd4a7445bc89ce37d18341))
* **validator:** validate control chars in header string fields ([bb2dbef](https://github.com/cacack/gedcom-go/commit/bb2dbef0ab89aa1cca3e02674fe0a1a991180da5))

## [1.0.0](https://github.com/cacack/gedcom-go/compare/v0.8.1...v1.0.0) (2026-01-19)


### Miscellaneous Chores

* prepare v1.0.0 release ([770219d](https://github.com/cacack/gedcom-go/commit/770219dfebaa918cf1a95f14d7bd597339ce6ec1))

## [0.8.1](https://github.com/cacack/gedcom-go/compare/v0.8.0...v0.8.1) (2026-01-19)


### Bug Fixes

* **makefile:** remove invalid install-staticcheck dependency from lint target ([ab5088a](https://github.com/cacack/gedcom-go/commit/ab5088af47b5b5ac4ea7d85256833861d73cae85))

## [0.8.0](https://github.com/cacack/gedcom-go/compare/v0.7.0...v0.8.0) (2026-01-18)


### Features

* add streaming APIs for memory-efficient large file processing ([dae7bf8](https://github.com/cacack/gedcom-go/commit/dae7bf8b957126182cb94bfd82e1eb0831dc50d7))
* **converter:** add GEDCOM version converter (5.5 &lt;-&gt; 5.5.1 &lt;-&gt; 7.0) ([5b42cd5](https://github.com/cacack/gedcom-go/commit/5b42cd5a84f4b2d12a3e44e410e6a940018b658d))
* **decoder:** add progress callbacks for large file processing ([a81a994](https://github.com/cacack/gedcom-go/commit/a81a9946bffaa24c40a8bc74aa7aa45576a34787))
* **validator:** add custom tag registry for vendor extensions ([4eea4e8](https://github.com/cacack/gedcom-go/commit/4eea4e836087dfc9e9a930a5bb9b0e0625db60ae))


### Bug Fixes

* **ci:** replace deprecated semgrep-action with Docker ([6bce604](https://github.com/cacack/gedcom-go/commit/6bce60426da8c286b81be44212486e13fa737a8a))
* **ci:** resolve security scan failures ([c87883b](https://github.com/cacack/gedcom-go/commit/c87883bdd2c36fd604499570406dd00d127b5023))
* **converter:** address lint and format warnings ([f5c9bb2](https://github.com/cacack/gedcom-go/commit/f5c9bb2f28cf96408e185beda58eeea6d141397f))
* **converter:** address remaining lint warnings ([d56daae](https://github.com/cacack/gedcom-go/commit/d56daaeb5aad3141e16f20a204348d7ba191827e))
* **validator:** address lint issues in test files ([5c796ad](https://github.com/cacack/gedcom-go/commit/5c796adfcb570d313d3fcf2828b10eda772d2ac0))

## [0.7.0](https://github.com/cacack/gedcom-go/compare/v0.6.0...v0.7.0) (2026-01-12)


### Features

* **gedcom:** add relationship traversal API ([00c7a71](https://github.com/cacack/gedcom-go/commit/00c7a716e49eb8bd7f2f839c2f9cb2a3e8d1fc4d))
* **validator:** add enhanced data validation helpers ([8865626](https://github.com/cacack/gedcom-go/commit/886562669e165e9c64566d15f378f4807170c86a)), closes [#38](https://github.com/cacack/gedcom-go/issues/38)


### Bug Fixes

* add .gitattributes to preserve LF line endings on Windows ([33c98c6](https://github.com/cacack/gedcom-go/commit/33c98c68e2efe9e1924d01bbca4c8dc433301a0e))
* mark UTF-16 test files as binary in .gitattributes ([0271274](https://github.com/cacack/gedcom-go/commit/027127476afcf53467c10ee922a17f7d6e3332a9))

## [0.6.0](https://github.com/cacack/gedcom-go/compare/v0.5.0...v0.6.0) (2026-01-03)


### Features

* **encoder:** add CONT/CONC line handling and inline Repository support ([89336e7](https://github.com/cacack/gedcom-go/commit/89336e7266aa042b55015dad5a18eea75a61c327))
* **gedcom:** add Ancestry and FamilySearch GEDCOM extensions ([b9bfe29](https://github.com/cacack/gedcom-go/commit/b9bfe29825978af082fbd75fdc5acfe6b02fd0ab))
* **gedcom:** add GEDCOM 7.0 ASSO/PHRASE and NAME TRAN support ([9b95beb](https://github.com/cacack/gedcom-go/commit/9b95beb6cd6e5eee37494d163dcdbf13e92670b9))
* **gedcom:** add vendor detection from GEDCOM header ([b49ee3c](https://github.com/cacack/gedcom-go/commit/b49ee3c27ce7a384190f1e09f69cbfcaf9e2e129))

## [0.5.0](https://github.com/cacack/gedcom-go/compare/v0.4.0...v0.5.0) (2025-12-26)


### Features

* **charset:** add ANSEL character encoding support ([4264f8a](https://github.com/cacack/gedcom-go/commit/4264f8a6e955410ed2679c198674215a094ce7cd))
* **charset:** add LATIN1 (ISO-8859-1) encoding support ([adf35a1](https://github.com/cacack/gedcom-go/commit/adf35a14589b82f222a42292cff00f607a181144))
* **charset:** add UTF-16 LE/BE encoding support ([d321d51](https://github.com/cacack/gedcom-go/commit/d321d51c3331f490eb9534a5357f69d65f689f98))
* **encoder:** support encoding high-level types ([4d42d94](https://github.com/cacack/gedcom-go/commit/4d42d944696e2be221f0b8b8f2d235eb40be8345))


### Bug Fixes

* **deps:** use golang.org/x/text v0.14.0 for Go 1.21 compatibility ([433e58c](https://github.com/cacack/gedcom-go/commit/433e58c4758f9211c54602a93031e75caba88662))

## [0.4.0](https://github.com/cacack/gedcom-go/compare/v0.3.0...v0.4.0) (2025-12-23)


### Features

* **date:** add calendar conversion with ToGregorian and cross-calendar Compare ([d0b2314](https://github.com/cacack/gedcom-go/commit/d0b23142a7601966d8bc793117b79cfe6c46cde7))
* **date:** add core Gregorian date parsing ([4ee5cc2](https://github.com/cacack/gedcom-go/commit/4ee5cc2eca0827b5ed95f40da4cd96df1143ff15))
* **date:** add core Gregorian date parsing ([39d3115](https://github.com/cacack/gedcom-go/commit/39d31155a9ef8779b5d036f1e04a72bc5991da7a))
* **date:** add Julian, Hebrew, and French Republican calendar parsing ([8e2bb33](https://github.com/cacack/gedcom-go/commit/8e2bb33bd9c8fcdb4654da18d27970f17873eec8))
* **date:** add validation and edge cases (Phase 2) ([6318f5a](https://github.com/cacack/gedcom-go/commit/6318f5a72df5a250f09cea48ef7bbfbc6d684c71))
* **date:** add validation and edge cases (Phase 2) ([692d726](https://github.com/cacack/gedcom-go/commit/692d7267b924de875a2538b1870a0802c7f9d3fc))
* **date:** integrate parsed dates with record types ([4b1e5b4](https://github.com/cacack/gedcom-go/commit/4b1e5b49c632cdbba63409fc8cd854abad8e174f))
* enforce per-package 85% test coverage ([aaf6cf0](https://github.com/cacack/gedcom-go/commit/aaf6cf0d070ef877c63607f8855f483f200704fb))


### Bug Fixes

* **ci:** exempt release-please PRs from title check ([6ca331e](https://github.com/cacack/gedcom-go/commit/6ca331ea088386e265e7d73358df22c95f2d034b))

## [0.3.0](https://github.com/cacack/gedcom-go/compare/v0.2.0...v0.3.0) (2025-12-16)


### Features

* **decoder:** add Submitter, Repository, and Note entity parsing ([3acc118](https://github.com/cacack/gedcom-go/commit/3acc11839f6386b0b1bacf27f8db8692a9015523))
* **decoder:** expand GEDCOM tag support for events, attributes, and structures ([dfc2030](https://github.com/cacack/gedcom-go/commit/dfc20305d24a13ebf2bde49bb438dbf7ebda623c))
* **decoder:** expand GEDCOM tag support for events, attributes, and structures ([3f69ad4](https://github.com/cacack/gedcom-go/commit/3f69ad4da533a60e17328de485ff563480567ceb))
* expand GEDCOM tag support and add entity parsing ([0ddb1ab](https://github.com/cacack/gedcom-go/commit/0ddb1aba84016ff1526b28a694781080bcfd36dc))
* **media:** add GEDCOM 7.0 multimedia support with CROP regions ([689b9e8](https://github.com/cacack/gedcom-go/commit/689b9e89d6f5f7d822c62dd0a6c5024d8748b7cc))
* **media:** add GEDCOM 7.0 multimedia support with CROP regions ([fb32acd](https://github.com/cacack/gedcom-go/commit/fb32acd6f1da98a7ee295e175c4496253f7d7964))


### Bug Fixes

* **ci:** pin gosec to v2.21.4 for Go 1.23 compatibility ([6e901f3](https://github.com/cacack/gedcom-go/commit/6e901f3a6f5eed3eb5a94c99a2dfc505cfdbdc55))
* **ci:** use direct install for security tools ([5599f6d](https://github.com/cacack/gedcom-go/commit/5599f6df113ea534f6a610c9f8b03052faafcbd6))
* **security:** suppress gosec G304 false positives in examples ([0a54a6e](https://github.com/cacack/gedcom-go/commit/0a54a6ec5912d682016d9e56c40c351db2686f2e))
* set go.mod to 1.21 (minimum supported version) ([d8970c3](https://github.com/cacack/gedcom-go/commit/d8970c3432c4dfec22f5f04c90923413e74c8ffe))

## [Unreleased]

### Added

#### Decoder Entity Parsing
- Submitter (`SUBM`) record entity parsing with name, address, and contact details
- Repository (`REPO`) record entity parsing with name and address
- Note (`NOTE`) record entity parsing with text content

#### Source Citations
- Full source citation structure with `PAGE` (page reference), `QUAY` (quality/reliability), and `DATA` subordinates
- Inline source text support via `DATA.TEXT`
- Source citations in individuals, families, and events

#### Individual Events (23+ types)
- Religious events: `BARM`, `BASM`, `BLES`, `CHRA`, `CONF`, `FCOM`
- Life events: `GRAD`, `RETI`, `NATU`, `ORDN`
- Legal/estate events: `PROB`, `WILL`
- Death-related: `CREM`
- All events include subordinate parsing: `TYPE`, `CAUS`, `AGE`, `AGNC`

#### Individual Attributes
- Full attribute parsing: `CAST`, `DSCR`, `EDUC`, `IDNO`, `NATI`, `SSN`, `TITL`, `RELI`
- Family statistics: `NCHI`, `NMR`, `PROP`

#### LDS Ordinances
- Individual ordinances: `BAPL`, `CONL`, `ENDL`, `SLGC`
- Family ordinances: `SLGS`
- Status, temple, and date subordinates

#### Pedigree Linkage (PEDI)
- `FamilyLink` struct to capture pedigree type for child relationships
- Support for `birth`, `adopted`, `foster`, `sealing` relationship types

#### Personal Name Extensions
- `NICK` (nickname) support
- `SPFX` (surname prefix: von, de, van der)
- `TYPE` (name type: birth, married, aka)

#### Associations
- `ASSO` tag parsing with `IndividualXRef` and `ROLE`
- Role support: `GODP`, `WITN`, `FATH`, `MOTH`, etc.

#### Place Structure
- `FORM` (place format) parsing
- `MAP` with `LATI`/`LONG` coordinates

#### Metadata
- `CHAN` (change date) with timestamp
- `CREA` (creation date, GEDCOM 7.0)
- `REFN` (reference number)
- `UID` (unique identifier)

#### Family Events
- Marriage-related: `MARB`, `MARC`, `MARL`, `MARS`
- `DIVF` (divorce filing)

#### Event Subordinates
- Address structure (`ADDR`) for events
- Administrative tags: `RESN`, `UID`, `SDATE`

### Changed
- `Individual.ChildInFamilies` changed from `[]string` to `[]FamilyLink`
- `Individual.Sources` replaced with `SourceCitations []*SourceCitation`
- `Family.Sources` replaced with `SourceCitations []*SourceCitation`

### Testing
- Comprehensive tests for all new entity types
- Security scanning integration

## [0.1.0] - 2025-01-20

### Added

#### Core Functionality
- Complete GEDCOM parser supporting versions 5.5, 5.5.1, and 7.0
- Stream-based parsing for efficient memory usage with large files
- Comprehensive validation against GEDCOM specifications
- Character encoding support (UTF-8, ANSEL, ASCII, LATIN1, UNICODE)
- UTF-8 validation with BOM (Byte Order Mark) removal
- GEDCOM file encoding/writing capability
- Cross-reference resolution and validation

#### Packages
- `charset` - Character encoding utilities with UTF-8 validation
- `decoder` - High-level GEDCOM decoding with automatic version detection
- `encoder` - GEDCOM document writing with configurable line endings
- `gedcom` - Core data types (Document, Individual, Family, Source, etc.)
- `parser` - Low-level line parsing with detailed error reporting
- `validator` - Document validation with error categorization
- `version` - GEDCOM version detection (header and heuristic-based)

#### Documentation
- Complete godoc comments for all public APIs
- Package-level documentation with usage examples
- Comprehensive CONTRIBUTING.md with:
  - Development environment setup
  - Code standards and style guidelines
  - Testing requirements (85% minimum, 90% target)
  - Pull request and issue templates
- Project roadmap in docs/TODO.md
- Four example programs:
  - `examples/parse` - Basic GEDCOM parsing and information display
  - `examples/encode` - Programmatic GEDCOM document creation
  - `examples/query` - Navigating and querying genealogy data
  - `examples/validate` - GEDCOM file validation

#### Testing
- Comprehensive test suite with >90% coverage across all packages:
  - charset: 100.0% coverage
  - gedcom: 100.0% coverage
  - encoder: 95.7% coverage
  - validator: 94.4% coverage
  - parser: 94.3% coverage
  - decoder: 92.1% coverage
  - version: 87.5% coverage
- Table-driven tests for edge cases
- Integration tests with real GEDCOM files
- Error handling and validation tests

### Changed

- **BREAKING**: Module path changed from `github.com/elliotchance/go-gedcom` to `github.com/cacack/gedcom-go`
  - Users must update import statements when upgrading
  - Example: `import "github.com/cacack/gedcom-go/decoder"`

### Technical Details

#### Supported GEDCOM Versions
- GEDCOM 5.5 (genealogical data standard)
- GEDCOM 5.5.1 (enhanced version with additional tags)
- GEDCOM 7.0 (latest specification)

#### Character Encodings
- UTF-8 (recommended, with BOM support)
- ANSEL (legacy genealogy encoding)
- ASCII
- LATIN1
- UNICODE

#### Key Features
- Zero external dependencies (uses only Go standard library)
- Detailed error reporting with line numbers and context
- Cross-reference (XRef) lookup and validation
- Helper methods for common queries (Individuals(), Families(), Sources())
- Configurable encoding options (line endings, etc.)
- Malformed file handling with clear error messages

### Requirements

- Go 1.21 or later

### Installation

```bash
go get github.com/cacack/gedcom-go
```

### Quick Start

```go
package main

import (
    "fmt"
    "log"
    "os"

    "github.com/cacack/gedcom-go/decoder"
)

func main() {
    f, err := os.Open("family.ged")
    if err != nil {
        log.Fatal(err)
    }
    defer f.Close()

    doc, err := decoder.Decode(f)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Found %d individuals\n", len(doc.Individuals()))
}
```

---

## Release Process

This release represents the initial public version of gedcom-go, providing
a solid foundation for GEDCOM file processing in Go with:

- Production-ready parser and validator
- Comprehensive test coverage
- Complete documentation
- Real-world examples
- Clear contribution guidelines

Future releases will focus on:
- Performance optimization
- Enhanced query APIs
- Additional export formats (JSON, XML)
- Merge/diff capabilities

[0.1.0]: https://github.com/cacack/gedcom-go/releases/tag/v0.1.0
