package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/stxkxs/tofui/internal/auth"
	"github.com/stxkxs/tofui/internal/handler/respond"
	"github.com/stxkxs/tofui/internal/repository"
	"github.com/stxkxs/tofui/internal/storage"
)

type StateHandler struct {
	queries *repository.Queries
	storage *storage.S3Storage
}

func NewStateHandler(queries *repository.Queries, store *storage.S3Storage) *StateHandler {
	return &StateHandler{queries: queries, storage: store}
}

func (h *StateHandler) List(w http.ResponseWriter, r *http.Request) {
	userCtx := auth.GetUser(r.Context())
	workspaceID := chi.URLParam(r, "workspaceID")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	versions, err := h.queries.ListStateVersionsByWorkspace(r.Context(), repository.ListStateVersionsParams{
		WorkspaceID: workspaceID,
		OrgID:       userCtx.OrgID,
		Limit:       int32(perPage),
		Offset:      int32((page - 1) * perPage),
	})
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to list state versions")
		return
	}

	respond.JSON(w, http.StatusOK, versions)
}

func (h *StateHandler) GetCurrent(w http.ResponseWriter, r *http.Request) {
	userCtx := auth.GetUser(r.Context())
	workspaceID := chi.URLParam(r, "workspaceID")

	sv, err := h.queries.GetLatestStateVersion(r.Context(), repository.GetLatestStateVersionParams{
		WorkspaceID: workspaceID,
		OrgID:       userCtx.OrgID,
	})
	if err != nil {
		respond.Error(w, http.StatusNotFound, "no state found")
		return
	}

	respond.JSON(w, http.StatusOK, sv)
}

func (h *StateHandler) Get(w http.ResponseWriter, r *http.Request) {
	userCtx := auth.GetUser(r.Context())
	stateID := chi.URLParam(r, "stateID")

	sv, err := h.queries.GetStateVersion(r.Context(), repository.GetStateVersionParams{
		ID:    stateID,
		OrgID: userCtx.OrgID,
	})
	if err != nil {
		respond.Error(w, http.StatusNotFound, "state version not found")
		return
	}

	respond.JSON(w, http.StatusOK, sv)
}

func (h *StateHandler) Download(w http.ResponseWriter, r *http.Request) {
	userCtx := auth.GetUser(r.Context())
	stateID := chi.URLParam(r, "stateID")

	sv, err := h.queries.GetStateVersion(r.Context(), repository.GetStateVersionParams{
		ID:    stateID,
		OrgID: userCtx.OrgID,
	})
	if err != nil {
		respond.Error(w, http.StatusNotFound, "state version not found")
		return
	}

	if h.storage == nil {
		respond.Error(w, http.StatusServiceUnavailable, "storage not configured")
		return
	}

	data, err := h.storage.GetState(r.Context(), sv.StateURL)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to download state")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=terraform.tfstate")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}
