"""
Hostvra AI Engineering — Human Approval System
Blocks sensitive operations until explicit human approval is received.
NEVER silently performs high-risk changes.
"""

import sys
from dataclasses import dataclass, field
from typing import List, Optional


@dataclass
class ApprovalRequest:
    operation: str
    reason: str
    risk_level: str  # LOW | MEDIUM | HIGH | CRITICAL
    affected_files: List[str] = field(default_factory=list)
    affected_services: List[str] = field(default_factory=list)
    affected_db_objects: List[str] = field(default_factory=list)
    commands: List[str] = field(default_factory=list)
    expected_impact: str = ""
    rollback_plan: str = ""
    tests_to_run: List[str] = field(default_factory=list)


class ApprovalError(Exception):
    """Raised when an operation is denied or times out."""
    pass


def _format_approval_display(req: ApprovalRequest) -> str:
    """Format the approval request for human review."""
    lines = [
        "=" * 70,
        "  HOSTVRA AI ENGINEERING — OPERATION APPROVAL REQUIRED",
        "=" * 70,
        "",
        f"  Operation:   {req.operation}",
        f"  Risk Level:  {req.risk_level}",
        f"  Reason:      {req.reason}",
        "",
    ]

    if req.affected_files:
        lines.append("  Affected Files:")
        for f in req.affected_files:
            lines.append(f"    • {f}")
        lines.append("")

    if req.affected_services:
        lines.append("  Affected Services:")
        for s in req.affected_services:
            lines.append(f"    • {s}")
        lines.append("")

    if req.affected_db_objects:
        lines.append("  Affected Database Objects:")
        for d in req.affected_db_objects:
            lines.append(f"    • {d}")
        lines.append("")

    if req.commands:
        lines.append("  Commands to Execute:")
        for c in req.commands:
            lines.append(f"    $ {c}")
        lines.append("")

    if req.expected_impact:
        lines.append(f"  Expected Impact: {req.expected_impact}")
        lines.append("")

    if req.rollback_plan:
        lines.append(f"  Rollback Plan: {req.rollback_plan}")
        lines.append("")

    if req.tests_to_run:
        lines.append("  Tests to Run After Change:")
        for t in req.tests_to_run:
            lines.append(f"    • {t}")
        lines.append("")

    lines += [
        "=" * 70,
        "",
    ]

    return "\n".join(lines)


def request_approval(req: ApprovalRequest, auto_approve_low_risk: bool = False) -> bool:
    """
    Request human approval for a sensitive operation.
    
    Returns True if approved, False if denied.
    Raises ApprovalError if the operation should be aborted immediately.
    
    CRITICAL and HIGH risk operations always require explicit approval.
    LOW risk operations can be auto-approved if auto_approve_low_risk=True.
    """
    # Auto-approve low risk if permitted (e.g. read-only diagnostics)
    if auto_approve_low_risk and req.risk_level == "LOW":
        return True

    # CRITICAL risk: always require explicit approval, never auto-approve
    if req.risk_level == "CRITICAL":
        print("\n⚠️  CRITICAL RISK OPERATION — MANUAL APPROVAL REQUIRED", file=sys.stderr)

    display = _format_approval_display(req)
    print(display)

    while True:
        try:
            response = input("  Approve this operation? [yes/no/abort]: ").strip().lower()
        except (EOFError, KeyboardInterrupt):
            print("\nOperation cancelled by user (EOF/interrupt).")
            raise ApprovalError(f"Operation '{req.operation}' cancelled by user interrupt")

        if response in ("yes", "y"):
            print(f"  ✅ Approved: {req.operation}")
            return True
        elif response in ("no", "n"):
            print(f"  ❌ Denied: {req.operation}")
            return False
        elif response == "abort":
            raise ApprovalError(f"Operation '{req.operation}' aborted by user request")
        else:
            print("  Please enter 'yes', 'no', or 'abort'")


def require_approval(req: ApprovalRequest) -> None:
    """
    Same as request_approval but raises ApprovalError if denied.
    Use this when the operation CANNOT safely proceed without approval.
    """
    approved = request_approval(req)
    if not approved:
        raise ApprovalError(
            f"Operation '{req.operation}' was denied. "
            "No changes were made."
        )


# Pre-built approval templates for common Hostvra operations

def approval_modify_go_file(file_path: str, description: str, tests: List[str]) -> ApprovalRequest:
    return ApprovalRequest(
        operation=f"Modify Go source file: {file_path}",
        reason=description,
        risk_level="MEDIUM",
        affected_files=[file_path],
        affected_services=["hostvra-api"],
        rollback_plan=f"git restore {file_path}",
        tests_to_run=tests,
    )


def approval_run_database_migration(migration_file: str) -> ApprovalRequest:
    return ApprovalRequest(
        operation=f"Apply database migration: {migration_file}",
        reason="Schema change to production PostgreSQL",
        risk_level="HIGH",
        affected_db_objects=["PostgreSQL schema"],
        affected_files=[migration_file],
        rollback_plan="Migrations are NOT automatically reversible. Backup required before applying.",
        expected_impact="Database schema will be permanently modified.",
        tests_to_run=["go test -count=1 ./... (apps/api)"],
    )


def approval_restart_service(service_name: str) -> ApprovalRequest:
    return ApprovalRequest(
        operation=f"Restart service: {service_name}",
        reason="Service restart required to apply configuration changes",
        risk_level="MEDIUM",
        affected_services=[service_name],
        commands=[f"systemctl restart {service_name}"],
        rollback_plan=f"systemctl start {service_name} (if it stops)",
        expected_impact="Brief service interruption during restart",
    )


def approval_execute_shell_command(command: str, reason: str, risk: str = "MEDIUM") -> ApprovalRequest:
    return ApprovalRequest(
        operation=f"Execute shell command",
        reason=reason,
        risk_level=risk,
        commands=[command],
        rollback_plan="Depends on command — document rollback before approving",
    )


def approval_delete_production_data(
    table: str,
    where_clause: str,
    estimated_rows: int,
) -> ApprovalRequest:
    return ApprovalRequest(
        operation=f"DELETE from {table}",
        reason=f"Deleting {estimated_rows} rows matching: {where_clause}",
        risk_level="CRITICAL",
        affected_db_objects=[table],
        commands=[f"DELETE FROM {table} WHERE {where_clause}"],
        rollback_plan="NO AUTOMATIC ROLLBACK. Require database backup before proceeding.",
        expected_impact=f"Permanently deletes ~{estimated_rows} rows from {table}",
    )
