package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/stxkxs/tofui/internal/handler/respond"
)

type HealthHandler struct {
	db *pgxpool.Pool
}

func NewHealthHandler(db *pgxpool.Pool) *HealthHandler {
	return &HealthHandler{db: db}
}

func (h *HealthHandler) Check(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	status := "ok"
	services := map[string]string{}

	if err := h.db.Ping(ctx); err != nil {
		status = "degraded"
		services["postgres"] = "unhealthy"
	} else {
		services["postgres"] = "ok"
	}

	httpStatus := http.StatusOK
	if status != "ok" {
		httpStatus = http.StatusServiceUnavailable
	}
	respond.JSON(w, httpStatus, map[string]any{"status": status, "services": services})
}
