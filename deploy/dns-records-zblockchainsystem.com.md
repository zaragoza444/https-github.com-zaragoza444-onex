# DNS — zblockchainsystem.com → OneX VPS

**Production domain:** `zblockchainsystem.com`  
**DNS provider:** Cloudflare — see **`deploy/DNS-CLOUDFLARE-zblockchainsystem.com.md`** for step-by-step  
**VPS origin IPv4:** `51.75.64.28` (A record target — not used in public app URLs)  
**Deploy / SSH host:** `zblockchainsystem.com` (after DNS points to VPS)  
**GitHub:** [zaragoza444](https://github.com/zaragoza444)  
**Gitea:** [Zaragoza/onex](https://git.anakatech.llc/Zaragoza/onex)

---

## Required records

| Type | Host / name | Value | TTL |
|------|-------------|-------|-----|
| A | `@` | `51.75.64.28` | Auto |
| A | `www` | `51.75.64.28` | Auto |

Set **DNS only** (grey cloud) in Cloudflare while testing. See `deploy/DNS-CLOUDFLARE-zblockchainsystem.com.md`.

---

## Verify

```bash
dig +short zblockchainsystem.com
# Expected: 51.75.64.28 (grey cloud) or Cloudflare proxy IPs (orange cloud + correct origin)
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
