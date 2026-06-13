package chains

import (
	"fmt"
	"os"
	"strings"
)

const userDeployerAddress = "0x05868c29D58d1EC275Cf078356c03F79B1975600"

func NormalizeHex(v string) string {
	return strings.TrimPrefix(strings.TrimSpace(v), "0x")
}

func IsPrivateKeyHex(v string) bool {
	h := NormalizeHex(v)
	if len(h) != 64 {
		return false
	}
	for _, c := range h {
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') {
			continue
		}
		return false
	}
	return true
}

func IsAddressHex(v string) bool {
	h := NormalizeHex(v)
	if len(h) != 40 {
		return false
	}
	for _, c := range h {
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') {
			continue
		}
		return false
	}
	return true
}

func FormatAddress(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(v), "0x") {
		return "0x" + NormalizeHex(v)
	}
	return "0x" + NormalizeHex(v)
}

func LoadDeployerAddress() string {
	for _, k := range []string{"FLASH_DEPLOYER_ADDRESS", "BSC_DEPLOYER_ADDRESS"} {
		if v := FormatAddress(os.Getenv(k)); IsAddressHex(v) {
			return v
		}
	}
	return userDeployerAddress
}

func LoadDeployerKey() (string, error) {
	for _, k := range []string{"FLASH_DEPLOYER_PRIVATE_KEY", "BSC_DEPLOYER_PRIVATE_KEY"} {
		v := strings.TrimSpace(os.Getenv(k))
		if v == "" || strings.Contains(v, "...") || strings.Contains(strings.ToUpper(v), "YOUR") {
			continue
		}
		if IsAddressHex(v) {
			return "", fmt.Errorf("%s is a wallet address, not a private key — use FLASH_DEPLOYER_ADDRESS for the address and paste the 64-char private key separately", k)
		}
		if !IsPrivateKeyHex(v) {
			continue
		}
		return strings.TrimPrefix(v, "0x"), nil
	}
	return "", fmt.Errorf("set FLASH_DEPLOYER_PRIVATE_KEY in bsc-launcher/.env (64 hex chars), or use MetaMask in Token Lab → Liquidity")
}
