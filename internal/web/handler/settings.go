package handler

import (
	"encoding/json"
	"net/http"

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
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(api.SettingsResponse{
		AccountName: current.AccountEmail,
		Provider:    p.Name(),
	})
}
