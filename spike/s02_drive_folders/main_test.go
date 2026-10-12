package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

func TestEscapeQueryValue(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{"", ""},
		{"plain.txt", "plain.txt"},
		{"writer's.txt", `writer\'s.txt`},
		{`folder\file.txt`, `folder\\file.txt`},
		{`quinn's paper\essay`, `quinn\'s paper\\essay`},
		{`\'`, `\\\'`},
	} {
		t.Run(tc.input, func(t *testing.T) {
			if got := escapeQueryValue(tc.input); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestManualVisibilityDiagnostics(t *testing.T) {
	for _, tc := range []struct {
		name       string
		roots      []*drive.File
		manual     []*drive.File
		rootOnly   bool
		fileHidden bool
	}{
		{"expected visibility", []*drive.File{{Id: "root"}}, nil, true, true},
		{"manual root visible", []*drive.File{{Id: "root"}, {Id: "manual-root"}}, nil, false, true},
		{"manual file visible", []*drive.File{{Id: "root"}}, []*drive.File{{Id: "manual-file"}}, true, false},
		{"both visible", []*drive.File{{Id: "root"}, {Id: "manual-root"}}, []*drive.File{{Id: "manual-file"}}, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				var result any
				switch {
				case r.URL.Path == "/files/root":
					result = &drive.File{Id: "root", Name: "synthetic-root"}
				case strings.Contains(r.URL.Query().Get("q"), "mimeType"):
					result = &drive.FileList{Files: tc.roots}
				default:
					result = &drive.FileList{Files: tc.manual}
				}
				if err := json.NewEncoder(w).Encode(result); err != nil {
					t.Errorf("encode mock response: %v", err)
				}
			}))
			defer server.Close()
			ctx := context.Background()
			service, err := drive.NewService(ctx, option.WithEndpoint(server.URL+"/"), option.WithHTTPClient(server.Client()), option.WithoutAuthentication())
			if err != nil {
				t.Fatal(err)
			}
			exp := experiment{rootID: "root", yearID: "year", manualFileName: "synthetic.txt"}
			// The callback discards buffered output on failure, so the error itself
			// must carry both flags for its existing local failure log.
			err = exp.checkManualVisibility(ctx, service, io.Discard)
			if tc.rootOnly && tc.fileHidden {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected visibility mismatch")
			}
			for flag, value := range map[string]bool{
				"visible_root_matches_app_created_root_only": tc.rootOnly,
				"manual_file_invisible_to_app":               tc.fileHidden,
			} {
				want := flag + ": false"
				if value {
					want = flag + ": true"
				}
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q missing %q", err, want)
				}
			}
		})
	}
}

func TestLoginRejectsAtCapacity(t *testing.T) {
	t.Parallel()
	var exp experiment
	for i := range 128 {
		if !exp.pending.Add(fmt.Sprint(i), "verifier", time.Now()) {
			t.Fatal("could not fill pending login store")
		}
	}
	res := httptest.NewRecorder()
	exp.login(res, httptest.NewRequest(http.MethodGet, "/login", nil))
	if res.Code != http.StatusTooManyRequests || res.Header().Get("Location") != "" {
		t.Errorf("login at capacity = %d, Location %q, want 429 without redirect", res.Code, res.Header().Get("Location"))
	}
}

func TestCallbackRejectsExpiredStateBeforeExchange(t *testing.T) {
	t.Parallel()
	var exp experiment
	exp.pending.Add("expired", "verifier", time.Now().Add(-10*time.Minute))
	res := httptest.NewRecorder()
	exp.callback(res, httptest.NewRequest(http.MethodGet, "/auth/google/callback?state=expired&code=unused", nil))
	if res.Code != http.StatusBadRequest {
		t.Errorf("callback expired state status = %d, want 400", res.Code)
	}
}
