# Astra ↔ OneX — Reciprocal Ecosystem Link

Linked-systems inventory for Cursor / Astra agents. **No passwords, PINs, or API keys in this file.**

Live secrets (local only): copy [`ECOSYSTEM-SECRETS.env.example`](ECOSYSTEM-SECRETS.env.example) → `ECOSYSTEM-SECRETS.env` (gitignored).

Probed from Cursor cloud agent: **2026-07-13**.

---

## 1. SSH link-up (agent key)

### Agent public key (install on Astra + CT59)

```
ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOhaLewYwzS4+21uaywhHRjqFb0EWiCR7vtv8JkHTiiv cursor-agent-ecosystem-link@onex
```

Fingerprint: `SHA256:bI9OnFS3hiSjFrv3HCdRGjZHjfWsefgZCeJtkpNrj2s`

### Install on Astra (`root@65.181.23.219`, SSH port **8443**)

```bash
mkdir -p ~/.ssh && chmod 700 ~/.ssh
echo 'ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOhaLewYwzS4+21uaywhHRjqFb0EWiCR7vtv8JkHTiiv cursor-agent-ecosystem-link@onex' >> ~/.ssh/authorized_keys
chmod 600 ~/.ssh/authorized_keys
```

### Install on CT59 (from Astra → `root@192.168.1.59`)

```bash
ssh root@192.168.1.59 'mkdir -p ~/.ssh && chmod 700 ~/.ssh && echo "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOhaLewYwzS4+21uaywhHRjqFb0EWiCR7vtv8JkHTiiv cursor-agent-ecosystem-link@onex" >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys'
```

### Agent connect commands (after key is installed)

```bash
ssh -p 8443 root@65.181.23.219
# then hop:
ssh root@192.168.1.59
```

### Auth blocker (2026-07-13)

This agent **could not** append `authorized_keys` remotely:

- TCP `65.181.23.219:8443` is **OPEN**
- OpenSSH client connects, then **`kex_exchange_identification: Connection reset by peer`** (no SSH banner)
- No root SSH password was provided; BatchMode key auth fails until the server accepts this handshake **and** the pubkey is installed

**Action needed from Astra side:** allowlist this agent’s egress IP for SSH on `:8443` (or fix whatever is resetting the KEX), then install the pubkey above. Optionally provide a one-time root password/key so the agent can install itself.

---

## 2. ONEX / Z side

### Primary VPS

| Field | Value |
|-------|-------|
| IPv4 | `51.75.64.28` |
| SSH | `ubuntu@51.75.64.28` port **22** (password via GitHub secret `SSH_PASS` / OVH panel) |
| Provider | OVH (documented) |
| App root | `/opt/onex` (typical deploy) |
| Env | `/etc/onex/onex.env` or project `.env` on VPS |

### Firewall / ports (VPS)

| Port | Service |
|------|---------|
| 22 | SSH |
| 80 / 443 | HTTP(S) / nginx |
| 9338 | OneX bridge + wallet + payments |
| 8545 | OneX node (JSON-RPC / explorer) |
| 9340 | BSC Token Lab / Flash Coin |
| 30303 | P2P |
| 3100 | NovaBank connector (VPS instance) |

### Domains

| Domain | Role |
|--------|------|
| `onexproduction.com` | Marketing + wallet + payments (target) |
| `novatrustee.digital` | Trustee / wallet |
| `zblockchainsystem.com` | Payment gateway production |
| `blockchainsystem.com` | Payment gateway alt |
| `zaragoza444.github.io/onex/` | GitHub Pages wallet |
| `git.anakatech.llc` | Gitea + Pages wallet |

### OneX bridge API (port 9338)

```
GET  /health
GET  /bridge/production/status
GET  /bridge/bridge7/status
GET  /bridge/bridge7/ledgers
POST /bridge/bridge7/sync
GET  /bridge/bank/hybx/middleware/status
GET  /wallet/
GET  /payments/
POST /bridge/payments/webhook   # Stripe
```

Optional write auth: header / wallet setting `ONEX_API_KEY` (see secrets file).

### Nova Bank Online (OneX module)

