// Package config loads application configuration from the environment.
// Values may be supplied via a .env file (loaded automatically) or via real
// environment variables, which always take precedence.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all runtime configuration.
type Config struct {
	CheckInterval time.Duration
	HTTPTimeout   time.Duration
	DBPath        string
	PriceMax      int // whole euros; 0 = no limit

	PushoverToken    string
	PushoverUser     string
	PushoverPriority int
	PushoverDevice   string

	BraucheKlimaEnabled bool
	BraucheKlimaURL     string
	BraucheKlimaProduct string

	// FlareSolverrURL, when set, routes Cloudflare-protected sources through a
	// FlareSolverr proxy. Empty disables it.
	FlareSolverrURL     string
	FlareSolverrTimeout time.Duration

	ObiEnabled   bool
	ObiProductID string

	MediaMarktEnabled bool
	MediaMarktURL     string

	EuronicsEnabled bool
	EuronicsURL     string

	GlobusEnabled bool
	GlobusURL     string

	AmazonEnabled bool
	AmazonURL     string

	BauhausEnabled bool
	BauhausURL     string

	HagebauEnabled bool
	HagebauURL     string

	HornbachEnabled bool
	HornbachURL     string

	ToomEnabled bool
	ToomURL     string

	// HomePLZ is the single location reference: OBI queries against it and the
	// in-store filter prefixes default to its leading digits.
	HomePLZ string

	// LocalPLZPrefixes keeps an in-store result only if its postal code starts
	// with one of these prefixes. Empty disables the filter; online is unaffected.
	LocalPLZPrefixes []string
}

// Load reads a local .env file (if present) and the environment, applies
// defaults, and validates required fields.
func Load() (*Config, error) {
	// A missing .env is fine when values come from real environment variables.
	_ = godotenv.Load()

	homePLZ := envStr("LOCAL_PLZ", "36037")

	cfg := &Config{
		CheckInterval: envDuration("CHECK_INTERVAL", 5*time.Minute),
		HTTPTimeout:   envDuration("HTTP_TIMEOUT", 30*time.Second),
		DBPath:        envStr("DB_PATH", "klima.db"),
		PriceMax:      envInt("PRICE_MAX", 0),

		PushoverToken:    os.Getenv("PUSHOVER_TOKEN"),
		PushoverUser:     os.Getenv("PUSHOVER_USER"),
		PushoverPriority: envInt("PUSHOVER_PRIORITY", 0),
		PushoverDevice:   os.Getenv("PUSHOVER_DEVICE"),

		BraucheKlimaEnabled: envBool("BRAUCHEKLIMA_ENABLED", true),
		BraucheKlimaURL:     envStr("BRAUCHEKLIMA_URL", "https://braucheklima.de/api/availability"),
		BraucheKlimaProduct: envStr("BRAUCHEKLIMA_PRODUCT", "Midea Portasplit"),

		FlareSolverrURL:     envStr("FLARESOLVERR_URL", ""),
		FlareSolverrTimeout: envDuration("FLARESOLVERR_TIMEOUT", 60*time.Second),

		ObiEnabled:   envBool("OBI_ENABLED", true),
		ObiProductID: envStr("OBI_PRODUCT_ID", "8620890"),

		MediaMarktEnabled: envBool("MEDIAMARKT_ENABLED", true),
		MediaMarktURL:     envStr("MEDIAMARKT_URL", ""),

		EuronicsEnabled: envBool("EURONICS_ENABLED", true),
		EuronicsURL:     envStr("EURONICS_URL", ""),

		GlobusEnabled: envBool("GLOBUS_ENABLED", true),
		GlobusURL:     envStr("GLOBUS_URL", ""),

		AmazonEnabled: envBool("AMAZON_ENABLED", true),
		AmazonURL:     envStr("AMAZON_URL", ""),

		BauhausEnabled: envBool("BAUHAUS_ENABLED", true),
		BauhausURL:     envStr("BAUHAUS_URL", ""),

		HagebauEnabled: envBool("HAGEBAU_ENABLED", true),
		HagebauURL:     envStr("HAGEBAU_URL", ""),

		HornbachEnabled: envBool("HORNBACH_ENABLED", true),
		HornbachURL:     envStr("HORNBACH_URL", ""),

		ToomEnabled: envBool("TOOM_ENABLED", true),
		ToomURL:     envStr("TOOM_URL", ""),

		HomePLZ:          homePLZ,
		LocalPLZPrefixes: envCSV("LOCAL_PLZ_PREFIXES", plzRegion(homePLZ)),
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
	anySource := c.BraucheKlimaEnabled || c.ObiEnabled || c.MediaMarktEnabled || c.EuronicsEnabled || c.GlobusEnabled || c.AmazonEnabled || c.BauhausEnabled || c.HagebauEnabled || c.HornbachEnabled || c.ToomEnabled
	if !anySource {
		return fmt.Errorf("at least one source must be enabled (*_ENABLED)")
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

// plzRegion returns a postal code's first two digits, e.g. "36037" -> "36".
func plzRegion(plz string) []string {
	if len(plz) >= 2 {
		return []string{plz[:2]}
	}
	if plz != "" {
		return []string{plz}
	}
	return nil
}

// envCSV parses a comma-separated env var into a trimmed, non-empty slice.
func envCSV(key string, def []string) []string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	var out []string
	for _, part := range strings.Split(v, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return def
	}
	return out
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
