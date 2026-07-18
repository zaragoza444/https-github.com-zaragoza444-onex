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

const (
	OMNLF20OfficerAccountID = "omnl-central-bank-eur"
	OMNLF20CommonAccountID  = "omnl-euro-common-folder"

	OMNLF20StatusArrived  = "arrived"
	OMNLF20StatusReleased = "released"
	OMNLF20StatusPulled   = "pulled_back"
)

// OMNLF20FolderItem is a pending Euro folder credit located by F20/output message.
type OMNLF20FolderItem struct {
	ID                  string `json:"id"`
	F20Number           string `json:"f20Number"`
	OutputMessageNumber string `json:"outputMessageNumber"`
	CommonAccount       string `json:"commonAccount"`
	Amount              string `json:"amount"`
	Currency            string `json:"currency"`
	Status              string `json:"status"`
	NoTracer            bool   `json:"noTracer"`
	ArrivesBy           int64  `json:"arrivesBy"`
	CreatedAt           int64  `json:"createdAt"`
	ReleasedAt          int64  `json:"releasedAt,omitempty"`
	ReleasedToAccount   string `json:"releasedToAccount,omitempty"`
	ReleaseTxID         string `json:"releaseTxId,omitempty"`
	PulledAt            int64  `json:"pulledAt,omitempty"`
}

type OMNLF20FolderOrderRequest struct {
	F20Number           string `json:"f20Number"`
	OutputMessageNumber string `json:"outputMessageNumber,omitempty"`
	CommonAccount       string `json:"commonAccount,omitempty"`
	Amount              string `json:"amount"`
	Currency            string `json:"currency,omitempty"`
}

type OMNLF20FolderLocateRequest struct {
	F20Number           string `json:"f20Number"`
	OutputMessageNumber string `json:"outputMessageNumber,omitempty"`
}

type OMNLF20FolderReleaseRequest struct {
	F20Number           string `json:"f20Number"`
	OutputMessageNumber string `json:"outputMessageNumber,omitempty"`
	OfficerPIN          string `json:"officerPin"`
	ToAccount           string `json:"toAccount"`
	OfficerAccount      string `json:"officerAccount,omitempty"`
}

type OMNLF20FolderReleaseResult struct {
	Status  string                   `json:"status"`
	Item    OMNLF20FolderItem        `json:"item"`
	Deposit *OnlineBankDepositResult `json:"deposit,omitempty"`
}

type omnlF20FolderFile struct {
	Service   string              `json:"service"`
	Bank      string              `json:"bank"`
	Currency  string              `json:"currency"`
	Items     []OMNLF20FolderItem `json:"items"`
	UpdatedAt int64               `json:"updatedAt"`
}

type OMNLF20FolderStore struct {
	mu   sync.Mutex
	path string
}

func DefaultOMNLF20FolderStore() *OMNLF20FolderStore {
	return &OMNLF20FolderStore{path: filepath.Join(legacy.HomeDir(), "omnl-f20-folder.json")}
}

func (s *OMNLF20FolderStore) load() (*omnlF20FolderFile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return &omnlF20FolderFile{
				Service:  "omnl-f20-euro-payment-folder",
				Bank:     "OMNL Central Bank",
				Currency: "EUR",
				Items:    []OMNLF20FolderItem{},
			}, nil
		}
		return nil, err
	}
	var f omnlF20FolderFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	if f.Service == "" {
		f.Service = "omnl-f20-euro-payment-folder"
	}
	if f.Bank == "" {
		f.Bank = "OMNL Central Bank"
	}
	if f.Currency == "" {
		f.Currency = "EUR"
	}
	return &f, nil
}

func (s *OMNLF20FolderStore) save(f *omnlF20FolderFile) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	f.UpdatedAt = time.Now().Unix()
	raw, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, raw, 0o600)
}

func (s *OMNLF20FolderStore) Status() map[string]interface{} {
	f, err := s.load()
	if err != nil {
		return map[string]interface{}{"online": false, "error": err.Error()}
	}
	pending := 0
	for _, it := range f.Items {
		if it.Status == OMNLF20StatusArrived {
			pending++
		}
	}
	return map[string]interface{}{
		"online": true, "service": f.Service, "bank": f.Bank, "currency": f.Currency,
		"items": len(f.Items), "pending": pending, "commonAccount": OMNLF20CommonAccountID,
		"officerAccount": OMNLF20OfficerAccountID, "requiresF20": true, "tracers": false,
		"updatedAt": f.UpdatedAt,
	}
}

