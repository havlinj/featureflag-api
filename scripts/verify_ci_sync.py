#!/usr/bin/env python3
"""
Verify that CI workflow runs all scripts referenced by scripts/test_all_full.sh.
"""

from __future__ import annotations

import re
import sys
from pathlib import Path


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
    matrix_items = set(re.findall(r"^\s*-\s*(scripts/[A-Za-z0-9_./-]+\.sh)\s*$", ci, flags=re.MULTILINE))
    return direct | matrix_items


def main() -> int:
    repo_root = Path(__file__).resolve().parents[1]
    expected = parse_expected(repo_root)
    actual = parse_actual(repo_root)

    missing = sorted(expected - actual)
    extra = sorted(actual - expected)

    if not missing and not extra:
        print("OK: CI script list is in sync with scripts/test_all_full.sh")
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
    return 1


if __name__ == "__main__":
    raise SystemExit(main())

