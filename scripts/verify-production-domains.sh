#!/usr/bin/env bash
# Verify production domains return HTTP 200 with real bridge/payments (not parking lander / Nova SPA).
set -euo pipefail

DOMAINS=(
  "${ONEX_VERIFY_DOMAIN:-blockchainsystem.com}"
  "www.blockchainsystem.com"
  "zblockchainsystem.com"
  "www.zblockchainsystem.com"
)

fail=0

check_json() {
  local url="$1"
  local body code
  code=$(curl -sS -o /tmp/onex-verify-body -w "%{http_code}" --max-time 20 "$url" || echo "000")
  body=$(head -c 400 /tmp/onex-verify-body 2>/dev/null || true)
  if [ "$code" != "200" ]; then
    echo "FAIL $url → HTTP $code"
    fail=1
    return
  fi
  if echo "$body" | grep -qiE '<!doctype|<html|/lander'; then
    echo "FAIL $url → HTML/SPA/lander (want JSON bridge)"
    echo "     ${body:0:120}"
    fail=1
    return
  fi
  if ! echo "$body" | grep -qE '\{|enabled|payment|status|ok'; then
    echo "WARN $url → 200 but unexpected body: ${body:0:120}"
  else
    echo "OK   $url → 200 JSON"
  fi
}

check_portal() {
  local url="$1"
  local code body
  code=$(curl -sS -o /tmp/onex-verify-body -w "%{http_code}" --max-time 20 "$url" || echo "000")
  body=$(head -c 300 /tmp/onex-verify-body 2>/dev/null || true)
  if [ "$code" != "200" ] && [ "$code" != "302" ]; then
    echo "FAIL $url → HTTP $code"
    fail=1
    return
  fi
  if echo "$body" | grep -qi '/lander'; then
    echo "FAIL $url → parking lander"
    fail=1
    return
  fi
  echo "OK   $url → HTTP $code"
}

echo "==> DNS"
for d in blockchainsystem.com zblockchainsystem.com; do
  ips=$(dig +short "$d" A 2>/dev/null | tr '\n' ' ')
  echo "  $d → $ips"
  if [ "$d" = "blockchainsystem.com" ] && echo "$ips" | grep -qE '76\.223\.54\.146|13\.248\.169\.48'; then
    echo "  !! PARKING DNS — set A @ and www to 51.75.64.28 (see deploy/FIX-blockchainsystem.com.md)"
    fail=1
  fi
  if ! echo "$ips" | grep -q '51.75.64.28'; then
    echo "  !! expected 51.75.64.28"
    fail=1
  fi
done

echo "==> Endpoints"
for d in "${DOMAINS[@]}"; do
  check_json "http://${d}/bridge/payments/status"
  check_portal "http://${d}/payments/"
  check_portal "http://${d}/health"
done

if [ "$fail" -ne 0 ]; then
  echo ""
  echo "NOT READY — fix DNS + run: bash scripts/fix-all-system.sh"
  exit 1
fi
echo ""
echo "ALL DOMAINS READY (HTTP 200 production)"
