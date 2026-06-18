#!/bin/bash
# Deploy OneX Production Platform (onexproduction.com or ONEX_PRODUCTION_DOMAIN).
set -euo pipefail

DOMAIN="${ONEX_PRODUCTION_DOMAIN:-onexproduction.com}"
EMAIL="${CERTBOT_EMAIL:-}"

echo "==> DNS check for $DOMAIN"
IPS=$(dig +short "$DOMAIN" 2>/dev/null | grep -E '^[0-9.]+$' || true)
if [ -z "$IPS" ]; then
  echo "WARN: No A record for $DOMAIN — continue anyway if using IP:port"
fi

if [ ! -f .env ]; then
  cp deploy/env.onexproduction.example .env
  echo "Created .env — set ONEX_API_KEY before going live"
fi

export ONEX_PRODUCTION_DOMAIN="$DOMAIN"

if [ -n "$EMAIL" ] && [ ! -f deploy/certs/fullchain.pem ]; then
  echo "==> TLS via certbot"
  sudo certbot certonly --standalone -d "$DOMAIN" -d "www.$DOMAIN" \
    --non-interactive --agree-tos -m "$EMAIL" || true
  sudo mkdir -p deploy/certs
  sudo cp "/etc/letsencrypt/live/$DOMAIN/fullchain.pem" deploy/certs/ 2>/dev/null || true
  sudo cp "/etc/letsencrypt/live/$DOMAIN/privkey.pem" deploy/certs/ 2>/dev/null || true
fi

echo "==> docker compose prod"
docker compose -f docker-compose.prod.yml --profile proxy up -d --build

sleep 8
echo "==> health"
curl -sf "http://127.0.0.1:9338/health" && echo " bridge OK" || echo " bridge FAIL"
curl -sf "http://127.0.0.1:9338/bridge/production/status" | head -c 400; echo
if [ -f deploy/certs/fullchain.pem ]; then
  curl -sf "https://$DOMAIN/bridge/production/status" | head -c 400; echo
  echo "Wallet: https://$DOMAIN/wallet/"
fi
