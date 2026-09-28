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

- [ ] PNG, JPEG, GIF, and SVG images
- [ ] image fallback thumbnails, rotation, alt text, and captions
- [ ] external hyperlinks
- [ ] literal OOXML insertion
- [ ] HTML altChunk with isolated relationship updates

## M3 — authoring compatibility

- [ ] query resolver interface
- [ ] command aliases
- [ ] configurable smart-quote normalization
- [ ] line-break processing
- [ ] metadata reader
- [ ] structured error classes and fail-fast/collect modes

## M4 — production hardening

- [ ] golden rendering fixtures for Word, LibreOffice, and Google Docs
- [ ] public compatibility fixtures derived from permissively licensed sources
- [ ] fuzzing for ZIP, XML, commands, and expressions
- [ ] allocation and throughput benchmarks
- [ ] semantic-versioned API and migration guide
- [ ] signed releases for the optional CLI

## M5 — application migration

- [ ] port existing report helper functions to typed Go
- [ ] run old and new engines against the same approved test fixtures
- [ ] compare document text, tables, media, and relationships automatically
- [ ] switch production only after calculation and report parity is proven
