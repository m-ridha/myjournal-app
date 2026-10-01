package handler

import (
	"encoding/json"
	"net/http"

	"github/mridha/myjournal-app/internal/web/api"
)

func (s *Server) GetSettings(w http.ResponseWriter, r *http.Request) {
	// TODO
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(api.SettingsResponse{
		AccountName: "dev@localhost",
		Provider:    s.message("provider.local.name"),
	})
}
