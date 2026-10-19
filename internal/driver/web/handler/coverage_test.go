package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	provider "myjournal-app/internal/core/connection"
	"myjournal-app/internal/core/journal"
	"myjournal-app/internal/driver/web/api"
	"myjournal-app/internal/driver/web/authsession"
)

func TestCalendarAndDeleteHTTPBoundaries(t *testing.T) {
	now := time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name, operation, month  string
		auth                    bool
		factoryError, errorWant error
		status                  int
	}{
		{name: "calendar", operation: "calendar", month: "2026-10", auth: true, status: 200},
		{name: "calendar auth", operation: "calendar", month: "2026-10", status: 401},
		{name: "invalid month", operation: "calendar", month: "bad", auth: true, status: 400},
		{name: "zero year", operation: "calendar", month: "0000-10", auth: true, status: 400},
		{name: "calendar unavailable", operation: "calendar", month: "2026-10", auth: true, errorWant: journal.ErrUnavailable, status: 503},
		{name: "factory unavailable", operation: "calendar", month: "2026-10", auth: true, factoryError: journal.ErrUnavailable, status: 503},
		{name: "delete", operation: "delete", auth: true, status: 204},
		{name: "delete auth", operation: "delete", status: 401},
		{name: "delete unavailable", operation: "delete", auth: true, errorWant: journal.ErrUnavailable, status: 503},
	} {
		t.Run(tc.name, func(t *testing.T) {
			codec, catalog, current := boundarySession(t, now)
			service := newMockJournalService(t)
			if tc.auth && tc.factoryError == nil && tc.status != 400 {
				if tc.operation == "delete" {
					service.EXPECT().Delete(mock.Anything, journal.Date{Year: 2026, Month: 10, Day: 13}).Return(tc.errorWant).Once()
				} else {
					service.EXPECT().Month(mock.Anything, 2026, time.October).Return(journal.MonthView{Year: 2026, Month: time.October, Weeks: [][]journal.Day{{{Date: journal.Date{Year: 2026, Month: 10, Day: 13}, HasEntry: true, InMonth: true, IsToday: true}}}}, tc.errorWant).Once()
				}
			}
			if tc.auth {
				expectBoundarySession(codec, catalog, current)
			}
			factory := func(context.Context, provider.Session) (JournalService, error) {
				return service, tc.factoryError
			}
			s := New(Options{Codec: codec}, catalog, factory, testCatalog(t), testClock(func() time.Time { return now }), newMockConnectionService(t))
			req := httptest.NewRequest(http.MethodGet, "/api/v1/synthetic", nil)
			if tc.auth {
				req.AddCookie(&http.Cookie{Name: authsession.CookieName, Value: "synthetic"})
			}
			res := httptest.NewRecorder()
			if tc.operation == "calendar" {
				s.GetCalendar(res, req, api.GetCalendarParams{Month: tc.month})
			} else {
				s.DeleteEntry(res, req, openapi_types.Date{Time: now})
			}
			if res.Code != tc.status {
				t.Fatalf("status=%d want=%d", res.Code, tc.status)
			}
			if tc.operation == "calendar" && tc.status == 200 {
				var result api.CalendarResponse
				err := json.Unmarshal(res.Body.Bytes(), &result)
				require.NoError(t, err)
				if result.Year != 2026 || len(result.Weeks) != 1 || !result.Weeks[0][0].HasEntry {
					t.Fatal("calendar response mapping lost day metadata")
				}
			}
		})
	}
}
func TestSettingsFailureAndHealthEnvelope(t *testing.T) {
	now := time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)
	codec, catalog, current := boundarySession(t, now)
	expectBoundarySession(codec, catalog, current)
	connections := newMockConnectionService(t)
	connections.EXPECT().Settings(mock.Anything, current).Return(provider.Settings{}, journal.ErrUnavailable).Once()
	s := New(Options{Codec: codec}, catalog, nil, testCatalog(t), testClock(func() time.Time { return now }), connections)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/settings", nil)
	req.AddCookie(&http.Cookie{Name: authsession.CookieName, Value: "synthetic"})
	res := httptest.NewRecorder()
	s.GetSettings(res, req)
	if res.Code != 503 {
		t.Fatal("settings failure became partial success")
	}
	res = httptest.NewRecorder()
	s.GetSettings(res, httptest.NewRequest(http.MethodGet, "/api/v1/settings", nil))
	if res.Code != 401 {
		t.Fatal("settings exposed without session")
	}
	res = httptest.NewRecorder()
	s.Healthz(res, req)
	if res.Code != 200 || res.Body.String() != `{"status":"ok"}` || res.Header().Get("Content-Type") != "application/json" {
		t.Fatal("health contract changed")
	}
}
