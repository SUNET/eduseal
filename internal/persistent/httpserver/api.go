package httpserver

import (
	"context"
	v1_status "eduseal/internal/gen/status/v1_status"
)

// Apiv1 interface
type Apiv1 interface {
	Status(ctx context.Context, req *v1_status.StatusRequest) (*v1_status.StatusReply, error)
}
