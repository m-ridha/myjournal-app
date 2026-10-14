"""Select PR checks from the complete Git diff; unknown files run code checks."""

import os
import subprocess


def is_non_runtime_file(path):
    return (
        path.endswith(".md")
        or path.startswith("benchmarks/results/")
        or path in {".gitignore", "LICENSE", ".markdownlint-cli2.jsonc"}
        or path.endswith("/.gitkeep")
    )


def requires_code_checks(paths):
    # Only known non-runtime files may skip checks; new file types default to running.
    return any(not is_non_runtime_file(path) for path in paths)


def requires_build_checks(paths):
    # This group includes workflow/configuration validation and the container build.
    return any(not is_non_runtime_file(path) and (
        path.startswith((".github/", ".railway/")) or path in {
            "Dockerfile", ".dockerignore", "railway.json", "go.mod", "go.sum",
            "Makefile", ".env.example", "RELEASE_VERSION", "scripts/python/deploy-railway.py",
            "scripts/python/test_deploy_railway.py", "scripts/python/pr_checks.py", "scripts/python/test_pr_checks.py",
        }
    ) for path in paths)


def main():
    changed = subprocess.check_output([
        "git", "diff", "--name-only", "--no-renames", "-z",
        f"{os.environ['BASE_SHA']}...{os.environ['HEAD_SHA']}",
    ])
    paths = [os.fsdecode(path) for path in changed.split(b"\0") if path]
    code_checks = requires_code_checks(paths)
    build_checks = requires_build_checks(paths)
    with open(os.environ["GITHUB_OUTPUT"], "a") as output:
        output.write(f"run_code_checks={str(code_checks).lower()}\n")
        output.write(f"run_build_checks={str(build_checks).lower()}\n")
    with open(os.environ["GITHUB_STEP_SUMMARY"], "a") as summary:
        summary.write("Running application PR checks.\n" if code_checks
                      else "Skipping checks: only documentation/results or other non-runtime files changed.\n")

        summary.write(f"Build/configuration validation: {'enabled' if build_checks else 'skipped'}.\n")


if __name__ == "__main__":
    main()
