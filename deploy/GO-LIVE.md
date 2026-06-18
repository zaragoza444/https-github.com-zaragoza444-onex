# Go live — public OneX chain + wallet

## Quick (VPS already has SSH)

```powershell
# Windows — set ubuntu password, then:
set SSH_PASS=your-password
powershell -File scripts/go-live.ps1
```

```bash
# On the VPS directly:
ssh ubuntu@51.75.64.28
cd ~/onex && git pull origin main && bash scripts/deploy-ali-ecosystem.sh
```

## Public URLs

| Service | URL |
|---------|-----|
| Wallet | http://51.75.64.28:9338/wallet/ |
| Ledger + settlement | http://51.75.64.28:9338/wallet/#ledger |
| Green health | http://51.75.64.28:9338/bridge/health/green |
| GitHub Pages wallet | https://zaragoza444.github.io/onex/wallet/ |
| OneX node API | http://51.75.64.28:8545/health |
| DBIS Chain 138 RPC | https://rpc-core.d-bis.org |

## Firewall (on VPS)

```bash
sudo ufw allow 9338/tcp   # wallet bridge
sudo ufw allow 8545/tcp   # node API
sudo ufw allow 30303/tcp  # P2P
sudo ufw reload
```

## GitHub Pages

1. Push `main` to https://github.com/zaragoza444/onex
2. Settings → Pages → **GitHub Actions**
3. Set repo variable `ONEX_BRIDGE_PUBLIC_URL` = `http://51.75.64.28:9338`
4. Run workflow **GitHub Pages** (or push triggers it)

## DBIS 138 only host

Use `bash scripts/deploy-dbis-138.sh` on your chain-138 server instead of `deploy-ali-ecosystem.sh`.

## After deploy

- Fund EVM sender: check `GET /bridge/ledger/settlement/capabilities` → `evmSenderAddress`
- Set `ONEX_API_KEY` from deploy output in wallet Settings for write operations
