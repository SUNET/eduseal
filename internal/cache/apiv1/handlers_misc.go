package apiv1

import (
	"context"
	"eduseal/internal/gen/status/v1_status"
	"eduseal/pkg/model"
)

// Status return status for each ladok instance
func (c *Client) Status(ctx context.Context, req *v1_status.StatusRequest) (*v1_status.StatusReply, error) {
	ctx, span := c.tp.Start(ctx, "apiv1:Status")
	defer span.End()

	probes := model.Probes{}

	status := probes.Check("cache")

	return status, nil
}
