package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/kundanmergu/cloud-search-engine/backend/internal/models"
	"github.com/kundanmergu/cloud-search-engine/backend/internal/retrieval"
	"github.com/kundanmergu/cloud-search-engine/backend/internal/storage"
)

// Server hosts HTTP handlers for the search API.
type Server struct {
	store    *storage.Store
	searcher *retrieval.Searcher
	logger   *slog.Logger
}

// NewServer constructs the API server.
func NewServer(store *storage.Store, searcher *retrieval.Searcher, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	return &Server{store: store, searcher: searcher, logger: logger}
}

// Router builds the Chi router with middleware.
func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(s.logRequests)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://127.0.0.1:3000"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/healthz", s.handleHealth)
	r.Route("/v1", func(r chi.Router) {
		r.Get("/search", s.handleSearchGet)
		r.Post("/search", s.handleSearchPost)
		r.Get("/documents", s.handleListDocuments)
	})

	return r
}

func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		s.logger.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"bytes", ww.BytesWritten(),
			"duration_ms", time.Since(start).Milliseconds(),
			"request_id", middleware.GetReqID(r.Context()),
		)
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	dbStatus := "ok"
	if err := s.store.Ping(r.Context()); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, models.HealthResponse{
			Status:   "degraded",
			Database: "unavailable",
		})
		return
	}
	writeJSON(w, http.StatusOK, models.HealthResponse{
		Status:   "ok",
		Database: dbStatus,
	})
}

func (s *Server) handleSearchGet(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		writeError(w, http.StatusBadRequest, "query parameter 'q' is required")
		return
	}

	limit := 0
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			writeError(w, http.StatusBadRequest, "limit must be a non-negative integer")
			return
		}
		limit = n
	}

	req := models.SearchRequest{
		Query: q,
		Limit: limit,
		Filter: models.SearchFilters{
			Provider: r.URL.Query().Get("provider"),
			Service:  r.URL.Query().Get("service"),
			Category: r.URL.Query().Get("category"),
		},
	}
	s.runSearch(w, r, req)
}

func (s *Server) handleSearchPost(w http.ResponseWriter, r *http.Request) {
	var req models.SearchRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	req.Query = strings.TrimSpace(req.Query)
	if req.Query == "" {
		writeError(w, http.StatusBadRequest, "query is required")
		return
	}
	s.runSearch(w, r, req)
}

func (s *Server) runSearch(w http.ResponseWriter, r *http.Request, req models.SearchRequest) {
	resp, err := s.searcher.Search(r.Context(), req)
	if err != nil {
		if strings.Contains(err.Error(), "query is required") {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		s.logger.Error("search failed", "error", err)
		writeError(w, http.StatusInternalServerError, "search failed")
		return
	}
	if resp.Results == nil {
		resp.Results = []models.SearchHit{}
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleListDocuments(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			writeError(w, http.StatusBadRequest, "limit must be a non-negative integer")
			return
		}
		limit = n
	}

	docs, err := s.store.ListDocuments(r.Context(), limit)
	if err != nil {
		s.logger.Error("list documents failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list documents")
		return
	}
	if docs == nil {
		docs = []models.Document{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"documents": docs, "count": len(docs)})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
