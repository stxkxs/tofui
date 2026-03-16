package handler

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/oklog/ulid/v2"

	"github.com/stxkxs/tofui/internal/auth"
	"github.com/stxkxs/tofui/internal/handler/respond"
	"github.com/stxkxs/tofui/internal/repository"
	"github.com/stxkxs/tofui/internal/secrets"
	"github.com/stxkxs/tofui/internal/service"
	"github.com/stxkxs/tofui/internal/storage"
	"github.com/stxkxs/tofui/internal/tfparse"
)

type VariableHandler struct {
	queries      *repository.Queries
	encryptor    *secrets.Encryptor
	auditSvc     *service.AuditService
	workspaceSvc *service.WorkspaceService
	storage      *storage.S3Storage
}

func NewVariableHandler(queries *repository.Queries, encryptor *secrets.Encryptor, auditSvc *service.AuditService, workspaceSvc *service.WorkspaceService, store *storage.S3Storage) *VariableHandler {
	return &VariableHandler{queries: queries, encryptor: encryptor, auditSvc: auditSvc, workspaceSvc: workspaceSvc, storage: store}
}

type CreateVariableRequest struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Sensitive   bool   `json:"sensitive"`
	Category    string `json:"category"`
	Description string `json:"description"`
}

type VariableResponse struct {
	repository.WorkspaceVariable
}

func (h *VariableHandler) List(w http.ResponseWriter, r *http.Request) {
	userCtx := auth.GetUser(r.Context())
	workspaceID := chi.URLParam(r, "workspaceID")

	vars, err := h.queries.ListWorkspaceVariables(r.Context(), repository.ListWorkspaceVariablesParams{
		WorkspaceID: workspaceID, OrgID: userCtx.OrgID,
	})
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to list variables")
		return
	}

	// Redact sensitive values in response
	for i := range vars {
		if vars[i].Sensitive {
			vars[i].Value = "***"
		}
	}

	respond.JSON(w, http.StatusOK, vars)
}

func (h *VariableHandler) Create(w http.ResponseWriter, r *http.Request) {
	userCtx := auth.GetUser(r.Context())
	workspaceID := chi.URLParam(r, "workspaceID")

	var req CreateVariableRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Key == "" {
		respond.Error(w, http.StatusBadRequest, "key is required")
		return
	}
	if len(req.Key) > 256 {
		respond.Error(w, http.StatusBadRequest, "key must be at most 256 characters")
		return
	}
	if len(req.Value) > 65536 {
		respond.Error(w, http.StatusBadRequest, "value must be at most 64KB")
		return
	}
	if req.Category == "" {
		req.Category = "terraform"
	}
	if req.Category != "terraform" && req.Category != "env" {
		respond.Error(w, http.StatusBadRequest, "category must be 'terraform' or 'env'")
		return
	}

	value := req.Value
	if req.Sensitive && h.encryptor != nil {
		encrypted, err := h.encryptor.Encrypt(req.Value)
		if err != nil {
			respond.Error(w, http.StatusInternalServerError, "failed to encrypt value")
			return
		}
		value = encrypted
	}

	v, err := h.queries.CreateWorkspaceVariable(r.Context(), repository.CreateWorkspaceVariableParams{
		ID:          ulid.Make().String(),
		WorkspaceID: workspaceID,
		OrgID:       userCtx.OrgID,
		Key:         req.Key,
		Value:       value,
		Sensitive:   req.Sensitive,
		Category:    req.Category,
		Description: req.Description,
	})
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to create variable")
		return
	}

	ip, ua := auditContext(r)
	auditVar := v
	auditVar.Value = "***" // never log variable values
	h.auditSvc.Log(r.Context(), service.AuditEntry{
		OrgID: userCtx.OrgID, UserID: userCtx.UserID,
		Action: "variable.create", EntityType: "variable", EntityID: v.ID,
		After: auditVar, IPAddress: ip, UserAgent: ua,
	})

	if v.Sensitive {
		v.Value = "***"
	}

	respond.JSON(w, http.StatusCreated, v)
}

