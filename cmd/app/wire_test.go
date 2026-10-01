package main

import (
	"testing"

	"github/mridha/myjournal-app/internal/config"
)

func TestBuildApp(t *testing.T) {
	cfg := config.Config{
		BaseURL:      "http://localhost:3030",
		Port:         3030,
		LogLevel:     "info",
		SessionKey:   "test-session-key",
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
	if app.Logger == nil {
		t.Fatal("BuildApp returned nil logger")
	}
	if cleanup == nil {
		t.Fatal("BuildApp returned nil cleanup function")
	}
	cleanup()
}
