package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"cloud.google.com/go/civil"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github/mridha/myjournal-app/internal/core/journal"
	"github/mridha/myjournal-app/internal/driver/web/api"
)

func (s *Server) ListEntries(w http.ResponseWriter, r *http.Request, params api.ListEntriesParams) {
	w.Header().Set("Content-Type", "application/json")
	service, err := s.service(r)
	if err != nil {
		s.writeServiceError(w, err)
		return
	}

	page := 1
	if params.Page != nil {
		page = *params.Page
	}

	size := 20
	if params.Size != nil {
		size = *params.Size
	}

	result, err := service.ListEntries(r.Context(), page, size)
	if err != nil {
		s.writeServiceError(w, err)
		return
	}

	items := make([]api.ListItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, api.ListItem{Date: toOpenAPIDate(item.Date), Preview: item.Preview})
	}
	resp := api.ListEntriesResponse{Page: result.Page, PageSize: result.PageSize, HasNext: result.HasNext, Items: items}
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) GetEntry(w http.ResponseWriter, r *http.Request, date openapi_types.Date) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	service, err := s.service(r)
	if err != nil {
		s.writeServiceError(w, err)
		return
	}

	open := service.Open
	for _, directive := range strings.Split(r.Header.Get("Cache-Control"), ",") {
		if strings.EqualFold(strings.TrimSpace(directive), "no-store") {
			open = service.OpenFresh
			break
		}
	}
	entry, err := open(r.Context(), civil.DateOf(date.Time))
	if err != nil {
		if errors.Is(err, journal.ErrNotFound) {
			_ = json.NewEncoder(w).Encode(api.Entry{Date: date, Exists: false})
			return
		}
		s.writeServiceError(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(api.Entry{
		Date:    toOpenAPIDate(entry.Date),
		Text:    entry.Text,
		Version: string(entry.Version),
		Exists:  entry.Exists,
	})
}

func (s *Server) SaveEntry(w http.ResponseWriter, r *http.Request, date openapi_types.Date) {
	w.Header().Set("Content-Type", "application/json")
	service, err := s.service(r)
	if err != nil {
		s.writeServiceError(w, err)
		return
	}

	// Bound the complete JSON envelope, including escaped text and base version.
	payload, err := io.ReadAll(http.MaxBytesReader(w, r.Body, MaxSaveRequestBytes))
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			api.WriteError(w, http.StatusRequestEntityTooLarge, s.message("errors.requestTooLarge"))
		} else {
			api.WriteError(w, http.StatusBadRequest, s.message("errors.invalidRequest"))
		}
		return
	}
	// Pointers distinguish explicit empty strings from absent or null fields.
	var req struct {
		Text        *string `json:"text"`
		BaseVersion *string `json:"baseVersion"`
	}
	if !utf8.Valid(payload) || json.Unmarshal(payload, &req) != nil || req.Text == nil || req.BaseVersion == nil {
		api.WriteError(w, http.StatusBadRequest, s.message("errors.invalidRequest"))
		return
	}

	if errors.Is(validateJSONUnicode(payload), journal.ErrInvalidText) {
		api.WriteError(w, http.StatusBadRequest, s.message("errors.invalidText"))
		return
	}
	base := journal.Version(*req.BaseVersion)

	entry, err := service.Save(r.Context(), civil.DateOf(date.Time), *req.Text, base)
	if err != nil {
		s.writeServiceError(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(api.Entry{
		Date:    toOpenAPIDate(entry.Date),
		Text:    entry.Text,
		Version: string(entry.Version),
		Exists:  entry.Exists,
	})
}

func (s *Server) DeleteEntry(w http.ResponseWriter, r *http.Request, date openapi_types.Date) {
	service, err := s.service(r)
	if err != nil {
		s.writeServiceError(w, err)
		return
	}
	if err := service.Delete(r.Context(), civil.DateOf(date.Time)); err != nil {
		s.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) GetCalendar(w http.ResponseWriter, r *http.Request, params api.GetCalendarParams) {
	w.Header().Set("Content-Type", "application/json")
	service, err := s.service(r)
	if err != nil {
		s.writeServiceError(w, err)
		return
	}

	monthKey, err := time.Parse("2006-01", params.Month)
	if err != nil || monthKey.Year() < 1 {
		api.WriteError(w, http.StatusBadRequest, s.message("errors.invalidRequest"))
		return
	}

	year, month := monthKey.Year(), monthKey.Month()
	monthView, err := service.Month(r.Context(), year, month)
	if err != nil {
		s.writeServiceError(w, err)
		return
	}
	weeks := make([][]api.CalendarDay, 0, len(monthView.Weeks))
	for _, week := range monthView.Weeks {
		row := make([]api.CalendarDay, 0, len(week))
		for _, day := range week {
			row = append(row, api.CalendarDay{Date: toOpenAPIDate(day.Date), HasEntry: day.HasEntry, InMonth: day.InMonth, IsFuture: day.IsFuture, IsToday: day.IsToday})
		}
		weeks = append(weeks, row)
	}

	_ = json.NewEncoder(w).Encode(api.CalendarResponse{
		Year:  year,
		Month: int(month),
		Weeks: weeks,
	})
}

func toOpenAPIDate(d civil.Date) openapi_types.Date {
	return openapi_types.Date{Time: time.Date(d.Year, time.Month(d.Month), d.Day, 0, 0, 0, 0, time.UTC)}
}

// MaxSaveRequestBytes caps the JSON envelope, not the character count.
const MaxSaveRequestBytes = 1 << 20

// encoding/json replaces unpaired UTF-16 escapes; reject them instead of changing text.
// Called only after JSON syntax validation, so each escape has a complete operand.
func validateJSONUnicode(payload []byte) error {
	for i := 0; i < len(payload); i++ {
		if payload[i] != '\\' {
			continue
		}
		i++
		if payload[i] != 'u' {
			continue
		}
		value, _ := strconv.ParseUint(string(payload[i+1:i+5]), 16, 16)
		i += 4
		if value >= 0xdc00 && value <= 0xdfff {
			return journal.ErrInvalidText
		}
		if value < 0xd800 || value > 0xdbff {
			continue
		}
		if i+6 >= len(payload) || payload[i+1] != '\\' || payload[i+2] != 'u' {
			return journal.ErrInvalidText
		}
		low, err := strconv.ParseUint(string(payload[i+3:i+7]), 16, 16)
		if err != nil || low < 0xdc00 || low > 0xdfff {
			return journal.ErrInvalidText
		}
		i += 6
	}
	return nil
}