func (s *OMNLF20FolderStore) Order(req OMNLF20FolderOrderRequest) (*OMNLF20FolderItem, error) {
	f20 := normalizeF20(req.F20Number)
	if f20 == "" {
		return nil, fmt.Errorf("f20Number required")
	}
	output := normalizeF20(req.OutputMessageNumber)
	if output == "" {
		output = f20
	}
	if output != f20 {
		return nil, fmt.Errorf("output message number must match F20 number")
	}
	amt, err := strconv.ParseFloat(strings.TrimSpace(req.Amount), 64)
	if err != nil || amt <= 0 {
		return nil, fmt.Errorf("invalid amount")
	}
	cur := strings.ToUpper(strings.TrimSpace(req.Currency))
	if cur == "" {
		cur = "EUR"
	}
	if cur != "EUR" {
		return nil, fmt.Errorf("OMNL F20 folder accepts EUR only")
	}
	common := strings.TrimSpace(req.CommonAccount)
	if common == "" {
		common = OMNLF20CommonAccountID
	}
	f, err := s.load()
	if err != nil {
		return nil, err
	}
	for i := range f.Items {
		if normalizeF20(f.Items[i].F20Number) == f20 && f.Items[i].Status == OMNLF20StatusArrived {
			return &f.Items[i], nil
		}
	}
	now := time.Now().Unix()
	item := OMNLF20FolderItem{
		ID:        fmt.Sprintf("omnl-f20-%d", time.Now().UnixNano()),
		F20Number: f20, OutputMessageNumber: output, CommonAccount: common,
		Amount: formatFloat(amt), Currency: cur, Status: OMNLF20StatusArrived,
		NoTracer: true, ArrivesBy: now + int64(36*time.Hour/time.Second), CreatedAt: now,
	}
	f.Items = append(f.Items, item)
	if err := s.save(f); err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *OMNLF20FolderStore) Locate(req OMNLF20FolderLocateRequest) (*OMNLF20FolderItem, error) {
	f20 := normalizeF20(req.F20Number)
	if f20 == "" {
		return nil, fmt.Errorf("f20Number required")
	}
	output := normalizeF20(req.OutputMessageNumber)
	f, err := s.load()
	if err != nil {
		return nil, err
	}
	for i := range f.Items {
		it := &f.Items[i]
		if normalizeF20(it.F20Number) != f20 {
			continue
		}
		if output != "" && normalizeF20(it.OutputMessageNumber) != output {
			return nil, fmt.Errorf("output message number does not match F20 number")
		}
		return it, nil
	}
	return nil, fmt.Errorf("F20 folder item not found")
}

func (s *OMNLF20FolderStore) Release(bank *OnlineBankStore, req OMNLF20FolderReleaseRequest) (*OMNLF20FolderReleaseResult, error) {
	if bank == nil {
		return nil, fmt.Errorf("online bank required")
	}
	item, err := s.Locate(OMNLF20FolderLocateRequest{
		F20Number: req.F20Number, OutputMessageNumber: req.OutputMessageNumber,
	})
	if err != nil {
		return nil, err
	}
	if item.Status != OMNLF20StatusArrived {
		return nil, fmt.Errorf("F20 item is %s", item.Status)
	}
	officerAcct := strings.TrimSpace(req.OfficerAccount)
	if officerAcct == "" {
		officerAcct = OMNLF20OfficerAccountID
	}
	if err := bank.AuthorizeOfficerPIN(officerAcct, req.OfficerPIN); err != nil {
		return nil, err
	}
	toAccount := strings.TrimSpace(req.ToAccount)
	if toAccount == "" {
		return nil, fmt.Errorf("toAccount required")
	}
	acct, err := bank.GetAccount(toAccount)
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(acct.Currency, item.Currency) {
		return nil, fmt.Errorf("currency mismatch: folder %s → account %s", item.Currency, acct.Currency)
	}
	dep, err := bank.Deposit(OnlineBankDepositRequest{
		ToAccount: toAccount, Amount: item.Amount, Source: "omnl-f20-folder",
		Reference: "F20-" + item.F20Number,
	})
	if err != nil {
		return nil, err
	}
	f, err := s.load()
	if err != nil {
		return nil, err
	}
	var released OMNLF20FolderItem
	for i := range f.Items {
		if f.Items[i].ID == item.ID {
			f.Items[i].Status = OMNLF20StatusReleased
			f.Items[i].ReleasedAt = time.Now().Unix()
			f.Items[i].ReleasedToAccount = toAccount
			if dep.Transaction != nil {
				f.Items[i].ReleaseTxID = dep.Transaction.ID
			}
			released = f.Items[i]
			break
		}
	}
	if released.ID == "" {
		return nil, fmt.Errorf("F20 folder item not found")
	}
	if err := s.save(f); err != nil {
		return nil, err
	}
	return &OMNLF20FolderReleaseResult{Status: OMNLF20StatusReleased, Item: released, Deposit: dep}, nil
}

func normalizeF20(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	replacer := strings.NewReplacer(" ", "", "-", "", "_", "")
	return replacer.Replace(s)
}
