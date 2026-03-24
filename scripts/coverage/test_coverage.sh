#!/usr/bin/env bash
# Runs coverage for production packages (unit + integration) and enforces a minimum threshold.
set -euo pipefail
export LC_ALL=C
if [[ -x "/usr/local/go/bin/go" ]]; then
  export PATH="/usr/local/go/bin:$PATH"
fi
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
cd "$ROOT_DIR"

# Risk-based baseline:
# - keep strong signal on critical paths
# - avoid inflating low-value tests only to chase a vanity percentage
MIN_COVERAGE=80
ENFORCE_PER_FILE=1
ENFORCE_FUNCTION_FLOOR=1

# Deterministic coverage by default:
# - clear Go test cache
# - disable test result cache via -count=1
# Set COVERAGE_ALLOW_CACHE=1 for faster local feedback loops.
COVERAGE_ALLOW_CACHE="${COVERAGE_ALLOW_CACHE:-0}"
RUN_META_FILE="scripts/coverage/state/coverage_run_meta.json"

# Per-file coverage floors by file role.
MIN_ANY_FILE_COVERAGE=40
MIN_SERVICE_FILE_COVERAGE=85
MIN_POSTGRES_FILE_COVERAGE=85
MIN_WIRING_FILE_COVERAGE=70
MIN_ENTITY_FILE_COVERAGE=75

# Function-level floor for core files to avoid low-covered functions hidden by file averages.
MIN_CORE_FUNCTION_COVERAGE=50

# Auto-filter function-level gate violations (see scripts/coverage/coverage_filter/):
# - gqlgen output: any *.go under graph/ (also skipped in write_function_violations_file)
# - thin delegates: single return that forwards a direct call with identifier-only args
AUTO_FILTER_THIN_DELEGATES=1

# Single source of truth for generated GraphQL Go files that must be excluded from enforcement.
GENERATED_GRAPH_GO_RE='/graph/.*[.]go:?$'

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

GLOBAL_FAIL=0
PER_FILE_FAIL=0
TEST_EXEC_FAIL=0
GO_TEST_EXIT=0

FUNC_REPORT_FILE=""
FILE_REPORT_FILE=""
PER_FILE_VIOLATIONS_FILE=""
FUNCTION_VIOLATIONS_FILE=""

coverage_imports_csv() {
  go list "${COVERAGE_PKGS[@]}" | awk 'BEGIN{ORS=","} {print}' | sed 's/,$//'
}

# Must not wrap go test in command substitution: test output goes to stdout and must reach the terminal.
run_tests_with_coverage() {
  local imports="$1"
  local go_test_extra_args=()

  if [[ "$COVERAGE_ALLOW_CACHE" == "1" ]]; then
    echo "== coverage mode: cached (fast local feedback)"
  else
    echo "== coverage mode: fresh (deterministic)"
    go clean -testcache
    go_test_extra_args+=(-count=1)
  fi

  set +e
  go test "${go_test_extra_args[@]}" -tags=integration -covermode=atomic -coverpkg="${imports}" -coverprofile=coverage.out \
    "${COVERAGE_PKGS[@]}" ./test/integration/...
  GO_TEST_EXIT=$?
  set -e
}

write_run_metadata() {
  local cache_mode="fresh"
  local coverage_sha256=""
  local go_version=""
  local generated_at=""
  local measured_packages=""
  local coverage_cmd=""

  if [[ "$COVERAGE_ALLOW_CACHE" == "1" ]]; then
    cache_mode="cached"
  fi

  if [[ -f coverage.out ]]; then
    coverage_sha256="$(sha256sum coverage.out | awk '{print $1}')"
  fi

  go_version="$(go version 2>/dev/null || true)"
  generated_at="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
  measured_packages="$(printf "%s " "${COVERAGE_PKGS[@]}" ./test/integration/... | sed 's/[[:space:]]*$//')"
  coverage_cmd="go test ${COVERAGE_PKGS[*]} ./test/integration/... -coverprofile=coverage.out"

  mkdir -p "$(dirname "$RUN_META_FILE")"
  cat > "$RUN_META_FILE" <<EOF
{
  "generated_at": "$generated_at",
  "cache_mode": "$cache_mode",
  "go_version": "$go_version",
  "go_test_exit": $GO_TEST_EXIT,
  "coverage_profile": "coverage.out",
  "coverage_profile_sha256": "$coverage_sha256",
  "command": "$coverage_cmd",
  "measured_packages": "$measured_packages",
  "includes_go_integration_tests": true,
  "includes_bash_integration_tests": false
}
EOF
}

