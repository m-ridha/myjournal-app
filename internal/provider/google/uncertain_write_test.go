package google

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"

	"github/mridha/myjournal-app/internal/journal"
)

func TestAppliedUploadWithFailedAcknowledgement(t *testing.T) {
	for _, create := range []bool{false, true} {
		name := "update"
		if create {
			name = "create"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			fake := &fakeDrive{files: map[string]*drive.File{}, bodies: map[string][]byte{}}
			var loseAcknowledgement atomic.Bool
			var uploads atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if loseAcknowledgement.Load() && strings.HasPrefix(r.URL.Path, "/upload/") {
					uploads.Add(1)
					applied := httptest.NewRecorder()
					fake.ServeHTTP(applied, r)
					if applied.Code != http.StatusOK {
						w.WriteHeader(applied.Code)
						return
					}
					w.WriteHeader(http.StatusServiceUnavailable)
					return
				}
				fake.ServeHTTP(w, r)
			}))
			t.Cleanup(server.Close)
			api, err := drive.NewService(ctx, option.WithHTTPClient(server.Client()), option.WithEndpoint(server.URL+"/drive/v3/"))
			if err != nil {
				t.Fatal(err)
			}
			lock := make(chan struct{}, 1)
			store := &Store{service: api, lock: func(ctx context.Context) (func(), error) { return acquire(ctx, lock) }}
			date := journal.Date{Year: 2020, Month: 1, Day: 1}
			original, err := store.Put(ctx, date, "initial synthetic content", "")
			if err != nil {
				t.Fatal(err)
			}
			base := original.Version
			if create {
				date.Day = 2 // Reuse existing folders; lose only the entry acknowledgement.
				base = ""
			}
			loseAcknowledgement.Store(true)
			const text = "applied synthetic content"
			entry, err := store.Put(ctx, date, text, base)
			if !errors.Is(err, journal.ErrUnavailable) || entry.Exists {
				t.Fatalf("Put with lost acknowledgement = (%+v, %v), want unavailable without success", entry, err)
			}
			if got := uploads.Load(); got != 1 {
				t.Fatalf("ambiguous %s uploads = %d, want 1", name, got)
			}
			observed, err := store.Get(ctx, date)
			if err != nil || !observed.Exists || observed.Text != text || observed.Version == "" || observed.Version == base {
				t.Fatalf("Get after lost acknowledgement = (%+v, %v), want applied text and fresh version", observed, err)
			}
			if _, err := store.Put(ctx, date, "stale retry", base); !errors.Is(err, journal.ErrConflict) {
				t.Errorf("Put with original base after applied %s = %v, want conflict", name, err)
			}
			if got := uploads.Load(); got != 1 {
				t.Errorf("stale retry uploads = %d, want original upload only", got)
			}
		})
	}
}
