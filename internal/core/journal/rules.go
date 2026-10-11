package journal

// ValidateEntryDate applies the shared editable-date policy without storage I/O.
func ValidateEntryDate(d Date, today Date) error {
	if !d.IsValid() || d.Year < 1 || d.Year > 9999 {
		return ErrInvalidDate
	}
	if d.After(today) {
		return ErrFutureDate
	}
	return nil
}
