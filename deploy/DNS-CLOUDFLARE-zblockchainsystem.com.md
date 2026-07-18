# DNS — Cloudflare setup for zblockchainsystem.com

**Registrar / DNS:** Cloudflare (`lee.ns.cloudflare.com`, `emma.ns.cloudflare.com`)  
**VPS origin IPv4:** `51.75.64.28` (OVH — SSH port 22)  
**Problem today:** Cloudflare returns Railway `Application not found` — origin is wrong.

---

## Fix in Cloudflare Dashboard

1. Log in: https://dash.cloudflare.com  
2. Select zone **zblockchainsystem.com**  
3. Go to **DNS** → **Records**

### Delete or fix wrong records

Remove any of these if present:

| Type | Name | Problem |
|------|------|---------|
| CNAME | `@` or `www` | Points to Railway / Render / parking |
| A | `@` | Points to `76.53.10.34` or other non-VPS IP |
| AAAA | `@` | Points away from your stack (optional to remove) |

### Add / update these records

| Type | Name | Content | Proxy | TTL |
|------|------|---------|-------|-----|
| **A** | `@` | `51.75.64.28` | **DNS only** (grey cloud) recommended first | Auto |
| **A** | `www` | `51.75.64.28` | Same as `@` | Auto |

**Grey cloud (DNS only)** while testing — traffic goes straight to your VPS nginx on port 80.

After Z Bank is live, you can turn **Proxied** (orange cloud) on for DDoS/TLS — ensure origin serves HTTP on port 80.

### SSL/TLS (after grey-cloud works)

**SSL/TLS** → Overview → **Flexible** or **Full** (if you add certbot on VPS use **Full**).

---

## Sister domain: blockchainsystem.com

Point `@` and `www` A records to **`51.75.64.28`** as well (remove parking `76.223.54.146`, `13.248.169.48`).

---

## Verify (wait 2–10 min after save)

```bash
# Grey cloud: should show VPS IP directly
dig +short zblockchainsystem.com

# Endpoints must return JSON / Z Bank HTML (not Railway 404)
curl -sf https://zblockchainsystem.com/bridge/payments/status
curl -sfI https://zblockchainsystem.com/payments/
bash scripts/verify-production-domains.sh
```

Expected `dig` with grey cloud: `51.75.64.28`  
Expected payments status: `"framework":"zbank"`

---

## After DNS propagates — deploy bridge on VPS

```bash
gh secret set SSH_PASS --body "YOUR_UBUNTU_VPS_PASSWORD"
gh workflow run auto-deploy-production.yml
```

Or on the server:

```bash
ssh ubuntu@51.75.64.28
cd ~/onex && git pull origin main && bash scripts/fix-all-system.sh
```

Public URLs use **https://zblockchainsystem.com** only (no raw IP in app links).
