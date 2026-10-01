package config

import (
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	cfg := Load()
	if cfg.Port == 0 {
		t.Fatal("expected a default port")
	}
	if cfg.BaseURL == "" {
		t.Fatal("expected a base URL")
	}
	if cfg.LogLevel == "" {
		t.Fatal("expected a log level")
	}
}
