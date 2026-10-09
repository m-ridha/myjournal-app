package api

import (
	"encoding/json"
	"net/http"
)

// WriteError uses the same public error envelope for handlers and request binding.
func WriteError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Error{Message: message})
}
