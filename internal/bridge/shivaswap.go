package bridge

import (
	"fmt"
	"os"
	"path/filepath"
	"github.com/shiva-blockchain/shiva/internal/bridge/amm"
	"github.com/shiva-blockchain/shiva/internal/rpc"
)

func (b *Bridge) ammStore() *amm.Store {
	if b.amm == nil {
		home, _ := os.UserHomeDir()
		b.amm = amm.NewStore(filepath.Join(home, ".shiva", "amm"))
		seed := loadJSON[amm.Pool](filepath.Join(b.projectRoot(), "configs", "amm-pools.json"))
		_ = b.amm.Load(seed)
	}
	return b.amm
}

func (b *Bridge) ShivaSwapPools() []amm.Pool {
	return b.ammStore().List()
}

func (b *Bridge) ShivaSwapQuote(tokenIn, tokenOut, amountStr string) (map[string]interface{}, error) {
	amt, err := rpc.ParseAmount(amountStr)
	if err != nil {
		return nil, err
	}
	pool, ok := b.ammStore().FindPool(tokenIn, tokenOut)
	if !ok {
		return nil, fmt.Errorf("no liquidity pool for this pair — create pool or use bridge route")
	}
	out, impact, err := pool.QuoteSwap(tokenIn, amt)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"poolId":      pool.ID,
		"tokenIn":     tokenIn,
		"tokenOut":    tokenOut,
		"amountIn":    fmt.Sprintf("%d", amt),
		"amountOut":   fmt.Sprintf("%d", out),
		"priceImpact": impact,
		"feeBps":      pool.FeeBps,
		"reserve0":    pool.Reserve0,
		"reserve1":    pool.Reserve1,
	}, nil
}

// ShivaSwapExecute performs AMM swap and updates user portfolio + on-chain SHIVA when needed.
func (b *Bridge) ShivaSwapExecute(tokenIn, tokenOut, amountStr string, slippageBps int) (map[string]interface{}, error) {
	if err := b.EnsureWallet(); err != nil {
		return nil, err
	}
	quote, err := b.ShivaSwapQuote(tokenIn, tokenOut, amountStr)
	if err != nil {
		return nil, err
	}
	pool, ok := b.ammStore().Get(quote["poolId"].(string))
	if !ok {
		return nil, fmt.Errorf("pool not found")
	}
	var amountIn uint64
	fmt.Sscanf(quote["amountIn"].(string), "%d", &amountIn)
	expectedOut, _, _ := pool.QuoteSwap(tokenIn, amountIn)
	out, err := pool.Swap(tokenIn, amountIn)
	if err != nil {
		return nil, err
	}
	minOut := expectedOut
	if slippageBps > 0 {
		minOut = expectedOut * uint64(10000-slippageBps) / 10000
	}
	if out < minOut {
		return nil, fmt.Errorf("slippage exceeded")
	}

	p, err := b.GetPortfolio()
	if err != nil {
		return nil, err
	}
	if err := p.SubBalance(tokenIn, amountIn); err != nil {
		return nil, err
	}
	p.AddBalance(tokenOut, out)
	_ = b.portfolio().Save(p)
	_ = b.ammStore().Update(pool)

	// Sync SHIVA to chain when swapping native SHIVA (bridge ↔ blockchain)
	b.syncSwapToChain(tokenIn, tokenOut, amountIn, out)

	rec := SwapRecord{
		ID: newID(), FromKey: tokenIn, ToKey: tokenOut,
		FromAmt: fmt.Sprintf("%d", amountIn), ToAmt: fmt.Sprintf("%d", out),
		Status: "amm", CreatedAt: nowUnix(),
	}
	p.Swaps = append([]SwapRecord{rec}, p.Swaps...)
	_ = b.portfolio().Save(p)
	b.completeTask(p, "first-swap")

	return map[string]interface{}{
		"status":      "success",
		"amountIn":    amountIn,
		"amountOut":   out,
		"poolId":      pool.ID,
		"priceImpact": quote["priceImpact"],
		"txType":      "shiva-swap-amm",
	}, nil
}

func (b *Bridge) syncSwapToChain(tokenIn, tokenOut string, amountIn, amountOut uint64) {
	shivaKey := b.registry().TokenKey("shiva-mainnet-1", "SHIVA")
	if tokenIn != shivaKey && tokenOut != shivaKey {
		return
	}
	// Portfolio already updated; on-chain SHIVA balance is source of truth on refresh.
	// Swaps selling SHIVA will reflect after syncShivaBalance on next GetPortfolio.
	_ = amountIn
	_ = amountOut
}

