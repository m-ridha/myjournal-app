package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWriteError(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		message string
		body    string
	}{
		{
			name:    "bad request",
			status:  http.StatusBadRequest,
			message: "Invalid date.",
			body:    `{"message":"Invalid date."}`,
		},
		{
			name:    "unauthorized",
			status:  http.StatusUnauthorized,
			message: "Please sign in again.",
			body:    `{"message":"Please sign in again."}`,
		},
		{
			name:    "escaped characters and Unicode",
			status:  http.StatusInternalServerError,
			message: "Could not save \"中文\".\nPlease retry.",
			body:    `{"message":"Could not save \"中文\".\nPlease retry."}`,
		},
		{
			name:   "empty message",
			status: http.StatusServiceUnavailable,
			body:   `{"message":""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			recorder.Header().Set("Content-Type", "text/plain")

			WriteError(recorder, tt.status, tt.message)

			response := recorder.Result()
			t.Cleanup(func() {
				assert.NoError(t, response.Body.Close())
			})
			assert.Equal(t, tt.status, response.StatusCode)
			assert.Equal(t, "application/json", response.Header.Get("Content-Type"))
			assert.JSONEq(t, tt.body, recorder.Body.String())
		})
	}
}
