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

	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"myjournal-app/assets"
	"myjournal-app/internal/config"
	provider "myjournal-app/internal/core/connection"
	"myjournal-app/internal/core/journal"
	"myjournal-app/internal/driven/clock"
	"myjournal-app/internal/driven/provider/local"
	"myjournal-app/internal/driven/session"
	"myjournal-app/internal/driver/web/api"
	"myjournal-app/internal/driver/web/authsession"
	"myjournal-app/internal/driver/web/i18n"
	"myjournal-app/internal/testutil"
)

func TestSaveAndGetEntryRoundTrip(t *testing.T) {
	registry := provider.NewRegistry(local.New(testutil.LocalBaseURL, t.TempDir()))
	h := newTestServer(config.Config{SessionKey: "test-key"}, registry, localServiceFactory(registry), testCatalog(t), clock.System{})
	date := openapi_types.Date{Time: time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC)}

	body := strings.NewReader(`{"text":"hello journal","baseVersion":""}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/entries/2026-09-21", body)
	addLocalSession(t, req)
	res := httptest.NewRecorder()

	h.SaveEntry(res, req, date)
	if res.Code != http.StatusOK {
		t.Fatalf("SaveEntry() status = %d, want %d", res.Code, http.StatusOK)
	}

	var saved api.Entry
	err := json.Unmarshal(res.Body.Bytes(), &saved)
	require.NoError(t, err, "SaveEntry() body is not valid JSON: %v", err)
	if saved.Text != "hello journal" {
		t.Fatalf("SaveEntry() saved text = %q, want %q", saved.Text, "hello journal")
	}
	if !saved.Exists {
		t.Fatal("SaveEntry() saved entry should exist")
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/entries/2026-09-21", nil)
	addLocalSession(t, getReq)
	getRes := httptest.NewRecorder()
	h.GetEntry(getRes, getReq, date)
	if getRes.Code != http.StatusOK {
		t.Fatalf("GetEntry() status = %d, want %d", getRes.Code, http.StatusOK)
	}

	var loaded api.Entry
	err = json.Unmarshal(getRes.Body.Bytes(), &loaded)
	require.NoError(t, err, "GetEntry() body is not valid JSON: %v", err)
	if loaded.Text != "hello journal" {
		t.Fatalf("GetEntry() text = %q, want %q", loaded.Text, "hello journal")
	}
}

func TestGetEntryRequiresSession(t *testing.T) {
	registry := provider.NewRegistry(local.New(testutil.LocalBaseURL, t.TempDir()))
	h := newTestServer(config.Config{SessionKey: "test-key"}, registry, localServiceFactory(registry), testCatalog(t), clock.System{})
	date := openapi_types.Date{Time: time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC)}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/entries/2026-09-21", nil)
	res := httptest.NewRecorder()
	h.GetEntry(res, req, date)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("GetEntry() status = %d, want %d", res.Code, http.StatusUnauthorized)
	}
}

func TestSaveInvalidJSONUsesLocaleMessage(t *testing.T) {
	registry := provider.NewRegistry(local.New(testutil.LocalBaseURL, t.TempDir()))
	h := newTestServer(config.Config{SessionKey: "test-key"}, registry, localServiceFactory(registry), testCatalog(t), clock.System{})
	date := openapi_types.Date{Time: time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC)}
	req := httptest.NewRequest(http.MethodPut, "/api/v1/entries/2026-09-21", strings.NewReader("{invalid"))
	addLocalSession(t, req)
	res := httptest.NewRecorder()
	h.SaveEntry(res, req, date)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("SaveEntry() status = %d, want %d", res.Code, http.StatusBadRequest)
	}
	assertJSONError(t, res, http.StatusBadRequest, testCatalog(t).Text("errors.invalidRequest"))
}

func TestPreviewTextUsesFirstMeaningfulLine(t *testing.T) {
	registry := provider.NewRegistry(local.New(testutil.LocalBaseURL, t.TempDir()))
	h := newTestServer(config.Config{SessionKey: "test-key"}, registry, localServiceFactory(registry), testCatalog(t), clock.System{})
	date := openapi_types.Date{Time: time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC)}

	body := strings.NewReader(`{"text":"\n\n   My first line\nSecond line","baseVersion":""}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/entries/2026-09-22", body)
	addLocalSession(t, req)
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
	require.NoError(t, err, "provider.Store() error = %v", err)

	service := journal.NewService(store, clock.System{})
	page, err := service.ListEntries(context.Background(), 1, 40)
	require.NoError(t, err, "ListEntries() error = %v", err)
	if len(page.Items) != 1 || page.Items[0].Preview != "My first line" {
		t.Fatalf("ListEntries() preview = %#v, want %q", page.Items, "My first line")
	}
}