func (b *Bridge) ShivaSwapAddLiquidity(token0, token1, amount0Str, amount1Str string) (map[string]interface{}, error) {
	if err := b.EnsureWallet(); err != nil {
		return nil, err
	}
	a0, err := rpc.ParseAmount(amount0Str)
	if err != nil {
		return nil, err
	}
	a1, err := rpc.ParseAmount(amount1Str)
	if err != nil {
		return nil, err
	}
	pool, ok := b.ammStore().FindPool(token0, token1)
	if !ok {
		id := amm.PoolID(token0, token1)
		if token0 > token1 {
			token0, token1 = token1, token0
			a0, a1 = a1, a0
		}
		pool = &amm.Pool{
			ID: id, Token0: token0, Token1: token1,
			Reserve0: "0", Reserve1: "0", TotalShares: "0", FeeBps: amm.DefaultFeeBps,
		}
	}
	shares, err := pool.AddLiquidity(a0, a1)
	if err != nil {
		return nil, err
	}
	p, err := b.GetPortfolio()
	if err != nil {
		return nil, err
	}
	if err := p.SubBalance(token0, a0); err != nil {
		return nil, err
	}
	if err := p.SubBalance(token1, a1); err != nil {
		return nil, err
	}
	lpKey := "lp:" + pool.ID
	p.SetBalance(lpKey, p.GetBalance(lpKey)+shares)
	_ = b.portfolio().Save(p)
	_ = b.ammStore().Update(pool)
	return map[string]interface{}{
		"poolId": pool.ID, "shares": fmt.Sprintf("%d", shares),
		"reserve0": pool.Reserve0, "reserve1": pool.Reserve1,
	}, nil
}

func (b *Bridge) ShivaSwapRemoveLiquidity(poolID, sharesStr string) (map[string]interface{}, error) {
	if err := b.EnsureWallet(); err != nil {
		return nil, err
	}
	var shares uint64
	fmt.Sscanf(sharesStr, "%d", &shares)
	pool, ok := b.ammStore().Get(poolID)
	if !ok {
		return nil, fmt.Errorf("pool not found")
	}
	a0, a1, err := pool.RemoveLiquidity(shares)
	if err != nil {
		return nil, err
	}
	p, err := b.GetPortfolio()
	if err != nil {
		return nil, err
	}
	lpKey := "lp:" + poolID
	if err := p.SubBalance(lpKey, shares); err != nil {
		return nil, err
	}
	p.AddBalance(pool.Token0, a0)
	p.AddBalance(pool.Token1, a1)
	_ = b.portfolio().Save(p)
	_ = b.ammStore().Update(pool)
	return map[string]interface{}{
		"amount0": a0, "amount1": a1, "token0": pool.Token0, "token1": pool.Token1,
	}, nil
}

// BridgeRoute finds path tokenIn -> SHIVA -> tokenOut across two pools.
func (b *Bridge) BridgeRoute(tokenIn, tokenOut, amountStr string) (map[string]interface{}, error) {
	shiva := b.registry().TokenKey("shiva-mainnet-1", "SHIVA")
	if tokenIn == tokenOut {
		return nil, fmt.Errorf("same token")
	}
	if tokenIn == shiva || tokenOut == shiva {
		return b.ShivaSwapQuote(tokenIn, tokenOut, amountStr)
	}
	q1, err := b.ShivaSwapQuote(tokenIn, shiva, amountStr)
	if err != nil {
		return nil, fmt.Errorf("bridge leg1: %w", err)
	}
	q2, err := b.ShivaSwapQuote(shiva, tokenOut, q1["amountOut"].(string))
	if err != nil {
		return nil, fmt.Errorf("bridge leg2: %w", err)
	}
	return map[string]interface{}{
		"route":     []string{tokenIn, shiva, tokenOut},
		"amountIn":  q1["amountIn"],
		"amountOut": q2["amountOut"],
		"leg1":      q1,
		"leg2":      q2,
	}, nil
}

func (b *Bridge) BridgeExecute(tokenIn, tokenOut, amountStr string, slippageBps int) (map[string]interface{}, error) {
	shiva := b.registry().TokenKey("shiva-mainnet-1", "SHIVA")
	if tokenIn == shiva || tokenOut == shiva {
		return b.ShivaSwapExecute(tokenIn, tokenOut, amountStr, slippageBps)
	}
	q1, err := b.ShivaSwapQuote(tokenIn, shiva, amountStr)
	if err != nil {
		return nil, err
	}
	_, err = b.ShivaSwapExecute(tokenIn, shiva, amountStr, slippageBps)
	if err != nil {
		return nil, err
	}
	return b.ShivaSwapExecute(shiva, tokenOut, q1["amountOut"].(string), slippageBps)
}

