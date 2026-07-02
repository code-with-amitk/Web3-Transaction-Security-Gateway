package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds runtime configuration loaded from environment variables.
type Config struct {
	HTTPAddr string

	// Ethereum RPC endpoint (Anvil default: http://localhost:8545).
	EthRPCURL string
	// Demo signing key — NEVER use a raw env var key in production; use HSM/KMS.
	SignerPrivateKeyHex string
	ChainID             int64

	PostgresDSN string

	KafkaBrokers []string
	KafkaTopic   string

	JWTSecret     string
	JWTIssuer     string
	JWTAudience   string
	JWTExpiry     time.Duration
	EnableAuth    bool
	EnableKafka   bool
	EnablePostgres bool

	// Policy defaults (overridden by DB in a full deployment).
	SpendingLimitWei       string
	InspectThresholdWei    string
	DenylistAddresses      []string
}

func Load() (*Config, error) {
	cfg := &Config{
		HTTPAddr:            env("HTTP_ADDR", ":8080"),
		EthRPCURL:           env("ETH_RPC_URL", "http://localhost:8545"),
		SignerPrivateKeyHex: env("SIGNER_PRIVATE_KEY", "0xac0974bec39a17e36ba4a6a679b5249696125143991808455148b4754e44583"),
		ChainID:             envInt64("CHAIN_ID", 31337),
		PostgresDSN:         env("POSTGRES_DSN", "postgres://gateway:gateway@localhost:5432/gateway?sslmode=disable"),
		KafkaBrokers:        []string{env("KAFKA_BROKER", "localhost:19092")},
		KafkaTopic:          env("KAFKA_TOPIC", "gateway.audit"),
		JWTSecret:           env("JWT_SECRET", "dev-secret-change-me"),
		JWTIssuer:           env("JWT_ISSUER", "web3-gateway"),
		JWTAudience:         env("JWT_AUDIENCE", "web3-gateway-api"),
		JWTExpiry:           envDuration("JWT_EXPIRY", 24*time.Hour),
		EnableAuth:          envBool("ENABLE_AUTH", false),
		EnableKafka:         envBool("ENABLE_KAFKA", true),
		EnablePostgres:      envBool("ENABLE_POSTGRES", true),
		SpendingLimitWei:    env("SPENDING_LIMIT_WEI", "1000000000000000000"),   // 1 ETH
		InspectThresholdWei: env("INSPECT_THRESHOLD_WEI", "500000000000000000"), // 0.5 ETH
		DenylistAddresses:   envSlice("DENYLIST_ADDRESSES", []string{"0x000000000000000000000000000000000000dead"}),
	}
	return cfg, nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt64(key string, fallback int64) int64 {
	if v := os.Getenv(key); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err == nil {
			return n
		}
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		b, err := strconv.ParseBool(v)
		if err == nil {
			return b
		}
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		d, err := time.ParseDuration(v)
		if err == nil {
			return d
		}
	}
	return fallback
}

func envSlice(key string, fallback []string) []string {
	if v := os.Getenv(key); v != "" {
		return splitComma(v)
	}
	return fallback
}

func splitComma(s string) []string {
	var out []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			part := trim(s[start:i])
			if part != "" {
				out = append(out, part)
			}
			start = i + 1
		}
	}
	return out
}

func trim(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}

func (c *Config) Validate() error {
	if c.EthRPCURL == "" {
		return fmt.Errorf("ETH_RPC_URL is required")
	}
	return nil
}
