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

	"github/mridha/myjournal-app/assets"
	"github/mridha/myjournal-app/internal/config"
	provider "github/mridha/myjournal-app/internal/core/connection"
	"github/mridha/myjournal-app/internal/core/journal"
	"github/mridha/myjournal-app/internal/driven/clock"
	"github/mridha/myjournal-app/internal/driven/provider/local"
	"github/mridha/myjournal-app/internal/driven/session"
	"github/mridha/myjournal-app/internal/driver/web/api"
	"github/mridha/myjournal-app/internal/driver/web/authsession"
	"github/mridha/myjournal-app/internal/driver/web/i18n"
	"github/mridha/myjournal-app/internal/testutil"
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
	addLocalSession(t, getReq)
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
	if err != nil {
		t.Fatalf("provider.Store() error = %v", err)
	}

	service := journal.NewService(store, clock.System{})
	page, err := service.ListEntries(context.Background(), 1, 40)
	if err != nil {
		t.Fatalf("ListEntries() error = %v", err)
	}
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
	if err != nil {
		t.Fatalf("session.Encode() error = %v", err)
	}
	req.AddCookie(&http.Cookie{Name: authsession.CookieName, Value: encoded})
}

func testCatalog(t *testing.T) i18n.Catalog {
	t.Helper()
	catalog, err := i18n.Load(assets.FS, "locales/en.json")
	if err != nil {
		t.Fatalf("i18n.Load() error = %v", err)
	}
	return catalog
}

func TestListRejectsOutOfRangePagination(t *testing.T) {
	registry := provider.NewRegistry(local.New(testutil.LocalBaseURL, t.TempDir()))
	server := newTestServer(config.Config{SessionKey: "test-key"}, registry, localServiceFactory(registry), testCatalog(t), clock.System{})
	for _, pair := range [][2]int{{0, 20}, {journal.MaxPage + 1, 20}, {1, 101}, {1, -1}} {
		page, size := pair[0], pair[1]
		request := httptest.NewRequest(http.MethodGet, "/api/v1/entries", nil)
		addLocalSession(t, request)
		response := httptest.NewRecorder()
		server.ListEntries(response, request, api.ListEntriesParams{Page: &page, Size: &size})
		if response.Code != http.StatusBadRequest {
			t.Fatalf("bounds=%v status=%d", pair, response.Code)
		}
	}
}

func TestSaveRejectsMalformedUnicodeAndOversizedEnvelope(t *testing.T) {
	registry := provider.NewRegistry(local.New(testutil.LocalBaseURL, t.TempDir()))
	h := newTestServer(config.Config{SessionKey: "test-key"}, registry, localServiceFactory(registry), testCatalog(t), clock.System{})
	date := openapi_types.Date{Time: time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC)}
	for _, tc := range []struct {
		body   string
		status int
	}{
		{"{\"text\":\"\xff\",\"baseVersion\":\"\"}", 400},
		{`{"text":"\ud800","baseVersion":""}`, 400}, {`{"text":"\udc00","baseVersion":""}`, 400}, {`{"text":"\ud800\u0041","baseVersion":""}`, 400},
		{`{"text":"a"} {"text":"b"}`, 400},
		{`{"text":"` + strings.Repeat("a", MaxSaveRequestBytes) + `"}`, 413},
	} {
		req := httptest.NewRequest(http.MethodPut, "/api/v1/entries/2026-09-21", strings.NewReader(tc.body))
		addLocalSession(t, req)
		res := httptest.NewRecorder()
		h.SaveEntry(res, req, date)
		if res.Code != tc.status {
			t.Fatalf("status=%d want=%d", res.Code, tc.status)
		}
	}
	for _, body := range []string{`{"text":"\ud83d\ude42"}`, `{"text":"\\ud800"}`, `{"text":"�"}`} {
		if err := validateJSONUnicode([]byte(body)); err != nil {
			t.Fatalf("valid Unicode rejected: %v", err)
		}
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/entries/2026-09-21", nil)
	addLocalSession(t, req)
	res := httptest.NewRecorder()
	h.GetEntry(res, req, date)
	if !strings.Contains(res.Body.String(), `"exists":false`) {
		t.Fatal("rejected save created entry")
	}
}

// Only the listing method is needed by this handler test. Embedding the contract
// leaves unexpected calls visible as test failures, without a real journal service.
type listServiceStub struct {
	JournalService
	calls      int
	page, size int
	err        error
}

var _ JournalService = (*listServiceStub)(nil)

func (s *listServiceStub) ListEntries(_ context.Context, page, size int) (journal.ListPage, error) {
	s.calls++
	s.page, s.size = page, size
	return journal.ListPage{Page: page, PageSize: size, Items: []journal.ListItem{{Date: journal.Date{Year: 2020, Month: 1, Day: 2}, Preview: "synthetic"}}}, s.err
}

func TestListEntriesAcceptsUseCaseStub(t *testing.T) {
	for _, tc := range []struct {
		name   string
		err    error
		status int
	}{{"success", nil, 200}, {"unavailable", journal.ErrUnavailable, 503}} {
		t.Run(tc.name, func(t *testing.T) {
			stub := &listServiceStub{err: tc.err}
			registry := provider.NewRegistry(local.New(testutil.LocalBaseURL, t.TempDir()))
			factory := func(context.Context, provider.Session) (JournalService, error) { return stub, nil }
			server := newTestServer(config.Config{SessionKey: "test-key"}, registry, factory, testCatalog(t), clock.System{})
			request := httptest.NewRequest(http.MethodGet, "/api/v1/entries?page=2&size=3", nil)
			addLocalSession(t, request)
			response := httptest.NewRecorder()
			page, size := 2, 3
			server.ListEntries(response, request, api.ListEntriesParams{Page: &page, Size: &size})
			if response.Code != tc.status || stub.calls != 1 || stub.page != 2 || stub.size != 3 {
				t.Fatalf("status=%d calls=%d page=%d size=%d", response.Code, stub.calls, stub.page, stub.size)
			}
			if tc.err == nil {
				var result api.ListEntriesResponse
				if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
					t.Fatal(err)
				}
				if len(result.Items) != 1 || result.Items[0].Preview != "synthetic" {
					t.Fatalf("response=%+v", result)
				}
			}
		})
	}
}
