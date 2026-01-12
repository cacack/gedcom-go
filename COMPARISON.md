# GEDCOM Library Comparison

> Comparing gedcom-go against other GEDCOM parsing libraries

## Overview

This document compares gedcom-go (github.com/cacack/gedcom-go) against competing GEDCOM libraries across multiple languages. The comparison covers feature completeness, API design, performance characteristics, and ecosystem health to help users choose the right library and identify improvement opportunities for gedcom-go.

**Comparison scope:**
- Go libraries: elliotchance/gedcom, iand/gedcom, funwithbots/go-gedcom
- Cross-language references: Java (gedcom4j, gedcom5-java), Python (python-gedcom, python-gedcom7), TypeScript (read-gedcom, js-gedcom), Rust (rust-gedcom), C# (GeneGenie.Gedcom)

**Research date:** January 2026

---

## Quick Comparison

| Aspect | gedcom-go | elliotchance | iand | funwithbots |
|--------|-----------|--------------|------|-------------|
| **Language** | Go | Go | Go | Go |
| **License** | MIT | MIT | Unlicense | GPL-3.0 |
| **GEDCOM 5.5** | Full | Full | ~80% | No |
| **GEDCOM 5.5.1** | Full | Full | Partial | No |
| **GEDCOM 7.0** | Full | No | No | Full |
| **Auto-detect version** | Yes | No | No | No |
| **Streaming** | Yes | No | Yes | Unknown |
| **ANSEL encoding** | Full | No | Limited | Unknown |
| **UTF-16** | Full | Unknown | No | Unknown |
| **Zero dependencies** | Yes* | No | Yes | Unknown |
| **Test coverage** | 93% | Unknown | Unknown | Unknown |
| **Active (2024+)** | Yes | No (Nov 2023) | Yes (Aug 2024) | No (Aug 2023) |
| **Stars** | - | 119 | 41 | 8 |

\* gedcom-go uses only `golang.org/x/text` for encoding transforms (standard extended library)

---

## Detailed Comparisons

### GEDCOM Version Support

| Library | 5.5 | 5.5.1 | 7.0 | Auto-detect | Multi-version API |
|---------|-----|-------|-----|-------------|-------------------|
| **gedcom-go** | Full | Full | Full | Yes | Yes |
| elliotchance/gedcom | Full | Full | No | No | Partial |
| iand/gedcom | ~80% | Partial | No | No | No |
| funwithbots/go-gedcom | No | No | Full | N/A | No |
| gedcom4j (Java) | Full | Full | No | Unknown | Partial |
| gedcom5-java (Java) | Full | Partial | No | Unknown | No |
| python-gedcom | Full | No | No | No | No |
| python-gedcom7 | No | No | Full | N/A | No |
| read-gedcom (TS) | Full | Full | No | Unknown | Partial |
| js-gedcom | Partial | Partial | Full | Yes | Yes |

**Analysis:**
- gedcom-go is one of only two Go libraries supporting all three GEDCOM versions
- The combination of 5.5/5.5.1/7.0 support with auto-detection is rare across all languages
- Most libraries force users to choose between legacy (5.x) or modern (7.0) support

### Feature Matrix

| Feature | gedcom-go | elliotchance | iand | funwithbots | gedcom5-java |
|---------|-----------|--------------|------|-------------|--------------|
| **Parsing** | Full | Full | ~80% | Full | Full |
| **Encoding/Writing** | Full | Full | No | Unknown | Full |
| **Validation** | Full | Extensive | Basic | Full | Full |
| **Round-trip** | Yes | Yes | No | Unknown | Yes (94%) |
| **Streaming** | Yes | No | Yes | Unknown | No |
| **Cross-ref resolution** | O(1) | Yes | Yes | Unknown | Yes |
| **Relationship traversal** | Via API | Rich API | Basic | Unknown | Full |
| **Query language** | No | Yes (gedcomq) | No | No | No |
| **CLI tools** | No | Yes | No | Yes | No |

### Record Type Support

