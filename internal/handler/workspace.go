package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/stxkxs/tofui/internal/auth"
	"github.com/stxkxs/tofui/internal/handler/respond"
	"github.com/stxkxs/tofui/internal/service"
)

type WorkspaceHandler struct {
	svc      *service.WorkspaceService
	auditSvc *service.AuditService
}

func NewWorkspaceHandler(svc *service.WorkspaceService, auditSvc *service.AuditService) *WorkspaceHandler {
	return &WorkspaceHandler{svc: svc, auditSvc: auditSvc}
}

type CreateWorkspaceRequest struct {
	Name              string `json:"name"`
	Description       string `json:"description"`
	RepoURL           string `json:"repo_url"`
	RepoBranch        string `json:"repo_branch"`
	WorkingDir        string `json:"working_dir"`
	TofuVersion       string `json:"tofu_version"`
	Environment       string `json:"environment"`
	AutoApply         bool   `json:"auto_apply"`
	RequiresApproval  bool   `json:"requires_approval"`
	VcsTriggerEnabled bool   `json:"vcs_trigger_enabled"`
}

type UpdateWorkspaceRequest struct {
	Name              string `json:"name"`
	Description       string `json:"description"`
	RepoURL           string `json:"repo_url"`
	RepoBranch        string `json:"repo_branch"`
	WorkingDir        string `json:"working_dir"`
	TofuVersion       string `json:"tofu_version"`
	Environment       string `json:"environment"`
	AutoApply         *bool  `json:"auto_apply"`
	RequiresApproval  *bool  `json:"requires_approval"`
	VcsTriggerEnabled *bool  `json:"vcs_trigger_enabled"`
}

func (h *WorkspaceHandler) List(w http.ResponseWriter, r *http.Request) {
	userCtx := auth.GetUser(r.Context())
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	workspaces, total, err := h.svc.List(r.Context(), userCtx.OrgID, page, perPage)
	if err != nil {
		respond.ErrorWithRequest(w, r, http.StatusInternalServerError, "failed to list workspaces")
		return
	}

	respond.JSON(w, http.StatusOK, respond.ListResponse[any]{
		Data:    workspaces,
		Total:   total,
		Page:    page,
		PerPage: perPage,
	})
}

func (h *WorkspaceHandler) Get(w http.ResponseWriter, r *http.Request) {
	userCtx := auth.GetUser(r.Context())
	workspaceID := chi.URLParam(r, "workspaceID")

	workspace, err := h.svc.Get(r.Context(), workspaceID, userCtx.OrgID)
	if err != nil {
		respond.Error(w, http.StatusNotFound, "workspace not found")
		return
	}

	respond.JSON(w, http.StatusOK, workspace)
}

func (h *WorkspaceHandler) Create(w http.ResponseWriter, r *http.Request) {
	userCtx := auth.GetUser(r.Context())

	var req CreateWorkspaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" || req.RepoURL == "" {
		respond.Error(w, http.StatusBadRequest, "name and repo_url are required")
		return
	}
	if len(req.Name) > 128 {
		respond.Error(w, http.StatusBadRequest, "name must be at most 128 characters")
		return
	}
	if len(req.RepoURL) > 2048 {
		respond.Error(w, http.StatusBadRequest, "repo_url must be at most 2048 characters")
		return
	}
	if len(req.Description) > 4096 {
		respond.Error(w, http.StatusBadRequest, "description must be at most 4096 characters")
		return
	}
	if req.Environment != "" && req.Environment != "development" && req.Environment != "staging" && req.Environment != "production" {
		respond.Error(w, http.StatusBadRequest, "environment must be development, staging, or production")
		return
	}

	workspace, err := h.svc.Create(r.Context(), service.CreateWorkspaceParams{
		OrgID:             userCtx.OrgID,
		Name:              req.Name,
		Description:       req.Description,
		RepoURL:           req.RepoURL,
		RepoBranch:        req.RepoBranch,
		WorkingDir:        req.WorkingDir,
		TofuVersion:       req.TofuVersion,
		Environment:       req.Environment,
		AutoApply:         req.AutoApply,
		RequiresApproval:  req.RequiresApproval,
		VcsTriggerEnabled: req.VcsTriggerEnabled,
		CreatedBy:         userCtx.UserID,
	})
	if err != nil {
		respond.ErrorWithRequest(w, r, http.StatusInternalServerError, "failed to create workspace")
		return
	}

	ip, ua := auditContext(r)
	h.auditSvc.Log(r.Context(), service.AuditEntry{
		OrgID: userCtx.OrgID, UserID: userCtx.UserID,
		Action: "workspace.create", EntityType: "workspace", EntityID: workspace.ID,
		After: workspace, IPAddress: ip, UserAgent: ua,
	})

	respond.JSON(w, http.StatusCreated, workspace)
}

