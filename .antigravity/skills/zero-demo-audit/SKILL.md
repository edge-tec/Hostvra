---
name: zero-demo-audit
description: |
  Mandatory policy for ALL Hostvra AI Engineering agents.
  This skill enforces the ZERO-DEMO policy. Read before every task.
  Every finding must be evidence-based. Every status must be accurate.
---

# Zero-Demo Audit Policy

## MANDATORY — NEVER VIOLATE

The AI Engineering system MUST NEVER:

1. Create fake implementations that look real
2. Create mock production functionality
3. Create placeholder handlers that return hardcoded success
4. Create fake API responses
5. Simulate successful infrastructure operations
6. Hardcode fake production data
7. Hide command failures
8. Ignore or suppress failed tests
9. Report SUCCESS without verified evidence
10. Claim a feature is implemented when it is only stubbed

## Evidence Requirements

Every finding must contain:
- Severity: CRITICAL | HIGH | MEDIUM | LOW | INFO
- Affected component: exact file path + line number where possible
- Evidence: actual command output, test output, or code excerpt
- Reproduction method: exact steps or commands
- Recommended fix: specific actionable change
- Verification result: how to confirm the fix worked

## Status Labels — Use Exactly These

| Label | Meaning |
|-------|---------|
| VERIFIED | Tested and confirmed working with evidence |
| PARTIALLY_VERIFIED | Partially tested — evidence documents what was and was not verified |
| NOT_VERIFIED | Not tested — reason documented |
| FAILED | Tested and failed — failure evidence documented |
| NOT_IMPLEMENTED | Feature does not exist — not a stub, genuinely absent |
| INCOMPLETE | Implementation exists but is not complete or functional |

Never convert uncertainty into VERIFIED.
Never convert FAILED into PARTIALLY_VERIFIED.

## Test Execution Policy

- Run `go test -count=1 ./...` — do NOT use cached results for evidence
- Record actual stdout/stderr output
- Report pass/fail counts exactly
- If tests cannot run (missing DB, environment), report as NOT_VERIFIED with reason
- Never report "tests passing" without running them

## Code Modification Policy

Before modifying ANY file:
1. Read the file completely first
2. Understand what it currently does
3. Document what will change and why
4. Create a git checkpoint if the change is significant
5. Make the minimal change required
6. Run affected tests after the change
7. Document the before/after state in evidence

## Reporting Policy

Every task report must include:
- What was requested
- What was actually done (with evidence)
- What could NOT be done (and why)
- What tests were run (and results)
- Any failures encountered
- Rollback information if applicable

A report that omits failures is INVALID.
A report that claims success without test evidence is INVALID.
