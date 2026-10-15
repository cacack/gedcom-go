# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [3.3.0](https://github.com/cacack/gedcom-go/compare/v3.2.0...v3.3.0) (2026-10-15)


### Features

* **charset:** decode CHAR ANSI, LATIN1 and ISO-8859-1 as Windows-1252 ([59262d1](https://github.com/cacack/gedcom-go/commit/59262d18f0235f2158bc7ca637bc7e1c6105fc2a)), closes [#564](https://github.com/cacack/gedcom-go/issues/564)

## [3.2.0](https://github.com/cacack/gedcom-go/compare/v3.1.0...v3.2.0) (2026-10-12)


### Bug Fixes

* **charset:** document ANSEL mark flush across CONC and dedupe emit loop ([cf21cb3](https://github.com/cacack/gedcom-go/commit/cf21cb3d46b8e1cffb3c896d6b4b7bb374ea242c)), closes [#431](https://github.com/cacack/gedcom-go/issues/431)
* **charset:** keep ANSEL combining mark before line terminator on its line ([573949c](https://github.com/cacack/gedcom-go/commit/573949c9f1c56bd6005a32296fe085dc7f9435b1)), closes [#431](https://github.com/cacack/gedcom-go/issues/431)
* **encoder:** guard hand-built header fields against line breaks ([c66814b](https://github.com/cacack/gedcom-go/commit/c66814bd638cbb5b6c5415da6508fe63c6c890e5)), closes [#466](https://github.com/cacack/gedcom-go/issues/466)
* **encoder:** reject a line break in a tag name or identifier ([59ed5ae](https://github.com/cacack/gedcom-go/commit/59ed5aea047af4ccfce70480588c16fbe1d7d2ae)), closes [#466](https://github.com/cacack/gedcom-go/issues/466)
* **gedcom:** bound Bounds to real days and make FrenchToJDN closed-form ([5e4225c](https://github.com/cacack/gedcom-go/commit/5e4225c680698ea280faf33427fb7100e6ccf43c)), closes [#566](https://github.com/cacack/gedcom-go/issues/566)
* **gedcom:** clear DualYear when ToGregorian converts a dual date ([9ed5142](https://github.com/cacack/gedcom-go/commit/9ed514249d7b5408b840cd51bdda7cd8f4cc7654)), closes [#631](https://github.com/cacack/gedcom-go/issues/631)
* **gedcom:** compare years across calendars via JDN ([df45b71](https://github.com/cacack/gedcom-go/commit/df45b714a00c8c5398d707aa323940d9b1bf6783)), closes [#618](https://github.com/cacack/gedcom-go/issues/618)
* **gedcom:** keep B.C. dual year off 0 and document dual-year contracts ([7550bc2](https://github.com/cacack/gedcom-go/commit/7550bc20bc04c6a7055c4e1ff5cf8a58336d3718)), closes [#621](https://github.com/cacack/gedcom-go/issues/621)
* **gedcom:** read dual dates in New Style in Compare and YearsBetween ([5572f6d](https://github.com/cacack/gedcom-go/commit/5572f6dcc3d33d1833c8d2816dbbd8e267715289)), closes [#563](https://github.com/cacack/gedcom-go/issues/563)
* **gedcom:** reject negative months in Bounds and make JDNToFrench closed-form ([2d60428](https://github.com/cacack/gedcom-go/commit/2d60428f819a856c5179f14f709a70e4bc1ec05e)), closes [#566](https://github.com/cacack/gedcom-go/issues/566)
* **gedcom:** resolve short dual-year suffixes to the following year ([7b41baf](https://github.com/cacack/gedcom-go/commit/7b41bafe5ca0136c532c0b2574b08a6289f48292)), closes [#621](https://github.com/cacack/gedcom-go/issues/621)
* **validator:** cap approximate-date ordering checks at WARNING ([5b750c8](https://github.com/cacack/gedcom-go/commit/5b750c867d7e7dd8e232140214ffb5d4ea0902ea)), closes [#632](https://github.com/cacack/gedcom-go/issues/632)
* **validator:** place cross-calendar birth dates with an invalid day on their month ([83ae670](https://github.com/cacack/gedcom-go/commit/83ae6700e7b27e2606c2351e619aa97a12bf907b)), closes [#618](https://github.com/cacack/gedcom-go/issues/618)
* **validator:** raise date-ordering ERRORs only when the dates prove them ([ee72e9a](https://github.com/cacack/gedcom-go/commit/ee72e9ab129cd772028778e91f0e580ec19d1100)), closes [#566](https://github.com/cacack/gedcom-go/issues/566)
* **validator:** read cross-calendar birth dates from their first day ([be5a921](https://github.com/cacack/gedcom-go/commit/be5a9212b83640b8788080c0c9665f1b39a7b563)), closes [#618](https://github.com/cacack/gedcom-go/issues/618)


### Documentation

* label v3.2.0 behaviour changes and release as 3.2.0 ([df7b351](https://github.com/cacack/gedcom-go/commit/df7b351aab65d02cd13cf77f6e452f0454b110b0))

## [3.1.0](https://github.com/cacack/gedcom-go/compare/v3.0.0...v3.1.0) (2026-10-11)


### Bug Fixes

* **charset:** document scanner step contract and test probe wiring ([413cafa](https://github.com/cacack/gedcom-go/commit/413cafa0b25a619ad8893171fb6cd60d504b1314)), closes [#592](https://github.com/cacack/gedcom-go/issues/592)
* **charset:** keep a read error that arrives with the 3rd BOM byte ([56c8096](https://github.com/cacack/gedcom-go/commit/56c8096bd76c44bd35518be54d5b8aa87fd5906a)), closes [#593](https://github.com/cacack/gedcom-go/issues/593)
* **charset:** keep bytes read before a BOM-detection read error ([52b6a85](https://github.com/cacack/gedcom-go/commit/52b6a853cbc5d47f58c1497d74d881b1834d0bf0)), closes [#593](https://github.com/cacack/gedcom-go/issues/593)
* **charset:** probe only the header when detecting encoding ([5c5e2c1](https://github.com/cacack/gedcom-go/commit/5c5e2c146ffdd123ba155e7c87c6fde87837c437)), closes [#577](https://github.com/cacack/gedcom-go/issues/577)
* **charset:** replay probed header bytes when the probe read fails ([40c62e5](https://github.com/cacack/gedcom-go/commit/40c62e5cbb989847bc20d2be084ef7bc71d2da74)), closes [#577](https://github.com/cacack/gedcom-go/issues/577)
* **charset:** scan the header probe incrementally ([d7a2cb8](https://github.com/cacack/gedcom-go/commit/d7a2cb810295b0709c98deb24d964b1a64e7d511)), closes [#592](https://github.com/cacack/gedcom-go/issues/592)
* **charset:** validate the bytes read while checking for a UTF-8 BOM ([5642df1](https://github.com/cacack/gedcom-go/commit/5642df1e72750ae24e6c565ff4c7629f7aefa9c5)), closes [#593](https://github.com/cacack/gedcom-go/issues/593)
* **converter:** add VERS under an existing GEDC instead of a second GEDC ([61b1a39](https://github.com/cacack/gedcom-go/commit/61b1a3980561882fe005b221149bd5b428808d0d)), closes [#602](https://github.com/cacack/gedcom-go/issues/602)
* **converter:** build the 7.0 -&gt; 7.0 XRef mapping once and pin its contract ([7a5c95a](https://github.com/cacack/gedcom-go/commit/7a5c95a02793f2de9b26c4db2ddab6947190a1a8)), closes [#604](https://github.com/cacack/gedcom-go/issues/604)
* **converter:** find spaced 5.5.1 pointers when rewriting XRefs for 7.0 ([ede7599](https://github.com/cacack/gedcom-go/commit/ede7599b2a47f5e9a2a51c178a6f86fd569b910f)), closes [#591](https://github.com/cacack/gedcom-go/issues/591)
* **converter:** keep @VOID@ as the null pointer in a 7.0 source ([04da14d](https://github.com/cacack/gedcom-go/commit/04da14ddb268e34d239ab1f6f8da6fe750476479)), closes [#603](https://github.com/cacack/gedcom-go/issues/603)
* **converter:** keep dangling XRef rewrites from joining unrelated records ([55557f6](https://github.com/cacack/gedcom-go/commit/55557f6707de54e2631dad45a8cff2b45f56dae9)), closes [#590](https://github.com/cacack/gedcom-go/issues/590)
* **converter:** normalize invalid xrefs when converting 7.0 to 7.0 ([eea52d9](https://github.com/cacack/gedcom-go/commit/eea52d9a90f51e9ee97829484fa6eb7379a7e47e)), closes [#604](https://github.com/cacack/gedcom-go/issues/604)
* **converter:** pin XRef rewriting alongside calendar translation ([f3df888](https://github.com/cacack/gedcom-go/commit/f3df88816bd319145463e19aab6f4fcc96392d36)), closes [#590](https://github.com/cacack/gedcom-go/issues/590)
* **converter:** rename a 5.5.1 record named @VOID@ when converting to 7.0 ([928f478](https://github.com/cacack/gedcom-go/commit/928f4783726f902cc3ec5f868a74fd408175ce0a)), closes [#603](https://github.com/cacack/gedcom-go/issues/603)
* **converter:** reserve valid dangling pointers when rewriting XRefs ([b74f7af](https://github.com/cacack/gedcom-go/commit/b74f7af39121c82c415ff56871f2de838eed814d)), closes [#590](https://github.com/cacack/gedcom-go/issues/590)
* **converter:** rewrite XRefs invalid in GEDCOM 7.0, not just uppercase them ([52ef167](https://github.com/cacack/gedcom-go/commit/52ef1675a486baeca0ea189f387ab94de451dd05)), closes [#590](https://github.com/cacack/gedcom-go/issues/590)
* **converter:** translate calendar syntax in header DATE values ([3d650bd](https://github.com/cacack/gedcom-go/commit/3d650bd3aa67067778f580834acf7f0071988292)), closes [#605](https://github.com/cacack/gedcom-go/issues/605)
* **converter:** translate citation dates and tighten calendar reporting ([ba1e9cd](https://github.com/cacack/gedcom-go/commit/ba1e9cd5e02b91da0f745535ea8a4885f3937243)), closes [#589](https://github.com/cacack/gedcom-go/issues/589)
* **converter:** translate DATE calendar syntax between 5.5.x and 7.0 ([5af1456](https://github.com/cacack/gedcom-go/commit/5af1456ad3ab7eb399a53299539170b7b1ea2627)), closes [#589](https://github.com/cacack/gedcom-go/issues/589)
* **decoder:** accept spaced identifiers in GEDCOM 5.5 and 5.5.1 ([66acea3](https://github.com/cacack/gedcom-go/commit/66acea3ef6eb80df71a9fda094aa638a00c1c8dd)), closes [#579](https://github.com/cacack/gedcom-go/issues/579)
* **decoder:** relax xref grammar only for a declared 5.5/5.5.1 header ([93b08a6](https://github.com/cacack/gedcom-go/commit/93b08a6424755e74172b6f96942b42ab97581eeb)), closes [#579](https://github.com/cacack/gedcom-go/issues/579)
* **gedcom:** accept GEDCOM 7 calendar keywords in ParseDate ([442a246](https://github.com/cacack/gedcom-go/commit/442a24651bba5a95ef97886cfe1ac3b74c974184)), closes [#581](https://github.com/cacack/gedcom-go/issues/581)
* **gedcom:** count BCE years astronomically in YearsBetween fallback ([09708f0](https://github.com/cacack/gedcom-go/commit/09708f0651ad708eafd8cbbc484e5a72cda518ec)), closes [#588](https://github.com/cacack/gedcom-go/issues/588)
* **gedcom:** document version-agnostic calendar parsing and pin keyword edge cases ([0190b19](https://github.com/cacack/gedcom-go/commit/0190b19cfdc754432fe1d43f0fa17f74c8dff932)), closes [#581](https://github.com/cacack/gedcom-go/issues/581)
* **gedcom:** honor a @VOID@ mapping key at pointer sites outside GEDCOM 7.0 ([2ba988e](https://github.com/cacack/gedcom-go/commit/2ba988e5935e5653b3fe911a6dddb7923f405285)), closes [#603](https://github.com/cacack/gedcom-go/issues/603)
* **gedcom:** recognize 5.5.1 spaced pointers in validator, merge and subset ([eaea05c](https://github.com/cacack/gedcom-go/commit/eaea05c5b2f5463a456184fb82ed838da13a2537)), closes [#591](https://github.com/cacack/gedcom-go/issues/591)
* **gedcom:** reject nonexistent days in Date.ToTime ([4ac3cd2](https://github.com/cacack/gedcom-go/commit/4ac3cd2b4315ec364dac6e023caae2ab62d183c6)), closes [#601](https://github.com/cacack/gedcom-go/issues/601)
* **gedcom:** rewrite spaced pointers in Apply and classify without allocating ([e334ecd](https://github.com/cacack/gedcom-go/commit/e334ecdc15b55a39c2c8c20877d4f24d121509c4)), closes [#579](https://github.com/cacack/gedcom-go/issues/579)
* **gedcom:** share the @VOID@ constant and document Apply's version gate ([86b17db](https://github.com/cacack/gedcom-go/commit/86b17db4c3e726738522182aeac9e6a22521ec1a)), closes [#603](https://github.com/cacack/gedcom-go/issues/603)
* **gedcom:** use astronomical years for BCE dates in Date.ToTime ([28f2af7](https://github.com/cacack/gedcom-go/commit/28f2af732cee192e8ec0c30f0dc3192a980c0d78)), closes [#588](https://github.com/cacack/gedcom-go/issues/588)
* **gedcom:** validate range end dates and BCE leap years in Date.Validate ([f946b08](https://github.com/cacack/gedcom-go/commit/f946b0877093273ed8edf0598617a50424b983fa)), closes [#580](https://github.com/cacack/gedcom-go/issues/580)
* **merge:** carry doc2's declared version into Combine when doc1 has none ([7fbeb87](https://github.com/cacack/gedcom-go/commit/7fbeb872d701b99f4b8b5b68fcccadf3448b52b7)), closes [#591](https://github.com/cacack/gedcom-go/issues/591)
* **merge:** keep DeclaredVersion in step with doc1's raw GEDC in Combine ([49f5aa7](https://github.com/cacack/gedcom-go/commit/49f5aa7c07c72ab340378d8272d9e63954ac8dfa)), closes [#591](https://github.com/cacack/gedcom-go/issues/591)
* **parser:** always record the error that stops TolerateOnly parsing ([ac585cd](https://github.com/cacack/gedcom-go/commit/ac585cd6ab9f2f5ead6980adb86db95c49268373)), closes [#579](https://github.com/cacack/gedcom-go/issues/579)
* **parser:** apply MaxLineBytes to every line scanner ([331f1b2](https://github.com/cacack/gedcom-go/commit/331f1b2a9d4a2cbbe77d175eec9b60c96378b03a)), closes [#578](https://github.com/cacack/gedcom-go/issues/578)
* **parser:** count MaxLineBytes as line content on every path ([86f5187](https://github.com/cacack/gedcom-go/commit/86f5187eb14e479e12bc28fe3d88f78cb2a7c963)), closes [#578](https://github.com/cacack/gedcom-go/issues/578)
* **parser:** report exact byte offsets from RecordIterator and LazyParser iterators ([2138fa1](https://github.com/cacack/gedcom-go/commit/2138fa1d50102e558cd1645ac10b972e133fed19)), closes [#502](https://github.com/cacack/gedcom-go/issues/502)
* **validator:** compare duplicate birth years astronomically ([33b066f](https://github.com/cacack/gedcom-go/commit/33b066f3200dedc76211530712649fd651016b48)), closes [#606](https://github.com/cacack/gedcom-go/issues/606)
* **validator:** give duplicate birth-year credit only to real years ([d2b8695](https://github.com/cacack/gedcom-go/commit/d2b8695de472cd6ca184f4a8301942b61696f152)), closes [#606](https://github.com/cacack/gedcom-go/issues/606)
* **validator:** give malformed "0 BC" birth years no year credit ([abed4c2](https://github.com/cacack/gedcom-go/commit/abed4c28edcd8c3264ddf884497bbb1d8cd811df)), closes [#606](https://github.com/cacack/gedcom-go/issues/606)
* **validator:** report dates that do not exist, such as 31 FEB ([abf3525](https://github.com/cacack/gedcom-go/commit/abf35258c3083a86b7ca26bae472bb4402b8bb34)), closes [#613](https://github.com/cacack/gedcom-go/issues/613)
* **validator:** skip dual-dated dates in the INVALID_DATE check ([220df9d](https://github.com/cacack/gedcom-go/commit/220df9d52bd9ada1926a291de960d96e66dbdb8e)), closes [#613](https://github.com/cacack/gedcom-go/issues/613)
* **validator:** still report impossible days on dual-dated dates ([03bfbbb](https://github.com/cacack/gedcom-go/commit/03bfbbb1cd5159b079f860acc92b6091dfc10a3f)), closes [#613](https://github.com/cacack/gedcom-go/issues/613)
* **version:** detect GEDCOM 7 patch versions (VERS 7.0.N) as 7.0 ([f94c9f2](https://github.com/cacack/gedcom-go/commit/f94c9f29cac05186e2755ad37041208c934e0863)), closes [#587](https://github.com/cacack/gedcom-go/issues/587)
* **version:** document 7.0.N normalization and pin leading-zero patch ([efdbca5](https://github.com/cacack/gedcom-go/commit/efdbca54b3cecc155113d8a6bac2ffebc207b99c)), closes [#587](https://github.com/cacack/gedcom-go/issues/587)
* **version:** ignore a HEAD record with an XRef in DeclaredVersion ([cea0889](https://github.com/cacack/gedcom-go/commit/cea08894f6e6669afd468eabe7df7b4d6a9dee21)), closes [#579](https://github.com/cacack/gedcom-go/issues/579)


### Documentation

* allow conformance fixes in minor releases and list v3.1.0 changes ([b8e6480](https://github.com/cacack/gedcom-go/commit/b8e64800801058eb5f18b85ade31d8954ab59bec))

## [3.0.0](https://github.com/cacack/gedcom-go/compare/v2.4.0...v3.0.0) (2026-09-27)


### ⚠ BREAKING CHANGES

* **converter:** converting a GEDCOM 7.0 document to 5.5 or 5.5.1 now removes negative assertions from the converted document's Record.Tags and Individual/Family.Events instead of carrying them through. A document with negated events and no raw tags now reports "NO tags" data loss, so ConvertOptions.StrictDataLoss fails on it where it previously succeeded. No signature changed, so make api-check does not flag this.
* **validator:** the validator no longer reports DEATH_BEFORE_BIRTH, MARRIAGE_BEFORE_BIRTH or the other date-logic issues from a GEDCOM 7.0 negative assertion, and QualityReport no longer counts a NO BIRT, NO DEAT or a place under a NO event toward IndividualsWithBirthDate, IndividualsWithDeathDate or IndividualsWithPlaces (so MISSING_BIRTH_DATE now fires for an individual whose only BIRT is negated). Issue counts and coverage figures change for documents carrying NO assertions; no signature changed, so make api-check does not flag this.
* **gedcom:** Individual.BirthEvent, DeathEvent, BirthDate and DeathDate no longer return a negated (IsNegative) event or its date. For an individual whose only BIRT/DEAT is a NO assertion they now return nil; where a NO assertion precedes the real event they return the real one. No signature changed, so make api-check does not flag this. Read Individual.NegativeAssertions() for the NO assertions.
* the module path is now github.com/cacack/gedcom-go/v3. Rewrite imports from .../gedcom-go/v2/... to .../gedcom-go/v3/... and run `go get github.com/cacack/gedcom-go/v3@latest`. See docs/guides/migration-v3.md, "Module path".
* **gedcom:** Individual/Family/Source.RefNumber string is now RefNumbers []RefNumber; MediaObject.RefNumbers is []RefNumber (was []string); Individual/Family/Source/Event.UID string is now UIDs []string (Attribute.UIDs is new). v2's scalar held the LAST value; index 0 of the slice is the FIRST. Read s[len(s)-1] to reproduce v2. See docs/guides/migration-v3.md, "REFN and UID are now slices".
* **gedcom:** Source.RepositoryLink is replaced by the slice Source.RepositoryLinks; range over it. Do not substitute RepositoryLinks[0]: it panics on a source with no REPO, and it is the first link where v2 held the last. SourceRepositoryLink.CallNumbers changes from []string to []*CallNumber, and its MediaType and CallNumberMedia fields are removed -- read each CallNumber's MediaType. StreamingValidator's ORPHANED_REPO Details["field"] now reads RepositoryLinks[N].XRef. See docs/guides/migration-v3.md.
* **validator:** DuplicateConfig.RequireExactSurname is removed (it was never read; delete it). DuplicateConfig.NormalizeNames is replaced by DisableNameNormalization with inverted polarity: NormalizeNames: true becomes nothing, NormalizeNames: false becomes DisableNameNormalization: true. MinNameSimilarity, MinConfidence and MaxBirthYearDiff set to 0 now mean the default rather than zero; use validator.ZeroThreshold for an explicit zero. DefaultDuplicateConfig returns *DuplicateConfig; drop the & at call sites. See docs/guides/migration-v3.md.
* **validator:** validator.ValidationError is removed. Every element Validate returns is a *validator.Issue: replace err.(*validator.ValidationError) with errors.As(err, &issue), Code -> Code, Message -> Message, Line -> LineNumber, XRef -> RecordXRef. The code strings are unchanged and now have constants: "BROKEN_XREF" -> CodeBrokenXRef, "MISSING_REQUIRED_FIELD" -> CodeMissingRequiredField, "EMPTY_FAMILY" -> CodeEmptyFamily. Validate now returns every ValidateAll finding (warnings included at the default strictness), honours all options, and formats errors as Issue.String(); ValidateAll gains the three codes; ConversionReport.ValidationIssues lists the wider set. Broken level-1 FAMC/FAMS/SOUR/HUSB/WIFE/CHIL pointers are reported as ORPHANED_* rather than BROKEN_XREF. See docs/guides/migration-v3.md.
* **gedcom:** gedcom.Address.Phone, .Email and .Website are removed. Read repo.Phone / repo.Email / repo.Website ([]string) instead, with a length check before indexing; REPO FAX is newly available as repo.Fax. On Event, Attribute and Submitter addresses the removed fields were always empty -- use the owner's own Phone/Email slices. repo.Address is now nil for a repository with contact lines but no ADDR. These fields carried no Deprecated marker in v2.5.0 and their replacement exists only in v3; this is a documented exception in docs/guides/migration-v3.md and the api-stability pre-major checklist.
* **gedcom:** gedcom.SourceCitationData.Text changed from string to []*gedcom.SourceText, and SourceCitationData is no longer comparable. Read cite.Data.Text[0].Value after a len check (note Text[0] is the first TEXT line; v2 held the last), or range over every entry. Build with []*gedcom.SourceText{{Value: s}}; use nil for no text. Values now include folded CONT/CONC continuation lines. Tags other than CONT, CONC, MIME, LANG or a _-prefixed extension under DATA.TEXT now produce an UNKNOWN_TAG diagnostic. See docs/guides/migration-v3.md.
* **gedcom:** gedcom.Header.Date changed from time.Time to string. It now holds the raw HEAD.DATE value (e.g. "7 AUG 2026"), or "" when the header has none. Replace IsZero() with == "", and parse with gedcom.ParseDate (then (*Date).ToTime) when a structured value is wanted. A hand-built header with no Tags now emits "1 DATE" when Date is set. See docs/guides/migration-v3.md.
* **gedcom:** gedcom.EventOccupation is removed; use gedcom.AttributeOccupation over Individual.Attributes. Attribute.Type is now gedcom.AttributeType.
* **decoder:** Decode and DecodeWithOptions (StrictMode false, the default) now recover from malformed lines instead of returning nil, err; level jumps are clamped rather than kept; and a document can arrive with an error when no line parses or a read fails partway. Set StrictMode: true on DecodeWithOptions to keep v2 behaviour. See docs/guides/migration-v3.md. Downstream, my-family's internal/gedcom/exporter.go re-decode should switch to StrictMode: true.
* **gedcom:** input that ParseCoordinate and Coordinates.AsDecimal previously accepted now returns an error. "Nnan" was NaN with a nil error; "N1e2" was 100; "N0x1p3" was 8; "N1_0.5" was 10.5. All now error. apidiff reports nothing, since no signature changed -- make api-check is clean for these symbols and lists ParseLatitude/ParseLongitude as compatible additions. A caller that checked the error and trusted the value was already correct and needs no change; it now gets the guarantee it was relying on. A caller that stored one of these spellings has data to fix -- the value was never a GEDCOM coordinate. Recorded in docs/governance/policies/api-stability.md with migration notes in docs/guides/migration-v3.md.
* **gedcom:** Document.Subset always includes the header's submitter record, so every subset gains a record -- including Subset(nil), which previously returned an empty document. The submitter record carries any Address, Phone and Email the source recorded, so those contact details are now present in every extract. The encoder also writes hand-built header fields in grammar order, changing the bytes for any hand-built header even when no Submitter is set. make api-check reports none of this: no signature changes. See docs/guides/migration-v3.md and the semantic-break table in docs/governance/policies/api-stability.md.
* **gedcom:** Individual.SpouseInFamilies changes from []string to []FamilyLink. A range loop needs link.FamilyXRef. A site that only prints or marshals the slice still compiles but changes output, so grep for the field as well as building; see docs/guides/migration-v3.md.
* **gedcom:** gedcom.MediaObject.NoteXRefs no longer contains SNOTE pointers; they appear only in SharedNoteXRefs. make api-check reports nothing for this change and adds no entry to its incompatible-changes list -- both fields keep their name and []string type, and only the values they carry change meaning. That is exactly the semantic break the API stability policy's Semantic Breaks section exists to record, and the entry is added there. A caller reading NoteXRefs alone must now also read SharedNoteXRefs; a caller who concatenated the two and deduped must stop deduping, because the slices are disjoint and a dedup would now discard a genuine repeat. MediaObject.AllNotes follows both and needs no change.
* **gedcom:** the deprecated `Notes []string` field is removed from `Individual`, `Family`, `Source`, `Repository`, `Submitter`, `MediaObject`, `Event`, `Attribute`, `SourceCitation`, `LDSOrdinance`, `ChangeDate`, `Association` and `SourceRepositoryLink`. Read `NoteXRefs` (pointers to NOTE/SNOTE records) and `InlineNotes` (note text, CONT/CONC folded), or `AllNotes(doc)` where it is defined. Encoding from the typed model now writes every pointer ahead of every inline note rather than preserving the original interleaved order. See docs/guides/migration-v3.md.
* **gedcom:** gedcom.MediaLink, gedcom.FamilyLink and gedcom.PlaceDetail are no longer comparable with ==. Each gained NoteXRefs and InlineNotes []string fields, and a struct containing a slice cannot be compared. Code that compared these values directly, or used them as map keys, must compare the fields it cares about instead. gedcom.Association and gedcom.SourceRepositoryLink already held slices and are unaffected.
* **gedcom:** gedcom.Event.Place and gedcom.Attribute.Place are removed, as are validator.PlaceConsistencyValidator, validator.NewPlaceConsistencyValidator and validator.CodePlaceCarrierMismatch. Read a place with PlaceName() and write one with SetPlaceName(); see docs/guides/migration-v3.md.
* **gedcom:** `gedcom.Event.Tags` is removed. Custom event subtags live on `Record.Tags`, which is authoritative on encode.
* **version:** `version.IsValidVersion(v)` is removed -- use `v.IsValid()`. `version.DetectVersion` returns `gedcom.Version` only; drop the error result.
* **validator:** validator.Issue.Details no longer carries a "line_number" key. Replace strconv.Atoi(issue.Details["line_number"]) with issue.LineNumber. make api-check is clean -- a map key appears in no signature, so apidiff cannot see it disappear and a caller reading the key silently gets "" rather than a compile error. Listed accordingly in the Semantic Breaks section of the API stability policy.
* **gedcom:** gedcom.Family.NumberOfChildren changes from a string field to a method pair. Replace reads with fam.NumberOfChildren() and writes with fam.SetNumberOfChildren(v). On a decoded family, edit the NCHI entry in Record.Tags to change encoded output.
* **gedcom:** gedcom.SourceCitation.Quality changes from int to *int. Replace `cite.Quality` with a nil check plus dereference, and `Quality: 3` with a pointer to 3. nil now means "no QUAY tag"; a pointer to 0 means a real QUAY 0, which v2 could not represent.
* **gedcom:** gedcom.Note.Continuation and gedcom.Note.FullText() are removed. Put the whole body in Note.Text with embedded newlines and let the encoder split it into CONT/CONC -- which also enforces MaxLineLength and prevents an embedded newline forging a record, neither of which a hand split does. Replace note.FullText() with note.Text.
* **converter:** converter.ConvertOptions.PreserveUnknownTags is replaced by ReportPreservedTags and MapEXIDToVendorTags. Set both true where the old field was true; omit both where it was false. A wholly zero &ConvertOptions{} now behaves as DefaultOptions() rather than as all-false.
* **gedcom:** Individual.ParentalFamilies is renamed to FamiliesAsChild and Individual.SpouseFamilies to FamiliesAsSpouse. A caller who had the two transposed will see corrected results, with no compiler signal that the behaviour changed.
* **encoder:** encoder.EncodeOptions.LineEnding now defaults to "\n" when empty. A caller who relied on the empty string producing output with no line separators -- which was unparseable GEDCOM -- gets "\n" instead.
* **validator:** validator.Strictness constants are renumbered. Old: Relaxed=0, Normal=1, Strict=2. New: Normal=0, Relaxed=1, Strict=2. Callers who named the constants are unaffected. A Strictness integer persisted to a config file, database column or API payload changes meaning on upgrade and must be remapped.
* **encoder:** encoder.EncodeOptions.PreserveUnknownTags is replaced by DropUnknownTags with the opposite sense. Delete `PreserveUnknownTags: true` from options literals — it is now the default; set `DropUnknownTags: true` only where dropping was actually wanted.
* **decoder:** decoder.DecodeOptions.MaxNestingDepth and gedcom/testing.WithHeaderTagComparison are removed. Delete the field assignment and delete the argument respectively; neither did anything.
* **gedcom:** Source.RepositoryRef and Source.Repository are removed. Use SourceRepositoryLink instead: RepositoryRef -> RepositoryLink.XRef, Repository -> RepositoryLink.Inline.
* **decoder:** gedcom.ChangeDate, gedcom.LDSOrdinance and gedcom.SourceCitation gained []string note fields and are therefore no longer comparable with ==. Callers comparing these by value must compare the fields they care about instead; pointer comparison is unaffected, and the API hands all three out as pointers.

### Features

* **converter:** split PreserveUnknownTags and fix the zero value ([62a9bd6](https://github.com/cacack/gedcom-go/commit/62a9bd66f36f7060c095550f4e62cfd28f3eb950))
* **decoder:** decode full EVENT_DETAIL on events, attributes and family attributes ([fd7a09c](https://github.com/cacack/gedcom-go/commit/fd7a09c27d633a099aceca25a43c4314833ee2e0))
* **decoder:** honour StrictMode in every decode entry point ([1c876cd](https://github.com/cacack/gedcom-go/commit/1c876cd88423567612d560867e65718cf7110cf3)), closes [#490](https://github.com/cacack/gedcom-go/issues/490)
* **decoder:** honour the two published "will be removed in v3" markers ([01872c7](https://github.com/cacack/gedcom-go/commit/01872c7d0bc420e207d6105c9d560bc6d24c0253))
* **encoder:** invert PreserveUnknownTags to DropUnknownTags ([9721eb1](https://github.com/cacack/gedcom-go/commit/9721eb1bdb77831866a8ededd8b8bac465ff1987))
* **gedcom:** add nil-safe PlaceName accessors on Event and Attribute ([4f5f0cb](https://github.com/cacack/gedcom-go/commit/4f5f0cb5c9e9128fa5cbdb2cb84b7f49903954a0)), closes [#506](https://github.com/cacack/gedcom-go/issues/506)
* **gedcom:** add OccurredEvents/NegativeAssertions helpers ([b7ef852](https://github.com/cacack/gedcom-go/commit/b7ef8524c4d7b9a81084ea004130610841dc6e28))
* **gedcom:** keep every source REPO link and pair each CALN with its MEDI ([18dd31d](https://github.com/cacack/gedcom-go/commit/18dd31d42982d7eb38ed9cd767e89748a1f045cc)), closes [#553](https://github.com/cacack/gedcom-go/issues/553)
* **gedcom:** move Address.Phone/Email/Website to Repository as slices ([b686dff](https://github.com/cacack/gedcom-go/commit/b686dff2dcf32bd20014b7bed2654c4841aa36a2)), closes [#494](https://github.com/cacack/gedcom-go/issues/494) [#508](https://github.com/cacack/gedcom-go/issues/508)
* **gedcom:** partition MediaObject NoteXRefs and SharedNoteXRefs ([1633407](https://github.com/cacack/gedcom-go/commit/16334075335f6001eb30c617eaea0816afa9b5a5)), closes [#499](https://github.com/cacack/gedcom-go/issues/499)
* **gedcom:** promote SpouseInFamilies to []FamilyLink ([6bb2de0](https://github.com/cacack/gedcom-go/commit/6bb2de088ba00460556dc638704da0975c40a9c7)), closes [#534](https://github.com/cacack/gedcom-go/issues/534)
* **gedcom:** remove Event.Tags, which no code path read or wrote ([5be5b45](https://github.com/cacack/gedcom-go/commit/5be5b455b5e75df159abc5ac09b8b277b1b17f22)), closes [#492](https://github.com/cacack/gedcom-go/issues/492)
* **gedcom:** remove EventOccupation and add AttributeType constant set ([92bde0d](https://github.com/cacack/gedcom-go/commit/92bde0d9f21d5750c778e1c23412d0292aa160c7)), closes [#484](https://github.com/cacack/gedcom-go/issues/484)
* **gedcom:** remove Note.Continuation and Note.FullText() ([f38b128](https://github.com/cacack/gedcom-go/commit/f38b12859ca828792f63791ddb03450448d9a87f))
* **gedcom:** remove Source.RepositoryRef and Source.Repository ([2d0effb](https://github.com/cacack/gedcom-go/commit/2d0effbac9845791822cf816fa7b95c4ff8576f8)), closes [#476](https://github.com/cacack/gedcom-go/issues/476)
* **gedcom:** remove the deprecated Notes fields ([9c13e11](https://github.com/cacack/gedcom-go/commit/9c13e11a905a6dfb12e36129fdef85cb679b9251))
* **gedcom:** remove the legacy Event.Place and Attribute.Place scalars ([09aabd3](https://github.com/cacack/gedcom-go/commit/09aabd3aea5ae93e1b6b04a5a120260d367cd238)), closes [#483](https://github.com/cacack/gedcom-go/issues/483)
* **gedcom:** rename ParentalFamilies/SpouseFamilies to state the role ([c8c1499](https://github.com/cacack/gedcom-go/commit/c8c1499bee0c3e9ecb39b9691746d7ccf75402c8))
* **gedcom:** replace Family.NumberOfChildren with an accessor pair ([409106b](https://github.com/cacack/gedcom-go/commit/409106b2c984d26a2b85af4a70b3bb3ce42ca6cf))
* **gedcom:** retype Header.Date to string and populate it ([4417290](https://github.com/cacack/gedcom-go/commit/4417290177c5915737a40e411ba1a605b55aa7c5)), closes [#495](https://github.com/cacack/gedcom-go/issues/495)
* **gedcom:** retype SourceCitation.Quality from int to *int ([7cd1782](https://github.com/cacack/gedcom-go/commit/7cd17828f82ae8c26c8850acf1c51d123fd64703))
* **gedcom:** retype SourceCitationData.Text to []*SourceText ([5e1c4ee](https://github.com/cacack/gedcom-go/commit/5e1c4ee2f1786376d72d744eb230865cba43f58b))
* **gedcom:** split notes on MediaLink, FamilyLink, PlaceDetail, Association and SourceRepositoryLink ([62fb04d](https://github.com/cacack/gedcom-go/commit/62fb04d83fc054b6506197d8966c8ebfb92c030b)), closes [#472](https://github.com/cacack/gedcom-go/issues/472) [#470](https://github.com/cacack/gedcom-go/issues/470)
* **gedcom:** type repeatable REFN and UID as slices ([8ef49e3](https://github.com/cacack/gedcom-go/commit/8ef49e3ebb587186c963881047ae49783481cd8c)), closes [#554](https://github.com/cacack/gedcom-go/issues/554)
* move the module path to github.com/cacack/gedcom-go/v3 ([fe20992](https://github.com/cacack/gedcom-go/commit/fe20992b4c68a07887b378862f4d61cc2f31aaeb)), closes [#516](https://github.com/cacack/gedcom-go/issues/516)
* **validator:** add Issue.LineNumber and populate it where the line is known ([b3130d6](https://github.com/cacack/gedcom-go/commit/b3130d60222b1b2cf772fdf4016e0c33b87598ae))
* **validator:** bound duplicate detection on adversarial input ([7a2a0a7](https://github.com/cacack/gedcom-go/commit/7a2a0a75eb28ee73b523853ccd910933ee852358)), closes [#530](https://github.com/cacack/gedcom-go/issues/530)
* **validator:** flag events whose two place carriers disagree ([29d849a](https://github.com/cacack/gedcom-go/commit/29d849a43c5f8f1fa94ea91ec333356898ab6241)), closes [#519](https://github.com/cacack/gedcom-go/issues/519)
* **validator:** rebuild Validate over ValidateAll and remove ValidationError ([1a5a1d6](https://github.com/cacack/gedcom-go/commit/1a5a1d65c9c082beca2988605e065cc3cf653095))
* **validator:** remove the line_number Details key now LineNumber exists ([5e4bb44](https://github.com/cacack/gedcom-go/commit/5e4bb44e7bd83d273ac7d2c7033a194ae36bd66f))
* **validator:** renumber Strictness so the zero value is Normal ([5dea800](https://github.com/cacack/gedcom-go/commit/5dea800ef569d74c284f76af829cb34075a81fc6))
* **validator:** report overlapping media object note pointers ([64811b6](https://github.com/cacack/gedcom-go/commit/64811b6499f7d4a2fc897b38ab96061d432c7a58)), closes [#536](https://github.com/cacack/gedcom-go/issues/536)
* **version:** remove IsValidVersion and DetectVersion's always-nil error ([52e253d](https://github.com/cacack/gedcom-go/commit/52e253db0ec8b724c20b44e765edae9dceedc683)), closes [#488](https://github.com/cacack/gedcom-go/issues/488)


### Bug Fixes

* **converter:** drop negative assertions on downgrade to 5.5/5.5.1 ([7a7049d](https://github.com/cacack/gedcom-go/commit/7a7049dde60765acc780d3f0e79c2496f133217c))
* **decoder:** keep the text of a non-pointer SNOTE on a media record ([581aa7a](https://github.com/cacack/gedcom-go/commit/581aa7a3a8abcf368cae1611829ec0fbaf98951c))
* **encoder:** default LineEnding so a bare EncodeOptions really is lossless ([e69637e](https://github.com/cacack/gedcom-go/commit/e69637e8b2911137816b56016318a7ae5067046a)), closes [#486](https://github.com/cacack/gedcom-go/issues/486)
* **encoder:** emit PLAC when only PlaceDetail is set ([d860eee](https://github.com/cacack/gedcom-go/commit/d860eee526586c21c144480571f0d74c4f3471c7)), closes [#505](https://github.com/cacack/gedcom-go/issues/505)
* **gedcom:** make BirthEvent and DeathEvent nil-receiver safe ([3e3ae10](https://github.com/cacack/gedcom-go/commit/3e3ae106303f9a58b5d1acbc56bcb5a1bc179ab8))
* **gedcom:** populate Header.Submitter from HEAD.SUBM ([4fc7a5f](https://github.com/cacack/gedcom-go/commit/4fc7a5fbb5f2537deae07e3101d4a6ecbd6b5ca0)), closes [#503](https://github.com/cacack/gedcom-go/issues/503) [#479](https://github.com/cacack/gedcom-go/issues/479)
* **gedcom:** reject non-finite and non-decimal coordinate values ([47649f6](https://github.com/cacack/gedcom-go/commit/47649f69016bdcad199aff8fa34917e78998afd2)), closes [#504](https://github.com/cacack/gedcom-go/issues/504) [#479](https://github.com/cacack/gedcom-go/issues/479)
* **gedcom:** skip negative assertions in birth/death accessors ([b8a2faf](https://github.com/cacack/gedcom-go/commit/b8a2fafdc1fccc7e543e6d774ad425e0be9a8897))
* **validator:** carry the record's line number, and stop overriding zero options ([e06bac9](https://github.com/cacack/gedcom-go/commit/e06bac9d820cee465186a3432ffc0d454071c8a1))
* **validator:** make DuplicateConfig zero values select its defaults ([5e85143](https://github.com/cacack/gedcom-go/commit/5e8514331a813509398556fc656e1574ac2249ea)), closes [#555](https://github.com/cacack/gedcom-go/issues/555)
* **validator:** skip negative assertions in date-logic and quality checks ([2f15a0f](https://github.com/cacack/gedcom-go/commit/2f15a0fe5fd292348cc249664b3d28cd371a718f))


### Performance Improvements

* **validator:** hoist name normalization out of duplicate pair loop ([6e99f5f](https://github.com/cacack/gedcom-go/commit/6e99f5fb6f2cf5a5aa04d5319a2468bb82aabb09)), closes [#529](https://github.com/cacack/gedcom-go/issues/529)

## [2.4.0](https://github.com/cacack/gedcom-go/compare/v2.3.1...v2.4.0) (2026-08-24)


### Features

* **gedcom:** add Attribute.TypeDetail and type FACT into Attributes ([e549a74](https://github.com/cacack/gedcom-go/commit/e549a74fb351d19a3f8212ce930dda85cc2dd7e2)), closes [#386](https://github.com/cacack/gedcom-go/issues/386)


### Bug Fixes

* **charset:** count bare CR as a line break when locating bad bytes ([f0b7d8b](https://github.com/cacack/gedcom-go/commit/f0b7d8b359e64c016d884c301c25167f3713178a)), closes [#376](https://github.com/cacack/gedcom-go/issues/376)
* **charset:** deliver the valid prefix of a rejected chunk ([a7d1b0e](https://github.com/cacack/gedcom-go/commit/a7d1b0e0adc0e7d3fc39259f0552b2cbd95bab6d)), closes [#382](https://github.com/cacack/gedcom-go/issues/382)
* **charset:** report the physical line of an invalid ANSEL byte ([d892c21](https://github.com/cacack/gedcom-go/commit/d892c211270926d28117020cad09c27af563c004)), closes [#403](https://github.com/cacack/gedcom-go/issues/403)
* **converter,merge:** reconcile the other producers of Header.Tags ([2dc3e64](https://github.com/cacack/gedcom-go/commit/2dc3e64aca64b293407f5b6728080d862f273626))
* **converter:** update the version tag, not just the version field ([b6c1183](https://github.com/cacack/gedcom-go/commit/b6c1183fae85386852e1ec5c5cabcfb1687c1660))
* **decoder:** decode individual EVEN into Individual.Events ([acf2887](https://github.com/cacack/gedcom-go/commit/acf28876d07a5e943c555f3354cbd19d3e73f9df)), closes [#378](https://github.com/cacack/gedcom-go/issues/378)
* **decoder:** fold CONT into Note.Text, not just CONC ([986119b](https://github.com/cacack/gedcom-go/commit/986119b53aed70d7d770f984cab8a39f923e6bfe)), closes [#439](https://github.com/cacack/gedcom-go/issues/439) [#442](https://github.com/cacack/gedcom-go/issues/442)
* **decoder:** keep a level-0 HEAD/TRLR line that carries an XRef ([cac4ff8](https://github.com/cacack/gedcom-go/commit/cac4ff8b2cd2d730fb522703c87d080b6781cb54)), closes [#396](https://github.com/cacack/gedcom-go/issues/396)
* **decoder:** keep a recovered XRef on the subordinate tag ([a31b90c](https://github.com/cacack/gedcom-go/commit/a31b90c14873a52a0e335202e3a1bb2935623736)), closes [#395](https://github.com/cacack/gedcom-go/issues/395)
* **decoder:** normalize padded token values after the delimiter fix ([7d545f0](https://github.com/cacack/gedcom-go/commit/7d545f02b3ffdecb19286a34ad8637f2dde337b5)), closes [#426](https://github.com/cacack/gedcom-go/issues/426)
* **decoder:** recognize standard INDI tags instead of UNKNOWN_TAG ([c5d61f5](https://github.com/cacack/gedcom-go/commit/c5d61f56026de04e137bc0b634ad14bb0e7401b4)), closes [#375](https://github.com/cacack/gedcom-go/issues/375)
* **encoder:** write a level-0 record's value, safely ([7901cd7](https://github.com/cacack/gedcom-go/commit/7901cd7310ac36d96f134ccdab2d543860c5f700))
* **encoder:** write a multiline tag value back as CONT lines ([75f5270](https://github.com/cacack/gedcom-go/commit/75f5270375f7ac6476d902271edf2c20f67caf55)), closes [#419](https://github.com/cacack/gedcom-go/issues/419)
* **encoder:** write a nil as nothing instead of panicking ([e51a5d3](https://github.com/cacack/gedcom-go/commit/e51a5d3bc1b927ac3dca4e08f3ac37147a2bc7f4)), closes [#438](https://github.com/cacack/gedcom-go/issues/438)
* **encoder:** write the header the document has, not four fields of it ([d71c35a](https://github.com/cacack/gedcom-go/commit/d71c35a07afa7d8a796611877f41208d9273a358))
* **gedcom:** carry note and external-ID fields through every Clone ([176e386](https://github.com/cacack/gedcom-go/commit/176e386fea07e2cf53388ef0fff20f0744917c9f))
* **gedcom:** compose calendar escapes with date modifiers ([e844494](https://github.com/cacack/gedcom-go/commit/e84449481a74cf5e2a202fb9dbb4850514a84138)), closes [#384](https://github.com/cacack/gedcom-go/issues/384)
* **gedcom:** copy IsInterpreted and InterpretedFrom in cloneDate ([1e9d1fe](https://github.com/cacack/gedcom-go/commit/1e9d1fe34b2be428c57fbf7e1e6f8d53469d1230)), closes [#400](https://github.com/cacack/gedcom-go/issues/400)
* **parser:** consume one delimiter space, not the whole run ([2b393ac](https://github.com/cacack/gedcom-go/commit/2b393acb59515e7d55b79a4ff7a802cba7e7e018)), closes [#426](https://github.com/cacack/gedcom-go/issues/426)
* **parser:** reject levels above 99 and classify drops as INVALID_LEVEL ([a1f09c6](https://github.com/cacack/gedcom-go/commit/a1f09c60f142a29d670f94ab5eb77fdc41a3b534)), closes [#379](https://github.com/cacack/gedcom-go/issues/379)
* **parser:** report and recover a level-0 XRef with no closing @ ([b53c7ad](https://github.com/cacack/gedcom-go/commit/b53c7ad4de99f22fb89fb32eb330f9f4b8db53ba)), closes [#385](https://github.com/cacack/gedcom-go/issues/385)
* **parser:** report and recover XRefs containing a space ([c3c264a](https://github.com/cacack/gedcom-go/commit/c3c264a825210ef09b1248457e745ef0f4400fd4)), closes [#377](https://github.com/cacack/gedcom-go/issues/377)
* **parser:** report the offending byte's physical line on read errors ([25d369d](https://github.com/cacack/gedcom-go/commit/25d369d9be8955c57a8f8efa616f3fecfe8eb729)), closes [#376](https://github.com/cacack/gedcom-go/issues/376)
* **parser:** report the reader error, not the line it truncated ([9b2dea5](https://github.com/cacack/gedcom-go/commit/9b2dea5ecc8fd175e8f2709daff7c845d3152579)), closes [#382](https://github.com/cacack/gedcom-go/issues/382)
* skip nils library-wide instead of panicking on them ([902f70c](https://github.com/cacack/gedcom-go/commit/902f70c54d1e3cb877577088c53ad10c2018ef1e))

## [2.3.1](https://github.com/cacack/gedcom-go/compare/v2.3.0...v2.3.1) (2026-07-19)


### Bug Fixes

* **converter:** escape pointer-shaped EXID value in _FSFTID ([#346](https://github.com/cacack/gedcom-go/issues/346)) ([49c053c](https://github.com/cacack/gedcom-go/commit/49c053c4400fee0e30f09ba8dacb659cccebc4aa))
* **decoder:** fold CONT/CONC under SNOTE TRAN into translation value ([42ae842](https://github.com/cacack/gedcom-go/commit/42ae8420f54caae3f2b0b49b287133c9d2ac125d))
* **decoder:** fold note continuations in O(n) ([#340](https://github.com/cacack/gedcom-go/issues/340)) ([0938546](https://github.com/cacack/gedcom-go/commit/0938546f47ae6623522acd2768f35c47a92c0c6e))
* **gedcom:** round-trip escaping for pointer-shaped _FSFTID values ([0323c47](https://github.com/cacack/gedcom-go/commit/0323c47b8275faebb6b73cae4931d43181577a22))

## [2.3.0](https://github.com/cacack/gedcom-go/compare/v2.2.2...v2.3.0) (2026-07-08)


### Features

* map FamilySearch ARK EXID to _FSFTID on 5.5.x downgrade ([3ff3ed6](https://github.com/cacack/gedcom-go/commit/3ff3ed6c91e1a9e71ff2897a5610bfd07dd7ca63))


### Bug Fixes

* reject EXID blocks with subordinates deeper than the ARK TYPE ([bc9a2cf](https://github.com/cacack/gedcom-go/commit/bc9a2cfe1b2fb79e3d824aaed475b8305d4aefb0))

## [2.2.2](https://github.com/cacack/gedcom-go/compare/v2.2.1...v2.2.2) (2026-07-06)


### Bug Fixes

* **decoder:** fold SNOTE CONT/CONC continuations into SharedNote.Text ([0355670](https://github.com/cacack/gedcom-go/commit/0355670671c4ad5ab5c7f8382929307849fc8c96))
* **encoder:** encode entity-only Note.Text via a safe textToTags split ([344f2cc](https://github.com/cacack/gedcom-go/commit/344f2cca852c88a95594427582319e47d429323a))
* **encoder:** encode multi-line SharedNote.Text on the entity-only path ([56a08f1](https://github.com/cacack/gedcom-go/commit/56a08f129a9d691ee5ee8f67896fa13a97ad8867))

## [2.2.1](https://github.com/cacack/gedcom-go/compare/v2.2.0...v2.2.1) (2026-07-06)


### Bug Fixes

* **encoder:** emit EXID on Individual/Family/Source/Repository/Submitter/Media ([d9f0d16](https://github.com/cacack/gedcom-go/commit/d9f0d1634f7cbacd26cb5f6d56f8733a97db4895))

## [2.2.0](https://github.com/cacack/gedcom-go/compare/v2.1.0...v2.2.0) (2026-06-29)


### Features

* **gedcom:** expose structured source-to-repository link ([45caf53](https://github.com/cacack/gedcom-go/commit/45caf5320c54537e2f97f67d2129e2ce59602ca3)), closes [#289](https://github.com/cacack/gedcom-go/issues/289)
* **gedcom:** split inline notes from shared-note XRefs in record types ([38a6247](https://github.com/cacack/gedcom-go/commit/38a62473f14288ff9edaa68e0669f81cc4d00c66)), closes [#290](https://github.com/cacack/gedcom-go/issues/290)


### Bug Fixes

* **decoder:** route SNOTE through split-note path ([5fdbcbe](https://github.com/cacack/gedcom-go/commit/5fdbcbeb6d78159ed7381da8d883cfa4166aea18))
* **encoder:** encode split note fields when they diverge from legacy Notes ([0df4af1](https://github.com/cacack/gedcom-go/commit/0df4af179c927312daa16fd1d2744e2b75cfdcb1))
* **encoder:** preserve note order and emit split notes for all record types ([40795b2](https://github.com/cacack/gedcom-go/commit/40795b23b311e707916031253fc50fbb0e89a28d))
* **gedcom:** correct source-to-repository link edge cases ([dd486de](https://github.com/cacack/gedcom-go/commit/dd486de7ddf1ed0956e0a07afd8257015f7ae71c))
* **gedcom:** include shared notes in MediaObject.AllNotes ([a5bee5c](https://github.com/cacack/gedcom-go/commit/a5bee5c14698f081a636a9ea5d8386a4121e5c6f))
* **gedcom:** return nil from allNotes when no notes resolve ([b29bc8f](https://github.com/cacack/gedcom-go/commit/b29bc8f123c78ebf36e88d927c86b52a00887f66))

## [2.1.0](https://github.com/cacack/gedcom-go/compare/v2.0.1...v2.1.0) (2026-06-01)


### Features

* **gedcom:** add coordinate decimal-degree helpers ([d62d08b](https://github.com/cacack/gedcom-go/commit/d62d08b01cc21296fc50dd49e60251f41b99721e)), closes [#288](https://github.com/cacack/gedcom-go/issues/288)
* **gedcom:** add RequiresGEDCOM7 and MinimumVersion helpers ([ced5327](https://github.com/cacack/gedcom-go/commit/ced5327c7080bc285bf7bbf84b0359c63ae99729)), closes [#291](https://github.com/cacack/gedcom-go/issues/291)


### Bug Fixes

* **decoder:** preserve custom header tags losslessly ([d3a87dd](https://github.com/cacack/gedcom-go/commit/d3a87ddb4d50b3bd12e6470002a5157aab7e4213))

## [2.0.1](https://github.com/cacack/gedcom-go/compare/v2.0.0...v2.0.1) (2026-05-17)


### Bug Fixes

* **build:** set module path to /v2 to make v2 release installable ([e05301b](https://github.com/cacack/gedcom-go/commit/e05301ba9bb8cf33731f289252313b23b223b3c6))

## [2.0.0](https://github.com/cacack/gedcom-go/compare/v1.2.0...v2.0.0) (2026-05-17)


### ⚠ BREAKING CHANGES

* **validator:** `ValidatorConfig` is no longer comparable with `==` due to the addition of the `SkipRules []string` field. Code that compares `ValidatorConfig` values directly will fail to compile — use field-by-field comparison or `reflect.DeepEqual` instead. ([98b3680](https://github.com/cacack/gedcom-go/commit/98b3680a9b6dc0dd44d5b43fd8795b579afcb935), closes [#156](https://github.com/cacack/gedcom-go/issues/156))


### Features

* **api:** add WithOptions facade variants and ValidateOptions alias ([453e98a](https://github.com/cacack/gedcom-go/commit/453e98a633192f5b918448e34e5ad567342375cc))
* **decoder:** recover from malformed indentation in lenient mode ([3d1e74f](https://github.com/cacack/gedcom-go/commit/3d1e74fecc7bd450c390723199b7fb7e264f8eee)), closes [#216](https://github.com/cacack/gedcom-go/issues/216)
* **encoder:** add `TargetVersion` option to override output GEDCOM version in header ([98b3680](https://github.com/cacack/gedcom-go/commit/98b3680a9b6dc0dd44d5b43fd8795b579afcb935)), closes [#156](https://github.com/cacack/gedcom-go/issues/156)
* **encoder:** add `PreserveUnknownTags` option to filter custom tags (default: `true`) ([98b3680](https://github.com/cacack/gedcom-go/commit/98b3680a9b6dc0dd44d5b43fd8795b579afcb935)), closes [#156](https://github.com/cacack/gedcom-go/issues/156)
* **gedcom:** add graph traversal and subset primitives ([83a1959](https://github.com/cacack/gedcom-go/commit/83a1959ee90da17f2a2f2b50e6f8bafffb5cef4e))
* **gedcom:** add Visit, Apply, IsPointerXRef walker primitives ([2d420dc](https://github.com/cacack/gedcom-go/commit/2d420dc3b47cbe43f5f787e7559cd1693e336f95)), closes [#276](https://github.com/cacack/gedcom-go/issues/276)
* **merge:** add RemapXRefs and Combine primitives ([139913a](https://github.com/cacack/gedcom-go/commit/139913a1273066deccb2525f6db19aefb50fb6ea)), closes [#276](https://github.com/cacack/gedcom-go/issues/276)
* **streaming:** stabilize streaming APIs and harden iterator contracts ([ec8214b](https://github.com/cacack/gedcom-go/commit/ec8214bd5e7d0bf310b71fb37ca12acc6873e701)), closes [#220](https://github.com/cacack/gedcom-go/issues/220)
* **validator:** add `MaxErrors` option to limit number of issues returned (default: `0` = unlimited) ([98b3680](https://github.com/cacack/gedcom-go/commit/98b3680a9b6dc0dd44d5b43fd8795b579afcb935)), closes [#156](https://github.com/cacack/gedcom-go/issues/156)
* **validator:** add `SkipRules` option to exclude specific validation rules by code ([98b3680](https://github.com/cacack/gedcom-go/commit/98b3680a9b6dc0dd44d5b43fd8795b579afcb935)), closes [#156](https://github.com/cacack/gedcom-go/issues/156)


### Bug Fixes

* **gedcom:** address CodeRabbit findings on PR [#279](https://github.com/cacack/gedcom-go/issues/279) ([bfda0c2](https://github.com/cacack/gedcom-go/commit/bfda0c2a7443f5b79707a75e28419d6d614f530c)), closes [#276](https://github.com/cacack/gedcom-go/issues/276)


### Miscellaneous Chores

* bump next release to 2.0.0 ([c4dbc15](https://github.com/cacack/gedcom-go/commit/c4dbc15caef3120d0553f401e2545beec8e6c92f))

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
