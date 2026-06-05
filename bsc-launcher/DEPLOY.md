# BSC Token Launcher — Production Deployment

## Prerequisites

- Linux VPS or Docker host (2 GB RAM minimum)
- Domain + TLS certificate (Let's Encrypt)
- BSC RPC URL (dedicated provider recommended)
- Etherscan API V2 key ([etherscan.io/apidashboard](https://etherscan.io/apidashboard))
- Strong `BSC_LAUNCHER_API_KEY` (32+ random chars)

## Option 1 — Docker (recommended)

```bash
cp bsc-launcher/.env.production.example bsc-launcher/.env
# Edit bsc-launcher/.env — set API key, CORS, BSCSCAN_API_KEY

docker compose -f docker-compose.bsc-launcher.yml up -d --build
```

With HTTPS reverse proxy:

```bash
# Place certs in deploy/certs/fullchain.pem and privkey.pem
docker compose -f docker-compose.bsc-launcher.yml --profile proxy up -d
```

Verify:

```bash
curl -s http://localhost:9340/health
curl -s http://localhost:9340/ready
```

## Option 2 — systemd (bare metal)

```bash
go build -o /opt/onex/bin/bsc-launcher ./bsc-launcher/server
cp -r bsc-launcher/{web,abi} /opt/onex/bsc-launcher/
cp bsc-launcher/.env.production.example /opt/onex/bsc-launcher/.env
# edit /opt/onex/bsc-launcher/.env

useradd -r -s /bin/false launcher
chown -R launcher:launcher /opt/onex/bsc-launcher/data

cp deploy/bsc-launcher.service /etc/systemd/system/
systemctl daemon-reload
systemctl enable --now bsc-launcher
```

## Option 3 — Add to full OneX stack

```bash
docker compose -f docker-compose.prod.yml --profile bsc-launcher up -d
```

## Production checklist

| Item | Setting |
|------|---------|
| Environment | `BSC_LAUNCHER_ENV=production` |
| API auth | `BSC_LAUNCHER_API_KEY` set |
| CORS | `BSC_LAUNCHER_CORS_ORIGINS=https://yourdomain.com` |
| BSCScan | `BSCSCAN_API_KEY` set |
| TLS | nginx with valid certs |
| Data backup | Volume `/data` or `bsc-launcher-data` |
| Deployer key | Funded hot wallet, minimal BNB balance |
| Firewall | Expose 443 only; keep 9340 internal |

## Users & API key

1. Share the site URL with users
2. Give trusted users the `BSC_LAUNCHER_API_KEY`
3. Users click **Settings** in the UI and paste the key (stored in browser localStorage)
4. MetaMask deploy and backend deploy both require the key when configured

## Health monitoring

| Endpoint | Use |
|----------|-----|
| `GET /health` | Liveness — always 200 if process up |
| `GET /ready` | Readiness — checks RPC, store, contract artifacts |

## Backup

```bash
# Docker
docker run --rm -v bsc-launcher-data:/data -v $(pwd):/backup alpine \
  tar czf /backup/tokens-backup.tar.gz -C /data .

# systemd
tar czf tokens-backup.tar.gz -C /opt/onex/bsc-launcher/data .
```

## Upgrade

```bash
docker compose -f docker-compose.bsc-launcher.yml up -d --build
# or
systemctl restart bsc-launcher
```

Token registry (`tokens.json`) persists across restarts via the data volume.
