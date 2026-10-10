// Package testutil contains dependency-free fixtures shared by test packages.
// Production code must not depend on this package.
package testutil

// LocalBaseURL is a reusable localhost origin for request and provider fixtures.
// It does not start a server; live HTTP tests should use httptest.Server.URL.
const LocalBaseURL = "http://localhost:3030"
