package config

import "testing"


func TestLoadConfig_Defaults(t *testing.T) {
	cfg := LoadConfig()

	if cfg.Port != "8080" {
		t.Errorf("expected default Port 8080, got %s", cfg.Port)
	}

	if cfg.Environment != "development" {
		t.Errorf("expected default Environment development, got %s", cfg.Environment)
	}
}

func TestLoadConfig_CustomEnvVars(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("ENV", "staging")


	cfg := LoadConfig()

	if cfg.Port != "9090" {
		t.Errorf("expected Port 9090, got %s", cfg.Port)
	}
	if cfg.Environment != "staging" {
		t.Errorf("expected Environment staging, got %s", cfg.Environment)
	}

}