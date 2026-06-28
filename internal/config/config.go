// Package config loads application configuration from the environment.
// Values may be supplied via a .env file (loaded automatically) or via real
// environment variables, which always take precedence.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all runtime configuration.
type Config struct {
	CheckInterval time.Duration
	HTTPTimeout   time.Duration
	DBPath        string
	Debug         bool
	// PriceMax caps the accepted offer price in whole euros. 0 = no limit.
	// Offers with a known price above this are ignored.
	PriceMax int

	PushoverToken    string
	PushoverUser     string
	PushoverPriority int
	PushoverDevice   string

	BraucheKlimaEnabled bool
	BraucheKlimaURL     string
	BraucheKlimaProduct string

	ObiEnabled    bool
	ObiProductID  string
	ObiPostalCode string
}

// Load reads configuration from a local .env file (if present) and the
// environment, applies defaults, and validates required fields.
func Load() (*Config, error) {
	// Ignore the error: a missing .env in production is perfectly fine when
	// all values are provided via real environment variables.
	_ = godotenv.Load()

	cfg := &Config{
		CheckInterval: envDuration("CHECK_INTERVAL", 5*time.Minute),
		HTTPTimeout:   envDuration("HTTP_TIMEOUT", 30*time.Second),
		DBPath:        envStr("DB_PATH", "klima.db"),
		Debug:         envBool("DEBUG", false),
		PriceMax:      envInt("PRICE_MAX", 0),

		PushoverToken:    os.Getenv("PUSHOVER_TOKEN"),
		PushoverUser:     os.Getenv("PUSHOVER_USER"),
		PushoverPriority: envInt("PUSHOVER_PRIORITY", 0),
		PushoverDevice:   os.Getenv("PUSHOVER_DEVICE"),

		BraucheKlimaEnabled: envBool("BRAUCHEKLIMA_ENABLED", true),
		BraucheKlimaURL:     envStr("BRAUCHEKLIMA_URL", "https://braucheklima.de/api/availability"),
		BraucheKlimaProduct: envStr("BRAUCHEKLIMA_PRODUCT", "Midea Portasplit"),

		ObiEnabled:    envBool("OBI_ENABLED", true),
		ObiProductID:  envStr("OBI_PRODUCT_ID", "8620890"),
		ObiPostalCode: envStr("OBI_POSTAL_CODE", "36043"),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	if c.PushoverToken == "" || c.PushoverUser == "" {
		return fmt.Errorf("PUSHOVER_TOKEN and PUSHOVER_USER are required")
	}
	if !c.BraucheKlimaEnabled && !c.ObiEnabled {
		return fmt.Errorf("at least one source must be enabled (BRAUCHEKLIMA_ENABLED/OBI_ENABLED)")
	}
	if c.CheckInterval <= 0 {
		return fmt.Errorf("CHECK_INTERVAL must be a positive duration")
	}
	return nil
}

func envStr(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func envBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

func envInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func envDuration(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}
