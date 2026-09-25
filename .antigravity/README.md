# Hostvra AI Engineering System

Enterprise AI Engineering Layer powered by the Google Antigravity SDK.

## Overview

The `.antigravity/` directory contains a production-grade multi-agent AI system built on top of the Hostvra hosting control panel. It provides:

- **7 Specialized Agents** — architect, backend, frontend, linux, database, security, qa
- **3 Policy Tiers** — read_only, analysis, development
- **Command Safety Engine** — programmatic allowlist/denylist (not AI-instruction-based)
- **Human Approval Gates** — blocks sensitive operations until explicit confirmation
- **Evidence System** — every task produces a timestamped audit record
- **Zero-Demo Policy** — no fake implementations, no placeholder handlers

## Prerequisites

| Requirement | Status | Notes |
|-------------|--------|-------|
| Python 3.12 (.venv) | ✅ Confirmed | Located at `.venv/bin/python` |
| google-antigravity 0.1.18 | ✅ Confirmed | Installed in `.venv` |
| google-genai 2.25.0 | ✅ Confirmed | Installed in `.venv` |
| GEMINI_API_KEY | ⚠️ Must set | `export GEMINI_API_KEY=...` |
| Go 1.27.1 | ✅ Confirmed | System Go |
| Node.js 22.22.0 | ✅ Confirmed | System Node |

## Quick Start

```bash
# 1. Set your API key
export GEMINI_API_KEY=your_key_here

# 2. Run the AI system
cd /Users/mizanurrahman/claude/Hostvra
.venv/bin/python hostvra-ai.py "Analyze the repository architecture"

# 3. Run with a specific mode
.venv/bin/python hostvra-ai.py --mode analysis "Audit the terminal handler for security issues"
.venv/bin/python hostvra-ai.py --mode read_only "List all API endpoints"
.venv/bin/python hostvra-ai.py --mode development "Add rate limiting to the auth login endpoint"

# 4. Target a specific agent directly
.venv/bin/python hostvra-ai.py --agent qa "Run the full test suite"
.venv/bin/python hostvra-ai.py --agent security "Audit CORS configuration"
.venv/bin/python hostvra-ai.py --agent architect "Map all handler dependencies"
```

## Architecture

```
hostvra-ai.py                    ← Entry point (prerequisite checks)
    │
.antigravity/
├── orchestrator/
│   ├── main.py                  ← Antigravity SDK orchestrator (7 sub-agents)
│   ├── evidence.py              ← Evidence recording system
│   └── approval.py              ← Human approval gate
├── agents/                      ← (Reserved for standalone agent scripts)
│   ├── architect/
│   ├── backend/
│   ├── frontend/
│   ├── linux/
│   ├── database/
│   ├── security/
│   └── qa/
├── skills/                      ← Knowledge base for agents
│   ├── hostvra-architecture/SKILL.md   ← Complete codebase reference
│   ├── zero-demo-audit/SKILL.md        ← Mandatory evidence policy
│   ├── go-backend/SKILL.md             ← Go development patterns
│   ├── security/SKILL.md               ← Security audit methodology
│   ├── linux-hosting/SKILL.md          ← Linux operations reference
│   ├── database/SKILL.md               ← PostgreSQL schema + safe ops
│   └── frontend/SKILL.md               ← Next.js dashboard reference
├── tools/
│   ├── command_safety.py        ← Programmatic command classification
│   ├── git_tools.py             ← Read-only git operations
│   ├── test_tools.py            ← Test suite runner
│   ├── diagnostics.py           ← Read-only system diagnostics
│   └── hostvra_api.py           ← Hostvra REST API client
├── policies/
│   └── policies.py              ← Policy tier definitions
├── reports/                     ← Evidence records (auto-created)
└── .env.ai.example              ← Environment variables template
```

## Policy Tiers

| Mode | Shell Commands | File Writes | Destructive Ops |
|------|---------------|-------------|-----------------|
| `read_only` | ❌ Blocked | ❌ Blocked | ❌ Blocked |
| `analysis` | ✅ SAFE only | ❌ Blocked | ❌ Blocked |
| `development` | ✅ SAFE auto / REVIEW with approval | ✅ With approval | ❌ Blocked |

## Agents

| Agent | Primary Role | Permissions |
|-------|-------------|-------------|
| `architect` | Repository & architecture analysis | Read-only |
| `backend` | Go API + Agent code analysis & dev | Read + write (approval) |
| `frontend` | Next.js dashboard analysis | Read + safe commands |
| `linux` | Linux/server diagnostics | Safe commands only |
| `database` | PostgreSQL schema + query analysis | Read + SELECT queries |
| `security` | Security audit (OWASP + hosting threats) | Read-only |
| `qa` | Test runner + code quality | go test + vet + npm build |

## Command Safety Engine

Commands are classified into 4 tiers before execution:

| Tier | Examples | Policy |
|------|---------|--------|
| SAFE | `ls`, `git status`, `go test`, `ps aux` | Always allowed |
| REVIEW | `systemctl restart`, `git commit`, `npm install` | Requires approval |
| DANGEROUS | `systemctl stop`, `DELETE FROM`, `rm -rf dir` | Requires explicit approval |
| BLOCKED | `rm -rf /`, `DROP DATABASE`, `ufw reset`, `sudo bash` | Never executable |

## Evidence Records

Every task produces two evidence files in `.antigravity/reports/`:
- `YYYY-MM-DD-{agent}-{task_id}.json` — Machine-readable full evidence
- `YYYY-MM-DD-{agent}-{task_id}.md` — Human-readable audit report

Evidence is **never deleted** even on failure.

## Known Security Findings (from Preflight)

| ID | Finding | Severity |
|----|---------|----------|
| SEC-001 | CORS allows all origins | MEDIUM |
| SEC-002 | Terminal handler has no command allowlist | HIGH |
| SEC-003 | No rate limiting on auth endpoints | MEDIUM |
| SEC-004 | No CSRF protection | MEDIUM |

## Active Development (DO NOT DISRUPT)

The following files represent in-progress email/webmail work:
- `apps/api/internal/handlers/email.go` (modified)
- `apps/api/internal/handlers/webmail.go` (new)
- `apps/api/internal/store/email_models.go` (modified)
- `apps/api/internal/store/email_store.go` (modified)
- `apps/web/src/app/email/page.tsx` (modified)
- `apps/web/src/components/WebmailClient.tsx` (modified)
- `apps/agent/pkg/email/` (new package)

## Running the Command Safety Self-Test

```bash
.venv/bin/python .antigravity/tools/command_safety.py
```

Expected output: All 11 classification tests pass.

## Running Diagnostics

```bash
.venv/bin/python .antigravity/tools/diagnostics.py
```

## Running Tests (QA Agent)

```bash
.venv/bin/python hostvra-ai.py --agent qa "Run the full API and Agent test suites and report all results"
```
