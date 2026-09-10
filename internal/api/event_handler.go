package api

import (
	"encoding/json"
	"io"
	"net/http"

	"edge-rule-engine/internal/model"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func (s *Server) handlePostEvent(w http.ResponseWriter, r *http.Request) {
	camera := chi.URLParam(r, "camera")

	payloadBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, `{"error": "failed to read body"}`, http.StatusBadRequest)
		return
	}

	var payload model.EventPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		http.Error(w, `{"error": "invalid json"}`, http.StatusBadRequest)
		return
	}

	if payload.EventId == "" || payload.EmittedAt == 0 || payload.Kind == "" {
		http.Error(w, `{"error": "eventId, emittedAt, and kind are required"}`, http.StatusBadRequest)
		return
	}
	if payload.Camera != camera {
		http.Error(w, `{"error": "camera mismatch"}`, http.StatusBadRequest)
		return
	}

	exists, err := s.eventStore.Exists(payload.EventId)
	if err != nil {
		s.logger.Error("failed to check event existence", zap.Error(err))
		http.Error(w, `{"error": "internal error"}`, http.StatusInternalServerError)
		return
	}
	if exists {
		w.WriteHeader(http.StatusOK)
		resp := map[string]string{"status": "duplicate", "eventId": payload.EventId}
		json.NewEncoder(w).Encode(resp)
		return
	}

	if err := s.eventStore.Save(payload.EventId, payload.Camera, payload.Kind, payload.EmittedAt, payloadBytes); err != nil {
		s.logger.Error("failed to save event", zap.Error(err))
		http.Error(w, `{"error": "internal error"}`, http.StatusInternalServerError)
		return
	}

	s.engine.ProcessEvent(r.Context(), payload)

	w.WriteHeader(http.StatusOK)
	resp := map[string]string{"status": "accepted", "eventId": payload.EventId}
	json.NewEncoder(w).Encode(resp)
}
