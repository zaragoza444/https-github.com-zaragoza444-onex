# Payment Gateway — Production Go-Live Checklist

## Status (automated preflight)

| Check | Expected | Notes |
|-------|----------|-------|
| Code on `main` | ✓ | Payment gateway merged |
| VPS node `:8545` | ✓ | `51.75.64.28` responds |
| VPS bridge `:9338` | **Restart required** | Connection reset — run go-live script |
| DNS `onexproduction.com` | **Fix required** | Currently parking IPs, not VPS |
| DNS `novatrustee.digital` | **Fix required** | Currently parking IPs, not VPS |
| Stripe live keys | **Your action** | Add to server env for real cards |

## One-shot VPS deploy (web console)

Log into your VPS provider web console as `ubuntu`, then:

```bash
curl -fsSL https://raw.githubusercontent.com/zaragoza444/https-github.com-zaragoza444-onex/main/scripts/go-live-payment-gateway.sh | bash
```

Or if repo already cloned:

```bash
cd ~/onex && git pull origin main && bash scripts/go-live-payment-gateway.sh
```

## DNS (required for HTTPS domains)

Point **one** A record per domain to your VPS IPv4 (`51.75.64.28`):

| Domain | Type | Value |
|--------|------|-------|
| `onexproduction.com` | A | `51.75.64.28` |
| `www.onexproduction.com` | A | `51.75.64.28` |
| `novatrustee.digital` | A | `51.75.64.28` |

Remove parking IPs (`107.161.23.204`, `198.251.81.30`, `209.141.38.71`).

Then TLS + nginx:

```bash
cd ~/onex
ONEX_PRODUCTION_DOMAIN=onexproduction.com CERTBOT_EMAIL=hello@onexproduction.com ./scripts/deploy-onexproduction.sh
```

## Stripe (live Visa / Mastercard / Amex)

Add to `/etc/onex/onex.env` or `~/onex/.env`:

```bash
ONEX_STRIPE_SECRET_KEY=sk_live_...
ONEX_STRIPE_PUBLISHABLE_KEY=pk_live_...
ONEX_STRIPE_WEBHOOK_SECRET=whsec_...
```

Stripe Dashboard → Webhooks → endpoint:

`https://onexproduction.com/bridge/payments/webhook`

Events: `payment_intent.succeeded`, `payment_intent.payment_failed`

Restart bridge:

```bash
sudo systemctl restart onex-bridge
# or: docker compose -f docker-compose.prod.yml restart onex-bridge
```

## Live URLs (after DNS + bridge restart)

| Page | URL |
|------|-----|
| Portal | https://onexproduction.com/payments/ |
| Donate | https://onexproduction.com/payments/?page=donate |
| Invoice | https://onexproduction.com/payments/?page=invoice |
| Collect | https://onexproduction.com/payments/?page=collect |
| API | https://onexproduction.com/bridge/payments/status |

IP-only (until DNS fixed): http://51.75.64.28:9338/payments/

## GitHub Actions deploy (optional)

1. Repo → Settings → Secrets → add `SSH_PASS` (ubuntu VPS password)
2. Actions → **Deploy payment gateway (production VPS)** → Run workflow

## Verify

```bash
curl -s http://51.75.64.28:9338/bridge/payments/status
curl -s -o /dev/null -w "%{http_code}\n" http://51.75.64.28:9338/payments/
```
