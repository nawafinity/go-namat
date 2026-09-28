# Contributing

Changes should remain pure Go and should not add a JavaScript runtime, external
helper executable, or hidden network dependency.

Before opening a change:

```bash
gofmt -w .
go vet ./...
go test -race ./...
go test -run '^$' -bench . -benchmem ./...
```

New template behavior needs focused unit tests and, when layout is affected, a
small synthetic DOCX fixture that can be rendered in Word or LibreOffice.
Fixtures must not contain customer documents, production data, credentials, or
proprietary templates.

Public API additions require Go documentation, an example, error behavior, and
security-limit analysis. Keep commits focused and use conventional commit
subjects where practical.
