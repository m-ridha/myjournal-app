package journal

import (
	"context"
	"errors"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestUnicodePreview(t *testing.T) {
	for _, tc := range []struct {
		text            string
		budget          int
		prefix, preview string
	}{
		{"\n\u2003\t  中文🙂next\nignored", 7, "中文", "中文..."},
		{"🙂", 3, "", "..."},
		{"éé", 3, "é", "é..."},
		{"a\u0301b", 2, "a", "a..."},
		{"\n  short\nignored", 40, "short", "short"},
		{"  ", 40, "", ""},
	} {
		prefix := PreviewPrefix(tc.text, tc.budget)
		if prefix != tc.prefix || !utf8.ValidString(prefix) || len(prefix) > tc.budget {
			t.Fatalf("prefix %q: %q", tc.text, prefix)
		}
		if got := PreviewText(tc.text, tc.budget); got != tc.preview {
			t.Fatalf("preview=%q want=%q", got, tc.preview)
		}
	}
	if PreviewPrefix("text", 0) != "" || PreviewPrefix("text", -1) != "" {
		t.Fatal("nonpositive budget")
	}
}

func TestSaveRejectsInvalidUTF8AndNormalizesLF(t *testing.T) {
	store := &inMemStore{entries: make(map[Date]Entry)}
	service := NewService(store, dummyClock{})
	date := Date{Year: 1, Month: 1, Day: 1}
	if _, err := service.Save(context.Background(), date, string([]byte{0xff}), ""); !errors.Is(err, ErrInvalidText) {
		t.Fatalf("error=%v", err)
	}
	if len(store.entries) != 0 {
		t.Fatal("invalid text reached storage")
	}
	saved, err := service.Save(context.Background(), date, "中文🙂\r\né\rfin", "")
	if err != nil || saved.Text != "中文🙂\né\nfin" {
		t.Fatalf("entry=%+v err=%v", saved, err)
	}
}

func TestPagePreviewTrimsBeforeBudgetAndMarksTruncation(t *testing.T) {
	date := Date{Year: 2026, Month: 10, Day: 8}
	store := &inMemStore{entries: map[Date]Entry{date: {Date: date, Text: strings.Repeat("\n ", 80) + strings.Repeat("中", 20)}}}
	page, err := NewService(store, dummyClock{}).Page(context.Background(), 1, 20)
	if err != nil || len(page.Items) != 1 || page.Items[0].Preview != strings.Repeat("中", 13)+"..." {
		t.Fatalf("page=%+v err=%v", page, err)
	}
}
