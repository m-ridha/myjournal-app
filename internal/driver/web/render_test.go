package web

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

func TestRenderFailureDoesNotCommitPartialSuccess(t *testing.T) {
	renderer := testRenderer(t)
	renderer.templates = template.Must(template.New("base").Funcs(template.FuncMap{"fail": func() (string, error) { return "", errors.New("synthetic") }}).Parse(`partial{{fail}}`))
	w := httptest.NewRecorder()
	renderer.renderPage(w, pageData{})
	if w.Code != http.StatusInternalServerError || strings.Contains(w.Body.String(), "partial") {
		t.Fatalf("code=%d body=%q", w.Code, w.Body.String())
	}
}

func TestStyledNotFoundContentType(t *testing.T) {
	for _, signedIn := range []bool{false, true} {
		t.Run(fmt.Sprintf("signedIn=%t", signedIn), func(t *testing.T) {
			w := httptest.NewRecorder()
			testRenderer(t).renderNotFoundPage(w, signedIn)
			if w.Code != http.StatusNotFound || !strings.HasPrefix(w.Header().Get("Content-Type"), "text/html") {
				t.Fatalf("code=%d content-type=%q", w.Code, w.Header().Get("Content-Type"))
			}
			if navigation := strings.Contains(w.Body.String(), `class="mj-nav"`); navigation != signedIn {
				t.Fatalf("navigation shown = %t, want %t", navigation, signedIn)
			}
		})
	}
}

func TestSettingsFolderFailureIsVisible(t *testing.T) {
	w := httptest.NewRecorder()
	testRenderer(t).renderPage(w, pageData{Page: "settings", SignedIn: true, FolderName: "myjournal", FolderUnavailable: true})
	if !strings.Contains(w.Body.String(), "The journal folder is unavailable") || !strings.Contains(w.Body.String(), `href="/settings"`) {
		t.Fatal("missing truthful folder failure/retry")
	}
}

func TestRendererInitializationErrors(t *testing.T) {
	for name, source := range map[string]fstest.MapFS{
		"missing locale":     {},
		"malformed locale":   {"locales/en.json": {Data: []byte("{")}},
		"malformed template": {"locales/en.json": {Data: []byte(`{"app.locale":"en"}`)}, "templates/layouts/base.html": {Data: []byte("{{")}, "templates/pages/page.html": {Data: []byte("page")}},
	} {
		t.Run(name, func(t *testing.T) {
			if renderer, err := loadRenderer(source); err == nil || renderer != nil {
				t.Fatal("invalid assets accepted")
			}
		})
	}
}

func TestRendererInstancesEscapeUserText(t *testing.T) {
	first, second := testRenderer(t), testRenderer(t)
	if first == second || first.templates == second.templates {
		t.Fatal("renderer instances share mutable templates")
	}
	w := httptest.NewRecorder()
	first.renderPage(w, pageData{Page: "editor", Title: `<script>synthetic</script>`, SignedIn: true})
	if w.Code != 200 || strings.Contains(w.Body.String(), "<script>synthetic</script>") || !strings.Contains(w.Body.String(), "&lt;script&gt;synthetic&lt;/script&gt;") {
		t.Fatal("title was not escaped")
	}
}

func TestFooterBuildVersion(t *testing.T) {
	for _, version := range []string{"", "v1.2.3", "v0.3.0-staging", "v0.5.1-closed-beta", "v1.2.3-beta.1", "v1.2.3-beta.01"} {
		t.Run(version, func(t *testing.T) {
			renderer, err := NewRendererWithVersion(version)
			require.NoError(t, err)
			for _, locale := range supportedLocales {
				w := httptest.NewRecorder()
				renderer.languages[locale].renderPage(w, pageData{Page: "help"})
				body := w.Body.String()
				start := strings.Index(body, `<footer`)
				end := strings.Index(body, `</footer>`)
				if start < 0 || end < start {
					t.Fatal("missing footer")
				}
				footer := body[start:end]
				if !strings.Contains(footer, version) || !strings.Contains(footer, `href="https://github.com/m-ridha"`) || !strings.Contains(footer, "2026") {
					t.Errorf("%s footer missing build metadata", locale)
				}
				if strings.Contains(footer, `aria-hidden="true"> · </span>`) != (version != "") {
					t.Errorf("%s footer separator does not match version presence", locale)
				}
				if strings.Contains(footer, "/releases/tag/") {
					t.Errorf("%s footer links to private release", locale)
				}
			}
		})
	}
	for _, version := range []string{"dev", "dev-v0.3.1", "dev-v01.2.3", "dev-v1.2.3-beta", "v01.2.3", "v1.2.3-", "v1.2.3-beta..1", "<script>", "v1.2.3 -X other=value"} {
		if _, err := NewRendererWithVersion(version); err == nil {
			t.Errorf("accepted invalid version %q", version)
		}
	}
}

func TestEntriesLimitIsSharedWithLocalizedDescription(t *testing.T) {
	renderer := testRenderer(t)
	for _, locale := range supportedLocales {
		t.Run(locale, func(t *testing.T) {
			response := httptest.NewRecorder()
			renderer.languages[locale].renderPage(response, pageData{Page: "entries", SignedIn: true})
			body := response.Body.String()
			if response.Code != http.StatusOK || !strings.Contains(body, fmt.Sprintf(`data-list-limit="%d"`, recentEntriesLimit)) {
				t.Fatal("Entries must expose its limit to the browser")
			}
			description := strings.ReplaceAll(renderer.languages[locale].catalog.Text("entries.recentDescription"), "{listLimit}", fmt.Sprint(recentEntriesLimit))
			if !strings.Contains(body, description) || strings.Contains(body, "{listLimit}</") {
				t.Fatal("Entries must render the localized limit")
			}
		})
	}
}

func TestSignInRecoveryPageLocales(t *testing.T) {
	renderer := testRenderer(t)
	for locale, heading := range map[string]string{"en": "Sign-in wasn’t completed", "id": "Proses masuk belum selesai"} {
		t.Run(locale, func(t *testing.T) {
			response := httptest.NewRecorder()
			renderer.languages[locale].renderPage(response, pageData{Page: "signinfailed", TitleKey: "signInFailed.title"})
			body := response.Body.String()
			if response.Code != http.StatusOK || !strings.Contains(body, heading) || !strings.Contains(body, `href="/">`+renderer.languages[locale].catalog.Text("signInFailed.home")) {
				t.Fatal("missing localized recovery heading or home action")
			}
		})
	}
}
