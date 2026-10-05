package api

import (
	"encoding/json"
	"net/http"
	"runtime"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lunoxd/cobalt/internal/auth"
	"github.com/lunoxd/cobalt/internal/database"
	"github.com/lunoxd/cobalt/internal/inspector"
)

// AdminHandler handles administration and database inspection routes.
type AdminHandler struct {
	db          *database.DB
	apiKeys     *auth.APIKeyService
	inspector   *inspector.Inspector
}

// NewAdminHandler creates an AdminHandler.
func NewAdminHandler(db *database.DB, apiKeys *auth.APIKeyService, ins *inspector.Inspector) *AdminHandler {
	return &AdminHandler{
		db:        db,
		apiKeys:   apiKeys,
		inspector: ins,
	}
}

// Routes sets up the admin sub-router.
func (h *AdminHandler) Routes() http.Handler {
	r := chi.NewRouter()

	// All admin routes require admin authorization
	r.Use(auth.RequireAdmin)

	// System & Database stats
	r.Get("/stats", h.GetSystemStats)

	// API Key management
	r.Get("/api-keys", h.ListAPIKeys)
	r.Post("/api-keys", h.CreateAPIKey)
	r.Delete("/api-keys/{id}", h.RevokeAPIKey)

	// Database Inspector
	r.Route("/inspector", func(ir chi.Router) {
		ir.Get("/stats", h.GetDatabaseStats)
		ir.Get("/tables", h.ListTables)
		ir.Get("/tables/{table}", h.DescribeTable)
		ir.Get("/tables/{table}/rows", h.GetRows)
		ir.Post("/tables/{table}/rows", h.InsertRow)
		ir.Patch("/tables/{table}/rows", h.UpdateRow)
		ir.Delete("/tables/{table}/rows", h.DeleteRow)
	})

	return r
}

// GetSystemStats returns general system and DB metrics.
func (h *AdminHandler) GetSystemStats(w http.ResponseWriter, r *http.Request) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	dbStats, err := h.inspector.DatabaseStats(r.Context())
	if err != nil {
		// Log but still return partial stats
		dbStats = &inspector.DatabaseStats{Version: "unknown"}
	}

	var poolStats map[string]any = map[string]any{"status": "disconnected"}
	if h.db != nil {
		poolStats = h.db.Stats()
	}

	JSON(w, http.StatusOK, map[string]any{
		"system": map[string]any{
			"go_version":     runtime.Version(),
			"goroutines":     runtime.NumGoroutine(),
			"alloc_mb":       memStats.Alloc / 1024 / 1024,
			"total_alloc_mb": memStats.TotalAlloc / 1024 / 1024,
			"sys_mb":         memStats.Sys / 1024 / 1024,
			"num_gc":         memStats.NumGC,
			"uptime":         time.Since(startTime).String(),
		},
		"database": dbStats,
		"pool":     poolStats,
	})
}

// ListAPIKeys handles listing API keys.
func (h *AdminHandler) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	caller, _ := auth.GetIdentity(r.Context())
	keys, err := h.apiKeys.List(r.Context(), caller)
	if err != nil {
		InternalError(w, err, "failed to list api keys")
		return
	}
	JSON(w, http.StatusOK, map[string]any{"data": keys})
}

type createAPIKeyReq struct {
	Name      string   `json:"name"`
	OwnerID   string   `json:"owner_id"`
	Scopes    []string `json:"scopes"`
	ExpiresIn string   `json:"expires_in,omitempty"` // e.g. "720h"
}

// CreateAPIKey generates and returns a new API key.
func (h *AdminHandler) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	var req createAPIKeyReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "Invalid request payload: "+err.Error())
		return
	}

	caller, _ := auth.GetIdentity(r.Context())
	ownerID := req.OwnerID
	if ownerID == "" {
		ownerID = caller.UserID
	}

	var expiresIn *time.Duration
	if req.ExpiresIn != "" {
		d, err := time.ParseDuration(req.ExpiresIn)
		if err != nil {
			BadRequest(w, "Invalid expires_in format (e.g. 24h, 720h)")
			return
		}
		expiresIn = &d
	}

	res, err := h.apiKeys.Create(r.Context(), ownerID, req.Name, req.Scopes, expiresIn)
	if err != nil {
		BadRequest(w, err.Error())
		return
	}

	JSON(w, http.StatusCreated, map[string]any{
		"data":    res,
		"warning": "Store plaintext_key safely. It will never be displayed again.",
	})
}

