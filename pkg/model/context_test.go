package model

import (
	"context"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestCopyTraceID(t *testing.T) {
	ginContext := &gin.Context{
		Keys: map[string]interface{}{
			"req_id": "test-uuid",
		},
	}

	ctx := CopyTraceID(context.Background(), ginContext)
	assert.Equal(t, "test-uuid", ctx.Value(ContextKey("req_id")))
}
