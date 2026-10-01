<div align="center">

<h1>
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="assets/brand/namat-logo-dark.svg">
    <source media="(prefers-color-scheme: light)" srcset="assets/brand/namat-logo-light.svg">
    <img src="assets/brand/namat-logo-light.svg" width="360" alt="Namat · نَمَط">
  </picture>
</h1>

**A native Go template engine for Microsoft Word**

Author reports in Word. Render them with Go. Ship one native application.

[![Go](https://img.shields.io/badge/Go-1.23%2B-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Go Reference](https://pkg.go.dev/badge/github.com/nawafinity/go-namat.svg)](https://pkg.go.dev/github.com/nawafinity/go-namat)
[![CI](https://github.com/nawafinity/go-namat/actions/workflows/ci.yml/badge.svg)](https://github.com/nawafinity/go-namat/actions/workflows/ci.yml)
[![Coverage](https://img.shields.io/badge/statement%20coverage-gated-brightgreen)](docs/TESTING.md)
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

<a id="why-namat" name="why-namat"></a>

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

<a id="quick-start" name="quick-start"></a>

## 🚀 Quick start

### Install

```bash
go get github.com/nawafinity/go-namat
```

### Author a Word template

Create a DOCX or DOCM file in Word and place commands directly in the document:

```text
Invoice for [[customer.name]]

[[#if invoice.total > 0]]
Total: [[invoice.total]]
[[#else]]
No balance is due.
[[/if]]

[[#each invoice.items as item]]
[[loop.number]]. [[item.name]]
[[/each]]
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
For a stress test, [`examples/advanced`](examples/advanced) nests task tables
inside repeated project rows and calls registered functions from both loop
levels, including functions that return links and generated charts.

<a id="template-language" name="template-language"></a>

## 🧩 Template language

Commands use `[[` and `]]` by default. Both delimiters are configurable through
`Options`. The only supported contract is `v1`; set
`LanguageVersion: namat.LanguageVersionV1` when an application wants to make
that choice explicit. Namat never auto-detects or falls back to a legacy syntax.

| Command | Purpose |
| --- | --- |
| `[[expression]]` | Insert a scalar value |
| `[[#let name = expression]]` | Declare an immutable lexical value without output |
| `[[#if expression]]` / `[[#else]]` / `[[/if]]` | Render a conditional paragraph or table-row block |
| `[[#each values as item]]` / `[[/each]]` | Repeat paragraphs or complete table rows |
| `[[@image expression]]` | Insert an inline image |
| `[[@link expression]]` | Insert an external hyperlink |
| `[[@html expression]]` | Insert a Word HTML altChunk |
| `[[@raw-xml expression]]` | Insert trusted OOXML when explicitly enabled |

Structural directives and `#let` must occupy their own paragraph or table row.
`@html` and `@raw-xml` must occupy their own paragraph.

Word may split a command across several XML runs, and occasionally across
adjacent paragraphs. Namat normalizes supported fragmented commands before
compilation.

### Expressions

The native expression language is designed for data access and bounded
calculations—not arbitrary code execution. It supports:

- maps, structs, exact JSON field names, pointers, arrays, slices, strings, and indexes;
- property access and optional access such as `customer?.name`;
- strict arithmetic, comparison, equality, logical, and unary operators;
- array and object literals: `[1, 2]`, `{ url: url, label: name }`;
- pipelines/filters such as `name | trim | upper` and
  `customer?.name | default("—")`;
- exact signed/unsigned integers and `namat.Decimal` values;
- explicitly registered, typed Go functions.

Loop variables have ordinary names. Loop metadata is available through
`loop.index`, `loop.number`, `loop.first`, `loop.last`, and `loop.parent`.
Conditions accept `bool` only. Missing fields, null insertion, mixed-type `+`,
and incompatible comparisons are errors unless handled explicitly. Function
values placed in render data are not callable; register intended functions
with a `FunctionSpec` signature through `Options.Functions`.

## 🖼️ Rich content

### Images

`@image` accepts `namat.Image` or an object with equivalent fields. Namat
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

- `@link` accepts `namat.Link` or an object expression. Only `http`, `https`,
  and `mailto` are allowed by default.
- `@html` uses OOXML `altChunk`. Microsoft Word supports it; LibreOffice and
  Google Docs may import it inconsistently. HTML and SVG payloads are embedded,
  not sanitized; use trusted or application-sanitized content.
- `@raw-xml` is disabled by default because it bypasses escaping. Enable it only
  when both the template and inserted values are trusted.

```text
[[@link ({ url: project.url, label: project.name })]]
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

### Exact JSON numbers

Use `namat.DecodeJSON` when report data comes from JSON. It preserves `int64`,
`uint64`, and exact decimal values instead of converting every number to
`float64`.

## 💻 Command-line interface

The optional CLI is built on the same public package:

```text
namat inspect template.docx
namat inspect --json template.docx
namat lint --data data.json template.docx
namat metadata document.docx
namat render --data data.json --out report.docx template.docx
```

`render` refuses to overwrite an existing file unless `--force` is explicit.
It writes and syncs a temporary file in the destination directory, then uses an
atomic no-replace hard link so a destination created concurrently is never
overwritten. With `--force`, it first attempts the platform's atomic replacement
rename. On platforms that reject that operation, it uses a recoverable sibling
backup while installing the new file; that fallback is not a single atomic
replacement operation.

```bash
go build -trimpath -ldflags="-s -w" ./cmd/namat
```

## 🛡️ Safety model

Templates can access the data supplied to `Render` and Go functions explicitly
registered by the host application. Function values in render data are not
callable. Templates receive no implicit access to the filesystem, processes,
environment variables, reflection APIs, or the network. For untrusted
templates, expose only deliberately reviewed functions.

Compiled templates are immutable and safe for concurrent rendering. That
guarantee does not make host callbacks or captured application state safe:
`Functions` and `ErrorHandler` must support the concurrency
with which the application uses them.

`Options` provides limits for:

- compressed template size;
- individual ZIP-part size;
- total uncompressed package size;
- package-part count;
- final output size;
- aggregate loop iterations;
- expression bytes, tokens, AST depth, and aggregate evaluation steps;
- rendering duration.

The render context and `Timeout` are cooperative. Namat checks them between
rendering operations but cannot preempt a blocked host callback. Registered
functions receive the render context; functions and error handlers must return promptly
and enforce their own downstream limits. See the [security policy](SECURITY.md)
for defaults and the complete trust model.

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
| Explicit raw OOXML | Opt-in | `@raw-xml` with trusted input only |
| Arbitrary JavaScript | Not supported | Replace with registered Go functions |
| Floating images | Not generated | Inline drawings are more portable |

See the [full compatibility matrix](docs/COMPATIBILITY.md) for exact behavior
and limitations.

## 📊 Quality and performance

| Quality gate | Current guarantee |
| --- | --- |
| Statement coverage | Per-package minimums enforced in CI; see `docs/TESTING.md` |
| Behavioral coverage | Verified core capabilities map to automated evidence; client-rendered visual compatibility remains a pre-v1 gate |
| Platforms | CI runs on Linux, Windows, and macOS with Go 1.23 and the current Go release line |
| Concurrency | Race detector and concurrent-rendering tests |
| Robustness | CI fuzzes commands, ZIP packages, reports, and expressions; LibreOffice must convert the public fixture |
| Test data | Generated, synthetic, English, product-neutral, and explicitly licensed public fixtures |

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
├── examples/advanced/  nested tables and functions inside loops
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

<a id="documentation" name="documentation"></a>

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
| [Migration](docs/MIGRATION.md) | Draft pre-v1-to-v1 contract and consumer checklist |
| [Releasing](docs/RELEASING.md) | Release gates, client verification, and artifact signing |
| [Security](SECURITY.md) | Trust model and vulnerability reporting |
| [Code of Conduct](CODE_OF_CONDUCT.md) | Community standards and reporting process |
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

Read [CONTRIBUTING.md](CONTRIBUTING.md) and the
[Code of Conduct](CODE_OF_CONDUCT.md) before submitting a change.

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
