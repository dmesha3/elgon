#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
THRESHOLDS_FILE="${ROOT_DIR}/benchmarks/thresholds.tsv"
OUT_FILE="${ROOT_DIR}/benchmarks/results/bench-ci.out"

if [[ ! -f "${THRESHOLDS_FILE}" ]]; then
  echo "thresholds file not found: ${THRESHOLDS_FILE}" >&2
  exit 1
fi

mkdir -p "$(dirname "${OUT_FILE}")"

go test ./benchmarks/router ./benchmarks/middleware -run='^$' -bench='.' -benchmem -count=1 | tee "${OUT_FILE}"

awk -v thresholds_file="${THRESHOLDS_FILE}" '
BEGIN {
  while ((getline < thresholds_file) > 0) {
    if ($0 ~ /^#/ || NF < 3) { continue }
    n = split($0, parts, "\t")
    if (n < 3) { continue }
    bench = parts[1]
    max_ns[bench] = parts[2] + 0
    max_allocs[bench] = parts[3] + 0
    seen_threshold[bench] = 1
  }
  close(thresholds_file)
}
/^Benchmark/ {
  raw = $1
  sub(/-[0-9]+$/, "", raw)
  for (i=1; i<=NF; i++) {
    if ($(i) == "ns/op") { ns = $(i-1) + 0 }
    if ($(i) == "allocs/op") { allocs = $(i-1) + 0 }
  }
  got_ns[raw] = ns
  got_allocs[raw] = allocs
}
END {
  failed = 0
  for (bench in seen_threshold) {
    if (!(bench in got_ns)) {
      printf("Missing benchmark result for %s\n", bench) > "/dev/stderr"
      failed = 1
      continue
    }
    if (got_ns[bench] > max_ns[bench]) {
      printf("Regression: %s ns/op %.2f > %.2f\n", bench, got_ns[bench], max_ns[bench]) > "/dev/stderr"
      failed = 1
    }
    if (got_allocs[bench] > max_allocs[bench]) {
      printf("Regression: %s allocs/op %.2f > %.2f\n", bench, got_allocs[bench], max_allocs[bench]) > "/dev/stderr"
      failed = 1
    }
  }
  if (failed) { exit 1 }
  print "Benchmark guard passed"
}
' "${OUT_FILE}"
