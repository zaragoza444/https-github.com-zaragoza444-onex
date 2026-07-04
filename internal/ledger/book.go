package ledger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// BookAccount is a transferable balance in the unified ledger book.
type BookAccount struct {
	ID        string     `json:"id"`
	Source    SourceKind `json:"source"`
	Mode      Mode       `json:"mode"`
	Asset     string     `json:"asset"`
	FundClass string     `json:"fundClass,omitempty"`
	TokenKey  string     `json:"tokenKey,omitempty"`
	ChainID   string     `json:"chainId,omitempty"`
	Balance   string     `json:"balance"`
	Account   string     `json:"account,omitempty"`
	UpdatedAt int64      `json:"updatedAt"`
}

// TransferRecord logs a completed or pending ledger movement.
type TransferRecord struct {
	ID          string `json:"id"`
	FromAccount string `json:"fromAccount"`
	ToAccount   string `json:"toAccount"`
	Asset       string `json:"asset"`
	Amount      string `json:"amount"`
	ToAmount    string `json:"toAmount,omitempty"`
	ConvertTo   string `json:"convertTo,omitempty"`
	Status      string `json:"status"` // completed, pending, on_chain
	TxRef       string `json:"txRef,omitempty"`
	Note        string `json:"note,omitempty"`
	CreatedAt   int64  `json:"createdAt"`
}

// Book is the mutable unified ledger for any bank/crypto source.
type Book struct {
	Accounts    map[string]*BookAccount `json:"accounts"`
	Transfers   []TransferRecord        `json:"transfers"`
	Settlements []SettlementRecord      `json:"settlements,omitempty"`
}

// BookStore persists ledger accounts and transfer history.
type BookStore struct {
	mu   sync.Mutex
	path string
	data *Book
}

func NewBookStore(dir string) *BookStore {
	return &BookStore{
		path: filepath.Join(dir, "ledger-book.json"),
		data: &Book{Accounts: make(map[string]*BookAccount)},
	}
}

func (s *BookStore) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data != nil && len(s.data.Accounts) > 0 {
		return nil
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			s.data = &Book{Accounts: make(map[string]*BookAccount)}
			return nil
		}
		return err
	}
	var b Book
	if err := json.Unmarshal(data, &b); err != nil {
		return err
	}
	if b.Accounts == nil {
		b.Accounts = make(map[string]*BookAccount)
	}
	s.data = &b
	return nil
}

func (s *BookStore) save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, raw, 0o600)
}

// SyncFromSnapshot upserts accounts from a real ledger read.
func (s *BookStore) SyncFromSnapshot(snap Snapshot) error {
	if err := s.load(); err != nil {
		return err
	}
	s.mu.Lock()
	now := time.Now().Unix()
	for _, e := range snap.Entries {
		if e.ID == "" {
			continue
		}
		bal := strings.TrimSpace(e.Human)
		if bal == "" && e.Atomic != "" {
			bal = atomicToHumanStr(e.Atomic, decimalsForSymbol(e.Asset))
		}
		if bal == "" {
			continue
		}
		existing, ok := s.data.Accounts[e.ID]
		if ok && parseHuman(existing.Balance) > parseHuman(bal) {
			// keep lower book balance if user transferred out since last sync
			continue
		}
		s.data.Accounts[e.ID] = &BookAccount{
			ID:        e.ID,
			Source:    e.Source,
			Mode:      e.Mode,
			Asset:     strings.ToUpper(e.Asset),
			FundClass: e.FundClass,
			TokenKey:  e.TokenKey,
			ChainID:   e.ChainID,
			Balance:   bal,
			Account:   e.Account,
			UpdatedAt: now,
		}
	}
	s.mu.Unlock()
	return s.save()
}

func (s *BookStore) ListAccounts() ([]BookAccount, error) {
	if err := s.load(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]BookAccount, 0, len(s.data.Accounts))
	for _, a := range s.data.Accounts {
		if a == nil || parseHuman(a.Balance) <= 0 {
			continue
		}
		cp := *a
		out = append(out, cp)
	}
	return out, nil
}

