# Compatibility

Namat preserves existing OOXML parts and changes only templated XML plus the
relationships and content types required by newly inserted rich content.

| Capability | Status | Notes |
| --- | --- | --- |
| DOCX round trip | Supported | Unmodified ZIP parts and media are preserved |
| DOCM round trip | Supported | VBA parts are copied unchanged |
| Commands split across runs | Supported | Includes commands split across paragraphs |
| Text insertion | Supported | Newlines become Word line breaks by default |
| Assignments | Supported | One native expression per `EXEC` or `SET` |
| Conditions | Supported | Nested paragraph and table-row blocks |
| Loops | Supported | Nested paragraph and table-row blocks with `$idx` |
| PNG JPEG GIF | Supported | Inline drawings with dimensions in centimeters |
| SVG | Supported | Optional fallback thumbnail for older clients |
| Hyperlinks | Supported | External relationship with scheme allowlist |
| HTML | Supported | Word altChunk; portability depends on the reader |
| Literal OOXML | Supported | Disabled by default; trusted input only |
| Query resolver | Supported | Application-defined Go callback |
| Command aliases | Supported | Complete-command aliases and `*name` references |
| Headers and footers | Supported | Text, loops, images, and links use part-local relationships |
| Footnotes and endnotes | Supported | Templated `word/*.xml` parts are processed |
| Metadata | Supported | Reads standard core and extended properties |
| Arbitrary JavaScript | Not supported | Replace scripts with registered Go functions |
| Floating images | Not generated | Inline drawings are intentionally more portable |
| HTML in headers or footers | Not supported | OOXML altChunk is limited to the main document |

Structural marker commands should occupy a complete paragraph or table row.
Use a ternary expression for small inline choices.
