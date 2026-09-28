# Benchmark snapshot

This snapshot is a development baseline, not a cross-machine guarantee.

- Date: 2026-09-28
- Go: 1.24.0 windows amd64
- CPU: Intel Core i9-13900
- Fixture: one heading and a 100-row generated table
- Command: `go test -run '^$' -bench . -benchmem -benchtime=300ms ./internal/engine`

| Benchmark | Time | Bytes per operation | Allocations |
| --- | ---: | ---: | ---: |
| Compile | 27.7 us | 25,019 | 349 |
| Render compiled template | 943.7 us | 4,108,339 | 8,547 |
| Compile and render | 1.028 ms | 4,160,019 | 8,900 |
| Parallel render | 200.8 us | 4,101,205 | 8,543 |

The parallel number is aggregate benchmark throughput divided per operation; it
is not the latency of one isolated report. Re-run the suite on the actual
deployment hardware and retain `benchstat` comparisons when optimizing.
