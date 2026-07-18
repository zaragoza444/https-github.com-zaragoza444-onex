# DNS — zblockchainsystem.com → OneX VPS

**Production domain:** `zblockchainsystem.com`  
**Deploy / SSH host:** `zblockchainsystem.com` (must resolve to your VPS IPv4)  
**GitHub:** [zaragoza444](https://github.com/zaragoza444)  
**Gitea:** [Zaragoza/onex](https://git.anakatech.llc/Zaragoza/onex)

---

## Required records

| Type | Host / name | Value | TTL |
|------|-------------|-------|-----|
| A | `@` | *your VPS IPv4* (from hosting panel) | 300 |
| A | `www` | *same VPS IPv4* | 300 |

**Do not** point `@` at parking IPs (`76.53.10.34`, `76.223.54.146`, `13.248.169.48`) or Railway unless the bridge runs there.

---

## Verify

```bash
dig +short zblockchainsystem.com
# Expected: your VPS IPv4 (not parking / Railway)
bash scripts/verify-production-domains.sh
```

---

## TLS (after DNS)

```bash
cd ~/onex
ONEX_PRODUCTION_DOMAIN=zblockchainsystem.com \
CERTBOT_EMAIL=hello@zblockchainsystem.com \
bash scripts/deploy-zblockchainsystem.sh
```

---

## Live URLs (use domain only — no raw IP in links)

| Service | URL |
|---------|-----|
| Dashboards hub | https://zblockchainsystem.com/dashboards/ |
| Payment portal | https://zblockchainsystem.com/payments/ |
| Payment admin | https://zblockchainsystem.com/payments/dashboard/ |
| Donations | https://zblockchainsystem.com/payments/?page=donate |
| Wallet | https://zblockchainsystem.com/wallet/ |
| Stripe webhook | https://zblockchainsystem.com/bridge/payments/webhook |
| API status | https://zblockchainsystem.com/bridge/payments/status |

---

## GitHub Pages wallet

```
ONEX_BRIDGE_PUBLIC_URL=https://zblockchainsystem.com
```
