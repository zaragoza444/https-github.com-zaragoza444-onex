# Production — blockchainsystem.com

| Item | Value |
|------|-------|
| **Domain** | `blockchainsystem.com` |
| **VPS** | `51.75.64.28` |
| **GitHub** | [zaragoza444](https://github.com/zaragoza444) → `https://github.com/zaragoza444/https-github.com-zaragoza444-onex` |
| **Gitea** | [Zaragoza](https://git.anakatech.llc/Zaragoza) → `https://git.anakatech.llc/Zaragoza/onex` |

---

## One-shot VPS bootstrap

```bash
curl -fsSL https://raw.githubusercontent.com/zaragoza444/https-github.com-zaragoza444-onex/main/scripts/production-bootstrap.sh | \
  ONEX_PRODUCTION_DOMAIN=blockchainsystem.com \
  ONEX_STRIPE_SECRET_KEY='sk_live_...' \
  ONEX_STRIPE_PUBLISHABLE_KEY='pk_live_...' \
  ONEX_STRIPE_WEBHOOK_SECRET='whsec_...' \
  bash
```

---

## DNS

See `deploy/dns-records-blockchainsystem.com.md` — point `@` and `www` to `51.75.64.28`.

---

## Stripe webhook

```
https://blockchainsystem.com/bridge/payments/webhook
```

Events: `payment_intent.succeeded`, `payment_intent.payment_failed`

---

## Live URLs

| Page | URL |
|------|-----|
| Portal | https://blockchainsystem.com/payments/ |
| Donate | https://blockchainsystem.com/payments/?page=donate |
| Invoice | https://blockchainsystem.com/payments/?page=invoice |
| Collect | https://blockchainsystem.com/payments/?page=collect |
| Wallet | https://blockchainsystem.com/wallet/ |

---

## GitHub Actions secrets

| Secret | Purpose |
|--------|---------|
| `SSH_PASS` | VPS ubuntu password |
| `ONEX_STRIPE_SECRET_KEY` | Stripe live secret |
| `ONEX_STRIPE_PUBLISHABLE_KEY` | Stripe live publishable |
| `ONEX_STRIPE_WEBHOOK_SECRET` | Stripe webhook signing |

Workflow input: host `51.75.64.28`, branch `main`

---

## Gitea + GitHub Pages wallet

Set repository variable:

```
ONEX_BRIDGE_PUBLIC_URL=https://blockchainsystem.com
```

On Gitea (`Zaragoza/onex`), mirror the same variable for Pages builds.

Push to both remotes:

```powershell
$env:GITEA_USER='Zaragoza'
.\scripts\push-all.ps1
```
