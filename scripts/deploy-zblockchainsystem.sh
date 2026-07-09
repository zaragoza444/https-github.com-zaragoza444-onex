#!/usr/bin/env bash
# Deploy OneX + Payment Gateway on zblockchainsystem.com
set -euo pipefail

DOMAIN="${ONEX_PRODUCTION_DOMAIN:-zblockchainsystem.com}"
EMAIL="${CERTBOT_EMAIL:-hello@zblockchainsystem.com}"
REPO="${ONEX_REPO:-$(cd "$(dirname "$0")/.." && pwd)}"

echo "==> Deploy OneX for $DOMAIN"
cd "$REPO"
git pull origin main 2>/dev/null || true

if [ ! -f .env ]; then
  cp deploy/env.zblockchainsystem.com.example .env
  KEY="$(openssl rand -base64 32 | tr -d '/+=' | head -c 32)"
  sed -i "s/CHANGE_ME_LONG_RANDOM_SECRET/$KEY/" .env
fi

export ONEX_PRODUCTION_DOMAIN="$DOMAIN"

if [ ! -f deploy/certs/fullchain.pem ]; then
  echo "==> TLS via certbot for $DOMAIN"
  sudo certbot certonly --standalone -d "$DOMAIN" -d "www.$DOMAIN" \
    --non-interactive --agree-tos -m "$EMAIL" || true
  sudo mkdir -p deploy/certs
  sudo cp "/etc/letsencrypt/live/$DOMAIN/fullchain.pem" deploy/certs/ 2>/dev/null || true
  sudo cp "/etc/letsencrypt/live/$DOMAIN/privkey.pem" deploy/certs/ 2>/dev/null || true
fi

docker compose -f docker-compose.prod.yml --profile proxy up -d --build

sleep 10
curl -sf "http://127.0.0.1:9338/bridge/payments/status" | head -c 400; echo
echo "Payments: https://$DOMAIN/payments/?page=donate"
