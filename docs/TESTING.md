# Testing and fixture policy

Namat tests validate the standalone library only. They must not encode a
consumer application's schema, report formulas, identifiers, templates, or
production values.

## Layout

Go discovers tests by package and directory. For that reason, Namat keeps test
files beside the package they exercise:

- root `*_test.go` files cover the public package and package-private OOXML
  behavior;
- `internal/expr/*_test.go` covers the native expression engine;
- `cmd/namat/*_test.go` covers CLI-only behavior;
- `testdata/fuzz` contains minimized regression inputs produced by Go fuzzing.

Moving every test into a top-level `tests` directory would create a different
Go package. That package could exercise the exported API, but it could not
directly verify package-private parsers, ZIP guards, or CLI helpers. Keeping
unit tests beside their package preserves focused coverage and follows Go's
test model.

Shared root-package DOCX builders and readers are centralized in
`test_helpers_test.go`. They compile only during tests and are not part of the
published library API or consumer binaries.

## Data policy

- Use only small, generated, synthetic DOCX packages.
- Use neutral names such as `record`, `group`, `item`, and `value`.
- Do not copy customer or product templates into this repository.
- Do not commit workbooks, reports, screenshots, database extracts, or logs
  from consuming applications.
- Keep application migration and parity fixtures in the application
  repository under its own confidentiality controls.
- A CI guard rejects known product identifiers in Go tests and fuzz fixtures.

## Quality commands

```bash
go fmt ./...
go vet ./...
go test ./...
go test -shuffle=on -count=3 ./...
go test -race ./...
go test -run '^$' -bench . -benchmem ./...
```

Fuzz targets should also be run for longer periods before a release.
