#!/usr/bin/env bash
# Verify production domains return HTTP 200 with real bridge/payments (not parking lander / Nova SPA).
set -euo pipefail

CANONICAL="${ONEX_PRODUCTION_DOMAIN:-zblockchainsystem.com}"
DOMAINS=(
  "${CANONICAL}"
  "www.${CANONICAL}"
  "${ONEX_VERIFY_DOMAIN:-blockchainsystem.com}"
  "www.blockchainsystem.com"
)

fail=0
PARKING_RE='76\.223\.54\.146|13\.248\.169\.48|76\.53\.10\.34'

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
  if echo "$body" | grep -qiE '<!doctype|<html|/lander|Application not found'; then
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
  if echo "$body" | grep -qiE '/lander|Application not found'; then
    echo "FAIL $url → parking lander or dead host"
    fail=1
    return
  fi
  echo "OK   $url → HTTP $code"
}

echo "==> DNS (canonical: ${CANONICAL})"
CANON_IP=$(dig +short "$CANONICAL" A 2>/dev/null | head -1 | tr -d ' ')
if [ -z "$CANON_IP" ]; then
  echo "  FAIL ${CANONICAL} — no A record (point @ and www to your VPS)"
  fail=1
else
  if echo "$CANON_IP" | grep -qE "$PARKING_RE"; then
    echo "  FAIL ${CANONICAL} → $CANON_IP (parking / wrong host — use VPS IPv4)"
    fail=1
  else
    echo "  OK   ${CANONICAL} → $CANON_IP"
  fi
fi

for d in blockchainsystem.com www.blockchainsystem.com; do
  ips=$(dig +short "$d" A 2>/dev/null | tr '\n' ' ')
  echo "  $d → $ips"
  if echo "$ips" | grep -qE "$PARKING_RE"; then
    echo "  !! PARKING DNS — set A @ and www to same IPv4 as ${CANONICAL}"
    fail=1
  elif [ -n "$CANON_IP" ] && ! echo "$ips" | grep -q "$CANON_IP"; then
    echo "  !! should match ${CANONICAL} (${CANON_IP})"
    fail=1
  fi
done

echo "==> Endpoints (https://${CANONICAL})"
check_json "https://${CANONICAL}/bridge/payments/status"
check_portal "https://${CANONICAL}/payments/"
check_portal "https://${CANONICAL}/dashboards/"
check_portal "https://${CANONICAL}/health"

for d in "${DOMAINS[@]}"; do
  [ "$d" = "$CANONICAL" ] && continue
  check_json "http://${d}/bridge/payments/status" || true
done

if [ "$fail" -ne 0 ]; then
  echo ""
  echo "NOT READY — point DNS to VPS, set gh secret SSH_PASS, run: bash scripts/fix-all-system.sh"
  exit 1
fi
echo ""
echo "ALL DOMAINS READY (${CANONICAL})"
