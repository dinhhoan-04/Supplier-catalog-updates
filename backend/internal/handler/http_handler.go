package handler

import (
	"catalog-sync/internal/domain"
	"catalog-sync/internal/repository/postgres"
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type SyncEngineInterface interface {
	StartNewSync(scenario string) (*domain.SyncJob, error)
	ResumeSync(jobID string) (*domain.SyncJob, error)
	SimulateInterruption() error
}

type HTTPHandler struct {
	syncEngine  SyncEngineInterface
	productRepo *postgres.ProductRepository
	jobRepo     *postgres.JobRepository
	sseHub      *SSEHub
	db          *postgres.DB
}

func NewHTTPHandler(
	syncEngine SyncEngineInterface,
	productRepo *postgres.ProductRepository,
	jobRepo *postgres.JobRepository,
	sseHub *SSEHub,
	db *postgres.DB,
) *HTTPHandler {
	return &HTTPHandler{
		syncEngine:  syncEngine,
		productRepo: productRepo,
		jobRepo:     jobRepo,
		sseHub:      sseHub,
		db:          db,
	}
}

func jsonResponse(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(data)
}

func errorResponse(w http.ResponseWriter, code int, message string) {
	jsonResponse(w, code, map[string]string{"error": message})
}

// StartSync handles POST /api/sync/start
func (h *HTTPHandler) StartSync(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Scenario string `json:"scenario"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Scenario == "" {
		req.Scenario = "standard"
	}

	job, err := h.syncEngine.StartNewSync(req.Scenario)
	if err != nil {
		errorResponse(w, http.StatusConflict, err.Error())
		return
	}

	jsonResponse(w, http.StatusAccepted, job)
}

// ResumeSync handles POST /api/sync/resume
func (h *HTTPHandler) ResumeSync(w http.ResponseWriter, r *http.Request) {
	var req struct {
		JobID string `json:"job_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.JobID == "" {
		// If job_id not provided, find the latest interrupted job
		latest, err := h.jobRepo.GetLatestJob(r.Context())
		if err != nil || latest == nil {
			errorResponse(w, http.StatusBadRequest, "No interrupted job found to resume")
			return
		}
		req.JobID = latest.ID
	}

	job, err := h.syncEngine.ResumeSync(req.JobID)
	if err != nil {
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, job)
}

// SimulateInterruption handles POST /api/sync/simulate-interrupt
func (h *HTTPHandler) SimulateInterruption(w http.ResponseWriter, r *http.Request) {
	if err := h.syncEngine.SimulateInterruption(); err != nil {
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, map[string]string{"message": "Interruption signal triggered successfully"})
}

// GetStatus handles GET /api/sync/status
func (h *HTTPHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	job, err := h.jobRepo.GetLatestJob(r.Context())
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	if job == nil {
		jsonResponse(w, http.StatusOK, map[string]interface{}{"status": "IDLE", "message": "No sync jobs have been run yet"})
		return
	}
	jsonResponse(w, http.StatusOK, job)
}

// GetChanges handles GET /api/sync/changes?job_id=...
func (h *HTTPHandler) GetChanges(w http.ResponseWriter, r *http.Request) {
	jobID := r.URL.Query().Get("job_id")
	if jobID == "" {
		latest, err := h.jobRepo.GetLatestJob(r.Context())
		if err != nil || latest == nil {
			jsonResponse(w, http.StatusOK, map[string]interface{}{"items": []domain.ChangeLog{}, "total": 0})
			return
		}
		jobID = latest.ID
	}

	limit := 100
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	logs, total, err := h.jobRepo.ListChangeLogs(r.Context(), jobID, limit, offset)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"job_id": jobID,
		"items":  logs,
		"total":  total,
	})
}

// GetIssues handles GET /api/sync/issues?job_id=...&resolved=false
func (h *HTTPHandler) GetIssues(w http.ResponseWriter, r *http.Request) {
	jobID := r.URL.Query().Get("job_id")
	var resolved *bool
	if resStr := r.URL.Query().Get("resolved"); resStr != "" {
		b := resStr == "true"
		resolved = &b
	}

	issues, err := h.jobRepo.ListIssues(r.Context(), jobID, resolved)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"items": issues,
		"total": len(issues),
	})
}

// ResolveIssue handles POST /api/sync/issues/{id}/resolve
func (h *HTTPHandler) ResolveIssue(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	issueID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		errorResponse(w, http.StatusBadRequest, "Invalid issue ID")
		return
	}

	if err := h.jobRepo.ResolveIssue(r.Context(), issueID); err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"message": "Issue marked as resolved"})
}

// ListProducts handles GET /api/products?status=...&search=...&limit=...&offset=...
func (h *HTTPHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	search := r.URL.Query().Get("search")
	limit := 50
	offset := 0

	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	products, total, err := h.productRepo.ListProducts(r.Context(), status, search, limit, offset)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"items":  products,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// ResetCatalog handles POST /api/catalog/reset (clears products and jobs for clean re-demo)
func (h *HTTPHandler) ResetCatalog(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	_, err := h.db.ExecContext(ctx, "TRUNCATE products, catalog_sync_jobs, job_batch_checkpoints, catalog_change_logs, catalog_update_issues RESTART IDENTITY CASCADE;")
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.sseHub.Broadcast("catalog_reset", map[string]string{"message": "Catalog wiped clean for new demonstration"})
	jsonResponse(w, http.StatusOK, map[string]string{"message": "Catalog wiped successfully"})
}
