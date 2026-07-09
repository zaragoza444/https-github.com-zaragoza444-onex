#!/usr/bin/env bash
# One-line production bootstrap for OneX Payment Gateway.
# Run on VPS web console as ubuntu:
#
#   curl -fsSL https://raw.githubusercontent.com/zaragoza444/https-github.com-zaragoza444-onex/main/scripts/production-bootstrap.sh | bash
#
# With Stripe keys (live cards):
#
#   curl -fsSL .../production-bootstrap.sh | \
#     ONEX_STRIPE_SECRET_KEY=sk_live_xxx \
#     ONEX_STRIPE_PUBLISHABLE_KEY=pk_live_xxx \
#     ONEX_STRIPE_WEBHOOK_SECRET=whsec_xxx \
#     bash
#
set -euo pipefail

REPO="${ONEX_REPO:-$HOME/onex}"
GITHUB="${GITHUB_REPO:-https://github.com/zaragoza444/https-github.com-zaragoza444-onex.git}"
BRANCH="main"
HOST_IP="${ONEX_PUBLIC_HOST:-51.75.64.28}"
DOMAIN="${ONEX_PRODUCTION_DOMAIN:-zblockchainsystem.com}"

echo "=============================================="
echo " OneX Payment Gateway — Production Bootstrap"
echo "=============================================="

# Stop conflicting listeners on 9338
if command -v fuser >/dev/null 2>&1; then
  sudo fuser -k 9338/tcp 2>/dev/null || true
  sleep 2
fi

# Clone or update
if [ ! -d "$REPO/.git" ]; then
  git clone "$GITHUB" "$REPO"
fi
cd "$REPO"
git fetch origin "$BRANCH"
git checkout "$BRANCH" 2>/dev/null || git checkout -b "$BRANCH"
git reset --hard "origin/$BRANCH"

export PATH="/usr/local/go/bin:$HOME/go/bin:$PATH"

echo "==> Build binaries"
mkdir -p "$REPO/bin" "$HOME/.onex/wallets"
go build -o "$REPO/bin/onexd" ./cmd/onexd
go build -o "$REPO/bin/onex-bridge" ./cmd/onex-bridge

# Apply production env
export ONEX_REPO="$REPO"
bash "$REPO/scripts/apply-production-env.sh" 2>/dev/null || {
  # apply-production-env may fail if sudo needs tty — fallback inline
  ENV_FILE="/etc/onex/onex.env"
  sudo mkdir -p /etc/onex
  if [ ! -f "$ENV_FILE" ]; then
    sudo cp "$REPO/deploy/env.zblockchainsystem.com.example" "$ENV_FILE"
    KEY="$(openssl rand -base64 32 | tr -d '/+=' | head -c 32)"
    sudo sed -i "s/CHANGE_ME_LONG_RANDOM_SECRET/$KEY/" "$ENV_FILE"
  fi
  for pair in \
    "ONEX_ONLINE_BANK=1" \
    "ONEX_PAYMENT_GATEWAY=1" \
    "ONEX_PAYMENT_GATEWAY_FILE=${REPO}/configs/payment-gateway.production.json" \
    "ONEX_PAYMENT_GATEWAY_FRAMEWORK=nova" \
    "ONEX_PAYMENT_GATEWAY_PROVIDER=stripe" \
    "ONEX_BANK_LEDGER_FILE=${REPO}/configs/bank-ledger.nova.example.json" \
    "ONEX_LEDGER_MODE=production" \
    "ONEX_PROJECT_ROOT=${REPO}" \
    "ONEX_PUBLIC_HOST=${HOST_IP}"; do
    k="${pair%%=*}"; v="${pair#*=}"
    if sudo grep -q "^${k}=" "$ENV_FILE" 2>/dev/null; then
      sudo sed -i "s|^${k}=.*|${k}=${v}|" "$ENV_FILE"
    else
      echo "${k}=${v}" | sudo tee -a "$ENV_FILE" >/dev/null
    fi
  done
  for var in ONEX_STRIPE_SECRET_KEY ONEX_STRIPE_PUBLISHABLE_KEY ONEX_STRIPE_WEBHOOK_SECRET; do
    val="${!var:-}"
    [ -n "$val" ] && sudo sed -i "s|^${var}=.*|${var}=${val}|" "$ENV_FILE" 2>/dev/null || \
      echo "${var}=${val}" | sudo tee -a "$ENV_FILE" >/dev/null
  done
}

# systemd units (always refresh to latest binary path)
sudo tee /etc/systemd/system/onexd.service >/dev/null <<UNIT
[Unit]
Description=OneX blockchain node
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=ubuntu
WorkingDirectory=${REPO}
EnvironmentFile=/etc/onex/onex.env
ExecStart=${REPO}/bin/onexd -datadir ${REPO}/data -genesis ${REPO}/configs/genesis.json -seeds ${REPO}/configs/seeds-mainnet.json -api :8545 -listen :30303
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
UNIT

sudo tee /etc/systemd/system/onex-bridge.service >/dev/null <<UNIT
[Unit]
Description=OneX Wallet Bridge + Payment Gateway
After=network-online.target onexd.service
Wants=onexd.service

[Service]
Type=simple
User=ubuntu
WorkingDirectory=${REPO}
EnvironmentFile=/etc/onex/onex.env
ExecStart=${REPO}/bin/onex-bridge -node http://127.0.0.1:8545 -listen 0.0.0.0:9338 -config ${HOME}/.onex/bridge.json -wallet ${HOME}/.onex/wallets/default.json
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
UNIT

sudo systemctl daemon-reload
sudo systemctl enable onexd onex-bridge
sudo systemctl restart onexd
sleep 4
sudo systemctl restart onex-bridge
sleep 6

echo ""
echo "==> Health checks"
curl -sf "http://127.0.0.1:8545/health" && echo "  onexd OK" || echo "  onexd FAIL"
curl -sf "http://127.0.0.1:9338/health" && echo "  bridge OK" || echo "  bridge FAIL"
curl -sf "http://127.0.0.1:9338/bridge/payments/status" | head -c 300; echo
HTTP=$(curl -sf -o /dev/null -w "%{http_code}" "http://127.0.0.1:9338/payments/" || echo "000")
echo "  payments portal HTTP $HTTP"

API_KEY=$(sudo grep '^ONEX_API_KEY=' /etc/onex/onex.env 2>/dev/null | cut -d= -f2- || echo "see /etc/onex/onex.env")

echo ""
echo "=============================================="
echo " PAYMENT GATEWAY BOOTSTRAP COMPLETE"
echo "=============================================="
echo "Portal:   http://${HOST_IP}:9338/payments/"
echo "Donate:   http://${HOST_IP}:9338/payments/?page=donate"
echo "Status:   http://${HOST_IP}:9338/bridge/payments/status"
echo "API key:  ${API_KEY}"
echo ""
if ! sudo grep -q 'sk_live_' /etc/onex/onex.env 2>/dev/null || sudo grep -q 'sk_live_CHANGE_ME' /etc/onex/onex.env 2>/dev/null; then
  echo "NEXT: Add Stripe live keys to /etc/onex/onex.env then:"
  echo "  sudo systemctl restart onex-bridge"
  echo "  bash scripts/setup-stripe-webhook.sh"
fi
echo "HTTPS (after DNS -> ${HOST_IP}):"
echo "  https://${DOMAIN}/payments/"
echo "  https://${DOMAIN}/payments/?page=donate"
echo "GitHub: zaragoza444 | Gitea: git.anakatech.llc/Zaragoza/onex"
echo "=============================================="
