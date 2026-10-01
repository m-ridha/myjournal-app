package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config contains the application-level settings used by the server at startup.
type Config struct {
	BaseURL      string
	Port         int
	LogLevel     string
	SessionKey   string
	CookieSecure bool
	Providers    string
	LocalDataDir string
}

func Load() Config {
	cfg := Config{
		BaseURL:      "http://localhost:3030",
		Port:         3030,
		LogLevel:     "info",
		SessionKey:   "dev-session-key-change-me",
		CookieSecure: false,
		Providers:    "local",
		LocalDataDir: "./.devdata",
	}

	if v := os.Getenv("APP_BASE_URL"); v != "" {
		cfg.BaseURL = v
	}
	if v := os.Getenv("PORT"); v != "" {
		port, err := strconv.Atoi(v)
		if err == nil && port > 0 && port <= 65535 {
			cfg.Port = port
		}
	}
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}
	if v := os.Getenv("SESSION_KEY"); v != "" {
		cfg.SessionKey = v
	}
	if v := os.Getenv("COOKIE_SECURE"); v != "" {
		secure, err := strconv.ParseBool(v)
		if err == nil {
			cfg.CookieSecure = secure
		}
	}
	if v := os.Getenv("PROVIDERS"); v != "" {
		cfg.Providers = v
	}
	if v := os.Getenv("LOCAL_DATA_DIR"); v != "" {
		cfg.LocalDataDir = v
	}
	return cfg
}

func (c Config) Validate() error {
	if c.BaseURL == "" {
		return fmt.Errorf("APP_BASE_URL is required")
	}
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("PORT must be between 1 and 65535")
	}
	if c.LogLevel == "" {
		return fmt.Errorf("LOG_LEVEL is required")
	}
	if c.SessionKey == "" {
		return fmt.Errorf("SESSION_KEY is required")
	}
	if c.Providers == "" {
		return fmt.Errorf("PROVIDERS is required")
	}
	return nil
}