Runs as part of `onex-bridge` (not a separate Railway app in this repo). CIS: [`docs/cis/CIS-Nova-Bank-Online-v1.md`](../docs/cis/CIS-Nova-Bank-Online-v1.md). SWIFT brand: `NSBKLAL2X`. External deps: HYBX `https://api.hybrix.io`, Fineract `https://fineract.hybxfinance.com/fineract-provider`.

### NovaBank VPS instance (shared note)

| Field | Value |
|-------|-------|
| Host | `51.75.64.28:3100` |
| Login email / PIN | See `ECOSYSTEM-SECRETS.env` (`NOVABANK_VPS_*`) |

### Chains known to OneX

| Chain | ID | RPC / notes |
|-------|-----|-------------|
| OneX native | (Ed25519 PoW) | Node `:8545` |
| DBIS | 138 | Public `https://rpc-core.d-bis.org` · explorer `https://explorer.d-bis.org` |
| Nova 1 | 22016 (`0x5600`) | Repo placeholder `https://rpc.nova1.chain` — **live node is on Astra** (see §3) |
| BSC / ETH / etc. | — | Public RPCs in `configs/chains.json` |

### Hosting notes

- This repo deploys bridge via **VPS Docker/systemd** and optionally **Render** (`render.yaml`) — **not Railway**.
- Auto-deploy: GitHub Actions → SSH `ubuntu@51.75.64.28` using `SSH_PASS`.

### NEED_FROM_Z (fill `ECOSYSTEM-SECRETS.env`)

| Secret | Where to get it |
|--------|-----------------|
| `ONEX_VPS_SSH_PASSWORD` / `SSH_PASS` | OVH panel / GitHub Actions secrets |
| `ONEX_API_KEY` | VPS `/etc/onex/onex.env` or deploy output |
| Stripe `sk_` / `pk_` / `whsec_` | Stripe dashboard / GitHub secrets |
| Fineract username/password | HYBX / Fineract admin |
| `ONEX_EVM_SENDER_KEY` | VPS `~/.onex/evm-sender.key` or env |
| `ONEX_BRIDGE_PUBLIC_URL` / Render URL | Gitea/GitHub Actions variables |

---

## 3. ASTRA / AnakaTech side

### Astra server (main)

| Field | Value |
|-------|-------|
| IP | `65.181.23.219` |
| SSH | `ssh root@65.181.23.219 -p 8443` |
| OS | Proxmox CT (Debian-based) |
| Tunnel | Cloudflare tunnel `anakatech-astra` |

### PM2 services (Astra)

| Service | Port | Role |
|---------|------|------|
| `anakabank-api` | 4003 | Main bank API (deposits, withdrawals, transfers) |
| `anakabank-ledger` | 3009 | Ledger service |
| `novabank-connector` | 3100 | NovaBank bridge connector |
| `multi-chain` | ? | Multi-chain operations |
| `securitization-engine` | ? | Asset securitization |
| `compliance-service` | 3050 | KYC/AML |
| `stripe-offramp` | ? | Stripe fiat off-ramp |
| `apex-receiver` | ? | Apex clearing receiver |
| `bank-terminal` | ? | Legacy bank terminal gateway |
| `signet-gate-api` | 8099 | Signet Wallet gate API |
| `ledger-pro` | 3009 | Ledger Pro |
| `anakaswap-web` | 3008 | AnakaSwap frontend |
| `anakaswap-health-proxy` | ? | AnakaSwap health monitor |
| `citadel-api` | ? | Citadel API |
| `citadel-web` | 3005 | Citadel web UI |
| `citadel-mcp` | ? | Citadel MCP |
| `tornation-command` | 3001 | Command handler |
| `heartbeat-fleet` | ? | Fleet heartbeat monitor |
| `anaka-kids` | 3101 | Anaka Kids app |

### Astra domains (Cloudflare tunnel)

`anakatech.llc`, `anakabank.anakatech.llc`, `api.anakachain.com`, `novabank-connector.anakatech.llc`, `signetwallet.com`, `anakaswap.anakatech.llc`, `novaone.anakatech.llc`, `citadel.anakatech.llc`, (+ ~40 more).

Credentials for AnakaBank admin, NovaBank Railway, htpasswd: **`ECOSYSTEM-SECRETS.env`**.

### CT59 (AnakaChain server)

