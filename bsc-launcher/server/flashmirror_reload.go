package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Minimal ERC-20 / FlashCoin read ABI (name, symbol, decimals, totalSupply).
const flashCoinReadABI = `[
  {"inputs":[],"name":"name","outputs":[{"type":"string"}],"stateMutability":"view","type":"function"},
  {"inputs":[],"name":"symbol","outputs":[{"type":"string"}],"stateMutability":"view","type":"function"},
  {"inputs":[],"name":"decimals","outputs":[{"type":"uint8"}],"stateMutability":"view","type":"function"},
  {"inputs":[],"name":"totalSupply","outputs":[{"type":"uint256"}],"stateMutability":"view","type":"function"},
  {"inputs":[],"name":"owner","outputs":[{"type":"address"}],"stateMutability":"view","type":"function"},
  {"inputs":[{"type":"address"}],"name":"balanceOf","outputs":[{"type":"uint256"}],"stateMutability":"view","type":"function"}
]`

type mirrorConfig struct {
	Name               string `json:"name"`
	Symbol             string `json:"symbol"`
	Decimals           int    `json:"decimals"`
	Supply             string `json:"supply"`
	OriginChain        string `json:"originChain"`
	WrapAmountPerChain string `json:"wrapAmountPerChain"`
	MirrorMode         string `json:"mirrorMode"`
}

func (s *Server) loadMirrorConfig() mirrorConfig {
	cfg := mirrorConfig{Name: "Flash Coin", Symbol: "FLASH", Decimals: 8, Supply: "1000000000", WrapAmountPerChain: "1000000000"}
	_, _, cfgPath := s.flashMirrorPaths()
	raw, err := os.ReadFile(cfgPath)
	if err != nil {
		return cfg
	}
	_ = json.Unmarshal(raw, &cfg)
	if cfg.Decimals == 0 {
		cfg.Decimals = 8
	}
	if cfg.WrapAmountPerChain == "" {
		cfg.WrapAmountPerChain = "1000000000"
	}
	return cfg
}

func formatSupplyHuman(raw *big.Int, decimals int) string {
	if raw == nil {
		return ""
	}
	if decimals <= 0 {
		decimals = 18
	}
	denom := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	rat := new(big.Rat).SetInt(raw)
	rat.Quo(rat, new(big.Rat).SetInt(denom))
	f, _ := rat.Float64()
	if f >= 1 {
		return strconv.FormatFloat(f, 'f', -1, 64)
	}
	return strconv.FormatFloat(f, 'f', int(decimals), 64)
}

func parseHumanAmount(s string) (float64, bool) {
	s = strings.ReplaceAll(strings.TrimSpace(s), ",", "")
	if s == "" {
		return 0, false
	}
	f, err := strconv.ParseFloat(s, 64)
	return f, err == nil && f > 0
}

func applyMirrorMarketFields(d *flashMirrorDeployment, quote *PriceQuote) {
	if d == nil {
		return
	}
	supplyHuman := d.TotalSupplyHuman
	if supplyHuman == "" {
		supplyHuman = d.SupplyHuman
	}
	if supplyHuman == "" {
		supplyHuman = d.WrapAmountHuman
	}
	supply, hasSupply := parseHumanAmount(supplyHuman)

	if quote != nil {
		d.PriceUSD = quote.PriceUSD
		d.LiquidityUSD = quote.LiquidityUSD
		d.HasLiquidity = quote.HasLiquidity
		d.DexID = quote.DexID
		d.PairAddress = quote.PairAddress
		d.OnChainMarketCapUSD = quote.MarketCap
		if d.OnChainMarketCapUSD == 0 && quote.PriceUSD > 0 && hasSupply {
			d.OnChainMarketCapUSD = supply * quote.PriceUSD
		}
		d.MarketCapUSD = d.OnChainMarketCapUSD
	}

	// $1 listing target — implied cap from configured supply (e.g. 1B → $1B mcap).
	if hasSupply {
		d.ImpliedPriceUSD = 1
		d.ImpliedMarketCapUSD = supply * d.ImpliedPriceUSD
		if d.MarketCapUSD == 0 {
			d.MarketCapUSD = d.ImpliedMarketCapUSD
		}
		if d.PriceUSD == 0 {
			d.ImpliedPriceUSD = 1
		}
	}
}

func chainFromMeta(id string, meta chainMeta) Chain {
	slug := normalizeRegistryChain(id)
	if slug == "ethereum" {
		slug = "eth"
	}
	c, err := chainBySlug(slug)
	if err != nil {
		c = Chain{Slug: id, Name: meta.Name, DexChainID: id}
	}
	if meta.Name != "" {
		c.Name = meta.Name
	}
	if meta.NetworkID > 0 {
		c.ChainID = meta.NetworkID
	}
	if meta.RPC != "" {
		c.RPCURL = meta.RPC
	}
	if meta.Explorer != "" {
		c.Explorer = meta.Explorer
	}
	if c.DexChainID == "" {
		c.DexChainID = id
	}
	return c
}

