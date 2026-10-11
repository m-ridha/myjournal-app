package web

import (
	"bytes"
	"html/template"
	"io/fs"
	"net/http"

	"github/mridha/myjournal-app/assets"
	"github/mridha/myjournal-app/internal/driver/web/i18n"
)

// Renderer owns an immutable catalog and parsed page templates.
type Renderer struct {
	catalog   i18n.Catalog
	templates *template.Template
}

func NewRenderer() (*Renderer, error) { return loadRenderer(assets.FS) }
func loadRenderer(source fs.FS) (*Renderer, error) {
	catalog, err := i18n.Load(source, "locales/en.json")
	if err != nil {
		return nil, err
	}
	templates, err := template.New("base").Funcs(template.FuncMap{"t": catalog.Text}).ParseFS(source, "templates/layouts/*.html", "templates/pages/*.html")
	if err != nil {
		return nil, err
	}
	return &Renderer{catalog: catalog, templates: templates}, nil
}

type pageData struct {
	Title             string
	TitleKey          string
	Description       string
	DescriptionKey    string
	SignedIn          bool
	GoogleLogin       bool
	LocalLogin        bool
	Theme             string
	Page              string
	Date              string
	Today             string
	IsToday           bool
	AccountName       string
	DraftScope        string
	FolderName        string
	FolderURL         string
	FolderUnavailable bool
	Locale            string
	Messages          string
}

func (renderer *Renderer) renderPage(w http.ResponseWriter, data pageData) {
	renderer.renderPageStatus(w, http.StatusOK, data)
}

func (renderer *Renderer) renderPageStatus(w http.ResponseWriter, status int, data pageData) {
	localeJSON, err := renderer.catalog.JSON()
	if err != nil {
		http.Error(w, renderer.catalog.Text("errors.internal"), http.StatusInternalServerError)
		return
	}
	data.Locale = renderer.catalog.Locale()
	data.Messages = localeJSON
	var body bytes.Buffer
	if err := renderer.templates.ExecuteTemplate(&body, "base", data); err != nil {
		http.Error(w, renderer.catalog.Text("errors.internal"), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(body.Bytes())
}
