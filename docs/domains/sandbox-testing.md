# ResellerClub Sandbox Testing Guide

This guide details how to verify domain search, checkout, registration, DNS, lock, and renewal operations using the **ResellerClub Sandbox Environment** before deploying to production.

---

## 1. Prerequisites

1. Create a free ResellerClub test account at: [https://test.httpapi.com/reseller](https://test.httpapi.com/reseller)
2. Retrieve your **Test Reseller ID** and generate a **Test API Key**.
3. Whitelist your workstation or staging server IP under *Settings > API*.
4. In your Hostvra `.env` file, set:
   ```bash
   DOMAIN_PROVIDER=resellerclub
   RESELLERCLUB_MODE=sandbox
   RESELLERCLUB_RESELLER_ID=your_test_reseller_id
   RESELLERCLUB_API_KEY=your_test_api_key
   ```
5. Restart the Hostvra API server:
   ```bash
   systemctl restart hostvra-api
   ```

---

## 2. Testing Domain Availability Search

### From Web UI:
1. Navigate to `/domains` in the Hostvra Control Panel.
2. Select the **Register Domain** tab.
3. Search for a test domain, e.g. `mytestcorp2026.com`.
4. Verify that:
   - Available domains show a green checkmark and Hostvra retail pricing.
   - Unavailable domains show a gray status.
   - TLD pricing reflects the PostgreSQL database pricing table.

### Via cURL / API:
```bash
curl -s -H "Authorization: Bearer <YOUR_JWT_TOKEN>" \
  "http://localhost:8080/api/v1/domains/search?query=mytestcorp2026.com" | jq .
```

---

## 3. Testing Registration Checkout & Provisioning

1. In the search results, click **Register Now** on an available domain.
2. Select registration duration (1-10 years).
3. Choose nameserver configuration:
   - Default Hostvra Nameservers (`ns1.hostvra.com`, `ns2.hostvra.com`)
   - Or custom nameservers.
4. Fill in or confirm the Registrant Contact form.
5. Select a payment gateway (e.g. `bKash`, `stripe`, or `balance`).
6. Click **Confirm Order & Pay**.
7. An invoice will be generated. Once paid (or simulated payment webhook received), the asynchronous registration worker will:
   - Acquire advisory lock on the domain order.
   - Provision customer and contact on ResellerClub test API.
   - Execute `/domains/register.json`.
   - Update order status to `completed` and domain status to `active`.

---

## 4. Testing Nameserver & DNS Updates

1. Go to the **My Registered Domains** tab on `/domains`.
2. Find the registered domain and click **Manage**.
3. Under the **Nameservers** tab, enter custom nameservers and click **Save Nameservers**.
4. Under the **DNS Records** tab, add an `A` record pointing `@` to your server IP.
5. Verify in ResellerClub test control panel that the records are reflected immediately.

---

## 5. Testing Registrar Theft Lock & EPP Code

1. In the domain management modal, switch to **Lock & EPP Code**.
2. Toggle the **Registrar Transfer Lock** to Unlocked, then Locked. Verify success notification.
3. Click **Reveal Authorization Code**. Verify the EPP auth code is returned directly from the registry.

---

## 6. Testing Background Reconciliation

To verify that Hostvra automatically detects expiry date extensions or status updates from ResellerClub:
1. Navigate to `/admin/domains`.
2. Click **Sync & Reconcile**.
3. Check the audit log to verify sync updates were recorded.
