// Package config loads runtime configuration from environment variables,
// with sensible defaults for the self-contained local demo.
package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds every setting the composition root (app/main/factory and
// cmd/main.go) needs to wire the library.
type Config struct {
	ServerAddr string

	STSTokenURL     string
	STSClientID     string
	STSClientSecret string
	STSScope        string

	// RefreshThreshold: renew the token once it is within this duration of
	// expiring.
	RefreshThreshold time.Duration
	// PollInterval: how often the background loop checks whether a
	// renewal is due. Should be smaller than RefreshThreshold.
	PollInterval time.Duration
}

// Load reads configuration from the environment, falling back to demo
// defaults that make `make run` work out of the box against the bundled
// mock STS endpoint (see app/presentation/http/mock_sts_handler.go).
func Load() Config {
	addr := getEnv("SERVER_ADDR", ":8080")
	return Config{
		ServerAddr:       addr,
		STSTokenURL:      getEnv("STS_TOKEN_URL", "http://localhost"+addr+"/mock/sts/token"),
		STSClientID:      getEnv("STS_CLIENT_ID", "demo-client"),
		STSClientSecret:  getEnv("STS_CLIENT_SECRET", "demo-secret"),
		STSScope:         getEnv("STS_SCOPE", ""),
		RefreshThreshold: getEnvDuration("STS_REFRESH_THRESHOLD", 8*time.Second),
		PollInterval:     getEnvDuration("STS_POLL_INTERVAL", 2*time.Second),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	seconds, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return time.Duration(seconds) * time.Second
}
