package local

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"cloud.google.com/go/civil"

	"myjournal-app/internal/core/journal"
)

// Local mutations share a bounded process-wide lock, including across Store instances.
// This does not coordinate separate processes or external filesystem writers.
var mutationLock = make(chan struct{}, 1)

func lockMutation(ctx context.Context) (func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	select {
	case mutationLock <- struct{}{}:
		return func() { <-mutationLock }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Store persists entries as plain text files on disk in the local dev directory.
type Store struct {
	root      string
	entryInfo func(os.DirEntry) (os.FileInfo, error)
}

var _ journal.Store = (*Store)(nil)

// NewStore constructs a filesystem store rooted in the development data directory.
func NewStore(root string) *Store {
	return &Store{root: root, entryInfo: os.DirEntry.Info}
}

func (s *Store) ListYears(ctx context.Context) ([]int, error) {
	if err := ctx.Err(); err != nil {
		return nil, storageError(err)
	}
	entries, err := os.ReadDir(filepath.Join(s.root, "entries"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, storageError(err)
	}
	years := make([]int, 0, len(entries))
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, storageError(err)
		}
		if !entry.IsDir() {
			continue
		}
		if year, err := strconv.Atoi(entry.Name()); err == nil {
			years = append(years, year)
		}
	}
	sort.Sort(sort.Reverse(sort.IntSlice(years)))
	return years, nil
}

func (s *Store) ListYear(ctx context.Context, year int) ([]journal.EntryMeta, error) {
	if err := ctx.Err(); err != nil {
		return nil, storageError(err)
	}
	path := s.yearDir(year)
	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, storageError(err)
	}
	meta := make([]journal.EntryMeta, 0, len(entries))
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, storageError(err)
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".txt") {
			continue
		}
		date, err := journal.ParseEntryDate(filepath.ToSlash(filepath.Join("entries", fmt.Sprintf("%04d", year), entry.Name())))
		if err != nil {
			continue
		}
		info, err := s.entryInfo(entry)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", journal.ErrUnavailable, err)
		}
		meta = append(meta, journal.EntryMeta{Date: date, ModifiedTime: info.ModTime()})
	}
	sort.Slice(meta, func(i, j int) bool {
		return meta[i].Date.After(meta[j].Date)
	})
	return meta, nil
}

func (s *Store) Get(ctx context.Context, d civil.Date) (journal.Entry, error) {
	if err := ctx.Err(); err != nil {
		return journal.Entry{}, storageError(err)
	}
	path := filepath.Join(s.yearDir(d.Year), d.String()+".txt")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return journal.Entry{}, journal.ErrNotFound
		}
		return journal.Entry{}, storageError(err)
	}
	if !utf8.Valid(data) {
		return journal.Entry{}, journal.ErrInvalidText
	}
	return journal.Entry{Date: d, Text: string(data), Version: journal.Version(hashText(string(data))), Exists: true}, nil
}

func (s *Store) Previews(ctx context.Context, dates []civil.Date, maxBytes int) (map[civil.Date]string, error) {
	out := make(map[civil.Date]string, len(dates))
	for _, d := range dates {
		entry, err := s.Get(ctx, d)
		if errors.Is(err, journal.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, storageError(err)
		}
		out[d] = journal.PreviewPrefix(entry.Text, maxBytes)
	}
	return out, nil
}

func (s *Store) Put(ctx context.Context, d civil.Date, text string, base journal.Version) (journal.Entry, error) {
	unlock, err := lockMutation(ctx)
	if err != nil {
		return journal.Entry{}, storageError(err)
	}
	defer unlock()
	if !utf8.ValidString(text) {
		return journal.Entry{}, journal.ErrInvalidText
	}
	if err := ctx.Err(); err != nil {
		return journal.Entry{}, storageError(err)
	}
	if err := ensureDir(filepath.Join(s.root, "entries")); err != nil {
		return journal.Entry{}, storageError(err)
	}
	if err := ensureDir(s.yearDir(d.Year)); err != nil {
		return journal.Entry{}, storageError(err)
	}
	path := filepath.Join(s.yearDir(d.Year), d.String()+".txt")
	_, statErr := os.Stat(path)
	switch {
	case statErr == nil:
		if base == "" {
			return journal.Entry{}, journal.ErrConflict
		}
		current, err := s.Get(ctx, d)
		if err != nil {
			return journal.Entry{}, storageError(err)
		}
		if current.Version != base {
			return journal.Entry{}, journal.ErrConflict
		}
	case os.IsNotExist(statErr):
		if base != "" {
			return journal.Entry{}, journal.ErrConflict
		}
	default:
		return journal.Entry{}, storageError(statErr)
	}
	if err := writeAtomically(ctx, path, []byte(text), os.Rename); err != nil {
		return journal.Entry{}, storageError(err)
	}
	return journal.Entry{Date: d, Text: text, Version: journal.Version(hashText(text)), Exists: true}, nil
}

func (s *Store) Trash(ctx context.Context, d civil.Date) error {
	unlock, err := lockMutation(ctx)
	if err != nil {
		return storageError(err)
	}
	defer unlock()
	if err := ctx.Err(); err != nil {
		return storageError(err)
	}
	path := filepath.Join(s.yearDir(d.Year), d.String()+".txt")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return journal.ErrNotFound
		}
		return storageError(err)
	}
	trashDir := filepath.Join(s.root, ".trash")
	if err := ensureDir(trashDir); err != nil {
		return storageError(err)
	}
	reserved, err := os.CreateTemp(trashDir, d.String()+"-*.txt")
	if err != nil {
		return storageError(err)
	}
	trashPath := reserved.Name()
	if err := reserved.Close(); err != nil {
		_ = os.Remove(trashPath)
		return storageError(err)
	}
	if err := ctx.Err(); err != nil {
		_ = os.Remove(trashPath)
		return storageError(err)
	}
	if err := os.Rename(path, trashPath); err != nil {
		_ = os.Remove(trashPath)
		if errors.Is(err, os.ErrNotExist) {
			return journal.ErrNotFound
		}
		return storageError(err)
	}
	return nil
}

// Keep pre-padding development directories usable without moving or deleting files.
func (s *Store) yearDir(year int) string {
	padded := filepath.Join(s.root, "entries", fmt.Sprintf("%04d", year))
	if year > 0 && year < 1000 {
		if _, err := os.Stat(padded); os.IsNotExist(err) {
			legacy := filepath.Join(s.root, "entries", strconv.Itoa(year))
			if info, err := os.Stat(legacy); err == nil && info.IsDir() {
				return legacy
			}
		}
	}
	return padded
}

func ensureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

func hashText(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// storageError translates filesystem failures while preserving their identity.
func storageError(err error) error {
	if err == nil {
		return nil
	}
	var pathErr *os.PathError
	var linkErr *os.LinkError
	if !errors.Is(err, journal.ErrUnavailable) && (errors.As(err, &pathErr) || errors.As(err, &linkErr)) {
		return fmt.Errorf("%w: %w", journal.ErrUnavailable, err)
	}
	return err
}