func (h *WorkspaceHandler) Update(w http.ResponseWriter, r *http.Request) {
	userCtx := auth.GetUser(r.Context())
	workspaceID := chi.URLParam(r, "workspaceID")

	var req UpdateWorkspaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.Name) > 128 {
		respond.Error(w, http.StatusBadRequest, "name must be at most 128 characters")
		return
	}
	if len(req.RepoURL) > 2048 {
		respond.Error(w, http.StatusBadRequest, "repo_url must be at most 2048 characters")
		return
	}
	if len(req.Description) > 4096 {
		respond.Error(w, http.StatusBadRequest, "description must be at most 4096 characters")
		return
	}
	if req.Environment != "" && req.Environment != "development" && req.Environment != "staging" && req.Environment != "production" {
		respond.Error(w, http.StatusBadRequest, "environment must be development, staging, or production")
		return
	}

	workspace, err := h.svc.Update(r.Context(), service.UpdateWorkspaceParams{
		ID:                workspaceID,
		OrgID:             userCtx.OrgID,
		Name:              req.Name,
		Description:       req.Description,
		RepoURL:           req.RepoURL,
		RepoBranch:        req.RepoBranch,
		WorkingDir:        req.WorkingDir,
		TofuVersion:       req.TofuVersion,
		Environment:       req.Environment,
		AutoApply:         req.AutoApply,
		RequiresApproval:  req.RequiresApproval,
		VcsTriggerEnabled: req.VcsTriggerEnabled,
	})

	if err != nil {
		respond.ErrorWithRequest(w, r, http.StatusInternalServerError, "failed to update workspace")
		return
	}

	ip, ua := auditContext(r)
	h.auditSvc.Log(r.Context(), service.AuditEntry{
		OrgID: userCtx.OrgID, UserID: userCtx.UserID,
		Action: "workspace.update", EntityType: "workspace", EntityID: workspaceID,
		After: workspace, IPAddress: ip, UserAgent: ua,
	})

	respond.JSON(w, http.StatusOK, workspace)
}

func (h *WorkspaceHandler) Lock(w http.ResponseWriter, r *http.Request) {
	userCtx := auth.GetUser(r.Context())
	workspaceID := chi.URLParam(r, "workspaceID")

	workspace, err := h.svc.Lock(r.Context(), workspaceID, userCtx.OrgID, userCtx.UserID)
	if err != nil {
		respond.Error(w, http.StatusConflict, "workspace is already locked or not found")
		return
	}

	ip, ua := auditContext(r)
	h.auditSvc.Log(r.Context(), service.AuditEntry{
		OrgID: userCtx.OrgID, UserID: userCtx.UserID,
		Action: "workspace.lock", EntityType: "workspace", EntityID: workspaceID,
		After: workspace, IPAddress: ip, UserAgent: ua,
	})

	respond.JSON(w, http.StatusOK, workspace)
}

func (h *WorkspaceHandler) Unlock(w http.ResponseWriter, r *http.Request) {
	userCtx := auth.GetUser(r.Context())
	workspaceID := chi.URLParam(r, "workspaceID")

	workspace, err := h.svc.Unlock(r.Context(), workspaceID, userCtx.OrgID)
	if err != nil {
		respond.Error(w, http.StatusNotFound, "workspace not found")
		return
	}

	ip, ua := auditContext(r)
	h.auditSvc.Log(r.Context(), service.AuditEntry{
		OrgID: userCtx.OrgID, UserID: userCtx.UserID,
		Action: "workspace.unlock", EntityType: "workspace", EntityID: workspaceID,
		After: workspace, IPAddress: ip, UserAgent: ua,
	})

	respond.JSON(w, http.StatusOK, workspace)
}

func (h *WorkspaceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userCtx := auth.GetUser(r.Context())
	workspaceID := chi.URLParam(r, "workspaceID")

	if err := h.svc.Delete(r.Context(), workspaceID, userCtx.OrgID); err != nil {
		if errors.Is(err, service.ErrWorkspaceHasRuns) {
			respond.Error(w, http.StatusConflict, "cannot delete workspace with existing runs")
			return
		}
		respond.ErrorWithRequest(w, r, http.StatusInternalServerError, "failed to delete workspace")
		return
	}

	ip, ua := auditContext(r)
	h.auditSvc.Log(r.Context(), service.AuditEntry{
		OrgID: userCtx.OrgID, UserID: userCtx.UserID,
		Action: "workspace.delete", EntityType: "workspace", EntityID: workspaceID,
		IPAddress: ip, UserAgent: ua,
	})

	respond.NoContent(w)
}
