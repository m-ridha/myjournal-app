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
		type alternateLink struct {
			XMLName  xml.Name `xml:"xhtml:link"`
			Rel      string   `xml:"rel,attr"`
			Hreflang string   `xml:"hreflang,attr"`
			Href     string   `xml:"href,attr"`
		}
		type sitemapURL struct {
			Loc        string `xml:"loc"`
			Alternates []alternateLink
		}
		sitemap := struct {
			XMLName    xml.Name     `xml:"urlset"`
			XMLNS      string       `xml:"xmlns,attr"`
			XMLNSXHTML string       `xml:"xmlns:xhtml,attr"`
			URLs       []sitemapURL `xml:"url"`
		}{
			XMLNS:      "http://www.sitemaps.org/schemas/sitemap/0.9",
			XMLNSXHTML: "http://www.w3.org/1999/xhtml",
		}
		for _, locale := range supportedLocales {
			for _, page := range publicPages {
				alternates := make([]alternateLink, 0, len(supportedLocales)+1)
				for _, altLocale := range supportedLocales {
					alternates = append(alternates, alternateLink{
						Rel:      "alternate",
						Hreflang: altLocale,
						Href:     baseURL + localizedPublicPath(altLocale, page.path),
					})
				}
				alternates = append(alternates, alternateLink{
					Rel:      "alternate",
					Hreflang: "x-default",
					Href:     baseURL + page.path,
				})
				sitemap.URLs = append(sitemap.URLs, sitemapURL{
					Loc:        baseURL + localizedPublicPath(locale, page.path),
					Alternates: alternates,
				})
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
