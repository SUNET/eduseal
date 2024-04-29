package apiv1

import (
	"context"
	"eduseal/internal/apigw/db"
	"eduseal/internal/apigw/simplequeue"
	"eduseal/pkg/kvclient"
	"eduseal/pkg/logger"
	"eduseal/pkg/model"
	"eduseal/pkg/trace"
)

var (
	BuildVarGitCommit string
)

//	@title		Datastore API
//	@version	0.1.0
//	@BasePath	/api/v1

// Client holds the public api object
type Client struct {
	cfg         *model.Cfg
	db          *db.Service
	log         *logger.Log
	tp          *trace.Tracer
	kv          *kvclient.Client
	simpleQueue *simplequeue.Service
}

// New creates a new instance of the public api
func New(ctx context.Context, kv *kvclient.Client, db *db.Service, simplequeue *simplequeue.Service, tp *trace.Tracer, cfg *model.Cfg, logger *logger.Log) (*Client, error) {
	c := &Client{
		cfg:         cfg,
		db:          db,
		log:         logger,
		kv:          kv,
		tp:          tp,
		simpleQueue: simplequeue,
	}

	c.log.Info("Started")

	return c, nil
}
