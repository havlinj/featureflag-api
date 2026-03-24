#!/usr/bin/env python3
"""
coverage_hotspots.py

Analyze Go coverage profile (coverage.out) and print:
1) Lowest-covered functions from `go tool cover -func`
2) Functions with the most uncovered blocks (`count=0`) from raw profile
3) Example uncovered snippets per function for targeted test writing

Usage:
  python3 scripts/coverage/coverage_hotspots.py
  python3 scripts/coverage/coverage_hotspots.py --coverage-file coverage.out --top 25 --examples 3
  python3 scripts/coverage/coverage_hotspots.py --files internal/flags/service.go internal/experiments/service.go
"""

from __future__ import annotations

import argparse
import json
import os
import re
import subprocess
import sys
from datetime import datetime, timezone
from collections import defaultdict
from dataclasses import dataclass
from typing import Dict, List, Set, Tuple


@dataclass
class ZeroBlock:
    file: str
    sl: int
    el: int
    num: int
    count: int


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Show coverage hotspots from coverage.out")
    parser.add_argument("--coverage-file", default="coverage.out", help="Path to coverage.out")
    parser.add_argument("--top", type=int, default=20, help="How many rows to print per section")
    parser.add_argument("--examples", type=int, default=2, help="Example snippets per function")
    parser.add_argument("--gate", type=float, default=50.0, help="Function coverage gate threshold in percent")
    parser.add_argument(
        "--state-file",
        default=".coverage_hotspots_state.json",
        help="Path to local state snapshot for delta comparison",
    )
    parser.add_argument(
        "--no-state",
        action="store_true",
        help="Do not read/write local state snapshot",
    )
    parser.add_argument(
        "--files",
        nargs="*",
        default=[],
        help="Optional relative file filters, e.g. internal/flags/service.go",
    )
    return parser.parse_args()


def run_cover_func(coverage_file: str) -> List[Tuple[float, str]]:
    go_bin = "go"
    if os.path.exists("/usr/local/go/bin/go"):
        go_bin = "/usr/local/go/bin/go"
    cmd = [go_bin, "tool", "cover", f"-func={coverage_file}"]
    proc = subprocess.run(cmd, capture_output=True, text=True)
    if proc.returncode != 0:
        raise RuntimeError(proc.stderr.strip() or proc.stdout.strip() or "go tool cover failed")

    out: List[Tuple[float, str]] = []
    pattern = re.compile(r"^(.+?):\s+(\S+)\s+([\d.]+)%$")
    for line in proc.stdout.splitlines():
        line = line.strip()
        if not line or line.startswith("total:"):
            continue
        match = pattern.match(line)
        if not match:
            continue
        name = f"{match.group(1)}: {match.group(2)}"
        pct = float(match.group(3))
        out.append((pct, name))
    return sorted(out, key=lambda item: item[0])


def in_file_scope(name: str, selected_files: List[str]) -> bool:
    if not selected_files:
        return True
    return any(file_filter in name for file_filter in selected_files)


def below_gate_functions(
    func_rows: List[Tuple[float, str]], gate: float, selected_files: List[str]
) -> List[Tuple[float, str]]:
    out: List[Tuple[float, str]] = []
    for pct, name in func_rows:
        if not in_file_scope(name, selected_files):
            continue
        if pct < gate:
            out.append((pct, name))
    return out


def parse_coverage_profile(coverage_file: str) -> List[ZeroBlock]:
    if not os.path.exists(coverage_file):
        raise FileNotFoundError(f"Coverage file not found: {coverage_file}")

    pattern = re.compile(
        r"^(?P<file>.+?):(?P<sl>\d+)\.\d+,(?P<el>\d+)\.\d+\s+(?P<num>\d+)\s+(?P<count>\d+)$"
    )
    out: List[ZeroBlock] = []
    with open(coverage_file, encoding="utf-8") as handle:
        _ = handle.readline()  # mode: atomic/set/count
        for line in handle:
            match = pattern.match(line.strip())
            if not match:
                continue
            count = int(match.group("count"))
            if count != 0:
                continue
            out.append(
                ZeroBlock(
                    file=match.group("file"),
                    sl=int(match.group("sl")),
                    el=int(match.group("el")),
                    num=int(match.group("num")),
                    count=count,
                )
            )
    return out


def load_file_lines(repo_root: str, module_path_file: str, cache: Dict[str, List[str]]) -> List[str]:
    rel = module_path_file.replace("github.com/havlinj/featureflag-api/", "")
    abs_path = os.path.join(repo_root, rel)
    if rel not in cache:
        with open(abs_path, encoding="utf-8") as handle:
            cache[rel] = handle.readlines()
    return cache[rel]


