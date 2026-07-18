package ledger

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/onex-blockchain/onex/internal/legacy"
)

// BankOfficer is an authorized Z Bank corporate signatory (PII public fields + secret hashes).
type BankOfficer struct {
	ID                  string   `json:"id"`
	Name                string   `json:"name"`
	Position            string   `json:"position,omitempty"`
	Title               string   `json:"title,omitempty"`
	Nationality         string   `json:"nationality,omitempty"`
	PassportNumber      string   `json:"passportNumber,omitempty"`
	PassportIssued      string   `json:"passportIssued,omitempty"`
	PassportExpires     string   `json:"passportExpires,omitempty"`
	DateOfBirth         string   `json:"dateOfBirth,omitempty"`
	ResidentialAddress  string   `json:"residentialAddress,omitempty"`
	Email               string   `json:"email,omitempty"`
	Phone               string   `json:"phone,omitempty"`
	Bank                string   `json:"bank,omitempty"`
	ClientCompany       string   `json:"clientCompany,omitempty"`
	ClientRegistration  string   `json:"clientRegistration,omitempty"`
	CISDocument         string   `json:"cisDocument,omitempty"`
	LinkedAccounts      []string `json:"linkedAccounts,omitempty"`
	Status              string   `json:"status,omitempty"`
	PinSalt             string   `json:"pinSalt,omitempty"`
	PinHash             string   `json:"pinHash,omitempty"`
	SignatureHash       string   `json:"signatureHash,omitempty"`
	HasPIN              bool     `json:"hasPin"`
	HasSignature        bool     `json:"hasSignature"`
	CreatedAt           int64    `json:"createdAt,omitempty"`
	UpdatedAt           int64    `json:"updatedAt,omitempty"`
}

// BankOfficerPublic is a safe listing view (no hash/salt).
type BankOfficerPublic struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	Position           string   `json:"position,omitempty"`
	Title              string   `json:"title,omitempty"`
	Nationality        string   `json:"nationality,omitempty"`
	PassportNumber     string   `json:"passportNumber,omitempty"`
	PassportIssued     string   `json:"passportIssued,omitempty"`
	PassportExpires    string   `json:"passportExpires,omitempty"`
	DateOfBirth        string   `json:"dateOfBirth,omitempty"`
	ResidentialAddress string   `json:"residentialAddress,omitempty"`
	Email              string   `json:"email,omitempty"`
	Phone              string   `json:"phone,omitempty"`
	Bank               string   `json:"bank,omitempty"`
	ClientCompany      string   `json:"clientCompany,omitempty"`
	ClientRegistration string   `json:"clientRegistration,omitempty"`
	CISDocument        string   `json:"cisDocument,omitempty"`
	LinkedAccounts     []string `json:"linkedAccounts,omitempty"`
	Status             string   `json:"status,omitempty"`
	HasPIN             bool     `json:"hasPin"`
	HasSignature       bool     `json:"hasSignature"`
	CreatedAt          int64    `json:"createdAt,omitempty"`
	UpdatedAt          int64    `json:"updatedAt,omitempty"`
}

type bankOfficerFile struct {
	Client   *bankOfficerClient `json:"client,omitempty"`
	Officers []BankOfficer      `json:"officers"`
	Updated  int64              `json:"updatedAt,omitempty"`
}

type bankOfficerClient struct {
	CompanyName             string `json:"companyName"`
	CountryOfIncorporation  string `json:"countryOfIncorporation,omitempty"`
	CorporateRegistration   string `json:"corporateRegistration,omitempty"`
	DateOfRegistration      string `json:"dateOfRegistration,omitempty"`
	CompanyAddress          string `json:"companyAddress,omitempty"`
	CISDocument             string `json:"cisDocument,omitempty"`
	CISDocumentID           string `json:"cisDocumentId,omitempty"`
}

// Officer seed JSON shape (plaintext pin/signature only via env at seed time).
type bankOfficerSeedFile struct {
	Client   *bankOfficerClient  `json:"client"`
	Officers []bankOfficerSeedRow `json:"officers"`
}

type bankOfficerSeedRow struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	Position           string   `json:"position"`
	Title              string   `json:"title"`
	Nationality        string   `json:"nationality"`
	PassportNumber     string   `json:"passportNumber"`
	PassportIssued     string   `json:"passportIssued"`
	PassportExpires    string   `json:"passportExpires"`
	DateOfBirth        string   `json:"dateOfBirth"`
	ResidentialAddress string   `json:"residentialAddress"`
	Email              string   `json:"email"`
	Phone              string   `json:"phone"`
	Bank               string   `json:"bank"`
	Status             string   `json:"status"`
	LinkedAccounts     []string `json:"linkedAccounts"`
}

