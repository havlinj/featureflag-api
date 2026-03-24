# Coverage scripts

This folder contains all tooling related to coverage measurement, quality gates, and hotspot analysis.

## What is here

- `test_coverage.sh`: runs unit + integration coverage for production packages and enforces gates
  - global gate (default 75%)
  - generic per-file floor (default 30%) for measured non-generated files
  - per-file floors for core file roles
  - function-level floor for core functions (default 40%)
  - deterministic mode by default (clears Go test cache + runs with `-count=1`)
  - optional fast mode via `COVERAGE_ALLOW_CACHE=1`
  - generated GraphQL Go files under `graph/**/*.go` are excluded from per-file/function enforcement
- `coverage_filter/`: Go helper that post-processes function-level violations
  - skips generated `graph/**/*.go`
  - skips thin delegate wrappers
- `coverage_hotspots.py`: coverage profile analyzer for targeted test planning
  - prints lowest-covered functions
  - prints count of functions below function gate
  - supports local delta against previous run via state file
  - keeps state history (default 10 snapshots), prints delta vs immediate previous run, and if project tree is unchanged it auto-walks older snapshots to find the last git-changed run for comparison
  - deduplicates identical consecutive snapshots via `repeat_count`

## How it is integrated

- CI coverage step calls `bash scripts/coverage/test_coverage.sh`.
- Local usage should call the same path directly.

## Usage

Run full coverage gates:

```bash
bash scripts/coverage/test_coverage.sh
COVERAGE_ALLOW_CACHE=1 bash scripts/coverage/test_coverage.sh
```

Hotspot analysis:

```bash
python3 scripts/coverage/coverage_hotspots.py
```

Useful options:

```bash
python3 scripts/coverage/coverage_hotspots.py --files internal/flags/service.go internal/experiments/service.go
python3 scripts/coverage/coverage_hotspots.py --top 30 --examples 3
python3 scripts/coverage/coverage_hotspots.py --gate 50 --state-file scripts/coverage/state/coverage_hotspots_state.json
python3 scripts/coverage/coverage_hotspots.py --gate 50 --state-file scripts/coverage/state/coverage_hotspots_demo_state.json
python3 scripts/coverage/coverage_hotspots.py --state-history-size 20
python3 scripts/coverage/coverage_hotspots.py --no-state
```

Generate HTML for visual drill-down:

```bash
go tool cover -html=coverage.out -o coverage.html
```