setup_temp_reports() {
  FUNC_REPORT_FILE="$(mktemp)"
  FILE_REPORT_FILE="$(mktemp)"
  PER_FILE_VIOLATIONS_FILE="$(mktemp)"
  FUNCTION_VIOLATIONS_FILE="$(mktemp)"
  trap 'rm -f "$FUNC_REPORT_FILE" "$FILE_REPORT_FILE" "$PER_FILE_VIOLATIONS_FILE" "$FUNCTION_VIOLATIONS_FILE"' EXIT
}

build_file_level_report() {
  go tool cover -func=coverage.out > "$FUNC_REPORT_FILE"
  # Keep per-file gate consistent with the package summary source:
  # aggregate function percentages from `go tool cover -func`.
  awk '
  $1 ~ /^total:/ {next}
  {
    split($1, a, ":")
    file = a[1]
    pct = $3
    gsub("%", "", pct)
    sum[file] += pct
    cnt[file] += 1
  }
  END {
    for (f in sum) {
      printf "%.2f\t%s\n", sum[f] / cnt[f], f
    }
  }
' "$FUNC_REPORT_FILE" | sort -n > "$FILE_REPORT_FILE"
}

print_package_summary() {
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
      avg = sum[p] / cnt[p]
      target = any
      if (p ~ /\/service\.go$/) {
        target = svc
      } else if (p ~ /\/postgres\.go$/) {
        target = pg
      } else if (p ~ /\/[^/]*resolvers\.go$/ || p ~ /\/resolver\.go$/ || p ~ /\/server\.go$/ || p ~ /\/chain\.go$/) {
        target = wiring
      } else if (p ~ /\/entity\.go$/) {
        target = ent
      }
      printf "%.1f\t%s\t%s\n", avg, target, p
    }
  }
' any="$MIN_ANY_FILE_COVERAGE" svc="$MIN_SERVICE_FILE_COVERAGE" pg="$MIN_POSTGRES_FILE_COVERAGE" wiring="$MIN_WIRING_FILE_COVERAGE" ent="$MIN_ENTITY_FILE_COVERAGE" "$FUNC_REPORT_FILE" \
  | sort -n \
  | awk -F'\t' '{printf "  %6.1f%%  (target: %s%%)  %s\n", $1, $2, $3}'
}

print_lowest_functions() {
  echo ""
  echo "== top 20 lowest-covered functions"
  awk '
  $1 ~ /^total:/ {next}
  {
    pct = $3
    gsub("%", "", pct)
    printf "%.1f\t%s\t%s\n", pct, $1, $2
  }
' "$FUNC_REPORT_FILE" | sort -n | \
    awk 'NR <= 20 {printf "  %6.1f%%  %s %s\n", $1, $2, $3}'
}

print_per_file_floor_banner() {
  echo ""
  echo "== per-file coverage floors"
  if [[ "$ENFORCE_PER_FILE" -eq 1 ]]; then
    echo "  any measured file >= ${MIN_ANY_FILE_COVERAGE}%"
    echo "  service.go >= ${MIN_SERVICE_FILE_COVERAGE}%"
    echo "  postgres.go >= ${MIN_POSTGRES_FILE_COVERAGE}%"
    echo "  wiring files (*resolvers.go,resolver.go,server.go,chain.go) >= ${MIN_WIRING_FILE_COVERAGE}%"
    echo "  entity.go >= ${MIN_ENTITY_FILE_COVERAGE}%"
  else
    echo "  per-file enforcement disabled"
  fi
}

