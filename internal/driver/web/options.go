package web

import "github/m-ridha/myjournal-app/internal/driver/web/handler"

// Options contains validated transport policy from the composition root.
type Options struct {
	BaseURL string
	Session handler.Options
	HTTPS   bool
}
