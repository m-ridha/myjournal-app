package main

import (
	"testing"
	"time"

	"github/mridha/myjournal-app/internal/config"
	"github/mridha/myjournal-app/internal/testutil"
)

func TestBuildApp(t *testing.T) {
	cfg := config.Config{
		BaseURL:      testutil.LocalBaseURL,
		Port:         3030,
		LogLevel:     "info",
		SessionKey:   "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		CookieSecure: false,
		Providers:    "local",
		LocalDataDir: ".devdata-test",
	}

	app, cleanup, err := BuildApp(cfg)
	if err != nil {
		t.Fatalf("BuildApp returned error: %v", err)
	}
	if app == nil {
		t.Fatal("BuildApp returned nil app")
	}
	if app.Server == nil {
		t.Fatal("BuildApp returned nil server")
	}
	if app.Server.ReadHeaderTimeout <= 0 || app.Server.ReadTimeout < app.Server.ReadHeaderTimeout || app.Server.WriteTimeout <= 30*time.Second || app.Server.IdleTimeout <= 0 {
		t.Fatal("HTTP deadlines must bound reads/idle connections and allow the 30-second handler deadline to respond")
	}
	if app.Logger == nil {
		t.Fatal("BuildApp returned nil logger")
	}
	if cleanup == nil {
		t.Fatal("BuildApp returned nil cleanup function")
	}
	cleanup()
}

func TestNewProviderRegistryFailsClosed(t *testing.T) {
	for _, cfg := range []config.Config{
		{BaseURL: "http://localhost.evil.com:3030", Providers: "local"},
		{BaseURL: testutil.LocalBaseURL, Providers: "gogle"},
		{BaseURL: testutil.LocalBaseURL, Providers: "google"},
	} {
		if _, err := newProviderRegistry(cfg); err == nil {
			t.Fatalf("newProviderRegistry(%#v) error = nil, want rejection", cfg)
		}
	}
	if registry, err := newProviderRegistry(config.Config{BaseURL: testutil.LocalBaseURL, Providers: "local"}); err != nil || registry == nil {
		t.Fatalf("newProviderRegistry(local) = %v, %v; want nonnil registry", registry, err)
	}
}

func TestGoogleIntegrationModeIsLocalOnly(t *testing.T) {
	t.Setenv("GOOGLE_CLIENT_ID", "test-client")
	t.Setenv("GOOGLE_CLIENT_SECRET", "test-secret")
	for _, base := range []string{"https://journal.example.com", "http://localhost.evil.com:3030"} {
		if _, err := newProviderRegistry(config.Config{BaseURL: base, Providers: "google", GoogleTestMode: true, GoogleClientID: "test-client", GoogleClientSecret: "test-secret"}); err == nil {
			t.Fatal("Google test mode accepted nonlocal deployment")
		}
	}
	registry, err := newProviderRegistry(config.Config{BaseURL: testutil.LocalBaseURL, Providers: "google", GoogleTestMode: true, GoogleClientID: "test-client", GoogleClientSecret: "test-secret"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := registry.Get("google"); !ok {
		t.Fatal("Google integration mode not registered")
	}
	if _, err := newProviderRegistry(config.Config{BaseURL: testutil.LocalBaseURL, Providers: "google,google", GoogleTestMode: true, GoogleClientID: "test-client", GoogleClientSecret: "test-secret"}); err == nil {
		t.Fatal("duplicate Google accepted")
	}
}

func TestGoogleHTTPSRegistration(t *testing.T) {
	for _, tc := range []struct {
		name   string
		origin string
		secure bool
		wantOK bool
	}{
		{"secure origin", "https://journal.example.com", true, true},
		{"insecure transport", "http://journal.example.com", true, false},
		{"insecure cookies", "https://journal.example.com", false, false},
		{"credentials", "https://user@journal.example.com", true, false},
		{"query", "https://journal.example.com?unexpected=1", true, false},
		{"fragment", "https://journal.example.com#unexpected", true, false},
		{"path", "https://journal.example.com/journal", true, false},
		{"missing host", "https://", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.Config{BaseURL: tc.origin, CookieSecure: tc.secure, Providers: "google", GoogleClientID: "test-client", GoogleClientSecret: "test-secret"}
			registry, err := newProviderRegistry(cfg)
			if (err == nil) != tc.wantOK {
				t.Fatalf("newProviderRegistry(origin=%q, secure=%t) error=%v, want success=%t", tc.origin, tc.secure, err, tc.wantOK)
			}
			if tc.wantOK {
				if _, ok := registry.Get("google"); !ok {
					t.Fatal("newProviderRegistry(HTTPS) did not register Google")
				}
			}
		})
	}
}

func TestJournalCacheLifecycle(t *testing.T) {
	cache, cleanup, err := newJournalCache()
	if err != nil {
		t.Fatal(err)
	}
	if cache == nil || cleanup == nil {
		t.Fatal("missing shared cache or cleanup")
	}
	cleanup()
	cleanup()
}
