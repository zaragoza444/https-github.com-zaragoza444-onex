#!/usr/bin/env bash
# Go-live payment gateway on existing OneX VPS (run on server as ubuntu).
#   bash scripts/go-live-payment-gateway.sh
set -euo pipefail

REPO="${ONEX_REPO:-$HOME/onex}"
GITHUB="${GITHUB_REPO:-https://github.com/zaragoza444/https-github.com-zaragoza444-onex.git}"
BRANCH="${ONEX_DEPLOY_BRANCH:-main}"

echo "==> OneX Payment Gateway — production go-live"
cd "$REPO" 2>/dev/null || { git clone "$GITHUB" "$REPO" && cd "$REPO"; }
git fetch origin "$BRANCH"
git checkout "$BRANCH"
git pull origin "$BRANCH" || git reset --hard "origin/$BRANCH"

export PATH="/usr/local/go/bin:$HOME/go/bin:$PATH"
echo "==> Build onex-bridge"
go build -o "$REPO/bin/onex-bridge" ./cmd/onex-bridge

ENV_FILE="/etc/onex/onex.env"
if [ ! -f "$ENV_FILE" ]; then
  sudo mkdir -p /etc/onex
  sudo cp deploy/env.onexproduction.example "$ENV_FILE"
  KEY="$(openssl rand -base64 32 | tr -d '/+=' | head -c 32)"
  sudo sed -i "s/CHANGE_ME_LONG_RANDOM_SECRET/$KEY/" "$ENV_FILE"
fi

upsert() {
  local key="$1" val="$2"
  if sudo grep -q "^${key}=" "$ENV_FILE" 2>/dev/null; then
    sudo sed -i "s|^${key}=.*|${key}=${val}|" "$ENV_FILE"
  else
    echo "${key}=${val}" | sudo tee -a "$ENV_FILE" >/dev/null
  fi
}

upsert ONEX_ONLINE_BANK 1
upsert ONEX_PAYMENT_GATEWAY 1
upsert ONEX_PAYMENT_GATEWAY_FILE "${REPO}/configs/payment-gateway.production.json"
upsert ONEX_PAYMENT_GATEWAY_FRAMEWORK nova
upsert ONEX_PAYMENT_GATEWAY_PROVIDER stripe
upsert ONEX_BANK_LEDGER_FILE "${REPO}/configs/bank-ledger.nova.example.json"
upsert ONEX_LEDGER_MODE production
upsert ONEX_PROJECT_ROOT "$REPO"

if command -v docker >/dev/null 2>&1 && [ -f docker-compose.prod.yml ]; then
  echo "==> Docker production stack"
  if [ ! -f .env ]; then
    cp deploy/env.onexproduction.example .env
  fi
  grep -q ONEX_PAYMENT_GATEWAY .env || cat >> .env <<EOF
ONEX_ONLINE_BANK=1
ONEX_PAYMENT_GATEWAY=1
ONEX_PAYMENT_GATEWAY_FILE=configs/payment-gateway.production.json
ONEX_PAYMENT_GATEWAY_FRAMEWORK=nova
ONEX_PAYMENT_GATEWAY_PROVIDER=stripe
ONEX_BANK_LEDGER_FILE=configs/bank-ledger.nova.example.json
EOF
  docker compose -f docker-compose.prod.yml --profile proxy up -d --build onex-bridge
  sleep 8
else
  echo "==> systemd onex-bridge restart"
  sudo systemctl daemon-reload
  sudo systemctl restart onex-bridge || sudo systemctl start onex-bridge
  sleep 5
fi

HOST="${ONEX_PUBLIC_HOST:-$(curl -sf --max-time 5 https://api.ipify.org || echo 127.0.0.1)}"
echo ""
echo "==> Verify"
curl -sf "http://127.0.0.1:9338/bridge/payments/status" && echo
curl -sf -o /dev/null -w "payments portal HTTP %{http_code}\n" "http://127.0.0.1:9338/payments/"

echo ""
echo "=== PAYMENT GATEWAY LIVE ==="
echo "Portal:  http://${HOST}:9338/payments/"
echo "Donate:  http://${HOST}:9338/payments/?page=donate"
echo "Invoice: http://${HOST}:9338/payments/?page=invoice"
echo "Collect: http://${HOST}:9338/payments/?page=collect"
echo "Status:  http://${HOST}:9338/bridge/payments/status"
echo ""
echo "For live Visa/MC/Amex: add ONEX_STRIPE_* keys to $ENV_FILE and restart onex-bridge"
echo "Webhook URL: https://YOUR_DOMAIN/bridge/payments/webhook"
