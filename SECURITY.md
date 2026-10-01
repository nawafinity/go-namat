# Security policy

## Reporting a vulnerability

Do not open a public issue for a suspected vulnerability. Use GitHub private
vulnerability reporting for this repository with the affected version or
commit, a minimal synthetic reproduction, and the expected impact. Avoid
including real documents or confidential report data.

Namat is pre-v1 and does not yet have a supported release line. The first
supported-version policy will be published with v1.0. Until then, security
fixes apply to the current development branch and do not imply compatibility
with an earlier commit.

## Template trust model

Namat does not execute JavaScript or implicitly expose filesystem, process,
environment, network, or reflection APIs to templates. A template can access
the values passed to `Render` and the functions explicitly registered in
`Options.Functions`. A Go function value reachable through render data is not
callable by the expression language. Expose a deliberately reviewed function
set through `Options.Functions` instead.

All host callbacks are part of the host application's trust boundary. They must
validate arguments and avoid returning secrets that should not appear in a
report. A compiled `Template` may be rendered by several goroutines at once;
`Functions`, `ErrorHandler`, and any state they capture must
therefore be safe for concurrent use when the template is shared.

`Options.Timeout` and the render context are cooperative bounds. Namat checks
the context between rendering operations, but it cannot preempt a callback
that is blocked or still running. Each registered `FunctionSpec.Call` receives
the render context and should observe it. `ErrorHandler` must return promptly
and enforce its own I/O, CPU, and downstream limits. A callback can therefore
cause a render to exceed `Timeout` before Namat returns a timeout error.

The explicit `@raw-xml` directive is disabled by default. Enabling
`AllowRawXML` is appropriate only when both the template and inserted XML
values are trusted. Ordinary text insertion never interprets embedded XML.

HTML altChunks and SVG images are embedded, not sanitized. Treat their payloads
as trusted document content: reject external resource references and active
content before passing them to Namat when reports may be opened in a client
that resolves or executes such content.

The default hyperlink schemes are `http`, `https`, and `mailto`. Do not add
`file` or application-specific schemes without reviewing the resulting Word
client behavior.

## Resource limits

Keep limits enabled for user-supplied files:

| Option | Default | Scope |
| --- | ---: | --- |
| `MaxTemplateBytes` | 256 MiB | Compressed input template |
| `MaxPartBytes` | 256 MiB | One uncompressed ZIP part |
| `MaxUncompressedBytes` | 1 GiB | Aggregate uncompressed package content |
| `MaxPackageParts` | 10,000 | ZIP entries |
| `MaxOutputBytes` | 512 MiB | Final compressed document |
| `MaxIterations` | 1,000,000 | Aggregate loop iterations in one render |
| `MaxExpressionBytes` | 16 KiB | Source length of one expression |
| `MaxExpressionTokens` | 2,048 | Tokens in one expression |
| `MaxExpressionDepth` | 128 | Parser and AST nesting |
| `MaxEvaluationSteps` | 100,000 | Aggregate expression steps in one render |
| `Timeout` | 30 seconds | Cooperative render deadline |

A zero or negative value selects the default; it does not disable the limit.
The defaults are compatibility ceilings, not recommended upload quotas. Set
smaller values that match the application's expected document sizes and
latency budget.

The ZIP reader rejects traversal names and duplicate package parts. These
controls reduce risk but do not replace normal upload authentication, malware
scanning, storage isolation, and audit logging.
