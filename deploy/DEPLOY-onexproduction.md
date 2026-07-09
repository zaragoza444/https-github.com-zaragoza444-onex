# OneX Production Platform

Full production stack: **node + wallet bridge + real ledger + token platform** on one HTTPS domain.

## DNS

| Type | Name | Value |
|------|------|--------|
| A | `@` or `onexproduction` | Your VPS public IPv4 |

## Deploy on VPS

```bash
git clone https://github.com/zaragoza444/onex.git
cd onex
cp deploy/env.onexproduction.example .env
# Edit ONEX_API_KEY

sudo certbot certonly --standalone -d onexproduction.com -d www.onexproduction.com
mkdir -p deploy/certs
sudo cp /etc/letsencrypt/live/onexproduction.com/fullchain.pem deploy/certs/
sudo cp /etc/letsencrypt/live/onexproduction.com/privkey.pem deploy/certs/

docker compose -f docker-compose.prod.yml --profile proxy up -d --build
```

Or: `CERTBOT_EMAIL=you@example.com ./scripts/deploy-onexproduction.sh`

## URLs

| Service | URL |
|---------|-----|
| Wallet | https://onexproduction.com/wallet/ |
| **Payment Gateway** | https://onexproduction.com/payments/ |
| Donations | https://onexproduction.com/payments/?page=donate |
| Invoice payments | https://onexproduction.com/payments/?page=invoice |
| Collections | https://onexproduction.com/payments/?page=collect |
| Marketing site | https://onexproduction.com/ |
| Contact | https://onexproduction.com/contact.html |
| Business email | hello@onexproduction.com · business@onexproduction.com |
| Real Ledger | https://onexproduction.com/wallet/#ledger |
| Production status | https://onexproduction.com/bridge/production/status |
| Token platform | https://onexproduction.com/bridge/platform/status |

## Connect static wallet (GitHub / Gitea Pages)

```powershell
.\scripts\connect-onexproduction.ps1 -ProductionUrl "https://onexproduction.com"
```

Set GitHub variable `ONEX_BRIDGE_PUBLIC_URL=https://onexproduction.com` so Pages wallet uses this bridge.

## Verify

```bash
curl -s https://onexproduction.com/bridge/production/status
curl -s https://onexproduction.com/bridge/payments/status
curl -s -o /dev/null -w "%{http_code}\n" https://onexproduction.com/payments/
```