evaluate_per_file_floors() {
  while IFS=$'\t' read -r pct file; do
    # Never enforce per-file floors for gqlgen-generated sources.
    if [[ "$file" =~ $GENERATED_GRAPH_GO_RE ]]; then
      continue
    fi

    local required="$MIN_ANY_FILE_COVERAGE"

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

    local whitelisted_reason=""
    local whitelisted_expires=""
    for entry in "${PER_FILE_WHITELIST[@]}"; do
      local wl_file="${entry%%|*}"
      local rest="${entry#*|}"
      local wl_reason="${rest%%|*}"
      local wl_expires="${rest##*|}"
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
          printf "whitelist expired: %s (%s)\n" "$file" "$whitelisted_expires" >> "$PER_FILE_VIOLATIONS_FILE"
        fi
      else
        printf "%s\t%s\t%s\n" "$pct" "$required" "$file" >> "$PER_FILE_VIOLATIONS_FILE"
      fi
    fi
  done < "$FILE_REPORT_FILE"
}

print_function_floor_banner() {
  echo ""
  echo "== function-level floor (core files)"
  if [[ "$ENFORCE_FUNCTION_FLOOR" -eq 1 ]]; then
    echo "  core functions in service/postgres/resolvers must be >= ${MIN_CORE_FUNCTION_COVERAGE}%"
  fi
}

write_function_violations_file() {
  awk -v min="$MIN_CORE_FUNCTION_COVERAGE" -v generated_re="$GENERATED_GRAPH_GO_RE" '
  $1 ~ /^total:/ {next}
  {
    pct = $3
    gsub("%", "", pct)
    loc = $1
    # target core files only; never gqlgen-generated Go under graph/
    if (loc ~ generated_re) { next }
    if (loc ~ /\/internal\/.*\/service\.go:/ || loc ~ /\/internal\/.*\/postgres\.go:/ || loc ~ /\/transport\/graphql\/.*resolvers\.go:/) {
      if ((pct + 0) < (min + 0)) {
        printf "%.1f\t%s\t%s\n", pct, loc, $2
      }
    }
  }
' "$FUNC_REPORT_FILE" | sort -n > "$FUNCTION_VIOLATIONS_FILE"
}

# Emit tab-separated violation rows from FUNCTION_VIOLATIONS_FILE (pct, loc, name).
emit_function_violation_lines() {
  awk -F'\t' '{printf "  %6.1f%% < %s%%  %s %s\n", $1, min, $2, $3}' min="$MIN_CORE_FUNCTION_COVERAGE" "$FUNCTION_VIOLATIONS_FILE"
}

auto_filter_thin_delegates() {
  if [[ "$AUTO_FILTER_THIN_DELEGATES" -ne 1 ]]; then
    return 0
  fi
  if [[ ! -s "$FUNCTION_VIOLATIONS_FILE" ]]; then
    echo "  PASS"
    return 0
  fi

  # Rewrites FUNCTION_VIOLATIONS_FILE and prints summary + only remaining violations (or PASS).
  if ! go run "${SCRIPT_DIR}/coverage_filter" \
    --violations "$FUNCTION_VIOLATIONS_FILE" \
    --repo-root "$ROOT_DIR" \
    --min "$MIN_CORE_FUNCTION_COVERAGE" ; then
    echo "  WARN: auto thin-delegate filtering failed; listing unfiltered violations"
    echo "  FAIL: functions below ${MIN_CORE_FUNCTION_COVERAGE}%"
    emit_function_violation_lines
  fi
}

print_function_gate_section() {
  if [[ -s "$FUNCTION_VIOLATIONS_FILE" ]]; then
    echo "  FAIL: functions below ${MIN_CORE_FUNCTION_COVERAGE}%"
    emit_function_violation_lines
  else
    echo "  PASS"
  fi
}

