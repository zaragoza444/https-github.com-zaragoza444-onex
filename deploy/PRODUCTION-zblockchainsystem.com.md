# Production — zblockchainsystem.com

| Item | Value |
|------|-------|
| **Domain** | `zblockchainsystem.com` |
| **VPS** | `51.75.64.28` |
| **GitHub** | `zaragoza444` |
| **Gitea** | `Zaragoza` @ git.anakatech.llc |

---

## VPS bootstrap

```bash
curl -fsSL https://raw.githubusercontent.com/zaragoza444/https-github.com-zaragoza444-onex/main/scripts/production-bootstrap.sh | \
  ONEX_PRODUCTION_DOMAIN=zblockchainsystem.com \
  ONEX_STRIPE_SECRET_KEY='sk_live_...' \
  ONEX_STRIPE_PUBLISHABLE_KEY='pk_live_...' \
  ONEX_STRIPE_WEBHOOK_SECRET='whsec_...' \
  bash
```

---

## Stripe webhook

```
https://zblockchainsystem.com/bridge/payments/webhook
```

---

## DNS

See `deploy/dns-records-zblockchainsystem.com.md`
