package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	contractCache struct {
		sync.Once
		abi      string
		bytecode string
		abiArr   []map[string]interface{}
		err      error
	}
)

func loadContractArtifacts() {
	abiPath := filepath.Join(projectRoot(), "abi", "SimpleERC20.abi.json")
	binPath := filepath.Join(projectRoot(), "abi", "SimpleERC20.bin")
	abiBytes, err := os.ReadFile(abiPath)
	if err != nil {
		contractCache.err = err
		return
	}
	binBytes, err := os.ReadFile(binPath)
	if err != nil {
		contractCache.err = err
		return
	}
	contractCache.abi = string(abiBytes)
	contractCache.bytecode = strings.TrimSpace(string(binBytes))
	if err := json.Unmarshal(abiBytes, &contractCache.abiArr); err != nil {
		contractCache.err = err
	}
}

func contractABI() string {
	contractCache.Once.Do(loadContractArtifacts)
	return contractCache.abi
}

func contractBytecodeHex() string {
	contractCache.Once.Do(loadContractArtifacts)
	return contractCache.bytecode
}

func contractABIArray() ([]map[string]interface{}, error) {
	contractCache.Once.Do(loadContractArtifacts)
	return contractCache.abiArr, contractCache.err
}
