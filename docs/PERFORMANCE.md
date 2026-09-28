# Performance

Namat separates compilation from rendering. Applications generating many
reports should compile each template once and reuse the resulting immutable
`Template` across goroutines.

Run the benchmarks on the deployment target:

```bash
go test -run '^$' -bench . -benchmem ./...
```

A dated local baseline is recorded in [`BENCHMARKS.md`](BENCHMARKS.md).

The benchmark fixture renders 100 table rows and reports compilation,
single-threaded rendering, compile-plus-render, and parallel rendering. Commit
benchmark output only when the machine, Go version, CPU policy, and benchmark
duration are recorded; comparisons from unlike machines are misleading.

Performance-sensitive choices in the implementation include:

- expressions are compiled once and cached by source;
- compiled templates are immutable and reusable;
- only `word/*.xml` parts containing the command delimiter are parsed;
- untouched ZIP parts are copied without decoding their contents;
- package output uses configurable DEFLATE compression;
- render loops have a shared aggregate iteration counter;
- rich-content relationships are updated only for their owning Word part.

`RenderTo` remains transactional and therefore assembles the package in memory
before writing. This avoids corrupt partial output and is appropriate for the
bounded report sizes controlled by `MaxOutputBytes`.
