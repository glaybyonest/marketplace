package observability

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"strings"

	"github.com/google/uuid"
)

type AuditEntry struct {
	ActorUserID *uuid.UUID
	Action      string
	EntityType  string
	EntityID    *uuid.UUID
	Metadata    map[string]any
}

type ErrorEvent struct {
	UserID    *uuid.UUID
	Severity  string
	Code      string
	Message   string
	Method    string
	Path      string
	Route     string
	Status    int
	Details   map[string]any
	RequestID string
}

type AuditWriter interface {
	Create(ctx context.Context, entry AuditEntry) error
}

type ErrorEventWriter interface {
	Create(ctx context.Context, event ErrorEvent) error
}

type AuditLogger struct {
	logger  *slog.Logger
	metrics *Metrics
	writer  AuditWriter
}

type ErrorReporter struct {
	logger  *slog.Logger
	metrics *Metrics
	writer  ErrorEventWriter
}

func NewAuditLogger(logger *slog.Logger, metrics *Metrics, writer AuditWriter) *AuditLogger {
	return &AuditLogger{
		logger:  logger,
		metrics: metrics,
		writer:  writer,
	}
}

func NewErrorReporter(logger *slog.Logger, metrics *Metrics, writer ErrorEventWriter) *ErrorReporter {
	return &ErrorReporter{
		logger:  logger,
		metrics: metrics,
		writer:  writer,
	}
}

func (l *AuditLogger) Record(ctx context.Context, entry AuditEntry) error {
	if l == nil {
		return nil
	}

	entry.Action = strings.TrimSpace(entry.Action)
	entry.EntityType = strings.TrimSpace(entry.EntityType)
	if entry.Action == "" || entry.EntityType == "" {
		return nil
	}
	if entry.ActorUserID == nil {
		if actor, ok := ActorFromContext(ctx); ok && actor.UserID != uuid.Nil {
			entry.ActorUserID = &actor.UserID
		}
	}
	if entry.Metadata == nil {
		entry.Metadata = map[string]any{}
	}

	meta := RequestMetaFromContext(ctx)
	attrs := []any{
		"action", entry.Action,
		"entity_type", entry.EntityType,
		"request_id", meta.RequestID,
	}
	if entry.ActorUserID != nil {
		attrs = append(attrs, "actor_user_id", entry.ActorUserID.String())
	}
	if entry.EntityID != nil {
		attrs = append(attrs, "entity_id", entry.EntityID.String())
	}
	if len(entry.Metadata) > 0 {
		attrs = append(attrs, "metadata", entry.Metadata)
	}
	l.logger.Info("audit_event", attrs...)

	if l.metrics != nil {
		l.metrics.RecordAudit(entry.Action)
	}

	if l.writer == nil {
		return nil
	}
	if err := l.writer.Create(ctx, entry); err != nil {
		l.logger.Error("persist_audit_event", "error", err, "action", entry.Action, "entity_type", entry.EntityType)
		return err
	}
	return nil
}

func (r *ErrorReporter) Capture(ctx context.Context, event ErrorEvent) {
	if r == nil {
		return
	}

	if event.RequestID == "" {
		event.RequestID = RequestIDFromContext(ctx)
	}
	if event.UserID == nil {
		if actor, ok := ActorFromContext(ctx); ok && actor.UserID != uuid.Nil {
			event.UserID = &actor.UserID
		}
	}
	if event.Details == nil {
		event.Details = map[string]any{}
	}

	attrs := []any{
		"severity", event.Severity,
		"code", event.Code,
		"status", event.Status,
		"method", event.Method,
		"path", event.Path,
		"route", event.Route,
		"request_id", event.RequestID,
		"message", event.Message,
	}
	if event.UserID != nil {
		attrs = append(attrs, "user_id", event.UserID.String())
	}
	if len(event.Details) > 0 {
		attrs = append(attrs, "details", event.Details)
	}
	r.logger.Error("tracked_error", attrs...)

	if r.metrics != nil {
		r.metrics.RecordError(event.Severity, event.Code)
	}
	if r.writer == nil {
		return
	}
	if err := r.writer.Create(ctx, event); err != nil {
		r.logger.Error("persist_error_event", "error", err, "code", event.Code, "request_id", event.RequestID)
	}
}

func (r *ErrorReporter) CapturePanic(ctx context.Context, recovered any, method, path, route string) {
	r.Capture(ctx, ErrorEvent{
		Severity: "panic",
		Code:     "panic",
		Message:  fmt.Sprint(recovered),
		Method:   method,
		Path:     path,
		Route:    route,
		Status:   500,
		Details: map[string]any{
			"stack": string(debug.Stack()),
		},
	})
}
