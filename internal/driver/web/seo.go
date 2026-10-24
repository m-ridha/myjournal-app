package web

import (
	"net/http"
	"strings"
)

// Only these informational pages participate in public search discovery.
var (
	supportedLocales = []string{"en", "id"}
	publicPages      = []struct{ path, page string }{
		{"/", "landing"}, {"/help", "help"}, {"/privacy", "privacy"}, {"/terms", "terms"},
	}
)

type publicMetadata struct {
	CanonicalURL       string
	EnglishURL         string
	IndonesianURL      string
	DefaultURL         string
	ImageURL           string
	OGLocale           string
	OGAlternateLocales []string
	Prefix             string
	EnglishPath        string
	IndonesianPath     string
}

func localizedPublicPath(locale, path string) string {
	return "/" + locale + path
}

func ogTerritoryLocale(locale string) string {
	if locale == "id" {
		return "id_ID"
	}
	return "en_US"
}

func ogLocales(current string) (primary string, alternates []string) {
	primary = ogTerritoryLocale(current)
	alternates = make([]string, 0, len(supportedLocales)-1)
	for _, candidate := range supportedLocales {
		if candidate == current {
			continue
		}
		alternates = append(alternates, ogTerritoryLocale(candidate))
	}
	return primary, alternates
}

// publicRoute recognizes only explicit language variants of public pages.
func publicRoute(path string) (locale, original string, ok bool) {
	for _, locale := range supportedLocales {
		for _, page := range publicPages {
			if path == localizedPublicPath(locale, page.path) {
				return locale, page.path, true
			}
		}
	}
	return "", "", false
}

func (renderer *Renderer) renderPublicPage(w http.ResponseWriter, baseURL string, data pageData) {
	baseURL = strings.TrimRight(baseURL, "/")
	locale := renderer.catalog.Locale()
	ogLocale, ogAlternates := ogLocales(locale)
	for _, page := range publicPages {
		if data.Page != page.page {
			continue
		}
		en := localizedPublicPath("en", page.path)
		id := localizedPublicPath("id", page.path)
		data.Public = &publicMetadata{
			CanonicalURL:       baseURL + localizedPublicPath(locale, page.path),
			EnglishURL:         baseURL + en,
			IndonesianURL:      baseURL + id,
			DefaultURL:         baseURL + page.path,
			ImageURL:           baseURL + "/static/banners/myjournal-og-" + locale + ".png",
			OGLocale:           ogLocale,
			OGAlternateLocales: ogAlternates,
			Prefix:             "/" + locale,
			EnglishPath:        en,
			IndonesianPath:     id,
		}
		data.TitleKey = "seo." + page.page + ".title"
		data.DescriptionKey = "seo." + page.page + ".description"
		break
	}
	renderer.renderPage(w, data)
}

// Search directives are discovery hints; authentication remains the access boundary.
func searchIndexing(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _, public := publicRoute(r.URL.Path)
		for _, page := range publicPages {
			public = public || r.URL.Path == page.path
		}
		if !public && !strings.HasPrefix(r.URL.Path, "/static/") && r.URL.Path != "/robots.txt" && r.URL.Path != "/sitemap.xml" {
			w.Header().Set("X-Robots-Tag", "noindex")
		}
		next.ServeHTTP(w, r)
	})
}
