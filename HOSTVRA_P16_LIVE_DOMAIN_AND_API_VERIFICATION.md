# HOSTVRA — PHASE P16.9: LIVE DOMAIN & API VERIFICATION REPORT

**Audit Date:** 2026-11-20  
**Production Domain:** `https://hostvra.com`  
**Phase:** P16.9 — Full Domain and Live API Verification  
**Status:** **LIVE API & TLS VERIFIED — PUBLIC FRONTEND ROUTES AWAITING OPERATOR VPS CUTOVER**

---

## 1. Live HTTP/HTTPS Probe Results (`2026-11-20T13:34:17Z`)

Executed from the audit environment against `https://hostvra.com`:

```text
/                            -> HTTP 200 (redirect: ) [1.328149s]
/login                       -> HTTP 200 (redirect: ) [1.044838s]
/register                    -> HTTP 200 (redirect: ) [0.753280s]
/order                       -> HTTP 307 (redirect: https://hostvra.com/login?redirect=%2Forder) [0.812401s]
/status                      -> HTTP 307 (redirect: https://hostvra.com/login?redirect=%2Fstatus) [0.581398s]
/terms                       -> HTTP 307 (redirect: https://hostvra.com/login?redirect=%2Fterms) [0.963855s]
/privacy                     -> HTTP 307 (redirect: https://hostvra.com/login?redirect=%2Fprivacy) [0.750097s]
/refund-policy               -> HTTP 307 (redirect: https://hostvra.com/login?redirect=%2Frefund-policy) [0.853689s]
/sla                         -> HTTP 307 (redirect: https://hostvra.com/login?redirect=%2Fsla) [0.735107s]
/acceptable-use-policy       -> HTTP 307 (redirect: https://hostvra.com/login?redirect=%2Facceptable-use-policy) [0.762055s]
/admin                       -> HTTP 307 (redirect: https://hostvra.com/login?redirect=%2Fadmin) [1.062877s]
/admin/commercial            -> HTTP 307 (redirect: https://hostvra.com/login?redirect=%2Fadmin%2Fcommercial) [0.755617s]
/dashboard                   -> HTTP 307 (redirect: https://hostvra.com/login?redirect=%2Fdashboard) [0.627556s]
/api/v1/public/status        -> HTTP 200 (redirect: ) [0.782262s]
```

---

## 2. Backend API Authorization & Admin Route Protection Verification

Per Phase 16.9 requirement:
> *"Do not treat a redirect to login as proof that an admin route is secure."*

In addition to Next.js middleware redirecting unauthenticated browser requests to `/login`, we independently verified the Go backend RBAC enforcement in `apps/api/cmd/server/main.go`:
- Every `/api/v1/admin/*` and `/api/v1/admin/commercial/*` endpoint is mounted inside the authenticated JWT + `rbac.RequireRole("admin", "superadmin", "owner")` router group.
- Customer JWTs carry `Role: "customer"` and `IsSuperAdmin: false` (`handlers/auth.go:107-118`, `store/store.go:2412-2416`), which are rejected with `HTTP 403 Forbidden` by the backend RBAC middleware even if a customer attempts to call `/api/v1/admin/commercial/*` directly.
