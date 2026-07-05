# PouchPay Plus (mobile)

Expo React Native app that loads the PouchPay Plus wallet UI in a WebView with PouchPay and Alltra Plus integration endpoints.

## Quick start

```bash
# Terminal 1 — node + bridge
cd ..
run-onex-wallet.bat

# Terminal 2 — mobile
cd mobile
cp .env.example .env
npm install
npm start
```

Scan the QR code with Expo Go, or press `a` / `i` for emulator.

## Configuration

| Variable | Description |
|----------|-------------|
| `EXPO_PUBLIC_WALLET_URL` | Wallet URL (default `http://127.0.0.1:9338/wallet/`) |
| `EXPO_PUBLIC_APP_NAME` | App display name (default `PouchPay Plus`) |
| `EXPO_PUBLIC_POUCHPAY_URL` | PouchPay integration base URL |
| `EXPO_PUBLIC_ALLTRA_PLUS_RPC_URL` | Alltra Plus JSON-RPC URL |
| `EXPO_PUBLIC_ALLTRA_PLUS_EXPLORER_URL` | Alltra Plus explorer URL |

Override in-app via **Settings** (gear icon).

## Deep links

- `pouchpayplus://swap`
- `pouchpayplus://ai`
- `pouchpayplus://earn`
- `pouchpayplus://discover`
- `pouchpayplus://web3`

## Store builds

See [PUBLISH.md](PUBLISH.md).
