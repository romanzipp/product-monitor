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

	BraucheKlimaEnabled  bool
	BraucheKlimaURL      string
	BraucheKlimaProducts []string

	FlareSolverrURL     string
	FlareSolverrTimeout time.Duration

	ObiEnabled     bool
	ObiProductIDs  []string
	ObiPostalCodes []string

	MediaMarktEnabled bool
	MediaMarktURLs    []string

	EuronicsEnabled bool
	EuronicsURLs    []string

	GlobusEnabled bool
	GlobusURLs    []string

	AmazonEnabled bool
	AmazonURLs    []string

	BauhausEnabled bool
	BauhausURLs    []string

	HagebauEnabled bool
	HagebauURLs    []string

	HornbachEnabled bool
	HornbachURLs    []string

	ToomEnabled bool
	ToomURLs    []string

	SolarProfiEnabled bool
	SolarProfiURLs    []string

	GalaxusEnabled bool
	GalaxusURLs    []string

	Solario24Enabled bool
	Solario24URLs    []string

	EvolarShopEnabled bool
	EvolarShopURLs    []string

	BueromarktEnabled bool
	BueromarktURLs    []string

	ExpertEnabled bool
	ExpertURLs    []string
	ExpertStoreID string

	ProsatechEnabled bool
	ProsatechURLs    []string

	TadoEnabled bool
	TadoURLs    []string

	SolarHandel24Enabled bool
	SolarHandel24URLs    []string

	SchwabKlimaEnabled bool
	SchwabKlimaURLs    []string

	GrzEnabled bool
	GrzURLs    []string

	SelfioEnabled bool
	SelfioURLs    []string

	KlimaVertriebEnabled bool
	KlimaVertriebURLs    []string

	GroupSumiEnabled bool
	GroupSumiURLs    []string

	WeinmannSchanzEnabled bool
	WeinmannSchanzURLs    []string

	TalentKingEnabled bool
	TalentKingURLs    []string

	HeizungBilligerEnabled bool
	HeizungBilligerURLs    []string

	TecedoEnabled bool
	TecedoURLs    []string

	MediaDealEnabled bool
	MediaDealURLs    []string

	KlimafyEnabled bool
	KlimafyURLs    []string

	EntratekEnabled bool
	EntratekURLs    []string

	BobsElektroEnabled bool
	BobsElektroURLs    []string

	GrSolarEnabled bool
	GrSolarURLs    []string

	BauhausStoreEnabled    bool
	BauhausStoreProductIDs []string
	BauhausStores          []BauhausStoreEntry

	LocalPLZPrefixes []string
}

// BauhausStoreEntry is one physical Bauhaus store: its numeric id and a name.
type BauhausStoreEntry struct {
	ID   string
	Name string
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

// sourceFile is the YAML shape for a page-check source (urls optional; empty uses
// the source's built-in default).
type sourceFile struct {
	Enabled bool     `yaml:"enabled"`
	URLs    []string `yaml:"urls"`
}

// fileConfig mirrors the YAML file. It is populated with defaults first, then the
// file is unmarshalled on top, so absent keys keep their defaults.
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

	Sources struct {
		BraucheKlima struct {
			Enabled  bool     `yaml:"enabled"`
			URL      string   `yaml:"url"`
			Products []string `yaml:"products"`
		} `yaml:"braucheklima"`
		Obi struct {
			Enabled     bool     `yaml:"enabled"`
			ProductIDs  []string `yaml:"productIDs"`
			PostalCodes []string `yaml:"postalCodes"`
		} `yaml:"obi"`
		MediaMarkt sourceFile `yaml:"mediamarkt"`
		Euronics   sourceFile `yaml:"euronics"`
		Globus     sourceFile `yaml:"globus"`
		Amazon     sourceFile `yaml:"amazon"`
		Bauhaus    sourceFile `yaml:"bauhaus"`
		Hagebau    sourceFile `yaml:"hagebau"`
		Hornbach   sourceFile `yaml:"hornbach"`
		Toom       sourceFile `yaml:"toom"`
		SolarProfi sourceFile `yaml:"solarprofi"`
		Galaxus    sourceFile `yaml:"galaxus"`
		Solario24  sourceFile `yaml:"solario24"`
		EvolarShop sourceFile `yaml:"evolarshop"`
		Bueromarkt sourceFile `yaml:"bueromarkt"`
		Expert     struct {
			Enabled bool     `yaml:"enabled"`
			URLs    []string `yaml:"urls"`
			StoreID string   `yaml:"storeId"`
		} `yaml:"expert"`
		Prosatech       sourceFile `yaml:"prosatech"`
		Tado            sourceFile `yaml:"tado"`
		SolarHandel24   sourceFile `yaml:"solarhandel24"`
		SchwabKlima     sourceFile `yaml:"schwabklima"`
		Grz             sourceFile `yaml:"grz"`
		Selfio          sourceFile `yaml:"selfio"`
		KlimaVertrieb   sourceFile `yaml:"klimavertrieb"`
		GroupSumi       sourceFile `yaml:"groupsumi"`
		WeinmannSchanz  sourceFile `yaml:"weinmannschanz"`
		TalentKing      sourceFile `yaml:"talentking"`
		HeizungBilliger sourceFile `yaml:"heizungbilliger"`
		Tecedo          sourceFile `yaml:"tecedo"`
		MediaDeal       sourceFile `yaml:"mediadeal"`
		Klimafy         sourceFile `yaml:"klimafy"`
		Entratek        sourceFile `yaml:"entratek"`
		BobsElektro     sourceFile `yaml:"bobselektro"`
		GrSolar         sourceFile `yaml:"grsolar"`
		BauhausStore    struct {
			Enabled    bool     `yaml:"enabled"`
			ProductIDs []string `yaml:"productIDs"`
			Stores     []struct {
				ID   string `yaml:"id"`
				Name string `yaml:"name"`
			} `yaml:"stores"`
		} `yaml:"bauhausStore"`
	} `yaml:"sources"`
}

