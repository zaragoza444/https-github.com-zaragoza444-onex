package ai

import (
	"strings"
)

const walletSystemHint = `You are OneX AI — assistant for OneX Blockchain (Ed25519 PoW chain) and OneX Wallet (OKX-style DeFi UI).
Features: multi-chain portfolio, send/receive, deposit, OneX Swap AMM (x·y=k), liquidity pools, cross-chain bridge, stake, loans, NFTs, tasks, create token.
Real Ledger: unified bank (M0/M1/NSB fund classes, IBAN accounts), on-chain EVM balances, and imports — valued at live USD. Settle to real crypto wallets or external bank IBAN/SEPA/SWIFT.
Saved destinations: wallet addresses and bank IBAN accounts for settlement and bridge.
Native coin: ONEX (8 decimals). Addresses are 64-char hex. MetaMask cannot sign; use OneX Wallet or the Chrome extension.
Wallet UI tabs: Wallet (home), Trade, Earn, Discover, Web3, Ledger (real assets), AI (this chat).
Node API: :8545 explorer, /rpc JSON-RPC, /health. Bridge: :9338/wallet/.`

func localReply(user string, ctx string) ChatResponse {
	q := strings.ToLower(strings.TrimSpace(user))
	reply := ""
	var act *Action
	suggestions := []string{"Show my real assets", "How do I swap?", "Settle to IBAN", "Bridge to BSC"}

	switch {
	case containsAny(q, "hello", "hi", "hey"):
		reply = "Hello! I'm OneX AI. I can help with your real ledger (bank IBAN, M0/M1/NSB), wallet, swaps, staking, and bridge. What would you like to do?"
		if ctx != "" {
			reply += "\n\n" + summarizeContext(ctx)
		}
	case containsAny(q, "hybx", "hybrix", "multi-ledger", "mirror bank"):
		reply = "Open Online Bank → HYBX tab. Mirror NSB balances, use HYBX exchange middleware (banks · chains · platform), issue HYBX virtual cards, and settle via federation."
	case containsAny(q, "bridge7", "local-ledger", "ledger pro", "crypto-ledger", "ledger-pro"):
		reply = "Bridge7 syncs local-ledger-2026, ledger-pro, and crypto-ledger into the real ledger. Wallet → Real Ledger → Bridge7 → Sync. API: POST /bridge/bridge7/sync"
		reply = "HYBX exchange middleware: GET /bridge/bank/hybx/exchange/routes, POST /bridge/bank/hybx/exchange for NSB↔HYBX, Fineract, blockchains, and token platform. Wallet: Online Bank → HYBX → exchange middleware panel."
		if ctx != "" {
			reply += "\n\n" + summarizeContext(ctx)
		}
		act = &Action{Type: "navigate", Tab: "onlinebank"}
	case containsAny(q, "online bank", "nsb online", "send iban", "internal transfer", "bank statement", "wire instructions", "payee"):
		reply = "Open Online Bank — Overview, Send, Deposit, Activity, Receive (IBAN/SWIFT), Payees, Cards, and Statements. Export CSV from Activity tab."
		if ctx != "" {
			reply += "\n\n" + summarizeContext(ctx)
		}
		act = &Action{Type: "navigate", Tab: "onlinebank"}
	case containsAny(q, "virtual card", "debit card", "apple pay", "google pay", "pay with card"):
		reply = "Open Online Bank → Virtual cards. NSB auto-issues Visa/Mastercard debit cards; HYBX tab → Sync → Issue HYBX virtual cards for multi-ledger mirror accounts. Production mode enables Apple Pay, Google Pay, and 3D Secure."
		if ctx != "" {
			reply += "\n\n" + summarizeContext(ctx)
		}
		act = &Action{Type: "navigate", Tab: "onlinebank"}
	case containsAny(q, "real", "ledger", "bank", "iban", "m0", "m1", "nsb", "sovereign", "fiat", "settle", "convert"):
		reply = "Your Real Ledger tab shows bank IBAN balances (M0 base money, M1 demand deposits, NSB sovereign), on-chain crypto, and live USD totals. I read that data when you chat here."
		if ctx != "" {
			reply += "\n\n" + summarizeContext(ctx)
		} else {
			reply += "\n\nSet ONEX_BANK_LEDGER_FILE or connect EVM address in Settings for full real asset context."
		}
		act = &Action{Type: "navigate", Tab: "ledger"}
	case containsAny(q, "balance", "portfolio", "assets", "how much", "worth", "holdings"):
		reply = "I combine your real ledger (bank + chain) with portfolio tokens. Here's what I see right now:"
		if ctx != "" {
			reply += "\n\n" + summarizeContext(ctx)
		} else {
			reply += "\n\nConnect the bridge and open Ledger tab to sync real assets."
		}
		act = &Action{Type: "navigate", Tab: "ledger"}
	case containsAny(q, "send", "transfer"):
		reply = "Tap Send on the home screen (or the send sheet). Pick chain + token, enter a 64-char recipient address and amount. On-chain ONEX sends need a small fee (default 0.001 ONEX)."
		act = &Action{Type: "sheet", Sheet: "send"}
	case containsAny(q, "receive", "address"):
		reply = "Tap Receive to copy your OneX address. Share it to receive ONEX on the OneX chain."
		act = &Action{Type: "sheet", Sheet: "receive"}
	case containsAny(q, "deposit"):
		reply = "Deposit credits portfolio tokens from other chains. Open Deposit, choose chain, copy the deposit address, then record the amount (and tx hash if you have one)."
		act = &Action{Type: "sheet", Sheet: "deposit"}
	case containsAny(q, "swap", "trade", "exchange"):
		reply = "OneX Swap is a Uniswap-style AMM (constant product x·y=k, ~0.3% fee). Go to Trade → Swap, pick tokens and amount, review price impact, then confirm."
		act = &Action{Type: "navigate", Tab: "trade"}
	case containsAny(q, "pool", "liquidity", "lp"):
		reply = "Trade → Pool: add liquidity to AMM pairs and earn LP shares. Remove liquidity by burning shares."
		act = &Action{Type: "navigate", Tab: "trade"}
	case containsAny(q, "bridge", "cross-chain", "cross chain"):
		reply = "Trade → Bridge routes swaps across chains via ONEX hub pools. Select from/to chain and token, quote, then bridge."
		act = &Action{Type: "navigate", Tab: "trade"}
	case containsAny(q, "stake", "staking", "apy", "earn"):
		reply = "Earn tab → Stake: lock tokens for APY and receipt tokens (e.g. sONEX). Check lock period before unstaking."
		act = &Action{Type: "navigate", Tab: "earn"}
	case containsAny(q, "loan", "borrow", "lend"):
		reply = "Earn tab → Loans: post collateral and borrow or lend against configured token pairs. Repay to close active loans."
		act = &Action{Type: "navigate", Tab: "earn"}
	case containsAny(q, "nft", "mint"):
		reply = "Discover → NFT: view your collection or mint with name, description, and image URL."
		act = &Action{Type: "navigate", Tab: "discover"}
	case containsAny(q, "task", "reward", "claim"):
		reply = "Discover → Rewards: complete open tasks to claim ONEX or wONEX bonuses."
		act = &Action{Type: "navigate", Tab: "discover"}
	case containsAny(q, "create token", "mint token", "launch", "token platform", "deploy token"):
		reply = "Discover → Token Platform: deploy on any of 13+ chains, wrap cross-chain, or use CLI: onex token-create / token-wrap. See docs/TOKEN-PLATFORM.md."
		act = &Action{Type: "navigate", Tab: "discover"}
	case containsAny(q, "network", "chain", "chains"):
		reply = "Discover → Networks lists 13+ supported chains (OneX, Ethereum, BSC, Polygon, and more)."
		act = &Action{Type: "navigate", Tab: "discover"}
	case containsAny(q, "web3", "dapp", "explorer"):
		reply = "Web3 tab opens dApps and the block explorer. OneX provider uses Ed25519; install the Chrome extension for dApp signing."
		act = &Action{Type: "navigate", Tab: "web3"}
	case containsAny(q, "metamask", "ethereum", "evm"):
		reply = "OneX uses Ed25519, not Ethereum keys. MetaMask can display network info but cannot sign OneX transactions. Use OneX Wallet or the extension."
	case containsAny(q, "node", "blockchain", "pow", "mining", "block"):
		reply = "OneX is a proof-of-work node (onexd) with REST + JSON-RPC. Explorer at /explorer/, health at /health. JSON-RPC includes onex_* and eth_* compat methods."
	case containsAny(q, "rpc", "api", "json"):
		reply = "Node JSON-RPC: POST /rpc (e.g. onex_getBalance, onex_sendTransaction, eth_chainId). Bridge RPC: :9338/rpc for wallet methods."
	case containsAny(q, "wallet", "create", "import"):
		reply = "Create a wallet via Settings (⚙) or the + button. Wallet file: ~/.onex/wallets/default.json. Ed25519 keys — keep backups offline."
		act = &Action{Type: "sheet", Sheet: "settings"}
	case containsAny(q, "saved", "receiver", "destination", "payout"):
		reply = "Saved wallet addresses and bank IBAN accounts live in the Ledger tab under Saved destinations. Use them for settlement, bridge, and convert."
		if ctx != "" {
			reply += "\n\n" + summarizeContext(ctx)
		}
		act = &Action{Type: "navigate", Tab: "ledger"}
	case containsAny(q, "fee", "gas"):
		reply = "OneX uses explicit min tx fees (not EVM gas). Default send fee is 0.001 ONEX. AMM swaps charge pool fee (~0.3%)."
	case containsAny(q, "cloud", "api key", "openai", "model"):
		reply = "Set ONEX_AI_API_KEY (and optional ONEX_AI_BASE_URL, ONEX_AI_MODEL) on the bridge/node to enable cloud AI. Without a key, I run in local assistant mode."
	default:
		reply = walletSystemHint + "\n\nAsk about: balance, send, swap, stake, bridge, NFTs, loans, or how to run the node. "
		if ctx != "" {
			reply += "Here's your current context:\n" + summarizeContext(ctx)
		} else {
			reply += "Connect a wallet and refresh for personalized answers."
		}
	}

	return ChatResponse{
		Reply:       reply,
		Mode:        "local",
		Action:      act,
		Suggestions: suggestions,
	}
}

func containsAny(s string, words ...string) bool {
	for _, w := range words {
		if strings.Contains(s, w) {
			return true
		}
	}
	return false
}
