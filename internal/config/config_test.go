package config

import (
	"encoding/base64"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"myjournal-app/internal/testutil"
)

// setConfigEnvironment clears all configuration environment variables for isolated test runs.
func setConfigEnvironment(t *testing.T) {
	t.Helper()
	for _, key := range []string{"APP_BASE_URL", "PORT", "LOG_LEVEL", "SESSION_KEY", "COOKIE_SECURE", "PROVIDERS", "LOCAL_DATA_DIR", "GOOGLE_TEST_MODE", "DEPLOY_TARGET", "CACHE_STORAGE"} {
		previous, existed := os.LookupEnv(key)
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("Unsetenv(%s) error = %v", key, err)
		}
		t.Cleanup(func() {
			if existed {
				_ = os.Setenv(key, previous)
			} else {
				_ = os.Unsetenv(key)
			}
		})
	}
}

func validConfig() Config {
	return Config{
		BaseURL:    testutil.LocalBaseURL,
		Port:       3030,
		LogLevel:   "info",
		SessionKey: base64.StdEncoding.EncodeToString(make([]byte, 32)),
		Providers:  "local",
	}
}

func TestLoadDefaults(t *testing.T) {
	setConfigEnvironment(t)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Port != 3030 || cfg.BaseURL != "http://localhost:3030" || cfg.LogLevel != "info" || cfg.Providers != "local" || cfg.DeployTarget != DeployTargetCloudflare || cfg.CacheStorage != CacheStorageLRU {
		t.Fatalf("Load() defaults = %#v", cfg)
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() accepted a missing SESSION_KEY")
	}
}

func TestValidateAcceptsValidConfiguration(t *testing.T) {
	if err := validConfig().Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestProviderIDsAreCanonical(t *testing.T) {
	for _, test := range []struct {
		name      string
		providers string
		want      []string
	}{
		{name: "plain", providers: "local", want: []string{"local"}},
		{name: "leading tab", providers: "\tlocal", want: []string{"local"}},
		{name: "trailing newline", providers: "local\n", want: []string{"local"}},
		{name: "mixed separators", providers: "google;\tlocal\n", want: []string{"google", "local"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := validConfig()
			cfg.Providers = test.providers
			assert.Equal(t, test.want, cfg.ProviderIDs())
			require.NoError(t, cfg.Validate())
		})
	}
}

func TestValidateRejectsInvalidConfiguration(t *testing.T) {
	cases := []struct {
		name   string
		change func(*Config)
	}{
		{name: "invalid session key", change: func(cfg *Config) { cfg.SessionKey = "known-default" }},
		{name: "empty providers", change: func(cfg *Config) { cfg.Providers = " , ; " }},
		{name: "unknown provider", change: func(cfg *Config) { cfg.Providers = "gogle" }},
		{name: "invalid base URL", change: func(cfg *Config) { cfg.BaseURL = "localhost:3030" }},
		{name: "invalid log level", change: func(cfg *Config) { cfg.LogLevel = "verbose" }},
		{name: "invalid deploy target", change: func(cfg *Config) { cfg.DeployTarget = "aws" }},
		{name: "invalid cache storage", change: func(cfg *Config) { cfg.CacheStorage = "redis" }},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			cfg := validConfig()
			test.change(&cfg)
			if err := cfg.Validate(); err == nil {
				t.Fatal("Validate() error = nil, want error")
			}
		})
	}
}

func TestLoadRejectsInvalidEnvironmentValues(t *testing.T) {
	cases := []struct {
		name  string
		key   string
		value string
	}{
		{name: "port", key: "PORT", value: "not-a-port"},
		{name: "secure cookie", key: "COOKIE_SECURE", value: "ture"},
		{name: "explicitly empty providers", key: "PROVIDERS", value: "  "},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			setConfigEnvironment(t)
			t.Setenv(test.key, test.value)
			cfg, err := Load()
			if err == nil {
				err = cfg.Validate()
			}
			if err == nil {
				t.Fatal("Load() error = nil, want error")
			}
		})
	}
}

func TestGoogleTestModeConfiguration(t *testing.T) {
	setConfigEnvironment(t)
	t.Setenv("GOOGLE_TEST_MODE", "true")
	cfg, err := Load()
	if err != nil || !cfg.GoogleTestMode {
		t.Fatal("test mode not loaded")
	}
	t.Setenv("GOOGLE_TEST_MODE", "invalid")
	if _, err := Load(); err == nil {
		t.Fatal("invalid test mode accepted")
	}
}

func TestDeployTargetAndCacheStorage(t *testing.T) {
	cases := []struct {
		name           string
		deployTarget   string
		cacheStorage   string
		wantTarget     string
		wantStorage    string
		wantEffTarget  string
		wantEffStorage string
	}{
		{
			name:           "default",
			wantTarget:     DeployTargetCloudflare,
			wantStorage:    CacheStorageLRU,
			wantEffTarget:  DeployTargetCloudflare,
			wantEffStorage: CacheStorageLRU,
		},
		{
			name:           "railway environment",
			deployTarget:   "railway",
			wantTarget:     DeployTargetRailway,
			wantStorage:    CacheStorageBigCache,
			wantEffTarget:  DeployTargetRailway,
			wantEffStorage: CacheStorageBigCache,
		},
		{
			name:           "railway with explicit lru override",
			deployTarget:   "railway",
			cacheStorage:   "lru",
			wantTarget:     DeployTargetRailway,
			wantStorage:    CacheStorageLRU,
			wantEffTarget:  DeployTargetRailway,
			wantEffStorage: CacheStorageLRU,
		},
		{
			name:           "cloudflare with explicit bigcache override",
			deployTarget:   "cloudflare",
			cacheStorage:   "bigcache",
			wantTarget:     DeployTargetCloudflare,
			wantStorage:    CacheStorageBigCache,
			wantEffTarget:  DeployTargetCloudflare,
			wantEffStorage: CacheStorageBigCache,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setConfigEnvironment(t)
			if tc.deployTarget != "" {
				t.Setenv("DEPLOY_TARGET", tc.deployTarget)
			}
			if tc.cacheStorage != "" {
				t.Setenv("CACHE_STORAGE", tc.cacheStorage)
			}
			cfg, err := Load()
			require.NoError(t, err)
			assert.Equal(t, tc.wantTarget, cfg.DeployTarget)
			assert.Equal(t, tc.wantStorage, cfg.CacheStorage)
			assert.Equal(t, tc.wantEffTarget, cfg.EffectiveDeployTarget())
			assert.Equal(t, tc.wantEffStorage, cfg.EffectiveCacheStorage())
			require.NoError(t, validConfig().Validate())
		})
	}
}

func TestEffectiveDefaultsOnEmptyFields(t *testing.T) {
	cfg := Config{}
	assert.Equal(t, DeployTargetCloudflare, cfg.EffectiveDeployTarget())
	assert.Equal(t, CacheStorageLRU, cfg.EffectiveCacheStorage())

	cfgRailway := Config{DeployTarget: DeployTargetRailway}
	assert.Equal(t, DeployTargetRailway, cfgRailway.EffectiveDeployTarget())
	assert.Equal(t, CacheStorageBigCache, cfgRailway.EffectiveCacheStorage())
}
