package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type flashMirrorDeployment struct {
	ChainID          string `json:"chainId"`
	ChainName        string `json:"chainName"`
	Symbol           string `json:"symbol"`
	ContractAddress  string `json:"contractAddress"`
	PredictedAddress string `json:"predictedAddress,omitempty"`
	TxHash           string `json:"txHash,omitempty"`
	Explorer         string `json:"explorer"`
	SupplyHuman      string `json:"supplyHuman"`
	Status           string `json:"status"`
	VerifiedOnChain  bool   `json:"verifiedOnChain"`
	DeployedAt       string `json:"deployedAt,omitempty"`
}

type flashMirrorBook struct {
	Name        string                  `json:"name"`
	OriginToken string                  `json:"originToken"`
	Symbol      string                  `json:"symbol"`
	UpdatedAt   string                  `json:"updatedAt"`
	Deployments []flashMirrorDeployment `json:"deployments"`
}

type chainMeta struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Explorer string `json:"explorer"`
}

func flashMirrorPaths() (livePath, mirrorPath, configPath string) {
	root := repoRoot()
	return filepath.Join(root, "configs", "flash-coin-live-addresses.json"),
		filepath.Join(root, "configs", "flash-coin-mirror-result.json"),
		filepath.Join(root, "configs", "flash-coin-mirror.json")
}

func repoRoot() string {
	if v := strings.TrimSpace(os.Getenv("BSC_LAUNCHER_ROOT")); v != "" {
		return filepath.Dir(v)
	}
	wd, _ := os.Getwd()
	if filepath.Base(wd) == "server" {
		return filepath.Dir(wd)
	}
	if fileExists(filepath.Join(wd, "bsc-launcher", "web", "index.html")) {
		return wd
	}
	if fileExists(filepath.Join(wd, "configs", "chains.json")) {
		return wd
	}
	return wd
}

func loadChainMeta() map[string]chainMeta {
	root := repoRoot()
	raw, err := os.ReadFile(filepath.Join(root, "configs", "chains.json"))
	if err != nil {
		return nil
	}
	var list []chainMeta
	if json.Unmarshal(raw, &list) != nil {
		return nil
	}
	out := make(map[string]chainMeta, len(list))
	for _, c := range list {
		out[c.ID] = c
	}
	return out
}

func loadWrapSupplyHuman() string {
	_, _, cfgPath := flashMirrorPaths()
	raw, err := os.ReadFile(cfgPath)
	if err != nil {
		return "100"
	}
	var cfg struct {
		WrapAmountPerChain string `json:"wrapAmountPerChain"`
	}
	if json.Unmarshal(raw, &cfg) != nil || cfg.WrapAmountPerChain == "" {
		return "100"
	}
	return cfg.WrapAmountPerChain
}

func loadFlashMirrorBook() flashMirrorBook {
	livePath, mirrorPath, _ := flashMirrorPaths()
	chains := loadChainMeta()
	supplyHuman := loadWrapSupplyHuman()

	book := flashMirrorBook{Name: "Flash Coin", Symbol: "wFLASH", OriginToken: "FLASH", Deployments: []flashMirrorDeployment{}}
	byChain := map[string]flashMirrorDeployment{}

	if raw, err := os.ReadFile(mirrorPath); err == nil {
		book = mirrorResultToBook(raw, chains, supplyHuman)
		for _, d := range book.Deployments {
			byChain[d.ChainID] = d
		}
	}

	if raw, err := os.ReadFile(livePath); err == nil {
		var live flashMirrorBook
		if json.Unmarshal(raw, &live) == nil {
			if live.Name != "" {
				book.Name = live.Name
			}
			if live.Symbol != "" {
				book.Symbol = live.Symbol
			}
			if live.OriginToken != "" {
				book.OriginToken = live.OriginToken
			}
			if live.UpdatedAt != "" {
				book.UpdatedAt = live.UpdatedAt
			}
			for _, d := range live.Deployments {
				if d.ChainID == "" {
					continue
				}
				if meta, ok := chains[d.ChainID]; ok {
					if d.ChainName == "" {
						d.ChainName = meta.Name
					}
					if d.Explorer == "" {
						d.Explorer = meta.Explorer
					}
				}
				if d.SupplyHuman == "" {
					d.SupplyHuman = supplyHuman
				}
				if pred, ok := byChain[d.ChainID]; ok && d.PredictedAddress == "" {
					d.PredictedAddress = pred.PredictedAddress
				}
				byChain[d.ChainID] = d
			}
		}
	}

	book.Deployments = make([]flashMirrorDeployment, 0, len(byChain))
	order := mirrorChainOrder()
	if len(order) == 0 {
		for _, d := range byChain {
			book.Deployments = append(book.Deployments, d)
		}
		return book
	}
	for _, id := range order {
		if d, ok := byChain[id]; ok {
			book.Deployments = append(book.Deployments, d)
		}
	}
	for id, d := range byChain {
		found := false
		for _, o := range order {
			if o == id {
				found = true
				break
			}
		}
		if !found {
			book.Deployments = append(book.Deployments, d)
		}
	}
	return book
}

func mirrorChainOrder() []string {
	_, _, cfgPath := flashMirrorPaths()
	raw, err := os.ReadFile(cfgPath)
	if err != nil {
		return nil
	}
	var cfg struct {
		MirrorChains []string `json:"mirrorChains"`
	}
	if json.Unmarshal(raw, &cfg) != nil {
		return nil
	}
	return cfg.MirrorChains
}

func mirrorResultToBook(raw []byte, chains map[string]chainMeta, supplyHuman string) flashMirrorBook {
	var result struct {
		Name    string `json:"name"`
		Symbol  string `json:"symbol"`
		TokenID string `json:"tokenId"`
		Steps   []struct {
			Chain  string `json:"chain"`
			Result struct {
				Wrapped *struct {
					ChainID         string `json:"chainId"`
					Symbol          string `json:"symbol"`
					ContractAddress string `json:"contractAddress"`
					DeployPayload   struct {
						Explorer string `json:"explorer"`
					} `json:"deployPayload"`
				} `json:"wrapped"`
			} `json:"result"`
		} `json:"steps"`
	}
	book := flashMirrorBook{Name: "Flash Coin", Symbol: "wFLASH", OriginToken: "FLASH", Deployments: []flashMirrorDeployment{}}
	if err := json.Unmarshal(raw, &result); err != nil {
		return book
	}
	if result.Name != "" {
		book.Name = result.Name
	}
	if result.Symbol != "" {
		book.OriginToken = result.Symbol
		book.Symbol = "w" + result.Symbol
	}
	for _, step := range result.Steps {
		if step.Chain == "onex-mainnet-1" || step.Result.Wrapped == nil {
			continue
		}
		w := step.Result.Wrapped
		chainName := step.Chain
		explorer := w.DeployPayload.Explorer
		if meta, ok := chains[w.ChainID]; ok {
			chainName = meta.Name
			if explorer == "" {
				explorer = meta.Explorer
			}
		}
		dep := flashMirrorDeployment{
			ChainID:          w.ChainID,
			ChainName:        chainName,
			Symbol:           w.Symbol,
			ContractAddress:  w.ContractAddress,
			PredictedAddress: w.ContractAddress,
			Explorer:         explorer,
			SupplyHuman:      supplyHuman,
			Status:           "predicted",
			VerifiedOnChain:  false,
		}
		book.Deployments = append(book.Deployments, dep)
	}
	return book
}

func (s *Server) handleFlashMirror(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	book := loadFlashMirrorBook()
	writeJSON(w, book)
}
