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

// Config holds all runtime configuration for the app.
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

	FlareSolverrURL     string
	FlareSolverrTimeout time.Duration

	LocalPLZPrefixes []string

	Products []Product
}

// Product is a monitored product plus the per-source config used to find it. The
// name is shown in notifications. A source is checked only if it is present under
// `sources`; an absent source is skipped (no enabled flag).
type Product struct {
	Name    string         `yaml:"name"`
	Sources ProductSources `yaml:"sources"`
}

// BauhausStoreEntry is one physical Bauhaus store: its numeric id and a name.
type BauhausStoreEntry struct {
	ID   string `yaml:"id"`
	Name string `yaml:"name"`
}

// URLSource is the config for a plain page-check source: a list of product URLs.
type URLSource struct {
	URLs []string `yaml:"urls"`
}

// ProductSources maps each supported source to its config. Pointer fields are nil
// when the key is absent from YAML, which means "don't check this source".
//
// Supported sources: braucheklima, obi, mediamarkt, euronics, globus, amazon,
// bauhaus, hagebau, hornbach, toom, solarprofi, galaxus, solario24, evolarshop,
// bueromarkt, expert, prosatech, tado, solarhandel24, schwabklima, grz, selfio,
// klimavertrieb, groupsumi, weinmannschanz, talentking, heizungbilliger, tecedo,
// mediadeal, klimafy, entratek, bobselektro, grsolar, bauhausStore.
type ProductSources struct {
	BraucheKlima *struct {
		URL      string   `yaml:"url"`
		Products []string `yaml:"products"`
	} `yaml:"braucheklima"`
	Obi *struct {
		ProductIDs  []string `yaml:"productIDs"`
		PostalCodes []string `yaml:"postalCodes"`
	} `yaml:"obi"`
	MediaMarkt *URLSource `yaml:"mediamarkt"`
	Euronics   *URLSource `yaml:"euronics"`
	Globus     *URLSource `yaml:"globus"`
	Amazon     *URLSource `yaml:"amazon"`
	Bauhaus    *URLSource `yaml:"bauhaus"`
	Hagebau    *URLSource `yaml:"hagebau"`
	Hornbach   *URLSource `yaml:"hornbach"`
	Toom       *URLSource `yaml:"toom"`
	SolarProfi *URLSource `yaml:"solarprofi"`
	Galaxus    *URLSource `yaml:"galaxus"`
	Solario24  *URLSource `yaml:"solario24"`
	EvolarShop *URLSource `yaml:"evolarshop"`
	Bueromarkt *URLSource `yaml:"bueromarkt"`
	Expert     *struct {
		URLs    []string `yaml:"urls"`
		StoreID string   `yaml:"storeId"`
	} `yaml:"expert"`
	Prosatech       *URLSource `yaml:"prosatech"`
	Tado            *URLSource `yaml:"tado"`
	SolarHandel24   *URLSource `yaml:"solarhandel24"`
	SchwabKlima     *URLSource `yaml:"schwabklima"`
	Grz             *URLSource `yaml:"grz"`
	Selfio          *URLSource `yaml:"selfio"`
	KlimaVertrieb   *URLSource `yaml:"klimavertrieb"`
	GroupSumi       *URLSource `yaml:"groupsumi"`
	WeinmannSchanz  *URLSource `yaml:"weinmannschanz"`
	TalentKing      *URLSource `yaml:"talentking"`
	HeizungBilliger *URLSource `yaml:"heizungbilliger"`
	Tecedo          *URLSource `yaml:"tecedo"`
	MediaDeal       *URLSource `yaml:"mediadeal"`
	Klimafy         *URLSource `yaml:"klimafy"`
	Entratek        *URLSource `yaml:"entratek"`
	BobsElektro     *URLSource `yaml:"bobselektro"`
	GrSolar         *URLSource `yaml:"grsolar"`
	BauhausStore    *struct {
		ProductIDs []string            `yaml:"productIDs"`
		Stores     []BauhausStoreEntry `yaml:"stores"`
	} `yaml:"bauhausStore"`
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

// fileConfig mirrors the YAML file.
type fileConfig struct {
	CheckInterval    Duration `yaml:"checkInterval"`
	HTTPTimeout      Duration `yaml:"httpTimeout"`
	DBPath           string   `yaml:"dbPath"`
	MetricsAddr      string   `yaml:"metricsAddr"`
	PriceMax         int      `yaml:"priceMax"`
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

	Products []Product `yaml:"products"`
}

// Load reads the YAML config at path and overlays the secrets from the
// environment (a .env file is loaded only to source the Pushover secrets in dev).
func Load(path string) (*Config, error) {
	_ = godotenv.Load()

	var fc fileConfig
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}
	if err := yaml.Unmarshal(data, &fc); err != nil {
		return nil, fmt.Errorf("parse config %q: %w", path, err)
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

		FlareSolverrURL:     fc.FlareSolverr.URL,
		FlareSolverrTimeout: time.Duration(fc.FlareSolverr.Timeout),

		LocalPLZPrefixes: fc.LocalPLZPrefixes,
		Products:         fc.Products,
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
	if len(c.Products) == 0 {
		return fmt.Errorf("at least one product must be configured")
	}
	if c.CheckInterval <= 0 {
		return fmt.Errorf("checkInterval must be a positive duration")
	}
	return nil
}