| Field | Value |
|-------|-------|
| IP | `192.168.1.59` (LAN; reach from Astra only) |
| SSH | `ssh root@192.168.1.59` (from Astra) |

| Port | Service |
|------|---------|
| 4000 | `anakabank-gateway` (Iroha API proxy, `x-api-key`) |
| 4002 | `anakachain-bridge-service` |
| 4003 | `chain5-consumer` API |
| 4005 | `crypto-ledger` (Docker) |
| 4006 | `anakachain-chain5-consumer` (Iroha CLI wrapper) |
| 4007 | chain service |
| 4008 | additional service |
| 3030 | AnakaBank API |
| 3020 | AnakaBank ledger |
| 3098 | AnakaBank direct-login |
| 3100 | NovaBank connector (VPS instance path / CT copy) |
| 8555 | AnakaChain Besu node (chain **11013**, QBFT) |
| 5432 | PostgreSQL (`anakachain` DB) |
| 6379 | Redis |

### AnakaChain sub-chains

| ID | Role |
|----|------|
| 11011 | Main (settlement) |
| 11012 | Asset |
| 11013 | Bridge (primary, live) |
| 11014 | Sovereign |
| 11015 | Enclave |

### Iroha (on CT59)

| Field | Value |
|-------|-------|
| Domain | `anakachain` |
| Assets | USD, EUR, AUD, BTC, ETH, USDC, USDT, NOVA, NRW, + more |
| Treasury | See `IROHA_TREASURY_ACCOUNT` in secrets env |
| Gateway API key | See `IROHA_GATEWAY_API_KEY` in secrets env |

### CT59 domains (Cloudflare tunnel)

`anakachain.com`, `rpc.anakachain.com`, `bridge.anakachain.com`, `bank.anakachain.com`, `explorer.anakachain.com`, (+ ~30 more).

### Public AnakaBank API

| Item | Value |
|------|-------|
| Base | `https://api.anakachain.com/api/v1/` |
| Deposit | `POST /api/v1/transactions/deposit` |
| Auth | `x-api-key` header |
| Swagger | `https://anakabank-api.anakatech.llc/api-docs` |

---

## 4. NovaOne Chain (Chain ID 22016) — live on Astra

| Field | Value |
|-------|-------|
| Chain ID | **22016** |
| Consensus | QBFT (Besu 24.12.0) |
| RPC (host-local) | `http://127.0.0.1:8554` (socat → Docker) |
| Node RPCs | `8551`, `8552`, `8553` (node1/2/3) |
| Enode | `enode://43896ef1deae764f9c8df0cefbf16cdcc3631fcc3824b020e0e89e7895807d8644c02f485bb56bfe183e434db43cdd7e40c154f089eacf976dd99d0ad040f832@172.20.0.11:30303` |
| Gas price | 0 (zero base fee) |
| Docker network | `nova-net` `172.20.0.0/16` |

**External RPC:** not published as a public URL in this handoff (domain `novaone.anakatech.llc` returned HTTP 404 from cloud probe). Prefer Cloudflare tunnel mapping or SSH tunnel:

```bash
ssh -p 8443 -L 8554:127.0.0.1:8554 root@65.181.23.219
# then http://127.0.0.1:8554
```

OneX repo CIS still lists placeholder `https://rpc.nova1.chain` — update `configs/chains.json` once an external RPC URL is stable.

**WebSocket / genesis / bridge contracts:** not provided in this handoff; pull from Astra Docker (`nova-net`) after SSH works.

---

## 5. Connectivity matrix (Cursor cloud agent → services)

Probed **2026-07-13**.

### TCP

| Target | Result |
|--------|--------|
| `65.181.23.219:8443` | OPEN (TCP accept), SSH KEX **reset by peer** |
| `51.75.64.28:22` | OPEN |
| `51.75.64.28:80` | OPEN |
| `51.75.64.28:443` | OPEN |
| `51.75.64.28:3100` | OPEN |
| `51.75.64.28:8545` | OPEN |
| `51.75.64.28:9338` | OPEN (TCP), HTTP **reset by peer** |

### HTTP(S)

