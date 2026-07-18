#!/usr/bin/env bash
# Run ON THE VPS (OVH web console or SSH) — no GitHub secrets required.
# Paste in ubuntu shell:
#   curl -fsSL https://raw.githubusercontent.com/zaragoza444/https-github.com-zaragoza444-onex/main/scripts/vps-console-bootstrap.sh | bash
set -euo pipefail

REPO="${ONEX_REPO:-$HOME/onex}"
GITHUB="${GITHUB_REPO:-https://github.com/zaragoza444/https-github.com-zaragoza444-onex.git}"
DOMAIN="${ONEX_PRODUCTION_DOMAIN:-zblockchainsystem.com}"

echo "==> Z Bank VPS bootstrap @ ${DOMAIN}"

if [ ! -d "$REPO/.git" ]; then
  git clone "$GITHUB" "$REPO"
fi
cd "$REPO"
git fetch origin main
git reset --hard origin/main

export ONEX_PRODUCTION_DOMAIN="$DOMAIN"
export PATH="/usr/local/go/bin:$HOME/go/bin:$PATH"

bash scripts/fix-all-system.sh

echo ""
echo "=== BOOTSTRAP DONE ==="
echo "https://${DOMAIN}/payments/"
echo "https://${DOMAIN}/dashboards/"
echo "https://${DOMAIN}/bridge/payments/status"
echo ""
echo "Set officer PIN/signature + Stripe in /etc/onex/onex.env if not done yet."