func (h *VariableHandler) Update(w http.ResponseWriter, r *http.Request) {
	userCtx := auth.GetUser(r.Context())
	varID := chi.URLParam(r, "variableID")

	// Fetch current state for audit log
	before, err := h.queries.GetWorkspaceVariable(r.Context(), repository.GetWorkspaceVariableParams{
		ID: varID, OrgID: userCtx.OrgID,
	})
	if err != nil {
		respond.Error(w, http.StatusNotFound, "variable not found")
		return
	}

	var req CreateVariableRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.Key) > 256 {
		respond.Error(w, http.StatusBadRequest, "key must be at most 256 characters")
		return
	}
	if len(req.Value) > 65536 {
		respond.Error(w, http.StatusBadRequest, "value must be at most 64KB")
		return
	}
	if req.Category != "" && req.Category != "terraform" && req.Category != "env" {
		respond.Error(w, http.StatusBadRequest, "category must be 'terraform' or 'env'")
		return
	}

	value := req.Value
	if req.Sensitive && h.encryptor != nil {
		encrypted, err := h.encryptor.Encrypt(req.Value)
		if err != nil {
			respond.Error(w, http.StatusInternalServerError, "failed to encrypt value")
			return
		}
		value = encrypted
	}

	v, err := h.queries.UpdateWorkspaceVariable(r.Context(), repository.UpdateWorkspaceVariableParams{
		ID: varID, OrgID: userCtx.OrgID, Value: value, Sensitive: req.Sensitive, Description: req.Description,
	})
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to update variable")
		return
	}

	ip, ua := auditContext(r)
	auditBefore := before
	auditBefore.Value = "***"
	auditVar := v
	auditVar.Value = "***"
	h.auditSvc.Log(r.Context(), service.AuditEntry{
		OrgID: userCtx.OrgID, UserID: userCtx.UserID,
		Action: "variable.update", EntityType: "variable", EntityID: varID,
		Before: auditBefore, After: auditVar, IPAddress: ip, UserAgent: ua,
	})

	if v.Sensitive {
		v.Value = "***"
	}

	respond.JSON(w, http.StatusOK, v)
}

func (h *VariableHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userCtx := auth.GetUser(r.Context())
	varID := chi.URLParam(r, "variableID")

	if err := h.queries.DeleteWorkspaceVariable(r.Context(), repository.DeleteWorkspaceVariableParams{
		ID: varID, OrgID: userCtx.OrgID,
	}); err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to delete variable")
		return
	}

	ip, ua := auditContext(r)
	h.auditSvc.Log(r.Context(), service.AuditEntry{
		OrgID: userCtx.OrgID, UserID: userCtx.UserID,
		Action: "variable.delete", EntityType: "variable", EntityID: varID,
		IPAddress: ip, UserAgent: ua,
	})

	respond.NoContent(w)
}

type DiscoverVariableResponse struct {
	Name        string  `json:"name"`
	Type        string  `json:"type,omitempty"`
	Description string  `json:"description,omitempty"`
	Default     *string `json:"default,omitempty"`
	Required    bool    `json:"required"`
	Configured  bool    `json:"configured"`
}

func (h *VariableHandler) Discover(w http.ResponseWriter, r *http.Request) {
	userCtx := auth.GetUser(r.Context())
	workspaceID := chi.URLParam(r, "workspaceID")

	ws, err := h.workspaceSvc.Get(r.Context(), workspaceID, userCtx.OrgID)
	if err != nil {
		respond.Error(w, http.StatusNotFound, "workspace not found")
		return
	}

	tmpDir, err := os.MkdirTemp("", "tofui-discover-*")
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to create temp directory")
		return
	}
	defer os.RemoveAll(tmpDir)

	if ws.Source == "upload" {
		// Extract config archive from S3
		if ws.CurrentConfigVersionID == "" {
			respond.Error(w, http.StatusBadRequest, "no configuration uploaded yet")
			return
		}
		if h.storage == nil {
			respond.Error(w, http.StatusServiceUnavailable, "storage not configured")
			return
		}
		key := fmt.Sprintf("configs/%s/%s.tar.gz", workspaceID, ws.CurrentConfigVersionID)
		data, err := h.storage.GetConfigArchive(r.Context(), key)
		if err != nil {
			respond.Error(w, http.StatusInternalServerError, "failed to download configuration")
			return
		}
		if err := extractDiscoverArchive(data, tmpDir); err != nil {
			respond.Error(w, http.StatusInternalServerError, "failed to extract configuration")
			return
		}
	} else {
		if ws.RepoURL == "" {
			respond.Error(w, http.StatusBadRequest, "workspace has no repository URL configured")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", "--branch", ws.RepoBranch, ws.RepoURL, tmpDir)
		if output, err := cmd.CombinedOutput(); err != nil {
			slog.Error("git clone failed", "error", err, "output", string(output), "repo", ws.RepoURL)
			respond.Error(w, http.StatusBadGateway, "failed to clone repository")
			return
		}
	}

	parseDir := tmpDir
	if ws.WorkingDir != "" && ws.WorkingDir != "." {
		parseDir = filepath.Join(tmpDir, ws.WorkingDir)
	}

	discovered, err := tfparse.ParseDirectory(parseDir)
	if err != nil {
		respond.Error(w, http.StatusBadGateway, "failed to parse terraform files")
		return
	}

	existing, err := h.queries.ListWorkspaceVariables(r.Context(), repository.ListWorkspaceVariablesParams{
		WorkspaceID: workspaceID, OrgID: userCtx.OrgID,
	})
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to list existing variables")
		return
	}

	configuredKeys := make(map[string]bool, len(existing))
	for _, v := range existing {
		configuredKeys[v.Key] = true
	}

	result := make([]DiscoverVariableResponse, len(discovered))
	for i, d := range discovered {
		result[i] = DiscoverVariableResponse{
			Name:        d.Name,
			Type:        d.Type,
			Description: d.Description,
			Default:     d.Default,
			Required:    d.Required,
			Configured:  configuredKeys[d.Name],
		}
	}

	respond.JSON(w, http.StatusOK, result)
}

