package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadReadsValuesVerbatim(t *testing.T) {
	t.Setenv("PUSHOVER_TOKEN", "tok")
	t.Setenv("PUSHOVER_USER", "usr")

	// Values are read exactly as written; there are no built-in defaults.
	p := writeConfig(t, `
checkInterval: 10m
httpTimeout: 15s
priceMax: 900
localPLZPrefixes: ["60", "63"]
pushover:
  priority: 2
products:
  - name: Halterung
    sources:
      obi:
        productIDs: ["123"]
        postalCodes: ["60311"]
      mediamarkt:
        urls:
          - https://example.com/a
          - https://example.com/b
`)
	cfg, err := Load(p)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.CheckInterval != 10*time.Minute {
		t.Errorf("CheckInterval = %v, want 10m", cfg.CheckInterval)
	}
	if cfg.HTTPTimeout != 15*time.Second {
		t.Errorf("HTTPTimeout = %v, want 15s", cfg.HTTPTimeout)
	}
	if cfg.PriceMax != 900 {
		t.Errorf("PriceMax = %d, want 900", cfg.PriceMax)
	}
	if len(cfg.LocalPLZPrefixes) != 2 || cfg.LocalPLZPrefixes[0] != "60" {
		t.Errorf("LocalPLZPrefixes = %v, want [60 63]", cfg.LocalPLZPrefixes)
	}
	if len(cfg.Products) != 1 || cfg.Products[0].Name != "Halterung" {
		t.Fatalf("Products = %+v, want one named Halterung", cfg.Products)
	}
	s := cfg.Products[0].Sources
	if s.Obi == nil || len(s.Obi.ProductIDs) != 1 || s.Obi.PostalCodes[0] != "60311" {
		t.Errorf("obi = %+v, want productIDs/postalCodes set", s.Obi)
	}
	if s.MediaMarkt == nil || len(s.MediaMarkt.URLs) != 2 {
		t.Errorf("mediamarkt = %+v, want 2 urls", s.MediaMarkt)
	}
	// Absent source stays nil (skipped), no enabled flag.
	if s.BraucheKlima != nil {
		t.Errorf("unspecified braucheklima should be nil, got %+v", s.BraucheKlima)
	}
	if cfg.PushoverToken != "tok" || cfg.PushoverUser != "usr" {
		t.Errorf("secrets not loaded from env")
	}
}

func TestLoadMissingSecrets(t *testing.T) {
	t.Setenv("PUSHOVER_TOKEN", "")
	t.Setenv("PUSHOVER_USER", "")
	p := writeConfig(t, "checkInterval: 5m\nproducts:\n  - name: X\n    sources:\n      obi:\n        productIDs: [\"1\"]\n")
	if _, err := Load(p); err == nil {
		t.Fatal("expected error for missing Pushover secrets")
	}
}

func TestLoadMissingCheckInterval(t *testing.T) {
	t.Setenv("PUSHOVER_TOKEN", "tok")
	t.Setenv("PUSHOVER_USER", "usr")
	// No checkInterval -> zero -> validation error (no default fills it in).
	p := writeConfig(t, "products:\n  - name: X\n    sources:\n      obi:\n        productIDs: [\"1\"]\n")
	if _, err := Load(p); err == nil {
		t.Fatal("expected error for missing checkInterval")
	}
}

func TestLoadNoProducts(t *testing.T) {
	t.Setenv("PUSHOVER_TOKEN", "tok")
	t.Setenv("PUSHOVER_USER", "usr")
	p := writeConfig(t, "checkInterval: 5m\n")
	if _, err := Load(p); err == nil {
		t.Fatal("expected error for no products")
	}
}

func TestExampleConfigLoads(t *testing.T) {
	t.Setenv("PUSHOVER_TOKEN", "tok")
	t.Setenv("PUSHOVER_USER", "usr")
	if _, err := Load("../../config.example.yaml"); err != nil {
		t.Fatalf("config.example.yaml must load: %v", err)
	}
}

func TestLoadMissingFile(t *testing.T) {
	t.Setenv("PUSHOVER_TOKEN", "tok")
	t.Setenv("PUSHOVER_USER", "usr")
	if _, err := Load(filepath.Join(t.TempDir(), "nope.yaml")); err == nil {
		t.Fatal("expected error for missing config file")
	}
}
