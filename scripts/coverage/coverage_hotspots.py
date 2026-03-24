#!/usr/bin/env python3
"""
Analyze Go coverage hotspots from `coverage.out`.

Outputs:
1) Lowest-covered functions from `go tool cover -func`
2) Functions with the most uncovered blocks (`count=0`) from the profile
3) Uncovered snippet examples per function
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import subprocess
import sys
from collections import defaultdict
from dataclasses import asdict, dataclass
from datetime import datetime, timezone
from typing import Dict, List, Optional, Sequence, Tuple


@dataclass(frozen=True)
class FunctionCoverage:
    percent: float
    name: str


@dataclass(frozen=True)
class ZeroBlock:
    module_file: str
    start_line: int
    end_line: int


@dataclass(frozen=True)
class GitMeta:
    available: bool
    head: str = ""
    dirty: bool = False
    status_count: int = 0
    status_digest: str = ""


@dataclass(frozen=True)
class RunMeta:
    generated_at: str = "unknown"
    cache_mode: str = "unknown"
    go_version: str = "unknown"
    coverage_profile_sha256: str = ""
    includes_bash_integration_tests: bool = False

    @classmethod
    def from_dict(cls, data: dict) -> "RunMeta":
        if not data:
            return cls()
        return cls(
            generated_at=str(data.get("generated_at", "unknown")),
            cache_mode=str(data.get("cache_mode", "unknown")),
            go_version=str(data.get("go_version", "unknown")),
            coverage_profile_sha256=str(data.get("coverage_profile_sha256", "")),
            includes_bash_integration_tests=bool(
                data.get("includes_bash_integration_tests", False)
            ),
        )


@dataclass
class Snapshot:
    saved_at: str
    gate: float
    selected_files: List[str]
    below_gate: List[FunctionCoverage]
    function_rows: List[FunctionCoverage]
    coverage_profile_sha256: str
    git: GitMeta
    repeat_count: int = 1

    @classmethod
    def from_dict(cls, data: dict) -> "Snapshot":
        return cls(
            saved_at=str(data.get("saved_at", "")),
            gate=float(data.get("gate", 0.0)),
            selected_files=[str(item) for item in data.get("selected_files", [])],
            below_gate=[
                FunctionCoverage(percent=float(item["pct"]), name=str(item["name"]))
                for item in data.get("below_gate", [])
                if "pct" in item and "name" in item
            ],
            function_rows=[
                FunctionCoverage(percent=float(item["pct"]), name=str(item["name"]))
                for item in data.get("function_rows", [])
                if "pct" in item and "name" in item
            ],
            coverage_profile_sha256=str(data.get("coverage_profile_sha256", "")),
            git=GitMeta(**data.get("git", {"available": False})),
            repeat_count=int(data.get("repeat_count", 1)),
        )

    def to_dict(self) -> dict:
        return {
            "saved_at": self.saved_at,
            "gate": self.gate,
            "selected_files": self.selected_files,
            "below_gate": [
                {"pct": row.percent, "name": row.name} for row in self.below_gate
            ],
            "below_gate_count": len(self.below_gate),
            "function_rows": [
                {"pct": row.percent, "name": row.name} for row in self.function_rows
            ],
            "coverage_profile_sha256": self.coverage_profile_sha256,
            "git": asdict(self.git),
            "repeat_count": self.repeat_count,
        }

    def signature(self) -> Tuple[Tuple[str, float], ...]:
        return tuple(
            sorted((row.name, round(row.percent, 1)) for row in self.function_rows)
        )

    def git_identity(self) -> Tuple[str, str]:
        return (self.git.head, self.git.status_digest)


class SnapshotStore:
    def __init__(self, path: str) -> None:
        self.path = path
        self._history: List[Snapshot] = []
        self._load()

    @property
    def history(self) -> List[Snapshot]:
        return self._history

    def _load(self) -> None:
        if not os.path.exists(self.path):
            self._history = []
            return
        with open(self.path, encoding="utf-8") as handle:
            payload = json.load(handle)
        raw_history = payload.get("history", [])
        if isinstance(raw_history, list):
            self._history = [
                Snapshot.from_dict(item)
                for item in raw_history
                if isinstance(item, dict)
            ]
        elif isinstance(payload, dict):
            self._history = [Snapshot.from_dict(payload)]
        else:
            self._history = []

    def save(self, snapshot: Snapshot, history_size: int) -> None:
        state_dir = os.path.dirname(self.path)
        if state_dir:
            os.makedirs(state_dir, exist_ok=True)

        if self._history and self._history[-1].signature() == snapshot.signature():
            last = self._history[-1]
            self._history[-1] = Snapshot(
                saved_at=snapshot.saved_at,
                gate=last.gate,
                selected_files=last.selected_files,
                below_gate=last.below_gate,
                function_rows=last.function_rows,
                coverage_profile_sha256=last.coverage_profile_sha256,
                git=snapshot.git,
                repeat_count=last.repeat_count + 1,
            )
        else:
            self._history.append(snapshot)

        if history_size > 0:
            self._history = self._history[-history_size:]

        latest = self._history[-1] if self._history else snapshot
        payload = {
            "history": [item.to_dict() for item in self._history],
            "latest": latest.to_dict(),
        }
        with open(self.path, "w", encoding="utf-8") as handle:
            json.dump(payload, handle, ensure_ascii=False, indent=2)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Show coverage hotspots from coverage.out"
    )
    parser.add_argument(
        "--coverage-file", default="coverage.out", help="Path to coverage.out"
    )
    parser.add_argument(
        "--top", type=int, default=20, help="How many rows to print per section"
    )
    parser.add_argument(
        "--examples", type=int, default=2, help="Example snippets per function"
    )
    parser.add_argument(
        "--gate",
        type=float,
        default=50.0,
        help="Function coverage gate threshold in percent",
    )
    parser.add_argument(
        "--state-file",
        default="scripts/coverage/state/coverage_hotspots_state.json",
        help="Path to local state snapshot for delta comparison",
    )
    parser.add_argument(
        "--no-state", action="store_true", help="Do not read/write local state snapshot"
    )
    parser.add_argument(
        "--files",
        nargs="*",
        default=[],
        help="Optional relative file filters, e.g. internal/flags/service.go",
    )
    parser.add_argument(
        "--state-history-size", type=int, default=10, help="How many snapshots to keep"
    )
    parser.add_argument(
        "--run-meta-file",
        default="scripts/coverage/state/coverage_run_meta.json",
        help="Path to metadata generated by test_coverage.sh",
    )
    return parser.parse_args()


def file_sha256(path: str) -> str:
    digest = hashlib.sha256()
    with open(path, "rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def run_cover_func(coverage_file: str) -> List[FunctionCoverage]:
    go_bin = "/usr/local/go/bin/go" if os.path.exists("/usr/local/go/bin/go") else "go"
    proc = subprocess.run(
        [go_bin, "tool", "cover", f"-func={coverage_file}"],
        capture_output=True,
        text=True,
    )
    if proc.returncode != 0:
        raise RuntimeError(
            proc.stderr.strip() or proc.stdout.strip() or "go tool cover failed"
        )

    pattern = re.compile(r"^(.+?):\s+(\S+)\s+([\d.]+)%$")
    rows: List[FunctionCoverage] = []
    for raw_line in proc.stdout.splitlines():
        line = raw_line.strip()
        if not line or line.startswith("total:"):
            continue
        match = pattern.match(line)
        if not match:
            continue
        rows.append(
            FunctionCoverage(
                percent=float(match.group(3)),
                name=f"{match.group(1)}: {match.group(2)}",
            )
        )
    return sorted(rows, key=lambda item: item.percent)


def in_file_scope(name: str, selected_files: Sequence[str]) -> bool:
    if not selected_files:
        return True
    return any(file_filter in name for file_filter in selected_files)


def below_gate_functions(
    rows: Sequence[FunctionCoverage], gate: float, selected_files: Sequence[str]
) -> List[FunctionCoverage]:
    return [
        row
        for row in rows
        if in_file_scope(row.name, selected_files) and row.percent < gate
    ]


def parse_coverage_profile(coverage_file: str) -> List[ZeroBlock]:
    pattern = re.compile(
        r"^(?P<file>.+?):(?P<sl>\d+)\.\d+,(?P<el>\d+)\.\d+\s+(?P<num>\d+)\s+(?P<count>\d+)$"
    )
    blocks: List[ZeroBlock] = []
    with open(coverage_file, encoding="utf-8") as handle:
        _ = handle.readline()
        for raw_line in handle:
            match = pattern.match(raw_line.strip())
            if not match:
                continue
            if int(match.group("count")) != 0:
                continue
            blocks.append(
                ZeroBlock(
                    module_file=match.group("file"),
                    start_line=int(match.group("sl")),
                    end_line=int(match.group("el")),
                )
            )
    return blocks


def load_file_lines(
    repo_root: str, module_path_file: str, cache: Dict[str, List[str]]
) -> List[str]:
    rel = module_path_file.replace("github.com/havlinj/featureflag-api/", "")
    if rel not in cache:
        with open(os.path.join(repo_root, rel), encoding="utf-8") as handle:
            cache[rel] = handle.readlines()
    return cache[rel]


def nearest_function_name(lines: Sequence[str], start_line: int) -> str:
    signature = re.compile(r"^func\s+(\([^)]*\)\s*)?([A-Za-z0-9_]+)\s*\(")
    idx = max(0, min(start_line - 1, len(lines) - 1))
    for i in range(idx, -1, -1):
        match = signature.match(lines[i].strip())
        if match:
            return match.group(2)
    return "(unknown)"


def summarize_zero_blocks(
    repo_root: str, blocks: Sequence[ZeroBlock], selected_files: Sequence[str]
) -> Tuple[
    List[Tuple[str, str, int, int]], Dict[Tuple[str, str], List[Tuple[int, int, str]]]
]:
    cache: Dict[str, List[str]] = {}
    grouped: Dict[Tuple[str, str], List[Tuple[int, int, str]]] = defaultdict(list)

    for block in blocks:
        rel = block.module_file.replace("github.com/havlinj/featureflag-api/", "")
        if selected_files and rel not in selected_files:
            continue
        lines = load_file_lines(repo_root, block.module_file, cache)
        name = nearest_function_name(lines, block.start_line)
        start_line = max(1, min(block.start_line, len(lines)))
        end_line = max(1, min(block.end_line, len(lines)))
        snippet = "".join(lines[start_line - 1 : end_line]).strip().replace("\n", " ")
        grouped[(rel, name)].append((start_line, end_line, snippet[:220]))

    summary: List[Tuple[str, str, int, int]] = []
    for (rel, name), spans in grouped.items():
        line_span = sum(end - start + 1 for start, end, _ in spans)
        summary.append((rel, name, len(spans), line_span))
    summary.sort(key=lambda item: (-item[2], -item[3], item[0], item[1]))
    return summary, grouped


def run_git(args: Sequence[str]) -> Tuple[bool, str]:
    proc = subprocess.run(["git", *args], capture_output=True, text=True)
    if proc.returncode != 0:
        return False, (proc.stderr.strip() or proc.stdout.strip())
    return True, proc.stdout.strip()


def collect_git_meta() -> GitMeta:
    ok, _ = run_git(["rev-parse", "--is-inside-work-tree"])
    if not ok:
        return GitMeta(available=False)

    ok_head, head = run_git(["rev-parse", "HEAD"])
    ok_status, status = run_git(["status", "--porcelain"])
    status_text = status if ok_status else ""
    status_lines = [line for line in status_text.splitlines() if line.strip()]
    status_digest = hashlib.sha256("\n".join(status_lines).encode("utf-8")).hexdigest()
    return GitMeta(
        available=True,
        head=head if ok_head else "",
        dirty=bool(status_lines),
        status_count=len(status_lines),
        status_digest=status_digest,
    )


def load_run_meta(path: str) -> RunMeta:
    if not os.path.exists(path):
        return RunMeta()
    with open(path, encoding="utf-8") as handle:
        return RunMeta.from_dict(json.load(handle))


def build_snapshot(
    gate: float,
    selected_files: Sequence[str],
    below_gate: Sequence[FunctionCoverage],
    function_rows: Sequence[FunctionCoverage],
    coverage_sha256: str,
) -> Snapshot:
    scoped_rows = [
        row for row in function_rows if in_file_scope(row.name, selected_files)
    ]
    return Snapshot(
        saved_at=datetime.now(timezone.utc).isoformat(),
        gate=gate,
        selected_files=list(selected_files),
        below_gate=list(below_gate),
        function_rows=scoped_rows,
        coverage_profile_sha256=coverage_sha256,
        git=collect_git_meta(),
    )


def print_profile_context(
    coverage_file: str, run_meta: RunMeta, coverage_sha256: str
) -> None:
    print("== coverage profile context")
    print(f"profile: {coverage_file}")
    print(
        "source: includes Go tests (including ./test/integration), excludes bash scripts under scripts/integration"
    )
    if run_meta.generated_at == "unknown":
        print("WARN: run metadata not found; profile provenance is unknown")
        print()
        return
    print(f"generated_at: {run_meta.generated_at}")
    print(f"cache_mode: {run_meta.cache_mode}")
    print(f"go_version: {run_meta.go_version}")
    print(
        f"includes_bash_integration_tests: {run_meta.includes_bash_integration_tests}"
    )
    if (
        run_meta.coverage_profile_sha256
        and run_meta.coverage_profile_sha256 != coverage_sha256
    ):
        print(
            "WARN: coverage.out hash does not match latest run metadata (profile may be stale)"
        )
    print()


def print_profile_delta(previous: Snapshot, current_sha256: str) -> None:
    if not previous.coverage_profile_sha256:
        return
    print("== coverage profile delta")
    if previous.coverage_profile_sha256 == current_sha256:
        print("same coverage profile content as compared snapshot")
    else:
        print("coverage profile content changed since compared snapshot")
    print()


def print_git_delta(previous: Snapshot, current: Snapshot) -> None:
    if not previous.git.available or not current.git.available:
        return
    print("== git workspace delta")
    if previous.git_identity() == current.git_identity():
        print("no repository change detected since compared snapshot")
        print()
        return
    if previous.git.head != current.git.head:
        print(f"HEAD changed: {previous.git.head[:12]} -> {current.git.head[:12]}")
    if previous.git.status_digest != current.git.status_digest:
        print(
            f"working tree changed: status entries {previous.git.status_count} -> {current.git.status_count}"
        )
    print()


def print_function_delta(
    previous: Snapshot,
    current_rows: Sequence[FunctionCoverage],
    gate: float,
    heading: str,
) -> None:
    previous_below = {row.name for row in previous.below_gate}
    current_below = {row.name for row in current_rows if row.percent < gate}
    diff = len(current_below) - len(previous_below)

    print(f"== {heading}")
    print(f"previous below gate ({gate:.1f}%): {len(previous_below)}")
    print(f"current  below gate ({gate:.1f}%): {len(current_below)}")
    print(f"delta: {'+' if diff > 0 else ''}{diff}")

    removed = sorted(previous_below - current_below)
    added = sorted(current_below - previous_below)
    if removed:
        print("improved (left below-gate set):")
        for name in removed[:10]:
            print(f"  - {name}")
        if len(removed) > 10:
            print(f"  ... and {len(removed) - 10} more")
    if added:
        print("regressed (newly below-gate):")
        for name in added[:10]:
            print(f"  - {name}")
        if len(added) > 10:
            print(f"  ... and {len(added) - 10} more")
    if not removed and not added:
        print("no set change in below-gate functions")
    print()

    previous_map = {row.name: row.percent for row in previous.function_rows}
    current_map = {row.name: row.percent for row in current_rows}
    changed: List[Tuple[float, float, float, str]] = []
    for name in sorted(set(previous_map) & set(current_map)):
        old = previous_map[name]
        new = current_map[name]
        delta = new - old
        if abs(delta) >= 0.1:
            changed.append((delta, old, new, name))

    print("== function coverage delta (%) vs previous run")
    if not changed:
        print("no per-function percentage changes detected")
        print()
        return
    improved = sorted(
        (item for item in changed if item[0] > 0),
        key=lambda item: item[0],
        reverse=True,
    )
    regressed = sorted(
        (item for item in changed if item[0] < 0), key=lambda item: item[0]
    )
    print("most improved:")
    for delta, old, new, name in improved[:10]:
        print(f"  +{delta:.1f} pp  {old:.1f}% -> {new:.1f}%  {name}")
    if regressed:
        print("most regressed:")
        for delta, old, new, name in regressed[:10]:
            print(f"  {delta:.1f} pp  {old:.1f}% -> {new:.1f}%  {name}")
    print()


def pick_last_distinct_snapshot(
    history: Sequence[Snapshot], current: Snapshot
) -> Optional[Snapshot]:
    current_signature = current.signature()
    for snapshot in reversed(history):
        if snapshot.signature() != current_signature:
            return snapshot
    return None


def pick_last_git_changed_snapshot(
    history: Sequence[Snapshot], current: Snapshot
) -> Optional[Snapshot]:
    current_identity = current.git_identity()
    for snapshot in reversed(history):
        if snapshot.git_identity() != current_identity:
            return snapshot
    return None


def print_top_functions(
    rows: Sequence[FunctionCoverage], top: int, selected_files: Sequence[str]
) -> None:
    print("== lowest-covered functions (go tool cover -func)")
    printed = 0
    for row in rows:
        if not in_file_scope(row.name, selected_files):
            continue
        print(f"{row.percent:7.1f}%  {row.name}")
        printed += 1
        if printed >= top:
            break
    print()


def print_below_gate(rows: Sequence[FunctionCoverage], gate: float, top: int) -> None:
    print(f"== function gate status (< {gate:.1f}%)")
    print(f"functions below gate: {len(rows)}")
    for row in rows[:top]:
        print(f"{row.percent:7.1f}%  {row.name}")
    print()


def main() -> int:
    args = parse_args()
    try:
        coverage_sha256 = file_sha256(args.coverage_file)
    except Exception as exc:  # noqa: BLE001
        print(f"ERROR: cannot read coverage profile: {exc}", file=sys.stderr)
        return 1

    run_meta = load_run_meta(args.run_meta_file)
    print_profile_context(args.coverage_file, run_meta, coverage_sha256)

    try:
        function_rows = run_cover_func(args.coverage_file)
    except Exception as exc:  # noqa: BLE001
        print(f"WARN: skipping go tool cover -func section: {exc}", file=sys.stderr)
        function_rows = []

    try:
        zero_blocks = parse_coverage_profile(args.coverage_file)
    except Exception as exc:  # noqa: BLE001
        print(f"ERROR: cannot parse coverage profile: {exc}", file=sys.stderr)
        return 1

    summary, grouped = summarize_zero_blocks(os.getcwd(), zero_blocks, args.files)

    if function_rows:
        print_top_functions(function_rows, args.top, args.files)
        below_gate = below_gate_functions(function_rows, args.gate, args.files)
        print_below_gate(below_gate, args.gate, args.top)

        if not args.no_state:
            current_snapshot = build_snapshot(
                args.gate, args.files, below_gate, function_rows, coverage_sha256
            )
            store = SnapshotStore(args.state_file)
            previous = store.history[-1] if store.history else None
            if previous:
                print_function_delta(
                    previous,
                    function_rows,
                    args.gate,
                    "function-gate delta vs immediate previous run",
                )
                print_profile_delta(previous, coverage_sha256)
                print_git_delta(previous, current_snapshot)
                if previous.git_identity() == current_snapshot.git_identity():
                    git_changed = pick_last_git_changed_snapshot(
                        store.history, current_snapshot
                    )
                    if git_changed:
                        print_function_delta(
                            git_changed,
                            function_rows,
                            args.gate,
                            "function-gate delta vs last git-changed run",
                        )
                        print_profile_delta(git_changed, coverage_sha256)
                        print_git_delta(git_changed, current_snapshot)
                    else:
                        distinct = pick_last_distinct_snapshot(
                            store.history, current_snapshot
                        )
                        if distinct:
                            print_function_delta(
                                distinct,
                                function_rows,
                                args.gate,
                                "function-gate delta vs last distinct run",
                            )
                            print_profile_delta(distinct, coverage_sha256)
                            print_git_delta(distinct, current_snapshot)
            store.save(current_snapshot, args.state_history_size)
    else:
        print("== function gate status")
        print("unavailable: go tool cover -func could not run in this environment")
        print()

    print("\n== functions with most uncovered blocks (count=0)")
    for rel, name, blocks_count, line_span in summary[: args.top]:
        print(f"{blocks_count:4d} blocks | {line_span:4d} linespan | {rel}:{name}")

    print("\n== uncovered snippet examples")
    for rel, name, _, _ in summary[: args.top]:
        print(f"- {rel}:{name}")
        for start_line, end_line, snippet in grouped[(rel, name)][: args.examples]:
            print(f"  L{start_line}-L{end_line}: {snippet}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
