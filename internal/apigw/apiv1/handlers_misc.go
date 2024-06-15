package apiv1

import (
	"context"
	apiv1_status "eduseal/internal/gen/status/apiv1.status"
	"eduseal/pkg/model"
)

// Health return health for this service and dependencies
func (c *Client) Health(ctx context.Context, req *apiv1_status.StatusRequest) (*apiv1_status.StatusReply, error) {
	c.log.Info("health handler")
	probes := model.Probes{}
	probes = append(probes, c.kv.Status(ctx))
	if !c.cfg.Common.Mongo.Disable {
		probes = append(probes, c.db.Status(ctx))
	}

	status := probes.Check("apigw")

	return status, nil
}
