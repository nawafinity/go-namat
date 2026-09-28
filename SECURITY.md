# Security policy

## Reporting a vulnerability

Do not open a public issue for a suspected vulnerability. Contact the project
maintainer privately with the affected version, a minimal reproduction, and the
expected impact. Avoid including real documents or confidential report data.

## Template trust model

Namat does not execute JavaScript or expose Go reflection, filesystem, process,
environment, or network APIs to templates. A template can access only the data
passed to `Render` and the functions explicitly registered in `Options`.

Registered functions are part of the host application's trust boundary. They
must validate arguments, honor context deadlines where relevant, and avoid
returning secrets that should not appear in a report.

Literal OOXML is disabled by default. Enabling `AllowRawXML` is appropriate only
when both the template and inserted XML values are trusted.

The default hyperlink schemes are `http`, `https`, and `mailto`. Do not add
`file` or application-specific schemes without reviewing the resulting Word
client behavior.

## Resource limits

Keep limits enabled for user-supplied files:

- `MaxTemplateBytes`
- `MaxPartBytes`
- `MaxUncompressedBytes`
- `MaxPackageParts`
- `MaxOutputBytes`
- `MaxIterations`
- `Timeout`

The ZIP reader rejects traversal names and duplicate package parts. These
controls reduce risk but do not replace normal upload authentication, malware
scanning, storage isolation, and audit logging.
