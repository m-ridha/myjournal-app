package web

import (
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
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
	w := httptest.NewRecorder()
	testRenderer(t).renderNotFoundPage(w)
	if w.Code != 404 || !strings.HasPrefix(w.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("code=%d content-type=%q", w.Code, w.Header().Get("Content-Type"))
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
			if err != nil {
				t.Fatal(err)
			}
			for _, locale := range []string{"en", "id"} {
				w := httptest.NewRecorder()
				renderer.languages[locale].renderPage(w, pageData{Page: "help"})
				body := w.Body.String()
				start := strings.Index(body, `<footer`)
				end := strings.Index(body, `</footer>`)
				if start < 0 || end < start {
					t.Fatal("missing footer")
				}
				footer := body[start:end]
				if !strings.Contains(footer, version) || !strings.Contains(footer, `href="https://github.com/mridha-work"`) || !strings.Contains(footer, "2026") {
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
