# Fix domain + dashboard

## What was broken
- `ONEX_PUBLIC_HOST` forced dashboard/public URLs to `http://IP:9338` instead of HTTPS domains
- Marketing site status probe ignored `blockchainsystem.com`
- Payments portal had no production status strip and failed silently when nginx returned SPA HTML
- Docker compose defaulted to Nova PG instead of Z Bank
- Live: `blockchainsystem.com` parking DNS; VPS bridge/nginx down without `SSH_PASS`

## Dashboard URLs (after deploy)
| Surface | URL |
|---------|-----|
| Payments dashboard | `/payments/` |
| Online bank | `/wallet/#onlinebank` |
| Production status | `/bridge/production/status` |
| Payments API | `/bridge/payments/status` |

Works on both apex hosts: `zblockchainsystem.com` and `blockchainsystem.com`.

## Go live checklist
1. DNS: `blockchainsystem.com` A `@` + `www` → `51.75.64.28` (remove parking)
2. `gh secret set SSH_PASS ...`
3. `SSH_PASS='...' python3 scripts/auto-deploy-vps.py`
4. `bash scripts/verify-production-domains.sh`