// Load reads the YAML config at path and overlays the secrets from the
// environment. There are no built-in defaults: every value comes from the file
// (a .env file is loaded only to source the Pushover secrets in dev).
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

	bauhausStores := make([]BauhausStoreEntry, len(fc.Sources.BauhausStore.Stores))
	for i, st := range fc.Sources.BauhausStore.Stores {
		bauhausStores[i] = BauhausStoreEntry{ID: st.ID, Name: st.Name}
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

		BraucheKlimaEnabled:  fc.Sources.BraucheKlima.Enabled,
		BraucheKlimaURL:      fc.Sources.BraucheKlima.URL,
		BraucheKlimaProducts: fc.Sources.BraucheKlima.Products,

		FlareSolverrURL:     fc.FlareSolverr.URL,
		FlareSolverrTimeout: time.Duration(fc.FlareSolverr.Timeout),

		ObiEnabled:     fc.Sources.Obi.Enabled,
		ObiProductIDs:  fc.Sources.Obi.ProductIDs,
		ObiPostalCodes: fc.Sources.Obi.PostalCodes,

		MediaMarktEnabled: fc.Sources.MediaMarkt.Enabled,
		MediaMarktURLs:    fc.Sources.MediaMarkt.URLs,
		EuronicsEnabled:   fc.Sources.Euronics.Enabled,
		EuronicsURLs:      fc.Sources.Euronics.URLs,
		GlobusEnabled:     fc.Sources.Globus.Enabled,
		GlobusURLs:        fc.Sources.Globus.URLs,
		AmazonEnabled:     fc.Sources.Amazon.Enabled,
		AmazonURLs:        fc.Sources.Amazon.URLs,
		BauhausEnabled:    fc.Sources.Bauhaus.Enabled,
		BauhausURLs:       fc.Sources.Bauhaus.URLs,
		HagebauEnabled:    fc.Sources.Hagebau.Enabled,
		HagebauURLs:       fc.Sources.Hagebau.URLs,
		HornbachEnabled:   fc.Sources.Hornbach.Enabled,
		HornbachURLs:      fc.Sources.Hornbach.URLs,
		ToomEnabled:       fc.Sources.Toom.Enabled,
		ToomURLs:          fc.Sources.Toom.URLs,
		SolarProfiEnabled: fc.Sources.SolarProfi.Enabled,
		SolarProfiURLs:    fc.Sources.SolarProfi.URLs,

		GalaxusEnabled:       fc.Sources.Galaxus.Enabled,
		GalaxusURLs:          fc.Sources.Galaxus.URLs,
		Solario24Enabled:     fc.Sources.Solario24.Enabled,
		Solario24URLs:        fc.Sources.Solario24.URLs,
		EvolarShopEnabled:    fc.Sources.EvolarShop.Enabled,
		EvolarShopURLs:       fc.Sources.EvolarShop.URLs,
		BueromarktEnabled:    fc.Sources.Bueromarkt.Enabled,
		BueromarktURLs:       fc.Sources.Bueromarkt.URLs,
		ExpertEnabled:        fc.Sources.Expert.Enabled,
		ExpertURLs:           fc.Sources.Expert.URLs,
		ExpertStoreID:        fc.Sources.Expert.StoreID,
		ProsatechEnabled:     fc.Sources.Prosatech.Enabled,
		ProsatechURLs:        fc.Sources.Prosatech.URLs,
		TadoEnabled:          fc.Sources.Tado.Enabled,
		TadoURLs:             fc.Sources.Tado.URLs,
		SolarHandel24Enabled: fc.Sources.SolarHandel24.Enabled,
		SolarHandel24URLs:    fc.Sources.SolarHandel24.URLs,
		SchwabKlimaEnabled:   fc.Sources.SchwabKlima.Enabled,
		SchwabKlimaURLs:      fc.Sources.SchwabKlima.URLs,
		GrzEnabled:           fc.Sources.Grz.Enabled,
		GrzURLs:              fc.Sources.Grz.URLs,
		SelfioEnabled:        fc.Sources.Selfio.Enabled,
		SelfioURLs:           fc.Sources.Selfio.URLs,
		KlimaVertriebEnabled: fc.Sources.KlimaVertrieb.Enabled,
		KlimaVertriebURLs:    fc.Sources.KlimaVertrieb.URLs,

		GroupSumiEnabled:       fc.Sources.GroupSumi.Enabled,
		GroupSumiURLs:          fc.Sources.GroupSumi.URLs,
		WeinmannSchanzEnabled:  fc.Sources.WeinmannSchanz.Enabled,
		WeinmannSchanzURLs:     fc.Sources.WeinmannSchanz.URLs,
		TalentKingEnabled:      fc.Sources.TalentKing.Enabled,
		TalentKingURLs:         fc.Sources.TalentKing.URLs,
		HeizungBilligerEnabled: fc.Sources.HeizungBilliger.Enabled,
		HeizungBilligerURLs:    fc.Sources.HeizungBilliger.URLs,

		TecedoEnabled:      fc.Sources.Tecedo.Enabled,
		TecedoURLs:         fc.Sources.Tecedo.URLs,
		MediaDealEnabled:   fc.Sources.MediaDeal.Enabled,
		MediaDealURLs:      fc.Sources.MediaDeal.URLs,
		KlimafyEnabled:     fc.Sources.Klimafy.Enabled,
		KlimafyURLs:        fc.Sources.Klimafy.URLs,
		EntratekEnabled:    fc.Sources.Entratek.Enabled,
		EntratekURLs:       fc.Sources.Entratek.URLs,
		BobsElektroEnabled: fc.Sources.BobsElektro.Enabled,
		BobsElektroURLs:    fc.Sources.BobsElektro.URLs,
		GrSolarEnabled:     fc.Sources.GrSolar.Enabled,
		GrSolarURLs:        fc.Sources.GrSolar.URLs,

		BauhausStoreEnabled:    fc.Sources.BauhausStore.Enabled,
		BauhausStoreProductIDs: fc.Sources.BauhausStore.ProductIDs,
		BauhausStores:          bauhausStores,

		LocalPLZPrefixes: fc.LocalPLZPrefixes,
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
	anySource := c.BraucheKlimaEnabled || c.ObiEnabled || c.MediaMarktEnabled || c.EuronicsEnabled || c.GlobusEnabled || c.AmazonEnabled || c.BauhausEnabled || c.HagebauEnabled || c.HornbachEnabled || c.ToomEnabled || c.SolarProfiEnabled ||
		c.GalaxusEnabled || c.Solario24Enabled || c.EvolarShopEnabled || c.BueromarktEnabled || c.ExpertEnabled || c.ProsatechEnabled || c.TadoEnabled || c.SolarHandel24Enabled || c.SchwabKlimaEnabled || c.GrzEnabled || c.SelfioEnabled || c.KlimaVertriebEnabled ||
		c.GroupSumiEnabled || c.WeinmannSchanzEnabled || c.TalentKingEnabled || c.HeizungBilligerEnabled ||
		c.TecedoEnabled || c.MediaDealEnabled || c.KlimafyEnabled || c.EntratekEnabled || c.BobsElektroEnabled || c.GrSolarEnabled ||
		c.BauhausStoreEnabled
	if !anySource {
		return fmt.Errorf("at least one source must be enabled")
	}
	if c.CheckInterval <= 0 {
		return fmt.Errorf("checkInterval must be a positive duration")
	}
	return nil
}
