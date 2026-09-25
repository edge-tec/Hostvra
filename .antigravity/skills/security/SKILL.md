---
name: security
description: |
  Security audit methodology for Hostvra.
  Covers OWASP Top 10, hosting-specific threats, and evidence requirements.
---

# Hostvra Security Audit Reference

## Security Audit Scope

When auditing Hostvra, check ALL of the following:

### 1. Authentication
- [ ] JWT secret strength (must be ≥32 chars, not default value)
- [ ] JWT expiry enforcement
- [ ] Password hashing algorithm (Argon2id required — not MD5, SHA1, bcrypt minimum)
- [ ] Brute-force protection on /auth/login (rate limiting — CURRENTLY MISSING)
- [ ] 2FA implementation correctness
- [ ] Session revocation on logout
- [ ] Refresh token rotation

### 2. Authorization (RBAC)
- [ ] All routes have RBAC middleware (RequirePermission)
- [ ] Super-admin bypass cannot be set via user input
- [ ] Organization isolation — users cannot access other orgs' data
- [ ] Horizontal privilege escalation — user cannot escalate own role

### 3. Injection
- [ ] SQL injection — all queries parameterized
- [ ] Command injection — os/exec calls have no user input in command string
- [ ] LDAP injection (N/A — no LDAP)
- [ ] Template injection (Next.js — dangerouslySetInnerHTML usage)

### 4. Path Traversal
- [ ] File Manager does not allow paths outside /var/www or configured root
- [ ] filepath.Clean() applied before stat/read/write
- [ ] Symlink escape — symlinks do not point outside allowed paths

### 5. SSRF
- [ ] Any URL fetch (support AI assistant, webhooks) must not target internal IPs
- [ ] RFC1918 and localhost blocked in outbound fetch targets

### 6. CSRF
- [ ] API uses JWT (not cookies) as primary auth — reduces CSRF risk
- [ ] CORS: current config allows all origins (FINDING — medium risk)

### 7. XSS
- [ ] React protects against XSS by default
- [ ] Check for dangerouslySetInnerHTML in Next.js components
- [ ] CSP headers (currently not set — FINDING)

### 8. File Upload
- [ ] File upload endpoint validates content type
- [ ] Upload directory is not web-accessible
- [ ] Max file size enforced (10MB body limit in middleware)
- [ ] No executable upload possible

### 9. Webhook Verification
- [ ] /billing/webhooks/{gateway} verifies HMAC signature
- [ ] Webhook endpoint cannot be used to forge billing events

### 10. Secrets
- [ ] No secrets in code (CONFIRMED — defaults clearly labeled)
- [ ] No secrets in git history (audit git log)
- [ ] .env file not committed (check .gitignore)

### 11. Privilege Escalation
- [ ] API runs as non-root in production (check systemd unit)
- [ ] Agent pkg operations that need root use sudo with specific commands
- [ ] No sudo -E or wide sudo grants

### 12. Tenant Isolation
- [ ] All store queries filter by organization_id
- [ ] Server enrollment tokens are organization-scoped
- [ ] File manager paths are user/website-scoped
- [ ] Database operations are server-scoped

### 13. Rate Limiting
- [ ] /auth/login — MISSING (finding)
- [ ] /auth/register — MISSING (finding)
- [ ] /agent/enroll — MISSING (finding)
- [ ] /terminal/execute — MISSING (finding)

### 14. Terminal Security (CRITICAL)
- [ ] Terminal access restricted by PermTerminalAccess RBAC
- [ ] Terminal command has NO allowlist — CRITICAL FINDING
- [ ] Terminal logs all commands to audit_logs — CONFIRMED
- [ ] Terminal timeout is 900 seconds — review if appropriate

## Evidence Requirements

For each finding:
```
FINDING: [ID]
Severity: CRITICAL|HIGH|MEDIUM|LOW
Component: [file:line]
Description: [what the vulnerability is]
Evidence: [code excerpt or command output proving it]
Reproduction: [exact steps]
Fix: [specific code change recommended]
Status: [VERIFIED|NOT_VERIFIED]
```

## Known Security Findings (from preflight)

| ID | Finding | Severity | Status |
|----|---------|----------|--------|
| SEC-001 | CORS allows all origins | MEDIUM | VERIFIED |
| SEC-002 | Terminal executes arbitrary shell — no allowlist | HIGH | VERIFIED |
| SEC-003 | No rate limiting on auth endpoints | MEDIUM | VERIFIED |
| SEC-004 | No CSRF token validation | MEDIUM | VERIFIED |
| SEC-005 | Redis declared but not used (session revocation impossible) | LOW | VERIFIED |
| SEC-006 | CSP header not set | LOW | VERIFIED |

## Safe Audit Commands

These commands are SAFE to run without approval:
```bash
# Check for hardcoded secrets patterns
grep -r "password\s*=\s*['\"][^$]" apps/ --include="*.go"
grep -r "api_key\s*=\s*['\"]" apps/ --include="*.go"

# Check for SQL injection patterns
grep -r "fmt.Sprintf.*SELECT\|fmt.Sprintf.*INSERT\|fmt.Sprintf.*UPDATE" apps/api --include="*.go"

# Check for unsafe exec patterns
grep -r "exec.Command\|exec.CommandContext" apps/ --include="*.go"

# Check .gitignore covers .env
cat .gitignore | grep "\.env"

# Check go vet
cd apps/api && go vet ./...
cd apps/agent && go vet ./...
```
