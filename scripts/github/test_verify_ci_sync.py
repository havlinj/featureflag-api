#!/usr/bin/env python3
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import verify_ci_sync


TEST_ALL_FULL_WITH_INTEGRATION_GLOB = """#!/usr/bin/env bash
"$SCRIPT_DIR/check.sh"
"$SCRIPT_DIR/test_unit.sh"
"$SCRIPT_DIR/test_integration.sh"
"$SCRIPT_DIR/coverage/test_coverage.sh"
"$SCRIPT_DIR/build.sh"
"$SCRIPT_DIR/test_binary_smoke.sh"
for test_script in "$SCRIPT_DIR"/integration/*.sh; do
  "$test_script"
done
"""

INTEGRATION_SCRIPTS = (
    "test_missing_jwt_secret.sh",
    "test_weak_jwt_secret.sh",
    "test_invalid_dsn.sh",
    "test_default_listen_addr.sh",
    "test_tls_config.sh",
    "test_invalid_tls_files.sh",
)


class VerifyCISyncTests(unittest.TestCase):
    @staticmethod
    def fixtures_dir() -> Path:
        return Path(__file__).resolve().parent / "fixtures"

    def fixture_text(self, filename: str) -> str:
        return (self.fixtures_dir() / filename).read_text(encoding="utf-8")

    def write_repo_files(self, root: Path, test_all_full: str, ci_yml: str) -> None:
        scripts_dir = root / "scripts"
        workflows_dir = root / ".github" / "workflows"
        integration_dir = scripts_dir / "integration"
        integration_dir.mkdir(parents=True, exist_ok=True)
        workflows_dir.mkdir(parents=True, exist_ok=True)
        (scripts_dir / "test_all_full.sh").write_text(test_all_full, encoding="utf-8")
        (workflows_dir / "ci.yml").write_text(ci_yml, encoding="utf-8")

    def create_integration_scripts(self, root: Path) -> None:
        integration_dir = root / "scripts" / "integration"
        for name in INTEGRATION_SCRIPTS:
            (integration_dir / name).write_text(
                "#!/usr/bin/env bash\n", encoding="utf-8"
            )

    def test_parse_expected_includes_direct_and_globbed_scripts(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            test_all_full = """#!/usr/bin/env bash
"$SCRIPT_DIR/check.sh"
"$SCRIPT_DIR/test_unit.sh"
for test_script in "$SCRIPT_DIR"/integration/*.sh; do
  "$test_script"
done
"""
            ci_yml = "name: CI\n"
            self.write_repo_files(root, test_all_full, ci_yml)
            (root / "scripts" / "integration" / "a.sh").write_text(
                "#!/usr/bin/env bash\n", encoding="utf-8"
            )
            (root / "scripts" / "integration" / "b.sh").write_text(
                "#!/usr/bin/env bash\n", encoding="utf-8"
            )

            expected = verify_ci_sync.parse_expected(root)

            self.assertIn("scripts/check.sh", expected)
            self.assertIn("scripts/test_unit.sh", expected)
            self.assertIn("scripts/integration/a.sh", expected)
            self.assertIn("scripts/integration/b.sh", expected)

    def test_parse_actual_reads_direct_and_matrix_scripts(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            test_all_full = "#!/usr/bin/env bash\n"
            ci_yml = """jobs:
  test:
    steps:
      - run: bash scripts/check.sh
  bash-integration:
    strategy:
      matrix:
        script:
          - scripts/integration/test_a.sh
"""
            self.write_repo_files(root, test_all_full, ci_yml)

            actual = verify_ci_sync.parse_actual(root)

            self.assertEqual(
                actual,
                {
                    "scripts/check.sh",
                    "scripts/integration/test_a.sh",
                },
            )

    def test_check_core_step_order_reports_out_of_order(self) -> None:
        direct_steps = [
            "scripts/test_unit.sh",
            "scripts/check.sh",
            "scripts/test_integration.sh",
            "scripts/coverage/test_coverage.sh",
            "scripts/build.sh",
            "scripts/test_binary_smoke.sh",
        ]

        issues = verify_ci_sync.check_core_step_order(direct_steps)

        self.assertTrue(any("out-of-order" in issue for issue in issues))

    def test_check_core_step_order_reports_missing_steps(self) -> None:
        direct_steps = [
            "scripts/check.sh",
            "scripts/test_unit.sh",
        ]

        issues = verify_ci_sync.check_core_step_order(direct_steps)

        self.assertTrue(any("missing core CI steps" in issue for issue in issues))
        self.assertTrue(
            any("scripts/test_binary_smoke.sh" in issue for issue in issues)
        )

    def test_find_duplicates(self) -> None:
        duplicates = verify_ci_sync.find_duplicates(
            [
                "scripts/integration/test_a.sh",
                "scripts/integration/test_b.sh",
                "scripts/integration/test_a.sh",
            ]
        )
        self.assertEqual(duplicates, ["scripts/integration/test_a.sh"])

    def test_fixture_valid_ci_has_no_sync_diff_or_duplicates(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            ci_yml = self.fixture_text("ci_matrix_valid.yml")
            self.write_repo_files(root, TEST_ALL_FULL_WITH_INTEGRATION_GLOB, ci_yml)
            self.create_integration_scripts(root)

            expected = verify_ci_sync.parse_expected(root)
            actual = verify_ci_sync.parse_actual(root)
            matrix_items = verify_ci_sync.parse_matrix_list(ci_yml)
            duplicates = verify_ci_sync.find_duplicates(matrix_items)

            self.assertEqual(sorted(expected), sorted(actual))
            self.assertEqual(duplicates, [])

    def test_fixture_duplicate_matrix_detected(self) -> None:
        ci_yml = self.fixture_text("ci_matrix_duplicate.yml")
        matrix_items = verify_ci_sync.parse_matrix_list(ci_yml)
        duplicates = verify_ci_sync.find_duplicates(matrix_items)

        self.assertEqual(duplicates, ["scripts/integration/test_invalid_dsn.sh"])

    def test_fixture_missing_matrix_entry_detected(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            ci_yml = self.fixture_text("ci_matrix_missing_entry.yml")
            self.write_repo_files(root, TEST_ALL_FULL_WITH_INTEGRATION_GLOB, ci_yml)
            self.create_integration_scripts(root)

            expected = verify_ci_sync.parse_expected(root)
            actual = verify_ci_sync.parse_actual(root)
            missing = sorted(expected - actual)

            self.assertEqual(missing, ["scripts/integration/test_invalid_tls_files.sh"])

    def test_fixture_out_of_order_core_steps_detected(self) -> None:
        ci_yml = self.fixture_text("ci_core_out_of_order.yml")
        direct_steps = verify_ci_sync.parse_direct_bash_steps(ci_yml)
        issues = verify_ci_sync.check_core_step_order(direct_steps)

        self.assertTrue(any("out-of-order" in issue for issue in issues))

    def test_fixture_disallowed_local_formatter_in_matrix_detected(self) -> None:
        ci_yml = self.fixture_text("ci_matrix_with_local_formatter.yml")
        matrix_items = verify_ci_sync.parse_matrix_list(ci_yml)
        disallowed = verify_ci_sync.find_disallowed_matrix_items(matrix_items)

        self.assertEqual(disallowed, ["scripts/format_python.sh"])


if __name__ == "__main__":
    unittest.main()
