package ledger

import (
	"fmt"
	"strings"
	"time"
)

// ImportRequest imports external ledger data with optional preview or active book sync.
type ImportRequest struct {
	Active       bool   `json:"active,omitempty"`
	Preview      bool   `json:"preview,omitempty"`
	FiatCurrency string `json:"fiatCurrency,omitempty"`
	Raw          []byte `json:"-"`
}

// ImportResult is the outcome of import & value.
type ImportResult struct {
	Status         string             `json:"status"` // preview, imported, active
	Path           string             `json:"path,omitempty"`
	Entries        int                `json:"entries"`
	TotalUSD       float64            `json:"totalUsd"`
	ImportUSD      float64            `json:"importUsd"`
	ByAsset        map[string]float64 `json:"byAssetUsd,omitempty"`
	AccountsSynced int                `json:"accountsSynced"`
	Snapshot       Snapshot           `json:"snapshot,omitempty"`
	Imported       []Entry            `json:"imported,omitempty"`
}

// ValueImportEntries applies live prices to parsed import rows.
func ValueImportEntries(entries []Entry, prices map[string]PriceQuote, fiat string) []Entry {
	if fiat == "" {
		fiat = "USD"
	}
	out := make([]Entry, len(entries))
	for i, e := range entries {
		e.FiatCurrency = fiat
		out[i] = ValueEntry(e, prices)
	}
	return out
}

// SummarizeImport builds USD totals for imported entries only.
func SummarizeImport(entries []Entry) (totalUSD float64, byAsset map[string]float64) {
	byAsset = make(map[string]float64)
	for _, e := range entries {
		if e.Source != SourceImport {
			continue
		}
		totalUSD += e.FiatUSD
		byAsset[strings.ToUpper(e.Asset)] += e.FiatUSD
	}
	return totalUSD, byAsset
}

// SyncImportEntries force-upserts import-source accounts into the ledger book.
func (s *BookStore) SyncImportEntries(entries []Entry) (int, error) {
	if err := s.load(); err != nil {
		return 0, err
	}
	s.mu.Lock()
	now := time.Now().Unix()
	synced := 0
	for _, e := range entries {
		if e.Source != SourceImport {
			continue
		}
		id := importAccountID(e)
		bal := strings.TrimSpace(e.Human)
		if bal == "" && e.Atomic != "" {
			bal = atomicToHumanStr(e.Atomic, decimalsForSymbol(e.Asset))
		}
		if bal == "" || parseHuman(bal) <= 0 {
			continue
		}
		s.data.Accounts[id] = &BookAccount{
			ID:        id,
			Source:    SourceImport,
			Mode:      e.Mode,
			Asset:     strings.ToUpper(e.Asset),
			FundClass: e.FundClass,
			TokenKey:  e.TokenKey,
			ChainID:   e.ChainID,
			Balance:   bal,
			Account:   e.Account,
			UpdatedAt: now,
		}
		synced++
	}
	s.mu.Unlock()
	if synced > 0 {
		if err := s.save(); err != nil {
			return synced, err
		}
	}
	return synced, nil
}

func importAccountID(e Entry) string {
	if e.ID != "" && strings.HasPrefix(e.ID, "import-") {
		return e.ID
	}
	asset := strings.ToLower(strings.TrimSpace(e.Asset))
	if acct := strings.TrimSpace(e.Account); acct != "" {
		return fmt.Sprintf("import-%s-%s", asset, strings.ToLower(acct))
	}
	return "import-" + asset
}

// ProcessImport parses, values, and optionally persists import data to the book.
func ProcessImport(req ImportRequest, parse func([]byte) ([]Entry, error), save func([]byte) (string, error), prices map[string]PriceQuote, readSnap func([]byte) Snapshot) (*ImportResult, error) {
	if len(req.Raw) == 0 {
		return nil, fmt.Errorf("import data required")
	}
	entries, err := parse(req.Raw)
	if err != nil {
		return nil, err
	}
	fiat := strings.ToUpper(strings.TrimSpace(req.FiatCurrency))
	if fiat == "" {
		fiat = "USD"
	}
	valued := ValueImportEntries(entries, prices, fiat)
	importUSD, byAsset := SummarizeImport(valued)

	res := &ImportResult{
		Entries:   len(valued),
		ImportUSD: importUSD,
		ByAsset:   byAsset,
		Imported:  valued,
	}

	if req.Preview {
		res.Status = "preview"
		res.TotalUSD = importUSD
		return res, nil
	}

	if save != nil && !req.Preview {
		path, err := save(req.Raw)
		if err != nil {
			return nil, err
		}
		res.Path = path
	}

	if readSnap != nil {
		res.Snapshot = readSnap(req.Raw)
		res.TotalUSD = res.Snapshot.TotalUSD
	}

	if req.Active {
		res.Status = "active"
	} else {
		res.Status = "imported"
	}

	return res, nil
}
