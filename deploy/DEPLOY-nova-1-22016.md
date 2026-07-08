# Deploy OneX to Nova 1 Chain 22016

Bridge Nova Bank Online ledger (M0/M1/NSB) to **Nova 1 Chain** (network ID **22016**, hex `0x5600`).

CIS reference: `docs/cis/CIS-Nova-1-Chain-22016-v1.md`

## Network (MetaMask)

| Field | Value |
|-------|--------|
| Network | Nova 1 Chain |
| Chain ID | **22016** |
| Chain ID (hex) | `0x5600` |
| Symbol | NOVA |
| RPC | `https://rpc.nova1.chain` (set live URL in env) |
| Explorer | `https://explorer.nova1.chain` |

## Deploy on bridge server

```bash
ssh ubuntu@YOUR_BRIDGE_SERVER
git clone https://github.com/zaragoza444/onex.git ~/onex
cd ~/onex
sudo mkdir -p /etc/onex
sudo cp deploy/env.nova-1-22016.example /etc/onex/onex.env
# Edit ONEX_API_KEY, ONEX_EVM_SENDER_KEY, NOVA1_RPC_URL
sudo systemctl restart onex-bridge
```

Or with Docker:

```bash
cp deploy/env.nova-1-22016.example .env
docker compose -f docker-compose.prod.yml up -d --build
```

## After deploy

- Wallet: `http://SERVER:9338/wallet/#ledger`
- Default bridge tab: **Nova 1 Chain**
- Ledger API: `GET /bridge/ledger/status` → `defaultBridgeChain: "nova-1"`
- Nova Bank: `GET /bridge/bank/status` → Nova Bank Online accounts

## Live settlement

Fund the sender wallet with **NOVA on chain 22016** for gas, then set:

```env
ONEX_EVM_SENDER_KEY=<64-hex-private-key>
ONEX_EVM_HOLDER=0xYourAddress
```

Restart: `sudo systemctl restart onex-bridge`

## Bridge example

```json
POST /bridge/ledger/settle
{
  "fromAccount": "nova-usd-checking",
  "amount": "500",
  "payoutAsset": "NOVA",
  "kind": "real_crypto",
  "externalTo": "nova-1:0xRecipientOnNova1"
}
```

## Verify

```bash
curl -s http://127.0.0.1:9338/bridge/ledger/status | jq '.defaultBridgeChain'
curl -s http://127.0.0.1:9338/bridge/bank/status | jq '.name, .online'
curl -s http://127.0.0.1:9338/bridge/production/status | jq '.api'
```

## Related

- Nova Bank Online CIS: `docs/cis/CIS-Nova-Bank-Online-v1.md`
- Integration matrix: `docs/cis/CIS-Nova-Integration-Matrix-v1.md`
- Bank ledger seed: `configs/bank-ledger.nova.example.json`
