package web

import "github/mridha/myjournal-app/internal/driver/web/handler"

// Options contains validated transport policy from the composition root.
type Options struct {
	Session handler.Options
	HTTPS   bool
}
