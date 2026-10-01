# Compatibility

Namat preserves existing OOXML parts and changes only templated XML plus the
relationships and content types required by newly inserted rich content.

The status below describes structural behavior covered by Go tests. It is not
yet a claim of pixel-identical rendering in every Word-compatible client.
Versioned visual fixtures for Microsoft Word, LibreOffice, and Google Docs are
still a pre-v1 roadmap item. Until those fixtures are published, client-specific
visual behavior should be verified with the application's own templates.

| Capability | Status | Notes |
| --- | --- | --- |
| DOCX round trip | Supported | Unmodified ZIP parts and media are preserved |
| DOCM round trip | Supported | VBA parts are copied unchanged |
| Commands split across runs | Supported | Includes commands split across paragraphs |
| Text insertion | Supported | Newlines become Word line breaks by default |
| Lexical declarations | Supported | Immutable `#let` in its containing scope |
| Conditions | Supported | Nested paragraph and table-row blocks |
| Loops | Supported | Nested paragraph and table-row blocks with `loop` metadata |
| PNG JPEG GIF | Supported | Inline drawings with dimensions in centimeters |
| SVG | Supported | Optional fallback thumbnail for older clients |
| Hyperlinks | Supported | External relationship with scheme allowlist |
| HTML | Supported | Word altChunk; portability depends on the reader |
| Explicit raw OOXML | Supported | `@raw-xml`; disabled by default and trusted input only |
| Headers and footers | Structurally supported | Direct tests cover text, loops, images, and part-local link relationships; dedicated client-rendered fixtures are pending |
| Footnotes | Structurally supported | Direct automated text-rendering evidence |
| Endnotes | Structurally supported | Direct tests cover text and part-local link relationships; client-rendered fixtures remain pending |
| Metadata | Supported | Reads standard core and extended properties |
| Arbitrary JavaScript | Not supported | Replace scripts with registered Go functions |
| Floating images | Not generated | Inline drawings are intentionally more portable |
| HTML in headers or footers | Not supported | OOXML altChunk is limited to the main document |

Structural marker commands should occupy a complete paragraph or table row.
Use separate `#if` blocks for choices; v1 intentionally has no ternary syntax.

## Client verification status

| Client | Current evidence | v1 requirement |
| --- | --- | --- |
| Microsoft Word | OOXML structure and round-trip tests | Record tested desktop versions and reviewed golden output |
| LibreOffice | Headless PDF conversion and manual review of `complex-tables.docx` with LibreOffice 7.4.7.2 on Linux, 2026-09-30 | Expand to the release client matrix and reviewed golden set |
| Google Docs | No committed client-rendered golden set yet | Record import/export workflow and expected differences |

HTML altChunk is especially client-dependent. The structural package can be
valid even when a non-Word client ignores or transforms the imported HTML.
