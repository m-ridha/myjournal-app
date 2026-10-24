package clock

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSystemNow(t *testing.T) {
	systemClock := System{}

	before := time.Now()
	got := systemClock.Now()
	after := time.Now()

	assert.False(t, got.Before(before))
	assert.False(t, got.After(after))
}
