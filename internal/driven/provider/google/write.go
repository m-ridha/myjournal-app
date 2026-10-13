package google

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"

	"github/mridha/myjournal-app/internal/core/journal"
)

func (s *Store) Put(ctx context.Context, date journal.Date, text string, base journal.Version) (journal.Entry, error) {
	defer s.clearDiscovery()
	if !date.IsValid() || date.Year < 1 || date.Year > 9999 {
		return journal.Entry{}, journal.ErrInvalidDate
	}
	if !utf8.ValidString(text) {
		return journal.Entry{}, journal.ErrInvalidText
	}
	if len(text) > maxContentBytes {
		return journal.Entry{}, journal.ErrUnavailable
	}
	unlock, err := s.lock(ctx)
	if err != nil {
		return journal.Entry{}, err
	}
	defer unlock()
	file, err := s.entryFile(ctx, date)
	if err != nil && !errors.Is(err, journal.ErrNotFound) {
		return journal.Entry{}, err
	}
	missing := errors.Is(err, journal.ErrNotFound)
	if missing && base != "" {
		return journal.Entry{}, journal.ErrConflict
	}
	if !missing && file.Md5Checksum == "" {
		return journal.Entry{}, journal.ErrUnavailable
	}
	if !missing && (base == "" || journal.Version(file.Md5Checksum) != base) {
		return journal.Entry{}, journal.ErrConflict
	}
	if !missing {
		// Re-read just before upload. This reduces but does not eliminate the external
		// writer race: Drive checksum comparison and upload are separate operations.
		file, err = s.metadata(ctx, file.Id)
		if errors.Is(err, journal.ErrNotFound) {
			return journal.Entry{}, journal.ErrConflict
		}
		if err != nil {
			return journal.Entry{}, err
		}
		if file.Md5Checksum == "" {
			return journal.Entry{}, journal.ErrUnavailable
		}
		if journal.Version(file.Md5Checksum) != base {
			return journal.Entry{}, journal.ErrConflict
		}
		if checksum([]byte(text)) == file.Md5Checksum {
			return journal.Entry{Date: date, Text: text, Version: base, Exists: true}, nil
		}
		// Entries are capped at 1 MiB; stream one request instead of allocating the SDK
		// default 16 MiB chunk. Ambiguous uploads remain non-retryable.
		call := s.service.Files.Update(file.Id, &drive.File{}).Media(strings.NewReader(text), googleapi.ContentType("text/plain; charset=utf-8"), googleapi.ChunkSize(0)).Fields(fileFields).Context(ctx)
		if etag := file.Header.Get("ETag"); etag != "" {
			call.Header().Set("If-Match", etag)
		}
		file, err = call.Do()
	} else {
		folder, folderErr := s.year(ctx, date.Year, true)
		if folderErr != nil {
			return journal.Entry{}, folderErr
		}
		file, err = s.service.Files.Create(&drive.File{Name: date.String() + ".txt", MimeType: textMIME, Parents: []string{folder.Id}, AppProperties: map[string]string{propertyApp: appTag, propertyRole: roleEntry}}).Media(strings.NewReader(text), googleapi.ContentType("text/plain; charset=utf-8"), googleapi.ChunkSize(0)).Fields(fileFields).Context(ctx).Do()
	}
	if err != nil {
		return journal.Entry{}, mapError(err)
	}
	if file.Md5Checksum == "" || file.Md5Checksum != checksum([]byte(text)) {
		return journal.Entry{}, journal.ErrUnavailable
	}
	return journal.Entry{Date: date, Text: text, Version: journal.Version(file.Md5Checksum), Exists: true}, nil
}

func (s *Store) Trash(ctx context.Context, date journal.Date) error {
	defer s.clearDiscovery()
	unlock, err := s.lock(ctx)
	if err != nil {
		return err
	}
	defer unlock()
	file, err := s.entryFile(ctx, date)
	if err != nil {
		return err
	}
	_, err = s.service.Files.Update(file.Id, &drive.File{Trashed: true}).Fields("id,trashed").Context(ctx).Do()
	return mapError(err)
}
