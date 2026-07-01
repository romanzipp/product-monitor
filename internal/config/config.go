// Package config loads runtime configuration from a YAML file. Secrets
// (Pushover token/user) come from the environment, not the file.
package config

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

// Config holds all runtime configuration in a flat shape for the rest of the app.
type Config struct {
	CheckInterval time.Duration
	HTTPTimeout   time.Duration
	DBPath        string
	MetricsAddr   string
	PriceMax      int

	PushoverToken    string // from env PUSHOVER_TOKEN
	PushoverUser     string // from env PUSHOVER_USER
	PushoverPriority int
	PushoverDevice   string
	PushoverRetry    int
	PushoverExpire   int

	BraucheKlimaEnabled bool
	BraucheKlimaURL     string
	BraucheKlimaProduct string

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

	BauhausStoreEnabled bool
	BauhausStoreID      string
	BauhausStoreName    string

	HomePLZ          string
	LocalPLZPrefixes []string
}

// Duration unmarshals a YAML duration string such as "5m" or "30s".
type Duration time.Duration

func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s, err)
	}
	*d = Duration(parsed)
	return nil
}

// sourceFile is the YAML shape for a page-check source (URL optional; empty uses
// the source's built-in default).
type sourceFile struct {
	Enabled bool   `yaml:"enabled"`
	URL     string `yaml:"url"`
}

// fileConfig mirrors the YAML file. It is populated with defaults first, then the
// file is unmarshalled on top, so absent keys keep their defaults.
type fileConfig struct {
	CheckInterval    Duration `yaml:"checkInterval"`
	HTTPTimeout      Duration `yaml:"httpTimeout"`
	DBPath           string   `yaml:"dbPath"`
	MetricsAddr      string   `yaml:"metricsAddr"`
	PriceMax         int      `yaml:"priceMax"`
	HomePLZ          string   `yaml:"homePLZ"`
	LocalPLZPrefixes []string `yaml:"localPLZPrefixes"`

	Pushover struct {
		Priority int    `yaml:"priority"`
		Retry    int    `yaml:"retry"`
		Expire   int    `yaml:"expire"`
		Device   string `yaml:"device"`
	} `yaml:"pushover"`

	FlareSolverr struct {
		URL     string   `yaml:"url"`
		Timeout Duration `yaml:"timeout"`
	} `yaml:"flaresolverr"`

	Sources struct {
		BraucheKlima struct {
			Enabled bool   `yaml:"enabled"`
			URL     string `yaml:"url"`
			Product string `yaml:"product"`
		} `yaml:"braucheklima"`
		Obi struct {
			Enabled   bool   `yaml:"enabled"`
			ProductID string `yaml:"productID"`
		} `yaml:"obi"`
		MediaMarkt   sourceFile `yaml:"mediamarkt"`
		Euronics     sourceFile `yaml:"euronics"`
		Globus       sourceFile `yaml:"globus"`
		Amazon       sourceFile `yaml:"amazon"`
		Bauhaus      sourceFile `yaml:"bauhaus"`
		Hagebau      sourceFile `yaml:"hagebau"`
		Hornbach     sourceFile `yaml:"hornbach"`
		Toom         sourceFile `yaml:"toom"`
		BauhausStore struct {
			Enabled   bool   `yaml:"enabled"`
			StoreID   string `yaml:"storeID"`
			StoreName string `yaml:"storeName"`
		} `yaml:"bauhausStore"`
	} `yaml:"sources"`
}

func defaults() fileConfig {
	var fc fileConfig
	fc.CheckInterval = Duration(5 * time.Minute)
	fc.HTTPTimeout = Duration(30 * time.Second)
	fc.DBPath = "klima.db"
	fc.MetricsAddr = ":8080"
	fc.HomePLZ = "36037"

	fc.Pushover.Priority = 2 // emergency
	fc.Pushover.Retry = 60
	fc.Pushover.Expire = 3600

	fc.FlareSolverr.Timeout = Duration(60 * time.Second)

	fc.Sources.BraucheKlima.Enabled = true
	fc.Sources.BraucheKlima.URL = "https://braucheklima.de/api/availability"
	fc.Sources.BraucheKlima.Product = "Midea Portasplit"
	fc.Sources.Obi.Enabled = true
	fc.Sources.Obi.ProductID = "8620890"
	fc.Sources.MediaMarkt.Enabled = true
	fc.Sources.Euronics.Enabled = true
	fc.Sources.Globus.Enabled = true
	fc.Sources.Amazon.Enabled = true
	fc.Sources.Bauhaus.Enabled = true
	fc.Sources.Hagebau.Enabled = true
	fc.Sources.Hornbach.Enabled = true
	fc.Sources.Toom.Enabled = true
	fc.Sources.BauhausStore.Enabled = true
	fc.Sources.BauhausStore.StoreID = "589"
	fc.Sources.BauhausStore.StoreName = "Bauhaus Frankfurt"
	return fc
}

