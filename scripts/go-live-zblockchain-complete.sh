#!/usr/bin/env bash
# Complete go-live: fix bridge :9338 + nginx proxy + TLS for zblockchainsystem.com
#
#   curl -fsSL https://raw.githubusercontent.com/zaragoza444/https-github.com-zaragoza444-onex/main/scripts/go-live-zblockchain-complete.sh | bash
#
set -euo pipefail

REPO="${ONEX_REPO:-$HOME/onex}"
GITHUB="${GITHUB_REPO:-https://github.com/zaragoza444/https-github.com-zaragoza444-onex.git}"
DOMAIN="${ONEX_PRODUCTION_DOMAIN:-zblockchainsystem.com}"
EMAIL="${CERTBOT_EMAIL:-hello@${DOMAIN}}"

echo "=============================================="
echo " zblockchainsystem.com — complete go-live"
echo "=============================================="

# --- 1. Fix bridge ---
if [ ! -d "$REPO/.git" ]; then git clone "$GITHUB" "$REPO"; fi
cd "$REPO" && git fetch origin main && git reset --hard origin/main

bash "$REPO/scripts/fix-bridge-9338.sh" || {
  echo "WARN: fix-bridge-9338 had issues — continuing with nginx setup"
}

# --- 2. Nginx proxy (payments + bridge via port 80) ---
if ! command -v nginx >/dev/null 2>&1; then
  sudo apt-get update -qq && sudo apt-get install -y nginx
fi

sudo tee /etc/nginx/sites-available/zblockchain-onex >/dev/null <<NGX
server {
    listen 80;
    server_name ${DOMAIN} www.${DOMAIN} zblockchainsystem.com;

    location /payments/ {
        proxy_pass http://127.0.0.1:9338/payments/;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }

    location /bridge/ {
        proxy_pass http://127.0.0.1:9338/bridge/;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }

    location /wallet/ {
        proxy_pass http://127.0.0.1:9338/wallet/;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
    }

    location /health {
        proxy_pass http://127.0.0.1:9338/health;
        default_type application/json;
    }

    location / {
        root /var/www/nova-bank;
        try_files \$uri \$uri/ /index.html;
    }
}
NGX

sudo ln -sf /etc/nginx/sites-available/zblockchain-onex /etc/nginx/sites-enabled/zblockchain-onex
sudo rm -f /etc/nginx/sites-enabled/default 2>/dev/null || true
sudo nginx -t && sudo systemctl reload nginx

# --- 3. TLS (optional, if certbot available and DNS resolves) ---
if command -v certbot >/dev/null 2>&1; then
  RESOLVED=$(dig +short "$DOMAIN" | head -1)
  VPS_IP=$(curl -sf --max-time 5 https://api.ipify.org || echo "")
  if [ "$RESOLVED" = "$VPS_IP" ] && [ -n "$VPS_IP" ]; then
    echo "==> TLS for $DOMAIN"
    sudo certbot --nginx -d "$DOMAIN" -d "www.$DOMAIN" \
      --non-interactive --agree-tos -m "$EMAIL" --redirect || \
      echo "WARN: certbot failed — HTTP still works"
  fi
fi

# --- 4. Verify ---
echo ""
echo "==> Verification"
LOCAL=$(curl -sf http://127.0.0.1:9338/health 2>/dev/null || echo "FAIL")
echo "  bridge local:  $LOCAL"
PAY80=$(curl -sf http://127.0.0.1/bridge/payments/status 2>/dev/null | head -c 80 || echo "FAIL")
echo "  payments :80:  $PAY80"
PAYDOM=$(curl -sf "http://${DOMAIN}/bridge/payments/status" 2>/dev/null | head -c 80 || echo "FAIL")
echo "  payments dom:  $PAYDOM"

echo ""
echo "=============================================="
if echo "$PAYDOM" | grep -q '"enabled"'; then
  echo " PAYMENT GATEWAY LIVE"
  echo " https://${DOMAIN}/payments/?page=donate"
  echo " https://${DOMAIN}/bridge/payments/webhook  (Stripe)"
else
  echo " PARTIAL — bridge may still need attention"
  echo " Run: sudo journalctl -u onex-bridge -n 30 --no-pager"
  echo " Then re-run this script"
fi
echo "=============================================="
