package journal

import (
	"errors"
	"testing"
	"time"
)

func TestValidateEntryDate(t *testing.T) {
	today := Date{Year: 2026, Month: time.October, Day: 11}
	for _, tc := range []struct {
		date Date
		want error
	}{
		{Date{Year: 0, Month: 1, Day: 1}, ErrInvalidDate},
		{Date{Year: 1, Month: 1, Day: 1}, nil},
		{Date{Year: 9999, Month: 12, Day: 31}, ErrFutureDate},
		{Date{Year: 10000, Month: 1, Day: 1}, ErrInvalidDate},
		{Date{Year: 2026, Month: 2, Day: 30}, ErrInvalidDate},
		{today, nil}, {Date{Year: 2026, Month: 10, Day: 12}, ErrFutureDate},
	} {
		if got := ValidateEntryDate(tc.date, today); !errors.Is(got, tc.want) {
			t.Errorf("date %v: got %v want %v", tc.date, got, tc.want)
		}
	}
	if err := ValidateEntryDate(Date{Year: 9999, Month: 12, Day: 31}, Date{Year: 9999, Month: 12, Day: 31}); err != nil {
		t.Fatal(err)
	}
}