| Record Type | gedcom-go | elliotchance | iand | funwithbots |
|-------------|-----------|--------------|------|-------------|
| Individual (INDI) | Full | Full | Yes | Yes |
| Family (FAM) | Full | Full | Yes | Yes |
| Source (SOUR) | Full | Full | Yes | Unknown |
| Repository (REPO) | Full | Full | Yes | Unknown |
| Note (NOTE) | Full | Full | Yes | Unknown |
| Media (OBJE) | Full | Full | Yes | Unknown |
| Submitter (SUBM) | Full | Full | Yes | Unknown |
| Submission (SUBN) | Partial | Unknown | Yes | Unknown |

**All major record types are supported by gedcom-go with full subordinate tag parsing.**

### Data Handling

| Data Type | gedcom-go | elliotchance | iand | read-gedcom |
|-----------|-----------|--------------|------|-------------|
| **Date modifiers** | ABT, CAL, EST, BEF, AFT | Full | Basic | Full |
| **Date ranges** | BET...AND, FROM...TO | Full | No | Yes |
| **Partial dates** | Year, Month-Year | Full | Basic | Yes |
| **B.C. dates** | Full | Unknown | Unknown | Unknown |
| **Dual dating** | Full (1750/51) | Unknown | Unknown | Unknown |
| **Date phrases** | Full | Full | Unknown | Yes |
| **Calendar systems** | Gregorian, Julian, Hebrew, French Republican | Unknown | Unknown | Multi-calendar |
| **Places with coords** | Full (MAP/LATI/LONG) | Yes | Unknown | Unknown |
| **Name components** | Full (GIVN, SURN, etc.) | Full | Yes | Yes |
| **Name transliterations** | Full (GEDCOM 7.0 TRAN) | No | No | No |
| **LDS ordinances** | Full | Full | Unknown | Unknown |
| **Associations** | Full (with PHRASE) | Yes | Unknown | Unknown |
| **Source citations** | Full (with QUAY) | Full | Yes | Yes |

**Analysis:**
- gedcom-go has comprehensive date handling including rarely-supported features (B.C., dual dating, non-Gregorian calendars)
- Name transliterations (GEDCOM 7.0 TRAN) are unique to gedcom-go among Go libraries
- Calendar system support (Hebrew, Julian, French Republican) is a differentiator

### Character Encoding

| Encoding | gedcom-go | elliotchance | iand | gedcom4j | gedcom5-java | read-gedcom |
|----------|-----------|--------------|------|----------|--------------|-------------|
| UTF-8 | Full | Full | Yes | Yes | Yes | Yes |
| UTF-8 BOM | Full | Yes | Unknown | Unknown | Unknown | Yes |
| UTF-16 LE | Full | Unknown | No | Unknown | Unknown | Yes |
| UTF-16 BE | Full | Unknown | No | Unknown | Unknown | Yes |
| ANSEL | Full | No | Limited | Yes | Yes | No |
| ASCII | Full | Yes | Yes | Yes | Yes | Yes |
| LATIN1 | Full | Unknown | Unknown | Unknown | Unknown | Yes |
| Auto-detect | Full | No | Limited | Unknown | Unknown | Yes |

**Analysis:**
- gedcom-go has the most comprehensive encoding support among Go libraries
- ANSEL support (critical for legacy GEDCOM files) sets gedcom-go apart from elliotchance/gedcom
- Auto-detection from both BOM and CHAR header tag provides robust handling

### API Design Comparison

#### gedcom-go

```go
// Clean io.Reader/io.Writer interfaces
doc, err := decoder.Decode(reader)

// O(1) cross-reference lookup
person := doc.GetIndividual("@I1@")

// Typed entity access
for _, individual := range doc.Individuals() {
    fmt.Println(individual.Names[0].Full)
}

// Structured date parsing with lossless representation
date, _ := gedcom.ParseDate("ABT 1850")
date.Modifier  // ModifierAbout
date.Year      // 1850
date.String()  // "ABT 1850" (original preserved)
```

**Strengths:**
- Standard Go idioms (io.Reader/io.Writer, error returns)
- Zero external dependencies for core functionality
- Type-safe entity access
- Lossless representation (original values preserved)
- Comprehensive godoc documentation

#### elliotchance/gedcom

```go
document := gedcom.NewDocumentFromGEDCOMFile("family.ged")
individuals := document.Individuals()
for _, ind := range individuals {
    name := ind.Name()
    birth, place := ind.Birth()
}
```

