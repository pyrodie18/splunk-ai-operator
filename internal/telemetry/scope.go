package telemetry

import "context"

type Scope struct {
	Namespace string
	Name      string
	Kind      string // "AIPlatform" | "AIService"
	Feature   string // "platform" for AIPlatform, or e.g. "saia" | "seca" for AIService
}

type scopeKey struct{}

func WithScope(ctx context.Context, s Scope) context.Context {
	return context.WithValue(ctx, scopeKey{}, s)
}

func FromContext(ctx context.Context) (Scope, bool) {
	s, ok := ctx.Value(scopeKey{}).(Scope)
	return s, ok
}