| URL | HTTP |
|-----|------|
| `https://api.anakachain.com/` | 200 |
| `https://api.anakachain.com/api/v1/` | 401 (auth required — expected) |
| `https://anakabank-api.anakatech.llc/api-docs` | **502** |
| `https://anakabank.anakatech.llc/` | 200 |
| `https://anakatech.llc/` | 200 |
| `https://novaone.anakatech.llc/` | **404** |
| `https://novabank-connector.anakatech.llc/` | **502** |
| `https://signetwallet.com/` | 200 |
| `https://anakaswap.anakatech.llc/` | 200 |
| `https://citadel.anakatech.llc/` | 401 |
| `https://anakachain.com/` | 200 |
| `https://rpc.anakachain.com/` | 201 |
| `https://bridge.anakachain.com/` | 201 |
| `https://bank.anakachain.com/` | 200 |
| `https://explorer.anakachain.com/` | 200 |
| `http://51.75.64.28:9338/health` | **ERR** (connection reset after TCP connect) |
| `http://51.75.64.28:8545/health` | 200 `{"status":"ok"}` |
| `http://51.75.64.28:3100/` | 200 (HTML) |
| `https://onexproduction.com/` | ERR / unreachable from agent |
| `https://novatrustee.digital/` | ERR / unreachable from agent |
| `https://zblockchainsystem.com/` | ERR / unreachable from agent |
| `https://blockchainsystem.com/` | 200 |
| `https://zaragoza444.github.io/onex/` | 404 |
| `https://rpc-core.d-bis.org/` | 405 (RPC alive; method not allowed on GET) |
| `https://git.anakatech.llc/` | 200 |

### Issues to fix

1. **Astra SSH `:8443`** — TCP open but KEX reset; cloud agent cannot log in or install keys until this is fixed/allowlisted.
2. **OneX bridge `:9338`** — TCP open but HTTP reset; node `:8545` is healthy — bridge process/firewall may need attention.
3. **502s** — `anakabank-api.anakatech.llc/api-docs`, `novabank-connector.anakatech.llc`.
4. **`novaone.anakatech.llc`** — 404; no external NovaOne RPC published yet.
5. Several OneX marketing domains unreachable from this agent (DNS or edge).

CT59 (`192.168.1.59`) is **not** reachable from the public internet; tests require Astra hop after SSH works.

---

## 6. ONEX product map (what OneX is / where it runs)

**OneX** is the Go blockchain + bridge platform in this repository:

- **Node** `onexd` — PoW chain, JSON-RPC `:8545`
- **Bridge** `onex-bridge` — wallet UI, Nova Bank Online, payments/Stripe, Bridge7 ledgers, HYBX, token platform `:9338`
- **BSC launcher** — Token Lab / Flash Coin `:9340`
- **Wallet** — browser UI under `/wallet/`, Chrome extension, Expo mobile
- **Runs on** — local Docker/Windows; production primarily **VPS `51.75.64.28`**; optional Render Blueprint

Access today without VPS SSH: public health on `:8545`; NovaBank UI on `:3100`; bridge `:9338` currently resetting from this agent.

---

## 7. Not in this handoff / out of scope gaps

| Item | Status |
|------|--------|
| NRW Bank UI / ERC contracts | Only Iroha asset name `NRW` listed; no contract addresses |
| Proxmox / Pandora full cluster | Only Astra CT + CT59 described; no Proxmox API endpoint here |
| FusionAGI / BigBrain | Not provided |
| NovaOne genesis / WS / bridge adapters | Need SSH to Astra Docker |
| Railway NovaBank login URL | Email/PIN in secrets; URL blank — fill when known |
| ZBank standalone | OneX has `zbank` payment-framework example only |

---

## 8. Reciprocal checklist

- [x] Non-secret inventory committed (`ECOSYSTEM-LINK.md`)
- [x] Secrets template committed (`ECOSYSTEM-SECRETS.env.example`)
- [x] Local secrets file written (gitignored `ECOSYSTEM-SECRETS.env`)
- [x] Agent SSH pubkey published
- [ ] Astra installs pubkey + fixes `:8443` KEX reset
- [ ] CT59 pubkey installed via Astra hop
- [ ] Z fills `NEED_FROM_Z` secrets in local env
- [ ] Re-probe SSH + local ports (`:4003`, `:8554`, CT59 `:5432`) after allowlist
