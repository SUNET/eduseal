package model

import (
	"context"

	"github.com/gin-gonic/gin"
)

type (
	// ContextKey key of
	ContextKey string
)

func (c ContextKey) String() string {
	return string(c)
}

// CopyTraceID copy trace ID from gin context to golang context
func CopyTraceID(ctx context.Context, c *gin.Context) context.Context {
	name := "req_id"
	id := c.GetString(name)

	ctxValue := context.WithValue(ctx, ContextKey(name), id)

	return ctxValue
}
