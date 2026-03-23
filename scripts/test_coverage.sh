#!/usr/bin/env bash
# Runs coverage for production packages (unit + integration) and enforces a minimum threshold.
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$ROOT_DIR"

MIN_COVERAGE=90
ENFORCE_PER_FILE=1

# Per-file coverage floors by file role.
MIN_SERVICE_FILE_COVERAGE=75
MIN_POSTGRES_FILE_COVERAGE=75
MIN_WIRING_FILE_COVERAGE=60
MIN_ENTITY_FILE_COVERAGE=70

# Optional whitelist entries:
#   "path|reason|expires(YYYY-MM-DD)"
# or
#   "path|reason|permanent"
#
# Guidance:
# - Prefer temporary exceptions with a real sunset date.
# - Use "permanent" only for justified, stable architectural exceptions.
# - Always provide a concrete reason (ideally with a ticket/reference).
# - Permanent exceptions are still printed as warnings to keep visibility.
PER_FILE_WHITELIST=(
)

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
FILE_REPORT_FILE="$(mktemp)"
VIOLATIONS_FILE="$(mktemp)"
trap 'rm -f "$FUNC_REPORT_FILE" "$FILE_REPORT_FILE" "$VIOLATIONS_FILE"' EXIT
go tool cover -func=coverage.out > "$FUNC_REPORT_FILE"
awk '
  NR == 1 {next}
  {
    split($0, parts, " ")
    file = parts[1]
    stmts = parts[2] + 0
    count = parts[3] + 0
    total[file] += stmts
    if (count > 0) {
      covered[file] += stmts
    }
  }
  END {
    for (f in total) {
      pct = 0
      if (total[f] > 0) {
        pct = (covered[f] * 100.0) / total[f]
      }
      printf "%.2f\t%s\n", pct, f
    }
  }
' coverage.out | sort -n > "$FILE_REPORT_FILE"

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

echo ""
echo "== per-file coverage floors (core files)"
if [[ "$ENFORCE_PER_FILE" -eq 1 ]]; then
  echo "  service.go >= ${MIN_SERVICE_FILE_COVERAGE}%"
  echo "  postgres.go >= ${MIN_POSTGRES_FILE_COVERAGE}%"
  echo "  wiring files (*resolvers.go,resolver.go,server.go,chain.go) >= ${MIN_WIRING_FILE_COVERAGE}%"
  echo "  entity.go >= ${MIN_ENTITY_FILE_COVERAGE}%"
else
  echo "  per-file enforcement disabled"
fi

while IFS=$'\t' read -r pct file; do
  required=""

  case "$file" in
    */service.go)
      required="$MIN_SERVICE_FILE_COVERAGE"
      ;;
    */postgres.go)
      required="$MIN_POSTGRES_FILE_COVERAGE"
      ;;
    */*resolvers.go|*/resolver.go|*/server.go|*/chain.go)
      required="$MIN_WIRING_FILE_COVERAGE"
      ;;
    */entity.go)
      required="$MIN_ENTITY_FILE_COVERAGE"
      ;;
  esac

  if [[ -z "$required" ]]; then
    continue
  fi

  whitelisted_reason=""
  whitelisted_expires=""
  for entry in "${PER_FILE_WHITELIST[@]}"; do
    wl_file="${entry%%|*}"
    rest="${entry#*|}"
    wl_reason="${rest%%|*}"
    wl_expires="${rest##*|}"
    if [[ "$wl_file" == "$file" ]]; then
      whitelisted_reason="$wl_reason"
      whitelisted_expires="$wl_expires"
      break
    fi
  done

  if awk "BEGIN {exit !($pct + 0 < $required + 0)}"; then
    if [[ -n "$whitelisted_reason" ]]; then
      printf "  WARN whitelist: %.2f%% < %s%%  %s  (reason: %s, expires: %s)\n" \
        "$pct" "$required" "$file" "$whitelisted_reason" "$whitelisted_expires"
      if [[ "$whitelisted_expires" != "permanent" && "$whitelisted_expires" < "$(date +%F)" ]]; then
        printf "whitelist expired: %s (%s)\n" "$file" "$whitelisted_expires" >> "$VIOLATIONS_FILE"
      fi
    else
      printf "  FAIL: %.2f%% < %s%%  %s\n" "$pct" "$required" "$file"
      printf "%s\t%s\t%s\n" "$pct" "$required" "$file" >> "$VIOLATIONS_FILE"
    fi
  fi
done < "$FILE_REPORT_FILE"

COVERAGE_TOTAL=$(awk '/^total:/ {gsub("%","",$3); print $3}' "$FUNC_REPORT_FILE")
echo ""
if [[ -s "$VIOLATIONS_FILE" ]]; then
  echo "Per-file coverage floor violations detected."
  exit 1
fi

if awk "BEGIN {exit !($COVERAGE_TOTAL + 0 >= $MIN_COVERAGE + 0)}"; then
  echo "Coverage ${COVERAGE_TOTAL}% >= ${MIN_COVERAGE}%"
else
  echo "Coverage ${COVERAGE_TOTAL}% < ${MIN_COVERAGE}%"
  exit 1
fi
