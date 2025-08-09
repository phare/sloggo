package db

import (
	"context"
	"log"
	"time"
)

type Hooks struct{}

// Before hook will print the query with it's args and return the context with the timestamp
func (h *Hooks) Before(ctx context.Context, query string, args ...any) (context.Context, error) {
	return context.WithValue(ctx, "queryStart", time.Now()), nil
}

// After hook will get the timestamp registered on the Before hook and print the elapsed time
func (h *Hooks) After(ctx context.Context, query string, args ...any) (context.Context, error) {
	queryStart := ctx.Value("queryStart").(time.Time)
	log.Printf("🔍 Query: %s | Args: %v | Duration: %s", query, args, time.Since(queryStart))
	return ctx, nil
}
