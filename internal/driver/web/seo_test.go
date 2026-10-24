package web

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"

	"myjournal-app/internal/config"
	provider "myjournal-app/internal/core/connection"
	"myjournal-app/internal/driven/clock"
	localProvider "myjournal-app/internal/driven/provider/local"
	"myjournal-app/internal/testutil"
)

func TestPublicSEO(t *testing.T) {
	cfg := config.Config{BaseURL: "https://myjournal.ink/", SessionKey: "0123456789abcdef0123456789abcdef", CookieSecure: true}
	h := newTestRouter(cfg, provider.NewRegistry(), clock.System{}, nil)
	type testCase struct{ path, locale, canonical, title string }
	cases := []testCase{
		{"/", "en", "/en/", "Private Online Journal &amp; Personal Diary"},
		{"/en/", "en", "/en/", "Private Online Journal &amp; Personal Diary"},
		{"/id/", "id", "/id/", "Jurnal Online &amp; Buku Harian Pribadi"},
		{"/en/help", "en", "/en/help", "How to Use Your Google Drive Journal"},
		{"/id/help", "id", "/id/help", "Panduan Jurnal Pribadi di Google Drive"},
		{"/en/privacy", "en", "/en/privacy", "Privacy Policy"},
		{"/id/privacy", "id", "/id/privacy", "Kebijakan Privasi"},
		{"/en/terms", "en", "/en/terms", "Terms of Service"},
		{"/id/terms", "id", "/id/terms", "Ketentuan Layanan"},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodGet, "https://host-injection.example"+tc.path+"?code=untrusted-query", nil)
			req.Header.Set("X-Forwarded-Host", "proxy-injection.example")
			if tc.path != "/" { // Explicit URLs must win over a conflicting saved preference.
				cookieLocale := "id"
				if tc.locale == "id" {
					cookieLocale = "en"
				}
				req.AddCookie(&http.Cookie{Name: languageCookieName, Value: cookieLocale})
			}
			res := httptest.NewRecorder()
			h.ServeHTTP(res, req)
			require.Equal(t, http.StatusOK, res.Code)
			assert.Empty(t, res.Header().Get("X-Robots-Tag"))
			assert.Equal(t, tc.locale, res.Header().Get("Content-Language"))
			assert.Equal(t, "no-store", res.Header().Get("Cache-Control"))
			body := res.Body.String()

			// Verify the rendered HTML structure.
			assertUniquePageAssets(t, body)
			assert.Equal(t, 1, countHTMLTags(t, body, "html"))
			assert.Equal(t, 1, countHTMLTags(t, body, "head"))
			assert.Equal(t, 1, countHTMLTags(t, body, "title"))
			assert.Equal(t, 1, countHTMLTags(t, body, "body"))
			assert.Equal(t, 1, countMetaByName(t, body, "description"))

			// Verify SEO metadata.
			wantOGLocale, wantOGAlternate := "en_US", "id_ID"
			if tc.locale == "id" {
				wantOGLocale, wantOGAlternate = "id_ID", "en_US"
			}
			assert.Contains(t, body, `<html lang="`+tc.locale+`"`)
			assert.Contains(t, body, `<title>`+tc.title+` · myjournal</title>`)
			assert.Contains(t, body, `<link rel="canonical" href="https://myjournal.ink`+tc.canonical+`">`)
			assert.Contains(t, body, `<meta property="og:locale" content="`+wantOGLocale+`">`)
			assert.Contains(t, body, `<meta property="og:locale:alternate" content="`+wantOGAlternate+`">`)
			assert.Contains(t, body, `<meta property="og:url" content="https://myjournal.ink`+tc.canonical+`">`)
			assert.Contains(t, body, `property="og:image" content="https://myjournal.ink/static/banners/myjournal-og-`+tc.locale+`.png"`)
			assert.Contains(t, body, `property="og:image:type" content="image/png"`)
			assert.Contains(t, body, `property="og:image:width" content="1200"`)
			assert.Contains(t, body, `property="og:image:height" content="630"`)
			assert.Contains(t, body, `name="twitter:card" content="summary_large_image"`)

			imageRes := httptest.NewRecorder()
			h.ServeHTTP(imageRes, httptest.NewRequest(http.MethodGet, "/static/banners/myjournal-og-"+tc.locale+".png", nil))
			assert.Equal(t, http.StatusOK, imageRes.Code)

			assert.Equal(t, 1, countHTMLAttributes(t, body, "link", "rel", "canonical"))
			original := strings.TrimPrefix(tc.canonical, "/"+tc.locale)
			assert.Contains(t, body, `hreflang="x-default" href="https://myjournal.ink`+original+`"`)
			for _, path := range []string{"/help", "/privacy", "/terms"} {
				assert.NotContains(t, body, `href="`+path+`"`)
				assert.NotContains(t, body, `href="`+path+`#`)
			}
			for _, language := range supportedLocales {
				assert.Contains(t, body, `hreflang="`+language+`" href="https://myjournal.ink/`+language+original+`"`)
				assert.Contains(t, body, `href="/`+language+original+`" lang="`+language+`"`)
			}

			// Verify untrusted request values do not leak into the response.
			assert.NotContains(t, body, "host-injection")
			assert.NotContains(t, body, "proxy-injection")
			assert.NotContains(t, body, "untrusted-query")

			// Verify only the intended public-page scripts are included.
			assert.NotContains(t, body, "/static/js/app.js")
			assert.NotContains(t, body, "/static/js/calendar-picker.js")
			assert.NotContains(t, body, "/static/js/timezone.js")
			assert.Contains(t, body, "/static/js/draft-cleanup.js")
			assert.NotContains(t, body, "calendar.dayLabel")

			// The WebSite, WebApplication, and FAQPage structured data are only included on the landing page.
			if original == "/" {
				assert.Equal(t, 1, countMetaByName(t, body, "google-site-verification"))
				assert.Contains(t, body, `itemtype="https://schema.org/WebSite"`)
				assert.Contains(t, body, `itemprop="url" href="https://myjournal.ink/"`)
				assert.Contains(t, body, `itemtype="https://schema.org/WebApplication"`)
				assert.Contains(t, body, `itemtype="https://schema.org/FAQPage"`)
				assert.Equal(t, 6, countHTMLAttributes(t, body, "div", "itemtype", "https://schema.org/Question"))
				assert.Equal(t, 6, countHTMLAttributes(t, body, "div", "itemtype", "https://schema.org/Answer"))
			} else {
				assert.NotContains(t, body, `itemtype="https://schema.org/WebSite"`)
				assert.NotContains(t, body, `itemtype="https://schema.org/WebApplication"`)
				assert.NotContains(t, body, `itemtype="https://schema.org/FAQPage"`)
			}

			// A localized page stores the selected language preference securely.
			if tc.path != "/" {
				require.Len(t, res.Result().Cookies(), 1)
				cookie := res.Result().Cookies()[0]
				assert.Equal(t, tc.locale, cookie.Value)
				assert.True(t, cookie.Secure)
				assert.True(t, cookie.HttpOnly)
			}

			// Verify localized messages and links in the rendered document.
			doc, err := html.Parse(strings.NewReader(body))
			require.NoError(t, err)
			var visit func(*html.Node)
			visit = func(n *html.Node) {
				if n.Type == html.ElementNode && n.Data == "body" {
					for _, attr := range n.Attr {
						if attr.Key == "data-messages" {
							messages := map[string]string{}
							require.NoError(t, json.Unmarshal([]byte(attr.Val), &messages))
							assert.Len(t, messages, 2)
							assert.NotEmpty(t, messages["theme.switchToLight"])
						}
					}
				}
				if n.Type == html.ElementNode && n.Data == "a" {
					for _, attr := range n.Attr {
						if attr.Key != "href" || !strings.HasPrefix(attr.Val, "/"+tc.locale+"/") {
							continue
						}
						target, err := url.Parse(attr.Val)
						require.NoError(t, err)
						linked := httptest.NewRecorder()
						h.ServeHTTP(linked, httptest.NewRequest(http.MethodGet, target.Path, nil))
						assert.Equal(t, http.StatusOK, linked.Code, "broken localized link %s", attr.Val)
						if target.Fragment != "" {
							assert.Equal(t, 1, countHTMLAttributes(t, linked.Body.String(), "", "id", target.Fragment), "missing or duplicate anchor: %s", attr.Val)
						}
					}
				}
				for child := n.FirstChild; child != nil; child = child.NextSibling {
					visit(child)
				}
			}
			visit(doc)
		})
	}
}

