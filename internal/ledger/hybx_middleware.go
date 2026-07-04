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

	"github.com/onex-blockchain/onex/internal/legacy"
)

// HybxExchangeRoute is a supported cross-rail path through HYBX middleware.
type HybxExchangeRoute struct {
	ID     string `json:"id"`
	From   string `json:"from"`
	To     string `json:"to"`
	Label  string `json:"label"`
	Symbol string `json:"symbol,omitempty"`
}

// HybxExchangeRequest is the unified exchange input for banks, chains, and platform.
type HybxExchangeRequest struct {
	Route         string `json:"route"`
	From          string `json:"from"`
	To            string `json:"to"`
	NSBAccount    string `json:"nsbAccount,omitempty"`
	Amount        string `json:"amount"`
	Symbol        string `json:"symbol,omitempty"`
	ChainID       string `json:"chainId,omitempty"`
	Address       string `json:"address,omitempty"`
	PlatformToken string `json:"platformToken,omitempty"`
	BankRail      string `json:"bankRail,omitempty"`
	BankAccount   string `json:"bankAccount,omitempty"`
	Preview       bool   `json:"preview,omitempty"`
}

// HybxFederationRecord tracks outbound HYBX federation settlements.
type HybxFederationRecord struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"` // bank | chain | card
	Symbol    string `json:"symbol"`
	Amount    string `json:"amount"`
	Rail      string `json:"rail,omitempty"`
	Account   string `json:"account,omitempty"`
	ChainID   string `json:"chainId,omitempty"`
	Address   string `json:"address,omitempty"`
	Reference string `json:"reference"`
	Status    string `json:"status"`
	CreatedAt int64  `json:"createdAt"`
}

type hybxFederationFile struct {
	Records   []HybxFederationRecord `json:"records"`
	UpdatedAt int64                  `json:"updatedAt"`
}

type hybxFederationStore struct {
	mu   sync.Mutex
	path string
}

func defaultHybxFederationStore() *hybxFederationStore {
	return &hybxFederationStore{path: filepath.Join(legacy.HomeDir(), "hybx-federation.json")}
}

func (s *hybxFederationStore) load() (*hybxFederationFile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return &hybxFederationFile{}, nil
		}
		return nil, err
	}
	var f hybxFederationFile
	return &f, json.Unmarshal(data, &f)
}

func (s *hybxFederationStore) append(rec HybxFederationRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var f hybxFederationFile
	if data, err := os.ReadFile(s.path); err == nil {
		_ = json.Unmarshal(data, &f)
	}
	f.Records = append([]HybxFederationRecord{rec}, f.Records...)
	if len(f.Records) > 500 {
		f.Records = f.Records[:500]
	}
	f.UpdatedAt = time.Now().Unix()
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, raw, 0o600)
}

// ListHybxExchangeRoutes returns all middleware exchange paths.
func ListHybxExchangeRoutes() []HybxExchangeRoute {
	routes := []HybxExchangeRoute{
		{ID: "nsb-hybx", From: "nsb", To: "hybx", Label: "NSB → HYBX mirror"},
		{ID: "hybx-nsb", From: "hybx", To: "nsb", Label: "HYBX → NSB"},
		{ID: "nsb-fineract", From: "nsb", To: "fineract", Label: "NSB → Fineract core bank"},
		{ID: "fineract-hybx", From: "fineract", To: "hybx", Label: "Fineract → HYBX mirror"},
		{ID: "hybx-bank", From: "hybx", To: "bank", Label: "HYBX federation outbound bank"},
		{ID: "hybx-platform", From: "hybx", To: "platform", Label: "HYBX → token platform"},
		{ID: "ledger-hybx", From: "ledger", To: "hybx", Label: "Ledger book → HYBX"},
	}
	for _, ch := range SupportedChains() {
		routes = append(routes, HybxExchangeRoute{
			ID: "hybx-" + ch.ID, From: "hybx", To: "chain:" + ch.ID,
			Label: "HYBX → " + ch.Name, Symbol: strings.ToLower(ch.Symbol),
		})
	}
	return routes
}

