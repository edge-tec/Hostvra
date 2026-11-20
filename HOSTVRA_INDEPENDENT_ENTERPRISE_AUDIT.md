# HOSTVRA — INDEPENDENT FULL-SYSTEM ENTERPRISE AUDIT

**Repository:** `edge-tec/Hostvra` (`/Volumes/EDGE/Hostvra-main`)  
**Audit Date:** 2026-11-20  
**Scope:** Backend (`apps/api`), Frontend (`apps/web`), Database Migrations (`0001`–`0036`), Payment Gateways, HD Crypto Treasury, Hosting Provisioning, Security & RBAC.

---

## Executive Summary

Hostvra has a strong enterprise hosting control panel foundation (50+ Go handlers, 48 Next.js routes, multi-tenant customer isolation, real Nginx/PHP/MySQL/Postfix/Dovecot/Webmail/DNS/SSL/File Manager/Docker/Terminal/Backups/WAF, and Phase 5 HD Crypto Gateway). However, independent inspection uncovered **specific gaps** preventing full parity with Hostinger / SiteGround / WHMCS commercial operations.

---

## Findings Classification

### Critical
1. **No Automated Subscription Dunning / Auto-Renewal / Auto-Suspend Worker** (`apps/api/internal/handlers/billing.go`):
   - Invoices settle cleanly when paid, but no background worker generates 7/3/1-day renewal invoices or suspends overdue subscriptions after a grace period.
2. **EVM ERC20/BEP20 Token Sweeps Lack Controlled Gas Station** (`apps/api/internal/crypto/treasury_engine.go`):
   - Native BNB/ETH/POL sweeps deduct gas automatically, but USDT BEP20/ERC20 deposit addresses have 0 native gas and stall without a policy-capped gas-feeder mechanism.

### High
3. **Simulated Panel Live Update Metadata** (`apps/api/internal/handlers/updates.go:140`):
   - `UpdateHandler.GetStatus` returned hardcoded `"Simulated latest production release metadata (1.1.0)"` instead of querying live `edge-tec/Hostvra` GitHub commit/release APIs.
4. **Missing Commercial Growth & Conversion Infrastructure**:
   - No Coupon & Promotion Engine (with concurrent redemption limits).
   - No Affiliate & Referral Program (with self-referral fraud prevention & commission ledger).
   - No Customer Credit Wallet & Immutable Double-Entry Ledger for Auto-Pay renewals.
   - No Public Real-Time System Status Page (`/status`) backed by live probes.
   - No Legal/Trust pages (`/terms`, `/privacy`, `/refund-policy`, `/sla`, `/acceptable-use-policy`) or dynamic SEO (`sitemap.xml`, `robots.txt`).
   - No unified Homepage -> Plan -> Domain -> Coupon -> Cart -> Direct Checkout (`/order`) funnel.

### Medium
5. **Non-EVM Crypto Auto-Sweep Broadcasters (TRON/BTC/SOL)**:
   - Unique HD derivation, payment detection, and package activation are operational for all 9 assets, while direct raw-tx broadcasting is live for EVM (`BSC`, `ETH`, `Polygon`) and queued for TRON/BTC/SOL.
6. **Live Merchant Certification for Regional Gateways (bKash/Nagad/SSLCommerz/PayPal)**:
   - Server-side verification code exists in `apps/api/internal/payment/gateways.go`, but live production certification requires live merchant API credentials.

### Low / Informational
7. **WAF Real ModSecurity Enforcement**:
   - ModSecurity config manager (`apps/agent/pkg/waf`) is wired to `/etc/nginx/modsec`; production VPS requires `libnginx-mod-http-modsecurity` + OWASP CRS installed to enforce live blocking beyond simulated probes.
8. **React Hook Exhaustive-Deps Warnings**:
   - Non-blocking ESLint warnings exist in legacy pages during `next build`; zero TypeScript or compilation errors.
