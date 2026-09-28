# Tiraz

Tiraz is a pure Go template engine for Microsoft Word DOCX files. It renders
data into documents while preserving the original Word package, styles,
relationships, media, headers, and footers.

The project does not embed Node.js or a JavaScript runtime. Template
expressions are parsed and evaluated by Tiraz itself.

> Status: pre-v1 development. Text insertion, assignments, conditions, loops,
> split Word runs, and package round trips are implemented. The public API may
> change before v1.0.

## Goals

- Pure Go with no runtime executables.
- Safe, deterministic expression evaluation.
- Commands split across Word runs.
- Conditions and loops over paragraphs and table rows.
- Dynamic images, links, HTML, and literal OOXML.
- Headers, footers, footnotes, endnotes, and DOCM round trips.
- Bounded execution with useful source errors.
- Compatibility tests against established DOCX template engines.

## Template syntax

Commands use `[[` and `]]` by default.

```text
Hello [[INS customer.name]]

[[IF order.total > 0]]
Total: [[INS money(order.total)]]
[[ELSE]]
No balance is due.
[[END-IF]]

[[FOR item IN order.items]]
[[INS $idx + 1]]. [[INS $item.name]]
[[END-FOR item]]
```

`INS` may be omitted for concise value insertion:

```text
[[customer.name]]
```

Variables introduced by loops use a `$` prefix. Expressions support property
access, indexes, function calls, arithmetic, comparisons, logical operators,
and null coalescing.

## Go API

```go
template, err := tiraz.Compile(templateBytes, tiraz.Options{})
if err != nil {
    return err
}

report, err := template.Render(ctx, data)
```

Host functions can expose application-specific formatting and lookup logic
without embedding a scripting runtime:

```go
options := tiraz.Options{
    Functions: map[string]tiraz.Function{
        "money": func(args ...any) (any, error) {
            return formatMoney(args[0]), nil
        },
    },
}
```

## Local inspection

The optional CLI validates templates without printing their expressions by
default:

```text
go run ./cmd/tiraz inspect template.docx
```

Pass `--details` only when template expressions are safe to display.

## Current scope

Implemented:

- DOCX ZIP/OOXML preservation using only the Go standard library.
- Commands split across runs and paragraphs by Microsoft Word.
- `INS`, `EXEC`, `SET`, `IF`/`ELSE`, and `FOR` over paragraphs and table rows.
- Headers, footers, footnotes, endnotes, and other templated `word/*.xml` parts.
- Custom delimiters, host functions, null handling, timeouts, and loop limits.
- Immutable compiled templates safe for concurrent rendering.

In progress before v1:

- Images, hyperlinks, HTML altChunk, and literal OOXML.
- Query resolvers and aliases.
- Metadata helpers and line-break compatibility options.
- A public compatibility and golden-document suite.

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) and
[docs/ROADMAP.md](docs/ROADMAP.md).

## Name

Tiraz (طِراز) is an Arabic word associated with decorated textile, style, and
craftsmanship. The name reflects weaving structured data into a finished
document.

## License

MIT
