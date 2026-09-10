package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"edge-rule-engine/internal/model"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func (s *Server) handleCreateRule(w http.ResponseWriter, r *http.Request) {
	var rule model.Rule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		http.Error(w, `{"error": "invalid json"}`, http.StatusBadRequest)
		return
	}

	if err := rule.Validate(); err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	if rule.ID == "" {
		rule.ID = fmt.Sprintf("rule-%d", time.Now().UnixNano())
	}
	now := time.Now().Unix()
	rule.CreatedAt = now
	rule.UpdatedAt = now

	// Try to register in engine to validate expression
	if err := s.engine.RegisterRule(rule); err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "invalid rule expression: %s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	if err := s.ruleStore.Create(&rule); err != nil {
		s.engine.UnregisterRule(rule.ID) // rollback
		s.logger.Error("failed to save rule", zap.Error(err))
		http.Error(w, `{"error": "internal error"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(rule)
}

func (s *Server) handleListRules(w http.ResponseWriter, r *http.Request) {
	rules, err := s.ruleStore.List()
	if err != nil {
		s.logger.Error("failed to list rules", zap.Error(err))
		http.Error(w, `{"error": "internal error"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(rules)
}

func (s *Server) handleGetRule(w http.ResponseWriter, r *http.Request) {
	ruleID := chi.URLParam(r, "ruleId")

	rule, err := s.ruleStore.GetByID(ruleID)
	if err != nil {
		http.Error(w, `{"error": "not found"}`, http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(rule)
}

func (s *Server) handleUpdateRule(w http.ResponseWriter, r *http.Request) {
	ruleID := chi.URLParam(r, "ruleId")

	var rule model.Rule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		http.Error(w, `{"error": "invalid json"}`, http.StatusBadRequest)
		return
	}

	rule.ID = ruleID
	if err := rule.Validate(); err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	rule.UpdatedAt = time.Now().Unix()

	if err := s.engine.RegisterRule(rule); err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "invalid rule expression: %s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	if err := s.ruleStore.Update(&rule); err != nil {
		s.logger.Error("failed to update rule", zap.Error(err))
		http.Error(w, `{"error": "internal error"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(rule)
}

func (s *Server) handleDeleteRule(w http.ResponseWriter, r *http.Request) {
	ruleID := chi.URLParam(r, "ruleId")

	if err := s.ruleStore.Delete(ruleID); err != nil {
		s.logger.Error("failed to delete rule", zap.Error(err))
		http.Error(w, `{"error": "internal error"}`, http.StatusInternalServerError)
		return
	}

	s.engine.UnregisterRule(ruleID)

	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}
