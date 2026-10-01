package i18n

import (
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

func TestLoadCatalogAndFormatDate(t *testing.T) {
	catalog, err := Load(fstest.MapFS{
		"locales/en.json": {Data: []byte(`{"app.locale":"en","format.date":"Monday, 02 January 2006","hello.greeting":"Hello, {name}"}`)},
	}, "locales/en.json")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got := catalog.Text("hello.greeting"); got != "Hello, {name}" {
		t.Fatalf("Text() = %q, want greeting", got)
	}
	if got := catalog.Text("missing.key"); got != "missing.key" {
		t.Fatalf("missing Text() = %q, want key fallback", got)
	}
	if got := catalog.FormatDate(time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC)); got != "Thursday, 01 October 2026" {
		t.Fatalf("FormatDate() = %q, want localized date layout", got)
	}
	encoded, err := catalog.JSON()
	if err != nil {
		t.Fatalf("JSON() error = %v", err)
	}
	if !strings.Contains(encoded, `"hello.greeting":"Hello, {name}"`) {
		t.Fatalf("JSON() = %s, want flattened dotted key for browser lookups", encoded)
	}
}

func TestLoadRejectsMissingLocale(t *testing.T) {
	_, err := Load(fstest.MapFS{"locales/en.json": {Data: []byte(`{"app.name":"myjournal"}`)}}, "locales/en.json")
	if err == nil {
		t.Fatal("Load() error = nil, want missing locale error")
	}
}

func TestLoadRejectsNonStringMessage(t *testing.T) {
	_, err := Load(fstest.MapFS{"locales/en.json": {Data: []byte(`{"app.locale":"en","message.count":4}`)}}, "locales/en.json")
	if err == nil {
		t.Fatal("Load() error = nil, want non-string message error")
	}
}
