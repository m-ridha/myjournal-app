package google

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"

	"github/mridha/myjournal-app/internal/core/journal"
	"github/mridha/myjournal-app/internal/driven/cache/journal"
	cacheAdapter "github/mridha/myjournal-app/internal/driven/cache/local/bigcache"
)

type benchmarkClock struct{}

func (benchmarkClock) Now() time.Time { return time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC) }

type benchmarkRequests struct {
	mu     sync.Mutex
	counts map[string]int
}

func (r *benchmarkRequests) handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		category := "upload"
		if req.Method == http.MethodGet {
			switch {
			case req.URL.Query().Get("alt") == "media":
				category = "download"
			case req.URL.Path == "/drive/v3/files":
				category = "list"
			default:
				category = "metadata"
			}
		}
		r.mu.Lock()
		r.counts[category]++
		r.mu.Unlock()
		next.ServeHTTP(w, req)
	})
}

// BenchmarkDriveBaseline measures real adapter/service/cache code with local HTTP.
// It excludes OAuth and real Drive/network latency; fixture setup is untimed.
func BenchmarkDriveBaseline(b *testing.B) {
	for _, scenario := range []string{"open", "changed-save", "unchanged-save", "entries-cold", "entries-warm", "entries-after-save", "entries-after-visible-save", "calendar-cold", "calendar-warm"} {
		b.Run(scenario, func(b *testing.B) {
			ctx := context.Background()
			fake := &fakeDrive{files: map[string]*drive.File{}, bodies: map[string][]byte{}, pageSize: 1000}
			requests := &benchmarkRequests{counts: make(map[string]int)}
			server := httptest.NewServer(requests.handler(fake))
			b.Cleanup(server.Close)
			api, err := drive.NewService(ctx, option.WithHTTPClient(server.Client()), option.WithEndpoint(server.URL+"/drive/v3/"))
			if err != nil {
				b.Fatal(err)
			}
			lock := make(chan struct{}, 1)
			store := &Store{service: api, lock: func(ctx context.Context) (func(), error) { return acquire(ctx, lock) }}
			text := strings.Repeat("synthetic benchmark text 🙂\n", 64)
			date := journal.Date{Year: 2020, Month: 1, Day: 1}
			if scenario == "entries-after-visible-save" {
				date.Day = 30
			}
			var current journal.Entry
			for day := 1; day <= 30; day++ {
				entry, err := store.Put(ctx, journal.Date{Year: 2020, Month: 1, Day: day}, text, "")
				if err != nil {
					b.Fatal(err)
				}
				if day == date.Day {
					current = entry
				}
			}
			var backend *cacheAdapter.Adapter
			b.Cleanup(func() {
				if backend != nil {
					if err := backend.Close(); err != nil {
						b.Error(err)
					}
				}
			})
			newService := func() *journal.Service {
				if backend != nil {
					if err := backend.Close(); err != nil {
						b.Fatal(err)
					}
				}
				var err error
				backend, err = cacheAdapter.New(2 * time.Minute)
				if err != nil {
					b.Fatal(err)
				}
				return journal.NewService(cachestore.New(backend, 2*time.Minute).Wrap("synthetic", store), benchmarkClock{})
			}
			service := newService()
			page := func() {
				result, err := service.Page(ctx, 1, 20)
				if err != nil || len(result.Items) != 20 {
					b.Fatalf("page count/err: %d %v", len(result.Items), err)
				}
			}
			calendar := func() {
				_, err := service.Month(ctx, 2020, time.January)
				if err != nil {
					b.Fatal(err)
				}
			}
			if scenario == "entries-warm" || (scenario == "entries-after-save" || scenario == "entries-after-visible-save") {
				page()
			}
			if scenario == "calendar-warm" {
				calendar()
			}
			requests.counts = make(map[string]int)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				switch scenario {
				case "open":
					result, err := service.Open(ctx, date)
					if err != nil || result.Text != current.Text {
						b.Fatalf("open mismatch: %v", err)
					}
				case "changed-save", "unchanged-save", "entries-after-save", "entries-after-visible-save":
					content := current.Text
					if scenario != "unchanged-save" {
						if i%2 == 0 {
							content = text + "A"
						} else {
							content = text + "B"
						}
					}
					current, err = service.Save(ctx, date, content, current.Version)
					if err != nil {
						b.Fatal(err)
					}
					if scenario == "entries-after-save" || scenario == "entries-after-visible-save" {
						page()
					}
				case "entries-cold":
					service = newService()
					page()
				case "entries-warm":
					page()
				case "calendar-cold":
					service = newService()
					calendar()
				case "calendar-warm":
					calendar()
				}
			}
			b.StopTimer()
			total := 0
			for category, count := range requests.counts {
				b.ReportMetric(float64(count)/float64(b.N), category+"/op")
				total += count
			}
			b.ReportMetric(float64(total)/float64(b.N), "http/op")
		})
	}
}