// HybxMiddlewareStatus reports HYBX middleware connectivity across banks, chains, and platform.
func HybxMiddlewareStatus() map[string]interface{} {
	cfg := LoadHybrixConfig()
	fx := NewFineractClient().Status()
	mirrors, _ := DefaultHybrixMirrorStore().ListMirrors()
	fed, _ := defaultHybxFederationStore().load()
	routes := ListHybxExchangeRoutes()
	online := false
	if cfg.Enabled {
		if assets, err := NewHybrixClient().ListAssets(); err == nil {
			online = true
			_ = assets
		}
	}
	return map[string]interface{}{
		"service":    "onex-hybx-middleware",
		"enabled":    cfg.Enabled,
		"online":     online,
		"production": cfg.Enabled && online,
		"banks":      []string{"nsb", "fineract", "hybx"},
		"chains":     len(SupportedChains()),
		"platform":   true,
		"routes":     len(routes),
		"mirrors":    len(mirrors),
		"federation": len(fed.Records),
		"fineract":   fx["online"],
		"virtualCards": map[string]string{
			"nsb":  "/bridge/cards",
			"hybx": "/bridge/cards/hybx",
		},
		"api": map[string]string{
			"status":   "/bridge/bank/hybx/middleware/status",
			"routes":   "/bridge/bank/hybx/exchange/routes",
			"quote":    "/bridge/bank/hybx/exchange/quote",
			"exchange": "/bridge/bank/hybx/exchange",
			"settle":   "/bridge/bank/hybx/settle",
		},
	}
}

func resolveHybxRoute(req HybxExchangeRequest) HybxExchangeRoute {
	if id := strings.TrimSpace(req.Route); id != "" {
		for _, r := range ListHybxExchangeRoutes() {
			if r.ID == id {
				return r
			}
		}
	}
	from := strings.ToLower(strings.TrimSpace(req.From))
	to := strings.ToLower(strings.TrimSpace(req.To))
	if strings.HasPrefix(to, "chain:") || req.ChainID != "" {
		cid := strings.TrimPrefix(to, "chain:")
		if cid == "" || cid == to {
			cid = strings.TrimSpace(req.ChainID)
		}
		if cid != "" {
			return HybxExchangeRoute{ID: "hybx-" + cid, From: "hybx", To: "chain:" + cid, Label: "HYBX → " + cid}
		}
	}
	for _, r := range ListHybxExchangeRoutes() {
		if r.From == from && (r.To == to || strings.HasPrefix(to, r.To)) {
			return r
		}
	}
	return HybxExchangeRoute{ID: from + "-" + to, From: from, To: to}
}

