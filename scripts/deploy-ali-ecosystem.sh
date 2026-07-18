#!/bin/bash
# Run ON the ALI/ALLTRA ecosystem VPS (ubuntu@zblockchainsystem.com) after git pull.
set -euo pipefail

REPO="${ALI_DEPLOY_ROOT:-/home/ubuntu/onex}"
GITHUB="${GITHUB_REPO:-https://github.com/zaragoza444/onex.git}"
DOMAIN="${ALI_PUBLIC_HOST:-zblockchainsystem.com}"
export PATH="/usr/local/go/bin:$HOME/go/bin:$PATH"

echo "==> firewall"
if command -v ufw >/dev/null 2>&1; then
  sudo ufw allow 22/tcp || true
  sudo ufw allow 80/tcp || true
  sudo ufw allow 443/tcp || true
  sudo ufw allow 9338/tcp || true
  sudo ufw allow 8545/tcp || true
  sudo ufw allow 9340/tcp || true
  sudo ufw allow 30303/tcp || true
  if sudo ufw status | grep -q inactive; then
    echo "y" | sudo ufw enable || true
  fi
  sudo ufw reload || true
fi

if [ ! -d "$REPO/.git" ]; then
  git clone "$GITHUB" "$REPO"
fi
cd "$REPO"
git fetch origin main
git reset --hard origin/main

mkdir -p "$HOME/.onex/wallets" "$HOME/.onex/portfolios" "$HOME/.onex/ledger-import" bin data

echo "==> build"
go build -o "$REPO/bin/onexd" ./cmd/onexd
go build -o "$REPO/bin/onex-bridge" ./cmd/onex-bridge
go build -o "$REPO/bin/bsc-launcher" ./bsc-launcher/server

API_KEY="${ONEX_API_KEY:-$(openssl rand -base64 32 | tr -d '/+=' | head -c 32)}"
sudo mkdir -p /etc/onex
sudo tee /etc/onex/onex.env >/dev/null <<EOF
ONEX_API_KEY=${API_KEY}
ONEX_CORS_ORIGINS=https://${DOMAIN},https://www.${DOMAIN},https://zaragoza444.github.io,https://zaragoza444.github.io/onex,https://git.anakatech.llc,https://explorer.d-bis.org
ONEX_LEDGER_MODE=production
ONEX_BANK_LEDGER_FILE=${REPO}/configs/bank-ledger.nova.example.json
ONEX_ONLINE_BANK=1
ONEX_PAYMENT_GATEWAY=1
ONEX_PAYMENT_GATEWAY_FILE=${REPO}/configs/payment-gateway.production.json
ONEX_PAYMENT_GATEWAY_FRAMEWORK=nova
ONEX_PAYMENT_GATEWAY_PROVIDER=stripe
ONEX_PROJECT_ROOT=${REPO}
ONEX_HOME_DIR=${HOME}/.onex
ONEX_NODE_URL=http://127.0.0.1:8545
ONEX_BRIDGE_LISTEN=0.0.0.0:9338
ONEX_PRODUCTION_DOMAIN=${DOMAIN}
ONEX_DEFAULT_BRIDGE_CHAIN=dbis-138
DBIS138_RPC_URL=https://rpc-core.d-bis.org
DBIS138_EXPLORER=https://explorer.d-bis.org
DBIS138_CHAIN_ID=138
EOF

sudo tee /etc/systemd/system/onexd.service >/dev/null <<UNIT
[Unit]
Description=OneX blockchain node (ALI ecosystem)
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=ubuntu
WorkingDirectory=${REPO}
ExecStart=${REPO}/bin/onexd -datadir ${REPO}/data -genesis ${REPO}/configs/genesis.json -seeds ${REPO}/configs/seeds-mainnet.json -api :8545 -listen :30303
EnvironmentFile=/etc/onex/onex.env
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
UNIT

sudo tee /etc/systemd/system/onex-bridge.service >/dev/null <<UNIT
[Unit]
Description=OneX Wallet bridge + Real Ledger (ALI ecosystem)
After=network-online.target onexd.service
Wants=onexd.service

[Service]
Type=simple
User=ubuntu
WorkingDirectory=${REPO}
ExecStart=${REPO}/bin/onex-bridge -node http://127.0.0.1:8545 -listen 0.0.0.0:9338 -config ${HOME}/.onex/bridge.json -wallet ${HOME}/.onex/wallets/default.json
EnvironmentFile=/etc/onex/onex.env
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
UNIT

if [ -f "$REPO/bsc-launcher/.env.production.example" ] && [ ! -f "$REPO/bsc-launcher/.env" ]; then
  cp "$REPO/bsc-launcher/.env.production.example" "$REPO/bsc-launcher/.env"
fi

sudo tee /etc/systemd/system/onex-token-lab.service >/dev/null <<UNIT
[Unit]
Description=OneX Token Lab (BSC Launcher)
After=network-online.target

[Service]
Type=simple
User=ubuntu
WorkingDirectory=${REPO}
EnvironmentFile=${REPO}/bsc-launcher/.env
Environment=BSC_LAUNCHER_ROOT=${REPO}/bsc-launcher
Environment=BSC_LAUNCHER_DATA_DIR=${REPO}/data/token-lab
ExecStart=${REPO}/bin/bsc-launcher
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
UNIT

sudo systemctl daemon-reload
sudo systemctl enable onexd onex-bridge onex-token-lab
sudo systemctl restart onexd
sleep 3
sudo systemctl restart onex-bridge onex-token-lab
sleep 3

echo "==> health"
curl -sf "http://127.0.0.1:8545/health" && echo " onexd OK" || echo " onexd FAIL"
curl -sf "http://127.0.0.1:9338/health" && echo " bridge OK" || echo " bridge FAIL"
curl -sf "http://127.0.0.1:9338/bridge/health/green" | head -c 400; echo
curl -sf "http://127.0.0.1:9338/bridge/ledger/status" | head -c 240; echo
curl -sf "http://127.0.0.1:9340/health" && echo " token-lab OK" || echo " token-lab FAIL"
systemctl is-active onexd onex-bridge onex-token-lab

echo ""
echo "=== PUBLIC (use domain URLs) ==="
echo "PUBLIC_WALLET=https://${DOMAIN}/wallet/"
echo "PUBLIC_LEDGER=https://${DOMAIN}/wallet/#ledger"
echo "PUBLIC_GREEN=https://${DOMAIN}/bridge/health/green"
echo "GITHUB_PAGES=https://zaragoza444.github.io/onex/wallet/?bridge=https://${DOMAIN}"
echo "ONEX_API_KEY=${API_KEY}"
