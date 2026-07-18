#!/usr/bin/env bash
# Verify Z Bank bridge endpoints (local or production).
# Usage: bash scripts/verify-zbank-local.sh [BASE_URL] [API_KEY]
set -euo pipefail

BASE="${1:-http://127.0.0.1:9338}"
API_KEY="${2:-${ONEX_API_KEY:-}}"
PASS=0
FAIL=0

check() {
  local name="$1"
  shift
  if "$@" >/dev/null 2>&1; then
    echo "  OK  $name"
    PASS=$((PASS + 1))
  else
    echo "  FAIL $name"
    FAIL=$((FAIL + 1))
  fi
}

json_field() {
  local url="$1" field="$2"
  curl -sf "$url" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('$field',''))" 2>/dev/null
}

echo "==> Z Bank verification: $BASE"
echo ""

echo "-- Payment gateway"
FW=$(json_field "$BASE/bridge/payments/status" framework)
ENABLED=$(json_field "$BASE/bridge/payments/status" enabled)
echo "  framework=$FW enabled=$ENABLED"
if [ "$FW" = "zbank" ]; then echo "  OK  framework is zbank"; PASS=$((PASS+1)); else echo "  FAIL framework"; FAIL=$((FAIL+1)); fi
if [ "$ENABLED" = "True" ] || [ "$ENABLED" = "true" ] || [ "$ENABLED" = "1" ]; then echo "  OK  payments enabled"; PASS=$((PASS+1)); else echo "  FAIL payments disabled"; FAIL=$((FAIL+1)); fi

echo ""
echo "-- Bank accounts"
ACCOUNTS=$(curl -sf "$BASE/bridge/bank/accounts" | python3 -c "import sys,json; print(' '.join(a['id'] for a in json.load(sys.stdin).get('accounts',[])))")
for id in zbank-usd-checking zbank-usd-safeguarded zbank-usd-treasury; do
  if echo "$ACCOUNTS" | grep -q "$id"; then echo "  OK  $id"; PASS=$((PASS+1)); else echo "  FAIL missing $id"; FAIL=$((FAIL+1)); fi
done

echo ""
echo "-- Ledger fund classes"
CLASSES=$(curl -sf "$BASE/bridge/ledger/status" | python3 -c "import sys,json; print(' '.join(json.load(sys.stdin).get('fundClasses',[])))")
for fc in m1 m2 m3 m4; do
  if echo "$CLASSES" | grep -q "$fc"; then echo "  OK  fund class $fc"; PASS=$((PASS+1)); else echo "  FAIL missing $fc"; FAIL=$((FAIL+1)); fi
done

echo ""
echo "-- Officer store"
READY=$(json_field "$BASE/bridge/bank/officer/status" ready)
echo "  ready=$READY"
if [ "$READY" = "True" ] || [ "$READY" = "true" ]; then echo "  OK  officer ready"; PASS=$((PASS+1)); else echo "  WARN officer not ready (run /bridge/bank/officer/ensure)"; fi

echo ""
echo "-- Static assets"
check "payments portal" curl -sf -o /dev/null "$BASE/payments/"
check "zbank logo" curl -sf -o /dev/null "$BASE/payments/assets/zbank-logo.png"
check "dashboards hub" curl -sf -o /dev/null "$BASE/dashboards/"
check "payment dashboard" curl -sf -o /dev/null "$BASE/payments/dashboard/"

if [ -n "$API_KEY" ]; then
  echo ""
  echo "-- Officer verify (API key provided)"
  VERIFY=$(curl -sf -X POST "$BASE/bridge/bank/officer/verify" \
    -H "Content-Type: application/json" \
    -H "X-OneX-Api-Key: $API_KEY" \
    -d '{"pin":"'"${ONEX_ZBANK_OFFICER_PIN:-918273}"'","signature":"'"${ONEX_ZBANK_OFFICER_SIGNATURE:-ProdSignature-DSSBOAT-01}"'"}' \
    | python3 -c "import sys,json; print(json.load(sys.stdin).get('valid',False))" 2>/dev/null || echo "false")
  if [ "$VERIFY" = "True" ] || [ "$VERIFY" = "true" ]; then
    echo "  OK  officer PIN+signature verify"
    PASS=$((PASS+1))
  else
    echo "  FAIL officer verify (ensure officer seeded and env PIN/signature match)"
    FAIL=$((FAIL+1))
  fi
fi

echo ""
echo "==> Results: $PASS passed, $FAIL failed"
[ "$FAIL" -eq 0 ]
