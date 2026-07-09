#!/usr/bin/env bash
# Emergency fix for broken onex-bridge on port 9338.
# Run on VPS as ubuntu:
#   curl -fsSL https://raw.githubusercontent.com/zaragoza444/https-github.com-zaragoza444-onex/main/scripts/fix-bridge-9338.sh | \
#     ONEX_PRODUCTION_DOMAIN=zblockchainsystem.com bash
set -euo pipefail

REPO="${ONEX_REPO:-$HOME/onex}"
GITHUB="${GITHUB_REPO:-https://github.com/zaragoza444/https-github.com-zaragoza444-onex.git}"
DOMAIN="${ONEX_PRODUCTION_DOMAIN:-zblockchainsystem.com}"

echo "==> Fix onex-bridge :9338 ($DOMAIN)"

# Stop everything that may hold 9338
sudo systemctl stop onex-bridge 2>/dev/null || true
if command -v docker >/dev/null 2>&1 && [ -f "$REPO/docker-compose.prod.yml" ]; then
  cd "$REPO" 2>/dev/null && docker compose -f docker-compose.prod.yml stop onex-bridge 2>/dev/null || true
fi
sudo fuser -k 9338/tcp 2>/dev/null || true
sleep 2

# Update source
if [ ! -d "$REPO/.git" ]; then
  git clone "$GITHUB" "$REPO"
fi
cd "$REPO"
git fetch origin main && git reset --hard origin/main

export PATH="/usr/local/go/bin:$HOME/go/bin:$PATH"
mkdir -p "$HOME/.onex/wallets" "$REPO/bin" "$REPO/data"

echo "==> Build onex-bridge"
go build -o "$REPO/bin/onex-bridge" ./cmd/onex-bridge

# Minimal bridge config
if [ ! -f "$HOME/.onex/bridge.json" ]; then
  cat > "$HOME/.onex/bridge.json" <<JSON
{
  "nodeUrl": "http://127.0.0.1:8545",
  "listen": "0.0.0.0:9338",
  "walletPath": "$HOME/.onex/wallets/default.json",
  "projectRoot": "$REPO"
}
JSON
fi

# Production env
ENV_FILE="/etc/onex/onex.env"
sudo mkdir -p /etc/onex
if [ -f "$REPO/deploy/env.zblockchainsystem.com.example" ]; then
  sudo cp "$REPO/deploy/env.zblockchainsystem.com.example" "$ENV_FILE"
else
  sudo cp "$REPO/deploy/env.production.live.example" "$ENV_FILE"
fi
KEY="$(openssl rand -base64 32 | tr -d '/+=' | head -c 32)"
sudo sed -i "s/CHANGE_ME_LONG_RANDOM_SECRET/$KEY/" "$ENV_FILE"
sudo sed -i "s|^ONEX_PRODUCTION_DOMAIN=.*|ONEX_PRODUCTION_DOMAIN=${DOMAIN}|" "$ENV_FILE"
sudo sed -i "s|^ONEX_PROJECT_ROOT=.*|ONEX_PROJECT_ROOT=${REPO}|" "$ENV_FILE"

for var in ONEX_STRIPE_SECRET_KEY ONEX_STRIPE_PUBLISHABLE_KEY ONEX_STRIPE_WEBHOOK_SECRET; do
  val="${!var:-}"
  [ -n "$val" ] && sudo sed -i "s|^${var}=.*|${var}=${val}|" "$ENV_FILE" 2>/dev/null || \
    echo "${var}=${val}" | sudo tee -a "$ENV_FILE" >/dev/null
done

sudo tee /etc/systemd/system/onex-bridge.service >/dev/null <<UNIT
[Unit]
Description=OneX Bridge + Payment Gateway
After=network-online.target onexd.service
Wants=onexd.service

[Service]
Type=simple
User=ubuntu
WorkingDirectory=${REPO}
EnvironmentFile=${ENV_FILE}
ExecStart=${REPO}/bin/onex-bridge -node http://127.0.0.1:8545 -listen 0.0.0.0:9338 -config ${HOME}/.onex/bridge.json -wallet ${HOME}/.onex/wallets/default.json
Restart=always
RestartSec=2
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
UNIT

sudo systemctl daemon-reload
sudo systemctl enable onex-bridge
sudo systemctl restart onex-bridge
sleep 8

echo "==> systemd status"
sudo systemctl is-active onex-bridge || true
sudo journalctl -u onex-bridge -n 15 --no-pager || true

echo ""
echo "==> Local health"
if curl -sf "http://127.0.0.1:9338/health"; then
  echo " OK"
else
  echo " FAIL — trying foreground debug (5s)..."
  timeout 5 "$REPO/bin/onex-bridge" -node http://127.0.0.1:8545 -listen 127.0.0.1:9339 -config "$HOME/.onex/bridge.json" -wallet "$HOME/.onex/wallets/default.json" 2>&1 || true
  exit 1
fi

curl -sf "http://127.0.0.1:9338/bridge/payments/status" | head -c 400; echo
echo ""
echo "=== FIXED ==="
echo "http://51.75.64.28:9338/payments/?page=donate"
echo "https://${DOMAIN}/payments/ (after DNS -> 51.75.64.28)"
