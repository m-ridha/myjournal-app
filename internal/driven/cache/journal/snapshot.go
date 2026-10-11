package cachestore

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"cloud.google.com/go/civil"

	"github/mridha/myjournal-app/internal/core/journal"
)

// Account encoding makes separators unambiguous without retaining credentials.
func namespace(account string) string {
	return "mj-cache:v1/" + base64.RawURLEncoding.EncodeToString([]byte(account)) + "/"
}
func previewPrefix(account string, date civil.Date) string {
	return namespace(account) + "preview/" + date.String() + "/"
}
func (k cacheKey) string() string {
	switch k.kind {
	case "years":
		return namespace(k.account) + "years"
	case "year":
		return namespace(k.account) + "year/" + strconv.Itoa(k.year)
	default:
		return previewPrefix(k.account, k.date) + strconv.Itoa(k.maxSize)
	}
}

type snapshot struct {
	Kind      string
	ExpiresAt time.Time
	Value     json.RawMessage
}

func encodeSnapshot(kind string, value any, expiry time.Time) ([]byte, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return json.Marshal(snapshot{Kind: kind, ExpiresAt: expiry, Value: payload})
}

func decodeSnapshot(data []byte, kind string, now time.Time) (any, error) {
	var item snapshot
	if err := json.Unmarshal(data, &item); err != nil {
		return nil, err
	}
	if item.Kind != kind || item.ExpiresAt.IsZero() || len(item.Value) == 0 {
		return nil, errors.New("invalid cache snapshot")
	}
	if !now.Before(item.ExpiresAt) {
		return nil, nil
	}
	switch kind {
	case "years":
		var value []int
		err := json.Unmarshal(item.Value, &value)
		return value, err
	case "year":
		var value []journal.EntryMeta
		err := json.Unmarshal(item.Value, &value)
		return value, err
	case "preview":
		var value string
		err := json.Unmarshal(item.Value, &value)
		return value, err
	default:
		return nil, errors.New("unknown cache snapshot kind")
	}
}
