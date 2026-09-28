<div align="center">

<h1>
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="assets/brand/namat-logo-dark.png">
    <source media="(prefers-color-scheme: light)" srcset="assets/brand/namat-logo-light.png">
    <img src="assets/brand/namat-logo-light.png" width="360" alt="Namat · نَمَط">
  </picture>
</h1>

**A native Go template engine for Microsoft Word**

Author reports in Word. Render them with Go. Ship one native application.

[![Go](https://img.shields.io/badge/Go-1.23%2B-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Go Reference](https://pkg.go.dev/badge/github.com/nawafinity/go-namat.svg)](https://pkg.go.dev/github.com/nawafinity/go-namat)
[![CI](https://github.com/nawafinity/go-namat/actions/workflows/ci.yml/badge.svg)](https://github.com/nawafinity/go-namat/actions/workflows/ci.yml)
[![Coverage](https://img.shields.io/badge/statement%20coverage-100%25-brightgreen)](docs/TESTING.md)
[![License](https://img.shields.io/badge/license-MIT-2ea44f)](LICENSE)

**English** · [العربية](README_AR.md)

[Why Namat?](#why-namat) · [Quick start](#quick-start) · [Template language](#template-language) · [Documentation](#documentation)

</div>

Namat turns DOCX and DOCM templates into data-driven reports while preserving
the document structure created in Word: styles, media, headers, footers,
macros, and untouched OOXML parts.

Written entirely in Go with zero third-party module dependencies, Namat embeds
cleanly into services, command-line tools, and desktop applications.

> [!IMPORTANT]
> Namat is currently **pre-v1**. The core API and template language are usable
> and thoroughly tested. Semantic-version compatibility guarantees begin with
> `v1.0`.

## ✨ Why Namat?

<table>
  <tr>
    <td width="50%">
      <img src="assets/icons/native.svg" width="30" alt=""><br>
      <strong>Pure Go</strong><br>
      Embed one focused library and ship one native application binary.
    </td>
    <td width="50%">
      <img src="assets/icons/word-first.svg" width="30" alt=""><br>
      <strong>Word-first authoring</strong><br>
      Create layouts and styles in Word instead of rebuilding them in code.
    </td>
  </tr>
  <tr>
    <td width="50%">
      <img src="assets/icons/safe.svg" width="30" alt=""><br>
      <strong>Safe by design</strong><br>
      Use bounded expressions, typed errors, timeouts, and explicit resource limits.
    </td>
    <td width="50%">
      <img src="assets/icons/structure.svg" width="30" alt=""><br>
      <strong>Structure-aware rendering</strong><br>
      Apply nested conditions and loops to paragraphs and complete table rows.
    </td>
  </tr>
  <tr>
    <td width="50%">
      <img src="assets/icons/rich-content.svg" width="30" alt=""><br>
      <strong>Rich content</strong><br>
      Generate text, images, links, HTML altChunks, line breaks, and trusted OOXML.
    </td>
    <td width="50%">
      <img src="assets/icons/concurrent.svg" width="30" alt=""><br>
      <strong>Concurrent reuse</strong><br>
      Compile once and render safely from multiple goroutines.
    </td>
  </tr>
</table>

## 🚀 Quick start

### Install

```bash
go get github.com/nawafinity/go-namat
```

### Author a Word template

Create a DOCX or DOCM file in Word and place commands directly in the document:

```text
Invoice for [[customer.name]]

[[IF invoice.total > 0]]
Total: [[invoice.total]]
[[ELSE]]
No balance is due.
[[END-IF]]

[[FOR item IN invoice.items]]
[[$idx + 1]]. [[$item.name]]
[[END-FOR item]]
```

### Compile once, render many

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

	tmpl, err := namat.Compile(source, namat.Options{})
	if err != nil {
		panic(err)
	}

	report, err := tmpl.Render(context.Background(), map[string]any{
		"customer": map[string]any{"name": "Acme"},
		"invoice": map[string]any{
			"total": 250.00,
			"items": []map[string]any{
				{"name": "Assessment"},
				{"name": "Implementation"},
			},
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

See [`examples/complete`](examples/complete) for a self-contained Arabic
report containing a table, image, hyperlink, conditions, and loops.

## 🧩 Template language

Commands use `[[` and `]]` by default. Both delimiters are configurable through
`Options`.

| Command | Purpose |
| --- | --- |
| `[[value]]`, `[[INS value]]`, `[[= value]]` | Insert text |
| `[[EXEC name = expression]]`, `[[! name = expression]]` | Assign a local value without output |
| `[[SET name = expression]]` | Explicit assignment form |
| `[[IF expression]]` / `[[ELSE]]` / `[[END-IF]]` | Render a conditional paragraph or table-row block |
| `[[FOR item IN values]]` / `[[END-FOR item]]` | Repeat paragraphs or complete table rows |
| `[[IMAGE expression]]` | Insert an inline image |
| `[[LINK expression]]` | Insert an external hyperlink |
| `[[HTML expression]]` | Insert a Word HTML altChunk |
| `[[RAW-XML expression]]` | Insert trusted OOXML when explicitly enabled |
| `[[QUERY query text]]` | Resolve report data through an application callback |
| `[[ALIAS name INS expression]]`, `[[*name]]` | Define and reuse a complete command |

Structural `IF` and `FOR` markers should occupy their own paragraph or table
row. `HTML` and `RAW-XML` must occupy their own paragraph.

Word may split a command across several XML runs, and occasionally across
adjacent paragraphs. Namat normalizes supported fragmented commands before
compilation.

### Expressions

The native expression language is designed for data access and bounded
calculations—not arbitrary code execution. It supports:

- maps, structs, JSON field names, pointers, arrays, slices, strings, and
  indexes;
- property access, optional chaining, and null coalescing with `??`;
- arithmetic, comparison, equality, logical, and unary operators;
- ternary expressions: `condition ? yes : no`;
- array and object literals: `[1, 2]`, `{ url: url, label: name }`;
- template strings: `` `Score: ${score}` ``;
- string helpers such as `.slice()`, `.trim()`, `.toUpperCase()`,
  `.contains()`, and `.startsWith()`;
- collection helpers such as `.join()`, `.includes()`, and `.length`;
- explicitly registered Go functions.

Loop variables use the `$` prefix, and `$idx` is zero-based. Registered Go
functions are the native replacement for JavaScript helpers: they can be unit
tested, profiled, and audited like ordinary Go code.

## 🖼️ Rich content

### Images

`IMAGE` accepts `namat.Image` or an object with equivalent fields. Namat
supports PNG, JPEG, GIF, and SVG as inline drawings, with dimensions in
centimeters, rotation, alt text, and optional captions.

```go
namat.Image{
	Data:      pngBytes,
	Extension: "png",
	Width:     8,
	Height:    4,
	Alt:       "Quarterly chart",
}
```

SVG images may include a PNG, JPEG, or GIF thumbnail for older Word versions
and document previews.

### Hyperlinks, HTML, and OOXML

- `LINK` accepts `namat.Link` or an object expression. Only `http`, `https`,
  and `mailto` are allowed by default.
- `HTML` uses OOXML `altChunk`. Microsoft Word supports it; LibreOffice and
  Google Docs may import it inconsistently.
- `RAW-XML` is disabled by default because it bypasses escaping. Enable it only
  when both the template and inserted values are trusted.

```text
[[LINK ({ url: project.url, label: project.name })]]
```

```go
options := namat.Options{AllowRawXML: true}
```

## 🧰 Library API

### Readers and writers

```go
tmpl, err := namat.CompileReader(reader, options)
err = tmpl.RenderTo(ctx, writer, data)
```

`RenderTo` is transactional from the writer's perspective: Namat assembles the
DOCX package in memory before writing, so a rendering failure does not leave a
partial document.

### Inspection and metadata

```go
commands, err := namat.ListCommands(templateBytes, options)
metadata, err := namat.GetMetadata(documentBytes)
```

`GetMetadata` returns cached Word properties. It does not claim to recalculate
layout-dependent page or word counts. The CLI inspector hides command
expressions by default to reduce accidental data exposure.

### Query resolver

A template may declare one `QUERY`. Namat passes its contents unchanged to an
application-owned callback before rendering.

```go
options := namat.Options{
	QueryResolver: func(ctx context.Context, query string) (any, error) {
		return database.ReportData(ctx, query)
	},
}
```

Namat does not interpret SQL, GraphQL, or another query language, and it never
opens a network connection itself.

## 💻 Command-line interface

The optional CLI is built on the same public package:

```text
namat inspect template.docx
namat inspect --json template.docx
namat metadata document.docx
namat render --data data.json --out report.docx template.docx
```

`render` refuses to overwrite an existing file unless `--force` is explicit.
It writes through an atomic temporary-file rename.

```bash
go build -trimpath -ldflags="-s -w" ./cmd/namat
```

## 🛡️ Safety model

Templates can access only the data supplied to `Render` and the Go functions
explicitly registered by the host application. They receive no implicit access
to the filesystem, processes, environment variables, reflection APIs, or the
network.

`Options` provides limits for:

- compressed template size;
- individual ZIP-part size;
- total uncompressed package size;
- package-part count;
- final output size;
- aggregate loop iterations;
- rendering duration.

The package reader rejects traversal paths, duplicate entries, oversized
parts, and invalid packages. Read the [security policy](SECURITY.md) before
accepting templates from untrusted users.

## 🔄 Compatibility

| Capability | Status | Notes |
| --- | --- | --- |
| DOCX round trip | Supported | Preserves untouched parts and media |
| DOCM round trip | Supported | Copies VBA parts unchanged |
| Commands split across Word runs | Supported | Includes supported adjacent-paragraph splits |
| Nested conditions and loops | Supported | Paragraphs and complete table rows |
| Headers, footers, and notes | Supported | Processes relevant `word/*.xml` parts |
| PNG, JPEG, GIF, and SVG | Supported | Inline drawings with optional SVG fallback |
| External hyperlinks | Supported | Scheme allowlist enforced |
| HTML altChunk | Supported | Main document only |
| Literal OOXML | Opt-in | Trusted input only |
| Arbitrary JavaScript | Not supported | Replace with registered Go functions |
| Floating images | Not generated | Inline drawings are more portable |

See the [full compatibility matrix](docs/COMPATIBILITY.md) for exact behavior
and limitations.

## 📊 Quality and performance

| Quality gate | Current guarantee |
| --- | --- |
| Statement coverage | **100%** independently for the public facade, rendering engine, expression engine, CLI, and complete example |
| Behavioral coverage | Every documented core capability maps to an automated test |
| Platforms | CI runs on Linux, Windows, and macOS |
| Concurrency | Race detector and concurrent-rendering tests |
| Robustness | Fuzz targets for commands, ZIP packages, reports, and expressions |
| Test data | Generated, synthetic, English, and product-neutral fixtures |

```bash
go test ./...
go test -race ./...
go test -shuffle=on -count=3 ./...
go vet ./...
go test -run '^$' -bench . -benchmem ./...
```

For high-throughput workloads, compile each template once and reuse the
immutable `Template` across goroutines. See the [performance guide](docs/PERFORMANCE.md)
and the dated [benchmark baseline](docs/BENCHMARKS.md).

## 🏗️ Repository structure

```text
go-namat/
├── assets/             project mark and feature icons
├── cmd/namat/          optional native CLI
├── docs/               architecture, compatibility, testing, and performance
├── examples/complete/  complete synthetic report example
├── internal/engine/    private DOCX compiler, renderer, and focused tests
├── internal/expr/      native expression lexer, parser, and evaluator
├── namat.go             stable public package facade
├── README_AR.md         complete Arabic documentation
├── types.go             public options, values, commands, and errors
└── *_test.go            public API examples and repository policy checks
```

The `internal` boundary prevents consumers from depending on implementation
details while keeping the public import path concise:

```go
import "github.com/nawafinity/go-namat"
```

<a id="documentation"></a>

## 📚 Documentation

| Document | Contents |
| --- | --- |
| [Architecture](docs/ARCHITECTURE.md) | Rendering pipeline and component boundaries |
| [Compatibility](docs/COMPATIBILITY.md) | Supported behavior and known limitations |
| [Core feature coverage](docs/FEATURE_COVERAGE.md) | Automated evidence for every core capability |
| [Testing](docs/TESTING.md) | Test layout, quality commands, and data-isolation policy |
| [Internationalization](docs/INTERNATIONALIZATION.md) | Unicode, RTL, and locale responsibilities |
| [Performance](docs/PERFORMANCE.md) | Performance model and benchmarking guidance |
| [Roadmap](docs/ROADMAP.md) | Remaining work toward v1.0 |
| [Security](SECURITY.md) | Trust model and vulnerability reporting |
| [Changelog](CHANGELOG.md) | Notable project changes |

## 🗺️ Roadmap

The native text engine, rich-content support, authoring compatibility, and
production-hardening foundations are complete. Remaining pre-v1 work focuses
on public visual-compatibility fixtures, a semantic-versioned migration guide,
and signed CLI releases.

Product-specific integrations and migrations intentionally live outside this
standalone repository. See the detailed [roadmap](docs/ROADMAP.md).

## 🤝 Contributing

Contributions are welcome when they preserve the focused Go architecture,
explicit dependency policy, and stable public surface. Tests must use synthetic
documents and values only—never customer templates, production data,
credentials, or confidential material.

Read [CONTRIBUTING.md](CONTRIBUTING.md) before submitting a change.

## 💡 Inspiration

Namat is inspired by the natural, Word-first authoring model of
[docx-templates](https://github.com/guigrpa/docx-templates). It reimagines that
workflow around Go's type system, concurrency model, and deployment strengths,
with its own expression engine, OOXML pipeline, API, and safety controls.

## 🏷️ The name

**Namat (نَمَط)** is the Arabic word for a pattern, mode, or template. It
describes the library directly and remains short in Go imports.

The mark pairs a folded document with interlocking geometric forms: a visual
shorthand for structured templates becoming finished reports.

## 📄 License

Namat is available under the [MIT License](LICENSE).

<div align="center">

**Author in Word. Render with Go.**

</div>
