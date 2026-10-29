package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"myjournal-app/internal/config"
	"myjournal-app/internal/driver/web/authsession"
	"myjournal-app/internal/testutil"
)

func TestLocalServerRejectsUnexpectedAuthorityBeforeLogin(t *testing.T) {
	app, cleanup, err := BuildApp(config.Config{
		BaseURL:      testutil.LocalBaseURL,
		Port:         3030,
		SessionKey:   "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		Providers:    "local",
		LocalDataDir: t.TempDir(),
	})
	require.NoError(t, err)
	t.Cleanup(cleanup)

	for _, authority := range []string{"attacker.example:3030", "localhost.evil:3030", "localhost:4040", "127.0.0.1:3030"} {
		t.Run(authority, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "http://"+authority+"/api/v1/auth/local/callback", nil)
			req.Header.Set("Origin", "http://"+authority)
			req.Header.Set("Sec-Fetch-Site", "same-origin")
			req.Header.Set("X-Forwarded-Host", "localhost:3030")
			res := httptest.NewRecorder()
			app.Server.Handler.ServeHTTP(res, req)

			assert.Equal(t, http.StatusForbidden, res.Code)
			assert.Empty(t, res.Result().Cookies())
			assert.Empty(t, res.Header().Get("Location"))
		})
	}
}

func TestLocalServerProtectsJournalWithExistingSession(t *testing.T) {
	app, cleanup, err := BuildApp(config.Config{
		BaseURL:      testutil.LocalBaseURL,
		Port:         3030,
		SessionKey:   "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		Providers:    "local",
		LocalDataDir: t.TempDir(),
	})
	require.NoError(t, err)
	t.Cleanup(cleanup)

	login := httptest.NewRecorder()
	app.Server.Handler.ServeHTTP(login, httptest.NewRequest(http.MethodGet, testutil.LocalBaseURL+"/api/v1/auth/local/callback", nil))
	require.Equal(t, http.StatusFound, login.Code)
	cookies := login.Result().Cookies()
	require.Len(t, cookies, 1)
	require.Equal(t, authsession.CookieName, cookies[0].Name)

	request := func(method, authority, path, body string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, "http://"+authority+path, strings.NewReader(body))
		req.AddCookie(cookies[0])
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "http://"+authority)
		req.Header.Set("Sec-Fetch-Site", "same-origin")
		res := httptest.NewRecorder()
		app.Server.Handler.ServeHTTP(res, req)
		return res
	}

	const entryPath = "/api/v1/entries/2020-01-01"
	saved := request(http.MethodPut, "localhost:3030", entryPath, `{"text":"synthetic journal","baseVersion":""}`)
	require.Equal(t, http.StatusOK, saved.Code)
	var entry struct {
		Text    string `json:"text"`
		Version string `json:"version"`
	}
	require.NoError(t, json.Unmarshal(saved.Body.Bytes(), &entry))

	for _, tc := range []struct {
		method, path, body string
	}{
		{http.MethodGet, entryPath, ""},
		{http.MethodPut, entryPath, `{"text":"attacker replacement","baseVersion":"` + entry.Version + `"}`},
		{http.MethodDelete, entryPath, ""},
		{http.MethodPost, "/api/v1/logout", ""},
		{http.MethodGet, "/entries/today", ""},
		{http.MethodGet, "/id/help", ""},
		{http.MethodGet, "/static/js/app.js", ""},
	} {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			res := request(tc.method, "attacker.example:3030", tc.path, tc.body)
			assert.Equal(t, http.StatusForbidden, res.Code)
			assert.Empty(t, res.Result().Cookies())
			assert.Empty(t, res.Header().Get("Location"))
			assert.NotContains(t, res.Body.String(), "synthetic journal")
		})
	}

	read := request(http.MethodGet, "localhost:3030", entryPath, "")
	require.Equal(t, http.StatusOK, read.Code)
	require.NoError(t, json.Unmarshal(read.Body.Bytes(), &entry))
	assert.Equal(t, "synthetic journal", entry.Text)

	crossOrigin := httptest.NewRequest(http.MethodPost, testutil.LocalBaseURL+"/api/v1/logout", nil)
	crossOrigin.AddCookie(cookies[0])
	crossOrigin.Header.Set("Origin", "http://attacker.example:3030")
	crossOrigin.Header.Set("Sec-Fetch-Site", "cross-site")
	res := httptest.NewRecorder()
	app.Server.Handler.ServeHTTP(res, crossOrigin)
	assert.Equal(t, http.StatusForbidden, res.Code)
	assert.Empty(t, res.Result().Cookies())

	assert.Equal(t, http.StatusNoContent, request(http.MethodDelete, "localhost:3030", entryPath, "").Code)
}

func TestHTTPServerAuthorityPolicyFollowsConfiguredProviders(t *testing.T) {
	for _, tc := range []struct {
		name, providers string
		status          int
	}{
		{name: "local", providers: "local", status: http.StatusForbidden},
		{name: "mixed", providers: "google; local", status: http.StatusForbidden},
		{name: "Google only", providers: "google", status: http.StatusNoContent},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := newHTTPServer(config.Config{BaseURL: testutil.LocalBaseURL, Providers: tc.providers}, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			}))
			res := httptest.NewRecorder()
			server.Handler.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "http://attacker.example/", nil))
			assert.Equal(t, tc.status, res.Code)
		})
	}
}
