#!/usr/bin/env python3
import shutil
import subprocess
import sys
from pathlib import Path
from typing import List, Optional


def run_command(cmd: List[str], work_dir_path: Optional[Path] = None) -> None:
    """
    Run a shell command in a subprocess, raising an error if it fails.
    """
    work_dir = str(work_dir_path or Path.cwd())
    print(f"[INFO] Running: {' '.join(cmd)} (cwd={work_dir})")
    try:
        subprocess.run(cmd, check=True, cwd=work_dir)
    except subprocess.CalledProcessError as err:
        print(f"[ERROR] Command failed: {' '.join(cmd)}\n{err}")
        sys.exit(1)


def ensure_terraform_docs_installed() -> None:
    """
    Check if `terraform-docs` is in PATH. If not, optionally install it
    or instruct the user to do so.
    """
    if shutil.which("terraform-docs") is None:
        print("[WARN] `terraform-docs` not found in PATH.")
        sys.exit(1)


def get_staged_files() -> List[str]:
    """
    Returns a list of staged (added/modified) file paths.
    """
    # --cached => use staged changes; --name-only => output only file names
    try:
        output = subprocess.check_output(
            ["git", "diff", "--cached", "--name-only"]
        )
        return output.decode().split()
    except subprocess.CalledProcessError as err:
        print(f"[ERROR] Could not retrieve staged files: {err}")
        sys.exit(1)


def main() -> None:
    staged_files = get_staged_files()

    # Separate files by directory of interest:
    infra_changed = [
        Path(f) for f in staged_files if f.startswith("cluster/infra/")
    ]
    api_gateway_changed = [
        Path(f) for f in staged_files if f.startswith("docker/api_gateway/")
    ]

    # 1) If files changed in cluster/infra
    if infra_changed:
        print("[INFO] Detected changes in cluster/infra")

        # a) Run terraform fmt on each .tf and .hcl file
        tf_or_hcl_files = [
            f for f in infra_changed if f.suffix.lower() in {".tf", ".hcl"}
        ]
        for file_path in tf_or_hcl_files:
            run_command(["terraform", "fmt", str(file_path)])

        # b) For each directory containing a main.tf,
        # run terraform-docs to create/update the README.md
        # we collect directories that have main.tf,
        # then run `terraform-docs markdown . --output-file README.md`.
        ensure_terraform_docs_installed()

        # Identify unique directories in which `main.tf` exists
        # We check the *entire* cluster/infra structure, not just changed files,
        # to run docs if that folder has a main.tf.
        # You could refine this logic if needed.
        changed_dirs = set(f.parent for f in infra_changed)
        # We expand these directories upward until we find a main.tf
        dirs_with_main_tf = set()
        for directory in changed_dirs:
            if (directory / "main.tf").is_file():
                dirs_with_main_tf.add(directory)

        for directory in dirs_with_main_tf:
            run_command(
                [
                    "terraform-docs",
                    "markdown",
                    ".",
                    "--output-file",
                    "README.md",
                ],
                work_dir_path=directory,
            )

    # 2) If files changed in docker/api_gateway
    if api_gateway_changed:
        print("[INFO] Detected changes in docker/api_gateway")

        # a) Run go fmt on the .go files
        go_files = [f for f in api_gateway_changed if f.suffix.lower() == ".go"]
        if go_files:
            run_command(
                ["go", "fmt", "./..."], work_dir_path=Path("docker/api_gateway")
            )

        # b) Build the api_gateway Docker image
        run_command(
            ["docker", "compose", "build", "api_gateway"],
            work_dir_path=Path("local"),
        )

    print("[INFO] Pre-commit checks completed successfully.")
    sys.exit(0)


if __name__ == "__main__":
    main()
