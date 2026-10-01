package clock

import "time"

// Clock exposes the current time. The system implementation is intentionally thin.
type Clock interface {
	Now() time.Time
}

type System struct{}

func (System) Now() time.Time {
	return time.Now()
}
