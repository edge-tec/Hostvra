# ResellerClub Production Deployment Checklist

Before switching Hostvra's domain subsystem from `sandbox` to `production`, execute every verification step in this checklist to ensure complete financial and operational integrity.

---

## 1. Production Deployment Checklist

| Step | Verification Task | Verified? |
| :--- | :--- | :---: |
| **1** | Production ResellerClub account funded with sufficient balance | [ ] |
| **2** | Production Reseller ID and API Key generated | [ ] |
| **3** | Hostvra production server public IPv4 whitelisted in ResellerClub API Settings | [ ] |
| **4** | TLD retail pricing and wholesale costs configured in Hostvra Admin Panel | [ ] |
| **5** | Payment gateways (bKash, Stripe, PayPal) verified with real webhook secrets | [ ] |
| **6** | Fail-fast startup validation tested | [ ] |
| **7** | Automated background reconciliation cron enabled | [ ] |

---

## 2. Server Configuration

Update `/etc/hostvra/hostvra.env` or `/opt/hostvra/.env`:

```bash
# Enable Production ResellerClub Provider
DOMAIN_PROVIDER=resellerclub
RESELLERCLUB_MODE=production
RESELLERCLUB_RESELLER_ID=YOUR_PRODUCTION_RESELLER_ID
RESELLERCLUB_API_KEY=YOUR_PRODUCTION_API_KEY
RESELLERCLUB_TIMEOUT_SECONDS=20

# Automated Sync
DOMAIN_RECONCILE_INTERVAL_HOURS=6
```

---

## 3. Fail-Fast Production Startup

Hostvra includes strict fail-fast validation in `apps/api/internal/config/config.go`.
If `ENV=production` or `RESELLERCLUB_MODE=production`:
- If `RESELLERCLUB_RESELLER_ID` is empty or invalid, the API refuses to launch.
- If `RESELLERCLUB_API_KEY` is empty, the API refuses to launch.
- If PostgreSQL connection fails, fallback to in-memory store is blocked.

This guarantees that domains are never registered against test sandboxes or with mock stores in a production environment.

---

## 4. Verifying Production Connection

After restarting the API server:
1. Log into Hostvra Control Panel with an admin or owner account.
2. Navigate to **Domain Reseller** (`/admin/domains`).
3. Click **Test Connection**.
4. Confirm:
   - Status: `Verified live connection`
   - Mode: `PRODUCTION`
   - Gateway: `https://httpapi.com/api/`
   - Latency: Measured in milliseconds (typically 100-300ms)
