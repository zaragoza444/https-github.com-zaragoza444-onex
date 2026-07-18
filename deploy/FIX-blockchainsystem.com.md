# Fix blockchainsystem.com — full production HTTP 200

**Right now:** `blockchainsystem.com` may resolve to **parking** (`76.223.54.146`, `13.248.169.48`) and serve a lander. It does **not** hit your VPS.

**Target:** A records → same VPS IPv4 as `zblockchainsystem.com`, nginx proxies to `onex-bridge`, endpoints return **HTTP 200** with real Z Bank content.

---

## 1. DNS (required — at the registrar)

| Type | Host | Value | TTL |
|------|------|-------|-----|
| A | `@` | *VPS IPv4 (match `dig +short zblockchainsystem.com`)* | 300 |
| A | `www` | *same VPS IPv4* | 300 |

**Delete** parking / forwarding records.

```bash
dig +short blockchainsystem.com zblockchainsystem.com
# Both must be the same VPS IPv4 (not parking)
```

---

## 2. Deploy bridge + nginx on VPS

```bash
gh secret set SSH_PASS --repo zaragoza444/https-github.com-zaragoza444-onex --body "YOUR_VPS_PASSWORD"
SSH_PASS='YOUR_VPS_PASSWORD' python3 scripts/auto-deploy-vps.py

# Or SSH to zblockchainsystem.com:
cd ~/onex && git pull origin main && bash scripts/fix-all-system.sh
```

`fix-all-system.sh` installs nginx for **both** `blockchainsystem.com` and `zblockchainsystem.com`.

---

## 3. Verify

```bash
bash scripts/verify-production-domains.sh
curl -sf https://zblockchainsystem.com/bridge/payments/status
curl -sf https://zblockchainsystem.com/payments/
```
