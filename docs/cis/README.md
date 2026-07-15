# CIS — Component Integration Specifications

Customer Integration Specifications (CIS) for **Nova Bank Online**, **Z Bank**, and **Nova 1 Chain** (network ID **22016**).

| Document | Description |
|----------|-------------|
| [CIS-Nova-Bank-Online-v1.md](./CIS-Nova-Bank-Online-v1.md) | Sovereign online banking — accounts, rails, API, deployment (M0/M1/NSB) |
| [CIS-Z-Bank-Online-v1.md](./CIS-Z-Bank-Online-v1.md) | Z Bank online banking — M1–M4 operational liquidity layers + payment gateway |
| [CIS-Nova-1-Chain-22016-v1.md](./CIS-Nova-1-Chain-22016-v1.md) | EVM chain 22016 — registry, settlement, token platform |
| [CIS-Nova-Integration-Matrix-v1.md](./CIS-Nova-Integration-Matrix-v1.md) | Cross-system flows between Nova bank and chain |

## Quick reference

| Component | ID | Key value |
|-----------|-----|-----------|
| Nova Bank Online | `nsb` / `nova` | SWIFT `NSBKLAL2X` |
| Z Bank | `zbank` | Framework `zbank`; fund classes M1–M4 |
| Nova 1 Chain | `nova-1` | Chain ID **22016** (`0x5600`) |

## Supporting files

- `configs/bank-ledger.nova.example.json` — Nova Bank account seed data
- `configs/bank-ledger.zbank.example.json` — Z Bank M0 + M1–M4 account seed data
- `configs/payment-gateway.zbank.example.json` — Z Bank payment gateway
- `configs/chains.json` — Nova 1 chain registry entry
- `deploy/env.nova-1-22016.example` — Combined Nova deploy environment
- `deploy/env.zbank.example` — Z Bank deploy environment
- `deploy/DEPLOY-nova-1-22016.md` — Nova deployment guide

## PDF exports

| PDF | Source |
|-----|--------|
| [CIS-Nova-Bank-Online-v1.pdf](./CIS-Nova-Bank-Online-v1.pdf) | Nova Bank Online CIS |
| [CIS-Nova-1-Chain-22016-v1.pdf](./CIS-Nova-1-Chain-22016-v1.pdf) | Nova 1 Chain 22016 CIS |
| [CIS-Nova-Integration-Matrix-v1.pdf](./CIS-Nova-Integration-Matrix-v1.pdf) | Integration matrix |
| [README.pdf](./README.pdf) | This index |

Regenerate PDFs:

```bash
python3 scripts/generate-cis-pdf.py
```

Requires `wkhtmltopdf` and Python `markdown`.

## Verify after deploy

```bash
curl -s https://HOST/bridge/production/status | jq '.onlineBank, .ledger'
curl -s https://HOST/bridge/bank/status | jq .
curl -s https://HOST/bridge/ledger/status | jq '.defaultBridgeChain'
```
