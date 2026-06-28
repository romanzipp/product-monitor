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

	// FlareSolverrURL, when set, routes Cloudflare-protected sources
	// (braucheklima) through a FlareSolverr proxy. Empty disables it.
	FlareSolverrURL     string
	FlareSolverrTimeout time.Duration

	ObiEnabled   bool
	ObiProductID string

	MediaMarktEnabled bool
	MediaMarktURL     string

	EuronicsEnabled bool
	EuronicsURL     string

	// HomePLZ is the reference postal code for this deployment. It is the single
	// source of location truth: location-aware sources (OBI) query against it,
	// and the in-store filter prefixes default to its leading digits.
	HomePLZ string

	// LocalPLZPrefixes filters in-store availability to nearby stores: an
	// in-store result is kept only if its postal code starts with one of these
	// prefixes. Empty means no filter. Online results are never filtered.
	// Defaults to the region of HomePLZ (its first two digits).
	LocalPLZPrefixes []string
}

// Load reads configuration from a local .env file (if present) and the
// environment, applies defaults, and validates required fields.
func Load() (*Config, error) {
	// Ignore the error: a missing .env in production is perfectly fine when
	// all values are provided via real environment variables.
	_ = godotenv.Load()

	homePLZ := envStr("LOCAL_PLZ", "36037")

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

		FlareSolverrURL:     envStr("FLARESOLVERR_URL", ""),
		FlareSolverrTimeout: envDuration("FLARESOLVERR_TIMEOUT", 60*time.Second),

		ObiEnabled:   envBool("OBI_ENABLED", true),
		ObiProductID: envStr("OBI_PRODUCT_ID", "8620890"),

		MediaMarktEnabled: envBool("MEDIAMARKT_ENABLED", true),
		MediaMarktURL: envStr("MEDIAMARKT_URL",
			"https://www.mediamarkt.de/de/product/_midea-porta-split-klimaanlage-grau-max-raumgrosse-42-m-eek-a-142245268.html"),

		EuronicsEnabled: envBool("EURONICS_ENABLED", true),
		EuronicsURL: envStr("EURONICS_URL",
			"https://www.euronics.de/haus-und-haushalt/heizen-lueften-kuehlen/kuehlen/split-klimageraete/porta-split-split-klimageraet-a-4065327878899"),

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
	if !c.BraucheKlimaEnabled && !c.ObiEnabled && !c.MediaMarktEnabled && !c.EuronicsEnabled {
		return fmt.Errorf("at least one source must be enabled " +
			"(BRAUCHEKLIMA_ENABLED/OBI_ENABLED/MEDIAMARKT_ENABLED/EURONICS_ENABLED)")
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

// plzRegion returns the default in-store filter prefix for a postal code: its
// first two digits, which group a German postal region (e.g. "36037" -> "36").
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
