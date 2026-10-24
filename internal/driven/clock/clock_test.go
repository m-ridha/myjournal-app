package clock

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSystemNow(t *testing.T) {
	clock := System{}

	clockTimeTruncated := clock.Now().Truncate(5 * time.Second)
	expectedTimeTruncated := time.Now().Truncate(5 * time.Second)

	assert.Equal(t, expectedTimeTruncated, clockTimeTruncated)
}
