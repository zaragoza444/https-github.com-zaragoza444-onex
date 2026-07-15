# Fix blockchainsystem.com — full production HTTP 200

**Right now:** `blockchainsystem.com` resolves to **parking** (`76.223.54.146`, `13.248.169.48`) and serves a lander HTML. It does **not** hit your VPS.

**Target:** A records → `51.75.64.28`, nginx proxies to `onex-bridge`, endpoints return **HTTP 200** with real Z Bank / bridge content (JSON or payments portal).

---

## 1. DNS (required — at the registrar)

| Type | Host | Value | TTL |
|------|------|-------|-----|
| A | `@` | `51.75.64.28` | 300 |
| A | `www` | `51.75.64.28` | 300 |

**Delete** parking / forwarding records (`76.223.54.146`, `13.248.169.48`) and any domain-forward “lander”.

```bash
dig +short blockchainsystem.com
# MUST be only: 51.75.64.28
```

---

## 2. Deploy bridge + nginx on VPS

```bash
# From laptop (needs VPS password):
gh secret set SSH_PASS --repo zaragoza444/https-github.com-zaragoza444-onex --body "YOUR_VPS_PASSWORD"
SSH_PASS='YOUR_VPS_PASSWORD' python3 scripts/auto-deploy-vps.py

# Or on the VPS:
cd ~/onex && git pull origin main && bash scripts/fix-all-system.sh
```

`fix-all-system.sh` installs `deploy/nginx-vps-zblockchain.conf` which serves **both**:
- `blockchainsystem.com`
- `zblockchainsystem.com`

---

## 3. Verify (must be JSON, not HTML lander/SPA)

```bash
bash scripts/verify-production-domains.sh
# or:
curl -sf http://blockchainsystem.com/bridge/payments/status
curl -sf http://blockchainsystem.com/health
curl -sf -o /dev/null -w "%{http_code}\n" http://blockchainsystem.com/payments/
```

Expected:
- `/bridge/payments/status` → **200** + JSON (`"enabled"` / payment fields)
- `/health` → **200** from bridge
- `/payments/` → **200** Z Bank portal HTML (not `/lander`)
