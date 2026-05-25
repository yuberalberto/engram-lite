// Package server provides the HTTP API for engram-lite.
package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/yuberalberto/engram-lite/internal/diagnostic"
	projectpkg "github.com/yuberalberto/engram-lite/internal/project"
	"github.com/yuberalberto/engram-lite/internal/store"
)

var loadServerStats = func(s *store.Store) (*store.Stats, error) {
	return s.Stats()
}

// Server holds the HTTP server state.
type Server struct {
	store   *store.Store
	port    int
	mux     *http.ServeMux
	onWrite func()
	listen  func(string, string) (net.Listener, error)
	serve   func(net.Listener, http.Handler) error
}

// New creates a new HTTP server with routes registered.
func New(s *store.Store, port int) *Server {
	srv := &Server{store: s, port: port, listen: net.Listen, serve: http.Serve}
	srv.mux = http.NewServeMux()
	srv.routes()
	return srv
}

// SetOnWrite configures a callback invoked after every successful local write.
func (s *Server) SetOnWrite(fn func()) {
	s.onWrite = fn
}

func (s *Server) notifyWrite() {
	if s.onWrite != nil {
		s.onWrite()
	}
}

func (s *Server) Start() error {
	addr := fmt.Sprintf("127.0.0.1:%d", s.port)
	listenFn := s.listen
	if listenFn == nil {
		listenFn = net.Listen
	}
	serveFn := s.serve
	if serveFn == nil {
		serveFn = http.Serve
	}

	ln, err := listenFn("tcp", addr)
	if err != nil {
		return fmt.Errorf("engram-lite server: listen %s: %w", addr, err)
	}
	log.Printf("[engram-lite] HTTP server listening on %s", addr)
	return serveFn(ln, s.mux)
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /health", s.handleHealth)

	// Sessions
	s.mux.HandleFunc("POST /sessions", s.handleCreateSession)
	s.mux.HandleFunc("POST /sessions/{id}/end", s.handleEndSession)
	s.mux.HandleFunc("GET /sessions/recent", s.handleRecentSessions)
	s.mux.HandleFunc("GET /sessions/{id}", s.handleGetSession)
	s.mux.HandleFunc("DELETE /sessions/{id}", s.handleDeleteSession)

	// Observations
	s.mux.HandleFunc("POST /observations", s.handleAddObservation)
	s.mux.HandleFunc("POST /observations/passive", s.handlePassiveCapture)
	s.mux.HandleFunc("GET /observations/recent", s.handleRecentObservations)
	s.mux.HandleFunc("PATCH /observations/{id}", s.handleUpdateObservation)
	s.mux.HandleFunc("DELETE /observations/{id}", s.handleDeleteObservation)

	// Search
	s.mux.HandleFunc("GET /search", s.handleSearch)

	// Timeline
	s.mux.HandleFunc("GET /timeline", s.handleTimeline)
	s.mux.HandleFunc("GET /observations/{id}", s.handleGetObservation)

	// Prompts
	s.mux.HandleFunc("POST /prompts", s.handleAddPrompt)
	s.mux.HandleFunc("GET /prompts/recent", s.handleRecentPrompts)
	s.mux.HandleFunc("GET /prompts/search", s.handleSearchPrompts)
	s.mux.HandleFunc("DELETE /prompts/{id}", s.handleDeletePrompt)

	// Context
	s.mux.HandleFunc("GET /context", s.handleContext)

	// Export / Import
	s.mux.HandleFunc("GET /export", s.handleExport)
	s.mux.HandleFunc("POST /import", s.handleImport)

	// Stats / diagnostics
	s.mux.HandleFunc("GET /stats", s.handleStats)
	s.mux.HandleFunc("GET /doctor", s.handleDoctor)

	// Project detection / migration
	s.mux.HandleFunc("GET /project/current", s.handleCurrentProject)
	s.mux.HandleFunc("POST /projects/migrate", s.handleMigrateProject)
}

// ─── Handlers ────────────────────────────────────────────────────────────────

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID        string `json:"id"`
		Project   string `json:"project"`
		Directory string `json:"directory"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.ID == "" {
		jsonError(w, http.StatusBadRequest, "id is required")
		return
	}
	if err := s.store.CreateSession(req.ID, req.Project, req.Directory); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.notifyWrite()
	jsonResponse(w, http.StatusCreated, map[string]string{"id": req.ID})
}

func (s *Server) handleEndSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		jsonError(w, http.StatusBadRequest, "session id is required")
		return
	}
	var req struct {
		Summary string `json:"summary"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if err := s.store.EndSession(id, req.Summary); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.notifyWrite()
	jsonResponse(w, http.StatusOK, map[string]string{"status": "ended"})
}

