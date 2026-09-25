"""
Hostvra AI Engineering — Orchestrator
Main entry point for the multi-agent Antigravity SDK system.

Architecture:
  Orchestrator → delegates to specialized sub-agents:
    - architect  : repository & architecture analysis
    - backend    : Go API + Agent code analysis
    - frontend   : Next.js dashboard analysis
    - linux      : Linux/server infrastructure diagnostics
    - database   : PostgreSQL schema + query analysis
    - security   : Security audit
    - qa         : Test suite runner + code quality

Zero-Demo Policy: Every task produces evidence. No fake implementations.
"""

import asyncio
import os
import sys
from pathlib import Path

# Ensure the repo root is accessible
REPO_ROOT = Path(__file__).resolve().parent.parent.parent
TOOLS_DIR = Path(__file__).resolve().parent.parent / "tools"
ORCHESTRATOR_DIR = Path(__file__).resolve().parent

# Use the .venv Python interpreter when running externally
VENV_PYTHON = REPO_ROOT / ".venv" / "bin" / "python"

# Add the tools and orchestrator path for imports
sys.path.insert(0, str(TOOLS_DIR))
sys.path.insert(0, str(ORCHESTRATOR_DIR))

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

load_env()

from google.antigravity import Agent, LocalAgentConfig, types
from google.antigravity.hooks import policy

from evidence import EvidenceRecord


# ============================================================================
# POLICY DEFINITIONS
# ============================================================================

def _build_read_only_policy():
    """
    Read-only policy: no shell execution, no file writes.
    Used by analysis agents that must not modify anything.
    """
    return [
        policy.allow("invoke_subagent"),
        policy.allow("finish"),
        policy.deny("run_command"),
        policy.deny("create_file"),
        policy.deny("edit_file"),
        policy.allow("view_file"),
        policy.allow("list_directory"),
        policy.allow("search_directory"),
        policy.allow("find_file"),
    ]


def _build_analysis_policy():
    """
    Analysis policy: allows safe read-only commands and file reading.
    No writes. Shell commands are restricted to safe patterns.
    """
    async def safe_command_handler(tool_call) -> bool:
        from command_safety import classify_command, CommandClass
        decision = classify_command(tool_call.args.get("CommandLine", ""))
        if decision.classification == CommandClass.SAFE:
            return True
        print(f"\n[SAFETY] Blocked command: {tool_call.args.get('CommandLine', '')}")
        print(f"[SAFETY] Reason: {decision.reason}")
        return False

    return [
        policy.allow("invoke_subagent"),
        policy.allow("finish"),
        policy.deny("create_file"),
        policy.deny("edit_file"),
        policy.ask_user("run_command", handler=safe_command_handler),
        policy.allow("view_file"),
        policy.allow("list_directory"),
        policy.allow("search_directory"),
        policy.allow("find_file"),
    ]


def _build_development_policy():
    """
    Development policy: file writes and safe commands allowed.
    Destructive operations require human confirmation.
    """
    async def development_command_handler(tool_call) -> bool:
        from command_safety import classify_command, CommandClass
        from approval import request_approval, ApprovalRequest
        command = tool_call.args.get("CommandLine", "")
        decision = classify_command(command)

        if decision.classification == CommandClass.BLOCKED:
            print(f"\n[SAFETY] BLOCKED (hardcoded): {command}")
            return False
        if decision.classification == CommandClass.DANGEROUS:
            req = ApprovalRequest(
                operation=f"Execute command: {command[:60]}",
                reason=f"Command classified as DANGEROUS: {decision.reason}",
                risk_level="HIGH",
                commands=[command],
            )
            return request_approval(req)
        if decision.classification == CommandClass.REVIEW:
            req = ApprovalRequest(
                operation=f"Execute command: {command[:60]}",
                reason=f"Command requires review: {decision.reason}",
                risk_level="MEDIUM",
                commands=[command],
            )
            return request_approval(req)
        return True  # SAFE

    async def file_write_handler(tool_call) -> bool:
        from approval import request_approval, ApprovalRequest
        path = tool_call.args.get("TargetFile", tool_call.args.get("path", "unknown"))
        req = ApprovalRequest(
            operation=f"Write file: {path}",
            reason="AI agent is writing to a file",
            risk_level="MEDIUM",
            affected_files=[path],
            rollback_plan=f"git restore {path}",
        )
        return request_approval(req)

    return [
        policy.allow("invoke_subagent"),
        policy.allow("finish"),
        policy.ask_user("run_command", handler=development_command_handler),
        policy.ask_user("create_file", handler=file_write_handler),
        policy.ask_user("edit_file", handler=file_write_handler),
        policy.allow("view_file"),
        policy.allow("list_directory"),
        policy.allow("search_directory"),
        policy.allow("find_file"),
    ]


