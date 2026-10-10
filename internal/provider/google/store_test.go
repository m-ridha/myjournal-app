package google

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github/mridha/myjournal-app/internal/journal"
	"github/mridha/myjournal-app/internal/journal/storetest"
	"github/mridha/myjournal-app/internal/provider"
	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

type fakeDrive struct {
	mu         sync.Mutex
	files      map[string]*drive.File
	bodies     map[string][]byte
	next       int
	writes     int
	lists      int
	failReads  int
	failWrites int
}

var propRE = regexp.MustCompile(`appProperties has \{ key = '([^']+)' and value = '([^']+)' \}`)
var parentRE = regexp.MustCompile(`'([^']+)' in parents`)
var nameRE = regexp.MustCompile(`name = '([^']+)'`)

func (f *fakeDrive) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodGet && f.failReads > 0 {
		f.failReads--
		w.WriteHeader(503)
		return
	}
	if r.Method != http.MethodGet && f.failWrites > 0 {
		f.failWrites--
		f.writes++
		w.WriteHeader(503)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/drive/v3/files/")
	id = strings.TrimPrefix(id, "/upload/drive/v3/files/")
	if r.Method == http.MethodGet && r.URL.Path == "/drive/v3/files" {
		f.lists++
		query := r.URL.Query().Get("q")
		var matches []*drive.File
		for _, file := range f.files {
			if file.Trashed {
				continue
			}
			ok := true
			for _, prop := range propRE.FindAllStringSubmatch(query, -1) {
				if file.AppProperties[prop[1]] != prop[2] {
					ok = false
				}
			}
			if strings.Contains(query, "mimeType = '"+folderMIME+"'") && file.MimeType != folderMIME {
				ok = false
			}
			if strings.Contains(query, "mimeType = 'text/plain'") && file.MimeType != "text/plain" {
				ok = false
			}
			if m := parentRE.FindStringSubmatch(query); m != nil {
				found := false
				for _, parent := range file.Parents {
					if parent == m[1] {
						found = true
					}
				}
				if !found {
					ok = false
				}
			}
			if m := nameRE.FindStringSubmatch(query); m != nil && file.Name != m[1] {
				ok = false
			}
			if ok {
				matches = append(matches, file)
			}
		}
		sort.Slice(matches, func(i, j int) bool { return matches[i].Id < matches[j].Id })
		offset, _ := strconv.Atoi(r.URL.Query().Get("pageToken"))
		end := offset + 2
		if end > len(matches) {
			end = len(matches)
		}
		if offset > len(matches) {
			w.WriteHeader(400)
			return
		}
		next := ""
		if end < len(matches) {
			next = strconv.Itoa(end)
		}
		_ = json.NewEncoder(w).Encode(drive.FileList{Files: matches[offset:end], NextPageToken: next})
		return
	}
	if r.Method == http.MethodGet {
		file := f.files[id]
		if file == nil || file.Trashed {
			w.WriteHeader(404)
			return
		}
		if r.URL.Query().Get("alt") == "media" {
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write(f.bodies[id])
			return
		}
		w.Header().Set("ETag", file.Md5Checksum)
		_ = json.NewEncoder(w).Encode(file)
		return
	}
	f.writes++
	var metadata drive.File
	var body []byte
	mediaType, params, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if strings.HasPrefix(mediaType, "multipart/") {
		parts := multipart.NewReader(r.Body, params["boundary"])
		first, err := parts.NextPart()
		if err != nil {
			w.WriteHeader(400)
			return
		}
		_ = json.NewDecoder(first).Decode(&metadata)
		second, err := parts.NextPart()
		if err == nil {
			body, _ = io.ReadAll(second)
		}
	} else if strings.HasPrefix(mediaType, "application/json") {
		_ = json.NewDecoder(r.Body).Decode(&metadata)
	} else {
		body, _ = io.ReadAll(r.Body)
	}
	if r.Method == http.MethodPost {
		f.next++
		metadata.Id = fmt.Sprintf("file-%04d", f.next)
		metadata.CreatedTime = time.Unix(int64(f.next), 0).UTC().Format(time.RFC3339)
		if metadata.MimeType != "application/vnd.google-apps.folder" {
			metadata.Md5Checksum = checksum(body)
			f.bodies[metadata.Id] = body
		}
		f.files[metadata.Id] = &metadata
		_ = json.NewEncoder(w).Encode(&metadata)
		return
	}
	file := f.files[id]
	if file == nil || file.Trashed {
		w.WriteHeader(404)
		return
	}
	if tag := r.Header.Get("If-Match"); tag != "" && tag != file.Md5Checksum {
		w.WriteHeader(412)
		return
	}
	if metadata.Trashed {
		file.Trashed = true
	} else {
		file.Md5Checksum = checksum(body)
		f.bodies[id] = body
	}
	file.ModifiedTime = time.Now().UTC().Format(time.RFC3339Nano)
	_ = json.NewEncoder(w).Encode(file)
}
func testStore(t *testing.T) (*Store, *fakeDrive) {
	t.Helper()
	fake := &fakeDrive{files: map[string]*drive.File{}, bodies: map[string][]byte{}}
	server := httptest.NewServer(fake)
	t.Cleanup(server.Close)
	service, err := drive.NewService(context.Background(), option.WithHTTPClient(server.Client()), option.WithEndpoint(server.URL+"/drive/v3/"))
	if err != nil {
		t.Fatal(err)
	}
	service.BasePath = server.URL + "/drive/v3/"
	lock := make(chan struct{}, 1)
	return &Store{service: service, lock: func(ctx context.Context) (func(), error) { return acquire(ctx, lock) }}, fake
}
func TestDriveStoreContract(t *testing.T) {
	storetest.Run(t, func(t *testing.T) journal.Store { s, _ := testStore(t); return s })
}
func TestDriveReadsAndFailures(t *testing.T) {
	ctx := context.Background()
	date := journal.Date{Year: 2026, Month: 10, Day: 9}
	s, fake := testStore(t)
	_, err := s.Get(ctx, date)
	if !errors.Is(err, journal.ErrNotFound) || fake.writes != 0 {
		t.Fatal("read created files")
	}
	original, err := s.Put(ctx, date, "original", "")
	if err != nil {
		t.Fatal(err)
	}
	file, err := s.entryFile(ctx, date)
	if err != nil {
		t.Fatal(err)
	}
	fake.mu.Lock()
	fake.files[file.Id].Md5Checksum = ""
	fake.mu.Unlock()
	if _, err := s.Get(ctx, date); !errors.Is(err, journal.ErrUnavailable) {
		t.Fatal("missing checksum accepted")
	}
	fake.mu.Lock()
	fake.files[file.Id].Md5Checksum = string(original.Version)
	fake.failReads = 1
	fake.mu.Unlock()
	if _, err := s.Get(ctx, date); err != nil {
		t.Fatal("safe read did not retry", err)
	}
	before := fake.writes
	fake.mu.Lock()
	fake.failWrites = 1
	fake.mu.Unlock()
	if _, err := s.Put(ctx, date, "changed", original.Version); !errors.Is(err, journal.ErrUnavailable) {
		t.Fatal("failed write reported success")
	}
	if fake.writes != before+1 {
		t.Fatal("ambiguous mutation retried")
	}
	got, err := s.Get(ctx, date)
	if err != nil || got.Text != "original" {
		t.Fatal("failed mutation changed text")
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := s.Get(canceled, date); !errors.Is(err, context.Canceled) {
		t.Fatal("context not honored")
	}
}
func TestDriveRenamesDuplicatesAndPagination(t *testing.T) {
	ctx := context.Background()
	s, f := testStore(t)
	for day := 1; day <= 5; day++ {
		if _, err := s.Put(ctx, journal.Date{Year: 2026, Month: 10, Day: day}, "text", ""); err != nil {
			t.Fatal(err)
		}
	}
	root, err := s.root(ctx, false)
	if err != nil {
		t.Fatal(err)
	}
	year, err := s.year(ctx, 2026, false)
	if err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	f.files[root.Id].Name = "renamed-root"
	f.files[year.Id].Name = "renamed-year"
	f.files[year.Id].Parents = []string{"moved-elsewhere"}
	f.files["duplicate"] = &drive.File{Id: "duplicate", Name: "myjournal", MimeType: folderMIME, CreatedTime: "2099-01-01T00:00:00Z", AppProperties: map[string]string{"app": appTag, "role": "root"}}
	before := len(f.files)
	f.mu.Unlock()
	items, err := s.ListYear(ctx, 2026)
	if err != nil || len(items) != 5 {
		t.Fatalf("pagination/rename/move: %d %v", len(items), err)
	}
	picked, err := s.root(ctx, false)
	if err != nil || picked.Id != root.Id {
		t.Fatal("duplicate selection unstable")
	}
	if len(f.files) != before || f.files["duplicate"].Trashed {
		t.Fatal("duplicate removed")
	}
}
func TestDriveConcurrentCreate(t *testing.T) {
	s, f := testStore(t)
	var group sync.WaitGroup
	for i := 0; i < 5; i++ {
		group.Go(func() {
			_, err := s.Put(context.Background(), journal.Date{Year: 2026, Month: 10, Day: 9}, "same", "")
			if err != nil && !errors.Is(err, journal.ErrConflict) {
				t.Error(err)
			}
		})
	}
	group.Wait()
	if len(f.files) != 4 {
		t.Fatalf("created %d objects, want root/entries/year/file", len(f.files))
	}
}

func TestProviderStoresShareMutationLockAndCredentials(t *testing.T) {
	fake := &fakeDrive{files: map[string]*drive.File{}, bodies: map[string][]byte{}}
	p := New(Config{ClientID: "client", ClientSecret: "secret"})
	p.client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("Authorization") != "Bearer synthetic-access" {
			t.Error("missing token")
		}
		w := httptest.NewRecorder()
		fake.ServeHTTP(w, r)
		return w.Result(), nil
	})}
	encoded, _ := json.Marshal(credentials{Version: 1, Subject: "account", Refresh: "refresh"})
	connection := provider.Connection{ProviderID: "google", AccountID: "account", Credentials: string(encoded)}
	p.cacheToken("refresh", &oauth2.Token{AccessToken: "synthetic-access", Expiry: time.Now().Add(time.Hour)})
	var group sync.WaitGroup
	for i := 0; i < 5; i++ {
		group.Go(func() {
			store, err := p.Store(context.Background(), connection)
			if err != nil {
				t.Error(err)
				return
			}
			_, err = store.Put(context.Background(), journal.Date{Year: 2026, Month: 10, Day: 9}, "text", "")
			if err != nil && !errors.Is(err, journal.ErrConflict) {
				t.Error(err)
			}
		})
	}
	group.Wait()
	if len(fake.files) != 4 {
		t.Fatalf("per-request stores created %d objects", len(fake.files))
	}
	if link, err := p.FolderURL(context.Background(), connection); err != nil || !strings.HasPrefix(link, "https://drive.google.com/drive/folders/") {
		t.Fatal("folder link missing", err)
	}
	connection.AccountID = "other"
	if _, err := p.Store(context.Background(), connection); !errors.Is(err, journal.ErrAuthRequired) {
		t.Fatal("credentials escaped account binding")
	}
}

