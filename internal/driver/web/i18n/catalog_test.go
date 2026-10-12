package i18n

import (
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

func TestLoadCatalogAndFormatDate(t *testing.T) {
	catalog, err := Load(fstest.MapFS{
		"locales/en.json": {Data: []byte(`{"app.locale":"en","date.weekday.4":"Thursday","date.month.10":"October","hello.greeting":"Hello, {name}"}`)},
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
	if got := catalog.FormatDate(time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC)); got != "Thursday, October 1st 2026" {
		t.Fatalf("FormatDate() = %q, want English ordinal date", got)
	}
	encoded, err := catalog.JSON()
	if err != nil {
		t.Fatalf("JSON() error = %v", err)
	}
	if !strings.Contains(encoded, `"hello.greeting":"Hello, {name}"`) {
		t.Fatalf("JSON() = %s, want flattened dotted key for browser lookups", encoded)
	}
}

func TestFormatDateFallsBackToLayout(t *testing.T) {
	unconfigured, err := Load(fstest.MapFS{
		"locales/xx.json": {Data: []byte(`{"app.locale":"xx"}`)},
	}, "locales/xx.json")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got := unconfigured.FormatDate(time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC)); got != "Thursday, 01 October 2026" {
		t.Fatalf("FormatDate() = %q, want default layout date", got)
	}
}

func TestOrdinalDay(t *testing.T) {
	cases := []struct {
		day  int
		want string
	}{
		{1, "1st"},
		{2, "2nd"},
		{3, "3rd"},
		{4, "4th"},
		{11, "11th"},
		{12, "12th"},
		{13, "13th"},
		{21, "21st"},
		{22, "22nd"},
		{23, "23rd"},
		{31, "31st"},
	}
	for _, tc := range cases {
		if got := ordinalDay(tc.day); got != tc.want {
			t.Errorf("ordinalDay(%d) = %q, want %q", tc.day, got, tc.want)
		}
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
