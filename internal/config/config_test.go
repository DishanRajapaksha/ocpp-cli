package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadProfile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	contents := `{"default_profile":"site-a","profiles":{"site-a":{"central_system_url":"wss://example.test/ocpp","charge_point_id":"CP-9","timeout":"15s","charge_point_model":"Model","charge_point_vendor":"Vendor"}}}`
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadClientConfigForProfile(path, "")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.CentralSystemURL != "wss://example.test/ocpp" || cfg.ChargePointID != "CP-9" {
		t.Fatalf("unexpected config: %#v", cfg)
	}
	if cfg.Timeout != 15*time.Second {
		t.Fatalf("unexpected timeout: %v", cfg.Timeout)
	}
}

func TestValidateRequiresCredentialPairs(t *testing.T) {
	cfg := DefaultClientConfig()
	cfg.Username = "user"
	if err := ValidateClientConfig(cfg); err == nil {
		t.Fatal("expected credential validation error")
	}
}

func TestDefaultConfigMayBeAbsent(t *testing.T) {
	cfg, err := LoadClientConfigForProfile(DefaultConfigPath, "")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ChargePointID != DefaultChargePointID {
		t.Fatalf("unexpected defaults: %#v", cfg)
	}
}
