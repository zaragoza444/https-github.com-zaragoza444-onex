#!/usr/bin/env bash
# Emergency fix for broken onex-bridge. Tries :9338 then :9339.
#   curl -fsSL .../fix-bridge-9338.sh | ONEX_PRODUCTION_DOMAIN=zblockchainsystem.com bash
set -euo pipefail

REPO="${ONEX_REPO:-$HOME/onex}"
GITHUB="${GITHUB_REPO:-https://github.com/zaragoza444/https-github.com-zaragoza444-onex.git}"
DOMAIN="${ONEX_PRODUCTION_DOMAIN:-zblockchainsystem.com}"
PORT="${ONEX_BRIDGE_PORT:-9338}"

echo "==> Fix onex-bridge :${PORT} ($DOMAIN)"

sudo systemctl stop onex-bridge 2>/dev/null || true
if command -v docker >/dev/null 2>&1 && [ -f "${REPO}/docker-compose.prod.yml" ]; then
  cd "$REPO" 2>/dev/null && docker compose -f docker-compose.prod.yml stop onex-bridge 2>/dev/null || true
fi
sudo fuser -k 9338/tcp 2>/dev/null || true
sudo fuser -k 9339/tcp 2>/dev/null || true
sleep 2

echo "==> Port check (before start)"
ss -tlnp 2>/dev/null | grep -E ':933[89]' || echo "  (9338/9339 free)"

if [ ! -d "$REPO/.git" ]; then git clone "$GITHUB" "$REPO"; fi
cd "$REPO"
git fetch origin main && git reset --hard origin/main

export PATH="/usr/local/go/bin:$HOME/go/bin:$PATH"
mkdir -p "$HOME/.onex/wallets" "$REPO/bin" "$REPO/data"

echo "==> Build onex-bridge"
go build -o "$REPO/bin/onex-bridge" ./cmd/onex-bridge

cat > "$HOME/.onex/bridge.json" <<JSON
{
  "nodeUrl": "http://127.0.0.1:8545",
  "listen": "0.0.0.0:${PORT}",
  "walletPath": "$HOME/.onex/wallets/default.json",
  "projectRoot": "$REPO"
}
JSON

ENV_FILE="/etc/onex/onex.env"
sudo mkdir -p /etc/onex
sudo cp "$REPO/deploy/env.zblockchainsystem.com.example" "$ENV_FILE" 2>/dev/null || \
  sudo cp "$REPO/deploy/env.production.live.example" "$ENV_FILE"
KEY="$(openssl rand -base64 32 | tr -d '/+=' | head -c 32)"
sudo sed -i "s/CHANGE_ME_LONG_RANDOM_SECRET/$KEY/" "$ENV_FILE"
sudo sed -i "s|^ONEX_PRODUCTION_DOMAIN=.*|ONEX_PRODUCTION_DOMAIN=${DOMAIN}|" "$ENV_FILE"
sudo sed -i "s|^ONEX_PROJECT_ROOT=.*|ONEX_PROJECT_ROOT=${REPO}|" "$ENV_FILE"
sudo sed -i "s|^ONEX_BRIDGE_LISTEN=.*|ONEX_BRIDGE_LISTEN=0.0.0.0:${PORT}|" "$ENV_FILE" 2>/dev/null || \
  echo "ONEX_BRIDGE_LISTEN=0.0.0.0:${PORT}" | sudo tee -a "$ENV_FILE" >/dev/null

for var in ONEX_STRIPE_SECRET_KEY ONEX_STRIPE_PUBLISHABLE_KEY ONEX_STRIPE_WEBHOOK_SECRET; do
  val="${!var:-}"
  [ -n "$val" ] && sudo sed -i "s|^${var}=.*|${var}=${val}|" "$ENV_FILE" 2>/dev/null || \
    echo "${var}=${val}" | sudo tee -a "$ENV_FILE" >/dev/null
done

start_bridge() {
  local p="$1"
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
ExecStart=${REPO}/bin/onex-bridge -node http://127.0.0.1:8545 -listen 0.0.0.0:${p} -config ${HOME}/.onex/bridge.json -wallet ${HOME}/.onex/wallets/default.json
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
}

start_bridge "$PORT"

if ! curl -sf "http://127.0.0.1:${PORT}/health" >/dev/null 2>&1; then
  echo "WARN: :${PORT} failed — trying :9339"
  PORT=9339
  sed -i "s/0.0.0.0:9338/0.0.0.0:9339/" "$HOME/.onex/bridge.json"
  start_bridge "$PORT"
fi

echo "$PORT" > "$REPO/.bridge-port"
echo "ONEX_BRIDGE_PORT=${PORT}" | sudo tee "$REPO/.bridge-port.env" >/dev/null

echo "==> systemd status (port ${PORT})"
sudo systemctl is-active onex-bridge || true
sudo journalctl -u onex-bridge -n 20 --no-pager || true

if ! curl -sf "http://127.0.0.1:${PORT}/health"; then
  echo "FAIL — bridge not responding on ${PORT}"
  echo "Run manually for error output:"
  echo "  ${REPO}/bin/onex-bridge -node http://127.0.0.1:8545 -listen 127.0.0.1:9339 -config ~/.onex/bridge.json -wallet ~/.onex/wallets/default.json"
  exit 1
fi

echo " OK on :${PORT}"
curl -sf "http://127.0.0.1:${PORT}/bridge/payments/status" | head -c 400; echo

# Update nginx to use active bridge port
if command -v nginx >/dev/null 2>&1; then
  sudo tee /etc/nginx/sites-available/zblockchain-onex >/dev/null <<NGX
server {
    listen 80;
    server_name ${DOMAIN} www.${DOMAIN} 51.75.64.28;

    location /payments/ {
        proxy_pass http://127.0.0.1:${PORT}/payments/;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
    }
    location /bridge/ {
        proxy_pass http://127.0.0.1:${PORT}/bridge/;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
    }
    location /wallet/ {
        proxy_pass http://127.0.0.1:${PORT}/wallet/;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
    }
    location /health {
        proxy_pass http://127.0.0.1:${PORT}/health;
    }
    location / {
        root /var/www/nova-bank;
        try_files \$uri \$uri/ /index.html;
    }
}
NGX
  sudo ln -sf /etc/nginx/sites-available/zblockchain-onex /etc/nginx/sites-enabled/zblockchain-onex
  sudo nginx -t && sudo systemctl reload nginx
  echo "nginx reloaded -> 127.0.0.1:${PORT}"
fi

echo ""
echo "=== FIXED on port ${PORT} ==="
echo "http://${DOMAIN}/payments/?page=donate"
echo "http://${DOMAIN}/bridge/payments/status"
