package httpserver

import (
	"context"
	apiv1_status "eduseal/internal/gen/status/apiv1.status"
)

// Apiv1 interface
type Apiv1 interface {
	Status(ctx context.Context, req *apiv1_status.StatusRequest) (*apiv1_status.StatusReply, error)
}
