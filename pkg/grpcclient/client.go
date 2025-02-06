package grpcclient

import (
	"context"
	"eduseal/pkg/logger"
	"eduseal/pkg/model"
	"eduseal/pkg/trace"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/resolver"
)

// Client is the gRPC client object
type Client struct {
	cfg *model.Cfg
	log *logger.Log
	tp  *trace.Tracer

	Validator *Validator
	Sealer    *Sealer
}

// New creates a new instance of the gRPC client
func New(ctx context.Context, cfg *model.Cfg, tp *trace.Tracer, log *logger.Log) (*Client, error) {
	c := &Client{
		cfg: cfg,
		log: log,
		tp:  tp,
	}

	c.Validator = &Validator{
		client: c,
		scheme: "validator",
		DNS: map[string][]string{
			c.cfg.Common.ValidatorServiceName: cfg.Common.ValidatorNodes,
		},
	}
	resolver.Register(c.Validator)

	c.Sealer = &Sealer{
		client: c,
		scheme: "sealer",
		DNS: map[string][]string{
			c.cfg.Common.SealerServiceName: cfg.Common.SealerNodes,
		},
	}
	resolver.Register(c.Sealer)

	return c, nil
}

func (c *Client) rrConn(ctx context.Context, scheme, serviceName string) (*grpc.ClientConn, error) {
	clientTLS, err := credentials.NewClientTLSFromFile(c.cfg.Common.RootCAPath, "")
	if err != nil {
		return nil, err
	}
	conn, err := grpc.NewClient(
		fmt.Sprintf("%s:///%s", scheme, serviceName),
		grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [{"round_robin":{}}]}`), // This sets the initial balancing policy.
		grpc.WithTransportCredentials(clientTLS),
	)
	if err != nil {
		//	c.log.Error(err, "Failed to connect to validator")
		return nil, err
	}

	return conn, nil
}
