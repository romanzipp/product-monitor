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
homePLZ: "60311"
localPLZPrefixes: ["60", "63"]
pushover:
  priority: 2
sources:
  amazon:
    enabled: false
  mediamarkt:
    enabled: true
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
	if cfg.AmazonEnabled {
		t.Errorf("amazon should be disabled")
	}
	if len(cfg.MediaMarktURLs) != 2 {
		t.Errorf("MediaMarktURLs = %v, want 2 entries", cfg.MediaMarktURLs)
	}
	// No defaults: an unspecified source stays disabled with empty fields.
	if cfg.BraucheKlimaEnabled || len(cfg.BraucheKlimaProducts) != 0 {
		t.Errorf("unspecified braucheklima should be zero-valued, got enabled=%v products=%v", cfg.BraucheKlimaEnabled, cfg.BraucheKlimaProducts)
	}
	if cfg.PushoverToken != "tok" || cfg.PushoverUser != "usr" {
		t.Errorf("secrets not loaded from env")
	}
}

func TestLoadMissingSecrets(t *testing.T) {
	t.Setenv("PUSHOVER_TOKEN", "")
	t.Setenv("PUSHOVER_USER", "")
	p := writeConfig(t, "checkInterval: 5m\nsources:\n  obi:\n    enabled: true\n")
	if _, err := Load(p); err == nil {
		t.Fatal("expected error for missing Pushover secrets")
	}
}

func TestLoadMissingCheckInterval(t *testing.T) {
	t.Setenv("PUSHOVER_TOKEN", "tok")
	t.Setenv("PUSHOVER_USER", "usr")
	// No checkInterval -> zero -> validation error (no default fills it in).
	p := writeConfig(t, "sources:\n  obi:\n    enabled: true\n")
	if _, err := Load(p); err == nil {
		t.Fatal("expected error for missing checkInterval")
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
