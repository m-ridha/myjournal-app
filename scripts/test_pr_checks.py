"""Regression coverage for CI routing policy."""

import os
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

from pr_checks import main, requires_build_checks, requires_code_checks


class ChangeSelectionTests(unittest.TestCase):
    def test_paths(self):
        cases = {
            "README.md": False,
            "benchmarks/results/run/raw.json": False,
            ".gitignore": False,
            "internal/core/journal/service.go": True,
            "go.sum": True,
            "assets/templates/pages/editor.html": True,
            "assets/locales/id.json": True,
            "assets/static/css/app.css": True,
            "assets/static/js/app.js": True,
            "tests/editor.test.js": True,
            ".golangci.yml": True,
            "scripts/check-architecture.sh": True,
            "scripts/deploy-railway.py": True,
            ".github/workflows/pr-checks.yml": True,
            ".github/workflows/release.yml": True,
            ".github/dependabot.yml": True,
            "Dockerfile": True,
            ".dockerignore": True,
            "railway.json": True,
            "Makefile": True,
            "scripts/pr_checks.py": True,
            "new-language.rs": True,
        }
        for path, expected in cases.items():
            with self.subTest(path=path):
                self.assertEqual(requires_code_checks([path]), expected)

    def test_mixed_changes_run_checks(self):
        self.assertEqual(requires_code_checks(["README.md", "app.go", "assets/static/js/app.js"]), True)

    def test_deleted_or_renamed_code_still_runs(self):
        # git diff --no-renames supplies the deleted old path as well as the new path.
        self.assertEqual(requires_code_checks(["old.go", "old.md"]), True)

    def test_empty_diff(self):
        self.assertEqual(requires_code_checks([]), False)


class BuildSelectionTests(unittest.TestCase):
    def test_build_inputs(self):
        for path in ["Dockerfile", ".dockerignore", "railway.json", "go.mod", "go.sum",
                     "Makefile", ".env.example", "RELEASE_VERSION", ".github/dependabot.yml",
                     ".github/workflows/release.yml", "scripts/deploy-railway.py",
                     "scripts/test_deploy_railway.py", "scripts/pr_checks.py", "scripts/test_pr_checks.py"]:
            with self.subTest(path=path):
                self.assertTrue(requires_build_checks([path]))

    def test_feature_and_documentation_changes(self):
        for path in ["internal/core/journal/service.go", "assets/static/js/app.js",
                     "assets/templates/pages/editor.html", "README.md", ".github/README.md",
                     "benchmarks/results/run/raw.json", ".gitignore"]:
            with self.subTest(path=path):
                self.assertFalse(requires_build_checks([path]))

    def test_mixed_and_renamed_build_files(self):
        self.assertTrue(requires_build_checks(["README.md", "Dockerfile"]))
        self.assertFalse(requires_build_checks([]))


class WorkflowOutputTests(unittest.TestCase):
    def test_outputs_match_workflow_flags(self):
        cases = [
            (b"README.md\0.github/README.md\0", "false", "false"),
            (b"assets/static/js/app.js\0", "true", "false"),
            (b"Dockerfile\0", "true", "true"),
            (b"old.go\0old.md\0", "true", "false"),
        ]
        for diff, code, build in cases:
            with self.subTest(diff=diff), tempfile.TemporaryDirectory() as folder:
                output = Path(folder) / "output"
                summary = Path(folder) / "summary"
                env = {"BASE_SHA": "base", "HEAD_SHA": "head",
                       "GITHUB_OUTPUT": str(output), "GITHUB_STEP_SUMMARY": str(summary)}
                with patch.dict(os.environ, env), patch("pr_checks.subprocess.check_output", return_value=diff) as git:
                    main()
                git.assert_called_once_with([
                    "git", "diff", "--name-only", "--no-renames", "-z", "base...head",
                ])
                self.assertEqual(output.read_text(),
                                 f"run_code_checks={code}\nrun_build_checks={build}\n")
                self.assertTrue(summary.read_text())


class GitDiffIntegrationTests(unittest.TestCase):
    def test_complete_pr_and_rename(self):
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)

            def git(*args):
                return subprocess.check_output(
                    ["git", "-C", folder, *args], stderr=subprocess.DEVNULL,
                ).decode().strip()

            git("init")
            git("config", "user.name", "CI fixture")
            git("config", "user.email", "fixture@example.invalid")
            (root / "entry.go").write_text("package entry\n")
            git("add", ".")
            git("commit", "-m", "base")
            base = git("rev-parse", "HEAD")
            git("mv", "entry.go", "entry.md")
            git("commit", "-m", "rename code to docs")
            (root / "README.md").write_text("docs\n")
            git("add", ".")
            git("commit", "-m", "docs-only latest commit")
            output = root / "output"
            env = os.environ | {
                "BASE_SHA": base, "HEAD_SHA": git("rev-parse", "HEAD"),
                "GITHUB_OUTPUT": str(output), "GITHUB_STEP_SUMMARY": str(root / "summary"),
            }
            subprocess.run(
                ["python3", "-B", str(Path(__file__).with_name("pr_checks.py").resolve())],
                cwd=folder, env=env, check=True,
            )
            self.assertEqual(output.read_text(),
                             "run_code_checks=true\nrun_build_checks=false\n")