func TestDriveErrorMappingAndPreviewLimits(t *testing.T) {
	for _, tc := range []struct {
		code int
		want error
	}{{401, journal.ErrAuthRequired}, {404, journal.ErrNotFound}, {412, journal.ErrConflict}, {429, journal.ErrUnavailable}, {503, journal.ErrUnavailable}} {
		if !errors.Is(mapError(&googleapi.Error{Code: tc.code}), tc.want) {
			t.Fatalf("HTTP %d mapping", tc.code)
		}
	}
	s, f := testStore(t)
	ctx := context.Background()
	date := journal.Date{Year: 2026, Month: 10, Day: 9}
	text := strings.Repeat(" ", 65535) + "🙂end"
	if _, err := s.Put(ctx, date, text, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Previews(ctx, []journal.Date{date}, 20); err != nil {
		t.Fatal("split UTF-8 boundary rejected", err)
	}
	file, err := s.entryFile(ctx, date)
	if err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	f.bodies[file.Id] = []byte{0xff}
	f.mu.Unlock()
	if _, err := s.Previews(ctx, []journal.Date{date}, 20); !errors.Is(err, journal.ErrInvalidText) {
		t.Fatal("invalid preview source accepted")
	}
}

func TestDriveETagPreconditionIsForwarded(t *testing.T) {
	fake := &fakeDrive{files: map[string]*drive.File{}, bodies: map[string][]byte{}}
	p := New(Config{ClientID: "client", ClientSecret: "secret"})
	race := false
	p.client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if race && r.Method == http.MethodPatch && strings.HasPrefix(r.URL.Path, "/upload/") {
			if r.Header.Get("If-Match") == "" {
				t.Error("ETag not forwarded")
			}
			fake.mu.Lock()
			for id, file := range fake.files {
				if file.MimeType == "text/plain" {
					file.Md5Checksum = checksum([]byte("external"))
					fake.bodies[id] = []byte("external")
				}
			}
			fake.mu.Unlock()
		}
		w := httptest.NewRecorder()
		fake.ServeHTTP(w, r)
		return w.Result(), nil
	})}
	raw, _ := json.Marshal(credentials{Version: 1, Subject: "account", Refresh: "refresh"})
	p.cacheToken("refresh", &oauth2.Token{AccessToken: "synthetic", Expiry: time.Now().Add(time.Hour)})
	store, err := p.Store(context.Background(), provider.Connection{ProviderID: "google", AccountID: "account", Credentials: string(raw)})
	if err != nil {
		t.Fatal(err)
	}
	date := journal.Date{Year: 2026, Month: 10, Day: 9}
	entry, err := store.Put(context.Background(), date, "original", "")
	if err != nil {
		t.Fatal(err)
	}
	race = true
	if _, err := store.Put(context.Background(), date, "overwrite", entry.Version); !errors.Is(err, journal.ErrConflict) {
		t.Fatal("mocked precondition did not reject external race")
	}
	// This proves forwarding/mapping only; real Drive conditional-upload support
	// remains a live production gate, not an assertion made by this mock.
}

func TestPreviewsDiscoverFoldersOncePerYear(t *testing.T) {
	s, f := testStore(t)
	ctx := context.Background()
	dates := []journal.Date{{Year: 2026, Month: 10, Day: 1}, {Year: 2026, Month: 10, Day: 2}, {Year: 2026, Month: 10, Day: 3}}
	for _, d := range dates {
		if _, err := s.Put(ctx, d, "preview", ""); err != nil {
			t.Fatal(err)
		}
	}
	before := f.lists
	got, err := s.Previews(ctx, dates, 20)
	if err != nil || len(got) != 3 {
		t.Fatalf("previews=%v err=%v", got, err)
	}
	if calls := f.lists - before; calls > 4 {
		t.Fatalf("discovery requests=%d, want root + year + two file pages", calls)
	}
}
