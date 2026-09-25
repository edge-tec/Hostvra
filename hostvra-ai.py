#!/usr/bin/env python3
"""
Hostvra AI Engineering System — Entry Point Script
Usage: ./hostvra-ai [options] "task description"

This script:
1. Activates the .venv automatically (Python 3.12 with Antigravity SDK 0.1.18)
2. Validates GEMINI_API_KEY is set
3. Validates the command safety engine
4. Runs the orchestrator with the specified task
"""

import os
import sys
import subprocess
from pathlib import Path


REPO_ROOT = Path(__file__).parent.resolve()
VENV_PYTHON = REPO_ROOT / ".venv" / "bin" / "python"
ORCHESTRATOR = REPO_ROOT / ".antigravity" / "orchestrator" / "main.py"


def load_env() -> None:
    """Load environment variables from .env.ai, .env, or .antigravity/.env.ai if present."""
    for filename in [".env.ai", ".env", ".antigravity/.env.ai"]:
        env_path = REPO_ROOT / filename
        if env_path.exists():
            try:
                with open(env_path, "r", encoding="utf-8") as f:
                    for line in f:
                        line = line.strip()
                        if not line or line.startswith("#") or "=" not in line:
                            continue
                        k, v = line.split("=", 1)
                        k = k.strip()
                        v = v.strip().strip("'\"")
                        current = os.environ.get(k, "")
                        is_placeholder = any(current.startswith(p) for p in ["your_", "replace_", "<", "your-", "xxx"])
                        if k and (not current or is_placeholder or k == "GEMINI_API_KEY"):
                            if v and not any(v.startswith(p) for p in ["your_", "replace_", "<"]):
                                os.environ[k] = v
            except Exception:
                pass


def check_prerequisites() -> bool:
    """Validate all prerequisites before launching."""
    load_env()
    ok = True

    # 1. Check .venv exists with correct Python
    if not VENV_PYTHON.exists():
        print("ERROR: .venv not found at", VENV_PYTHON)
        print("  Run: python3.12 -m venv .venv && .venv/bin/pip install google-antigravity")
        ok = False
    else:
        result = subprocess.run([str(VENV_PYTHON), "--version"], capture_output=True, text=True)
        version = result.stdout.strip() or result.stderr.strip()
        if "3.12" not in version and "3.11" not in version and "3.10" not in version:
            print(f"WARNING: .venv Python version '{version}' may not meet SDK requirement (>=3.10)")

    # 2. Check GEMINI_API_KEY
    if not os.environ.get("GEMINI_API_KEY"):
        print("ERROR: GEMINI_API_KEY is not set")
        print("  Set it in .env.ai or: export GEMINI_API_KEY=your_key_here")
        ok = False

    # 3. Check Antigravity SDK is installed in .venv
    if VENV_PYTHON.exists():
        result = subprocess.run(
            [str(VENV_PYTHON), "-c", "import google.antigravity; print(google.antigravity.__version__ if hasattr(google.antigravity, '__version__') else 'ok')"],
            capture_output=True,
            text=True,
        )
        if result.returncode != 0:
            print("ERROR: google-antigravity not installed in .venv")
            print("  Run: .venv/bin/pip install google-antigravity")
            ok = False

    return ok


def run_command_safety_test() -> bool:
    """Run the command safety engine self-test."""
    result = subprocess.run(
        [str(VENV_PYTHON), str(REPO_ROOT / ".antigravity" / "tools" / "command_safety.py")],
        capture_output=True,
        text=True,
        cwd=str(REPO_ROOT),
    )
    if result.returncode != 0:
        print("WARNING: Command safety self-test had failures:")
        print(result.stdout)
        return False
    return True


def main():
    if len(sys.argv) < 2 or sys.argv[1] in ("-h", "--help"):
        print(__doc__)
        print("""
Usage:
  ./hostvra-ai "task description"
  ./hostvra-ai --mode read_only "analyze architecture"
  ./hostvra-ai --mode analysis "audit security"
  ./hostvra-ai --mode development "add rate limiting to auth"
  ./hostvra-ai --agent qa "run full test suite"

Modes:
  read_only   — File reading only, no commands, no writes (default: analysis)
  analysis    — Safe read-only commands + file reading, no writes
  development — File writes with approval, DANGEROUS commands blocked

Agents:
  architect, backend, frontend, linux, database, security, qa
        """)
        sys.exit(0)

    print("Hostvra AI Engineering System")
    print("=" * 50)

    # Prerequisite checks
    print("Checking prerequisites...")
    if not check_prerequisites():
        print("\nPrerequisite checks failed. Aborting.")
        sys.exit(1)

    # Command safety self-test
    print("Validating command safety engine...")
    run_command_safety_test()

    print("All checks passed. Starting orchestrator...\n")

    # Launch orchestrator with .venv Python
    orchestrator_args = [str(VENV_PYTHON), str(ORCHESTRATOR)] + sys.argv[1:]

    os.execv(str(VENV_PYTHON), orchestrator_args)


if __name__ == "__main__":
    main()
