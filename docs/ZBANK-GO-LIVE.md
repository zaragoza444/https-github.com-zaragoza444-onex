# Z Bank — Go-Live Checklist

Production checklist for **Z Bank** on OneX Bridge (`framework: zbank`). Canonical domain: **zblockchainsystem.com**.

## Prerequisites

| Item | Location |
|------|----------|
| CIS (online banking) | `docs/cis/CIS-Z-Bank-Online-v1.md` |
| CIS (DSSBOaT officer) | `docs/cis/CIS-Z-Bank-DSSBOAT-Officer-v1.md` |
| Ledger seed | `configs/bank-ledger.zbank.production.json` |
| Payment gateway | `configs/payment-gateway.zbank.production.json` |
| Officer seed | `configs/zbank-officers.dssboat.example.json` |
| Env template | `deploy/env.zbank.production.example` |

## 1. Server environment

Copy and edit production env (never commit live secrets):

```bash
sudo mkdir -p /etc/onex
sudo cp deploy/env.zbank.production.example /etc/onex/onex.env
sudo nano /etc/onex/onex.env
```

Required variables:

| Variable | Value |
|----------|--------|
| `ONEX_LEDGER_MODE` | `production` |
| `ONEX_ONLINE_BANK` | `1` |
| `ONEX_PAYMENT_GATEWAY` | `1` |
| `ONEX_PAYMENT_GATEWAY_FRAMEWORK` | `zbank` |
| `ONEX_BANK_LEDGER_FILE` | `configs/bank-ledger.zbank.production.json` |
| `ONEX_PAYMENT_GATEWAY_FILE` | `configs/payment-gateway.zbank.production.json` |
| `ONEX_ZBANK_OFFICERS_FILE` | `configs/zbank-officers.dssboat.example.json` |
| `ONEX_ZBANK_OFFICER_PIN` | 4–8 digits (no demo defaults) |
| `ONEX_ZBANK_OFFICER_SIGNATURE` | ≥ 8 characters |
| `ONEX_API_KEY` | Long random secret |
| `ONEX_STRIPE_SECRET_KEY` | `sk_live_…` (for live cards) |
| `ONEX_STRIPE_PUBLISHABLE_KEY` | `pk_live_…` |
| `ONEX_STRIPE_WEBHOOK_SECRET` | `whsec_…` |

## 2. Build and deploy

**Automated (push to `main`):** GitHub Actions runs `scripts/fix-all-system.sh` on the VPS. See `deploy/AUTO-DEPLOY-SETUP.md`.

**Manual on VPS:**

```bash
cd ~/onex && git pull origin main
bash scripts/go-live-payment-gateway.sh
# or full system repair:
bash scripts/fix-all-system.sh
```

**Docker:**

```bash
cp deploy/env.zbank.production.example .env   # edit secrets first
docker compose -f docker-compose.prod.yml --profile proxy up -d --build onex-bridge
```

## 3. Bootstrap officer credentials

After PIN and signature are set in `/etc/onex/onex.env`:

```bash
curl -X POST -H "X-OneX-Api-Key: $ONEX_API_KEY" \
  https://zblockchainsystem.com/bridge/bank/officer/ensure
```

## 4. Stripe webhook

Point Stripe webhook to:

```
https://zblockchainsystem.com/bridge/payments/webhook
```

Use `scripts/setup-stripe-webhook.sh` or configure in the Stripe dashboard.

## 5. Verify production

```bash
bash scripts/verify-zbank-local.sh https://zblockchainsystem.com
```

Or manually:

```bash
curl -s https://zblockchainsystem.com/bridge/payments/status | jq '.framework, .enabled'
curl -s https://zblockchainsystem.com/bridge/bank/accounts | jq '.accounts[].id'
curl -s https://zblockchainsystem.com/bridge/bank/officer/status | jq '.ready, .productionReady'
curl -sI https://zblockchainsystem.com/payments/assets/zbank-logo.png | head -1
```

**Expect:**

- `"framework": "zbank"` and `"enabled": true`
- Accounts include `zbank-usd-checking`, `zbank-usd-safeguarded`, `zbank-usd-treasury`, `zbank-usd-wholesale`
- Officer `ready: true`, `productionReady: true`
- Logo HTTP `200`
- Payments portal at `/payments/` shows Z Bank branding

## 6. Local dev (quick test)

```bash
go build -o bin/onex-bridge ./cmd/onex-bridge
export ONEX_PROJECT_ROOT=$PWD
export ONEX_LEDGER_MODE=production ONEX_ONLINE_BANK=1
export ONEX_BANK_LEDGER_FILE=configs/bank-ledger.zbank.production.json
export ONEX_PAYMENT_GATEWAY=1
export ONEX_PAYMENT_GATEWAY_FRAMEWORK=zbank
export ONEX_PAYMENT_GATEWAY_FILE=configs/payment-gateway.zbank.production.json
export ONEX_ZBANK_OFFICERS_FILE=configs/zbank-officers.dssboat.example.json
export ONEX_ZBANK_OFFICER_PIN=918273
export ONEX_ZBANK_OFFICER_SIGNATURE=ProdSignature-DSSBOAT-01
export ONEX_API_KEY=local-dev-key
./bin/onex-bridge -listen 127.0.0.1:9338
bash scripts/verify-zbank-local.sh http://127.0.0.1:9338
```

## 7. CIS PDF exports

```bash
python3 scripts/generate-cis-pdf.py
```

Generates PDFs under `docs/cis/` including Z Bank CIS documents.

## 8. Contrast with Nova Bank

| Topic | Nova | Z Bank |
|-------|------|--------|
| Framework | `nova` | `zbank` |
| Fund model | M0 / M1 / NSB | M0 + M1–M4 |
| Settlement IDs | `nova-*` | `zbank-*` only |

Do not mix account IDs across brands on the same host.

## Related PRs (merged)

| PR | Branch | Summary |
|----|--------|---------|
| #5 | `cursor/payment-gateway-b3f1` | Nova / Z Bank payment gateway |
| #8 | `cursor/zbank-m1-m4-setup-5e76` | M1–M4 liquidity layers |
| #9 | `cursor/zbank-dssboat-officer-5e76` | DSSBOaT officer PIN + signature |
| #12 | `cursor/zbank-pof-certificate-5e76` | Proof of Funds certificate PDFs |

Open: PR #6 (`cursor/payment-gateway-dashboard-b3f1`) — payment gateway admin dashboard.
