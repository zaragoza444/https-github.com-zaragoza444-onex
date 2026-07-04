# Deploy OneX to IDBIS / DBIS Chain 138

Bridge local ledger (M0/M1/NSB) to **DeFi Oracle Meta Mainnet** (chain ID **138**, SMOM-DBIS-138).

## Network (MetaMask)

| Field | Value |
|-------|--------|
| Network | Defi Oracle Meta Mainnet / DBIS |
| Chain ID | 138 |
| Symbol | ETH |
| RPC | https://rpc-core.d-bis.org |
| Explorer | https://explorer.d-bis.org |

Alternate RPCs: `https://rpc.d-bis.org`, `https://rpc2.d-bis.org`, `https://rpc.defi-oracle.io`

## Deploy on chain 138 server

```bash
ssh ubuntu@YOUR_CHAIN138_SERVER
git clone https://github.com/zaragoza444/onex.git ~/onex
cd ~/onex
bash scripts/deploy-dbis-138.sh
```

Or copy env first:

```bash
sudo mkdir -p /etc/onex
sudo cp deploy/env.dbis-138.example /etc/onex/onex.env
# edit ONEX_API_KEY, ONEX_EVM_SENDER_KEY, ONEX_EVM_HOLDER
```

## Windows remote

```bat
set DBIS138_PUBLIC_HOST=YOUR_SERVER_IP
set SSH_PASS=your_password
scripts\deploy-dbis-138.bat
```

## After deploy

- Wallet: `http://SERVER:9338/wallet/#ledger`
- Default bridge tab: **Bridge to DBIS 138**
- Ledger API: `GET /bridge/ledger/status` → `defaultBridgeChain: "dbis-138"`

## Live settlement

Fund the sender wallet with **ETH on chain 138** for gas, then set:

```env
ONEX_EVM_SENDER_KEY=<64-hex-private-key>
ONEX_EVM_HOLDER=0xYourAddress
```

Restart: `sudo systemctl restart onex-bridge`

## Bridge example

```json
POST /bridge/ledger/settle
{
  "fromAccount": "m1-usd-checking",
  "amount": "500",
  "payoutAsset": "ETH",
  "kind": "real_crypto",
  "externalTo": "dbis-138:0xRecipientOnChain138"
}
```
