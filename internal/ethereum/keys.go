package ethereum

import (
	"fmt"
	"strings"
)

// normalizePrivateKeyHex strips common env-var noise and validates secp256k1 key length.
func normalizePrivateKeyHex(privateKeyHex string) (string, error) {
	key := strings.TrimSpace(privateKeyHex)
	key = strings.Trim(key, "\"'")
	key = strings.TrimPrefix(key, "0x")
	key = strings.TrimPrefix(key, "0X")

	if len(key) != 64 {
		return "", fmt.Errorf("private key must be 64 hex characters (32 bytes), got %d after normalization", len(key))
	}
	for _, r := range key {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') && (r < 'A' || r > 'F') {
			return "", fmt.Errorf("private key contains non-hex character %q", r)
		}
	}
	return key, nil
}
