# Roadmap to v1

## M1 — native text engine

- [x] DOCX package round trip
- [x] prefix-preserving OOXML tree
- [x] commands split across runs and paragraphs
- [x] native expression parser and evaluator
- [x] strict insertion, lexical declarations, boolean conditions, and loops
- [x] exact integers and decimals with bounded parser and evaluator complexity
- [x] table-row repetition
- [x] deterministic rendering and concurrency tests
- [x] privacy-preserving template inspector

## M2 — rich content

- [x] PNG, JPEG, GIF, and SVG images
- [x] image fallback thumbnails, rotation, alt text, and captions
- [x] external hyperlinks
- [x] explicit opt-in raw OOXML insertion
- [x] HTML altChunk with isolated relationship updates

## M3 — authoring compatibility

- [x] typed registered-function interface with render context
- [x] data-aware template lint command
- [x] configurable smart-quote normalization
- [x] line-break processing
- [x] metadata reader
- [x] structured error categories and collect mode

## M4 — production hardening

- [ ] golden rendering fixtures for Word, LibreOffice, and Google Docs
  - Acceptance: a synthetic fixture set exercises text, nested structures,
    tables, rich content, headers and footers, notes, DOCM preservation, and
    Arabic/RTL content.
  - Acceptance: results record the client name, client version, operating
    system, test date, and expected visual differences.
  - Acceptance: reviewed reference output or screenshots are stored without
    customer or product data.
- [ ] public compatibility fixtures derived from permissively licensed sources
  - Acceptance: every external source has recorded provenance, license, and
    permitted redistribution terms.
  - Acceptance: committed templates and values are minimized, synthetic, and
    comply with [`TESTING.md`](TESTING.md).
  - Acceptance: the compatibility matrix links each fixture to its evidence.
- [x] fuzz seeds for ZIP, commands, reports, and expressions
- [x] allocation and throughput benchmarks
- [ ] semantic-versioned API and migration guide
  - Acceptance: the exported Go API, template-language contract, error
    categories, defaults, and compatibility policy are reviewed and frozen for
    v1.0.
  - Acceptance: [`MIGRATION.md`](MIGRATION.md) contains the final pre-v1 to
    v1.0 change ledger and an executable consumer checklist.
  - Acceptance: the changelog has a dated v1.0.0 section and the repository has
    a corresponding tag.
- [ ] signed releases for the optional CLI
  - Acceptance: release artifacts declare supported operating-system and
    architecture targets and embed the release version.
  - Acceptance: artifacts have published checksums, signatures, and documented
    verification steps.
  - Acceptance: the release procedure in [`RELEASING.md`](RELEASING.md) has
    been exercised without bypassing the quality or visual-compatibility gates.

Application-specific integrations and migrations intentionally live in their
own product repositories. They are outside the scope of the standalone Namat
library roadmap.
