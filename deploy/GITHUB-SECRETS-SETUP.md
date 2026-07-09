# GitHub Secrets — blockchainsystem.com production

| Item | Value |
|------|-------|
| Domain | **blockchainsystem.com** |
| GitHub | **zaragoza444** |
| Gitea | **Zaragoza** @ git.anakatech.llc |
| VPS | **51.75.64.28** |

Repo: **https://github.com/zaragoza444/https-github.com-zaragoza444-onex**

## Steps (2 minutes)

1. Open **Settings → Secrets and variables → Actions**
2. Click **New repository secret** for each row:

| Name | Value | Where to get it |
|------|-------|-----------------|
| `SSH_PASS` | Your ubuntu VPS password | OVH / hosting panel for `51.75.64.28` |
| `ONEX_STRIPE_SECRET_KEY` | `sk_live_...` | [Stripe → API keys](https://dashboard.stripe.com/apikeys) (live mode) |
| `ONEX_STRIPE_PUBLISHABLE_KEY` | `pk_live_...` | Same page |
| `ONEX_STRIPE_WEBHOOK_SECRET` | `whsec_...` | After webhook created (see below) |

3. Go to **Actions → Deploy payment gateway (production VPS) → Run workflow**
4. Leave host as `51.75.64.28`, branch `main`, click **Run workflow**

## Stripe webhook (if you don't have whsec yet)

After first deploy (bridge must be up):

```bash
# On VPS:
export ONEX_STRIPE_SECRET_KEY=sk_live_YOUR_KEY
bash ~/onex/scripts/setup-stripe-webhook.sh
```

Or manually in Stripe Dashboard:

- URL: `https://blockchainsystem.com/bridge/payments/webhook`
- Events: `payment_intent.succeeded`, `payment_intent.payment_failed`

Then add `whsec_...` as GitHub secret `ONEX_STRIPE_WEBHOOK_SECRET` and re-run the workflow.

## DNS (required for HTTPS)

Point `onexproduction.com` A record → `51.75.64.28` (remove parking IPs).

## Verify

```bash
curl -s http://51.75.64.28:9338/bridge/payments/status
curl -s https://onexproduction.com/payments/
```

## Immediate fix (no GitHub — VPS web console)

Paste this in your VPS web console as `ubuntu`:

```bash
curl -fsSL https://raw.githubusercontent.com/zaragoza444/https-github.com-zaragoza444-onex/main/scripts/production-bootstrap.sh | bash
```

With Stripe keys in one shot:

```bash
curl -fsSL https://raw.githubusercontent.com/zaragoza444/https-github.com-zaragoza444-onex/main/scripts/production-bootstrap.sh | \
  ONEX_STRIPE_SECRET_KEY='sk_live_PASTE' \
  ONEX_STRIPE_PUBLISHABLE_KEY='pk_live_PASTE' \
  ONEX_STRIPE_WEBHOOK_SECRET='whsec_PASTE' \
  bash
```
