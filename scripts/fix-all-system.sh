#!/usr/bin/env bash
# Fix the full Z Bank / blockchainsystem.com + zblockchainsystem.com production stack on the VPS.
# Safe to re-run. Preserves Stripe + officer secrets already in /etc/onex/onex.env.
#
# Usage (on VPS as ubuntu):
#   bash scripts/fix-all-system.sh
#
# Or from a machine with SSH:
#   SSH_PASS='...' python3 scripts/auto-deploy-vps.py
#   # (auto-deploy now calls this script when present)
set -euo pipefail

REPO="${ONEX_REPO:-$HOME/onex}"
GITHUB="${GITHUB_REPO:-https://github.com/zaragoza444/https-github.com-zaragoza444-onex.git}"
DOMAIN="${ONEX_PRODUCTION_DOMAIN:-zblockchainsystem.com}"
BRANCH="${ONEX_DEPLOY_BRANCH:-main}"
PORT="${ONEX_BRIDGE_PORT:-9338}"
ENV_FILE="/etc/onex/onex.env"

echo "==> Fix ALL system — Z Bank @ ${DOMAIN}"

sudo systemctl stop onex-bridge 2>/dev/null || true
if command -v docker >/dev/null 2>&1 && [ -f "${REPO}/docker-compose.prod.yml" ]; then
  (cd "$REPO" 2>/dev/null && docker compose -f docker-compose.prod.yml stop onex-bridge 2>/dev/null) || true
fi
sudo fuser -k 9338/tcp 2>/dev/null || true
sudo fuser -k 9339/tcp 2>/dev/null || true
sleep 2

if [ ! -d "$REPO/.git" ]; then
  git clone "$GITHUB" "$REPO"
fi
cd "$REPO"
git fetch origin "$BRANCH"
git checkout "$BRANCH"
git reset --hard "origin/${BRANCH}"

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

upsert_env() {
  local key="$1" val="$2"
  sudo mkdir -p /etc/onex
  if [ ! -f "$ENV_FILE" ]; then
    if [ -f "$REPO/deploy/env.zbank.production.example" ]; then
      sudo cp "$REPO/deploy/env.zbank.production.example" "$ENV_FILE"
    else
      sudo cp "$REPO/deploy/env.zblockchainsystem.com.example" "$ENV_FILE"
    fi
    local keygen
    keygen="$(openssl rand -base64 32 | tr -d '/+=' | head -c 32)"
    sudo sed -i "s/CHANGE_ME_LONG_RANDOM_SECRET/${keygen}/" "$ENV_FILE"
  fi
  if sudo grep -q "^${key}=" "$ENV_FILE" 2>/dev/null; then
    local cur
    cur="$(sudo grep "^${key}=" "$ENV_FILE" | head -1 | cut -d= -f2-)"
    # Do not clobber real secrets with placeholders
    if [[ "$val" == CHANGE_ME* ]] && [[ "$cur" != CHANGE_ME* ]] && [[ -n "$cur" ]]; then
      return 0
    fi
    sudo sed -i "s|^${key}=.*|${key}=${val}|" "$ENV_FILE"
  else
    echo "${key}=${val}" | sudo tee -a "$ENV_FILE" >/dev/null
  fi
}

echo "==> Ensure Z Bank production env (preserve secrets)"
upsert_env ONEX_LEDGER_MODE production
upsert_env ONEX_ONLINE_BANK 1
upsert_env ONEX_PROJECT_ROOT "$REPO"
upsert_env ONEX_PRODUCTION_DOMAIN "$DOMAIN"
upsert_env ONEX_CORS_ORIGINS "https://zblockchainsystem.com,https://www.zblockchainsystem.com,https://blockchainsystem.com,https://www.blockchainsystem.com,http://blockchainsystem.com,http://www.blockchainsystem.com,https://git.anakatech.llc,https://zaragoza444.github.io,http://51.75.64.28:9338"
upsert_env ONEX_BRIDGE_LISTEN "0.0.0.0:${PORT}"
upsert_env ONEX_BANK_LEDGER_FILE "${REPO}/configs/bank-ledger.zbank.production.json"
upsert_env ONEX_PAYMENT_GATEWAY 1
upsert_env ONEX_PAYMENT_GATEWAY_FILE "${REPO}/configs/payment-gateway.zbank.production.json"
upsert_env ONEX_PAYMENT_GATEWAY_FRAMEWORK zbank
upsert_env ONEX_PAYMENT_GATEWAY_PROVIDER stripe
upsert_env ONEX_ZBANK_OFFICERS_FILE "${REPO}/configs/zbank-officers.dssboat.example.json"

for var in ONEX_STRIPE_SECRET_KEY ONEX_STRIPE_PUBLISHABLE_KEY ONEX_STRIPE_WEBHOOK_SECRET \
           ONEX_ZBANK_OFFICER_PIN ONEX_ZBANK_OFFICER_SIGNATURE ONEX_API_KEY; do
  val="${!var:-}"
  if [ -n "$val" ]; then
    upsert_env "$var" "$val"
  fi
done

