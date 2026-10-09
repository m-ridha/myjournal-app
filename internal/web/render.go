package web

import (
	"html/template"
	"net/http"

	"github/mridha/myjournal-app/assets"
	"github/mridha/myjournal-app/internal/i18n"
)

var localeCatalog = mustLoadLocale()

var pageTemplates = template.Must(template.New("base").Funcs(template.FuncMap{
	"t": localeCatalog.Text,
}).ParseFS(
	assets.FS,
	"templates/layouts/*.html",
	"templates/pages/*.html",
))

func mustLoadLocale() i18n.Catalog {
	catalog, err := i18n.Load(assets.FS, "locales/en.json")
	if err != nil {
		panic(err)
	}
	return catalog
}

type pageData struct {
	Title          string
	TitleKey       string
	Description    string
	DescriptionKey string
	SignedIn       bool
	Theme          string
	Page           string
	Date           string
	Today          string
	IsToday        bool
	AccountName    string
	DraftScope     string
	FolderName     string
	Locale         string
	Messages       string
}

func renderPage(w http.ResponseWriter, data pageData) {
	localeJSON, err := localeCatalog.JSON()
	if err != nil {
		http.Error(w, localeCatalog.Text("errors.internal"), http.StatusInternalServerError)
		return
	}
	data.Locale = localeCatalog.Locale()
	data.Messages = localeJSON
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pageTemplates.ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, localeCatalog.Text("errors.internal"), http.StatusInternalServerError)
	}
}
