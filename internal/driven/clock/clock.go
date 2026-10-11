// Package clock supplies the system clock adapter.
package clock

import (
	"time"

	"github/mridha/myjournal-app/internal/core/journal"
)

// System returns the current wall-clock time.
type System struct{}

var _ journal.Clock = System{}

func (System) Now() time.Time { return time.Now() }
