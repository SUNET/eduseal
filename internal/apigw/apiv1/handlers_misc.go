package apiv1

import (
	"context"
	"eduseal/internal/gen/status/v1_status"
	"eduseal/pkg/model"
)

// Health return health for this service and dependencies
func (c *Client) Health(ctx context.Context, req *v1_status.StatusRequest) (*v1_status.StatusReply, error) {
	c.log.Info("health handler")
	probes := model.Probes{}

	//probes = append(probes, c.kv.Probe)

	if !c.cfg.Common.Mongo.Disable {
		probes = append(probes, c.db.Status(ctx))
	}

	status := probes.Check("apigw")

	return status, nil
}
