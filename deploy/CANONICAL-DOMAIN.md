# Canonical domain — zblockchainsystem.com

All public OneX / Z Bank / payments / wallet / bridge surfaces use **one hostname**:

| Surface | URL |
|---------|-----|
| Home / payments | https://zblockchainsystem.com/payments/ |
| Bridge API | https://zblockchainsystem.com/bridge/ |
| Wallet | https://zblockchainsystem.com/wallet/ |
| Bank status | https://zblockchainsystem.com/bridge/bank/status |
| Officer status | https://zblockchainsystem.com/bridge/bank/officer/status |
| Stripe webhook | https://zblockchainsystem.com/bridge/payments/webhook |

## Legacy domains (301 → canonical)

These hostnames must redirect to `zblockchainsystem.com` (path preserved):

- `onexproduction.com` / `www.onexproduction.com`
- `novatrustee.digital`
- `blockchainsystem.com` / `www.blockchainsystem.com`

Nginx: `deploy/nginx-vps-zblockchain.conf` and `deploy/nginx.prod.conf`.

## Env

```bash
ONEX_PRODUCTION_DOMAIN=zblockchainsystem.com
ONEX_CORS_ORIGINS=https://zblockchainsystem.com,https://www.zblockchainsystem.com,https://git.anakatech.llc,https://zaragoza444.github.io,http://51.75.64.28:9338
```

Prefer `deploy/env.zbank.production.example` or `deploy/env.zblockchainsystem.com.example`.
