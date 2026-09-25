"""
Hostvra AI Engineering — Git Tools
Read-only git operations for analysis and checkpoint management.
"""

import subprocess
import os
from pathlib import Path
from typing import Dict, List, Optional


REPO_ROOT = Path(__file__).parent.parent.parent


def _run_git(args: List[str], cwd: Optional[Path] = None) -> Dict[str, str]:
    """Run a git command and return stdout/stderr/returncode."""
    result = subprocess.run(
        ["git"] + args,
        capture_output=True,
        text=True,
        cwd=str(cwd or REPO_ROOT),
    )
    return {
        "stdout": result.stdout,
        "stderr": result.stderr,
        "returncode": result.returncode,
    }


def get_status() -> Dict[str, str]:
    """Get current git status (read-only)."""
    return _run_git(["status"])


def get_current_branch() -> str:
    """Get the current branch name."""
    result = _run_git(["branch", "--show-current"])
    return result["stdout"].strip()


def get_last_commit() -> Dict[str, str]:
    """Get the last commit info."""
    result = _run_git(["log", "-1", "--pretty=format:%H|%s|%an|%ai"])
    if result["returncode"] != 0:
        return {"error": result["stderr"]}
    parts = result["stdout"].strip().split("|", 3)
    if len(parts) >= 4:
        return {
            "hash": parts[0],
            "subject": parts[1],
            "author": parts[2],
            "date": parts[3],
        }
    return {"raw": result["stdout"]}


def get_diff(file_path: Optional[str] = None) -> str:
    """Get current uncommitted diff (read-only)."""
    args = ["diff"]
    if file_path:
        args.append(file_path)
    result = _run_git(args)
    return result["stdout"]


def get_log(n: int = 10) -> str:
    """Get recent commit log (read-only)."""
    result = _run_git(["log", f"-{n}", "--oneline", "--graph"])
    return result["stdout"]


def get_untracked_files() -> List[str]:
    """List untracked files."""
    result = _run_git(["ls-files", "--others", "--exclude-standard"])
    return [f.strip() for f in result["stdout"].splitlines() if f.strip()]


def get_modified_files() -> List[str]:
    """List modified (not staged) files."""
    result = _run_git(["diff", "--name-only"])
    return [f.strip() for f in result["stdout"].splitlines() if f.strip()]


def create_checkpoint(message: str) -> Dict[str, str]:
    """
    Create a recoverable checkpoint commit.
    Only stages files that are already tracked.
    Does NOT commit untracked files.
    Requires human approval before calling.
    """
    # Stage only tracked modified files
    stage_result = _run_git(["add", "-u"])
    if stage_result["returncode"] != 0:
        return {"error": f"git add -u failed: {stage_result['stderr']}"}

    commit_result = _run_git(["commit", "-m", f"[AI CHECKPOINT] {message}"])
    if commit_result["returncode"] != 0:
        # Nothing to commit is not an error
        if "nothing to commit" in commit_result["stdout"] + commit_result["stderr"]:
            return {"status": "no_changes", "message": "No tracked changes to checkpoint"}
        return {"error": f"git commit failed: {commit_result['stderr']}"}

    return {"status": "success", "output": commit_result["stdout"]}


def restore_file(file_path: str) -> Dict[str, str]:
    """
    Restore a single file to its last committed state.
    REQUIRES HUMAN APPROVAL before calling — this discards changes.
    """
    result = _run_git(["restore", file_path])
    return {
        "returncode": result["returncode"],
        "stdout": result["stdout"],
        "stderr": result["stderr"],
    }


def get_blame(file_path: str, start_line: int = 1, end_line: int = 10) -> str:
    """Get git blame for a file range (read-only)."""
    result = _run_git([
        "blame",
        f"-L{start_line},{end_line}",
        "--line-porcelain",
        file_path,
    ])
    return result["stdout"]


def get_repository_info() -> Dict[str, str]:
    """Get comprehensive repository status snapshot."""
    return {
        "branch": get_current_branch(),
        "last_commit": get_last_commit(),
        "modified_files": get_modified_files(),
        "untracked_files": get_untracked_files(),
        "status": get_status()["stdout"],
    }
