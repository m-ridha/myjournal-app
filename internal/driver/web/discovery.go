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
		sitemap := struct {
			XMLName xml.Name `xml:"urlset"`
			XMLNS   string   `xml:"xmlns,attr"`
			URLs    []string `xml:"url>loc"`
		}{
			XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
			URLs:  []string{baseURL + "/", baseURL + "/help", baseURL + "/privacy", baseURL + "/terms"},
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
