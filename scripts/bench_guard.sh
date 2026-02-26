#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
THRESHOLDS_FILE="${ROOT_DIR}/benchmarks/thresholds.tsv"
COMPARE_FILE="${ROOT_DIR}/benchmarks/compare_thresholds.tsv"
OUT_FILE="${ROOT_DIR}/benchmarks/results/bench-ci.out"

for file in "${THRESHOLDS_FILE}" "${COMPARE_FILE}"; do
  if [[ ! -f "${file}" ]]; then
    echo "thresholds file not found: ${file}" >&2
    exit 1
  fi
done

mkdir -p "$(dirname "${OUT_FILE}")"

go test ./benchmarks/router ./benchmarks/middleware ./benchmarks/compare -run='^$' -bench='.' -benchmem -count=1 | tee "${OUT_FILE}"

awk -v thresholds_file="${THRESHOLDS_FILE}" -v compare_file="${COMPARE_FILE}" '
BEGIN {
  load_thresholds(thresholds_file)
  load_thresholds(compare_file)
}

function load_thresholds(file,   line,n,parts,bench) {
  while ((getline line < file) > 0) {
    if (line ~ /^#/ || line ~ /^[[:space:]]*$/) { continue }
    n = split(line, parts, "\t")
    if (n < 3) { continue }
    bench = parts[1]
    max_ns[bench] = parts[2] + 0
    max_allocs[bench] = parts[3] + 0
    seen_threshold[bench] = 1
    if (n >= 5 && parts[4] != "-" && parts[5] != "-") {
      ratio_ref[bench] = parts[4]
      max_ratio[bench] = parts[5] + 0
    }
  }
  close(file)
}

/^Benchmark/ {
  raw = $1
  sub(/-[0-9]+$/, "", raw)
  ns = -1
  allocs = -1
  for (i=1; i<=NF; i++) {
    if ($(i) == "ns/op") { ns = $(i-1) + 0 }
    if ($(i) == "allocs/op") { allocs = $(i-1) + 0 }
  }
  if (ns >= 0) { got_ns[raw] = ns }
  if (allocs >= 0) { got_allocs[raw] = allocs }
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
    if ((bench in ratio_ref) && (ratio_ref[bench] in got_ns)) {
      ratio = got_ns[bench] / got_ns[ratio_ref[bench]]
      if (ratio > max_ratio[bench]) {
        printf("Regression: %s ratio_vs_%s %.3f > %.3f\n", bench, ratio_ref[bench], ratio, max_ratio[bench]) > "/dev/stderr"
        failed = 1
      }
    }
  }
  if (failed) { exit 1 }
  print "Benchmark guard passed"
}
' "${OUT_FILE}"