// Tokenize the response without the tree parser's automatic HTML repairs.
func countHTMLTags(t *testing.T, body, tagName string) int {
	t.Helper()
	return countHTMLAttributes(t, body, tagName, "", "")
}

func countMetaByName(t *testing.T, body, name string) int {
	t.Helper()
	return countHTMLAttributes(t, body, "meta", "name", name)
}

func countHTMLAttributes(t *testing.T, body, tag, attribute, value string) int {
	t.Helper()
	tokenizer := html.NewTokenizer(strings.NewReader(body))
	count := 0
	for {
		switch tokenizer.Next() {
		case html.ErrorToken:
			require.ErrorIs(t, tokenizer.Err(), io.EOF)
			return count
		case html.StartTagToken, html.SelfClosingTagToken:
			token := tokenizer.Token()
			if tag != "" && token.Data != tag {
				continue
			}
			if attribute == "" {
				count++
				continue
			}
			for _, attr := range token.Attr {
				if attr.Key == attribute && attr.Val == value {
					count++
					break
				}
			}
		}
	}
}

func assertUniquePageAssets(t *testing.T, body string) {
	t.Helper()
	assert.Equal(t, 1, countHTMLAttributes(t, body, "a", "class", "mj-brand"))
	for _, script := range []string{"theme-init", "theme", "language", "draft-cleanup", "timezone", "calendar-picker", "app"} {
		count := countHTMLAttributes(t, body, "script", "src", "/static/js/"+script+".js")
		assert.LessOrEqual(t, count, 1, "duplicate script: %s", script)
	}
}

