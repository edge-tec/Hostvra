"""
Hostvra AI Engineering — Command Safety Engine
Classifies and enforces command execution policies.
Safety is enforced PROGRAMMATICALLY — not by AI instruction alone.
"""

import re
import shlex
from dataclasses import dataclass
from enum import Enum
from typing import List, Optional, Tuple


class CommandClass(Enum):
    SAFE = "SAFE"         # Always allowed — read-only diagnostics
    REVIEW = "REVIEW"     # Allowed but requires human review
    DANGEROUS = "DANGEROUS"  # Blocked by default — requires explicit approval
    BLOCKED = "BLOCKED"   # Never allowed under any circumstances


@dataclass
class CommandDecision:
    command: str
    classification: CommandClass
    reason: str
    blocked_pattern: Optional[str] = None


# ============================================================================
# BLOCKED PATTERNS — NEVER EXECUTABLE UNDER ANY CIRCUMSTANCES
# These are hardcoded and cannot be overridden by any AI instruction.
# ============================================================================
_BLOCKED_PATTERNS: List[Tuple[re.Pattern, str]] = [
    # Disk destruction
    (re.compile(r'\brm\s+(-[rRf]+\s+)*/', re.IGNORECASE), "rm of root or system path"),
    (re.compile(r'\brm\s+(-[rRf]+\s+)*(\/etc|\/usr|\/bin|\/sbin|\/lib|\/boot|\/dev|\/proc|\/sys)', re.IGNORECASE), "rm of critical system directory"),
    (re.compile(r'\bmkfs\b', re.IGNORECASE), "disk formatting command"),
    (re.compile(r'\bdd\s+.*of=\/dev\/', re.IGNORECASE), "disk wipe via dd"),
    (re.compile(r'\bshred\s+.*\/dev\/', re.IGNORECASE), "disk shred"),
    (re.compile(r'\bformat\s+', re.IGNORECASE), "disk format"),
    
    # Firewall destruction  
    (re.compile(r'\bufw\s+reset\b', re.IGNORECASE), "firewall reset (blocks all traffic)"),
    (re.compile(r'\biptables\s+(-F|-X|--flush)\b', re.IGNORECASE), "iptables flush (clears all rules)"),
    
    # Database destruction (without scope)
    (re.compile(r'\bDROP\s+DATABASE\b', re.IGNORECASE), "DROP DATABASE (destroys entire database)"),
    (re.compile(r'\bDROP\s+TABLE\s+(users|organizations|servers|audit_logs)\b', re.IGNORECASE), "DROP of critical table"),
    (re.compile(r'\bTRUNCATE\s+TABLE\s+(users|organizations|servers|audit_logs)\b', re.IGNORECASE), "TRUNCATE of critical table"),
    
    # Credential destruction
    (re.compile(r'\bpasswd\s+--delete\b', re.IGNORECASE), "delete user password"),
    (re.compile(r'\buserdel\s+-r\s+root\b', re.IGNORECASE), "delete root user"),
    
    # Fork bomb / resource exhaustion
    (re.compile(r':\(\)\s*\{.*:\|:&\s*\}', re.IGNORECASE), "fork bomb"),
    (re.compile(r'\byes\s*>\s*\/dev\/null\b', re.IGNORECASE), "infinite output (resource exhaustion)"),
    
    # Privilege escalation
    (re.compile(r'\bsudo\s+su\b|\bsudo\s+-i\b|\bsudo\s+bash\b|\bsudo\s+sh\b', re.IGNORECASE), "unrestricted root shell via sudo"),
    (re.compile(r'\bchmod\s+777\s+\/', re.IGNORECASE), "world-writable root"),
    (re.compile(r'\bchmod\s+-R\s+777\b', re.IGNORECASE), "recursive world-writable"),
]

# ============================================================================
# DANGEROUS PATTERNS — Blocked by default, require explicit approval
# ============================================================================
_DANGEROUS_PATTERNS: List[Tuple[re.Pattern, str]] = [
    (re.compile(r'\bsystemctl\s+(stop|disable)\b', re.IGNORECASE), "stopping/disabling a service"),
    (re.compile(r'\bcertbot\s+revoke\b', re.IGNORECASE), "SSL certificate revocation"),
    (re.compile(r'\bufw\s+disable\b', re.IGNORECASE), "disabling firewall"),
    (re.compile(r'\bDELETE\s+FROM\b', re.IGNORECASE), "SQL DELETE statement"),
    (re.compile(r'\bDROP\s+TABLE\b', re.IGNORECASE), "SQL DROP TABLE"),
    (re.compile(r'\bTRUNCATE\b', re.IGNORECASE), "SQL TRUNCATE"),
    (re.compile(r'\bALTER\s+TABLE\b', re.IGNORECASE), "SQL ALTER TABLE"),
    (re.compile(r'\brm\s+(-[rRf]+\s+)', re.IGNORECASE), "recursive rm"),
    (re.compile(r'\bnpm\s+publish\b|\bnpx\s+.*publish\b', re.IGNORECASE), "npm publish"),
    (re.compile(r'\bgit\s+push\s+.*--force\b', re.IGNORECASE), "git force push"),
    (re.compile(r'\bgit\s+reset\s+--hard\b', re.IGNORECASE), "git reset hard"),
    (re.compile(r'\bgit\s+clean\s+-fd\b', re.IGNORECASE), "git clean -fd (deletes untracked)"),
]

