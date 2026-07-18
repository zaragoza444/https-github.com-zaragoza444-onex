# One-time setup — then every push to `main` deploys automatically

## 1. Add GitHub secret (required for auto-deploy)

```bash
gh secret set SSH_PASS --repo zaragoza444/https-github.com-zaragoza444-onex --body "YOUR_UBUNTU_VPS_PASSWORD"
```

Optional Stripe secrets:

```bash
gh secret set ONEX_STRIPE_SECRET_KEY --body "sk_live_..."
gh secret set ONEX_STRIPE_PUBLISHABLE_KEY --body "pk_live_..."
gh secret set ONEX_STRIPE_WEBHOOK_SECRET --body "whsec_..."
```

Officer PIN/signature can live only on the VPS in `/etc/onex/onex.env` (`ONEX_ZBANK_OFFICER_PIN`, `ONEX_ZBANK_OFFICER_SIGNATURE`).

## 2. Automatic deploy

Every push to `main` runs `.github/workflows/auto-deploy-production.yml` which SSHs to `51.75.64.28` and runs:

```bash
bash scripts/fix-all-system.sh
```

That script:

- Pulls `main`, builds `onex-bridge`
- Ensures Z Bank ledger + Stripe PG + officer paths
- Preserves existing Stripe/officer secrets in `/etc/onex/onex.env`
- Restarts systemd `onex-bridge`
- Installs nginx Z Bank routes (`/payments/`, `/bridge/`, `/wallet/`) and redirects `/` → `/payments/`

## 3. Deploy / fix from your PC now

```bash
cd onex
SSH_PASS='YOUR_VPS_PASSWORD' python3 scripts/auto-deploy-vps.py
```

Or SSH then:

```bash
cd ~/onex && git pull origin main && bash scripts/fix-all-system.sh
```

## Verify

```bash
curl -s http://zblockchainsystem.com/bridge/payments/status
curl -s http://zblockchainsystem.com/bridge/bank/officer/status
curl -sI http://zblockchainsystem.com/payments/assets/zbank-logo.png
```

Expect JSON with `"framework":"zbank"` / `"enabled": true`, not the Nova HTML SPA.

## Canonical domain

All public URLs use **zblockchainsystem.com**. Legacy domains 301 redirect. See `deploy/CANONICAL-DOMAIN.md`.
