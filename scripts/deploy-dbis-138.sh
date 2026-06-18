#!/bin/bash
# Deploy OneX bridge + ledger middleware on the IDBIS/DBIS Chain 138 server.
# Run ON the VPS (ubuntu user) after git clone/pull.
set -euo pipefail

REPO="${DBIS138_DEPLOY_ROOT:-/home/ubuntu/onex}"
GITHUB="${GITHUB_REPO:-https://github.com/zaragoza444/onex.git}"
HOST_IP="${DBIS138_PUBLIC_HOST:-$(curl -sf ifconfig.me 2>/dev/null || hostname -I | awk '{print $1}')}"
DBIS_RPC="${DBIS138_RPC_URL:-https://rpc-core.d-bis.org}"

export PATH="/usr/local/go/bin:$HOME/go/bin:$PATH"

echo "==> IDBIS/DBIS Chain 138 deploy"
echo "    host: $HOST_IP"
echo "    rpc:  $DBIS_RPC"

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

API_KEY="${ONEX_API_KEY:-$(openssl rand -base64 32 | tr -d '/+=' | head -c 32)}"
sudo mkdir -p /etc/onex
sudo tee /etc/onex/onex.env >/dev/null <<EOF
ONEX_API_KEY=${API_KEY}
ONEX_CORS_ORIGINS=http://${HOST_IP}:9338,http://127.0.0.1:9338,https://explorer.d-bis.org,https://git.anakatech.llc,https://zaragoza444.github.io
ONEX_LEDGER_MODE=production
ONEX_BANK_LEDGER_FILE=${REPO}/configs/bank-ledger.example.json
ONEX_PROJECT_ROOT=${REPO}
ONEX_HOME_DIR=${HOME}/.onex
ONEX_NODE_URL=http://127.0.0.1:8545
ONEX_BRIDGE_LISTEN=0.0.0.0:9338
ONEX_DEFAULT_BRIDGE_CHAIN=dbis-138
DBIS138_RPC_URL=${DBIS_RPC}
DBIS138_EXPLORER=https://explorer.d-bis.org
DBIS138_CHAIN_ID=138
EOF

sudo tee /etc/systemd/system/onexd.service >/dev/null <<UNIT
[Unit]
Description=OneX node (DBIS 138 bridge host)
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=ubuntu
WorkingDirectory=${REPO}
ExecStart=${REPO}/bin/onexd -datadir ${REPO}/data -genesis ${REPO}/configs/genesis.json -api :8545 -listen :30303
EnvironmentFile=/etc/onex/onex.env
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
UNIT

sudo tee /etc/systemd/system/onex-bridge.service >/dev/null <<UNIT
[Unit]
Description=OneX bridge — ledger to DBIS Chain 138
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

sudo systemctl daemon-reload
sudo systemctl enable onexd onex-bridge
sudo systemctl restart onexd onex-bridge

sleep 3
echo "==> health"
curl -sf "http://127.0.0.1:9338/health" && echo " bridge OK" || echo " bridge FAIL"
curl -sf "http://127.0.0.1:9338/bridge/ledger/status" | head -c 320; echo

echo ""
echo "PUBLIC_WALLET=http://${HOST_IP}:9338/wallet/"
echo "PUBLIC_LEDGER=http://${HOST_IP}:9338/wallet/#ledger"
echo "DBIS138_RPC=${DBIS_RPC}"
echo "Add MetaMask: Chain 138 | RPC ${DBIS_RPC} | Explorer https://explorer.d-bis.org"
