#!/usr/bin/env python3
"""Deploy full OneX stack (node + wallet bridge + ledger) to ALI/ALLTRA ecosystem VPS."""
import os
import secrets
import sys
from pathlib import Path

import paramiko

ROOT = Path(__file__).resolve().parents[1]
HOST = os.environ.get("SSH_HOST", "51.75.64.28")
USER = os.environ.get("SSH_USER", "ubuntu")
REMOTE = os.environ.get("ALI_DEPLOY_ROOT", "/home/ubuntu/onex")
GITHUB = os.environ.get(
    "GITHUB_REPO", "https://github.com/zaragoza444/onex.git"
)


def remote_script(api_key: str) -> str:
    return f"""set -e
export PATH=/usr/local/go/bin:$HOME/go/bin:$PATH
REPO={REMOTE}
GITHUB={GITHUB}

if [ ! -d "$REPO/.git" ]; then
  git clone "$GITHUB" "$REPO" || true
fi
cd "$REPO"
git fetch origin main
git reset --hard origin/main

mkdir -p "$HOME/.onex/wallets" "$HOME/.onex/portfolios" "$HOME/.onex/ledger-import" bin data

echo "==> build binaries"
go build -o "$REPO/bin/onexd" ./cmd/onexd
go build -o "$REPO/bin/onex-bridge" ./cmd/onex-bridge
go build -o "$REPO/bin/bsc-launcher" ./bsc-launcher/server

sudo mkdir -p /etc/onex
sudo tee /etc/onex/onex.env >/dev/null <<EOF
ONEX_API_KEY={api_key}
ONEX_CORS_ORIGINS=http://{HOST}:9338,http://{HOST}:8545,https://zaragoza444.github.io,https://zaragoza444.github.io/onex,https://git.anakatech.llc,https://explorer.d-bis.org
ONEX_LEDGER_MODE=production
ONEX_BANK_LEDGER_FILE=$REPO/configs/bank-ledger.example.json
ONEX_PROJECT_ROOT=$REPO
ONEX_HOME_DIR=$HOME/.onex
ONEX_NODE_URL=http://127.0.0.1:8545
ONEX_BRIDGE_LISTEN=0.0.0.0:9338
ONEX_DEFAULT_BRIDGE_CHAIN=dbis-138
ONEX_PUBLIC_HOST={HOST}
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
WorkingDirectory={REMOTE}
ExecStart={REMOTE}/bin/onexd -datadir {REMOTE}/data -genesis {REMOTE}/configs/genesis.json -seeds {REMOTE}/configs/seeds-mainnet.json -api :8545 -listen :30303
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
WorkingDirectory={REMOTE}
ExecStart={REMOTE}/bin/onex-bridge -node http://127.0.0.1:8545 -listen 0.0.0.0:9338 -config /home/ubuntu/.onex/bridge.json -wallet /home/ubuntu/.onex/wallets/default.json
EnvironmentFile=/etc/onex/onex.env
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
UNIT

# Token Lab (existing ALI ecosystem dashboard :9340)
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
WorkingDirectory={REMOTE}
EnvironmentFile={REMOTE}/bsc-launcher/.env
Environment=BSC_LAUNCHER_ROOT={REMOTE}/bsc-launcher
Environment=BSC_LAUNCHER_DATA_DIR={REMOTE}/data/token-lab
ExecStart={REMOTE}/bin/bsc-launcher
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
curl -sf http://127.0.0.1:8545/health && echo " onexd OK" || echo " onexd FAIL"
curl -sf http://127.0.0.1:9338/health && echo " bridge OK" || echo " bridge FAIL"
curl -sf http://127.0.0.1:9338/bridge/health/green | head -c 300; echo
curl -sf http://127.0.0.1:9338/bridge/ledger/status | head -c 200; echo
curl -sf http://127.0.0.1:9340/health && echo " token-lab OK" || echo " token-lab FAIL"
systemctl is-active onexd onex-bridge onex-token-lab
echo "PUBLIC_WALLET=http://{HOST}:9338/wallet/"
echo "PUBLIC_LEDGER=http://{HOST}:9338/wallet/#ledger"
echo "PUBLIC_GREEN=http://{HOST}:9338/bridge/health/green"
echo "GITHUB_PAGES=https://zaragoza444.github.io/onex/wallet/?bridge=http://{HOST}:9338"
echo "PUBLIC_TOKEN_LAB=http://{HOST}:9340/"
"""


def main() -> int:
    password = os.environ.get("SSH_PASS")
    if not password:
        print("SSH_PASS required (ubuntu@51.75.64.28)", file=sys.stderr)
        return 1

    api_key = os.environ.get("ONEX_API_KEY") or secrets.token_urlsafe(32)
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    print(f"Connecting to {USER}@{HOST}...")
    client.connect(HOST, username=USER, password=password, timeout=45)

    _, stdout, stderr = client.exec_command(remote_script(api_key), get_pty=True)
    out = stdout.read().decode("utf-8", errors="replace")
    err = stderr.read().decode("utf-8", errors="replace")
    print(out)
    if err:
        print(err, file=sys.stderr)
    client.close()

    if "bridge OK" in out and "active" in out:
        print("\n=== PUBLIC LINKS ===")
        print(f"Wallet:    http://{HOST}:9338/wallet/")
        print(f"Ledger:    http://{HOST}:9338/wallet/#ledger")
        print(f"Bridge API: http://{HOST}:9338/bridge/ledger/status")
        print(f"Token Lab: http://{HOST}:9340/")
        print(f"Node:      http://{HOST}:8545/health")
        if not os.environ.get("ONEX_API_KEY"):
            print(f"\nGenerated ONEX_API_KEY (save it): {api_key}")
        return 0
    return 1


if __name__ == "__main__":
    raise SystemExit(main())
