#!/usr/bin/env bash
# Configure Stripe webhook for OneX Payment Gateway.
# Usage (on your machine or VPS):
#   export ONEX_STRIPE_SECRET_KEY=sk_live_...
#   export ONEX_PRODUCTION_DOMAIN=onexproduction.com
#   bash scripts/setup-stripe-webhook.sh
set -euo pipefail

DOMAIN="${ONEX_PRODUCTION_DOMAIN:-blockchainsystem.com}"
SECRET_KEY="${ONEX_STRIPE_SECRET_KEY:-}"
ALT_DOMAIN="${ONEX_ALT_DOMAIN:-novatrustee.digital}"

if [ -z "$SECRET_KEY" ]; then
  echo "ERROR: set ONEX_STRIPE_SECRET_KEY (sk_live_... or sk_test_...)" >&2
  exit 1
fi

WEBHOOK_URL="https://${DOMAIN}/bridge/payments/webhook"
ALT_WEBHOOK_URL="https://${ALT_DOMAIN}/bridge/payments/webhook"

echo "==> Stripe webhook setup for OneX Payment Gateway"
echo "    Primary URL: $WEBHOOK_URL"
echo ""

# List existing endpoints
echo "==> Existing webhook endpoints"
curl -sf -u "${SECRET_KEY}:" \
  "https://api.stripe.com/v1/webhook_endpoints?limit=10" | \
  python3 -c "
import json,sys
d=json.load(sys.stdin)
for e in d.get('data',[]):
    print(' -', e.get('url'), '|', e.get('status'), '|', e.get('id'))
" 2>/dev/null || echo "    (could not list — check API key)"

echo ""
echo "==> Creating webhook endpoint (if not exists)"
RESP=$(curl -sf -u "${SECRET_KEY}:" \
  -d "url=${WEBHOOK_URL}" \
  -d "enabled_events[]=payment_intent.succeeded" \
  -d "enabled_events[]=payment_intent.payment_failed" \
  -d "description=OneX Nova Bank Payment Gateway" \
  "https://api.stripe.com/v1/webhook_endpoints" 2>&1) || RESP=""

if echo "$RESP" | grep -q '"secret"'; then
  WHSEC=$(echo "$RESP" | python3 -c "import json,sys; print(json.load(sys.stdin).get('secret',''))")
  WH_ID=$(echo "$RESP" | python3 -c "import json,sys; print(json.load(sys.stdin).get('id',''))")
  echo ""
  echo "=== SUCCESS ==="
  echo "Webhook ID:     $WH_ID"
  echo "Webhook URL:    $WEBHOOK_URL"
  echo ""
  echo "Add this to /etc/onex/onex.env (or .env):"
  echo "ONEX_STRIPE_WEBHOOK_SECRET=$WHSEC"
  echo ""
  echo "Then restart bridge:"
  echo "  sudo systemctl restart onex-bridge"
  echo "  # or: docker compose -f docker-compose.prod.yml restart onex-bridge"
elif echo "$RESP" | grep -qi "already exists\|url already"; then
  echo "    Webhook may already exist for this URL — check Stripe Dashboard → Developers → Webhooks"
  echo "    Copy the signing secret (whsec_...) into ONEX_STRIPE_WEBHOOK_SECRET"
else
  echo "    API response: $RESP"
  echo ""
  echo "=== MANUAL SETUP (Stripe Dashboard) ==="
  echo "1. Go to https://dashboard.stripe.com/webhooks"
  echo "2. Click '+ Add endpoint'"
  echo "3. Endpoint URL: $WEBHOOK_URL"
  echo "4. Events to send:"
  echo "     - payment_intent.succeeded"
  echo "     - payment_intent.payment_failed"
  echo "5. Click 'Add endpoint'"
  echo "6. Click the endpoint → 'Signing secret' → Reveal"
  echo "7. Copy whsec_... into ONEX_STRIPE_WEBHOOK_SECRET"
  echo ""
  echo "Optional second endpoint for $ALT_WEBHOOK_URL if using novatrustee.digital"
fi

echo ""
echo "==> Verify after deploy"
echo "curl -s https://${DOMAIN}/bridge/payments/status"
echo "curl -s -o /dev/null -w '%{http_code}' https://${DOMAIN}/payments/"