func localServiceFactory(registry *provider.Registry) ServiceFactory {
	return func(ctx context.Context, current provider.Session) (JournalService, error) {
		p, ok := registry.Get(current.ProviderID)
		if !ok {
			return nil, fmt.Errorf("session provider is not configured")
		}
		store, err := p.Store(ctx, current.Connection)
		if err != nil {
			return nil, err
		}
		return journal.NewService(store, clock.System{}), nil
	}
}

func addLocalSession(t *testing.T, req *http.Request) {
	t.Helper()
	encoded, err := session.Encode("test-key", provider.Session{
		Connection: provider.Connection{ProviderID: "local", AccountID: "dev", AccountEmail: "dev@localhost", Credentials: "local-dev-user"},
		IssuedAt:   time.Now().UTC(),
	})
	require.NoError(t, err, "session.Encode() error = %v", err)
	req.AddCookie(&http.Cookie{Name: authsession.CookieName, Value: encoded})
}

func testCatalog(t *testing.T) i18n.Catalog {
	t.Helper()
	catalog, err := i18n.Load(assets.FS, "locales/en.json")
	require.NoError(t, err, "i18n.Load() error = %v", err)
	return catalog
}

func TestListRejectsOutOfRangePagination(t *testing.T) {
	registry := provider.NewRegistry(local.New(testutil.LocalBaseURL, t.TempDir()))
	server := newTestServer(config.Config{SessionKey: "test-key"}, registry, localServiceFactory(registry), testCatalog(t), clock.System{})
	for _, tc := range []struct {
		name       string
		page, size int
	}{
		{"zero page", 0, 20}, {"page above maximum", journal.MaxPage + 1, 20}, {"size above maximum", 1, 101}, {"negative size", 1, -1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/api/v1/entries", nil)
			addLocalSession(t, request)
			response := httptest.NewRecorder()
			server.ListEntries(response, request, api.ListEntriesParams{Page: &tc.page, Size: &tc.size})
			assert.Equal(t, http.StatusBadRequest, response.Code)
		})
	}
}

func TestSaveRejectsMalformedUnicodeAndOversizedEnvelope(t *testing.T) {
	registry := provider.NewRegistry(local.New(testutil.LocalBaseURL, t.TempDir()))
	h := newTestServer(config.Config{SessionKey: "test-key"}, registry, localServiceFactory(registry), testCatalog(t), clock.System{})
	date := openapi_types.Date{Time: time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC)}
	for _, tc := range []struct {
		name, body string
		status     int
	}{
		{"invalid UTF-8", "{\"text\":\"\xff\",\"baseVersion\":\"\"}", 400},
		{"unpaired high surrogate", `{"text":"\ud800","baseVersion":""}`, 400},
		{"unpaired low surrogate", `{"text":"\udc00","baseVersion":""}`, 400},
		{"invalid surrogate pair", `{"text":"\ud800\u0041","baseVersion":""}`, 400},
		{"multiple JSON values", `{"text":"a"} {"text":"b"}`, 400},
		{"oversized envelope", `{"text":"` + strings.Repeat("a", MaxSaveRequestBytes) + `"}`, 413},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPut, "/api/v1/entries/2026-09-21", strings.NewReader(tc.body))
			addLocalSession(t, req)
			res := httptest.NewRecorder()
			h.SaveEntry(res, req, date)
			assert.Equal(t, tc.status, res.Code)
		})
	}
	for _, body := range []string{`{"text":"\ud83d\ude42"}`, `{"text":"\\ud800"}`, `{"text":"�"}`} {
		err := validateJSONUnicode([]byte(body))
		require.NoError(t, err, "valid Unicode rejected: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/entries/2026-09-21", nil)
	addLocalSession(t, req)
	res := httptest.NewRecorder()
	h.GetEntry(res, req, date)
	if !strings.Contains(res.Body.String(), `"exists":false`) {
		t.Fatal("rejected save created entry")
	}
}