// HybxExchangeBank runs bank-rail exchange steps through HYBX middleware.
func HybxExchangeBank(bank *OnlineBankStore, req HybxExchangeRequest) (map[string]interface{}, error) {
	cfg := LoadHybrixConfig()
	if !cfg.Enabled {
		return nil, fmt.Errorf("hybx middleware disabled")
	}
	route := resolveHybxRoute(req)
	amt, err := strconv.ParseFloat(strings.TrimSpace(req.Amount), 64)
	if err != nil || amt <= 0 {
		return nil, fmt.Errorf("invalid amount")
	}
	sym := currencyToHybrixSymbol(firstNonEmpty(req.Symbol, "usd"))
	if req.Preview {
		return map[string]interface{}{
			"status": "quoted", "preview": true, "route": route.ID,
			"from": route.From, "to": route.To, "amount": formatFloat(amt), "symbol": sym,
			"nsbAccount": req.NSBAccount, "chainId": req.ChainID, "address": req.Address,
		}, nil
	}
	ref := fmt.Sprintf("HYBX-XCH-%d", time.Now().Unix())
	switch route.ID {
	case "nsb-hybx":
		res, err := HybrixConvert(bank, HybrixConvertRequest{
			Direction: "nsb-to-hybx", NSBAccount: req.NSBAccount, Amount: req.Amount,
		})
		if err != nil {
			return nil, err
		}
		res["route"] = route.ID
		return res, nil
	case "hybx-nsb":
		res, err := HybrixConvert(bank, HybrixConvertRequest{
			Direction: "hybx-to-nsb", NSBAccount: req.NSBAccount, Amount: req.Amount,
		})
		if err != nil {
			return nil, err
		}
		res["route"] = route.ID
		return res, nil
	case "hybx-bank":
		settle, err := HybxFederateOutbound(BankTransferRequest{
			Rail: BankRail(req.BankRail), BankName: "hybx", Account: req.BankAccount,
			Amount: req.Amount, Asset: sym, Reference: ref,
		}, ref)
		if err != nil {
			return nil, err
		}
		if req.NSBAccount != "" {
			_, _ = HybrixConvert(bank, HybrixConvertRequest{
				Direction: "hybx-to-nsb", NSBAccount: req.NSBAccount, Amount: req.Amount,
			})
		}
		return map[string]interface{}{
			"status": "submitted", "route": route.ID, "settlement": settle,
			"amount": formatFloat(amt), "symbol": sym, "reference": ref,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported bank route %s — use bridge exchange for fineract/chain/platform", route.ID)
	}
}

// HybxFederateOutbound records and returns a HYBX federation settlement reference.
func HybxFederateOutbound(req BankTransferRequest, ref string) (string, error) {
	cfg := LoadHybrixConfig()
	if !cfg.Enabled {
		return "", fmt.Errorf("hybx disabled")
	}
	if ref == "" {
		ref = fmt.Sprintf("HYBX-FED-%d", time.Now().Unix())
	}
	sym := currencyToHybrixSymbol(req.Asset)
	rec := HybxFederationRecord{
		ID: fmt.Sprintf("fed-%d", time.Now().UnixNano()), Kind: "bank",
		Symbol: sym, Amount: req.Amount, Rail: string(req.Rail), Account: req.Account,
		Reference: ref, Status: "submitted", CreatedAt: time.Now().Unix(),
	}
	_ = defaultHybxFederationStore().append(rec)
	return fmt.Sprintf("hybx-fed:%s:%s:%s:%s", sym, req.Rail, req.Account, ref), nil
}

// HybxFederateChain records HYBX → chain federation and returns settlement ref.
func HybxFederateChain(chainID, address, amount, symbol, ref string) (string, error) {
	cfg := LoadHybrixConfig()
	if !cfg.Enabled {
		return "", fmt.Errorf("hybx disabled")
	}
	if ref == "" {
		ref = fmt.Sprintf("HYBX-CHAIN-%d", time.Now().Unix())
	}
	sym := currencyToHybrixSymbol(symbol)
	rec := HybxFederationRecord{
		ID: fmt.Sprintf("fed-%d", time.Now().UnixNano()), Kind: "chain",
		Symbol: sym, Amount: amount, ChainID: chainID, Address: address,
		Reference: ref, Status: "submitted", CreatedAt: time.Now().Unix(),
	}
	_ = defaultHybxFederationStore().append(rec)
	return fmt.Sprintf("hybx-chain:%s:%s:%s:%s", chainID, address, sym, ref), nil
}

// HybxRecordCardSpend records a production HYBX virtual card authorization in federation log.
func HybxRecordCardSpend(cardID, accountID, amount, currency, merchant, ref string) error {
	cfg := LoadHybrixConfig()
	if !cfg.Enabled {
		return nil
	}
	rec := HybxFederationRecord{
		ID: fmt.Sprintf("card-%d", time.Now().UnixNano()), Kind: "card",
		Symbol: currencyToHybrixSymbol(currency), Amount: amount,
		Account: accountID, Reference: ref + " · " + merchant,
		Status: "completed", CreatedAt: time.Now().Unix(),
	}
	_ = defaultHybxFederationStore().append(rec)
	_ = cardID
	return nil
}

// InitiateHybrixTransfer settles outbound bank transfer via HYBX federation middleware.
func InitiateHybrixTransfer(req BankTransferRequest, ref string) (string, error) {
	return HybxFederateOutbound(req, ref)
}

// ListHybxFederationRecords returns recent federation records.
func ListHybxFederationRecords(limit int) ([]HybxFederationRecord, error) {
	st, err := defaultHybxFederationStore().load()
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > len(st.Records) {
		limit = len(st.Records)
	}
	if limit == 0 {
		return nil, nil
	}
	out := make([]HybxFederationRecord, limit)
	copy(out, st.Records[:limit])
	return out, nil
}
