package config

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
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

func Load() (Config, error) {
	cfg := Config{
		BaseURL:      "http://localhost:3030",
		Port:         3030,
		LogLevel:     "info",
		CookieSecure: false,
		Providers:    "local",
		LocalDataDir: "./.devdata",
	}

	if v, ok := os.LookupEnv("APP_BASE_URL"); ok {
		cfg.BaseURL = v
	}
	if v, ok := os.LookupEnv("PORT"); ok {
		port, err := strconv.Atoi(v)
		if err != nil || port < 1 || port > 65535 {
			return Config{}, fmt.Errorf("PORT must be an integer between 1 and 65535")
		}
		cfg.Port = port
	}
	if v, ok := os.LookupEnv("LOG_LEVEL"); ok {
		cfg.LogLevel = v
	}
	if v, ok := os.LookupEnv("SESSION_KEY"); ok {
		cfg.SessionKey = v
	}
	if v, ok := os.LookupEnv("COOKIE_SECURE"); ok {
		secure, err := strconv.ParseBool(v)
		if err != nil {
			return Config{}, fmt.Errorf("COOKIE_SECURE must be true or false")
		}
		cfg.CookieSecure = secure
	}
	if v, ok := os.LookupEnv("PROVIDERS"); ok {
		cfg.Providers = v
	}
	if v, ok := os.LookupEnv("LOCAL_DATA_DIR"); ok {
		cfg.LocalDataDir = v
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if c.BaseURL == "" {
		return fmt.Errorf("APP_BASE_URL is required")
	}
	baseURL, err := url.Parse(c.BaseURL)
	if err != nil || (baseURL.Scheme != "http" && baseURL.Scheme != "https") || baseURL.Hostname() == "" || baseURL.User != nil || baseURL.RawQuery != "" || baseURL.Fragment != "" {
		return fmt.Errorf("APP_BASE_URL must be an absolute http or https URL without credentials, query, or fragment")
	}
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("PORT must be between 1 and 65535")
	}
	switch c.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("LOG_LEVEL must be debug, info, warn, or error")
	}
	key, err := base64.StdEncoding.DecodeString(c.SessionKey)
	if err != nil || len(key) != 32 {
		return fmt.Errorf("SESSION_KEY must be base64 encoding of exactly 32 random bytes")
	}
	providers := strings.FieldsFunc(c.Providers, func(r rune) bool { return r == ',' || r == ' ' || r == ';' })
	if len(providers) == 0 {
		return fmt.Errorf("PROVIDERS is required")
	}
	for _, name := range providers {
		switch strings.TrimSpace(name) {
		case "local", "google":
		default:
			return fmt.Errorf("PROVIDERS contains an unsupported provider")
		}
	}
	return nil
}
