package web

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"myjournal-app/internal/config"
	provider "myjournal-app/internal/core/connection"
	"myjournal-app/internal/driven/clock"
)

func TestPublicDiscoveryRoutes(t *testing.T) {
	t.Parallel()
	for _, baseURL := range []string{"https://myjournal-ink.up.railway.app", "http://localhost:3030/"} {
		t.Run(baseURL, func(t *testing.T) {
			cfg := config.Config{BaseURL: baseURL, SessionKey: "0123456789abcdef0123456789abcdef"}
			router := newTestRouter(cfg, provider.NewRegistry(), clock.System{}, nil)
			origin := baseURL
			if origin[len(origin)-1] == '/' {
				origin = origin[:len(origin)-1]
			}
			for _, path := range []string{"/sitemap.xml", "/robots.txt"} {
				res := httptest.NewRecorder()
				// A client-supplied host must never appear in discovery URLs.
				router.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "https://untrusted.example"+path, nil))
				if res.Code != http.StatusOK {
					t.Fatalf("GET %s status = %d, want 200", path, res.Code)
				}
				if len(res.Result().Cookies()) != 0 {
					t.Errorf("GET %s unexpectedly sets cookies", path)
				}
				if path == "/robots.txt" {
					want := "User-agent: *\nAllow: /\n\nSitemap: " + origin + "/sitemap.xml\n"
					if res.Body.String() != want || res.Header().Get("Content-Type") != "text/plain; charset=utf-8" {
						t.Errorf("GET robots.txt = %q (%q), want %q (text/plain; charset=utf-8)", res.Body.String(), res.Header().Get("Content-Type"), want)
					}
					continue
				}
				var sitemap struct {
					XMLName xml.Name
					URLs    []struct {
						Locations []string `xml:"loc"`
					} `xml:"url"`
				}
				err := xml.Unmarshal(res.Body.Bytes(), &sitemap)
				require.NoError(t, err, "decode sitemap: %v", err)
				want := []string{origin + "/", origin + "/help", origin + "/privacy", origin + "/terms"}
				if len(sitemap.URLs) != 4 {
					t.Fatalf("sitemap URL records = %d, want 4", len(sitemap.URLs))
				}
				var locations []string
				for i, record := range sitemap.URLs {
					if len(record.Locations) != 1 {
						t.Fatalf("sitemap URL record %d locations = %v, want exactly one", i, record.Locations)
					}
					locations = append(locations, record.Locations[0])
				}
				if !reflect.DeepEqual(locations, want) {
					t.Errorf("sitemap URLs = %v, want %v", locations, want)
				}
				if sitemap.XMLName.Local != "urlset" || sitemap.XMLName.Space != "http://www.sitemaps.org/schemas/sitemap/0.9" {
					t.Errorf("sitemap root = %v, want sitemap namespace urlset", sitemap.XMLName)
				}
				if got := res.Header().Get("Content-Type"); got != "application/xml; charset=utf-8" {
					t.Errorf("sitemap Content-Type = %q, want application/xml; charset=utf-8", got)
				}
			}
		})
	}
}
