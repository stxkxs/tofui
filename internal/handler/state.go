package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/stxkxs/tofui/internal/auth"
	"github.com/stxkxs/tofui/internal/handler/respond"
	"github.com/stxkxs/tofui/internal/repository"
	"github.com/stxkxs/tofui/internal/storage"
	"github.com/stxkxs/tofui/internal/tfstate"
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

func (h *StateHandler) Resources(w http.ResponseWriter, r *http.Request) {
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

	if h.storage == nil {
		respond.Error(w, http.StatusServiceUnavailable, "storage not configured")
		return
	}

	data, err := h.storage.GetState(r.Context(), sv.StateURL)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to fetch state")
		return
	}

	resources, err := tfstate.ParseResources(data)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to parse state")
		return
	}

	respond.JSON(w, http.StatusOK, resources)
}

func (h *StateHandler) Diff(w http.ResponseWriter, r *http.Request) {
	userCtx := auth.GetUser(r.Context())
	workspaceID := chi.URLParam(r, "workspaceID")

	fromSerial, err := strconv.Atoi(r.URL.Query().Get("from"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid 'from' serial")
		return
	}
	toSerial, err := strconv.Atoi(r.URL.Query().Get("to"))
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid 'to' serial")
		return
	}

	if h.storage == nil {
		respond.Error(w, http.StatusServiceUnavailable, "storage not configured")
		return
	}

	fromSV, err := h.queries.GetStateVersionBySerial(r.Context(), repository.GetStateVersionBySerialParams{
		WorkspaceID: workspaceID,
		OrgID:       userCtx.OrgID,
		Serial:      int32(fromSerial),
	})
	if err != nil {
		respond.Error(w, http.StatusNotFound, "state version not found for 'from' serial")
		return
	}

	toSV, err := h.queries.GetStateVersionBySerial(r.Context(), repository.GetStateVersionBySerialParams{
		WorkspaceID: workspaceID,
		OrgID:       userCtx.OrgID,
		Serial:      int32(toSerial),
	})
	if err != nil {
		respond.Error(w, http.StatusNotFound, "state version not found for 'to' serial")
		return
	}

	fromData, err := h.storage.GetState(r.Context(), fromSV.StateURL)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to fetch 'from' state")
		return
	}

	toData, err := h.storage.GetState(r.Context(), toSV.StateURL)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to fetch 'to' state")
		return
	}

	diff, err := tfstate.DiffStates(fromData, toData)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to diff states")
		return
	}

	respond.JSON(w, http.StatusOK, diff)
}