func (s *Server) handleRecentSessions(w http.ResponseWriter, r *http.Request) {
	limit := queryInt(r, "limit", 10)
	sessions, err := s.store.RecentSessions("", limit)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, sessions)
}

func (s *Server) handleGetSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	session, err := s.store.GetSession(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			jsonError(w, http.StatusNotFound, "session not found")
			return
		}
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, session)
}

func (s *Server) handleAddObservation(w http.ResponseWriter, r *http.Request) {
	var req store.AddObservationParams
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.SessionID == "" || req.Content == "" {
		jsonError(w, http.StatusBadRequest, "session_id and content are required")
		return
	}
	id, err := s.store.AddObservation(req)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.notifyWrite()
	jsonResponse(w, http.StatusCreated, map[string]any{"id": id})
}

func (s *Server) handlePassiveCapture(w http.ResponseWriter, r *http.Request) {
	var req store.AddObservationParams
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.SessionID == "" || req.Content == "" {
		jsonError(w, http.StatusBadRequest, "session_id and content are required")
		return
	}
	if req.Type == "" {
		req.Type = "passive"
	}
	id, err := s.store.AddObservation(req)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.notifyWrite()
	jsonResponse(w, http.StatusCreated, map[string]any{"id": id})
}

func (s *Server) handleRecentObservations(w http.ResponseWriter, r *http.Request) {
	limit := queryInt(r, "limit", 20)
	obs, err := s.store.RecentObservations("", "", limit)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, obs)
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		jsonError(w, http.StatusBadRequest, "q parameter is required")
		return
	}
	opts := store.SearchOptions{
		Type:    r.URL.Query().Get("type"),
		Project: r.URL.Query().Get("project"),
		Scope:   r.URL.Query().Get("scope"),
		Limit:   queryInt(r, "limit", 10),
	}
	results, err := s.store.Search(query, opts)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, results)
}

func (s *Server) handleGetObservation(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid observation id")
		return
	}
	obs, err := s.store.GetObservation(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			jsonError(w, http.StatusNotFound, "observation not found")
			return
		}
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, obs)
}

func (s *Server) handleUpdateObservation(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid observation id")
		return
	}
	var req struct {
		Title   *string `json:"title"`
		Content *string `json:"content"`
		Type    *string `json:"type"`
		Scope   *string `json:"scope"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	params := store.UpdateObservationParams{}
	if req.Title != nil {
		params.Title = req.Title
	}
	if req.Content != nil {
		params.Content = req.Content
	}
	if req.Type != nil {
		params.Type = req.Type
	}
	if req.Scope != nil {
		params.Scope = req.Scope
	}
	if _, err := s.store.UpdateObservation(id, params); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			jsonError(w, http.StatusNotFound, "observation not found")
			return
		}
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.notifyWrite()
	jsonResponse(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) handleDeleteObservation(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid observation id")
		return
	}
	hard := queryBool(r, "hard", false)
	if err := s.store.DeleteObservation(id, hard); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.notifyWrite()
	jsonResponse(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleTimeline(w http.ResponseWriter, r *http.Request) {
	obsIDStr := r.URL.Query().Get("observation_id")
	if obsIDStr == "" {
		jsonError(w, http.StatusBadRequest, "observation_id is required")
		return
	}
	obsID, err := strconv.ParseInt(obsIDStr, 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid observation_id")
		return
	}
	before := queryInt(r, "before", 5)
	after := queryInt(r, "after", 5)
	result, err := s.store.Timeline(obsID, before, after)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, result)
}

func (s *Server) handleAddPrompt(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SessionID string `json:"session_id"`
		Content   string `json:"content"`
		Project   string `json:"project"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.SessionID == "" || req.Content == "" {
		jsonError(w, http.StatusBadRequest, "session_id and content are required")
		return
	}
	id, err := s.store.AddPrompt(store.AddPromptParams{
		SessionID: req.SessionID,
		Content:   req.Content,
		Project:   req.Project,
	})
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.notifyWrite()
	jsonResponse(w, http.StatusCreated, map[string]any{"id": id})
}

