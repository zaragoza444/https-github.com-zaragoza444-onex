# Payment Gateway — Production Setup Walkthrough

**Recommended primary domain:** `onexproduction.com`  
**VPS IP:** `zblockchainsystem.com`

---

## Step 1 — Fix DNS (5–30 min propagation)

At your domain registrar, **delete** parking A records and add:

| Host | Type | Value |
|------|------|-------|
| `@` (onexproduction.com) | A | `zblockchainsystem.com` |
| `www` | A | `zblockchainsystem.com` |
| `@` (novatrustee.digital) | A | `zblockchainsystem.com` |

Verify:

```bash
dig +short onexproduction.com
# Must show only: zblockchainsystem.com
```

---

## Step 2 — Stripe account (live mode)

1. Log in to [Stripe Dashboard](https://dashboard.stripe.com)
2. Toggle **Test mode OFF** (live mode)
3. Go to **Developers → API keys**
4. Copy:
   - **Secret key** → `sk_live_...`
   - **Publishable key** → `pk_live_...`

Ensure Visa, Mastercard, and American Express are enabled under **Settings → Payment methods**.

---

## Step 3 — VPS env file

SSH or open **VPS web console** as `ubuntu`, then:

```bash
cd ~/onex || git clone https://github.com/zaragoza444/https-github.com-zaragoza444-onex.git ~/onex
cd ~/onex && git pull origin main

cp deploy/env.production.live.example /tmp/onex.env
nano /tmp/onex.env   # paste your Stripe keys, save
```

**Values to replace in the file:**

| Variable | Where to get it |
|----------|-----------------|
| `ONEX_API_KEY` | Auto-generated on deploy, or any long random string |
| `ONEX_STRIPE_SECRET_KEY` | Stripe → Developers → API keys → Secret key |
| `ONEX_STRIPE_PUBLISHABLE_KEY` | Stripe → Developers → API keys → Publishable key |
| `ONEX_STRIPE_WEBHOOK_SECRET` | Step 4 below (`whsec_...`) |

**One-command deploy** (after exporting Stripe keys in your shell):

```bash
export ONEX_STRIPE_SECRET_KEY=sk_live_YOUR_KEY
export ONEX_STRIPE_PUBLISHABLE_KEY=pk_live_YOUR_KEY
export ONEX_STRIPE_WEBHOOK_SECRET=whsec_YOUR_SECRET
bash scripts/apply-production-env.sh
```

---

## Step 4 — Stripe webhook

### Option A — Script (recommended)

After DNS points to your VPS and bridge is running:

```bash
export ONEX_STRIPE_SECRET_KEY=sk_live_YOUR_KEY
export ONEX_PRODUCTION_DOMAIN=onexproduction.com
bash scripts/setup-stripe-webhook.sh
```

Copy the printed `ONEX_STRIPE_WEBHOOK_SECRET=whsec_...` into your env file, then restart bridge.

### Option B — Stripe Dashboard (manual)

1. [Stripe → Webhooks](https://dashboard.stripe.com/webhooks) → **+ Add endpoint**
2. **Endpoint URL:**

   ```
   https://onexproduction.com/bridge/payments/webhook
   ```

3. **Events:**
   - `payment_intent.succeeded`
   - `payment_intent.payment_failed`

4. Save → open the endpoint → **Signing secret** → Reveal → copy `whsec_...`
5. Add to env:

   ```bash
   ONEX_STRIPE_WEBHOOK_SECRET=whsec_...
   ```

6. Restart:

   ```bash
   sudo systemctl restart onex-bridge
   # Docker: docker compose -f docker-compose.prod.yml restart onex-bridge
   ```

### Test webhook (Stripe Dashboard)

Send a test `payment_intent.succeeded` event. Expect HTTP **200** from your endpoint.

---

## Step 5 — TLS / HTTPS (after DNS propagates)

```bash
cd ~/onex
ONEX_PRODUCTION_DOMAIN=onexproduction.com \
CERTBOT_EMAIL=hello@onexproduction.com \
./scripts/deploy-onexproduction.sh
```

---

## Step 6 — Verify live

```bash
curl -s https://onexproduction.com/bridge/payments/status | python3 -m json.tool
```

Expected:

```json
{
  "enabled": true,
  "framework": "nova",
  "provider": "stripe",
  "stripeConfigured": true,
  "stripeLiveReady": true,
  "acceptedCards": ["visa", "mastercard", "amex"]
}
```

Open in browser:

| Page | URL |
|------|-----|
| Portal | https://onexproduction.com/payments/ |
| Donate | https://onexproduction.com/payments/?page=donate |
| Invoice | https://onexproduction.com/payments/?page=invoice |
| Collect | https://onexproduction.com/payments/?page=collect |

---

## GitHub Actions (optional — no manual SSH)

Add these **Repository secrets** (Settings → Secrets → Actions):

| Secret | Value |
|--------|-------|
| `SSH_PASS` | Ubuntu VPS password |
| `ONEX_STRIPE_SECRET_KEY` | `sk_live_...` |
| `ONEX_STRIPE_PUBLISHABLE_KEY` | `pk_live_...` |
| `ONEX_STRIPE_WEBHOOK_SECRET` | `whsec_...` |

Then: **Actions → Deploy payment gateway (production VPS) → Run workflow**

---

## Settlement destinations

Configured in `configs/payment-gateway.production.json`. To route to a specific bank, edit `settlementDestinations` and restart bridge — no code changes needed.

---

## Support checklist

- [ ] DNS → `zblockchainsystem.com`
- [ ] Bridge running (`curl https://zblockchainsystem.com/health`)
- [ ] Stripe live keys in env
- [ ] Webhook registered + `whsec_` in env
- [ ] `stripeLiveReady: true` in status API
- [ ] Test donation on `/payments/?page=donate`