type OfficerAuthRequest struct {
	OfficerID string `json:"officerId"`
	PIN       string `json:"pin"`
	Signature string `json:"signature"`
}

type OfficerAuthResult struct {
	Valid      bool   `json:"valid"`
	OfficerID  string `json:"officerId,omitempty"`
	Name       string `json:"name,omitempty"`
	Position   string `json:"position,omitempty"`
	ApprovalID string `json:"approvalId,omitempty"`
	Error      string `json:"error,omitempty"`
}

type OfficerTransferRequest struct {
	OfficerID   string `json:"officerId"`
	PIN         string `json:"pin"`
	Signature   string `json:"signature"`
	FromAccount string `json:"fromAccount"`
	ToAccount   string `json:"toAccount,omitempty"`
	Amount      string `json:"amount"`
	Rail        string `json:"rail,omitempty"`
	ToBank      string `json:"toBank,omitempty"`
	ToIBAN      string `json:"toIban,omitempty"`
	Reference   string `json:"reference,omitempty"`
	Preview     bool   `json:"preview,omitempty"`
}

type OfficerTransferResult struct {
	Status      string                 `json:"status"`
	ApprovalID  string                 `json:"approvalId,omitempty"`
	OfficerID   string                 `json:"officerId,omitempty"`
	OfficerName string                 `json:"officerName,omitempty"`
	Preview     bool                   `json:"preview,omitempty"`
	Transfer    *OnlineBankTransferResult `json:"transfer,omitempty"`
	Error       string                 `json:"error,omitempty"`
}

// BankOfficerStore persists Z Bank corporate officers.
type BankOfficerStore struct {
	mu   sync.Mutex
	path string
}

func DefaultBankOfficerStore() *BankOfficerStore {
	return &BankOfficerStore{path: filepath.Join(legacy.HomeDir(), "zbank-officers.json")}
}

func BankOfficerSeedFile() string {
	v := strings.TrimSpace(legacy.EnvOrLegacy("ONEX_ZBANK_OFFICERS_FILE", "ONEX_ZBANK_OFFICERS_FILE"))
	if v != "" {
		return v
	}
	return "configs/zbank-officers.dssboat.example.json"
}

func (s *BankOfficerStore) load() (*bankOfficerFile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return &bankOfficerFile{Officers: []BankOfficer{}}, nil
		}
		return nil, err
	}
	var f bankOfficerFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	if f.Officers == nil {
		f.Officers = []BankOfficer{}
	}
	return &f, nil
}

func (s *BankOfficerStore) save(f *bankOfficerFile) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	f.Updated = time.Now().Unix()
	raw, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, raw, 0o600)
}

func (s *BankOfficerStore) Status() map[string]interface{} {
	f, err := s.load()
	if err != nil {
		return map[string]interface{}{"ready": false, "error": err.Error()}
	}
	active := 0
	credReady := 0
	for _, o := range f.Officers {
		if strings.EqualFold(o.Status, "active") {
			active++
		}
		if o.HasPIN && o.HasSignature {
			credReady++
		}
	}
	clientName := ""
	if f.Client != nil {
		clientName = f.Client.CompanyName
	}
	prod := LoadConfig().Production()
	return map[string]interface{}{
		"ready":                 true,
		"production":            prod,
		"count":                 len(f.Officers),
		"active":                active,
		"credentialsReady":      credReady,
		"productionReady":       prod && credReady > 0 && active > 0,
		"client":                clientName,
		"seedFile":              BankOfficerSeedFile(),
		"authFactors":           []string{"pin", "signature"},
		"requiresEnvSecrets":    []string{"ONEX_ZBANK_OFFICER_PIN", "ONEX_ZBANK_OFFICER_SIGNATURE"},
		"endpoints": map[string]string{
			"status":      "/bridge/bank/officer/status",
			"list":        "/bridge/bank/officer/list",
			"get":         "/bridge/bank/officer",
			"verify":      "/bridge/bank/officer/verify",
			"transfer":    "/bridge/bank/officer/transfer",
			"ensure":      "/bridge/bank/officer/ensure",
			"credentials": "/bridge/bank/officer/credentials",
		},
	}
}

