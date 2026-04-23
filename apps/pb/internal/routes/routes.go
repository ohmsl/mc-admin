package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/ohmsl/mc-admin/apps/pb/internal/audit"
	"github.com/ohmsl/mc-admin/apps/pb/internal/authz"
	"github.com/ohmsl/mc-admin/apps/pb/internal/config"
	"github.com/ohmsl/mc-admin/apps/pb/internal/mc"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/tools/security"
)

const (
	StatusRouteMethod  = http.MethodGet
	StatusRoutePath    = "/api/mc/status"
	ExecuteRouteMethod = http.MethodPost
	ExecuteRoutePath   = "/api/mc/execute"
)

type executeRequest struct {
	Action    string          `json:"action"`
	Payload   json.RawMessage `json:"payload"`
	RequestID string          `json:"requestId"`
}

type executeResponse struct {
	OK         bool      `json:"ok"`
	Action     string    `json:"action"`
	Message    string    `json:"message"`
	AuditLogID string    `json:"auditLogId,omitempty"`
	ExecutedAt time.Time `json:"executedAt"`
}

type routeHandler struct {
	app         core.App
	cfg         config.Config
	mcService   *mc.Service
	auditLogger *audit.Logger
	bootstrapMu sync.Mutex
	bootstrapped bool
}

type authPrincipal struct {
	ActorID    string
	ActorEmail string
	Role       authz.Role
}

func Register(app core.App) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	handler := &routeHandler{
		app:         app,
		cfg:         cfg,
		mcService:   mc.NewService(mc.NewRCONExecutor(cfg.RCON)),
		auditLogger: audit.NewLogger(app, cfg.Collections.AuditLogs),
	}

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		handler.ensureCollections()

		e.Router.AddRoute(echo.Route{
			Method: StatusRouteMethod,
			Path:   StatusRoutePath,
			Handler: func(c echo.Context) error {
				return handler.handleStatus(c)
			},
			Middlewares: []echo.MiddlewareFunc{
				apis.RequireAdminOrRecordAuth(),
			},
		})

		e.Router.AddRoute(echo.Route{
			Method: ExecuteRouteMethod,
			Path:   ExecuteRoutePath,
			Handler: func(c echo.Context) error {
				return handler.handleExecute(c)
			},
			Middlewares: []echo.MiddlewareFunc{
				apis.RequireAdminOrRecordAuth(),
			},
		})

		// Serve the built frontend SPA from /app/pb_public in containerized runtime.
		// Missing routes fall back to index.html for client-side routing.
		e.Router.GET("/*", apis.StaticDirectoryHandler(os.DirFS("pb_public"), true))

		return nil
	})

	return nil
}

func (h *routeHandler) handleStatus(c echo.Context) error {
	h.ensureCollections()

	principal, err := h.resolveAuth(c)
	if err != nil {
		return err
	}

	if !authz.CanViewStatus(principal.Role) {
		return apis.NewForbiddenError("insufficient role for status", nil)
	}

	ctx, cancel := withRequestTimeout(c.Request().Context(), h.cfg.RCON.Timeout)
	defer cancel()

	status, statusErr := h.mcService.Status(ctx)
	if statusErr != nil {
		c.Response().Header().Set("X-MC-Status", "degraded")
	}

	statusOutcome := audit.OutcomeSuccess
	statusErrorMessage := ""
	if statusErr != nil {
		statusOutcome = audit.OutcomeFailed
		statusErrorMessage = statusErr.Error()
	}

	_ = h.writeAudit(c, audit.Entry{
		ActorID:       principal.ActorID,
		ActorEmail:    principal.ActorEmail,
		ActorRole:     string(principal.Role),
		Action:        string("status"),
		Payload:       map[string]any{},
		Outcome:       statusOutcome,
		Error:         statusErrorMessage,
		RequestID:     c.Request().Header.Get("X-Request-Id"),
		IP:            c.RealIP(),
		UserAgent:     c.Request().UserAgent(),
		OccurredAtUTC: time.Now().UTC(),
	})

	return c.JSON(http.StatusOK, status)
}

