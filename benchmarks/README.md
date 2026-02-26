# Benchmarks

This suite tracks hot paths and throughput characteristics for `elgon`.

## Structure

- `router/`: static, param, wildcard route matching overhead
- `middleware/`: middleware chain overhead
- `json/`: JSON encode/decode throughput
- `e2e/`: end-to-end API stack benchmark

## Run

Quick CI subset:

```bash
make bench-ci
```

Full benchmark run:

```bash
make bench
```

Direct Go command:

```bash
go test ./benchmarks/... -run=^$ -bench=. -benchmem -count=3
```

## Interpreting results

Track at least these metrics over time:

- `ns/op`
- `B/op`
- `allocs/op`

Treat increasing `allocs/op` on router and middleware benchmarks as a regression signal.

## CI Regression Gates

- Absolute thresholds:
  - `benchmarks/thresholds.tsv`
- Compare benchmark thresholds:
  - `benchmarks/compare_thresholds.tsv`
- Guard script:
  - `scripts/bench_guard.sh`
