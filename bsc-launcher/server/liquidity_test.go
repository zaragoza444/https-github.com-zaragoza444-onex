package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLiquidityQuote1BUsdtBSC(t *testing.T) {
	srv := NewServer(LoadConfig())
	req := httptest.NewRequest(http.MethodGet, "/api/liquidity/quote?tokenAmount=1000000000&targetUsd=1&quote=usdt&chain=bsc", nil)
	rec := httptest.NewRecorder()
	srv.handleLiquidityQuote(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out["quoteSymbol"] != "USDT" {
		t.Fatalf("quoteSymbol: %v", out["quoteSymbol"])
	}
	cap, ok := out["marketCapUsd"].(float64)
	if !ok || cap != 1e9 {
		t.Fatalf("marketCapUsd: %v", out["marketCapUsd"])
	}
	qamt, _ := out["quoteAmount"].(string)
	if qamt == "" || qamt == "0" {
		t.Fatalf("quoteAmount empty: %q", qamt)
	}
}
