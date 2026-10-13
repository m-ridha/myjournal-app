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
	"github.com/stretchr/testify/require"

	"github/m-ridha/myjournal-app/internal/config"
	provider "github/m-ridha/myjournal-app/internal/core/connection"
	"github/m-ridha/myjournal-app/internal/core/journal"
	"github/m-ridha/myjournal-app/internal/driven/clock"
	"github/m-ridha/myjournal-app/internal/driven/provider/local"
	"github/m-ridha/myjournal-app/internal/driver/web/api"
	"github/m-ridha/myjournal-app/internal/testutil"
)

func assertJSONError(t *testing.T, res *httptest.ResponseRecorder, status int, message string) {
	t.Helper()
	if res.Code != status || res.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("status=%d content-type=%q body=%s", res.Code, res.Header().Get("Content-Type"), res.Body.String())
	}
	var body api.Error
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil || body.Message != message {
		t.Fatalf("error body=%s err=%v", res.Body.String(), err)
	}
}

func TestSaveRequiredFieldsPreserveExistingEntry(t *testing.T) {
	registry := provider.NewRegistry(local.New(testutil.LocalBaseURL, t.TempDir()))
	h := newTestServer(config.Config{SessionKey: "test-key"}, registry, localServiceFactory(registry), testCatalog(t), clock.System{})
	date := openapi_types.Date{Time: time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC)}
	save := func(body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodPut, "/api/v1/entries/2026-09-21", strings.NewReader(body))
		addLocalSession(t, r)
		w := httptest.NewRecorder()
		h.SaveEntry(w, r, date)
		return w
	}
	first := save(`{"text":"original","baseVersion":""}`)
	var entry api.Entry
	if first.Code != 200 {
		t.Fatal(first.Body.String())
	}
	err := json.Unmarshal(first.Body.Bytes(), &entry)
	require.NoError(t, err)
	for _, body := range []string{`null`, `{}`, `{"text":"replacement"}`, `{"text":null,"baseVersion":"` + entry.Version + `"}`, `{"baseVersion":"` + entry.Version + `"}`, `{"text":"replacement","baseVersion":null}`, `{"text":42,"baseVersion":""}`} {
		t.Run(body, func(t *testing.T) { assertJSONError(t, save(body), 400, testCatalog(t).Text("errors.invalidRequest")) })
	}
	r := httptest.NewRequest(http.MethodGet, "/api/v1/entries/2026-09-21", nil)
	addLocalSession(t, r)
	w := httptest.NewRecorder()
	h.GetEntry(w, r, date)
	var loaded api.Entry
	err = json.Unmarshal(w.Body.Bytes(), &loaded)
	require.NoError(t, err)
	if loaded.Text != entry.Text || loaded.Version != entry.Version {
		t.Fatal("invalid request changed saved entry")
	}
	assertJSONError(t, save(`{"text":"stale","baseVersion":""}`), 409, testCatalog(t).Text("errors.conflict"))
	updated := save(fmt.Sprintf(`{"text":"","baseVersion":%q}`, entry.Version))
	if updated.Code != 200 {
		t.Fatalf("explicit empty text rejected: %s", updated.Body.String())
	}
}

func TestCalendarRejectsInvalidMonth(t *testing.T) {
	registry := provider.NewRegistry(local.New(testutil.LocalBaseURL, t.TempDir()))
	h := newTestServer(config.Config{SessionKey: "test-key"}, registry, localServiceFactory(registry), testCatalog(t), clock.System{})
	for _, month := range []string{"", "2026-13", "2026-00", "2026-1", "2026-01-01", "0000-01", "garbage"} {
		t.Run(month, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/api/v1/calendar", nil)
			addLocalSession(t, r)
			w := httptest.NewRecorder()
			h.GetCalendar(w, r, api.GetCalendarParams{Month: month})
			assertJSONError(t, w, 400, testCatalog(t).Text("errors.invalidRequest"))
		})
	}
}

func TestWrappedServiceErrors(t *testing.T) {
	h := newTestServer(config.Config{}, nil, nil, testCatalog(t), clock.System{})
	for _, tc := range []struct {
		err    error
		status int
		key    string
	}{{ErrUnauthenticated, 401, "errors.unauthorized"}, {journal.ErrAuthRequired, 401, "errors.unauthorized"}, {journal.ErrUnavailable, 503, "errors.entryUnavailable"}, {context.DeadlineExceeded, 503, "errors.entryUnavailable"}, {journal.ErrConflict, 409, "errors.conflict"}, {journal.ErrNotFound, 404, "errors.entryUnavailable"}, {journal.ErrFutureDate, 400, "errors.futureDate"}, {journal.ErrInvalidPagination, 400, "errors.pagination"}, {journal.ErrInvalidText, 400, "errors.invalidText"}, {fmt.Errorf("private storage details"), 500, "errors.internal"}} {
		t.Run(tc.err.Error(), func(t *testing.T) {
			w := httptest.NewRecorder()
			h.writeServiceError(w, fmt.Errorf("adapter: %w", tc.err))
			assertJSONError(t, w, tc.status, testCatalog(t).Text(tc.key))
		})
	}
}
