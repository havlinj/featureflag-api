#!/usr/bin/env python3
"""
Verify that CI workflow runs all scripts referenced by scripts/test_all_full.sh.
"""

from __future__ import annotations

import re
import sys
from pathlib import Path


CORE_CI_SEQUENCE = (
    "scripts/check.sh",
    "scripts/test_unit.sh",
    "scripts/test_integration.sh",
    "scripts/coverage/test_coverage.sh",
    "scripts/build.sh",
    "scripts/test_binary_smoke.sh",
)

DISALLOWED_CI_MATRIX_SCRIPTS = (
    "scripts/format_python.sh",
)


def parse_expected(repo_root: Path) -> set[str]:
    full = (repo_root / "scripts" / "test_all_full.sh").read_text(encoding="utf-8")
    expected: set[str] = set()

    for match in re.findall(r'"\$SCRIPT_DIR/([^"]+\.sh)"', full):
        expected.add(f"scripts/{match}")

    if re.search(r'"\$SCRIPT_DIR"/integration/\*\.sh', full):
        for path in sorted((repo_root / "scripts" / "integration").glob("*.sh")):
            expected.add(f"scripts/integration/{path.name}")

    return expected


def parse_actual(repo_root: Path) -> set[str]:
    ci = (repo_root / ".github" / "workflows" / "ci.yml").read_text(encoding="utf-8")
    direct = set(re.findall(r"\bbash\s+(scripts/[A-Za-z0-9_./-]+\.sh)\b", ci))
    matrix_items = set(
        re.findall(
            r"^\s*-\s*(scripts/[A-Za-z0-9_./-]+\.sh)\s*$", ci, flags=re.MULTILINE
        )
    )
    return direct | matrix_items


def parse_ci_content(repo_root: Path) -> str:
    return (repo_root / ".github" / "workflows" / "ci.yml").read_text(encoding="utf-8")


def parse_matrix_list(ci_content: str) -> list[str]:
    return re.findall(
        r"^\s*-\s*(scripts/[A-Za-z0-9_./-]+\.sh)\s*$", ci_content, flags=re.MULTILINE
    )


def parse_direct_bash_steps(ci_content: str) -> list[str]:
    return re.findall(r"\bbash\s+(scripts/[A-Za-z0-9_./-]+\.sh)\b", ci_content)


def has_strict_order_flag(argv: list[str]) -> bool:
    return "--strict-order" in argv


def find_duplicates(items: list[str]) -> list[str]:
    counts: dict[str, int] = {}
    for item in items:
        counts[item] = counts.get(item, 0) + 1
    return sorted(item for item, count in counts.items() if count > 1)


def check_core_step_order(direct_steps: list[str]) -> list[str]:
    positions: dict[str, int] = {}
    for index, step in enumerate(direct_steps):
        if step not in positions:
            positions[step] = index

    problems: list[str] = []
    missing = [step for step in CORE_CI_SEQUENCE if step not in positions]
    if missing:
        problems.append("missing core CI steps in test job:")
        for step in missing:
            problems.append(f"  - {step}")
        return problems

    for first, second in zip(CORE_CI_SEQUENCE, CORE_CI_SEQUENCE[1:]):
        if positions[first] >= positions[second]:
            problems.append(f"out-of-order: expected {first} before {second}")

    return problems


def print_expected_matrix_block(expected: set[str]) -> None:
    integration_paths = sorted(
        path for path in expected if path.startswith("scripts/integration/")
    )
    if not integration_paths:
        return
    print("Suggested bash-integration matrix list:")
    for path in integration_paths:
        print(f"  - {path}")


def find_disallowed_matrix_items(matrix_items: list[str]) -> list[str]:
    disallowed = set(DISALLOWED_CI_MATRIX_SCRIPTS)
    return sorted(item for item in matrix_items if item in disallowed)


def main() -> int:
    repo_root = Path(__file__).resolve().parents[2]
    ci_content = parse_ci_content(repo_root)
    expected = parse_expected(repo_root)
    actual = parse_actual(repo_root)
    matrix_items = parse_matrix_list(ci_content)
    direct_steps = parse_direct_bash_steps(ci_content)
    strict_order = has_strict_order_flag(sys.argv[1:])

    missing = sorted(expected - actual)
    extra = sorted(actual - expected)
    duplicates = find_duplicates(matrix_items)
    disallowed_matrix_items = find_disallowed_matrix_items(matrix_items)
    order_problems = check_core_step_order(direct_steps) if strict_order else []

    if not missing and not extra and not duplicates and not disallowed_matrix_items and not order_problems:
        print("OK: CI script list is in sync with scripts/test_all_full.sh")
        if strict_order:
            print("OK: core CI step order is valid")
        return 0

    print("CI script list mismatch:")
    if missing:
        print("Missing in CI:")
        for item in missing:
            print(f"  - {item}")
    if extra:
        print("Extra in CI (not referenced by test_all_full.sh):")
        for item in extra:
            print(f"  - {item}")
    if duplicates:
        print("Duplicate scripts in CI bash-integration matrix:")
        for item in duplicates:
            print(f"  - {item}")
    if disallowed_matrix_items:
        print("Disallowed scripts found in CI bash-integration matrix:")
        for item in disallowed_matrix_items:
            print(f"  - {item}")
    if order_problems:
        print("Core CI order violations (--strict-order):")
        for problem in order_problems:
            print(f"  - {problem}")
    if missing or extra or duplicates:
        print_expected_matrix_block(expected)
    return 1


if __name__ == "__main__":
    raise SystemExit(main())
