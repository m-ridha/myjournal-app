package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github/m-ridha/myjournal-app/internal/config"
	"github/m-ridha/myjournal-app/internal/testutil"
)

func TestBuildApp(t *testing.T) {
	cfg := config.Config{
		BaseURL:      testutil.LocalBaseURL,
		Port:         3030,
		LogLevel:     "info",
		SessionKey:   "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		CookieSecure: false,
		Providers:    "local",
		LocalDataDir: t.TempDir(),
	}

	app, cleanup, err := BuildApp(cfg)
	require.NoError(t, err)
	require.NotNil(t, cleanup)
	t.Cleanup(cleanup)
	require.NotNil(t, app)
	require.NotNil(t, app.Server)
	assert.Positive(t, app.Server.ReadHeaderTimeout)
	assert.GreaterOrEqual(t, app.Server.ReadTimeout, app.Server.ReadHeaderTimeout)
	assert.Greater(t, app.Server.WriteTimeout, 30*time.Second)
	assert.Positive(t, app.Server.IdleTimeout)
	assert.NotNil(t, app.Logger)
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

func TestHTTPListenAddress(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, providers, baseURL, want string
		testMode                       bool
	}{
		{name: "local", providers: "local", baseURL: "http://localhost:3030", want: "127.0.0.1:3030"},
		{name: "IPv6 local", providers: "local", baseURL: "http://[::1]:3030", want: "[::1]:3030"},
		{name: "mixed test providers", providers: "google; local", baseURL: "http://localhost:3030", testMode: true, want: "127.0.0.1:3030"},
		{name: "Google test", providers: "google", baseURL: "http://localhost:3030", testMode: true, want: "127.0.0.1:3030"},
		{name: "production Google", providers: "google", baseURL: "https://journal.example.com", want: ":3030"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.Config{Port: 3030, Providers: tc.providers, BaseURL: tc.baseURL, GoogleTestMode: tc.testMode}
			if got := newHTTPServer(cfg, nil).Addr; got != tc.want {
				t.Errorf("newHTTPServer(%s).Addr = %q, want %q", tc.name, got, tc.want)
			}
		})
	}
}