// Load reads the YAML config at path (over built-in defaults) and overlays the
// secrets from the environment. A .env file is loaded for convenience in dev.
func Load(path string) (*Config, error) {
	_ = godotenv.Load()

	fc := defaults()
	if data, err := os.ReadFile(path); err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	} else if err := yaml.Unmarshal(data, &fc); err != nil {
		return nil, fmt.Errorf("parse config %q: %w", path, err)
	}

	prefixes := fc.LocalPLZPrefixes
	if len(prefixes) == 0 {
		prefixes = plzRegion(fc.HomePLZ)
	}

	cfg := &Config{
		CheckInterval: time.Duration(fc.CheckInterval),
		HTTPTimeout:   time.Duration(fc.HTTPTimeout),
		DBPath:        fc.DBPath,
		MetricsAddr:   fc.MetricsAddr,
		PriceMax:      fc.PriceMax,

		PushoverToken:    os.Getenv("PUSHOVER_TOKEN"),
		PushoverUser:     os.Getenv("PUSHOVER_USER"),
		PushoverPriority: fc.Pushover.Priority,
		PushoverDevice:   fc.Pushover.Device,
		PushoverRetry:    fc.Pushover.Retry,
		PushoverExpire:   fc.Pushover.Expire,

		BraucheKlimaEnabled: fc.Sources.BraucheKlima.Enabled,
		BraucheKlimaURL:     fc.Sources.BraucheKlima.URL,
		BraucheKlimaProduct: fc.Sources.BraucheKlima.Product,

		FlareSolverrURL:     fc.FlareSolverr.URL,
		FlareSolverrTimeout: time.Duration(fc.FlareSolverr.Timeout),

		ObiEnabled:   fc.Sources.Obi.Enabled,
		ObiProductID: fc.Sources.Obi.ProductID,

		MediaMarktEnabled: fc.Sources.MediaMarkt.Enabled,
		MediaMarktURL:     fc.Sources.MediaMarkt.URL,
		EuronicsEnabled:   fc.Sources.Euronics.Enabled,
		EuronicsURL:       fc.Sources.Euronics.URL,
		GlobusEnabled:     fc.Sources.Globus.Enabled,
		GlobusURL:         fc.Sources.Globus.URL,
		AmazonEnabled:     fc.Sources.Amazon.Enabled,
		AmazonURL:         fc.Sources.Amazon.URL,
		BauhausEnabled:    fc.Sources.Bauhaus.Enabled,
		BauhausURL:        fc.Sources.Bauhaus.URL,
		HagebauEnabled:    fc.Sources.Hagebau.Enabled,
		HagebauURL:        fc.Sources.Hagebau.URL,
		HornbachEnabled:   fc.Sources.Hornbach.Enabled,
		HornbachURL:       fc.Sources.Hornbach.URL,
		ToomEnabled:       fc.Sources.Toom.Enabled,
		ToomURL:           fc.Sources.Toom.URL,

		BauhausStoreEnabled: fc.Sources.BauhausStore.Enabled,
		BauhausStoreID:      fc.Sources.BauhausStore.StoreID,
		BauhausStoreName:    fc.Sources.BauhausStore.StoreName,

		HomePLZ:          fc.HomePLZ,
		LocalPLZPrefixes: prefixes,
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	if c.PushoverToken == "" || c.PushoverUser == "" {
		return fmt.Errorf("PUSHOVER_TOKEN and PUSHOVER_USER environment variables are required")
	}
	anySource := c.BraucheKlimaEnabled || c.ObiEnabled || c.MediaMarktEnabled || c.EuronicsEnabled || c.GlobusEnabled || c.AmazonEnabled || c.BauhausEnabled || c.HagebauEnabled || c.HornbachEnabled || c.ToomEnabled || c.BauhausStoreEnabled
	if !anySource {
		return fmt.Errorf("at least one source must be enabled")
	}
	if c.CheckInterval <= 0 {
		return fmt.Errorf("checkInterval must be a positive duration")
	}
	return nil
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