coverage_total_percent() {
  awk '/^total:/ {gsub("%","",$3); print $3}' "$FUNC_REPORT_FILE"
}

print_global_gate() {
  local total="$1"
  if awk "BEGIN {exit !($total + 0 >= $MIN_COVERAGE + 0)}"; then
    echo "GLOBAL gate: PASS (${total}% >= ${MIN_COVERAGE}%)"
  else
    echo "GLOBAL gate: FAIL (${total}% < ${MIN_COVERAGE}%)"
    GLOBAL_FAIL=1
  fi
}

print_per_file_gate() {
  if [[ -s "$PER_FILE_VIOLATIONS_FILE" ]]; then
    PER_FILE_FAIL=1
    echo "PER-FILE gate: FAIL (one or more files below required minimum)"
    echo "== per-file violations (actual < required)"
    awk -F'\t' '{printf "  %6.2f%% < %s%%  %s\n", $1, $2, $3}' "$PER_FILE_VIOLATIONS_FILE"
  else
    echo "PER-FILE gate: PASS"
  fi
}

print_function_gate_flag() {
  if [[ -s "$FUNCTION_VIOLATIONS_FILE" ]]; then
    FUNCTION_FAIL=1
    echo "FUNCTION gate: FAIL (one or more core functions below minimum)"
  else
    FUNCTION_FAIL=0
    echo "FUNCTION gate: PASS"
  fi
}

print_test_execution_gate() {
  if [[ "$TEST_EXEC_FAIL" -eq 1 ]]; then
    echo "TEST EXECUTION gate: FAIL (go test command failed before/while generating report)"
  else
    echo "TEST EXECUTION gate: PASS"
  fi
}

finalize_gates() {
  echo ""
  if [[ "$GLOBAL_FAIL" -eq 0 && "$PER_FILE_FAIL" -eq 0 && "$FUNCTION_FAIL" -eq 0 && "$TEST_EXEC_FAIL" -eq 0 ]]; then
    echo "Coverage gates: PASS (global + per-file + function)"
  else
    echo "Coverage gates: FAIL"
    if [[ "$GLOBAL_FAIL" -eq 1 ]]; then
      echo "Reason: global coverage threshold not met."
    fi
    if [[ "$PER_FILE_FAIL" -eq 1 ]]; then
      echo "Reason: per-file minimum threshold violations."
    fi
    if [[ "$FUNCTION_FAIL" -eq 1 ]]; then
      echo "Reason: function-level minimum threshold violations in core files."
    fi
    if [[ "$TEST_EXEC_FAIL" -eq 1 ]]; then
      echo "Reason: go test execution failed."
    fi
    exit 1
  fi
}

main() {
  local COVERAGE_IMPORTS
  COVERAGE_IMPORTS="$(coverage_imports_csv)"

  echo "== coverage (unit + integration)"
  echo "== measured packages: ${COVERAGE_PKGS[*]}"

  run_tests_with_coverage "$COVERAGE_IMPORTS"
  write_run_metadata
  if [[ "$GO_TEST_EXIT" -ne 0 ]]; then
    TEST_EXEC_FAIL=1
    echo "go test execution returned non-zero exit code: ${GO_TEST_EXIT}"
  fi

  setup_temp_reports
  build_file_level_report
  print_package_summary
  print_lowest_functions
  print_per_file_floor_banner
  evaluate_per_file_floors
  print_function_floor_banner
  write_function_violations_file
  auto_filter_thin_delegates
  if [[ "$AUTO_FILTER_THIN_DELEGATES" -ne 1 ]]; then
    print_function_gate_section
  fi

  local COVERAGE_TOTAL
  COVERAGE_TOTAL="$(coverage_total_percent)"
  echo ""

  print_global_gate "$COVERAGE_TOTAL"
  print_per_file_gate
  print_function_gate_flag
  print_test_execution_gate
  finalize_gates
}

main "$@"
