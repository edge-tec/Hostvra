"""
Hostvra AI Engineering — Diagnostics Tools
Read-only system diagnostics. All commands are pre-classified as SAFE.
"""

import subprocess
import sys
from pathlib import Path
from typing import Dict, List, Optional


REPO_ROOT = Path(__file__).parent.parent.parent


def _safe_run(args: List[str], timeout: int = 30) -> Dict:
    """Run a pre-classified SAFE command."""
    try:
        result = subprocess.run(
            args,
            capture_output=True,
            text=True,
            timeout=timeout,
        )
        return {
            "command": " ".join(args),
            "stdout": result.stdout,
            "stderr": result.stderr,
            "returncode": result.returncode,
            "success": result.returncode == 0,
        }
    except subprocess.TimeoutExpired:
        return {
            "command": " ".join(args),
            "stdout": "",
            "stderr": f"Command timed out after {timeout}s",
            "returncode": -1,
            "success": False,
        }
    except FileNotFoundError:
        return {
            "command": " ".join(args),
            "stdout": "",
            "stderr": f"Command not found: {args[0]}",
            "returncode": -1,
            "success": False,
        }


def get_system_info() -> Dict:
    """Collect system hardware and OS info."""
    return {
        "uname": _safe_run(["uname", "-a"])["stdout"].strip(),
        "uptime": _safe_run(["uptime"])["stdout"].strip(),
        "free": _safe_run(["free", "-m"])["stdout"].strip(),
        "disk": _safe_run(["df", "-h"])["stdout"].strip(),
        "cpu": _safe_run(["lscpu"])["stdout"].strip()[:2000],
    }


def get_go_info() -> Dict:
    """Get Go runtime information."""
    return {
        "version": _safe_run(["go", "version"])["stdout"].strip(),
        "env": _safe_run(["go", "env"])["stdout"].strip()[:2000],
    }


def get_node_info() -> Dict:
    """Get Node.js runtime information."""
    return {
        "version": _safe_run(["node", "--version"])["stdout"].strip(),
        "npm_version": _safe_run(["npm", "--version"])["stdout"].strip(),
    }


def get_python_info() -> Dict:
    """Get Python runtime information."""
    venv_python = str(REPO_ROOT / ".venv" / "bin" / "python")
    return {
        "system_python": _safe_run(["python3", "--version"])["stdout"].strip(),
        "venv_python": _safe_run([venv_python, "--version"])["stdout"].strip(),
        "venv_packages": _safe_run([venv_python, "-m", "pip", "list"])["stdout"].strip()[:3000],
    }


def get_service_status(service_name: str) -> Dict:
    """Check systemd service status (read-only)."""
    return _safe_run(["systemctl", "status", service_name, "--no-pager"])


def get_api_health() -> Dict:
    """Check if the Hostvra API is responding (read-only)."""
    result = _safe_run(["curl", "-s", "-o", "/dev/null", "-w", "%{http_code}", "http://localhost:8080/health"], timeout=5)
    return {
        "endpoint": "http://localhost:8080/health",
        "http_code": result["stdout"].strip(),
        "reachable": result["returncode"] == 0,
    }


def scan_for_secrets_patterns() -> Dict:
    """
    Read-only scan for potential secret patterns in code.
    This is a diagnostic — does NOT modify any files.
    """
    patterns = [
        # Hardcoded passwords
        (["grep", "-rn", "--include=*.go", r'password\s*=\s*"[^$][^"]{3,}"', str(REPO_ROOT / "apps")], "Hardcoded password strings"),
        # Hardcoded API keys
        (["grep", "-rn", "--include=*.go", r'api_key\s*=\s*"[^$][^"]{8,}"', str(REPO_ROOT / "apps")], "Hardcoded API key strings"),
        # Private key markers
        (["grep", "-rn", "--include=*.go", "BEGIN PRIVATE KEY", str(REPO_ROOT / "apps")], "Private key in source"),
        # fmt.Sprintf SQL injection patterns
        (["grep", "-rn", "--include=*.go", r'fmt\.Sprintf.*SELECT\|fmt\.Sprintf.*DELETE\|fmt\.Sprintf.*UPDATE', str(REPO_ROOT / "apps")], "Potential SQL injection via fmt.Sprintf"),
    ]

    findings = []
    for args, description in patterns:
        result = _safe_run(args)
        if result["stdout"].strip():
            findings.append({
                "description": description,
                "matches": result["stdout"].strip()[:2000],
            })

    return {
        "scan_complete": True,
        "findings_count": len(findings),
        "findings": findings,
    }


def get_repo_file_counts() -> Dict:
    """Count files by type in the repository."""
    go_files = _safe_run(["find", str(REPO_ROOT / "apps"), "-name", "*.go", "-not", "-path", "*/vendor/*"])
    test_files = _safe_run(["find", str(REPO_ROOT / "apps"), "-name", "*_test.go"])
    ts_files = _safe_run(["find", str(REPO_ROOT / "apps" / "web"), "-name", "*.tsx", "-o", "-name", "*.ts"])

    return {
        "go_files": len(go_files["stdout"].splitlines()),
        "go_test_files": len(test_files["stdout"].splitlines()),
        "typescript_files": len(ts_files["stdout"].splitlines()),
    }


def check_gitignore() -> Dict:
    """Verify .gitignore covers sensitive files."""
    gitignore_path = REPO_ROOT / ".gitignore"
    if not gitignore_path.exists():
        return {"exists": False, "coverage": []}

    content = gitignore_path.read_text()
    checks = {
        ".env": ".env" in content,
        "*.env": "*.env" in content or ".env" in content,
        ".env.local": ".env.local" in content or ".env*" in content,
        "node_modules": "node_modules" in content,
        ".venv": ".venv" in content or "venv" in content,
        "bin/": "bin/" in content or "/bin/" in content,
    }

    return {
        "exists": True,
        "content_preview": content[:500],
        "coverage": checks,
        "missing": [k for k, v in checks.items() if not v],
    }


def run_go_build_check(module: str = "api") -> Dict:
    """
    Verify the Go code compiles without errors.
    Uses go build -o /dev/null to avoid producing artifacts.
    """
    cwd = REPO_ROOT / "apps" / module
    if module == "api":
        cmd = ["go", "build", "-o", "/dev/null", "./cmd/server/"]
    else:
        cmd = ["go", "build", "-o", "/dev/null", "./cmd/agent/"]

    result = subprocess.run(
        cmd,
        capture_output=True,
        text=True,
        cwd=str(cwd),
        timeout=60,
    )

    return {
        "module": module,
        "command": " ".join(cmd),
        "compiles": result.returncode == 0,
        "stderr": result.stderr,
    }


if __name__ == "__main__":
    print("Hostvra Diagnostics")
    print("=" * 50)
    info = get_system_info()
    print(f"System: {info['uname']}")
    print(f"Uptime: {info['uptime']}")
    go_info = get_go_info()
    print(f"Go: {go_info['version']}")
    node_info = get_node_info()
    print(f"Node: {node_info['version']}")
    python_info = get_python_info()
    print(f"Python (system): {python_info['system_python']}")
    print(f"Python (venv): {python_info['venv_python']}")
    print("=" * 50)
    print("Build checks:")
    api_build = run_go_build_check("api")
    print(f"  API builds: {'✅' if api_build['compiles'] else '❌'}")
    if not api_build["compiles"]:
        print(f"    {api_build['stderr'][:200]}")