func (o BankOfficer) Public() BankOfficerPublic {
	return BankOfficerPublic{
		ID: o.ID, Name: o.Name, Position: o.Position, Title: o.Title,
		Nationality: o.Nationality, PassportNumber: o.PassportNumber,
		PassportIssued: o.PassportIssued, PassportExpires: o.PassportExpires,
		DateOfBirth: o.DateOfBirth, ResidentialAddress: o.ResidentialAddress,
		Email: o.Email, Phone: o.Phone, Bank: o.Bank,
		ClientCompany: o.ClientCompany, ClientRegistration: o.ClientRegistration,
		CISDocument: o.CISDocument, LinkedAccounts: append([]string{}, o.LinkedAccounts...),
		Status: o.Status, HasPIN: o.HasPIN, HasSignature: o.HasSignature,
		CreatedAt: o.CreatedAt, UpdatedAt: o.UpdatedAt,
	}
}

func (s *BankOfficerStore) List() ([]BankOfficerPublic, error) {
	f, err := s.load()
	if err != nil {
		return nil, err
	}
	out := make([]BankOfficerPublic, 0, len(f.Officers))
	for _, o := range f.Officers {
		out = append(out, o.Public())
	}
	return out, nil
}

func (s *BankOfficerStore) Get(id string) (*BankOfficerPublic, error) {
	f, err := s.load()
	if err != nil {
		return nil, err
	}
	id = strings.TrimSpace(id)
	for _, o := range f.Officers {
		if o.ID == id {
			p := o.Public()
			return &p, nil
		}
	}
	return nil, fmt.Errorf("officer not found")
}