// RevokeAPIKey revokes an API key.
func (h *AdminHandler) RevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		BadRequest(w, "Invalid API key UUID format")
		return
	}

	caller, _ := auth.GetIdentity(r.Context())
	if err := h.apiKeys.Revoke(r.Context(), id, caller); err != nil {
		BadRequest(w, err.Error())
		return
	}

	JSON(w, http.StatusOK, map[string]any{
		"message": "API key revoked successfully",
		"id":      idParam,
	})
}

// GetDatabaseStats returns database inspector stats.
func (h *AdminHandler) GetDatabaseStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.inspector.DatabaseStats(r.Context())
	if err != nil {
		InternalError(w, err, "failed to get database stats")
		return
	}
	JSON(w, http.StatusOK, map[string]any{"data": stats})
}

// ListTables handles table listing.
func (h *AdminHandler) ListTables(w http.ResponseWriter, r *http.Request) {
	tables, err := h.inspector.ListTables(r.Context())
	if err != nil {
		InternalError(w, err, "failed to list tables")
		return
	}
	JSON(w, http.StatusOK, map[string]any{"data": tables})
}

// DescribeTable handles table description.
func (h *AdminHandler) DescribeTable(w http.ResponseWriter, r *http.Request) {
	tableName := chi.URLParam(r, "table")
	detail, err := h.inspector.DescribeTable(r.Context(), tableName)
	if err != nil {
		BadRequest(w, err.Error())
		return
	}
	JSON(w, http.StatusOK, map[string]any{"data": detail})
}

// GetRows handles paginated table data inspection.
func (h *AdminHandler) GetRows(w http.ResponseWriter, r *http.Request) {
	tableName := chi.URLParam(r, "table")
	limit := 25
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}
	orderBy := r.URL.Query().Get("order_by")
	sortOrder := r.URL.Query().Get("sort")

	result, err := h.inspector.GetRows(r.Context(), tableName, limit, offset, orderBy, sortOrder)
	if err != nil {
		BadRequest(w, err.Error())
		return
	}

	JSON(w, http.StatusOK, result)
}

// InsertRow handles inserting a record into a table.
func (h *AdminHandler) InsertRow(w http.ResponseWriter, r *http.Request) {
	tableName := chi.URLParam(r, "table")
	var data map[string]any
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		BadRequest(w, "Invalid JSON data: "+err.Error())
		return
	}

	res, err := h.inspector.InsertRow(r.Context(), tableName, data)
	if err != nil {
		BadRequest(w, err.Error())
		return
	}

	JSON(w, http.StatusCreated, map[string]any{"data": res})
}

type updateRowReq struct {
	PrimaryKeyColumn string         `json:"pk_column"`
	PrimaryKeyValue  any            `json:"pk_value"`
	Data             map[string]any `json:"data"`
}

// UpdateRow handles updating a record in a table.
func (h *AdminHandler) UpdateRow(w http.ResponseWriter, r *http.Request) {
	tableName := chi.URLParam(r, "table")
	var req updateRowReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "Invalid JSON data: "+err.Error())
		return
	}

	if req.PrimaryKeyColumn == "" || req.PrimaryKeyValue == nil {
		BadRequest(w, "pk_column and pk_value are required")
		return
	}

	res, err := h.inspector.UpdateRow(r.Context(), tableName, req.PrimaryKeyColumn, req.PrimaryKeyValue, req.Data)
	if err != nil {
		BadRequest(w, err.Error())
		return
	}

	JSON(w, http.StatusOK, map[string]any{"data": res})
}

// DeleteRow handles deleting a row from a table.
func (h *AdminHandler) DeleteRow(w http.ResponseWriter, r *http.Request) {
	tableName := chi.URLParam(r, "table")
	pkCol := r.URL.Query().Get("pk_column")
	pkVal := r.URL.Query().Get("pk_value")

	if pkCol == "" || pkVal == "" {
		BadRequest(w, "pk_column and pk_value query parameters are required")
		return
	}

	err := h.inspector.DeleteRow(r.Context(), tableName, pkCol, pkVal)
	if err != nil {
		BadRequest(w, err.Error())
		return
	}

	JSON(w, http.StatusOK, map[string]any{"message": "Row deleted successfully"})
}
