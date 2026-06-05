package main

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]interface{}{
		"status":  "ok",
		"service": "bsc-launcher",
		"chainId": s.cfg.ChainID,
	})
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	checks := map[string]string{"store": "ok", "rpc": "ok", "web": "ok"}
	status := http.StatusOK

	if _, err := s.store.Load(); err != nil {
		checks["store"] = err.Error()
		status = http.StatusServiceUnavailable
	}
	if _, err := s.isContract(ctx, "0x55d398326f99059fF775485246999027B3197955"); err != nil {
		checks["rpc"] = err.Error()
		status = http.StatusServiceUnavailable
	}
	if contractBytecodeHex() == "" {
		checks["web"] = "contract artifacts missing"
		status = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  map[bool]string{true: "ready", false: "degraded"}[status == http.StatusOK],
		"checks":  checks,
		"backend": s.cfg.DeployerKey != "",
		"bscscan": s.cfg.BSCScanAPIKey != "",
	})
}