**Strengths:**
- Rich convenience accessors (.Parents(), .Spouses())
- Deep comparison and merging algorithms
- Query language (gedcomq) inspired by jq
- Extensive validation warnings

**Weaknesses:**
- File-based API (not io.Reader)
- Large dependency footprint
- No GEDCOM 7.0 support

#### iand/gedcom

```go
d := gedcom.NewDecoder(bytes.NewReader(data))
g, err := d.Decode()
for _, rec := range g.Individual {
    println(rec.Name[0].Name)
}
```

**Strengths:**
- Clean, minimal API
- Streaming-capable design
- Public domain license

**Weaknesses:**
- Only ~80% spec coverage
- Limited date parsing
- No encoder

### Performance Characteristics

| Metric | gedcom-go | elliotchance | iand |
|--------|-----------|--------------|------|
| Parser (simple line) | 66 ns/op | Unknown | Unknown |
| Decode (1000 individuals) | 13 ms | Unknown | Unknown |
| Encode (1000 individuals) | 1.15 ms | Unknown | Unknown |
| Validate (1000 individuals) | 5.91 us | Unknown | Unknown |
| Validator allocations | Zero | Unknown | Unknown |
| Streaming support | Yes | No | Yes |
| Large file handling | Good | Memory-intensive | Good |

**Notes:**
- gedcom-go has documented benchmarks with regression testing
- Zero-allocation validator is notable for large-scale processing
- Streaming design enables efficient memory usage for large files

### Ecosystem and Maintenance

| Library | Last Release | Commits | Stars | Documentation | Examples |
|---------|--------------|---------|-------|---------------|----------|
| **gedcom-go** | Active (2026) | Active | - | pkg.go.dev, USAGE.md | Yes |
| elliotchance/gedcom | Nov 2023 | 171 releases | 119 | pkg.go.dev | CLI-focused |
| iand/gedcom | Aug 2024 | Moderate | 41 | pkg.go.dev | Basic |
| funwithbots/go-gedcom | Aug 2023 | Low | 8 | README only | Minimal |
| gedcom4j | ~2020 | 877 | 58 | Site expired | JavaDoc |
| gedcom5-java | ~2020 | High | 87 | README | Yes |
| python-gedcom | Apr 2019 | Low | 170 | readthedocs | Basic |
| read-gedcom | Jun 2022 | Moderate | 24 | docs.arbre.app | Yes |

---

## gedcom-go Strengths

### 1. Multi-Version Support (Unique in Go)
gedcom-go is the only Go library supporting GEDCOM 5.5, 5.5.1, AND 7.0 with automatic version detection. This is also rare across all languages.

### 2. Lossless Representation
Following the project constitution, gedcom-go preserves original values while providing parsed convenience methods. Date "ABT 1850" stores both the modifier enum AND the original string.

### 3. Comprehensive Encoding Support
Full support for UTF-8, UTF-16 (LE/BE), ANSEL, ASCII, and LATIN1 with automatic detection. This exceeds most competitors, particularly ANSEL support which is critical for legacy files.

### 4. Calendar System Support
Parsing for Gregorian, Julian, Hebrew, and French Republican calendars with proper month code handling. Few libraries support non-Gregorian calendars.

### 5. GEDCOM 7.0 Features
Name transliterations (TRAN), association phrases (ASSO/PHRASE), and structured source citations from the newest standard.

### 6. Performance with Documentation
Benchmarked performance with regression testing. Zero-allocation validator for efficient large-scale processing.

### 7. API Design
Clean Go idioms with io.Reader/io.Writer interfaces, comprehensive godoc, and example code.

### 8. Vendor Extension Support
Structured parsing for Ancestry (_APID, _TREE) and FamilySearch (_FSFTID) extensions with URL helpers.

### 9. MIT License
Permissive license enables maximum adoption, unlike GPL-licensed competitors (funwithbots, python-gedcom).

---

## gedcom-go Gaps

### High Priority (for my-family use case)

| Gap | Available In | Priority | Notes |
|-----|--------------|----------|-------|
| **Relationship traversal API** | elliotchance | High | .Parents(), .Spouses(), .Children() accessors would simplify navigation |
| **Deep comparison/merging** | elliotchance | Medium | Useful for deduplication and tree merging |

