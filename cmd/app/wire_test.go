package main

import (
	"testing"
	"time"

	"github/mridha/myjournal-app/internal/config"
)

func TestBuildApp(t *testing.T) {
	cfg := config.Config{
		BaseURL:      "http://localhost:3030",
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
		{BaseURL: "http://localhost:3030", Providers: "gogle"},
		{BaseURL: "http://localhost:3030", Providers: "google"},
	} {
		if _, err := newProviderRegistry(cfg); err == nil {
			t.Fatalf("newProviderRegistry(%#v) error = nil, want rejection", cfg)
		}
	}
	if registry, err := newProviderRegistry(config.Config{BaseURL: "http://localhost:3030", Providers: "local"}); err != nil || registry == nil {
		t.Fatalf("newProviderRegistry(local) = %v, %v; want nonnil registry", registry, err)
	}
}
