#!/usr/bin/env bash
# run_bench.sh — Run all GoCPU benchmarks and write a timestamped report.
#
# Usage:
#   bash benchmarks/run_bench.sh              # default: 3s per benchmark
#   BENCHTIME=10s bash benchmarks/run_bench.sh
#
# Output is tee'd to both stdout and benchmarks/baseline_<timestamp>.txt.
# To compare two runs:
#   diff benchmarks/baseline_20240101_120000.txt benchmarks/baseline_20240102_093000.txt

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(dirname "$SCRIPT_DIR")"

BENCHTIME="${BENCHTIME:-3s}"
TIMESTAMP="$(date +%Y%m%d_%H%M%S)"
OUTFILE="${SCRIPT_DIR}/baseline_${TIMESTAMP}.txt"

echo "=== GoCPU Benchmark Run ===" | tee "$OUTFILE"
echo "Date      : $(date)" | tee -a "$OUTFILE"
echo "Benchtime : ${BENCHTIME}" | tee -a "$OUTFILE"
go_version="$(go version 2>/dev/null || echo 'unknown')"
echo "Go version: ${go_version}" | tee -a "$OUTFILE"
uname_info="$(uname -srm 2>/dev/null || echo 'unknown')"
echo "Platform  : ${uname_info}" | tee -a "$OUTFILE"
echo "" | tee -a "$OUTFILE"

cd "$REPO_ROOT"

go test \
    -bench=. \
    -benchmem \
    -benchtime="${BENCHTIME}" \
    -count=1 \
    ./pkg/cpu/... \
    ./compiler/... \
    ./pkg/asm/... \
  | tee -a "$OUTFILE"

echo ""
echo "Report written to: ${OUTFILE}"
