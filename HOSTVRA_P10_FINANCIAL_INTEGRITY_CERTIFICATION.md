# HOSTVRA — P10 FINANCIAL INTEGRITY CERTIFICATION

**Certification Date:** 2026-11-20  
**Scope:** Credit Wallet, Coupons, Order Checkout, Affiliate Commissions, Withdrawals, Webhooks, Refunds

---

## 1. Financial Controls Verification Matrix

| Control | Implementation | Verification Method | Status |
| :--- | :--- | :--- | :--- |
| **1. Atomic Wallet Transactions** | `CreditWalletAtomic` / `DebitWalletAtomic` (`SELECT ... FOR UPDATE` in PostgreSQL; mutex in MemoryStore) | `TestCommercial_WalletDoubleSpendPreventionAndPublicStatus` (`-race`) | **PASS** |
| **2. Double-Credit Prevention** | Strict `idempotency_key` unique lookup before mutating balance (`aff_payout:<id>`, `wallet_pay:<id>`) | Code audit + `TestP10_WithdrawalRejectThenApproveBlockedAndConcurrentApproval` | **PASS** |
| **3. Duplicate Payment Prevention** | `inv.Status == InvoiceStatusPaid` guard + transaction replay check + `wallet_pay:<inv_id>` idempotency | `TestBillingWebhook_SecurityAndReplay` + code audit | **PASS** |
| **4. Idempotent Invoice Activation** | `ActivatePaidInvoice`, `SettleCryptoPayment`, `HandleWebhook` check invoice status before provisioning | Code audit + `-race` tests | **PASS** |
| **5. Affiliate Commission Idempotency** | `ON CONFLICT (invoice_id) DO NOTHING` (PostgreSQL) & `affiliateCommissions[invoiceID]` map guard (MemoryStore) | `TestCommercial_AffiliateSelfReferralAndReversal` + `TestCommercial_EndToEndReferralCheckoutAndCommission` | **PASS** |
| **6. Commission Reversal After Refunds** | `HandleWebhook` handles `refunded`/`chargeback`/`reversed` before paid short-circuit & calls `ReverseAffiliateCommissionByInvoice` | `TestP10_WebhookRefundReversesAffiliateCommission` (`-race`) | **PASS** |
| **7. Withdrawal Approval & Rejection Integrity** | `ApproveAffiliateWithdrawal` returns `ErrWithdrawalAlreadyProcessed` if status != `pending`; handler credits wallet ONLY when `req.Approve && wd.Status == "paid"` | `TestP10_WithdrawalRejectThenApproveBlockedAndConcurrentApproval` (`-race`) | **PASS** |
| **8. Concurrent Wallet & Coupon Safety** | Row-level locking (`FOR UPDATE`) / mutex serialization | 10-way concurrent coupon test + 2-way double-spend test + 5-way concurrent withdrawal approval (`-race`) | **PASS** |
| **9. Negative Balance & Price Manipulation Prevention** | Server-side authoritative plan pricing, `before+1e-9 < amountUSD` check, coupon field validation (`0 < pct <= 100`, `fixed > 0`), wallet pre-check before burning single-use coupons | `TestP10_CouponInputValidationAndWalletPrecheckPreservesCoupon` + `TestP10_WalletAdjustAndWithdrawalInputValidation` | **PASS** |
| **10. Live Gateway / On-Chain Mainnet Settlement** | Stripe / bKash / Nagad / Mainnet RPCs | External credentials & live funds not executed in local audit | **NOT VERIFIED** |

---

## 2. Critical & High Financial Defects Discovered and Fixed in P10

1. **CRITICAL — Rejected Withdrawal Re-Approval Wallet Credit Exploit (`apps/api/internal/handlers/commercial.go` & `apps/api/internal/store/commercial_store.go`)**
   - **Root Cause**: `ApproveAffiliateWithdrawal` returned `(wd, nil)` unchanged for already-processed withdrawals. Calling `AdminProcessWithdrawal` with `approve: true` on a previously `"rejected"` `credit_wallet` withdrawal still invoked `CreditWalletAtomic`, creating unbacked wallet credit while the affiliate had already received their rejected balance back.
   - **Fix**: `ApproveAffiliateWithdrawal` now returns `ErrWithdrawalAlreadyProcessed` when `status != "pending"` (both MemoryStore & PostgresStore), and `AdminProcessWithdrawal` additionally requires `wd.Status == "paid"` before crediting the wallet.
   - **Regression Test**: `TestP10_WithdrawalRejectThenApproveBlockedAndConcurrentApproval` (**PASS** with `-race`).

2. **HIGH — Refund Webhooks Ignored & Affiliate Commissions Not Reversed (`apps/api/internal/handlers/billing.go`)**
   - **Root Cause**: `HandleWebhook` returned `200 idempotent_success` whenever `inv.Status == InvoiceStatusPaid`, before inspecting `payload.Status == "refunded"`. Refunds never updated invoice status and never reversed affiliate commissions.
   - **Fix**: Added `InvoiceStatusRefunded` and handled `"refunded"`, `"chargeback"`, and `"reversed"` webhooks prior to the paid short-circuit, invoking `ReverseAffiliateCommissionByInvoice`, recording replay protection, and writing an audit log.
   - **Regression Test**: `TestP10_WebhookRefundReversesAffiliateCommission` (**PASS** with `-race`).

3. **HIGH — Single-Use Coupon Burned on Failed Credit Wallet Checkout (`apps/api/internal/handlers/commercial.go`)**
   - **Root Cause**: `CheckoutOrder` redeemed the coupon and persisted an unpaid invoice before verifying the customer's credit wallet had sufficient balance.
   - **Fix**: Added credit wallet balance pre-check before calling `RedeemCouponAtomic` so insufficient funds return `400` without consuming the coupon or creating an orphan invoice.
   - **Regression Test**: `TestP10_CouponInputValidationAndWalletPrecheckPreservesCoupon` (**PASS** with `-race`).

4. **HIGH — Duplicate Referral Count Inflation in PostgreSQL (`apps/api/internal/store/commercial_store.go`)**
   - **Root Cause**: `PostgresStore.RecordAffiliateReferral` incremented `total_referrals` without checking `res.RowsAffected() > 0` after `ON CONFLICT (referred_user_id) DO NOTHING`.
   - **Fix**: Guarded the `UPDATE affiliate_accounts` with `if n, _ := res.RowsAffected(); n > 0`.
