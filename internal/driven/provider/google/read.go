package google

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"sync"
	"unicode/utf8"

	"golang.org/x/sync/errgroup"

	"github/mridha/myjournal-app/internal/core/journal"
)

func checksum(data []byte) string { sum := md5.Sum(data); return hex.EncodeToString(sum[:]) }

func (s *Store) Get(ctx context.Context, date journal.Date) (journal.Entry, error) {
	file, err := s.readEntryFile(ctx, date)
	if err != nil {
		return journal.Entry{}, err
	}
	if file.Md5Checksum == "" {
		return journal.Entry{}, journal.ErrUnavailable
	}
	response, err := retryRead(ctx, func() (*http.Response, error) { return s.service.Files.Get(file.Id).Context(ctx).Download() })
	if err != nil {
		return journal.Entry{}, mapError(err)
	}
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(response.Body, maxContentBytes+1))
	if err != nil {
		return journal.Entry{}, mapError(err)
	}
	if len(body) > maxContentBytes {
		return journal.Entry{}, journal.ErrUnavailable
	}
	if !utf8.Valid(body) {
		return journal.Entry{}, journal.ErrInvalidText
	}
	if checksum(body) != file.Md5Checksum {
		return journal.Entry{}, journal.ErrUnavailable
	}
	return journal.Entry{Date: date, Text: string(body), Version: journal.Version(file.Md5Checksum), Exists: true}, nil
}

func (s *Store) Previews(ctx context.Context, dates []journal.Date, maxBytes int) (map[journal.Date]string, error) {
	out := make(map[journal.Date]string, len(dates))
	if len(dates) == 0 {
		return out, nil
	}
	// Direct callers get one operation view; decorators already supply one.
	scoped := s
	if scoped.discovery == nil {
		scoped = s.ForOperation().(*Store)
	}
	var mu sync.Mutex
	group, ctx := errgroup.WithContext(ctx)
	group.SetLimit(previewWorkers)
	seen := make(map[journal.Date]bool, len(dates))
	for _, date := range dates {
		if seen[date] {
			continue
		}
		seen[date] = true
		group.Go(func() error {
			entry, err := scoped.Get(ctx, date)
			if errors.Is(err, journal.ErrNotFound) {
				return nil
			}
			if err != nil {
				return err
			}
			mu.Lock()
			out[date] = journal.PreviewPrefix(entry.Text, maxBytes)
			mu.Unlock()
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return nil, err
	}
	return out, nil
}