func humanSupplyToRaw(human string, decimals int) string {
	human = strings.TrimSpace(human)
	if human == "" || decimals < 0 {
		return ""
	}
	parts := strings.SplitN(human, ".", 2)
	whole := parts[0]
	frac := ""
	if len(parts) == 2 {
		frac = parts[1]
	}
	if len(frac) > decimals {
		frac = frac[:decimals]
	}
	for len(frac) < decimals {
		frac += "0"
	}
	whole = strings.TrimLeft(whole, "0")
	if whole == "" {
		whole = "0"
	}
	if frac == "" {
		return whole
	}
	raw := whole + frac
	raw = strings.TrimLeft(raw, "0")
	if raw == "" {
		return "0"
	}
	return raw
}

func (s *Server) readFlashCoinToken(ctx context.Context, rpcURL, address string) (*OnChainTokenInfo, string, string, bool, error) {
	ok, err := s.isContractOn(ctx, rpcURL, address)
	if err != nil {
		return nil, "", "", false, err
	}
	if !ok {
		return nil, "", "", false, nil
	}

	client, err := s.rpcClient(ctx, rpcURL)
	if err != nil {
		return nil, "", "", false, err
	}
	defer client.Close()

	parsed, err := abi.JSON(strings.NewReader(flashCoinReadABI))
	if err != nil {
		return nil, "", "", false, err
	}
	addr := common.HexToAddress(address)
	info := &OnChainTokenInfo{ContractAddress: addr.Hex(), IsContract: true}

	if name, err := callString(ctx, client, parsed, addr, "name"); err == nil {
		info.TokenName = name
	}
	if symbol, err := callString(ctx, client, parsed, addr, "symbol"); err == nil {
		info.Symbol = symbol
	}
	if decimals, err := callUint8(ctx, client, parsed, addr, "decimals"); err == nil {
		info.Divisor = fmt.Sprintf("%d", decimals)
	}
	if supply, err := callBigInt(ctx, client, parsed, addr, "totalSupply"); err == nil {
		info.TotalSupply = supply.String()
	}

	ownerAddr := ""
	ownerBal := ""
	if owner, err := callAddress(ctx, client, parsed, addr, "owner"); err == nil && owner != (common.Address{}) {
		ownerAddr = owner.Hex()
		if bal, err := callBalanceOf(ctx, client, parsed, addr, owner); err == nil {
			ownerBal = bal.String()
		}
	}

	// FlashCoin has unrestricted transfer(); contract present + supply > 0 => transferable.
	transferable := info.TotalSupply != "" && info.TotalSupply != "0"
	return info, ownerAddr, ownerBal, transferable, nil
}

func callAddress(ctx context.Context, client *ethclient.Client, parsed abi.ABI, addr common.Address, method string) (common.Address, error) {
	data, err := parsed.Pack(method)
	if err != nil {
		return common.Address{}, err
	}
	out, err := client.CallContract(ctx, ethereum.CallMsg{To: &addr, Data: data}, nil)
	if err != nil {
		return common.Address{}, err
	}
	vals, err := parsed.Unpack(method, out)
	if err != nil || len(vals) == 0 {
		return common.Address{}, err
	}
	if a, ok := vals[0].(common.Address); ok {
		return a, nil
	}
	return common.Address{}, fmt.Errorf("unexpected %s return", method)
}

func callBalanceOf(ctx context.Context, client *ethclient.Client, parsed abi.ABI, token, holder common.Address) (*big.Int, error) {
	data, err := parsed.Pack("balanceOf", holder)
	if err != nil {
		return nil, err
	}
	out, err := client.CallContract(ctx, ethereum.CallMsg{To: &token, Data: data}, nil)
	if err != nil {
		return nil, err
	}
	vals, err := parsed.Unpack("balanceOf", out)
	if err != nil || len(vals) == 0 {
		return nil, err
	}
	if v, ok := vals[0].(*big.Int); ok {
		return v, nil
	}
	return nil, fmt.Errorf("unexpected balanceOf return")
}

