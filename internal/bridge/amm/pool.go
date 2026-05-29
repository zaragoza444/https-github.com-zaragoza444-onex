package amm

import (
	"fmt"
	"math/big"
	"strings"
)

const DefaultFeeBps = 30 // 0.30% like Uniswap V2

// Pool is a constant-product AMM pair (x * y = k).
type Pool struct {
	ID          string `json:"id"`
	Token0      string `json:"token0"`
	Token1      string `json:"token1"`
	Reserve0    string `json:"reserve0"`
	Reserve1    string `json:"reserve1"`
	TotalShares string `json:"totalShares"`
	FeeBps      int    `json:"feeBps"`
}

func PoolID(tokenA, tokenB string) string {
	if tokenA > tokenB {
		tokenA, tokenB = tokenB, tokenA
	}
	return tokenA + "|" + tokenB
}

func (p *Pool) r0() *big.Int {
	return mustBig(p.Reserve0)
}

func (p *Pool) r1() *big.Int {
	return mustBig(p.Reserve1)
}

func (p *Pool) shares() *big.Int {
	return mustBig(p.TotalShares)
}

func mustBig(s string) *big.Int {
	z, _ := new(big.Int).SetString(s, 10)
	if z == nil {
		return big.NewInt(0)
	}
	return z
}

func fmtBig(z *big.Int) string {
	if z == nil {
		return "0"
	}
	return z.String()
}

// QuoteSwap returns amount out for exact in (Uniswap V2 style).
func (p *Pool) QuoteSwap(tokenIn string, amountIn uint64) (amountOut uint64, priceImpact string, err error) {
	ain := new(big.Int).SetUint64(amountIn)
	var rin, rout *big.Int
	switch {
	case strings.EqualFold(tokenIn, p.Token0):
		rin, rout = p.r0(), p.r1()
	case strings.EqualFold(tokenIn, p.Token1):
		rin, rout = p.r1(), p.r0()
	default:
		return 0, "", fmt.Errorf("token not in pool")
	}
	if rin.Sign() == 0 || rout.Sign() == 0 {
		return 0, "", fmt.Errorf("insufficient liquidity")
	}
	feeNum := big.NewInt(int64(10000 - p.FeeBps))
	ainWithFee := new(big.Int).Mul(ain, feeNum)
	num := new(big.Int).Mul(ainWithFee, rout)
	den := new(big.Int).Add(new(big.Int).Mul(rin, big.NewInt(10000)), ainWithFee)
	out := new(big.Int).Div(num, den)
	if !out.IsUint64() {
		return 0, "", fmt.Errorf("amount too large")
	}
	// price impact approx
	spot := new(big.Rat).SetFrac(rout, rin)
	exec := new(big.Rat).SetFrac(out, ain)
	impact := new(big.Rat).Sub(big.NewRat(1, 1), new(big.Rat).Quo(exec, spot))
	pct, _ := impact.Float64()
	return out.Uint64(), fmt.Sprintf("%.4f%%", pct*100), nil
}

// Swap updates reserves after swap; returns amount out.
func (p *Pool) Swap(tokenIn string, amountIn uint64) (uint64, error) {
	out, _, err := p.QuoteSwap(tokenIn, amountIn)
	if err != nil {
		return 0, err
	}
	ain := new(big.Int).SetUint64(amountIn)
	aout := new(big.Int).SetUint64(out)
	if strings.EqualFold(tokenIn, p.Token0) {
		p.Reserve0 = fmtBig(new(big.Int).Add(p.r0(), ain))
		p.Reserve1 = fmtBig(new(big.Int).Sub(p.r1(), aout))
	} else {
		p.Reserve1 = fmtBig(new(big.Int).Add(p.r1(), ain))
		p.Reserve0 = fmtBig(new(big.Int).Sub(p.r0(), aout))
	}
	if p.r0().Sign() <= 0 || p.r1().Sign() <= 0 {
		return 0, fmt.Errorf("pool drained")
	}
	return out, nil
}

// AddLiquidity mints LP shares.
func (p *Pool) AddLiquidity(amount0, amount1 uint64) (shares uint64, err error) {
	a0 := new(big.Int).SetUint64(amount0)
	a1 := new(big.Int).SetUint64(amount1)
	ts := p.shares()
	r0, r1 := p.r0(), p.r1()
	var mint *big.Int
	if ts.Sign() == 0 {
		mint = new(big.Int).Sqrt(new(big.Int).Mul(a0, a1))
		if mint.Sign() == 0 {
			return 0, fmt.Errorf("insufficient amounts")
		}
	} else {
		s0 := new(big.Int).Mul(a0, ts)
		s0.Div(s0, r0)
		s1 := new(big.Int).Mul(a1, ts)
		s1.Div(s1, r1)
		if s0.Cmp(s1) < 0 {
			mint = s0
		} else {
			mint = s1
		}
	}
	if !mint.IsUint64() {
		return 0, fmt.Errorf("shares overflow")
	}
	p.Reserve0 = fmtBig(new(big.Int).Add(r0, a0))
	p.Reserve1 = fmtBig(new(big.Int).Add(r1, a1))
	p.TotalShares = fmtBig(new(big.Int).Add(ts, mint))
	return mint.Uint64(), nil
}

// RemoveLiquidity burns shares for underlying.
func (p *Pool) RemoveLiquidity(sharesBurn uint64) (amount0, amount1 uint64, err error) {
	burn := new(big.Int).SetUint64(sharesBurn)
	ts := p.shares()
	if burn.Cmp(ts) >= 0 {
		burn = new(big.Int).Sub(ts, big.NewInt(1))
	}
	if burn.Sign() <= 0 {
		return 0, 0, fmt.Errorf("invalid shares")
	}
	r0, r1 := p.r0(), p.r1()
	a0 := new(big.Int).Mul(burn, r0)
	a0.Div(a0, ts)
	a1 := new(big.Int).Mul(burn, r1)
	a1.Div(a1, ts)
	p.Reserve0 = fmtBig(new(big.Int).Sub(r0, a0))
	p.Reserve1 = fmtBig(new(big.Int).Sub(r1, a1))
	p.TotalShares = fmtBig(new(big.Int).Sub(ts, burn))
	return a0.Uint64(), a1.Uint64(), nil
}
