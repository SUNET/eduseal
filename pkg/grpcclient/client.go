package grpcclient

import (
	"context"
	"eduseal/pkg/logger"
	"eduseal/pkg/model"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/resolver"
)

// Client is the gRPC client object
type Client struct {
	cfg *model.Cfg
	log *logger.Log

	Validator *Validator
	Sealer    *Sealer
}

// New creates a new instance of the gRPC client
func New(ctx context.Context, cfg *model.Cfg, log *logger.Log) (*Client, error) {
	c := &Client{
		cfg: cfg,
		log: log,
	}

	c.Validator = &Validator{
		client:      c,
		scheme:      "validator",
		serviceName: "validator.eduseal.sunet.docker",
		DNS: map[string][]string{
			"validator.eduseal.sunet.docker": cfg.Common.ValidatorGRPCHosts,
		},
	}
	resolver.Register(c.Validator)

	c.Sealer = &Sealer{
		client:      c,
		scheme:      "sealer",
		serviceName: "sealer.eduseal.sunet.docker",
		DNS: map[string][]string{
			"sealer.eduseal.sunet.docker": cfg.Common.SealerGRPCHosts,
		},
	}
	resolver.Register(c.Sealer)

	return c, nil
}

func (c *Client) rrConn(ctx context.Context, scheme, serviceName string) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(
		fmt.Sprintf("%s:///%s", scheme, serviceName),
		grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [{"round_robin":{}}]}`), // This sets the initial balancing policy.
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		//	c.log.Error(err, "Failed to connect to validator")
		return nil, err
	}

	return conn, nil
}
