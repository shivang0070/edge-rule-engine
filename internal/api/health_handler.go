package api

import (
	"encoding/json"
	"net/http"
	"time"
)

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "ok",
		"activeRules": s.engine.ActiveRuleCount(),
		"uptime":      time.Since(s.startTime).String(),
	})
}
