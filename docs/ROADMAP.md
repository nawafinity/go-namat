# Roadmap to v1

## M1 — native text engine

- [x] DOCX package round trip
- [x] prefix-preserving OOXML tree
- [x] commands split across runs and paragraphs
- [x] native expression parser and evaluator
- [x] insertion, assignment, conditions, and loops
- [x] table-row repetition
- [x] deterministic rendering and concurrency tests
- [x] privacy-preserving template inspector

## M2 — rich content

- [x] PNG, JPEG, GIF, and SVG images
- [x] image fallback thumbnails, rotation, alt text, and captions
- [x] external hyperlinks
- [x] literal OOXML insertion
- [x] HTML altChunk with isolated relationship updates

## M3 — authoring compatibility

- [x] query resolver interface
- [x] command aliases
- [x] configurable smart-quote normalization
- [x] line-break processing
- [x] metadata reader
- [x] structured error categories and collect mode

## M4 — production hardening

- [ ] golden rendering fixtures for Word, LibreOffice, and Google Docs
- [ ] public compatibility fixtures derived from permissively licensed sources
- [x] fuzz seeds for ZIP, commands, reports, and expressions
- [x] allocation and throughput benchmarks
- [ ] semantic-versioned API and migration guide
- [ ] signed releases for the optional CLI

Application-specific integrations and migrations intentionally live in their
own product repositories. They are outside the scope of the standalone Namat
library roadmap.
