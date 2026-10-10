package handler

import (
	"encoding/json"
	"net/http"

	"github/mridha/myjournal-app/internal/provider"
	"github/mridha/myjournal-app/internal/web/api"
)

func (s *Server) GetSettings(w http.ResponseWriter, r *http.Request) {
	current, err := s.currentSession(r)
	if err != nil {
		s.writeServiceError(w, err)
		return
	}
	p, ok := s.registry.Get(current.ProviderID)
	if !ok {
		s.writeServiceError(w, ErrUnauthenticated)
		return
	}
	var folderURL *string
	if locator, ok := p.(provider.FolderLocator); ok {
		value, err := locator.FolderURL(r.Context(), current.Connection)
		if err != nil {
			s.writeServiceError(w, err)
			return
		}
		if value != "" {
			folderURL = &value
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(api.SettingsResponse{
		AccountName: current.AccountEmail,
		Provider:    p.Name(),
		FolderUrl:   folderURL,
	})
}
