package local

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github/mridha/myjournal-app/internal/journal"

	"cloud.google.com/go/civil"
)

// Store persists entries as plain text files on disk in the local dev directory.
type Store struct {
	root string
}

func NewStore(root string) *Store {
	return &Store{root: root}
}

func (s *Store) ListYears(ctx context.Context) ([]int, error) {
	entries, err := os.ReadDir(filepath.Join(s.root, "entries"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	years := make([]int, 0, len(entries))
	for _, entry := range entries {
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
	path := filepath.Join(s.root, "entries", fmt.Sprintf("%d", year))
	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	meta := make([]journal.EntryMeta, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".txt") {
			continue
		}
		date, err := journal.ParseEntryDate(filepath.ToSlash(filepath.Join("entries", fmt.Sprintf("%d", year), entry.Name())))
		if err != nil {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		meta = append(meta, journal.EntryMeta{Date: date, ModifiedTime: info.ModTime()})
	}
	sort.Slice(meta, func(i, j int) bool {
		return meta[i].Date.After(meta[j].Date)
	})
	return meta, nil
}

func (s *Store) Get(ctx context.Context, d civil.Date) (journal.Entry, error) {
	path := filepath.Join(s.root, journal.EntryPath(d))
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return journal.Entry{}, journal.ErrNotFound
		}
		return journal.Entry{}, err
	}
	return journal.Entry{Date: d, Text: string(data), Version: journal.Version(hashText(string(data))), Exists: true}, nil
}

func (s *Store) Previews(ctx context.Context, dates []civil.Date, maxBytes int) (map[civil.Date]string, error) {
	out := make(map[civil.Date]string, len(dates))
	for _, d := range dates {
		entry, err := s.Get(ctx, d)
		if err != nil {
			continue
		}
		preview := entry.Text
		if len(preview) > maxBytes {
			preview = preview[:maxBytes]
		}
		out[d] = preview
	}
	return out, nil
}

func (s *Store) Put(ctx context.Context, d civil.Date, text string, base journal.Version) (journal.Entry, error) {
	if err := ensureDir(filepath.Join(s.root, "entries")); err != nil {
		return journal.Entry{}, err
	}
	if err := ensureDir(filepath.Join(s.root, "entries", fmt.Sprintf("%d", d.Year))); err != nil {
		return journal.Entry{}, err
	}
	path := filepath.Join(s.root, journal.EntryPath(d))
	_, statErr := os.Stat(path)
	switch {
	case statErr == nil:
		if base == "" {
			return journal.Entry{}, journal.ErrConflict
		}
		current, err := s.Get(ctx, d)
		if err != nil {
			return journal.Entry{}, err
		}
		if current.Version != base {
			return journal.Entry{}, journal.ErrConflict
		}
	case os.IsNotExist(statErr):
		if base != "" {
			return journal.Entry{}, journal.ErrConflict
		}
	default:
		return journal.Entry{}, statErr
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		return journal.Entry{}, err
	}
	return journal.Entry{Date: d, Text: text, Version: journal.Version(hashText(text)), Exists: true}, nil
}

func (s *Store) Trash(ctx context.Context, d civil.Date) error {
	path := filepath.Join(s.root, journal.EntryPath(d))
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return journal.ErrNotFound
		}
		return err
	}
	trashDir := filepath.Join(s.root, ".trash")
	if err := ensureDir(trashDir); err != nil {
		return err
	}
	trashPath := filepath.Join(trashDir, fmt.Sprintf("%d-%02d-%02d.txt", d.Year, d.Month, d.Day))
	return os.Rename(path, trashPath)
}

func ensureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

func hashText(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
