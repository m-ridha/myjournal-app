package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestLogOmitsSecrets(t *testing.T) {
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	defer slog.SetDefault(previous)
	req := httptest.NewRequest(http.MethodPost, "http://localhost/auth/callback?code=private-code&state=private-state", strings.NewReader("private-body"))
	req.Header.Set("Cookie", "mj_session=private-cookie")
	res := httptest.NewRecorder()
	RequestLog(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })).ServeHTTP(res, req)
	got := logs.String()
	for _, secret := range []string{"private-code", "private-state", "private-body", "private-cookie", "?code"} {
		if strings.Contains(got, secret) {
			t.Fatal("secret leaked into request log")
		}
	}
	if !strings.Contains(got, `"path":"/auth/callback"`) || !strings.Contains(got, `"status":204`) {
		t.Fatalf("missing request metadata: %s", got)
	}
}