// ListTransfers returns recent transfer records newest first.
func (s *BookStore) ListTransfers(limit int) []TransferRecord {
	if err := s.load(); err != nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if limit <= 0 || limit > len(s.data.Transfers) {
		limit = len(s.data.Transfers)
	}
	if limit > 50 {
		limit = 50
	}
	out := make([]TransferRecord, limit)
	copy(out, s.data.Transfers[:limit])
	return out
}

func (s *BookStore) GetAccount(id string) (*BookAccount, error) {
	if err := s.load(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.data.Accounts[id]
	if !ok || a == nil {
		return nil, fmt.Errorf("account not found: %s", id)
	}
	cp := *a
	return &cp, nil
}

// TransferRequest moves value between ledger accounts with optional conversion.
type TransferRequest struct {
	FromAccount  string `json:"fromAccount"`
	ToAccount    string `json:"toAccount"`
	Amount       string `json:"amount"`
	Asset        string `json:"asset,omitempty"`
	ConvertTo    string `json:"convertTo,omitempty"`
	ExternalTo   string `json:"externalTo,omitempty"`
	ExternalChain string `json:"externalChain,omitempty"` // chain id e.g. ethereum
	ExternalBank  string `json:"externalBank,omitempty"`  // bank id e.g. hsbc
	BankRail      string `json:"bankRail,omitempty"`      // ach, sepa, swift, wire
	ExternalAddress string `json:"externalAddress,omitempty"`
	Note         string `json:"note,omitempty"`
	Preview      bool   `json:"preview,omitempty"`
}

// TransferResult is the outcome of a ledger transfer.
type TransferResult struct {
	Status      string                `json:"status"`
	Transfer    TransferRecord        `json:"transfer"`
	Convert     *ConvertResult        `json:"convert,omitempty"`
	External    *ExternalDestination  `json:"external,omitempty"`
	Settlement  string                `json:"settlement,omitempty"`
}

func (s *BookStore) Transfer(req TransferRequest, prices map[string]PriceQuote, settle func(TransferRecord, *ExternalDestination) (string, error)) (*TransferResult, error) {
	if err := s.load(); err != nil {
		return nil, err
	}
	from, err := s.GetAccount(req.FromAccount)
	if err != nil {
		return nil, err
	}
	amt := parseHuman(req.Amount)
	if amt <= 0 {
		return nil, fmt.Errorf("invalid amount")
	}
	fromBal := parseHuman(from.Balance)
	if fromBal < amt {
		return nil, fmt.Errorf("insufficient balance: have %s %s", from.Balance, from.Asset)
	}

	outAsset := strings.ToUpper(strings.TrimSpace(from.Asset))
	outAmt := amt
	var conv *ConvertResult
	if c := strings.ToUpper(strings.TrimSpace(req.ConvertTo)); c != "" && c != outAsset {
		cr, err := ConvertAmount(ConvertRequest{
			FromAsset: outAsset,
			ToAsset:   c,
			Amount:    formatFloat(amt),
		}, prices, nil)
		if err != nil {
			return nil, err
		}
		conv = cr
		outAsset = c
		outAmt = parseHuman(cr.ToAmount)
	}

	toID := strings.TrimSpace(req.ToAccount)
	var extDest *ExternalDestination
	extRaw := ResolveExternalRaw(req)
	if toID == "" && extRaw != "" {
		var err error
		toID, extDest, err = ExternalTo(extRaw, outAsset)
		if err != nil {
			return nil, err
		}
	}
	if toID == "" {
		return nil, fmt.Errorf("toAccount or externalTo required")
	}

	if req.Preview {
		rec := TransferRecord{
			ID:          "preview",
			FromAccount: from.ID,
			ToAccount:   toID,
			Asset:       from.Asset,
			Amount:      formatFloat(amt),
			ToAmount:    formatFloat(outAmt),
			ConvertTo:   req.ConvertTo,
			Status:      "preview",
			Note:        req.Note,
			CreatedAt:   time.Now().Unix(),
		}
		return &TransferResult{Status: "preview", Transfer: rec, Convert: conv, External: extDest}, nil
	}

	s.mu.Lock()
	from.Balance = formatFloat(fromBal - amt)
	from.UpdatedAt = time.Now().Unix()
	s.data.Accounts[from.ID] = from

	to := s.data.Accounts[toID]
	if to == nil {
		to = &BookAccount{
			ID:        toID,
			Source:    SourceImport,
			Mode:      ModeReal,
			Asset:     outAsset,
			Balance:   "0",
			UpdatedAt: time.Now().Unix(),
		}
		if extDest != nil {
			switch extDest.Kind {
			case ExternalOneX:
				to.Source = SourceOneX
				to.ChainID = extDest.ChainID
				to.TokenKey = "onex-mainnet-1:ONEX"
			case ExternalBank:
				to.Source = SourceBank
				to.Mode = ModeBank
				to.Account = extDest.Label
			case ExternalEVM, ExternalSolana, ExternalBitcoin, ExternalTron:
				to.Source = SourceEVM
				to.ChainID = extDest.ChainID
				to.Account = extDest.Address
			}
		} else if strings.HasPrefix(toID, "external:onex:") {
			to.Source = SourceOneX
			to.ChainID = "onex-mainnet-1"
			to.TokenKey = "onex-mainnet-1:ONEX"
			to.Mode = ModeReal
		} else if strings.HasPrefix(toID, "external:bank:") {
			to.Source = SourceBank
			to.Mode = ModeBank
		} else if strings.HasPrefix(toID, "external:crypto:") {
			to.Source = SourceEVM
			to.Mode = ModeReal
		}
	}
	to.Balance = formatFloat(parseHuman(to.Balance) + outAmt)
	to.Asset = outAsset
	to.UpdatedAt = time.Now().Unix()
	s.data.Accounts[toID] = to

	rec := TransferRecord{
		ID:          fmt.Sprintf("xfer-%d", time.Now().UnixNano()),
		FromAccount: from.ID,
		ToAccount:   toID,
		Asset:       from.Asset,
		Amount:      formatFloat(amt),
		ToAmount:    formatFloat(outAmt),
		ConvertTo:   req.ConvertTo,
		Status:      "completed",
		Note:        req.Note,
		CreatedAt:   time.Now().Unix(),
	}
	if extRaw != "" {
		rec.Status = "pending"
	}
	s.data.Transfers = append([]TransferRecord{rec}, s.data.Transfers...)
	if len(s.data.Transfers) > 500 {
		s.data.Transfers = s.data.Transfers[:500]
	}
	s.mu.Unlock()

	if err := s.save(); err != nil {
		return nil, err
	}

	if settle != nil && extRaw != "" {
		txRef, err := settle(rec, extDest)
		if err != nil {
			return &TransferResult{Status: "failed", Transfer: rec, Convert: conv, External: extDest}, err
		}
		rec.TxRef = txRef
		if extDest != nil && extDest.Kind == ExternalOneX {
			rec.Status = "on_chain"
		} else if txRef != "" && !strings.HasPrefix(txRef, "chain-pending:") && txRef != "pending-settlement" {
			if strings.Contains(txRef, "/tx/") || strings.HasPrefix(txRef, "tx:0x") {
				rec.Status = "on_chain"
			} else {
				rec.Status = "submitted"
			}
		}
		s.mu.Lock()
		if len(s.data.Transfers) > 0 {
			s.data.Transfers[0] = rec
		}
		s.mu.Unlock()
		_ = s.save()
	}

	return &TransferResult{Status: rec.Status, Transfer: rec, Convert: conv, External: extDest, Settlement: rec.TxRef}, nil
}

// ConvertActive debits a ledger account and credits the target asset vault using live rates.
func (s *BookStore) ConvertActive(req ConvertRequest, prices map[string]PriceQuote, tokens map[string]TokenMeta) (*ConvertResult, error) {
	fromID := strings.TrimSpace(req.FromAccount)
	if fromID == "" {
		var err error
		fromID, err = s.pickAccount(req.FromAsset)
		if err != nil {
			return nil, err
		}
	}
	toAsset := strings.ToUpper(strings.TrimSpace(req.ToAsset))
	if toAsset == "" {
		return nil, fmt.Errorf("toAsset required")
	}
	fromAsset := strings.ToUpper(strings.TrimSpace(req.FromAsset))
	if fromAsset == toAsset {
		return nil, fmt.Errorf("from and to asset must differ")
	}

	conv, err := ConvertAmount(req, prices, tokens)
	if err != nil {
		return nil, err
	}

	fromAcct, _ := s.GetAccount(fromID)
	toID := bookVaultID(toAsset)
	res, err := s.Transfer(TransferRequest{
		FromAccount: fromID,
		ToAccount:   toID,
		Amount:      req.Amount,
		ConvertTo:   toAsset,
		Note:        "ledger-convert",
	}, prices, nil)
	if err != nil {
		return nil, err
	}
	conv.FromAccount = fromID
	conv.ToAccount = toID
	if fromAcct != nil {
		conv.FundClass = fromAcct.FundClass
	}
	conv.Status = "completed"
	if res != nil {
		conv.TransferID = res.Transfer.ID
		if res.Status != "" {
			conv.Status = res.Status
		}
	}
	return conv, nil
}

func bookVaultID(asset string) string {
	return "book:" + strings.ToUpper(strings.TrimSpace(asset))
}

func (s *BookStore) pickAccount(asset string) (string, error) {
	asset = strings.ToUpper(strings.TrimSpace(asset))
	accounts, err := s.ListAccounts()
	if err != nil {
		return "", err
	}
	var best string
	var bestBal float64
	for _, a := range accounts {
		if strings.ToUpper(a.Asset) != asset {
			continue
		}
		b := parseHuman(a.Balance)
		if b > bestBal {
			bestBal = b
			best = a.ID
		}
	}
	if best == "" {
		return "", fmt.Errorf("no ledger account with asset %s", asset)
	}
	return best, nil
}

// ResolveExternalRaw builds a canonical external destination string from a transfer request.
func ResolveExternalRaw(req TransferRequest) string {
	if strings.TrimSpace(req.ExternalTo) != "" {
		return strings.TrimSpace(req.ExternalTo)
	}
	addr := strings.TrimSpace(req.ExternalAddress)
	if addr == "" {
		return ""
	}
	if chain := strings.TrimSpace(req.ExternalChain); chain != "" {
		return chain + ":" + addr
	}
	if bank := strings.TrimSpace(req.ExternalBank); bank != "" {
		rail := strings.TrimSpace(req.BankRail)
		if rail == "" {
			rail = "swift"
		}
		return "bank:" + bank + ":" + rail + ":" + addr
	}
	if rail := strings.TrimSpace(req.BankRail); rail != "" {
		return "bank:" + rail + ":" + addr
	}
	return addr
}

func externalAccountID(external, asset string) string {
	external = strings.TrimSpace(external)
	if external == "" {
		return ""
	}
	if strings.Contains(external, ":") {
		parts := strings.SplitN(external, ":", 2)
		switch strings.ToLower(parts[0]) {
		case "onex", "bank", "crypto", "evm":
			return "external:" + strings.ToLower(parts[0]) + ":" + strings.TrimSpace(parts[1])
		}
	}
	if strings.HasPrefix(strings.ToLower(external), "0x") {
		return "external:crypto:" + strings.ToLower(external)
	}
	if len(external) > 10 && strings.Contains(external, "US") {
		return "external:bank:" + external
	}
	return "external:acct:" + asset + ":" + external
}

func parseHuman(s string) float64 {
	v, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return v
}
