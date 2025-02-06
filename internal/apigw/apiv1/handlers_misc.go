package apiv1

import (
	"context"
	"eduseal/internal/gen/status/v1_status"
	"eduseal/pkg/model"
)

// Health return health for this service and dependencies
func (c *Client) Health(ctx context.Context) (*v1_status.StatusReply, error) {
	ctx, span := c.tp.Start(ctx, "apiv1:Health")
	defer span.End()

	natsStatus := c.stream.Status(ctx)
	kvStatus := c.kv.Status(ctx)

	probes := model.Probes{
		natsStatus,
		kvStatus,
	}

	if !c.cfg.Common.Mongo.Disable {
		probes = append(probes, c.db.Status(ctx))
	}

	status := probes.Check("apigw")

	return status, nil
}

// MetricReply is the reply for metrics
type MetricReply struct {
	Signings    int64
	Fetches     int64
	Validations int64
}

// Metrics return metrics for this service
func (c *Client) Metrics(ctx context.Context) (*MetricReply, error) {
	c.log.Info("metrics handler")

	signingMetric, err := c.kv.MetricSigning.Get(ctx)
	if err != nil {
		c.log.Error(err, "failed to get signing metric")
		return nil, err
	}

	fetchMetric, err := c.kv.MetricFetching.Get(ctx)
	if err != nil {
		c.log.Error(err, "failed to get fetching metric")
		return nil, err
	}

	validationMetric, err := c.kv.MetricValidations.Get(ctx)
	if err != nil {
		c.log.Error(err, "failed to get validation metric")
		return nil, err
	}

	reply := &MetricReply{
		Signings:    signingMetric,
		Fetches:     fetchMetric,
		Validations: validationMetric,
	}

	return reply, nil
}
