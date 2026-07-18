# Z Bank Dashboards

Central index for operator and customer surfaces on the OneX bridge.

**Hub page (after deploy):** `https://zblockchainsystem.com/dashboards/`

## Operator dashboards

| Dashboard | URL | Description |
|-----------|-----|-------------|
| **Dashboards hub** | `/dashboards/` | Links to all surfaces below |
| **Payment gateway admin** | `/payments/dashboard/` | Sessions, volume, fees, settlements, recent payments |
| **Production platform** | `/bridge/production/status` | JSON — ledger, online bank, Hybrix, Fineract, cards, Bridge7 |
| **Bank status** | `/bridge/bank/status` | Z Bank accounts, cash codes, integrations |
| **Officer status** | `/bridge/bank/officer/status` | DSSBOaT signatory readiness |
| **Payments status** | `/bridge/payments/status` | Gateway framework, Stripe, settlement destinations |
| **Dashboard API** | `/bridge/payments/dashboard` | JSON aggregate for admin UI |
| **Ledger status** | `/bridge/ledger/status` | Fund classes M0–M4, middleware |

## Customer surfaces

| Surface | URL |
|---------|-----|
| Card payments portal | `/payments/` |
| Donate | `/payments/?page=donate` |
| Invoice | `/payments/?page=invoice` |
| Collect | `/payments/?page=collect` |
| Online bank (wallet tab) | `/wallet/#onlinebank` |
| Full DeFi wallet | `/wallet/` |

## Other dashboards (separate services)

| Product | Location | Notes |
|---------|----------|-------|
| BSC Launcher Mission Control | `bsc-launcher/web/` (port 9340) | Token deploy wizard + NASA-style console |
| Token Lab / Flash Coin | `:9340` when Token Lab running | See `docs/FLASH-COIN.md` |
| Stripe Dashboard | [dashboard.stripe.com](https://dashboard.stripe.com) | Live card acquiring, webhooks |

## Local dev

```bash
# Start bridge with Z Bank env (see docs/ZBANK-GO-LIVE.md)
./bin/onex-bridge -listen 127.0.0.1:9338

# Open dashboards
open http://127.0.0.1:9338/dashboards/
open http://127.0.0.1:9338/payments/dashboard/
```

Verify all endpoints:

```bash
bash scripts/verify-zbank-local.sh http://127.0.0.1:9338
```

## Production

Requires VPS deploy + DNS → `51.75.64.28`. See `docs/ZBANK-GO-LIVE.md` and `deploy/FIX-domain-dashboard.md`.
