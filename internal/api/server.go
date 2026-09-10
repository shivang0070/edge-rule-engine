package api

import (
	"net/http"
	"time"

	"edge-rule-engine/internal/engine"
	"edge-rule-engine/internal/store"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type Server struct {
	router     chi.Router
	engine     *engine.Engine
	ruleStore  store.RuleStore
	stateStore store.StateStore
	eventStore store.EventStore
	execStore  store.ExecutionStore
	sseBroker  *SSEBroker
	logger     *zap.Logger
	startTime  time.Time
	uiFS       http.FileSystem
}

func NewServer(
	engine *engine.Engine,
	ruleStore store.RuleStore,
	stateStore store.StateStore,
	eventStore store.EventStore,
	execStore store.ExecutionStore,
	sseBroker *SSEBroker,
	logger *zap.Logger,
	uiFS http.FileSystem,
) *Server {
	s := &Server{
		router:     chi.NewRouter(),
		engine:     engine,
		ruleStore:  ruleStore,
		stateStore: stateStore,
		eventStore: eventStore,
		execStore:  execStore,
		sseBroker:  sseBroker,
		logger:     logger,
		startTime:  time.Now(),
		uiFS:       uiFS,
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	s.router.Use(recoveryMiddleware(s.logger))
	s.router.Use(requestLoggingMiddleware(s.logger))

	// Ingestion
	s.router.Route("/{camera}", func(r chi.Router) {
		r.Use(jsonContentTypeMiddleware)
		r.Post("/state", s.handlePostState)
		r.Post("/event", s.handlePostEvent)
	})

	// Management API
	s.router.Route("/api/v1/rules", func(r chi.Router) {
		r.Use(jsonContentTypeMiddleware)
		r.Post("/", s.handleCreateRule)
		r.Get("/", s.handleListRules)
		r.Get("/{ruleId}", s.handleGetRule)
		r.Put("/{ruleId}", s.handleUpdateRule)
		r.Delete("/{ruleId}", s.handleDeleteRule)
	})

	s.router.Get("/api/v1/executions", s.handleListExecutions)
	s.router.Get("/api/v1/stream", s.sseBroker.ServeHTTP)
	s.router.Get("/health", s.handleHealth)

	// Serve React UI
	fileServer := http.FileServer(s.uiFS)
	s.router.Get("/assets/*", func(w http.ResponseWriter, r *http.Request) {
		fileServer.ServeHTTP(w, r)
	})
	s.router.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		fileServer.ServeHTTP(w, r)
	})
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}