# ============================================================================
# REVIEW PATTERNS — Allowed with approval workflow
# ============================================================================
_REVIEW_PATTERNS: List[Tuple[re.Pattern, str]] = [
    (re.compile(r'\bsystemctl\s+restart\b', re.IGNORECASE), "service restart"),
    (re.compile(r'\bsystemctl\s+reload\b', re.IGNORECASE), "service reload"),
    (re.compile(r'\bnginx\s+-s\s+reload\b', re.IGNORECASE), "nginx reload"),
    (re.compile(r'\bUPDATE\s+\w+\s+SET\b', re.IGNORECASE), "SQL UPDATE statement"),
    (re.compile(r'\bINSERT\s+INTO\b', re.IGNORECASE), "SQL INSERT statement"),
    (re.compile(r'\bgo\s+build\b', re.IGNORECASE), "Go build (modifies binary)"),
    (re.compile(r'\bapt\s+install\b|\bapt-get\s+install\b', re.IGNORECASE), "package installation"),
    (re.compile(r'\bgit\s+commit\b', re.IGNORECASE), "git commit"),
    (re.compile(r'\bgit\s+push\b', re.IGNORECASE), "git push"),
    (re.compile(r'\bcertbot\s+certonly\b|\bcertbot\s+--nginx\b', re.IGNORECASE), "SSL certificate issuance"),
    (re.compile(r'\bchown\b|\bchmod\b', re.IGNORECASE), "file permission change"),
]

# ============================================================================
# SAFE PATTERNS — Always allowed (read-only operations)
# ============================================================================
_SAFE_PATTERNS: List[Tuple[re.Pattern, str]] = [
    (re.compile(r'^(ls|ll|la|find|cat|head|tail|grep|awk|sed|wc|sort|uniq|echo|pwd|whoami|id|hostname|uname|uptime)\b'), "read-only system command"),
    (re.compile(r'^cd\s+[^;&|]+\b', re.IGNORECASE), "directory navigation"),
    (re.compile(r'^git\s+(status|log|diff|show|branch|tag|ls-files|remote)\b', re.IGNORECASE), "read-only git command"),
    (re.compile(r'^go\s+(version|env|list|vet|test)\b', re.IGNORECASE), "read-only/test Go command"),
    (re.compile(r'^(free|df|du|ps|top|htop|ss|netstat|lscpu|lsmem|lsblk|dmidecode)\b', re.IGNORECASE), "read-only system diagnostic"),
    (re.compile(r'^(journalctl|systemctl\s+status|systemctl\s+list)\b', re.IGNORECASE), "read-only service status"),
    (re.compile(r'^(nginx\s+-t|php-fpm\s+--test)\b', re.IGNORECASE), "config syntax test"),
    (re.compile(r'^(certbot\s+certificates|ufw\s+status)\b', re.IGNORECASE), "read-only SSL/firewall status"),
    (re.compile(r'^(curl\s+.*localhost|wget\s+.*localhost)\b', re.IGNORECASE), "local HTTP health check"),
    (re.compile(r'^python3?\s+.*\.py\b', re.IGNORECASE), "Python script"),
    (re.compile(r'^npm\s+(run|list|audit|outdated)\b', re.IGNORECASE), "npm read/test command"),
]


