package i18n

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"strconv"
	"time"
)

// defaultDateLayout formats dates for locales without a dedicated formatter.
const defaultDateLayout = "Monday, 02 January 2006"

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
	switch c.locale {
	case "en":
		return fmt.Sprintf("%s, %s %s %d", c.Text(fmt.Sprintf("date.weekday.%d", date.Weekday())), c.Text(fmt.Sprintf("date.month.%d", date.Month())), ordinalDay(date.Day()), date.Year())
	case "id":
		return fmt.Sprintf("%s, %02d %s %d", c.Text(fmt.Sprintf("date.weekday.%d", date.Weekday())), date.Day(), c.Text(fmt.Sprintf("date.month.%d", date.Month())), date.Year())
	default:
		return date.Format(defaultDateLayout)
	}
}

// ordinalDay renders a day of month with an English ordinal suffix, such as 1st, 2nd, 3rd, or 4th.
func ordinalDay(day int) string {
	suffix := "th"
	switch day % 100 {
	case 11, 12, 13:
	default:
		switch day % 10 {
		case 1:
			suffix = "st"
		case 2:
			suffix = "nd"
		case 3:
			suffix = "rd"
		}
	}
	return strconv.Itoa(day) + suffix
}

func (c Catalog) JSON() (string, error) {
	data, err := json.Marshal(c.messages)
	if err != nil {
		return "", fmt.Errorf("encode locale %q: %w", c.locale, err)
	}
	return string(data), nil
}
