# ResellerClub Troubleshooting Guide

Common issues encountered when integrating and operating the ResellerClub API, their root causes, and standard remediation procedures.

---

## 1. Common API Error Codes

### `IP_NOT_AUTHORIZED`
- **Root Cause**: The outgoing IP address of your Hostvra server is not in your ResellerClub API Authorized IP list.
- **Fix**: Log into the ResellerClub control panel (*Settings > API*), find your server's current public IP via `curl -s https://api.ipify.org`, and add it to the whitelist.

### `INSUFFICIENT_FUNDS`
- **Root Cause**: Your ResellerClub account debit wallet balance is below the wholesale cost of the requested domain or renewal.
- **Fix**: Fund your ResellerClub account via wire transfer, credit card, or PayPal in the ResellerClub portal. Once funded, click **Retry** on the failed order in `/admin/domains`.

### `DOMAIN_ALREADY_EXISTS`
- **Root Cause**: The domain was registered by another party between the availability check and payment confirmation, or already exists under your reseller account.
- **Fix**: Check domain status via `/domains/whois` or ResellerClub control panel. If registered elsewhere, issue an immediate refund to the customer.

### `INVALID_CONTACT_PHONE`
- **Root Cause**: The registrant phone number did not adhere to international ITU-T E.164 formatting (country code + number without leading 0 or +).
- **Fix**: Hostvra automatically cleans and normalizes phone numbers. Ensure customer inputs a valid country dial code.

---

## 2. Order Provisioning Recovery

If a domain order fails during automatic provisioning (e.g. temporary network timeout or insufficient reseller funds):
1. The domain order status is marked as `failed`.
2. The specific API error message is saved to `domain_orders.error_message`.
3. The customer payment remains recorded as `paid`.
4. Administrators can review failed orders at `/admin/domains` under the **Orders & Provisioning Log** tab.
5. Click **Retry** on the order to safely re-dispatch the registration worker with idempotency safeguards.

---

## 3. Webhook & Payment Reconciliation

Domain registration is strictly decoupled from payment capture:
- Domain orders are initially placed in `pending_payment` status.
- Once a verified payment webhook is received from bKash, Stripe, or PayPal, Hostvra updates the invoice to `paid` and invokes `domainSvc.Registration.ProcessPaidOrder(...)` asynchronously.
- Registration will **never** execute on unconfirmed or unpaid orders.

---

## 4. Manual Health Verification Command

To test connection health directly from the server CLI:

```bash
# Verify API health
curl -s http://localhost:8080/healthz | jq .

# Test ResellerClub connectivity via admin endpoint
curl -s -X POST -H "Authorization: Bearer <ADMIN_JWT_TOKEN>" \
  http://localhost:8080/api/v1/admin/domains/test-connection | jq .
```