def nearest_function_name(lines: List[str], start_line: int) -> str:
    signature = re.compile(r"^func\s+(\([^)]*\)\s*)?([A-Za-z0-9_]+)\s*\(")
    idx = max(0, min(start_line - 1, len(lines) - 1))
    for i in range(idx, -1, -1):
        match = signature.match(lines[i].strip())
        if match:
            return match.group(2)
    return "(unknown)"


def keep_file(block: ZeroBlock, selected_files: List[str]) -> bool:
    if not selected_files:
        return True
    rel = block.file.replace("github.com/havlinj/featureflag-api/", "")
    return rel in selected_files


def summarize_zero_blocks(
    repo_root: str, blocks: List[ZeroBlock], selected_files: List[str]
) -> Tuple[List[Tuple[str, str, int, int]], Dict[Tuple[str, str], List[Tuple[int, int, str]]]]:
    cache: Dict[str, List[str]] = {}
    grouped: Dict[Tuple[str, str], List[Tuple[int, int, str]]] = defaultdict(list)

    for block in blocks:
        if not keep_file(block, selected_files):
            continue
        lines = load_file_lines(repo_root, block.file, cache)
        rel = block.file.replace("github.com/havlinj/featureflag-api/", "")
        name = nearest_function_name(lines, block.sl)
        sl = max(1, min(block.sl, len(lines)))
        el = max(1, min(block.el, len(lines)))
        snippet = "".join(lines[sl - 1 : el]).strip().replace("\n", " ")
        grouped[(rel, name)].append((sl, el, snippet[:220]))

    summary: List[Tuple[str, str, int, int]] = []
    for (rel, name), spans in grouped.items():
        line_span = sum(el - sl + 1 for sl, el, _ in spans)
        summary.append((rel, name, len(spans), line_span))

    summary.sort(key=lambda item: (-item[2], -item[3], item[0], item[1]))
    return summary, grouped


def load_previous_state(path: str) -> dict:
    if not os.path.exists(path):
        return {}
    with open(path, encoding="utf-8") as handle:
        return json.load(handle)


def save_state(path: str, gate: float, selected_files: List[str], below_gate: List[Tuple[float, str]]) -> None:
    payload = {
        "saved_at": datetime.now(timezone.utc).isoformat(),
        "gate": gate,
        "selected_files": selected_files,
        "below_gate": [{"pct": pct, "name": name} for pct, name in below_gate],
        "below_gate_count": len(below_gate),
    }
    with open(path, "w", encoding="utf-8") as handle:
        json.dump(payload, handle, ensure_ascii=False, indent=2)


def print_delta(previous: dict, current_set: Set[str], gate: float) -> None:
    prev_items = previous.get("below_gate", [])
    prev_set = {item.get("name", "") for item in prev_items if item.get("name")}
    prev_count = len(prev_set)
    curr_count = len(current_set)
    diff = curr_count - prev_count

    print("== function-gate delta vs previous run")
    print(f"previous below gate ({gate:.1f}%): {prev_count}")
    print(f"current  below gate ({gate:.1f}%): {curr_count}")
    sign = "+" if diff > 0 else ""
    print(f"delta: {sign}{diff}")

    removed = sorted(prev_set - current_set)
    added = sorted(current_set - prev_set)
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


def main() -> int:
    args = parse_args()
    repo_root = os.getcwd()

    func_rows: List[Tuple[float, str]] = []
    try:
        func_rows = run_cover_func(args.coverage_file)
    except Exception as exc:  # noqa: BLE001
        print(f"WARN: skipping go tool cover -func section: {exc}", file=sys.stderr)

    try:
        zero_blocks = parse_coverage_profile(args.coverage_file)
    except Exception as exc:  # noqa: BLE001
        print(f"ERROR: cannot parse coverage profile: {exc}", file=sys.stderr)
        return 1

    summary, grouped = summarize_zero_blocks(repo_root, zero_blocks, args.files)

    if func_rows:
        print("== lowest-covered functions (go tool cover -func)")
        printed = 0
        for pct, name in func_rows:
            if args.files:
                if not any(f in name for f in args.files):
                    continue
            print(f"{pct:7.1f}%  {name}")
            printed += 1
            if printed >= args.top:
                break
        print()

        below_gate = below_gate_functions(func_rows, args.gate, args.files)
        print(f"== function gate status (< {args.gate:.1f}%)")
        print(f"functions below gate: {len(below_gate)}")
        for pct, name in below_gate[: args.top]:
            print(f"{pct:7.1f}%  {name}")
        print()

        if not args.no_state:
            previous = load_previous_state(args.state_file)
            if previous:
                current_set = {name for _, name in below_gate}
                print_delta(previous, current_set, args.gate)
            save_state(args.state_file, args.gate, args.files, below_gate)
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
        for sl, el, snippet in grouped[(rel, name)][: args.examples]:
            print(f"  L{sl}-L{el}: {snippet}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

