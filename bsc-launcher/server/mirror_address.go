package main

import (
	"encoding/json"
	"os"
	"strings"
)

func (s *Server) loadMirrorResultCanonical() string {
	_, mirrorPath, _ := s.flashMirrorPaths()
	raw, err := os.ReadFile(mirrorPath)
	if err != nil {
		return ""
	}
	var result struct {
		Steps []struct {
			Result struct {
				Wrapped *struct {
					ContractAddress string `json:"contractAddress"`
				} `json:"wrapped"`
			} `json:"result"`
		} `json:"steps"`
	}
	if json.Unmarshal(raw, &result) != nil {
		return ""
	}
	for _, step := range result.Steps {
		if step.Result.Wrapped == nil {
			continue
		}
		addr := strings.TrimSpace(step.Result.Wrapped.ContractAddress)
		if addr != "" && strings.HasPrefix(addr, "0x") {
			return addr
		}
	}
	return ""
}

func (s *Server) ensureMirrorContractAddresses(book *flashMirrorBook) {
	if book == nil {
		return
	}
	canonical := strings.TrimSpace(book.CanonicalAddress)
	if canonical == "" {
		for _, d := range book.Deployments {
			if addr := strings.TrimSpace(d.ContractAddress); addr != "" {
				canonical = addr
				break
			}
			if addr := strings.TrimSpace(d.PredictedAddress); addr != "" {
				canonical = addr
				break
			}
		}
	}
	if canonical == "" {
		canonical = s.loadMirrorResultCanonical()
	}
	if canonical == "" {
		return
	}
	book.CanonicalAddress = canonical
	for i := range book.Deployments {
		d := &book.Deployments[i]
		if strings.TrimSpace(d.ContractAddress) == "" {
			d.ContractAddress = canonical
		}
		if strings.TrimSpace(d.PredictedAddress) == "" {
			d.PredictedAddress = canonical
		}
		exp := strings.TrimSuffix(strings.TrimSpace(d.Explorer), "/")
		if exp != "" && d.ExplorerTokenURL == "" {
			d.ExplorerTokenURL = explorerTokenURL(exp, canonical)
		}
	}
}
