package web

import (
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRenderFailureDoesNotCommitPartialSuccess(t *testing.T) {
	old := pageTemplates
	defer func() { pageTemplates = old }()
	pageTemplates = template.Must(template.New("base").Funcs(template.FuncMap{"fail": func() (string, error) { return "", errors.New("synthetic") }}).Parse(`partial{{fail}}`))
	w := httptest.NewRecorder()
	renderPage(w, pageData{})
	if w.Code != http.StatusInternalServerError || strings.Contains(w.Body.String(), "partial") {
		t.Fatalf("code=%d body=%q", w.Code, w.Body.String())
	}
}

func TestStyledNotFoundContentType(t *testing.T) {
	w := httptest.NewRecorder()
	renderNotFoundPage(w)
	if w.Code != 404 || !strings.HasPrefix(w.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("code=%d content-type=%q", w.Code, w.Header().Get("Content-Type"))
	}
}

func TestSettingsFolderFailureIsVisible(t *testing.T) {
	w := httptest.NewRecorder()
	renderPage(w, pageData{Page: "settings", FolderName: "myjournal", FolderUnavailable: true})
	if !strings.Contains(w.Body.String(), "The journal folder is unavailable") || !strings.Contains(w.Body.String(), `href="/settings"`) {
		t.Fatal("missing truthful folder failure/retry")
	}
}