# ============================================================================
# SUB-AGENT DEFINITIONS
# ============================================================================

def _make_architect_agent() -> types.SubagentConfig:
    """Architecture analysis agent — read-only."""
    return types.SubagentConfig(
        name="architect",
        description=(
            "Analyzes the Hostvra repository architecture. "
            "Understands Go API, Agent, Next.js web, PostgreSQL schema, and CI/CD. "
            "Produces architecture diagrams, dependency maps, and component relationships. "
            "READ-ONLY — never modifies files."
        ),
        capabilities=types.SubagentCapabilities(
            enabled_tools=[
                types.BuiltinTools.VIEW_FILE,
                types.BuiltinTools.LIST_DIR,
                types.BuiltinTools.SEARCH_DIR,
                types.BuiltinTools.FIND_FILE,
                types.BuiltinTools.FINISH,
            ],
        ),
        system_instructions=(
            "You are the Hostvra Architecture Agent. "
            "You analyze the repository structure and document REAL architecture. "
            "You NEVER assume capabilities not confirmed in files. "
            "You NEVER create fake implementations. "
            "Your job is to read and understand — not to write code. "
            "\n"
            "CRITICAL TOOL RULE: "
            "To inspect directories or list files in a folder, ALWAYS use list_dir. "
            "Only use view_file on specific file paths (.go, .ts, .md, .sql). "
            "NEVER call view_file on a directory path. "
            "\n"
            f"Repository root: {REPO_ROOT}"
        ),
    )


def _make_backend_agent() -> types.SubagentConfig:
    """Go backend analysis and development agent."""
    return types.SubagentConfig(
        name="backend",
        description=(
            "Analyzes and implements Go backend code for Hostvra API and Agent. "
            "Reads handler files, store implementations, tests, and RBAC. "
            "Can write Go code changes after human approval. "
            "Follows handler pattern exactly as defined in go-backend skill."
        ),
        capabilities=types.SubagentCapabilities(
            enabled_tools=[
                types.BuiltinTools.VIEW_FILE,
                types.BuiltinTools.LIST_DIR,
                types.BuiltinTools.SEARCH_DIR,
                types.BuiltinTools.FIND_FILE,
                types.BuiltinTools.RUN_COMMAND,
                types.BuiltinTools.EDIT_FILE,
                types.BuiltinTools.CREATE_FILE,
                types.BuiltinTools.FINISH,
            ],
        ),
        system_instructions=(
            "You are the Hostvra Backend Agent. "
            "You analyze and modify Go code in apps/api and apps/agent. "
            "Before ANY modification: read the file, understand the pattern, document the change. "
            "After ANY modification: run go vet ./... and go test -count=1 ./... "
            "NEVER skip tests after a change. "
            "NEVER create fake handler implementations. "
            "ALWAYS follow the existing handler pattern in internal/handlers/. "
            "Zero-Demo Policy: every status must be evidence-backed. "
            f"Repository root: {REPO_ROOT}"
        ),
    )


