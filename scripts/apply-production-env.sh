#!/usr/bin/env bash
# Apply production env + go-live payment gateway on VPS.
# Usage:
#   export ONEX_STRIPE_SECRET_KEY=sk_live_...
#   export ONEX_STRIPE_PUBLISHABLE_KEY=pk_live_...
#   export ONEX_STRIPE_WEBHOOK_SECRET=whsec_...
#   bash scripts/apply-production-env.sh
set -euo pipefail

REPO="${ONEX_REPO:-$HOME/onex}"
ENV_SRC="${ENV_SRC:-$REPO/deploy/env.production.live.example}"
ENV_DEST="${ENV_DEST:-/etc/onex/onex.env}"
DOCKER_ENV="${DOCKER_ENV:-$REPO/.env}"

cd "$REPO"
git pull origin main 2>/dev/null || true

if [ ! -f "$ENV_SRC" ]; then
  echo "Missing $ENV_SRC" >&2
  exit 1
fi

# Generate API key if still placeholder
API_KEY="${ONEX_API_KEY:-}"
if [ -z "$API_KEY" ] || [ "$API_KEY" = "CHANGE_ME_LONG_RANDOM_SECRET" ]; then
  API_KEY="$(openssl rand -base64 32 | tr -d '/+=' | head -c 32)"
fi

sudo mkdir -p /etc/onex
sudo cp "$ENV_SRC" "$ENV_DEST"
sudo sed -i "s/CHANGE_ME_LONG_RANDOM_SECRET/$API_KEY/" "$ENV_DEST"

# Inject Stripe keys from environment if provided
for var in ONEX_STRIPE_SECRET_KEY ONEX_STRIPE_PUBLISHABLE_KEY ONEX_STRIPE_WEBHOOK_SECRET; do
  val="${!var:-}"
  if [ -n "$val" ] && [ "$val" != "CHANGE_ME" ]; then
    key="${var}"
    placeholder=""
    case "$var" in
      ONEX_STRIPE_SECRET_KEY) placeholder="sk_live_CHANGE_ME" ;;
      ONEX_STRIPE_PUBLISHABLE_KEY) placeholder="pk_live_CHANGE_ME" ;;
      ONEX_STRIPE_WEBHOOK_SECRET) placeholder="whsec_CHANGE_ME" ;;
    esac
    sudo sed -i "s|^${key}=.*|${key}=${val}|" "$ENV_DEST"
  fi
done

# Docker .env mirror
cp "$ENV_SRC" "$DOCKER_ENV"
sed -i "s/CHANGE_ME_LONG_RANDOM_SECRET/$API_KEY/" "$DOCKER_ENV"
for var in ONEX_STRIPE_SECRET_KEY ONEX_STRIPE_PUBLISHABLE_KEY ONEX_STRIPE_WEBHOOK_SECRET; do
  val="${!var:-}"
  if [ -n "$val" ]; then
    sed -i "s|^${var}=.*|${var}=${val}|" "$DOCKER_ENV" 2>/dev/null || echo "${var}=${val}" >> "$DOCKER_ENV"
  fi
done

echo "==> Applied $ENV_DEST"
echo "    ONEX_API_KEY=$API_KEY"
grep -E '^ONEX_PAYMENT|^ONEX_STRIPE|^ONEX_ONLINE|^ONEX_PRODUCTION_DOMAIN' "$ENV_DEST" 2>/dev/null | sudo sed 's/sk_live_.*/sk_live_***/' | sudo sed 's/whsec_.*/whsec_***/' || true

bash "$REPO/scripts/go-live-payment-gateway.sh"

if [ -n "${ONEX_STRIPE_SECRET_KEY:-}" ] && [ -z "${ONEX_STRIPE_WEBHOOK_SECRET:-}" ]; then
  echo ""
  echo "==> Stripe keys set but webhook secret missing — run:"
  echo "    ONEX_STRIPE_SECRET_KEY=\$ONEX_STRIPE_SECRET_KEY bash scripts/setup-stripe-webhook.sh"
fi
