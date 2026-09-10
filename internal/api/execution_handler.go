package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"go.uber.org/zap"
)

func (s *Server) handleListExecutions(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 100 // default limit
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	executions, err := s.execStore.ListAll(limit)
	if err != nil {
		s.logger.Error("failed to list executions", zap.Error(err))
		http.Error(w, `{"error": "internal error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(executions)
}
