package testutil

import "time"

// StubClock returns a test-controlled time. Its zero value returns the zero time.
// Changing Time concurrently with calls to Now is unsafe; keep mutations sequential.
// Pass a pointer when the caller must observe later changes to Time.
type StubClock struct{ Time time.Time }

// Now returns Time.
func (sc StubClock) Now() time.Time { return sc.Time }