def classify_command(command: str) -> CommandDecision:
    """
    Classify a command and return its safety decision.
    
    Evaluation order:
    1. BLOCKED (hardcoded — no override possible)
    2. DANGEROUS (requires explicit approval)
    3. REVIEW (requires review/approval workflow)
    4. SAFE (always allowed)
    5. Default: REVIEW (unknown commands default to requiring review)
    """
    if not command or not command.strip():
        return CommandDecision(command, CommandClass.SAFE, "empty command")

    cmd = command.strip()

    # 1. Check BLOCKED first — these are absolute
    for pattern, reason in _BLOCKED_PATTERNS:
        if pattern.search(cmd):
            return CommandDecision(
                command=cmd,
                classification=CommandClass.BLOCKED,
                reason=f"BLOCKED: {reason}",
                blocked_pattern=pattern.pattern,
            )

    # 2. Check compound && commands
    if "&&" in cmd:
        sub_cmds = [s.strip() for s in cmd.split("&&") if s.strip()]
        sub_decisions = [classify_command(s) for s in sub_cmds]
        # Any blocked?
        for d in sub_decisions:
            if d.classification == CommandClass.BLOCKED:
                return d
        # Any dangerous?
        for d in sub_decisions:
            if d.classification == CommandClass.DANGEROUS:
                return d
        # Any review?
        for d in sub_decisions:
            if d.classification == CommandClass.REVIEW:
                return d
        return CommandDecision(cmd, CommandClass.SAFE, "all chained commands are safe")

    # 2. Check DANGEROUS
    for pattern, reason in _DANGEROUS_PATTERNS:
        if pattern.search(cmd):
            return CommandDecision(
                command=cmd,
                classification=CommandClass.DANGEROUS,
                reason=f"DANGEROUS: {reason} — requires explicit approval",
                blocked_pattern=pattern.pattern,
            )

    # 3. Check REVIEW
    for pattern, reason in _REVIEW_PATTERNS:
        if pattern.search(cmd):
            return CommandDecision(
                command=cmd,
                classification=CommandClass.REVIEW,
                reason=f"REVIEW: {reason}",
            )

    # 4. Check SAFE
    for pattern, reason in _SAFE_PATTERNS:
        if pattern.search(cmd):
            return CommandDecision(
                command=cmd,
                classification=CommandClass.SAFE,
                reason=f"SAFE: {reason}",
            )

    # 5. Unknown — default to REVIEW
    return CommandDecision(
        command=cmd,
        classification=CommandClass.REVIEW,
        reason="REVIEW: Unknown command — manual review required",
    )


def is_allowed(command: str, allow_review: bool = False) -> Tuple[bool, str]:
    """
    Quick check: can this command run?
    
    Returns (allowed, reason).
    BLOCKED/DANGEROUS always return False.
    SAFE always returns True.
    REVIEW returns True only if allow_review=True.
    """
    decision = classify_command(command)

    if decision.classification == CommandClass.BLOCKED:
        return False, decision.reason
    if decision.classification == CommandClass.DANGEROUS:
        return False, decision.reason
    if decision.classification == CommandClass.SAFE:
        return True, decision.reason
    if decision.classification == CommandClass.REVIEW:
        if allow_review:
            return True, decision.reason
        return False, decision.reason

    return False, "Unknown classification"


def validate_path(path: str) -> Tuple[bool, str]:
    """
    Validate that a file path is within allowed boundaries.
    Blocks access to critical system paths.
    """
    import os
    abs_path = os.path.abspath(path)

    # Blocked system paths
    blocked_prefixes = [
        "/etc/passwd", "/etc/shadow", "/etc/sudoers",
        "/proc/", "/sys/", "/dev/",
        "/boot/",
    ]

    for blocked in blocked_prefixes:
        if abs_path.startswith(blocked):
            return False, f"Path blocked: {blocked} is a critical system path"

    return True, "Path allowed"


if __name__ == "__main__":
    # Self-test
    test_commands = [
        ("ls -la /var/www", CommandClass.SAFE),
        ("git status", CommandClass.SAFE),
        ("go test ./...", CommandClass.SAFE),
        ("systemctl restart nginx", CommandClass.REVIEW),
        ("git commit -m 'fix'", CommandClass.REVIEW),
        ("systemctl stop hostvra-api", CommandClass.DANGEROUS),
        ("DELETE FROM users WHERE id = '1'", CommandClass.DANGEROUS),
        ("rm -rf /", CommandClass.BLOCKED),
        ("DROP DATABASE hostvra", CommandClass.BLOCKED),
        ("ufw reset", CommandClass.BLOCKED),
        ("sudo bash", CommandClass.BLOCKED),
    ]

    print("Command Safety Engine — Self Test")
    print("=" * 60)
    all_passed = True
    for cmd, expected in test_commands:
        decision = classify_command(cmd)
        passed = decision.classification == expected
        all_passed = all_passed and passed
        status = "✅" if passed else "❌"
        print(f"{status} [{decision.classification.value}] {cmd[:50]}")
        if not passed:
            print(f"   Expected: {expected.value}, Got: {decision.classification.value}")
            print(f"   Reason: {decision.reason}")

    print()
    print("=" * 60)
    print(f"Result: {'ALL PASSED' if all_passed else 'SOME FAILED'}")
    exit(0 if all_passed else 1)