func (s *Server) handleRecentPrompts(w http.ResponseWriter, r *http.Request) {
	limit := queryInt(r, "limit", 20)
	prompts, err := s.store.RecentPrompts("", limit)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, prompts)
}

func (s *Server) handleSearchPrompts(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		jsonError(w, http.StatusBadRequest, "q parameter is required")
		return
	}
	limit := queryInt(r, "limit", 10)
	prompts, err := s.store.SearchPrompts(query, "", limit)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, prompts)
}

func (s *Server) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		jsonError(w, http.StatusBadRequest, "session id is required")
		return
	}
	if err := s.store.DeleteSession(id); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.notifyWrite()
	jsonResponse(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleDeletePrompt(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid prompt id")
		return
	}
	if err := s.store.DeletePrompt(id); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.notifyWrite()
	jsonResponse(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	data, err := s.store.Export()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=engram-export.json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		log.Printf("[engram-lite] export encode error: %v", err)
	}
}

func (s *Server) handleImport(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 100*1024*1024))
	if err != nil {
		jsonError(w, http.StatusBadRequest, "cannot read body")
		return
	}
	var data store.ExportData
	if err := json.Unmarshal(body, &data); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	result, err := s.store.Import(&data)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.notifyWrite()
	jsonResponse(w, http.StatusOK, result)
}

func (s *Server) handleContext(w http.ResponseWriter, r *http.Request) {
	project := r.URL.Query().Get("project")
	scope := r.URL.Query().Get("scope")
	ctx, err := s.store.FormatContext(project, scope)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, map[string]string{"context": ctx})
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	stats, err := loadServerStats(s.store)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, stats)
}

func (s *Server) handleDoctor(w http.ResponseWriter, r *http.Request) {
	project := r.URL.Query().Get("project")
	check := r.URL.Query().Get("check")

	project, _ = store.NormalizeProject(project)
	project = strings.TrimSpace(project)
	check = strings.TrimSpace(check)

	runner := diagnostic.NewRunner()
	scope := diagnostic.Scope{Store: s.store, Project: project, Now: time.Now()}

	var report diagnostic.Report
	var diagErr error
	if check != "" {
		report, diagErr = runner.RunOne(r.Context(), scope, check)
	} else {
		report, diagErr = runner.RunAll(r.Context(), scope)
	}
	if diagErr != nil {
		report = diagnostic.ErrorReport(project, diagErr)
	}
	jsonResponse(w, http.StatusOK, report)
}

func (s *Server) handleCurrentProject(w http.ResponseWriter, r *http.Request) {
	directory := r.URL.Query().Get("directory")
	if directory == "" {
		cwd, err := os.Getwd()
		if err != nil {
			jsonError(w, http.StatusInternalServerError, "cannot determine working directory")
			return
		}
		directory = cwd
	}

	result := projectpkg.DetectProjectFull(directory)
	resp := map[string]any{
		"project": result.Project,
		"source":  result.Source,
		"path":    result.Path,
	}
	if result.Warning != "" {
		resp["warning"] = result.Warning
	}
	if result.Error != nil {
		resp["error"] = result.Error.Error()
		if len(result.AvailableProjects) > 0 {
			resp["available_projects"] = result.AvailableProjects
		}
	}
	jsonResponse(w, http.StatusOK, resp)
}

func (s *Server) handleMigrateProject(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Sources   []string `json:"sources"`
		Canonical string   `json:"canonical"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if len(req.Sources) == 0 || req.Canonical == "" {
		jsonError(w, http.StatusBadRequest, "sources and canonical are required")
		return
	}
	result, err := s.store.MergeProjects(req.Sources, req.Canonical)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.notifyWrite()
	jsonResponse(w, http.StatusOK, result)
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func jsonResponse(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, status int, msg string) {
	jsonResponse(w, status, map[string]string{"error": msg})
}

func queryInt(r *http.Request, key string, defaultVal int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return defaultVal
	}
	return n
}

func queryBool(r *http.Request, key string, defaultVal bool) bool {
	v := strings.ToLower(strings.TrimSpace(r.URL.Query().Get(key)))
	if v == "" {
		return defaultVal
	}
	return v == "1" || v == "true" || v == "yes"
}
