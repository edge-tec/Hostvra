# ResellerClub Integration Guide for Hostvra

## Architecture Overview

Hostvra provides a production-grade, zero-mock domain registrar subsystem that directly communicates with the **ResellerClub HTTP API** (LogicBoxes platform).

The architecture separates provider concerns from core business logic via the `DomainRegistrar` interface:

```
+-------------------------------------------------------------+
|                      Hostvra API Layer                       |
|   /api/v1/domains/*  <--->  internal/domains/service.go     |
+-------------------------------------------------------------+
                               |
               +---------------+---------------+
               |                               |
       [DomainRegistrar]               [Store / PostgreSQL]
        Interface Layer                 Domain Entity State
               |
    +----------+----------+
    |                     |
[Sandbox Client]    [Production Client]
https://test.httpapi.com   https://httpapi.com
```

---

## 1. Credentials & Configuration

To integrate with ResellerClub, you need:
1. **Reseller ID**: Your numerical Reseller ID from the ResellerClub Control Panel.
2. **API Key**: Generated from ResellerClub Control Panel under *Settings > API*.
3. **Server IP Whitelist**: ResellerClub mandates that all API calls originate from authorized IP addresses.

### Environment Variables

Add the following configuration to your server's `.env` file:

```bash
# Domain Registrar System Configuration
DOMAIN_PROVIDER=resellerclub

# Mode: "sandbox" (testing) or "production" (live)
RESELLERCLUB_MODE=sandbox

# Your numerical Reseller ID
RESELLERCLUB_RESELLER_ID=123456

# Your ResellerClub API Key
RESELLERCLUB_API_KEY=AbCdEfGhIjKlMnOpQrStUvWxYz123456

# Client HTTP Timeout (seconds)
RESELLERCLUB_TIMEOUT_SECONDS=20

# Automated Background Sync / Reconciliation Interval (hours)
DOMAIN_RECONCILE_INTERVAL_HOURS=6
```

---

## 2. API Endpoints

Hostvra automatically switches base URLs depending on `RESELLERCLUB_MODE`:

| Mode | Base URL | LogicBoxes Gateway |
| :--- | :--- | :--- |
| **Sandbox** | `https://test.httpapi.com/api/` | ResellerClub Demo Engine |
| **Production** | `https://httpapi.com/api/` | Global Live Registry Gateway |

---

## 3. IP Whitelisting (Mandatory)

ResellerClub validates every API call against your whitelisted server IP address.

### In Sandbox:
1. Log in to [https://test.httpapi.com/reseller](https://test.httpapi.com/reseller).
2. Navigate to **Settings > API**.
3. Under **Authorized IP Addresses**, add your staging/development server's public IPv4 address.

### In Production:
1. Log in to [https://httpapi.com/reseller](https://httpapi.com/reseller).
2. Navigate to **Settings > API**.
3. Under **Authorized IP Addresses**, add your live Hostvra production server's public IPv4 address.
4. If your server uses multiple egress IPs or Cloudflare proxies, whitelist the direct outgoing public IP.

---

## 4. Lifecycle Capabilities

The Hostvra ResellerClub driver implements the full domain lifecycle:

1. **Customer Account Provisioning**: Automated creation and mapping of customer IDs (`/customers/v2/signup.json` or `/customers/search.json`).
2. **Contact Creation**: Automated creation of Registrant, Administrative, Technical, and Billing contact IDs (`/contacts/add.json`).
3. **Availability Checks**: Real-time multi-TLD availability queries (`/domains/available.json`).
4. **Instant Registration**: Automated registration upon verified payment invoice (`/domains/register.json`).
5. **Nameserver Orchestration**: Real-time nameserver queries and modifications (`/domains/modify-ns.json`).
6. **DNS Record Management**: Full CRUD on A, AAAA, CNAME, MX, TXT, and SRV records (`/dns/manage/add-*.json`, `/dns/manage/delete-*.json`).
7. **Theft Protection / Registrar Lock**: Enable or disable client transfer locks (`/domains/enable-theft-protection.json`, `/domains/disable-theft-protection.json`).
8. **Auth / EPP Code**: Secure retrieval of transfer authorization codes (`/domains/locks.json`).
9. **Domain Renewals**: Subscription extension for 1 to 10 years (`/domains/renew.json`).
10. **Inbound Transfers**: Secure transfer initiation with encrypted auth codes (`/domains/transfer.json`).
11. **Background Reconciliation**: Automated synchronization loop verifying registry expiration dates and status with Hostvra PostgreSQL database.
