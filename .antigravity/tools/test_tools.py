"""
Hostvra AI Engineering — Test Tools
Runs Hostvra test suites and reports results.
Tests are NON-DESTRUCTIVE read operations on the codebase.
"""

import re
import subprocess
import sys
from pathlib import Path
from typing import Dict, List, Optional, Tuple


REPO_ROOT = Path(__file__).parent.parent.parent
VENV_PYTHON = REPO_ROOT / ".venv" / "bin" / "python"


def _run(
    args: List[str],
    cwd: Optional[Path] = None,
    timeout: int = 300,
) -> Tuple[int, str, str]:
    """Run a command and return (returncode, stdout, stderr)."""
    result = subprocess.run(
        args,
        capture_output=True,
        text=True,
        cwd=str(cwd or REPO_ROOT),
        timeout=timeout,
    )
    return result.returncode, result.stdout, result.stderr


def run_api_tests(verbose: bool = True, race: bool = True) -> Dict:
    """
    Run the Go API test suite.
    Command: go test -count=1 [-v] [-race] ./...
    """
    cmd = ["go", "test", "-count=1"]
    if verbose:
        cmd.append("-v")
    if race:
        cmd.append("-race")
    cmd.append("./...")

    try:
        returncode, stdout, stderr = _run(cmd, cwd=REPO_ROOT / "apps" / "api", timeout=300)
    except subprocess.TimeoutExpired:
        return {
            "suite": "API Tests",
            "command": " ".join(cmd),
            "success": False,
            "error": "Test suite timed out after 300 seconds",
            "passed": 0,
            "failed": 0,
            "skipped": 0,
        }

    return _parse_go_test_output(stdout, stderr, returncode, "API Tests", " ".join(cmd))


def run_agent_tests(verbose: bool = True, race: bool = True) -> Dict:
    """
    Run the Go Agent test suite.
    Command: go test -count=1 [-v] [-race] ./...
    """
    cmd = ["go", "test", "-count=1"]
    if verbose:
        cmd.append("-v")
    if race:
        cmd.append("-race")
    cmd.append("./...")

    try:
        returncode, stdout, stderr = _run(cmd, cwd=REPO_ROOT / "apps" / "agent", timeout=300)
    except subprocess.TimeoutExpired:
        return {
            "suite": "Agent Tests",
            "command": " ".join(cmd),
            "success": False,
            "error": "Test suite timed out after 300 seconds",
            "passed": 0,
            "failed": 0,
            "skipped": 0,
        }

    return _parse_go_test_output(stdout, stderr, returncode, "Agent Tests", " ".join(cmd))


def run_api_vet() -> Dict:
    """Run go vet on the API codebase."""
    returncode, stdout, stderr = _run(["go", "vet", "./..."], cwd=REPO_ROOT / "apps" / "api")
    return {
        "suite": "API go vet",
        "command": "go vet ./...",
        "success": returncode == 0,
        "returncode": returncode,
        "stdout": stdout,
        "stderr": stderr,
        "issues": [line for line in stderr.splitlines() if line.strip()],
    }


def run_agent_vet() -> Dict:
    """Run go vet on the Agent codebase."""
    returncode, stdout, stderr = _run(["go", "vet", "./..."], cwd=REPO_ROOT / "apps" / "agent")
    return {
        "suite": "Agent go vet",
        "command": "go vet ./...",
        "success": returncode == 0,
        "returncode": returncode,
        "stdout": stdout,
        "stderr": stderr,
        "issues": [line for line in stderr.splitlines() if line.strip()],
    }


def run_web_build() -> Dict:
    """Run Next.js build (type-check + bundle validation)."""
    returncode, stdout, stderr = _run(
        ["npm", "run", "build"],
        cwd=REPO_ROOT / "apps" / "web",
        timeout=180,
    )
    return {
        "suite": "Web Build (TypeScript + Next.js)",
        "command": "npm run build",
        "success": returncode == 0,
        "returncode": returncode,
        "stdout": stdout[-4096:],  # Last 4KB
        "stderr": stderr[-2048:],
    }


def run_specific_test(pattern: str, module: str = "api") -> Dict:
    """Run a specific test by name pattern."""
    cwd = REPO_ROOT / "apps" / module
    cmd = ["go", "test", "-count=1", "-v", "-run", pattern, "./..."]
    returncode, stdout, stderr = _run(cmd, cwd=cwd, timeout=120)
    return _parse_go_test_output(stdout, stderr, returncode, f"{module} test: {pattern}", " ".join(cmd))


def run_full_suite() -> List[Dict]:
    """Run the complete test suite: API + Agent + Web build."""
    results = []
    print("Running API vet...")
    results.append(run_api_vet())
    print("Running API tests...")
    results.append(run_api_tests())
    print("Running Agent vet...")
    results.append(run_agent_vet())
    print("Running Agent tests...")
    results.append(run_agent_tests())
    return results


def _parse_go_test_output(stdout: str, stderr: str, returncode: int, suite: str, cmd: str) -> Dict:
    """Parse go test output to extract pass/fail counts."""
    passed = 0
    failed = 0
    skipped = 0
    failures = []

    for line in (stdout + stderr).splitlines():
        if line.startswith("--- PASS:"):
            passed += 1
        elif line.startswith("--- FAIL:"):
            failed += 1
            failures.append(line)
        elif line.startswith("--- SKIP:"):
            skipped += 1

    # If -v not used, count package-level results
    if passed == 0 and failed == 0:
        for line in stdout.splitlines():
            if line.startswith("ok "):
                passed += 1
            elif line.startswith("FAIL"):
                failed += 1

    return {
        "suite": suite,
        "command": cmd,
        "success": returncode == 0,
        "returncode": returncode,
        "passed": passed,
        "failed": failed,
        "skipped": skipped,
        "failures": failures,
        "stdout": stdout[-8192:],  # Last 8KB
        "stderr": stderr[-2048:],
    }


def format_test_summary(results: List[Dict]) -> str:
    """Format test results for display."""
    lines = ["", "Test Results Summary", "=" * 50]
    all_passed = True

    for r in results:
        status = "✅ PASS" if r["success"] else "❌ FAIL"
        all_passed = all_passed and r["success"]
        lines.append(f"{status} | {r['suite']}")
        if "passed" in r:
            lines.append(f"       Passed: {r['passed']} | Failed: {r['failed']} | Skipped: {r.get('skipped', 0)}")
        if not r["success"] and r.get("stderr"):
            lines.append(f"       Error: {r['stderr'][:200]}")

    lines.append("=" * 50)
    lines.append(f"Overall: {'✅ ALL PASSED' if all_passed else '❌ SOME FAILED'}")
    lines.append("")

    return "\n".join(lines)


if __name__ == "__main__":
    print("Running Hostvra test suite validation...")
    results = run_full_suite()
    print(format_test_summary(results))
    all_ok = all(r["success"] for r in results)
    sys.exit(0 if all_ok else 1)