### Medium Priority (best-in-class)

| Gap | Available In | Priority | Notes |
|-----|--------------|----------|-------|
| **Query language** | elliotchance (gedcomq) | Medium | jq-like querying for complex searches |
| **Validation warnings** | elliotchance | Medium | Semantic warnings (ChildBornBeforeParent, etc.) |
| **Date similarity scoring** | elliotchance | Low | Fuzzy date matching for record linkage |
| **HTML generation** | elliotchance | Low | Library-focused, but useful for tooling |

### Low Priority (nice-to-have)

| Gap | Available In | Priority | Notes |
|-----|--------------|----------|-------|
| **Progress callbacks** | python-gedcom, read-gedcom | Low | For large file UI feedback |
| **JSON serialization** | rust-gedcom, gedcom5-java | Low | Alternative output format |

### Not Gaps (by design)

| Feature | Notes |
|---------|-------|
| CLI tools | Library-first design; CLI would be separate package |
| GUI | Out of scope for library |
| Database storage | Application-level concern |

---

## Recommendations

### Near-term Improvements

1. **Relationship Traversal API** (High)
   - Add `Individual.Parents()`, `Individual.Spouses()`, `Individual.Children()` convenience methods
   - Add `Family.Members()` for all individuals in a family
   - Follows elliotchance pattern but with gedcom-go's type-safe design

2. **Semantic Validation Warnings** (Medium)
   - Add optional validation layer for logical checks:
     - Birth before parent's birth
     - Death before birth
     - Age reasonableness
   - Keep separate from structural validation

### Future Considerations

3. **Record Comparison** (Medium)
   - Compare two individuals/families for potential matches
   - Useful for deduplication in genealogy workflows

4. **Query Interface** (Low)
   - Consider fluent query API for complex searches
   - Could be simpler than full query language

---

## Methodology

### Data Sources
- GitHub repositories (stars, commits, releases, last activity)
- Package documentation (pkg.go.dev, readthedocs, etc.)
- README files and feature lists
- Source code examination for API design
- Direct testing where feasible

### Evaluation Criteria
- Feature completeness against GEDCOM specifications
- API design quality and Go idiomaticness
- Documentation quality and availability
- Maintenance activity and community health
- License compatibility

### Limitations
- Performance comparisons are limited where benchmarks unavailable
- Some features marked "Unknown" when documentation insufficient
- Star counts reflect popularity, not necessarily quality
- Research reflects snapshot at comparison date

---

## References

### Go Libraries
- [cacack/gedcom-go](https://github.com/cacack/gedcom-go) - This library
- [elliotchance/gedcom](https://github.com/elliotchance/gedcom) - MIT, most feature-rich Go library
- [iand/gedcom](https://github.com/iand/gedcom) - Unlicense, streaming-focused
- [funwithbots/go-gedcom](https://github.com/funwithbots/go-gedcom) - GPL-3.0, GEDCOM 7.0 only

### Other Languages
- [gedcom4j](https://github.com/frizbog/gedcom4j) - Java, MIT
- [gedcom5-java](https://github.com/FamilySearch/gedcom5-java) - Java, Apache-2.0
- [python-gedcom](https://github.com/nickreynke/python-gedcom) - Python, GPL-2.0
- [python-gedcom7](https://github.com/DavidMStraub/python-gedcom7) - Python, MIT
- [read-gedcom](https://github.com/arbre-app/read-gedcom) - TypeScript, MIT
- [js-gedcom](https://github.com/gedcom7code/js-gedcom) - JavaScript, MIT/Unlicense
- [rust-gedcom](https://github.com/pirtleshell/rust-gedcom) - Rust, MIT
- [GeneGenie.Gedcom](https://github.com/TheGeneGenieProject/GeneGenie.Gedcom) - C#, AGPL-3.0

### GEDCOM Specifications
- [GEDCOM 5.5 Specification](https://www.gedcom.org/gedcom.html)
- [GEDCOM 5.5.1 Specification](https://edge.fscdn.org/assets/img/documents/ged551-5bac5e57fe88dd37df0e153d9c515335.pdf)
- [GEDCOM 7.0 Specification](https://gedcom.io/specifications/FamilySearchGEDCOMv7.html)
