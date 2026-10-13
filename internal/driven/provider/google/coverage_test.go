package google

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/synctest"

	"golang.org/x/oauth2"
	"google.golang.org/api/googleapi"

	provider "github/mridha/myjournal-app/internal/core/connection"
	"github/mridha/myjournal-app/internal/core/journal"
)

func TestProviderIdentityConfigurationAndErrorBoundaries(t *testing.T) {
	scopes := []string{"synthetic"}
	p := New(Config{Scopes: scopes})
	scopes[0] = "changed"
	if p.ID() != "google" || p.Name() != "Google" || !p.RequiresLoginFlow() || p.cfg.Scopes[0] != "synthetic" {
		t.Fatal("provider configuration not isolated")
	}
	if _, err := p.validate(context.Background(), "invalid-token", "synthetic-audience"); err == nil {
		t.Fatal("invalid identity token accepted")
	}
	for _, tc := range []struct{ input, errorWant error }{{context.Canceled, context.Canceled}, {context.DeadlineExceeded, context.DeadlineExceeded}, {&oauth2.RetrieveError{ErrorCode: "invalid_token"}, journal.ErrAuthRequired}, {&oauth2.RetrieveError{ErrorCode: "access_denied"}, journal.ErrAuthRequired}, {errors.New("synthetic"), journal.ErrUnavailable}} {
		if !errors.Is(authError(tc.input), tc.errorWant) {
			t.Fatal("authentication error misclassified")
		}
	}
	for _, tc := range []struct{ input, want error }{{nil, nil}, {context.Canceled, context.Canceled}, {context.DeadlineExceeded, context.DeadlineExceeded}, {&googleapi.Error{Code: 401}, journal.ErrAuthRequired}, {&googleapi.Error{Code: 404}, journal.ErrNotFound}, {&googleapi.Error{Code: 409}, journal.ErrConflict}, {&googleapi.Error{Code: 412}, journal.ErrConflict}, {&googleapi.Error{Code: 403, Errors: []googleapi.ErrorItem{{Reason: "insufficientPermissions"}}}, journal.ErrAuthRequired}, {&googleapi.Error{Code: 400}, journal.ErrUnavailable}} {
		if !errors.Is(mapError(tc.input), tc.want) {
			t.Fatal("Drive error misclassified")
		}
	}
}
func TestDriveLockWaitCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := New(Config{})
		unlock, err := p.lock(context.Background(), "synthetic")
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() { _, err := p.lock(ctx, "synthetic"); done <- err }()
		synctest.Wait()
		cancel()
		if err := <-done; !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
		unlock()
		gate := make(chan struct{}, 1)
		gate <- struct{}{}
		ctx, cancel = context.WithCancel(context.Background())
		go func() { _, err := acquire(ctx, gate); done <- err }()
		synctest.Wait()
		cancel()
		if err := <-done; !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
		<-gate
	})
}

type failedBody struct{}

func (failedBody) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }
func (failedBody) Close() error             { return io.ErrClosedPipe }
func TestTimingBodyReportsReadAndCloseFailures(t *testing.T) {
	var outcomes []string
	body := timingBody{ReadCloser: failedBody{}, finish: func(outcome string) { outcomes = append(outcomes, outcome) }}
	if _, err := body.Read(make([]byte, 1)); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatal(err)
	}
	if err := body.Close(); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatal(err)
	}
	if len(outcomes) != 2 || outcomes[0] != "body_error" || outcomes[1] != "body_error" {
		t.Fatal(outcomes)
	}
}
func TestDriveListingAndMutationOperationalFailures(t *testing.T) {
	ctx := context.Background()
	date := journal.Date{Year: 2020, Month: 1, Day: 1}
	for _, operation := range []string{"years", "year", "scoped-read", "put", "trash", "metadata", "folder-url"} {
		t.Run(operation, func(t *testing.T) {
			fail := false
			p, _, _ := collapseProvider(t, func(w http.ResponseWriter, r *http.Request) bool {
				if fail && r.Method == http.MethodGet {
					w.WriteHeader(http.StatusBadRequest)
					return true
				}
				return false
			})
			store := collapseStore(t, p, ctx)
			entry, err := store.Put(ctx, date, "synthetic", "")
			if err != nil {
				t.Fatal(err)
			}
			fail = true
			switch operation {
			case "years":
				_, err = store.ListYears(ctx)
			case "year":
				_, err = store.ListYear(ctx, 2020)
			case "scoped-read":
				_, err = store.ForOperation().Get(ctx, date)
			case "put":
				_, err = store.Put(ctx, date, "changed", entry.Version)
			case "trash":
				err = store.Trash(ctx, date)
			case "metadata":
				_, err = store.metadata(ctx, "missing")
			case "folder-url":
				_, err = p.FolderURL(ctx, collapseConnection())
			}
			if !errors.Is(err, journal.ErrUnavailable) {
				t.Fatalf("error=%v", err)
			}
		})
	}
	if _, err := New(Config{}).FolderURL(ctx, provider.Connection{}); !errors.Is(err, journal.ErrAuthRequired) {
		t.Fatal(err)
	}
	p, _, _ := collapseProvider(t, nil)
	if url, err := p.FolderURL(ctx, collapseConnection()); err != nil || url != "" {
		t.Fatal("missing folder must not be created")
	}
	store := collapseStore(t, p, ctx)
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := store.Put(canceled, date, "synthetic", ""); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if err := store.Trash(canceled, date); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := store.Put(ctx, journal.Date{}, "synthetic", ""); !errors.Is(err, journal.ErrInvalidDate) {
		t.Fatal(err)
	}
	if _, err := store.Put(ctx, date, strings.Repeat("x", maxContentBytes+1), ""); !errors.Is(err, journal.ErrUnavailable) {
		t.Fatal(err)
	}
}

func TestDriveConcurrentMetadataChangesRejectWrites(t *testing.T) {
	for _, change := range []string{"deleted", "unavailable", "checksum", "missing-checksum", "unverified-upload"} {
		t.Run(change, func(t *testing.T) {
			active := false
			p, _, _ := collapseProvider(t, func(w http.ResponseWriter, r *http.Request) bool {
				if !active {
					return false
				}
				metadata := r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/drive/v3/files/") && r.URL.Query().Get("alt") != "media"
				if change == "unverified-upload" && r.Method == http.MethodPatch {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"id":"synthetic"}`))
					return true
				}
				if !metadata {
					return false
				}
				w.Header().Set("Content-Type", "application/json")
				switch change {
				case "deleted":
					w.WriteHeader(404)
				case "unavailable":
					w.WriteHeader(400)
				case "checksum":
					_, _ = w.Write([]byte(`{"id":"synthetic","md5Checksum":"external-change"}`))
				case "missing-checksum":
					_, _ = w.Write([]byte(`{"id":"synthetic"}`))
				default:
					return false
				}
				return true
			})
			store := collapseStore(t, p, context.Background())
			date := journal.Date{Year: 2020, Month: 1, Day: 1}
			entry, err := store.Put(context.Background(), date, "synthetic", "")
			if err != nil {
				t.Fatal(err)
			}
			active = true
			_, err = store.Put(context.Background(), date, "changed", entry.Version)
			expected := journal.ErrUnavailable
			if change == "deleted" || change == "checksum" {
				expected = journal.ErrConflict
			}
			if !errors.Is(err, expected) {
				t.Fatalf("error=%v, want %v", err, expected)
			}
		})
	}
}
