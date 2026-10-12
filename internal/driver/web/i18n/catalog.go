package i18n

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"time"
)

// Catalog contains the immutable messages loaded for one locale.
type Catalog struct {
	locale   string
	messages map[string]string
}

func Load(filesystem fs.FS, path string) (Catalog, error) {
	data, err := fs.ReadFile(filesystem, path)
	if err != nil {
		return Catalog{}, fmt.Errorf("read locale %q: %w", path, err)
	}
	var messages map[string]string
	if err := json.Unmarshal(data, &messages); err != nil {
		return Catalog{}, fmt.Errorf("parse locale %q: %w", path, err)
	}
	locale := messages["app.locale"]
	if locale == "" {
		return Catalog{}, fmt.Errorf("locale %q is missing app.locale", path)
	}
	return Catalog{locale: locale, messages: messages}, nil
}

func (c Catalog) Text(key string) string {
	if value, ok := c.messages[key]; ok {
		return value
	}
	return key
}

func (c Catalog) Locale() string {
	return c.locale
}

func (c Catalog) FormatDate(date time.Time) string {
	if c.locale == "id" {
		return fmt.Sprintf("%s, %02d %s %d", c.Text(fmt.Sprintf("date.weekday.%d", date.Weekday())), date.Day(), c.Text(fmt.Sprintf("date.month.%d", date.Month())), date.Year())
	}
	layout := c.Text("format.date")
	if layout == "format.date" {
		layout = "Monday, 02 January 2006"
	}
	return date.Format(layout)
}

func (c Catalog) JSON() (string, error) {
	data, err := json.Marshal(c.messages)
	if err != nil {
		return "", fmt.Errorf("encode locale %q: %w", c.locale, err)
	}
	return string(data), nil
}
