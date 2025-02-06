package apiv1

import (
	"context"
	"eduseal/internal/apigw/db"
	"eduseal/internal/apigw/stream"
	"eduseal/pkg/grpcclient"
	"eduseal/pkg/kvclient"
	"eduseal/pkg/logger"
	"eduseal/pkg/model"
	"eduseal/pkg/trace"
)

//	@title		Datastore API
//	@version	0.1.0
//	@BasePath	/api/v1

// Client holds the public api object
type Client struct {
	cfg        *model.Cfg
	db         *db.Service
	stream     *stream.Service
	log        *logger.Log
	tp         *trace.Tracer
	kv         *kvclient.Client
	grpcClient *grpcclient.Client
}

// New creates a new instance of the public api
func New(ctx context.Context, kv *kvclient.Client, grpcClient *grpcclient.Client, db *db.Service, streamService *stream.Service, tp *trace.Tracer, cfg *model.Cfg, logger *logger.Log) (*Client, error) {
	c := &Client{
		cfg:        cfg,
		db:         db,
		stream:     streamService,
		log:        logger,
		tp:         tp,
		kv:         kv,
		grpcClient: grpcClient,
	}

	c.log.Info("Started")

	return c, nil
}
