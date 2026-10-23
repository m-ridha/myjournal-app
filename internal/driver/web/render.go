package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"regexp"
	"strings"

	"myjournal-app/assets"
	"myjournal-app/internal/driver/web/i18n"
)

// recentEntriesLimit controls browser browsing; the API retains its own pagination bounds.
const recentEntriesLimit = 10

// Renderer owns an immutable catalog and parsed page templates.
type Renderer struct {
	catalog   i18n.Catalog
	templates *template.Template
	languages map[string]*Renderer
	version   string
}

// NewRenderer loads the embedded catalog and templates at startup.
func NewRenderer() (*Renderer, error) { return NewRendererWithVersion("") }

var releaseVersion = regexp.MustCompile(`^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$`)

// NewRendererWithVersion loads page assets with trusted build-time release metadata.
func NewRendererWithVersion(version string) (*Renderer, error) {
	if version != "" && !releaseVersion.MatchString(version) {
		return nil, fmt.Errorf("invalid build version")
	}
	renderer, err := loadRenderer(assets.FS)
	if err != nil {
		return nil, err
	}
	for _, localized := range renderer.languages {
		localized.version = version
	}
	return renderer, nil
}

func loadRenderer(source fs.FS) (*Renderer, error) {
	renderer, err := loadLocaleRenderer(source, "en")
	if err != nil {
		return nil, err
	}
	indonesian, err := loadLocaleRenderer(source, "id")
	if err != nil {
		return nil, err
	}
	renderer.languages = map[string]*Renderer{"en": renderer, "id": indonesian}
	return renderer, nil
}

func loadLocaleRenderer(source fs.FS, locale string) (*Renderer, error) {
	catalog, err := i18n.Load(source, "locales/"+locale+".json")
	if err != nil {
		return nil, err
	}
	templates, err := template.New("base").Funcs(template.FuncMap{"t": catalog.Text, "upper": strings.ToUpper}).ParseFS(source, "templates/layouts/*.html", "templates/pages/*.html")
	if err != nil {
		return nil, err
	}
	return &Renderer{catalog: catalog, templates: templates}, nil
}

type pageData struct {
	Public             *publicMetadata
	AppScripts         bool
	EntriesLimit       int
	EntriesDescription string
	Version            string
	Title              string
	TitleKey           string
	Description        string
	DescriptionKey     string
	SignedIn           bool
	GoogleLogin        bool
	LocalLogin         bool
	Theme              string
	Page               string
	Date               string
	Today              string
	IsToday            bool
	AccountName        string
	DraftScope         string
	FolderName         string
	FolderURL          string
	FolderUnavailable  bool
	Locale             string
	Messages           string
}

func (renderer *Renderer) renderPage(w http.ResponseWriter, data pageData) {
	renderer.renderPageStatus(w, http.StatusOK, data)
}

func (renderer *Renderer) renderPageStatus(w http.ResponseWriter, status int, data pageData) {
	data.EntriesLimit = recentEntriesLimit
	data.EntriesDescription = strings.ReplaceAll(renderer.catalog.Text("entries.recentDescription"), "{listLimit}", fmt.Sprint(data.EntriesLimit))
	data.Version = renderer.version
	data.Locale = renderer.catalog.Locale()
	data.AppScripts = data.SignedIn || data.Page == "settings" || data.Page == "editor" || data.Page == "entries" || data.Page == "calendar"
	if data.AppScripts {
		localeJSON, err := renderer.catalog.JSON()
		if err != nil {
			http.Error(w, renderer.catalog.Text("errors.internal"), http.StatusInternalServerError)
			return
		}
		data.Messages = localeJSON
	} else {
		// A map containing only string keys and values cannot fail JSON encoding.
		messages, _ := json.Marshal(map[string]string{
			"theme.switchToLight": renderer.catalog.Text("theme.switchToLight"),
			"theme.switchToDark":  renderer.catalog.Text("theme.switchToDark"),
		})
		data.Messages = string(messages)
	}
	if data.Title == "" && data.TitleKey != "" {
		data.Title = renderer.catalog.Text(data.TitleKey)
	}
	if data.DescriptionKey != "" {
		data.Description = renderer.catalog.Text(data.DescriptionKey)
	}
	if data.Description == "" {
		data.Description = renderer.catalog.Text("app.description")
	}
	if data.Public == nil {
		w.Header().Set("X-Robots-Tag", "noindex")
	}
	w.Header().Set("Content-Language", data.Locale)
	var body bytes.Buffer
	if err := renderer.templates.ExecuteTemplate(&body, "base", data); err != nil {
		http.Error(w, renderer.catalog.Text("errors.internal"), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(body.Bytes())
}
