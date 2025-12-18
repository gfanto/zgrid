package api

import (
	"context"
	"net/http"
	"zgrid/business"
)

// ContextKey differentiates values stored in request contexts.
type ContextKey int

const (
	// GridEventsKey is the context key used to store the event channel.
	GridEventsKey ContextKey = iota
)

// GridEventsMiddleware injects the shared event channel into the request context.
func GridEventsMiddleware(evts chan<- business.Event) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			ctx = context.WithValue(ctx, GridEventsKey, evts)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func getStateEvents(ctx context.Context) chan<- business.Event {
	events, ok := ctx.Value(GridEventsKey).(chan<- business.Event)
	if !ok {
		return nil
	}
	return events
}
