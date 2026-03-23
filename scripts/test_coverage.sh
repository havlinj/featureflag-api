#!/usr/bin/env bash
# Runs coverage for production packages (unit + integration) and enforces a minimum threshold.
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$ROOT_DIR"

MIN_COVERAGE=80

# Note: in Go, "./internal/..." includes nested subpackages such as
# "./internal/*/mock" and "./internal/testutil". We keep an explicit include
# list to measure coverage of production packages only.
COVERAGE_PKGS=(
  ./internal/audit
  ./internal/auth
  ./internal/config
  ./internal/db
  ./internal/experiments
  ./internal/flags
  ./internal/users
  ./transport/graphql
  ./transport/graphql/middleware
)

COVERAGE_IMPORTS="$(go list "${COVERAGE_PKGS[@]}" | awk 'BEGIN{ORS=","} {print}' | sed 's/,$//')"

echo "== coverage (unit + integration)"
echo "== measured packages: ${COVERAGE_PKGS[*]}"
go test -tags=integration -covermode=atomic -coverpkg="${COVERAGE_IMPORTS}" -coverprofile=coverage.out \
  "${COVERAGE_PKGS[@]}" ./test/integration/...

FUNC_REPORT_FILE="$(mktemp)"
trap 'rm -f "$FUNC_REPORT_FILE"' EXIT
go tool cover -func=coverage.out > "$FUNC_REPORT_FILE"

echo ""
echo "== coverage summary by package (function-average, quick signal)"
awk '
  $1 ~ /^total:/ {next}
  {
    split($1, a, ":")
    pkg = a[1]
    pct = $3
    gsub("%", "", pct)
    sum[pkg] += pct
    cnt[pkg] += 1
  }
  END {
    for (p in sum) {
      printf "%.1f\t%s\n", sum[p] / cnt[p], p
    }
  }
' "$FUNC_REPORT_FILE" | sort -n | awk '{printf "  %6.1f%%  %s\n", $1, $2}'

echo ""
echo "== top 20 lowest-covered functions"
awk '
  $1 ~ /^total:/ {next}
  {
    pct = $3
    gsub("%", "", pct)
    printf "%.1f\t%s\t%s\n", pct, $1, $2
  }
' "$FUNC_REPORT_FILE" | sort -n | head -n 20 | \
  awk '{printf "  %6.1f%%  %s %s\n", $1, $2, $3}'

COVERAGE_TOTAL=$(awk '/^total:/ {gsub("%","",$3); print $3}' "$FUNC_REPORT_FILE")
echo ""
if awk "BEGIN {exit !($COVERAGE_TOTAL >= $MIN_COVERAGE)}"; then
  echo "Coverage ${COVERAGE_TOTAL}% >= ${MIN_COVERAGE}%"
else
  echo "Coverage ${COVERAGE_TOTAL}% < ${MIN_COVERAGE}%"
  exit 1
fi
