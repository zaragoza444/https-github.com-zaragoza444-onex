# Production deployment

Run Shiva blockchain + OKX-style wallet bridge on a server (Docker or systemd).

## Docker (recommended)

```bash
git clone <your-repo-url> shiva-blockchain
cd shiva-blockchain
cp .env.example .env
# Edit .env: SHIVA_API_KEY, SHIVA_CORS_ORIGINS=https://your-domain.com

docker compose -f docker-compose.prod.yml up -d --build
```

| Service | Port | URL |
|---------|------|-----|
| Node API + explorer | 8545 | `http://HOST:8545/explorer/` |
| Wallet (bridge) | 9338 | `http://HOST:9338/wallet/` |

With TLS reverse proxy:

```bash
mkdir -p deploy/certs
# Place fullchain.pem + privkey.pem in deploy/certs/
docker compose -f docker-compose.prod.yml --profile proxy up -d
```

- Wallet: `https://your-domain/wallet/`
- Explorer: `https://your-domain/explorer/`

## systemd (bare metal)

```bash
sudo mkdir -p /opt/shiva/bin /var/lib/shiva /var/lib/shiva-bridge/wallets
sudo cp bin/shivad bin/shiva bin/shiva-bridge /opt/shiva/bin/
sudo cp -r configs /opt/shiva/
sudo cp deploy/shivad.service deploy/shiva-bridge.service /etc/systemd/system/
sudo cp .env.example /etc/shiva/shiva.env
# Edit /etc/shiva/shiva.env

sudo systemctl daemon-reload
sudo systemctl enable --now shivad shiva-bridge
```

## Security checklist

- Set `SHIVA_API_KEY` on the node for `POST /api/v1/tx` and `/rpc`
- Restrict `SHIVA_CORS_ORIGINS` to your wallet domain
- Use TLS (`--profile proxy` or nginx in front)
- Never commit `.env`, wallet JSON, or private keys
- Testnet faucet: set `SHIVA_FAUCET_PRIVATE_KEY` only on testnet hosts

## Shiva Wallet mobile apps

React Native (Expo) WebView app in [`mobile/`](mobile/).

1. Deploy backend with HTTPS (`https://YOUR_DOMAIN/wallet/`).
2. Set `EXPO_PUBLIC_WALLET_URL` in `mobile/.env` (see `mobile/.env.example`).
3. Build and publish: **[mobile/PUBLISH.md](mobile/PUBLISH.md)**.

```bash
cd mobile && npm install && eas build --platform all --profile production
```

## Publish to GitHub + Gitea

```powershell
# Windows
.\scripts\publish-remotes.ps1 -GitHub "git@github.com:USER/shiva-blockchain.git" -Gitea "git@git.example.com:USER/shiva-blockchain.git"
```

```bash
# Linux
./scripts/publish-remotes.sh git@github.com:USER/shiva-blockchain.git git@git.example.com:USER/shiva-blockchain.git
```

Create empty repos on GitHub and Gitea first, then run the script.
