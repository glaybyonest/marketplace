package observability

import (
	"context"

	"github.com/google/uuid"
)

type requestMetaKey struct{}
type actorKey struct{}
type errorReporterKey struct{}

type RequestMeta struct {
	RequestID string
	Method    string
	Path      string
	RemoteIP  string
	UserAgent string
}

type Actor struct {
	UserID uuid.UUID
	Email  string
	Role   string
}

func WithRequestMeta(ctx context.Context, meta RequestMeta) context.Context {
	return context.WithValue(ctx, requestMetaKey{}, meta)
}

func RequestMetaFromContext(ctx context.Context) RequestMeta {
	value, _ := ctx.Value(requestMetaKey{}).(RequestMeta)
	return value
}

func WithActor(ctx context.Context, actor Actor) context.Context {
	return context.WithValue(ctx, actorKey{}, actor)
}

func ActorFromContext(ctx context.Context) (Actor, bool) {
	value, ok := ctx.Value(actorKey{}).(Actor)
	return value, ok
}

func WithErrorReporter(ctx context.Context, reporter *ErrorReporter) context.Context {
	return context.WithValue(ctx, errorReporterKey{}, reporter)
}

func ErrorReporterFromContext(ctx context.Context) (*ErrorReporter, bool) {
	value, ok := ctx.Value(errorReporterKey{}).(*ErrorReporter)
	return value, ok
}
