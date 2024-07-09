package kvclient

import (
	"context"
	"crypto/x509"
	"eduseal/internal/gen/status/v1_status"
	"eduseal/pkg/logger"
	"eduseal/pkg/model"
	"eduseal/pkg/trace"
	"os"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/redis/go-redis/v9"
)

// Client holds the kv object
type Client struct {
	RedictCC     *redis.ClusterClient
	cfg          *model.Cfg
	log          *logger.Log
	probeStore   *v1_status.StatusProbeStore
	tp           *trace.Tracer
	statusResult statusResults
	statusTick   *time.Ticker

	Doc *Doc
}
type statusResults map[string]statusResult

type statusResult struct {
	healthy bool
	leader  bool
}

// New creates a new instance of kv
func New(ctx context.Context, cfg *model.Cfg, tracer *trace.Tracer, log *logger.Log) (*Client, error) {
	c := &Client{
		cfg:        cfg,
		log:        log,
		probeStore: &v1_status.StatusProbeStore{},
		tp:         tracer,
		statusTick: time.NewTicker(time.Second * 10),
	}

	//clientCert, err := tls.LoadX509KeyPair(cfg.APIGW.ClientCert.CertFilePath, cfg.APIGW.ClientCert.KeyFilePath)
	//if err != nil {
	//	return nil, err
	//}

	// Load CA cert
	caCertByte, err := os.ReadFile(cfg.APIGW.ClientCert.RootCAPath)
	if err != nil {
		return nil, err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCertByte)

	c.RedictCC = redis.NewClusterClient(
		&redis.ClusterOptions{
			Addrs:    cfg.Common.Redict.Nodes,
			Password: cfg.Common.Redict.Password,
		},
	)

	c.Doc = &Doc{
		client: c,
		key:    "doc:%s:%s",
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-c.statusTick.C:
				c.log.Info("Checking status")
				res, err := c.RedictCC.Ping(ctx).Result()
				if err != nil {
					c.log.Error(err, "Error checking status")
				}
				c.log.Info("Status", "result", res)
			}
		}
	}()

	c.log.Info("Started")

	return c, nil
}

// Status returns the status of the database
func (c *Client) Status(ctx context.Context) *v1_status.StatusProbe {
	if time.Now().Before(c.probeStore.NextCheck.AsTime()) {
		return c.probeStore.PreviousResult
	}
	probe := &v1_status.StatusProbe{
		Name:          "kv",
		Healthy:       true,
		Message:       "OK",
		LastCheckedTS: timestamppb.Now(),
	}

	_, err := c.RedictCC.Ping(ctx).Result()
	if err != nil {
		probe.Message = err.Error()
		probe.Healthy = false
	}
	c.probeStore.PreviousResult = probe
	c.probeStore.NextCheck = timestamppb.New(time.Now().Add(time.Second * 10))

	return probe
}

// Close closes the connection to the database
func (c *Client) Close(ctx context.Context) error {
	return c.RedictCC.Close()
}
