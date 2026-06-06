package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadNormalizesClientDefaults(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "csi.hcl")
	if err := os.WriteFile(path, []byte(`client {
  server_url = "http://192.168.84.10:9766"
  access_token = "cluster-token"
}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Driver.Name != "storage.csi.kloud.team" {
		t.Fatalf("unexpected driver name: %q", cfg.Driver.Name)
	}
	if cfg.Client.ServerURL != "http://192.168.84.10:9766" {
		t.Fatalf("unexpected client server url: %q", cfg.Client.ServerURL)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
}

func TestValidateRequiresAccessToken(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Driver: &DriverConf{},
		Client: &ClientConf{ServerURL: "http://127.0.0.1:9766"},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for missing access token")
	}
}

func TestValidateAcceptsLegacyTokenField(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Driver: &DriverConf{},
		Client: &ClientConf{
			ServerURL: "http://127.0.0.1:9766",
			Token:     "legacy-token",
		},
	}
	if err := cfg.normalize(); err != nil {
		t.Fatalf("normalize failed: %v", err)
	}
	if cfg.Client.AccessToken != "legacy-token" {
		t.Fatalf("expected legacy token migration, got %q", cfg.Client.AccessToken)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
}

func TestApplyEnvOverrides(t *testing.T) {
	t.Setenv(envCSIServerURL, "http://csi.example:9766")
	t.Setenv(envCSIAccessToken, "cluster-token")
	t.Setenv(envCSIDriverMode, "node")
	t.Setenv(envCSINodeID, "worker-1")

	cfg := &Config{
		Driver: &DriverConf{},
		Client: &ClientConf{},
	}
	ApplyEnvOverrides(cfg)

	if cfg.Client.ServerURL != "http://csi.example:9766" {
		t.Fatalf("unexpected server url: %q", cfg.Client.ServerURL)
	}
	if cfg.Client.AccessToken != "cluster-token" {
		t.Fatalf("unexpected token: %q", cfg.Client.AccessToken)
	}
	if cfg.Driver.Mode != CSIModeNode {
		t.Fatalf("unexpected mode: %q", cfg.Driver.Mode)
	}
	if cfg.Driver.NodeID != "worker-1" {
		t.Fatalf("unexpected node id: %q", cfg.Driver.NodeID)
	}
}
