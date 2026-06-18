package ledger

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ParseAnyLedger normalizes bank, crypto, exchange, or wallet export JSON into entries.
func ParseAnyLedger(data []byte) ([]Entry, error) {
	if rows, err := parseImportFile(data); err == nil && len(rows) > 0 {
		return rows, nil
	}
	if rows, err := parseBankJSON(data); err == nil && len(rows) > 0 {
		return rows, nil
	}
	if rows, err := parseBalanceMap(data); err == nil && len(rows) > 0 {
		return rows, nil
	}
	if rows, err := parseRowArray(data); err == nil && len(rows) > 0 {
		return rows, nil
	}
	return nil, fmt.Errorf("unrecognized ledger format")
}

func parseBalanceMap(data []byte) ([]Entry, error) {
	var wrap struct {
		Balances map[string]string `json:"balances"`
	}
	if err := json.Unmarshal(data, &wrap); err != nil || len(wrap.Balances) == 0 {
		var flat map[string]string
		if err2 := json.Unmarshal(data, &flat); err2 != nil || len(flat) == 0 {
			return nil, fmt.Errorf("not a balance map")
		}
		wrap.Balances = flat
	}
	return mapToEntries(wrap.Balances, SourceImport), nil
}

func parseRowArray(data []byte) ([]Entry, error) {
	var rows []map[string]interface{}
	if err := json.Unmarshal(data, &rows); err != nil {
		return nil, err
	}
	var out []Entry
	for i, row := range rows {
		asset := pickString(row, "asset", "symbol", "currency", "token")
		amt := pickString(row, "amount", "balance", "value", "quantity")
		if asset == "" || amt == "" {
			continue
		}
		out = append(out, Entry{
			ID:      importStableID(asset, pickString(row, "account", "address", "wallet", "name"), i),
			Source:  SourceImport,
			Mode:    modeForAsset(asset),
			Asset:   strings.ToUpper(asset),
			Human:   amt,
			Account: pickString(row, "account", "address", "wallet", "name"),
		})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("empty row array")
	}
	return out, nil
}

func mapToEntries(m map[string]string, src SourceKind) []Entry {
	var out []Entry
	i := 0
	for asset, amt := range m {
		asset = strings.ToUpper(strings.TrimSpace(asset))
		amt = strings.TrimSpace(amt)
		if asset == "" || amt == "" || amt == "0" {
			continue
		}
		out = append(out, Entry{
			ID:     importStableID(asset, "", i),
			Source: src,
			Mode:   modeForAsset(asset),
			Asset:  asset,
			Human:  amt,
		})
		i++
	}
	return out
}

func pickString(m map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			switch t := v.(type) {
			case string:
				return strings.TrimSpace(t)
			case float64:
				return formatFloat(t)
			}
		}
	}
	return ""
}

func modeForAsset(asset string) Mode {
	if isFiat(asset) {
		return ModeFiat
	}
	return ModeReal
}

func importStableID(asset, account string, rowIndex int) string {
	asset = strings.ToLower(strings.TrimSpace(asset))
	if account = strings.TrimSpace(account); account != "" {
		return fmt.Sprintf("import-%s-%s", asset, strings.ToLower(account))
	}
	if asset != "" {
		return "import-" + asset
	}
	return fmt.Sprintf("import-row-%d", rowIndex)
}
