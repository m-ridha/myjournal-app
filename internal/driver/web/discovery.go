package web

import (
	"encoding/xml"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

func registerDiscoveryRoutes(r chi.Router, baseURL string) {
	baseURL = strings.TrimRight(baseURL, "/")
	r.Get("/sitemap.xml", func(w http.ResponseWriter, _ *http.Request) {
		type sitemapURL struct {
			Loc string `xml:"loc"`
		}
		sitemap := struct {
			XMLName xml.Name     `xml:"urlset"`
			XMLNS   string       `xml:"xmlns,attr"`
			URLs    []sitemapURL `xml:"url"`
		}{
			XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
		}
		for _, locale := range []string{"en", "id"} {
			for _, page := range publicPages {
				sitemap.URLs = append(sitemap.URLs, sitemapURL{Loc: baseURL + localizedPublicPath(locale, page.path)})
			}
		}
		body, err := xml.Marshal(sitemap)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		_, _ = w.Write(append([]byte(xml.Header), body...))
	})
	r.Get("/robots.txt", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("User-agent: *\nAllow: /\n\nSitemap: " + baseURL + "/sitemap.xml\n"))
	})
}
