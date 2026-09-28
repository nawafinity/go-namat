# Changelog

All notable changes will be documented here. The project follows Semantic
Versioning beginning with v1.0.

## Unreleased

### Added

- Pure Go DOCX and DOCM package engine.
- Native bounded expression compiler and evaluator.
- Text insertion, assignments, conditions, loops, table-row repetition,
  aliases, and query resolution.
- Inline PNG, JPEG, GIF, and SVG images with fallback thumbnails.
- External hyperlinks, HTML altChunk, line breaks, and opt-in literal OOXML.
- Metadata reader and privacy-preserving template inspector.
- Context, timeout, iteration, input, ZIP, and output limits.
- Unit, concurrency, security, fuzz, benchmark, and visual fixture tests.

### Changed

- The module root is now a small public facade; DOCX implementation and focused
  package-private tests live in `internal/engine` without changing the public
  import path.
- Test fixtures use synthetic, product-neutral values with an enforced privacy
  policy.
- Shared DOCX test infrastructure is centralized while package-private tests
  remain colocated with the Go packages they verify.
- Public API documentation now describes command constants, rich-content
  fields, and every option.
- The committed test suite is English and product-neutral, with escaped
  multilingual cases for Unicode and RTL behavior.
- A core feature matrix maps every supported capability to automated evidence.
- CI now runs on Linux, Windows, and macOS with package-specific coverage
  thresholds.

### Fixed

- `ListCommands` now honors configured template and ZIP resource limits.
- `RenderTo` now reports `io.ErrShortWrite` when a writer accepts only part
  of the rendered document.
- Empty explicit commands and ambiguous `END-FOR` prefixes are rejected
  instead of being interpreted as insertion expressions.
