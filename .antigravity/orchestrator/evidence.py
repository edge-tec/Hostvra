"""
Hostvra AI Engineering — Evidence System
Records timestamped evidence for every AI task.
Evidence is NEVER deleted on failure.
"""

import json
import os
import uuid
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Dict, List, Optional


REPORTS_DIR = Path(__file__).parent.parent / "reports"


class EvidenceRecord:
    """Immutable evidence record for a single AI task."""

    def __init__(
        self,
        agent: str,
        request: str,
        task_id: Optional[str] = None,
    ):
        self.task_id = task_id or str(uuid.uuid4())[:8]
        self.agent = agent
        self.request = request
        self.timestamp_start = datetime.now(timezone.utc).isoformat()
        self.timestamp_end: Optional[str] = None
        self.plan: List[str] = []
        self.files_inspected: List[str] = []
        self.files_changed: List[Dict[str, str]] = []
        self.commands_executed: List[Dict[str, Any]] = []
        self.tests_run: List[Dict[str, Any]] = []
        self.approvals: List[Dict[str, Any]] = []
        self.findings: List[Dict[str, Any]] = []
        self.final_status: str = "IN_PROGRESS"
        self.errors: List[str] = []
        self.rollback_info: Optional[str] = None

    def set_plan(self, steps: List[str]) -> None:
        self.plan = steps

    def record_file_inspected(self, path: str) -> None:
        if path not in self.files_inspected:
            self.files_inspected.append(path)

    def record_file_changed(self, path: str, change_type: str, description: str) -> None:
        self.files_changed.append({
            "path": path,
            "change_type": change_type,  # created | modified | deleted
            "description": description,
            "timestamp": datetime.now(timezone.utc).isoformat(),
        })

    def record_command(
        self,
        command: str,
        stdout: str,
        stderr: str,
        exit_code: int,
        working_dir: str = "",
    ) -> None:
        self.commands_executed.append({
            "command": command,
            "working_dir": working_dir,
            "stdout": stdout[:4096],  # truncate large outputs
            "stderr": stderr[:4096],
            "exit_code": exit_code,
            "success": exit_code == 0,
            "timestamp": datetime.now(timezone.utc).isoformat(),
        })

    def record_test_result(
        self,
        suite: str,
        command: str,
        passed: int,
        failed: int,
        skipped: int,
        output: str,
    ) -> None:
        self.tests_run.append({
            "suite": suite,
            "command": command,
            "passed": passed,
            "failed": failed,
            "skipped": skipped,
            "output": output[:8192],
            "all_passed": failed == 0,
            "timestamp": datetime.now(timezone.utc).isoformat(),
        })

    def record_approval(
        self,
        operation: str,
        approved: bool,
        approved_by: str = "human",
        notes: str = "",
    ) -> None:
        self.approvals.append({
            "operation": operation,
            "approved": approved,
            "approved_by": approved_by,
            "notes": notes,
            "timestamp": datetime.now(timezone.utc).isoformat(),
        })

    def add_finding(
        self,
        severity: str,
        component: str,
        description: str,
        evidence: str,
        recommendation: str,
    ) -> None:
        """Add a security or quality finding."""
        self.findings.append({
            "severity": severity,
            "component": component,
            "description": description,
            "evidence": evidence,
            "recommendation": recommendation,
            "timestamp": datetime.now(timezone.utc).isoformat(),
        })

    def add_error(self, error: str) -> None:
        self.errors.append(f"[{datetime.now(timezone.utc).isoformat()}] {error}")

    def set_rollback_info(self, info: str) -> None:
        self.rollback_info = info

    def finalize(self, status: str) -> None:
        """
        Status must be one of:
        COMPLETED | COMPLETED_WITH_ERRORS | FAILED | PARTIALLY_COMPLETED | BLOCKED
        """
        valid_statuses = {
            "COMPLETED",
            "COMPLETED_WITH_ERRORS",
            "FAILED",
            "PARTIALLY_COMPLETED",
            "BLOCKED",
        }
        if status not in valid_statuses:
            raise ValueError(f"Invalid status: {status}. Must be one of {valid_statuses}")
        self.final_status = status
        self.timestamp_end = datetime.now(timezone.utc).isoformat()

    def to_dict(self) -> Dict[str, Any]:
        return {
            "task_id": self.task_id,
            "agent": self.agent,
            "request": self.request,
            "timestamp_start": self.timestamp_start,
            "timestamp_end": self.timestamp_end,
            "final_status": self.final_status,
            "plan": self.plan,
            "files_inspected": self.files_inspected,
            "files_changed": self.files_changed,
            "commands_executed": self.commands_executed,
            "tests_run": self.tests_run,
            "approvals": self.approvals,
            "findings": self.findings,
            "errors": self.errors,
            "rollback_info": self.rollback_info,
            "summary": {
                "files_inspected_count": len(self.files_inspected),
                "files_changed_count": len(self.files_changed),
                "commands_executed_count": len(self.commands_executed),
                "commands_failed_count": sum(
                    1 for c in self.commands_executed if not c["success"]
                ),
                "tests_run_count": len(self.tests_run),
                "tests_all_passed": all(t["all_passed"] for t in self.tests_run) if self.tests_run else None,
                "findings_count": len(self.findings),
                "critical_findings": sum(
                    1 for f in self.findings if f["severity"] == "CRITICAL"
                ),
                "high_findings": sum(
                    1 for f in self.findings if f["severity"] == "HIGH"
                ),
                "errors_count": len(self.errors),
                "approvals_count": len(self.approvals),
                "approvals_denied": sum(1 for a in self.approvals if not a["approved"]),
            },
        }

    def save(self) -> Path:
        """Persist evidence to disk. Never raises — always saves what it can."""
        REPORTS_DIR.mkdir(parents=True, exist_ok=True)
        date_str = datetime.now(timezone.utc).strftime("%Y-%m-%d")
        filename = f"{date_str}-{self.agent}-{self.task_id}.json"
        path = REPORTS_DIR / filename

        try:
            with open(path, "w") as f:
                json.dump(self.to_dict(), f, indent=2)
        except Exception as e:
            # Last resort: write to stderr — do not suppress
            import sys
            print(f"EVIDENCE SAVE ERROR: {e}", file=sys.stderr)
            # Try a fallback path
            fallback = Path("/tmp") / f"hostvra-evidence-{self.task_id}.json"
            with open(fallback, "w") as f:
                json.dump(self.to_dict(), f, indent=2)
            return fallback

        return path

    def save_markdown_report(self) -> Path:
        """Save a human-readable markdown audit report."""
        REPORTS_DIR.mkdir(parents=True, exist_ok=True)
        date_str = datetime.now(timezone.utc).strftime("%Y-%m-%d")
        filename = f"{date_str}-{self.agent}-{self.task_id}.md"
        path = REPORTS_DIR / filename

        lines = [
            f"# Hostvra AI Audit Report",
            f"",
            f"**Date:** {self.timestamp_start}",
            f"**Agent:** {self.agent}",
            f"**Task ID:** {self.task_id}",
            f"**Status:** {self.final_status}",
            f"",
            f"## Request",
            f"",
            f"{self.request}",
            f"",
        ]

        if self.plan:
            lines += ["## Plan", ""]
            for i, step in enumerate(self.plan, 1):
                lines.append(f"{i}. {step}")
            lines.append("")

        if self.findings:
            lines += ["## Findings", ""]
            lines += ["| Severity | Component | Description |", "|----------|-----------|-------------|"]
            for f in self.findings:
                lines.append(f"| {f['severity']} | {f['component']} | {f['description']} |")
            lines.append("")

        if self.commands_executed:
            lines += ["## Commands Executed", ""]
            for cmd in self.commands_executed:
                status = "✅" if cmd["success"] else "❌"
                lines.append(f"### {status} `{cmd['command'][:80]}`")
                lines.append(f"- Exit code: {cmd['exit_code']}")
                if cmd["stdout"]:
                    lines.append(f"- Output: `{cmd['stdout'][:200]}`")
                if cmd["stderr"] and not cmd["success"]:
                    lines.append(f"- Error: `{cmd['stderr'][:200]}`")
                lines.append("")

        if self.tests_run:
            lines += ["## Test Results", ""]
            for t in self.tests_run:
                status = "✅ PASS" if t["all_passed"] else "❌ FAIL"
                lines.append(f"### {t['suite']} — {status}")
                lines.append(f"- Passed: {t['passed']} | Failed: {t['failed']} | Skipped: {t['skipped']}")
                lines.append("")

        if self.errors:
            lines += ["## Errors", ""]
            for err in self.errors:
                lines.append(f"- {err}")
            lines.append("")

        if self.files_changed:
            lines += ["## Files Changed", ""]
            for fc in self.files_changed:
                lines.append(f"- `{fc['path']}` ({fc['change_type']}): {fc['description']}")
            lines.append("")

        if self.rollback_info:
            lines += ["## Rollback Information", "", self.rollback_info, ""]

        lines += [
            "## Summary",
            "",
            f"- Files inspected: {len(self.files_inspected)}",
            f"- Files changed: {len(self.files_changed)}",
            f"- Commands executed: {len(self.commands_executed)}",
            f"- Commands failed: {sum(1 for c in self.commands_executed if not c['success'])}",
            f"- Tests run: {len(self.tests_run)}",
            f"- Findings: {len(self.findings)}",
            f"- Errors: {len(self.errors)}",
        ]

        with open(path, "w") as f:
            f.write("\n".join(lines))

        return path


def load_evidence(task_id: str) -> Optional[Dict[str, Any]]:
    """Load a previously saved evidence record by task_id."""
    for f in REPORTS_DIR.glob(f"*-{task_id}.json"):
        with open(f) as fp:
            return json.load(fp)
    return None