def _make_frontend_agent() -> types.SubagentConfig:
    """Next.js frontend analysis agent."""
    return types.SubagentConfig(
        name="frontend",
        description=(
            "Analyzes Hostvra Next.js 15 dashboard (React 19, TypeScript, TailwindCSS). "
            "Inspects pages, components, API routes, and middleware. "
            "Can identify TypeScript errors, component issues, and missing features."
        ),
        capabilities=types.SubagentCapabilities(
            enabled_tools=[
                types.BuiltinTools.VIEW_FILE,
                types.BuiltinTools.LIST_DIR,
                types.BuiltinTools.SEARCH_DIR,
                types.BuiltinTools.FIND_FILE,
                types.BuiltinTools.RUN_COMMAND,
                types.BuiltinTools.FINISH,
            ],
        ),
        system_instructions=(
            "You are the Hostvra Frontend Agent. "
            "You analyze the Next.js 15 dashboard in apps/web/. "
            "Stack: Next.js 15 App Router, React 19, TypeScript 5.7, TailwindCSS 3.4. "
            "CAUTION: apps/web/src/app/email/ and WebmailClient.tsx are IN PROGRESS — do not modify. "
            "You can run 'npm run build' to validate TypeScript. "
            "NEVER create placeholder React components. "
            f"Repository root: {REPO_ROOT}"
        ),
    )


def _make_linux_agent() -> types.SubagentConfig:
    """Linux/server diagnostics agent — safe commands only."""
    return types.SubagentConfig(
        name="linux",
        description=(
            "Runs safe Linux system diagnostics for Hostvra infrastructure. "
            "Checks service status, disk usage, memory, logs, and network. "
            "ONLY executes pre-classified SAFE read-only commands. "
            "Cannot restart services or modify system configuration."
        ),
        capabilities=types.SubagentCapabilities(
            enabled_tools=[
                types.BuiltinTools.RUN_COMMAND,
                types.BuiltinTools.VIEW_FILE,
                types.BuiltinTools.FINISH,
            ],
        ),
        system_instructions=(
            "You are the Hostvra Linux Diagnostics Agent. "
            "You run ONLY safe, read-only diagnostic commands. "
            "Safe commands include: ls, cat, grep, find, ps, df, free, uname, "
            "systemctl status, journalctl (read), go version, node --version, etc. "
            "You NEVER run: rm, dd, mkfs, systemctl stop/disable, ufw reset, DROP DATABASE. "
            "Any command not explicitly safe requires human approval before running. "
            "Report actual command output — never fabricate results."
        ),
    )


def _make_database_agent() -> types.SubagentConfig:
    """Database analysis agent — read-only queries."""
    return types.SubagentConfig(
        name="database",
        description=(
            "Analyzes Hostvra PostgreSQL schema and data. "
            "Reads migration files and runs safe SELECT queries. "
            "NEVER runs DROP, DELETE, TRUNCATE, or ALTER without explicit approval."
        ),
        capabilities=types.SubagentCapabilities(
            enabled_tools=[
                types.BuiltinTools.VIEW_FILE,
                types.BuiltinTools.LIST_DIR,
                types.BuiltinTools.SEARCH_DIR,
                types.BuiltinTools.FIND_FILE,
                types.BuiltinTools.RUN_COMMAND,
                types.BuiltinTools.FINISH,
            ],
        ),
        system_instructions=(
            "You are the Hostvra Database Agent. "
            "You analyze PostgreSQL schema in /migrations/ and /apps/api/internal/store/. "
            "You can run psql with SELECT queries using DATABASE_URL from environment. "
            "BLOCKED: DROP DATABASE, DROP TABLE (users/organizations/audit_logs), TRUNCATE, DELETE without WHERE. "
            "REQUIRE APPROVAL: DELETE with WHERE, INSERT, UPDATE, ALTER TABLE, new migrations. "
            "All findings must include actual query output as evidence."
        ),
    )