func extractDiscoverArchive(data []byte, destDir string) error {
	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("invalid gzip: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		cleanName := filepath.Clean(hdr.Name)
		if strings.HasPrefix(cleanName, "..") {
			continue
		}
		target := filepath.Join(destDir, cleanName)
		switch hdr.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(target, 0755)
		case tar.TypeReg:
			os.MkdirAll(filepath.Dir(target), 0755)
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			io.Copy(f, tr)
			f.Close()
		}
	}
	return nil
}

type BulkCreateVariablesRequest struct {
	Variables []CreateVariableRequest `json:"variables"`
}

func (h *VariableHandler) BulkCreate(w http.ResponseWriter, r *http.Request) {
	userCtx := auth.GetUser(r.Context())
	workspaceID := chi.URLParam(r, "workspaceID")

	var req BulkCreateVariablesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.Variables) == 0 {
		respond.Error(w, http.StatusBadRequest, "variables array is required")
		return
	}
	if len(req.Variables) > 50 {
		respond.Error(w, http.StatusBadRequest, "maximum 50 variables per batch")
		return
	}

	// Check for duplicate keys within the batch
	seen := make(map[string]bool, len(req.Variables))
	for _, v := range req.Variables {
		if v.Key == "" {
			respond.Error(w, http.StatusBadRequest, "all variables must have a key")
			return
		}
		if seen[v.Key] {
			respond.Error(w, http.StatusBadRequest, "duplicate key: "+v.Key)
			return
		}
		seen[v.Key] = true
	}

	created := make([]repository.WorkspaceVariable, 0, len(req.Variables))
	ip, ua := auditContext(r)

	for _, rv := range req.Variables {
		if rv.Category == "" {
			rv.Category = "terraform"
		}
		if rv.Category != "terraform" && rv.Category != "env" {
			respond.Error(w, http.StatusBadRequest, "category must be 'terraform' or 'env' for key: "+rv.Key)
			return
		}

		value := rv.Value
		if rv.Sensitive && h.encryptor != nil {
			encrypted, err := h.encryptor.Encrypt(rv.Value)
			if err != nil {
				respond.Error(w, http.StatusInternalServerError, "failed to encrypt value")
				return
			}
			value = encrypted
		}

		v, err := h.queries.CreateWorkspaceVariable(r.Context(), repository.CreateWorkspaceVariableParams{
			ID:          ulid.Make().String(),
			WorkspaceID: workspaceID,
			OrgID:       userCtx.OrgID,
			Key:         rv.Key,
			Value:       value,
			Sensitive:   rv.Sensitive,
			Category:    rv.Category,
			Description: rv.Description,
		})
		if err != nil {
			respond.Error(w, http.StatusInternalServerError, "failed to create variable: "+rv.Key)
			return
		}

		auditVar := v
		auditVar.Value = "***"
		h.auditSvc.Log(r.Context(), service.AuditEntry{
			OrgID: userCtx.OrgID, UserID: userCtx.UserID,
			Action: "variable.create", EntityType: "variable", EntityID: v.ID,
			After: auditVar, IPAddress: ip, UserAgent: ua,
		})

		if v.Sensitive {
			v.Value = "***"
		}
		created = append(created, v)
	}

	respond.JSON(w, http.StatusCreated, created)
}

func (h *VariableHandler) RevealValue(w http.ResponseWriter, r *http.Request) {
	userCtx := auth.GetUser(r.Context())
	varID := chi.URLParam(r, "variableID")

	v, err := h.queries.GetWorkspaceVariable(r.Context(), repository.GetWorkspaceVariableParams{
		ID: varID, OrgID: userCtx.OrgID,
	})
	if err != nil {
		respond.Error(w, http.StatusNotFound, "variable not found")
		return
	}

	value := v.Value
	if v.Sensitive && h.encryptor != nil {
		decrypted, err := h.encryptor.Decrypt(v.Value)
		if err != nil {
			respond.Error(w, http.StatusInternalServerError, "failed to decrypt variable")
			return
		}
		value = decrypted
	}

	ip, ua := auditContext(r)
	h.auditSvc.Log(r.Context(), service.AuditEntry{
		OrgID: userCtx.OrgID, UserID: userCtx.UserID,
		Action: "variable.reveal", EntityType: "variable", EntityID: varID,
		IPAddress: ip, UserAgent: ua,
	})

	respond.JSON(w, http.StatusOK, map[string]string{"value": value})
}
