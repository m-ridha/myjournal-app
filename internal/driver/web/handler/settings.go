package handler

import (
	"encoding/json"
	"net/http"

	"myjournal-app/internal/driver/web/api"
)

func (s *Server) GetSettings(w http.ResponseWriter, r *http.Request) {
	current, err := s.currentSession(r)
	if err != nil {
		s.writeServiceError(w, err)
		return
	}
	result, err := s.connections.Settings(r.Context(), current)
	if err != nil {
		s.writeServiceError(w, err)
		return
	}
	value := result.FolderURL
	var folderURL *string
	if value != "" {
		folderURL = &value
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(api.SettingsResponse{
		AccountName: result.AccountName,
		Provider:    result.ProviderName,
		FolderUrl:   folderURL,
	})
}