func TestListEntriesUsesJournalService(t *testing.T) {
	for _, tc := range []struct {
		name   string
		err    error
		status int
	}{{"success", nil, 200}, {"unavailable", journal.ErrUnavailable, 503}} {
		t.Run(tc.name, func(t *testing.T) {
			stub := newMockJournalService(t)
			stub.EXPECT().ListEntries(mock.Anything, 2, 3).Return(journal.ListPage{Page: 2, PageSize: 3, Items: []journal.ListItem{{Date: journal.Date{Year: 2020, Month: 1, Day: 2}, Preview: "synthetic"}}}, tc.err).Once()
			registry := provider.NewRegistry(local.New(testutil.LocalBaseURL, t.TempDir()))
			factory := func(context.Context, provider.Session) (JournalService, error) { return stub, nil }
			server := newTestServer(config.Config{SessionKey: "test-key"}, registry, factory, testCatalog(t), clock.System{})
			request := httptest.NewRequest(http.MethodGet, "/api/v1/entries?page=2&size=3", nil)
			addLocalSession(t, request)
			response := httptest.NewRecorder()
			page, size := 2, 3
			server.ListEntries(response, request, api.ListEntriesParams{Page: &page, Size: &size})
			assert.Equal(t, tc.status, response.Code)
			if tc.err == nil {
				var result api.ListEntriesResponse
				err := json.Unmarshal(response.Body.Bytes(), &result)
				require.NoError(t, err)
				if len(result.Items) != 1 || result.Items[0].Preview != "synthetic" {
					t.Fatalf("response=%+v", result)
				}
			}
		})
	}
}

func TestGetEntryFreshReadDirective(t *testing.T) {
	for _, directive := range []string{"", "no-store", "max-age=0, No-Store"} {
		t.Run(directive, func(t *testing.T) {
			stub := newMockJournalService(t)
			date := journal.Date{Year: 2020, Month: 1, Day: 1}
			if directive == "" {
				stub.EXPECT().Open(mock.Anything, date).Return(journal.Entry{Date: date, Text: "cached", Exists: true}, nil).Once()
			} else {
				stub.EXPECT().OpenFresh(mock.Anything, date).Return(journal.Entry{Date: date, Text: "fresh", Exists: true}, nil).Once()
			}
			registry := provider.NewRegistry(local.New(testutil.LocalBaseURL, t.TempDir()))
			factory := func(context.Context, provider.Session) (JournalService, error) { return stub, nil }
			server := newTestServer(config.Config{SessionKey: "test-key"}, registry, factory, testCatalog(t), clock.System{})
			request := httptest.NewRequest(http.MethodGet, "/api/v1/entries/2020-01-01", nil)
			request.Header.Set("Cache-Control", directive)
			addLocalSession(t, request)
			response := httptest.NewRecorder()
			server.GetEntry(response, request, openapi_types.Date{Time: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)})
			assert.Equal(t, http.StatusOK, response.Code)
			assert.Equal(t, "no-store", response.Header().Get("Cache-Control"))
			var entry api.Entry
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &entry))
			want := "cached"
			if directive != "" {
				want = "fresh"
			}
			assert.Equal(t, want, entry.Text)
		})
	}
}
