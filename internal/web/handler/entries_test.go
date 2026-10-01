package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github/mridha/myjournal-app/assets"
	"github/mridha/myjournal-app/internal/clock"
	"github/mridha/myjournal-app/internal/config"
	"github/mridha/myjournal-app/internal/i18n"
	"github/mridha/myjournal-app/internal/journal"
	"github/mridha/myjournal-app/internal/provider"
	"github/mridha/myjournal-app/internal/provider/local"
	"github/mridha/myjournal-app/internal/web/api"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

func TestSaveAndGetEntryRoundTrip(t *testing.T) {
	registry := provider.NewRegistry(local.New("http://localhost:3030", t.TempDir()))
	h := New(config.Config{SessionKey: "test-key"}, registry, localServiceFactory(registry), testCatalog(t))
	date := openapi_types.Date{Time: time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC)}

	body := strings.NewReader(`{"text":"hello journal"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/entries/2026-09-21", body)
	res := httptest.NewRecorder()

	h.SaveEntry(res, req, date)
	if res.Code != http.StatusOK {
		t.Fatalf("SaveEntry() status = %d, want %d", res.Code, http.StatusOK)
	}

	var saved api.Entry
	if err := json.Unmarshal(res.Body.Bytes(), &saved); err != nil {
		t.Fatalf("SaveEntry() body is not valid JSON: %v", err)
	}
	if saved.Text != "hello journal" {
		t.Fatalf("SaveEntry() saved text = %q, want %q", saved.Text, "hello journal")
	}
	if !saved.Exists {
		t.Fatal("SaveEntry() saved entry should exist")
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/entries/2026-09-21", nil)
	getRes := httptest.NewRecorder()
	h.GetEntry(getRes, getReq, date)
	if getRes.Code != http.StatusOK {
		t.Fatalf("GetEntry() status = %d, want %d", getRes.Code, http.StatusOK)
	}

	var loaded api.Entry
	if err := json.Unmarshal(getRes.Body.Bytes(), &loaded); err != nil {
		t.Fatalf("GetEntry() body is not valid JSON: %v", err)
	}
	if loaded.Text != "hello journal" {
		t.Fatalf("GetEntry() text = %q, want %q", loaded.Text, "hello journal")
	}
}

func TestSaveInvalidJSONUsesLocaleMessage(t *testing.T) {
	registry := provider.NewRegistry(local.New("http://localhost:3030", t.TempDir()))
	h := New(config.Config{SessionKey: "test-key"}, registry, localServiceFactory(registry), testCatalog(t))
	date := openapi_types.Date{Time: time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC)}
	req := httptest.NewRequest(http.MethodPut, "/api/v1/entries/2026-09-21", strings.NewReader("{invalid"))
	res := httptest.NewRecorder()
	h.SaveEntry(res, req, date)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("SaveEntry() status = %d, want %d", res.Code, http.StatusBadRequest)
	}
	if got := strings.TrimSpace(res.Body.String()); got != testCatalog(t).Text("errors.invalidRequest") {
		t.Fatalf("SaveEntry() body = %q, want localized invalid request message", got)
	}
}

func TestPreviewTextUsesFirstMeaningfulLine(t *testing.T) {
	registry := provider.NewRegistry(local.New("http://localhost:3030", t.TempDir()))
	h := New(config.Config{SessionKey: "test-key"}, registry, localServiceFactory(registry), testCatalog(t))
	date := openapi_types.Date{Time: time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC)}

	body := strings.NewReader(`{"text":"\n\n   My first line\nSecond line"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/entries/2026-09-22", body)
	res := httptest.NewRecorder()
	h.SaveEntry(res, req, date)
	if res.Code != http.StatusOK {
		t.Fatalf("SaveEntry() status = %d, want %d", res.Code, http.StatusOK)
	}

	providerImpl, ok := registry.Get("local")
	if !ok {
		t.Fatal("registry.Get(\"local\") = false, want true")
	}
	store, err := providerImpl.Store(context.Background(), provider.Connection{ProviderID: "local", AccountID: "dev", AccountEmail: "dev@localhost", Credentials: "local-dev-user"})
	if err != nil {
		t.Fatalf("provider.Store() error = %v", err)
	}

	service := journal.NewService(store, clock.System{})
	page, err := service.Page(context.Background(), 1, 40)
	if err != nil {
		t.Fatalf("Page() error = %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].Preview != "My first line" {
		t.Fatalf("Page() preview = %#v, want %q", page.Items, "My first line")
	}
}

func localServiceFactory(registry *provider.Registry) ServiceFactory {
	return func(ctx context.Context) (*journal.Service, error) {
		p, ok := registry.Get("local")
		if !ok {
			return nil, fmt.Errorf("local provider is not configured")
		}
		store, err := p.Store(ctx, provider.Connection{ProviderID: "local", AccountID: "dev", AccountEmail: "dev@localhost", Credentials: "local-dev-user"})
		if err != nil {
			return nil, err
		}
		return journal.NewService(store, clock.System{}), nil
	}
}

func testCatalog(t *testing.T) i18n.Catalog {
	t.Helper()
	catalog, err := i18n.Load(assets.FS, "locales/en.json")
	if err != nil {
		t.Fatalf("i18n.Load() error = %v", err)
	}
	return catalog
}
