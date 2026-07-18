# DNS — blockchainsystem.com → OneX VPS (REQUIRED FOR HTTP 200)

> **LIVE STATUS:** If `dig +short blockchainsystem.com` shows `76.223.54.146` or `13.248.169.48`, the domain is still on a **parking lander**. Point A records to the **same VPS IPv4 as `zblockchainsystem.com`**, then run `bash scripts/fix-all-system.sh`.

**Sister domain:** `blockchainsystem.com` (same VPS as `zblockchainsystem.com`)  
**Canonical domain:** `zblockchainsystem.com`  
**GitHub:** [zaragoza444](https://github.com/zaragoza444)  
**Gitea:** [Zaragoza/onex](https://git.anakatech.llc/Zaragoza/onex)

---

## Required records

At your domain registrar for **blockchainsystem.com**:

| Type | Host / name | Value | TTL |
|------|-------------|-------|-----|
| A | `@` | *same IPv4 as `dig +short zblockchainsystem.com`* | 300 |
| A | `www` | *same IPv4* | 300 |

**Remove** parking A records (`76.223.54.146`, `13.248.169.48`).

---

## Verify propagation

```bash
dig +short blockchainsystem.com
dig +short zblockchainsystem.com
# Both should return the same VPS IPv4
bash scripts/verify-production-domains.sh
```

---

## Live URLs

| Service | URL |
|---------|-----|
| Payment portal | https://blockchainsystem.com/payments/ |
| Dashboards | https://zblockchainsystem.com/dashboards/ |
| API status | https://blockchainsystem.com/bridge/payments/status |

Use **https://zblockchainsystem.com** as the canonical public host in env and Stripe webhooks.
