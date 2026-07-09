# DNS — zblockchainsystem.com → OneX VPS

**Production domain:** `zblockchainsystem.com`  
**VPS IPv4:** `51.75.64.28`  
**GitHub:** [zaragoza444](https://github.com/zaragoza444)  
**Gitea:** [Zaragoza/onex](https://git.anakatech.llc/Zaragoza/onex)

---

## Required records

| Type | Host / name | Value | TTL |
|------|-------------|-------|-----|
| A | `@` | `51.75.64.28` | 300 |
| A | `www` | `51.75.64.28` | 300 |

**Current DNS** may show `76.53.10.34` — replace with `51.75.64.28` only.

---

## Verify

```bash
dig +short zblockchainsystem.com
# Expected: 51.75.64.28
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

## Live URLs

| Service | URL |
|---------|-----|
| Payment portal | https://zblockchainsystem.com/payments/ |
| Donations | https://zblockchainsystem.com/payments/?page=donate |
| Invoices | https://zblockchainsystem.com/payments/?page=invoice |
| Collections | https://zblockchainsystem.com/payments/?page=collect |
| Wallet | https://zblockchainsystem.com/wallet/ |
| Stripe webhook | https://zblockchainsystem.com/bridge/payments/webhook |
| API status | https://zblockchainsystem.com/bridge/payments/status |

IP-only (before DNS): http://51.75.64.28:9338/payments/

---

## GitHub / Gitea Pages wallet

```
ONEX_BRIDGE_PUBLIC_URL=https://zblockchainsystem.com
```
