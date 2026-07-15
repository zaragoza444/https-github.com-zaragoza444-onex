# DNS — blockchainsystem.com → OneX VPS (REQUIRED FOR HTTP 200)

> **LIVE STATUS:** If `dig +short blockchainsystem.com` shows `76.223.54.146` or `13.248.169.48`, the domain is still on a **parking lander**. Replace those A records with `51.75.64.28` only, then run `bash scripts/fix-all-system.sh`.

**Production domain:** `blockchainsystem.com`  
**Production domain:** `blockchainsystem.com`  
**VPS IPv4:** `51.75.64.28`  
**GitHub:** [zaragoza444](https://github.com/zaragoza444)  
**Gitea:** [Zaragoza/onex](https://git.anakatech.llc/Zaragoza/onex)

---

## Required records

At your domain registrar for **blockchainsystem.com**:

| Type | Host / name | Value | TTL |
|------|-------------|-------|-----|
| A | `@` | `51.75.64.28` | 300 |
| A | `www` | `51.75.64.28` | 300 |

**Remove** any parking or third-party A records (e.g. `76.223.54.146`, `13.248.169.48`) so only your VPS IP remains.

---

## Verify propagation

```bash
dig +short blockchainsystem.com
dig +short www.blockchainsystem.com
# Expected: 51.75.64.28
```

---

## TLS (after DNS resolves)

On the VPS:

```bash
cd ~/onex
ONEX_PRODUCTION_DOMAIN=blockchainsystem.com \
CERTBOT_EMAIL=hello@blockchainsystem.com \
./scripts/deploy-blockchainsystem.sh
```

---

## Live URLs (after DNS + TLS)

| Service | URL |
|---------|-----|
| Site | https://blockchainsystem.com/ |
| Wallet | https://blockchainsystem.com/wallet/ |
| **Payment portal** | https://blockchainsystem.com/payments/ |
| Donations | https://blockchainsystem.com/payments/?page=donate |
| Invoices | https://blockchainsystem.com/payments/?page=invoice |
| Collections | https://blockchainsystem.com/payments/?page=collect |
| Payment API | https://blockchainsystem.com/bridge/payments/status |
| Stripe webhook | https://blockchainsystem.com/bridge/payments/webhook |

IP-only (before DNS): http://51.75.64.28:9338/payments/

---

## Static wallet (GitHub / Gitea Pages)

| Host | Bridge URL variable |
|------|---------------------|
| GitHub Pages (`zaragoza444.github.io/onex`) | `ONEX_BRIDGE_PUBLIC_URL=https://blockchainsystem.com` |
| Gitea Pages (`git.anakatech.llc/Zaragoza/onex`) | Same |

GitHub: Repo → Settings → Secrets and variables → Actions → Variable `ONEX_BRIDGE_PUBLIC_URL`

---

## Email (optional)

Configure routing for `@blockchainsystem.com` via Cloudflare Email Routing or your registrar (see `docs/BUSINESS-EMAIL.md`).