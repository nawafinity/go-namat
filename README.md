# Namat

**Namat (نَمَط)** is a pure Go template engine for Microsoft Word DOCX and
DOCM files. It turns documents designed in Word into data-driven reports while
preserving the package, styles, media, headers, footers, macros, and other
untouched OOXML parts.

Namat does not bundle Node.js, execute JavaScript, start helper processes, or
require Microsoft Office at runtime. Template expressions are compiled and
evaluated by a small, bounded Go engine.

> Project status: pre-v1. The core API and template language are usable and
> tested, but semantic-version compatibility begins with v1.0.

## Why Namat

- **Pure Go:** one library, one application binary, and no sidecar executable.
- **Word-first authoring:** keep page layout and styling in a normal DOCX
  template instead of rebuilding documents in code.
- **Safe expressions:** templates cannot access files, processes, environment
  variables, reflection APIs, or the network unless the application exposes a
  specific Go function.
- **Accurate structure:** conditions and loops operate on paragraphs and whole
  table rows, including nested blocks.
- **Rich content:** PNG, JPEG, GIF, SVG with fallback thumbnails, external
  links, HTML altChunk, line breaks, and opt-in literal OOXML.
- **Production controls:** context cancellation, timeouts, aggregate loop
  limits, ZIP bomb limits, output limits, typed errors, and atomic CLI output.
- **Reusable compilation:** compile a template once, then render it safely from
  multiple goroutines.

## Installation

```bash
go get github.com/nawafinity/go-namat
```

Namat has no third-party Go dependencies.

## Quick start

Create a DOCX in Word and type commands directly into it:

```text
Report for [[customer.name]]

[[IF invoice.total > 0]]
Total: [[money(invoice.total)]]
[[ELSE]]
No balance is due.
[[END-IF]]

[[FOR item IN invoice.items]]
[[$idx + 1]]. [[$item.name]]
[[END-FOR item]]
```

Compile once and render many reports:

```go
package main

import (
    "context"
    "os"

    "github.com/nawafinity/go-namat"
)

func main() {
    source, err := os.ReadFile("invoice-template.docx")
    if err != nil {
        panic(err)
    }

    template, err := namat.Compile(source, namat.Options{})
    if err != nil {
        panic(err)
    }

    report, err := template.Render(context.Background(), map[string]any{
        "customer": map[string]any{"name": "Nawaf"},
        "invoice": map[string]any{
            "total": 250,
            "items": []map[string]any{{"name": "Assessment"}},
        },
    })
    if err != nil {
        panic(err)
    }
    if err := os.WriteFile("invoice.docx", report, 0o644); err != nil {
        panic(err)
    }
}
```

See [`examples/complete`](examples/complete) for a self-contained Arabic report
with a table, image, link, conditions, and loops.

## Commands

Commands use `[[` and `]]` by default. Delimiters are configurable.

| Command | Purpose |
| --- | --- |
| `[[value]]`, `[[INS value]]`, `[[= value]]` | Insert text |
| `[[EXEC name = expression]]`, `[[! name = expression]]` | Assign a local value without output |
| `[[SET name = expression]]` | Explicit assignment alias |
| `[[IF expression]]` / `[[ELSE]]` / `[[END-IF]]` | Conditional paragraphs or rows |
| `[[FOR item IN values]]` / `[[END-FOR item]]` | Repeat paragraphs or table rows |
| `[[IMAGE expression]]` | Insert an inline image |
| `[[LINK expression]]` | Insert an external hyperlink |
| `[[HTML expression]]` | Insert a Word HTML altChunk |
| `[[RAW-XML expression]]` | Insert literal OOXML when explicitly enabled |
| `[[QUERY query text]]` | Resolve report data through a Go callback |
| `[[ALIAS name INS expression]]`, `[[*name]]` | Define and reuse a complete command |

Structural `IF` and `FOR` markers should occupy their own paragraph or their
own table row. `HTML` and `RAW-XML` must occupy their own paragraph.

Word often splits commands into multiple XML runs, and may occasionally split
one command across adjacent paragraphs. Namat normalizes both forms before
compilation.

## Expression language

The expression engine intentionally supports data access rather than arbitrary
code execution:

- maps, structs, JSON names, slices, arrays, strings, and pointers;
- property access, optional chaining, and indexes;
- `+ - * / %`, comparisons, `&&`, `||`, `!`, and `??`;
- ternary expressions: `condition ? yes : no`;
- arrays and objects: `[1, 2]`, `{ url: url, label: name }`;
- template strings: `` `Score: ${score}` ``;
- string helpers such as `.slice()`, `.trim()`, `.toUpperCase()`,
  `.contains()`, `.startsWith()`, and `.length`;
- collection helpers such as `.join()`, `.includes()`, and `.length`;
- explicitly registered Go functions.

Loop variables use a `$` prefix. `$idx` is the zero-based loop index.

```go
options := namat.Options{
    Functions: map[string]namat.Function{
        "money": func(args ...any) (any, error) {
            return formatMoney(args[0]), nil
        },
    },
}
```

