# OneX — DNS records for onexproduction.com

Replace `YOUR_VPS_IP` with your server public IPv4 (e.g. `zblockchainsystem.com`).

## Website (required)

| Type | Name | Content | Proxy | TTL |
|------|------|---------|-------|-----|
| A | `@` | YOUR_VPS_IP | DNS only (grey cloud) or Proxied | Auto |
| A | `www` | YOUR_VPS_IP | Same as `@` | Auto |

**Cloudflare:** If using orange-cloud (Proxied), TLS is handled by Cloudflare — still run certbot on VPS for origin or use Full (strict) with origin cert.

**Registrar-only (no Cloudflare):** Add the two A records above. Wait 5–30 minutes for propagation.

---

## Business email — Option A: Cloudflare Email Routing (free)

1. Add domain to Cloudflare and point nameservers to Cloudflare.
2. **Email → Email Routing → Get started → Enable**.
3. Verify destination inbox (e.g. your Gmail).
4. **Routing rules → Create address** for each:

| Custom address | Action |
|----------------|--------|
| `hello@onexproduction.com` | Send to → your inbox |
| `business@onexproduction.com` | Send to → your inbox |
| `support@onexproduction.com` | Send to → your inbox |
| `security@onexproduction.com` | Send to → your inbox |

Cloudflare adds MX automatically. Do **not** add conflicting MX at your registrar.

Optional (after routing works):

| Type | Name | Content |
|------|------|---------|
| TXT | `@` | `v=spf1 include:_spf.mx.cloudflare.net ~all` |
| TXT | `_dmarc` | `v=DMARC1; p=none; rua=mailto:hello@onexproduction.com` |

---

## Business email — Option B: Google Workspace

| Type | Name | Priority | Content |
|------|------|----------|---------|
| MX | `@` | 1 | `ASPMX.L.GOOGLE.COM` |
| MX | `@` | 5 | `ALT1.ASPMX.L.GOOGLE.COM` |
| MX | `@` | 5 | `ALT2.ASPMX.L.GOOGLE.COM` |
| MX | `@` | 10 | `ALT3.ASPMX.L.GOOGLE.COM` |
| MX | `@` | 10 | `ALT4.ASPMX.L.GOOGLE.COM` |
| TXT | `@` | — | `v=spf1 include:_spf.google.com ~all` |
| TXT | `_dmarc` | — | `v=DMARC1; p=quarantine; rua=mailto:hello@onexproduction.com` |

Create users or **Groups → Aliases** for hello, business, support, security in Admin console.

---

## Verify (PowerShell)

```powershell
.\scripts\deploy-onexproduction.ps1 -VpsIp YOUR_VPS_IP
```

```powershell
Resolve-DnsName onexproduction.com -Type A
Resolve-DnsName onexproduction.com -Type MX
```

---

## Deploy on VPS (after DNS points to VPS)

```bash
ssh ubuntu@YOUR_VPS_IP
git clone https://github.com/zaragoza444/onex.git /opt/onex
cd /opt/onex
cp deploy/env.onexproduction.example .env
nano .env   # set ONEX_API_KEY

CERTBOT_EMAIL=hello@onexproduction.com ./scripts/deploy-onexproduction.sh
```

Then open:

- https://onexproduction.com/ — marketing site  
- https://onexproduction.com/wallet/ — wallet  
- https://onexproduction.com/contact.html — business email  
