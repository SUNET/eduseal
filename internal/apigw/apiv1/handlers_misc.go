package apiv1

import (
	"context"
	"eduseal/internal/gen/status/v1_status"
	"eduseal/pkg/model"
)

// Health return health for this service and dependencies
func (c *Client) Health(ctx context.Context) (*v1_status.StatusReply, error) {
	ctx, span := c.tracer.Start(ctx, "apiv1:Health")
	defer span.End()

	natsStatus := c.stream.Status(ctx)
	kvStatus := c.kv.Status(ctx)

	probes := model.Probes{
		natsStatus,
		kvStatus,
	}

	status := probes.Check("apigw")

	return status, nil
}
