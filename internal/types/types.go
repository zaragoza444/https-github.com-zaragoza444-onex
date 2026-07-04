package types

import (
	"encoding/hex"
	"encoding/json"
	"time"
)

const CoinDecimals = 8

type Address string

func (a Address) String() string { return string(a) }

func (a Address) Valid() bool {
	if len(a) != 64 {
		return false
	}
	_, err := hex.DecodeString(string(a))
	return err == nil && len(a) == 64
}

type Transaction struct {
	From      Address `json:"from"`
	To        Address `json:"to"`
	Amount    uint64  `json:"amount"`
	Fee       uint64  `json:"fee"`
	Nonce     uint64  `json:"nonce"`
	Signature string  `json:"signature"`
}

type BlockHeader struct {
	Index        uint64 `json:"index"`
	Timestamp    int64  `json:"timestamp"`
	PreviousHash string `json:"previousHash"`
	StateRoot    string `json:"stateRoot"`
	Difficulty   uint64 `json:"difficulty"`
	Miner        Address `json:"miner"`
	Nonce        uint64 `json:"nonce"`
}

type Block struct {
	Header       BlockHeader   `json:"header"`
	Transactions []Transaction `json:"transactions"`
	Hash         string        `json:"hash"`
}

func NewGenesisBlock(alloc map[Address]uint64, difficulty uint64, chainID string) *Block {
	txs := make([]Transaction, 0, len(alloc))
	for addr, bal := range alloc {
		txs = append(txs, Transaction{
			From:   "",
			To:     addr,
			Amount: bal,
			Fee:    0,
			Nonce:  0,
		})
	}
	b := &Block{
		Header: BlockHeader{
			Index:        0,
			Timestamp:    time.Now().Unix(),
			PreviousHash: "0",
			StateRoot:    "",
			Difficulty:   difficulty,
			Miner:        "",
		},
		Transactions: txs,
	}
	return b
}

type AccountState struct {
	Balance uint64 `json:"balance"`
	Nonce   uint64 `json:"nonce"`
}

type ChainState map[Address]AccountState

func (s ChainState) Clone() ChainState {
	out := make(ChainState, len(s))
	for k, v := range s {
		out[k] = v
	}
	return out
}

type GenesisConfig struct {
	ChainID    string            `json:"chainId"`
	NetworkID  uint64            `json:"networkId"` // EIP-155 style numeric id for wallets (mainnet 9001)
	Difficulty uint64            `json:"difficulty"`
	Alloc      map[string]uint64 `json:"alloc"`
	Reward     uint64            `json:"blockReward"`
}

type PeerInfo struct {
	ID      string `json:"id"`
	Address string `json:"address"`
	Height  uint64 `json:"height"`
}

type APIStatus struct {
	ChainID   string `json:"chainId"`
	NetworkID uint64 `json:"networkId"`
	Height    uint64 `json:"height"`
	Hash      string `json:"hash"`
	Peers     int    `json:"peers"`
	Mining    bool   `json:"mining"`
	RPCURL    string `json:"rpcUrl,omitempty"`
}

func MustJSON(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}
