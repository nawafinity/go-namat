# Changelog

All notable changes will be documented here. The project follows Semantic
Versioning beginning with v1.0.

## Unreleased

### Added

- Pure Go DOCX and DOCM package engine.
- Native bounded expression compiler and evaluator.
- Strict expression insertion, immutable lexical declarations, boolean
  conditions, nested loops, and table-row repetition.
- Inline PNG, JPEG, GIF, and SVG images with fallback thumbnails.
- External hyperlinks, HTML altChunk, line breaks, and explicit opt-in raw OOXML.
- Metadata reader and privacy-preserving template inspector.
- Context, timeout, iteration, expression-complexity, input, ZIP, and output limits.
- Exact integer and decimal values plus `DecodeJSON` for lossless JSON numbers.
- Typed registered-function signatures and a data-aware `namat lint` command.
- Unit, concurrency, security, fuzz, and benchmark tests.
- A public CC0 complex-table DOCX fixture with merged cells, nested tables,
  nested row loops, and byte-preservation evidence.
- LibreOffice headless conversion in CI and a tag-triggered CLI release
  workflow with checksums and GitHub build-provenance attestations.
- An advanced runnable example with three-level data, nested Word tables, and
  registered text, aggregate, link, and image functions called inside loops.

### Changed

- The prototype template syntax was replaced with the unambiguous strict v1
  grammar: `#if`, `#each`, `#let`, closing directives, and `@` rich-content
  directives. There is deliberately no legacy compatibility mode.
- Expression semantics are now strict: boolean-only conditions, exact numeric
  arithmetic, no mixed-type coercion, exact JSON names, filters, and distinct
  missing/null values.
- Every templated DOCX part renders with an independent lexical variable scope.
- The command scanner now understands quoting, escaping, and nested brackets,
  including `]]` inside a string literal.

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
  gates fixed at 100%, tests both Go 1.23 and the current Go release line, and
  fuzzes all five declared targets.
- Rendering now enforces output size while streaming ZIP bytes and rechecks
  package-part and uncompressed-size limits after generated content is added.
- Commands remain discoverable when delimiter characters themselves are split
  across Word runs, and nested-table row markers no longer attach to an outer
  table row.
- Security documentation now describes the registered-function boundary,
  cooperative timeout behavior, callback concurrency, and default resource
  limits.
- The roadmap now defines acceptance criteria for visual compatibility,
  migration, and signed CLI releases; draft migration and release guides were
  added without marking those pre-v1 gates complete.
- Compatibility documentation now separates structurally tested behavior from
  pending client-rendered visual verification.

### Fixed

- Expression calls now accept only built-in members and functions registered in
  `Options.Functions`; function values in render data are no longer invoked
  through reflection.
- Forced CLI replacement now attempts an atomic platform rename first and uses
  a recoverable sibling backup when direct replacement is unavailable.
- Non-forced CLI output now commits through an atomic no-replace link, preventing
  a racing process from having its newly created destination overwritten.
- CLI output writes now detect and report `io.ErrShortWrite`.
- Structural conditions now reject non-boolean values.
- OOXML serialization now preserves document-level whitespace instead of
  emitting character references that strict clients reject outside the root
  element.
- Nil and zero-value public templates now return `ErrInvalidTemplate` from
  `Render` and `RenderTo` instead of panicking.
- `ListCommands` now honors configured template and ZIP resource limits.
- `RenderTo` now reports `io.ErrShortWrite` when a writer accepts only part
  of the rendered document.
- Empty explicit commands and ambiguous `END-FOR` prefixes are rejected
  instead of being interpreted as insertion expressions.
