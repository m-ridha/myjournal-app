package config

import "testing"

func TestLoadExplicitEnvironmentAndRejectInvalidValues(t *testing.T) {
	setConfigEnvironment(t)
	for key, value := range map[string]string{"APP_BASE_URL": "https://example.invalid", "PORT": "8443", "LOG_LEVEL": "debug", "SESSION_KEY": "synthetic", "COOKIE_SECURE": "true", "PROVIDERS": "google", "LOCAL_DATA_DIR": "./synthetic", "GOOGLE_TEST_MODE": "true", "GOOGLE_CLIENT_ID": "synthetic-id", "GOOGLE_CLIENT_SECRET": "synthetic-secret"} {
		t.Setenv(key, value)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != 8443 || cfg.LogLevel != "debug" || !cfg.CookieSecure || !cfg.GoogleTestMode || cfg.BaseURL != "https://example.invalid" || cfg.Providers != "google" || cfg.LocalDataDir != "./synthetic" || cfg.GoogleClientID != "synthetic-id" || cfg.GoogleClientSecret != "synthetic-secret" || cfg.SessionKey != "synthetic" {
		t.Fatal("explicit startup configuration lost")
	}
	for _, tc := range []struct{ key, value string }{{"PORT", "bad"}, {"PORT", "0"}, {"PORT", "65536"}, {"COOKIE_SECURE", "bad"}, {"GOOGLE_TEST_MODE", "bad"}} {
		t.Run(tc.key+tc.value, func(t *testing.T) {
			t.Setenv(tc.key, tc.value)
			if _, err := Load(); err == nil {
				t.Fatal("invalid configuration accepted")
			}
		})
	}
}

func TestDevelopmentOriginBoundary(t *testing.T) {
	for _, origin := range []string{"http://localhost:3030", "http://127.0.0.1:3030", "http://[::1]:3030"} {
		if err := ValidateLoopbackURL(origin); err != nil {
			t.Fatal(err)
		}
	}
	for _, origin := range []string{"http://localhost.evil.invalid", "https://example.invalid", "http://user@localhost", "http://localhost?secret=x", "%"} {
		if err := ValidateLoopbackURL(origin); err == nil {
			t.Fatal("non-development origin accepted")
		}
	}
}