func (h *routeHandler) handleExecute(c echo.Context) error {
	h.ensureCollections()

	principal, err := h.resolveAuth(c)
	if err != nil {
		return err
	}

	request := executeRequest{}
	if err := json.NewDecoder(c.Request().Body).Decode(&request); err != nil {
		return apis.NewBadRequestError("invalid json body", err)
	}

	action, err := mc.ParseAction(request.Action)
	if err != nil {
		_ = h.writeAudit(c, audit.Entry{
			ActorID:       principal.ActorID,
			ActorEmail:    principal.ActorEmail,
			ActorRole:     string(principal.Role),
			Action:        request.Action,
			Payload:       map[string]any{},
			Outcome:       audit.OutcomeDenied,
			Error:         err.Error(),
			RequestID:     sanitizeRequestID(request.RequestID),
			IP:            c.RealIP(),
			UserAgent:     c.Request().UserAgent(),
			OccurredAtUTC: time.Now().UTC(),
		})
		return apis.NewBadRequestError("unknown action", err)
	}

	if !authz.CanExecuteAction(principal.Role, action, h.cfg.Permissions.AllowRawCommand) {
		_ = h.writeAudit(c, audit.Entry{
			ActorID:       principal.ActorID,
			ActorEmail:    principal.ActorEmail,
			ActorRole:     string(principal.Role),
			Action:        string(action),
			Payload:       map[string]any{},
			Outcome:       audit.OutcomeDenied,
			Error:         "insufficient role",
			RequestID:     sanitizeRequestID(request.RequestID),
			IP:            c.RealIP(),
			UserAgent:     c.Request().UserAgent(),
			OccurredAtUTC: time.Now().UTC(),
		})
		return apis.NewForbiddenError("insufficient role for action", nil)
	}

	plan, err := mc.BuildActionPlan(action, request.Payload, h.cfg.Permissions.AllowRawCommand)
	if err != nil {
		_ = h.writeAudit(c, audit.Entry{
			ActorID:       principal.ActorID,
			ActorEmail:    principal.ActorEmail,
			ActorRole:     string(principal.Role),
			Action:        string(action),
			Payload:       map[string]any{},
			Outcome:       audit.OutcomeDenied,
			Error:         err.Error(),
			RequestID:     sanitizeRequestID(request.RequestID),
			IP:            c.RealIP(),
			UserAgent:     c.Request().UserAgent(),
			OccurredAtUTC: time.Now().UTC(),
		})
		return apis.NewBadRequestError("invalid action payload", err)
	}

	ctx, cancel := withRequestTimeout(c.Request().Context(), h.cfg.RCON.Timeout)
	defer cancel()

	result, err := h.mcService.ExecuteAction(ctx, plan)
	if err != nil {
		auditID := h.writeAudit(c, audit.Entry{
			ActorID:       principal.ActorID,
			ActorEmail:    principal.ActorEmail,
			ActorRole:     string(principal.Role),
			Action:        string(action),
			Payload:       plan.AuditPayload,
			Outcome:       audit.OutcomeFailed,
			Error:         err.Error(),
			RequestID:     sanitizeRequestID(request.RequestID),
			IP:            c.RealIP(),
			UserAgent:     c.Request().UserAgent(),
			OccurredAtUTC: time.Now().UTC(),
		})
		response := executeResponse{
			OK:         false,
			Action:     string(action),
			Message:    "minecraft action failed",
			AuditLogID: auditID,
			ExecutedAt: time.Now().UTC(),
		}
		return c.JSON(http.StatusBadGateway, response)
	}

	auditID := h.writeAudit(c, audit.Entry{
		ActorID:       principal.ActorID,
		ActorEmail:    principal.ActorEmail,
		ActorRole:     string(principal.Role),
		Action:        string(action),
		Payload:       plan.AuditPayload,
		Outcome:       audit.OutcomeSuccess,
		RequestID:     sanitizeRequestID(request.RequestID),
		IP:            c.RealIP(),
		UserAgent:     c.Request().UserAgent(),
		OccurredAtUTC: time.Now().UTC(),
	})

	return c.JSON(http.StatusOK, executeResponse{
		OK:         true,
		Action:     string(result.Action),
		Message:    result.Message,
		AuditLogID: auditID,
		ExecutedAt: result.ExecutedAt,
	})
}

func (h *routeHandler) ensureCollections() {
	h.bootstrapMu.Lock()
	defer h.bootstrapMu.Unlock()

	if h.bootstrapped {
		return
	}

	if err := config.EnsureCollections(h.app, h.cfg); err != nil {
		// Avoid crashing startup if the DB bootstrap isn't ready yet.
		h.app.Logger().Warn("collection bootstrap deferred", "error", err)
		return
	}

	h.bootstrapped = true
}

func (h *routeHandler) resolveAuth(c echo.Context) (authPrincipal, error) {
	admin, _ := c.Get(apis.ContextAdminKey).(*models.Admin)
	if admin != nil {
		return authPrincipal{
			ActorID:    "admin:" + admin.Id,
			ActorEmail: admin.Email,
			Role:       authz.RoleOwner,
		}, nil
	}

	authRecord, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
	if authRecord == nil {
		return authPrincipal{}, apis.NewUnauthorizedError("missing auth", nil)
	}

	return authPrincipal{
		ActorID:    authRecord.Id,
		ActorEmail: authRecord.Email(),
		Role:       authz.RoleFromRecord(authRecord, h.cfg.Permissions.RoleField),
	}, nil
}

func (h *routeHandler) writeAudit(c echo.Context, entry audit.Entry) string {
	if entry.RequestID == "" {
		entry.RequestID = sanitizeRequestID(c.Request().Header.Get("X-Request-Id"))
	}

	id, err := h.auditLogger.Log(entry)
	if err != nil {
		return ""
	}

	return id
}

func withRequestTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		return context.WithCancel(ctx)
	}

	return context.WithTimeout(ctx, timeout)
}

func sanitizeRequestID(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return security.RandomString(12)
	}
	if len(trimmed) > 128 {
		return trimmed[:128]
	}
	return trimmed
}
