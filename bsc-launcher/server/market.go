package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

type marketClient struct {
	ttl   time.Duration
	mu    sync.Mutex
	cache map[string]cacheEntry
}

func newMarketClient() *marketClient {
	return &marketClient{
		ttl:   90 * time.Second,
		cache: make(map[string]cacheEntry),
	}
}

func (c *marketClient) getCached(key string) (float64, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.cache[key]
	if !ok || time.Since(e.at) > c.ttl {
		return 0, false
	}
	v, ok := e.data.(float64)
	return v, ok
}

func (c *marketClient) setCached(key string, v float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[key] = cacheEntry{at: time.Now(), data: v}
}

func (c *marketClient) BNBUSD() (float64, error) {
	return c.NativeUSD("bsc")
}

func (c *marketClient) NativeUSD(chainSlug string) (float64, error) {
	key := "native:" + normalizeRegistryChain(chainSlug)
	if v, ok := c.getCached(key); ok {
		return v, nil
	}
	cgID := nativeCoinGeckoID(chainSlug)
	if cgID == "" {
		return 0, fmt.Errorf("no price feed for %s", chainSlug)
	}
	u := "https://api.coingecko.com/api/v3/simple/price?ids=" + cgID + "&vs_currencies=usd"
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return 0, err
	}
	client := &http.Client{Timeout: 10 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return 0, err
	}
	var out map[string]struct {
		USD float64 `json:"usd"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return 0, err
	}
	entry, ok := out[cgID]
	if !ok || entry.USD <= 0 {
		return 0, fmt.Errorf("%s price unavailable", chainSlug)
	}
	c.setCached(key, entry.USD)
	return entry.USD, nil
}

func nativeCoinGeckoID(chainSlug string) string {
	switch normalizeRegistryChain(chainSlug) {
	case "bsc":
		return "binancecoin"
	case "ethereum", "arbitrum", "optimism", "base":
		return "ethereum"
	case "polygon":
		return "matic-network"
	case "avalanche":
		return "avalanche-2"
	case "tron":
		return "tron"
	default:
		return "ethereum"
	}
}
