package api

import (
	"encoding/json"
	"io"
	"net/http"

	"edge-rule-engine/internal/model"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func (s *Server) handlePostState(w http.ResponseWriter, r *http.Request) {
	camera := chi.URLParam(r, "camera")

	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, `{"error": "failed to read body"}`, http.StatusBadRequest)
		return
	}

	var payload model.StatePayload
	if err := json.Unmarshal(data, &payload); err != nil {
		http.Error(w, `{"error": "invalid json"}`, http.StatusBadRequest)
		return
	}

	if payload.EmittedAt == 0 {
		http.Error(w, `{"error": "emittedAt is required"}`, http.StatusBadRequest)
		return
	}
	if payload.Camera != camera {
		http.Error(w, `{"error": "camera mismatch"}`, http.StatusBadRequest)
		return
	}
	if len(payload.Regions) == 0 {
		http.Error(w, `{"error": "regions are required"}`, http.StatusBadRequest)
		return
	}

	if err := s.stateStore.Save(payload.Camera, payload.EmittedAt, data); err != nil {
		s.logger.Error("failed to save state", zap.Error(err))
		http.Error(w, `{"error": "internal error"}`, http.StatusInternalServerError)
		return
	}

	s.engine.ProcessState(r.Context(), payload)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "accepted"}`))
}
