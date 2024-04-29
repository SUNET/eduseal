package apiv1

import (
	"context"
	apiv1_status "eduseal/internal/gen/status/apiv1.status"
	"eduseal/pkg/model"
)

// Status return status for each ladok instance
func (c *Client) Status(ctx context.Context, req *apiv1_status.StatusRequest) (*apiv1_status.StatusReply, error) {
	ctx, span := c.tp.Start(ctx, "apiv1:Status")
	defer span.End()

	probes := model.Probes{}
	probes = append(probes, c.kv.Status(ctx))

	status := probes.Check("cache")

	return status, nil
}