// EnsureSeeded loads DSSBOaT (or configured) officer seed when the store is empty.
func (s *BankOfficerStore) EnsureSeeded(seedPath string) error {
	f, err := s.load()
	if err != nil {
		return err
	}
	if len(f.Officers) > 0 {
		return nil
	}
	path := strings.TrimSpace(seedPath)
	if path == "" {
		path = BankOfficerSeedFile()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var seed bankOfficerSeedFile
	if err := json.Unmarshal(data, &seed); err != nil {
		return err
	}
	pin, sig, ok := OfficerSecretsFromEnv()
	if !ok {
		// Never embed demo credentials. Seed only when production secrets are present.
		return nil
	}
	now := time.Now().Unix()
	f.Client = seed.Client
	for _, row := range seed.Officers {
		id := strings.TrimSpace(row.ID)
		if id == "" {
			continue
		}
		salt := officerSalt(id)
		o := BankOfficer{
			ID: id, Name: row.Name, Position: row.Position, Title: row.Title,
			Nationality: row.Nationality, PassportNumber: row.PassportNumber,
			PassportIssued: row.PassportIssued, PassportExpires: row.PassportExpires,
			DateOfBirth: row.DateOfBirth, ResidentialAddress: row.ResidentialAddress,
			Email: row.Email, Phone: row.Phone, Bank: row.Bank,
			LinkedAccounts: append([]string{}, row.LinkedAccounts...),
			Status: row.Status, PinSalt: salt,
			CreatedAt: now, UpdatedAt: now,
		}
		if o.Bank == "" {
			o.Bank = "zbank"
		}
		if o.Status == "" {
			o.Status = "active"
		}
		if seed.Client != nil {
			o.ClientCompany = seed.Client.CompanyName
			o.ClientRegistration = seed.Client.CorporateRegistration
			o.CISDocument = seed.Client.CISDocument
		}
		if err := setOfficerCredentials(&o, pin, sig); err != nil {
			return err
		}
		f.Officers = append(f.Officers, o)
	}
	return s.save(f)
}

func setOfficerCredentials(o *BankOfficer, pin, signature string) error {
	pin = strings.TrimSpace(pin)
	signature = strings.TrimSpace(signature)
	if err := validateOfficerPIN(pin); err != nil {
		return err
	}
	if err := validateOfficerSignature(signature); err != nil {
		return err
	}
	if o.PinSalt == "" {
		o.PinSalt = officerSalt(o.ID)
	}
	o.PinHash = hashOfficerPIN(pin, o.PinSalt)
	o.SignatureHash = hashOfficerSignature(signature, o.PinSalt)
	o.HasPIN = true
	o.HasSignature = true
	o.UpdatedAt = time.Now().Unix()
	return nil
}

func (s *BankOfficerStore) Verify(req OfficerAuthRequest) (*OfficerAuthResult, error) {
	f, err := s.load()
	if err != nil {
		return nil, err
	}
	o, ok, resolveErr := resolveOfficer(f, req.OfficerID)
	if !ok {
		return &OfficerAuthResult{Valid: false, Error: resolveErr}, nil
	}
	if !strings.EqualFold(o.Status, "active") {
		return &OfficerAuthResult{Valid: false, OfficerID: o.ID, Error: "officer inactive"}, nil
	}
	if err := verifyOfficerFactors(o, req.PIN, req.Signature); err != nil {
		return &OfficerAuthResult{Valid: false, OfficerID: o.ID, Error: err.Error()}, nil
	}
	approval := officerApprovalID(o.ID, req.PIN, req.Signature)
	return &OfficerAuthResult{
		Valid: true, OfficerID: o.ID, Name: o.Name, Position: o.Position, ApprovalID: approval,
	}, nil
}

func (s *BankOfficerStore) AuthorizeTransfer(req OfficerTransferRequest, bank *OnlineBankStore) (*OfficerTransferResult, error) {
	auth, err := s.Verify(OfficerAuthRequest{OfficerID: req.OfficerID, PIN: req.PIN, Signature: req.Signature})
	if err != nil {
		return nil, err
	}
	if !auth.Valid {
		return &OfficerTransferResult{Status: "denied", Error: auth.Error, OfficerID: req.OfficerID}, nil
	}
	f, err := s.load()
	if err != nil {
		return nil, err
	}
	o, ok, resolveErr := resolveOfficer(f, req.OfficerID)
	if !ok {
		return &OfficerTransferResult{Status: "denied", Error: resolveErr, OfficerID: req.OfficerID}, nil
	}
	from := strings.TrimSpace(req.FromAccount)
	if from == "" {
		return nil, fmt.Errorf("fromAccount required")
	}
	if !officerMayOperate(o, from) {
		return &OfficerTransferResult{
			Status: "denied", OfficerID: o.ID, OfficerName: o.Name,
			Error: "officer not authorized for account " + from,
		}, nil
	}
	xferReq := OnlineBankTransferRequest{
		FromAccount: from, ToAccount: req.ToAccount, Amount: req.Amount,
		Rail: req.Rail, ToBank: req.ToBank, ToIBAN: req.ToIBAN,
		Reference: req.Reference, Preview: req.Preview,
	}
	if strings.TrimSpace(xferReq.Reference) == "" {
		xferReq.Reference = "officer:" + o.ID + ":" + auth.ApprovalID
	}
	res, err := bank.Send(xferReq)
	if err != nil {
		return nil, err
	}
	return &OfficerTransferResult{
		Status: "authorized", ApprovalID: auth.ApprovalID,
		OfficerID: o.ID, OfficerName: o.Name, Preview: req.Preview, Transfer: res,
	}, nil
}

func findOfficer(f *bankOfficerFile, id string) (BankOfficer, bool) {
	id = strings.TrimSpace(id)
	for _, o := range f.Officers {
		if o.ID == id {
			return o, true
		}
	}
	return BankOfficer{}, false
}

// resolveOfficer finds by id, or defaults to the sole active officer when id is empty.
func resolveOfficer(f *bankOfficerFile, id string) (BankOfficer, bool, string) {
	id = strings.TrimSpace(id)
	if id != "" {
		o, ok := findOfficer(f, id)
		return o, ok, ""
	}
	var active []BankOfficer
	for _, o := range f.Officers {
		if strings.EqualFold(o.Status, "active") {
			active = append(active, o)
		}
	}
	if len(active) == 1 {
		return active[0], true, ""
	}
	if len(active) == 0 {
		return BankOfficer{}, false, "officer not found"
	}
	return BankOfficer{}, false, "officerId required"
}

func officerMayOperate(o BankOfficer, accountID string) bool {
	if len(o.LinkedAccounts) == 0 {
		return strings.HasPrefix(strings.ToLower(accountID), "zbank-")
	}
	for _, a := range o.LinkedAccounts {
		if a == accountID {
			return true
		}
	}
	return false
}

func verifyOfficerFactors(o BankOfficer, pin, signature string) error {
	if !o.HasPIN || o.PinHash == "" {
		return fmt.Errorf("officer pin not configured")
	}
	if !o.HasSignature || o.SignatureHash == "" {
		return fmt.Errorf("officer signature not configured")
	}
	if err := validateOfficerPIN(pin); err != nil {
		return fmt.Errorf("invalid pin")
	}
	if err := validateOfficerSignature(signature); err != nil {
		return fmt.Errorf("invalid signature")
	}
	if hashOfficerPIN(pin, o.PinSalt) != o.PinHash {
		return fmt.Errorf("invalid pin")
	}
	if hashOfficerSignature(signature, o.PinSalt) != o.SignatureHash {
		return fmt.Errorf("invalid signature")
	}
	return nil
}

func validateOfficerPIN(pin string) error {
	pin = strings.TrimSpace(pin)
	if len(pin) < 4 || len(pin) > 8 {
		return fmt.Errorf("pin must be 4-8 digits")
	}
	for _, r := range pin {
		if !unicode.IsDigit(r) {
			return fmt.Errorf("pin must be 4-8 digits")
		}
	}
	return nil
}

func validateOfficerSignature(sig string) error {
	sig = strings.TrimSpace(sig)
	if len(sig) < 8 {
		return fmt.Errorf("signature must be at least 8 characters")
	}
	return nil
}

func normalizeOfficerSignature(sig string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(sig)), " ")
}

