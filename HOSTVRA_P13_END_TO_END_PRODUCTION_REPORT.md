# HOSTVRA — P13 END-TO-END PRODUCTION REPORT

**Execution Date & Time:** 2026-11-20T17:42:00+06:00

---

## 1. Live Production vs. Staging End-to-End Verification Matrix

| Workflow / Endpoint | Environment Tested | Actual Result / HTTP Status | Status |
| :--- | :--- | :--- | :--- |
| `GET https://hostvra.com/api/v1/public/status` | **Live Production VPS** | `HTTP 200 OK` (`overall_status: "operational"`, 1596s uptime, 7 active crypto networks) | **PASS (Live)** |
| `hostvra.com:443` TLS Handshake | **Live Production VPS** | Valid Let's Encrypt X.509 cert (`notAfter=Dec 30 14:33:57 2026 GMT`) | **PASS (Live)** |
| `GET https://hostvra.com/admin/commercial` (Unauthenticated) | **Live Production VPS** | `HTTP 307` redirect to `/login?redirect=%2Fadmin%2Fcommercial` | **PASS (Live)** |
| Public Commercial & Legal Routes (`/order`, `/status`, `/terms`, `/privacy`, `/refund-policy`, `/sla`, `/acceptable-use-policy`) | **Local Next.js Build + Live Probe** | Live probe caught `HTTP 307` redirect bug in `middleware.ts`; fixed `PUBLIC_PATHS` in `apps/web/src/middleware.ts` and verified 59/59 Next.js build. | **FIXED IN P13 (Pending VPS Pull)** |
| Referral Signup -> Coupon Checkout -> Wallet Payment -> Provisioning -> Affiliate Commission | **Local Go Race Suite (`-race`)** | `TestCommercial_EndToEndReferralCheckoutAndCommission` (`HTTP 201 Created`, `$20 -> $15` discount, `$2.25` commission, provisioning hook fired) | **PASS** |
| Rejected Withdrawal Replay Exploit Block & 5-Way Concurrent Approval | **Local Go Race Suite (`-race`)** | `TestP10_WithdrawalRejectThenApproveBlockedAndConcurrentApproval` (`HTTP 400` on rejected re-approval, exact `$20.00` single credit) | **PASS** |
| HMAC-Signed Payment & Refund Webhook -> Commission Reversal | **Local Go Race Suite (`-race`)** | `TestP10_WebhookRefundReversesAffiliateCommission` (`HTTP 200 OK`, `$15.00 -> $0.00`) | **PASS** |
| On-Disk `store.json` Persistence & Disaster Recovery Reload | **Local Go Race Suite (`-race`)** | `TestP13_RealOnDiskDatabaseBackupAndRestore` (`41,079 bytes`, `RTO = 2.97ms`, `RPO = 0`) | **PASS** |
