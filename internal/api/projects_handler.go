package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lunoxd/cobalt/internal/auth"
	"github.com/lunoxd/cobalt/internal/projects"
)

// ProjectsHandler handles HTTP routes for projects.
type ProjectsHandler struct {
	service *projects.Service
}

// NewProjectsHandler creates a new ProjectsHandler.
func NewProjectsHandler(svc *projects.Service) *ProjectsHandler {
	return &ProjectsHandler{service: svc}
}

// Routes returns the sub-router for /api/projects.
func (h *ProjectsHandler) Routes() http.Handler {
	r := chi.NewRouter()

	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Get("/{id}", h.Get)
	r.Patch("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)

	return r
}

// List handles GET /api/projects.
func (h *ProjectsHandler) List(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.GetIdentity(r.Context())
	if !ok || identity == nil {
		Unauthorized(w, "Authentication required to list projects.")
		return
	}

	limit := 20
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

	items, total, err := h.service.ListProjects(r.Context(), identity, limit, offset)
	if err != nil {
		InternalError(w, err, "failed to list projects")
		return
	}

	JSON(w, http.StatusOK, map[string]any{
		"data":   items,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// Create handles POST /api/projects.
func (h *ProjectsHandler) Create(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.GetIdentity(r.Context())
	if !ok || identity == nil {
		Unauthorized(w, "Authentication required to create a project.")
		return
	}

	var input projects.CreateProjectInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		BadRequest(w, "Invalid JSON request body: "+err.Error())
		return
	}

	project, err := h.service.CreateProject(r.Context(), identity.UserID, input)
	if err != nil {
		BadRequest(w, err.Error())
		return
	}

	JSON(w, http.StatusCreated, map[string]any{
		"data": project,
	})
}

// Get handles GET /api/projects/{id}.
func (h *ProjectsHandler) Get(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		BadRequest(w, "Invalid project ID format (UUID expected)")
		return
	}

	identity, _ := auth.GetIdentity(r.Context())
	project, err := h.service.GetProject(r.Context(), id, identity)
	if err != nil {
		if errors.Is(err, projects.ErrNotFound) {
			NotFound(w, "Project not found")
			return
		}
		if errors.Is(err, projects.ErrUnauthorizedAccess) {
			Forbidden(w, "You do not have permission to view this project")
			return
		}
		InternalError(w, err, "failed to fetch project")
		return
	}

	JSON(w, http.StatusOK, map[string]any{
		"data": project,
	})
}

// Update handles PATCH /api/projects/{id}.
func (h *ProjectsHandler) Update(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.GetIdentity(r.Context())
	if !ok || identity == nil {
		Unauthorized(w, "Authentication required to update a project.")
		return
	}

	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		BadRequest(w, "Invalid project ID format (UUID expected)")
		return
	}

	var input projects.UpdateProjectInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		BadRequest(w, "Invalid JSON request body: "+err.Error())
		return
	}

	project, err := h.service.UpdateProject(r.Context(), id, identity, input)
	if err != nil {
		if errors.Is(err, projects.ErrNotFound) {
			NotFound(w, "Project not found")
			return
		}
		if errors.Is(err, projects.ErrUnauthorizedAccess) {
			Forbidden(w, "You do not have permission to edit this project")
			return
		}
		BadRequest(w, err.Error())
		return
	}

	JSON(w, http.StatusOK, map[string]any{
		"data": project,
	})
}

// Delete handles DELETE /api/projects/{id}.
func (h *ProjectsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.GetIdentity(r.Context())
	if !ok || identity == nil {
		Unauthorized(w, "Authentication required to delete a project.")
		return
	}

	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		BadRequest(w, "Invalid project ID format (UUID expected)")
		return
	}

	err = h.service.DeleteProject(r.Context(), id, identity)
	if err != nil {
		if errors.Is(err, projects.ErrNotFound) {
			NotFound(w, "Project not found")
			return
		}
		if errors.Is(err, projects.ErrUnauthorizedAccess) {
			Forbidden(w, "You do not have permission to delete this project")
			return
		}
		InternalError(w, err, "failed to delete project")
		return
	}

	JSON(w, http.StatusOK, map[string]any{
		"message": "Project deleted successfully",
		"id":      idParam,
	})
}
