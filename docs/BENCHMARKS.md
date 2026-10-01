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

## v1 hardening check

The compatibility and safety changes were measured before and after on the same
machine. These short runs are a regression check rather than a portable
performance promise.

- Date: 2026-09-30
- Go: 1.24.13 windows amd64
- CPU: AMD Ryzen 9 9950X3D
- Duration: 300 ms per benchmark

| Benchmark | Before | After | Change | After bytes/op | After allocs/op |
| --- | ---: | ---: | ---: | ---: | ---: |
| Compile | 22.127 us | 22.346 us | +1.0% | 25,080 | 351 |
| Render compiled template | 757.427 us | 731.037 us | -3.5% | 4,108,086 | 8,546 |
| Compile and render | 836.146 us | 822.511 us | -1.6% | 4,153,858 | 8,901 |
| Parallel render | 203.512 us | 187.676 us | -7.8% | 4,101,590 | 8,542 |

No material throughput regression was observed. Because this is one short run
rather than a multi-sample `benchstat` comparison, small percentage changes
should be treated as noise unless reproduced with longer runs.
