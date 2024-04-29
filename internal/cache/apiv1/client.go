package apiv1

import (
	"context"
	"eduseal/pkg/kvclient"
	"eduseal/pkg/logger"
	"eduseal/pkg/model"
	"eduseal/pkg/trace"
)

//	@title		Datastore API
//	@version	0.1.0
//	@BasePath	/datastore/api/v1

// Client holds the public api object
type Client struct {
	cfg *model.Cfg
	log *logger.Log
	tp  *trace.Tracer
	kv  *kvclient.Client
}

// New creates a new instance of the public api
func New(ctx context.Context, kv *kvclient.Client, tp *trace.Tracer, cfg *model.Cfg, logger *logger.Log) (*Client, error) {
	c := &Client{
		cfg: cfg,
		kv:  kv,
		tp:  tp,
		log: logger,
	}

	c.log.Info("Started")

	return c, nil
}