Host functions are the native replacement for JavaScript helpers: they are
typed Go code that can be unit tested, profiled, and audited normally.

## Images

An `IMAGE` expression returns `namat.Image` or an object with equivalent fields:

```go
return namat.Image{
    Data:      pngBytes,
    Extension: "png",
    Width:     8, // centimeters
    Height:    4,
    Alt:       "Quarterly chart",
    Rotation:  0,
}, nil
```

Supported formats are PNG, JPEG, GIF, and SVG. SVG images may provide a PNG,
JPEG, or GIF `Thumbnail` for older Word versions and Explorer previews. Images
are inline because inline drawings are the most portable option across Word and
LibreOffice.

## Hyperlinks

`LINK` accepts `namat.Link` or an object expression:

```text
[[LINK ({ url: project.url, label: project.name })]]
```

Only `http`, `https`, and `mailto` are allowed by default. Change
`AllowedLinkSchemes` only when the surrounding application has an explicit
reason to permit another scheme.

## HTML and literal OOXML

`HTML` uses the OOXML `altChunk` mechanism. Microsoft Word supports altChunk;
LibreOffice and Google Docs may not import it consistently.

Literal OOXML is disabled by default because it bypasses text escaping. Enable
it only for trusted templates and trusted values:

```go
options := namat.Options{AllowRawXML: true}
```

With that option, either use `RAW-XML` or embed fragments between the default
`||` delimiters in inserted text:

```text
first||<w:br/>||second
```

## Query resolver

A template may declare one `QUERY`. Namat passes its text unchanged to the
application before rendering:

```go
options := namat.Options{
    QueryResolver: func(ctx context.Context, query string) (any, error) {
        return database.ReportData(ctx, query)
    },
}
```

Namat does not interpret SQL, GraphQL, or another query language and never
opens a network connection itself.

## Streaming-style APIs

The package supports byte slices and `io.Reader`/`io.Writer` integrations:

```go
template, err := namat.CompileReader(reader, options)
err = template.RenderTo(ctx, writer, data)
```

The DOCX package is still assembled transactionally in memory so an error never
leaves a partially written document.

## Inspection and metadata

```go
commands, err := namat.ListCommands(templateBytes, options)
metadata, err := namat.GetMetadata(documentBytes)
```

Metadata page and word counts are cached Word properties; Namat reports them
but does not pretend to recalculate layout-dependent values.

## CLI

The optional command uses the same library:

```text
namat inspect template.docx
namat inspect --json template.docx
namat metadata document.docx
namat render --data data.json --out report.docx template.docx
```

`inspect` hides command expressions unless `--details` is supplied. `render`
refuses to overwrite an existing output unless `--force` is explicit and uses
an atomic temporary-file rename.

Build a small native executable:

```bash
go build -trimpath -ldflags="-s -w" ./cmd/namat
```

## Safety limits

`Options` includes limits for template bytes, individual ZIP parts, total
uncompressed content, package part count, output bytes, loop iterations, and
render duration. Defaults are conservative enough for normal report templates
and can be tightened by server applications.

Raw XML is opt-in. Object results from `INS` are rejected by default to prevent
accidental `map[...]` output. Links use an allowlist. ZIP paths, duplicate
parts, oversized parts, and traversal names are rejected.

Read [`SECURITY.md`](SECURITY.md) before accepting templates from untrusted
users.

## Testing

```bash
go test ./...
go test -race ./...
go vet ./...
go test -run '^$' -bench . -benchmem ./...
go test -fuzz=Fuzz -fuzztime=30s ./...
```

The suite covers expressions, split runs, split paragraphs, nested structures,
table rows, package preservation, images, SVG fallback, links, HTML, raw XML,
metadata, query resolution, aliases, cancellation, resource limits,
concurrent rendering, fuzz seeds, and benchmarks. A generated Arabic fixture is
also opened and rendered with Microsoft Word during local visual QA.

Test fixtures are synthetic and product-neutral. See
[`docs/TESTING.md`](docs/TESTING.md) for package layout and data-isolation
rules, and [`docs/FEATURE_COVERAGE.md`](docs/FEATURE_COVERAGE.md) for the
behavioral coverage matrix.

For Unicode, RTL templates, and locale-sensitive host functions, see
[`docs/INTERNATIONALIZATION.md`](docs/INTERNATIONALIZATION.md).

## Compatibility

Namat aims for workflow compatibility with mature DOCX template engines while
remaining a native Go system. It does not execute arbitrary JavaScript. Complex
JavaScript snippets should become registered Go functions or explicit data
preparation code.

See [`docs/COMPATIBILITY.md`](docs/COMPATIBILITY.md),
[`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md), and
[`docs/PERFORMANCE.md`](docs/PERFORMANCE.md).

## Name

Namat (نَمَط) is the Arabic word for a pattern, mode, or template. It describes
the library directly and remains short in Go imports.

## License

MIT
