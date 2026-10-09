package journal

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"cloud.google.com/go/civil"
)

// Date is the journal entry date; it is timezone-neutral and stays fixed once selected.
type Date = civil.Date

// Version is an opaque token passed back and forth by stores.
type Version string

var (
	ErrConflict          = errors.New("entry changed elsewhere")
	ErrNotFound          = errors.New("entry not found")
	ErrAuthRequired      = errors.New("reconnect required")
	ErrUnavailable       = errors.New("storage unavailable")
	ErrInvalidPagination = errors.New("invalid pagination")
	ErrFutureDate        = errors.New("future date is not allowed")
)

type Entry struct {
	Date    Date
	Text    string
	Version Version
	Exists  bool
}

type EntryMeta struct {
	Date         Date
	ModifiedTime time.Time
}

type MonthView struct {
	Year  int
	Month time.Month
	Weeks [][]Day
}

type Day struct {
	Date     Date
	InMonth  bool
	HasEntry bool
	IsToday  bool
	IsFuture bool
}

type ListPage struct {
	Items    []ListItem
	Page     int
	PageSize int
	HasNext  bool
}

type ListItem struct {
	Date    Date
	Preview string
}

var entryFileRE = regexp.MustCompile(`^entries/(\d{4})/(\d{4}-\d{2}-\d{2})\.txt$`)

func EntryPath(d Date) string {
	return fmt.Sprintf("entries/%d/%04d-%02d-%02d.txt", d.Year, d.Year, d.Month, d.Day)
}

func ParseEntryDate(path string) (Date, error) {
	matches := entryFileRE.FindStringSubmatch(strings.TrimPrefix(path, "./"))
	if matches == nil {
		return Date{}, fmt.Errorf("invalid entry path %q", path)
	}
	year, err := time.Parse("2006", matches[1])
	if err != nil {
		return Date{}, fmt.Errorf("invalid entry year %q: %w", matches[1], err)
	}
	date, err := civil.ParseDate(matches[2])
	if err != nil {
		return Date{}, fmt.Errorf("invalid entry date %q: %w", matches[2], err)
	}
	if date.Year != year.Year() {
		return Date{}, fmt.Errorf("year mismatch in %q", path)
	}
	return date, nil
}