func TestPrivatePagesStayOutOfSearch(t *testing.T) {
	cfg := config.Config{BaseURL: testutil.LocalBaseURL, SessionKey: "0123456789abcdef0123456789abcdef"}
	h := newTestRouter(cfg, provider.NewRegistry(localProvider.New(testutil.LocalBaseURL, "")), clock.System{}, nil)
	for _, path := range []string{"/", "/entries", "/entries/2026-01-01", "/calendar", "/settings", "/sign-in", "/sign-in-failed", "/missing"} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			addLocalSessionCookie(t, req, cfg.SessionKey)
			res := httptest.NewRecorder()
			h.ServeHTTP(res, req)
			assertUniquePageAssets(t, res.Body.String())
			assert.Equal(t, "noindex", res.Header().Get("X-Robots-Tag"))
			assert.Equal(t, "no-store", res.Header().Get("Cache-Control"))
			assert.NotContains(t, res.Body.String(), `rel="canonical"`)
			assert.NotContains(t, res.Body.String(), `property="og:`)
			if path == "/" || path == "/calendar" {
				assert.Contains(t, res.Body.String(), "/static/js/timezone.js")
			}
			if path == "/" {
				assert.Contains(t, res.Body.String(), "/static/js/app.js")
			}
		})
	}
	for _, path := range []string{"/settings", "/entries", "/api/v1/entries", "/api/v1/unknown", "/id/entries", "/en/settings", "/fr/", "/id/help/extra"} {
		res := httptest.NewRecorder()
		h.ServeHTTP(res, httptest.NewRequest(http.MethodGet, path, nil))
		assert.Equal(t, "noindex", res.Header().Get("X-Robots-Tag"), path)
		assert.NotContains(t, res.Body.String(), `rel="canonical"`, path)
	}
}

func TestLocalizedLanguageReturn(t *testing.T) {
	cfg := config.Config{BaseURL: testutil.LocalBaseURL, SessionKey: "0123456789abcdef0123456789abcdef"}
	h := newTestRouter(cfg, provider.NewRegistry(), clock.System{}, nil)
	for _, tc := range []struct{ path, locale, want string }{
		{"/en/", "id", "/id/"}, {"/id/help", "en", "/en/help"},
		{"/en/privacy", "id", "/id/privacy"}, {"/id/terms", "en", "/en/terms"},
		{"https://evil.example/id/", "en", "/settings"}, {"/id/entries", "en", "/settings"},
		{"/id/help?code=private", "en", "/settings"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/settings/language", strings.NewReader(url.Values{"language": {tc.locale}, "returnTo": {tc.path}}.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			res := httptest.NewRecorder()
			h.ServeHTTP(res, req)
			assert.Equal(t, http.StatusSeeOther, res.Code)
			assert.Equal(t, tc.want, res.Header().Get("Location"))
		})
	}
}

func TestAuthenticatedLocalizedHomeRemainsPublic(t *testing.T) {
	cfg := config.Config{BaseURL: testutil.LocalBaseURL, SessionKey: "0123456789abcdef0123456789abcdef"}
	h := newTestRouter(cfg, provider.NewRegistry(localProvider.New(testutil.LocalBaseURL, "")), clock.System{}, nil)
	for _, path := range []string{"/en/", "/id/"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		addLocalSessionCookie(t, req, cfg.SessionKey)
		res := httptest.NewRecorder()
		h.ServeHTTP(res, req)
		assert.Equal(t, http.StatusOK, res.Code)
		assert.Empty(t, res.Header().Get("X-Robots-Tag"))
		assert.Contains(t, res.Body.String(), `href="/entries/today"`)
		assert.NotContains(t, res.Body.String(), `data-entry-editor`)
		assert.Contains(t, res.Body.String(), `rel="canonical"`)
	}
}

func TestRejectedLocalizedPostDoesNotSetPreference(t *testing.T) {
	cfg := config.Config{BaseURL: testutil.LocalBaseURL, SessionKey: "0123456789abcdef0123456789abcdef"}
	h := newTestRouter(cfg, provider.NewRegistry(), clock.System{}, nil)
	req := httptest.NewRequest(http.MethodPost, "/id/help", nil)
	req.Header.Set("Origin", "https://untrusted.example")
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	assert.Equal(t, http.StatusForbidden, res.Code)
	assert.Empty(t, res.Result().Cookies())
}
