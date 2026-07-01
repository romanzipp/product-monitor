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

func TestLoadDefaultsAndOverrides(t *testing.T) {
	t.Setenv("PUSHOVER_TOKEN", "tok")
	t.Setenv("PUSHOVER_USER", "usr")

	// Minimal file: overrides a few keys; everything else keeps defaults.
	p := writeConfig(t, `
checkInterval: 10m
homePLZ: "60311"
sources:
  amazon:
    enabled: false
`)
	cfg, err := Load(p)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.CheckInterval != 10*time.Minute {
		t.Errorf("CheckInterval = %v, want 10m", cfg.CheckInterval)
	}
	if cfg.HTTPTimeout != 30*time.Second {
		t.Errorf("HTTPTimeout default lost: %v", cfg.HTTPTimeout)
	}
	if cfg.AmazonEnabled {
		t.Errorf("amazon should be disabled")
	}
	if !cfg.BraucheKlimaEnabled {
		t.Errorf("braucheklima should keep default enabled")
	}
	// localPLZPrefixes not set -> derived from homePLZ region.
	if len(cfg.LocalPLZPrefixes) != 1 || cfg.LocalPLZPrefixes[0] != "60" {
		t.Errorf("LocalPLZPrefixes = %v, want [60]", cfg.LocalPLZPrefixes)
	}
	if cfg.PushoverPriority != 2 {
		t.Errorf("PushoverPriority default = %d, want 2", cfg.PushoverPriority)
	}
	if cfg.PushoverToken != "tok" || cfg.PushoverUser != "usr" {
		t.Errorf("secrets not loaded from env")
	}
}

func TestLoadMissingSecrets(t *testing.T) {
	t.Setenv("PUSHOVER_TOKEN", "")
	t.Setenv("PUSHOVER_USER", "")
	p := writeConfig(t, "checkInterval: 5m\n")
	if _, err := Load(p); err == nil {
		t.Fatal("expected error for missing Pushover secrets")
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