sudo tee /etc/systemd/system/onex-bridge.service >/dev/null <<UNIT
[Unit]
Description=OneX Bridge + Z Bank Payment Gateway
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=ubuntu
WorkingDirectory=${REPO}
EnvironmentFile=${ENV_FILE}
ExecStart=${REPO}/bin/onex-bridge -node http://127.0.0.1:8545 -listen 0.0.0.0:${PORT} -config ${HOME}/.onex/bridge.json -wallet ${HOME}/.onex/wallets/default.json
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

if ! curl -sf "http://127.0.0.1:${PORT}/health" >/dev/null 2>&1; then
  echo "WARN: :${PORT} failed — trying :9339"
  PORT=9339
  upsert_env ONEX_BRIDGE_LISTEN "0.0.0.0:${PORT}"
  sed -i "s/0.0.0.0:9338/0.0.0.0:9339/" "$HOME/.onex/bridge.json" 2>/dev/null || true
  sudo sed -i "s|listen 0.0.0.0:.*|listen 0.0.0.0:${PORT}|" /etc/systemd/system/onex-bridge.service
  sudo systemctl daemon-reload
  sudo systemctl restart onex-bridge
  sleep 8
fi

if ! curl -sf "http://127.0.0.1:${PORT}/health"; then
  echo "FAIL — bridge not responding"
  sudo journalctl -u onex-bridge -n 40 --no-pager || true
  exit 1
fi
echo " OK on :${PORT}"

# Nginx — force Z Bank routes, disable conflicting default that serves Nova SPA
if command -v nginx >/dev/null 2>&1; then
  echo "==> Install Z Bank nginx site"
  # Render port into conf
  sed "s/127.0.0.1:9338/127.0.0.1:${PORT}/g" "$REPO/deploy/nginx-vps-zblockchain.conf" \
    | sudo tee /etc/nginx/sites-available/zblockchain-onex >/dev/null
  sudo ln -sf /etc/nginx/sites-available/zblockchain-onex /etc/nginx/sites-enabled/zblockchain-onex
  # Remove common conflicting SPA defaults that steal /bridge/
  sudo rm -f /etc/nginx/sites-enabled/default 2>/dev/null || true
  sudo rm -f /etc/nginx/sites-enabled/nova-bank 2>/dev/null || true
  sudo rm -f /etc/nginx/sites-enabled/zblockchain 2>/dev/null || true
  sudo rm -f /etc/nginx/sites-enabled/onexproduction 2>/dev/null || true
  sudo rm -f /etc/nginx/sites-enabled/blockchainsystem 2>/dev/null || true
  # Disable any other enabled site still serving a catch-all SPA ahead of ours
  if [ -d /etc/nginx/sites-enabled ]; then
    for f in /etc/nginx/sites-enabled/*; do
      base="$(basename "$f")"
      [ "$base" = "zblockchain-onex" ] && continue
      if sudo grep -qE 'try_files .*index\.html|root .*/dist|root .*/spa' "$f" 2>/dev/null; then
        echo "Disabling conflicting SPA site: $base"
        sudo rm -f "$f"
      fi
    done
  fi
  sudo nginx -t && sudo systemctl reload nginx
  echo "nginx OK → Z Bank routes on :${PORT} for zblockchainsystem.com + blockchainsystem.com"
fi

echo "==> Verify (must be JSON, not HTML)"
PAY_JSON="$(curl -sf "http://127.0.0.1:${PORT}/bridge/payments/status" || true)"
echo "$PAY_JSON" | head -c 500; echo
if echo "$PAY_JSON" | grep -qiE '<!doctype|<html'; then
  echo "FAIL: bridge still returning HTML — check onex-bridge + nginx"
  exit 1
fi
if ! echo "$PAY_JSON" | grep -qiE 'enabled|payment|status'; then
  echo "FAIL: unexpected payments status body"
  exit 1
fi
curl -sf "http://127.0.0.1:${PORT}/bridge/bank/officer/status" | head -c 400; echo || true
curl -sf -o /dev/null -w "payments portal HTTP %{http_code}\n" "http://127.0.0.1:${PORT}/payments/" || true
curl -sf -o /dev/null -w "logo HTTP %{http_code}\n" "http://127.0.0.1:${PORT}/payments/assets/zbank-logo.png" || true
# Host-header checks as public domains will see them once DNS points here
for h in zblockchainsystem.com blockchainsystem.com; do
  code=$(curl -sf -o /tmp/onex-host-check -w "%{http_code}" -H "Host: $h" "http://127.0.0.1/bridge/payments/status" || echo 000)
  body=$(head -c 80 /tmp/onex-host-check 2>/dev/null || true)
  echo "Host $h /bridge/payments/status → HTTP $code | ${body}"
done

echo ""
echo "=== SYSTEM FIXED ==="
echo "http://blockchainsystem.com/payments/"
echo "http://blockchainsystem.com/bridge/payments/status"
echo "http://zblockchainsystem.com/payments/"
echo "http://zblockchainsystem.com/bridge/payments/status"
echo "http://${DOMAIN}/wallet/"
echo ""
echo "DNS: dig +short blockchainsystem.com  → must be 51.75.64.28 (not parking)"
echo "Verify: bash scripts/verify-production-domains.sh"
echo ""
echo "If officer status shows credentialsReady:0, set PIN+signature in ${ENV_FILE} then:"
echo "  curl -X POST -H \"X-OneX-Api-Key: \$ONEX_API_KEY\" http://127.0.0.1:${PORT}/bridge/bank/officer/ensure"
