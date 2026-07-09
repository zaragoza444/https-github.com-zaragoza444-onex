# One-time setup — then every push to `main` deploys automatically

## 1. Add GitHub secret (one time, ~30 seconds)

On your PC (with [GitHub CLI](https://cli.github.com/) logged in):

```bash
gh secret set SSH_PASS --repo zaragoza444/https-github.com-zaragoza444-onex --body "YOUR_UBUNTU_VPS_PASSWORD"
```

Optional Stripe secrets:

```bash
gh secret set ONEX_STRIPE_SECRET_KEY --body "sk_live_..."
gh secret set ONEX_STRIPE_PUBLISHABLE_KEY --body "pk_live_..."
gh secret set ONEX_STRIPE_WEBHOOK_SECRET --body "whsec_..."
```

## 2. Automatic deploy

After step 1, **every push to `main`** runs `.github/workflows/auto-deploy-production.yml` which:

- SSHs to `51.75.64.28`
- Pulls latest code
- Runs `scripts/fix-bridge-9338.sh`
- Configures nginx for `zblockchainsystem.com/payments/`

Manual trigger: **Actions → Auto-deploy production → Run workflow**

## 3. Or deploy from your PC now

```bash
cd onex
SSH_PASS='YOUR_VPS_PASSWORD' python3 scripts/auto-deploy-vps.py
```

## Verify

```bash
curl -s http://zblockchainsystem.com/bridge/payments/status
```

Expect JSON with `"enabled": true`.
