# Contributing

By participating in Namat, you agree to follow the
[Code of Conduct](CODE_OF_CONDUCT.md).

Changes should remain pure Go and should not add a JavaScript runtime, external
helper executable, or hidden network dependency.

Before opening a change:

```bash
go fmt ./...
go vet ./...
go test -race ./...
go test -shuffle=on -count=3 ./...
go test -run '^$' -bench . -benchmem ./...
```

New template behavior needs focused unit tests and, when layout is affected, a
small synthetic DOCX fixture that can be rendered in Word or LibreOffice.
Fixtures must not contain customer documents, production data, credentials, or
proprietary templates.

All committed fixtures and test values must be synthetic. Product-specific
schemas, field names, identifiers, and migration fixtures belong in the
product repository, never in Namat.

Public API additions require Go documentation, an example, error behavior, and
security-limit analysis. Keep commits focused and use conventional commit
subjects where practical.

See [docs/TESTING.md](docs/TESTING.md) for the test layout and privacy rules.
