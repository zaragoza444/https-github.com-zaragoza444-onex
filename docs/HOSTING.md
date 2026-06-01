# Anakatech Gitea Pages + bridge API

| Component | URL |
|-----------|-----|
| **Git** | https://git.anakatech.llc/zardashtways44/onex-blockchain |
| **Wallet UI (Pages)** | https://git.anakatech.llc/pages/zardashtways44/onex-blockchain/wallet/ |
| **Bridge API** | Your hosted `onex-bridge` (required for send/swap/AI) |

**Quick connect (external):** open  
`https://git.anakatech.llc/pages/zardashtways44/onex-blockchain/wallet/?bridge=https://YOUR-bridge.onrender.com`  
or set the URL under **Settings → Bridge API**.

Confirm the exact Pages URL in your repo on [Anakatech Gitea](https://git.anakatech.llc/) → **Settings → Pages** after the first workflow run.

## Step 1 — Deploy bridge (Render, recommended)

1. Open [Render Dashboard](https://dashboard.render.com/) → **New** → **Blueprint**
2. Connect repo (Gitea or GitHub mirror)
3. Apply [`render.yaml`](../render.yaml) (creates `onex-node` + `onex-bridge`)
4. When finished, copy the **onex-bridge** public URL, e.g. `https://onex-bridge-xxxx.onrender.com`

Free tier sleeps after inactivity; use a paid plan or your own VPS for production.

## Step 2 — Wire bridge URL into Pages wallet

**Option A — Gitea variable (best)**

1. Repo on https://git.anakatech.llc/ → **Settings** → **Actions** → **Variables**
2. Add: `ONEX_BRIDGE_PUBLIC_URL` = `https://onex-bridge-xxxx.onrender.com` (no trailing slash)
3. **Actions** → **Gitea Pages** → **Run workflow**

**Option B — Local script**

```powershell
.\scripts\set-bridge-url.ps1 -BridgeUrl "https://onex-bridge-xxxx.onrender.com"
git add docs/wallet/config.js
git commit -m "Configure bridge URL for Pages wallet"
git push gitea main
```

## Step 3 — CORS on bridge

On Render, `ONEX_CORS_ORIGINS` includes `https://git.anakatech.llc` in `render.yaml`.

If you host bridge elsewhere:

```env
ONEX_CORS_ORIGINS=https://git.anakatech.llc,https://your-bridge-host
```

## Step 4 — Mobile app

[`mobile/.env`](../mobile/.env) should use the Gitea Pages wallet URL. See [`mobile/.env.example`](../mobile/.env.example).

## Push to Anakatech Gitea

```powershell
git remote set-url gitea https://git.anakatech.llc/zardashtways44/onex-blockchain.git
git push -u gitea main
```

Create the empty repository on https://git.anakatech.llc/ first if push fails with “repository not found”.