def _make_security_agent() -> types.SubagentConfig:
    """Security audit agent."""
    return types.SubagentConfig(
        name="security",
        description=(
            "Performs security audits on Hostvra codebase. "
            "Checks authentication, authorization, injection, path traversal, CORS, secrets. "
            "Uses the security skill methodology. READ-ONLY."
        ),
        capabilities=types.SubagentCapabilities(
            enabled_tools=[
                types.BuiltinTools.VIEW_FILE,
                types.BuiltinTools.LIST_DIR,
                types.BuiltinTools.SEARCH_DIR,
                types.BuiltinTools.FIND_FILE,
                types.BuiltinTools.RUN_COMMAND,
                types.BuiltinTools.FINISH,
            ],
        ),
        system_instructions=(
            "You are the Hostvra Security Audit Agent. "
            "You audit the codebase against OWASP Top 10 and hosting-specific threats. "
            "\n"
            "Key security files:\n"
            "- Handlers: apps/api/internal/handlers/auth.go, handlers/terminal.go, handlers/firewall.go\n"
            "- Auth packages: apps/api/internal/auth/hasher.go (Argon2id), auth/jwt.go, auth/middleware.go\n"
            "- RBAC: apps/api/internal/rbac/rbac.go\n"
            "\n"
            "Known findings from preflight: "
            "  SEC-001: CORS allows all origins (main.go) "
            "  SEC-002: Terminal executes arbitrary shell — no allowlist (handlers/terminal.go) "
            "  SEC-003: No rate limiting on auth endpoints "
            "  SEC-004: No CSRF token validation "
            "\n"
            "CRITICAL TOOL RULE: "
            "Always check directory contents with list_dir or find_file before calling view_file. "
            "Never assume or invent file names (e.g. password.go does not exist, use hasher.go). "
            "You use grep, cat, and read-only tools. "
            "You NEVER modify security-sensitive files without explicit approval. "
            "Every finding must have: severity, component, evidence (code/output), recommendation."
        ),
    )


def _make_qa_agent() -> types.SubagentConfig:
    """QA and test runner agent."""
    return types.SubagentConfig(
        name="qa",
        description=(
            "Runs Hostvra test suites and reports results with evidence. "
            "Executes: go test -count=1 -v -race ./... for API and Agent. "
            "Reports exact pass/fail counts. Never hides failures. "
            "Can also run go vet and npm run build."
        ),
        capabilities=types.SubagentCapabilities(
            enabled_tools=[
                types.BuiltinTools.RUN_COMMAND,
                types.BuiltinTools.VIEW_FILE,
                types.BuiltinTools.FINISH,
            ],
        ),
        system_instructions=(
            "You are the Hostvra QA Agent. "
            "You run test suites and report EXACT results. "
            "Commands to run: "
            "  cd apps/api && go test -count=1 -v -race ./... "
            "  cd apps/agent && go test -count=1 -v -race ./... "
            "  cd apps/api && go vet ./... "
            "  cd apps/agent && go vet ./... "
            "NEVER: report tests as passing without running them. "
            "NEVER: hide failed tests or suppressed output. "
            "ALWAYS: report the exact number of passed/failed/skipped tests. "
            "ALWAYS: include full failure output in evidence. "
            f"Working directory: {REPO_ROOT}"
        ),
    )


# ============================================================================
# MAIN ORCHESTRATOR
# ============================================================================

AGENT_FACTORIES = {
    "architect": _make_architect_agent,
    "backend": _make_backend_agent,
    "frontend": _make_frontend_agent,
    "linux": _make_linux_agent,
    "database": _make_database_agent,
    "security": _make_security_agent,
    "qa": _make_qa_agent,
}


