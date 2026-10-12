package i18n

import (
	"encoding/json"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github/mridha/myjournal-app/assets"
)

func TestEmbeddedTranslationsAreComplete(t *testing.T) {
	catalogs := make(map[string]map[string]string)
	for _, locale := range []string{"en", "id"} {
		raw, err := assets.FS.ReadFile("locales/" + locale + ".json")
		if err != nil {
			t.Fatal(err)
		}
		var messages map[string]string
		if err := json.Unmarshal(raw, &messages); err != nil {
			t.Fatal(err)
		}
		if messages["app.locale"] != locale {
			t.Errorf("app.locale = %q, want %q", messages["app.locale"], locale)
		}
		catalogs[locale] = messages
	}
	placeholders := regexp.MustCompile(`\{[a-zA-Z]+\}`)
	for locale, messages := range catalogs {
		if len(messages) != len(catalogs["en"]) {
			t.Errorf("%s message count = %d, want %d", locale, len(messages), len(catalogs["en"]))
		}
		for key, english := range catalogs["en"] {
			translated, ok := messages[key]
			if !ok || strings.TrimSpace(translated) == "" {
				t.Errorf("%s missing translation %s", locale, key)
			}
			expected := placeholders.FindAllString(english, -1)
			actual := placeholders.FindAllString(translated, -1)
			slices.Sort(expected)
			slices.Sort(actual)
			if !slices.Equal(expected, actual) {
				t.Errorf("%s %s placeholders = %v, want %v", locale, key, actual, expected)
			}
		}
	}
}

func TestIndonesianDate(t *testing.T) {
	catalog, err := Load(assets.FS, "locales/id.json")
	if err != nil {
		t.Fatal(err)
	}
	if got := catalog.FormatDate(time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC)); got != "Kamis, 01 Oktober 2026" {
		t.Errorf("FormatDate() = %q, want Kamis, 01 Oktober 2026", got)
	}
}