func (s *Server) reloadFlashMirrorPayload(ctx context.Context, book *flashMirrorBook, chains map[string]chainMeta, deep bool) {
	if book == nil {
		return
	}
	cfg := s.loadMirrorConfig()
	book.OriginSupplyHuman = cfg.Supply
	book.Decimals = cfg.Decimals
	if book.WrapAmountPerChain == "" {
		book.WrapAmountPerChain = cfg.WrapAmountPerChain
	}
	if book.MirrorMode == "" {
		book.MirrorMode = cfg.MirrorMode
	}

	addr := book.CanonicalAddress
	for i := range book.Deployments {
		d := &book.Deployments[i]
		if addr == "" {
			addr = d.ContractAddress
		}
		if addr == "" {
			addr = d.PredictedAddress
		}
		chainAddr := addr
		if d.ContractAddress != "" {
			chainAddr = d.ContractAddress
		} else if d.PredictedAddress != "" {
			chainAddr = d.PredictedAddress
		}

		d.WrapAmountHuman = book.WrapAmountPerChain
		if d.WrapAmountHuman == "" {
			d.WrapAmountHuman = cfg.WrapAmountPerChain
		}
		d.SupplyHuman = d.WrapAmountHuman
		d.Decimals = cfg.Decimals
		if d.TotalSupply == "" && d.WrapAmountHuman != "" {
			d.TotalSupply = humanSupplyToRaw(d.WrapAmountHuman, d.Decimals)
			d.TotalSupplyHuman = d.WrapAmountHuman
		}
		d.Transferable = true
		d.TokenStandard = "real-token"
		if d.Symbol == "" {
			d.Symbol = book.Symbol
		}
		if d.TokenName == "" {
			d.TokenName = book.Name
		}

		if chainAddr != "" {
			d.ContractAddress = chainAddr
			d.PredictedAddress = chainAddr
			if d.Explorer != "" {
				d.ExplorerTokenURL = explorerTokenURL(d.Explorer, chainAddr)
			}
		}

		meta, hasMeta := chains[d.ChainID]
		if !hasMeta {
			continue
		}
		chain := chainFromMeta(d.ChainID, meta)
		dexChain := chain.DexChainID
		if dexChain == "" {
			dexChain = d.ChainID
		}

		if chainAddr != "" && meta.RPC != "" && deep {
			chainCtx, chainCancel := context.WithTimeout(ctx, 12*time.Second)
			live, _ := s.isContractOn(chainCtx, meta.RPC, chainAddr)
			if live {
				d.VerifiedOnChain = true
				d.Status = "live"
			} else if d.Status == "" || d.Status == "predicted" {
				d.Status = "predicted"
				d.VerifiedOnChain = false
			}
			if live {
				onchain, ownerAddr, ownerBal, xfer, err := s.readFlashCoinToken(chainCtx, meta.RPC, chainAddr)
				if err == nil && onchain != nil && onchain.IsContract {
					if onchain.TokenName != "" {
						d.TokenName = onchain.TokenName
					}
					if onchain.Symbol != "" {
						d.Symbol = onchain.Symbol
					}
					if onchain.Divisor != "" {
						if dec, e := strconv.Atoi(onchain.Divisor); e == nil {
							d.Decimals = dec
						}
					}
					if onchain.TotalSupply != "" {
						d.TotalSupply = onchain.TotalSupply
						if raw, ok := new(big.Int).SetString(onchain.TotalSupply, 10); ok {
							d.TotalSupplyHuman = formatSupplyHuman(raw, d.Decimals)
							d.SupplyHuman = d.TotalSupplyHuman
						}
					}
					d.OwnerAddress = ownerAddr
					d.OwnerBalance = ownerBal
					if ownerBal != "" {
						if raw, ok := new(big.Int).SetString(ownerBal, 10); ok {
							d.OwnerBalanceHuman = formatSupplyHuman(raw, d.Decimals)
						}
					}
					d.Transferable = xfer
				}
			}
			chainCancel()
		}

		if deep && chainAddr != "" && s.price != nil {
			var quote *PriceQuote
			if q, err := s.price.Quote(dexChain, chainAddr); err == nil {
				quote = q
			}
			applyMirrorMarketFields(d, quote)
		} else if deep {
			applyMirrorMarketFields(d, nil)
		}

		if deep && chain.ChainID > 0 && chainAddr != "" && s.bscscan != nil {
			if scan, err := s.bscscan.TokenInfoForChain(chain.ChainID, chainAddr); err == nil && scan != nil {
				if scan.TokenName != "" && d.TokenName == "" {
					d.TokenName = scan.TokenName
				}
				if scan.Symbol != "" && d.Symbol == book.Symbol {
					d.Symbol = scan.Symbol
				}
				if scan.TotalSupply != "" && d.TotalSupply == "" {
					d.TotalSupply = scan.TotalSupply
				}
				if n, e := strconv.Atoi(strings.TrimSpace(scan.Holders)); e == nil {
					d.Holders = n
				}
			}
		}
	}
}