async def run_orchestrator(task: str, policy_mode: str = "analysis", model: str | None = None, direct_agent: str | None = None) -> None:
    """
    Run the Hostvra AI Engineering Orchestrator or direct specialized agent.

    Args:
        task: The engineering task to perform
        policy_mode: "read_only" | "analysis" | "development"
        model: Model identifier (e.g. "gemini-3.5-flash", "gemini-3.5-flash-lite")
        direct_agent: Optional name of specific agent to run directly
    """
    api_key = os.environ.get("GEMINI_API_KEY")
    if not api_key:
        print("ERROR: GEMINI_API_KEY environment variable is not set.")
        print("Set it with: export GEMINI_API_KEY=your_key_here")
        sys.exit(1)

    chosen_model = model or os.environ.get("GEMINI_MODEL", "gemini-3.5-flash-lite")
    agent_label = direct_agent if direct_agent else "orchestrator"

    # Build evidence record
    evidence = EvidenceRecord(agent=agent_label, request=task)
    evidence.set_plan([
        f"Execute task with {agent_label}",
        "Enforce strict permission policies",
        "Collect results with evidence",
        "Produce evidence report",
    ])

    print(f"\n{'='*70}")
    print(f"  HOSTVRA AI ENGINEERING SYSTEM")
    print(f"  Agent: {agent_label.upper()}")
    print(f"  Policy Mode: {policy_mode.upper()}")
    print(f"  Model: {chosen_model}")
    print(f"  Task: {task[:60]}...")
    print(f"{'='*70}\n")

    # Select policy and tool restrictions
    if policy_mode == "read_only":
        policies = _build_read_only_policy()
        disabled_tools = [
            types.BuiltinTools.RUN_COMMAND,
            types.BuiltinTools.CREATE_FILE,
            types.BuiltinTools.EDIT_FILE,
        ]
    elif policy_mode == "development":
        policies = _build_development_policy()
        disabled_tools = []
    else:
        policies = _build_analysis_policy()
        disabled_tools = [
            types.BuiltinTools.CREATE_FILE,
            types.BuiltinTools.EDIT_FILE,
        ]

    retry_cfg = types.RetryConfig(
        api_retry=types.ModelAPIRetryConfig(
            max_retries=10,
            initial_sleep_duration_ms=6000,
            exponential_multiplier=1.5,
        )
    )

    if direct_agent and direct_agent in AGENT_FACTORIES:
        sub = AGENT_FACTORIES[direct_agent]()
        active_tools = [t for t in (sub.capabilities.enabled_tools or []) if t not in disabled_tools]
        config = LocalAgentConfig(
            api_key=api_key,
            model=chosen_model,
            workspaces=[str(REPO_ROOT)],
            skills_paths=[str(REPO_ROOT / ".antigravity" / "skills")],
            retry_config=retry_cfg,
            system_instructions=sub.system_instructions,
            capabilities=types.CapabilitiesConfig(
                enabled_tools=active_tools,
            ),
            policies=policies,
        )
    else:
        config = LocalAgentConfig(
            api_key=api_key,
            model=chosen_model,
            workspaces=[str(REPO_ROOT)],
            skills_paths=[str(REPO_ROOT / ".antigravity" / "skills")],
            retry_config=retry_cfg,
            system_instructions=(
                "You are the Hostvra AI Engineering Orchestrator. "
                "You delegate engineering tasks to specialized sub-agents. "
                "Zero-Demo Policy: every claim must be evidence-backed. "
                "Never create fake implementations. Never hide failures. "
                "\n"
                f"Current Policy Mode: {policy_mode.upper()}\n"
                "- In 'READ_ONLY' mode: All shell commands are strictly blocked. Use file reading, directory search, or the 'architect' subagent. Do NOT call 'run_command' or invoke 'qa' runner.\n"
                "- In 'ANALYSIS' mode: Safe read-only diagnostic commands (go test, go vet, git status, find, grep) are permitted.\n"
                "- In 'DEVELOPMENT' mode: Code modifications are permitted with approval.\n"
                "\n"
                "Hostvra repository structure:\n"
                "- apps/api/: Go REST API backend (cmd/server/main.go, internal/handlers, internal/store)\n"
                "- apps/agent/: Go Node Agent daemon (cmd/agent/main.go, pkg/docker, pkg/ssl, pkg/firewall)\n"
                "- apps/web/: Next.js 15 Web Dashboard (src/app, src/components)\n"
                "- Note: There is NO root go.mod. Each Go service has its own go.mod under apps/api and apps/agent.\n"
                "\n"
                "Available sub-agents: architect, backend, frontend, linux, database, security, qa.\n"
                f"Repository root: {REPO_ROOT}"
            ),
            capabilities=types.CapabilitiesConfig(
                enable_subagents=True,
                max_subagent_depth=2,
                allowed_subagents=["architect", "backend", "frontend", "linux", "database", "security", "qa"],
                disabled_tools=disabled_tools,
            ),
            subagents=[
                _make_architect_agent(),
                _make_backend_agent(),
                _make_frontend_agent(),
                _make_linux_agent(),
                _make_database_agent(),
                _make_security_agent(),
                _make_qa_agent(),
            ],
            policies=policies,
        )

    try:
        max_attempts = 4
        for attempt in range(1, max_attempts + 1):
            try:
                async with Agent(config=config) as agent:
                    response = await agent.chat(task)
                    result_text = await response.text()

                    print("\n" + "="*70)
                    print("ORCHESTRATOR RESULT")
                    print("="*70)
                    print(result_text)

                    evidence.finalize("COMPLETED")
                    break

            except Exception as e:
                err_msg = str(e)
                if ("429" in err_msg or "RESOURCE_EXHAUSTED" in err_msg) and attempt < max_attempts:
                    import re
                    m = re.search(r'retry in ([0-9.]+)s', err_msg, re.IGNORECASE)
                    m_delay = re.search(r'retryDelay:\s*([0-9]+)s', err_msg, re.IGNORECASE)
                    wait_sec = 60.0
                    if m:
                        wait_sec = float(m.group(1)) + 3.0
                    elif m_delay:
                        wait_sec = float(m_delay.group(1)) + 3.0

                    print(f"\n⏳ [Rate Limit] Free tier quota window limit reached.")
                    print(f"   Auto-waiting {wait_sec:.0f}s for Google API quota to reset (attempt {attempt}/{max_attempts})...")
                    await asyncio.sleep(wait_sec)
                    print("🔄 Resuming task execution...\n")
                    continue
                else:
                    evidence.add_error(str(e))
                    evidence.finalize("FAILED")
                    print(f"\nERROR: {e}", file=sys.stderr)
                    raise

    finally:
        json_path = evidence.save()
        md_path = evidence.save_markdown_report()
        print(f"\n📋 Evidence saved:")
        print(f"   JSON: {json_path}")
        print(f"   Report: {md_path}")


def main():
    import argparse

    parser = argparse.ArgumentParser(
        description="Hostvra AI Engineering Orchestrator",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  python main.py "Analyze the repository architecture and list all API endpoints"
  python main.py --mode analysis "Audit the firewall handler for security issues"
  python main.py --mode development "Add rate limiting to the auth login endpoint"
  python main.py --agent qa "Run the full test suite and report results"
        """,
    )
    parser.add_argument("task", help="The engineering task to perform")
    parser.add_argument(
        "--mode",
        choices=["read_only", "analysis", "development"],
        default="analysis",
        help="Policy mode: read_only | analysis | development (default: analysis)",
    )
    parser.add_argument(
        "--agent",
        choices=["architect", "backend", "frontend", "linux", "database", "security", "qa"],
        help="Directly invoke a specific sub-agent",
    )
    parser.add_argument(
        "--model",
        default=None,
        help="Gemini model to use (default: gemini-3.5-flash or gemini-3.5-flash-lite)",
    )

    args = parser.parse_args()

    task = args.task

    asyncio.run(run_orchestrator(task=task, policy_mode=args.mode, model=args.model, direct_agent=args.agent))


if __name__ == "__main__":
    main()
