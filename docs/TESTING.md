# Testing and fixture policy

Namat tests validate the standalone library only. They must not encode a
consumer application's schema, report formulas, identifiers, templates, or
production values.

## Layout

Go discovers tests by package and directory. For that reason, Namat keeps test
files beside the package they exercise while leaving the module root small:

- root `*_test.go` files cover the public facade, examples, and repository-wide
  quality policies;
- `internal/engine/*_test.go` covers package-private OOXML, rendering, rich
  content, security, and integration behavior;
- `internal/expr/*_test.go` covers the native expression engine;
- `cmd/namat/*_test.go` covers CLI-only behavior;
- `internal/engine/testdata/fuzz` contains any minimized rendering-engine
  regression inputs committed from Go fuzzing. Other fuzz targets may keep
  their initial seeds directly in `*_test.go`.

Moving every test into a top-level `tests` directory would create a different
Go package. That package could exercise the exported API, but it could not
directly verify package-private parsers, ZIP guards, or CLI helpers. The
current layout keeps focused tests beside their implementation without
cluttering the public package root.

Shared engine-test DOCX builders and readers are centralized in
`internal/engine/test_helpers_test.go`. They compile only during tests and are
not part of the published library API or consumer binaries.

## Data policy

- Use only small, generated, synthetic DOCX packages.
- Use neutral names such as `record`, `group`, `item`, and `value`.
- Do not copy customer or product templates into this repository.
- Do not commit workbooks, reports, screenshots, database extracts, or logs
  from consuming applications.
- Keep application migration and parity fixtures in the application
  repository under its own confidentiality controls.
- A CI guard rejects known product identifiers in Go tests and fuzz fixtures.

## Compatibility evidence

The ordinary Go suite validates package structure and rendering behavior; it
does not launch Microsoft Word, LibreOffice, or Google Docs and does not prove
pixel-identical output in those clients. `COMPATIBILITY.md` must distinguish
structural support from versioned client verification.

The pre-v1 visual fixture set remains unfinished. When added, each fixture must:

- use synthetic content and contain no consumer-application data;
- record source provenance and redistribution terms;
- record the client, client version, operating system, and test date;
- state whether comparison is automated or manually reviewed;
- document expected client-specific differences rather than hiding them.

The committed `internal/engine/testdata/compatibility/complex-tables.docx`
fixture currently provides structural package evidence for complex tables and
a LibreOffice-to-PDF smoke test. CI installs LibreOffice Writer and requires
that conversion to produce a non-empty PDF. The test skips only on developer
machines where neither `soffice` nor `libreoffice` is installed. It does not
provide reviewed golden output and does not run Microsoft Word or Google Docs;
therefore it does not complete the visual-compatibility roadmap gate.

## Quality commands

```bash
go fmt ./...
go vet ./...
go test ./...
go test -shuffle=on -count=3 ./...
go test -race ./...
go test -run '^$' -bench . -benchmem ./...
```

CI requires 100% statement coverage independently for the public facade, the
rendering engine, the expression engine, exact value types, the CLI, and both
runnable examples. Generate a combined local profile with:

```bash
go test -coverprofile=coverage-all.out ./...
go tool cover -func=coverage-all.out
```

CI fuzzes all five targets for 10 seconds each. The tag-triggered release
workflow fuzzes every target for 60 seconds; maintainers may run longer local
campaigns when parser, package, or expression code changes materially.

Host callbacks require separate tests when an application uses them. In
particular, test concurrent renders, callback failure, and downstream timeouts.
The render timeout is cooperative and cannot interrupt a callback that does not
return.