func officerSalt(id string) string {
	sum := sha256.Sum256([]byte("onex-zbank-officer-salt:" + strings.TrimSpace(id)))
	return hex.EncodeToString(sum[:8])
}

func hashOfficerPIN(pin, salt string) string {
	sum := sha256.Sum256([]byte("onex-zbank-officer-pin:" + strings.TrimSpace(pin) + ":" + salt))
	return hex.EncodeToString(sum[:])
}

func hashOfficerSignature(sig, salt string) string {
	sum := sha256.Sum256([]byte("onex-zbank-officer-sig:" + normalizeOfficerSignature(sig) + ":" + salt))
	return hex.EncodeToString(sum[:])
}

func officerApprovalID(officerID, pin, signature string) string {
	mac := hmac.New(sha256.New, []byte("onex-zbank-officer-approval"))
	_, _ = mac.Write([]byte(officerID + "|" + pin + "|" + normalizeOfficerSignature(signature) + "|" + fmt.Sprintf("%d", time.Now().Unix()/60)))
	return "oa-" + hex.EncodeToString(mac.Sum(nil)[:10])
}

// OfficerCredentialsRequest sets or rotates PIN + signature for an officer (production).
type OfficerCredentialsRequest struct {
	OfficerID       string `json:"officerId"`
	PIN             string `json:"pin"`
	Signature       string `json:"signature"`
	CurrentPIN      string `json:"currentPin,omitempty"`
	CurrentSignature string `json:"currentSignature,omitempty"`
}

func OfficerSecretsFromEnv() (pin, signature string, ok bool) {
	pin = strings.TrimSpace(legacy.EnvOrLegacy("ONEX_ZBANK_OFFICER_PIN", "ONEX_ZBANK_OFFICER_PIN"))
	signature = strings.TrimSpace(legacy.EnvOrLegacy("ONEX_ZBANK_OFFICER_SIGNATURE", "ONEX_ZBANK_OFFICER_SIGNATURE"))
	if strings.HasPrefix(strings.ToUpper(pin), "CHANGE_ME") {
		pin = ""
	}
	if strings.HasPrefix(strings.ToUpper(signature), "CHANGE_ME") {
		signature = ""
	}
	ok = pin != "" && signature != ""
	return pin, signature, ok
}

func RequireProductionOfficerSecrets() error {
	if !LoadConfig().Production() {
		return nil
	}
	if _, _, ok := OfficerSecretsFromEnv(); !ok {
		return fmt.Errorf("production requires ONEX_ZBANK_OFFICER_PIN and ONEX_ZBANK_OFFICER_SIGNATURE — no demo defaults")
	}
	return nil
}

// SetCredentials sets PIN + signature. First-time setup needs no current factors;
// rotation requires valid current PIN + signature.
func (s *BankOfficerStore) SetCredentials(req OfficerCredentialsRequest) (*BankOfficerPublic, error) {
	f, err := s.load()
	if err != nil {
		return nil, err
	}
	idx := -1
	for i := range f.Officers {
		if f.Officers[i].ID == strings.TrimSpace(req.OfficerID) {
			idx = i
			break
		}
	}
	if idx < 0 {
		return nil, fmt.Errorf("officer not found")
	}
	o := &f.Officers[idx]
	if o.HasPIN && o.HasSignature {
		if err := verifyOfficerFactors(*o, req.CurrentPIN, req.CurrentSignature); err != nil {
			return nil, fmt.Errorf("current credentials invalid")
		}
	}
	if err := setOfficerCredentials(o, req.PIN, req.Signature); err != nil {
		return nil, err
	}
	if err := s.save(f); err != nil {
		return nil, err
	}
	p := o.Public()
	return &p, nil
}

// SeedFromEnv forces officer seed using required env secrets (production bootstrap).
func (s *BankOfficerStore) SeedFromEnv(seedPath string) error {
	if _, _, ok := OfficerSecretsFromEnv(); !ok {
		return fmt.Errorf("ONEX_ZBANK_OFFICER_PIN and ONEX_ZBANK_OFFICER_SIGNATURE required")
	}
	f, err := s.load()
	if err != nil {
		return err
	}
	if len(f.Officers) > 0 {
		return s.EnsureSeeded(seedPath) // already seeded
	}
	return s.EnsureSeeded(seedPath)
}
