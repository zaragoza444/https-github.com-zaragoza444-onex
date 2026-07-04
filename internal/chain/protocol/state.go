package protocol

import (
	"encoding/json"
	"sort"

	"github.com/onex-blockchain/onex/internal/crypto"
	"github.com/onex-blockchain/onex/internal/types"
)

// State holds on-chain DeFi protocol data for OneX L1.
type State struct {
	TokenBalances map[string]map[string]uint64 `json:"tokenBalances"` // addr -> tokenId -> amount
	Pools         map[string]Pool              `json:"pools"`
	LPShares      map[string]map[string]uint64 `json:"lpShares"` // addr -> poolId -> shares
	Stakes        map[string]Stake             `json:"stakes"`   // stakeId -> record
	NFTs          map[string]NFT               `json:"nfts"`
	Loans         map[string]Loan              `json:"loans"`
}

type Pool struct {
	ID          string `json:"id"`
	Token0      string `json:"token0"`
	Token1      string `json:"token1"`
	Reserve0    uint64 `json:"reserve0"`
	Reserve1    uint64 `json:"reserve1"`
	TotalShares uint64 `json:"totalShares"`
	FeeBps      int    `json:"feeBps"`
}

type Stake struct {
	ID         string `json:"id"`
	Owner      string `json:"owner"`
	PoolID     string `json:"poolId"`
	StakeToken string `json:"stakeToken"`
	ReceiptToken string `json:"receiptToken"`
	Amount     uint64 `json:"amount"`
	ReceiptAmt uint64 `json:"receiptAmt"`
	StakedAt   int64  `json:"stakedAt"`
	UnlockAt   int64  `json:"unlockAt"`
	Status     string `json:"status"`
}

type NFT struct {
	ID          string `json:"id"`
	Owner       string `json:"owner"`
	ChainID     string `json:"chainId"`
	Name        string `json:"name"`
	Description string `json:"description"`
	ImageURL    string `json:"imageUrl"`
	CreatedAt   int64  `json:"createdAt"`
}

type Loan struct {
	ID            string  `json:"id"`
	Owner         string  `json:"owner"`
	Type          string  `json:"type"`
	CollateralKey string  `json:"collateralKey"`
	CollateralAmt uint64  `json:"collateralAmt"`
	DebtKey       string  `json:"debtKey"`
	DebtAmt       uint64  `json:"debtAmt"`
	APY           float64 `json:"apy"`
	Status        string  `json:"status"`
	CreatedAt     int64   `json:"createdAt"`
}

func NewState() *State {
	return &State{
		TokenBalances: make(map[string]map[string]uint64),
		Pools:         make(map[string]Pool),
		LPShares:      make(map[string]map[string]uint64),
		Stakes:        make(map[string]Stake),
		NFTs:          make(map[string]NFT),
		Loans:         make(map[string]Loan),
	}
}

func (s *State) Clone() *State {
	b, _ := json.Marshal(s)
	var out State
	_ = json.Unmarshal(b, &out)
	if out.TokenBalances == nil {
		out.TokenBalances = make(map[string]map[string]uint64)
	}
	if out.Pools == nil {
		out.Pools = make(map[string]Pool)
	}
	if out.LPShares == nil {
		out.LPShares = make(map[string]map[string]uint64)
	}
	if out.Stakes == nil {
		out.Stakes = make(map[string]Stake)
	}
	if out.NFTs == nil {
		out.NFTs = make(map[string]NFT)
	}
	if out.Loans == nil {
		out.Loans = make(map[string]Loan)
	}
	return &out
}

func (s *State) TokenBalance(addr types.Address, tokenID string) uint64 {
	if m, ok := s.TokenBalances[string(addr)]; ok {
		return m[tokenID]
	}
	return 0
}

func (s *State) SetTokenBalance(addr types.Address, tokenID string, amt uint64) {
	key := string(addr)
	if s.TokenBalances[key] == nil {
		s.TokenBalances[key] = make(map[string]uint64)
	}
	if amt == 0 {
		delete(s.TokenBalances[key], tokenID)
		if len(s.TokenBalances[key]) == 0 {
			delete(s.TokenBalances, key)
		}
		return
	}
	s.TokenBalances[key][tokenID] = amt
}

func (s *State) AddTokenBalance(addr types.Address, tokenID string, delta uint64) error {
	cur := s.TokenBalance(addr, tokenID)
	s.SetTokenBalance(addr, tokenID, cur+delta)
	return nil
}

func (s *State) SubTokenBalance(addr types.Address, tokenID string, delta uint64) error {
	cur := s.TokenBalance(addr, tokenID)
	if cur < delta {
		return errInsufficientToken
	}
	s.SetTokenBalance(addr, tokenID, cur-delta)
	return nil
}

func RootHash(s *State) string {
	if s == nil {
		return crypto.Hash([]byte("protocol:empty"))
	}
	b, err := json.Marshal(s.canonical())
	if err != nil {
		return crypto.Hash([]byte("protocol:err"))
	}
	return crypto.Hash(b)
}

func (s *State) canonical() map[string]interface{} {
	return map[string]interface{}{
		"tokenBalances": canonicalTokenBalances(s.TokenBalances),
		"pools":         canonicalPools(s.Pools),
		"lpShares":      canonicalTokenBalances(s.LPShares),
		"stakes":        canonicalStakes(s.Stakes),
		"nfts":          canonicalNFTs(s.NFTs),
		"loans":         canonicalLoans(s.Loans),
	}
}

func canonicalTokenBalances(m map[string]map[string]uint64) map[string]map[string]uint64 {
	if len(m) == 0 {
		return map[string]map[string]uint64{}
	}
	addrs := sortedKeys(m)
	out := make(map[string]map[string]uint64, len(addrs))
	for _, a := range addrs {
		tokens := sortedKeys(m[a])
		inner := make(map[string]uint64, len(tokens))
		for _, t := range tokens {
			inner[t] = m[a][t]
		}
		out[a] = inner
	}
	return out
}

func canonicalPools(m map[string]Pool) map[string]Pool {
	if len(m) == 0 {
		return map[string]Pool{}
	}
	ids := sortedPoolKeys(m)
	out := make(map[string]Pool, len(ids))
	for _, id := range ids {
		out[id] = m[id]
	}
	return out
}

func canonicalStakes(m map[string]Stake) map[string]Stake {
	if len(m) == 0 {
		return map[string]Stake{}
	}
	ids := sortedStakeKeys(m)
	out := make(map[string]Stake, len(ids))
	for _, id := range ids {
		out[id] = m[id]
	}
	return out
}

func canonicalNFTs(m map[string]NFT) map[string]NFT {
	if len(m) == 0 {
		return map[string]NFT{}
	}
	ids := sortedNFTKeys(m)
	out := make(map[string]NFT, len(ids))
	for _, id := range ids {
		out[id] = m[id]
	}
	return out
}

func canonicalLoans(m map[string]Loan) map[string]Loan {
	if len(m) == 0 {
		return map[string]Loan{}
	}
	ids := sortedLoanKeys(m)
	out := make(map[string]Loan, len(ids))
	for _, id := range ids {
		out[id] = m[id]
	}
	return out
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedPoolKeys(m map[string]Pool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedStakeKeys(m map[string]Stake) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedNFTKeys(m map[string]NFT) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedLoanKeys(m map[string]Loan) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
