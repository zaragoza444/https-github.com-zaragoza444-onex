# Production domains — Z Bank full stack (HTTP 200)

Both apex hostnames serve the **same** live bridge / payments / wallet stack:

| Surface | blockchainsystem.com | zblockchainsystem.com |
|---------|----------------------|------------------------|
| Payments | http://blockchainsystem.com/payments/ | https://zblockchainsystem.com/payments/ |
| Bridge API | http://blockchainsystem.com/bridge/ | https://zblockchainsystem.com/bridge/ |
| Wallet | http://blockchainsystem.com/wallet/ | https://zblockchainsystem.com/wallet/ |
| Bank status | …/bridge/bank/status | …/bridge/bank/status |
| Health | …/health | …/health |

Nginx: `deploy/nginx-vps-zblockchain.conf` (both names → `onex-bridge`).

## Legacy redirects (not production fronts)

- `onexproduction.com` / `www.onexproduction.com`
- `novatrustee.digital`

→ 301 to `zblockchainsystem.com`.

## Env

```bash
ONEX_PRODUCTION_DOMAIN=zblockchainsystem.com
ONEX_CORS_ORIGINS=https://zblockchainsystem.com,https://www.zblockchainsystem.com,https://blockchainsystem.com,https://www.blockchainsystem.com,http://blockchainsystem.com,http://www.blockchainsystem.com,https://git.anakatech.llc,https://zaragoza444.github.io
```

Prefer `deploy/env.zbank.production.example`.

## If blockchainsystem.com shows a lander

DNS is still on parking. See `deploy/FIX-blockchainsystem.com.md`.
