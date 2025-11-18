package apiv1

import (
	"context"
	"eduseal/internal/apigw/stream"
	"eduseal/pkg/grpcclient"
	"eduseal/pkg/kvclient"
	"eduseal/pkg/logger"
	"eduseal/pkg/metric"
	"eduseal/pkg/model"
	"eduseal/pkg/trace"
)

//	@title		EduSeal API
//	@version	0.1.0
//	@BasePath	/api/v1

// Client holds the public api object
type Client struct {
	cfg        *model.Cfg
	stream     *stream.Service
	log        *logger.Log
	tracer     *trace.Tracer
	kv         *kvclient.Client
	metric     *metric.Metric
	grpcClient *grpcclient.Client
}

// New creates a new instance of the public api
func New(ctx context.Context, kv *kvclient.Client, grpcClient *grpcclient.Client, streamService *stream.Service, tracer *trace.Tracer, metric *metric.Metric, cfg *model.Cfg, logger *logger.Log) (*Client, error) {
	c := &Client{
		cfg:        cfg,
		stream:     streamService,
		log:        logger,
		tracer:     tracer,
		metric:     metric,
		kv:         kv,
		grpcClient: grpcClient,
	}

	c.log.Info("Started")

	return c, nil
}
